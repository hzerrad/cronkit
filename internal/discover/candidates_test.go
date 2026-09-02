package discover

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// candidatePaths extracts the Path field of each candidate.
func candidatePaths(candidates []Candidate) []string {
	paths := make([]string, len(candidates))
	for i, c := range candidates {
		paths[i] = c.Path
	}
	return paths
}

// newRoot builds a Root directly from dir, bypassing FindRoot, resolving symlinks the same way FindRoot does.
func newRoot(t *testing.T, dir string, isRepo bool) Root {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	return Root{Dir: resolved, IsRepo: isRepo}
}

func TestEnumerate_DeterministicOrdering(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "zebra.yml"), "z")
	writeFile(t, filepath.Join(dir, "alpha.yml"), "a")
	mkdirAll(t, filepath.Join(dir, "mid"))
	writeFile(t, filepath.Join(dir, "mid", "middle.yml"), "m")

	root := newRoot(t, dir, false)

	first, problems1, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems1)

	second, problems2, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems2)

	assert.Equal(t, candidatePaths(first), candidatePaths(second))
	assert.True(t, sort.StringsAreSorted(candidatePaths(first)))
	assert.Equal(t, []string{"alpha.yml", "mid/middle.yml", "zebra.yml"}, candidatePaths(first))
}

func TestEnumerate_ScopeLimitsTheFallbackWalkButPathsStayRootRelative(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "top.txt"), "top")
	mkdirAll(t, filepath.Join(dir, "sub"))
	writeFile(t, filepath.Join(dir, "sub", "inner.txt"), "inner")

	root := newRoot(t, dir, false)

	candidates, problems, err := Enumerate(root, filepath.Join(dir, "sub"), EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)

	assert.Equal(t, []string{"sub/inner.txt"}, candidatePaths(candidates),
		"scoping to sub must both exclude top.txt (outside scope) and report sub/inner.txt with its full root-relative path, not just \"inner.txt\"")
}

func TestEnumerate_ScopeLimitsGitListingButPathsStayRootRelative(t *testing.T) {
	gitPath := requireGit(t)

	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	runGit("init", "-q")

	writeFile(t, filepath.Join(dir, "top.txt"), "top")
	mkdirAll(t, filepath.Join(dir, "sub"))
	writeFile(t, filepath.Join(dir, "sub", "tracked.txt"), "tracked")
	runGit("add", "-A")
	runGit("-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "base")
	writeFile(t, filepath.Join(dir, "sub", "untracked.txt"), "untracked")

	root := newRoot(t, dir, true)

	candidates, problems, err := Enumerate(root, filepath.Join(dir, "sub"), EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)

	paths := candidatePaths(candidates)
	assert.Equal(t, []string{"sub/tracked.txt", "sub/untracked.txt"}, paths,
		"git-backed scoping must cover both tracked and untracked files under sub, with full root-relative paths, and nothing from outside sub")
}

// scopeLiteralGitFixture builds a git repository with a single tracked file at <dirName>/cron.yaml.
func scopeLiteralGitFixture(t *testing.T, dirName string) (root Root, dir string) {
	t.Helper()
	gitPath := requireGit(t)

	dir = t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	runGit("init", "-q")

	scopeDir := filepath.Join(dir, dirName)
	require.NoError(t, os.MkdirAll(scopeDir, 0o755))
	writeFile(t, filepath.Join(scopeDir, "cron.yaml"), "kind: CronJob\n")
	runGit("add", "-A")
	runGit("-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "base")

	return newRoot(t, dir, true), dir
}

func TestEnumerate_ScopeDirectoryStartingWithColonIsTreatedLiterally(t *testing.T) {
	// Without ":(literal)", git would read a scope named ":weird" as pathspec magic and report zero candidates.
	root, dir := scopeLiteralGitFixture(t, ":weird")

	candidates, problems, err := Enumerate(root, filepath.Join(dir, ":weird"), EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{":weird/cron.yaml"}, candidatePaths(candidates))
}

func TestEnumerate_ScopeDirectoryStartingWithDoubleDashIsTreatedLiterally(t *testing.T) {
	// A scope directory named "--not-an-option" must not be read as a
	// flag either, on top of not being read as pathspec magic.
	root, dir := scopeLiteralGitFixture(t, "--not-an-option")

	candidates, problems, err := Enumerate(root, filepath.Join(dir, "--not-an-option"), EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"--not-an-option/cron.yaml"}, candidatePaths(candidates))
}

func TestEnumerate_ScopeDirectoryWithGlobMetacharactersStillMatches(t *testing.T) {
	// ":(literal)" must not regress a scope directory containing a glob metacharacter (*, ?, [).
	root, dir := scopeLiteralGitFixture(t, "sub*dir[1]")

	candidates, problems, err := Enumerate(root, filepath.Join(dir, "sub*dir[1]"), EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"sub*dir[1]/cron.yaml"}, candidatePaths(candidates))
}

func TestEnumerate_ScopeOutsideRootIsAnError(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "root")
	mkdirAll(t, dir)
	root := newRoot(t, dir, false)

	outside := filepath.Join(base, "outside")
	mkdirAll(t, outside)

	_, _, err := Enumerate(root, outside, EnumerateOptions{})
	assert.Error(t, err)
}

func TestEnumerate_ScopeIsRootItselfEnumeratesEverything(t *testing.T) {
	// scope == root.Dir must behave exactly like an unrestricted scan --
	// this is the "." pathspec case for the git-backed path.
	gitPath := requireGit(t)

	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	cmd := exec.Command(gitPath, "init", "-q")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git init: %s", out)

	writeFile(t, filepath.Join(dir, "top.txt"), "top")
	mkdirAll(t, filepath.Join(dir, "sub"))
	writeFile(t, filepath.Join(dir, "sub", "inner.txt"), "inner")

	root := newRoot(t, dir, true)

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"sub/inner.txt", "top.txt"}, candidatePaths(candidates))
}

func TestEnumerate_FileScopeReturnsThatOneFileOnTheFallbackPath(t *testing.T) {
	// Root.IsRepo is false, so this always takes the fallback walk
	// regardless of whether git is installed.
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "top.txt"), "top")
	mkdirAll(t, filepath.Join(dir, "sub"))
	writeFile(t, filepath.Join(dir, "sub", "cron.yaml"), "kind: CronJob\n")
	writeFile(t, filepath.Join(dir, "sub", "other.yaml"), "kind: CronJob\n")

	root := newRoot(t, dir, false)

	candidates, problems, err := Enumerate(root, filepath.Join(dir, "sub", "cron.yaml"), EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"sub/cron.yaml"}, candidatePaths(candidates),
		"a scope naming a single file must select exactly that file, not zero files and not its containing directory")
}

func TestEnumerate_FileScopeReturnsThatOneFileOnTheGitPath(t *testing.T) {
	// This must agree with the fallback path's behaviour above exactly.
	gitPath := requireGit(t)

	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	runGit("init", "-q")

	mkdirAll(t, filepath.Join(dir, "sub"))
	writeFile(t, filepath.Join(dir, "sub", "tracked.yaml"), "kind: CronJob\n")
	writeFile(t, filepath.Join(dir, "sub", "other.yaml"), "kind: CronJob\n")
	runGit("add", "-A")
	runGit("-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "base")
	writeFile(t, filepath.Join(dir, "sub", "untracked.yaml"), "kind: CronJob\n")

	root := newRoot(t, dir, true)

	trackedCandidates, problems, err := Enumerate(root, filepath.Join(dir, "sub", "tracked.yaml"), EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"sub/tracked.yaml"}, candidatePaths(trackedCandidates))

	untrackedCandidates, problems, err := Enumerate(root, filepath.Join(dir, "sub", "untracked.yaml"), EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"sub/untracked.yaml"}, candidatePaths(untrackedCandidates))
}

func TestEnumerate_FallbackSkipsSkipList(t *testing.T) {
	dir := t.TempDir()
	// Root.IsRepo is false, so this always takes the fallback walk
	// regardless of whether git is installed.
	root := newRoot(t, dir, false)

	for _, skipped := range []string{".git", "vendor", "node_modules"} {
		mkdirAll(t, filepath.Join(dir, skipped))
		writeFile(t, filepath.Join(dir, skipped, "inside.txt"), "should not appear")
	}
	writeFile(t, filepath.Join(dir, "keep.txt"), "kept")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"keep.txt"}, candidatePaths(candidates))
}

func TestEnumerate_FallbackSkipsAGitFileTooNotJustADirectory(t *testing.T) {
	// A worktree or submodule checkout can record its root with a ".git" file rather than a directory.
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	writeFile(t, filepath.Join(dir, ".git"), "gitdir: /elsewhere/.git/worktrees/example\n")
	writeFile(t, filepath.Join(dir, "keep.txt"), "kept")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"keep.txt"}, candidatePaths(candidates))
}

func TestEnumerate_OversizedFileIsExcludedAndReported(t *testing.T) {
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	writeFile(t, filepath.Join(dir, "big.txt"), "0123456789")
	writeFile(t, filepath.Join(dir, "small.txt"), "ok")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{MaxFileSize: 5})
	require.NoError(t, err)

	assert.Equal(t, []string{"small.txt"}, candidatePaths(candidates))
	require.Len(t, problems, 1)
	assert.Equal(t, "big.txt", problems[0].Path)
	assert.Error(t, problems[0].Err)
}

func TestEnumerate_FileExactlyAtMaxFileSizeIsIncluded(t *testing.T) {
	// Pins the boundary: the size check is "over the limit", so a file
	// exactly at MaxFileSize must be included, not excluded.
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	writeFile(t, filepath.Join(dir, "exact.txt"), "0123456789")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{MaxFileSize: 10})
	require.NoError(t, err)

	assert.Empty(t, problems)
	assert.Equal(t, []string{"exact.txt"}, candidatePaths(candidates))
}

func TestEnumerate_BinaryFileExcludedSilently(t *testing.T) {
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "binary.dat"), []byte("abc\x00def"), 0o644))
	// Not valid UTF-8, but no NUL byte within the sniff window: this must
	// not be treated as binary.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notutf8.txt"), []byte{0xff, 0xfe, 'a', 'b'}, 0o644))

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"notutf8.txt"}, candidatePaths(candidates))
}

func TestEnumerate_DirectoriesAndOutsideSymlinksExcludedInsideSymlinksFollowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}

	base := t.TempDir()
	dir := filepath.Join(base, "root")
	mkdirAll(t, dir)
	root := newRoot(t, dir, false)

	// A plain subdirectory must not itself appear as a candidate.
	mkdirAll(t, filepath.Join(dir, "subdir"))
	writeFile(t, filepath.Join(dir, "subdir", "real.txt"), "hello")

	// A relative symlink pointing inside the root is followed and included.
	require.NoError(t, os.Symlink(filepath.Join("subdir", "real.txt"), filepath.Join(dir, "inside-link.txt")))

	// A symlink pointing outside the root is a containment failure and must be excluded.
	outsideTarget := filepath.Join(base, "outside.txt")
	writeFile(t, outsideTarget, "should never appear")
	require.NoError(t, os.Symlink(outsideTarget, filepath.Join(dir, "outside-link.txt")))

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)

	paths := candidatePaths(candidates)
	assert.Contains(t, paths, "subdir/real.txt")
	assert.Contains(t, paths, "inside-link.txt")
	assert.NotContains(t, paths, "outside-link.txt", "a symlink escaping the root must never be included")
	assert.Len(t, paths, 2)
}

func TestEnumerate_AbsoluteSymlinkExcludedEvenWhenItPointsInsideRoot(t *testing.T) {
	// os.Root refuses an absolute symlink outright, regardless of where it resolves.
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}

	dir := t.TempDir()
	root := newRoot(t, dir, false)

	writeFile(t, filepath.Join(dir, "real.txt"), "hello")
	require.NoError(t, os.Symlink(filepath.Join(dir, "real.txt"), filepath.Join(dir, "abs-link.txt")))

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)

	paths := candidatePaths(candidates)
	assert.Contains(t, paths, "real.txt")
	assert.NotContains(t, paths, "abs-link.txt")
}

func TestEnumerate_IrregularFileExcludedSilently(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFOs are not available on windows")
	}

	dir := t.TempDir()
	root := newRoot(t, dir, false)

	writeFile(t, filepath.Join(dir, "regular.txt"), "hi")
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "fifo"), 0o600))

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"regular.txt"}, candidatePaths(candidates))
}

func TestEnumerate_UnreadableDirectoryProducesProblemAndContinues(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not restrict directory search on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores permission bits")
	}

	dir := t.TempDir()
	root := newRoot(t, dir, false)

	blocked := filepath.Join(dir, "blocked")
	mkdirAll(t, blocked)
	writeFile(t, filepath.Join(blocked, "secret.txt"), "unreachable")
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	require.NoError(t, os.Chmod(blocked, 0))

	writeFile(t, filepath.Join(dir, "visible.txt"), "still here")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)

	assert.Equal(t, []string{"visible.txt"}, candidatePaths(candidates))
	require.Len(t, problems, 1)
	assert.Equal(t, "blocked", problems[0].Path)
	assert.Error(t, problems[0].Err)
}

func TestEnumerate_ExcludesRemoveMatchingPaths(t *testing.T) {
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	writeFile(t, filepath.Join(dir, "keep.txt"), "a")
	writeFile(t, filepath.Join(dir, "drop.log"), "b")
	mkdirAll(t, filepath.Join(dir, "nested"))
	writeFile(t, filepath.Join(dir, "nested", "also.log"), "c")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{Excludes: []string{"*.log"}})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"keep.txt"}, candidatePaths(candidates))
}

func TestEnumerate_ExcludeWithoutSlashMatchesADirectorySegmentAtAnyDepth(t *testing.T) {
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	mkdirAll(t, filepath.Join(dir, "thirdparty"))
	writeFile(t, filepath.Join(dir, "thirdparty", "v.cron"), "0 * * * * /bin/true\n")
	mkdirAll(t, filepath.Join(dir, "thirdparty", "nested"))
	writeFile(t, filepath.Join(dir, "thirdparty", "nested", "deep.cron"), "0 * * * * /bin/true\n")
	writeFile(t, filepath.Join(dir, "keep.cron"), "0 * * * * /bin/true\n")

	candidates, _, err := Enumerate(root, root.Dir, EnumerateOptions{Excludes: []string{"thirdparty"}})
	require.NoError(t, err)
	assert.Equal(t, []string{"keep.cron"}, candidatePaths(candidates),
		"a slash-free pattern must exclude the named directory and everything beneath it, at any depth, not just a same-named file")
}

func TestEnumerate_ExcludeWithSlashMatchesOnlyTheFullPath(t *testing.T) {
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	mkdirAll(t, filepath.Join(dir, "build"))
	writeFile(t, filepath.Join(dir, "build", "generated"), "not a schedule")
	mkdirAll(t, filepath.Join(dir, "other", "build"))
	writeFile(t, filepath.Join(dir, "other", "build", "generated"), "not a schedule either")
	writeFile(t, filepath.Join(dir, "keep.txt"), "a")

	candidates, _, err := Enumerate(root, root.Dir, EnumerateOptions{Excludes: []string{"build/generated"}})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"keep.txt", "other/build/generated"}, candidatePaths(candidates),
		"a slashed pattern matches only that exact root-relative path, not the same path rooted elsewhere")
}

func TestEnumerate_InvalidExcludePatternIsAnError(t *testing.T) {
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	_, _, err := Enumerate(root, root.Dir, EnumerateOptions{Excludes: []string{"["}})
	assert.Error(t, err)
}

// requireGit skips the test, with an explicit message, when no git binary
// is on PATH.
func requireGit(t *testing.T) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git binary not found on PATH; skipping git-backed comparison")
	}
	return gitPath
}

func TestEnumerate_GitAndFallbackAgree(t *testing.T) {
	gitPath := requireGit(t)

	dir := t.TempDir()
	// Isolate from the invoking user's global git config so nothing in
	// their ~/.gitconfig excludes files unexpectedly.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	runGit("init", "-q")

	writeFile(t, filepath.Join(dir, "tracked.yml"), "on: {}\n")
	mkdirAll(t, filepath.Join(dir, ".github", "workflows"))
	writeFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"), "on: {}\n")
	writeFile(t, filepath.Join(dir, "untracked.yml"), "on: {}\n")

	root := newRoot(t, dir, true)

	gitResult, gitProblems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, gitProblems)

	fallbackResult, fallbackProblems, err := Enumerate(root, root.Dir, EnumerateOptions{NoIgnore: true})
	require.NoError(t, err)
	assert.Empty(t, fallbackProblems)

	assert.Equal(t, candidatePaths(fallbackResult), candidatePaths(gitResult))
	assert.Equal(t, []string{".github/workflows/ci.yml", "tracked.yml", "untracked.yml"}, candidatePaths(gitResult))
}

func TestEnumerate_GitAndFallbackAgreeOnExcludes(t *testing.T) {
	gitPath := requireGit(t)

	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	runGit("init", "-q")

	mkdirAll(t, filepath.Join(dir, "vendor"))
	writeFile(t, filepath.Join(dir, "vendor", "dep.cron"), "0 * * * * /bin/true\n")
	writeFile(t, filepath.Join(dir, "notes.bak"), "scratch")
	mkdirAll(t, filepath.Join(dir, "build"))
	writeFile(t, filepath.Join(dir, "build", "generated"), "not a schedule")
	writeFile(t, filepath.Join(dir, "keep.cron"), "0 * * * * /bin/true\n")

	root := newRoot(t, dir, true)
	opts := EnumerateOptions{Excludes: []string{"vendor", "*.bak", "build/generated"}}

	gitResult, gitProblems, err := Enumerate(root, root.Dir, opts)
	require.NoError(t, err)
	assert.Empty(t, gitProblems)

	fallbackOpts := opts
	fallbackOpts.NoIgnore = true
	fallbackResult, fallbackProblems, err := Enumerate(root, root.Dir, fallbackOpts)
	require.NoError(t, err)
	assert.Empty(t, fallbackProblems)

	assert.Equal(t, candidatePaths(fallbackResult), candidatePaths(gitResult),
		"the three exclude pattern shapes must be applied identically regardless of which enumeration path produced the raw candidates")
	assert.Equal(t, []string{"keep.cron"}, candidatePaths(gitResult))
}

func TestGitEnumerate_DoesNotRunTheScannedRepositoriesFsmonitorHook(t *testing.T) {
	// Reproduces a real exploit: a scanned repository's own core.fsmonitor config must never be executed.
	if runtime.GOOS == "windows" {
		t.Skip("the fsmonitor hook script below requires a unix shell")
	}
	gitPath := requireGit(t)

	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	runGit("init", "-q")

	marker := filepath.Join(dir, "PWNED")
	script := "#!/bin/sh\ntouch " + marker + "\n" +
		// An empty NUL-length-prefixed reply satisfies the fsmonitor hook protocol.
		"printf '\\0\\0\\0\\0'\n"
	writeFile(t, filepath.Join(dir, "evil.sh"), script)
	require.NoError(t, os.Chmod(filepath.Join(dir, "evil.sh"), 0o755))
	runGit("config", "core.fsmonitor", "./evil.sh")

	writeFile(t, filepath.Join(dir, "present.txt"), "x")

	root := newRoot(t, dir, true)
	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Contains(t, candidatePaths(candidates), "present.txt")

	_, statErr := os.Stat(marker)
	assert.True(t, os.IsNotExist(statErr),
		"core.fsmonitor from the scanned repository's own config must never be executed")
}

func TestFilterEnv_NeverReturnsNil(t *testing.T) {
	// exec.Cmd treats a nil Env as "inherit everything", so filterEnv must return non-nil even when empty.
	got := filterEnv([]string{"UNRELATED_VAR=x", "ANOTHER_ONE=y"})

	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestGitEnumerate_FallsBackWhenGitMissing(t *testing.T) {
	dir := t.TempDir()
	// An empty PATH directory guarantees exec.LookPath("git") fails,
	// regardless of whether the host actually has git installed.
	t.Setenv("PATH", t.TempDir())

	root := newRoot(t, dir, true)
	writeFile(t, filepath.Join(dir, "present.txt"), "x")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"present.txt"}, candidatePaths(candidates))
}

// writeFakeGit installs an executable named "git" that runs script, and
// returns the directory it lives in so a test can point PATH at it.
func writeFakeGit(t *testing.T, script string) string {
	t.Helper()
	binDir := t.TempDir()
	path := filepath.Join(binDir, "git")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return binDir
}

func TestGitEnumerate_FallsBackWhenGitExitsNonZero(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell-script git binary requires a unix shell")
	}

	dir := t.TempDir()
	t.Setenv("PATH", writeFakeGit(t, "#!/bin/sh\nexit 1\n"))

	root := newRoot(t, dir, true)
	writeFile(t, filepath.Join(dir, "present.txt"), "x")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"present.txt"}, candidatePaths(candidates))
}

func TestGitEnumerate_FallsBackWhenTheSecondInvocationFails(t *testing.T) {
	// A failure in either of the two git invocations must fall back to the filesystem walk.
	if runtime.GOOS == "windows" {
		t.Skip("fake shell-script git binary requires a unix shell")
	}

	dir := t.TempDir()
	// Succeeds for the --cached call, fails for the --others call.
	t.Setenv("PATH", writeFakeGit(t, `#!/bin/sh
case "$*" in
  *--others*) exit 1 ;;
  *) exit 0 ;;
esac
`))

	root := newRoot(t, dir, true)
	writeFile(t, filepath.Join(dir, "present.txt"), "x")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"present.txt"}, candidatePaths(candidates))
}

func TestGitEnumerate_FallsBackWhenGitTimesOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell-script git binary requires a unix shell")
	}
	sleepPath, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep binary on PATH; skipping git-timeout test")
	}

	orig := gitTimeout
	gitTimeout = 50 * time.Millisecond
	t.Cleanup(func() { gitTimeout = orig })

	dir := t.TempDir()
	// sleep is named by its absolute path since PATH below is restricted to just the fake git.
	t.Setenv("PATH", writeFakeGit(t, "#!/bin/sh\n"+sleepPath+" 3\n"))

	root := newRoot(t, dir, true)
	writeFile(t, filepath.Join(dir, "present.txt"), "x")

	start := time.Now()
	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	elapsed := time.Since(start)
	t.Logf("Enumerate returned after %s (gitTimeout=%s, fake git sleeps 3s)", elapsed, gitTimeout)

	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"present.txt"}, candidatePaths(candidates))
	assert.Lessf(t, elapsed, 1*time.Second,
		"Enumerate took %s to return; gitTimeout (50ms) should have bounded the wait well under the fake git's 3s sleep", elapsed)
}

func TestEnumerate_MultipleProblemsAreSortedByPath(t *testing.T) {
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	writeFile(t, filepath.Join(dir, "zzz-big.txt"), "0123456789")
	writeFile(t, filepath.Join(dir, "aaa-big.txt"), "0123456789")

	_, problems, err := Enumerate(root, root.Dir, EnumerateOptions{MaxFileSize: 5})
	require.NoError(t, err)

	require.Len(t, problems, 2)
	assert.Equal(t, []string{"aaa-big.txt", "zzz-big.txt"}, []string{problems[0].Path, problems[1].Path})
}

func TestEnumerate_EmptyGitRepoProducesNoCandidates(t *testing.T) {
	gitPath := requireGit(t)

	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	cmd := exec.Command(gitPath, "init", "-q")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git init: %s", out)

	root := newRoot(t, dir, true)

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, candidates)
	assert.Empty(t, problems)
}

func TestSplitNulPaths_DeduplicatesRepeatedEntries(t *testing.T) {
	// An unmerged path has one index entry per conflict stage; splitNulPaths must collapse that to one.
	out := []byte("a.txt\x00f.txt\x00f.txt\x00f.txt\x00z.txt\x00")

	got := splitNulPaths(out)

	assert.Equal(t, []string{"a.txt", "f.txt", "z.txt"}, got)
}

func TestEnumerate_ConflictedPathListedOnce(t *testing.T) {
	// Reproduces a real unresolved merge conflict, which without dedup would report the file three times.
	gitPath := requireGit(t)

	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	// A failing merge is the point of this sequence, so its own exit
	// status is checked separately rather than through runGit.
	mergeGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		_ = cmd.Run()
	}

	runGit("init", "-q")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	runGit("checkout", "-q", "-b", "main")

	writeFile(t, filepath.Join(dir, "f.txt"), "base\n")
	runGit("add", "-A")
	runGit("commit", "-q", "-m", "base")

	runGit("checkout", "-q", "-b", "other")
	writeFile(t, filepath.Join(dir, "f.txt"), "other\n")
	runGit("commit", "-q", "-am", "other")

	runGit("checkout", "-q", "main")
	writeFile(t, filepath.Join(dir, "f.txt"), "main\n")
	runGit("commit", "-q", "-am", "main")

	mergeGit("merge", "-q", "--no-edit", "other")

	// Confirm the conflict actually landed multiple stages in the index
	// before trusting Enumerate's answer about it.
	stageCmd := exec.Command(gitPath, "ls-files", "--stage", "f.txt")
	stageCmd.Dir = dir
	stageOut, err := stageCmd.Output()
	require.NoError(t, err)
	require.GreaterOrEqualf(t, len(bytes.Split(bytes.TrimSpace(stageOut), []byte("\n"))), 2,
		"test setup did not produce an unmerged path: %s", stageOut)

	root := newRoot(t, dir, true)

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"f.txt"}, candidatePaths(candidates))
}

func TestEnumerate_GitResultOfZeroFilesIsNotTreatedAsUnconsulted(t *testing.T) {
	// git legitimately returning no files must not be mistaken for "git was not consulted".
	gitPath := requireGit(t)

	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	runGit("init", "-q")

	writeFile(t, filepath.Join(dir, ".gitignore"), "*\n")
	writeFile(t, filepath.Join(dir, "secret.txt"), "should stay hidden")

	root := newRoot(t, dir, true)

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Empty(t, candidatePaths(candidates), "git says everything is ignored; the fallback walk must not override that")

	fallback, _, err := Enumerate(root, root.Dir, EnumerateOptions{NoIgnore: true})
	require.NoError(t, err)
	assert.NotEmpty(t, candidatePaths(fallback), "sanity check: the fallback walk does see the ignored files when asked to")
}

func TestEnumerate_GitIndexEntryUnderSymlinkedDirectoryDoesNotEscapeRoot(t *testing.T) {
	// Commits a/secret.yaml, then replaces "a" on disk with an escaping symlink, to prove os.Root catches it.
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}
	gitPath := requireGit(t)

	base := t.TempDir()
	dir := filepath.Join(base, "repo")
	mkdirAll(t, filepath.Join(dir, "a"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	writeFile(t, filepath.Join(dir, "a", "secret.yaml"), "kind: CronJob\n")
	runGit("init", "-q")
	runGit("add", "-A")
	runGit("-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "base")

	outside := filepath.Join(base, "outside")
	mkdirAll(t, outside)
	writeFile(t, filepath.Join(outside, "secret.yaml"), "REALLY-SECRET-CRON-DATA")

	require.NoError(t, os.RemoveAll(filepath.Join(dir, "a")))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "a")))

	root := newRoot(t, dir, true)
	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.NotContains(t, candidatePaths(candidates), "a/secret.yaml",
		"a git index entry under a directory later replaced by a symlink must not become a candidate")

	// Reading through the same sandboxed FS Walk uses must not reach the outside file either.
	rt, err := os.OpenRoot(root.Dir)
	require.NoError(t, err)
	defer func() { _ = rt.Close() }()
	data, err := rt.ReadFile("a/secret.yaml")
	assert.Error(t, err)
	assert.NotContains(t, string(data), "REALLY-SECRET-CRON-DATA")
}

func TestEnumerate_TrackedSubmoduleContentsAreVisible(t *testing.T) {
	// Without --recurse-submodules, a submodule's tracked contents would be invisible to git ls-files.
	gitPath := requireGit(t)

	base := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	subDir := filepath.Join(base, "sub")
	mkdirAll(t, subDir)
	runGitIn := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	writeFile(t, filepath.Join(subDir, "cron.yaml"), "kind: CronJob\n")
	runGitIn(subDir, "init", "-q")
	runGitIn(subDir, "add", "-A")
	runGitIn(subDir, "-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "sub")

	topDir := filepath.Join(base, "top")
	mkdirAll(t, topDir)
	writeFile(t, filepath.Join(topDir, "top.txt"), "hello")
	runGitIn(topDir, "init", "-q")
	runGitIn(topDir, "add", "-A")
	runGitIn(topDir, "-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "top")
	// protocol.file.allow=always is needed because modern git refuses a
	// file:// (here, a bare local path) submodule URL by default.
	runGitIn(topDir, "-c", "protocol.file.allow=always", "submodule", "add", "-q", subDir, "sub")
	runGitIn(topDir, "-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "addsub")

	root := newRoot(t, topDir, true)
	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Contains(t, candidatePaths(candidates), "sub/cron.yaml",
		"a file tracked inside an initialized submodule must be visible, the same as the filesystem-walk fallback already sees it")
	assert.Contains(t, candidatePaths(candidates), "top.txt")
}

func TestEnumerate_UninitializedSubmoduleDoesNotError(t *testing.T) {
	// An uninitialized submodule must degrade gracefully to the plain gitlink entry, not fail the whole listing.
	gitPath := requireGit(t)

	base := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	subDir := filepath.Join(base, "sub")
	mkdirAll(t, subDir)
	runGitIn := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	writeFile(t, filepath.Join(subDir, "cron.yaml"), "kind: CronJob\n")
	runGitIn(subDir, "init", "-q")
	runGitIn(subDir, "add", "-A")
	runGitIn(subDir, "-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "sub")

	topDir := filepath.Join(base, "top")
	mkdirAll(t, topDir)
	writeFile(t, filepath.Join(topDir, "top.txt"), "hello")
	runGitIn(topDir, "init", "-q")
	runGitIn(topDir, "add", "-A")
	runGitIn(topDir, "-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "top")
	runGitIn(topDir, "-c", "protocol.file.allow=always", "submodule", "add", "-q", subDir, "sub")
	runGitIn(topDir, "-c", "user.email=t@t.com", "-c", "user.name=t", "commit", "-q", "-m", "addsub")

	// A fresh clone that does not initialize submodules leaves "sub" an
	// empty directory: registered in .gitmodules, but never checked out.
	clone := filepath.Join(base, "clone")
	runGitIn(base, "clone", "-q", "--no-recurse-submodules", topDir, clone)

	root := newRoot(t, clone, true)
	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Contains(t, candidatePaths(candidates), "top.txt")
	assert.NotContains(t, candidatePaths(candidates), "sub/cron.yaml")
}

func TestClassify_MissingPathSkippedSilently(t *testing.T) {
	dir := t.TempDir()
	root := newRoot(t, dir, false)

	candidates, problems := classify(root, EnumerateOptions{}, []string{"does-not-exist.txt"})
	assert.Empty(t, candidates)
	assert.Empty(t, problems)
}

func TestClassify_LstatErrorOtherThanMissingIsAProblem(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not restrict directory search on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores permission bits")
	}

	dir := t.TempDir()
	root := newRoot(t, dir, false)

	blocked := filepath.Join(dir, "blocked")
	mkdirAll(t, blocked)
	writeFile(t, filepath.Join(blocked, "x.txt"), "y")
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	require.NoError(t, os.Chmod(blocked, 0))

	candidates, problems := classify(root, EnumerateOptions{}, []string{"blocked/x.txt"})
	assert.Empty(t, candidates)
	require.Len(t, problems, 1)
	assert.Equal(t, "blocked/x.txt", problems[0].Path)
	assert.Error(t, problems[0].Err)
}

func TestClassify_BrokenSymlinkExcludedSilently(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}

	dir := t.TempDir()
	root := newRoot(t, dir, false)

	dangling := filepath.Join(dir, "dangling")
	require.NoError(t, os.Symlink(filepath.Join(dir, "missing-target"), dangling))

	candidates, problems := classify(root, EnumerateOptions{}, []string{"dangling"})
	assert.Empty(t, candidates)
	assert.Empty(t, problems)
}

func TestClassify_SymlinkedParentDirectoryDoesNotEscapeRoot(t *testing.T) {
	// os.Root must enforce containment on every path component, not just the leaf, since "a" can be a symlink.
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}

	base := t.TempDir()
	dir := filepath.Join(base, "root")
	mkdirAll(t, dir)
	root := newRoot(t, dir, false)

	outside := filepath.Join(base, "outside")
	mkdirAll(t, outside)
	writeFile(t, filepath.Join(outside, "secret.yaml"), "REALLY-SECRET-CRON-DATA")

	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "a")))

	candidates, problems := classify(root, EnumerateOptions{}, []string{"a/secret.yaml"})
	assert.Empty(t, candidates, "a symlinked ancestor must not let its leaf become a candidate")
	assert.Empty(t, problems, "an escaping path is silent, the same as any other containment failure")
}

func TestClassify_NestedSymlinkedAncestorDoesNotEscapeRoot(t *testing.T) {
	// The escaping symlink here is an ancestor two levels up, not the immediate parent.
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}

	base := t.TempDir()
	dir := filepath.Join(base, "root")
	mkdirAll(t, filepath.Join(dir, "a"))
	root := newRoot(t, dir, false)

	outside := filepath.Join(base, "outside")
	mkdirAll(t, outside)
	writeFile(t, filepath.Join(outside, "secret.yaml"), "REALLY-SECRET-CRON-DATA")

	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "a", "b")))

	candidates, problems := classify(root, EnumerateOptions{}, []string{"a/b/secret.yaml"})
	assert.Empty(t, candidates)
	assert.Empty(t, problems)
}

func TestClassify_UnreadableFileIsAProblem(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not restrict file reads on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores permission bits")
	}

	dir := t.TempDir()
	root := newRoot(t, dir, false)

	file := filepath.Join(dir, "secret.txt")
	writeFile(t, file, "top secret")
	t.Cleanup(func() { _ = os.Chmod(file, 0o644) })
	require.NoError(t, os.Chmod(file, 0))

	candidates, problems := classify(root, EnumerateOptions{}, []string{"secret.txt"})
	assert.Empty(t, candidates)
	require.Len(t, problems, 1)
	assert.Equal(t, "secret.txt", problems[0].Path)
	assert.Error(t, problems[0].Err)
}

func TestClassify_EmptyRawPathsSkipsOpeningTheRoot(t *testing.T) {
	// A nonexistent root.Dir would make os.OpenRoot fail; classify must
	// not even try when there is nothing to classify.
	root := newRoot(t, t.TempDir(), false)
	root.Dir = filepath.Join(root.Dir, "does-not-exist")

	candidates, problems := classify(root, EnumerateOptions{}, nil)
	assert.Empty(t, candidates)
	assert.Empty(t, problems)
}

func TestClassify_UnopenableRootReportsEveryPathAsAProblem(t *testing.T) {
	// If the root can't be opened as a sandbox, every raw path is reported, not silently dropped.
	root := newRoot(t, t.TempDir(), false)
	root.Dir = filepath.Join(root.Dir, "does-not-exist")

	candidates, problems := classify(root, EnumerateOptions{}, []string{"a.txt", "b.txt"})
	assert.Empty(t, candidates)
	require.Len(t, problems, 2)
	assert.Equal(t, []string{"a.txt", "b.txt"}, []string{problems[0].Path, problems[1].Path})
	assert.Error(t, problems[0].Err)
	assert.Error(t, problems[1].Err)
}

func TestEnumerate_ContainmentEnforcedOnEveryPathNotJustSymlinks(t *testing.T) {
	// Simulates a misbehaving source reporting a ".." component, proving containment is checked before any stat.
	if runtime.GOOS == "windows" {
		t.Skip("fake shell-script git binary requires a unix shell")
	}

	base := t.TempDir()
	rootDir := filepath.Join(base, "root")
	mkdirAll(t, rootDir)
	writeFile(t, filepath.Join(base, "escape.txt"), "must never be read")

	t.Setenv("PATH", writeFakeGit(t, "#!/bin/sh\nprintf '../escape.txt\\0'\n"))

	root := newRoot(t, rootDir, true)

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Empty(t, candidatePaths(candidates), "a path escaping the root must never become a candidate")
}

func TestEnumerate_NoIgnoreForcesFallbackEvenInARepo(t *testing.T) {
	dir := t.TempDir()
	root := newRoot(t, dir, true)
	writeFile(t, filepath.Join(dir, "vendor-marker.txt"), "x")
	mkdirAll(t, filepath.Join(dir, "vendor"))
	writeFile(t, filepath.Join(dir, "vendor", "dep.go"), "package vendor")

	candidates, problems, err := Enumerate(root, root.Dir, EnumerateOptions{NoIgnore: true})
	require.NoError(t, err)
	assert.Empty(t, problems)
	assert.Equal(t, []string{"vendor-marker.txt"}, candidatePaths(candidates))
}

// TestEnumerate_RecognisesSkipsFilesNoSourceWants pins that a path no source claims is never
// stat'ed or read: Source.Match is path-only, so the whole tree does not need opening.
func TestEnumerate_RecognisesSkipsFilesNoSourceWants(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "wanted.yaml"), "kind: CronJob\n")
	writeFile(t, filepath.Join(dir, "ignored.bin"), "whatever\n")
	root := newRootForTest(t, dir)

	var asked []string
	opts := EnumerateOptions{
		Recognises: func(path string) bool {
			asked = append(asked, path)
			return strings.HasSuffix(path, ".yaml")
		},
	}

	candidates, problems, err := Enumerate(root, root.Dir, opts)
	require.NoError(t, err)
	assert.Empty(t, problems)

	require.Len(t, candidates, 1)
	assert.Equal(t, "wanted.yaml", candidates[0].Path)
	assert.ElementsMatch(t, []string{"ignored.bin", "wanted.yaml"}, asked,
		"every path is offered to Recognises, and only the wanted one is opened")
}

// TestEnumerate_OversizedFileNoSourceWantsIsNotAProblem covers --max-file-size: a huge binary
// nothing would parse is noise, not a diagnostic.
func TestEnumerate_OversizedFileNoSourceWantsIsNotAProblem(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "big.bin"), strings.Repeat("x", 200))
	writeFile(t, filepath.Join(dir, "big.yaml"), strings.Repeat("y", 200))
	root := newRootForTest(t, dir)

	opts := EnumerateOptions{
		MaxFileSize: 10,
		Recognises:  func(path string) bool { return strings.HasSuffix(path, ".yaml") },
	}

	_, problems, err := Enumerate(root, root.Dir, opts)
	require.NoError(t, err)

	require.Len(t, problems, 1, "only the file a source would have read is worth reporting")
	assert.Equal(t, "big.yaml", problems[0].Path)
}
