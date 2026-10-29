# stall-watchdog automation

**Purpose.** An hourly check on unattended delivery. It measures time since last merge, finds sessions and PRs that stopped moving, names the stall class, and proposes one change or none. It stays silent when delivery is healthy. It's the hourly form of [skills/run-health-review](../../skills/run-health-review/SKILL.md); [automation-health](../automation-health/README.md) is the weekly one.

The human should never have to notice a stall, compute time since last merge by eye, or rekick a session by hand.

## Runner

| Part | Kind | Runs as |
|---|---|---|
| `check`: time since last merge (during `ACTIVE_HOURS_UTC`, weekdays), in-progress lever runs with no update for `IDLE_AFTER`, failed or timed-out lever runs, share of short successful runs as a no-op proxy against `NOOP_STORM`, API quota headroom | Deterministic | Plain GitHub Actions job (`gh` + `jq`). No `claude-code-action`, no model, no API key. |
| `diagnose`: classify each stall with the `run-health-review` taxonomy, takeover candidates, one proposal, the report comment | LLM-driven | `RUNNER: gha` (job `diagnose`, runs only when `check` raises a signal) or `RUNNER: routine`. `automate.runner` in `~/.claude/lever.yml` can preset it; otherwise ask once. |

A healthy hour costs one cheap job and posts nothing. The gate is deliberately sensitive: any signal hands off, and the LLM may still end `NOOP(healthy)`. The no-op proxy (a successful run shorter than 90 s) and the update-age test for busy-but-idle are heuristics; the model judges by the last log or tool timestamp.

**`RUNNER: gha`**: install steps below.

**`RUNNER: routine`**: one hourly cloud routine does both parts (it computes the check itself and exits `NOOP(healthy)` quietly). Nothing is committed to `.github/workflows/`.
1. In Claude Code run `/schedule` (alias `/routines`) and create a routine named `lever-stall-watchdog`, attached to this repository and a GitHub connector that can read PRs and Actions runs and comment on issues. Install the lever plugin where it runs so `run-health-review` resolves.
2. Trigger: **schedule, hourly** (the minimum interval). Run it only while agents work unattended.
3. Prompt: `MODE: dry_run` / `TRIGGER: schedule` / `Follow .github/lever/automations/stall-watchdog/prompt.md exactly. There is no CHECK_FACTS: compute rule 1 yourself and stay silent when healthy.`
4. Paused = create it disabled. Dry run = "run now" with `MODE: dry_run`; check one healthy, one stalled and one rate-limited window. Go live: set `MODE: act` and enable. Pause it in the routines page.
5. Least privilege: pick only this repository and the connectors above. A routine has no `--allowedTools` list, so tool limits are weaker than `gha`; the CONTRACT (no merge, no push, no workflow edits, takeover only proposed) is the control. Takeover stays at `propose`. Claude Code sessions on your own machine are visible only to a routine that can reach them; Actions cannot see them.

## Triggers

| Trigger | Workflow config | Why |
|---|---|---|
| `schedule` | `cron: "11 * * * *"` (hourly) | The watchdog. Run it only while agents work unattended. |
| `workflow_dispatch` | inputs `mode`, `force_signal` | Dry runs on past windows; `force_signal` hands off to the LLM job even when healthy. |

## Permissions and tools (least privilege)

| Job | `permissions:` | `--allowedTools` |
|---|---|---|
| `check` | `contents: read`, `pull-requests: read`, `actions: read` | n/a: plain shell, no model |
| `diagnose` | `contents: read`, `pull-requests: read`, `actions: read`, `issues: write` | `Read`, `Grep`, `Glob`, `Skill`, `Bash(gh pr list:*)`, `Bash(gh pr view:*)`, `Bash(git log:*)`, `Bash(gh run list:*)`, `Bash(gh run view:*)`, `Bash(gh workflow list:*)`, `Bash(gh workflow view:*)`, `Bash(gh issue comment:*)` (one comment on `REPORT_ISSUE`) |
| Rekick (optional) | `actions: write` | + `Bash(gh run rerun:*)`, `Bash(gh workflow run:*)`. Add them only when you raise `TAKEOVER` above `propose` |

In Actions the live "sessions" are in-progress lever workflow runs, read with `gh run view`. Claude Code sessions elsewhere (`claude --remote`, `/loop`) are not visible from a workflow. No merge, push, `gh workflow enable|disable` or edit tools. The prompt follows the [`run-health-review`](../../skills/run-health-review/SKILL.md) skill: the workflow checks out the lever plugin at the release tag `lever-v0.2.0` into `.lever-plugin/` and passes that directory as the action's `plugin_marketplaces` (a local path: the action accepts marketplace URLs only when they end in `.git`, so a URL can't be pinned to a ref) with `plugins: lever@nicholls`, and the prompt loads `lever:run-health-review` with the `Skill` tool. Don't copy the skill into `.claude/skills/`.

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
2. Copy [workflow.yml](workflow.yml) to `.github/workflows/lever-stall-watchdog.yml` (set `ACTIVE_HOURS_UTC` and the thresholds in the `check` job's `env:`), commit and push, then create it paused: `gh workflow disable lever-stall-watchdog.yml`.
3. Verify with `gh workflow view lever-stall-watchdog.yml` and read the committed YAML back: no merge, push or workflow-edit entry; `check` has no secrets.
4. Dry-run against 3 past windows: `gh workflow enable lever-stall-watchdog.yml` (the schedule stays skipped while `LEVER_STALL_WATCHDOG_LIVE` is unset), then `gh workflow run lever-stall-watchdog.yml -f mode=dry_run -f force_signal=true` with WINDOW in PARAMETERS set to each past window's dates: one healthy, one real stall, one rate-limited. Check each class against what you saw.
5. Go live on approval with `MODE: act` in PARAMETERS: `gh variable set LEVER_STALL_WATCHDOG_LIVE --body true`. Keep `TAKEOVER: propose` until its proposals have been right for a week.
