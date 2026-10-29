---
name: refactor-pipeline
user-invocable: false
description: Runs a huge, mechanical refactor (crate or package splits, module extraction, API migrations, dead-code removal) as three independent lanes, producer, packager and shepherd, that coordinate only through git and a PR stack. It pushes branches and opens PRs, so it runs only on `/lever refactor` or an explicit ask ("split this crate", "break up this monolith", "this module is huge, cut it down", "migrate every caller").
---

# Refactor pipeline

Some refactors matter a lot and nobody wants to do them: splitting a crate that has grown to half a million lines, extracting modules, migrating every caller of an API. The work is mechanical but long. It's too big to review as one PR, and too tedious to babysit for a week.

This skill runs it as **three lanes that never talk to each other**. Each lane has one job and its own cadence. Git is the message bus, and the PR stack is the shared state.

```
PRODUCER  ──push every commit──▶  source branch  ──▶  PACKAGER  ──adds PRs──▶  PR stack  ◀──  SHEPHERD
(one split per commit,                               (rebase, test,                        (conflicts, CI,
 reports vs a fixed baseline)                         reviewable PRs)                       review threads)
                    ▲                                                                           │
                    └──────── daily: rebase onto what landed, drop merged commits ◀─────────────┘
```

You set the target, merge what's green, and route context between lanes when one of them needs to know something the others learned. The lanes do the rest.

## When to use

- The target is measurable and mechanical, for example "cut non-test LoC of `core` by 50%", "no file in `src/legacy` imports `old_client`", or "zero callers of `v1::fetch`".
- The change splits naturally into many small, independently correct units.
- The result has to land as reviewable PRs, not one giant diff.

**Don't use it for:**
- A refactor that needs design decisions at every step. Use [`brief`](${CLAUDE_PLUGIN_ROOT}/skills/brief/SKILL.md) and [`autonomous-delivery`](${CLAUDE_PLUGIN_ROOT}/skills/autonomous-delivery/SKILL.md).
- Making something measurably *better*, such as faster or less flaky. Use [`find-the-loop`](${CLAUDE_PLUGIN_ROOT}/skills/find-the-loop/SKILL.md) and [`experiment-arms`](${CLAUDE_PLUGIN_ROOT}/skills/experiment-arms/SKILL.md).
- A single PR that needs babysitting. Use [`pr-shepherd`](${CLAUDE_PLUGIN_ROOT}/skills/pr-shepherd/SKILL.md) directly.

## The three lanes

Run each lane in its own session, such as its own `claude` terminal, git worktree (`claude --worktree`) or Claude Code on the web session (`claude --remote "<prompt>"`). Separate processes keep their context small and let one lane die without taking the others down.

### 1. Producer: does the work

A long-running goal with a measurable target and a heartbeat. The whole kickoff fits in one paragraph. It needs one sentence each for:

| Part | Example |
|---|---|
| Target | "bring the non-test LoC of this crate down by 50%" |
| Unit of work | "one logical split per commit" |
| Baseline | "measure the baseline now" |
| Heartbeat | "report every 30 min against the goal, in absolute LoC and %" |
| Validation | "run the tests for the affected crates before each commit" |
| Constraint | "don't make other crates significantly bigger" |

Rules the producer must follow:
- **Push after every commit.** This one rule is what lets the packager consume work while the producer keeps going.
- **The baseline never moves.** After each daily rebase, report against the *original* baseline. No shifting the goalposts.
- **Never manufacture progress.** Don't delete, weaken or reclassify tests. Don't change what's counted. Don't leave backward-compatibility wrappers behind just to keep callers compiling.
- **Move tests with the code.** Move whole test modules and their fixtures to the new owner, not only the code under test.
- **Daily rebase:** rebase onto the latest landed state, drop commits that have already merged, and re-measure.

### 2. Packager: makes it reviewable

A scheduled lane that turns raw commits into a reviewable stack.

- **Checkpoint, don't remember.** Save the last source commit you processed, plus a stable identity for every commit you've packaged: its `git patch-id --stable`, or a `Change-Id:` trailer if the producer writes one. Every run starts with a diff: "Processed 8 new commits through `5bf1e1f`", or "No new commits since `d3137f5`. Nothing to do." The second answer is silent.
- **Reconcile after the producer's rebase.** If the source branch moved non-fast-forward, the old checkpoint may not be an ancestor anymore, and packaged commits have new SHAs. Don't diff from the checkpoint. Match source commits to packaged ones by patch-id or Change-Id, skip what's already packaged, and package only the rest. A commit whose patch changed in the rebase gets flagged, not packaged twice.
- Rebase new commits onto the top of the stack, run the tests, and fold fixups into commits that aren't in a PR yet. The packager only appends PRs on top. It never rewrites a PR that's already in the stack.
- Size each PR for a human reviewer, for example **under 2,000 changed lines, excluding pure moves**. Add PRs to the stack with the repo's stacking tool (for example `gh stack`).
- Cadence follows batch size. Start at every few hours. If runs keep finding only a commit or two, slow it down.

### 3. Shepherd: keeps the stack mergeable

A frequent lane that runs [`pr-shepherd`](${CLAUDE_PLUGIN_ROOT}/skills/pr-shepherd/SKILL.md) across every PR in the stack.

- **Rediscover stack membership on every run.** Another lane is adding PRs, and merges restack it.
- **The shepherd is the only lane that rewrites the stack.** Restacks, conflict rebases and fixups to PRs already in the stack all go through it. The packager sends it a note instead of pushing.
- **Check mergeability against the real target.** A stacked PR can say "no conflicts" against its stacked base and still conflict with `main` once the PRs below it merge. Verify with `git merge-tree` against the branch it will actually land on.
- **On conflict, push the rebase immediately, then verify.** A stale PR that's waiting on local validation blocks everyone above it. CI on the pushed head is the real check.
- **Stay armed, within budget.** A green stack is a checkpoint, not a finish line. Keep going until the stack is merged, a human says stop, the budget runs out, or 6 ticks in a row did nothing. Then post one line saying which, and stop the schedule.
- Rerun a red check only after showing that the stack didn't cause it.

## Shared rules

- **One API budget.** All three lanes poll the same GitHub quota. Gate each PR to at most one status poll per ~10 minutes, combine queries (one GraphQL call per stack, not one per check), and back off until `resetAt` on 403 or 429. In one run, this cut a shepherd cycle from 11 GraphQL requests to 3.
- **Merging is the human's call by default.** The shepherd keeps PRs green and the human merges whenever they like. Raise the [autonomy dial](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md) if you want the shepherd to land green PRs itself.
- **Corrections go into the standing order.** When you correct a lane ("push immediately, *then* verify"), the lane rewrites its own recurring prompt to include the rule, so you say it once. See [`steer-once`](${CLAUDE_PLUGIN_ROOT}/skills/steer-once/SKILL.md).
- **The human routes context.** You're the only one who sees all three lanes. Tell a lane things like "rebase onto `extract-projections`, it has most of your work adapted to a later main" or "another job is adding PRs to this stack".

## Running it

**Local:** one `claude` terminal per lane. There is no `/goal`; put the producer's target in the kickoff prompt and re-state it in the `/loop` prompt. Use `/loop` for each lane's cadence, for example producer report `/loop 30m <prompt>`, packager `/loop 3h <prompt>` and shepherd `/loop 20m <prompt>`. `/loop` is session-scoped and expires after 7 days, so re-arm long runs. Put the terminals on a machine that stays awake (a dev box or VM), not a laptop that sleeps.

**Claude Code on the web:** one `claude --remote` session per lane, with the same target and cadence written into its prompt. Each session keeps its own context and checkpoint between turns.

**In CI:** the packager and shepherd are good [`automate-it`](${CLAUDE_PLUGIN_ROOT}/skills/automate-it/SKILL.md) candidates. They are scheduled, bounded and repeatable. Keep the producer as a long-running session that you check in on.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `target` | required | metric + direction + amount, measured against a fixed baseline |
| `pr_size` | < 2,000 changed lines excluding moves | what your reviewers can actually read |
| `cadence` | producer report 30 m · packager 3 h · shepherd 20 m · producer rebase 24 h | each lane runs at the speed of what it watches |
| `autonomy` | shepherd L2 (keeps green, human merges) | see [`docs/autonomy.md`](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md) |
| `api_budget` | ≤ 1 poll per PR per 10 min | shared across all lanes |
| `spend` | your tier and budget, from `/lever` | a multi-day run makes thousands of model calls; prompt caching matters a lot |

## What good looks like

In one real five-day run on a ~564k-line Rust crate, three lanes made 135 splits and cut non-test LoC by about a third. 19 of 20 stacked PRs merged, and one of the slowest CI jobs got about 28% faster as a side effect. The human typed 38 messages, and scheduled runs fired 373 times. Most of those 38 messages were one-line corrections that the lanes wrote into their own standing orders, plus "why did you stop the schedule?"

## This skill's own feedback loop

- The producer's heartbeat is the loop. If the metric hasn't moved in three reports, stop and ask the human whether the target or the unit of work is wrong.
- Every correction you have to type twice is a missing rule. Add it to the lane's standing order and, if it's general, to this file.
