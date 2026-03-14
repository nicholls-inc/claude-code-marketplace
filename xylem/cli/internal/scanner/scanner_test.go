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

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
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
	vessels, _ := q.List()
	if len(vessels) != 2 {
		t.Errorf("expected 2 vessels in queue, got %d", len(vessels))
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
	_ = q.Enqueue(queue.Vessel{ID: "issue-1", IssueNum: 1, IssueURL: "https://github.com/owner/repo/issues/1", Skill: "fix-bug", State: queue.StatePending, CreatedAt: now})

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
	// The PR search now uses head branch pattern and fetches headRefName
	r.set([]byte(`[{"number":99,"headRefName":"fix/issue-55-null-fix"}]`),
		"gh", "pr", "list", "--repo", "owner/repo", "--search", "head:fix/issue-55-", "--state", "open", "--json", "number,headRefName", "--limit", "5")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 0 {
		t.Errorf("expected 0 added (open PR exists), got %d", result.Added)
	}
}

func TestScanPRFalsePositiveIgnored(t *testing.T) {
	// Regression test: a PR whose title/body mentions "#1" but whose head branch
	// does NOT match the xylem naming convention should NOT cause issue #1 to
	// be skipped.
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	issues := []ghIssue{
		{Number: 1, Title: "real bug", URL: "https://github.com/owner/repo/issues/1", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
	}
	r.set(issueJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")

	// Simulate an unrelated PR returned by the search whose head branch does NOT
	// match the expected pattern. The old code would have incorrectly skipped
	// issue #1 because ANY returned PR counted as a match.
	r.set([]byte(`[{"number":200,"headRefName":"chore/priority-1-ci-fix"}]`),
		"gh", "pr", "list", "--repo", "owner/repo", "--search", "head:fix/issue-1-", "--state", "open", "--json", "number,headRefName", "--limit", "5")
	// feat/ prefix search returns nothing
	r.set([]byte(`[]`),
		"gh", "pr", "list", "--repo", "owner/repo", "--search", "head:feat/issue-1-", "--state", "open", "--json", "number,headRefName", "--limit", "5")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 1 {
		t.Errorf("expected 1 added (false positive PR should be ignored), got %d", result.Added)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}
}

func TestScanCrossTaskDedupDeterministic(t *testing.T) {
	// When the same issue appears in multiple tasks, it should only be enqueued
	// once regardless of map iteration order.
	dir := t.TempDir()
	cfg := makeConfig(dir)
	cfg.Tasks = map[string]config.Task{
		"fix-bugs":  {Labels: []string{"bug"}, Skill: "fix-bug"},
		"emergency": {Labels: []string{"urgent"}, Skill: "fix-bug"},
	}
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	// Same issue #10 appears under both label searches
	sharedIssue := []ghIssue{
		{Number: 10, Title: "shared issue", URL: "https://github.com/owner/repo/issues/10", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}, {Name: "urgent"}}},
	}
	r.set(issueJSON(sharedIssue), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")
	r.set(issueJSON(sharedIssue), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "urgent")

	_ = New(cfg, q, r) // verify construction; actual test uses per-iteration queues

	// Run multiple times to exercise different map iteration orders
	for i := 0; i < 5; i++ {
		qFile := filepath.Join(dir, fmt.Sprintf("queue-%d.jsonl", i))
		qi := queue.New(qFile)
		si := New(cfg, qi, r)
		result, err := si.Scan(context.Background())
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		vessels, _ := qi.List()
		if len(vessels) != 1 {
			t.Errorf("run %d: expected exactly 1 vessel in queue, got %d", i, len(vessels))
		}
		if result.Added+result.Skipped != 2 {
			t.Errorf("run %d: expected 2 total (added+skipped), got %d", i, result.Added+result.Skipped)
		}
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
	vessels, _ := q.List()
	skills := make(map[string]bool)
	for _, j := range vessels {
		skills[j.Skill] = true
	}
	if !skills["fix-bug"] || !skills["implement-feature"] {
		t.Errorf("expected both skills queued, got: %v", skills)
	}
}

func TestScanGHReturnsMalformedJSON(t *testing.T) {
	// When `gh search issues` returns invalid JSON, Scan should return an error.
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	r.set([]byte(`{not valid json`), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")

	s := New(cfg, q, r)
	_, err := s.Scan(context.Background())
	if err == nil {
		t.Fatal("expected error for malformed gh JSON output")
	}
	if !strings.Contains(err.Error(), "parse gh search output") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

func TestHasOpenPRMalformedJSONIgnored(t *testing.T) {
	// When gh pr list returns malformed JSON, hasOpenPR should return false
	// (the issue should NOT be skipped).
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	issues := []ghIssue{
		{Number: 77, Title: "test issue", URL: "https://github.com/owner/repo/issues/77", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
	}
	r.set(issueJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")
	// Return malformed JSON for PR check
	r.set([]byte(`not json at all`),
		"gh", "pr", "list", "--repo", "owner/repo", "--search", "head:fix/issue-77-", "--state", "open", "--json", "number,headRefName", "--limit", "5")
	r.set([]byte(`not json`),
		"gh", "pr", "list", "--repo", "owner/repo", "--search", "head:feat/issue-77-", "--state", "open", "--json", "number,headRefName", "--limit", "5")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 1 {
		t.Errorf("expected 1 added (malformed PR JSON should not block), got %d", result.Added)
	}
}

func TestHasOpenPRGHErrorIgnored(t *testing.T) {
	// When gh pr list returns an error, hasOpenPR should return false.
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	issues := []ghIssue{
		{Number: 88, Title: "test issue", URL: "https://github.com/owner/repo/issues/88", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
	}
	r.set(issueJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")
	r.setErr(errors.New("gh auth error"),
		"gh", "pr", "list", "--repo", "owner/repo", "--search", "head:fix/issue-88-", "--state", "open", "--json", "number,headRefName", "--limit", "5")
	r.setErr(errors.New("gh auth error"),
		"gh", "pr", "list", "--repo", "owner/repo", "--search", "head:feat/issue-88-", "--state", "open", "--json", "number,headRefName", "--limit", "5")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 1 {
		t.Errorf("expected 1 added (gh errors for PR check should not block), got %d", result.Added)
	}
}

func TestScanExistingBranchFeatPrefix(t *testing.T) {
	// Test that hasBranch also checks "feat/" prefix, not just "fix/".
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()

	issues := []ghIssue{
		{Number: 99, Title: "has feat branch", URL: "https://github.com/owner/repo/issues/99", Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}}},
	}
	r.set(issueJSON(issues), "gh", "search", "issues", "--repo", "owner/repo", "--state", "open", "--json", "number,title,url,labels", "--limit", "20", "--label", "bug")
	// fix/ prefix returns nothing, but feat/ prefix has a match
	r.set([]byte(""), "git", "ls-remote", "--heads", "origin", "fix/issue-99-*")
	r.set([]byte("abc123\trefs/heads/feat/issue-99-add-feature"), "git", "ls-remote", "--heads", "origin", "feat/issue-99-*")

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 0 {
		t.Errorf("expected 0 added (feat branch exists), got %d", result.Added)
	}
}

func TestScanEmptyIssuesList(t *testing.T) {
	// When gh returns an empty array, Scan should succeed with 0 added.
	dir := t.TempDir()
	cfg := makeConfig(dir)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	r := newMock()
	// Default mock returns "[]" for unknown keys, so no special setup needed.

	s := New(cfg, q, r)
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Added != 0 {
		t.Errorf("expected 0 added, got %d", result.Added)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}
}
