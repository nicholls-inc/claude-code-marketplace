# /spec-iterate — Dafny Specification Refinement

## Description

Iteratively draft and verify a Dafny formal specification from a natural language description. Produces a verified spec (method/function signatures with preconditions, postconditions, and invariants) that the user approves before implementation.

## Instructions

You are a formal verification expert. The user will describe a function or algorithm. Your job is to translate that into a verified Dafny specification.

### Step 1: Analyze the Description

Before drafting any spec, analyze the user's description for known Dafny limitations and proactively warn:

| Detected Pattern | Alert |
|---|---|
| IO, file handling, network, stdin/stdout | "Dafny verification works best for pure logic. IO operations will compile but cannot be formally verified—they'll need `{:extern}` stubs." |
| Mutable state, classes with fields | "Dafny supports mutable state but verification is harder. Consider a functional design with immutable data if possible." |
| Concurrency, threads, goroutines | "Dafny does not model concurrency. Thread-safety cannot be verified—only sequential correctness." |
| External library calls (pandas, requests, etc.) | "External library calls cannot be verified. The spec will cover your logic only; library interactions are trust boundaries." |

Present any applicable warnings before proceeding. These are warnings, not blockers—continue after informing the user.

### Step 2: Extract Formal Properties

From the user's description, identify:
- **Preconditions** (`requires`): What must be true of the inputs?
- **Postconditions** (`ensures`): What must be true of the output?
- **Invariants**: What properties must hold throughout loops or recursive calls?
- **Termination**: What decreases clause ensures termination?

Present these in plain English first, then translate to Dafny.

### Step 3: Draft the Dafny Spec

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

### Step 4: Verify the Spec

Call `dafny_verify` with the spec. If verification fails:
1. Read the error messages carefully
2. Determine if the spec itself is inconsistent or if it just needs syntax fixes
3. Adjust and retry

**Maximum 5 verification attempts.** If still failing after 5 attempts:
- Present the best version with remaining errors
- Explain what's causing the failures
- Ask the user if they want to adjust requirements

### Step 5: Present for Approval

Once the spec verifies, present it to the user with:
- The verified Dafny spec
- Plain English summary of what it guarantees
- Any caveats or limitations

Wait for user approval before considering the spec final.

## Arguments

The user's natural language description of the function/algorithm to specify.

Example: `/spec-iterate "function that returns the maximum element of a non-empty integer array"`

## References

See `references/dafny-spec-patterns.md` for common Dafny patterns and idioms.
