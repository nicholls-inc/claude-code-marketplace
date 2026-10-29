# find-the-loop: decision tree detail

Each row expands one line of the table in [`SKILL.md`](SKILL.md). For each problem type it gives:

- **Proof:** what proves the agent ran the loop itself.
- **Stop rule:** when the loop is finished.
- **Lies:** how the loop most often deceives you.

Fill the Loop Card from the matching row.

## Logic bug: failing test first
- **Instrument:** the smallest unit or integration test that reproduces the report.
- **Proof:** the test fails on the current head for the reported reason (quote the assertion message), then passes after the fix.
- **Noise:** rerun 3 times. If it is not deterministic, this is a flaky problem; use that row.
- **Stop rule:** repro green, affected suite green, no new failures.
- **Lies:**
  - A test that fails for a different reason than the bug.
  - A fix that disables the check, the lint or the test.
  - "Fixed" without running the repro on the final commit.

## UI or visual bug: self-run app plus capture
- **Instrument:** the repo's E2E or browser harness. Drive the exact flow, assert on the DOM or state, and capture a screenshot or short video.
- **Proof:** the agent opens the captured image or video and describes what it sees. It also reports the assertion result. A capture nobody looked at is not proof.
- **Noise:** pin viewport, theme, fixture data, fonts and animation timing.
- **Stop rule:** the assertion passes. Before and after media are attached to the PR. The human spot-checks at most once.
- **Lies:**
  - Asserting on a component in isolation when the bug only shows up in the full app.
  - Screenshots taken before the UI settles.
  - The user pasting screenshots instead. That is the smell this skill exists to remove.

## TUI or CLI behavior: scripted PTY run
- **Instrument:** a script that runs the binary in a pseudo-terminal at a fixed size, sends keys, captures the output and enables debug logs.
- **Proof:** the captured output contains the expected text or frames. Never tell the user "start it" before the agent has seen it produce output.
- **Stop rule:** the expected output is observed by the agent, and a regression test for it exists.
- **Lies:** a black screen that "works on my machine"; output buffered until exit; testing a different entry point.

## UI performance or jank: A/A, then probe
- **Instrument:** an automated probe on a replayable flow. Measure frame time or FPS, render counts and long tasks.
- **Proof:** the probe prints a number for p50 and p90 frame time.
- **Noise:** run A/A at least 5 times on the same machine with the same build mode. Background load matters, so record it.
- **Keep bar:** p50 *and* p90 improve by more than the A/A spread.
- **Stop rule:** you hit the target, or the remaining proposals fall below the detectable effect.
- **Lies:** dev builds; profiling overhead; a human eyeballing smoothness. Localize first, for example by disabling a component to see if the jank disappears, then optimize.

## CI or build speed: paired real-CI runs
- **Instrument:** PR time-to-green p50/p90 on the critical-path lane, plus billed minutes as a guard.
- **Proof:** links to the paired workflow runs with their durations.
- **Noise:**
  - Same base SHA for both arms.
  - The measurement nonce touches **both code and the workflow file**, so caches and path filters can't skip work.
  - Arms are interleaved.
  - Reruns are capped the way a human would cap them (2 or fewer).
  - n≥6 per arm, 12 for small effects, and at least 10 pairs before a headline claim.
- **Keep bar:** the p50 critical path improves by more than the floor, for example 1 minute or more, or there is a clear cost or flake win. No change may weaken the signal (skipping tests, widening caches that mask failures).
- **Stop rule:** each layer wins in every round, and a combined run confirms the win.
- **Lies:**
  - Empty commits that hit warm caches.
  - Shaving a job that isn't on the critical path.
  - Busy-hour hypotheses that are really reruns of flaky jobs.

## Runtime or engine performance: experiment arms
- **Instrument:** a benchmark subset that runs in minutes, a correctness check against a reference implementation, and holdout inputs the proposer never sees.
- **Proof:** a per-arm table of timings with a confidence interval.
- **Noise:** measure the A/A spread first. Then run interleaved ABBA reps and a bootstrap CI. If an A/A run "wins", raise n.
- **Keep bar:** the CI excludes "no change", the point estimate is above A/A drift, and the guard metrics (memory, size, load time) hold.
- **Stop rule:** k consecutive rejects, or proposals below the detectable effect. Iterate with `experiment-arms`.
- **Lies:** special-casing the benchmark queries; the proposer running its own benchmark; a linter the candidate is allowed to loosen.

## Prompt or agent behavior: eval plus holdout
- **Instrument:** pre-registered eval cases (gold cases plus canaries) with k≥3 repeats and a held-out set.
- **Proof:** the pass rate per case over k runs, plus raw outputs for every failure.
- **Noise:** k repeats per case. Calibrate the grader by reading every FAIL before climbing, because some "failures" are grader bugs.
- **Keep bar:** the pass rate improves on the holdout, not just the training cases. Every added prompt line has to pass a generalization check:
  - **Redaction:** does it still make sense with the motivating case removed?
  - **Behavior delta:** does it actually change behavior?
  - **Swap:** is there a case where its conclusion is false?
- **Stop rule:** a plateau over N experiments, or the leash expires.
- **Lies:** prompts tuned to one case; trusting the label; an eval loop whose data pipeline silently produces nothing.

## Classification, triage or review: past decisions as labels
- **Instrument:** a dataset of decisions humans already made, such as triaged issues or review comments that were accepted versus declined, with a held-out split.
- **Proof:** precision and recall of the current prompt on the held-out split.
- **Stop rule:** a plateau, or the label budget is spent.
- **Lies:** label leakage into the prompt; measuring only on cases the prompt was tuned on.

## Data or analytics claim: independent ground truth
- **Instrument:** a query against the raw source, plus a perturbation test that injects known values and checks they are detected.
- **Proof:** two independent sources agree, and the injected values come back.
- **Lies:** treating a proxy-derived "ground truth" as truth; reasoning from a stale local copy instead of the live source.

## Flaky test or infra: repeat-N stats
- **Instrument:** run the test N times on the same runner class. Record the failure rate k/N and the distinct failure signatures.
- **Proof:** "k/N before" for the current head.
- **Stop rule:** 0/N at a preset N, sized from the before-rate. A 5% flake needs about 60 clean runs for ~95% confidence.
- **Lies:**
  - Retry-until-green.
  - Calling a real race a "flake". The top flake is often a product bug.
  - A "corruption" that is really resource exhaustion.

## Production behavior: post-merge telemetry
- **Instrument:** dashboards, error rates and latency SLOs, compared week over week or by cohort.
- **Stop rule:** the metric holds in the expected direction for the window, and a regression alarm is in place.
- **Lies:** a seasonal swing mistaken for the effect; dashboards that read from a cache.

## Vague request: loop interview
Run the five questions in `SKILL.md` step 2a, then map the answers onto a row above. If the user cannot name what would make them say "worse", propose two candidate metrics and ask them to pick one. No code changes until the Loop Card has an Outcome, a Metric and a Keep bar.
