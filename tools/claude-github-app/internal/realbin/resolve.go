// Package realbin resolves the on-disk path of the real `claude` binary the
// wrapper should exec, defending against the wrapper accidentally invoking
// itself (self-loop) when it has shadowed the real binary on PATH.
package realbin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// ResolveByName walks $PATH and returns the first executable named `name`
// that is NOT located in the same directory as selfPath. This is the shim
// resolver used by the gh and git wrappers: it lets ~/bin/gh shadow gh while
// still being able to find the real gh under /opt/homebrew/bin, /usr/bin, etc.
//
// selfPath should be os.Executable() of the calling shim. If empty, the
// same-directory filter is skipped — the resolver returns the first match
// on PATH regardless. This matches the soft-fail policy used by Resolve.
//
// Errors:
//   - "PATH is empty": $PATH unset or empty
//   - "<name> not found in PATH": no executable match anywhere
func ResolveByName(name, selfPath string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("ResolveByName: name is empty")
	}
	if strings.ContainsRune(name, filepath.Separator) {
		return "", fmt.Errorf("ResolveByName: name %q must be a bare filename, not a path", name)
	}

	path := os.Getenv("PATH")
	if path == "" {
		return "", fmt.Errorf("PATH is empty")
	}

	var skipDir string
	if selfPath != "" {
		if resolved, err := filepath.EvalSymlinks(selfPath); err == nil {
			skipDir = filepath.Dir(filepath.Clean(resolved))
		}
	}

	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			continue
		}
		// Compare against the resolved form so e.g. /Users/me/bin via
		// symlink doesn't masquerade as a different dir.
		var dirResolved string
		if r, err := filepath.EvalSymlinks(dir); err == nil {
			dirResolved = filepath.Clean(r)
		} else {
			dirResolved = filepath.Clean(dir)
		}
		if skipDir != "" && dirResolved == skipDir {
			continue
		}
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		if info.Mode().Perm()&0o111 == 0 {
			continue
		}
		// EvalSymlinks the candidate, then do a final self-loop guard
		// (covers the case where two PATH dirs happen to symlink to the
		// same physical file).
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			continue
		}
		resolved = filepath.Clean(resolved)
		if selfPath != "" {
			if selfResolved, err := filepath.EvalSymlinks(selfPath); err == nil {
				if filepath.Clean(selfResolved) == resolved {
					continue
				}
			}
		}
		return resolved, nil
	}
	return "", fmt.Errorf("%s not found in PATH (excluding shim dir %q)", name, skipDir)
}
