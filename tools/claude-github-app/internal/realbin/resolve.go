// Package realbin resolves the on-disk path of the real `claude` binary the
// wrapper should exec, defending against the wrapper accidentally invoking
// itself (self-loop) when it has shadowed the real binary on PATH.
package realbin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultRealPath is the canonical install path of the Claude Code CLI on
// macOS (a symlink into ~/.local/share/claude/versions/<ver>). The wrapper
// re-resolves this on every launch since claude self-updates rewrite the
// symlink target.
const DefaultRealPath = "~/.local/bin/claude"

// Resolve returns an absolute, EvalSymlinks'd path to the real claude binary.
//
//   - If override is non-empty, it is used verbatim (only EvalSymlinks'd).
//   - Otherwise DefaultRealPath is consulted.
//
// The result is checked against selfPath (typically os.Executable()) to
// detect a recursive shadowing situation. If selfPath is empty (e.g.
// os.Executable returned an error), the guard is skipped — refusing to
// launch on guard failure would brick the wrapper.
func Resolve(override, selfPath string) (string, error) {
	target := override
	if target == "" {
		target = DefaultRealPath
	}
	expanded, err := expandTilde(target)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve real claude (%s): %w", expanded, err)
	}
	resolved = filepath.Clean(resolved)

	if selfPath != "" {
		selfResolved, err := filepath.EvalSymlinks(selfPath)
		if err == nil && filepath.Clean(selfResolved) == resolved {
			return "", fmt.Errorf("self-loop guard: real claude resolved to %s which equals this wrapper; "+
				"set claude_binary in config to the correct path", resolved)
		}
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("stat real claude (%s): %w", resolved, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("real claude path is a directory: %s", resolved)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("real claude is not executable: %s", resolved)
	}
	return resolved, nil
}

func expandTilde(p string) (string, error) {
	if len(p) > 0 && p[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			return home, nil
		}
		if p[1] == '/' {
			return filepath.Join(home, p[2:]), nil
		}
	}
	return p, nil
}

// ErrNotFound is the underlying class of error returned when the real claude
// binary cannot be located. Callers translate this into the §8 "Real claude
// not found" failure mode (exit 127).
var ErrNotFound = errors.New("real claude binary not found")
