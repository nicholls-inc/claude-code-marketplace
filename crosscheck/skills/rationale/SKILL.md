---
name: rationale
description: >-
  Build a hierarchical claim tree arguing that code adequately satisfies its
  requirements. Each leaf claim is classified by verification method (formal,
  behavioral, static, semantic) and verified where possible. Bridges formal
  verification and semi-formal reasoning. Triggers: "build rationale",
  "is this code adequate", "adequacy argument", "verification coverage".
argument-hint: "[code path] [optional: requirements description]"
---

# /rationale — Structured Adequacy Argument

## Description

Build a hierarchical claim tree arguing that code adequately satisfies its requirements. Each leaf claim is classified by verification method — formal (Dafny), behavioral (tests), static (code reading), or semantic (human judgment) — creating a traceable checklist that bridges formal and informal verification.

## Instructions

You are a verification expert building a structured adequacy argument. The goal is to decompose the high-level claim "this code is adequate" into verifiable subclaims, attempt automatic verification where possible, and produce a traceable checklist showing what has been verified and what remains.

This skill bridges Crosscheck's formal verification (Dafny) and semi-formal reasoning (evidence certificates) into a unified argument structure.

**Persistence (per `crosscheck/docs/orchestrator-coordination.md` §3).** All artifacts produced by a `/rationale` run land at `.crosscheck/work/rationale/<YYYY-MM-DD-HHMMSS>-<short-slug>/`. The slug is derived from the target (`<function>` or `<module-path-slug>`). The directory contains: `claim-tree.md` (the deliverable), `tests/` (generated test files), and `verification-summary.md` (the per-leaf verdict table). Chat output references the directory and summarises; the directory contents are the artifact reviewers and orchestrators consume.

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

**Decomposition strategy — split into trust-boundary, structural, behavioral, and non-functional subclaims:**

```
ROOT: Code is adequate for [requirements]
├── C0: Trust boundaries
│   ├── C0.1: External dependencies enumerated (extern methods, IO, network, third-party libraries)
│   ├── C0.2: Domain limitations documented (float precision, concurrency, generic-type erasure)
│   └── C0.3: Trust assumptions stated (what is trusted to be correct without verification — comparison operators, allocator, OS primitives)
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

**Why C0 is first-class.** Trust boundaries bound what the rest of the tree can verify at all. A STATIC leaf claiming *"`sort.py:42` implements quicksort correctly"* is meaningless if the comparison function is `{:extern}` or the input arrives from a network call the agent never reads. Promoting trust boundaries to a top-level branch — rather than a footnote on the final checklist — makes them part of the argument structure: every downstream claim is conditional on the C0 leaves, and that conditioning is visible.

Adapt the tree to the specific code. Not all branches apply to every function. Prune irrelevant branches and add domain-specific ones.

**Each claim must be:**
- Specific enough to verify (not "code works correctly")
- Independent of sibling claims where possible
- Traceable to a specific requirement

### Step 3: Classify Leaf Claims

Assign each leaf claim a verification strategy:

| Tag | Meaning | Verification Method |
|-----|---------|-------------------|
| `[FORMAL]` | Can be proven mathematically | Layer 1 (pure code → ships): candidate Dafny spec → `/spec-iterate`. Layer 4 (effectful/networked/concurrent): candidate Lean spec → `/lean-spec` → `/lean-impl` → `/correspondence-review` → `/drt-oracle` |
| `[BEHAVIORAL]` | Can be tested by running code | Generate property-based tests or test cases |
| `[STATIC]` | Can be verified by reading code | Cite evidence at specific `file:line` locations |
| `[SEMANTIC]` | Requires human domain judgment | Add to human review checklist |

**Classification guidelines:**
- Quantified properties ("for all", "there exists", "is a permutation of") → `[FORMAL]`
- Invariant preservation (sorting, bounds, conservation laws) → `[FORMAL]`
- FORMAL routing — pure functional shape, no IO/network/concurrency/shipping floats → **Layer 1** (Dafny); effectful, networked, concurrent, or shipping-float implementation → **Layer 4** (Lean pipeline). See Step 4 for the discharge mechanics
- Edge case handling (empty input, null, overflow) → `[BEHAVIORAL]`
- Type correctness, field presence, structural shape → `[STATIC]`
- Business rule alignment, UX quality, naming conventions → `[SEMANTIC]`
- Performance under load → `[BEHAVIORAL]` (benchmark)
- Error message quality → `[SEMANTIC]`

### Step 4: Attempt Verification

For each leaf claim, attempt verification using the classified method:

**`[FORMAL]` claims — hand off to byfuglien.** The Layer 1 / Layer 4 routing decision and the discharge of Dafny or Lean pipeline steps belong to byfuglien (`crosscheck/agents/byfuglien.md`), which owns the implementation-chain pipelines end-to-end. This skill does **not** enumerate `/spec-iterate` / `/lean-spec` / `/lean-impl` / `/correspondence-review` / `/drt-oracle` as commands for the user to run.

For each `[FORMAL]` leaf:

1. Classify the leaf by purity profile and record the routing verdict in `claim-tree.md`:
   - **Layer 1 candidate** — pure functional shape, no IO/network/concurrency, not shipping floats. Pipeline chain owned by byfuglien starts at `/spec-iterate`.
   - **Layer 4 candidate** — effectful, networked, concurrent, or shipping floats. Pipeline chain owned by byfuglien starts at `/lean-spec`.
   - **Ambiguous** — record the leaf as `Routing: ambiguous (see <evidence>)` and let byfuglien (or the human reviewer) pick at pipeline-dispatch time.

2. Mark the leaf as `Pending byfuglien dispatch` with the routing verdict. Do not draft the Dafny or Lean stub here; byfuglien's pipeline draws those when the chain runs.

3. If the run is happening under an orchestrator (`add-orchestrator` marker present per `crosscheck/docs/orchestrator-coordination.md` §1), record the routing verdict in `claim-tree.md`'s frontmatter so the orchestrator can dispatch byfuglien directly. Otherwise, the chat summary names byfuglien as the next step; the user does not re-type pipeline-step commands.

**`[BEHAVIORAL]` claims:**

Write the generated test code to a file under `<run-dir>/tests/` (per the persistence convention introduced in the Description above). One file per leaf, named `test_<leaf-id>.<ext>` where `<ext>` matches the target language. Do **not** paste the test code inline as the primary output — the file IS the artifact.

- Record the file path in `claim-tree.md` as the verification evidence: `Evidence: tests/test_C2_3.py (generated)`.
- Mark the leaf as `Pending test execution` — an orchestrator or CI run executes the file; the user does not paste and run by hand.
- If the run is under an orchestrator and the orchestrator has a test-runner subagent, the orchestrator dispatches it on the new files. Otherwise the chat summary points at the written paths and notes "run via `pytest <run-dir>/tests/` / `go test ./...` / equivalent".

**`[STATIC]` claims:**
- Read the relevant code and cite specific evidence in `claim-tree.md`.
- Example: `Evidence: model.py:42-48 — all required fields set in constructor`.
- Mark as `Verified (static)` — these are closed-loop in the same run.

**`[SEMANTIC]` claims:**
- State what the human must judge.
- Record the surrounding code context (function path + line range) in `claim-tree.md` so the human has the evidence at PR-review time.
- Mark as `Pending human review` — these are the irreducible human-judgment leaves and are the legitimate governance surface for `/rationale`.

### Step 5: Emit Traceable Checklist Artifact

Write `<run-dir>/claim-tree.md` containing the claim tree, per-leaf classification + evidence, and the verdict summary table. The file is the deliverable. The chat output references the file path and summarises counts; it does not paste the full tree as primary output.

```
## Rationale: [function/module] is adequate for [requirements]

### Claim Tree

ROOT: Code is adequate for [requirements summary]
├── C0: Trust boundaries
│   ├── C0.1 [STATIC]: No extern methods, IO, or network calls in implementation
│   └── C0.2 [SEMANTIC]: Element comparison operator (`<=`) assumed total and transitive
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

- [x] C0.1 [STATIC]: No extern methods, IO, or network calls — verified at `sort.py:1-30` (pure function, imports limited to stdlib types)
- [ ] C0.2 [SEMANTIC]: `<=` operator totality and transitivity — **human review required** (downstream FORMAL claims condition on this)
- [x] C1.1 [STATIC]: Output type is `List[int]` — verified at `sort.py:15` (return type annotation)
- [x] C1.2 [STATIC]: Result length equals input length — verified at `sort.py:28` (no elements added/removed in loop)
- [ ] C2.1 [FORMAL]: Output is sorted — Routing: Layer 1 (pure functional). Pending byfuglien dispatch.
- [ ] C2.2 [FORMAL]: Output is permutation of input — Routing: Layer 1. Pending byfuglien dispatch.
- [ ] C2.3 [BEHAVIORAL]: Empty input → empty output — `<run-dir>/tests/test_C2_3.py` written. Pending test execution.
- [ ] C2.4 [BEHAVIORAL]: Single-element → unchanged — `<run-dir>/tests/test_C2_4.py` written. Pending test execution.
- [ ] C3.1 [BEHAVIORAL]: Performance under 100ms for 10k elements — `<run-dir>/tests/test_C3_1.py` written. Pending test execution.
- [ ] C3.2 [SEMANTIC]: Function name and docstring accurate — Pending human review at PR time. Evidence: `sort.py:1-30`.

### Summary

| Verification Method | Total | Verified | Pending |
|-------------------|-------|----------|---------|
| FORMAL | 2 | 2 | 0 |
| STATIC | 3 | 3 | 0 |
| BEHAVIORAL | 3 | 0 | 3 |
| SEMANTIC | 2 | 0 | 2 |

**If all pending items pass, the root claim holds by construction.**
```

### Step 6: Evidence Summary and Decisions for Review

Split the post-run handoff into two blocks. The Evidence Summary block lists what the agent verified during the run; the Decisions block lists the irreducible human-judgment items the human acts on at PR time.

```
## Evidence Summary (agent-verified during this run)

- All requirements mapped to at least one leaf claim in the tree (or flagged as a gap).
- No leaf claim left unclassified.
- C0 trust-boundary branch enumerated for this code (extern methods, IO, network, float precision, generic-type erasure, concurrency).
- [STATIC] leaves verified inline with file:line evidence — count: <N>.
- [FORMAL] leaves classified by Layer 1 / Layer 4 routing — count: <N> Layer 1, <M> Layer 4, <K> ambiguous. Pending byfuglien dispatch.
- [BEHAVIORAL] leaves materialised as test files under <run-dir>/tests/ — count: <N>. Pending test execution.
- Artifact written to <run-dir>/claim-tree.md.

## Decisions for Review (human owns these at PR time)

- [ ] [SEMANTIC] leaves: <N> claims require human domain judgment. See claim-tree.md for the per-leaf code context.
- [ ] [FORMAL] ambiguous-routing leaves: <K> claims could not be cleanly placed in Layer 1 or Layer 4. Reviewer (or byfuglien at dispatch time) selects the route.
- [ ] Unaddressed requirements: <N> requirements have no covering leaf claim. Decide whether to add a leaf, accept the gap with a documented reason, or revise the requirements.

If all Decisions resolve favorably (judgments approved, ambiguous routings selected, gaps addressed), and the Pending byfuglien dispatch + Pending test execution items run clean, the root claim holds by construction.
```

The Evidence Summary lists what the agent did. The Decisions block is the legitimate human-judgment surface. Mixing the two is the "blank checklist of the analysis" anti-pattern the rest of this refactor removes.

## Arguments

Code target and optional requirements description.

Examples:
- `/rationale src/sort.py "must return a sorted permutation of the input"`
- `/rationale billing/calc.py:42 "energy conservation: period1 + period2 == total"`
- `/rationale src/auth/ "JWT validation per RFC 7519"`
- `/rationale` — analyze the most recently discussed function
