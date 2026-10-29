# stall-watchdog automation

**Purpose.** An hourly check on unattended delivery. It measures time since last merge, finds sessions and PRs that stopped moving, names the stall class, and proposes one change or none. It stays silent when delivery is healthy. It's the hourly form of [skills/run-health-review](../../skills/run-health-review/SKILL.md); [automation-health](../automation-health/README.md) is the weekly one.

The human should never have to notice a stall, compute time since last merge by eye, or rekick a session by hand.

## Triggers

| Trigger | Workflow config | Why |
|---|---|---|
| `schedule` | `cron: "11 * * * *"` (hourly) | The watchdog. Run it only while agents work unattended. |
| `workflow_dispatch` | input `mode` | Dry runs on past windows. |

## Permissions and tools (least privilege)

| Need | `permissions:` | `--allowedTools` |
|---|---|---|
| Read PRs, runs and workflows | `contents: read`, `pull-requests: read`, `actions: read` | `Read`, `Grep`, `Glob`, `Skill`, `Bash(gh pr list:*)`, `Bash(gh pr view:*)`, `Bash(git log:*)`, `Bash(gh run list:*)`, `Bash(gh run view:*)`, `Bash(gh workflow list:*)`, `Bash(gh workflow view:*)` |
| Read sessions | n/a | In Actions the live "sessions" are in-progress lever workflow runs, read with `gh run view`. Claude Code sessions elsewhere (`claude --remote`, `/loop`) are not visible from a workflow; run the same prompt as a cloud scheduled task or locally with `/loop 30m /lever health` if you need those. |
| Report | `issues: write` | `Bash(gh issue comment:*)` (one comment on `REPORT_ISSUE`) |
| Rekick (optional) | `actions: write` | + `Bash(gh run rerun:*)`, `Bash(gh workflow run:*)`. Add them only when you raise `TAKEOVER` above `propose` |

No merge, push, `gh workflow enable|disable` or edit tools. The prompt follows the [`run-health-review`](../../skills/run-health-review/SKILL.md) skill: copy it to `.claude/skills/run-health-review/` in the repository, or load the lever plugin in the action (check the action's current docs for its plugin inputs).

## Alternative: a Claude Code cloud scheduled task

An hourly watchdog is an interval-only job, so it can run as a cloud scheduled task instead of a workflow. In Claude Code run `/schedule`, paste [prompt.md](prompt.md) with PARAMETERS filled in, choose hourly (the minimum interval), and attach the repository and a GitHub connector that can read PRs and Actions runs and comment on issues. Tasks are managed at claude.ai/code/routines. Install the lever plugin where the task runs so `run-health-review` resolves. Trade-offs: no workflow file or `--allowedTools` list to review (the connector's scopes are the boundary), no `LEVER_STALL_WATCHDOG_LIVE` switch (pause it in the routines page), and takeover stays at `propose`.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `STALL_THRESHOLD` | 2 h without a merge during active hours | Below it, and with no other class firing, the run is `NOOP(healthy)` |
| `IDLE_AFTER` | 30 min with no new turn or tool call | Busy-but-idle threshold |
| `ORPHAN_AFTER` | 2 h without commits and no live owner | Takeover candidates |
| `TAKEOVER` | `propose` | `continue`, `continue_and_close`; see the takeover protocol in `run-health-review`. Above `propose`, a rekick is a rerun or re-dispatch of the stalled lever workflow |
| `NOOP_STORM` | 50% no-op wakes | Self-echo / no-op storm threshold |
| `REPORT_ISSUE` | A pinned "Stall watchdog" issue | |
| Model tier | Standard (`sonnet`) | |

## Outcome metric

- **Primary:** human rekicks per week, target 0.
- **Also:** busy-but-idle minutes per day, and stall minutes per day (time since last merge above `STALL_THRESHOLD`).
- **Guardrail:** share of hourly runs that post anything. Most should be silent.

## Install

1. Run [`automate-it`](../../skills/automate-it/SKILL.md). Create the pinned report issue and fill PARAMETERS in [prompt.md](prompt.md) with `MODE: dry_run`; commit it to `.github/lever/automations/stall-watchdog/prompt.md`. Make `run-health-review` available (see above).
2. Copy [workflow.yml](workflow.yml) to `.github/workflows/lever-stall-watchdog.yml`, commit and push, then create it paused: `gh workflow disable lever-stall-watchdog.yml`.
3. Verify with `gh workflow view lever-stall-watchdog.yml` and read the committed YAML back: no merge, push or workflow-edit entry; `actions: read`.
4. Dry-run against 3 past windows: `gh workflow enable lever-stall-watchdog.yml` (the schedule stays skipped while `LEVER_STALL_WATCHDOG_LIVE` is unset), then `gh workflow run lever-stall-watchdog.yml -f mode=dry_run` with WINDOW in PARAMETERS set to each past window's dates: one healthy, one real stall, one rate-limited. Check each class against what you saw.
5. Go live on approval with `MODE: act` in PARAMETERS: `gh variable set LEVER_STALL_WATCHDOG_LIVE --body true`. Keep `TAKEOVER: propose` until its proposals have been right for a week.
