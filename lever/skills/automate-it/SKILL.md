---
name: automate-it
user-invocable: false
description: Turns a task you keep doing by hand into a paused GitHub Copilot Automation with an event trigger, a least-privilege tool list, a PARAMETERS + CONTRACT prompt, a quiet exit and a propose-only improver sibling. It creates automations, so it runs only on `/lever automate` or an explicit ask ("automate this", "run this on every PR/issue", "do this nightly", "make this run in the cloud"). When the same task has been done 3 or more times by hand, or a local self-wake has fired 5 or more times for the same job, suggest it in one line instead of running it.
---

# automate-it

Repeatable work belongs in the cloud, on an event, with nobody babysitting it. This skill moves a task you already know how to do (a prompt you keep retyping, a local template, a self-wake loop) into a GitHub Copilot Automation. The automation starts **paused**, holds only the tools its allowed actions need, ends every run with an explicit outcome, and gets a sibling that proposes improvements for a human to apply. You stop being the motor and the update loop; you become the approver.

## When to use

- The same task shape appears 3 or more times in your session history.
- A local session automation or self-wake has fired 5 or more times for the same job (polling that an event could replace).
- The user asks to automate, schedule, or "run this on every X".

## When not to use

- One-off tasks, or tasks with no checkable outcome. Run `find-the-loop` first if you can't name the metric.
- Tasks that need a human judgment call on every run. Automate the evidence-gathering and leave the call to the human.
- You have never done the task by hand. Do it manually at least once; the corrections you make become the contract.

## Steps

1. **Name the outcome and the metric.** One sentence for "done" (a comment posted, a label applied, a PR landed, a digest read). A metric with a denominator: `useful outputs / eligible events`, for example "opted-in PRs merged within 24 h of first green / opted-in PRs". A duration is not a finish condition. Refuse to continue without a denominator.
2. **Mine prior runs.** Find the last 5–20 times the task was done by hand or by a local template. Extract the decision rules, the actions actually taken, and **every correction the user made**. Each correction becomes a numbered contract rule. Example: "returning failures is lame, why can't it just return nothing to do" became the rule `NOOP is success`.
3. **Pick the trigger** from the decision table in [reference.md](reference.md#trigger-decision-table). Prefer an event over polling. Add an interval sweep as a fallback for missed events. Add a query filter (for example `-is:draft`, `label:shepherd`, `branch:main`) on every PR and issue trigger. **Don't wake on your own writes.** Exclude events whose actor is the automation's own identity, and events for a head SHA the automation just pushed. In one long-running coordinator, about 1 in 6 wakes was the agent reacting to its own push, PR or merge, and about 6 in 10 wakes did nothing. Filter at the subscription; a prompt rule still pays for the wake. **Triggers decide cost**: an unfiltered per-push trigger can multiply runs per PR without adding signal.
4. **Derive the minimal tool set from the allowed actions.** List the actions the contract allows, then map each one to tool ids with the least-privilege table in [reference.md](reference.md#least-privilege-tool-table). Read tools are always fine; every write tool needs a matching allowed action. A comment-only reviewer needs about 8 tools, not 49. Never grant `github/update_automation` to a worker or an improver.
5. **Write the prompt** as a PARAMETERS block plus a stable CONTRACT using the [template](reference.md#prompt-template). PARAMETERS is the only section that varies between repos and revisions. The CONTRACT ends every run with exactly one outcome — `ACTED(summary)`, `NOOP(reason)` or `BLOCKED(reason)` — and writes one record per run.
6. **Define the quiet exit.** State exactly when the run posts nothing: draft PR, stale head SHA, duplicate of an earlier run, nothing above the confidence floor, no change since the last digest. Silence is the default; a no-op is a successful run, not a failure. After writing the run record, **end the run**. Don't wait for the host's idle timeout. Hosted runs that did 1–2 minutes of work routinely held a runner for 15–20 minutes.
7. **Preflight (read-only).**
   - The automation tools (`create_automation` and friends) have to be available. The Copilot app has them built in. Copilot CLI hides them by default: restart with `copilot --enable-all-github-mcp-tools`. If they're missing, stop here, hand the user the drafted spec, and tell them how to get the tools. Don't pretend you created anything.
   - `get_automations_configuration_for_repository`: automations are enabled and you can create them; note allowed models and ownership.
   - `list_automations`: no existing automation with the same name or trigger.
   - `list_available_automation_triggers` and `list_available_automation_tools`: every trigger name, field and tool id in your spec exists.
8. **Create paused.** Call `create_automation` with `disabled: true`, `write_scope: "trigger"` unless a repository-wide write is required, `allow_copilot_triggers: false` unless the job is loop-safe and capped, and `require_actor_write_permission: true` on public repos where untrusted users can fire the trigger.
9. **Verify.** `get_automation` must echo the exact triggers, tools, model, write scope and `disabled: true`. Diff it against your spec. Tool and trigger edits **replace** the whole list; always resubmit the full list.
10. **Dry-run on 3 historical items.** Set `MODE: dry_run` in PARAMETERS, then either run the same prompt and tools in a local session (parity: manual and automated runs should be the same thing), or call `run_automation` with `resource_url` set to each historical item. `run_automation` is a real run, so for a strict dry run temporarily submit a read-only tool list, then restore the full list. Pick one item that should ACT, one that should NOOP and one edge case. Show the user the three records.
11. **Enable on approval.** Ask once, with the dry-run records attached. Then `update_automation` with `disabled: false` and `MODE: act`. Don't ask again for permission you already have (see [autonomy](../../docs/autonomy.md)).
12. **Pair with an improver sibling.** Create it from [automations/improver](../../automations/improver/README.md), also paused. Optionally group both with `create_automation_group` and `add_automation_to_group`.
13. **Qualify before you trust it.** Treat the automation as migrated only once it has produced real outputs *and* kept picking up work from events, with no local prompting, over an agreed window (for example a week). Stop any local job doing the same work so the two never compete. One good run isn't a migration.
14. **Hand off.** Report the automation id, revision number, model tier and why, how to pause it, where its records live, and the metric's baseline.

## Dials

| Dial | Default | Options |
|---|---|---|
| Trigger breadth | One event trigger + query filter + daily or hourly sweep | Event only; sweep only; more event types (each adds runs) |
| Query filter | `-is:draft` on PR triggers; label or branch filter where possible | None (costly), path filter, label opt-in |
| Wake filter | Exclude self-actor and self-pushed SHAs | Include (only for loop-safe, capped jobs) |
| Model tier | Standard tier | Premium tier costs about 15× per call. Record why in the description; let the improver re-test the choice |
| Write scope | `trigger` (only the resource that fired) | `repository` (digests, tracking issues, ledgers) |
| Autonomy level (`AUTONOMY`) | The level the recipe states, never above the repo default; L0 if the recipe doesn't say | Up to L5 per [docs/autonomy.md](../../docs/autonomy.md), capped by repo rules; the tool list enforces it |
| Writes per run | 1 | Integer; plus open-automation-PR cap (5) |
| Evidence budget | 3 items assessed, 10 logs read, 20 records scanned | Integers |
| API budget | 75 REST calls per useful output, shared across automations | Integer; see `run-health-review` |
| `allow_copilot_triggers` | `false` | `true` with a daily usage limit (it can trigger itself) |
| Ownership | `user` for the pilot | `repository` once proven |
| Improver authority | Propose-only | There is no "apply" setting; a human applies |
| Rollout | paused → dry_run → act on one repo | → more repos → org |

The tool list is the enforcement; PARAMETERS is the intent. Below L3 the merge tool is not in the list at all.

## Outputs

- A paused automation (id, revision 1) whose definition matches the spec.
- A prompt file with PARAMETERS + CONTRACT, kept in the repo (for example `automations/<job>/prompt.md`) so revisions diff like code. Start from a [recipe](../../automations/README.md) when one fits.
- Three dry-run records and the metric baseline.
- A paused improver sibling.

## This skill's own feedback loop

- **Metric:** cost per useful output = (runs × model multiplier) / useful outputs, tracked weekly with the post rate and NOOP mix.
- **Active ratio:** tool-active minutes / wall minutes per run. Below 0.3 means runs idle out instead of ending. **Self-echo rate:** wakes caused by the automation's own writes / all wakes; the target is 0.
- **Check at 7 days:** runs/day vs eligible events (are triggers too broad?), BLOCKED reasons, failures that were really no-ops, write tools never used (remove them).
- Every change is a new dated revision with one reason. "No change" is a valid weekly outcome.

## Failure modes it prevents

- Review-only bots holding `create_pull_request` and `update_issue_state`.
- No-op runs reported as failures, waking the improver on noise.
- Per-push triggers without a draft filter multiplying cost.
- A known prompt bug running for days because a human is the only update loop.
- Silent automations: running all day and producing no measurable data point.
- Adding one tool and accidentally dropping the rest.

## Why this exists

- Code review ran as a local template for about 3.3k turns across 6 repos before it became 2 cloud automations, which were then graded about 230 times. Automate what you have already done by hand.
- Those review-only automations held 46–49 tools. The allowed actions (read, comment, review) need about 8.
- "I was the update loop": the review automations took 7 hand revisions in two months, and an issue-repro automation ran five more days with a known prompt bug before a human edited it.
- "I didn't add more agents, I added more pushes": runs per PR went up ~2–3× because a per-push trigger had no draft filter.
- One user typed the same morning-brief prompt by hand 28 days in a row. "Monitor CI and push fixes until green" was hand-typed 109 times. Both are one trigger each.
