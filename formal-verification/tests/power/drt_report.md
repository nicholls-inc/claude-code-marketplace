# DRT report: Power

Run: smoke (`python-reference` oracle)
RNG seed: 20260508
Inputs per def: 200
D4 case: (a) hand-written production code with no formal verification
Correspondence doc: formal-verification/correspondence/Power.md

## Verdict summary

| Status | Count |
|---|---|
| Passed (no divergence) | 72 |
| Failed (divergence found) | 128 |
| Skipped (approximation) | 0 |
| Excluded (fully Dafny-verified) | 0 |

## Failures

### `power` — 128 divergences

- **D4 case applied.** (a)
- **Witness (minimised).** Input `(base=2, exp=0)` produced oracle output `1` and SUT output `2`.
- **RNG iteration of first witness.** 2
- **Divergence classification.** general implementation bug (loop off-by-one in the iterative SUT — D7a's most common Cedar-bug class).
- **Reproducer.** `python drt_harness.py --seed 20260508 --count 200`

Total divergences (post-shrink): 128 of 200 inputs.

## Passes

- `power`: 72 inputs no divergence (seed 20260508).

## Skips (approximation)

(none)

## Exclusions (fully Dafny-verified slices)

(none — no Dafny spec for `power` in this smoke)

## Smoke-test scope note

The oracle in this run is the Python-faithful reference at `formal-verification/tests/power/oracle_reference.py`, used because the Lean image is not yet built in this session (sandbox restriction on `~/.docker/buildx/activity/`). Once the image builds, swap to the Lean runner via `--oracle lean-runner`. The harness logic, divergence minimisation, and witness reporting are exercised end-to-end by this run.
