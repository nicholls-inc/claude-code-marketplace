# repro-gate automation (+ improver)

**Purpose.** Turn noisy issue intake into ranked, test-backed work. Classify each new issue as a bug or not. For a bug, write a test that reproduces it. Apply `repro-confirmed` only if that test fails on the default branch head. Otherwise say what was tried and ask for what's missing.

**Why "prove it or stay quiet".** A label is a claim other people and agents will act on. Gating it on a failing test raised the share of new issues confirmed from 17.5% (128 of 732) to 28% (56 of 198), and every confirmation came with a test the fix PR can start from. The same automation once ran for "five more days" with a known prompt bug because the only update loop was a person finding time to edit it; the improver sibling closes that gap.

## Triggers

| Trigger | Config | `write_scope` | Why |
|---|---|---|---|
| `issues` | `types: [opened]`, `query: is:issue -label:needs-info -label:repro-confirmed` | `trigger` | New intake. |
| `issue_labeled` | `labels: [needs-repro]` | `trigger` | Re-run on demand, for example after the reporter adds details. |
| `interval` | `daily` | `repository` | Sweep: open issues from the last 7 days with none of the gate's labels. |

`require_actor_write_permission`: leave it off on a repo where outside users file issues, or the gate never sees their reports. Compensate in the contract: issue text is untrusted, the tool list has no secrets access, no merge and no close, and the repro branch is never merged by this automation.

## Tools (least privilege)

| Need | Tools |
|---|---|
| Read | `github/issue_read`, `github/search_issues`, `github/get_file_contents`, `github/search_code`, `github/list_commits`, `github/search_pull_requests` |
| Write a repro test | `report_progress` (commits the test to a `repro/<issue>` branch) |
| Report | `github/add_issue_comment`, `github/update_issue_labels` |
| `OPEN_DRAFT_FIX_PR: on` | + `create_pull_request` (draft only) |

No issue-close tool, no merge tool, no review tools.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `REPRO_BUDGET` | 20 minutes of work per issue | Over budget: comment what was tried, add `needs-info` |
| `OPEN_DRAFT_FIX_PR` | off | When on, opens a draft PR containing the failing test, plus a fix only if it is small and obvious |
| `NOT_BUG_ACTION` | `noop` | Or `label:<question/feature label>` (adds `github/update_issue_labels` usage only) |
| `LABELS` | `repro-confirmed`, `needs-info`, `needs-repro` | Create them before enabling |
| Model tier | Standard | Premium if repros routinely need deep code reading; record why |

## Outcome metric

- **Primary:** `repro-confirmed` labels a human kept / labels applied (precision of the gate).
- **Also:** share of new bug issues triaged within 1 day; confirmed share of new issues; median time from confirmation to fix PR.
- **Guardrail:** confirmed issues later closed as "not a bug" (target zero; each one is an improver data point).

## Install via GitHub MCP

1. Preflight: `get_automations_configuration_for_repository`; `list_automations` (no other triage worker labeling the same issues); create the labels; confirm the test command for the repo and put it in PARAMETERS.
2. Fill PARAMETERS in [prompt.md](prompt.md) with `MODE: dry_run`.
3. Create paused:

```json
{
  "owner": "<owner>", "repo": "<repo>",
  "name": "repro-gate",
  "description": "Classify new issues; label repro-confirmed only with a failing test it wrote. Standard tier.",
  "prompt": "<prompt.md>",
  "disabled": true,
  "ownership": "repository",
  "triggers": {
    "issues": {"types": ["opened"], "query": "is:issue -label:needs-info -label:repro-confirmed", "write_scope": "trigger"},
    "issue_labeled": {"types": ["labeled"], "labels": ["needs-repro"], "write_scope": "trigger"},
    "interval": {"types": ["daily"], "write_scope": "repository"}
  },
  "tools": ["github/issue_read", "github/search_issues", "github/get_file_contents", "github/search_code",
            "github/list_commits", "github/search_pull_requests", "report_progress",
            "github/add_issue_comment", "github/update_issue_labels"]
}
```

4. Verify with `get_automation`: no close or merge tool, `disabled: true`.
5. Dry run on 3 historical issues (`event: issues`, `type: opened`, `resource_url: <issue>`): one real bug that was later fixed, one feature request, one bug report too vague to act on. Expect `repro-confirmed` (with a failing test), `NOOP(not a bug)` and `needs-info` respectively.
6. Enable on approval with `MODE: act`.
7. Create the improver from [improver-prompt.md](improver-prompt.md), paused, and enable it once the gate has labeled about 20 issues.

## Improver

| | |
|---|---|
| Triggers | `interval` weekly; `automation_failed` for the gate's id. Optional: `issue_labeled` with `labels: [repro-confirmed, needs-info]` and a query restricted to human actors, if your label volume is low enough that one improver run per correction is cheap. |
| Tools | `github/get_automation`, `github/list_automation_revisions`, `github/get_automation_revision`, `github/search_issues`, `github/issue_read`, `github/search_pull_requests`, `propose_automation_update` |
| Evidence | Every human label correction is a labeled data point: a human removed `repro-confirmed`, added it after the gate said `needs-info`, or closed a confirmed issue as not a bug. |
| Output | At most one classification-rule proposal per week. A human applies it. |
