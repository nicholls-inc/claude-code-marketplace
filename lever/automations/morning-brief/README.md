# morning-brief automation

**Purpose.** Tell one person what to focus on today, before they ask. It triages the last 24 hours of issues, lists the PRs waiting on them, reports main CI health, and ends with the top three things to do. It posts to a pinned digest issue, and posts nothing when nothing has changed.

**Why it exists.** This started as a prompt typed by hand: "Create a morning brief that tells me what to focus on today… triage issues that came in in the last 24 hours, select the top 10…". It was run 28 days in a row. Anything you do that regularly, with the same prompt, is an automation; see [skills/automate-it](../../skills/automate-it/SKILL.md).

## Triggers

| Trigger | Workflow config | Why |
|---|---|---|
| `schedule` | `cron: "13 15 * * 1-5"` (your wake hour in UTC, off the hour) | One run a day, weekdays. Cron has no timezone, so daylight saving moves it an hour. |
| `workflow_dispatch` | input `mode` | Dry runs. |

Cron can do weekdays-only, so Saturday and Sunday cost nothing; the contract still ends `NOOP(weekend)` in `TIMEZONE` as a guard for dispatches and late cron runs. For a team version, run it weekly instead (`cron: "13 15 * * 1"`). `schedule` is best-effort and can start late; see [known gaps](../README.md#known-gaps-in-github-actions).

## Permissions and tools (least privilege)

| Need | `permissions:` | `--allowedTools` |
|---|---|---|
| Read | `contents: read`, `pull-requests: read`, `actions: read` | `Read`, `Grep`, `Glob`, `Bash(gh issue list:*)`, `Bash(gh issue view:*)`, `Bash(gh pr list:*)`, `Bash(gh pr view:*)`, `Bash(gh run list:*)`, `Bash(git log:*)` |
| Automation health counts (optional) | (same) | + `Bash(gh workflow list:*)` |
| Post | `issues: write` | `Bash(gh issue comment:*)` (on the digest issue only; the contract enforces the target) |

No labels, no closes, no code. A brief that changes things isn't a brief. If you want it to triage too, pair it with [repro-gate](../repro-gate/README.md) instead of widening this one.

## Alternative: a Claude Code cloud scheduled task

This recipe only reads and posts one comment, so it fits a cloud scheduled task as well as a workflow. In Claude Code run `/schedule`, paste the contents of [prompt.md](prompt.md) with PARAMETERS filled in, choose a weekday daily cadence at your wake hour, and attach the repository and a GitHub connector that can comment on issues. Tasks are managed at claude.ai/code/routines; the minimum interval is hourly, which is no limit for a daily brief. Trade-offs: no workflow file, `permissions:` block or `--allowedTools` list to review (the connector's scopes are the boundary), no `LEVER_MORNING_BRIEF_LIVE` switch (pause the task in the routines page instead), and the run record lives in the task's run output rather than a job summary. The workflow is the better choice when you want the least-privilege lint in [automation-health](../automation-health/README.md) to cover it.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `AUDIENCE` | One person (`@you`) | Or a team; then change the trigger to weekly and the sections to merged-by-area and blocked-PR lists |
| `TOP_ISSUES` | 10 | New issues to rank |
| `FOCUS_ITEMS` | 3 | The "do these today" list |
| `DIGEST_ISSUE` | A pinned issue you own | Or a discussion (post with `gh api graphql`; add that tool to the list) |
| `QUIET_WHEN_UNCHANGED` | true | Matches `status.quiet_when_unchanged` in `.github/lever.yml` |
| `TIMEZONE` | Your timezone | Used for weekends and timestamps |
| Model tier | Standard (`sonnet`) | Ranking ten issues doesn't need premium |

## Outcome metric

- **Primary:** share of the brief's focus items acted on within 24 h (a comment, label, review, commit or close by you).
- **Also:** "status?" or "what should I look at?" asks per day (should fall); minutes from wake to first useful action.
- **Guardrail:** a high-priority issue that waited more than a day without appearing in the brief.

## Install

1. Preflight: create and pin the digest issue (title like "Morning brief"); note its number. `gh workflow list` for an existing brief. The repo needs an `ANTHROPIC_API_KEY` (or `CLAUDE_CODE_OAUTH_TOKEN`) secret (`/install-github-app`).
2. Copy [prompt.md](prompt.md) to `.github/lever/automations/morning-brief/prompt.md` and fill PARAMETERS with `MODE: dry_run`.
3. Copy [workflow.yml](workflow.yml) to `.github/workflows/lever-morning-brief.yml`, set the cron hour (UTC), commit and push, then create it paused: `gh workflow disable lever-morning-brief.yml`.
4. Verify with `gh workflow view lever-morning-brief.yml` and read the YAML back: one write (`gh issue comment`), the hour you meant (in UTC).
5. Dry run three times: `gh workflow enable lever-morning-brief.yml` (the schedule stays skipped while `LEVER_MORNING_BRIEF_LIVE` is unset), then `gh workflow run lever-morning-brief.yml -f mode=dry_run` on a weekday, and compare its ranking with your own read of the same 24 hours. If the top three don't match what you'd have done, fix the ranking rule in the contract before going live. `mode=dry_run` keeps the run from posting.
6. Go live on approval with `MODE: act` in PARAMETERS: `gh variable set LEVER_MORNING_BRIEF_LIVE --body true`. After two weeks, read the primary metric. If focus items are rarely acted on, the ranking is wrong; if the brief is mostly ignored, turn it off. Both are valid results.
