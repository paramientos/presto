package trust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

const storeVersion = 1

type Entry struct {
	Path        string    `json:"path"`
	ScriptsHash string    `json:"scripts-hash"`
	TrustedAt   time.Time `json:"trusted-at"`
}

type storeFile struct {
	Version  int              `json:"version"`
	Projects map[string]Entry `json:"projects"`
}

type Store struct {
	path string
	file storeFile
}

func Load() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}

	return LoadFrom(path)
}

func LoadFrom(path string) (*Store, error) {
	store := &Store{
		path: path,
		file: storeFile{Version: storeVersion, Projects: map[string]Entry{}},
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	if err := json.Unmarshal(data, &store.file); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if store.file.Projects == nil {
		store.file.Projects = map[string]Entry{}
	}

	return store, nil
}

func DefaultPath() (string, error) {
	if dir := os.Getenv("PRESTO_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "trust.json"), nil
	}

	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "presto", "trust.json"), nil
	}

	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "presto", "trust.json"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "presto", "trust.json"), nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Allows(projectDir, scriptsHash string) bool {
	entry, ok := s.file.Projects[Resolve(projectDir)]
	if !ok {
		return false
	}

	return entry.ScriptsHash == scriptsHash
}

func (s *Store) Allow(projectDir, scriptsHash string) error {
	k := Resolve(projectDir)

	s.file.Projects[k] = Entry{
		Path:        k,
		ScriptsHash: scriptsHash,
		TrustedAt:   time.Now().UTC(),
	}

	return s.save()
}

func (s *Store) Revoke(projectDir string) (bool, error) {
	k := Resolve(projectDir)

	if _, ok := s.file.Projects[k]; !ok {
		return false, nil
	}

	delete(s.file.Projects, k)

	return true, s.save()
}

func (s *Store) Entries() []Entry {
	entries := make([]Entry, 0, len(s.file.Projects))
	for _, entry := range s.file.Projects {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	return entries
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("failed to create %s: %w", filepath.Dir(s.path), err)
	}

	s.file.Version = storeVersion

	data, err := json.MarshalIndent(s.file, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", s.path, err)
	}

	return nil
}

func Hash(scripts []Script) string {
	sorted := make([]Script, len(scripts))
	copy(sorted, scripts)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Event < sorted[j].Event
	})

	sum := sha256.New()
	for _, script := range sorted {
		_, _ = fmt.Fprintf(sum, "%s\n", script.Event)
		for _, command := range script.Commands {
			_, _ = fmt.Fprintf(sum, "\t%s\n", command)
		}
	}

	return hex.EncodeToString(sum.Sum(nil))
}

// Resolve turns a project directory into the key the store is written under.
func Resolve(projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return projectDir
	}

	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}

	return abs
}
