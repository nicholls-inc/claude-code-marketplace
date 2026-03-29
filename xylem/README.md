# xylem

Generic multi-source session scheduler — scans pluggable sources, queues tasks, and launches Claude Code sessions in isolated git worktrees with phase-based execution and quality gates.

## Overview

xylem is a two-layer system:

- **Go CLI** (`xylem`) — control plane: scans configured sources for actionable tasks, manages a persistent work queue, and launches Claude Code sessions in isolated git worktrees
- **Skills** — execution plane: multi-phase skill definitions (e.g. `fix-bug`, `implement-feature`) run inside each Claude session with quality gates between phases

Sources are pluggable. The built-in `github` source scans GitHub issues by label. The `manual` source backs the `enqueue` command for ad-hoc tasks. You can configure multiple sources in a single config — xylem handles scheduling, deduplication, concurrency, and worktree isolation across all of them.

### Agent harness library

The CLI includes an internal library implementing foundational agent harness components. These are building blocks for mission-scoped agent orchestration:

| Package | Purpose |
|---------|---------|
| `bootstrap` | Repo analysis, AGENTS.md generation, docs scaffolding, convention detection |
| `catalog` | Tool catalog with descriptions, parameter types, overlap detection, permission scopes |
| `cost` | Token tracking by agent role and purpose, budget enforcement, model ladders, anomaly detection |
| `ctxmgr` | Named, ordered context processors with compaction, strategy selection, durable/working segment separation |
| `evaluator` | Generator-evaluator loops with signal-gated intensity, configurable iterations, structured reports |
| `gate` | Inter-phase quality gates (command execution, label polling) |
| `intermediary` | Deterministic intent validation with policy rules, audit logging, glob-based permissions |
| `memory` | Mission-scoped typed memory (procedural/semantic/episodic), progress files, handoff artifacts, KV store, scratchpads |
| `mission` | Complexity analysis, task decomposition, sprint contracts, persona/scope constraints, blast radius checks |
| `observability` | OpenTelemetry span attributes for missions, agents, and signals with OTLP gRPC export |
| `orchestrator` | Multi-agent topology (sequential/parallel/orchestrator-workers/handoff), sub-agent context firewalls, failure handling |
| `phase` | Prompt template rendering with issue data, previous outputs, and gate results |
| `reporter` | Phase result collection and output management |
| `signal` | Behavioral heuristics (repetition, tool failure rate, context thrash, efficiency, task stall) with health levels |
| `skill` | Skill definition loading and validation from YAML |

## Prerequisites

- **Go 1.22+** — to build the CLI
- **git** — must be on PATH
- **[claude](https://docs.anthropic.com/en/docs/claude-code)** — Claude Code CLI
- **[gh](https://cli.github.com/)** — GitHub CLI, authenticated (`gh auth login`). Only required when a `github` source is configured.

## Installation

```bash
# Add the marketplace
claude plugin marketplace add nicholls-inc/claude-code-marketplace

# Install the plugin
claude plugin install xylem@nicholls

# Install the Go CLI
go install github.com/nicholls-inc/claude-code-marketplace/xylem/cli/cmd/xylem@latest
```

## Quick start

```bash
# Bootstrap config and state directory
xylem init

# Edit .xylem.yml with your repo and task config
# Edit .xylem/HARNESS.md with your project details

# Preview what would be queued
xylem scan --dry-run

# Scan and process
xylem scan && xylem drain
```

## Configuration

Create `.xylem.yml` in your target repository, or run `xylem init` to generate one:

```yaml
sources:
  bugs:
    type: github
    repo: owner/name
    exclude: [wontfix, duplicate, in-progress, no-bot]
    tasks:
      fix-bugs:
        labels: [bug, ready-for-work]
        skill: fix-bug
  features:
    type: github
    repo: owner/name
    exclude: [wontfix, duplicate, in-progress, no-bot]
    tasks:
      implement-features:
        labels: [enhancement, low-effort, ready-for-work]
        skill: implement-feature

concurrency: 2
max_turns: 50
timeout: "30m"
state_dir: ".xylem"

claude:
  command: "claude"
  flags: "--bare --dangerously-skip-permissions"
  env:
    ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"

daemon:
  scan_interval: "60s"
  drain_interval: "30s"
```

### Configuration reference

| Field | Default | Description |
|-------|---------|-------------|
| `sources` | required | Map of source names to source configs |
| `sources.<name>.type` | required | Source type (`github`) |
| `sources.<name>.repo` | required (github) | GitHub repo in `owner/name` format |
| `sources.<name>.exclude` | `[]` | Labels that prevent an issue from being queued |
| `sources.<name>.tasks` | required | Map of task names to label+skill configs |
| `sources.<name>.tasks.<t>.labels` | required | Labels that trigger this task |
| `sources.<name>.tasks.<t>.skill` | required | Skill name to invoke (e.g. `fix-bug`) |
| `concurrency` | `2` | Max simultaneous Claude sessions |
| `max_turns` | `50` | Max turns per Claude session |
| `timeout` | `"30m"` | Per-session timeout (Go duration string) |
| `state_dir` | `".xylem"` | Directory for queue, state files, skills, and prompts |
| `default_branch` | auto-detected | Branch to create worktrees from |
| `cleanup_after` | `"168h"` | Age threshold for worktree cleanup (7 days) |
| `claude.command` | `"claude"` | Claude CLI binary name |
| `claude.flags` | `""` | Additional CLI flags passed to every session |
| `claude.env` | `{}` | Environment variables for Claude sessions |
| `daemon.scan_interval` | `"60s"` | How often the daemon scans for new work |
| `daemon.drain_interval` | `"30s"` | How often the daemon drains pending vessels |

### Legacy config format

The top-level `repo`/`tasks`/`exclude` format is still supported for backward compatibility. On load, it is automatically normalized into a single `github` source:

```yaml
# Legacy format — still works, auto-migrated at load time
repo: owner/name

tasks:
  fix-bugs:
    labels: [bug, ready-for-work]
    skill: fix-bug

concurrency: 2
max_turns: 50
timeout: "30m"
state_dir: ".xylem"
exclude: [wontfix, duplicate, in-progress, no-bot]
```

Note: `claude.template` is no longer supported. Migrate to phase-based skills in `.xylem/skills/`.

## Skills

Skills are multi-phase execution plans defined in YAML. Each phase runs a prompt template against Claude with a configurable turn limit, and phases can have quality gates that must pass before the next phase begins.

### Skill definition format

```yaml
# .xylem/skills/fix-bug.yaml
name: fix-bug
description: "Diagnose and fix a bug from a GitHub issue"
phases:
  - name: analyze
    prompt_file: .xylem/prompts/fix-bug/analyze.md
    max_turns: 5
  - name: plan
    prompt_file: .xylem/prompts/fix-bug/plan.md
    max_turns: 3
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

### Gate types

| Type | Description | Key fields |
|------|-------------|------------|
| `command` | Runs a shell command; phase retries if it fails | `run`, `retries`, `retry_delay` |
| `label` | Polls a GitHub issue for a label; blocks until found or timeout | `wait_for`, `timeout`, `poll_interval` |

### Prompt templates

Prompt files are Go templates with access to:

| Variable | Description |
|----------|-------------|
| `{{.Issue.Title}}` | Issue title |
| `{{.Issue.URL}}` | Issue URL |
| `{{.Issue.Body}}` | Issue body |
| `{{.PreviousOutputs.<phase>}}` | Output from a previous phase |
| `{{.GateResult}}` | Output from the last gate failure (for retries) |
| `{{.Vessel.ID}}` | Vessel identifier |
| `{{.Vessel.Ref}}` | Task reference |

### Built-in skills

**fix-bug** — Diagnoses and fixes a GitHub issue in 4 phases: Analyze → Plan → Implement → PR. The implement phase gates on `make test`.

**implement-feature** — Implements a feature request in 4 phases: Analyze → Plan → Implement → PR. The plan phase gates on a `plan-approved` label (human-in-the-loop approval).

`xylem init` scaffolds both skills with prompt templates you can customize.

## Usage

### init

Bootstrap the config file, state directory, skill definitions, and prompt templates:

```bash
xylem init
# Created .xylem.yml
# Ensured .xylem/ directory exists
# Created .xylem/HARNESS.md
# Created .xylem/skills/fix-bug.yaml
# Created .xylem/skills/implement-feature.yaml
# Created .xylem/prompts/fix-bug/analyze.md
# ...

xylem init --force   # Overwrite existing files
```

The init command auto-detects your GitHub remote and pre-fills the config.

### scan

Query configured sources for actionable tasks and add them to the queue:

```bash
xylem scan
# Added 3 vessels, skipped 2

xylem scan --dry-run
# Shows candidates without writing to queue
```

### drain

Dequeue pending vessels and launch Claude sessions:

```bash
xylem drain
# Completed 2, failed 0, skipped 1

xylem drain --dry-run
# Shows pending vessels and commands that would run
```

Drain handles SIGINT/SIGTERM gracefully: running sessions finish, pending vessels are not started.

### daemon

Run a continuous scan-drain loop instead of using cron:

```bash
xylem daemon
# daemon started: scan_interval=60s drain_interval=30s
# daemon: scan complete — added=1 skipped=0
# daemon: drain complete — completed=1 failed=0 skipped=0
# daemon: tick summary — pending=0 running=0 completed=1 failed=0
```

Handles SIGINT/SIGTERM for graceful shutdown. Configure intervals in `.xylem.yml`:

```yaml
daemon:
  scan_interval: "2m"
  drain_interval: "30s"
```

### enqueue

Manually enqueue a task without scanning any source:

```bash
# Enqueue using a skill + reference
xylem enqueue --skill fix-bug --ref "https://github.com/owner/repo/issues/99"

# Enqueue with a direct prompt
xylem enqueue --prompt "Refactor the auth middleware to use JWT"

# Enqueue from a prompt file
xylem enqueue --prompt-file task.md --skill implement-feature

# Custom vessel ID and source tag
xylem enqueue --skill fix-bug --ref "#42" --id "hotfix-42" --source "jira"
```

| Flag | Default | Description |
|------|---------|-------------|
| `--skill` | `""` | Skill to invoke (e.g. `fix-bug`) |
| `--ref` | `""` | Task reference (URL, ticket ID, description) |
| `--prompt` | `""` | Direct prompt to pass to Claude |
| `--prompt-file` | `""` | Read prompt from file (mutually exclusive with `--prompt`) |
| `--source` | `"manual"` | Source identifier |
| `--id` | auto-generated | Custom vessel ID |

At least one of `--skill` or `--prompt`/`--prompt-file` is required. When `--prompt` is used, the template is bypassed and the prompt is passed directly to Claude.

### retry

Retry a failed vessel with failure context carried forward:

```bash
xylem retry issue-42
# Created retry vessel issue-42-retry-1 (retrying issue-42)
```

The retry vessel inherits the original's config and includes metadata about the failure (error message, failed phase, gate output) so the next session can learn from it.

### status

Show queue state and vessel summary:

```bash
xylem status
# ID              Source          Skill                 State       Started       Duration
# issue-42        github-issue    fix-bug               completed   10:30 UTC     12m
# issue-55        github-issue    implement-feature     running     10:45 UTC     3m
# task-1710504000 manual          (prompt)              pending     —             —
#
# Summary: 1 pending, 1 running, 1 completed, 0 failed

xylem status --state pending     # filter by state
xylem status --state cancelled   # show cancelled vessels
xylem status --json              # machine-readable JSON array
```

### pause / resume

Pause and resume scanning (does not affect running sessions):

```bash
xylem pause
# Scanning paused. Run `xylem resume` to resume.

xylem resume
# Scanning resumed.
```

### cancel

Cancel a pending vessel by ID:

```bash
xylem cancel issue-42
# Cancelled vessel issue-42
```

Note: cancel only removes pending vessels from the queue. It does not kill running Claude sessions.

### cleanup

Remove stale git worktrees created by xylem:

```bash
xylem cleanup
# Removed .claude/worktrees/fix/issue-42-something
# Removed 1 worktree(s)

xylem cleanup --dry-run
# Shows what would be removed without removing
```

Worktrees older than `cleanup_after` (default: 7 days) are removed.

## Automation

### Daemon mode (recommended)

```bash
# Run as a background service
xylem daemon &

# Or with systemd, launchd, etc.
```

### Cron

```cron
0 * * * * cd /path/to/repo && xylem scan && xylem drain >> /tmp/xylem.log 2>&1
```

Or use separate schedules:

```cron
*/15 * * * * cd /path/to/repo && xylem scan >> /tmp/xylem-scan.log 2>&1
0,30 * * * * cd /path/to/repo && xylem drain >> /tmp/xylem-drain.log 2>&1
```

## Architecture

```
Sources                     xylem scan            Queue
┌─────────────┐             ┌──────────┐          ┌──────────────────────┐
│ github      │──Scan()───→ │ Scanner  │──Enqueue→│ .xylem/queue.jsonl   │
│ (manual)    │             └──────────┘          └──────────┬───────────┘
└─────────────┘                                              │
                            xylem drain                      │ Dequeue
                            ┌──────────┐          ┌──────────▼───────────┐
                            │ Runner   │←─────────│ Pending vessels      │
                            └────┬─────┘          └──────────────────────┘
                                 │
                 ┌───────────────┼───────────────┐
                 ▼               ▼               ▼
          source.OnStart   worktree.Create   Phase execution
          (side effects)   (git worktree)    (skill phases in worktree)
                                                  │
                                          ┌───────┼───────┐
                                          ▼       ▼       ▼
                                       analyze → plan → implement → pr
                                                  │              │
                                              label gate    command gate
                                             (wait for      (run tests,
                                              approval)      retry on fail)
```

The Go CLI is the **control plane** — it handles scheduling, deduplication, concurrency limits, worktree lifecycle, and phase-based execution. The Claude sessions are the **execution plane** — they run inside each isolated worktree and do the actual implementation work, one phase at a time.

Each source implements the `Source` interface: `Scan()`, `OnStart()`, and `BranchName()`. The GitHub source scans issues by label and names branches `fix/issue-<N>-<slug>` or `feat/issue-<N>-<slug>`. The manual source names branches `task/<id>-<slug>`.

Vessels enqueued via `xylem enqueue --prompt` bypass skill phases entirely — the prompt is passed directly to Claude.

## Known limitations

- **Sequential correctness only** — no concurrency modeling in the skills themselves
- **GitHub only** — `github` is the only built-in scanning source; other integrations require manual enqueue
- **Cancel does not kill sessions** — only removes pending vessels; running sessions run to completion
- **No priority queues** — FIFO order only
- **No webhooks** — polling only (cron or daemon mode)
