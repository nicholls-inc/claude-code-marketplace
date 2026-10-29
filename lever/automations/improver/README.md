# improver (generic sibling template)

**Purpose.** Every worker automation gets a sibling that watches its outcomes and drafts one revision at a time for a human to apply. Without it, the human is the update loop: a known prompt bug keeps running until someone finds time to edit it. The improver turns that lag into a review queue. It never updates the worker itself.

**Pairs with:** any worker in this folder. Filled-in versions: [precision-review/improver-prompt.md](../precision-review/improver-prompt.md), [repro-gate/improver-prompt.md](../repro-gate/improver-prompt.md).

## Triggers

| Trigger | Config | Why |
|---|---|---|
| `interval` | `types: [weekly]`, `day_of_week: 1`, `hour_utc: 15` | Main path. Enough samples per revision to judge. |
| `automation_failed` | `types: [failed]`, `automation_ids: [<worker id>]`, `reasons: [crashed, timed_out, reported]` | React to real failures quickly. |
| `automation_completed` | `types: [completed]`, `automation_ids: [<worker id>]` | **Only for low-volume workers** (a few runs a day). On a busy worker this doubles the run count. |

## Tools (least privilege)

`github/get_automation`, `github/list_automation_revisions`, `github/get_automation_revision`, `github/search_issues`, `github/search_pull_requests`, `github/issue_read`, `github/pull_request_read`, `propose_automation_update`.

Optional: `github/add_issue_comment` to post the weekly summary on a tracking issue (needs `write_scope: repository`).

Never: `github/update_automation`. A human applies every proposal.

## Dials

| Dial | Default | Options |
|---|---|---|
| `MIN_SAMPLES` | 20 eligible worker outputs on the current revision | Integer; 30 for noisy metrics |
| `MAX_PROPOSALS` | 1 per run | 2 |
| Proposal scope | One PARAMETERS value or one CONTRACT rule | Model or trigger-filter change, with evidence |
| May propose | Narrowing: fewer tools, tighter filters, lower caps, stricter quiet exit | Widening tools, autonomy or write scope is human-only: describe it, don't propose it |
| Model tier | Standard | Premium only if proposals are being rejected for poor reasoning |
| Ownership | Same as the worker | — |

## Outcome metric

Accepted proposals that moved the worker's metric in the predicted direction / proposals made. Also track time from evidence to applied fix (the lag it removes) and the share of runs ending `NOOP(no new evidence)`, which should be most of them.

## Install via GitHub MCP

1. Preflight: `get_automations_configuration_for_repository`; `get_automation` on the worker to copy its id and current revision.
2. Fill PARAMETERS in [prompt.md](prompt.md) (worker id, metric, evidence query).
3. Create paused:

```json
{
  "owner": "<owner>", "repo": "<repo>",
  "name": "<worker-name>-improver",
  "description": "Proposes one revision at a time for <worker-name>. Propose-only; a human applies.",
  "prompt": "<prompt.md with PARAMETERS filled>",
  "disabled": true,
  "ownership": "user",
  "triggers": {
    "interval": {"types": ["weekly"], "day_of_week": 1, "hour_utc": 15},
    "automation_failed": {"types": ["failed"], "automation_ids": ["<worker id>"]}
  },
  "tools": ["github/get_automation", "github/list_automation_revisions", "github/get_automation_revision",
            "github/search_issues", "github/search_pull_requests", "github/issue_read",
            "github/pull_request_read", "propose_automation_update"]
}
```

4. Verify with `get_automation` (tools contain no `update_automation`; `disabled: true`).
5. Run once by hand with `run_automation` (`event: manual`) after the worker has some history. Read the proposal, then enable.
6. Optionally group the worker and improver with `create_automation_group` + `add_automation_to_group`.

## Applying a proposal (the human step)

Read the evidence and the predicted delta. Apply it, reject it with a one-line reason, or ask for a smaller change. Each applied proposal is a new dated revision; keep the worker's `prompt.md` in the repo in sync so revisions diff like code.
