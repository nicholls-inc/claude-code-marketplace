package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

type mockRunner struct {
	calls   [][]string
	outputs map[string][]byte
	errs    map[string]error
}

func newMock() *mockRunner {
	return &mockRunner{
		outputs: make(map[string][]byte),
		errs:    make(map[string]error),
	}
}

func (m *mockRunner) set(out []byte, args ...string) {
	m.outputs[strings.Join(args, " ")] = out
}

func (m *mockRunner) setErr(err error, args ...string) {
	m.errs[strings.Join(args, " ")] = err
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	parts := append([]string{name}, args...)
	m.calls = append(m.calls, parts)
	key := strings.Join(parts, " ")
	if err, ok := m.errs[key]; ok {
		return nil, err
	}
	if out, ok := m.outputs[key]; ok {
		return out, nil
	}
	return []byte("[]"), nil
}

func makeConfig(dir string) *config.Config {
	return &config.Config{
		Repo:        "owner/repo",
		Concurrency: 2,
		MaxTurns:    50,
		Timeout:     "30m",
		StateDir:    dir,
		Exclude:     []string{"wontfix", "duplicate"},
		Claude:      config.ClaudeConfig{Command: "claude", Template: "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}"},
		Tasks: map[string]config.Task{
			"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"},
		},
	}
}

func issueJSON(issues []ghIssue) []byte {
	b, _ := json.Marshal(issues)
	return b
}

func TestScanFindsIssues(t *testing.T) {
	dir := t.TempDir()
	queueFile := filepath.Join(dir, "queue.jsonl")
	cfg := makeConfig(dir)
	q := queue.New(queueFile)
	r := newMock()

	issues := []ghIssue{
		{Number: 1, Title: "fix null response", URL: "https://github.com/owner/repo/issues/1", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
		{Number: 2, Title: "fix panic on empty", URL: "https://github.com/owner/repo/issues/2", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
	}
	r.set(issueJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 2 {
		t.Errorf("expected 2 added, got %d", result.Added)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}
	if result.Paused {
		t.Error("expected not paused")
	}
	jobs, _ := q.List()
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs in queue, got %d", len(jobs))
	}
}

func TestScanExcludedLabel(t *testing.T) {
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	issues := []ghIssue{
		{Number: 1, Title: "won't fix", URL: "https://github.com/owner/repo/issues/1", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}, {Name: "wontfix"}}},
	}
	r.set(issueJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 0 {
		t.Errorf("expected 0 added (excluded), got %d", result.Added)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
}

func TestScanAlreadyQueued(t *testing.T) {
	dir := t.TempDir()
	cfg := makeConfig(dir)
	queueFile := filepath.Join(dir, "queue.jsonl")
	q := queue.New(queueFile)
	r := newMock()

	now := time.Now().UTC()
	_ = q.Enqueue(queue.Job{ID: "issue-1", IssueNum: 1, IssueURL: "https://github.com/owner/repo/issues/1", Skill: "fix-bug", State: queue.StatePending, CreatedAt: now})

	issues := []ghIssue{
		{Number: 1, Title: "already queued", URL: "https://github.com/owner/repo/issues/1", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
	}
	r.set(issueJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 0 {
		t.Errorf("expected 0 added (already queued), got %d", result.Added)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
}

func TestScanExistingBranch(t *testing.T) {
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	issues := []ghIssue{
		{Number: 42, Title: "has branch", URL: "https://github.com/owner/repo/issues/42", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
	}
	r.set(issueJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")
	r.set([]byte("abc123\trefs/heads/fix/issue-42-something"), "git", "ls-remote", "--heads", "origin", "fix/issue-42-*")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 0 {
		t.Errorf("expected 0 added (existing branch), got %d", result.Added)
	}
}

func TestScanExistingPR(t *testing.T) {
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	issues := []ghIssue{
		{Number: 55, Title: "has pr", URL: "https://github.com/owner/repo/issues/55", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
	}
	r.set(issueJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")
	r.set([]byte(`[{"number":99}]`), "gh", "pr", "list", "--repo", "owner/repo", "--search", "#55", "--state", "open", "--json", "number", "--limit", "1")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 0 {
		t.Errorf("expected 0 added (open PR exists), got %d", result.Added)
	}
}

func TestScanPaused(t *testing.T) {
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	os.WriteFile(filepath.Join(dir, "paused"), []byte{}, 0o644)

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Paused {
		t.Error("expected Paused=true")
	}
	if result.Added != 0 {
		t.Errorf("expected 0 added when paused, got %d", result.Added)
	}
	if len(r.calls) != 0 {
		t.Errorf("expected no gh calls when paused, got %d", len(r.calls))
	}
}

func TestScanGHFailure(t *testing.T) {
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	r.setErr(errors.New("network error"), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")

	s := New(cfg, q, r)
	_, err := s.Scan(context.Background())
	if err == nil {
		t.Fatal("expected error from gh failure, got nil")
	}
}

func TestScanMultipleTasks(t *testing.T) {
	dir := t.TempDir()
	cfg := makeConfig(dir)
	cfg.Tasks = map[string]config.Task{
		"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"},
		"features": {Labels: []string{"low-effort"}, Skill: "implement-feature"},
	}
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	bugIssues := []ghIssue{
		{Number: 1, Title: "null bug", URL: "https://github.com/owner/repo/issues/1", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
	}
	featureIssues := []ghIssue{
		{Number: 2, Title: "add feature", URL: "https://github.com/owner/repo/issues/2", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "low-effort"}}},
	}

	r.set(issueJSON(bugIssues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")
	r.set(issueJSON(featureIssues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "low-effort")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 2 {
		t.Errorf("expected 2 total added, got %d", result.Added)
	}
	jobs, _ := q.List()
	skills := make(map[string]bool)
	for _, j := range jobs {
		skills[j.Skill] = true
	}
	if !skills["fix-bug"] || !skills["implement-feature"] {
		t.Errorf("expected both skills queued, got: %v", skills)
	}
	fmt.Println("TestScanMultipleTasks: PASS")
}
