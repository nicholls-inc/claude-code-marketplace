package main

import (
	"bytes"
	"io"
	"os"
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

	old := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw

	dryRunDrain(cfg, q)

	pw.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, pr) //nolint:errcheck
	out := buf.String()

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

func TestDrainDryRunEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := makeDrainConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	old := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw

	dryRunDrain(cfg, q)

	pw.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, pr) //nolint:errcheck
	out := buf.String()

	if !strings.Contains(out, "No pending") {
		t.Errorf("expected empty message, got: %s", out)
	}
}
