package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

type mockCmdRunner struct {
	processErr error
	outputErr  error
	started    int32
}

func (m *mockCmdRunner) RunOutput(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return []byte{}, m.outputErr
}

func (m *mockCmdRunner) RunProcess(_ context.Context, _ string, _ string, _ ...string) error {
	atomic.AddInt32(&m.started, 1)
	return m.processErr
}

type countingCmdRunner struct {
	concurrent int32
	maxSeen    int32
	delay      time.Duration
}

func (c *countingCmdRunner) RunOutput(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return []byte{}, nil
}

func (c *countingCmdRunner) RunProcess(_ context.Context, _ string, _ string, _ ...string) error {
	cur := atomic.AddInt32(&c.concurrent, 1)
	for {
		old := atomic.LoadInt32(&c.maxSeen)
		if cur <= old {
			break
		}
		if atomic.CompareAndSwapInt32(&c.maxSeen, old, cur) {
			break
		}
	}
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	atomic.AddInt32(&c.concurrent, -1)
	return nil
}

type mockWorktree struct {
	createErr error
	path      string
}

func (m *mockWorktree) Create(_ context.Context, branchName string) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	if m.path != "" {
		return m.path, nil
	}
	return ".claude/worktrees/" + branchName, nil
}

func makeTestConfig(dir string, concurrency int) *config.Config {
	return &config.Config{
		Repo:        "owner/repo",
		Concurrency: concurrency,
		MaxTurns:    50,
		Timeout:     "30s",
		StateDir:    dir,
		Exclude:     []string{"wontfix"},
		Claude:      config.ClaudeConfig{Command: "claude", Template: "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}"},
		Tasks:       map[string]config.Task{"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"}},
	}
}

func makeJob(num int, skill string) queue.Job {
	return queue.Job{
		ID:        fmt.Sprintf("issue-%d", num),
		IssueURL:  fmt.Sprintf("https://github.com/owner/repo/issues/%d", num),
		IssueNum:  num,
		Skill:     skill,
		State:     queue.StatePending,
		CreatedAt: time.Now().UTC(),
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"https://github.com/owner/repo/issues/42", "42"},
		{"simple text", "simple-text"},
		{"ALL CAPS", "all-caps"},
		{"special!@#chars", "special-chars"},
	}
	for _, tc := range cases {
		got := slugify(tc.input)
		if got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestBuildCommand(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:  "claude",
			Template: "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}",
		},
	}
	job := &queue.Job{
		Skill:    "fix-bug",
		IssueURL: "https://github.com/owner/repo/issues/42",
	}
	cmd, args, err := buildCommand(cfg, job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "claude" {
		t.Errorf("expected cmd 'claude', got %q", cmd)
	}
	full := cmd + " " + strings.Join(args, " ")
	if !strings.Contains(full, "fix-bug") {
		t.Errorf("expected skill in command, got: %s", full)
	}
	if !strings.Contains(full, "42") {
		t.Errorf("expected issue URL in command, got: %s", full)
	}
	if !strings.Contains(full, "--max-turns") {
		t.Errorf("expected --max-turns in command, got: %s", full)
	}
}

func TestDrainSingleJob(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeJob(1, "fix-bug"))

	cmdRunner := &mockCmdRunner{}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Completed)
	}
	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}
	if atomic.LoadInt32(&cmdRunner.started) != 1 {
		t.Errorf("expected claude started once, got %d", cmdRunner.started)
	}
	jobs, _ := q.List()
	if jobs[0].State != queue.StateCompleted {
		t.Errorf("expected job completed, got %s", jobs[0].State)
	}
}

func TestDrainJobFails(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeJob(1, "fix-bug"))

	cmdRunner := &mockCmdRunner{processErr: errors.New("exit status 1")}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
	jobs, _ := q.List()
	if jobs[0].State != queue.StateFailed {
		t.Errorf("expected job failed, got %s", jobs[0].State)
	}
}

func TestDrainWorktreeCreateFails(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeJob(1, "fix-bug"))

	cmdRunner := &mockCmdRunner{}
	wt := &mockWorktree{createErr: errors.New("git fetch failed")}
	r := New(cfg, q, wt, cmdRunner)

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed (worktree error), got %d", result.Failed)
	}
	if atomic.LoadInt32(&cmdRunner.started) != 0 {
		t.Error("claude should NOT be started when worktree fails")
	}
}

func TestDrainConcurrencyLimit(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	for i := 1; i <= 4; i++ {
		_ = q.Enqueue(makeJob(i, "fix-bug"))
	}

	counter := &countingCmdRunner{delay: 50 * time.Millisecond}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, counter)

	_, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	max := atomic.LoadInt32(&counter.maxSeen)
	if max > 2 {
		t.Errorf("concurrency exceeded limit: max concurrent = %d, limit = 2", max)
	}
}

func TestDrainContextCancel(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 1)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	for i := 1; i <= 5; i++ {
		_ = q.Enqueue(makeJob(i, "fix-bug"))
	}

	ctx, cancel := context.WithCancel(context.Background())

	counter := &countingCmdRunner{delay: 20 * time.Millisecond}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, counter)

	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	result, err := r.Drain(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	total := result.Completed + result.Failed + result.Skipped
	if total >= 5 {
		t.Errorf("expected cancellation to stop some jobs, but all 5 ran")
	}
}

func TestDrainTimeout(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 1)
	cfg.Timeout = "50ms"
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeJob(1, "fix-bug"))

	slow := &struct {
		mockCmdRunner
		delay time.Duration
	}{
		delay: 200 * time.Millisecond,
	}
	slow.delay = 200 * time.Millisecond

	called := false
	cmdRunner := &mockCmdRunner{
		processErr: context.DeadlineExceeded,
	}
	_ = called

	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("expected timed-out job to be marked failed, got completed=%d failed=%d", result.Completed, result.Failed)
	}
}

func TestMakeJobHighNumbers(t *testing.T) {
	// Verify makeJob works correctly for num >= 10 (was previously broken)
	for _, num := range []int{0, 1, 9, 10, 42, 100, 999} {
		job := makeJob(num, "fix-bug")
		wantID := fmt.Sprintf("issue-%d", num)
		wantURL := fmt.Sprintf("https://github.com/owner/repo/issues/%d", num)
		if job.ID != wantID {
			t.Errorf("makeJob(%d).ID = %q, want %q", num, job.ID, wantID)
		}
		if job.IssueURL != wantURL {
			t.Errorf("makeJob(%d).IssueURL = %q, want %q", num, job.IssueURL, wantURL)
		}
		if job.IssueNum != num {
			t.Errorf("makeJob(%d).IssueNum = %d, want %d", num, job.IssueNum, num)
		}
	}
}

func TestSlugifyEdgeCases(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", "task"},
		{"all special chars", "!@#$%^&*()", "task"},
		{"consecutive hyphens", "a---b---c", "a-b-c"},
		{"leading trailing special", "---hello---", "hello"},
		{"very long input", "abcdefghijklmnopqrstuvwxyz0123456789", "abcdefghijklmnopqrst"},
		{"truncation trims trailing hyphen at 21 chars", "abcdefghijklmnopqrs-x", "abcdefghijklmnopqrs"},
		{"truncation trims trailing hyphen", "abcdefghijklmnopqrs-xyz", "abcdefghijklmnopqrs"},
		{"no truncation at exactly 20 chars", "abcdefghijklmnopqrst", "abcdefghijklmnopqrst"},
		{"url with trailing slash", "https://example.com/path/", "path"},
		{"single char", "x", "x"},
		{"only hyphens after clean", "---", "task"},
		{"mixed case", "Hello-World", "hello-world"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := slugify(tc.input)
			if got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestBuildCommandEdgeCases(t *testing.T) {
	t.Run("empty template result", func(t *testing.T) {
		cfg := &config.Config{
			Claude: config.ClaudeConfig{
				Command:  "",
				Template: "{{.Command}}",
			},
		}
		job := &queue.Job{Skill: "fix-bug", IssueURL: "https://example.com"}
		_, _, err := buildCommand(cfg, job)
		if err == nil {
			t.Error("expected error for empty command, got nil")
		}
	})

	t.Run("invalid template syntax", func(t *testing.T) {
		cfg := &config.Config{
			Claude: config.ClaudeConfig{
				Template: "{{.Invalid",
			},
		}
		job := &queue.Job{}
		_, _, err := buildCommand(cfg, job)
		if err == nil {
			t.Error("expected error for invalid template, got nil")
		}
	})

	t.Run("template with only whitespace", func(t *testing.T) {
		cfg := &config.Config{
			Claude: config.ClaudeConfig{
				Command:  "   ",
				Template: "{{.Command}}",
			},
		}
		job := &queue.Job{}
		_, _, err := buildCommand(cfg, job)
		if err == nil {
			t.Error("expected error for whitespace-only command, got nil")
		}
	})

	t.Run("command with multiple args", func(t *testing.T) {
		cfg := &config.Config{
			MaxTurns: 10,
			Claude: config.ClaudeConfig{
				Command:  "claude",
				Template: "{{.Command}} --flag1 --flag2 value",
			},
		}
		job := &queue.Job{Skill: "fix-bug"}
		cmd, args, err := buildCommand(cfg, job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd != "claude" {
			t.Errorf("expected cmd 'claude', got %q", cmd)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d: %v", len(args), args)
		}
	})
}

func TestDrainConcurrencyLimitEnforced(t *testing.T) {
	// Test with more jobs and higher concurrency limit to verify the semaphore
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 3)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	for i := 1; i <= 20; i++ {
		_ = q.Enqueue(makeJob(i, "fix-bug"))
	}

	counter := &countingCmdRunner{delay: 30 * time.Millisecond}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, counter)

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Completed != 20 {
		t.Errorf("expected 20 completed, got %d (failed=%d, skipped=%d)", result.Completed, result.Failed, result.Skipped)
	}
	max := atomic.LoadInt32(&counter.maxSeen)
	if max > 3 {
		t.Errorf("concurrency exceeded limit: max concurrent = %d, limit = 3", max)
	}
	if max == 0 {
		t.Error("expected at least some concurrent execution, got max=0")
	}
}

func TestDrainEmptyQueue(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	// No jobs enqueued

	cmdRunner := &mockCmdRunner{}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Completed != 0 {
		t.Errorf("expected 0 completed, got %d", result.Completed)
	}
	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}
	if atomic.LoadInt32(&cmdRunner.started) != 0 {
		t.Error("no processes should have started on empty queue")
	}
}

func TestBranchPrefixSelection(t *testing.T) {
	// Verify that the branch prefix is "fix" for fix-related skills
	// and "feat" for non-fix skills.
	tests := []struct {
		skill      string
		wantPrefix string
	}{
		{"fix-bug", "fix"},
		{"Fix-Bug", "fix"},
		{"hotfix", "fix"},
		{"implement-feature", "feat"},
		{"add-docs", "feat"},
		{"refactor", "feat"},
	}

	for _, tc := range tests {
		t.Run(tc.skill, func(t *testing.T) {
			dir := t.TempDir()
			cfg := makeTestConfig(dir, 1)
			q := queue.New(filepath.Join(dir, "queue.jsonl"))
			_ = q.Enqueue(makeJob(1, tc.skill))

			var createdBranch string
			wt := &mockWorktree{}
			origCreate := wt.Create
			_ = origCreate // not a func field, use a tracking worktree instead

			// Use a tracking worktree to capture branch name
			tracker := &trackingWorktree{}
			cmdRunner := &mockCmdRunner{}
			r := New(cfg, q, tracker, cmdRunner)

			_, err := r.Drain(context.Background())
			if err != nil {
				t.Fatalf("drain: %v", err)
			}

			createdBranch = tracker.lastBranch
			wantPrefix := tc.wantPrefix + "/issue-1-"
			if !strings.HasPrefix(createdBranch, wantPrefix) {
				t.Errorf("for skill %q, expected branch prefix %q, got %q", tc.skill, wantPrefix, createdBranch)
			}
		})
	}
}

type trackingWorktree struct {
	lastBranch string
}

func (tw *trackingWorktree) Create(_ context.Context, branchName string) (string, error) {
	tw.lastBranch = branchName
	return ".claude/worktrees/" + branchName, nil
}

func TestBuildCommandTemplateExecutionError(t *testing.T) {
	// Template that references a non-existent field
	cfg := &config.Config{
		Claude: config.ClaudeConfig{
			Template: "{{.NonExistentField}}",
		},
	}
	job := &queue.Job{Skill: "fix-bug", IssueURL: "https://example.com"}
	_, _, err := buildCommand(cfg, job)
	if err == nil {
		t.Error("expected error for template referencing non-existent field")
	}
	if !strings.Contains(err.Error(), "execute template") {
		t.Errorf("expected execute template error, got: %v", err)
	}
}

func TestDrainMultipleFailures(t *testing.T) {
	// All jobs fail — verify all are counted.
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	for i := 1; i <= 5; i++ {
		_ = q.Enqueue(makeJob(i, "fix-bug"))
	}

	cmdRunner := &mockCmdRunner{processErr: errors.New("exit 1")}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Failed != 5 {
		t.Errorf("expected 5 failed, got %d (completed=%d)", result.Failed, result.Completed)
	}
	if result.Completed != 0 {
		t.Errorf("expected 0 completed, got %d", result.Completed)
	}

	// All jobs should be in failed state
	jobs, _ := q.List()
	for _, j := range jobs {
		if j.State != queue.StateFailed {
			t.Errorf("job %s: expected failed, got %s", j.ID, j.State)
		}
	}
}

func TestSlugifyUnicodeAndSpecialChars(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"unicode chars", "héllo-wörld", "h-llo-w-rld"},
		{"numbers only", "12345", "12345"},
		{"path with query", "https://example.com/issues/42?ref=main", "42-ref-main"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := slugify(tc.input)
			if got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
