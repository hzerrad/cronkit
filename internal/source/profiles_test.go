package source

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mixedDir is the corpus directory name that isn't a profile ID; its cases span multiple profiles.
const mixedDir = "mixed"

// TestConformance runs every corpus case through the same Default registry a real scan uses.
func TestConformance(t *testing.T) {
	registry, err := Default()
	require.NoError(t, err)

	// known lists every corpus directory expected to exist, read from the registry; mixedDir is exempt.
	known := make(map[string]bool, len(registry.Sources()))
	for _, s := range registry.Sources() {
		known[s.ID()] = true
	}

	root := filepath.Join("..", "..", "testdata", "sources")
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.NotEmpty(t, entries, "the corpus must not be empty")

	visited := make(map[string]bool, len(known))

	for _, sourceDir := range entries {
		if !sourceDir.IsDir() {
			// A stray file directly under testdata/sources, such as a
			// .gitkeep or a README, names no corpus and is not an error.
			continue
		}
		name := sourceDir.Name()
		if name != mixedDir {
			require.True(t, known[name], "corpus directory %q names no registered source", name)
			visited[name] = true
		}

		cases, err := os.ReadDir(filepath.Join(root, name))
		require.NoError(t, err)
		require.NotEmpty(t, cases)

		for _, c := range cases {
			if !c.IsDir() {
				// A stray file sitting next to the case directories, such as a per-source README, is not a case.
				continue
			}
			t.Run(name+"/"+c.Name(), func(t *testing.T) {
				dir := filepath.Join(root, name, c.Name())
				assertCorpusCase(t, registry, dir)
			})
		}
	}

	// Mirrors the check above: reports every source still missing a corpus dir, not just the first.
	var missing []string
	for id := range known {
		if !visited[id] {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	assert.Empty(t, missing, "sources with no corpus directory under testdata/sources: %v", missing)
}

func assertCorpusCase(t *testing.T, registry *Registry, dir string) {
	t.Helper()

	fsys := os.DirFS(dir)
	names, err := findInputFile(fsys)
	require.NoError(t, err)
	require.Len(t, names, 1, "each case holds exactly one input file")

	info, err := fs.Stat(fsys, names[0])
	require.NoError(t, err)

	unit := Unit{Path: names[0], Info: info}
	sources := registry.MatchAll(unit)
	require.NotEmpty(t, sources, "the input must match at least one source")

	var got []inventory.Item
	for _, src := range sources {
		items, err := src.Extract(unit, fsys)
		require.NoError(t, err)
		got = append(got, items...)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "expected.json"))
	require.NoError(t, err)

	var want []inventory.Item
	require.NoError(t, json.Unmarshal(raw, &want))

	// "[]" in expected.json unmarshals to non-nil, while Extract returns nil; both mean the same thing.
	assert.Equal(t, nonNilItems(want), nonNilItems(got))
}

// nonNilItems normalises a nil slice to an empty one, so "no items" compares
// equal regardless of which side produced the nil.
func nonNilItems(items []inventory.Item) []inventory.Item {
	if items == nil {
		return []inventory.Item{}
	}
	return items
}

// findInputFile locates the input.* file wherever it sits, since a DirPrefix profile nests its fixture.
func findInputFile(fsys fs.FS) ([]string, error) {
	var names []string
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if matched, _ := filepath.Match("input.*", d.Name()); matched {
			names = append(names, p)
		}
		return nil
	})
	return names, err
}

func TestDefault_RegistersEveryBuiltInProfile(t *testing.T) {
	registry, err := Default()
	require.NoError(t, err)

	var ids []string
	for _, s := range registry.Sources() {
		ids = append(ids, s.ID())
	}
	assert.Equal(t, []string{"crontab", "k8s", "argo", "gha"}, ids,
		"registration order is part of the contract, since it decides the order results come back in")
}

func TestBuiltInProfiles_AreValid(t *testing.T) {
	for _, p := range []Profile{K8sCronJob, ArgoCronWorkflow, GitHubActions} {
		t.Run(p.ID, func(t *testing.T) {
			_, err := NewProfileSource(p)
			assert.NoError(t, err)
		})
	}
}

func TestGitHubActions_OnlyMatchesWorkflowDirectory(t *testing.T) {
	src, err := NewProfileSource(GitHubActions)
	require.NoError(t, err)

	assert.True(t, src.Match(Unit{Path: ".github/workflows/ci.yml"}))
	assert.False(t, src.Match(Unit{Path: "docs/example.yml"}),
		"a workflow-shaped file elsewhere is not a workflow")
}
