# Correspondence: Power

Pipeline step: 4 of 5 — classified by /correspondence-review.

## Verdict summary

| Class | Count | DRT effect |
|---|---|---|
| exact | 1 | runs |
| abstraction | 0 | — |
| approximation | 0 | — |
| mismatch | 0 | — |

Overall DRT readiness: **READY**.

## Definitions

### `power` — class: exact

- **Source.** `formal-verification/tests/power/sut/power.py:11-17` (the iterative `for _ in range(...)` form). The Lean def is the recursive characterisation; it computes the same function on every input *under the spec's intended postcondition Q2*. The fact that the production code has been intentionally bugged for the smoke test does not change the correspondence verdict — `/correspondence-review` classifies the *modelling*, not the production code's correctness. DRT is the right place to find the bug.
- **Behaviour.** `power b n` returns `b^n` per the standard recursive definition `b^0 = 1`, `b^(n+1) = b * b^n`.
- **Rubric step that drove the class.** Step 2.1: a literal transliteration of the source's intended behaviour exists (the recursion in the Lean def). The source's chosen iterative form is a behavioural equivalent of the recursion when the loop bound is correct; the smoke-test bug perturbs that equivalence and is exactly the kind of finding DRT is designed to surface.
- **Divergences.** None at the modelling level.
- **Theorem impact.**
  - `power_zero` (Q1): well-formed; oracle is `exact`, so a proof would discharge against the correct semantics.
  - `power_succ` (Q2): same.
  - `power_one_base` (I2): same.
  - `power_one_exp` (I3): same.
  All four theorems remain `sorry` — proof discharge is out of scope for the K3 smoke test.
- **DRT note.** Run on this def. Expected outcome: the planted bug in the iterative SUT is caught by a witness at small `(base, exp)`.

## Modelling decisions (carried forward from /lean-impl)

- **Decision.** Lean models the function as structural recursion on `exp`. The production code uses an iterative `for _ in range(...)` loop.
- **Why.** Both compute the same mathematical function `b^n` when the loop bound is correct. The Lean form is the canonical recursive characterisation; the iterative form is a behavioural equivalent.
- **Risk.** If the iterative loop's bound is off by one (as it is in the planted bug), DRT will witness the divergence at `(base, exp)` with `base != 1` and `exp > 0`.

## Mismatch issues opened

No mismatch issues. DRT may run on `power`.

## Open questions for /drt-oracle

- No `abstraction` regions, so no held-constant side-channels to coordinate.
- No `approximation` regions to skip.
- The iterative SUT is the only system under test for this smoke. The Lean side is replayed via the `oracle_reference.py` faithful equivalent until the Lean image is built (see `formal-verification/tests/power/README.md` for the carve-out).

## Source files modelled

- `formal-verification/tests/power/sut/power.py` (lines 11-17 for `power`).
