// Package discover locates the boundaries a discovery walk operates within.
package discover

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Root anchors every path a discovery walk reports, relative to Dir, not the walk's start directory.
type Root struct {
	// Dir is the absolute, symlink-resolved root directory.
	Dir string
	// IsRepo reports whether Dir was located via a .git entry, rather than falling back to the start directory.
	IsRepo bool
}

// FindRoot walks upward for the nearest ancestor with a .git entry, else returns start with IsRepo false.
func FindRoot(start string) (Root, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return Root{}, fmt.Errorf("discover: resolve %q: %w", start, err)
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return Root{}, fmt.Errorf("discover: resolve %q: %w", start, err)
	}

	// A Stat failure here is left for the .git lookup below to surface.
	dirStart := resolved
	if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
		dirStart = filepath.Dir(resolved)
	}

	for dir := dirStart; ; {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return Root{Dir: dir, IsRepo: true}, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return Root{}, fmt.Errorf("discover: check %q for .git: %w", dir, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return Root{Dir: dirStart, IsRepo: false}, nil
}

// Rel returns target's path relative to the root, refusing one that would climb outside it.
func (r Root) Rel(target string) (string, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("discover: resolve %q: %w", target, err)
	}

	rel, err := filepath.Rel(r.Dir, abs)
	if err != nil {
		return "", fmt.Errorf("discover: %q is not relative to %q: %w", target, r.Dir, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("discover: %q is outside root %q", target, r.Dir)
	}

	return filepath.ToSlash(rel), nil
}
