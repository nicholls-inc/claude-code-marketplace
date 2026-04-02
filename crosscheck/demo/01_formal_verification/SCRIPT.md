# Demo 1: Formal Verification — SWE-bench sympy__sympy-13177

**"When is x^n mod x actually zero? Let's prove it."**

## The bug

SymPy's `Mod` function simplifies `x**n % x` to `0` unconditionally.
But `1.5**2 % 1.5 = 0.75`, not `0`. The simplification is only valid
when both values are integers and the exponent is positive.

Source: https://github.com/sympy/sympy/issues/13169

## Setup (before the demo)

- Docker running, Dafny image built: `cd crosscheck && ./scripts/build-docker.sh`
- Pre-warm: `docker run --rm crosscheck-dafny:latest --version`
- Open Claude Code in this directory

## Step 1: Show the Bug (1 min)

```bash
pytest test_mod.py -v
```

Point out:
- `TestModPowerIntegers` — all pass. For integers with positive exponents, x^n % x IS 0.
- `TestModPowerCounterexamples` — `1.5**2 % 1.5 = 0.75`. SymPy returned 0. Wrong.
- `TestCanSimplify` — the buggy guard function says "yes, simplify" even for non-integers and negative exponents.

> "SymPy's code assumed x^n mod x is always 0. The tests show it's
> only true sometimes. But WHEN exactly? Let's formally prove it."

## Step 2: Formally Specify with Dafny (2-3 min)

In Claude Code:

```
Use crosscheck to write a formal Dafny specification for the following:

Write a function Power(base, exp) that computes base^exp for non-negative
exponents, and a lemma PowerModBase that proves: for any non-zero integer
base and any positive integer exponent, Power(base, exp) % base == 0.

The lemma should use induction on the exponent. The preconditions (requires
clauses) must state exactly when the property holds — these map directly
to the conditions that SymPy's Mod simplification rule was missing.
```

**What the audience sees:**
1. Crosscheck assesses this as high verification value (quantified mathematical property)
2. Writes a Dafny `function Power(base: int, exp: nat): int` with recursive definition
3. Writes `lemma PowerModBase` with:
   - `requires base != 0`
   - `requires exp >= 1`
   - `ensures Power(base, exp) % base == 0`
4. Calls `dafny_verify` — the Z3 solver checks the proof

**Key moment:** When the `requires` clauses appear:

> "These two lines — `requires base != 0` and `requires exp >= 1` —
> are EXACTLY the conditions missing from SymPy's code. Dafny won't
> let us prove the property without them, because counterexamples
> exist (1.5^2 mod 1.5, 2^(-2) mod 2). The formal spec forces
> completeness."

Approve the spec.

## Step 3: Generate Verified Implementation (1-2 min)

```
Now generate a verified implementation of the Power function and prove the
PowerModBase lemma.
```

**What the audience sees:**
1. The Power function body (recursive multiplication)
2. The lemma proof — by induction on `exp`:
   - Base case: `exp == 1` → `Power(base, 1) == base`, and `base % base == 0`
   - Inductive step: `Power(base, exp) == base * Power(base, exp-1)`, and by IH `Power(base, exp-1) % base == 0`
3. Dafny verifies the proof passes

**Key moment:** The inductive proof structure.

> "This is a mathematical proof, verified by machine. Not a test that
> checks 10 examples — a proof that covers ALL integers and ALL
> positive exponents. Infinitely many cases, one proof."

## Step 4: Extract to Python (1 min)

```
Extract the verified code to Python.
```

**What the audience sees:**
1. A clean Python `power(base, exp)` function
2. A guard function or assertions encoding the preconditions
3. Property-based test suggestions using Hypothesis

**Key moment:** Map back to the SymPy fix.

> "The actual SWE-bench fix was a one-line change: add `q.is_integer`
> and `p.exp.is_positive` to the condition. But HOW do you know those
> are the right conditions? Dafny told us — the `requires` clauses
> are exactly the set of preconditions needed to make the proof go
> through."

## Step 5: Show the Actual Fix (30s)

Show the real diff from the PR:

```python
# Before (buggy):
p.is_Pow and p.exp.is_Integer and p.base == q

# After (fixed):
p.is_Pow and p.exp.is_integer and p.base == q and q.is_integer and p.exp.is_positive
```

> "Three missing conditions. Dafny's `requires` clauses found all three."

## Timing Budget

| Step | Duration |
|------|----------|
| Show the bug | 1:00 |
| /spec-iterate | 2:30 |
| /generate-verified | 1:30 |
| /extract-code | 1:00 |
| Show the actual fix | 0:30 |
| **Total** | **~6:30** |

## Contingencies

**Dafny verification takes many iterations:**
Narrate: "The solver is checking the induction. Each iteration adds
proof hints — assertions and calc blocks that guide Z3."

**Docker/MCP issues:**
Walk through the spec verbally. The `requires` clauses are the star.

**Dafny can't prove the lemma in 5 attempts:**
This is a non-trivial inductive proof over modular arithmetic.
If it stalls: "Even Dafny found this hard! The modular arithmetic
identity requires careful induction. But the SPEC — the requires
and ensures — is already the valuable output. It tells us exactly
what conditions are needed."
