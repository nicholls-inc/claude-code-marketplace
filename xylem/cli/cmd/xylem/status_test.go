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

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
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

func testStatusVessel(id string, state queue.VesselState) queue.Vessel {
	return queue.Vessel{
		ID: id, Source: "github-issue", Skill: "fix-bug",
		State: state, CreatedAt: time.Now().UTC(),
	}
}

func TestStatusEmpty(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	var err error
	out := captureStdout(func() { err = cmdStatus(q, false, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No vessels") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestStatusTable(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	q.Enqueue(testStatusVessel("issue-42", queue.StatePending))   //nolint:errcheck
	q.Enqueue(testStatusVessel("issue-55", queue.StateCompleted)) //nolint:errcheck

	var err error
	out := captureStdout(func() { err = cmdStatus(q, false, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "issue-42") {
		t.Errorf("expected issue-42 in output, got: %s", out)
	}
	if !strings.Contains(out, "issue-55") {
		t.Errorf("expected issue-55 in output, got: %s", out)
	}
	if !strings.Contains(out, "pending") {
		t.Errorf("expected 'pending' state in output, got: %s", out)
	}
	if !strings.Contains(out, "completed") {
		t.Errorf("expected 'completed' state in output, got: %s", out)
	}
	if !strings.Contains(out, "Summary:") {
		t.Errorf("expected summary line, got: %s", out)
	}
}

func TestStatusTableHeaders(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	q.Enqueue(testStatusVessel("issue-1", queue.StatePending)) //nolint:errcheck

	var err error
	out := captureStdout(func() { err = cmdStatus(q, false, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, header := range []string{"ID", "Source", "Skill", "State", "Started", "Duration"} {
		if !strings.Contains(out, header) {
			t.Errorf("expected header %q in table output, got: %s", header, out)
		}
	}
}

func TestStatusJSON(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	q.Enqueue(testStatusVessel("issue-1", queue.StatePending)) //nolint:errcheck

	var err error
	out := captureStdout(func() { err = cmdStatus(q, true, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var vessels []queue.Vessel
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &vessels); err != nil {
		t.Fatalf("expected valid JSON, got: %s\nerr: %v", out, err)
	}
	if len(vessels) != 1 {
		t.Errorf("expected 1 vessel in JSON, got %d", len(vessels))
	}
}

func TestStatusStateFilter(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	q.Enqueue(testStatusVessel("issue-1", queue.StatePending))   //nolint:errcheck
	q.Enqueue(testStatusVessel("issue-2", queue.StateCompleted)) //nolint:errcheck

	var err error
	out := captureStdout(func() { err = cmdStatus(q, false, "pending") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "issue-1") {
		t.Errorf("expected issue-1 in filtered output, got: %s", out)
	}
	if strings.Contains(out, "issue-2") {
		t.Errorf("expected issue-2 filtered out, got: %s", out)
	}
}

func TestStatusJSONEmpty(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	var err error
	out := captureStdout(func() { err = cmdStatus(q, true, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trimmed := strings.TrimSpace(out)
	if trimmed != "[]" {
		t.Errorf("expected '[]' for empty JSON output, got: %q", trimmed)
	}
}

func TestStatusRunningVesselShowsDuration(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	started := now.Add(-2 * time.Minute)
	q.Enqueue(queue.Vessel{ //nolint:errcheck
		ID: "issue-10", Source: "github-issue", Skill: "fix-bug",
		State: queue.StateRunning, CreatedAt: now, StartedAt: &started,
	})

	var err error
	out := captureStdout(func() { err = cmdStatus(q, false, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "issue-10") {
		t.Errorf("expected issue-10 in output, got: %s", out)
	}
	if !strings.Contains(out, "running") {
		t.Errorf("expected 'running' state in output, got: %s", out)
	}
	if !strings.Contains(out, "2m") {
		t.Errorf("expected duration ~2m in output, got: %s", out)
	}
}

func TestStatusCompletedVesselShowsFixedDuration(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	started := now.Add(-5 * time.Minute)
	ended := now.Add(-2 * time.Minute)
	q.Enqueue(queue.Vessel{ //nolint:errcheck
		ID: "issue-20", Source: "github-issue", Skill: "fix-bug",
		State: queue.StateCompleted, CreatedAt: now,
		StartedAt: &started, EndedAt: &ended,
	})

	var err error
	out := captureStdout(func() { err = cmdStatus(q, false, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "3m0s") {
		t.Errorf("expected duration '3m0s' in output, got: %s", out)
	}
}

func TestStatusFailedVessel(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	started := now.Add(-1 * time.Minute)
	ended := now.Add(-30 * time.Second)
	q.Enqueue(queue.Vessel{ //nolint:errcheck
		ID: "issue-99", Source: "github-issue", Skill: "fix-bug",
		State: queue.StateFailed, CreatedAt: now,
		StartedAt: &started, EndedAt: &ended,
		Error: "compilation failed",
	})

	var err error
	out := captureStdout(func() { err = cmdStatus(q, false, "") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "failed") {
		t.Errorf("expected 'failed' state in output, got: %s", out)
	}
	if !strings.Contains(out, "issue-99") {
		t.Errorf("expected issue-99 in output, got: %s", out)
	}
}

func TestStatusStateFilterNoMatches(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	q.Enqueue(testStatusVessel("issue-1", queue.StatePending)) //nolint:errcheck

	var err error
	out := captureStdout(func() { err = cmdStatus(q, false, "completed") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No vessels") {
		t.Errorf("expected 'No vessels' for filtered empty result, got: %s", out)
	}
}
