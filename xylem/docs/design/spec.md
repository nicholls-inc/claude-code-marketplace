# Xylem v2: Phased Execution Spec

## Document Purpose

User stories and acceptance criteria for xylem v2 — a clean break from v1's single-shot `claude -p` invocations, replacing them with a multi-phase, gate-verified execution system with harness engineering principles. There is no backward compatibility with v1's `claude.template` config format. Every claim about Claude Code CLI behavior is verified against official documentation. Each user story includes concrete acceptance criteria that can be tested during development shakedowns.

---

## Verified Claude Code CLI Capabilities

The following flags and behaviors are confirmed in the official Claude Code documentation (code.claude.com) as of March 2026. The spec only uses these capabilities.

| Flag | Behavior | Source |
|------|----------|--------|
| `-p` / `--print` | Non-interactive mode. Executes a single prompt and exits. | code.claude.com/docs/en/headless |
| `--bare` | Skips hooks, LSP, plugin sync, skill directory walks. Requires `ANTHROPIC_API_KEY` or `apiKeyHelper` via `--settings`. Skips OAuth/keychain reads. | code.claude.com/docs/en/headless |
| `--append-system-prompt <text>` | Appends to the default system prompt. Works in both interactive and non-interactive modes. | code.claude.com/docs/en/cli-reference |
| `--system-prompt-file <path>` | Replaces the system prompt with contents of a file. Mutually exclusive with `--system-prompt`. | code.claude.com/docs/en/cli-reference |
| `--max-turns <n>` | Limits the number of agent iterations. | code.claude.com/docs/en/headless |
| `--output-format json` | Returns structured JSON with metadata. | code.claude.com/docs/en/headless |
| `--output-format stream-json` | Real-time streaming with newline-delimited JSON. | code.claude.com/docs/en/headless |
| `--output-format text` | Returns readable text (default). | code.claude.com/docs/en/headless |
| `--allowedTools <list>` | Pre-approves specific tools without prompting. Uses permission rule syntax. | code.claude.com/docs/en/headless |
| `--dangerously-skip-permissions` | Bypasses all permission prompts. | code.claude.com/docs/en/headless |
| `--continue` | Continues the most recent conversation. | code.claude.com/docs/en/headless |
| `--resume <session-id>` | Continues a specific conversation by session ID. | code.claude.com/docs/en/headless |
| stdin piping | Content piped to stdin is available to Claude. | code.claude.com/docs/en/headless |
| Exit code 0 | Indicates success. Non-zero indicates error. | code.claude.com/docs/en/headless |

### Authentication for Automated Use

- `--bare` **requires** `ANTHROPIC_API_KEY` or `apiKeyHelper`. It does not support Max/Pro subscription OAuth.
- Without `--bare`, `claude -p` loads hooks, plugins, skills, CLAUDE.md, and can authenticate via OAuth (Max subscription). However, startup is slower.
- If `ANTHROPIC_API_KEY` is set in the environment, `claude -p` will use it regardless of subscription status — this has caused unintended API billing for subscription users.
- Anthropic's published guidance states: subscription accounts are designed for interactive, human-directed usage. Automated/CI usage should use the API with proper rate limiting.

### Xylem v1 Baseline (Being Replaced)

Per the existing README, xylem v1 provides the following. This spec replaces the runner and config schema while preserving the CLI commands and source/queue architecture:

- **`xylem scan`**: Queries GitHub issues by label via `gh` CLI, enqueues matching issues as "vessels" — **kept**
- **`xylem drain`**: Dequeues pending vessels, creates git worktrees, launches `claude -p` — **runner logic replaced with multi-phase execution**
- **`xylem enqueue`**: Manual task enqueue with `--workflow`, `--ref`, `--prompt`, or `--prompt-file` — **kept**
- **`xylem status`**: Shows queue state (pending/running/completed/failed) — **extended with `waiting` and `timed_out` states**
- **`xylem pause` / `resume`**: Pauses/resumes scanning — **kept**
- **`xylem cancel`**: Cancels pending vessels — **kept**
- **`xylem cleanup`**: Removes stale git worktrees — **kept**
- **Pluggable sources**: `github` and `manual` — **kept**
- **Configurable concurrency**: Max simultaneous Claude sessions — **kept**
- **Go template for claude invocation**: `{{.Command}} -p "/{{.Workflow}} {{.Ref}}" --max-turns {{.MaxTurns}}` — **removed, replaced by phase-based invocation**
- **`.xylem.yml` config with `claude.template`** — **removed, replaced by `claude.flags` + workflow YAML files**
- **JSONL queue**: Persistent state in `.xylem/queue.jsonl` — **kept, extended with new vessel states and phase tracking**
- **Signal handling**: SIGINT/SIGTERM handled gracefully in `drain` — **kept**

### Known Limitations (from README)

- No auto-retry — failed vessels remain as `failed`
- No webhooks — polling only via cron
- No priority queues — FIFO only
- Cancel does not kill running sessions
- Sequential correctness only — no concurrency modeling in workflows
- GitHub is the only built-in scanning source

---

## Code Architecture

### Existing Package Structure

```
xylem/cli/
├── cmd/xylem/
│   ├── main.go           # cobra root setup
│   ├── root.go           # root command, config loading
│   ├── scan.go           # `xylem scan` command
│   ├── drain.go          # `xylem drain` command
│   ├── enqueue.go        # `xylem enqueue` command
│   ├── status.go         # `xylem status` command
│   ├── pause.go          # `xylem pause` command
│   ├── resume.go         # `xylem resume` command
│   ├── cancel.go         # `xylem cancel` command
│   ├── cleanup.go        # `xylem cleanup` command
│   ├── init.go           # `xylem init` command (already exists)
│   └── exec.go           # realCmdRunner (Run, RunOutput, RunProcess, RunPhase)
├── internal/
│   ├── config/config.go  # Config, SourceConfig, Task, ClaudeConfig structs
│   ├── queue/queue.go    # Vessel, Queue, VesselState, state transitions
│   ├── runner/runner.go  # Runner, Drain(), runVessel() — rewritten for phase-based execution
│   ├── scanner/scanner.go # Scanner, Scan(), buildSources()
│   ├── source/
│   │   ├── source.go     # Source interface (Name, Scan, OnStart, BranchName)
│   │   ├── github.go     # GitHub source (scans issues via gh CLI)
│   │   └── manual.go     # Manual source (no-op scan, used by enqueue)
│   └── worktree/worktree.go # Manager (Create, Remove, List)
├── go.mod
└── go.sum
```

### New Packages and Files

```
xylem/cli/
├── cmd/xylem/
│   ├── daemon.go         # NEW: `xylem daemon` command
│   └── retry.go          # NEW: `xylem retry` command
├── internal/
│   ├── config/config.go  # MODIFIED: new Config fields, remove ClaudeConfig.Template
│   ├── queue/queue.go    # MODIFIED: new vessel states, phase tracking fields
│   ├── runner/runner.go  # MODIFIED: replace runVessel with phase-based execution
│   ├── workflow/          # NEW PACKAGE
│   │   └── workflow.go    # Workflow, Phase, Gate structs; Load() and Validate()
│   ├── phase/            # NEW PACKAGE
│   │   └── phase.go      # PhaseRunner; executes a single phase, captures output
│   ├── gate/             # NEW PACKAGE
│   │   └── gate.go       # CommandGate, LabelGate; executes gates between phases
│   └── reporter/         # NEW PACKAGE
│       └── reporter.go   # Posts GitHub issue comments; wraps gh CLI calls
```

### Key Interfaces

The following interfaces exist in v1 and are preserved:

```go
// source.Source — unchanged
type Source interface {
    Name() string
    Scan(ctx context.Context) ([]queue.Vessel, error)
    OnStart(ctx context.Context, vessel queue.Vessel) error
    BranchName(vessel queue.Vessel) string
}

// runner.CommandRunner — unchanged
type CommandRunner interface {
    RunOutput(ctx context.Context, name string, args ...string) ([]byte, error)
    RunProcess(ctx context.Context, dir string, name string, args ...string) error
}

// runner.WorktreeManager — unchanged
type WorktreeManager interface {
    Create(ctx context.Context, branchName string) (string, error)
}
```

The `realCmdRunner` in `exec.go` is the production implementation. `RunProcess` sets `cmd.Dir`, connects `cmd.Stdout`/`cmd.Stderr` to `os.Stdout`/`os.Stderr`, and calls `cmd.Run()`.

### What Changes in the Runner

The current `runVessel()` method calls `buildCommand()` once and runs a single `RunProcess()` call. In v2, `runVessel()` is replaced with a loop that iterates through the workflow's phases. Each phase invocation uses the new `RunPhase` method which accepts stdin for prompt delivery and captures stdout (see Prompt Delivery below). The `buildCommand()` function is removed.

### Runner Flow (v2)

The `runVessel()` method in v2 follows this sequence:

```
runVessel(ctx, vessel):
  1. Resolve source → call source.OnStart(vessel)
  2. Worktree:
     - If vessel.WorktreePath is set (resuming from waiting): verify it exists, reuse it
     - Otherwise: create worktree → worktreeManager.Create(branchName)
       Store path in vessel.WorktreePath, persist to queue
  3. Load workflow definition → workflow.Load(".xylem/workflows/<vessel.Workflow>.yaml")
  4. If vessel has no Workflow (prompt-only): run single claude -p with vessel.Prompt via stdin, return
  5. Fetch issue data (GitHub source only, skip if already in vessel.Meta from previous run):
     → gh issue view <num> --repo <repo> --json title,body,labels,url
     → Store in vessel.Meta["issue_title"], vessel.Meta["issue_body"], vessel.Meta["issue_labels"]
     → If gh fails: vessel → failed, return
  6. Read harness file (if .xylem/HARNESS.md exists in repo root)
  7. Rebuild previousOutputs map from .xylem/phases/<id>/*.output (covers resume case)
  8. For each phase in workflow.Phases, starting from vessel.CurrentPhase:
     a. Build TemplateData from vessel, phase index, previousOutputs map, gateResult
     b. Render prompt template → write to .xylem/phases/<id>/<phase>.prompt
     c. Construct claude args: -p --max-turns N <flags> [--allowedTools X] [--append-system-prompt harness]
     d. Call RunPhase(ctx, worktreePath, promptReader, "claude", args...)
     e. Write captured stdout → .xylem/phases/<id>/<phase>.output
     f. Store output in previousOutputs[phase.Name]
     g. Update vessel.CurrentPhase = phase index + 1, persist to queue
     h. Report phase completion to GitHub issue (non-fatal)
     i. If phase has a gate:
        - If type=command: run gate command in worktree
          - If exit 0: continue to next phase
          - If exit non-zero and retries > 0: re-render prompt with gate output appended, go to (c)
          - If exit non-zero and retries exhausted: vessel → failed, report, return
        - If type=label: vessel → waiting, set WaitingSince/WaitingFor, return
          (next drain invocation will check and resume)
  9. All phases complete: vessel → completed, report summary
```

Steps 5, 6, and 7 happen once at the start, not per phase. The issue data and harness content are cached in local variables for the duration of the vessel's execution. Step 7 is critical for the resume case — it reads existing output files so that `{{.PreviousOutputs}}` is populated correctly even when resuming from a label gate.

---

## Go Types

### New Types

```go
// internal/workflow/workflow.go

package workflow

import (
    "fmt"
    "os"
    "time"
    "gopkg.in/yaml.v3"
)

type Workflow struct {
    Name        string  `yaml:"name"`
    Description string  `yaml:"description,omitempty"`
    Phases      []Phase `yaml:"phases"`
}

type Phase struct {
    Name         string `yaml:"name"`
    PromptFile   string `yaml:"prompt_file"`
    MaxTurns     int    `yaml:"max_turns"`
    Gate         *Gate  `yaml:"gate,omitempty"`
    AllowedTools string `yaml:"allowed_tools,omitempty"`
}

type Gate struct {
    Type         string `yaml:"type"`                      // "command" or "label"
    Run          string `yaml:"run,omitempty"`              // shell command (type=command)
    Retries      int    `yaml:"retries,omitempty"`          // default 0
    RetryDelay   string `yaml:"retry_delay,omitempty"`      // default "10s"
    WaitFor      string `yaml:"wait_for,omitempty"`         // label name (type=label)
    Timeout      string `yaml:"timeout,omitempty"`          // default "24h" (type=label)
    PollInterval string `yaml:"poll_interval,omitempty"`    // default "60s" (type=label)
}

func Load(path string) (*Workflow, error) { /* read file, unmarshal, validate */ }
func (w *Workflow) Validate(basePath string) error { /* see US-10 for validation rules */ }
```

### Modified Types

```go
// internal/queue/queue.go — additions to existing file

// New states added to VesselState
const (
    StateWaiting  VesselState = "waiting"
    StateTimedOut VesselState = "timed_out"
)

// Updated validTransitions map
var validTransitions = map[VesselState]map[VesselState]bool{
    StatePending: {
        StateRunning:   true,
        StateCancelled: true,
    },
    StateRunning: {
        StateCompleted: true,
        StateFailed:    true,
        StateCancelled: true,
        StateWaiting:   true,
    },
    StateWaiting: {
        StateRunning:   true,  // label gate passed, resume execution
        StateTimedOut:  true,  // label gate timed out
        StateCancelled: true,  // manually cancelled while waiting
    },
    StateFailed: {
        StatePending: true,    // allow retry
    },
    StateCompleted: {},
    StateCancelled: {},
    StateTimedOut:  {},
}

// New fields on Vessel struct
type Vessel struct {
    ID        string            `json:"id"`
    Source    string            `json:"source"`
    Ref       string            `json:"ref,omitempty"`
    Workflow   string            `json:"workflow,omitempty"`
    Prompt    string            `json:"prompt,omitempty"`
    Meta      map[string]string `json:"meta,omitempty"`
    State     VesselState       `json:"state"`
    CreatedAt time.Time         `json:"created_at"`
    StartedAt *time.Time        `json:"started_at,omitempty"`
    EndedAt   *time.Time        `json:"ended_at,omitempty"`
    Error     string            `json:"error,omitempty"`

    // v2 additions
    CurrentPhase  int               `json:"current_phase,omitempty"`      // 0-indexed
    PhaseOutputs  map[string]string `json:"phase_outputs,omitempty"`      // phase name → output file path
    GateRetries   int               `json:"gate_retries,omitempty"`       // remaining retries for current gate
    WaitingSince  *time.Time        `json:"waiting_since,omitempty"`      // when label gate started waiting
    WaitingFor    string            `json:"waiting_for,omitempty"`        // label name being waited on
    WorktreePath  string            `json:"worktree_path,omitempty"`      // worktree dir, set on first run, reused on resume
    FailedPhase   string            `json:"failed_phase,omitempty"`       // phase name that failed
    GateOutput    string            `json:"gate_output,omitempty"`        // last gate command output (for failure context)
    RetryOf       string            `json:"retry_of,omitempty"`           // ID of the vessel this retries
}
```

```go
// internal/config/config.go — updated ClaudeConfig

type ClaudeConfig struct {
    Command string            `yaml:"command"`
    Flags   string            `yaml:"flags,omitempty"`       // replaces Template
    Env     map[string]string `yaml:"env,omitempty"`         // environment variables for claude invocations
}

// Template field is removed. If present in YAML, config.Load returns an error.
```

### Template Data (passed to Go templates in prompt files)

```go
// internal/phase/phase.go

type TemplateData struct {
    Issue           IssueData
    Phase           PhaseData
    PreviousOutputs map[string]string  // phase name → output text
    GateResult      string             // most recent gate command output
    Vessel          VesselData
}

type IssueData struct {
    URL    string
    Title  string
    Body   string
    Labels []string
    Number int
}

type PhaseData struct {
    Name  string
    Index int
}

type VesselData struct {
    ID     string
    Source string
}
```

`IssueData` is populated from the vessel's `Meta` map. The GitHub source already stores `issue_num` in Meta. The issue title and body are fetched via `gh issue view <number> --repo <repo> --json title,body,labels` at the start of the first phase and cached in the vessel's Meta for subsequent phases.

---

## Prompt Delivery

### Problem

Rendered prompts include the issue body, previous phase outputs, and gate results. These can exceed the Linux `ARG_MAX` limit (2MB total for all args + environment) or the per-argument `MAX_ARG_STRLEN` limit (~128KB).

### Solution

Prompts are piped via stdin. The `claude -p` flag accepts input from either a CLI argument or stdin. When stdin is provided without a prompt argument, stdin is used as the prompt. This is confirmed behavior documented at code.claude.com/docs/en/headless and demonstrated in the official example: `gh pr diff "$1" | claude -p --append-system-prompt "..."`.

### Implementation

The runner writes the rendered prompt to a temp file, then pipes it to Claude via Go's `exec.Cmd.Stdin`:

```go
// In the phase runner
promptFile := filepath.Join(stateDir, "phases", vessel.ID, phase.Name+".prompt")
os.WriteFile(promptFile, []byte(renderedPrompt), 0o644)

f, _ := os.Open(promptFile)
defer f.Close()

args := []string{"-p"}  // no prompt argument — stdin is the prompt
args = append(args, "--max-turns", strconv.Itoa(phase.MaxTurns))
args = append(args, strings.Fields(cfg.Claude.Flags)...)
if phase.AllowedTools != "" {
    args = append(args, "--allowedTools", phase.AllowedTools)
}
if harnessContent != "" {
    args = append(args, "--append-system-prompt", harnessContent)
}

cmd := exec.CommandContext(ctx, cfg.Claude.Command, args...)
cmd.Dir = worktreePath
cmd.Stdin = f
cmd.Stderr = os.Stderr

// Set environment: inherit current env, overlay claude.env values
cmd.Env = os.Environ()
for k, v := range cfg.Claude.Env {
    // Resolve ${VAR} references in values
    resolved := os.ExpandEnv(v)
    cmd.Env = append(cmd.Env, k+"="+resolved)
}

// Capture stdout for phase output
var stdout bytes.Buffer
cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)  // tee to terminal and buffer

err := cmd.Run()
// Write phase output
outputFile := filepath.Join(stateDir, "phases", vessel.ID, phase.Name+".output")
os.WriteFile(outputFile, stdout.Bytes(), 0o644)
```

The prompt file is also written to disk (`.prompt` extension) for debugging. The output file (`.output` extension) captures stdout for use in subsequent phases.

### `RunProcess` Changes

The current `RunProcess` method in `exec.go` does not support stdin. It must be extended:

```go
// Updated interface
type CommandRunner interface {
    RunOutput(ctx context.Context, name string, args ...string) ([]byte, error)
    RunProcess(ctx context.Context, dir string, name string, args ...string) error
    // New method for phase execution with stdin and stdout capture
    RunPhase(ctx context.Context, dir string, stdin io.Reader, name string, args ...string) ([]byte, error)
}
```

`RunPhase` sets `cmd.Dir`, `cmd.Stdin`, captures stdout into a `bytes.Buffer` while teeing to `os.Stdout`, connects stderr to `os.Stderr`, and returns the captured stdout bytes plus any error.

Concrete implementation in `exec.go`:

```go
func (r *realCmdRunner) RunPhase(ctx context.Context, dir string, stdin io.Reader, name string, args ...string) ([]byte, error) {
    cmd := exec.CommandContext(ctx, name, args...)
    cmd.Dir = dir
    cmd.Stdin = stdin
    cmd.Stderr = os.Stderr

    var stdout bytes.Buffer
    cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)

    err := cmd.Run()
    return stdout.Bytes(), err
}
```

The existing `RunProcess` method is kept for gate command execution (gates don't need stdin or stdout capture — their exit code is what matters). Gate commands use `RunOutput` which already captures combined output.

---

## Phase Output Storage

### Location

Phase outputs are stored under the state directory (default `.xylem`), not the worktree:

```
.xylem/
├── queue.jsonl
├── paused                  # marker file when scanning is paused
└── phases/
    └── <vessel-id>/
        ├── analyze.prompt   # rendered prompt sent to Claude
        ├── analyze.output   # captured stdout from Claude
        ├── plan.prompt
        ├── plan.output
        ├── implement.prompt
        ├── implement.output
        └── pr.prompt
        └── pr.output
```

This location is relative to the repo root, not the worktree. The state directory is shared across all worktrees since worktrees are created under `.claude/worktrees/`.

### Cleanup

`xylem cleanup` already removes stale worktrees. In v2, it also removes phase output directories for vessels in terminal states (`completed`, `failed`, `cancelled`, `timed_out`) older than 7 days (configurable via `cleanup_after` in config, default `"168h"`).

---

## Prompt Size Management

### Truncation Rules

When rendering `{{.PreviousOutputs}}`, each value is truncated to 16,000 characters. If truncated, the suffix `\n\n[... output truncated at 16000 characters]` is appended.

When rendering `{{.GateResult}}`, the value is truncated to 8,000 characters with the same suffix pattern.

When rendering `{{.Issue.Body}}`, the value is truncated to 32,000 characters.

These limits are constants in `internal/phase/phase.go`, not config values.

### Total Prompt Size

No hard limit is enforced on total prompt size since stdin has no length limit. The truncation rules above keep individual components reasonable for Claude's context window.

---

## Drain Behavior With Label Gates

### One-Shot Mode (`xylem drain`)

When a vessel hits a label gate during a one-shot `drain`:

1. The vessel transitions to `waiting` state with `WaitingSince` and `WaitingFor` set.
2. The vessel is written to the queue.
3. The `drain` goroutine for this vessel exits (releasing the concurrency slot).
4. `drain` continues processing other pending vessels.
5. `drain` returns normally. The waiting vessel remains in `waiting` state.

On the next `xylem drain` invocation, the runner checks all `waiting` vessels before dequeuing `pending` ones:

1. For each `waiting` vessel, poll the issue labels via `gh issue view <num> --repo <repo> --json labels`.
2. If the `WaitingFor` label is found: transition to `running`, increment `CurrentPhase` by 1, and dispatch the vessel to the runner. The runner reads `CurrentPhase` and skips completed phases, resuming from the next one. The vessel's worktree still exists from the previous run and is reused (not recreated).
3. If `time.Since(WaitingSince) > gate.Timeout`: transition to `timed_out`. Post timeout comment.
4. If neither: leave in `waiting` state. No concurrency slot consumed.

Note: `CurrentPhase` is persisted in the JSONL queue. When the runner picks up a vessel with `CurrentPhase > 0`, it skips phases 0 through `CurrentPhase - 1` (their outputs are already in `PhaseOutputs`). It also reads the existing phase output files from `.xylem/phases/<id>/` to repopulate the `previousOutputs` map.

This means label gates require repeated `drain` invocations (via cron or daemon mode) to make progress. A single `drain` invocation does not block.

### Daemon Mode (`xylem daemon`)

The daemon tick checks waiting vessels on every cycle, so label gates are resolved automatically without user intervention.

### Concurrency

Waiting vessels do NOT count against the concurrency limit. Only vessels actively running a `claude -p` subprocess occupy a slot.

---

## `gh` CLI Invocation Patterns

### Fetching Issue Details

Used at the start of the first phase to populate `IssueData`:

```bash
gh issue view <number> --repo <owner/name> --json title,body,labels,url
```

Response shape (verified from source/github.go):
```json
{
  "title": "Fix auth timeout",
  "body": "When users...",
  "url": "https://github.com/owner/name/issues/42",
  "labels": [{"name": "bug"}, {"name": "ready-for-work"}]
}
```

The issue number comes from `vessel.Meta["issue_num"]`, which the GitHub source populates during scan. The repo comes from the source config.

### Posting Comments

```bash
gh issue comment <number> --repo <owner/name> --body "<comment text>"
```

The `--body` argument has no practical length limit (it's sent via the GitHub API, not as a shell argument). However, GitHub truncates issue comments at 65,536 characters. The reporter truncates phase output to 64,000 characters to leave room for the surrounding markdown.

### Checking Labels (for label gates)

```bash
gh issue view <number> --repo <owner/name> --json labels
```

Response shape:
```json
{
  "labels": [{"name": "plan-approved"}, {"name": "bug"}]
}
```

### Auth Check

Before any `gh` operations, verify auth status:
```bash
gh auth status
```

This is checked once at the start of `drain` or `daemon` startup. If it fails, a warning is printed but xylem continues — `gh` failures in the reporter are non-fatal (see US-06 AC #6).

### Extracting Repo and Issue Number From Vessel

The GitHub source stores `issue_num` in `vessel.Meta["issue_num"]`. The repo (`owner/name`) is stored in the source config, not on the vessel itself. The runner needs access to the source config to resolve the repo for a given vessel. This is already available via `runner.Sources[vessel.Source]` — the `GitHub` struct has a `Repo` field.

---

## Error Handling

### Startup Errors (Fatal)

These cause xylem to exit with code 1 before any work begins:

| Error | Message |
|-------|---------|
| Config file not found | `error: read config file ".xylem.yml": no such file or directory` |
| Config YAML parse error | `error: parse config yaml: <yaml error>` |
| Config validation error | `error: <validation message>` (see config.Validate) |
| v1 `claude.template` present | `error: claude.template is no longer supported; migrate to phase-based workflows in .xylem/workflows/` |
| `--bare` without API key | `error: --bare requires ANTHROPIC_API_KEY in claude.env` |
| Workflow file not found | `error: workflow file not found: .xylem/workflows/<name>.yaml` |
| Workflow validation error | `error: .xylem/workflows/<name>.yaml: <validation message>` |
| Prompt file not found | `error: .xylem/workflows/<name>.yaml: phase "<phase>": prompt_file not found: <path>` |

### Runtime Errors (Per-Vessel)

These mark the individual vessel as `failed` but do not stop other vessels:

| Error | Behavior |
|-------|----------|
| Worktree creation fails | Vessel → `failed` with error message. |
| `gh issue view` fails during issue fetch | Vessel → `failed`. Cannot populate IssueData. |
| Prompt template render fails | Vessel → `failed` with template error. No `claude -p` invoked. |
| `claude -p` exits non-zero | Vessel → `failed` with exit code and stderr. |
| Gate command exits non-zero (retries exhausted) | Vessel → `failed` with gate output. |
| Label gate timeout | Vessel → `timed_out`. Comment posted. |
| Per-vessel timeout (config `timeout`) | Vessel context cancelled. `claude -p` process killed. Vessel → `failed`. |

### Non-Fatal Warnings

These are logged but do not affect vessel state:

| Warning | Behavior |
|---------|----------|
| `gh issue comment` fails | Warning logged. Phase execution continues. |
| `gh auth status` fails at startup | Warning logged. xylem continues. |
| Source `OnStart` fails | Warning logged (existing behavior). |
| Phase output file write fails | Warning logged. Subsequent phases may lack `PreviousOutputs` data. |
| Harness file read fails | Warning logged. `--append-system-prompt` omitted. |

---

## Testing

### Mock Strategy

The codebase already uses interface-based mocking. All external commands (`claude`, `gh`, `git`) are called through `CommandRunner` interfaces. Tests provide mock implementations that return canned responses.

The existing test files demonstrate the pattern:
- `runner/runner_test.go` (18KB) mocks `CommandRunner` and `WorktreeManager`
- `scanner/scanner_test.go` (16KB) mocks `CommandRunner` for `gh` calls
- `queue/queue_test.go` (16KB) uses temp files for the JSONL queue
- `worktree/worktree_test.go` (22KB) mocks `CommandRunner` for `git` calls

### New Test Requirements

#### `internal/workflow/workflow_test.go`

Table-driven tests for `Load()` and `Validate()`:
- Valid workflow file → loads without error
- Missing `phases` → returns specific error
- Duplicate phase names → returns specific error
- Missing `prompt_file` → returns specific error
- Non-existent `prompt_file` → returns specific error
- Invalid gate type → returns specific error
- Invalid duration string → returns specific error
- Empty `name` → returns specific error

#### `internal/phase/phase_test.go`

Table-driven tests for template rendering:
- Template with `{{.Issue.Title}}` → renders correctly
- Template with `{{.PreviousOutputs.analyze}}` → renders previous phase output
- Template with `{{.PreviousOutputs.nonexistent}}` → renders empty string, no error
- Template with syntax error → returns error, no `claude -p` invoked
- Template with `{{.GateResult}}` after gate failure → includes gate output
- Previous output exceeding 16,000 chars → truncated with suffix

#### `internal/gate/gate_test.go`

- Command gate with exit 0 → passes
- Command gate with exit 1, retries=0 → fails
- Command gate with exit 1, retries=2 → retries invoked, gate output captured
- Label gate with label present → passes
- Label gate with timeout → returns timeout error

#### `internal/reporter/reporter_test.go`

- Phase complete → `gh issue comment` called with correct format
- Vessel failed → failure comment posted
- `gh` command fails → warning logged, no error returned
- Manual source vessel → no comment posted
- Phase output > 64,000 chars → truncated in comment

#### `internal/runner/runner_test.go` (updated)

- Multi-phase workflow: all phases complete → vessel `completed`
- Multi-phase workflow: phase 2 fails → vessel `failed`, phase 3 not invoked
- Prompt-only vessel (no workflow) → single `claude -p` call, vessel `completed`
- Label gate: vessel transitions to `waiting`, next drain checks labels
- Waiting vessel does not count against concurrency
- Per-vessel timeout kills `claude -p` process

### Mocking `claude -p`

Tests do NOT invoke the real `claude` binary. The `CommandRunner` mock for `RunPhase` returns canned stdout bytes and a configurable exit code. Example:

```go
type mockRunner struct {
    phaseOutputs map[string][]byte  // phase name → canned output
    phaseErrors  map[string]error   // phase name → error to return
    gateExitCode int
}

func (m *mockRunner) RunPhase(ctx context.Context, dir string, stdin io.Reader, name string, args ...string) ([]byte, error) {
    // Read stdin to verify prompt was rendered correctly
    prompt, _ := io.ReadAll(stdin)
    // Return canned output based on args or prompt content
    // ...
}
```

---

## Full System User Stories

These describe the complete target state of xylem v2. Development phases below break these into incremental deliverables.

### US-01: Phase-Based Workflow Execution

**As** a developer using xylem to process GitHub issues,
**I want** each workflow to execute as a series of discrete phases with separate `claude -p` invocations,
**so that** I can validate intermediate outputs, gate on mechanical checks, and get deterministic, observable execution.

**Acceptance Criteria:**

1. A workflow definition (YAML file in the repo) specifies an ordered list of phases.
2. Each phase has: `name` (string), `prompt_file` (path to a markdown prompt template), `max_turns` (integer), and an optional `gate` configuration.
3. The runner executes phases sequentially. Each phase is a separate `claude -p` invocation in the vessel's worktree.
4. Each phase invocation receives the phase prompt (rendered from the template), not the prompts of other phases.
5. The runner captures the stdout of each phase invocation and writes it to `.xylem/phases/<vessel-id>/<phase-name>.output` in the worktree.
6. If a phase's `claude -p` exits with a non-zero exit code, the vessel is marked `failed` and the runner stops processing subsequent phases. The exit code and stderr are recorded in the vessel state.
7. The runner logs phase transitions (phase started, phase completed, phase failed) to stdout with timestamps.
8. Vessels created via `xylem enqueue --prompt` (no workflow) bypass phase-based execution entirely. The runner executes a single `claude -p` invocation with the provided prompt, using the global `max_turns` and `claude.flags` from config. No phase output files are created. The vessel transitions directly to `completed` or `failed`.

### US-02: Mechanical Gates Between Phases

**As** a developer,
**I want** to run shell commands between phases that must pass before the next phase starts,
**so that** I get deterministic, non-LLM verification (build, test, lint) at each checkpoint.

**Acceptance Criteria:**

1. A phase's `gate` field supports a `command` type with fields: `run` (shell command string), `retries` (integer, default 0), and `retry_delay` (duration string, default "10s").
2. After a phase's `claude -p` invocation completes successfully (exit 0), the runner executes the gate command in the vessel's worktree using the system shell (`exec.Command("sh", "-c", gate.Run)` with `cmd.Dir` set to the worktree path). This allows pipes, redirects, and compound commands in the `run` field (e.g., `"go build ./... && go test ./..."`).
3. The gate command's combined stdout+stderr is captured via `CombinedOutput()`. If the gate exits 0, the gate passes and the runner proceeds to the next phase.
4. If the gate command exits non-zero and `retries > 0`, the runner re-invokes the *same phase* with the gate's stderr/stdout appended to the prompt as context (prefixed with `"The following gate check failed after the previous phase. Fix the issues and try again:\n\n"`). The retry count is decremented.
5. If the gate command exits non-zero and retries are exhausted, the vessel is marked `failed`. The gate command's stdout and stderr are recorded in the vessel state.
6. Gate commands inherit the vessel worktree's working directory and environment.

### US-03: Label-Based Gate (Human Approval)

**As** a developer,
**I want** certain phases to pause and wait for a human-applied GitHub label before proceeding,
**so that** I can review an agent's plan before it starts implementing.

**Acceptance Criteria:**

1. A phase's `gate` field supports a `label` type with fields: `wait_for` (label string, e.g., `"plan-approved"`), `timeout` (duration string, default "24h"), and `poll_interval` (duration string, default "60s").
2. After the gated phase completes, the runner posts the phase output as a comment on the GitHub issue (via `gh issue comment`).
3. The runner then polls the issue's labels at `poll_interval` until `wait_for` is found or `timeout` expires.
4. If the label is found, the gate passes and the runner proceeds to the next phase.
5. If the timeout expires, the vessel is marked `timed_out` (a new terminal state). The runner posts a comment on the issue indicating the timeout.
6. While waiting, the vessel's state is `waiting` (a new non-terminal state). `xylem status` shows waiting vessels with the label they're waiting for and elapsed time.
7. A waiting vessel does NOT count against the concurrency limit — it is not occupying a Claude session.

### US-04: Prompt Templating With Phase Context

**As** a developer writing workflow prompts,
**I want** my prompt templates to have access to the issue context, previous phase outputs, and gate results,
**so that** each phase has the information it needs without me manually assembling context.

**Acceptance Criteria:**

1. Prompt template files are Go templates (`.md` files with `{{ }}` syntax).
2. The following variables are available in every phase template:

| Variable | Type | Description |
|----------|------|-------------|
| `{{.Issue.URL}}` | string | GitHub issue URL |
| `{{.Issue.Title}}` | string | Issue title |
| `{{.Issue.Body}}` | string | Issue body (markdown) |
| `{{.Issue.Labels}}` | []string | Issue labels |
| `{{.Issue.Number}}` | int | Issue number |
| `{{.Phase.Name}}` | string | Current phase name |
| `{{.Phase.Index}}` | int | Current phase index (0-based) |
| `{{.PreviousOutputs}}` | map[string]string | Map of phase name → output text for all completed phases |
| `{{.GateResult}}` | string | Stdout/stderr from the most recent gate command (empty if no gate or first phase) |
| `{{.Vessel.ID}}` | string | Vessel identifier |
| `{{.Vessel.Source}}` | string | Source identifier (e.g., "github") |

3. If a template fails to render (missing variable, syntax error), the vessel is marked `failed` with the template error in the vessel state. The runner does not invoke `claude -p`.

### US-05: System Prompt Injection From Repo Harness File

**As** a developer maintaining a repository,
**I want** xylem to automatically inject a repo-level harness file into every Claude invocation,
**so that** Claude has access to my project's architecture decisions, golden principles, and build conventions.

**Acceptance Criteria:**

1. If a file named `.xylem/HARNESS.md` exists in the repository root (not the worktree — the main repo), the runner passes it via `--append-system-prompt` on every `claude -p` invocation for vessels from that repo.
2. If the file does not exist, the runner proceeds without `--append-system-prompt`. No error is raised.
3. The harness file is read once at the start of each `drain` run (not re-read per phase). If the file changes during a drain, the old content is used for in-progress vessels.
4. The `--append-system-prompt` flag is used (not `--system-prompt-file`) so that Claude Code's built-in capabilities are preserved.

### US-06: Progress Reporting to GitHub Issues

**As** a developer monitoring xylem's work,
**I want** xylem to post progress comments on GitHub issues as phases complete,
**so that** I can see what the agent did at each step without checking logs.

**Acceptance Criteria:**

1. When a phase completes successfully, the runner posts a comment on the associated GitHub issue (if the vessel originated from a `github` source).
2. The comment format is:
   ```
   **xylem — phase `<phase-name>` completed** (<duration>)

   <details>
   <summary>Phase output (click to expand)</summary>

   <phase output, truncated to 64000 characters if longer>

   </details>
   ```
3. When all phases complete and the vessel transitions to `completed`, the runner posts a final summary comment listing all phases and their durations.
4. When a vessel fails, the runner posts a comment with the failure details (phase name, exit code, gate output if applicable).
5. Comments are posted via `gh issue comment <number> --body <text>` in the vessel's worktree (where `gh` is available and authenticated).
6. If the `gh` command fails (e.g., auth issue, network), the runner logs a warning but does not mark the vessel as failed. Phase execution continues.

### US-07: Failure Context on Retry

**As** a developer,
**I want** to be able to retry a failed vessel with the failure context included,
**so that** the agent's next attempt benefits from knowing what went wrong.

**Acceptance Criteria:**

1. `xylem retry <vessel-id>` creates a new vessel with the same configuration as the failed one.
2. The new vessel's first phase prompt is augmented with the failure context from the original vessel: which phase failed, the exit code, the gate output (if gate failure), and the phase output.
3. The failure context is injected as a prefix to the first phase's rendered prompt, formatted as:
   ```
   ## Previous Attempt Failed

   This task was previously attempted and failed at phase `<name>`.

   **Exit code:** <code>
   **Phase output:**
   <output, truncated to 8000 characters>

   **Gate output (if applicable):**
   <gate output, truncated to 4000 characters>

   ---

   ```
4. The new vessel gets a new ID with a `-retry-<n>` suffix (e.g., `issue-42-retry-1`).
5. If the original vessel's ID cannot be found, `retry` exits with an error message.

### US-08: Daemon Mode

**As** a developer,
**I want** to run xylem as a single long-running process that continuously scans and drains,
**so that** I don't have to manage crontab entries.

**Acceptance Criteria:**

1. `xylem daemon` starts a long-running process that alternates between scan and drain on a configurable interval.
2. The configuration adds a `daemon` section:
   ```yaml
   daemon:
     scan_interval: "60s"    # how often to scan sources
     drain_interval: "30s"   # how often to drain pending vessels
   ```
3. The daemon runs `scan` logic, then `drain` logic, then sleeps for the shorter of the two intervals, in a loop.
4. SIGINT/SIGTERM causes a graceful shutdown: running sessions finish, no new vessels are dispatched, the process exits.
5. `xylem pause` / `xylem resume` work as before — pausing stops scanning but running sessions continue.
6. The daemon logs each tick with a summary: vessels scanned, vessels dispatched, vessels running, vessels waiting.
7. The existing `xylem scan` and `xylem drain` commands continue to work as standalone one-shot commands. Daemon mode does not replace them.

### US-09: Harness Bootstrap

**As** a developer setting up xylem in a new repository,
**I want** a command that scaffolds the harness file and workflow definitions,
**so that** I don't have to create the directory structure and boilerplate from scratch.

**Acceptance Criteria:**

1. `xylem init` creates the following in the current directory:
   - `.xylem/HARNESS.md` — a starter harness file with sections for: project description, architecture overview, build/test/lint commands, golden principles (empty, with comments explaining what to add).
   - `.xylem/workflows/fix-bug.yaml` — a default multi-phase bug fix workflow definition.
   - `.xylem/workflows/implement-feature.yaml` — a default multi-phase feature implementation workflow definition.
   - `.xylem/prompts/` — directory containing default phase prompt templates for each workflow.
   - `.xylem.yml` — a starter config file with the source and claude sections.
2. If any of these files already exist, `xylem init` skips them and logs a message. It does not overwrite.
3. `xylem init --force` overwrites existing files with fresh defaults.

### US-10: Workflow Definition Schema

**As** a developer defining workflows,
**I want** workflow definitions to be validated at load time,
**so that** I catch configuration errors before Claude sessions are launched.

**Acceptance Criteria:**

1. Workflow definitions are YAML files located in `.xylem/workflows/<name>.yaml`.
2. The schema is:
   ```yaml
   name: fix-bug                          # required, must match filename
   description: "Diagnose and fix a bug"  # optional, for display
   phases:
     - name: analyze                      # required, unique within workflow
       prompt_file: .xylem/prompts/fix-bug/analyze.md  # required, relative to repo root
       max_turns: 5                       # required, positive integer
       gate:                              # optional
         type: command                    # "command" or "label"
         run: "make test"                 # required if type=command
         retries: 2                       # optional, default 0
         retry_delay: "10s"              # optional, default "10s"
       allowed_tools: "Read,Edit,Bash"    # optional, passed to --allowedTools

     - name: plan
       prompt_file: .xylem/prompts/fix-bug/plan.md
       max_turns: 3
       gate:
         type: label
         wait_for: "plan-approved"
         timeout: "24h"
         poll_interval: "60s"

     - name: implement
       prompt_file: .xylem/prompts/fix-bug/implement.md
       max_turns: 15
       gate:
         type: command
         run: "make test"
         retries: 2

     - name: pr
       prompt_file: .xylem/prompts/fix-bug/pr.md
       max_turns: 3
   ```
3. When a workflow is referenced in `.xylem.yml`, xylem validates the workflow file at load time (during `scan`, `drain`, or `daemon` startup).
4. Validation errors are reported with the file path, field name, and reason. Xylem exits with a non-zero code on validation failure.
5. The following validations are performed:
   - All required fields are present
   - `prompt_file` points to an existing file
   - `max_turns` is a positive integer
   - Phase names are unique within a workflow
   - Gate `type` is one of `command` or `label`
   - Duration strings parse as valid Go durations
   - `allowed_tools` is a non-empty string if provided

### US-11: Updated Config Schema

**As** a developer configuring xylem,
**I want** the `.xylem.yml` config to reference workflow definitions by name instead of inlining them,
**so that** workflows are reusable across sources and the config stays concise.

**Acceptance Criteria:**

1. The `.xylem.yml` task config changes from referencing a single workflow name to referencing a workflow definition file:
   ```yaml
   sources:
     bugs:
       type: github
       repo: owner/name
       exclude: [wontfix, duplicate, in-progress, no-bot]
       tasks:
         fix-bugs:
           labels: [bug, ready-for-work]
           workflow: fix-bug                # resolves to .xylem/workflows/fix-bug.yaml
     features:
       type: github
       repo: owner/name
       exclude: [wontfix, duplicate, in-progress, no-bot]
       tasks:
         implement-features:
           labels: [enhancement, low-effort, ready-for-work]
           workflow: implement-feature      # resolves to .xylem/workflows/implement-feature.yaml

   concurrency: 2
   max_turns: 50           # fallback if a phase doesn't specify max_turns
   timeout: "30m"          # per-vessel timeout (all phases combined)
   state_dir: ".xylem"

   claude:
     command: "claude"
     flags: "--bare --dangerously-skip-permissions"  # base flags for all invocations
     env:
       ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"     # explicit, not inherited from shell
   ```
2. The `claude.flags` field specifies base flags applied to every `claude -p` invocation. If `--bare` is included, `claude.env.ANTHROPIC_API_KEY` must be set (validated at load time).
3. The v1 `claude.template` field is no longer recognized. If present, xylem exits with code 1 and prints `error: claude.template is no longer supported; migrate to phase-based workflows in .xylem/workflows/`.

---

## Development Phase Breakdown

### Phase 1: Multi-Phase Runner

**Goal:** Replace the single `claude -p` invocation per vessel with sequential phase execution and command gates.

**Scope:** US-01, US-02, US-04, US-10 (workflow schema), US-11 (config changes). Does not include label gates, GitHub comments, retry, daemon mode, or harness injection.

#### Phase 1 User Stories

**P1-US-01: Workflow file loading and validation**

**As** a developer running `xylem drain`,
**I want** xylem to load and validate workflow definitions from `.xylem/workflows/`,
**so that** I know my workflow configuration is correct before any Claude sessions launch.

**Acceptance Criteria:**

1. Given a `.xylem.yml` with `workflow: fix-bug` in a task definition, and a file `.xylem/workflows/fix-bug.yaml` exists with valid schema — `xylem drain` starts without error.
2. Given a `.xylem.yml` referencing `workflow: nonexistent` — `xylem drain` exits with code 1 and prints `error: workflow file not found: .xylem/workflows/nonexistent.yaml`.
3. Given a workflow file missing the `phases` key — `xylem drain` exits with code 1 and prints `error: .xylem/workflows/fix-bug.yaml: "phases" is required`.
4. Given a workflow file where a phase's `prompt_file` points to a non-existent file — `xylem drain` exits with code 1 and prints `error: .xylem/workflows/fix-bug.yaml: phase "analyze": prompt_file not found: .xylem/prompts/fix-bug/analyze.md`.
5. Given a workflow file with duplicate phase names — `xylem drain` exits with code 1 and prints `error: .xylem/workflows/fix-bug.yaml: duplicate phase name "implement"`.

**Shakedown test:**
```bash
# Setup: create a minimal .xylem.yml referencing "test-workflow"
# Create .xylem/workflows/test-workflow.yaml with valid phases
xylem drain --dry-run
# Expected: exits 0, shows "0 pending vessels"

# Break the workflow file (remove phases key)
xylem drain --dry-run
# Expected: exits 1, error message about missing phases
```

---

**P1-US-02: Sequential phase execution**

**As** a developer draining the queue,
**I want** each vessel to execute its workflow's phases in order, each as a separate `claude -p` call,
**so that** I get a fresh context window per phase and can observe each step.

**Acceptance Criteria:**

1. Given a queued vessel with workflow `test-workflow` having phases [analyze, implement], and `claude -p` returns exit 0 for both — the vessel transitions to `completed` and both phase output files exist at `.xylem/phases/<vessel-id>/analyze.output` and `.xylem/phases/<vessel-id>/implement.output`.
2. Given a workflow with 3 phases and `claude -p` returns exit 1 on the 2nd phase — the vessel transitions to `failed`, the 1st phase output file exists, and the 3rd phase was never invoked (no output file).
3. The runner logs each phase start and completion:
   ```
   [2026-03-28T14:00:00Z] vessel issue-42: starting phase "analyze" (1/3)
   [2026-03-28T14:02:15Z] vessel issue-42: phase "analyze" completed (2m15s)
   [2026-03-28T14:02:16Z] vessel issue-42: starting phase "implement" (2/3)
   ```
4. Each `claude -p` invocation is constructed as:
   ```
   claude -p --max-turns <phase.max_turns> <claude.flags> [--allowedTools <phase.allowed_tools>] [--append-system-prompt <harness>]
   ```
   The rendered prompt is piped via stdin (not passed as a CLI argument — see Prompt Delivery section). The command runs in the vessel's worktree directory via `RunPhase`. Stdout is captured to the phase output file while also being teed to the terminal.
5. The rendered prompt is the phase's prompt template file, processed as a Go template with the variables defined in US-04.

**Shakedown test:**
```bash
# Setup: enqueue a vessel manually
xylem enqueue --workflow test-workflow --ref "https://github.com/owner/repo/issues/1"

# Create a test-workflow with 2 phases: analyze and implement
# Phase prompts: "echo the word ANALYZE" and "echo the word IMPLEMENT"

xylem drain
# Expected: vessel completes, both output files exist
# Verify: cat .xylem/phases/<id>/analyze.output contains ANALYZE-related output
# Verify: cat .xylem/phases/<id>/implement.output contains IMPLEMENT-related output

# Check logs show both phases starting and completing
```

---

**P1-US-03: Command gate execution**

**As** a developer,
**I want** the runner to execute a shell command after a phase and fail the vessel if it fails,
**so that** I get mechanical verification (build, test) between phases.

**Acceptance Criteria:**

1. Given a phase with `gate: { type: command, run: "true" }` — after the phase completes, the gate runs, exits 0, and the next phase starts.
2. Given a phase with `gate: { type: command, run: "false", retries: 0 }` — after the phase completes, the gate runs, exits 1, and the vessel is marked `failed`.
3. Given a phase with `gate: { type: command, run: "false", retries: 2 }` — the gate fails, the runner re-invokes the same phase with gate failure context appended to the prompt, the gate runs again after the re-invocation. This repeats up to 2 times. If still failing, the vessel is marked `failed`.
4. The gate command runs in the vessel's worktree directory.
5. Gate stdout and stderr are captured. On failure, they are included in the re-invocation prompt and in the vessel's failure state.
6. The runner logs gate results:
   ```
   [2026-03-28T14:02:16Z] vessel issue-42: running gate for phase "implement": make test
   [2026-03-28T14:02:45Z] vessel issue-42: gate failed (exit 1), retrying phase "implement" (1/2 retries remaining)
   ```

**Shakedown test:**
```bash
# Setup: workflow with phase "implement" gated by "exit 1" with retries: 1
xylem enqueue --workflow gated-workflow --ref "#1"
xylem drain

# Expected: vessel fails after 1 retry
# Verify: logs show gate failure, retry, second gate failure, vessel failed
# Verify: vessel state includes gate stderr

# Repeat with gate "exit 0" — vessel should complete
```

---

**P1-US-04: Prompt template rendering with previous outputs**

**As** a developer writing prompts,
**I want** later phases to have access to the outputs of earlier phases,
**so that** I can write prompts like "Based on the analysis above, implement the fix."

**Acceptance Criteria:**

1. Given a 2-phase workflow where phase 1 produces output "The bug is in auth.go line 42", and phase 2's template contains `{{.PreviousOutputs.analyze}}` — the rendered prompt for phase 2 includes the text "The bug is in auth.go line 42".
2. Given a template referencing `{{.PreviousOutputs.nonexistent}}` — the rendered output contains an empty string (Go template `map` behavior with `index`). No error is raised.
3. Given a template with a syntax error like `{{.BadSyntax` — the vessel is marked `failed` with the template parse error. No `claude -p` is invoked.
4. Given a phase that is the first phase — `{{.PreviousOutputs}}` is an empty map. Templates must not assume any keys exist.

**Shakedown test:**
```bash
# Setup: 2-phase workflow
# Phase 1 prompt: "List all files in the current directory"
# Phase 2 prompt template:
#   "The previous phase found these files:
#    {{.PreviousOutputs.phase1}}
#    Now count them."

xylem enqueue --workflow template-test --ref "#1"
xylem drain

# Verify: phase 2 output references the file listing from phase 1
```

---

**P1-US-05: Config schema with workflow references**

**As** a developer configuring xylem,
**I want** `.xylem.yml` to reference workflow definitions by name,
**so that** the runner knows which workflow YAML to load for each task.

**Acceptance Criteria:**

1. Given a `.xylem.yml` with `workflow: fix-bug` in a task definition — xylem resolves this to `.xylem/workflows/fix-bug.yaml` and loads it.
2. Given a `.xylem.yml` with `claude.flags` containing `--bare` and `claude.env.ANTHROPIC_API_KEY` set — xylem starts without error and passes both to every `claude -p` invocation.
3. Given a `.xylem.yml` with `claude.flags` containing `--bare` but no `claude.env.ANTHROPIC_API_KEY` — xylem exits with code 1 and prints `error: --bare requires ANTHROPIC_API_KEY in claude.env`.
4. Given a `.xylem.yml` containing the v1 `claude.template` field — xylem exits with code 1 and prints `error: claude.template is no longer supported; migrate to phase-based workflows in .xylem/workflows/`.

**Shakedown test:**
```bash
# Test valid config
# .xylem.yml with claude.flags and claude.env, .xylem/workflows/ populated
xylem drain --dry-run
# Expected: exits 0

# Test --bare without API key
# Remove claude.env.ANTHROPIC_API_KEY, keep --bare in flags
xylem drain --dry-run
# Expected: exits 1, error about missing ANTHROPIC_API_KEY

# Test rejected v1 field
# Add claude.template to .xylem.yml
xylem drain --dry-run
# Expected: exits 1, error about claude.template no longer supported
```

---

### Phase 2: Harness Contract & Issue Reporting

**Goal:** Add system prompt injection from a repo harness file, and post phase progress as GitHub issue comments.

**Scope:** US-05 (harness file injection), US-06 (progress comments). Builds on Phase 1.

#### Phase 2 User Stories

**P2-US-01: Harness file injection**

**As** a developer with a `.xylem/HARNESS.md` in my repo,
**I want** its contents injected into every Claude invocation as an appended system prompt,
**so that** Claude knows my project's architecture, conventions, and build commands.

**Acceptance Criteria:**

1. Given a repo with `.xylem/HARNESS.md` containing "Always use gofmt" — every `claude -p` invocation for vessels from that repo includes `--append-system-prompt` with the file's contents.
2. Given a repo without `.xylem/HARNESS.md` — `claude -p` invocations do not include `--append-system-prompt`. No error or warning.
3. The harness file is read from the repo root (the parent of the worktree), not from the worktree itself.
4. The runner logs: `loaded harness file: .xylem/HARNESS.md (1234 bytes)` on startup.

**Shakedown test:**
```bash
# Create .xylem/HARNESS.md with a unique marker string
echo "HARNESS_MARKER_12345" > .xylem/HARNESS.md

# Enqueue and drain a vessel with a workflow whose prompt is:
# "Print the contents of your system prompt"
xylem enqueue --workflow harness-test --ref "#1"
xylem drain

# Verify: phase output contains "HARNESS_MARKER_12345"
# (Claude will surface system prompt content when asked directly)

# Remove HARNESS.md, repeat — marker should NOT appear
```

---

**P2-US-02: Phase completion comments on GitHub issues**

**As** a developer monitoring agent work,
**I want** xylem to post a comment on the GitHub issue when each phase completes,
**so that** I can track progress without reading server logs.

**Acceptance Criteria:**

1. Given a vessel from a `github` source that completes phase "analyze" — a comment is posted on the issue containing the phase name, duration, and output (in a collapsed `<details>` block).
2. Given a vessel from a `manual` source (no associated issue) — no comment is posted. No error.
3. Given `gh issue comment` fails (network error, auth issue) — the runner logs a warning but does not fail the vessel.
4. Given a phase output exceeding 64000 characters — the comment truncates the output with a note: `(output truncated — full output in .xylem/phases/<id>/<phase>.output)`.
5. Given a vessel that fails — a comment is posted with the failure details (phase name, exit code, gate output).

**Shakedown test:**
```bash
# Create a real GitHub issue with label "ready-for-work"
# Run scan + drain
xylem scan && xylem drain

# Verify: GitHub issue has comments for each phase
# Verify: final comment shows completion summary with durations
```

---

### Phase 3: Label Gates & Retry

**Goal:** Add human-in-the-loop approval gates and retry with failure context.

**Scope:** US-03 (label gates), US-07 (retry). Builds on Phase 2.

#### Phase 3 User Stories

**P3-US-01: Label gate — pause and wait for human approval**

**As** a developer,
**I want** a phase to post its output to the issue and wait for a label before continuing,
**so that** I can review the agent's plan before it implements anything.

**Acceptance Criteria:**

1. Given a phase with `gate: { type: label, wait_for: "plan-approved" }` — after the phase completes, the runner posts the output as an issue comment (per US-06), then transitions the vessel to `waiting` state.
2. `xylem status` shows the vessel as `waiting` with the expected label and elapsed time.
3. In one-shot `drain` mode: the vessel transitions to `waiting`, the drain goroutine for this vessel exits (releasing its concurrency slot), and `drain` returns normally. On the next `drain` invocation, the runner checks all `waiting` vessels before dequeuing `pending` ones by polling `gh issue view --json labels`. If the label is found, the vessel transitions to `running` and resumes from the next phase. If the timeout has expired, the vessel transitions to `timed_out`.
4. In `daemon` mode: waiting vessels are checked on every tick automatically.
5. When `timeout` expires (default 24h) — the vessel transitions to `timed_out`, and a comment is posted: `"xylem — timed out waiting for label \`plan-approved\` on phase \`plan\` after 24h"`.
6. A `waiting` vessel does NOT count against the concurrency limit.
7. `xylem cancel <vessel-id>` cancels a waiting vessel (transitions to `cancelled`).

**Shakedown test:**
```bash
# Setup: workflow with plan phase gated by label "plan-approved"
xylem enqueue --workflow label-gate-test --ref "https://github.com/owner/repo/issues/5"
xylem drain
# Expected: drain returns (does not block). Vessel is in "waiting" state.

xylem status
# Expected: vessel shows "waiting" for "plan-approved"

# Run drain again WITHOUT adding label
xylem drain
# Expected: vessel still "waiting" (label not found, timeout not expired)

# Add label via gh:
gh issue edit 5 --repo owner/repo --add-label "plan-approved"

# Run drain again
xylem drain
# Expected: vessel resumes, proceeds to next phase

# Test timeout: repeat without adding label, set timeout to "30s" in workflow
xylem enqueue --workflow label-gate-test-short-timeout --ref "https://github.com/owner/repo/issues/6"
xylem drain
# Expected: vessel enters "waiting"
sleep 35
xylem drain
# Expected: vessel transitions to "timed_out", timeout comment posted on issue
```

---

**P3-US-02: Retry failed vessels with failure context**

**As** a developer,
**I want** to retry a failed vessel with information about why it failed,
**so that** the agent's next attempt can avoid the same mistake.

**Acceptance Criteria:**

1. `xylem retry issue-42` creates a new vessel `issue-42-retry-1` with the same workflow and ref.
2. The retry vessel's first phase prompt is prepended with the failure context from the original (as specified in US-07).
3. `xylem retry issue-42` when `issue-42` is not in `failed` state — exits with code 1 and prints `error: vessel issue-42 is not in failed state (current: completed)`.
4. `xylem retry nonexistent` — exits with code 1 and prints `error: vessel not found: nonexistent`.
5. Multiple retries increment the suffix: `issue-42-retry-1`, `issue-42-retry-2`, etc.

**Shakedown test:**
```bash
# Setup: workflow with a gate that always fails (exit 1, retries: 0)
xylem enqueue --workflow always-fail --ref "#1" --id "test-fail"
xylem drain
# Expected: vessel "test-fail" in failed state

xylem retry test-fail
# Expected: new vessel "test-fail-retry-1" in pending state

xylem status
# Expected: shows both vessels

# Fix the gate (change to exit 0), drain again
xylem drain
# Expected: "test-fail-retry-1" completes
```

---

### Phase 4: Daemon Mode & Init

**Goal:** Add the long-running daemon and repo bootstrap command.

**Scope:** US-08 (daemon), US-09 (init). Builds on Phase 3.

#### Phase 4 User Stories

**P4-US-01: Daemon mode**

**As** a developer,
**I want** to run `xylem daemon` and have it continuously scan and drain,
**so that** new issues are picked up automatically without crontab management.

**Acceptance Criteria:**

1. `xylem daemon` starts a loop: scan → drain → sleep → repeat.
2. The scan/drain intervals are configured via `.xylem.yml`:
   ```yaml
   daemon:
     scan_interval: "60s"
     drain_interval: "30s"
   ```
3. Each tick logs a summary: `[2026-03-28T14:00:00Z] tick: scanned=3 dispatched=1 running=2 waiting=0 completed=5 failed=0`.
4. SIGINT/SIGTERM triggers graceful shutdown: a log message is printed, no new vessels are dispatched, running Claude sessions finish, waiting vessels remain in `waiting` state, the process exits 0.
5. `xylem pause` (from another terminal) pauses scanning. `xylem resume` resumes. This works the same as with cron-based usage.
6. Waiting vessels (label gates) are checked each tick without counting against concurrency.

**Shakedown test:**
```bash
# Start daemon in background
xylem daemon &
DAEMON_PID=$!

# Create a GitHub issue with appropriate labels
gh issue create --title "Test daemon pickup" --label "bug,ready-for-work"

# Wait for 2 scan intervals
sleep 120

# Verify: issue was picked up and processed
xylem status
# Expected: vessel for the issue exists, completed or running

# Graceful shutdown
kill -SIGTERM $DAEMON_PID
wait $DAEMON_PID
# Expected: exits 0
```

---

**P4-US-02: Repository bootstrap**

**As** a developer setting up xylem for the first time,
**I want** `xylem init` to create the directory structure and starter files,
**so that** I have a working starting point.

**Acceptance Criteria:**

1. `xylem init` in an empty directory creates:
   - `.xylem.yml`
   - `.xylem/HARNESS.md`
   - `.xylem/workflows/fix-bug.yaml`
   - `.xylem/workflows/implement-feature.yaml`
   - `.xylem/prompts/fix-bug/analyze.md`
   - `.xylem/prompts/fix-bug/plan.md`
   - `.xylem/prompts/fix-bug/implement.md`
   - `.xylem/prompts/fix-bug/pr.md`
   - `.xylem/prompts/implement-feature/analyze.md`
   - `.xylem/prompts/implement-feature/plan.md`
   - `.xylem/prompts/implement-feature/implement.md`
   - `.xylem/prompts/implement-feature/pr.md`
2. Each file contains useful starter content (not empty):
   - **`.xylem.yml`**: sources section with a `github` source template (repo placeholder `owner/name`), tasks for `fix-bugs` and `implement-features`, concurrency 2, max_turns 50, timeout 30m, claude section with `command: "claude"`, `flags: "--bare --dangerously-skip-permissions"`, and `env: { ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}" }`.
   - **`.xylem/HARNESS.md`**: sections with markdown headers and HTML comments explaining what to fill in: `# Project Overview` (describe what this project does), `# Architecture` (describe the codebase structure), `# Build & Test` (list the exact build, test, and lint commands), `# Golden Principles` (list rules the agent must always follow — e.g., "always run gofmt", "never modify generated files"), `# Dependencies` (note any external services or tools the agent needs).
   - **`.xylem/workflows/fix-bug.yaml`**: 4 phases (analyze, plan, implement, pr) with `max_turns` of 5, 3, 15, 3 respectively. The `implement` phase has a `gate: { type: command, run: "make test", retries: 2 }`. No label gates in the default.
   - **`.xylem/workflows/implement-feature.yaml`**: same 4 phases. The `plan` phase has a `gate: { type: label, wait_for: "plan-approved", timeout: "24h" }`. The `implement` phase has a `gate: { type: command, run: "make test", retries: 2 }`.
   - **Prompt templates** (`.xylem/prompts/<workflow>/<phase>.md`): each contains a Go template with the phase's task instruction and references to `{{.Issue.URL}}`, `{{.Issue.Title}}`, `{{.Issue.Body}}`, and `{{.PreviousOutputs}}` where appropriate. The `analyze` phase asks Claude to read the issue and the codebase, identify relevant files, and write an analysis. The `plan` phase asks Claude to write an implementation plan based on the analysis. The `implement` phase asks Claude to implement the plan. The `pr` phase asks Claude to commit changes and open a PR via `gh pr create`.
3. `xylem init` when files already exist — skips existing files, prints `skipped: .xylem/HARNESS.md (already exists)` for each.
4. `xylem init --force` overwrites all files.
5. After `xylem init`, running `xylem drain --dry-run` succeeds with no validation errors (all referenced files exist).

**Shakedown test:**
```bash
mkdir /tmp/test-init && cd /tmp/test-init
git init && git commit --allow-empty -m "init"

xylem init

# Verify all files exist
ls .xylem/HARNESS.md
ls .xylem/workflows/fix-bug.yaml
ls .xylem/prompts/fix-bug/analyze.md

# Verify config is valid
xylem drain --dry-run
# Expected: exits 0, no validation errors

# Verify idempotency
xylem init
# Expected: all files skipped

# Verify force
xylem init --force
# Expected: all files overwritten
```

---

## Cross-Phase Regression Tests

These tests should pass after every phase is implemented. Run them as part of each phase's shakedown.

### Regression 1: Manual enqueue still works

```bash
xylem enqueue --prompt "List files in current directory"
xylem drain
# Expected: vessel completes
# Expected: no GitHub comments posted (manual source)
```

### Regression 2: Concurrency limit respected

```bash
# Set concurrency: 1 in config
# Enqueue 3 vessels
xylem drain
# Expected: vessels process one at a time, not in parallel
# Verify via timestamps in logs
```

### Regression 3: Graceful shutdown

```bash
# Start drain with 2 vessels queued (concurrency: 1)
xylem drain &
PID=$!
sleep 5
kill -SIGTERM $PID
wait $PID
# Expected: first vessel finishes, second remains pending
# Expected: exit code 0
```

### Regression 4: Dry run never invokes Claude

```bash
xylem scan --dry-run
xylem drain --dry-run
# Expected: no claude processes spawned
# Verify: no new phase output files created
```

### Regression 5: v1 config is rejected

```bash
# Add claude.template to .xylem.yml
xylem drain --dry-run
# Expected: exits 1
# Expected: error message: "claude.template is no longer supported"
```

---

## Out of Scope

The following are explicitly not part of this spec. Documenting them here prevents scope creep.

- **Webhook-based triggers** — xylem remains poll-based (via cron or daemon). Webhooks add infrastructure complexity (HTTP server, ngrok/tunnels, GitHub webhook config) without proportional value for a solo developer workflow.
- **Priority queues** — FIFO is sufficient for the target use case. Priority adds queue complexity and decision-making overhead.
- **Multi-agent coordination within a vessel** — each vessel is a single-agent, single-worktree execution. Coordination between agents is out of scope.
- **Non-GitHub sources** (Jira, Linear, etc.) — the source interface is pluggable, but only `github` and `manual` are in scope for this spec.
- **Web dashboard** — `xylem status` and GitHub issue comments provide sufficient observability for the target user.
- **Claude Code `--worktree` flag usage** — xylem manages its own worktrees via `git worktree add`. Claude Code's `--worktree` flag creates worktrees at `.claude/worktrees/` with its own lifecycle management, which would conflict with xylem's control over the worktree path and cleanup.
