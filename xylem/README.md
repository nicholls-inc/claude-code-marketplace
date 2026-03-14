# xylem

Autonomous agent scheduling for GitHub issues — scans, queues, and launches Claude Code sessions to fix bugs, implement features, and refine issue descriptions across any repository.

## Overview

xylem is a two-layer system:

- **Go CLI** (`xylem`) — control plane: scans GitHub for actionable issues, manages a persistent work queue, and launches Claude Code sessions in isolated git worktrees
- **Skills** — execution plane: `fix-bug` and `implement-feature` skills run inside each Claude session to do the actual work

You configure which issues to act on via labels. xylem handles the scheduling, deduplication, concurrency, and worktree isolation. Claude handles the implementation.

## Prerequisites

- **Go 1.22+** — to build the CLI
- **[gh](https://cli.github.com/)** — GitHub CLI, authenticated (`gh auth login`)
- **git** — must be on PATH
- **[claude](https://docs.anthropic.com/en/docs/claude-code)** — Claude Code CLI
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
repo: owner/name

tasks:
  fix-bugs:
    labels: [bug, ready-for-work]
    skill: fix-bug
  implement-features:
    labels: [enhancement, low-effort, ready-for-work]
    skill: implement-feature
  refine-issues:
    labels: [needs-refinement]
    skill: refine-issue

concurrency: 2
max_turns: 50
timeout: "30m"
state_dir: ".xylem"
exclude: [wontfix, duplicate, in-progress, no-bot]

claude:
  command: "claude"
  template: "{{.Command}} -p \"/{{.Skill}} {{.IssueURL}}\" --max-turns {{.MaxTurns}}"
```

### Configuration reference

| Field | Default | Description |
|-------|---------|-------------|
| `repo` | required | GitHub repo in `owner/name` format |
| `tasks` | required | Map of task names to label+skill configs |
| `tasks.<name>.labels` | required | GitHub labels that trigger this task |
| `tasks.<name>.skill` | required | Skill name to invoke (e.g. `fix-bug`) |
| `concurrency` | `2` | Max simultaneous Claude sessions |
| `max_turns` | `50` | Max turns per Claude session |
| `timeout` | `"30m"` | Per-session timeout (Go duration string) |
| `state_dir` | `".xylem"` | Directory for queue and state files |
| `exclude` | `[wontfix, duplicate, in-progress, no-bot]` | Labels that prevent an issue from being queued |
| `claude.command` | `"claude"` | Claude CLI binary name |
| `claude.template` | see above | Go template for the claude invocation |

## Usage

### scan

Query GitHub for actionable issues and add them to the queue:

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

### status

Show queue state and vessel summary:

```bash
xylem status
# ID              Issue  Skill                 State       Started       Duration
# issue-42        #42    fix-bug               completed   10:30 UTC     12m
# issue-55        #55    implement-feature     running     10:45 UTC     3m
# issue-78        #78    fix-bug               pending     —             —
#
# Summary: 1 pending, 1 running, 1 completed, 0 failed

xylem status --state pending     # filter by state
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
xylem scan             →  GitHub API (gh search issues)
                       →  Queue (.xylem/queue.jsonl)

xylem drain            →  Queue (dequeue pending)
                       →  git worktree create (.claude/worktrees/<branch>)
                       →  claude -p "/<skill> <issue-url>" (in worktree)
                       →  Queue (update state)
```

The Go CLI is the **control plane** — it handles scheduling, deduplication, concurrency limits, and worktree lifecycle. The Claude skills are the **execution plane** — they run inside each isolated worktree session and do the actual implementation work.

Each vessel runs in its own git worktree on a dedicated branch (`fix/issue-<N>-<slug>` or `feat/issue-<N>-<slug>`), so concurrent sessions never interfere with each other.

## Known limitations

- **No auto-retry** — failed vessels stay in the queue as `failed`; re-queue manually
- **No webhooks** — polling only (cron-based)
- **Single repo per config** — one `.xylem.yml` per repository
- **No priority queues** — FIFO order only
- **Cancel does not kill sessions** — only removes pending vessels; running sessions run to completion
- **Sequential correctness only** — no concurrency modeling in the skills themselves
