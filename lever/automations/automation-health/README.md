# automation-health automation

**Purpose.** A weekly review of every lever workflow (`.github/workflows/lever-*.yml`) in the repository: which ones do useful work, which ones mostly no-op, which ones fail, what each useful output costs, and which ones hold more privilege than they use. It proposes **one** change, or none. It's the cloud form of [skills/run-health-review](../../skills/run-health-review/SKILL.md).

Per-worker [improvers](../improver/README.md) tune one automation's prompt against its own outcomes. This automation looks across all of them: overlapping triggers, a shared API budget, a worker that should be turned off.

## Triggers

| Trigger | Workflow config | Why |
|---|---|---|
| `schedule` | `cron: "37 16 * * 1"` (weekly, Monday) | The review. |
| `workflow_run` | `workflows: [<each lever worker>]`, `types: [completed]`, job `if:` on `conclusion` `failure` or `timed_out` | Failures between reviews. The contract files an issue only when the same failure repeats, so one crash doesn't page anyone. Add each new worker's `name:` to the list when you create it. |
| `workflow_dispatch` | input `mode` | Dry runs. |

Don't trigger it on every completion; it would run once for every run of every worker. The job `if:` drops successful runs before any model call.

## Permissions and tools (least privilege)

| Need | `permissions:` | `--allowedTools` |
|---|---|---|
| Read workflows and prompts | `contents: read` | `Read`, `Grep`, `Glob`, `Bash(gh workflow list:*)`, `Bash(gh workflow view:*)`, `Bash(git log:*)`, `Bash(git show:*)`, `Bash(git diff:*)` |
| Read outcomes | `actions: read`, `pull-requests: read` | `Bash(gh run list:*)`, `Bash(gh run view:*)`, `Bash(gh issue list:*)`, `Bash(gh issue view:*)`, `Bash(gh pr list:*)`, `Bash(gh pr view:*)` |
| Report and propose | `issues: write` | `Bash(gh issue create:*)`, `Bash(gh issue comment:*)` |
| State | (same) | `Bash(gh issue edit:*)`, only for the `## lever state` section of `REPORT_ISSUE` |

Never `Bash(gh workflow enable:*)` or `disable`, never a write to `.github/`, never a push. A proposal is an issue containing a prompt or workflow diff; a human applies it by PR.

## Alternative: a Claude Code cloud scheduled task

The weekly review can run as a cloud scheduled task instead of a workflow: `/schedule`, paste [prompt.md](prompt.md) with PARAMETERS filled in, weekly cadence (hourly is the minimum interval; weekly is fine), attach this repository and a GitHub connector that can read Actions and comment on issues. Tasks are managed at claude.ai/code/routines. You lose the `workflow_run` failure path (it has no equivalent in a scheduled task), so repeated failures wait for the next weekly run, and the task's connector scopes replace the `permissions:` block it would otherwise lint. Keep the workflow if you want failures between reviews.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `MAX_PROPOSALS` | 1 per week | Across all automations. "No change" is a valid outcome. |
| `REPEAT_THRESHOLD` | 3 identical failures in 24 h | Before filing an issue on a worker failure (`workflow_run`) |
| `NOOP_ALERT` | 90% no-op over 2 weeks | Suggests narrowing a trigger or adding an `if:`. A heartbeat or sweep is expected to no-op most of the time; judge those by what they caught. |
| `MODEL_MULTIPLIERS` | `sonnet` 1, `opus` and `haiku` from your price sheet | Relative cost per call; premium costs several times standard. Never estimated: fill in the ratios or the cost figure reads "unavailable" |
| `API_BUDGET` | ≤ 75 REST calls per merged PR across all automations | Mirrors `budgets.github_api_calls_per_merged_pr` in `.github/lever.yml` |
| `REPORT_ISSUE` | A pinned "Automation health" issue | Weekly summary goes in comments. Its body's `## lever state` section holds failure counts and last week's prediction, rewritten every run (see [durable state](../README.md#conventions-shared-by-every-recipe)) |
| Model tier | Standard (`sonnet`) | |

## Outcome metric

- **Primary:** cost per useful output across the repository's lever workflows, week over week (runs that reached the model × model multiplier / useful outputs).
- **Also:** share of proposals a human accepted, and whether the predicted effect showed up the next week.
- **Guardrail:** repeated automation failures that went more than a day without an issue.

## What it checks

Runs come from `gh run list --workflow lever-*`. Outcomes (`ACTED` / `NOOP` / `BLOCKED`) come from each worker's `RECORD_SINK` (a ledger or state issue) or, failing that, the final line and JSON record in the run log (`gh run view <id> --log`); job summaries are not exposed by the API, so workers you want audited precisely should use a `ledger_issue` sink.

| Check | Signal | Typical proposal |
|---|---|---|
| No-op share | NOOP runs / runs, per trigger. Runs skipped by an `if:` are counted apart: they cost no model call | Add an `if:` filter (draft, label, `paths:`) or drop a trigger |
| Failures | By reason: crashed, timed out, reported | Fix the contract rule that crashes; lower a cap or `--max-turns` that times out; turn real no-ops reported as failures into NOOP |
| Cost per useful output | runs × multiplier / useful outputs | Drop to `sonnet`, or narrow triggers before touching the prompt |
| Unused write permissions and tools | `permissions:` scopes and `--allowedTools` entries never exercised in the window | Remove them |
| Unfiltered triggers | `pull_request*` / `issues` with no `if:` or `paths:` | Add a draft or label filter |
| Bot-triggered runs | No `github.actor != 'claude[bot]'` guard on a job not documented as loop-safe; `allowed_bots` set without a narrowing `if:` | Add the guard, or narrow the filter |
| Premium without a reason | `--model` is `opus` with no reason in the top-of-file comment | Record the reason or drop the tier |
| No improver | Worker with no `lever-<job>-improver.yml` | Recommend one (human-only; this automation can't create it) |
| Overlap | Two workflows acting on the same event and resource | Merge them or split scope |
| Unbounded runs | No `timeout-minutes`, `--max-turns` or `concurrency:` | Add them |
| API budget | Calls per merged PR above `API_BUDGET`, if readable | Lower polling or sweep frequency first |

### Least-privilege lint

For each `.github/workflows/lever-*.yml` it reads the job's `permissions:` block and the `--allowedTools` list and checks both against the worker's contract (its prompt's `ALLOWED ACTIONS` and its recipe's permissions table):

- a `permissions:` scope above what an allowed action needs, or no job-level `permissions:` at all;
- a tool no allowed action needs, a bare `Bash`, or no `--allowedTools` list (the default grants more than the contract);
- always flagged: `Bash(gh workflow enable:*)`, `Bash(gh workflow disable:*)`, writes to `.github/workflows/`, `Bash(gh pr merge:*)` below L3 or without a standing grant, `pull_request_target`, literal secrets.

It proposes narrowing changes only, as an issue with the diff.

When a number can't be read, it reports "unavailable" and why. It never estimates a count it didn't observe.

## Install

1. Preflight: `gh auth status` has `workflow` scope; the repo has an `ANTHROPIC_API_KEY` (or `CLAUDE_CODE_OAUTH_TOKEN`) secret (`/install-github-app`); create the pinned report issue with an empty `## lever state` section (a fenced `json` block holding `{}`); `gh workflow list` to collect worker names for the `workflow_run` trigger.
2. Copy [prompt.md](prompt.md) to `.github/lever/automations/automation-health/prompt.md` and fill PARAMETERS with `MODE: dry_run`.
3. Copy [workflow.yml](workflow.yml) to `.github/workflows/lever-automation-health.yml`, list your workers' `name:` values under `workflow_run`, commit and push, then create it paused: `gh workflow disable lever-automation-health.yml`.
4. Verify with `gh workflow view lever-automation-health.yml` and read the committed YAML back: read-only apart from `issues: write`, no `gh workflow enable|disable` entry.
5. Dry run once: `gh workflow enable lever-automation-health.yml` (events stay skipped while `LEVER_AUTOMATION_HEALTH_LIVE` is unset), then `gh workflow run lever-automation-health.yml -f mode=dry_run`, and read the report. Check every number against `gh run list --workflow lever-<job>.yml`; anything it couldn't read should say "unavailable".
6. Go live on approval with `MODE: act` in PARAMETERS: `gh variable set LEVER_AUTOMATION_HEALTH_LIVE --body true`. Whenever you add a worker, add its `name:` to the `workflow_run` list.
