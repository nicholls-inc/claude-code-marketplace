package worktree

import (
	"context"
	"errors"
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
	porcelain := "worktree /repo/.claude/worktrees/fix/issue-42-test\nHEAD abc123\nbranch refs/heads/fix/issue-42-test\n\n"
	r.setOutput("git worktree list --porcelain", []byte(porcelain))
	m := New("/repo", r)
	err := m.Remove(context.Background(), ".claude/worktrees/fix/issue-42-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.called("git", "worktree", "remove", ".claude/worktrees/fix/issue-42-test", "--force") {
		t.Error("expected 'git worktree remove ... --force' to be called")
	}
}

// TestRemoveDeletesCorrectBranchWithSlash verifies that Remove() deletes the
// full branch name (e.g., "fix/issue-42") instead of just the last path
// component ("issue-42"), which was the bug caused by filepath.Base().
func TestRemoveDeletesCorrectBranchWithSlash(t *testing.T) {
	r := newMock()
	porcelain := "worktree /repo/.claude/worktrees/fix/issue-42-slug\nHEAD abc123\nbranch refs/heads/fix/issue-42-slug\n\n"
	r.setOutput("git worktree list --porcelain", []byte(porcelain))
	m := New("/repo", r)

	err := m.Remove(context.Background(), ".claude/worktrees/fix/issue-42-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The fix: must delete the FULL branch name "fix/issue-42-slug",
	// NOT the truncated "issue-42-slug" that filepath.Base() would return.
	if !r.called("git", "branch", "-d", "fix/issue-42-slug") {
		t.Errorf("expected 'git branch -d fix/issue-42-slug', got calls: %v", r.calls)
	}
	if r.called("git", "branch", "-d", "issue-42-slug") {
		t.Error("should NOT call 'git branch -d issue-42-slug' (truncated by filepath.Base)")
	}
}

// TestRemoveWithAbsoluteWorktreePath verifies Remove works when given an
// absolute worktree path (as returned by ListXylem with absolute paths).
func TestRemoveWithAbsoluteWorktreePath(t *testing.T) {
	r := newMock()
	porcelain := "worktree /repo/.claude/worktrees/feat/add-logging\nHEAD def456\nbranch refs/heads/feat/add-logging\n\n"
	r.setOutput("git worktree list --porcelain", []byte(porcelain))
	m := New("/repo", r)

	err := m.Remove(context.Background(), "/repo/.claude/worktrees/feat/add-logging")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.called("git", "branch", "-d", "feat/add-logging") {
		t.Errorf("expected 'git branch -d feat/add-logging', got calls: %v", r.calls)
	}
}

// TestRemoveSkipsBranchDeleteWhenNotFound verifies Remove() skips branch
// deletion gracefully when the worktree is not found in porcelain output.
func TestRemoveSkipsBranchDeleteWhenNotFound(t *testing.T) {
	r := newMock()
	// Empty porcelain output — worktree not found
	r.setOutput("git worktree list --porcelain", []byte(""))
	m := New("/repo", r)

	err := m.Remove(context.Background(), ".claude/worktrees/orphan-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should NOT attempt branch deletion when branch name is unknown
	for _, call := range r.calls {
		if len(call) >= 2 && call[0] == "git" && call[1] == "branch" {
			t.Errorf("should not call git branch when branch name is unknown, got: %v", call)
		}
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

func TestListXylemFilters(t *testing.T) {
	porcelain := "worktree /home/user/repo\nHEAD abc123\nbranch refs/heads/main\n\nworktree /home/user/repo/.claude/worktrees/fix/issue-42\nHEAD def456\nbranch refs/heads/fix/issue-42\n\n"
	r := newMock()
	r.setOutput("git worktree list --porcelain", []byte(porcelain))
	m := New("/home/user/repo", r)

	list, err := m.ListXylem(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 xylem worktree, got %d: %v", len(list), list)
	}
	if !strings.Contains(list[0].Path, ".claude/worktrees/") {
		t.Errorf("expected path under .claude/worktrees/, got %q", list[0].Path)
	}
}

func TestParsePorcelainEmpty(t *testing.T) {
	result := parsePorcelain("")
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %d entries", len(result))
	}
}

func TestParsePorcelainSpacesInPath(t *testing.T) {
	porcelain := "worktree /home/user/my project/repo\nHEAD abc123\nbranch refs/heads/main\n\n"
	result := parsePorcelain(porcelain)
	if len(result) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(result))
	}
	if result[0].Path != "/home/user/my project/repo" {
		t.Errorf("expected path with spaces, got %q", result[0].Path)
	}
}

func TestParsePorcelainDetachedHEAD(t *testing.T) {
	// Detached HEAD worktrees have no "branch" line; instead they have "detached"
	porcelain := "worktree /home/user/repo\nHEAD abc123\ndetached\n\n"
	result := parsePorcelain(porcelain)
	if len(result) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(result))
	}
	if result[0].Branch != "" {
		t.Errorf("expected empty branch for detached HEAD, got %q", result[0].Branch)
	}
	if result[0].HeadCommit != "abc123" {
		t.Errorf("expected commit 'abc123', got %q", result[0].HeadCommit)
	}
}

func TestParsePorcelainWindowsLineEndings(t *testing.T) {
	porcelain := "worktree /home/user/repo\r\nHEAD abc123\r\nbranch refs/heads/main\r\n\r\n"
	result := parsePorcelain(porcelain)
	if len(result) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(result))
	}
	if result[0].Path != "/home/user/repo" {
		t.Errorf("expected clean path, got %q", result[0].Path)
	}
	if result[0].Branch != "main" {
		t.Errorf("expected branch 'main', got %q", result[0].Branch)
	}
}

func TestParsePorcelainNoTrailingNewline(t *testing.T) {
	// No trailing blank line — last entry must still be captured
	porcelain := "worktree /repo\nHEAD abc\nbranch refs/heads/dev"
	result := parsePorcelain(porcelain)
	if len(result) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(result))
	}
	if result[0].Branch != "dev" {
		t.Errorf("expected branch 'dev', got %q", result[0].Branch)
	}
}

func TestParsePorcelainMultipleEntries(t *testing.T) {
	porcelain := strings.Join([]string{
		"worktree /repo",
		"HEAD aaa111",
		"branch refs/heads/main",
		"",
		"worktree /repo/.claude/worktrees/fix/bug-1",
		"HEAD bbb222",
		"branch refs/heads/fix/bug-1",
		"",
		"worktree /repo/.claude/worktrees/feat/new-feature",
		"HEAD ccc333",
		"branch refs/heads/feat/new-feature",
		"",
	}, "\n")
	result := parsePorcelain(porcelain)
	if len(result) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(result))
	}
	expected := []struct {
		path   string
		branch string
		commit string
	}{
		{"/repo", "main", "aaa111"},
		{"/repo/.claude/worktrees/fix/bug-1", "fix/bug-1", "bbb222"},
		{"/repo/.claude/worktrees/feat/new-feature", "feat/new-feature", "ccc333"},
	}
	for i, exp := range expected {
		if result[i].Path != exp.path {
			t.Errorf("[%d] expected path %q, got %q", i, exp.path, result[i].Path)
		}
		if result[i].Branch != exp.branch {
			t.Errorf("[%d] expected branch %q, got %q", i, exp.branch, result[i].Branch)
		}
		if result[i].HeadCommit != exp.commit {
			t.Errorf("[%d] expected commit %q, got %q", i, exp.commit, result[i].HeadCommit)
		}
	}
}

func TestCopyFilePreservesPermissions(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create an executable file
	srcPath := filepath.Join(srcDir, "script.sh")
	if err := os.WriteFile(srcPath, []byte("#!/bin/sh\necho hi"), 0o755); err != nil {
		t.Fatal(err)
	}

	dstPath := filepath.Join(dstDir, "script.sh")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("permissions not preserved: source=%v, dest=%v", srcInfo.Mode(), dstInfo.Mode())
	}
}

func TestCopyFilePreservesReadOnlyPermissions(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcPath := filepath.Join(srcDir, "readonly.txt")
	if err := os.WriteFile(srcPath, []byte("data"), 0o444); err != nil {
		t.Fatal(err)
	}

	dstPath := filepath.Join(dstDir, "readonly.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("permissions not preserved: source=%v, dest=%v", srcInfo.Mode(), dstInfo.Mode())
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
}

// --- Additional coverage tests ---

func TestDefaultBranchBothFail(t *testing.T) {
	r := newMock()
	r.setErr("gh repo view --json defaultBranchRef", errors.New("gh not available"))
	r.setErr("git remote show origin", errors.New("network error"))

	m := New("/repo", r)
	_, err := m.DefaultBranch(context.Background())
	if err == nil {
		t.Fatal("expected error when both gh and git fail")
	}
	if !strings.Contains(err.Error(), "detect default branch") {
		t.Errorf("expected detect-default-branch error, got: %v", err)
	}
}

func TestDefaultBranchGitOutputNoHeadLine(t *testing.T) {
	// git remote show origin returns output but no "HEAD branch:" line
	r := newMock()
	r.setErr("gh repo view --json defaultBranchRef", errors.New("no gh"))
	r.setOutput("git remote show origin", []byte("  Remote branch:\n    main tracked\n"))

	m := New("/repo", r)
	_, err := m.DefaultBranch(context.Background())
	if err == nil {
		t.Fatal("expected error when HEAD branch not found in output")
	}
	if !strings.Contains(err.Error(), "could not detect default branch") {
		t.Errorf("expected could-not-detect error, got: %v", err)
	}
}

func TestDefaultBranchGHReturnsMalformedJSON(t *testing.T) {
	// gh succeeds but returns garbage; should fall back to git
	r := newMock()
	r.setOutput("gh repo view --json defaultBranchRef", []byte(`not json`))
	r.setOutput("git remote show origin", []byte("  HEAD branch: develop\n"))

	m := New("/repo", r)
	branch, err := m.DefaultBranch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "develop" {
		t.Errorf("expected 'develop', got %q", branch)
	}
}

func TestDefaultBranchGHReturnsEmptyName(t *testing.T) {
	// gh returns valid JSON but with empty name; should fall back to git
	r := newMock()
	r.setOutput("gh repo view --json defaultBranchRef", []byte(`{"defaultBranchRef":{"name":""}}`))
	r.setOutput("git remote show origin", []byte("  HEAD branch: master\n"))

	m := New("/repo", r)
	branch, err := m.DefaultBranch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "master" {
		t.Errorf("expected 'master', got %q", branch)
	}
}

func TestCreateWorktreeAddFails(t *testing.T) {
	r := newMock()
	r.setOutput("gh repo view --json defaultBranchRef", []byte(`{"defaultBranchRef":{"name":"main"}}`))
	r.setErr("git worktree add .claude/worktrees/fix/issue-1-test -b fix/issue-1-test origin/main",
		errors.New("fatal: 'fix/issue-1-test' is already checked out"))

	m := New("/repo", r)
	_, err := m.Create(context.Background(), "fix/issue-1-test")
	if err == nil {
		t.Fatal("expected error when worktree add fails")
	}
	if !strings.Contains(err.Error(), "git worktree add") {
		t.Errorf("expected worktree-add error, got: %v", err)
	}
}

func TestCreateDefaultBranchFails(t *testing.T) {
	r := newMock()
	r.setErr("gh repo view --json defaultBranchRef", errors.New("no gh"))
	r.setErr("git remote show origin", errors.New("no remote"))

	m := New("/repo", r)
	_, err := m.Create(context.Background(), "fix/issue-1-test")
	if err == nil {
		t.Fatal("expected error when default branch detection fails")
	}
	if !strings.Contains(err.Error(), "create worktree") {
		t.Errorf("expected create-worktree error, got: %v", err)
	}
}

func TestListError(t *testing.T) {
	r := newMock()
	r.setErr("git worktree list --porcelain", errors.New("not a git repo"))

	m := New("/repo", r)
	_, err := m.List(context.Background())
	if err == nil {
		t.Fatal("expected error when git worktree list fails")
	}
}

func TestListXylemError(t *testing.T) {
	r := newMock()
	r.setErr("git worktree list --porcelain", errors.New("not a git repo"))

	m := New("/repo", r)
	_, err := m.ListXylem(context.Background())
	if err == nil {
		t.Fatal("expected error propagation from ListXylem")
	}
}

func TestCopyFileNonExistentSource(t *testing.T) {
	dstPath := filepath.Join(t.TempDir(), "output.txt")
	err := copyFile("/nonexistent/path/file.txt", dstPath)
	if err == nil {
		t.Fatal("expected error for non-existent source")
	}
}

func TestCopyFileContentPreserved(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	content := "hello world\nsecond line\n"
	srcPath := filepath.Join(srcDir, "data.txt")
	if err := os.WriteFile(srcPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	dstPath := filepath.Join(dstDir, "subdir", "data.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != content {
		t.Errorf("content mismatch: got %q, want %q", string(got), content)
	}
}

func TestCopyDirNested(t *testing.T) {
	srcDir := t.TempDir()
	// Create nested structure: a/b/c.txt, a/d.txt
	os.MkdirAll(filepath.Join(srcDir, "a", "b"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "a", "b", "c.txt"), []byte("deep"), 0o644)
	os.WriteFile(filepath.Join(srcDir, "a", "d.txt"), []byte("shallow"), 0o644)

	dstDir := filepath.Join(t.TempDir(), "copy")
	if err := copyDir(filepath.Join(srcDir, "a"), dstDir); err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	// Verify nested file
	got, err := os.ReadFile(filepath.Join(dstDir, "b", "c.txt"))
	if err != nil {
		t.Fatalf("read nested: %v", err)
	}
	if string(got) != "deep" {
		t.Errorf("nested content = %q, want 'deep'", string(got))
	}

	// Verify shallow file
	got, err = os.ReadFile(filepath.Join(dstDir, "d.txt"))
	if err != nil {
		t.Fatalf("read shallow: %v", err)
	}
	if string(got) != "shallow" {
		t.Errorf("shallow content = %q, want 'shallow'", string(got))
	}
}

func TestCopyClaudeConfigNoClaudeDir(t *testing.T) {
	// When .claude/ doesn't exist in source, copyClaudeConfig should succeed silently.
	src := t.TempDir()
	dst := t.TempDir()
	m := &Manager{RepoRoot: src}
	if err := m.copyClaudeConfig(dst); err != nil {
		t.Fatalf("expected nil error when .claude/ absent, got: %v", err)
	}
}

func TestCopyClaudeConfigSettingsLocal(t *testing.T) {
	src := t.TempDir()
	claudeDir := filepath.Join(src, ".claude")
	os.MkdirAll(claudeDir, 0o755)
	os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(`{"local":true}`), 0o644)
	// Also add a file NOT in the allowlist
	os.WriteFile(filepath.Join(claudeDir, "conversations"), []byte("skip"), 0o644)

	dst := t.TempDir()
	m := &Manager{RepoRoot: src}
	if err := m.copyClaudeConfig(dst); err != nil {
		t.Fatalf("copyClaudeConfig failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, ".claude", "settings.local.json")); err != nil {
		t.Error("settings.local.json should have been copied")
	}
	if _, err := os.Stat(filepath.Join(dst, ".claude", "conversations")); !os.IsNotExist(err) {
		t.Error("conversations should NOT have been copied (not in allowlist)")
	}
}

func TestParsePorcelainBareWorktree(t *testing.T) {
	// A bare worktree has "bare" instead of "branch"
	porcelain := "worktree /home/user/repo.git\nHEAD abc123\nbare\n\n"
	result := parsePorcelain(porcelain)
	if len(result) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(result))
	}
	if result[0].Branch != "" {
		t.Errorf("expected empty branch for bare worktree, got %q", result[0].Branch)
	}
	if result[0].HeadCommit != "abc123" {
		t.Errorf("expected commit 'abc123', got %q", result[0].HeadCommit)
	}
}

func TestRemoveWorktreeRemoveFails(t *testing.T) {
	r := newMock()
	porcelain := "worktree /repo/.claude/worktrees/fix/issue-1\nHEAD abc\nbranch refs/heads/fix/issue-1\n\n"
	r.setOutput("git worktree list --porcelain", []byte(porcelain))
	r.setErr("git worktree remove .claude/worktrees/fix/issue-1 --force", errors.New("lock file exists"))

	m := New("/repo", r)
	err := m.Remove(context.Background(), ".claude/worktrees/fix/issue-1")
	if err == nil {
		t.Fatal("expected error when worktree remove fails")
	}
	if !strings.Contains(err.Error(), "git worktree remove") {
		t.Errorf("expected worktree-remove error, got: %v", err)
	}
	// Should NOT attempt branch delete if remove failed
	if r.called("git", "branch", "-d", "fix/issue-1") {
		t.Error("should not delete branch when worktree remove fails")
	}
}
