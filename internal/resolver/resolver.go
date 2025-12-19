package resolver

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/aras/presto/internal/packagist"
	"github.com/aras/presto/internal/parser"
)

// Resolver handles dependency resolution
type Resolver struct {
	client   *packagist.Client
	resolved map[string]string
	visited  map[string]bool
}

// Package represents a resolved package
type Package struct {
	Name    string
	Version string
	URL     string
	Require map[string]string
}

// NewResolver creates a new dependency resolver
func NewResolver(client *packagist.Client) *Resolver {
	return &Resolver{
		client:   client,
		resolved: make(map[string]string),
		visited:  make(map[string]bool),
	}
}

// Resolve resolves all dependencies for a composer.json
func (r *Resolver) Resolve(composer *parser.ComposerJSON) ([]*Package, error) {
	var packages []*Package

	// Resolve production dependencies
	for name, constraint := range composer.Require {
		// Skip platform requirements
		if r.isPlatformPackage(name) {
			continue
		}

		if err := r.resolveDependency(name, constraint, &packages); err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", name, err)
		}
	}

	// Resolve dev dependencies
	for name, constraint := range composer.RequireDev {
		if r.isPlatformPackage(name) {
			continue
		}

		if err := r.resolveDependency(name, constraint, &packages); err != nil {
			return nil, fmt.Errorf("failed to resolve dev dependency %s: %w", name, err)
		}
	}

	return packages, nil
}

// resolveDependency resolves a single dependency recursively
func (r *Resolver) resolveDependency(name, constraint string, packages *[]*Package) error {
	// Check if already resolved
	if r.visited[name] {
		return nil
	}
	r.visited[name] = true

	// Fetch package info
	info, err := r.client.GetPackage(name)
	if err != nil {
		return err
	}

	// Find matching version
	version, err := r.findMatchingVersion(info, constraint)
	if err != nil {
		return fmt.Errorf("no matching version for %s %s: %w", name, constraint, err)
	}

	// Get version details
	versionInfo, err := r.client.GetVersion(name, version)
	if err != nil {
		return err
	}

	// Get download URL
	downloadURL := versionInfo.Dist.URL
	if downloadURL == "" && versionInfo.Source.URL != "" {
		downloadURL = versionInfo.Source.URL
	}

	// Add to packages
	pkg := &Package{
		Name:    name,
		Version: version,
		URL:     downloadURL,
		Require: versionInfo.Require,
	}
	*packages = append(*packages, pkg)

	// Store resolved version
	r.resolved[name] = version

	// Resolve dependencies of this package
	for depName, depConstraint := range versionInfo.Require {
		if r.isPlatformPackage(depName) {
			continue
		}

		if err := r.resolveDependency(depName, depConstraint, packages); err != nil {
			return err
		}
	}

	return nil
}

// findMatchingVersion finds a version that matches the constraint
func (r *Resolver) findMatchingVersion(info *packagist.PackageInfo, constraint string) (string, error) {
	// Clean constraint
	constraint = r.normalizeConstraint(constraint)

	// Parse constraint
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		// If constraint parsing fails, return latest stable
		return r.findLatestStable(info), nil
	}

	// Find matching version
	for version := range info.Versions {
		// Skip dev versions
		if strings.Contains(version, "dev") {
			continue
		}

		// Normalize version
		v, err := semver.NewVersion(r.normalizeVersion(version))
		if err != nil {
			continue
		}

		// Check if version matches constraint
		if c.Check(v) {
			return version, nil
		}
	}

	return "", fmt.Errorf("no version matches constraint: %s", constraint)
}

// findLatestStable finds the latest stable version
func (r *Resolver) findLatestStable(info *packagist.PackageInfo) string {
	var latest string
	var latestSemver *semver.Version

	for version := range info.Versions {
		// Skip dev versions
		if strings.Contains(version, "dev") {
			continue
		}

		v, err := semver.NewVersion(r.normalizeVersion(version))
		if err != nil {
			continue
		}

		if latestSemver == nil || v.GreaterThan(latestSemver) {
			latestSemver = v
			latest = version
		}
	}

	if latest == "" {
		// Fallback to any version
		for version := range info.Versions {
			return version
		}
	}

	return latest
}

// normalizeConstraint normalizes version constraints for semver
func (r *Resolver) normalizeConstraint(constraint string) string {
	constraint = strings.TrimSpace(constraint)

	// Handle common Composer constraints
	constraint = strings.ReplaceAll(constraint, "~", "~")
	constraint = strings.ReplaceAll(constraint, "^", "^")

	// Convert || to ,
	constraint = strings.ReplaceAll(constraint, "||", ",")

	return constraint
}

// normalizeVersion normalizes version strings for semver
func (r *Resolver) normalizeVersion(version string) string {
	// Remove 'v' prefix
	version = strings.TrimPrefix(version, "v")

	// Handle version suffixes
	version = strings.ReplaceAll(version, "-dev", "-alpha")

	return version
}

// isPlatformPackage checks if a package is a platform requirement
func (r *Resolver) isPlatformPackage(name string) bool {
	return strings.HasPrefix(name, "php") ||
		strings.HasPrefix(name, "ext-") ||
		strings.HasPrefix(name, "lib-")
}

// BuildDependencyTree builds a visual dependency tree for a package
func (r *Resolver) BuildDependencyTree(composer *parser.ComposerJSON, targetPackage string) (string, error) {
	var tree strings.Builder

	// Check if directly required
	if version, ok := composer.Require[targetPackage]; ok {
		tree.WriteString(fmt.Sprintf("Your project\n"))
		tree.WriteString(fmt.Sprintf("  └─ %s (%s)\n", targetPackage, version))
		return tree.String(), nil
	}

	// Search in dependencies
	tree.WriteString(fmt.Sprintf("Your project\n"))
	found := r.searchInDependencies(composer, targetPackage, &tree, "  ", composer.Require)

	if !found {
		return "", fmt.Errorf("package %s not found in dependency tree", targetPackage)
	}

	return tree.String(), nil
}

// searchInDependencies recursively searches for a package in dependencies
func (r *Resolver) searchInDependencies(composer *parser.ComposerJSON, target string, tree *strings.Builder, indent string, deps map[string]string) bool {
	for pkg, version := range deps {
		if r.isPlatformPackage(pkg) {
			continue
		}

		info, err := r.client.GetPackage(pkg)
		if err != nil {
			continue
		}

		versionInfo, err := r.client.GetVersion(pkg, r.findLatestStable(info))
		if err != nil {
			continue
		}

		if _, ok := versionInfo.Require[target]; ok {
			tree.WriteString(fmt.Sprintf("%s└─ %s (%s)\n", indent, pkg, version))
			tree.WriteString(fmt.Sprintf("%s    └─ %s\n", indent, target))
			return true
		}

		// Recurse
		if r.searchInDependencies(composer, target, tree, indent+"    ", versionInfo.Require) {
			tree.WriteString(fmt.Sprintf("%s└─ %s (%s)\n", indent, pkg, version))
			return true
		}
	}

	return false
}

// CheckConflicts checks for conflicts when installing a package
func (r *Resolver) CheckConflicts(composer *parser.ComposerJSON, packageName, version string) ([]string, error) {
	var conflicts []string

	// Get package info
	versionInfo, err := r.client.GetVersion(packageName, version)
	if err != nil {
		return nil, err
	}

	// Check PHP version requirement
	if phpVersion, ok := versionInfo.Require["php"]; ok {
		conflicts = append(conflicts, fmt.Sprintf("Requires PHP %s (check your version)", phpVersion))
	}

	// Check extension requirements
	for req := range versionInfo.Require {
		if strings.HasPrefix(req, "ext-") {
			conflicts = append(conflicts, fmt.Sprintf("Requires PHP extension: %s", req))
		}
	}

	// Check conflicts with existing packages
	for existingPkg, existingVersion := range composer.Require {
		if r.isPlatformPackage(existingPkg) {
			continue
		}

		if requiredVersion, ok := versionInfo.Require[existingPkg]; ok {
			// Check if existing version matches requirement
			if !r.versionsCompatible(existingVersion, requiredVersion) {
				conflicts = append(conflicts, fmt.Sprintf("%s requires %s %s (you have %s)", packageName, existingPkg, requiredVersion, existingVersion))
			}
		}
	}

	return conflicts, nil
}

// versionsCompatible checks if two version constraints are compatible
func (r *Resolver) versionsCompatible(v1, v2 string) bool {
	// Simplified compatibility check
	// In production, this should use proper semver constraint checking
	return true
}
