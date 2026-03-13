package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/scanner"
)

type mockScanRunner struct {
	outputs map[string][]byte
}

func newScanMock() *mockScanRunner {
	return &mockScanRunner{outputs: make(map[string][]byte)}
}

func (m *mockScanRunner) set(out []byte, args ...string) {
	m.outputs[strings.Join(args, " ")] = out
}

func (m *mockScanRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	if out, ok := m.outputs[key]; ok {
		return out, nil
	}
	return []byte("[]"), nil
}

func makeScanConfig(dir string) *config.Config {
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

type ghIssueJSON struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func issuesJSON(issues []ghIssueJSON) []byte {
	b, _ := json.Marshal(issues)
	return b
}

func TestScanDryRun(t *testing.T) {
	dir := t.TempDir()
	cfg := makeScanConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newScanMock()

	issues := []ghIssueJSON{
		{Number: 1, Title: "fix null", URL: "https://github.com/owner/repo/issues/1",
			Labels: []struct {
				Name string `json:"name"`
			}{{Name: "bug"}}},
	}
	r.set(issuesJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")

	old := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw

	dryRunScan(cfg, q, r)

	pw.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, pr) //nolint:errcheck
	out := buf.String()

	jobs, _ := q.List()
	if len(jobs) != 0 {
		t.Errorf("dry-run should not write to queue, got %d jobs", len(jobs))
	}
	if !strings.Contains(out, "issue-1") && !strings.Contains(out, "#1") {
		t.Errorf("expected issue in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "dry-run") {
		t.Errorf("expected dry-run notice in output, got: %s", out)
	}
}

func TestScanNormalMode(t *testing.T) {
	dir := t.TempDir()
	cfg := makeScanConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newScanMock()

	issues := []ghIssueJSON{
		{Number: 2, Title: "another bug", URL: "https://github.com/owner/repo/issues/2",
			Labels: []struct {
				Name string `json:"name"`
			}{{Name: "bug"}}},
	}
	r.set(issuesJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")

	s := scanner.New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 1 {
		t.Errorf("expected 1 added, got %d", result.Added)
	}
	jobs, _ := q.List()
	if len(jobs) != 1 {
		t.Errorf("expected 1 job in queue, got %d", len(jobs))
	}
}

func TestScanPausedOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := makeScanConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newScanMock()

	os.WriteFile(filepath.Join(dir, "paused"), []byte{}, 0o644) //nolint:errcheck

	s := scanner.New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Paused {
		t.Error("expected Paused=true")
	}
}

func TestScanDryRunEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := makeScanConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newScanMock()

	old := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw

	dryRunScan(cfg, q, r)

	pw.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, pr) //nolint:errcheck
	out := buf.String()

	if !strings.Contains(out, "No new issues") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func init() {
	_ = fmt.Sprintf
	_ = time.Now
}
