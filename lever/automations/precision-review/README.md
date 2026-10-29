# precision-review automation (+ improver)

**Purpose.** A code reviewer that comments only when it can prove a defect. At most two findings per PR, each backed by a proof table, posted as a COMMENT review. Silence is the default and the most common outcome.

**Why it looks like this.** Review automations tend to drift toward noise. In one set of about 2,200 review runs a day, 37% posted a review and only 9.5% left an inline comment, so most runs paid for reading a PR and produced nothing anyone acted on. The reviewers in that set had 46–49 tools each; about 8 are needed. Two review automations needed 7 hand revisions between them before they stabilised; the improver sibling exists so a person isn't the update loop.

## Triggers

| Trigger | Config | `write_scope` | Why |
|---|---|---|---|
| `pull_request_ready_for_review` | — | `trigger` | Drafts are not reviewed. Ready is the first review. |
| `pull_request_opened` | `query: -is:draft` | `trigger` | PRs opened ready. |
| `pull_request_synchronize` | `query: -is:draft` | `trigger` | New commits. The contract reviews only the delta since the last reviewed SHA. |

The `-is:draft` filter matters. Without it, an agent that pushes many small commits to a draft triggers a review on each one and multiplies cost without adding signal. Add `paths:` to skip generated or vendored files if your repo has them.

Leave `allow_copilot_triggers` off unless you want agent-authored PRs reviewed. If you turn it on, keep the `-is:draft` filter so agents' work-in-progress isn't reviewed on every push.

## Tools (least privilege: 8)

`github/pull_request_read`, `github/get_file_contents`, `github/search_code`, `github/list_commits`, `github/get_commit`, `github/create_pull_request_review`, `github/add_pull_request_review_comment`, `github/submit_pending_pull_request_review`.

No push, no approve, no request-changes, no merge, no issue writes. The contract asks for COMMENT reviews only; leaving out every other write tool makes that hard to get wrong.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `MAX_FINDINGS` | 2 | Per PR head. Not per run. |
| `MIN_PRIORITY` | P2 | P1 = wrong behaviour or data loss reachable in production; P2 = reachable bug with a workaround; P3 = latent risk. P3 posts only when `MIN_PRIORITY: P3`. |
| `MIN_CONFIDENCE` | high | `high` = traced end to end; `medium` is never posted by default |
| `REVIEW_SCOPE` | Delta since last reviewed SHA | Or full diff on `ready_for_review` |
| `SKIP_PATHS` | lockfiles, generated, vendored | Glob list |
| Model tier | Premium | Proof quality is the product here. Record why in the description; premium costs about 15× a standard call, so the trigger filters carry the budget. |

## Outcome metric

- **Primary:** findings addressed before merge / findings posted (a commit touched the cited lines, or the thread was resolved after a fix).
- **Also:** runs that posted anything / runs (expect it to be low; that's the point), runs per PR. **Review latency vs merge**: the share of reviews that finished after the PR merged ("retrospective"). Above 10% means the trigger fires too late or the queue is too long. Fix the trigger or concurrency, not the prompt. Also track **runs per PR**, with an alert above 5. One PR drew 24 review sessions in 3 days.
- **Guardrail:** bugs later traced to a reviewed PR that the reviewer missed (sample them; don't chase zero).

## Install via GitHub MCP

1. Preflight: `get_automations_configuration_for_repository`; `list_automations` to find existing reviewers. If one exists, revise it instead of adding a second voice on every PR. Note the identity it posts as; you need it for the improver's `OUTPUT_QUERY`.
2. Fill PARAMETERS in [prompt.md](prompt.md) with `MODE: dry_run`.
3. Create paused:

```json
{
  "owner": "<owner>", "repo": "<repo>",
  "name": "precision-review",
  "description": "Contradiction-first review: <=2 proven findings, COMMENT only, silent by default. Premium tier because proof quality is the product; -is:draft filter carries the cost.",
  "prompt": "<prompt.md>",
  "disabled": true,
  "ownership": "repository",
  "model": "<premium model>",
  "triggers": {
    "pull_request_ready_for_review": {"types": ["ready_for_review"], "write_scope": "trigger"},
    "pull_request_opened": {"types": ["opened"], "query": "-is:draft", "write_scope": "trigger"},
    "pull_request_synchronize": {"types": ["synchronize"], "query": "-is:draft", "write_scope": "trigger"}
  },
  "tools": ["github/pull_request_read", "github/get_file_contents", "github/search_code",
            "github/list_commits", "github/get_commit", "github/create_pull_request_review",
            "github/add_pull_request_review_comment", "github/submit_pending_pull_request_review"]
}
```

4. Verify with `get_automation`: exactly 8 tools, the `-is:draft` queries present, `disabled: true`.
5. Dry run with `MODE: dry_run`, which is a read-only replay: it accepts merged PRs, reviews the head they merged at, and posts nothing. Run it on 3 merged PRs (`event: pull_request_ready_for_review`, `resource_url: <PR>`): one that later needed a bug-fix follow-up, one clean, one large refactor. Expect a finding on the first, silence on the second, and at most two findings on the third. Each finding in the record must carry its proof table.
6. Enable on approval with `MODE: act`.
7. Create the improver from [improver-prompt.md](improver-prompt.md), paused, with the tools and triggers below. Enable it after the reviewer has about 20 PRs of output.

## Improver

| | |
|---|---|
| Triggers | `interval` weekly (`day_of_week`, `hour_utc` of your choice); `automation_failed` for the reviewer's id |
| Tools | `github/get_automation`, `github/list_automation_revisions`, `github/get_automation_revision`, `github/search_pull_requests`, `github/pull_request_read`, `github/list_commits`, `github/get_commit`, `propose_automation_update` |
| Output | At most one proposal per week, for example "raise the floor for P3 style findings: 7 of 9 were dismissed". A human applies it with `update_automation` or rejects it. |
| Metric | The reviewer's addressed-before-merge rate, per revision |

`automation_completed` is deliberately not a trigger here: the reviewer runs on every push, and an improver run per review would double the cost to learn nothing new before the week's sample is in.
