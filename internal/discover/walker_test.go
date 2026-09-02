package discover

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/hzerrad/cronkit/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scanFixture copies testdata/scan into a fresh temp directory, marked as a repo root with a bare ".git".
func scanFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "scan")
	require.NoError(t, os.CopyFS(dir, os.DirFS(src)))
	mkdirAll(t, filepath.Join(dir, ".git"))
	return dir
}

// defaultRegistry builds the real, built-in source registry, failing the
// test on error.
func defaultRegistry(t *testing.T) *source.Registry {
	t.Helper()
	reg, err := source.Default()
	require.NoError(t, err)
	return reg
}

// itemsBySource returns the items whose SourceID matches id.
func itemsBySource(items []inventory.Item, id string) []inventory.Item {
	var out []inventory.Item
	for _, it := range items {
		if it.SourceID == id {
			out = append(out, it)
		}
	}
	return out
}

// problemPaths returns the Path of every problem.
func problemPaths(problems []Problem) []string {
	out := make([]string, len(problems))
	for i, p := range problems {
		out[i] = p.Path
	}
	return out
}

func TestWalker_EverySourceItemAppearsWithRootRelativePath(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{})

	inv, _, err := w.Walk([]string{dir})
	require.NoError(t, err)

	assert.Equal(t, dir, inv.Root)

	crontabItems := itemsBySource(inv.Items, "crontab")
	require.Len(t, crontabItems, 1)
	assert.Equal(t, "crontab", crontabItems[0].Locator.File)

	k8sItems := itemsBySource(inv.Items, "k8s")
	require.Len(t, k8sItems, 1)
	assert.Equal(t, "deploy/cronjob.yaml", k8sItems[0].Locator.File)

	argoItems := itemsBySource(inv.Items, "argo")
	require.Len(t, argoItems, 2, "spec.schedules holds two entries")
	for _, it := range argoItems {
		assert.Equal(t, "workflows/cronworkflow.yaml", it.Locator.File)
	}

	ghaItems := itemsBySource(inv.Items, "gha")
	require.Len(t, ghaItems, 2, "on.schedule holds two entries")
	for _, it := range ghaItems {
		assert.Equal(t, ".github/workflows/ci.yml", it.Locator.File)
	}

	// configmap.yaml matches k8s and argo by path (any .yaml) but holds
	// neither kind, so both extract zero items from it, silently.
	for _, it := range inv.Items {
		assert.NotEqual(t, "deploy/configmap.yaml", it.Locator.File)
		assert.NotEqual(t, "notes.txt", it.Locator.File)
	}
}

func TestWalker_DecodesAFileOnceEvenWhenTwoSourcesMatch(t *testing.T) {
	dir := scanFixture(t)

	reads := map[string]int{}
	orig := newRootFS
	newRootFS = func(dir string) (fs.FS, func() error, error) {
		fsys, closeFS, err := orig(dir)
		if err != nil {
			return nil, nil, err
		}
		return &countingFS{FS: fsys, reads: reads}, closeFS, nil
	}
	t.Cleanup(func() { newRootFS = orig })

	w := New(defaultRegistry(t), Options{})
	inv, _, err := w.Walk([]string{dir})
	require.NoError(t, err)
	require.NotEmpty(t, itemsBySource(inv.Items, "k8s"), "sanity check: the file was actually extracted from")

	assert.Equal(t, 1, reads["deploy/cronjob.yaml"],
		"k8s and argo both match this file by path; it must be decoded once via the shared DocumentCache")
}

// countingFS counts Open calls per name, mirroring internal/source's own
// cache test (see countingFS in source_test.go) at the walker level.
type countingFS struct {
	fs.FS
	reads map[string]int
}

func (c *countingFS) Open(name string) (fs.File, error) {
	c.reads[name]++
	return c.FS.Open(name)
}

func TestWalker_ScanningSubdirectoryScopesToItButFramesPathsAgainstTheRoot(t *testing.T) {
	// Scope limits enumeration, but Locator.File is still relative to the repository root, not the scope.
	dir := scanFixture(t)
	sub := filepath.Join(dir, ".github", "workflows")

	w := New(defaultRegistry(t), Options{})
	inv, _, err := w.Walk([]string{sub})
	require.NoError(t, err)

	assert.Equal(t, dir, inv.Root,
		"FindRoot must walk up to the repository root, not stop at the scanned subdirectory")

	ghaItems := itemsBySource(inv.Items, "gha")
	require.Len(t, ghaItems, 2,
		"gha only recognises a file whose parent is exactly .github/workflows; a scope-relative path would break that match")
	for _, it := range ghaItems {
		assert.Equal(t, ".github/workflows/ci.yml", it.Locator.File)
	}

	assert.Empty(t, itemsBySource(inv.Items, "crontab"),
		"scanning .github/workflows must not also report a cron living elsewhere in the repository")
	for _, it := range inv.Items {
		assert.NotEqual(t, "crontab", it.Locator.File)
	}
}

func TestWalker_WalkIsDeterministic(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{})

	inv1, problems1, err := w.Walk([]string{dir})
	require.NoError(t, err)
	inv2, problems2, err := w.Walk([]string{dir})
	require.NoError(t, err)

	b1, err := json.Marshal(inv1)
	require.NoError(t, err)
	b2, err := json.Marshal(inv2)
	require.NoError(t, err)
	assert.Equal(t, b1, b2, "two Walk calls over the same tree must produce byte-identical inventories")

	require.Equal(t, problemPaths(problems1), problemPaths(problems2))
	for i := range problems1 {
		assert.Equal(t, problems1[i].Err.Error(), problems2[i].Err.Error())
	}
}

func TestWalker_SourcesFilterRestrictsToNamedIDs(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{Sources: []string{"crontab"}})

	inv, _, err := w.Walk([]string{dir})
	require.NoError(t, err)

	require.NotEmpty(t, inv.Items)
	for _, it := range inv.Items {
		assert.Equal(t, "crontab", it.SourceID)
	}
}

func TestWalker_ExcludeSourcesDropsNamedIDs(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{ExcludeSources: []string{"crontab"}})

	inv, _, err := w.Walk([]string{dir})
	require.NoError(t, err)

	require.NotEmpty(t, inv.Items)
	for _, it := range inv.Items {
		assert.NotEqual(t, "crontab", it.SourceID)
	}
}

func TestWalker_SourcesAndExcludeSourcesCombine(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{
		Sources:        []string{"crontab", "k8s"},
		ExcludeSources: []string{"k8s"},
	})

	inv, _, err := w.Walk([]string{dir})
	require.NoError(t, err)

	require.NotEmpty(t, inv.Items)
	for _, it := range inv.Items {
		assert.Equal(t, "crontab", it.SourceID)
	}
}

func TestWalker_UnknownSourceIDIsAnError(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{Sources: []string{"nope"}})

	_, _, err := w.Walk([]string{dir})
	assert.Error(t, err, "a typo in --source must not look like \"no results\"")
}

func TestWalker_UnknownExcludeSourceIDIsAnError(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{ExcludeSources: []string{"nope"}})

	_, _, err := w.Walk([]string{dir})
	assert.Error(t, err)
}

func TestWalker_UnknownSourceIDErrorsEvenWithNoRoots(t *testing.T) {
	// Source-list validation must not be contingent on there being
	// anything to walk: it is a configuration error, checked up front.
	w := New(defaultRegistry(t), Options{Sources: []string{"nope"}})

	_, _, err := w.Walk(nil)
	assert.Error(t, err)
}

func TestWalker_ExtractionFailureIsAProblemNotAnAbort(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{})

	inv, problems, err := w.Walk([]string{dir})
	require.NoError(t, err)

	brokenProblems := 0
	for _, p := range problems {
		if p.Path == "deploy/broken.yaml" {
			brokenProblems++
			assert.Error(t, p.Err)
		}
	}
	assert.Equal(t, 1, brokenProblems, "k8s and argo share one decode of broken.yaml, so they share its failure")

	// The walk did not abort: items from files sorted both before and
	// after broken.yaml are still present.
	assert.NotEmpty(t, itemsBySource(inv.Items, "crontab"))
	assert.NotEmpty(t, itemsBySource(inv.Items, "gha"))
}

func TestWalker_UnmatchedFileProducesNoProblemAndNoItems(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{})

	inv, problems, err := w.Walk([]string{dir})
	require.NoError(t, err)

	assert.NotContains(t, problemPaths(problems), "notes.txt")
	for _, it := range inv.Items {
		assert.NotEqual(t, "notes.txt", it.Locator.File)
	}
}

func TestWalker_MatchButNoItemsProducesNoProblem(t *testing.T) {
	// configmap.yaml matches k8s and argo by path but holds neither profile's kind, which is not a Problem.
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{})

	_, problems, err := w.Walk([]string{dir})
	require.NoError(t, err)

	assert.NotContains(t, problemPaths(problems), "deploy/configmap.yaml")
}

func TestWalker_EnumerateOptionsAreForwarded(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{
		EnumerateOptions: EnumerateOptions{Excludes: []string{"notes.txt"}},
	})

	candidates, _, err := Enumerate(newRootForTest(t, dir), dir, EnumerateOptions{Excludes: []string{"notes.txt"}})
	require.NoError(t, err)
	assert.NotContains(t, candidatePaths(candidates), "notes.txt", "sanity check on the exclude pattern itself")

	inv, _, err := w.Walk([]string{dir})
	require.NoError(t, err)
	for _, it := range inv.Items {
		assert.NotEqual(t, "notes.txt", it.Locator.File)
	}
}

// newRootForTest builds a Root directly, for a sanity check that does not
// exercise the Walker itself.
func newRootForTest(t *testing.T, dir string) Root {
	t.Helper()
	root, err := FindRoot(dir)
	require.NoError(t, err)
	return root
}

func TestWalker_InvalidExcludePatternIsAnError(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{EnumerateOptions: EnumerateOptions{Excludes: []string{"["}}})

	_, _, err := w.Walk([]string{dir})
	assert.Error(t, err)
}

func TestWalker_NonexistentRootIsAnError(t *testing.T) {
	w := New(defaultRegistry(t), Options{})

	_, _, err := w.Walk([]string{filepath.Join(t.TempDir(), "does-not-exist")})
	assert.Error(t, err)
}

func TestWalker_OverlappingRootsDoNotDoubleReportAFile(t *testing.T) {
	// "dir" and "dir/deploy" overlap; a file found by both must be reported only once.
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{})

	inv, _, err := w.Walk([]string{dir, filepath.Join(dir, "deploy")})
	require.NoError(t, err)

	k8sItems := itemsBySource(inv.Items, "k8s")
	require.Len(t, k8sItems, 1, "deploy/cronjob.yaml is found by both the whole-repository scope and the narrower deploy scope; it must be reported once")
	assert.Equal(t, "deploy/cronjob.yaml", k8sItems[0].Locator.File)

	// crontab lives outside deploy, so the dedup must not wrongly suppress it too.
	assert.Len(t, itemsBySource(inv.Items, "crontab"), 1)
}

func TestWalker_IdenticalRootGivenTwiceIsNotDoubleCounted(t *testing.T) {
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{})

	inv, _, err := w.Walk([]string{dir, dir})
	require.NoError(t, err)

	assert.Len(t, itemsBySource(inv.Items, "crontab"), 1)
}

func TestWalker_DisjointRootsAreEachScopedAndBothReported(t *testing.T) {
	// Two non-overlapping roots must not pull in anything from outside either of them.
	dir := scanFixture(t)
	w := New(defaultRegistry(t), Options{})

	inv, _, err := w.Walk([]string{filepath.Join(dir, "deploy"), filepath.Join(dir, "workflows")})
	require.NoError(t, err)

	assert.Len(t, itemsBySource(inv.Items, "k8s"), 1, "deploy/cronjob.yaml, from the first root")
	assert.Len(t, itemsBySource(inv.Items, "argo"), 2, "workflows/cronworkflow.yaml's two schedules, from the second root")
	assert.Empty(t, itemsBySource(inv.Items, "crontab"), "crontab lives outside both requested roots")
	assert.Empty(t, itemsBySource(inv.Items, "gha"), "the gha workflow lives outside both requested roots")
}

func TestWalker_OverlappingRootsDoNotDoubleReportAClassifyLevelProblem(t *testing.T) {
	// The seenPaths dedup must also gate a raw Problem from Enumerate, not only a successful candidate.
	dir := t.TempDir()
	mkdirAll(t, filepath.Join(dir, ".git"))
	mkdirAll(t, filepath.Join(dir, "sub"))
	writeFile(t, filepath.Join(dir, "sub", "big.txt"), "0123456789")

	w := New(defaultRegistry(t), Options{
		EnumerateOptions: EnumerateOptions{MaxFileSize: 5},
	})

	_, problems, err := w.Walk([]string{dir, filepath.Join(dir, "sub")})
	require.NoError(t, err)

	count := 0
	for _, p := range problems {
		if p.Path == "sub/big.txt" {
			count++
		}
	}
	assert.Equal(t, 1, count, "the oversized-file Problem must be reported once despite being found by both overlapping roots")
}

func TestWalker_EmptyRootsProducesEmptyInventory(t *testing.T) {
	w := New(defaultRegistry(t), Options{})

	inv, problems, err := w.Walk(nil)
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Empty(t, inv.Items)
	assert.Empty(t, inv.Root)
}

func TestWalker_ReadIsSandboxedEvenIfAFileIsSwappedAfterClassification(t *testing.T) {
	// A file swapped for an escaping symlink after classification but before the read must still be refused.
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}

	dir := t.TempDir()
	mkdirAll(t, filepath.Join(dir, ".git"))
	writeFile(t, filepath.Join(dir, "crontab"), "0 2 * * * /usr/bin/backup.sh\n")

	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret"), "REALLY-SECRET-CRON-DATA\n0 3 * * * /bin/should-not-appear\n")

	orig := newRootFS
	newRootFS = func(d string) (fs.FS, func() error, error) {
		// Swap "crontab" for an escaping symlink right before Walk reads it.
		require.NoError(t, os.Remove(filepath.Join(d, "crontab")))
		require.NoError(t, os.Symlink(filepath.Join(outside, "secret"), filepath.Join(d, "crontab")))
		return orig(d)
	}
	t.Cleanup(func() { newRootFS = orig })

	w := New(defaultRegistry(t), Options{})
	inv, problems, err := w.Walk([]string{dir})
	require.NoError(t, err)

	assert.Empty(t, inv.Items, "the swapped-in outside file must never be read, so it contributes no items")
	for _, it := range inv.Items {
		assert.NotContains(t, it.Command, "should-not-appear")
	}

	found := false
	for _, p := range problems {
		if p.Path == "crontab" {
			found = true
		}
	}
	assert.True(t, found, "the crontab source matched and attempted to read \"crontab\"; the refusal must surface as a Problem")
}

func TestWalker_NewRootFSErrorAbortsTheWalk(t *testing.T) {
	dir := scanFixture(t)

	orig := newRootFS
	newRootFS = func(string) (fs.FS, func() error, error) {
		return nil, nil, errors.New("boom")
	}
	t.Cleanup(func() { newRootFS = orig })

	w := New(defaultRegistry(t), Options{})
	_, _, err := w.Walk([]string{dir})
	assert.Error(t, err)
}

func TestWalker_CloseRootFSErrorAbortsTheWalk(t *testing.T) {
	dir := scanFixture(t)

	orig := newRootFS
	newRootFS = func(d string) (fs.FS, func() error, error) {
		fsys, _, err := orig(d)
		if err != nil {
			return nil, nil, err
		}
		return fsys, func() error { return errors.New("boom") }, nil
	}
	t.Cleanup(func() { newRootFS = orig })

	w := New(defaultRegistry(t), Options{})
	_, _, err := w.Walk([]string{dir})
	assert.Error(t, err)
}

func TestResolveScope_ResolvesToAnAbsoluteSymlinkFreePath(t *testing.T) {
	dir := t.TempDir()

	got, err := resolveScope(dir)
	require.NoError(t, err)

	resolved, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	assert.Equal(t, resolved, got)
}

func TestResolveScope_NonexistentPathErrors(t *testing.T) {
	_, err := resolveScope(filepath.Join(t.TempDir(), "does-not-exist"))
	assert.Error(t, err)
}

func TestResolveScope_UnresolvableRelativePathErrors(t *testing.T) {
	chdirIntoDeletedDir(t)

	_, err := resolveScope("relative")
	assert.Error(t, err)
}

// TestWalker_ADecodeFailureIsReportedOnce covers a malformed yaml every profile recognises: the
// sources share one decode, so they must not each report the shared failure.
func TestWalker_ADecodeFailureIsReportedOnce(t *testing.T) {
	dir := t.TempDir()
	mkdirAll(t, filepath.Join(dir, ".git"))
	writeFile(t, filepath.Join(dir, "broken.yaml"), "kind: [unclosed\n")

	_, problems, err := New(defaultRegistry(t), Options{}).Walk([]string{dir})
	require.NoError(t, err)

	require.Len(t, problems, 1, "one bad file is one problem, however many sources tried to read it")
	assert.Equal(t, "broken.yaml", problems[0].Path)
}
