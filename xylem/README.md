# xylem

Generic multi-source session scheduler — scans pluggable sources, queues tasks, and launches Claude Code sessions in isolated git worktrees.

## Overview

xylem is a two-layer system:

- **Go CLI** (`xylem`) — control plane: scans configured sources for actionable tasks, manages a persistent work queue, and launches Claude Code sessions in isolated git worktrees
- **Skills** — execution plane: `fix-bug` and `implement-feature` skills run inside each Claude session to do the actual work

Sources are pluggable. The built-in `github` source scans GitHub issues by label. The `manual` source backs the `enqueue` command for ad-hoc tasks. You can configure multiple sources in a single config — xylem handles scheduling, deduplication, concurrency, and worktree isolation across all of them.

## Prerequisites

- **Go 1.22+** — to build the CLI
- **git** — must be on PATH
- **[claude](https://docs.anthropic.com/en/docs/claude-code)** — Claude Code CLI
- **[gh](https://cli.github.com/)** — GitHub CLI, authenticated (`gh auth login`). Only required when a `github` source is configured.
- **[refine-issue](https://github.com/nicholls-inc/claude-code-marketplace)** skill — external dependency for the `refine-issue` task type; install separately via `claude skill install`

## Installation

```bash
# Add the marketplace
claude plugin marketplace add nicholls-inc/claude-code-marketplace

# Install the plugin
claude plugin install xylem@nicholls

# Install the Go CLI
go install github.com/nicholls-inc/claude-code-marketplace/xylem/cli/cmd/xylem@latest
```

## Configuration

Create `.xylem.yml` in your target repository:

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
  template: "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}"
```

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

claude:
  command: "claude"
  template: "{{.Command}} -p \"/{{.Skill}} {{.Ref}}\" --max-turns {{.MaxTurns}}"
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
| `state_dir` | `".xylem"` | Directory for queue and state files |
| `claude.command` | `"claude"` | Claude CLI binary name |
| `claude.template` | see above | Go template for the claude invocation |

### Template variables

The `claude.template` Go template has access to:

| Variable | Description |
|----------|-------------|
| `{{.Command}}` | Claude CLI binary |
| `{{.Skill}}` | Skill name from the matched task |
| `{{.Ref}}` | Task reference (URL, ticket ID, etc.) |
| `{{.Prompt}}` | Direct prompt (for `enqueue --prompt`) |
| `{{.MaxTurns}}` | Max turns from config |
| `{{.Meta}}` | Source-specific metadata map |
| `{{.IssueURL}}` | Backward-compat alias for `{{.Ref}}` (GitHub source only) |

## Usage

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

## Cron setup

Run scan and drain on a schedule:

```cron
0 * * * * cd /path/to/repo && xylem scan && xylem drain >> /tmp/xylem.log 2>&1
```

Or use separate schedules:

```cron
*/15 * * * * cd /path/to/repo && xylem scan >> /tmp/xylem-scan.log 2>&1
0,30 * * * * cd /path/to/repo && xylem drain >> /tmp/xylem-drain.log 2>&1
```

## Skills

### fix-bug

Diagnoses and fixes a GitHub issue in 5 phases: Parse → Diagnose → Implement → Validate → PR. Language-agnostic — works with any codebase.

### implement-feature

Implements a low-effort GitHub feature request in 5 phases: Parse → Plan → Implement → Validate → PR. Language-agnostic.

### refine-issue (external dependency)

Refines issue descriptions to make them agent-ready. Install separately — this is not bundled with xylem.

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
          source.OnStart   worktree.Create   claude session
          (side effects)   (git worktree)    (in worktree)
```

The Go CLI is the **control plane** — it handles scheduling, deduplication, concurrency limits, and worktree lifecycle. The Claude skills are the **execution plane** — they run inside each isolated worktree session and do the actual implementation work.

Each source implements the `Source` interface: `Scan()`, `OnStart()`, and `BranchName()`. The GitHub source scans issues by label and names branches `fix/issue-<N>-<slug>` or `feat/issue-<N>-<slug>`. The manual source names branches `task/<id>-<slug>`.

Vessels enqueued via `xylem enqueue --prompt` bypass the template entirely — the prompt is passed directly to Claude.

## Known limitations

- **No auto-retry** — failed vessels stay in the queue as `failed`; re-queue manually
- **No webhooks** — polling only (cron-based)
- **No priority queues** — FIFO order only
- **Cancel does not kill sessions** — only removes pending vessels; running sessions run to completion
- **Sequential correctness only** — no concurrency modeling in the skills themselves
- **GitHub only** — `github` is the only built-in scanning source; other integrations require manual enqueue
