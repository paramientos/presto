package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ComposerJSON represents the structure of composer.json
type ComposerJSON struct {
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Type         string                 `json:"type,omitempty"`
	License      string                 `json:"license,omitempty"`
	Authors      []Author               `json:"authors,omitempty"`
	Require      map[string]string      `json:"require,omitempty"`
	RequireDev   map[string]string      `json:"require-dev,omitempty"`
	Autoload     AutoloadConfig         `json:"autoload,omitempty"`
	AutoloadDev  AutoloadConfig         `json:"autoload-dev,omitempty"`
	Scripts      map[string]interface{} `json:"scripts,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Repositories interface{}            `json:"repositories,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// Author represents a package author
type Author struct {
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	Homepage string `json:"homepage,omitempty"`
	Role     string `json:"role,omitempty"`
}

// AutoloadConfig represents autoload configuration
type AutoloadConfig struct {
	PSR4                map[string]interface{} `json:"psr-4,omitempty"`
	PSR0                map[string]interface{} `json:"psr-0,omitempty"`
	Classmap            []string               `json:"classmap,omitempty"`
	Files               []string               `json:"files,omitempty"`
	ExcludeFromClassmap []string               `json:"exclude-from-classmap,omitempty"`
}

// ComposerLock represents the structure of composer.lock
type ComposerLock struct {
	Readme           []string          `json:"_readme,omitempty"`
	ContentHash      string            `json:"content-hash"`
	Packages         []LockedPackage   `json:"packages"`
	PackagesDev      []LockedPackage   `json:"packages-dev"`
	Aliases          []interface{}     `json:"aliases"`
	MinimumStability string            `json:"minimum-stability,omitempty"`
	StabilityFlags   map[string]int    `json:"stability-flags,omitempty"`
	PreferStable     bool              `json:"prefer-stable,omitempty"`
	PreferLowest     bool              `json:"prefer-lowest,omitempty"`
	Platform         map[string]string `json:"platform,omitempty"`
	PlatformDev      map[string]string `json:"platform-dev,omitempty"`
}

// LockedPackage represents a package in composer.lock
type LockedPackage struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Source          SourceInfo        `json:"source,omitempty"`
	Dist            DistInfo          `json:"dist,omitempty"`
	Require         map[string]string `json:"require,omitempty"`
	RequireDev      map[string]string `json:"require-dev,omitempty"`
	Type            string            `json:"type,omitempty"`
	Autoload        AutoloadConfig    `json:"autoload,omitempty"`
	NotificationURL string            `json:"notification-url,omitempty"`
	License         []string          `json:"license,omitempty"`
	Authors         []Author          `json:"authors,omitempty"`
	Description     string            `json:"description,omitempty"`
	Keywords        []string          `json:"keywords,omitempty"`
	Time            string            `json:"time,omitempty"`
}

// SourceInfo represents source repository information
type SourceInfo struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference"`
}

// DistInfo represents distribution archive information
type DistInfo struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference,omitempty"`
	Shasum    string `json:"shasum,omitempty"`
}

// ParseComposerJSON reads and parses composer.json
func ParseComposerJSON(path string) (*ComposerJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var composer ComposerJSON
	if err := json.Unmarshal(data, &composer); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &composer, nil
}

// WriteComposerJSON writes composer.json
func WriteComposerJSON(path string, composer *ComposerJSON) error {
	data, err := json.MarshalIndent(composer, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

// ParseComposerLock reads and parses composer.lock
func ParseComposerLock(path string) (*ComposerLock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var lock ComposerLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &lock, nil
}

// WriteComposerLock writes composer.lock
func WriteComposerLock(path string, lock *ComposerLock) error {
	data, err := json.MarshalIndent(lock, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

// NormalizePackageName normalizes package names to lowercase
func NormalizePackageName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// IsValidPackageName checks if a package name is valid
func IsValidPackageName(name string) bool {
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return false
	}
	return parts[0] != "" && parts[1] != ""
}

// GetAllDependencies returns all dependencies (require + require-dev)
func (c *ComposerJSON) GetAllDependencies() map[string]string {
	all := make(map[string]string)

	for pkg, version := range c.Require {
		all[pkg] = version
	}

	for pkg, version := range c.RequireDev {
		all[pkg] = version
	}

	return all
}

// GetProductionDependencies returns only production dependencies
func (c *ComposerJSON) GetProductionDependencies() map[string]string {
	deps := make(map[string]string)

	for pkg, version := range c.Require {
		// Skip platform requirements
		if !strings.HasPrefix(pkg, "php") && !strings.HasPrefix(pkg, "ext-") && !strings.HasPrefix(pkg, "lib-") {
			deps[pkg] = version
		}
	}

	return deps
}
