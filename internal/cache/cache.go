package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Dir falls back to the temp directory rather than failing, so a container with
// no HOME still installs, just without carrying the cache between runs.
func Dir() string {
	if dir := os.Getenv("PRESTO_CACHE_DIR"); dir != "" {
		return dir
	}

	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "presto")
	}

	if runtime.GOOS == "windows" {
		if dir, err := os.UserCacheDir(); err == nil {
			return filepath.Join(dir, "presto")
		}

		return filepath.Join(os.TempDir(), "presto")
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".cache", "presto")
	}

	return filepath.Join(os.TempDir(), "presto")
}

// Metadata returns the file a package manifest is cached in, creating its
// directory. The companion .etag file next to it carries the revalidation token.
func Metadata(name string) (string, error) {
	path := filepath.Join(Dir(), "metadata", filepath.FromSlash(safe(name))+".json")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	return path, nil
}

// Archive keys a downloaded zip by package, version and dist URL, so a package
// republished at the same version is not served from a stale entry.
func Archive(name, version, url string) (string, error) {
	sum := sha256.Sum256([]byte(url))
	file := safe(version) + "-" + hex.EncodeToString(sum[:])[:12] + ".zip"
	path := filepath.Join(Dir(), "archives", filepath.FromSlash(safe(name)), file)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	return path, nil
}

func Clear() (string, error) {
	dir := Dir()

	return dir, os.RemoveAll(dir)
}

// safe keeps a package name usable as a path on every platform without letting
// it climb out of the cache directory.
func safe(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))

	var b strings.Builder

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.', r == '/':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}

	return strings.ReplaceAll(b.String(), "..", "--")
}
