# main-red-triage automation

**Purpose.** When the default branch goes red, classify the failure within minutes (flake, regression or infrastructure), take the cheapest action that restores green, and leave one clear record. A red main blocks everyone's merges; the cost compounds by the minute.

## Triggers

| Trigger | Config | `write_scope` | Why |
|---|---|---|---|
| `workflow_run` | `types: [completed]`, `conclusions: [failure]`, `workflows: [<CI workflow>]`, `query: branch:<default branch>` | `repository` | The event. The prompt re-checks the branch, so a missing or unsupported query filter costs runs, not correctness. |
| `interval` | `hourly` | `repository` | Sweep: catches missed events and closes the loop on earlier actions (did the rerun pass, did the fix PR land). |

Do not add `automation_completed` for this worker. It would double its runs without adding signal.

## Tools (least privilege)

| Need | Tools |
|---|---|
| Read | `github/actions_list`, `github/actions_get`, `github/get_job_logs`, `github/list_commits`, `github/get_commit`, `github/pull_request_read`, `github/search_pull_requests`, `github/search_issues`, `github/get_file_contents`, `github/search_code` |
| Notify | `github/create_issue`, `github/add_issue_comment` |
| `ACTION: notify+rerun` (default) | + `retry_workflow_run` |
| `ACTION: fix_pr` | + `report_progress`, `create_pull_request` |
| Granted test-only fixes on main (L3) | + `github/merge_pull_request` |

## Dials

| Dial | Default | Options |
|---|---|---|
| `ACTION` | `notify+rerun` | `notify` (file/update one issue), `fix_pr` (also open a fix PR) |
| `AUTONOMY` | L1 (fix PR, never merged by the automation) | L3 only with a standing grant such as `test-only-main-fix` in `.github/lever.yml` |
| `QUARANTINE` | `propose` (the issue includes the exact skip/quarantine diff) | `pr` (open the quarantine PR) |
| `MAX_RERUNS` | 1 per failed run | 0–2 |
| `ATTRIBUTION_WINDOW` | Commits since the last green run on the default branch | Integer commit cap |
| Model tier | Standard | Premium if log reading is the bottleneck; record why |

## Outcome metric

- **Primary:** minutes the default branch is red per week (first red run → next green run on the default branch).
- **Also:** time to first classification, flake rate (reruns that passed / reruns), share of regressions attributed to the right commit (a human confirms or corrects the issue).
- **Guardrail:** zero tests skipped or deleted without a linked issue.

## Install via GitHub MCP

1. Preflight: `get_automations_configuration_for_repository`; `list_automations` (no existing CI-triage worker); confirm the CI workflow name with `actions_list`; create labels `main-red` and `flaky-test` if missing.
2. Fill PARAMETERS in [prompt.md](prompt.md) with `MODE: dry_run`.
3. Create paused:

```json
{
  "owner": "<owner>", "repo": "<repo>",
  "name": "main-red-triage",
  "description": "Classify default-branch CI failures (flake/regression/infra), act once, keep one issue per cause. Standard tier.",
  "prompt": "<prompt.md>",
  "disabled": true,
  "ownership": "repository",
  "triggers": {
    "workflow_run": {"types": ["completed"], "conclusions": ["failure"], "workflows": ["<CI workflow>"],
                     "query": "branch:<default branch>", "write_scope": "repository"},
    "interval": {"types": ["hourly"], "write_scope": "repository"}
  },
  "tools": ["github/actions_list", "github/actions_get", "github/get_job_logs", "github/list_commits",
            "github/get_commit", "github/pull_request_read", "github/search_pull_requests",
            "github/search_issues", "github/get_file_contents", "github/search_code",
            "github/create_issue", "github/add_issue_comment", "retry_workflow_run"]
}
```

4. Verify with `get_automation`. Confirm the `query` was accepted; if not, remove it and rely on the prompt's branch check.
5. Dry run on 3 historical failed runs on the default branch (`event: workflow_run`, `type: completed`, `resource_url: <run URL>`): one known flake, one real regression, one infrastructure failure. Compare each classification with what actually happened.
6. Enable on approval with `MODE: act`. Pair with an [improver](../improver/README.md): `GOOD_OUTCOME: classification matched the eventual fix`, `BAD_OUTCOME: a human relabeled the issue or the blamed commit was wrong`.
