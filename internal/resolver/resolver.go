package resolver

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/aras/presto/internal/packagist"
	"github.com/aras/presto/internal/parser"
)

type Resolver struct {
	client   *packagist.Client
	resolved map[string]string
	visited  map[string]bool
}

type Package struct {
	Name     string
	Version  string
	URL      string
	Require  map[string]string
	Autoload json.RawMessage
	IsDev    bool
}

func NewResolver(client *packagist.Client) *Resolver {
	return &Resolver{
		client:   client,
		resolved: make(map[string]string),
		visited:  make(map[string]bool),
	}
}

func (r *Resolver) Resolve(composer *parser.ComposerJSON) ([]*Package, error) {
	var packages []*Package

	for name, constraint := range composer.Require {
		if r.isPlatformPackage(name) {
			continue
		}

		if err := r.resolveDependency(name, constraint, false, &packages); err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", name, err)
		}
	}

	for name, constraint := range composer.RequireDev {
		if r.isPlatformPackage(name) {
			continue
		}

		if err := r.resolveDependency(name, constraint, true, &packages); err != nil {
			return nil, fmt.Errorf("failed to resolve dev dependency %s: %w", name, err)
		}
	}

	return packages, nil
}

func (r *Resolver) ResolveFromLock(lock *parser.ComposerLock) ([]*Package, error) {
	var packages []*Package

	for _, lp := range lock.Packages {
		autoloadJSON, _ := json.Marshal(lp.Autoload)

		pkg := &Package{
			Name:     lp.Name,
			Version:  lp.Version,
			URL:      lp.Dist.URL,
			Require:  lp.Require,
			Autoload: autoloadJSON,
			IsDev:    false,
		}
		packages = append(packages, pkg)
	}

	for _, lp := range lock.PackagesDev {
		autoloadJSON, _ := json.Marshal(lp.Autoload)

		pkg := &Package{
			Name:     lp.Name,
			Version:  lp.Version,
			URL:      lp.Dist.URL,
			Require:  lp.Require,
			Autoload: autoloadJSON,
			IsDev:    true,
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (r *Resolver) resolveDependency(name, constraint string, isDev bool, packages *[]*Package) error {
	if r.visited[name] {
		if resolvedVersion, ok := r.resolved[name]; ok {
			c, err := semver.NewConstraint(r.normalizeConstraint(constraint))
			if err == nil {
				v, err := semver.NewVersion(r.normalizeVersion(resolvedVersion))
				if err == nil {
					if !c.Check(v) {
						fmt.Printf("⚠️  CONFLICT FIX: Package %s v%s does not satisfy '%s'. Re-resolving with new constraint...\n", name, resolvedVersion, constraint)

						for i, pkg := range *packages {
							if pkg.Name == name {
								*packages = append((*packages)[:i], (*packages)[i+1:]...)
								break
							}
						}

						r.visited[name] = false
					} else {
						return nil
					}
				}
			}
		} else {
			return nil
		}
	}

	if r.visited[name] {
		return nil
	}
	r.visited[name] = true

	info, err := r.client.GetPackage(name)
	if err != nil {
		return err
	}

	version, err := r.findMatchingVersion(info, constraint)
	if err != nil {
		return fmt.Errorf("no matching version for %s %s: %w", name, constraint, err)
	}
	versionInfo, err := r.client.GetVersion(name, version)
	if err != nil {
		return err
	}

	downloadURL, err := r.client.DownloadPackage(name, version)
	if err != nil {
		if !strings.Contains(err.Error(), "no download URL found") {
			return err
		}

		downloadURL = ""
	}

	if downloadURL == "" {
		r.resolved[name] = version

		for depName, depConstraint := range versionInfo.Require {
			if r.isPlatformPackage(depName) {
				continue
			}

			if err := r.resolveDependency(depName, depConstraint, isDev, packages); err != nil {
				return err
			}
		}

		return nil
	}

	r.resolved[name] = version

	for depName, depConstraint := range versionInfo.Require {
		if r.isPlatformPackage(depName) {
			continue
		}

		if err := r.resolveDependency(depName, depConstraint, isDev, packages); err != nil {
			return err
		}
	}

	pkg := &Package{
		Name:     name,
		Version:  version,
		URL:      downloadURL,
		Require:  versionInfo.Require,
		Autoload: versionInfo.Autoload,
		IsDev:    isDev,
	}
	*packages = append(*packages, pkg)

	return nil
}

func (r *Resolver) findMatchingVersion(info *packagist.PackageInfo, constraint string) (string, error) {
	constraint = r.normalizeConstraint(constraint)
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return r.findLatestStable(info), nil
	}

	var bestVersion string
	var bestSemver *semver.Version
	var bestFourth int // tiebreaker for four-part versions

	for version := range info.Versions {
		if strings.Contains(version, "dev") {
			continue
		}

		normalized := r.normalizeVersion(version)
		v, err := semver.NewVersion(normalized)
		if err != nil {
			continue
		}
		if c.Check(v) {
			fourth := fourthSegment(version)
			if bestSemver == nil || v.GreaterThan(bestSemver) ||
				(v.Equal(bestSemver) && fourth > bestFourth) {
				bestSemver = v
				bestVersion = version
				bestFourth = fourth
			}
		}
	}

	if bestVersion != "" {
		return bestVersion, nil
	}

	return "", fmt.Errorf("no version matches constraint: %s", constraint)
}

func (r *Resolver) findLatestStable(info *packagist.PackageInfo) string {
	var latest string
	var latestSemver *semver.Version
	var latestFourth int

	for version := range info.Versions {
		if strings.Contains(version, "dev") {
			continue
		}

		v, err := semver.NewVersion(r.normalizeVersion(version))
		if err != nil {
			continue
		}
		if v.Prerelease() != "" {
			continue
		}
		fourth := fourthSegment(version)
		if latestSemver == nil || v.GreaterThan(latestSemver) ||
			(v.Equal(latestSemver) && fourth > latestFourth) {
			latestSemver = v
			latest = version
			latestFourth = fourth
		}
	}

	if latest == "" {
		for version := range info.Versions {
			return version
		}
	}

	return latest
}

// fourthSegment extracts the numeric fourth version segment from a Composer
// four-part version string (e.g. "v9.18.1.10" → 10). Returns 0 if absent.
func fourthSegment(version string) int {
	version = strings.TrimPrefix(version, "v")
	parts := strings.SplitN(version, ".", 5)
	if len(parts) != 4 {
		return 0
	}
	// Only parse if purely numeric (no pre-release suffix)
	if strings.ContainsAny(parts[3], "-+") {
		return 0
	}
	n := 0
	for _, ch := range parts[3] {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func (r *Resolver) normalizeConstraint(constraint string) string {
	constraint = strings.TrimSpace(constraint)

	constraint = strings.ReplaceAll(constraint, " ", "")

	constraint = strings.ReplaceAll(constraint, "~", "~")
	constraint = strings.ReplaceAll(constraint, "^", "^")

	return constraint
}

func (r *Resolver) normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "v")

	version = strings.ReplaceAll(version, "-dev", "-alpha")

	// Composer allows four-part versions like 9.18.1.10 (MAJOR.MINOR.PATCH.BUILD).
	// The Masterminds semver library only understands three parts, so we strip the
	// fourth segment before parsing. Constraints are written against the first three
	// parts (e.g. ^9.18), so this is safe and matches Composer's own behaviour.
	if parts := strings.SplitN(version, ".", 5); len(parts) == 4 {
		// Only truncate when the fourth part is purely numeric (no pre-release suffix).
		// If there's a pre-release tag it will be attached to the third segment already.
		if !strings.ContainsAny(parts[3], "-+") {
			version = strings.Join(parts[:3], ".")
		}
	}

	return version
}

func (r *Resolver) isPlatformPackage(name string) bool {
	if name == "composer-plugin-api" || name == "composer-runtime-api" {
		return true
	}

	if strings.Contains(name, "/") {
		if strings.HasSuffix(name, "-implementation") {
			return true
		}
		return false
	}

	return name == "php" ||
		strings.HasPrefix(name, "php-") || // php-64bit, etc.
		strings.HasPrefix(name, "ext-") ||
		strings.HasPrefix(name, "lib-")
}

func (r *Resolver) BuildDependencyTree(composer *parser.ComposerJSON, targetPackage string) (string, error) {
	var tree strings.Builder

	if version, ok := composer.Require[targetPackage]; ok {
		tree.WriteString("Your project\n")
		tree.WriteString(fmt.Sprintf("  └─ %s (%s)\n", targetPackage, version))
		return tree.String(), nil
	}

	tree.WriteString("Your project\n")
	found := r.searchInDependencies(composer, targetPackage, &tree, "  ", composer.Require)

	if !found {
		return "", fmt.Errorf("package %s not found in dependency tree", targetPackage)
	}

	return tree.String(), nil
}

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

		if r.searchInDependencies(composer, target, tree, indent+"    ", versionInfo.Require) {
			tree.WriteString(fmt.Sprintf("%s└─ %s (%s)\n", indent, pkg, version))
			return true
		}
	}

	return false
}

func (r *Resolver) CheckConflicts(composer *parser.ComposerJSON, packageName, version string) ([]string, error) {
	var conflicts []string

	versionInfo, err := r.client.GetVersion(packageName, version)
	if err != nil {
		return nil, err
	}

	if phpVersion, ok := versionInfo.Require["php"]; ok {
		conflicts = append(conflicts, fmt.Sprintf("Requires PHP %s (check your version)", phpVersion))
	}

	for req := range versionInfo.Require {
		if strings.HasPrefix(req, "ext-") {
			conflicts = append(conflicts, fmt.Sprintf("Requires PHP extension: %s", req))
		}
	}

	for existingPkg, existingVersion := range composer.Require {
		if r.isPlatformPackage(existingPkg) {
			continue
		}

		if requiredVersion, ok := versionInfo.Require[existingPkg]; ok {
			if !r.versionsCompatible(existingVersion, requiredVersion) {
				conflicts = append(conflicts, fmt.Sprintf("%s requires %s %s (you have %s)", packageName, existingPkg, requiredVersion, existingVersion))
			}
		}
	}

	return conflicts, nil
}

func (r *Resolver) versionsCompatible(v1, v2 string) bool {
	return true
}
