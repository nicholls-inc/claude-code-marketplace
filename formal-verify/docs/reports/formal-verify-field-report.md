# Formal Verify Plugin — Field Report

**Task:** Split charging session events across month boundaries for billing attribution
**Date:** 2026-03-05
**Outcome:** Verification passed on first attempt, 0 iterations needed

## Context

The verify-orchestrator was used to formally verify energy-splitting invariants before integrating them into a Django billing system. The core logic: when a charging session spans a month boundary, split its energy into two periods such that the sum is preserved and each period is non-negative.

## What was verified

- Energy conservation: `period1 + period2 == total` (off-peak and on-peak independently)
- Non-negativity: all output values >= 0
- Month attribution: period1 ends before boundary, period2 starts at/after boundary
- On-peak derivation: `on_peak = total - off_peak` consistency

Dafny 4.11.0 reported: **5 verified, 0 errors.** All lemma bodies were empty — the SMT solver discharged every proof obligation automatically.

## Assessment

### The verification added little value here

1. **Trivial arithmetic.** The core invariant is `a + (total - a) == total`. This is not a property where humans make mistakes or where testing has blind spots. Dafny proving it required zero insight.

2. **No iteration.** The spec passed first try. In a productive formal verification session, you'd expect multiple rounds of spec refinement as you discover underspecified edge cases. The absence of iteration suggests the problem was too simple for the tool.

3. **Extracted code wasn't usable.** The verified Python operates on pure integer inputs/outputs. The actual Django implementation uses ORM aggregates (`Sum("usage")`), `Decimal` quantization, and `sum_off_peak_energy()` with tariff lookups. The gap between the verified abstraction and the real code is too wide for the verification to provide meaningful guarantees about the shipped code.

4. **Real bugs were elsewhere.** The actual issues encountered during implementation:
   - `Decimal` precision mismatch causing model validation errors (`energy_delivered` had 6 decimal places, model allows 3)
   - Custom `TestCase` disallowing `assertEqual` with `Decimal` types
   - `transaction_period` not being set in test factories, causing split logic to be silently skipped
   - `session.type.value` returning an integer (`0`) not a string (`"SMART"`)

   None of these are expressible in Dafny. All were caught by running Django tests.

## Recommendations for the plugin

### 1. Add a complexity gate before launching verification

Before running the orchestrator, evaluate whether the specification is likely to benefit from formal verification. Heuristics:

- **Does the spec have non-trivial control flow?** (loops, recursion, branching on computed values)
- **Are there quantifiers or inductive properties?** (for-all, there-exists, loop invariants)
- **Did the user express uncertainty about correctness?**

If the answer to all is "no," suggest property-based testing (e.g., Hypothesis) instead. It provides similar confidence for simple arithmetic properties at a fraction of the cost.

### 2. Report verification difficulty, not just success

The current output says "5 verified, 0 errors" which sounds impressive but hides that the proofs were trivial. Useful metrics to surface:

- **Proof hints required:** 0 (all automatic) — indicates trivial verification
- **Solver time:** if under 1 second for all obligations, flag as likely-trivial
- **Empty lemma bodies:** if the user wrote no proof steps, note this explicitly

A message like _"All proofs were discharged automatically by the SMT solver, suggesting these properties may be simple enough for property-based testing"_ would help users calibrate expectations.

### 3. Bridge the abstraction gap

The biggest weakness: the verified code and the shipped code are structurally different. Two potential improvements:

- **Generate property-based tests** from the Dafny postconditions that run against the *actual* codebase (e.g., Hypothesis tests that call the Django methods and check the invariants hold on real ORM objects).
- **Warn when the extraction target diverges from the integration target.** If the user is working in Django/Python with ORM code, and the extraction produces pure functions, flag that the verification covers the algorithm but not the integration.

### 4. Better task-fitness signaling

The plugin should help users understand when formal verification is high-value vs. overkill:

| Good fit | Poor fit |
|----------|----------|
| Concurrent state machines | Simple arithmetic derivations |
| Cryptographic protocols | CRUD with framework-specific quirks |
| Distributed consensus | ORM-heavy business logic |
| Parser/compiler correctness | Timezone/serialization edge cases |
| Financial rounding across many splits | Single subtraction with conservation |

### 5. Consider a "lightweight mode"

For cases where the user wants *some* formal rigor but full Dafny is overkill, offer a lighter alternative:
- Generate Python `assert` preconditions/postconditions (design-by-contract)
- Generate Hypothesis property tests from the spec
- Skip Dafny entirely and produce documented invariants with runtime checks

This would cover the 80% of cases where the spec is simple but the user still wants to be explicit about invariants.

## Conclusion

The formal verification was correct but not useful for this task. The implementation bugs were all in the Django integration layer (Decimal handling, ORM semantics, test framework quirks), which Dafny cannot model. The plugin would benefit from better triage of when to apply full formal verification vs. lighter alternatives.
