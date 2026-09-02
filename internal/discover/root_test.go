package discover

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mkdirAll creates dir and every missing parent, failing the test on error.
func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
}

// writeFile creates path with the given contents, failing the test on error.
func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
}

func TestFindRoot_DirContainingGitIsItsOwnRoot(t *testing.T) {
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, ".git"))

	got, err := FindRoot(root)
	require.NoError(t, err)

	resolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)

	assert.Equal(t, resolved, got.Dir)
	assert.True(t, got.IsRepo)
}

func TestFindRoot_NestedDirFindsAncestorAndRelIsFullPath(t *testing.T) {
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, ".git"))

	workflows := filepath.Join(root, ".github", "workflows")
	mkdirAll(t, workflows)
	ci := filepath.Join(workflows, "ci.yml")
	writeFile(t, ci, "on: {}\n")

	// Scanning .github/workflows directly must still resolve paths relative to the repository root.
	got, err := FindRoot(workflows)
	require.NoError(t, err)
	assert.True(t, got.IsRepo)

	rel, err := got.Rel(ci)
	require.NoError(t, err)
	assert.Equal(t, ".github/workflows/ci.yml", rel)
}

func TestFindRoot_FileStartClimbsFromItsContainingDirectory(t *testing.T) {
	// start naming a file directly must still find the repository root above it, not error.
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, ".git"))

	nested := filepath.Join(root, "a", "b")
	mkdirAll(t, nested)
	file := filepath.Join(nested, "job.cron")
	writeFile(t, file, "0 2 * * * /usr/bin/backup.sh\n")

	got, err := FindRoot(file)
	require.NoError(t, err)

	resolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)

	assert.Equal(t, resolved, got.Dir, "the root must be the file's containing directory's ancestor, not the file itself")
	assert.True(t, got.IsRepo)

	rel, err := got.Rel(file)
	require.NoError(t, err)
	assert.Equal(t, "a/b/job.cron", rel)
}

func TestFindRoot_FileStartWithNoGitAnywhereUsesItsContainingDirectory(t *testing.T) {
	// With no .git found, Root.Dir must still be a directory (the file's parent), not the file itself.
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	mkdirAll(t, nested)
	file := filepath.Join(nested, "job.cron")
	writeFile(t, file, "0 2 * * * /usr/bin/backup.sh\n")

	got, err := FindRoot(file)
	require.NoError(t, err)

	resolved, err := filepath.EvalSymlinks(nested)
	require.NoError(t, err)

	assert.Equal(t, resolved, got.Dir)
	assert.False(t, got.IsRepo)

	rel, err := got.Rel(file)
	require.NoError(t, err)
	assert.Equal(t, "job.cron", rel)
}

func TestFindRoot_NoGitAnywhereReturnsStartNotARepo(t *testing.T) {
	// A fresh TempDir subdirectory ensures no ancestor can plausibly contain a .git.
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	mkdirAll(t, nested)

	got, err := FindRoot(nested)
	require.NoError(t, err)

	resolved, err := filepath.EvalSymlinks(nested)
	require.NoError(t, err)

	assert.Equal(t, resolved, got.Dir)
	assert.False(t, got.IsRepo)

	file := filepath.Join(nested, "job.yml")
	writeFile(t, file, "")

	rel, err := got.Rel(file)
	require.NoError(t, err)
	assert.Equal(t, "job.yml", rel)
}

func TestFindRoot_GitAsAFileCounts(t *testing.T) {
	// A worktree or submodule records its root with a .git *file*
	// (containing a "gitdir: ..." pointer) rather than a .git directory.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".git"), "gitdir: /elsewhere/.git/worktrees/example\n")

	nested := filepath.Join(root, "pkg", "sub")
	mkdirAll(t, nested)

	got, err := FindRoot(nested)
	require.NoError(t, err)

	resolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)

	assert.Equal(t, resolved, got.Dir)
	assert.True(t, got.IsRepo)
}

func TestFindRoot_ResolvesSymlinkedStart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}

	base := t.TempDir()
	root := filepath.Join(base, "real-root")
	mkdirAll(t, filepath.Join(root, ".git"))

	link := filepath.Join(base, "link")
	require.NoError(t, os.Symlink(root, link))

	got, err := FindRoot(link)
	require.NoError(t, err)

	resolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)

	assert.Equal(t, resolved, got.Dir)
	assert.True(t, got.IsRepo)
	assert.NotEqual(t, link, got.Dir, "the symlink itself must not be treated as the root")
}

func TestFindRoot_NonexistentStartErrors(t *testing.T) {
	root := t.TempDir()

	_, err := FindRoot(filepath.Join(root, "does-not-exist"))
	assert.Error(t, err)
}

// chdirIntoDeletedDir puts the process into a directory and removes it, so os.Getwd() later fails.
func chdirIntoDeletedDir(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("removing the current working directory behaves differently on windows")
	}

	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	gone := filepath.Join(t.TempDir(), "gone")
	mkdirAll(t, gone)
	require.NoError(t, os.Chdir(gone))
	require.NoError(t, os.Remove(gone))
}

func TestFindRoot_UnresolvableRelativeStartErrors(t *testing.T) {
	chdirIntoDeletedDir(t)

	_, err := FindRoot("relative")
	assert.Error(t, err)
}

func TestRel_UnresolvableRelativeTargetErrors(t *testing.T) {
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, ".git"))
	r, err := FindRoot(root)
	require.NoError(t, err)

	chdirIntoDeletedDir(t)

	_, err = r.Rel("relative")
	assert.Error(t, err)
}

func TestFindRoot_UnreadableDirectoryErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not restrict directory search on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores permission bits")
	}

	parent := t.TempDir()
	blocked := filepath.Join(parent, "blocked")
	mkdirAll(t, blocked)
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	require.NoError(t, os.Chmod(blocked, 0))

	_, err := FindRoot(blocked)
	assert.Error(t, err)
}

func TestRel_SlashSeparatedRegardlessOfPlatform(t *testing.T) {
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, ".git"))
	r, err := FindRoot(root)
	require.NoError(t, err)

	nested := filepath.Join(root, "deep", "nested", "path.yml")
	mkdirAll(t, filepath.Dir(nested))
	writeFile(t, nested, "")

	rel, err := r.Rel(nested)
	require.NoError(t, err)
	assert.Equal(t, "deep/nested/path.yml", rel)
	assert.NotContains(t, rel, `\`)
}

func TestRel_RefusesPathOutsideRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	mkdirAll(t, filepath.Join(root, ".git"))
	outside := filepath.Join(base, "elsewhere", "file.yml")
	mkdirAll(t, filepath.Dir(outside))
	writeFile(t, outside, "")

	r, err := FindRoot(root)
	require.NoError(t, err)

	_, err = r.Rel(outside)
	assert.Error(t, err)
}

func TestRel_RefusesRootsParentItself(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	mkdirAll(t, filepath.Join(root, ".git"))

	r, err := FindRoot(root)
	require.NoError(t, err)

	_, err = r.Rel(base)
	assert.Error(t, err, "the root's own parent is outside the root")
}

func TestFindRoot_ClimbsPastADistantAncestorsGitWithoutASeparateBound(t *testing.T) {
	// FindRoot has no bound of its own beyond the filesystem root; pins that a bound is added deliberately.
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, ".git")) // the distant ancestor's own repository
	nested := filepath.Join(root, "a", "b", "c")
	mkdirAll(t, nested)

	got, err := FindRoot(nested)
	require.NoError(t, err)

	resolvedRoot, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	assert.True(t, got.IsRepo, "FindRoot has no bound short of the filesystem's own root")
	assert.Equal(t, resolvedRoot, got.Dir)
}
