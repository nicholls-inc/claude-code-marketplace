# Informal specification: power

## Fitness assessment

Smoke-test fixture for the Phase 3b-β Lean pipeline (K3 kill criterion). Every prerequisite passes:

- **#1 Deterministic algebraic semantics** — pass. `power(base, exp)` is a pure function of two `Nat` inputs.
- **#2 Provable properties** — pass. Postcondition is the standard recursive characterisation `b^0 = 1`, `b^(n+1) = b · b^n`, exists in any first-order theory of arithmetic.
- **#3 Tractable input generation** — pass. The input space is `Nat × Nat`; bounded random sampling is straightforward (the harness clamps `exp` to ≤ 16 to keep outputs in machine-int range).
- **#4 Dual-development resources** — flagged (D6 hypothesis). Treated as untested working assumption.

## Module boundary

In scope:
- `power(base, exp)` — natural-number exponentiation.

Out of scope:
- Negative or floating-point bases/exponents.
- Modular exponentiation (a separate module if/when that becomes a need).

Assumed callee contracts: none — the implementation uses only built-in arithmetic.

## Input shape

Shape (c) — bare signature plus prose intent. The smoke-test fixture is self-contained; there is no Dafny source or invariant doc.

## Preconditions

P1. `base : Nat`. (No negativity check — `Nat` is non-negative by definition.)
P2. `exp : Nat`. (Same.)

(For the smoke test, the harness additionally clamps `exp ≤ 16` so outputs stay in 64-bit int range. That is a harness choice, not a precondition of the function under spec.)

## Postconditions

Q1. `power(b, 0) = 1` for every `b : Nat`. (Including `b = 0` — by convention `0^0 = 1` in this spec.)
Q2. `power(b, n+1) = b * power(b, n)` for every `b : Nat`, `n : Nat`. (Recursive characterisation.)

## Invariants

I1. The result is always a non-negative natural — implied by typing.
I2. `power(1, n) = 1` for every `n : Nat`. (Useful sanity check for harness fuzzing.)
I3. `power(b, 1) = b` for every `b : Nat`. (Same.)

## Termination

`power` recurses on the exponent argument, decreasing toward 0. Well-founded measure: `exp`. Terminates on every input.

## Edge cases

- `exp = 0` — output is `1` (Q1), regardless of base. Includes `0^0 = 1`.
- `base = 0`, `exp > 0` — output is `0` (by Q2 unfolding: `0 * power(0, n-1) = 0`).
- `base = 1` — output is `1` for every `exp` (I2).
- `exp = 1` — output is `base` (I3).
- Large `exp` — overflows 64-bit ints when `base > 1`. The harness clamps `exp ≤ 16` to avoid this; this is a harness limitation, not a spec change.

## Worked examples

1. input: `(2, 0)`, expected: `1`, exercises: Q1, edge `exp = 0`.
2. input: `(2, 3)`, expected: `8`, exercises: Q2 (3 unfolds), I2's negation (base ≠ 1).
3. input: `(0, 5)`, expected: `0`, exercises: Q2 with absorbing zero base.
4. input: `(0, 0)`, expected: `1`, exercises: edge `0^0`.
5. input: `(1, 100)`, expected: `1`, exercises: I2.
6. input: `(7, 1)`, expected: `7`, exercises: I3.

## Ambiguities

(none — this is a smoke-test fixture)

## Pipeline forward references

- /lean-spec (sub-phase 3b.3) — translates this prose into a Lean 4 spec stub with `sorry` proof bodies.
- /lean-impl (sub-phase 3b.4) — Lean functional model of the implementation.
- /correspondence-review (sub-phase 3b.5) — classifies model-vs-source correspondence.
- /drt-oracle (sub-phase 3b.6) — differential random testing against the model.

Human sign-off: 2026-05-08
