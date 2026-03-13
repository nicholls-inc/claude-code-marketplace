package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw
	fn()
	pw.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, pr) //nolint:errcheck
	return buf.String()
}

func TestStatusEmpty(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	out := captureStdout(func() { cmdStatus(q, nil) })
	if !strings.Contains(out, "No jobs") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestStatusTable(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	q.Enqueue(queue.Job{ID: "issue-42", IssueNum: 42, Skill: "fix-bug", State: queue.StatePending, CreatedAt: now})   //nolint:errcheck
	q.Enqueue(queue.Job{ID: "issue-55", IssueNum: 55, Skill: "fix-bug", State: queue.StateCompleted, CreatedAt: now}) //nolint:errcheck

	out := captureStdout(func() { cmdStatus(q, nil) })
	if !strings.Contains(out, "issue-42") {
		t.Errorf("expected issue-42 in output, got: %s", out)
	}
	if !strings.Contains(out, "issue-55") {
		t.Errorf("expected issue-55 in output, got: %s", out)
	}
	if !strings.Contains(out, "Summary:") {
		t.Errorf("expected summary line, got: %s", out)
	}
}

func TestStatusJSON(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	q.Enqueue(queue.Job{ID: "issue-1", IssueNum: 1, Skill: "fix-bug", State: queue.StatePending, CreatedAt: now}) //nolint:errcheck

	out := captureStdout(func() { cmdStatus(q, []string{"--json"}) })
	var jobs []queue.Job
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &jobs); err != nil {
		t.Fatalf("expected valid JSON, got: %s\nerr: %v", out, err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job in JSON, got %d", len(jobs))
	}
}

func TestStatusStateFilter(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	q.Enqueue(queue.Job{ID: "issue-1", IssueNum: 1, Skill: "fix-bug", State: queue.StatePending, CreatedAt: now})   //nolint:errcheck
	q.Enqueue(queue.Job{ID: "issue-2", IssueNum: 2, Skill: "fix-bug", State: queue.StateCompleted, CreatedAt: now}) //nolint:errcheck

	out := captureStdout(func() { cmdStatus(q, []string{"--state", "pending"}) })
	if !strings.Contains(out, "issue-1") {
		t.Errorf("expected issue-1 in filtered output, got: %s", out)
	}
	if strings.Contains(out, "issue-2") {
		t.Errorf("expected issue-2 filtered out, got: %s", out)
	}
}
