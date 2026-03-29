package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
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

func TestCancelPendingVessel(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	q.Enqueue(queue.Vessel{ID: "issue-1", Source: "github-issue", Workflow: "fix-bug", State: queue.StatePending, CreatedAt: now}) //nolint:errcheck

	if err := cmdCancel(q, "issue-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vessels, _ := q.List()
	if vessels[0].State != queue.StateCancelled {
		t.Errorf("expected cancelled, got %s", vessels[0].State)
	}
}

func TestCancelNonExistentVessel(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	err := cmdCancel(q, "issue-999")
	if err == nil {
		t.Fatal("expected error cancelling non-existent vessel")
	}
	if !strings.Contains(err.Error(), "cancel error:") {
		t.Errorf("expected wrapped 'cancel error:', got: %v", err)
	}
}

