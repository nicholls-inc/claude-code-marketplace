---
name: find-the-loop
user-invocable: false
description: >-
  Before iterating on a speedup or quality problem, build the feedback loop
  first and write it down as a Loop Card that names the outcome, the instrument
  the agent runs itself, the noise floor, the keep bar and the stop rule. Use on
  `/lever loop`, when asked to speed up, optimize, hill-climb or "make X better"
  against something measurable, and when a loop smell shows up mid-task: the
  user has pasted two or more screenshots or numbers, "still broken" has come up
  twice, the agent said "please try it" without running it, someone claims a
  win from one run, or the human is acting as the instrument. Not for one-shot
  fixes that existing tests already prove.
---

# Find the loop

The loop is the product and the agent is the motor. An agent with no trustworthy signal keeps changing code and asks the human whether it worked. That turns the human into the test runner, and progress stops. This skill runs **before any iteration**. It produces a Loop Card ([`templates/loop-card.md`](${CLAUDE_PLUGIN_ROOT}/templates/loop-card.md)) that states what is measured, how the agent measures it without help, how noisy that measurement is, what counts as a win, and when to stop. Hand the finished card to whoever does the iterating, whether that is you, a subagent, a separate Claude Code session or `experiment-arms`.

## When to use

- **The task is iterative.** Fix, speed up, reduce, improve, tune or hill-climb anything, including requests as vague as "make the editor feel snappier".
- **A loop smell appears mid-session.** Stop the current work and run this skill when you see any of these:
  - The user has pasted 2 or more screenshots, logs or numbers they collected by hand.
  - "still broken", "still slow" or "same thing" has come up twice.
  - You are about to write "please try it", "let me know if it works" or "start it now" without having run it yourself.
  - A percentage improvement is claimed from a single run (n=1), or no baseline exists.
  - The human reports a measurement such as "still 3 fps" or "CI took 40 min". They are acting as the instrument.
  - Three or more fix attempts have failed with no new signal between them.

**Do not use** it for one-shot mechanical edits (renames, copy changes, config bumps) where existing CI already proves correctness. Do not use it when a Loop Card already exists for this outcome; reuse that card and update the baseline instead. Do not force a loop onto [build-mode](${CLAUDE_PLUGIN_ROOT}/docs/modes.md) work: a large new prototype often has no signal to climb yet. File issues, run them in parallel with `autonomous-delivery`, and come back here once there's something real to measure.

## Steps

1. **Name the felt outcome.** Ask who notices when this gets better and what they would measure with a stopwatch, their eyes or a spreadsheet. Write it the way they would say it, for example "PR time-to-green on the critical path", not "job durations". Reject proxy metrics unless you can show they track the outcome.
2. **Classify the problem** with the decision tree below. It picks the loop type and the instrument. See [`decision-tree.md`](${CLAUDE_SKILL_DIR}/decision-tree.md) for the proof, noise control and pitfalls of each row. If nothing fits or the ask is vague, run the **loop interview** (step 2a).
   - **2a. Loop interview.** Ask at most these five questions, and only those you cannot answer by exploring the repo:
     1. Who notices this getting better?
     2. What would you look at to know?
     3. What is it today?
     4. What result would make you say "no, that's worse"?
     5. How long is this worth?

     No code changes until the card's Outcome, Metric and Keep bar rows are filled in.
3. **Find before you build.** Look for an instrument that already exists: tests, benchmarks, eval suites, perf probes, E2E harnesses, telemetry dashboards, CI history. Past human decisions also count, such as triaged issues and accepted or rejected review comments. Check `package.json`, `Makefile`, `scripts/`, `.github/workflows/`, `CLAUDE.md` / `AGENTS.md` and installed skills. Build a new instrument only when none fits.
4. **Prove you can operate the instrument yourself.** Run it end to end without the human. Proof is the command plus the output you observed: a number, a failing assertion, or an artifact path that you opened and inspected. That includes a screenshot or video you viewed yourself. If you would ask the user to look, build a probe instead. If you cannot run it (it needs hardware, credentials or production), say so on the card and pick the closest instrument you *can* run.
5. **Measure the noise floor (A/A).** Run the unchanged code at least twice for a deterministic check and at least 5 times for anything timed. Record the spread (CV or min to max). Use it to set n and the smallest effect you can detect. If one run with no change already looks like a "win", n is too small.
6. **Record a baseline** on a pinned ref, with n and the date. Defeat anything that would fake a result. Caches, path filters and empty commits all lie. The measured change must exercise the real path.
7. **Pre-register the keep bar** before the first change: the effect size required above the noise floor, guard metrics (cost, size, flake rate, correctness) and a generalization check (holdout set, second machine, or real traffic after merge). Once results are visible, the bar can only change as a new card version with a reason.
8. **Separate the proposer from the judge.** The agent that makes a change does not grade it. The instrument does. One change per attempt. For prompt and eval loops, read every FAIL before climbing, because the grader can be the bug.
9. **Set the leash, stop rule and autonomy level.** Tune n with short runs first, and lengthen the leash only after the instrument is proven. Take the autonomy level from [`docs/autonomy.md`](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md); this skill does not decide who merges.
10. **Name the ledger.** Record accepts *and* rejects in [`templates/ledger.md`](${CLAUDE_PLUGIN_ROOT}/templates/ledger.md), and feed the rejects back to the proposer so they are not retried.
11. **Publish the card and start iterating.** Post the Loop Card in the conversation or PR. After merge, re-measure on real traffic. If the outcome matters for longer than this task, schedule the loop, for example a nightly baseline that files an issue on regression: a `lever-*` GitHub Actions workflow (`schedule:`), a Claude Code cloud scheduled task (`/schedule`), or `/loop` for a session-scoped check (expires after 7 days).

## Decision tree

| If the problem is… | Loop | Instrument the agent runs itself | Noise control | Keep bar |
|---|---|---|---|---|
| A logic bug | Failing test first | Unit or integration repro test | Deterministic; rerun 3 times | Repro red → green, suite green |
| A UI or visual bug | Self-run app plus capture | E2E harness with DOM assertions; screenshot or video **the agent opens and inspects** | Fixed viewport, data, theme | Assertion passes; before/after media attached |
| TUI or CLI behavior | Scripted run in a PTY | PTY script, captured output and debug logs | Fixed terminal size and seed | Agent observed the expected output |
| UI performance or jank | A/A, then probe | FPS, render count, long-task trace on a replayed flow | A/A, at least 5 runs, same machine | p50 *and* p90 beat the floor |
| CI or build speed | Paired real-CI runs | Time-to-green p50/p90 on the critical path, plus billed minutes | Same SHA; nonce touches code and workflow; interleaved; n≥6 (12 for small effects) | Above the floor on p50, signal not weakened |
| Runtime or engine perf | Experiment arms | Benchmark subset, reference-correctness check, holdout | A/A spread; interleaved ABBA reps; bootstrap CI | CI excludes no change; guards hold |
| Prompt or agent behavior | Eval plus holdout | Pre-registered evals (gold plus canary), k≥3, held-out set | k repeats; calibrated grader | Pass-rate delta on holdout; generalization check |
| Classification, triage or review | Past decisions as labels | Dataset of past human decisions, held-out split | Held-out split | Precision/recall vs labels, holdout holds |
| A data or analytics claim | Independent ground truth | Raw-source query; injected known values | Two independent sources | Matches truth; injected values detected |
| A flaky test or infra | Repeat-N stats | Run the test N times; failure rate k/N | Large N; same runner class | 0/N at a preset N |
| Production behavior | Post-merge telemetry | Dashboard, error rate, latency SLO | Week over week; cohorts | Moves the right way over the window |
| "Make X better" (vague) | Loop interview | — | — | Loop Card agreed before any code |

## Dials

| Dial | Default | Options |
|---|---|---|
| Smell sensitivity | Trigger on any one smell | Two smells; manual invocation only |
| Primary metric and guards | One felt metric plus one cost or quality guard | Several guards, with every "more X" paired with its cost |
| Noise budget | A/A first; n=5 timed, n=3 deterministic | n up to 12+; bootstrap CI level |
| Keep bar | Above the A/A spread at p50 and p90 | A fixed effect size; CI must exclude no-change |
| Generalization check | Holdout for evals; re-measure after merge for perf | None; full gate (redaction, swap and lookup tests) |
| Loop speed vs fidelity | Targeted inside the loop, full suite at the gate | Full every time |
| Leash | Short runs to tune n, then the brief's timebox | Minutes up to overnight |
| Concurrency and API budget | From `.github/lever.yml` `budgets:` | Arms in parallel; poll interval |
| Judge | The instrument | Instrument plus a review by a subagent on a different model (`lever:blind-reviewer`); human audits the deltas |
| Autonomy level | Resolved by `autonomy-dial` | L0–L5, see `docs/autonomy.md` |
| Continuity | Re-measure once after merge | Nightly baseline that files an issue on regression |

## Outputs

- A filled Loop Card, posted in the conversation, PR or issue, plus the instrument committed or linked.
- A noise-floor and baseline row, the first entries in the ledger.
- A one-line proof that you ran the instrument yourself: `ran <command> → <observed result>`.

## This skill's own feedback loop

- **Metric:** human-collected signals per task. That means screenshots, numbers and "still broken" reports the user had to supply. The target is 0 after the card exists.
- **Guard:** turns from card to accepted change. If cards add overhead without shortening that, simplify the card for that problem class.
- Note each card where the instrument later turned out to lie, such as a cache hit, a grader bug or a proxy that didn't track the outcome. Add the pitfall to `decision-tree.md`.

## Failure modes it prevents

- The human as FPS meter, screenshot pump or "try it now" button.
- Shipping noise: a win from one run that disappears in an A/A run.
- Benchmark special-casing: the proposer grading its own change, or prompt edits that fit one eval case.
- Lying instruments: empty-commit nonces, warm caches, skipped paths, uncalibrated graders.
- Rediscovering dead ends, because rejects were never recorded.
- Loops that silently die after merge.

## Why this exists

- Pasted screenshots in one developer's prompts went from 49 a month to about 0 as loops moved from his eyes to agent-run instruments. Over the same period, measurement and baseline language rose from about 54 to about 292 mentions a month.
- In one app-perf session, a human acted as the FPS meter for about 9 hours at 3 fps. The fix was to ask the agent to construct a perf test, after which frame rate went 3 → 32 → ~55 fps. A 185-turn "still broken" bug and a 121-turn rewrite driven by about 15 pasted screenshots followed the same pattern.
- In one CI speedup project, 79 do-not-merge experiment PRs were measured with paired real-CI runs and 27 were kept. p50 time-to-green fell from 46.3 to 26.5 min (−43%).
- An honest loop says no. An engine hill-climb rejected 3 of 3 proposals as inside the noise. A 3-rep A/A run "found" a win with no change at all, which set the real n.
- Across ~1.8k sessions, 23 stuck agents were rescued by the same move: "you need to establish a feedback loop", not a better instruction.
