# PR-review regression eval

Guards crosscheck review quality against regressions when prompts, agents, or models change. Built from **real bug-introducing PRs mined from ev-energy/core** (revert archaeology + Sentry-issue culprit tracing), not contrived cases.

## Provenance

Mined 2026-08-27: 100 merged revert PRs classified (`data/reverts.json` — 27 confirmed production bugs) and 17 Sentry-fix PRs traced to culprits (`data/sentry_fix_pairs.json`, enriched with Sentry first-seen/occurrences/status). An AutoSaddler-style optimization loop over this dataset was evaluated and rejected: the baseline is already near ceiling (see below), so the dataset's value is regression detection, not optimization.

## Cases (`cases.json`)

- **8 bug cases** — PRs that verifiably introduced production bugs, each with the expected defect mechanism and evidence (revert rationale, Sentry issue, incident ticket). Two slices: `clean` (obvious regressions, reverted within days) and `latent` (bugs that surfaced months/years later).
- **3 control cases** — presumed-innocent merged PRs, scored for false-positive noise.

## Baseline (2026-08-27, opus, headless)

`baseline/` holds the reference review transcripts and adversarial-judge verdicts:

- Bug cases: **6 CATCH, 2 PARTIAL, 0 MISS** (partials: #12762 right conclusion wrong attribution; #4614 right risk class, missed specific guard).
- Controls: 3 definite HIGH findings total, all plausible-real, none speculative; one control fully clean.

## Running

```bash
# one case
./run_case.sh 16635 /tmp/eval-out opus

# all 11
for pr in 16635 14205 13705 9688 12762 5326 10279 4614 15200 15143 15201; do
  ./run_case.sh $pr /tmp/eval-out &
done; wait
```

Requires: `claude` CLI, `gh` authenticated for ev-energy/core, local clone at `$CORE_REPO` (default `~/repos/core`).

**Judging:** give an adversarial judge (opus, high effort) the review outputs plus each case's `expected_mechanism` from `cases.json`. Verdicts: CATCH (specific mechanism identified), PARTIAL (right area/risk class), MISS. For controls, count definite HIGH/CRITICAL findings and classify plausible-real vs speculative. The judge prompts used for the baseline are reproduced in the verdict JSONs' structure.

## Pass thresholds

- Bug cases: **0 MISS**, ≥6/8 CATCH.
- Controls: no speculative HIGH/CRITICAL; ≤3 definite HIGH/CRITICAL total.

A run below threshold means the harness change under test degraded review quality — investigate before merging.

## Caveats

- Reviews run against live GitHub PR diffs; if PRs are deleted or the repo history is rewritten, pin diffs locally.
- Controls are *presumed* innocent; a "false positive" there may be a real undiscovered issue (e.g. #15143's NOT-NULL-without-default finding). Threshold is calibrated to the baseline, not to zero.
- LLM-judged; keep the judge model/effort at least as strong as the reviewer under test.
