package discover

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/hzerrad/cronkit/internal/source"
)

// newRootFS builds the sandboxed fs.FS Walk reads through (os.Root, not os.DirFS); a var for tests.
var newRootFS = func(dir string) (fs.FS, func() error, error) {
	rt, err := os.OpenRoot(dir)
	if err != nil {
		return nil, nil, err
	}
	return rt.FS(), rt.Close, nil
}

// Options configures a Walker: which files Enumerate selects, and which sources run.
type Options struct {
	EnumerateOptions

	// Sources restricts a walk to these source IDs; an unknown ID errors, not silently empties.
	Sources []string

	// ExcludeSources drops these source IDs from whichever set Sources already selected.
	ExcludeSources []string
}

// Walker joins Enumerate's candidate file list to a source registry, folding the results into one inventory.
type Walker struct {
	registry *source.Registry
	opts     Options
}

// New builds a Walker from registry and opts; Sources/ExcludeSources are validated on first Walk.
func New(registry *source.Registry, opts Options) *Walker {
	return &Walker{registry: registry, opts: opts}
}

// Walk enumerates each root, decodes each matched candidate once, and returns one sorted inventory.
func (w *Walker) Walk(roots []string) (*inventory.Inventory, []Problem, error) {
	registry, err := w.filteredRegistry()
	if err != nil {
		return nil, nil, err
	}

	var (
		items    []inventory.Item
		problems []Problem
		invRoot  string
	)
	seenScopes := make(map[string]bool, len(roots))
	// seenPaths stops a file rediscovered by overlapping entries in roots from being reported twice.
	seenPaths := make(map[string]bool)

	for _, start := range roots {
		root, err := FindRoot(start)
		if err != nil {
			return nil, nil, err
		}
		scope, err := resolveScope(start)
		if err != nil {
			return nil, nil, err
		}

		scopeKey := root.Dir + "\x00" + scope
		if seenScopes[scopeKey] {
			continue
		}
		seenScopes[scopeKey] = true
		if invRoot == "" {
			invRoot = root.Dir
		}

		candidates, rootProblems, err := Enumerate(root, scope, w.opts.EnumerateOptions)
		if err != nil {
			return nil, nil, err
		}
		for _, p := range rootProblems {
			pathKey := root.Dir + "\x00" + p.Path
			if seenPaths[pathKey] {
				continue
			}
			seenPaths[pathKey] = true
			problems = append(problems, p)
		}

		fsys, closeFS, err := newRootFS(root.Dir)
		if err != nil {
			return nil, nil, fmt.Errorf("discover: open %q: %w", root.Dir, err)
		}
		for _, c := range candidates {
			pathKey := root.Dir + "\x00" + c.Path
			if seenPaths[pathKey] {
				continue
			}
			seenPaths[pathKey] = true

			// One Unit and Cache per candidate lets multiple matching sources share a single decode.
			unit := source.Unit{Path: c.Path, Info: c.Info, Cache: source.NewDocumentCache()}
			// Sources sharing that decode also share its failure, so an identical error is
			// reported once. A source failing differently still gets its own problem.
			seenErrs := make(map[string]bool)
			for _, s := range registry.MatchAll(unit) {
				extracted, err := s.Extract(unit, fsys)
				if err != nil {
					if seenErrs[err.Error()] {
						continue
					}
					seenErrs[err.Error()] = true
					problems = append(problems, Problem{
						Path: c.Path,
						Err:  fmt.Errorf("discover: %s: %w", s.ID(), err),
					})
					continue
				}
				items = append(items, extracted...)
			}
		}
		if err := closeFS(); err != nil {
			return nil, nil, fmt.Errorf("discover: close %q: %w", root.Dir, err)
		}
	}

	sort.SliceStable(problems, func(i, j int) bool { return problems[i].Path < problems[j].Path })

	inv := inventory.New(invRoot, items)
	inv.Sort()

	return inv, problems, nil
}

// resolveScope resolves start to an absolute, symlink-free path, the same way FindRoot does internally.
func resolveScope(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("discover: resolve %q: %w", start, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("discover: resolve %q: %w", start, err)
	}
	return resolved, nil
}

// filteredRegistry resolves Sources and ExcludeSources into the registry a walk should run.
func (w *Walker) filteredRegistry() (*source.Registry, error) {
	all := w.registry.Sources()

	known := make(map[string]bool, len(all))
	for _, s := range all {
		known[s.ID()] = true
	}
	for _, id := range w.opts.Sources {
		if !known[id] {
			return nil, fmt.Errorf("discover: unknown source %q", id)
		}
	}
	for _, id := range w.opts.ExcludeSources {
		if !known[id] {
			return nil, fmt.Errorf("discover: unknown source %q", id)
		}
	}

	var selected []source.Source
	for _, s := range all {
		if w.included(s.ID()) {
			selected = append(selected, s)
		}
	}
	return source.NewRegistry(selected...), nil
}

// included reports whether id survives both Options.Sources and Options.ExcludeSources.
func (w *Walker) included(id string) bool {
	if len(w.opts.Sources) > 0 && !containsString(w.opts.Sources, id) {
		return false
	}
	return !containsString(w.opts.ExcludeSources, id)
}

// containsString reports whether want appears in list.
func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
