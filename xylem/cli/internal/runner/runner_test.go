package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/workflow"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/source"
)

// --- Mock types ---

type mockCmdRunner struct {
	processErr   error
	outputErr    error
	outputData   []byte
	phaseOutputs map[string][]byte // keyed by substring in prompt
	phaseErr     error
	started      int32
	gateOutput   []byte
	gateErr      error
	// Track calls for assertion
	phaseCalls []phaseCall
	outputArgs [][]string
}

type phaseCall struct {
	dir    string
	prompt string
	name   string
	args   []string
}

func (m *mockCmdRunner) RunOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	m.outputArgs = append(m.outputArgs, append([]string{name}, args...))
	if m.gateOutput != nil {
		// If the command looks like a gate command (sh -c), return gate output
		for _, a := range args {
			if strings.Contains(a, "cd ") {
				return m.gateOutput, m.gateErr
			}
		}
	}
	if m.outputData != nil {
		return m.outputData, m.outputErr
	}
	return []byte{}, m.outputErr
}

func (m *mockCmdRunner) RunProcess(_ context.Context, _ string, _ string, _ ...string) error {
	atomic.AddInt32(&m.started, 1)
	return m.processErr
}

func (m *mockCmdRunner) RunPhase(_ context.Context, dir string, stdin io.Reader, name string, args ...string) ([]byte, error) {
	prompt, _ := io.ReadAll(stdin)
	m.phaseCalls = append(m.phaseCalls, phaseCall{
		dir:    dir,
		prompt: string(prompt),
		name:   name,
		args:   args,
	})
	atomic.AddInt32(&m.started, 1)

	// Return canned output based on prompt content
	for key, output := range m.phaseOutputs {
		if bytes.Contains(prompt, []byte(key)) {
			return output, m.phaseErr
		}
	}
	return []byte("mock output"), m.phaseErr
}

// mockExitError implements the exitCoder interface that gate.RunCommandGate checks.
type mockExitError struct {
	code int
}

func (e *mockExitError) Error() string    { return fmt.Sprintf("exit status %d", e.code) }
func (e *mockExitError) ExitCode() int    { return e.code }

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

func (c *countingCmdRunner) RunPhase(_ context.Context, _ string, stdin io.Reader, _ string, _ ...string) ([]byte, error) {
	io.ReadAll(stdin)
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
	return []byte("mock output"), nil
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

type trackingWorktree struct {
	lastBranch string
}

func (tw *trackingWorktree) Create(_ context.Context, branchName string) (string, error) {
	tw.lastBranch = branchName
	return ".claude/worktrees/" + branchName, nil
}

// --- Helpers ---

func makeTestConfig(dir string, concurrency int) *config.Config {
	return &config.Config{
		Concurrency: concurrency,
		MaxTurns:    50,
		Timeout:     "30s",
		StateDir:    dir,
		Claude: config.ClaudeConfig{
			Command: "claude",
		},
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

func makeVessel(num int, wf string) queue.Vessel {
	return queue.Vessel{
		ID:     fmt.Sprintf("issue-%d", num),
		Source: "github-issue",
		Ref:    fmt.Sprintf("https://github.com/owner/repo/issues/%d", num),
		Workflow: wf,
		Meta:   map[string]string{"issue_num": strconv.Itoa(num)},
		State:  queue.StatePending,
		CreatedAt: time.Now().UTC(),
	}
}

func makePromptVessel(num int, prompt string) queue.Vessel {
	return queue.Vessel{
		ID:     fmt.Sprintf("prompt-%d", num),
		Source: "manual",
		Prompt: prompt,
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
			Command: "claude",
		},
	}
	vessel := &queue.Vessel{
		Source: "github-issue",
		Workflow:  "fix-bug",
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
		t.Errorf("expected workflow in command, got: %s", full)
	}
	if !strings.Contains(full, "42") {
		t.Errorf("expected issue URL in command, got: %s", full)
	}
	if !strings.Contains(full, "--max-turns") {
		t.Errorf("expected --max-turns in command, got: %s", full)
	}
}

// writeWorkflowFile creates a workflow YAML and its prompt files in the given dir.
func writeWorkflowFile(t *testing.T, dir, name string, phases []testPhase) {
	t.Helper()
	workflowDir := filepath.Join(dir, ".xylem", "workflows")
	os.MkdirAll(workflowDir, 0o755)

	var phaseYAML strings.Builder
	for _, p := range phases {
		promptPath := filepath.Join(dir, ".xylem", "prompts", name, p.name+".md")
		os.MkdirAll(filepath.Dir(promptPath), 0o755)
		os.WriteFile(promptPath, []byte(p.promptContent), 0o644)

		phaseYAML.WriteString(fmt.Sprintf("  - name: %s\n", p.name))
		phaseYAML.WriteString(fmt.Sprintf("    prompt_file: %s\n", promptPath))
		phaseYAML.WriteString(fmt.Sprintf("    max_turns: %d\n", p.maxTurns))
		if p.gate != "" {
			phaseYAML.WriteString(fmt.Sprintf("    gate:\n%s\n", p.gate))
		}
		if p.allowedTools != "" {
			phaseYAML.WriteString(fmt.Sprintf("    allowed_tools: %q\n", p.allowedTools))
		}
	}

	workflowContent := fmt.Sprintf("name: %s\nphases:\n%s", name, phaseYAML.String())
	os.WriteFile(filepath.Join(workflowDir, name+".yaml"), []byte(workflowContent), 0o644)
}

func TestBuildCommandDirectPrompt(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command: "claude",
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
			Command: "claude",
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

func TestBuildCommandSkillBased(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command: "claude",
		},
	}
	vessel := &queue.Vessel{
		Source: "github-issue",
		Workflow:  "fix-bug",
		Ref:    "https://github.com/owner/repo/issues/42",
	}
	cmd, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "claude" {
		t.Errorf("expected cmd 'claude', got %q", cmd)
	}
	// Should produce: -p "/fix-bug https://..." --max-turns 50
	if args[0] != "-p" {
		t.Errorf("expected -p flag, got %q", args[0])
	}
	if !strings.Contains(args[1], "/fix-bug") {
		t.Errorf("expected workflow in prompt, got %q", args[1])
	}
	if !strings.Contains(args[1], "issues/42") {
		t.Errorf("expected ref in prompt, got %q", args[1])
	}
}

type testPhase struct {
	name          string
	promptContent string
	maxTurns      int
	gate          string
	allowedTools  string
}

// --- Tests ---

func TestDrainSingleVessel(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{name: "analyze", promptContent: "Analyze the issue", maxTurns: 5},
	})
	cfg.StateDir = filepath.Join(dir, ".xylem")

	// Override workflow path: change working directory context
	// We need the workflow to be loadable from .xylem/workflows/fix-bug.yaml
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

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

func TestDrainMultiPhaseSkill(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{name: "analyze", promptContent: "Analyze: {{.Issue.Title}}", maxTurns: 5},
		{name: "implement", promptContent: "Implement based on: {{.PreviousOutputs.analyze}}", maxTurns: 10},
		{name: "pr", promptContent: "Create PR", maxTurns: 3},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cmdRunner := &mockCmdRunner{
		phaseOutputs: map[string][]byte{
			"Analyze":   []byte("analysis result"),
			"Implement": []byte("implementation done"),
			"Create PR": []byte("PR created"),
		},
	}
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

	// All 3 phases should have been invoked
	if len(cmdRunner.phaseCalls) != 3 {
		t.Fatalf("expected 3 phase calls, got %d", len(cmdRunner.phaseCalls))
	}

	// Verify output files exist
	for _, pName := range []string{"analyze", "implement", "pr"} {
		outputPath := filepath.Join(dir, ".xylem", "phases", "issue-1", pName+".output")
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("expected output file %s to exist", outputPath)
		}
	}
}

func TestDrainPhaseFailsStopsSubsequent(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{name: "analyze", promptContent: "Analyze", maxTurns: 5},
		{name: "implement", promptContent: "Implement", maxTurns: 10},
		{name: "pr", promptContent: "PR", maxTurns: 3},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cmdRunner := &mockCmdRunner{
		phaseOutputs: map[string][]byte{
			"Analyze": []byte("analysis done"),
		},
		phaseErr: errors.New("exit status 1"),
	}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}

	// Only 1 phase should have been called (first one fails)
	if len(cmdRunner.phaseCalls) != 1 {
		t.Errorf("expected 1 phase call (stopped at failure), got %d", len(cmdRunner.phaseCalls))
	}

	vessels, _ := q.List()
	if vessels[0].State != queue.StateFailed {
		t.Errorf("expected vessel failed, got %s", vessels[0].State)
	}
}

func TestDrainPromptOnlyVessel(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makePromptVessel(1, "Fix the null pointer in handler.go"))

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

	// Should be exactly 1 RunPhase call
	if len(cmdRunner.phaseCalls) != 1 {
		t.Fatalf("expected 1 phase call for prompt-only, got %d", len(cmdRunner.phaseCalls))
	}

	// Verify the prompt was passed via stdin
	call := cmdRunner.phaseCalls[0]
	if !strings.Contains(call.prompt, "Fix the null pointer in handler.go") {
		t.Errorf("expected prompt in stdin, got: %s", call.prompt)
	}

	// Verify -p flag is present
	hasP := false
	for _, a := range call.args {
		if a == "-p" {
			hasP = true
			break
		}
	}
	if !hasP {
		t.Errorf("expected -p flag in args, got: %v", call.args)
	}

	vessels, _ := q.List()
	if vessels[0].State != queue.StateCompleted {
		t.Errorf("expected vessel completed, got %s", vessels[0].State)
	}
}

func TestDrainPromptOnlyWithRef(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	v := makePromptVessel(1, "Fix the null pointer")
	v.Ref = "https://github.com/owner/repo/issues/99"
	_ = q.Enqueue(v)

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

	call := cmdRunner.phaseCalls[0]
	if !strings.Contains(call.prompt, "Ref: https://github.com/owner/repo/issues/99") {
		t.Errorf("expected ref in prompt, got: %s", call.prompt)
	}
	if !strings.Contains(call.prompt, "Fix the null pointer") {
		t.Errorf("expected original prompt in stdin, got: %s", call.prompt)
	}
}

func TestDrainCommandGatePasses(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{
			name: "implement", promptContent: "Implement fix", maxTurns: 10,
			gate: "      type: command\n      run: \"make test\"",
		},
		{name: "pr", promptContent: "Create PR", maxTurns: 3},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cmdRunner := &mockCmdRunner{
		gateOutput: []byte("all tests passed"),
		gateErr:    nil, // gate passes
	}
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

	// Both phases should have run
	if len(cmdRunner.phaseCalls) != 2 {
		t.Errorf("expected 2 phase calls (gate passed), got %d", len(cmdRunner.phaseCalls))
	}
}

func TestDrainCommandGateFailsNoRetries(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{
			name: "implement", promptContent: "Implement fix", maxTurns: 10,
			gate: "      type: command\n      run: \"make test\"\n      retries: 0",
		},
		{name: "pr", promptContent: "Create PR", maxTurns: 3},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cmdRunner := &mockCmdRunner{
		gateOutput: []byte("FAIL: TestFoo"),
		gateErr:    errors.New("exit status 1"), // gate fails
	}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)
	r.Sources = map[string]source.Source{
		"github-issue": makeGitHubSource(),
	}

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}

	// Only 1 phase call (gate failed, no retry, phase 2 not invoked)
	if len(cmdRunner.phaseCalls) != 1 {
		t.Errorf("expected 1 phase call, got %d", len(cmdRunner.phaseCalls))
	}
}

func TestDrainCommandGateFailsWithRetries(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{
			name: "implement", promptContent: "Implement fix\n{{.GateResult}}", maxTurns: 10,
			gate: "      type: command\n      run: \"make test\"\n      retries: 2\n      retry_delay: \"1ms\"",
		},
		{name: "pr", promptContent: "Create PR", maxTurns: 3},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cmdRunner := &mockCmdRunner{
		gateOutput: []byte("FAIL: TestFoo"),
		gateErr:    &mockExitError{code: 1}, // gate always fails (non-zero exit)
	}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)
	r.Sources = map[string]source.Source{
		"github-issue": makeGitHubSource(),
	}

	result, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}

	// Phase should be invoked 3 times total: initial + 2 retries
	if len(cmdRunner.phaseCalls) != 3 {
		t.Errorf("expected 3 phase calls (1 + 2 retries), got %d", len(cmdRunner.phaseCalls))
	}

	// The 2nd and 3rd calls should include gate result context in rendered prompt
	for i := 1; i < len(cmdRunner.phaseCalls); i++ {
		if !strings.Contains(cmdRunner.phaseCalls[i].prompt, "gate check failed") {
			t.Errorf("retry %d prompt should contain gate failure context, got: %s", i, cmdRunner.phaseCalls[i].prompt)
		}
	}
}

func TestDrainLabelGateTransitionsToWaiting(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{
			name: "plan", promptContent: "Create plan", maxTurns: 5,
			gate: "      type: label\n      wait_for: \"plan-approved\"\n      timeout: \"24h\"",
		},
		{name: "implement", promptContent: "Implement", maxTurns: 10},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

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
	if result.Waiting != 1 {
		t.Errorf("expected 1 waiting, got %d", result.Waiting)
	}

	// Only 1 phase should have been invoked (plan)
	if len(cmdRunner.phaseCalls) != 1 {
		t.Errorf("expected 1 phase call, got %d", len(cmdRunner.phaseCalls))
	}

	vessels, _ := q.List()
	if vessels[0].State != queue.StateWaiting {
		t.Errorf("expected vessel waiting, got %s", vessels[0].State)
	}
	if vessels[0].WaitingFor != "plan-approved" {
		t.Errorf("expected WaitingFor=plan-approved, got %s", vessels[0].WaitingFor)
	}
	if vessels[0].WaitingSince == nil {
		t.Error("expected WaitingSince to be set")
	}
}

func TestDrainVesselFails(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{name: "analyze", promptContent: "Analyze", maxTurns: 5},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cmdRunner := &mockCmdRunner{phaseErr: errors.New("exit status 1")}
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
	cfg.StateDir = filepath.Join(dir, ".xylem")
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
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	// Create workflow and enqueue vessels
	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{name: "analyze", promptContent: "Analyze", maxTurns: 5},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	for i := 1; i <= 4; i++ {
		_ = q.Enqueue(makeVessel(i, "fix-bug"))
	}

	counter := &countingCmdRunner{delay: 50 * time.Millisecond}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, counter)
	r.Sources = map[string]source.Source{
		"github-issue": makeGitHubSource(),
	}

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
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{name: "analyze", promptContent: "Analyze", maxTurns: 5},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	for i := 1; i <= 5; i++ {
		_ = q.Enqueue(makeVessel(i, "fix-bug"))
	}

	ctx, cancel := context.WithCancel(context.Background())

	counter := &countingCmdRunner{delay: 20 * time.Millisecond}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, counter)
	r.Sources = map[string]source.Source{
		"github-issue": makeGitHubSource(),
	}

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

func TestBuildCommandEmptyCommand(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command: "",
		},
	}
	vessel := &queue.Vessel{Workflow: "fix-bug", Ref: "https://example.com"}
	cmd, _, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty command is returned as-is; the OS will fail to exec it
	if cmd != "" {
		t.Errorf("expected empty command, got %q", cmd)
	}
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

func TestDrainHarnessAppended(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{name: "analyze", promptContent: "Analyze", maxTurns: 5},
	})

	// Write harness file
	harnessDir := filepath.Join(dir, ".xylem")
	os.MkdirAll(harnessDir, 0o755)
	os.WriteFile(filepath.Join(harnessDir, "HARNESS.md"), []byte("Golden rules for this repo"), 0o644)

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cmdRunner := &mockCmdRunner{}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)
	r.Sources = map[string]source.Source{
		"github-issue": makeGitHubSource(),
	}

	_, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cmdRunner.phaseCalls) != 1 {
		t.Fatalf("expected 1 phase call, got %d", len(cmdRunner.phaseCalls))
	}

	// Check that --append-system-prompt is in the args
	call := cmdRunner.phaseCalls[0]
	found := false
	for i, a := range call.args {
		if a == "--append-system-prompt" && i+1 < len(call.args) {
			if strings.Contains(call.args[i+1], "Golden rules") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected --append-system-prompt with harness content, got args: %v", call.args)
	}
}

func TestDrainPreviousOutputsAvailable(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 2)
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{name: "analyze", promptContent: "Analyze the issue", maxTurns: 5},
		{name: "implement", promptContent: "Previous analysis: {{.PreviousOutputs.analyze}}", maxTurns: 10},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cmdRunner := &mockCmdRunner{
		phaseOutputs: map[string][]byte{
			"Analyze": []byte("found root cause in auth.go"),
		},
	}
	wt := &mockWorktree{}
	r := New(cfg, q, wt, cmdRunner)
	r.Sources = map[string]source.Source{
		"github-issue": makeGitHubSource(),
	}

	_, err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cmdRunner.phaseCalls) != 2 {
		t.Fatalf("expected 2 phase calls, got %d", len(cmdRunner.phaseCalls))
	}

	// Second phase prompt should contain the first phase's output
	secondPrompt := cmdRunner.phaseCalls[1].prompt
	if !strings.Contains(secondPrompt, "found root cause in auth.go") {
		t.Errorf("expected previous output in second phase prompt, got: %s", secondPrompt)
	}
}

func TestBranchPrefixSelection(t *testing.T) {
	tests := []struct {
		workflow   string
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
		t.Run(tc.workflow, func(t *testing.T) {
			dir := t.TempDir()
			cfg := makeTestConfig(dir, 1)
			cfg.StateDir = filepath.Join(dir, ".xylem")
			q := queue.New(filepath.Join(dir, "queue.jsonl"))
			_ = q.Enqueue(makeVessel(1, tc.workflow))

			writeWorkflowFile(t, dir, tc.workflow, []testPhase{
				{name: "work", promptContent: "Do work", maxTurns: 5},
			})

			oldWd, _ := os.Getwd()
			os.Chdir(dir)
			defer os.Chdir(oldWd)

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
				t.Errorf("for workflow %q, expected branch prefix %q, got %q", tc.workflow, wantPrefix, createdBranch)
			}
		})
	}
}

func TestBuildCommandWithFlags(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command: "claude",
			Flags:   "--bare --dangerously-skip-permissions",
		},
	}
	vessel := &queue.Vessel{
		Source: "github-issue",
		Workflow:  "fix-bug",
		Ref:    "https://github.com/owner/repo/issues/42",
	}
	_, args, err := buildCommand(cfg, vessel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have: -p, prompt, --max-turns, 50, --bare, --dangerously-skip-permissions
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d: %v", len(args), args)
	}
	if args[4] != "--bare" {
		t.Errorf("expected --bare flag, got %q", args[4])
	}
	if args[5] != "--dangerously-skip-permissions" {
		t.Errorf("expected --dangerously-skip-permissions flag, got %q", args[5])
	}
}

func TestBuildCommandNoFlags(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command: "claude",
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
		t.Fatalf("expected 4 args (no extra flags), got %d: %v", len(args), args)
	}
}

func TestBuildCommandDirectPromptWithFlags(t *testing.T) {
	cfg := &config.Config{
		MaxTurns: 50,
		Claude: config.ClaudeConfig{
			Command: "claude",
			Flags:   "--bare",
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
	// Should have: -p, prompt, --max-turns, 50, --bare
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d: %v", len(args), args)
	}
	if args[4] != "--bare" {
		t.Errorf("expected --bare flag, got %q", args[4])
	}
}

func TestBuildPhaseArgs(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		cfg := &config.Config{
			Claude: config.ClaudeConfig{Command: "claude"},
		}
		p := &workflow.Phase{Name: "analyze", MaxTurns: 5}
		args := buildPhaseArgs(cfg, p, "")

		if args[0] != "-p" {
			t.Errorf("expected -p, got %s", args[0])
		}
		if args[1] != "--max-turns" || args[2] != "5" {
			t.Errorf("expected --max-turns 5, got %v", args[1:3])
		}
	})

	t.Run("with flags", func(t *testing.T) {
		cfg := &config.Config{
			Claude: config.ClaudeConfig{Command: "claude", Flags: "--bare --dangerously-skip-permissions"},
		}
		p := &workflow.Phase{Name: "analyze", MaxTurns: 5}
		args := buildPhaseArgs(cfg, p, "")

		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--bare") {
			t.Errorf("expected --bare in args, got: %v", args)
		}
		if !strings.Contains(joined, "--dangerously-skip-permissions") {
			t.Errorf("expected --dangerously-skip-permissions in args, got: %v", args)
		}
	})

	t.Run("with allowed tools", func(t *testing.T) {
		cfg := &config.Config{
			Claude: config.ClaudeConfig{Command: "claude", AllowedTools: []string{"WebFetch"}},
		}
		allowedTools := "Read,Edit"
		p := &workflow.Phase{Name: "analyze", MaxTurns: 5, AllowedTools: &allowedTools}
		args := buildPhaseArgs(cfg, p, "")

		// Should have both phase-level and config-level tools
		toolCount := 0
		for _, a := range args {
			if a == "--allowedTools" {
				toolCount++
			}
		}
		if toolCount != 2 {
			t.Errorf("expected 2 --allowedTools flags, got %d in %v", toolCount, args)
		}
	})

	t.Run("with harness", func(t *testing.T) {
		cfg := &config.Config{
			Claude: config.ClaudeConfig{Command: "claude"},
		}
		p := &workflow.Phase{Name: "analyze", MaxTurns: 5}
		args := buildPhaseArgs(cfg, p, "harness content")

		found := false
		for i, a := range args {
			if a == "--append-system-prompt" && i+1 < len(args) && args[i+1] == "harness content" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected --append-system-prompt with harness, got: %v", args)
		}
	})
}

func TestDrainTimeoutV2Phase(t *testing.T) {
	dir := t.TempDir()
	cfg := makeTestConfig(dir, 1)
	cfg.Timeout = "50ms"
	cfg.StateDir = filepath.Join(dir, ".xylem")
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	_ = q.Enqueue(makeVessel(1, "fix-bug"))

	writeWorkflowFile(t, dir, "fix-bug", []testPhase{
		{name: "analyze", promptContent: "Analyze", maxTurns: 5},
	})

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cmdRunner := &mockCmdRunner{
		phaseErr: context.DeadlineExceeded,
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
