# /compare-patches — Patch Equivalence Verification

## Description

Determine whether two code patches are semantically equivalent by tracing their execution through the test suite using semi-formal reasoning. Produces a structured proof of equivalence or a specific counterexample.

## Instructions

You are a patch equivalence verifier using semi-formal reasoning. The user will provide two patches (code diffs) that address the same problem. Your job is to determine whether they produce identical test outcomes — without executing the code.

### Step 1: Establish Definitions

```
DEFINITIONS:
D1: Two patches are EQUIVALENT MODULO TESTS iff executing the
    existing repository test suite produces identical pass/fail
    outcomes for both patches.
D2: The relevant tests are ONLY those in FAIL_TO_PASS and
    PASS_TO_PASS (the existing test suite in the repository).
```

### Step 2: State Premises

Read both patches and the relevant test files, then document:

```
PREMISES (state what each patch does):
P1: Patch 1 modifies [file(s)] by [specific change description]
P2: Patch 2 modifies [file(s)] by [specific change description]
P3: The FAIL_TO_PASS tests check [specific behavior being tested]
P4: The PASS_TO_PASS tests check [specific behavior, if relevant]
```

CRITICAL: Read the actual test implementations, don't guess from test names.

### Step 3: Analyze Test Behavior

For each relevant test, trace execution through BOTH patches:

```
ANALYSIS OF TEST BEHAVIOR:

For FAIL_TO_PASS test(s):
  Claim 1.1: With Patch 1 applied, test [name] will [PASS/FAIL]
          because [trace through the code behavior]
  Claim 1.2: With Patch 2 applied, test [name] will [PASS/FAIL]
          because [trace through the code behavior]
  Comparison: [SAME/DIFFERENT] outcome

For PASS_TO_PASS test(s) (if patches could affect them differently):
  Claim 2.1: With Patch 1 applied, test behavior is [description]
  Claim 2.2: With Patch 2 applied, test behavior is [description]
  Comparison: [SAME/DIFFERENT] outcome
```

For each claim, trace the actual execution path:
- Follow function calls to their definitions (don't assume from names)
- Check for name shadowing (local/module definitions that override builtins)
- Verify argument types match parameter expectations
- Note any side effects

### Step 4: Analyze Edge Cases

```
EDGE CASES RELEVANT TO EXISTING TESTS:
(Only analyze edge cases that the ACTUAL tests exercise)

E1: [Edge case that existing tests exercise]
  - Patch 1 behavior: [specific output/behavior]
  - Patch 2 behavior: [specific output/behavior]
  - Test outcome same: [YES/NO]
```

### Step 5: State Counterexample or Equivalence Proof

If NOT equivalent:

```
COUNTEREXAMPLE (required if claiming NOT EQUIVALENT):
Test [name] will [PASS/FAIL] with Patch 1 because [reason]
Test [name] will [FAIL/PASS] with Patch 2 because [reason]
Therefore patches produce DIFFERENT test outcomes.
```

If equivalent:

```
NO COUNTEREXAMPLE EXISTS (required if claiming EQUIVALENT):
All existing tests produce identical outcomes because [reason]
```

### Step 6: Formal Conclusion

```
FORMAL CONCLUSION:
By Definition D1:
- Test outcomes with Patch 1: [PASS/FAIL for each test]
- Test outcomes with Patch 2: [PASS/FAIL for each test]
- Since test outcomes are [IDENTICAL/DIFFERENT], patches are
  [EQUIVALENT/NOT EQUIVALENT] modulo the existing tests.

ANSWER: [YES/NO] (are the patches equivalent?)

CONFIDENCE: [HIGH/MEDIUM/LOW]
- HIGH: All execution paths fully traced, all tests analyzed
- MEDIUM: Most paths traced, some library behavior assumed
- LOW: Key paths rely on unverified assumptions
```

### Key Principles

- Name resolution is critical — always check for local definitions that shadow builtins (the paper's motivating example shows `format()` being shadowed by a module-level function)
- Trace ACTUAL execution paths, don't assume from function names
- Only analyze edge cases that existing tests actually exercise
- The counterexample must be specific: which test, which patch, what happens
- Surface-level code similarity is misleading — similar-looking patches can have different semantics

## Arguments

Two patches (as diffs or code blocks) and optionally the test file path.

Examples:
- `/compare-patches` (with two diffs in the conversation)
- `/compare-patches "Patch 1: ... Patch 2: ..." tests/test_feature.py`
