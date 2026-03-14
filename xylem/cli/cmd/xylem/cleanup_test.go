package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/worktree"
)

type emptyWorktreeRunner struct{}

func (e *emptyWorktreeRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return []byte{}, nil
}

// mockCleanupRunner returns porcelain output for worktree list and tracks
// worktree remove calls.
type mockCleanupRunner struct {
	porcelain   string
	removeCalls []string
	removeErr   error
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
	tests := []struct {
		name   string
		dryRun bool
	}{
		{"actual", false},
		{"dry-run", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			wt := worktree.New(dir, &emptyWorktreeRunner{})

			var err error
			out := captureStdout(func() { err = cmdCleanup(wt, tt.dryRun) })
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out, "No xylem worktrees") {
				t.Errorf("expected empty message, got: %s", out)
			}
		})
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

	var err error
	out := captureStdout(func() { err = cmdCleanup(wt, true) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	var err error
	out := captureStdout(func() { err = cmdCleanup(wt, false) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Removed") {
		t.Errorf("expected 'Removed' in output, got: %s", out)
	}
	if !strings.Contains(out, "1 worktree(s)") {
		t.Errorf("expected removal count, got: %s", out)
	}
	// Verify remove was actually called
	if len(r.removeCalls) != 1 {
		t.Errorf("expected 1 remove call, got %d", len(r.removeCalls))
	}
	if len(r.removeCalls) > 0 && !strings.Contains(r.removeCalls[0], "issue-5-test") {
		t.Errorf("expected remove call for issue-5-test worktree, got: %s", r.removeCalls[0])
	}
}

func TestCleanupRemovalError(t *testing.T) {
	porcelain := strings.Join([]string{
		"worktree /repo",
		"HEAD aaa",
		"branch refs/heads/main",
		"",
		"worktree /repo/.claude/worktrees/fix/issue-7-broken",
		"HEAD bbb",
		"branch refs/heads/fix/issue-7-broken",
		"",
	}, "\n")

	r := &mockCleanupRunner{
		porcelain: porcelain,
		removeErr: errors.New("permission denied"),
	}
	wt := worktree.New("/repo", r)

	// Capture stderr too to verify error message is logged
	oldErr := os.Stderr
	errPr, errPw, _ := os.Pipe()
	os.Stderr = errPw

	var err error
	out := captureStdout(func() { err = cmdCleanup(wt, false) })

	errPw.Close()
	os.Stderr = oldErr
	var errBuf bytes.Buffer
	io.Copy(&errBuf, errPr) //nolint:errcheck
	stderrOut := errBuf.String()

	// cmdCleanup returns nil (best-effort removal)
	if err != nil {
		t.Fatalf("expected nil error (best-effort), got: %v", err)
	}
	// Verify error was logged to stderr
	if !strings.Contains(stderrOut, "error removing") {
		t.Errorf("expected error logged to stderr, got: %s", stderrOut)
	}
	// Count should be 0 (failed removal doesn't increment)
	if strings.Contains(out, "1 worktree(s)") {
		t.Errorf("expected 0 removed (not 1) after error, got: %s", out)
	}
}

func TestCleanupPartialFailure(t *testing.T) {
	porcelain := strings.Join([]string{
		"worktree /repo",
		"HEAD aaa",
		"branch refs/heads/main",
		"",
		"worktree /repo/.claude/worktrees/fix/issue-1-ok",
		"HEAD bbb",
		"branch refs/heads/fix/issue-1-ok",
		"",
		"worktree /repo/.claude/worktrees/fix/issue-2-fail",
		"HEAD ccc",
		"branch refs/heads/fix/issue-2-fail",
		"",
	}, "\n")

	// Use a runner that fails only for the second worktree
	callCount := 0
	r := &partialFailRunner{porcelain: porcelain, failOnCall: 2, callCount: &callCount}
	wt := worktree.New("/repo", r)

	var err error
	out := captureStdout(func() { err = cmdCleanup(wt, false) })
	if err != nil {
		t.Fatalf("expected nil error (best-effort), got: %v", err)
	}

	// Only 1 of 2 should be removed
	if !strings.Contains(out, "1 worktree(s)") {
		t.Errorf("expected 1 worktree removed (partial), got: %s", out)
	}
}

// partialFailRunner fails on a specific remove call number.
type partialFailRunner struct {
	porcelain  string
	failOnCall int
	callCount  *int
}

func (m *partialFailRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	all := append([]string{name}, args...)
	key := strings.Join(all, " ")
	if strings.Contains(key, "worktree list --porcelain") {
		return []byte(m.porcelain), nil
	}
	if strings.Contains(key, "worktree remove") {
		*m.callCount++
		if *m.callCount == m.failOnCall {
			return []byte{}, errors.New("remove failed")
		}
		return []byte{}, nil
	}
	if strings.Contains(key, "branch -d") {
		return []byte{}, nil
	}
	return []byte{}, nil
}
