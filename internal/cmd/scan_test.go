package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/hzerrad/cronkit/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireGit skips the test, with an explicit message, when no git binary
// is on PATH.
func requireGit(t *testing.T) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git binary not found on PATH; skipping git-backed test")
	}
	return gitPath
}

// failingWriter always errors, for exercising an Encode failure path that a
// normal buffer never hits.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

// scanFixture copies testdata/scan into a fresh temp dir and adds a bare
// ".git" so discover.FindRoot treats it as a repository root.
func scanFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "scan")
	require.NoError(t, os.CopyFS(dir, os.DirFS(src)))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	return dir
}

// emptyFixture is a repository root holding no schedules at all.
func emptyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("nothing here\n"), 0o644))
	return dir
}

// runScan executes a fresh scan command with args, overriding osExit so a
// --strict exit doesn't kill the test process, and reports whether it was called.
func runScan(t *testing.T, args ...string) (out, errOut string, exitCode int, exited bool, err error) {
	t.Helper()
	sc := newScanCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	sc.SetOut(outBuf)
	sc.SetErr(errBuf)
	sc.SetArgs(args)

	oldExit := osExit
	osExit = func(code int) {
		exitCode = code
		exited = true
	}
	defer func() { osExit = oldExit }()

	err = sc.Execute()
	return outBuf.String(), errBuf.String(), exitCode, exited, err
}

func TestScanCommand_Registered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"scan"})
	assert.NoError(t, err)
	assert.Equal(t, "scan", cmd.Name())
}

func TestScanCommand_Metadata(t *testing.T) {
	sc := newScanCommand()
	assert.NotEmpty(t, sc.Short)
	assert.NotEmpty(t, sc.Long)
	assert.Contains(t, sc.Use, "scan")
	assert.Contains(t, sc.Long, "Exit codes")

	for _, name := range []string{"json", "source", "exclude-source", "exclude", "no-ignore", "max-file-size", "strict"} {
		assert.NotNil(t, sc.Flag(name), "expected --%s to be registered", name)
	}
	// --fail-on and --file are deliberately absent: scan discovers (check owns
	// severity gating) and takes paths as positional args, not a single --file.
	assert.Nil(t, sc.Flag("fail-on"))
	assert.Nil(t, sc.Flag("file"))
}

// TestScanCommand_SourceFlagHelpListsEveryValidID guards against the
// --source/--exclude-source help text drifting from the actual source registry.
func TestScanCommand_SourceFlagHelpListsEveryValidID(t *testing.T) {
	registry, err := source.Default()
	require.NoError(t, err)

	sc := newScanCommand()
	for _, s := range registry.Sources() {
		assert.Contains(t, sc.Flag("source").Usage, s.ID(), "--source help should list %q", s.ID())
		assert.Contains(t, sc.Flag("exclude-source").Usage, s.ID(), "--exclude-source help should list %q", s.ID())
	}
}

func TestScanCommand_JSONOutputDecodesAndContainsExpectedItems(t *testing.T) {
	dir := scanFixture(t)

	out, errOut, _, exited, err := runScan(t, dir, "--json")
	require.NoError(t, err)
	assert.False(t, exited)
	assert.Contains(t, errOut, "broken.yaml", "the fixture's malformed manifest is reported as a problem, on stderr")

	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	assert.Equal(t, dir, inv.Root)

	bySource := map[string]int{}
	for _, it := range inv.Items {
		bySource[it.SourceID]++
	}
	assert.Equal(t, 1, bySource["crontab"])
	assert.Equal(t, 1, bySource["k8s"])
	assert.Equal(t, 2, bySource["argo"])
	assert.Equal(t, 2, bySource["gha"])
}

// TestScanCommand_AcceptsASingleFileArgument pins that scanning a single file
// path (outside any ".git"-bearing directory) succeeds, since a file cannot
// have a ".git" child.
func TestScanCommand_AcceptsASingleFileArgument(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "backup.cron")
	require.NoError(t, os.WriteFile(file, []byte("0 2 * * * /usr/bin/backup.sh\n"), 0o644))

	out, _, _, exited, err := runScan(t, file, "--json")
	require.NoError(t, err)
	assert.False(t, exited)

	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	require.Len(t, inv.Items, 1)
	assert.Equal(t, "0 2 * * *", inv.Items[0].Expression)
	assert.Equal(t, "backup.cron", inv.Items[0].Locator.File)
}

// TestScanCommand_JSONRootIsRelativeToTheInvocationDirectory pins that
// scanning the working directory reports root as "." and scanning a
// descendant reports a relative path, never an absolute one.
func TestScanCommand_JSONRootIsRelativeToTheInvocationDirectory(t *testing.T) {
	dir := scanFixture(t)
	t.Chdir(dir)

	out, _, _, _, err := runScan(t, ".", "--json")
	require.NoError(t, err)
	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	assert.Equal(t, ".", inv.Root, "scanning the working directory itself must report root as \".\"")

	// Climbing to the parent and scanning the fixture as a subdirectory
	// exercises the "descendant" case rather than "is the working directory".
	parent := filepath.Dir(dir)
	t.Chdir(parent)
	base := filepath.Base(dir)

	out2, _, _, _, err2 := runScan(t, base, "--json")
	require.NoError(t, err2)
	inv2, decodeErr2 := inventory.Decode(strings.NewReader(out2))
	require.NoError(t, decodeErr2)
	assert.Equal(t, filepath.ToSlash(base), inv2.Root, "scanning a descendant of the working directory must report a relative path")
	assert.False(t, filepath.IsAbs(inv2.Root), "root must not be an absolute path when it sits at or below the working directory")
}

// TestScanCommand_JSONRootStaysAbsoluteAboveTheInvocationDirectory pins that
// a root above the working directory stays absolute rather than becoming a
// "../.." relative path.
func TestScanCommand_JSONRootStaysAbsoluteAboveTheInvocationDirectory(t *testing.T) {
	dir := scanFixture(t)
	nestedCwd := filepath.Join(dir, "deploy")
	t.Chdir(nestedCwd)

	out, _, _, _, err := runScan(t, dir, "--json")
	require.NoError(t, err)
	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	assert.Equal(t, dir, inv.Root, "a root above the working directory must stay absolute")
}

// TestDisplayPath exercises displayPath directly: relative at or below the
// working directory, absolute otherwise, and the empty string returned unchanged.
func TestDisplayPath(t *testing.T) {
	assert.Equal(t, "", displayPath(""))

	base := t.TempDir()
	wd := filepath.Join(base, "wd")
	require.NoError(t, os.MkdirAll(wd, 0o755))
	t.Chdir(wd)

	assert.Equal(t, ".", displayPath(wd), "root equal to the working directory must report \".\"")

	child := filepath.Join(wd, "repo")
	require.NoError(t, os.MkdirAll(child, 0o755))
	assert.Equal(t, "repo", displayPath(child), "a descendant of the working directory must be relative")

	parent := base
	assert.Equal(t, parent, displayPath(parent), "a root above the working directory must stay absolute")

	sibling := filepath.Join(base, "elsewhere")
	require.NoError(t, os.MkdirAll(sibling, 0o755))
	assert.Equal(t, sibling, displayPath(sibling), "a root outside the working directory's own subtree must stay absolute")
}

// TestDisplayPath_UnrelatableRootStaysAbsolute covers filepath.Rel's error
// path: a non-absolute root falls back to being returned unchanged rather
// than erroring or fabricating a path.
func TestDisplayPath_UnrelatableRootStaysAbsolute(t *testing.T) {
	t.Chdir(t.TempDir())

	assert.Equal(t, "relative/path", displayPath("relative/path"))
}

// TestDisplayPath_UnresolvableWorkingDirectoryStaysAbsolute covers
// os.Getwd's error path: with no working directory to relate to, displayPath
// falls back to root unchanged.
func TestDisplayPath_UnresolvableWorkingDirectoryStaysAbsolute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("removing the current working directory behaves differently on windows")
	}

	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	gone := filepath.Join(t.TempDir(), "gone")
	require.NoError(t, os.MkdirAll(gone, 0o755))
	require.NoError(t, os.Chdir(gone))
	require.NoError(t, os.Remove(gone))

	assert.Equal(t, "/some/root", displayPath("/some/root"))
	// A relative path cannot even be made absolute without a working directory, so it stays as given.
	assert.Equal(t, "relative/path", displayPath("relative/path"))
}

func TestScanCommand_SameScanTwiceIsByteIdentical(t *testing.T) {
	dir := scanFixture(t)

	out1, _, _, _, err1 := runScan(t, dir, "--json")
	require.NoError(t, err1)
	out2, _, _, _, err2 := runScan(t, dir, "--json")
	require.NoError(t, err2)

	assert.Equal(t, out1, out2, "two scans of an unchanged tree must emit byte-identical JSON")
}

func TestScanCommand_NoPathsDefaultsToCurrentDirectory(t *testing.T) {
	dir := scanFixture(t)
	t.Chdir(dir)

	out, _, _, _, err := runScan(t, "--json")
	require.NoError(t, err)

	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	assert.NotEmpty(t, inv.Items)
}

func TestScanCommand_UnknownSourceIDIsAnError(t *testing.T) {
	dir := scanFixture(t)

	_, _, _, exited, err := runScan(t, dir, "--source=nope")
	require.Error(t, err)
	assert.False(t, exited, "an unknown source id is a configuration error, not a --strict-style osExit")
	assert.Contains(t, err.Error(), `"nope"`)
	assert.Contains(t, err.Error(), "crontab", "the error should name the valid sources too")
}

func TestScanCommand_UnknownExcludeSourceIDIsAnError(t *testing.T) {
	dir := scanFixture(t)

	_, _, _, _, err := runScan(t, dir, "--exclude-source=nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"nope"`)
}

func TestScanCommand_UnreadableRootIsAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	_, _, _, _, err := runScan(t, missing)
	require.Error(t, err)
}

func TestScanCommand_NoSchedulesExitsZeroWithEmptyItemsArray(t *testing.T) {
	dir := emptyFixture(t)

	out, _, _, exited, err := runScan(t, dir, "--json")
	require.NoError(t, err)
	assert.False(t, exited)

	assert.Contains(t, out, `"items": []`, "an empty inventory must serialize items as [], never null")

	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	assert.Empty(t, inv.Items)
}

func TestScanCommand_StrictTurnsAPerFileProblemIntoANonZeroExit(t *testing.T) {
	dir := scanFixture(t)

	// Without --strict: the same tree (broken.yaml causes k8s and argo to
	// each report a Problem) exits 0.
	_, errOut, _, exited, err := runScan(t, dir, "--json")
	require.NoError(t, err)
	assert.False(t, exited)
	assert.Contains(t, errOut, "broken.yaml")

	// With --strict: the walk still completes (err is nil, output is still
	// produced) but osExit(1) is invoked because problems were reported.
	out2, errOut2, exitCode, exited2, err2 := runScan(t, dir, "--json", "--strict")
	require.NoError(t, err2)
	assert.True(t, exited2)
	assert.Equal(t, 1, exitCode)
	assert.Contains(t, errOut2, "broken.yaml")
	assert.NotEmpty(t, out2, "the inventory is still emitted even when --strict trips the exit code")
}

func TestScanCommand_StrictWithNoProblemsExitsZero(t *testing.T) {
	dir := emptyFixture(t)

	_, _, _, exited, err := runScan(t, dir, "--json", "--strict")
	require.NoError(t, err)
	assert.False(t, exited, "no problems were reported, so --strict must not trip the exit code")
}

func TestScanCommand_SourceFilterRestrictsToNamedIDs(t *testing.T) {
	dir := scanFixture(t)

	out, _, _, _, err := runScan(t, dir, "--json", "--source=crontab")
	require.NoError(t, err)

	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	require.NotEmpty(t, inv.Items)
	for _, it := range inv.Items {
		assert.Equal(t, "crontab", it.SourceID)
	}
}

func TestScanCommand_ExcludeSourceFilterDropsNamedIDs(t *testing.T) {
	dir := scanFixture(t)

	out, _, _, _, err := runScan(t, dir, "--json", "--exclude-source=crontab")
	require.NoError(t, err)

	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	require.NotEmpty(t, inv.Items)
	for _, it := range inv.Items {
		assert.NotEqual(t, "crontab", it.SourceID)
	}
}

func TestScanCommand_ExcludeFlagIsForwardedToTheWalk(t *testing.T) {
	dir := scanFixture(t)

	out, _, _, _, err := runScan(t, dir, "--json", "--exclude=crontab")
	require.NoError(t, err)

	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	require.NotEmpty(t, inv.Items)
	for _, it := range inv.Items {
		assert.NotEqual(t, "crontab", it.SourceID, "the crontab file itself was excluded by path")
	}
}

func TestScanCommand_ProblemsGoToStderrNeverStdout(t *testing.T) {
	dir := scanFixture(t)

	out, errOut, _, _, err := runScan(t, dir, "--json")
	require.NoError(t, err)

	assert.Contains(t, errOut, "broken.yaml")
	assert.NotContains(t, out, "broken.yaml", "a consumer piping --json into jq must not receive diagnostics in the stream")

	// The stdout stream must still be exactly one valid JSON document.
	_, decodeErr := inventory.Decode(strings.NewReader(out))
	assert.NoError(t, decodeErr)
}

func TestScanCommand_TextOutputListsDiscoveredSchedules(t *testing.T) {
	dir := scanFixture(t)

	out, _, _, _, err := runScan(t, dir)
	require.NoError(t, err)

	assert.Contains(t, out, "SOURCE")
	assert.Contains(t, out, "crontab")
	assert.Contains(t, out, "schedule(s) across")
}

// TestScanCommand_TextOutputPrintsProblemsAfterTheTable pins that when
// stdout and stderr share a terminal, the table prints before the problem
// diagnostics.
func TestScanCommand_TextOutputPrintsProblemsAfterTheTable(t *testing.T) {
	dir := scanFixture(t)

	sc := newScanCommand()
	shared := new(bytes.Buffer)
	sc.SetOut(shared)
	sc.SetErr(shared)
	sc.SetArgs([]string{dir})

	require.NoError(t, sc.Execute())

	out := shared.String()
	tableIdx := strings.Index(out, "schedule(s) across")
	problemIdx := strings.Index(out, "broken.yaml")
	require.NotEqual(t, -1, tableIdx, "expected the summary line to appear")
	require.NotEqual(t, -1, problemIdx, "expected the broken.yaml problem to be reported")
	assert.Less(t, tableIdx, problemIdx, "problems must print after the table, not above it")
}

func TestScanCommand_TextOutputReportsNoSchedulesFound(t *testing.T) {
	dir := emptyFixture(t)

	out, _, _, _, err := runScan(t, dir)
	require.NoError(t, err)
	assert.Contains(t, out, "No schedules found")
}

func TestScanCommand_EncodeFailureIsAnError(t *testing.T) {
	dir := scanFixture(t)

	sc := newScanCommand()
	sc.SetOut(failingWriter{})
	sc.SetErr(new(bytes.Buffer))
	sc.SetArgs([]string{dir, "--json"})

	err := sc.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to encode inventory")
}

// TestScanCommand_NoIgnoreFlagIsForwardedToTheWalk guards against
// --no-ignore being parsed but never reaching discover.Options, using a real
// git repo with a .gitignore'd file to prove it.
func TestScanCommand_NoIgnoreFlagIsForwardedToTheWalk(t *testing.T) {
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

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("secret.cron\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tracked.cron"), []byte("0 2 * * * /usr/bin/backup.sh\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.cron"), []byte("0 3 * * * /usr/bin/ignored.sh\n"), 0o644))

	// Without --no-ignore, git's own listing applies .gitignore: the
	// ignored file must not appear at all.
	out, _, _, _, err := runScan(t, dir, "--json")
	require.NoError(t, err)
	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	for _, it := range inv.Items {
		assert.NotEqual(t, "secret.cron", it.Locator.File, "without --no-ignore, a .gitignore'd file must be honoured")
	}

	// With --no-ignore, the walk falls back to a plain filesystem walk that
	// ignores .gitignore, surfacing the file.
	out2, _, _, _, err2 := runScan(t, dir, "--json", "--no-ignore")
	require.NoError(t, err2)
	inv2, decodeErr2 := inventory.Decode(strings.NewReader(out2))
	require.NoError(t, decodeErr2)
	found := false
	for _, it := range inv2.Items {
		if it.Locator.File == "secret.cron" {
			found = true
		}
	}
	assert.True(t, found, "--no-ignore must reach discover.Options.NoIgnore and surface the otherwise-ignored file")
}

// TestScanCommand_MaxFileSizeFlagIsForwardedToTheWalk guards against
// --max-file-size being parsed but never reaching discover.Options.
func TestScanCommand_MaxFileSizeFlagIsForwardedToTheWalk(t *testing.T) {
	dir := emptyFixture(t)

	content := "0 2 * * * /usr/bin/backup.sh # padded well past the limit below\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "crontab"), []byte(content), 0o644))

	// Without a limit, the file is read normally.
	out, errOut, _, _, err := runScan(t, dir, "--json")
	require.NoError(t, err)
	assert.Empty(t, errOut)
	inv, decodeErr := inventory.Decode(strings.NewReader(out))
	require.NoError(t, decodeErr)
	require.NotEmpty(t, inv.Items, "sanity check: the file is read and produces an item when no limit applies")

	// With a limit below the file's actual size, it must be excluded and
	// reported as a problem instead.
	out2, errOut2, _, _, err2 := runScan(t, dir, "--json", "--max-file-size=10")
	require.NoError(t, err2)
	assert.Contains(t, errOut2, "byte limit", "--max-file-size must reach discover.Options.MaxFileSize and be reported as a problem")
	inv2, decodeErr2 := inventory.Decode(strings.NewReader(out2))
	require.NoError(t, decodeErr2)
	assert.Empty(t, inv2.Items, "the oversized file must be excluded once --max-file-size reaches the walk")
}
