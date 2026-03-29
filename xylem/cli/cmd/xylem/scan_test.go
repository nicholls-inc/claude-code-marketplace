package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
)

type mockScanRunner struct {
	outputs map[string][]byte
	errs    map[string]error
}

func newScanMock() *mockScanRunner {
	return &mockScanRunner{
		outputs: make(map[string][]byte),
		errs:    make(map[string]error),
	}
}

func (m *mockScanRunner) set(out []byte, args ...string) {
	m.outputs[strings.Join(args, " ")] = out
}

func (m *mockScanRunner) setErr(err error, args ...string) {
	m.errs[strings.Join(args, " ")] = err
}

func (m *mockScanRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	if err, ok := m.errs[key]; ok {
		return nil, err
	}
	if out, ok := m.outputs[key]; ok {
		return out, nil
	}
	return nil, fmt.Errorf("unstubbed command: %s", key)
}

func makeScanConfig(dir string) *config.Config {
	return &config.Config{
		Repo:        "owner/repo",
		Concurrency: 2,
		MaxTurns:    50,
		Timeout:     "30m",
		StateDir:    dir,
		Exclude:     []string{"wontfix"},
		Claude: config.ClaudeConfig{
			Command: "claude",
		},
		Tasks: map[string]config.Task{"fix-bugs": {Labels: []string{"bug"}, Workflow: "fix-bug"}},
		Sources: map[string]config.SourceConfig{
			"github": {
				Type:    "github",
				Repo:    "owner/repo",
				Exclude: []string{"wontfix"},
				Tasks:   map[string]config.Task{"fix-bugs": {Labels: []string{"bug"}, Workflow: "fix-bug"}},
			},
		},
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

// stubScanCommands stubs all the gh/git commands that the scanner issues for
// a set of issues. This avoids hitting the "unstubbed command" error for
// the branch/PR filtering paths.
func stubScanCommands(r *mockScanRunner, cfg *config.Config, issues []ghIssueJSON) {
	ghSrc := cfg.Sources["github"]
	for _, task := range ghSrc.Tasks {
		r.set(issuesJSON(issues), "gh", "search", "issues", "--repo", ghSrc.Repo,
			"--state", "open", "--json", "number,title,url,labels",
			"--limit", "20", "--label", task.Labels[0])
	}
	// Stub branch checks and PR checks for each issue
	for _, issue := range issues {
		for _, prefix := range []string{"fix", "feat"} {
			// git ls-remote returns empty = no branch
			r.set([]byte(""), "git", "ls-remote", "--heads", "origin",
				fmt.Sprintf("%s/issue-%d-*", prefix, issue.Number))
			// gh pr list returns empty array = no PRs
			r.set([]byte("[]"), "gh", "pr", "list", "--repo", ghSrc.Repo,
				"--search", fmt.Sprintf("head:%s/issue-%d-", prefix, issue.Number),
				"--state", "open", "--json", "number,headRefName", "--limit", "5")
		}
	}
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
	stubScanCommands(r, cfg, issues)

	old := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw

	err := dryRunScan(cfg, q, r)

	pw.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, pr) //nolint:errcheck
	out := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vessels, _ := q.List()
	if len(vessels) != 0 {
		t.Errorf("dry-run should not write to queue, got %d vessels", len(vessels))
	}
	// Check table headers
	if !strings.Contains(out, "ID") || !strings.Contains(out, "Source") ||
		!strings.Contains(out, "Workflow") || !strings.Contains(out, "Ref") {
		t.Errorf("expected table headers (ID, Source, Workflow, Ref), got: %s", out)
	}
	// Check formatted issue row
	if !strings.Contains(out, "issue-1") {
		t.Errorf("expected issue-1 in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "fix-bug") {
		t.Errorf("expected workflow in dry-run output, got: %s", out)
	}
	// Check count message
	if !strings.Contains(out, "1 candidate(s) would be queued") {
		t.Errorf("expected count message, got: %s", out)
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
	stubScanCommands(r, cfg, issues)

	out := captureStdout(func() {
		err := cmdScan(cfg, q, r, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Added 1") {
		t.Errorf("expected 'Added 1' in output, got: %s", out)
	}

	vessels, _ := q.List()
	if len(vessels) != 1 {
		t.Errorf("expected 1 vessel in queue, got %d", len(vessels))
	}
	if vessels[0].ID != "issue-2" {
		t.Errorf("expected vessel ID issue-2, got %s", vessels[0].ID)
	}
}

func TestScanPausedOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := makeScanConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newScanMock()

	os.WriteFile(filepath.Join(dir, "paused"), []byte{}, 0o644) //nolint:errcheck

	// Stub the gh search command (paused scan short-circuits before this,
	// but the mock would error on unstubbed commands)
	r.set([]byte("[]"), "gh", "search", "issues", "--repo", "owner/repo",
		"--state", "open", "--json", "number,title,url,labels",
		"--limit", "20", "--label", "bug")

	out := captureStdout(func() {
		err := cmdScan(cfg, q, r, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "paused") {
		t.Errorf("expected paused message in output, got: %s", out)
	}
}

func TestScanDryRunEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := makeScanConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newScanMock()

	// Stub the gh search to return empty array
	r.set([]byte("[]"), "gh", "search", "issues", "--repo", "owner/repo",
		"--state", "open", "--json", "number,title,url,labels",
		"--limit", "20", "--label", "bug")

	old := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw

	err := dryRunScan(cfg, q, r)

	pw.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, pr) //nolint:errcheck
	out := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "No new issues") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestScanError(t *testing.T) {
	dir := t.TempDir()
	cfg := makeScanConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newScanMock()

	// Stub gh search to return an error
	r.setErr(errors.New("gh: not authenticated"),
		"gh", "search", "issues", "--repo", "owner/repo",
		"--state", "open", "--json", "number,title,url,labels",
		"--limit", "20", "--label", "bug")

	err := cmdScan(cfg, q, r, false)
	if err == nil {
		t.Fatal("expected error from cmdScan, got nil")
	}
	if !strings.Contains(err.Error(), "scan error") {
		t.Errorf("expected wrapped 'scan error', got: %v", err)
	}
}
