---
name: spec-iterate
description: >-
  Draft and verify a Dafny formal specification from a natural language description.
  Produces a verified spec with preconditions, postconditions, and invariants.
  Use when the user wants to formally specify a function, algorithm, or correctness
  property. Triggers: "specify", "formal spec", "write a spec", "preconditions",
  "postconditions", "verify properties".
argument-hint: "[natural language description of the function to specify]"
---

# /spec-iterate — Dafny Specification Refinement

## Description

Iteratively draft and verify a Dafny formal specification from a natural language description. Produces a verified spec (method/function signatures with preconditions, postconditions, and invariants) that the user approves before implementation.

## Instructions

You are a formal verification expert. The user will describe a function or algorithm. Your job is to translate that into a verified Dafny specification.

### Step 0: Assess Verification Value

Before doing any Dafny work, quickly assess whether the user's request benefits from formal verification.

**High-value indicators (proceed with formal verification):**
- Non-trivial control flow (nested loops, recursion, complex branching on computed values)
- Quantified properties ("for all elements...", "there exists...")
- Safety-critical logic (financial calculations with many splits, access control, crypto primitives)
- Subtle correctness conditions (off-by-one risks, overflow, permutation preservation)
- User explicitly requests formal guarantees

**Low-value indicators (suggest alternatives):**
- Pure data transformation with no branching (map/filter/reduce pipelines)
- Simple CRUD operations
- String formatting or template generation
- Configuration parsing
- Thin wrappers around library calls
- Simple arithmetic derivations (a + (total - a) == total)

**Task-fitness quick reference:**

| Good fit | Poor fit |
|----------|----------|
| Concurrent state machines | Simple arithmetic derivations |
| Cryptographic protocols | CRUD with framework-specific quirks |
| Distributed consensus | ORM-heavy business logic |
| Parser/compiler correctness | Timezone/serialization edge cases |
| Financial rounding across many splits | Single subtraction with conservation |
| Sorting/searching algorithms | String formatting |

**Assessment output:**
- If mostly low-value indicators: present a recommendation suggesting property-based testing (Hypothesis/rapid) or `/lightweight-verify` as alternatives. Ask if the user wants to proceed anyway.
- If mixed or high-value: note the assessment briefly ("This function involves [reason]. Formal verification is well-suited.") and continue.
- Never block the user — always allow them to proceed with full verification if they choose.

### Step 2: Analyze the Description

Before drafting any spec, analyze the user's description for known Dafny limitations and proactively warn:

| Detected Pattern | Alert |
|---|---|
| IO, file handling, network, stdin/stdout | "Dafny verification works best for pure logic. IO operations will compile but cannot be formally verified—they'll need `{:extern}` stubs." |
| Mutable state, classes with fields | "Dafny supports mutable state but verification is harder. Consider a functional design with immutable data if possible." |
| Concurrency, threads, goroutines | "Dafny does not model concurrency. Thread-safety cannot be verified—only sequential correctness." |
| External library calls (pandas, requests, etc.) | "External library calls cannot be verified. The spec will cover your logic only; library interactions are trust boundaries." |

Present any applicable warnings before proceeding. These are warnings, not blockers—continue after informing the user.

### Step 3: Extract Formal Properties

From the user's description, identify:
- **Preconditions** (`requires`): What must be true of the inputs?
- **Postconditions** (`ensures`): What must be true of the output?
- **Invariants**: What properties must hold throughout loops or recursive calls?
- **Termination**: What decreases clause ensures termination?

Present these in plain English first, then translate to Dafny.

### Step 4: Draft the Dafny Spec

Write Dafny method/function signatures with `requires` and `ensures` clauses. Use method bodies with only `assume false;` or `...` (unimplemented) — the spec defines the contract, not the implementation.

Example structure:
```dafny
method MaxOfArray(a: array<int>) returns (max: int)
  requires a.Length > 0
  ensures forall i :: 0 <= i < a.Length ==> a[i] <= max
  ensures exists i :: 0 <= i < a.Length && a[i] == max
{
  // Implementation will be added by /generate-verified
  assume false;
}
```

### Step 5: Verify the Spec

Call `dafny_verify` with the spec. If verification fails:
1. Read the error messages carefully
2. Determine if the spec itself is inconsistent or if it just needs syntax fixes
3. Adjust and retry

**Maximum 5 verification attempts.** If still failing after 5 attempts, do not ask the user a chat-blocking question. Instead, emit a structured failure artifact at `.crosscheck/work/dafny/<spec-id>/spec-iterate-failure.md` containing the best version, per-attempt error logs, the agent's diagnosis of why each attempt failed, and a triage block with three explicit paths the human can pick at PR review:

```markdown
**Triage (mark exactly one):**
- [ ] Relax requirements — <one-line description of what to weaken in the spec>
- [ ] Fix the spec semantics — <one-line description of the semantic gap the agent identified>
- [ ] Abandon — formal verification is not the right tool for this property
```

Stop after writing the artifact. The reviewer red-pens at PR time; an orchestrator can re-dispatch `/spec-iterate` with the relaxed input.

### Step 6: Write Spec Artifact and Present for Approval

Persist the verified spec to `.crosscheck/work/dafny/<spec-id>/spec.dfy` per the persistence convention (`crosscheck/docs/orchestrator-coordination.md` §3). `<spec-id>` defaults to the slugified primary function name; the user may override at invocation time. Subsequent skills in the Dafny chain (`/generate-verified`, `/extract-code`, `/check-regressions`) consume this file directly — the user is not the state-carrier.

Present to the user:
- The path to the written spec file.
- Plain English summary of what it guarantees.
- The Evidence Summary block (below).

Wait for user approval before treating the spec as final. The approval is the legitimate governance moment — the user is signing off on the binding artifact downstream skills consume.

### Step 7: Evidence Summary and Decisions for Review

Split the post-verification handoff. The agent has already detected limitations during Step 2 — pre-fill them. The human's role is to ratify intent, not re-run the analysis.

```
## Evidence Summary (agent-verified during this run)

- Spec verifies via dafny_verify — all `requires`/`ensures` clauses internally consistent.
- {:extern} trust boundaries detected during Step 2: <list with file:line refs, or "none">.
- Dafny limitation gaps detected during Step 2 (IO/concurrency/float/external libraries): <list, or "none applicable">.
- Spec written to .crosscheck/work/dafny/<spec-id>/spec.dfy.

## Decisions for Review (human owns these)

- [ ] Does the spec capture all intended behavior? Review each `requires`/`ensures` clause against the original natural-language description.
- [ ] Are there informally-stated requirements not formalized? Either add them to the spec or document as out-of-scope.
- [ ] If trust boundaries were flagged: are the `{:extern}` assumptions safe in the production context?
```

The Evidence Summary block reports what the agent already detected (no checklist for the user to redo). The Decisions block is the irreducible human-judgment surface.

## Arguments

The user's natural language description of the function/algorithm to specify.

Example: `/spec-iterate "function that returns the maximum element of a non-empty integer array"`

## References

See `references/dafny-spec-patterns.md` for common Dafny patterns and idioms.
