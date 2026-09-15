package resolver

import (
	"strings"

	"github.com/aras/presto/internal/parser"
)

// pending is one unresolved requirement waiting in the queue. An empty
// constraint asks for a fresh decision without adding a requirement, which is
// how a package already picked reacts to a conflict declared later.
type pending struct {
	name       string
	constraint string
	by         string
	dev        bool
}

// solver holds every constraint seen for a package and the version currently
// picked for it. A pick only ever narrows, which is why the loop terminates.
type solver struct {
	resolver  *Resolver
	reqs      map[string][]requirement
	conflicts map[string][]requirement
	picked    map[string]string
	dev       map[string]bool
	declared  map[string]bool
}

func (r *Resolver) solve(composer *parser.ComposerJSON) (*solver, error) {
	s := &solver{
		resolver:  r,
		reqs:      make(map[string][]requirement),
		conflicts: make(map[string][]requirement),
		picked:    make(map[string]string),
		dev:       make(map[string]bool),
		declared:  make(map[string]bool),
	}

	queue := rootRequirements(composer)

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if r.isPlatformPackage(item.name) {
			continue
		}

		changed, err := s.require(item)
		if err != nil {
			return nil, err
		}

		if !changed {
			continue
		}

		recheck, err := s.declareConflicts(item.name)
		if err != nil {
			return nil, err
		}

		deps, err := s.dependenciesOf(item.name, item.dev)
		if err != nil {
			return nil, err
		}

		queue = append(queue, deps...)
		queue = append(queue, recheck...)
	}

	return s, nil
}

// require records one constraint and reports whether it moved the pick, which is
// the only reason to walk that package's dependencies again.
func (s *solver) require(item pending) (bool, error) {
	if item.constraint != "" {
		s.reqs[item.name] = append(s.reqs[item.name], requirement{by: item.by, constraint: item.constraint})
	}

	if wasDev, seen := s.dev[item.name]; !seen || (wasDev && !item.dev) {
		s.dev[item.name] = item.dev
	}

	info, err := s.resolver.client.GetPackage(item.name)
	if err != nil {
		return false, err
	}

	if s.resolver.onPackage != nil {
		s.resolver.onPackage(item.name)
	}

	best, err := s.resolver.bestVersionExcluding(info, s.reqs[item.name], s.conflicts[item.name])
	if err != nil {
		return false, err
	}

	if s.picked[item.name] == best {
		return false, nil
	}

	if previous := s.picked[item.name]; previous != "" {
		s.resolver.logf("%s %s narrowed to %s by %s (%s)", item.name, previous, best, item.by, item.constraint)
	}

	s.picked[item.name] = best

	return true, nil
}

// declareConflicts records what the newly picked version refuses to sit beside,
// and asks for a fresh decision on any package that is already picked.
func (s *solver) declareConflicts(name string) ([]pending, error) {
	stamp := name + " " + s.picked[name]
	if s.declared[stamp] {
		return nil, nil
	}

	s.declared[stamp] = true

	info, err := s.resolver.client.GetVersion(name, s.picked[name])
	if err != nil {
		return nil, err
	}

	var recheck []pending

	for _, target := range sortedKeys(info.Conflict) {
		if s.resolver.isPlatformPackage(target) {
			continue
		}

		s.conflicts[target] = append(s.conflicts[target], requirement{by: stamp, constraint: info.Conflict[target]})

		if _, picked := s.picked[target]; picked {
			recheck = append(recheck, pending{name: target, by: stamp, dev: s.dev[target]})
		}
	}

	return recheck, nil
}

func (s *solver) dependenciesOf(name string, dev bool) ([]pending, error) {
	info, err := s.resolver.client.GetVersion(name, s.picked[name])
	if err != nil {
		return nil, err
	}

	by := name + " " + s.picked[name]
	deps := make([]pending, 0, len(info.Require))

	for _, dep := range sortedKeys(info.Require) {
		if s.resolver.isPlatformPackage(dep) {
			continue
		}

		deps = append(deps, pending{name: dep, constraint: info.Require[dep], by: by, dev: dev})
	}

	return deps, nil
}

func rootRequirements(composer *parser.ComposerJSON) []pending {
	queue := make([]pending, 0, len(composer.Require)+len(composer.RequireDev))

	for _, name := range sortedKeys(composer.Require) {
		queue = append(queue, pending{name: name, constraint: composer.Require[name], by: "composer.json", dev: false})
	}

	for _, name := range sortedKeys(composer.RequireDev) {
		queue = append(queue, pending{name: name, constraint: composer.RequireDev[name], by: "composer.json require-dev", dev: true})
	}

	return queue
}

// collect walks the solved graph from the roots, so a package that only the
// discarded version of another package needed is left behind.
func (r *Resolver) collect(composer *parser.ComposerJSON, s *solver) ([]*Package, error) {
	seen := make(map[string]bool, len(s.picked))

	var packages []*Package

	var walk func(name string) error

	walk = func(name string) error {
		if seen[name] || r.isPlatformPackage(name) {
			return nil
		}

		seen[name] = true

		picked, ok := s.picked[name]
		if !ok {
			return nil
		}

		info, err := r.client.GetVersion(name, picked)
		if err != nil {
			return err
		}

		url, err := r.client.DownloadPackage(name, picked)
		if err != nil {
			if !strings.Contains(err.Error(), "no download URL found") {
				return err
			}

			url = ""
		}

		for _, dep := range sortedKeys(info.Require) {
			if err := walk(dep); err != nil {
				return err
			}
		}

		if url == "" {
			return nil
		}

		packages = append(packages, &Package{
			Name:     name,
			Version:  picked,
			URL:      url,
			Require:  info.Require,
			Autoload: info.Autoload,
			IsDev:    s.dev[name],
		})

		return nil
	}

	for _, item := range rootRequirements(composer) {
		if err := walk(item.name); err != nil {
			return nil, err
		}
	}

	return packages, nil
}
