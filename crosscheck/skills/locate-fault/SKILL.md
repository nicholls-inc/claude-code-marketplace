---
name: locate-fault
description: >-
  Locate the root cause of a failing test using 5-phase semi-formal reasoning:
  test semantics analysis, code path tracing, divergence analysis, ranked
  predictions, and integration validation. Traces from failure to root cause
  rather than fixating on crash sites. Integration validation ensures analysis
  covers full execution paths across component boundaries. Triggers: "locate fault",
  "find the bug", "why does this fail", "root cause", stack traces, test failures.
argument-hint: "[failing test name or error message] [optional: test file path]"
---
# /locate-fault — Fault Localization via Semi-Formal Reasoning

## Description

Locate the root cause of a failing test using semi-formal reasoning. Uses a 5-phase structured approach — test semantics analysis, code path tracing, divergence analysis, ranked predictions, and integration validation — to systematically trace from test failure to root cause rather than fixating on crash sites. Integration validation ensures that multi-component analysis doesn't stop at interface boundaries without verifying assumptions across layers.

## Instructions

You are a fault localization expert using semi-formal reasoning. The user will provide a failing test (name and/or code) and you must find the buggy code. The structured template prevents you from stopping at the crash site (a common failure mode) and forces you to trace back to the root cause.

CRITICAL: You must complete ALL PHASES. Do not skip any phase or jump to conclusions early.

### Step 1: Phase 1 — Test Semantics Analysis

Read the failing test and establish formal premises about what it expects:

```
## Phase 1: Test Semantics Analysis

- What does the failing test method do step by step?
- What are the explicit assertions / expected exceptions?
- What is the expected behavior vs. the observed failure mode?

State these as formal PREMISES (with classification tags):
  PREMISE T1 [STATIC]: The test calls X.method(args) and expects [behavior]
  PREMISE T2 [STATIC]: The test asserts [condition]
  PREMISE T3 [STATIC|BEHAVIORAL]: The test expects [return value / exception / state]

**Claim classification tags** — tag each premise and claim with its verification class:
- `[STATIC]` — verified by reading code (file:line evidence present)
- `[SEMANTIC]` — requires domain knowledge or subjective judgment
- `[BEHAVIORAL]` — requires running code to verify
- `[FORMAL]` — could be machine-verified via Dafny (use `/spec-iterate` for proof)
  ...
```

Read the ACTUAL test code — do not guess from the test name.

### Step 2: Phase 2 — Code Path Tracing

Trace the execution path from the test's entry point into production code. For EACH file you read, follow this structured format:

**Before reading a file:**
```
### When requesting a file:

HYPOTHESIS H[N]: [What you expect to find and why it may
                  contain the bug]
EVIDENCE: [What from the test or previously read files
           supports this hypothesis]
CONFIDENCE: [high/medium/low]
```

**After reading a file:**
```
### After reading a file:

OBSERVATIONS from [filename]:
  O[N]: [Key observation about the code, with line numbers]
  O[N]: [Another observation]

HYPOTHESIS UPDATE:
  H[M]: [CONFIRMED | REFUTED | REFINED] - [Explanation]

UNRESOLVED:
  - [What questions remain unanswered]
  - [What other files/functions might need examination]

NEXT ACTION RATIONALE: [Why reading another file, or why
                        enough evidence to predict]
```

For each significant method call, document:
```
METHOD: ClassName.methodName(params)
LOCATION: file:line
BEHAVIOR: what this method does
RELEVANT: why it matters to the test
```

Build a call sequence showing the flow from test to production code.

### Step 3: Phase 3 — Divergence Analysis

For each code path traced, identify where the implementation could diverge from the test's expectations:

```
## Phase 3: Divergence Analysis

For each code path traced, identify where the implementation
could diverge from the test's expectations:

CLAIM D1 [STATIC|BEHAVIORAL]: At [file:line], [code] would produce [behavior]
         which contradicts PREMISE T[N] because [reason]
CLAIM D2 [STATIC|BEHAVIORAL]: At [file:line], [code] would produce [behavior]
         which contradicts PREMISE T[N] because [reason]
...
```

Rules:
- Each CLAIM must reference a specific PREMISE from Phase 1
- Each CLAIM must cite a specific code location from Phase 2
- This is the PREMISE -> CLAIM -> PREDICTION chain that makes the reasoning verifiable

### Step 4: Phase 4 — Ranked Predictions

Based on the divergence claims, produce ranked predictions:

```
## Phase 4: Ranked Predictions

Rank 1 ([high/medium/low] confidence): [file:lines]
  -- [description of the bug]
  -- Supports: CLAIM D[N]

Rank 2 ([high/medium/low] confidence): [file:lines]
  -- [description]
  -- Supports: CLAIM D[N]

Rank 3 ([high/medium/low] confidence): [file:lines]
  -- [description]
  -- Supports: CLAIM D[N]
```

Produce up to 5 ranked predictions. Each MUST cite the supporting CLAIM(s).

### Step 5: Phase 5 — Integration Validation

After ranking predictions, validate that the analysis covered the full execution path across component boundaries. **Skip if analysis is single-file or single-function; integration validation only applies to multi-component traces.**

**Component boundary definition**: A boundary is crossed when execution moves from one file to another via function call, method invocation, or module import followed by invocation. Intra-file function calls are not boundaries.

```
## Phase 5: Integration Validation

Multi-component analysis: [YES / NO]
  -- (YES if the test execution path crosses 2+ files; NO if single-file)

If YES, perform integration validation:

COMPONENT BOUNDARIES TRACED:
  -- Boundary 1: [caller_file:line] → [callee_file].[function] at [callee_file:line]
     Status: [CROSSED - read callee implementation | TRUST BOUNDARY - see below]
  -- Boundary 2: [caller_file:line] → [callee_file].[function] at [callee_file:line]
     Status: [CROSSED - read callee implementation | TRUST BOUNDARY - see below]
  ...

TRUST BOUNDARIES (if any):
  -- Trust boundary at [caller_file:line] → [callee_name]
     Assumption: [expected behavior of the callee]
     Reason: [library/extern/proprietary code - unreadable]

FILES READ: [list all files examined in Phase 2]
FILES ON EXECUTION PATH: [list all files the test actually executes through]

COVERAGE CHECK:
  -- [ ] Files read ⊇ Files on execution path (or trust boundaries documented for unread files)
  -- [ ] Traced across at least N component boundaries (where N = count of distinct file-to-file calls on test execution path)
```

Rules:
- For each file-to-file function call in the test's execution path, verify you either: (a) read the callee's implementation in Phase 2, or (b) documented it as a trust boundary with explicit assumptions
- If validation reaches an unreadable callee (library, extern, proprietary), document the trust boundary with explicit assumptions about its expected behavior
- Count the component boundaries you actually traced and compare to the number present in the test execution path
- This phase ensures you didn't stop at interface boundaries without validating assumptions across layers

### Step 6: Summary

Present a concise summary:
- The most likely root cause (Rank 1) with a plain-English explanation
- Why the crash site (if different) is a symptom, not the cause
- Suggested fix direction

### Step 7: Verification Checklist

Present this checklist alongside the summary:

```
## Verification Checklist

- [ ] All phases completed (none skipped)
- [ ] Top-ranked prediction traces back through CLAIM -> PREMISE chain
- [ ] Crash site vs. root cause distinction made explicit
- [ ] Alternative fault locations considered: [list Rank 2+ predictions]
- [ ] Claims requiring running code to confirm: [list any [BEHAVIORAL] items]
- [ ] Integration validation performed: traced beyond interface boundaries, or documented trust boundary with callee assumptions
```

### Key Principles

- The crash site is often NOT the root cause — trace backwards
- Phase 2's structured exploration prevents pattern-matching on function names
- The PREMISE -> CLAIM -> PREDICTION chain ensures every prediction is grounded in evidence
- Indirection bugs (test calls A, bug is in B that A calls) are the hardest — the template's call tracing catches these
- Multi-file bugs require following the call chain across files, not stopping at the first file
- Always form a HYPOTHESIS before reading each file — this prevents aimless exploration
- Phase 5 integration validation prevents stopping at interface boundaries without verifying cross-component assumptions

## Arguments

The failing test name and/or test code, plus optional context about the failure.

Examples:
- `/locate-fault "test_year_before_1000 fails with AttributeError" tests/test_dateformat.py`
- `/locate-fault` (with test failure output in the conversation)
- `/locate-fault "TestTypeResolution.typeVariable_of_self_type causes StackOverflowError"`