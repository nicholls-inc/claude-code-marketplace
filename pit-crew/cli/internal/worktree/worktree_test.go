package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockRunner captures calls for verification.
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

func (m *mockRunner) setOutput(key string, out []byte) { m.outputs[key] = out }
func (m *mockRunner) setErr(key string, err error)     { m.errs[key] = err }

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
	return []byte{}, nil
}

func (m *mockRunner) called(name string, args ...string) bool {
	target := append([]string{name}, args...)
	for _, call := range m.calls {
		if len(call) != len(target) {
			continue
		}
		match := true
		for i := range call {
			if call[i] != target[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func TestDefaultBranchFromGH(t *testing.T) {
	r := newMock()
	r.setOutput("gh repo view --json defaultBranchRef", []byte(`{"defaultBranchRef":{"name":"main"}}`))
	m := New("/repo", r)
	branch, err := m.DefaultBranch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected 'main', got %q", branch)
	}
}

func TestDefaultBranchFallback(t *testing.T) {
	r := newMock()
	r.setErr("gh repo view --json defaultBranchRef", errors.New("gh not available"))
	r.setOutput("git remote show origin", []byte(`  HEAD branch: develop`))
	m := New("/repo", r)
	branch, err := m.DefaultBranch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "develop" {
		t.Errorf("expected 'develop', got %q", branch)
	}
}

func TestCreateIssuesCorrectCommands(t *testing.T) {
	r := newMock()
	r.setOutput("gh repo view --json defaultBranchRef", []byte(`{"defaultBranchRef":{"name":"main"}}`))

	m := New("/repo", r)
	_, err := m.Create(context.Background(), "fix/issue-42-null-response")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("git", "fetch", "origin", "main") {
		t.Error("expected 'git fetch origin main' to be called")
	}
	if !r.called("git", "worktree", "add", ".claude/worktrees/fix/issue-42-null-response", "-b", "fix/issue-42-null-response", "origin/main") {
		t.Errorf("expected 'git worktree add' to be called, calls were: %v", r.calls)
	}
}

func TestCreateFetchFailure(t *testing.T) {
	r := newMock()
	r.setOutput("gh repo view --json defaultBranchRef", []byte(`{"defaultBranchRef":{"name":"main"}}`))
	r.setErr("git fetch origin main", errors.New("network unreachable"))

	m := New("/repo", r)
	_, err := m.Create(context.Background(), "fix/issue-42-test")
	if err == nil {
		t.Fatal("expected error from fetch failure, got nil")
	}
	for _, call := range r.calls {
		if len(call) > 2 && call[0] == "git" && call[1] == "worktree" && call[2] == "add" {
			t.Error("git worktree add should NOT be called when fetch fails")
		}
	}
}

func TestRemoveIssuesCorrectCommand(t *testing.T) {
	r := newMock()
	m := New("/repo", r)
	err := m.Remove(context.Background(), ".claude/worktrees/fix/issue-42-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.called("git", "worktree", "remove", ".claude/worktrees/fix/issue-42-test", "--force") {
		t.Error("expected 'git worktree remove ... --force' to be called")
	}
}

func TestListParsesPorcelain(t *testing.T) {
	porcelain := "worktree /home/user/repo\nHEAD abc123\nbranch refs/heads/main\n\nworktree /home/user/repo/.claude/worktrees/fix/issue-42\nHEAD def456\nbranch refs/heads/fix/issue-42\n\n"
	r := newMock()
	r.setOutput("git worktree list --porcelain", []byte(porcelain))
	m := New("/home/user/repo", r)

	list, err := m.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(list))
	}
	if list[0].Branch != "main" {
		t.Errorf("expected branch 'main', got %q", list[0].Branch)
	}
	if list[1].Branch != "fix/issue-42" {
		t.Errorf("expected branch 'fix/issue-42', got %q", list[1].Branch)
	}
	if list[1].HeadCommit != "def456" {
		t.Errorf("expected commit 'def456', got %q", list[1].HeadCommit)
	}
}

func TestListPitCrewFilters(t *testing.T) {
	porcelain := "worktree /home/user/repo\nHEAD abc123\nbranch refs/heads/main\n\nworktree /home/user/repo/.claude/worktrees/fix/issue-42\nHEAD def456\nbranch refs/heads/fix/issue-42\n\n"
	r := newMock()
	r.setOutput("git worktree list --porcelain", []byte(porcelain))
	m := New("/home/user/repo", r)

	list, err := m.ListPitCrew(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 pit-crew worktree, got %d: %v", len(list), list)
	}
	if !strings.Contains(list[0].Path, ".claude/worktrees/") {
		t.Errorf("expected path under .claude/worktrees/, got %q", list[0].Path)
	}
}

func TestCopyClaudeConfig(t *testing.T) {
	src := t.TempDir()
	claudeDir := filepath.Join(src, ".claude")
	os.MkdirAll(filepath.Join(claudeDir, "worktrees"), 0o755)
	os.MkdirAll(filepath.Join(claudeDir, "rules"), 0o755)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{}`), 0o644)
	os.WriteFile(filepath.Join(claudeDir, "rules", "test.md"), []byte("# rule"), 0o644)

	dst := t.TempDir()
	m := &Manager{RepoRoot: src}
	if err := m.copyClaudeConfig(dst); err != nil {
		t.Fatalf("copyClaudeConfig failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, ".claude", "settings.json")); err != nil {
		t.Error("settings.json should have been copied")
	}
	if _, err := os.Stat(filepath.Join(dst, ".claude", "rules", "test.md")); err != nil {
		t.Error("rules/test.md should have been copied")
	}
	if _, err := os.Stat(filepath.Join(dst, ".claude", "worktrees")); !os.IsNotExist(err) {
		t.Error("worktrees/ should NOT have been copied")
	}
	fmt.Println("TestCopyClaudeConfig: PASS")
}
