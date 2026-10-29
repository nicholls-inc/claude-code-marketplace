# automation-health automation

**Purpose.** A weekly review of every automation in the repository: which ones do useful work, which ones mostly no-op, which ones fail, what each useful output costs, and which ones hold more privilege than they use. It proposes **one** change, or none. It's the cloud form of [skills/run-health-review](../../skills/run-health-review/SKILL.md).

Per-worker [improvers](../improver/README.md) tune one automation's prompt against its own outcomes. This automation looks across all of them: overlapping triggers, a shared API budget, a worker that should be turned off.

## Triggers

| Trigger | Config | `write_scope` | Why |
|---|---|---|---|
| `interval` | `weekly`, `day_of_week: 1`, `hour_utc: <morning>` | `repository` | The review. |
| `automation_failed` | `automation_ids: [<workers>]`, `reasons: [crashed, timed_out]` | `repository` | Failures between reviews. The contract files an issue only when the same failure repeats, so one crash doesn't page anyone. |

Don't trigger it on `automation_completed`; it would run once for every run of every worker.

## Tools (least privilege)

| Need | Tools |
|---|---|
| Read automations | `github/list_automations`, `github/get_automation`, `github/list_automation_revisions`, `github/get_automation_revision` |
| Read outcomes | `github/search_issues`, `github/search_pull_requests`, `github/issue_read`, `github/pull_request_read`, `github/actions_list` |
| Report | `github/create_issue`, `github/add_issue_comment` |
| State | `github/update_issue_body`, only for the `## lever state` section of `REPORT_ISSUE` |
| Propose | `propose_automation_update` |

Never `update_automation` or `delete_automation`. A human applies the proposal.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `MAX_PROPOSALS` | 1 per week | Across all automations. "No change" is a valid outcome. |
| `REPEAT_THRESHOLD` | 3 identical failures in 24 h | Before filing an issue on `automation_failed` |
| `NOOP_ALERT` | 90% no-op over 2 weeks | Suggests narrowing a trigger. A heartbeat or sweep is expected to no-op most of the time; judge those by what they caught. |
| `MODEL_MULTIPLIERS` | standard 1, premium 15 | Relative cost per call; premium is about 15× |
| `API_BUDGET` | ≤ 75 REST calls per merged PR across all automations | Mirrors `budgets.github_api_calls_per_merged_pr` in `.github/lever.yml` |
| `REPORT_ISSUE` | A pinned "Automation health" issue | Weekly summary goes in comments. Its body's `## lever state` section holds failure counts and last week's prediction, rewritten every run (see [durable state](../README.md#conventions-shared-by-every-recipe)) |
| Model tier | Standard | |

## Outcome metric

- **Primary:** cost per useful output across the repository's automations, week over week (runs × model multiplier / useful outputs).
- **Also:** share of proposals a human accepted, and whether the predicted effect showed up the next week.
- **Guardrail:** repeated automation failures that went more than a day without an issue.

## What it checks

| Check | Signal | Typical proposal |
|---|---|---|
| No-op share | NOOP runs / runs, per trigger | Add a query filter (`-is:draft`, a label) or drop a trigger |
| Failures | By reason: crashed, timed out, reported | Fix the contract rule that crashes; lower a cap that times out; turn real no-ops reported as failures into NOOP |
| Cost per useful output | runs × multiplier / useful outputs | Drop to standard tier, or narrow triggers before touching the prompt |
| Unused write tools | Write tools never used in the window | Remove them |
| Unfiltered PR triggers | `pull_request_*` with no `query` | Add `-is:draft` or a label filter |
| Agent-triggered runs | `allow_copilot_triggers` on with no filter | Add a filter, or turn it off |
| Premium without a reason | Premium model, no reason in the description | Record the reason or drop the tier |
| No improver | Worker with no sibling improver | Recommend one (human-only; this automation can't create it) |
| Overlap | Two automations acting on the same event and resource | Merge them or split scope |
| API budget | Calls per merged PR above `API_BUDGET`, if readable | Lower polling or sweep frequency first |

When a number can't be read, it reports "unavailable" and why. It never estimates a count it didn't observe.

## Install via GitHub MCP

1. Preflight: create the pinned report issue with an empty `## lever state` section (a fenced `json` block holding `{}`); `list_automations` to collect worker ids for the `automation_failed` trigger.
2. Fill PARAMETERS in [prompt.md](prompt.md) with `MODE: dry_run`.
3. Create paused:

```json
{
  "owner": "<owner>", "repo": "<repo>",
  "name": "automation-health",
  "description": "Weekly automation health: no-op %, failures, cost per useful output, least-privilege lint. Proposes one change or none. Standard tier.",
  "prompt": "<prompt.md>",
  "disabled": true,
  "ownership": "repository",
  "triggers": {
    "interval": {"types": ["weekly"], "day_of_week": 1, "hour_utc": 16, "write_scope": "repository"},
    "automation_failed": {"automation_ids": ["<worker ids>"], "reasons": ["crashed", "timed_out"], "write_scope": "repository"}
  },
  "tools": ["github/list_automations", "github/get_automation", "github/list_automation_revisions",
            "github/get_automation_revision", "github/search_issues", "github/search_pull_requests",
            "github/issue_read", "github/pull_request_read", "github/actions_list",
            "github/create_issue", "github/add_issue_comment", "github/update_issue_body",
            "propose_automation_update"]
}
```

If the API rejects `automation_ids` or `reasons`, create it with the interval trigger only and add `automation_failed` in the Automations UI.

4. Verify with `get_automation`.
5. Dry run once (`event: interval`) and read the report. Check every number against what you can see in the Automations view; anything it couldn't read should say "unavailable".
6. Enable on approval with `MODE: act`. Whenever you add a worker, add its id to the `automation_failed` trigger.
