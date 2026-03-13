# /rationale — Structured Adequacy Argument

## Description

Build a hierarchical claim tree arguing that code adequately satisfies its requirements. Each leaf claim is classified by verification method — formal (Dafny), behavioral (tests), static (code reading), or semantic (human judgment) — creating a traceable checklist that bridges formal and informal verification.

## Instructions

You are a verification expert building a structured adequacy argument. The goal is to decompose the high-level claim "this code is adequate" into verifiable subclaims, attempt automatic verification where possible, and produce a traceable checklist showing what has been verified and what remains.

This skill bridges Crosscheck's formal verification (Dafny) and semi-formal reasoning (evidence certificates) into a unified argument structure.

### Step 1: Gather Requirements

Identify the code and its requirements:

**Code:** Read the function(s) or module the user wants to argue about. Understand the implementation thoroughly — read the actual code, not just signatures.

**Requirements:** Gather from the user's description and any available sources:
- Natural language description or user story
- Docstrings and type annotations
- Existing tests (what do they assert?)
- Acceptance criteria if provided
- Implicit requirements from calling context

If requirements are vague, ask the user to clarify before proceeding. A rationale built on unclear requirements is worse than no rationale.

### Step 2: Construct Claim Tree

Build a hierarchical argument tree rooted at "The code adequately satisfies the requirements."

**Root claim:** "Code `[function/module]` is adequate for `[requirements summary]`"

**Decomposition strategy — split into structural, behavioral, and non-functional subclaims:**

```
ROOT: Code is adequate for [requirements]
├── C1: Structural correctness
│   ├── C1.1: Output has correct type/shape
│   ├── C1.2: All required fields/values populated
│   └── C1.3: Data structure invariants maintained
├── C2: Behavioral correctness
│   ├── C2.1: Core algorithm produces correct results
│   ├── C2.2: Edge cases handled (empty input, boundary values, overflow)
│   ├── C2.3: Error conditions handled appropriately
│   └── C2.4: Business logic aligns with domain rules
└── C3: Non-functional adequacy
    ├── C3.1: Performance acceptable for expected inputs
    ├── C3.2: No resource leaks (memory, file handles, connections)
    └── C3.3: Error messages are actionable
```

Adapt the tree to the specific code. Not all branches apply to every function. Prune irrelevant branches and add domain-specific ones.

**Each claim must be:**
- Specific enough to verify (not "code works correctly")
- Independent of sibling claims where possible
- Traceable to a specific requirement

### Step 3: Classify Leaf Claims

Assign each leaf claim a verification strategy:

| Tag | Meaning | Verification Method |
|-----|---------|-------------------|
| `[FORMAL]` | Can be proven mathematically | Candidate Dafny spec → offer `/spec-iterate` |
| `[BEHAVIORAL]` | Can be tested by running code | Generate property-based tests or test cases |
| `[STATIC]` | Can be verified by reading code | Cite evidence at specific `file:line` locations |
| `[SEMANTIC]` | Requires human domain judgment | Add to human review checklist |

**Classification guidelines:**
- Quantified properties ("for all", "there exists", "is a permutation of") → `[FORMAL]`
- Invariant preservation (sorting, bounds, conservation laws) → `[FORMAL]`
- Edge case handling (empty input, null, overflow) → `[BEHAVIORAL]`
- Type correctness, field presence, structural shape → `[STATIC]`
- Business rule alignment, UX quality, naming conventions → `[SEMANTIC]`
- Performance under load → `[BEHAVIORAL]` (benchmark)
- Error message quality → `[SEMANTIC]`

### Step 4: Attempt Verification

For each leaf claim, attempt verification using the classified method:

**`[FORMAL]` claims:**
- Draft a candidate Dafny specification (preconditions + postconditions)
- Offer to call `dafny_verify` with the candidate spec
- If the user approves and verification passes, mark as verified
- If verification fails, note the failure and suggest `/spec-iterate` for iterative refinement

**`[BEHAVIORAL]` claims:**
- Generate concrete test cases or property-based test code
- Present the test code to the user
- Mark as "tests generated — run to verify"

**`[STATIC]` claims:**
- Read the relevant code and cite specific evidence
- Example: "C1.2 verified — all required fields set at `model.py:42-48`"
- Mark as verified with evidence

**`[SEMANTIC]` claims:**
- State what the user must judge
- Provide relevant code context to aid the judgment
- Mark as "human review required"

### Step 5: Present Traceable Checklist

The final output is a checklist where each item traces back through the tree to the root. The checklist is the deliverable — if all items pass, the root claim holds by construction.

```
## Rationale: [function/module] is adequate for [requirements]

### Claim Tree

ROOT: Code is adequate for [requirements summary]
├── C1: Structural correctness
│   ├── C1.1 [STATIC]: Output type is List[int]
│   └── C1.2 [STATIC]: Result length equals input length
├── C2: Behavioral correctness
│   ├── C2.1 [FORMAL]: Output is sorted in ascending order
│   ├── C2.2 [FORMAL]: Output is a permutation of input
│   ├── C2.3 [BEHAVIORAL]: Empty input returns empty output
│   └── C2.4 [BEHAVIORAL]: Single-element input returns unchanged
└── C3: Non-functional adequacy
    ├── C3.1 [BEHAVIORAL]: Handles lists up to 10,000 elements within 100ms
    └── C3.2 [SEMANTIC]: Function name and docstring accurately describe behavior

### Verification Results

- [x] C1.1 [STATIC]: Output type is `List[int]` — verified at `sort.py:15` (return type annotation)
- [x] C1.2 [STATIC]: Result length equals input length — verified at `sort.py:28` (no elements added/removed in loop)
- [x] C2.1 [FORMAL]: Output is sorted — verified via `dafny_verify` (spec: `ensures forall i :: 0 <= i < |result|-1 ==> result[i] <= result[i+1]`)
- [x] C2.2 [FORMAL]: Output is permutation of input — verified via `dafny_verify` (spec: `ensures multiset(result) == multiset(input)`)
- [ ] C2.3 [BEHAVIORAL]: Empty input → empty output — test generated, run `test_sort_empty()` to verify
- [ ] C2.4 [BEHAVIORAL]: Single-element → unchanged — test generated, run `test_sort_single()` to verify
- [ ] C3.1 [BEHAVIORAL]: Performance under 100ms for 10k elements — benchmark generated, run to verify
- [ ] C3.2 [SEMANTIC]: Function name and docstring accurate — **human review required**

### Summary

| Verification Method | Total | Verified | Pending |
|-------------------|-------|----------|---------|
| FORMAL | 2 | 2 | 0 |
| STATIC | 2 | 2 | 0 |
| BEHAVIORAL | 3 | 0 | 3 |
| SEMANTIC | 1 | 0 | 1 |

**If all pending items pass, the root claim holds by construction.**
```

### Step 6: Verification Checklist

```
## Verification Checklist

- [ ] All requirements have at least one corresponding leaf claim in the tree
- [ ] No leaf claim is left unclassified
- [ ] [FORMAL] claims have candidate Dafny specs (verified or offered for `/spec-iterate`)
- [ ] [BEHAVIORAL] claims have generated test code the user can run
- [ ] [STATIC] claims cite specific file:line evidence
- [ ] [SEMANTIC] claims clearly state what the user must judge
- [ ] The claim tree structure is sound — if all leaves hold, the root holds
- [ ] Trust boundaries noted (Dafny limitations, extern methods, float precision)
- [ ] Unaddressed requirements flagged as gaps rather than silently omitted
```

## Arguments

Code target and optional requirements description.

Examples:
- `/rationale src/sort.py "must return a sorted permutation of the input"`
- `/rationale billing/calc.py:42 "energy conservation: period1 + period2 == total"`
- `/rationale src/auth/ "JWT validation per RFC 7519"`
- `/rationale` — analyze the most recently discussed function
