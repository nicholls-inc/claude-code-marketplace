# Autonomous delivery: reference

Supporting material for [`SKILL.md`](${CLAUDE_SKILL_DIR}/SKILL.md). Load it when you need a template or an example. You don't need it on every wake.

## Setup interview script

Send this as one message, and leave out any line the prompt already answered:

> Before I start, choose a preset or override any line (defaults shown):
> - **Preset:** pair / overnight / unattended. Default: overnight, because you said "while I sleep".
> - **Who lands it:** your dial says L2 (hand-off). Want me to land when green (L4) or land plus release (L5) for this run? The repo ceiling is L4.
> - **Absence window:** until 08:30 your time?
> - **Lanes:** 3–5. When I run out, I'll draw from: flaky tests, an audit against the reference, error-log triage. Change anything?
> - **Budget:** at most 75 API calls per merged PR (from `.github/lever.yml`). I'll stop and report at 80% of the CI-minute cap.
> - **Questions for you:** queued for the morning report unless something is unsafe.

If you get no answer within one turn in `overnight` or `unattended` mode, take the preset's pacing defaults and keep the operator's existing autonomy and spend dials. Without an explicit merge grant, use at most L2. A preset never raises either dial. Record `source: preset default` for each pacing line, and start. `unattended` with no finite budget doesn't start: say so and stop.

## Delegation envelope (child kickoff)

```
You own: <outcome>, labelled "<milestone label>". 1 of N lanes.
User's words: "<verbatim>"
Done when: <observable check>. Feedback loop: <instrument + command>.
You own files/paths: <list>. Shared dependencies (coordinate via parent): <list>.
Autonomy: L<n> (ceiling L<m>; source <who>, <when>). You may narrow it, never widen it.
Constraints:
  - <constraint> | source: operator | set 10-12 | expires: freeze ends | why: <reason>
  - <constraint> | source: agent:lane-3 | set 10-14 | expires: end of this assignment | why: <reason>
Budget slice: <API calls / CI minutes / time>. Report use at milestones.
Mode: unattended (auto mode / headless). Escalate to the parent, not the human. Ask a question once, completely.
Report: milestone label, state, evidence level, ETA, x → y (Δz%). Stay silent if nothing changed.
```

## Wakeup prompt (same-session sweep)

Keep the prompt short and stable, and keep state in the ledger. Wakeup prompts that grow into state dumps drift out of date.

> Continue the authorized delivery run. Read the ledger's dials and CURRENT STATE, plus any newer user messages. Reconcile live state: descendants, PRs, CI, and the last merge time. Since `<watermark>`, rekick any stalled children with what changed, hand green-but-unlanded PRs to pr-shepherd, refill lanes below `min_lanes`, apply the stuck rule to any lane past `stuck_threshold`, and act on time since last merge if it is over `stall_threshold`. Stay within the level and the budget. Update the watermark. If nothing changed, end silently.

For a fresh-session scheduled job, which has no memory of the run, also tell it where the ledger is and how to tell an active owner apart from an orphaned one, so it never starts a duplicate writer.

## Takeover checklist

- [ ] The previous owner is truly stopped or unresponsive: two rekicks with new information and no progress, or the session is gone.
- [ ] Live head SHA, CI on that head, open threads and draft state have been read from GitHub, not from the ledger.
- [ ] Any timed-out external write has been reconciled against live state before a retry.
- [ ] The claim is recorded in the ledger and on the PR or issue: "Taking over from <lane>: <reason>".
- [ ] Your work builds on the existing branch. No resets or force-pushes over another owner's commits.
- [ ] The previous owner has been archived once the handoff is confirmed.

## Standing backlog (examples for `idle_policy = refill`)

Agree on these during setup. They are examples, not a checklist. Each one must produce mergeable work or a decision, not activity for its own sake.
- Flaky-test fixes, worst first.
- Parity or UX audit against a reference build, filing one issue per gap.
- Tool-failure or error-log triage, clustered by signature.
- Performance regression review against the last baseline.
- A backlog re-plan: close stale items and split oversized ones.
- Ready-pool top-up via `keep-the-pool-full`.

## Example organizations

These are illustrations to adapt, combine or reject.
- **Small bug in one subsystem.** One agent fixes it directly. If a subtle assumption appears, get a rubber-duck consultation and keep the original owner. Judge it by the regression test, not by whether you orchestrated.
- **Cross-repo feature.** The coordinator owns the user journey and the shared contract. Each repo gets a lane. If integration keeps exposing contract drift, stop adding parallelism and agree on examples first.
- **Research under uncertainty.** Lanes are competing hypotheses, not roles. Each lane returns what would disprove its explanation. Collapse redundant branches early.
- **Broad migration.** Start with a representative slice, then batch. When batches keep hitting the same exception, fix the rule once rather than in every lane. Regroup by file ownership if overlapping edits cause rework.
- **End-to-end delivery.** Implementation lanes, then `pr-shepherd` per PR, then the release train. Bind evidence to the exact integrated SHA, because green branches don't prove that their combination works.

## Changes since v2.1.0

| v2.1.0 | v3 | Why |
|---|---|---|
| Authority inferred, then clarified | A setup interview with three presets; dials stored and re-armed | Merge authority changed four times in two weeks |
| "Recipient may narrow, not widen" | Constraints carry a source and an expiry, and agent-sourced ones are scoped to their subtree | Inherited narrowing drove authority toward zero |
| Event-driven pickup, schedules as backstop | A recursive liveness sweep with automatic rekicks | Children stalled silently after restarts |
| No watchdog | The watchdog metric is time since last merge | Measured by hand every morning for two weeks |
| Wakeups reconcile and act | Wakes with nothing new stay silent | Wakeup chatter crowded out delivery |
| Escalate on concrete uncertainty | Two hours without progress means re-plan and rubber-duck | A two-day migration that kept being polished |
| End-to-end release as one example | A release train with a single writer | Concurrent deploys, and the batched-release cadence |
| Duration is not a finish condition | `idle_policy` dial; refill from a standing backlog | "The answer can never be that we have nothing to do" |
| No status format | Status goes through the progress contract, with ETA vs. actual | Status asks were the largest category of human messages |
| Ownership transfer mentioned | An explicit takeover protocol | Interrupted work was picked up routinely |
| Reflection | Reflection plus "remove the process you invented" | Agents added gates and then waited on them |

The adaptive orchestrator-worker pattern, outcome-based delegation, model balancing, and the boundaries on authority, data, effects and evidence all carry over from v2.1.0 unchanged in spirit.
