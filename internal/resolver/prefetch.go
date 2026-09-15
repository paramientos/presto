package resolver

import (
	"sync"

	"github.com/aras/presto/internal/packagist"
	"github.com/aras/presto/internal/parser"
)

// prefetch walks the dependency graph breadth-first and warms the client's
// manifest cache in parallel, so the resolver below finds everything in memory.
// It only decides what to fetch early: a wrong guess costs one spare manifest,
// never a different resolution.
func (r *Resolver) prefetch(composer *parser.ComposerJSON) {
	level := make(map[string]string)

	for _, deps := range []map[string]string{composer.Require, composer.RequireDev} {
		for name, constraint := range deps {
			if !r.isPlatformPackage(name) {
				level[name] = constraint
			}
		}
	}

	seen := make(map[string]bool, len(level))

	for len(level) > 0 {
		for name := range level {
			seen[name] = true
		}

		next := r.fetchLevel(level)

		level = make(map[string]string, len(next))

		for name, constraint := range next {
			if !seen[name] {
				level[name] = constraint
			}
		}
	}
}

// fetchLevel resolves one breadth-first level in parallel and returns the
// dependencies of the versions it picked.
func (r *Resolver) fetchLevel(level map[string]string) map[string]string {
	type job struct {
		name       string
		constraint string
	}

	jobs := make(chan job)
	next := make(map[string]string)

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	for i := 0; i < packagist.MaxConnections; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := range jobs {
				info, err := r.client.GetPackage(j.name)
				if err != nil {
					continue
				}

				if r.onPackage != nil {
					r.onPackage(j.name)
				}

				version, err := r.bestVersion(info, []requirement{{by: "prefetch", constraint: j.constraint}})
				if err != nil {
					continue
				}

				versionInfo, ok := info.Versions[version]
				if !ok {
					continue
				}

				mu.Lock()
				for dep, constraint := range versionInfo.Require {
					if !r.isPlatformPackage(dep) {
						next[dep] = constraint
					}
				}
				mu.Unlock()
			}
		}()
	}

	for name, constraint := range level {
		jobs <- job{name: name, constraint: constraint}
	}

	close(jobs)
	wg.Wait()

	return next
}
