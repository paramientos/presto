package resolver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aras/presto/internal/packagist"
	"github.com/aras/presto/internal/parser"
)

// fakeRegistry maps package name to version to that version's requirements.
type fakeRegistry map[string]map[string]map[string]string

func (f fakeRegistry) GetPackage(name string) (*packagist.PackageInfo, error) {
	versions, ok := f[name]
	if !ok {
		return nil, fmt.Errorf("package not found: %s", name)
	}

	info := &packagist.PackageInfo{Name: name, Versions: map[string]*packagist.VersionInfo{}}

	for release, require := range versions {
		info.Versions[release] = &packagist.VersionInfo{
			Name:     name,
			Version:  release,
			Require:  require,
			Conflict: conflicts[name+"@"+release],
			Dist:     packagist.DistInfo{Type: "zip", URL: "https://example.test/" + name + "/" + release + ".zip"},
		}
	}

	return info, nil
}

// conflicts carries the conflict block for a fake package version, keyed
// name@version, so the registry literal stays readable.
var conflicts = map[string]map[string]string{}

func (f fakeRegistry) GetVersion(name, release string) (*packagist.VersionInfo, error) {
	info, err := f.GetPackage(name)
	if err != nil {
		return nil, err
	}

	versionInfo, ok := info.Versions[release]
	if !ok {
		return nil, fmt.Errorf("version %s not found for %s", release, name)
	}

	return versionInfo, nil
}

func (f fakeRegistry) DownloadPackage(name, release string) (string, error) {
	versionInfo, err := f.GetVersion(name, release)
	if err != nil {
		return "", err
	}

	return versionInfo.Dist.URL, nil
}

func resolved(t *testing.T, packages []*Package) map[string]string {
	t.Helper()

	out := make(map[string]string, len(packages))
	for _, pkg := range packages {
		out[pkg.Name] = pkg.Version
	}

	return out
}

func TestIsPlatformPackage(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"php", true},
		{"ext-json", true},
		{"lib-curl", true},
		{"composer-plugin-api", true},
		{"symfony/console", false},
	}

	r := NewResolver(fakeRegistry{})

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := r.isPlatformPackage(tt.input); got != tt.expected {
				t.Errorf("isPlatformPackage(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// The tilde operator is where Composer and npm disagree: Composer reads ~1.0 as
// >=1.0 <2.0, npm as >=1.0 <1.1. Reading it the npm way pinned an ancient
// polyfill in real installs.
func TestTheTildeOperatorFollowsComposerNotNpm(t *testing.T) {
	r := NewResolver(fakeRegistry{})

	info := &packagist.PackageInfo{
		Name: "acme/polyfill",
		Versions: map[string]*packagist.VersionInfo{
			"v1.0.1": {}, "v1.8.0": {}, "v1.38.2": {},
		},
	}

	got, err := r.bestVersion(info, []requirement{{by: "a", constraint: "~1.0"}})
	if err != nil {
		t.Fatal(err)
	}

	if got != "v1.38.2" {
		t.Fatalf("expected ~1.0 to allow v1.38.2, got %s", got)
	}
}

func TestAVersionMustSatisfyEveryRequirementAtOnce(t *testing.T) {
	r := NewResolver(fakeRegistry{})

	info := &packagist.PackageInfo{
		Name: "acme/lib",
		Versions: map[string]*packagist.VersionInfo{
			"1.1.0": {}, "1.3.0": {}, "1.4.0": {}, "1.9.0": {},
		},
	}

	got, err := r.bestVersion(info, []requirement{
		{by: "a", constraint: ">=1.2"},
		{by: "b", constraint: "<=1.4"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got != "1.4.0" {
		t.Fatalf("expected the intersection of both constraints, got %s", got)
	}
}

func TestFourPartVersionsMatchTheirConstraint(t *testing.T) {
	r := NewResolver(fakeRegistry{})

	info := &packagist.PackageInfo{
		Name: "scrivo/highlight.php",
		Versions: map[string]*packagist.VersionInfo{
			"v9.18.1.10": {}, "v9.18.1.4": {}, "v9.12.0.0": {}, "v9.17.1.0": {},
		},
	}

	for _, constraint := range []string{"^9.18", "^9.12", "~9.18.1", ">=9.17"} {
		t.Run(constraint, func(t *testing.T) {
			got, err := r.bestVersion(info, []requirement{{by: "root", constraint: constraint}})
			if err != nil {
				t.Fatal(err)
			}

			if got != "v9.18.1.10" {
				t.Errorf("bestVersion(%q) = %q, want v9.18.1.10", constraint, got)
			}
		})
	}
}

func TestAnImpossibleSetOfConstraintsIsReportedNotQuietlyResolved(t *testing.T) {
	registry := fakeRegistry{
		"acme/low":  {"1.0.0": {"acme/lib": "<=1.4"}},
		"acme/high": {"1.0.0": {"acme/lib": ">=1.6"}},
		"acme/lib":  {"1.4.0": nil, "1.9.0": nil},
	}

	composer := &parser.ComposerJSON{
		Require: map[string]string{"acme/low": "^1.0", "acme/high": "^1.0"},
	}

	_, err := NewResolver(registry).Resolve(composer)
	if err == nil {
		t.Fatal("expected contradictory constraints to fail rather than pick a version that breaks one")
	}

	for _, want := range []string{"acme/lib", "<=1.4", ">=1.6"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected the error to mention %q, got: %v", want, err)
		}
	}
}

func TestALaterRequirementNarrowsAnEarlierPick(t *testing.T) {
	registry := fakeRegistry{
		"acme/app": {"1.0.0": {"acme/lib": "^1.0"}},
		"acme/pin": {"1.0.0": {"acme/lib": "<=1.4"}},
		"acme/lib": {"1.4.0": nil, "1.9.0": nil},
	}

	composer := &parser.ComposerJSON{
		Require: map[string]string{"acme/app": "^1.0", "acme/pin": "^1.0"},
	}

	packages, err := NewResolver(registry).Resolve(composer)
	if err != nil {
		t.Fatal(err)
	}

	if got := resolved(t, packages)["acme/lib"]; got != "1.4.0" {
		t.Fatalf("expected the pin to hold acme/lib at 1.4.0, got %s", got)
	}
}

func TestAPackageOnlyTheDiscardedVersionNeededIsLeftOut(t *testing.T) {
	registry := fakeRegistry{
		"acme/app":   {"1.4.0": nil, "1.9.0": {"acme/extra": "^1.0"}},
		"acme/pin":   {"1.0.0": {"acme/app": "<=1.4"}},
		"acme/extra": {"1.0.0": nil},
	}

	composer := &parser.ComposerJSON{
		Require: map[string]string{"acme/app": "^1.0", "acme/pin": "^1.0"},
	}

	packages, err := NewResolver(registry).Resolve(composer)
	if err != nil {
		t.Fatal(err)
	}

	got := resolved(t, packages)

	if got["acme/app"] != "1.4.0" {
		t.Fatalf("expected acme/app narrowed to 1.4.0, got %s", got["acme/app"])
	}

	if _, present := got["acme/extra"]; present {
		t.Fatal("expected the dependency of the discarded version to be left out")
	}
}

func TestResolutionDoesNotDependOnMapOrder(t *testing.T) {
	registry := fakeRegistry{
		"acme/app":   {"1.0.0": {"acme/lib": "^1.0", "acme/other": "^1.0"}},
		"acme/pin":   {"1.0.0": {"acme/lib": "<=1.4"}},
		"acme/lib":   {"1.4.0": nil, "1.9.0": nil},
		"acme/othr":  {"1.0.0": nil},
		"acme/other": {"1.0.0": {"acme/lib": "^1.0"}},
	}

	composer := &parser.ComposerJSON{
		Require: map[string]string{"acme/app": "^1.0", "acme/pin": "^1.0"},
	}

	var first string

	for i := 0; i < 25; i++ {
		packages, err := NewResolver(registry).Resolve(composer)
		if err != nil {
			t.Fatal(err)
		}

		var b strings.Builder
		for _, pkg := range packages {
			fmt.Fprintf(&b, "%s@%s ", pkg.Name, pkg.Version)
		}

		if i == 0 {
			first = b.String()
			continue
		}

		if b.String() != first {
			t.Fatalf("run %d resolved differently:\n  %s\n  %s", i, first, b.String())
		}
	}
}

func TestDevDependenciesAreFlagged(t *testing.T) {
	registry := fakeRegistry{
		"acme/app":    {"1.0.0": nil},
		"acme/test":   {"1.0.0": {"acme/helper": "^1.0"}},
		"acme/helper": {"1.0.0": nil},
	}

	composer := &parser.ComposerJSON{
		Require:    map[string]string{"acme/app": "^1.0"},
		RequireDev: map[string]string{"acme/test": "^1.0"},
	}

	packages, err := NewResolver(registry).Resolve(composer)
	if err != nil {
		t.Fatal(err)
	}

	for _, pkg := range packages {
		wantDev := pkg.Name != "acme/app"

		if pkg.IsDev != wantDev {
			t.Errorf("%s: IsDev = %v, want %v", pkg.Name, pkg.IsDev, wantDev)
		}
	}
}

func TestAConflictKeepsADependencyOffTheForbiddenRange(t *testing.T) {
	conflicts["acme/framework@1.0.0"] = map[string]string{"acme/types": ">=3.0"}
	t.Cleanup(func() { delete(conflicts, "acme/framework@1.0.0") })

	registry := fakeRegistry{
		"acme/framework": {"1.0.0": {"acme/dates": "^1.0"}},
		"acme/dates":     {"1.0.0": {"acme/types": "*"}},
		"acme/types":     {"2.1.1": nil, "3.2.1": nil},
	}

	composer := &parser.ComposerJSON{
		Require: map[string]string{"acme/framework": "^1.0"},
	}

	packages, err := NewResolver(registry).Resolve(composer)
	if err != nil {
		t.Fatal(err)
	}

	if got := resolved(t, packages)["acme/types"]; got != "2.1.1" {
		t.Fatalf("expected the conflict to hold acme/types at 2.1.1, got %s", got)
	}
}

func TestAConflictDeclaredAfterAPickStillTakesEffect(t *testing.T) {
	conflicts["acme/late@1.0.0"] = map[string]string{"acme/types": ">=3.0"}
	t.Cleanup(func() { delete(conflicts, "acme/late@1.0.0") })

	registry := fakeRegistry{
		"acme/early": {"1.0.0": {"acme/types": "*"}},
		"acme/late":  {"1.0.0": nil},
		"acme/types": {"2.1.1": nil, "3.2.1": nil},
	}

	composer := &parser.ComposerJSON{
		Require: map[string]string{"acme/early": "^1.0", "acme/late": "^1.0"},
	}

	packages, err := NewResolver(registry).Resolve(composer)
	if err != nil {
		t.Fatal(err)
	}

	if got := resolved(t, packages)["acme/types"]; got != "2.1.1" {
		t.Fatalf("expected acme/types re-picked as 2.1.1 once the conflict appeared, got %s", got)
	}
}
