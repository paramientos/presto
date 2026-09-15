package resolver

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aras/presto/internal/packagist"
	version "github.com/shyim/go-version"
)

// requirement records who asked for what, so an impossible set can name its cause.
type requirement struct {
	by         string
	constraint string
}

// bestVersion returns the highest stable release satisfying every requirement at
// once. Taking them one at a time is what let an older constraint be forgotten.
func (r *Resolver) bestVersion(info *packagist.PackageInfo, reqs []requirement) (string, error) {
	return r.bestVersionExcluding(info, reqs, nil)
}

// bestVersionExcluding also honours the conflict ranges other packages declared
// against this one, which is how a framework keeps a dependency off a major it
// cannot work with.
func (r *Resolver) bestVersionExcluding(info *packagist.PackageInfo, reqs, conflicts []requirement) (string, error) {
	required := r.parseConstraints(reqs)
	forbidden := r.parseConstraints(conflicts)

	var (
		best *version.Version
		raw  string
	)

	for _, candidate := range sortedVersions(info) {
		parsed, err := version.NewVersion(candidate)
		if err != nil {
			continue
		}

		if !satisfiesAll(parsed, required) || matchesAny(parsed, forbidden) {
			continue
		}

		if best == nil || parsed.GreaterThan(best) {
			best, raw = parsed, candidate
		}
	}

	if raw == "" {
		return "", unsatisfiable(info.Name, reqs, conflicts)
	}

	return raw, nil
}

func (r *Resolver) parseConstraints(reqs []requirement) []version.Constraints {
	parsed := make([]version.Constraints, 0, len(reqs))

	for _, req := range reqs {
		c, err := version.NewConstraint(req.constraint)
		if err != nil {
			r.logf("ignoring unreadable constraint %q from %s: %v", req.constraint, req.by, err)
			continue
		}

		parsed = append(parsed, c)
	}

	return parsed
}

func matchesAny(v *version.Version, constraints []version.Constraints) bool {
	for _, c := range constraints {
		if c.Check(v) {
			return true
		}
	}

	return false
}

func (r *Resolver) latestStable(info *packagist.PackageInfo) string {
	raw, err := r.bestVersion(info, nil)
	if err != nil {
		return ""
	}

	return raw
}

func satisfiesAll(v *version.Version, constraints []version.Constraints) bool {
	for _, c := range constraints {
		if !c.Check(v) {
			return false
		}
	}

	return true
}

// sortedVersions keeps resolution independent of Go's map ordering.
func sortedVersions(info *packagist.PackageInfo) []string {
	versions := make([]string, 0, len(info.Versions))

	for candidate := range info.Versions {
		if version.Stability(candidate) != "stable" {
			continue
		}

		versions = append(versions, candidate)
	}

	sort.Strings(versions)

	return versions
}

func unsatisfiable(name string, reqs, conflicts []requirement) error {
	var b strings.Builder

	fmt.Fprintf(&b, "no released version of %s satisfies every requirement:", name)

	for _, req := range sortRequirements(reqs) {
		fmt.Fprintf(&b, "\n  %s requires %s", req.by, req.constraint)
	}

	for _, req := range sortRequirements(conflicts) {
		fmt.Fprintf(&b, "\n  %s conflicts with %s", req.by, req.constraint)
	}

	return errors.New(b.String())
}

func sortRequirements(reqs []requirement) []requirement {
	sorted := make([]requirement, len(reqs))
	copy(sorted, reqs)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].by == sorted[j].by {
			return sorted[i].constraint < sorted[j].constraint
		}

		return sorted[i].by < sorted[j].by
	})

	return sorted
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))

	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
