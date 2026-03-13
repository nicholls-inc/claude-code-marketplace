package runner

import (
	"context"
	"errors"
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
		ID:        "issue-" + strings.Repeat("0", 0) + string(rune('0'+num)),
		IssueURL:  "https://github.com/owner/repo/issues/" + string(rune('0'+num)),
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
