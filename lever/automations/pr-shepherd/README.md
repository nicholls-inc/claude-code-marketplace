# pr-shepherd automation

**Purpose.** Drive opted-in PRs to landed without anyone babysitting them: green on the exact head SHA, every review thread dispositioned, landed at the autonomy level you set. This is the cloud form of [skills/pr-shepherd](../../skills/pr-shepherd/SKILL.md); both use the same contract, so a manual run and an automated run behave the same.

It's the loop people hand-type as "monitor CI and push fixes until green, then merge" and the 20-minute self-wakes that poll for it.

## Triggers

| Trigger | Config | `write_scope` | Why |
|---|---|---|---|
| `pull_request_labeled` | `labels: [shepherd]` | `trigger` | Opt-in. The first run reads state and acts. |
| `workflow_run` | `conclusions: [failure]`, `workflows: [<required CI workflow>]` | `repository` | Red CI is the main reason to act. The run maps the failed run's head branch to an opted-in PR or ends `NOOP(not opted in)`. |
| `pull_request_review_submitted` | `query: label:shepherd`, states `changes_requested`, `commented` | `trigger` | New threads to disposition. |
| `pull_request_merge_conflict` | `query: label:shepherd` | `trigger` | Conflicts block landing. |
| `interval` | `hourly` | `repository` | Sweep: lands PRs that went green (there is no "CI succeeded" trigger) and catches missed events. |

`pull_request_synchronize` is left out on purpose. A push only matters once CI reports, and `workflow_run` plus the sweep cover both outcomes. Adding it would fire a run on every push that ends `NOOP(ci pending)`. Triggers decide cost.

If your PRs go green slowly and you want landing sooner than hourly, add `pull_request_synchronize` with `query: label:shepherd -is:draft` and accept the extra no-op runs, or ask the PR author's session to schedule a one-time wake.

## Tools (least privilege, by level)

| Level | Tools |
|---|---|
| Read (all levels) | `github/pull_request_read`, `github/search_pull_requests`, `github/get_file_contents`, `github/search_code`, `github/list_commits`, `github/actions_list`, `github/actions_get`, `github/get_job_logs`, `github/search_issues` |
| L1 (fix and push) | + `report_progress`, `retry_workflow_run`, `github/add_reply_to_pull_request_comment`, `github/add_issue_comment`, `github/create_issue` (one flake issue per test) |
| L2 (default) | + `github/resolve_review_thread`, `github/update_pull_request_draft_state` |
| L3 / L4 | + `github/merge_pull_request` |

Add the merge tool only when you set `AUTONOMY` to L3 or L4. Without it, no prompt text can merge.

## Dials

| Dial | Default | Options |
|---|---|---|
| `AUTONOMY` | L2 (mark ready + hand-off; a human lands) | L0–L4, capped by rulesets, CODEOWNERS, merge queue and `.github/lever.yml` |
| `SHEPHERD_LABEL` | `shepherd` | Any label; or scope by author |
| `MAX_FIX_PUSHES` | 3 per PR | Integer |
| `FLAKE_POLICY` | `rerun_once_then_file` | `rerun_once`, `never_rerun` |
| `RESOLVE_POLICY` | Resolve threads you fixed; leave declined ones for the reviewer | Resolve all once dispositioned; reviewer resolves all |
| `UPDATE_STRATEGY` | Merge base into head | Rebase (own branches only) |
| `API_BUDGET` | ≤10 calls per run; ≤75 REST calls per merged PR | Integers |
| Model tier | Standard (fast) | Premium for repos where fixes are subtle; record why |
| `STOP_AFTER` | 3 consecutive `BLOCKED` with the same reason | Integer; then remove nothing, escalate once |

## Outcome metric

- **Primary:** opted-in PRs landed (or handed off at L2) within 24 h of first green / opted-in PRs.
- **Also:** time to green, time to merge, reruns per PR (and how often a rerun actually passed), API calls per merged PR.
- **Guardrail:** reverted shepherd merges (target zero); human overrides of thread dispositions.

## Install via GitHub MCP

1. Preflight: `get_automations_configuration_for_repository`, `list_automations` (no duplicate shepherd), and `list_available_automation_tools` / `list_available_automation_triggers` to confirm ids. Create the `shepherd` label if missing.
2. Fill PARAMETERS in [prompt.md](prompt.md). Start at `AUTONOMY: L2` and `MODE: dry_run`.
3. Create paused:

```json
{
  "owner": "<owner>", "repo": "<repo>",
  "name": "pr-shepherd",
  "description": "Green on exact head, threads dispositioned, landed at AUTONOMY. Standard model tier: most runs are reads and small fixes.",
  "prompt": "<prompt.md>",
  "disabled": true,
  "ownership": "user",
  "require_actor_write_permission": true,
  "triggers": {
    "pull_request_labeled": {"types": ["labeled"], "labels": ["shepherd"], "write_scope": "trigger"},
    "workflow_run": {"types": ["completed"], "conclusions": ["failure"], "workflows": ["<CI workflow>"], "write_scope": "repository"},
    "pull_request_review_submitted": {"types": ["submitted"], "query": "label:shepherd", "write_scope": "trigger"},
    "pull_request_merge_conflict": {"types": ["conflicted"], "query": "label:shepherd", "write_scope": "trigger"},
    "interval": {"types": ["hourly"], "write_scope": "repository"}
  },
  "tools": ["github/pull_request_read", "github/search_pull_requests", "github/get_file_contents",
            "github/search_code", "github/list_commits", "github/actions_list", "github/actions_get",
            "github/get_job_logs", "github/search_issues", "report_progress", "retry_workflow_run",
            "github/add_reply_to_pull_request_comment", "github/add_issue_comment", "github/create_issue",
            "github/resolve_review_thread", "github/update_pull_request_draft_state"]
}
```

4. Verify with `get_automation`: triggers, tools (no merge tool at L2), `disabled: true`.
5. Dry run on 3 historical PRs with `run_automation` (`event: pull_request_labeled`, `type: labeled`, `resource_url: <PR>`): one red from a real bug, one red from a flake, one green with open threads. Read the three records.
6. Enable on approval with `MODE: act`. Create the [improver](../improver/README.md) paused, with `OUTPUT_QUERY: is:pr label:shepherd updated:>={window_start}`, `GOOD_OUTCOME: merged without a human push or revert`, `BAD_OUTCOME: human reverted, re-opened a thread, or pushed a fix the shepherd missed`.
7. To raise to L3/L4 later: add `github/merge_pull_request` to the full tool list and change `AUTONOMY` in the same revision.
