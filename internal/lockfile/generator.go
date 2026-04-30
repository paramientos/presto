package lockfile

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/aras/presto/internal/packagist"
	"github.com/aras/presto/internal/parser"
	"github.com/aras/presto/internal/resolver"
)

type Generator struct {
	client *packagist.Client
}

func NewGenerator() *Generator {
	return &Generator{}
}

func NewGeneratorWithClient(client *packagist.Client) *Generator {
	return &Generator{client: client}
}

func (g *Generator) Generate(composer *parser.ComposerJSON, packages []*resolver.Package) error {
	lock := &parser.ComposerLock{
		Readme: []string{
			"This file locks the dependencies of your project to a known state",
			"Read more about it at https://getcomposer.org/doc/01-basic-usage.md#installing-dependencies",
			"This file is @generated automatically",
		},
		ContentHash:      g.GenerateContentHash(composer),
		Packages:         g.convertToLockedPackages(packages, false),
		PackagesDev:      g.convertToLockedPackages(packages, true),
		Aliases:          []interface{}{},
		MinimumStability: g.minimumStability(composer),
		StabilityFlags:   map[string]int{},
		PreferStable:     g.preferStable(composer),
		PreferLowest:     false,
	}

	// Platform requirements from require (production)
	if composer.Require != nil {
		lock.Platform = make(map[string]string)
		for name, version := range composer.Require {
			if g.isPlatformRequirement(name) {
				lock.Platform[name] = version
			}
		}
		if len(lock.Platform) == 0 {
			lock.Platform = map[string]string{}
		}
	}

	// Platform requirements from require-dev
	if composer.RequireDev != nil {
		lock.PlatformDev = make(map[string]string)
		for name, version := range composer.RequireDev {
			if g.isPlatformRequirement(name) {
				lock.PlatformDev[name] = version
			}
		}
		if len(lock.PlatformDev) == 0 {
			lock.PlatformDev = map[string]string{}
		}
	}

	return parser.WriteComposerLock("composer.lock", lock)
}

// GenerateContentHash replicates Composer's content hash algorithm.
// Composer hashes only specific fields from composer.json, sorted by key.
// See: Composer\Package\Locker::getContentHash()
func (g *Generator) GenerateContentHash(composer *parser.ComposerJSON) string {
	relevantKeys := []string{
		"name", "version", "require", "require-dev", "conflict",
		"replace", "provide", "minimum-stability", "prefer-stable",
		"repositories", "extra",
	}

	// Re-marshal composer.json to a generic map so we can pick only relevant keys
	raw, err := json.Marshal(composer)
	if err != nil {
		return ""
	}

	var full map[string]json.RawMessage
	if err := json.Unmarshal(raw, &full); err != nil {
		return ""
	}

	// Build a sorted map of only the relevant keys that are present
	relevant := make(map[string]json.RawMessage)
	for _, key := range relevantKeys {
		if val, ok := full[key]; ok {
			relevant[key] = val
		}
	}

	// Sort keys for deterministic output (Composer sorts them too)
	sortedKeys := make([]string, 0, len(relevant))
	for k := range relevant {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	// Build a JSON object with sorted keys
	var sb strings.Builder
	sb.WriteString("{")
	for i, k := range sortedKeys {
		if i > 0 {
			sb.WriteString(",")
		}
		keyJSON, _ := json.Marshal(k)
		sb.Write(keyJSON)
		sb.WriteString(":")
		sb.Write(relevant[k])
	}
	sb.WriteString("}")

	hash := md5.Sum([]byte(sb.String()))
	return fmt.Sprintf("%x", hash)
}

func (g *Generator) convertToLockedPackages(packages []*resolver.Package, devOnly bool) []parser.LockedPackage {
	var locked []parser.LockedPackage

	for _, pkg := range packages {
		if pkg.IsDev != devOnly {
			continue
		}

		lockedPkg := g.buildLockedPackage(pkg)
		locked = append(locked, lockedPkg)
	}

	if locked == nil {
		locked = []parser.LockedPackage{}
	}

	return locked
}

func (g *Generator) buildLockedPackage(pkg *resolver.Package) parser.LockedPackage {
	lockedPkg := parser.LockedPackage{
		Name:    pkg.Name,
		Version: pkg.Version,
		Type:    "library",
	}

	// Populate metadata from Packagist if client is available
	if g.client != nil {
		if versionInfo, err := g.client.GetVersion(pkg.Name, pkg.Version); err == nil {
			lockedPkg.Description = versionInfo.Description
			lockedPkg.Keywords = versionInfo.Keywords
			lockedPkg.License = versionInfo.License
			lockedPkg.Time = versionInfo.Time

			if versionInfo.Type != "" {
				lockedPkg.Type = versionInfo.Type
			}

			if versionInfo.NotificationURL != "" {
				lockedPkg.NotificationURL = versionInfo.NotificationURL
			} else {
				lockedPkg.NotificationURL = "https://packagist.org/downloads/"
			}

			// Authors: convert from packagist.Author to parser.Author
			if len(versionInfo.Authors) > 0 {
				for _, a := range versionInfo.Authors {
					lockedPkg.Authors = append(lockedPkg.Authors, parser.Author{
						Name:     a.Name,
						Email:    a.Email,
						Homepage: a.Homepage,
						Role:     a.Role,
					})
				}
			}

			// Source info
			if versionInfo.Source.URL != "" {
				lockedPkg.Source = parser.SourceInfo{
					Type:      versionInfo.Source.Type,
					URL:       versionInfo.Source.URL,
					Reference: versionInfo.Source.Reference,
				}
			}

			// Dist info with reference and shasum
			lockedPkg.Dist = parser.DistInfo{
				Type:      versionInfo.Dist.Type,
				URL:       versionInfo.Dist.URL,
				Reference: versionInfo.Dist.Reference,
				Shasum:    versionInfo.Dist.Shasum,
			}
			if lockedPkg.Dist.Type == "" {
				lockedPkg.Dist.Type = "zip"
			}
			if lockedPkg.Dist.URL == "" {
				lockedPkg.Dist.URL = pkg.URL
			}

			// Autoload
			if len(versionInfo.Autoload) > 0 && string(versionInfo.Autoload) != "null" {
				var autoload parser.AutoloadConfig
				if err := json.Unmarshal(versionInfo.Autoload, &autoload); err == nil {
					lockedPkg.Autoload = autoload
				}
			}

			lockedPkg.Require = versionInfo.Require
			lockedPkg.RequireDev = versionInfo.RequireDev
		}
	}

	// Fallback values if client wasn't available or lookup failed
	if lockedPkg.Dist.URL == "" {
		lockedPkg.Dist = parser.DistInfo{
			Type: "zip",
			URL:  pkg.URL,
		}
	}
	if lockedPkg.Require == nil {
		lockedPkg.Require = pkg.Require
	}

	return lockedPkg
}

func (g *Generator) isPlatformRequirement(name string) bool {
	return name == "php" ||
		strings.HasPrefix(name, "php-") ||
		strings.HasPrefix(name, "ext-") ||
		strings.HasPrefix(name, "lib-") ||
		name == "composer-plugin-api" ||
		name == "composer-runtime-api"
}

func (g *Generator) minimumStability(composer *parser.ComposerJSON) string {
	if composer.MinimumStability != "" {
		return composer.MinimumStability
	}
	return "stable"
}

func (g *Generator) preferStable(composer *parser.ComposerJSON) bool {
	return composer.PreferStable
}
