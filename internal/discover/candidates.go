package discover

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Candidate is a file Enumerate selected to parse, with the fs.FileInfo already stat'ed.
type Candidate struct {
	Path string
	Info fs.FileInfo
}

// Problem is a path Enumerate could not include and why.
type Problem struct {
	Path string
	Err  error
}

// EnumerateOptions tunes what Enumerate selects.
type EnumerateOptions struct {
	// MaxFileSize excludes files larger than this many bytes as a Problem; zero means no limit.
	MaxFileSize int64
	// NoIgnore forces the filesystem-walk fallback, opting out of .gitignore even when git is available.
	NoIgnore bool
	// Excludes are additional path.Match glob patterns; a pattern without "/" matches any single path segment.
	Excludes []string
}

// skipDirs are directories the filesystem-walk fallback never descends into, standing in for .gitignore.
var skipDirs = map[string]bool{
	".git":         true,
	"vendor":       true,
	"node_modules": true,
}

// gitTimeout bounds how long Enumerate waits on the git subprocess; it's a var so tests can shrink it.
var gitTimeout = 10 * time.Second

// binarySniffLen is how many leading bytes Enumerate inspects for a NUL
// byte when deciding whether a file is binary.
const binarySniffLen = 8192

// gitEnv lists the environment variables passed through to the git subprocess.
var gitEnv = []string{"PATH", "HOME", "USERPROFILE", "SYSTEMROOT", "XDG_CONFIG_HOME"}

// Enumerate lists the files under scope, preferring git's ignore semantics over a filesystem walk.
func Enumerate(root Root, scope string, opts EnumerateOptions) ([]Candidate, []Problem, error) {
	if err := validateExcludes(opts.Excludes); err != nil {
		return nil, nil, err
	}

	scopeRel, err := root.Rel(scope)
	if err != nil {
		return nil, nil, fmt.Errorf("discover: scope %q is outside root %q: %w", scope, root.Dir, err)
	}

	var (
		rawPaths     []string
		walkProblems []Problem
		usedGit      bool
	)

	if !opts.NoIgnore && root.IsRepo {
		if paths, ok := gitEnumerate(root, scopeRel); ok {
			rawPaths = paths
			usedGit = true
		}
	}
	// usedGit, not rawPaths' emptiness, decides whether the fallback runs.
	if !usedGit {
		rawPaths, walkProblems = walkFS(root, scope)
	}

	candidates, problems := classify(root, opts, rawPaths)
	problems = append(walkProblems, problems...)

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Path < candidates[j].Path })
	sort.Slice(problems, func(i, j int) bool { return problems[i].Path < problems[j].Path })

	return candidates, problems, nil
}

// validateExcludes rejects a malformed glob pattern up front, so Enumerate fails fast.
func validateExcludes(patterns []string) error {
	for _, p := range patterns {
		if _, err := path.Match(p, ""); err != nil {
			return fmt.Errorf("discover: invalid exclude pattern %q: %w", p, err)
		}
	}
	return nil
}

// gitEnumerate lists candidate paths via git ls-files, returning ok=false if the listing can't be trusted.
func gitEnumerate(root Root, scopeRel string) (paths []string, ok bool) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	// Two calls are required since git refuses to combine --others with --recurse-submodules.
	cached, ok := runLsFiles(ctx, gitPath, root, scopeRel, "--cached", "--recurse-submodules")
	if !ok {
		return nil, false
	}
	others, ok := runLsFiles(ctx, gitPath, root, scopeRel, "--others")
	if !ok {
		return nil, false
	}

	// git ls-files -z always trails output with a NUL, so the two buffers can just be concatenated.
	return splitNulPaths(append(cached, others...)), true
}

// runLsFiles runs one git ls-files invocation, returning ok=false if the result can't be trusted.
func runLsFiles(ctx context.Context, gitPath string, root Root, scopeRel string, extra ...string) (out []byte, ok bool) {
	args := append([]string{
		// core.fsmonitor=false and --no-optional-locks stop a scanned repo's .git/config from executing code.
		"-c", "core.fsmonitor=false", "--no-optional-locks", "ls-files", "-z",
	}, extra...)
	// ":(literal)" matches scopeRel byte-for-byte so it can't be misread as a pathspec-magic prefix.
	args = append(args, "--exclude-standard", "--", ":(literal)"+scopeRel)

	cmd := exec.CommandContext(ctx, gitPath, args...)
	cmd.Dir = root.Dir
	cmd.Env = filterEnv(os.Environ())
	// WaitDelay bounds Output()'s wait after cancellation, so a grandchild process can't outlast gitTimeout.
	cmd.WaitDelay = gitTimeout

	result, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	return result, true
}

// filterEnv keeps only variables named in gitEnv; it never returns nil, which exec.Cmd reads as inherit-all.
func filterEnv(environ []string) []string {
	keep := make(map[string]bool, len(gitEnv))
	for _, name := range gitEnv {
		keep[name] = true
	}

	out := make([]string, 0, len(gitEnv))
	for _, kv := range environ {
		name, _, found := strings.Cut(kv, "=")
		if found && keep[name] {
			out = append(out, kv)
		}
	}
	return out
}

// splitNulPaths splits git ls-files -z output on NUL and deduplicates paths.
func splitNulPaths(out []byte) []string {
	out = bytes.TrimSuffix(out, []byte{0})
	if len(out) == 0 {
		return nil
	}

	parts := bytes.Split(out, []byte{0})
	seen := make(map[string]bool, len(parts))
	paths := make([]string, 0, len(parts))
	for _, p := range parts {
		s := string(p)
		if seen[s] {
			continue
		}
		seen[s] = true
		paths = append(paths, s)
	}
	return paths
}

// walkFS lists candidate paths by walking scope, skipping skipDirs and reporting unreadable entries.
func walkFS(root Root, scope string) ([]string, []Problem) {
	var (
		paths    []string
		problems []Problem
	)

	_ = filepath.WalkDir(scope, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// scope is always root.Dir or a descendant (enforced by Enumerate), so Rel cannot fail here.
			rel, _ := root.Rel(p)
			problems = append(problems, Problem{Path: rel, Err: err})
			return nil
		}
		if p == scope && d.IsDir() {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// A worktree or submodule checkout records its root as a ".git" file, not a directory.
		if d.Name() == ".git" {
			return nil
		}

		rel, _ := root.Rel(p)
		paths = append(paths, rel)
		return nil
	})

	return paths, problems
}

// classify turns raw candidate paths into Candidates and Problems, applying excludes and the size limit.
//
// Every read here goes through an os.Root, since a lexical join can't catch a symlinked ancestor directory.
func classify(root Root, opts EnumerateOptions, rawPaths []string) ([]Candidate, []Problem) {
	var (
		candidates []Candidate
		problems   []Problem
	)

	if len(rawPaths) == 0 {
		return candidates, problems
	}

	rt, err := os.OpenRoot(root.Dir)
	if err != nil {
		// The root itself failed to open as a sandbox, so every path is reported as a Problem.
		for _, rel := range rawPaths {
			problems = append(problems, Problem{Path: rel, Err: err})
		}
		return candidates, problems
	}
	defer func() {
		_ = rt.Close()
	}()

	for _, rel := range rawPaths {
		if matchesExclude(rel, opts.Excludes) {
			continue
		}

		info, err := rt.Lstat(rel)
		if err != nil {
			if isRootEscape(err) || errors.Is(err, fs.ErrNotExist) {
				// A containment failure or a vanished git index entry is dropped silently.
				continue
			}
			problems = append(problems, Problem{Path: rel, Err: err})
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := rt.Stat(rel)
			if err != nil {
				if isRootEscape(err) || errors.Is(err, fs.ErrNotExist) {
					// An escaping or broken symlink: excluded silently,
					// like any other irregular entry.
					continue
				}
				problems = append(problems, Problem{Path: rel, Err: err})
				continue
			}
			info = resolved
		}

		if !info.Mode().IsRegular() {
			// Directories and irregular files (FIFOs, devices, ...)
			// are excluded silently.
			continue
		}

		if opts.MaxFileSize > 0 && info.Size() > opts.MaxFileSize {
			problems = append(problems, Problem{
				Path: rel,
				Err:  fmt.Errorf("discover: %q is %d bytes, over the %d byte limit", rel, info.Size(), opts.MaxFileSize),
			})
			continue
		}

		binary, err := isBinary(rt, rel)
		if err != nil {
			problems = append(problems, Problem{Path: rel, Err: err})
			continue
		}
		if binary {
			continue
		}

		candidates = append(candidates, Candidate{Path: rel, Info: info})
	}

	return candidates, problems
}

// isRootEscape matches os.Root's escape error by message text; the os package exports no sentinel for it.
func isRootEscape(err error) bool {
	return err != nil && strings.Contains(err.Error(), "path escapes from parent")
}

// matchesExclude reports whether rel matches any pattern; see EnumerateOptions.Excludes for the slash split.
func matchesExclude(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(pattern, "/") {
			if ok, _ := path.Match(pattern, rel); ok {
				return true
			}
			continue
		}
		for _, seg := range strings.Split(rel, "/") {
			if ok, _ := path.Match(pattern, seg); ok {
				return true
			}
		}
	}
	return false
}

// isBinary reports whether rel's leading bytes contain a NUL byte, the same heuristic git itself uses.
func isBinary(rt *os.Root, rel string) (bool, error) {
	f, err := rt.Open(rel)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = f.Close()
	}()

	buf := make([]byte, binarySniffLen)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false, err
	}

	return bytes.IndexByte(buf[:n], 0) != -1, nil
}
