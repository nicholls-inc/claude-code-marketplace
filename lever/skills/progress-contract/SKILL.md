---
name: progress-contract
user-invocable: false
description: Pushes progress on long-running or delegated work so the human never has to poll. It publishes milestones with ETAs and the evidence each needs, sends reports only when something meaningful changed, labels every claim with an evidence level, logs ETA accuracy, watches time since last merge, and audits "done" against evidence before declaring it. Use for any delegated, multi-hour or overnight run, when a brief sets a report contract, or when the user asks "status?", "ETA?", "where are we?", "what happened overnight?", "hour by hour" or "are you done?".
---

# Progress contract

Every "status?" a human types is a polling loop they are running by hand. This skill turns progress into a push. At kickoff it publishes milestones with ETAs and the evidence that will prove each one. After that it reports only when something meaningful changed, in a fixed shape ([`templates/status-digest.md`](${CLAUDE_PLUGIN_ROOT}/templates/status-digest.md)). It holds itself to its own forecasts, and it watches for stalls. It ends with a **done-audit** that checks every claimed-done item against evidence.

## When to use

- Work that will run longer than about 30 minutes, run unattended, or be delegated. The `brief` skill's report contract invokes this skill.
- The user asks for status or an ETA. Answer it, then fix the contract so they don't need to ask again (step 6).
- Before any "done", "ready" or "complete" claim (step 8).

**Do not use** it for short interactive tasks that the user is watching live.

## Evidence levels

Use the weakest level that is true. Never round up.

| Level | Meaning | Proof to cite |
|---|---|---|
| claimed | The agent says it's done | Nothing yet. Say "claimed". |
| pushed | Code is on a remote branch or PR | Branch or PR link |
| green | Required checks pass on the **exact head SHA** | Check-run link + SHA |
| merged | It is on the default branch | Merge commit link |
| deployed | It is running where users are | Release or deploy link + version |
| measured | The outcome metric moved on real traffic | `metric: x → y (Δz%)`, n, link |

A status record counts only after you have **read it back**. Write the artifact, fetch it, then report. Key artifacts by `<lane>/<pr>/<head-sha>`, so a re-run on the same head is a no-op by construction.

## `/lever status` with no args

List what lever is running for the user: `is:open label:lever assignee:@me` in this repo and every repo with a dial in `~/.claude/lever.yml`. Print `lever:needs-you` items first, each with its one question and a recommended answer. Then one line per issue: outcome, lanes done/total, weakest evidence level, last check, runner. A check older than 2 hours, or a workflow run waiting for approval, is flagged as stalled. Given `#issue`, print that issue's digest. See [`docs/tracking.md`](${CLAUDE_PLUGIN_ROOT}/docs/tracking.md).

`/lever status` is read-only. It never merges, rekicks, refills, edits the issue or schedules anything. For a stall, it prints the command that acts on it, usually `/lever deliver #<issue>`.

When a run is tracked on an issue, the digest goes in the issue's `## Status` section, edited in place. Comment only for a milestone, a `needs-you` question (add the `lever:needs-you` label) or the final done-audit.

## Steps

1. **Publish the plan at kickoff.** List 3–7 milestones with human-friendly labels, not PR numbers or SHAs. For each milestone give an ETA in the operator's timezone (`status.timezone` in `.github/lever.yml`, otherwise ask once), the evidence level that counts as done, and a confidence. If there is a deadline, add a checkpoint projection: what will be true at each checkpoint up to the deadline ("hour by hour until midnight").
2. **Choose the triggers.** Report on events first: a milestone's evidence level changes, a PR merges, CI goes red, a blocker appears, a human-only question arises. Add a timed sweep only as a backstop (dial). Use the sweep to compute the watchdog and the ETA drift.
3. **Stay silent unless something changed.** A report goes out only if at least one of these is true:
   - an evidence level advanced or regressed;
   - an ETA moved by more than the slip tolerance, or a checkpoint passed;
   - a new blocker or human-only question appeared;
   - the watchdog crossed its threshold;
   - the headline metric moved beyond its noise floor.

   A scheduled wake with no changes produces no message. Never let status writing replace delivery.
4. **Report in the fixed shape.** Lead with the headline `metric: x → y (Δz%)` and p50/p90 where relevant. Then give:
   - the merged count (with the change since the last report) and the open and blocked counts;
   - time since the last merge;
   - the ETA alongside the previous forecast;
   - the milestone table with evidence links;
   - **Needs you:** at most one question, with a recommended answer, or "nothing".

   Every human-only item is surfaced once. Don't repeat it unless something about it changed.
5. **Run the watchdog.** Track time since last merge (or since the last evidence advance, for work that doesn't merge). When it passes the stall threshold during active work, the run that owns the work acts before reporting. In a `deliver` run, that's its sweep ([`autonomous-delivery`](${CLAUDE_PLUGIN_ROOT}/skills/autonomous-delivery/SKILL.md) step 4), which:
   - resumes or rekicks stalled workers;
   - hands verified-green PRs to `pr-shepherd`, which lands them if `autonomy-dial`'s `may_merge` allows it;
   - refills an empty queue;
   - re-plans an outcome stuck past the stuck threshold: a simpler plan, a second-model critique, pick one.

   Then report what it did. Don't report an alarm and wait. A bare `/lever status` only reports the stall and the command to fix it.
6. **Treat any human status ask as a miss.** Answer it plainly in the fixed shape. Then tighten the contract one step: raise the cadence, add the field they asked about, or move the report where they will see it. Log what was missing.
7. **Keep an ETA accuracy log.** For each milestone, record the forecast at kickoff, each revision with its reason, and the actual time. Report the error and its direction, early (sandbagging) or late (optimism), at the end of the run and whenever the bias is systematic. Use the log to calibrate the next forecast. If the projection slips more than one checkpoint, re-plan from the objective: cut scope, add parallelism or escalate. Don't silently extend.
8. **Run the done-audit before declaring done.** Re-read the original ask (verbatim) and the brief's done-when. For each criterion and milestone:
   - cite evidence at the required level;
   - mark any item without that evidence **not done**, even if it was claimed;
   - give a 0–100 confidence with one line of reasoning;
   - for anything under the confidence threshold, get an independent check (a second model or a fresh agent), then keep working until it passes or a stop condition fires.

   Report the audit table as the final digest, including the ETA accuracy summary.

## Dials

| Dial | Default | Options |
|---|---|---|
| Cadence | Event-driven, plus a 30-min backstop sweep | 15 min; hourly; milestones only; daily digest |
| Destination | The delegator's chat or session | Pinned issue; PR comment; discussion; morning brief automation |
| Silence threshold | The change list in step 3 | Always report on schedule; headline-only heartbeat |
| Timezone and labels | Operator's timezone; milestone names | UTC; include PR numbers |
| Planning horizon | Next milestone + end of run | Next 2 h; overnight; hour by hour to a deadline |
| Slip tolerance | 25% of remaining time, or one checkpoint | Any minutes value |
| Watchdog metric | Time since last merge | Time since last evidence advance; since last release |
| Stall threshold | 2 h during active hours, report only | 45 min with auto-rekick; off |
| Stuck threshold | 2 h with no progress on one outcome → re-plan | 30 min – 8 h |
| Done-audit confidence threshold | 80 | 50–100; require a second reviewer always |
| Verbosity | Headline + table + links | Headline only; full detail |

## Outputs

- A kickoff plan: milestones, ETAs, evidence required, confidence.
- Digests in the shape of `templates/status-digest.md`, sent only when something changed.
- An ETA accuracy log, in the ledger or the run record.
- A final done-audit table: criterion → evidence link → level → confidence → done or not done.
- The cloud version is the morning brief / status digest recipe under `automations/`, on a daily `schedule` trigger (a GitHub Actions cron, or a Claude Code cloud scheduled task).

## This skill's own feedback loop

- **Primary:** human status asks per day, or as a share of the user's messages. The target is near 0. Each ask is logged as a miss in step 6.
- **Calibration:** mean absolute ETA error and its direction across runs.
- **Honesty:** how often the done-audit flips "done" to "not done". A high rate means the brief's done-when was vague. Feed it back to `brief`.
- **Liveness:** stall minutes per night, and human rekicks, with a target of 0.

## Failure modes it prevents

- The human as poller ("ETA?", "where are we?") and as watchdog ("nothing merged in 8 hours").
- Reports that say "done" when the work is only claimed or pushed.
- Status chatter that replaces delivery, and wake-ups that report nothing new.
- ETAs nobody checks, and silent schedule slips.
- Premature "complete" that the user has to audit by hand.

## Why this exists

- In one month, 13% of one developer's typed messages asked for status, ETAs or "hour by hour" plans. In one long-running coordination session it was 266 of 1,164 messages.
- Once a fixed report shape (done, in progress, next, plus an ETA table) was checked in, status asks fell from 21–26% to 7% of messages.
- "Time since last merge" was computed by hand every morning for about two weeks ("no merged commits in 19 hours") before it became a 30-minute check. After that, stalls became visible within the hour instead of the next morning.
- ETA accountability had to be requested explicitly: "how are you tracking against those estimates?" and "I think you are sandbagging." Agents also admitted "marking this complete was wrong", which is why the done-audit exists.
- 250 scheduled wake prompts had to repeat "continue actual delivery, not recurring planning/status". The silent-unless-changed rule replaces that line.
- Validation is a number, not a screenshot. Of 112 merged PRs in one stretch, only 5 had demo media, and all had a measured result.
