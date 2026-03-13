package main

import (
	"context"
	"strings"
	"testing"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/worktree"
)

type emptyWorktreeRunner struct{}

func (e *emptyWorktreeRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return []byte{}, nil
}

// mockCleanupRunner returns porcelain output for worktree list and tracks
// worktree remove calls.
type mockCleanupRunner struct {
	porcelain    string
	removeCalls  []string
	removeErr    error
}

func (m *mockCleanupRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	all := append([]string{name}, args...)
	key := strings.Join(all, " ")
	if strings.Contains(key, "worktree list --porcelain") {
		return []byte(m.porcelain), nil
	}
	if strings.Contains(key, "worktree remove") {
		m.removeCalls = append(m.removeCalls, key)
		return []byte{}, m.removeErr
	}
	// branch -d is best-effort
	if strings.Contains(key, "branch -d") {
		return []byte{}, nil
	}
	return []byte{}, nil
}

func TestCleanupNoWorktrees(t *testing.T) {
	dir := t.TempDir()
	wt := worktree.New(dir, &emptyWorktreeRunner{})

	out := captureStdout(func() { cmdCleanup(wt, nil) })
	if !strings.Contains(out, "No pit-crew worktrees") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestCleanupDryRunNoWorktrees(t *testing.T) {
	dir := t.TempDir()
	wt := worktree.New(dir, &emptyWorktreeRunner{})

	out := captureStdout(func() { cmdCleanup(wt, []string{"--dry-run"}) })
	if !strings.Contains(out, "No pit-crew worktrees") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestCleanupDryRunWithWorktrees(t *testing.T) {
	porcelain := strings.Join([]string{
		"worktree /repo",
		"HEAD aaa",
		"branch refs/heads/main",
		"",
		"worktree /repo/.claude/worktrees/fix/issue-1-bug",
		"HEAD bbb",
		"branch refs/heads/fix/issue-1-bug",
		"",
		"worktree /repo/.claude/worktrees/feat/issue-2-feature",
		"HEAD ccc",
		"branch refs/heads/feat/issue-2-feature",
		"",
	}, "\n")

	r := &mockCleanupRunner{porcelain: porcelain}
	wt := worktree.New("/repo", r)

	out := captureStdout(func() { cmdCleanup(wt, []string{"--dry-run"}) })

	if !strings.Contains(out, "Would remove") {
		t.Errorf("expected 'Would remove' in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "2 worktree(s) would be removed") {
		t.Errorf("expected count of 2 worktrees, got: %s", out)
	}
	// No actual remove calls should have been made
	if len(r.removeCalls) != 0 {
		t.Errorf("dry-run should not make remove calls, got %d", len(r.removeCalls))
	}
}

func TestCleanupActualRemoval(t *testing.T) {
	porcelain := strings.Join([]string{
		"worktree /repo",
		"HEAD aaa",
		"branch refs/heads/main",
		"",
		"worktree /repo/.claude/worktrees/fix/issue-5-test",
		"HEAD bbb",
		"branch refs/heads/fix/issue-5-test",
		"",
	}, "\n")

	r := &mockCleanupRunner{porcelain: porcelain}
	wt := worktree.New("/repo", r)

	out := captureStdout(func() { cmdCleanup(wt, nil) })

	if !strings.Contains(out, "Removed") {
		t.Errorf("expected 'Removed' in output, got: %s", out)
	}
	if !strings.Contains(out, "1 worktree(s)") {
		t.Errorf("expected removal count, got: %s", out)
	}
}
