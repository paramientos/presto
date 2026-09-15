package resolver

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aras/presto/internal/packagist"
	"github.com/aras/presto/internal/parser"
)

// Registry is the slice of the packagist client the resolver needs, so the
// solver can be exercised without a network.
type Registry interface {
	GetPackage(name string) (*packagist.PackageInfo, error)
	GetVersion(name, version string) (*packagist.VersionInfo, error)
	DownloadPackage(name, version string) (string, error)
}

type Resolver struct {
	client    Registry
	onPackage func(name string)
	log       func(format string, args ...interface{})
}

type Package struct {
	Name     string
	Version  string
	URL      string
	Require  map[string]string
	Autoload json.RawMessage
	IsDev    bool
}

func NewResolver(client Registry) *Resolver {
	return &Resolver{client: client}
}

// OnPackage reports each package as resolution reaches it.
func (r *Resolver) OnPackage(fn func(name string)) {
	r.onPackage = fn
}

// Log receives the backtracking detail that is too noisy for normal output.
func (r *Resolver) Log(fn func(format string, args ...interface{})) {
	r.log = fn
}

func (r *Resolver) logf(format string, args ...interface{}) {
	if r.log != nil {
		r.log(format, args...)
	}
}

func (r *Resolver) Resolve(composer *parser.ComposerJSON) ([]*Package, error) {
	r.prefetch(composer)

	solved, err := r.solve(composer)
	if err != nil {
		return nil, err
	}

	return r.collect(composer, solved)
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

func (r *Resolver) isPlatformPackage(name string) bool {
	if name == "composer-plugin-api" || name == "composer-runtime-api" {
		return true
	}

	if strings.Contains(name, "/") {
		return strings.HasSuffix(name, "-implementation")
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
		fmt.Fprintf(&tree, "  └─ %s (%s)\n", targetPackage, version)
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

		versionInfo, err := r.client.GetVersion(pkg, r.latestStable(info))
		if err != nil {
			continue
		}

		if _, ok := versionInfo.Require[target]; ok {
			fmt.Fprintf(tree, "%s└─ %s (%s)\n", indent, pkg, version)
			fmt.Fprintf(tree, "%s    └─ %s\n", indent, target)
			return true
		}

		if r.searchInDependencies(composer, target, tree, indent+"    ", versionInfo.Require) {
			fmt.Fprintf(tree, "%s└─ %s (%s)\n", indent, pkg, version)
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
