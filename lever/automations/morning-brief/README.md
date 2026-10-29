# morning-brief automation

**Purpose.** Tell one person what to focus on today, before they ask. It triages the last 24 hours of issues, lists the PRs waiting on them, reports main CI health, and ends with the top three things to do. It posts to a pinned digest issue, and posts nothing when nothing has changed.

**Why it exists.** This started as a prompt typed by hand: "Create a morning brief that tells me what to focus on today… triage issues that came in in the last 24 hours, select the top 10…". It was run 28 days in a row. Anything you do that regularly, with the same prompt, is an automation; see [skills/automate-it](../../skills/automate-it/SKILL.md).

## Triggers

| Trigger | Config | `write_scope` | Why |
|---|---|---|---|
| `interval` | `daily`, `hour_utc: <your wake hour in UTC>`, `minute_utc: 0` | `repository` | One run a day. Interval runs have no triggering resource, so writing to the digest issue needs repository scope. |

The interval trigger has no weekdays-only option. The contract ends `NOOP(weekend)` on Saturday and Sunday in `TIMEZONE`, which costs two short runs a week. For a team version, run it weekly instead (`weekly`, `day_of_week: 1`).

## Tools (least privilege)

| Need | Tools |
|---|---|
| Read | `github/search_issues`, `github/issue_read`, `github/search_pull_requests`, `github/pull_request_read`, `github/actions_list`, `github/list_commits` |
| Automation health counts (optional) | `github/list_automations` |
| Post | `github/add_issue_comment` (on the digest issue only) |

No labels, no closes, no code. A brief that changes things isn't a brief. If you want it to triage too, pair it with [repro-gate](../repro-gate/README.md) instead of widening this one.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `AUDIENCE` | One person (`@you`) | Or a team; then change the trigger to weekly and the sections to merged-by-area and blocked-PR lists |
| `TOP_ISSUES` | 10 | New issues to rank |
| `FOCUS_ITEMS` | 3 | The "do these today" list |
| `DIGEST_ISSUE` | A pinned issue you own | Or a discussion (swap in `github/discussion_comment_write`) |
| `QUIET_WHEN_UNCHANGED` | true | Matches `status.quiet_when_unchanged` in `.github/lever.yml` |
| `TIMEZONE` | Your timezone | Used for weekends and timestamps |
| Model tier | Standard | Ranking ten issues doesn't need premium |

## Outcome metric

- **Primary:** share of the brief's focus items acted on within 24 h (a comment, label, review, commit or close by you).
- **Also:** "status?" or "what should I look at?" asks per day (should fall); minutes from wake to first useful action.
- **Guardrail:** a high-priority issue that waited more than a day without appearing in the brief.

## Install via GitHub MCP

1. Preflight: create and pin the digest issue (title like "Morning brief"); note its number. `list_automations` for an existing brief.
2. Fill PARAMETERS in [prompt.md](prompt.md) with `MODE: dry_run`.
3. Create paused:

```json
{
  "owner": "<owner>", "repo": "<repo>",
  "name": "morning-brief",
  "description": "Daily focus brief for <you>: new issues, PRs waiting on you, main CI, top 3. Quiet when unchanged. Standard tier.",
  "prompt": "<prompt.md>",
  "disabled": true,
  "ownership": "user",
  "triggers": {
    "interval": {"types": ["daily"], "hour_utc": 15, "minute_utc": 0, "write_scope": "repository"}
  },
  "tools": ["github/search_issues", "github/issue_read", "github/search_pull_requests",
            "github/pull_request_read", "github/actions_list", "github/list_commits",
            "github/list_automations", "github/add_issue_comment"]
}
```

4. Verify with `get_automation`: one tool that writes, the hour you meant (in UTC).
5. Dry run three times (`event: interval`): on a weekday, then compare its ranking with your own read of the same 24 hours. If the top three don't match what you'd have done, fix the ranking rule in the contract before enabling. Remember `run_automation` is a real run; `MODE: dry_run` keeps it from posting.
6. Enable on approval with `MODE: act`. After two weeks, read the primary metric. If focus items are rarely acted on, the ranking is wrong; if the brief is mostly ignored, turn it off. Both are valid results.
