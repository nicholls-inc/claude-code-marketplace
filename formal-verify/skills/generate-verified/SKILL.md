# /generate-verified — Verified Dafny Implementation

## Description

Generate a Dafny implementation that satisfies a verified spec. Iteratively add proof hints, loop invariants, and lemmas until the verifier accepts the code.

## Instructions

You are a formal verification expert. The user has an approved Dafny specification (from `/spec-iterate` or provided directly). Your job is to write an implementation body that Dafny's verifier accepts.

### Step 1: Review the Spec

Read the spec carefully. Identify:
- What methods/functions need implementation bodies
- What the requires/ensures clauses demand
- What data structures are involved (arrays, sequences, datatypes)
- Whether ghost state or lemmas will likely be needed

### Step 2: Generate the Implementation

Replace placeholder bodies (`assume false;` or `...`) with actual Dafny code. Key strategies:
- **Start simple**: Write the most straightforward implementation first
- **Add loop invariants**: Every `while` loop needs invariants that:
  1. Are true before the loop starts
  2. Are preserved by each iteration
  3. Together with the negated loop guard, imply the postcondition
- **Add assertions**: Strategic `assert` statements help the verifier at intermediate points
- **Use calc blocks**: For complex arithmetic proofs
- **Add lemmas**: Factor out reusable proof obligations into separate lemmas

### Step 3: Verify

Call `dafny_verify` with the full program (spec + implementation).

If verification fails, analyze the errors and apply these repair strategies:

| Error Type | Repair Strategy |
|---|---|
| "postcondition might not hold" | Strengthen loop invariants or add assertions before the return |
| "loop invariant might not be maintained" | The invariant is too strong or the loop body has a bug — weaken or fix |
| "loop invariant might not hold on entry" | The invariant doesn't match initial state — adjust initialization or invariant |
| "decreases clause might not decrease" | Fix the termination measure or restructure the recursion |
| "index out of range" | Add bounds checks or strengthen preconditions |
| "assertion might not hold" | The assertion is wrong or needs intermediate lemma support |
| "cannot prove termination" | Add explicit `decreases` clause |

**Maximum 5 verification attempts.** On each attempt:
1. Show the current errors
2. Explain your repair strategy
3. Apply the fix
4. Re-verify

### Step 4: Post-Generation Checks

After generating verified code, check for these patterns and warn:

| Detected Pattern | Alert |
|---|---|
| `real` type usage | "Dafny `real` compiles to `_dafny.BigRational` in Python—you may want to replace with native `float` (losing formal precision guarantees)." |
| `seq<char>` / string operations | "Go backend has string/seq ambiguity at runtime. Test string operations carefully in extracted code." |
| Identifiers with underscores | "Go: Dafny identifiers starting with `_` may conflict with Go's file-naming rules. Renaming recommended." |
| Generics / type parameters | "Go: generic type parameters compile via type erasure to `interface{}`. Type assertions may be needed in extracted code." |

### Step 5: Present the Result

If verification succeeds, present:
- The full verified Dafny program
- Summary of what was proven
- Any proof artifacts added (lemmas, ghost variables) with explanations

If all 5 attempts fail, present:
- The best version achieved
- Remaining verification errors with explanations
- Suggestions for simplifying the spec or implementation

## Arguments

Optionally, the Dafny spec to implement. If not provided, assumes the spec was established in the current conversation via `/spec-iterate`.

Example: `/generate-verified`
