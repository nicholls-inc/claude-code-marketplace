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

#### Interpret Difficulty Metrics

After each `dafny_verify` call, check the response for a `difficulty` field. If present, interpret the metrics:

- **Solver time**: If `solverTimeMs > 10000` (10s), flag as computationally expensive and suggest simplifying the spec or breaking into smaller lemmas
- **Resource count**: If `resourceCount > 500000`, warn about high resource usage and proof fragility across Dafny versions
- **Proof hints**: If `proofHintCount > 5`, note moderate/high proof complexity
- **Empty lemma bodies**: If `emptyLemmaBodyCount > 0`, flag that these may indicate trivially true properties — review needed
- **Trivial proof**: If `trivialProof` is true AND the spec has meaningful postconditions, note that property-based testing would have sufficed

If the `difficulty` field is absent (older server version), skip this section gracefully.

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
- A difficulty summary (if the `difficulty` field was present in the `dafny_verify` response):

**Proof Difficulty Summary:**
| Metric | Value | Assessment |
|--------|-------|------------|
| Solver time | {X}ms | Low (<2s) / Medium (2-10s) / High (>10s) |
| Resource count | {N} | Low (<100K) / Medium (100K-500K) / High (>500K) |
| Proof hints needed | {N} | Minimal (0) / Moderate (1-5) / Heavy (>5) |
| Empty lemma bodies | {N} | OK (0) / Review needed (>0) |
| Overall | Trivial/Moderate/Complex | — |

If overall assessment is Trivial, add the note: "Consider using `/lightweight-verify` for similar future functions."

If all 5 attempts fail, present:
- The best version achieved
- Remaining verification errors with explanations
- Suggestions for simplifying the spec or implementation

### Step 6: Verification Checklist

Present this checklist alongside the verified implementation:

```
## Verification Checklist

Before proceeding to extraction, verify:
- [ ] All postconditions are meaningful (no trivially-true ensures clauses)
- [ ] Proof complexity is acceptable (review difficulty summary table)
- [ ] Empty lemma bodies reviewed (if any flagged)
- [ ] Target-language pitfalls noted (`real` types, generics, underscore identifiers)
```

## Arguments

Optionally, the Dafny spec to implement. If not provided, assumes the spec was established in the current conversation via `/spec-iterate`.

Example: `/generate-verified`
