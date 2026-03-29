package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
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

func testCleanupConfig(t *testing.T) (*config.Config, *queue.Queue) {
	t.Helper()
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".xylem")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}
	cfg := &config.Config{StateDir: stateDir}
	q := queue.New(filepath.Join(stateDir, "queue.jsonl"))
	return cfg, q
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
			cfg := &config.Config{StateDir: filepath.Join(dir, ".xylem")}
			q := queue.New(filepath.Join(cfg.StateDir, "queue.jsonl"))
			wt := worktree.New(dir, &emptyWorktreeRunner{})

			var err error
			out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, tt.dryRun) })
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
	cfg := &config.Config{StateDir: "/nonexistent/.xylem"}
	q := queue.New(filepath.Join(cfg.StateDir, "queue.jsonl"))

	var err error
	out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, true) })
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
	cfg := &config.Config{StateDir: "/nonexistent/.xylem"}
	q := queue.New(filepath.Join(cfg.StateDir, "queue.jsonl"))

	var err error
	out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, false) })
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
	cfg := &config.Config{StateDir: "/nonexistent/.xylem"}
	q := queue.New(filepath.Join(cfg.StateDir, "queue.jsonl"))

	// Capture stderr too to verify error message is logged
	oldErr := os.Stderr
	errPr, errPw, _ := os.Pipe()
	os.Stderr = errPw

	var err error
	out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, false) })

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
	cfg := &config.Config{StateDir: "/nonexistent/.xylem"}
	q := queue.New(filepath.Join(cfg.StateDir, "queue.jsonl"))

	var err error
	out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, false) })
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

// --- Phase output cleanup tests ---

func TestCleanupPhaseDirs(t *testing.T) {
	cfg, q := testCleanupConfig(t)
	wt := worktree.New(t.TempDir(), &emptyWorktreeRunner{})

	// Create phases directory with a vessel's output
	phaseDir := filepath.Join(cfg.StateDir, "phases", "issue-1")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatalf("failed to create phase dir: %v", err)
	}
	// Write a dummy file inside
	if err := os.WriteFile(filepath.Join(phaseDir, "output.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write phase output: %v", err)
	}

	// Create a completed vessel that ended 8 days ago
	now := time.Now().UTC()
	started := now.Add(-9 * 24 * time.Hour)
	ended := now.Add(-8 * 24 * time.Hour)
	q.Enqueue(queue.Vessel{ //nolint:errcheck
		ID: "issue-1", Source: "github-issue", Workflow: "fix-bug",
		State: queue.StateCompleted, CreatedAt: started,
		StartedAt: &started, EndedAt: &ended,
	})

	var err error
	out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, false) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory was removed
	if _, statErr := os.Stat(phaseDir); !os.IsNotExist(statErr) {
		t.Errorf("expected phase dir to be removed, but it still exists")
	}
	if !strings.Contains(out, "Removed phase outputs") {
		t.Errorf("expected 'Removed phase outputs' in output, got: %s", out)
	}
	if !strings.Contains(out, "Cleaned up 1 phase output directory") {
		t.Errorf("expected cleanup count in output, got: %s", out)
	}
}

func TestCleanupSkipsRecentPhaseDirs(t *testing.T) {
	cfg, q := testCleanupConfig(t)
	wt := worktree.New(t.TempDir(), &emptyWorktreeRunner{})

	// Create phases directory
	phaseDir := filepath.Join(cfg.StateDir, "phases", "issue-2")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatalf("failed to create phase dir: %v", err)
	}

	// Create a completed vessel that ended only 1 day ago
	now := time.Now().UTC()
	started := now.Add(-2 * 24 * time.Hour)
	ended := now.Add(-1 * 24 * time.Hour)
	q.Enqueue(queue.Vessel{ //nolint:errcheck
		ID: "issue-2", Source: "github-issue", Workflow: "fix-bug",
		State: queue.StateCompleted, CreatedAt: started,
		StartedAt: &started, EndedAt: &ended,
	})

	var err error
	out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, false) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory still exists
	if _, statErr := os.Stat(phaseDir); os.IsNotExist(statErr) {
		t.Errorf("expected phase dir to still exist (too recent), but it was removed")
	}
	if strings.Contains(out, "Removed phase outputs") {
		t.Errorf("expected no phase removal output, got: %s", out)
	}
}

func TestCleanupDryRunPhaseDirs(t *testing.T) {
	cfg, q := testCleanupConfig(t)
	wt := worktree.New(t.TempDir(), &emptyWorktreeRunner{})

	// Create phases directory
	phaseDir := filepath.Join(cfg.StateDir, "phases", "issue-3")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatalf("failed to create phase dir: %v", err)
	}

	// Create a failed vessel that ended 10 days ago
	now := time.Now().UTC()
	started := now.Add(-11 * 24 * time.Hour)
	ended := now.Add(-10 * 24 * time.Hour)
	q.Enqueue(queue.Vessel{ //nolint:errcheck
		ID: "issue-3", Source: "github-issue", Workflow: "fix-bug",
		State: queue.StateFailed, CreatedAt: started,
		StartedAt: &started, EndedAt: &ended,
	})

	var err error
	out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, true) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory still exists (dry-run)
	if _, statErr := os.Stat(phaseDir); os.IsNotExist(statErr) {
		t.Errorf("expected phase dir to still exist in dry-run, but it was removed")
	}
	if !strings.Contains(out, "Would remove phase outputs") {
		t.Errorf("expected 'Would remove phase outputs' in dry-run output, got: %s", out)
	}
}

func TestCleanupNoPhasesDir(t *testing.T) {
	cfg, q := testCleanupConfig(t)
	wt := worktree.New(t.TempDir(), &emptyWorktreeRunner{})

	// Do NOT create the phases directory -- it should not exist
	var err error
	out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, false) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should succeed without error and not mention phase cleanup
	if strings.Contains(out, "phase") {
		t.Errorf("expected no phase-related output when phases dir doesn't exist, got: %s", out)
	}
}

func TestCleanupTimedOutPhaseDirs(t *testing.T) {
	cfg, q := testCleanupConfig(t)
	wt := worktree.New(t.TempDir(), &emptyWorktreeRunner{})

	// Create phases directory for timed_out vessel
	phaseDir := filepath.Join(cfg.StateDir, "phases", "issue-4")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatalf("failed to create phase dir: %v", err)
	}

	// Create a timed_out vessel that ended 8 days ago
	now := time.Now().UTC()
	started := now.Add(-9 * 24 * time.Hour)
	ended := now.Add(-8 * 24 * time.Hour)
	q.Enqueue(queue.Vessel{ //nolint:errcheck
		ID: "issue-4", Source: "github-issue", Workflow: "fix-bug",
		State: queue.StateTimedOut, CreatedAt: started,
		StartedAt: &started, EndedAt: &ended,
	})

	var err error
	out := captureStdout(func() { err = cmdCleanup(cfg, q, wt, false) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory was removed
	if _, statErr := os.Stat(phaseDir); !os.IsNotExist(statErr) {
		t.Errorf("expected phase dir for timed_out vessel to be removed, but it still exists")
	}
	if !strings.Contains(out, "Removed phase outputs") {
		t.Errorf("expected 'Removed phase outputs' in output, got: %s", out)
	}
}
