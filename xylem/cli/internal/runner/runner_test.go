package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/source"
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
		Concurrency: concurrency,
		MaxTurns:    50,
		Timeout:     "30s",
		StateDir:    dir,
		Claude:      config.ClaudeConfig{Command: "claude", Template: "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}"},
		Sources: map[string]config.SourceConfig{
			"github": {
				Type:    "github",
				Repo:    "owner/repo",
				Exclude: []string{"wontfix"},
				Tasks:   map[string]config.Task{"fix-bugs": {Labels: []string{"bug"}, Skill: "fix-bug"}},
			},
		},
	}
}

func makeVessel(num int, skill string) queue.Vessel {
	return queue.Vessel{
		ID:     fmt.Sprintf("issue-%d", num),
		Source: "github-issue",
		Ref:    fmt.Sprintf("https://github.com/owner/repo/issues/%d", num),
		Skill:  skill,
		Meta:   map[string]string{"issue_num": strconv.Itoa(num)},
		State:  queue.StatePending,
		CreatedAt: time.Now().UTC(),
	}
}

func makeGitHubSource() *source.GitHub {
	return &source.GitHub{
		Repo: "owner/repo",
	}
}

func TestBuildCommand(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:  "claude",
			Template: "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}",
		},
	}
	vessel := &queue.Vessel{
		Source: "github-issue",
		Skill:  "fix-bug",
		Ref:    "https://github.com/owner/repo/issues/42",
	}
	cmd, args, err := buildCommand(cfg, vessel)
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

func TestBuildCommandDirectPrompt(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:  "claude",
			Template: "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}",
		},
	}
	vessel := &queue.Vessel{
		Source: "manual",
		Prompt: "Fix the null pointer in handler.go",
	}
	cmd, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "claude" {
		t.Errorf("expected cmd 'claude', got %q", cmd)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if args[0] != "-p" {
		t.Errorf("expected -p flag, got %q", args[0])
	}
	if args[1] != "Fix the null pointer in handler.go" {
		t.Errorf("expected prompt text, got %q", args[1])
	}
}

func TestBuildCommandDirectPromptWithRef(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:  "claude",
			Template: "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}",
		},
	}
	vessel := &queue.Vessel{
		Source: "manual",
		Prompt: "Fix the null pointer in handler.go",
		Ref:    "https://github.com/owner/repo/issues/99",
	}
	cmd, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "claude" {
		t.Errorf("expected cmd 'claude', got %q", cmd)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if args[0] != "-p" {
		t.Errorf("expected -p flag, got %q", args[0])
	}
	// The prompt should contain the ref prepended
	if !strings.Contains(args[1], "Ref: https://github.com/owner/repo/issues/99") {
		t.Errorf("expected ref URL in prompt, got %q", args[1])
	}
	if !strings.Contains(args[1], "Fix the null pointer in handler.go") {
		t.Errorf("expected original prompt text in prompt, got %q", args[1])
	}
	// Ref should come before the prompt text
	refIdx := strings.Index(args[1], "Ref:")
	promptIdx := strings.Index(args[1], "Fix the null pointer")
	if refIdx >= promptIdx {
		t.Errorf("expected ref to come before prompt, ref at %d, prompt at %d", refIdx, promptIdx)
	}
}

func TestBuildCommandBackwardCompatIssueURL(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:  "claude",
			Template: "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}",
		},
	}
	vessel := &queue.Vessel{
		Source: "github-issue",
		Skill:  "fix-bug",
		Ref:    "https://github.com/owner/repo/issues/42",
	}
	cmd, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	full := cmd + " " + strings.Join(args, " ")
	if !strings.Contains(full, "issues/42") {
		t.Errorf("expected IssueURL backward compat to work, got: %s", full)
	}
}

func TestDrainSingleVessel(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	cmdRunner := &mockCmdRunner{}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)
	r.Sources = map[string]source.Source{
		"github-issue": makeGitHubSource(),
	}

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
	vessels, _ := q.List()
	if vessels[0].State != queue.StateCompleted {
		t.Errorf("expected vessel completed, got %s", vessels[0].State)
	}
}

func TestDrainVesselFails(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

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
	vessels, _ := q.List()
	if vessels[0].State != queue.StateFailed {
		t.Errorf("expected vessel failed, got %s", vessels[0].State)
	}
}

func TestDrainWorktreeCreateFails(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

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
		_ = q.Enqueue(makeVessel(i, "fix-bug"))
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
		_ = q.Enqueue(makeVessel(i, "fix-bug"))
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
		t.Errorf("expected cancellation to stop some vessels, but all 5 ran")
	}
}

func TestDrainTimeout(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 1)
	cfg.Timeout = "50ms"
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	cmdRunner := &mockCmdRunner{
		processErr: context.DeadlineExceeded,
	}

	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("expected timed-out vessel to be marked failed, got completed=%d failed=%d", result.Completed, result.Failed)
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
		vessel := &queue.Vessel{Skill: "fix-bug", Ref: "https://example.com"}
		_, _, err := buildCommand(cfg, vessel)
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
		vessel := &queue.Vessel{}
		_, _, err := buildCommand(cfg, vessel)
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
		vessel := &queue.Vessel{}
		_, _, err := buildCommand(cfg, vessel)
		if err == nil {
			t.Error("expected error for whitespace-only command, got nil")
		}
	})

}

func TestDrainEmptyQueue(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

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
			_ = q.Enqueue(makeVessel(1, tc.skill))

			tracker := &trackingWorktree{}
			cmdRunner := &mockCmdRunner{}
			r := New(cfg, q, tracker, cmdRunner)
			r.Sources = map[string]source.Source{
				"github-issue": makeGitHubSource(),
			}

			_, err := r.Drain(context.Background())
			if err != nil {
				t.Fatalf("drain: %v", err)
			}

			createdBranch := tracker.lastBranch
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

func TestBuildCommandAllowedToolsNone(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:  "claude",
			Template: "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}",
		},
	}
	vessel := &queue.Vessel{
		Source: "github-issue",
		Skill:  "fix-bug",
		Ref:    "https://github.com/owner/repo/issues/42",
	}
	_, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, arg := range args {
		if arg == "--allowedTools" {
			t.Error("expected no --allowedTools flag when AllowedTools is empty")
		}
	}
}

func TestBuildCommandAllowedToolsOne(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:      "claude",
			Template:     "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}",
			AllowedTools: []string{"WebFetch"},
		},
	}
	vessel := &queue.Vessel{
		Source: "github-issue",
		Skill:  "fix-bug",
		Ref:    "https://github.com/owner/repo/issues/42",
	}
	_, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Count --allowedTools flags
	count := 0
	for i, arg := range args {
		if arg == "--allowedTools" {
			count++
			if i+1 >= len(args) {
				t.Fatal("--allowedTools at end of args with no value")
			}
			if args[i+1] != "WebFetch" {
				t.Errorf("expected value %q after --allowedTools, got %q", "WebFetch", args[i+1])
			}
		}
	}
	if count != 1 {
		t.Errorf("expected 1 --allowedTools flag, got %d in args: %v", count, args)
	}
}

func TestBuildCommandAllowedToolsMultiple(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:      "claude",
			Template:     "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}",
			AllowedTools: []string{"Bash(gh issue view *)", "Bash(gh pr create *)", "WebFetch"},
		},
	}
	vessel := &queue.Vessel{
		Source: "github-issue",
		Skill:  "fix-bug",
		Ref:    "https://github.com/owner/repo/issues/42",
	}
	_, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count --allowedTools flags
	count := 0
	for i, arg := range args {
		if arg == "--allowedTools" {
			count++
			if i+1 >= len(args) {
				t.Fatal("--allowedTools at end of args with no value")
			}
		}
	}
	if count != 3 {
		t.Errorf("expected 3 --allowedTools flags, got %d in args: %v", count, args)
	}

	// Verify each tool appears as a value after --allowedTools
	wantTools := []string{"Bash(gh issue view *)", "Bash(gh pr create *)", "WebFetch"}
	toolIdx := 0
	for i, arg := range args {
		if arg == "--allowedTools" && i+1 < len(args) {
			if args[i+1] != wantTools[toolIdx] {
				t.Errorf("expected tool %q at position %d, got %q", wantTools[toolIdx], toolIdx, args[i+1])
			}
			toolIdx++
		}
	}
}

func TestBuildCommandAllowedToolsDirectPrompt(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:      "claude",
			Template:     "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}",
			AllowedTools: []string{"Bash(gh issue view *)", "WebFetch"},
		},
	}
	vessel := &queue.Vessel{
		Source: "manual",
		Prompt: "Fix the null pointer in handler.go",
	}
	cmd, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "claude" {
		t.Errorf("expected cmd 'claude', got %q", cmd)
	}

	// Should have: -p, prompt, --max-turns, 50, --allowedTools, tool1, --allowedTools, tool2
	expectedLen := 8
	if len(args) != expectedLen {
		t.Fatalf("expected %d args, got %d: %v", expectedLen, len(args), args)
	}

	// Verify the --allowedTools flags come after --max-turns
	if args[4] != "--allowedTools" || args[5] != "Bash(gh issue view *)" {
		t.Errorf("expected first allowed tool, got args[4:6]=%v", args[4:6])
	}
	if args[6] != "--allowedTools" || args[7] != "WebFetch" {
		t.Errorf("expected second allowed tool, got args[6:8]=%v", args[6:8])
	}
}

func TestBuildCommandAllowedToolsDirectPromptNone(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command:  "claude",
			Template: "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}",
			// No AllowedTools
		},
	}
	vessel := &queue.Vessel{
		Source: "manual",
		Prompt: "Fix the null pointer in handler.go",
	}
	_, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have exactly: -p, prompt, --max-turns, 50
	if len(args) != 4 {
		t.Fatalf("expected 4 args (no --allowedTools), got %d: %v", len(args), args)
	}
}

func TestBuildCommandTemplateExecutionError(t *testing.T) {
	cfg := &config.Config{
		Claude: config.ClaudeConfig{
			Template: "{{.NonExistentField}}",
		},
	}
	vessel := &queue.Vessel{Skill: "fix-bug", Ref: "https://example.com"}
	_, _, err := buildCommand(cfg, vessel)
	if err == nil {
		t.Error("expected error for template referencing non-existent field")
	}
	if !strings.Contains(err.Error(), "execute template") {
		t.Errorf("expected execute template error, got: %v", err)
	}
}

