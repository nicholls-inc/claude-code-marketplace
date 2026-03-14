package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

func makeControlConfig(dir string) *config.Config {
	return &config.Config{
		Repo:     "owner/repo",
		StateDir: dir,
		Exclude:  []string{},
		Tasks:    map[string]config.Task{},
	}
}

func TestPauseCreatesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	if err := cmdPause(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "paused")); err != nil {
		t.Error("expected pause marker to exist")
	}
}

func TestPauseOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	out := captureStdout(func() {
		if err := cmdPause(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Scanning paused.") {
		t.Errorf("expected 'Scanning paused.' in output, got: %s", out)
	}
}

func TestPauseIdempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	if err := cmdPause(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := captureStdout(func() {
		if err := cmdPause(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Already paused.") {
		t.Errorf("expected 'Already paused.' on second call, got: %s", out)
	}

	if _, err := os.Stat(filepath.Join(dir, "paused")); err != nil {
		t.Error("expected pause marker to still exist after double pause")
	}
}

func TestResumeRemovesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	if err := cmdPause(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cmdResume(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "paused")); err == nil {
		t.Error("expected pause marker to be removed")
	}
}

func TestResumeOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	// Pause first, then resume
	if err := cmdPause(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := captureStdout(func() {
		if err := cmdResume(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Scanning resumed.") {
		t.Errorf("expected 'Scanning resumed.' in output, got: %s", out)
	}
}

func TestResumeIdempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	out := captureStdout(func() {
		if err := cmdResume(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Not paused.") {
		t.Errorf("expected 'Not paused.' when not paused, got: %s", out)
	}

	// Verify no pause marker exists
	if _, err := os.Stat(filepath.Join(dir, "paused")); err == nil {
		t.Error("expected no pause marker to exist")
	}
}

func TestPauseResumeRoundtrip(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	if isPaused(cfg) {
		t.Error("should not be paused initially")
	}
	if err := cmdPause(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isPaused(cfg) {
		t.Error("should be paused after pause")
	}
	if err := cmdResume(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isPaused(cfg) {
		t.Error("should not be paused after resume")
	}
}

func TestCancelPendingJob(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	q.Enqueue(queue.Job{ID: "issue-1", IssueNum: 1, Skill: "fix-bug", State: queue.StatePending, CreatedAt: now}) //nolint:errcheck

	if err := cmdCancel(q, "issue-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jobs, _ := q.List()
	if jobs[0].State != queue.StateCancelled {
		t.Errorf("expected cancelled, got %s", jobs[0].State)
	}
}

func TestCancelOutput(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	q.Enqueue(queue.Job{ID: "issue-1", IssueNum: 1, Skill: "fix-bug", State: queue.StatePending, CreatedAt: now}) //nolint:errcheck

	out := captureStdout(func() {
		if err := cmdCancel(q, "issue-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Cancelled job issue-1") {
		t.Errorf("expected 'Cancelled job issue-1' in output, got: %s", out)
	}
}

func TestCancelNonExistentJob(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	err := cmdCancel(q, "issue-999")
	if err == nil {
		t.Fatal("expected error cancelling non-existent job")
	}
	if !strings.Contains(err.Error(), "cancel error:") {
		t.Errorf("expected wrapped 'cancel error:', got: %v", err)
	}
}

func TestCancelCompletedJob(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	started := now.Add(-1 * time.Minute)
	ended := now
	q.Enqueue(queue.Job{ //nolint:errcheck
		ID: "issue-1", IssueNum: 1, Skill: "fix-bug",
		State: queue.StateCompleted, CreatedAt: now,
		StartedAt: &started, EndedAt: &ended,
	})

	err := cmdCancel(q, "issue-1")
	if err == nil {
		t.Fatal("expected error cancelling completed job")
	}
	if !strings.Contains(err.Error(), "cancel error:") {
		t.Errorf("expected wrapped 'cancel error:', got: %v", err)
	}
}
