package worktree

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CommandRunner abstracts shell command execution for testing.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// WorktreeInfo holds metadata about a git worktree.
type WorktreeInfo struct {
	Path       string
	Branch     string
	HeadCommit string
}

// Manager manages pit-crew git worktrees.
type Manager struct {
	RepoRoot string
	Runner   CommandRunner
}

// New creates a Manager for the given repository root.
func New(repoRoot string, runner CommandRunner) *Manager {
	return &Manager{RepoRoot: repoRoot, Runner: runner}
}

// DefaultBranch detects the repository's default branch dynamically.
// It first tries `gh repo view`, then falls back to `git remote show origin`.
func (m *Manager) DefaultBranch(ctx context.Context) (string, error) {
	type ghResp struct {
		DefaultBranchRef struct {
			Name string `json:"name"`
		} `json:"defaultBranchRef"`
	}
	out, err := m.Runner.Run(ctx, "gh", "repo", "view", "--json", "defaultBranchRef")
	if err == nil {
		var resp ghResp
		if jsonErr := json.Unmarshal(out, &resp); jsonErr == nil && resp.DefaultBranchRef.Name != "" {
			return resp.DefaultBranchRef.Name, nil
		}
	}

	// Fallback: git remote show origin
	out, err = m.Runner.Run(ctx, "git", "remote", "show", "origin")
	if err != nil {
		return "", fmt.Errorf("detect default branch: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "HEAD branch:") {
			branch := strings.TrimSpace(strings.TrimPrefix(line, "HEAD branch:"))
			if branch != "" {
				return branch, nil
			}
		}
	}
	return "", fmt.Errorf("could not detect default branch from remote")
}

// Create creates a git worktree at .claude/worktrees/<branchName> branched from origin/<defaultBranch>.
// It also copies .claude/ config files (settings.json, settings.local.json, rules/) into the worktree.
func (m *Manager) Create(ctx context.Context, branchName string) (string, error) {
	defaultBranch, err := m.DefaultBranch(ctx)
	if err != nil {
		return "", fmt.Errorf("create worktree: %w", err)
	}

	// Fetch the default branch
	if _, err := m.Runner.Run(ctx, "git", "fetch", "origin", defaultBranch); err != nil {
		return "", fmt.Errorf("git fetch origin %s: %w", defaultBranch, err)
	}

	// Create the worktree
	worktreePath := filepath.Join(".claude", "worktrees", branchName)
	startPoint := "origin/" + defaultBranch
	if _, err := m.Runner.Run(ctx, "git", "worktree", "add", worktreePath, "-b", branchName, startPoint); err != nil {
		return "", fmt.Errorf("git worktree add: %w", err)
	}

	// Copy .claude/ config files from repo root into worktree (non-fatal)
	if err := m.copyClaudeConfig(worktreePath); err != nil {
		fmt.Fprintf(os.Stderr, "warn: copy .claude/ config: %v\n", err)
	}

	return worktreePath, nil
}

// copyClaudeConfig copies selected .claude/ files from the repo root into the worktree.
// Copies: settings.json, settings.local.json, rules/
// Skips: worktrees/, conversations/, projects/ (session-specific)
func (m *Manager) copyClaudeConfig(worktreePath string) error {
	srcClaudeDir := filepath.Join(m.RepoRoot, ".claude")
	dstClaudeDir := filepath.Join(worktreePath, ".claude")

	if _, err := os.Stat(srcClaudeDir); os.IsNotExist(err) {
		return nil
	}

	allowlist := map[string]bool{
		"settings.json":       true,
		"settings.local.json": true,
		"rules":               true,
	}

	entries, err := os.ReadDir(srcClaudeDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !allowlist[entry.Name()] {
			continue
		}
		src := filepath.Join(srcClaudeDir, entry.Name())
		dst := filepath.Join(dstClaudeDir, entry.Name())
		if entry.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return err
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				return err
			}
		}
	}
	return nil
}

// Remove removes a git worktree and its branch.
// It looks up the branch name from `git worktree list --porcelain` before removal
// to correctly handle branch names containing path separators (e.g., "fix/issue-42").
func (m *Manager) Remove(ctx context.Context, worktreePath string) error {
	// Look up the branch name before removing the worktree, because
	// filepath.Base() would incorrectly truncate "fix/issue-42" to "issue-42".
	branchName := m.branchForWorktree(ctx, worktreePath)

	if _, err := m.Runner.Run(ctx, "git", "worktree", "remove", worktreePath, "--force"); err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}
	// Best-effort branch deletion using the real branch name
	if branchName != "" {
		m.Runner.Run(ctx, "git", "branch", "-d", branchName) //nolint:errcheck
	}
	return nil
}

// branchForWorktree looks up the branch name for a worktree path by parsing
// `git worktree list --porcelain`. Returns empty string if not found.
func (m *Manager) branchForWorktree(ctx context.Context, worktreePath string) string {
	out, err := m.Runner.Run(ctx, "git", "worktree", "list", "--porcelain")
	if err != nil {
		return ""
	}
	// Normalize the target path for comparison
	absTarget := worktreePath
	if !filepath.IsAbs(worktreePath) {
		absTarget = filepath.Join(m.RepoRoot, worktreePath)
	}
	absTarget = filepath.Clean(absTarget)

	for _, wt := range parsePorcelain(string(out)) {
		candidate := filepath.Clean(wt.Path)
		if candidate == absTarget {
			return wt.Branch
		}
	}
	return ""
}

// List returns all git worktrees by parsing `git worktree list --porcelain`.
func (m *Manager) List(ctx context.Context) ([]WorktreeInfo, error) {
	out, err := m.Runner.Run(ctx, "git", "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	return parsePorcelain(string(out)), nil
}

// ListPitCrew returns only worktrees under .claude/worktrees/.
func (m *Manager) ListPitCrew(ctx context.Context) ([]WorktreeInfo, error) {
	all, err := m.List(ctx)
	if err != nil {
		return nil, err
	}
	var out []WorktreeInfo
	for _, wt := range all {
		if strings.Contains(filepath.ToSlash(wt.Path), ".claude/worktrees/") {
			out = append(out, wt)
		}
	}
	return out, nil
}

// parsePorcelain parses `git worktree list --porcelain` output.
func parsePorcelain(output string) []WorktreeInfo {
	var results []WorktreeInfo
	var current WorktreeInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current.Path != "" {
				results = append(results, current)
			}
			current = WorktreeInfo{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			current.HeadCommit = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		}
	}
	if current.Path != "" {
		results = append(results, current)
	}
	return results
}

// copyFile copies a single file src → dst, creating parent dirs as needed.
// It preserves the source file's permissions.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// copyDir recursively copies a directory src → dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath)
	})
}
