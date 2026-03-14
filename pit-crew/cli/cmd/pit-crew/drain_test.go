package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

func makeDrainConfig(dir string) *config.Config {
	return &config.Config{
		Repo:        "owner/repo",
		Concurrency: 2,
		MaxTurns:    50,
		Timeout:     "30m",
		StateDir:    dir,
		Exclude:     []string{"wontfix"},
		Claude:      config.ClaudeConfig{Command: "claude", Template: "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}"},
		Tasks:       map[string]config.Task{"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"}},
	}
}

func TestDrainDryRun(t *testing.T) {
	dir := t.TempDir()
	cfg := makeDrainConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	now := time.Now().UTC()
	q.Enqueue(queue.Job{ //nolint:errcheck
		ID: "issue-1", IssueNum: 1,
		IssueURL:  "https://github.com/owner/repo/issues/1",
		Skill:     "fix-bug",
		State:     queue.StatePending,
		CreatedAt: now,
	})
	q.Enqueue(queue.Job{ //nolint:errcheck
		ID: "issue-2", IssueNum: 2,
		IssueURL:  "https://github.com/owner/repo/issues/2",
		Skill:     "fix-bug",
		State:     queue.StatePending,
		CreatedAt: now,
	})

	out := captureStdout(func() {
		err := dryRunDrain(cfg, q)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	jobs, _ := q.ListByState(queue.StatePending)
	if len(jobs) != 2 {
		t.Errorf("dry-run should not drain queue, got %d pending", len(jobs))
	}
	if !strings.Contains(out, "issue-1") {
		t.Errorf("expected job in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "dry-run") {
		t.Errorf("expected dry-run notice in output, got: %s", out)
	}
}

func TestDrainDryRunCommandFormat(t *testing.T) {
	dir := t.TempDir()
	cfg := makeDrainConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	now := time.Now().UTC()
	q.Enqueue(queue.Job{ //nolint:errcheck
		ID: "issue-1", IssueNum: 1,
		IssueURL:  "https://github.com/owner/repo/issues/1",
		Skill:     "fix-bug",
		State:     queue.StatePending,
		CreatedAt: now,
	})

	out := captureStdout(func() {
		err := dryRunDrain(cfg, q)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Verify the exact command template rendering
	expectedCmd := `claude -p "/fix-bug https://github.com/owner/repo/issues/1" --max-turns 50`
	if !strings.Contains(out, expectedCmd) {
		t.Errorf("expected command %q in output, got: %s", expectedCmd, out)
	}
	// Verify table headers
	if !strings.Contains(out, "ID") || !strings.Contains(out, "Issue") ||
		!strings.Contains(out, "Skill") || !strings.Contains(out, "Command") {
		t.Errorf("expected table headers (ID, Issue, Skill, Command), got: %s", out)
	}
	// Verify count message
	if !strings.Contains(out, "1 job(s) would be drained") {
		t.Errorf("expected count message, got: %s", out)
	}
}

func TestDrainDryRunEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := makeDrainConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	out := captureStdout(func() {
		err := dryRunDrain(cfg, q)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "No pending") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestDrainDryRunQueueReadError(t *testing.T) {
	dir := t.TempDir()
	cfg := makeDrainConfig(dir)
	// Point to a directory instead of a file to force a read error
	q := queue.New(dir)

	err := dryRunDrain(cfg, q)
	if err == nil {
		t.Fatal("expected error from dryRunDrain with bad queue path")
	}
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitError, got %T: %v", err, err)
	}
	if ee.code != 2 {
		t.Errorf("expected exit code 2, got %d", ee.code)
	}
}

func TestExitErrorCodes(t *testing.T) {
	// Test exitError type directly to verify code semantics
	t.Run("code2_with_wrapped_error", func(t *testing.T) {
		inner := errors.New("drain error: something went wrong")
		ee := &exitError{code: 2, err: inner}
		if ee.code != 2 {
			t.Errorf("expected code 2, got %d", ee.code)
		}
		if ee.Error() != "drain error: something went wrong" {
			t.Errorf("unexpected error message: %s", ee.Error())
		}
		if !errors.Is(ee.Unwrap(), inner) {
			t.Error("expected Unwrap to return inner error")
		}
	})

	t.Run("code1_no_wrapped_error", func(t *testing.T) {
		ee := &exitError{code: 1}
		if ee.code != 1 {
			t.Errorf("expected code 1, got %d", ee.code)
		}
		if ee.Error() != "exit code 1" {
			t.Errorf("unexpected error message: %s", ee.Error())
		}
		if ee.Unwrap() != nil {
			t.Error("expected Unwrap to return nil")
		}
	})
}
