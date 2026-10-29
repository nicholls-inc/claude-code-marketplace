---
name: run-health-review
user-invocable: false
description: Stall watchdog and automation health review (lever-* GitHub Actions workflows plus Claude Code sessions and loops). Pulls recent runs, merges, no-op rate, failures, rate-limit hits and stuck PRs; classifies each stall; proposes exactly one change or "no change"; and takes over orphaned work. Use when unattended delivery stalled ("why did it stall overnight", "no merges in N hours", "it was fast the first few days, now slow"), when an automation fails or its output drops, when rate limits or quotas are hit, or on a schedule while agents run unattended.
---

# run-health-review

Unattended work fails quietly. A host restarts and children sit idle, a quota runs out at 2am, a prompt drifts, the pool empties, CI goes red and every worker politely no-ops. Without a watchdog, the human computes "time since last merge" by eye every morning. This skill makes that check a job: measure health from GitHub state, name the stall, change one thing (or nothing), and pick up work that was left half-done.

## When to use

- "Why did delivery stall overnight?", "nothing merged for 8 hours", "this was great the first few days, now it's slow".
- An automation (a `lever-*` GitHub Actions workflow) failed, or its useful-output rate dropped.
- 403/429 rate-limit errors, or CI minutes or another quota ran out.
- On a schedule: every 30–60 minutes during an active unattended run (`/loop 30m /lever health`, session-scoped, expires after 7 days); weekly for automations at steady state (a `lever-*` scheduled workflow or a Claude Code cloud scheduled task).

## When not to use

- Nothing is running unattended. Use `progress-contract` for attended work.
- A single PR is red. That is `pr-shepherd`'s job.

## Steps

1. **Set the window and baseline.** Default: last 24 h vs the prior 7-day daily average.
2. **Pull health signals, read-only, in one pass:**
   - **Time since last merge** to the default branch (the headline watchdog metric) and since the last release.
   - Merged PRs, open PRs by state: green-unmerged, red, conflicted, waiting on review, draft; age of each.
   - Ready-pool size (see `keep-the-pool-full`).
   - Per automation or worker: runs, `ACTED / NOOP / BLOCKED` mix, failures by reason (`reported`, `crashed`, `timed_out`), latest revision and date. For `lever-*` workflows: `gh run list --workflow lever-<job>.yml --json databaseId,conclusion,createdAt,event --limit 50`, then `gh run view <id> --log` for failures and the job summary (`$GITHUB_STEP_SUMMARY`) for the `ACTED/NOOP/BLOCKED` record. `cancelled` and `timed_out` are `timed_out`; a red conclusion with no record is `crashed`. For local sessions and loops, read the last tool timestamp in the session transcript (`~/.claude/projects/<encoded-cwd>/<session-id>.jsonl`; the format is internal and may drift).
   - Rate-limit and quota hits: 403/429 counts, CI minutes, API calls per merged PR.
   - Idle or interrupted Claude Code sessions (background subagents, `claude --remote` sessions, `/loop` runs) and branches with no commits since the stall began.
3. **Stay silent if healthy.** If time since last merge is under `STALL_THRESHOLD` and no signal crossed its threshold, end `NOOP(healthy)` and post nothing.
4. **Classify each stall** with the taxonomy below. Cite the evidence for each (the run, PR, log line or count). One stall can have one primary cause; list secondary causes, don't act on them.
5. **Take over orphaned work** if `TAKEOVER` allows (protocol below).
6. **Propose one change, or none.** Pick the single change with the largest expected effect on the headline metric. State the prediction ("time since last merge back under 45 min within 24 h") and how you will check it. "No change" is a valid outcome when the stall was transient or the cause is outside your control; say what you'll watch instead.
7. **Route human-only blockers once.** Permissions, budget increases, sign-ins and policy calls go in a single list with a recommended answer, surfaced in the next scheduled report. Never ask the same question twice; never block a stream on it.
8. **Report** in the status format: `time since last merge: x → y`, merged count, stalls by class, the one change (or "no change") with its prediction, takeovers, human-only list. Next run, check last run's prediction first.

### Stall taxonomy

| Class | Signal | Typical one change |
|---|---|---|
| **Supply** | Pool empty or low; workers `NOOP(no eligible work)` | Run `keep-the-pool-full`; raise its target |
| **CI** | Main red, or PRs red on the same job | Route to main-red triage; quarantine a proven flake |
| **Rate limit (secondary)** | 403 or 429 with a rate-limit body or Retry-After; bursts of content-creating writes from many lanes on one identity | Classify as **retryable, not permission**. Record "not attempted, retry after <absolute time>". Pace all lanes through one budget; serialize writes |
| **Quota (primary)** | Hourly API, CI-minute or credit cap exhausted | Lower concurrency or trigger breadth; adopt the shared API budget below |
| **Permission** | `BLOCKED` on a missing tool, scope or ruleset | One human-only request; or remove the step that needs it |
| **Prompt drift** | Rising `BLOCKED` or invalid outputs after a revision; humans overriding outputs | Revert to the last good revision, or propose one contract-rule change |
| **Orphaned** | Branch or PR with an owner that stopped (host restart, interrupted session) | Take over (below) |
| **Trigger** | Eligible items with no run (filter too narrow); or many runs per item (too broad) | Adjust one trigger filter; add a sweep |
| **Review / merge** | Green PRs waiting on threads or approvals | Thread-resolution sweep; check the autonomy level is what the human intended |
| **Busy-but-idle** | Session reports busy, but there has been no new turn or tool call for longer than `IDLE_AFTER` | Rekick with what changed and what's expected (`SendMessage` to a background subagent, or re-run the `/loop` prompt). Judge liveness by the last tool timestamp, not a status flag |
| **Host restart** | Coordinator and children went idle at the same time, right after a deploy or restart of the host they run on (a laptop sleep or closed terminal counts; `/loop` is session-scoped and does not survive it) | Resume the root first, then its children. Make "resume self" the last step of any self-deploy |
| **Self-echo / no-op storm** | No-op wakes above 50%, or any wakes triggered by the agent's own writes | Tighten the wake filter (see `automate-it` step 3) |
| **Tool-profile mismatch** | A tool fails on every call; a write tool sits in a read-only profile | Remove it, or fix the profile. Every-call failures are a config bug, not flakiness |

### Takeover protocol

1. Find candidates: open PRs or branches with no commits for longer than `ORPHAN_AFTER`, whose owner session or automation is no longer running.
2. Verify state: head SHA, CI on that SHA, unresolved threads, mergeability, and the linked issue's acceptance check.
3. Claim visibly: comment "taking over from <owner> at <sha>: <next step>" and assign yourself (or the worker that will continue).
4. Continue from the existing branch. Don't rewrite or force-push someone else's commits; add commits on top.
5. If the work is obsolete (superseded, already fixed on main), close it with a one-line reason and a link, only if `TAKEOVER` allows closing.
6. Record the takeover. If the original owner reappears, hand back with the current state.

### Shared API budget

Quota is the failure mode that grows with success: more agents, more polling, more calls per merge.
- **Inventory callers**: every automation, worker and local loop that reads GitHub, with calls per run.
- **Set a per-unit budget**, for example at most 75 REST calls per merged PR across all callers, and a per-hour ceiling below the token's limit.
- **Pace through one budget**, not N independent ones. Priority order: merge-path work first, reports last.
- **Cut calls before adding retries**: read the head SHA only; one batched GraphQL read per wake; bounded pages; cached job lists; wait on events instead of polling.
- **Alert on overspend**: calls per merged PR above budget, or any 403/429, is a `Rate limit` or `Quota` stall.

## Dials

| Dial | Default | Options |
|---|---|---|
| `STALL_THRESHOLD` | 2 h without a merge during active hours, report only | Minutes; per repo |
| Schedule | Hourly during active runs; weekly otherwise | `/loop` interval; workflow `schedule:` cron; `workflow_run` on a worker's failure |
| `TAKEOVER` | `propose` (list candidates, don't act) | `continue` (claim and push), `continue_and_close` |
| `ORPHAN_AFTER` | 2 h without commits and no live owner | Duration |
| Change authority | Propose only | Apply prompt changes via a human PR; never self-update automations or grant a worker write access to `.github/workflows/` |
| Merge stalled green PRs | Off | On, at the level set in [docs/autonomy.md](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md) |
| API budget | 75 REST calls per merged PR | Integer; per-hour ceiling |
| Quiet hours | None | No human pings between X and Y; batch into the morning report |

## Outputs

- A health report (silent when healthy) with stalls, evidence, one change or "no change", and a prediction.
- Takeover comments and records.
- At most one proposed change per run, for example an issue holding a prompt diff for a `lever-*` workflow, or a PR to its prompt file.

To run the weekly automation half unattended, install [automations/automation-health](${CLAUDE_PLUGIN_ROOT}/automations/automation-health/README.md).

## This skill's own feedback loop

- **Headline metric:** stall minutes per day (time since last merge above threshold), and human rekicks or "why did it stall?" asks per week (target zero).
- **Prediction score:** for each proposed change, did the predicted metric move within the window? Keep a running hit rate. If "no change" runs are followed by the same stall twice, escalate the class.

## Failure modes it prevents

- The human as the watchdog, computing time since last merge by eye.
- Changing three things at once and learning nothing.
- Half-done PRs left by interrupted sessions.
- Agents exhausting a shared API quota and taking every other agent down with them.
- Asking the same human-only question every hour.

## Why this exists

- Morning messages in one long unattended run: "no merged commits in 19 hours", "no commit all night", "lots of draft PRs, nothing merged for 8 hours". The human computed the metric himself for about two weeks before it became a 30-minute job; once encoded, overnight stalls became visible and short.
- Rekicking interrupted child sessions was the top manual chore (at least 7 explicit requests) until it became a rule that the human should never have to ask.
- Scheduled single-shot runs spend 7–13% of runs checking state and exiting. **Event-woken, long-lived sessions are different**: in one sample, about 62% of wakes were no-ops and about 16% were self-echo. A watchdog must know which kind of lane it is judging.
- CI babysitters once made about 45k API calls in 3 hours and hit the rate limit. The fix was a shared budget and fewer calls per merge, not retries.
- In one fleet of hosted agents, the secondary rate limit was mentioned more than 15 times as often as flaky tests. The damage came from misreading a rate-limited 403 as access-denied and reporting merges as failed. The fix was classification plus one paced budget, not retries.
- Revisions that changed one thing (a model swap, a prompt cut 15% with the same rules) could be judged; "no change" was often the right answer.
