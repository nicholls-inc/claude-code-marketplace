---
name: lightweight-verify
description: >-
  Generate lightweight verification artifacts for functions where full formal
  verification is overkill: design-by-contract assertions, property-based tests,
  and documented runtime invariant checks. Use for simple transformations, CRUD,
  concurrency, or floating-point code that Dafny cannot verify. Triggers:
  "lightweight verify", "add contracts", "property-based tests", "assertions".
argument-hint: "[function description or code] [python|go]"
---

# /lightweight-verify — Lightweight Verification Alternatives

## Description

For functions where full formal verification is overkill, generate lightweight verification artifacts: design-by-contract assertions, property-based tests, and documented runtime invariant checks. This skill provides a pragmatic middle ground between no verification and full Dafny-backed formal verification.

## Instructions

You are a verification expert helping the user add lightweight correctness guarantees to their code. The user will describe a function or provide existing code. Your job is to generate verification artifacts appropriate to their needs.

### Step 1: Determine Target Language and Function

Identify the target language and function to verify:
- If the user specifies a language, use it (Python or Go)
- If Dafny source is provided, extract `ensures`/`requires` clauses as the specification
- If natural language is provided, extract implicit contracts from the description
- If code is provided without a language specification, infer the language from syntax

### Step 2: Extract Contracts

From the description, code, or Dafny spec, identify:
- **Preconditions**: What must be true of the inputs?
- **Postconditions**: What must be true of the output?
- **Invariants**: What properties must hold throughout execution?

Present these in a table:

| Contract | Type | Description |
|---|---|---|
| `items` is non-empty | Precondition | The input list must contain at least one element |
| Result is a member of `items` | Postcondition | The returned value exists in the original input |
| All examined elements are <= current max | Invariant | The running maximum is always >= all previously seen elements |

### Step 3: Choose Verification Strategy

Present three options and recommend one based on the use case:

#### Option A: Design-by-Contract (Lowest overhead)

**Best for:** internal functions, prototyping, simple transformations, scripts.

Generate the function with embedded `assert` statements for pre/postconditions.

**Python example:**
```python
def max_of_list(items: list[int]) -> int:
    # Preconditions
    assert len(items) > 0, "items must be non-empty"

    result = items[0]
    for x in items[1:]:
        if x > result:
            result = x

    # Postconditions
    assert result in items, "result must be a member of items"
    assert all(x <= result for x in items), "result must be >= all elements"
    return result
```

**Go example:**
```go
func MaxOfSlice(items []int) int {
    // Preconditions
    if len(items) == 0 {
        panic("items must be non-empty")
    }

    result := items[0]
    for _, x := range items[1:] {
        if x > result {
            result = x
        }
    }

    // Postconditions — debug builds only
    for _, x := range items {
        if x > result {
            panic("result must be >= all elements")
        }
    }
    return result
}
```

#### Option B: Property-Based Testing (Medium overhead)

**Best for:** pure functions, data transformations, algorithms with clear invariants.

Generate property-based tests using Hypothesis (Python) or rapid (Go) that exercise the contracts across many random inputs.

**Python example (Hypothesis):**
```python
from hypothesis import given, strategies as st

@given(st.lists(st.integers(), min_size=1))
def test_max_is_member(items):
    result = max_of_list(items)
    assert result in items

@given(st.lists(st.integers(), min_size=1))
def test_max_is_upper_bound(items):
    result = max_of_list(items)
    assert all(x <= result for x in items)
```

**Go example (rapid):**
```go
func TestMaxIsUpperBound(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        items := rapid.SliceOfN(rapid.Int(), 1, 100).Draw(t, "items")
        result := MaxOfSlice(items)
        for _, x := range items {
            if x > result {
                t.Fatalf("found element %d greater than result %d", x, result)
            }
        }
    })
}
```

#### Option C: Documented Invariants with Runtime Checks (Medium-high overhead)

**Best for:** long-lived production code, complex state transitions, code that others will maintain.

Generate a complete package with:
1. Docstring/comment documenting contracts formally
2. Runtime precondition checks (always on)
3. Postcondition checks gated behind a debug flag or environment variable
4. Companion test file with property-based tests

**Python example:**
```python
import os

_DEBUG_CONTRACTS = os.environ.get("DEBUG_CONTRACTS", "").lower() in ("1", "true")

def max_of_list(items: list[int]) -> int:
    """Return the maximum element of a non-empty integer list.

    Contracts:
        Requires: len(items) > 0
        Ensures: result in items
        Ensures: all(x <= result for x in items)
    """
    # Preconditions — always checked
    if not items:
        raise ValueError("items must be non-empty")

    result = items[0]
    for x in items[1:]:
        if x > result:
            result = x

    # Postconditions — checked in debug mode
    if _DEBUG_CONTRACTS:
        assert result in items, "postcondition violated: result not in items"
        assert all(x <= result for x in items), "postcondition violated: result not upper bound"

    return result
```

**Go example:**
```go
// MaxOfSlice returns the maximum element of a non-empty integer slice.
//
// Contracts:
//   Requires: len(items) > 0
//   Ensures: result is an element of items
//   Ensures: result >= every element in items
func MaxOfSlice(items []int) int {
    // Preconditions — always checked
    if len(items) == 0 {
        panic("MaxOfSlice: requires len(items) > 0")
    }

    result := items[0]
    for _, x := range items[1:] {
        if x > result {
            result = x
        }
    }

    // Postconditions — checked when DEBUG_CONTRACTS is set
    if os.Getenv("DEBUG_CONTRACTS") != "" {
        for _, x := range items {
            if x > result {
                panic("MaxOfSlice: postcondition violated: result not upper bound")
            }
        }
    }

    return result
}
```

### Step 4: Generate Artifacts

Based on the user's choice (or your recommendation if they defer), generate:

1. **Function with embedded contracts** — the annotated function ready to drop into their codebase
2. **Companion test file** (if Option B or C) — property-based tests covering all identified contracts
3. **Verification gap note** — a brief summary of what full formal verification would have additionally guaranteed:
   - Proofs hold for ALL inputs, not just tested samples
   - Termination is guaranteed
   - Absence of runtime errors (index out of bounds, integer overflow) is proven
   - Properties are checked statically at compile time, not at runtime

### Step 5: Verification Checklist

Present this checklist alongside the generated artifacts:

```
## Verification Checklist

- [ ] Contracts match intended behavior (review pre/postconditions table)
- [ ] Property-based tests cover edge cases (empty, boundary, negative)
- [ ] Runtime checks enabled in appropriate environments
- [ ] Properties that would benefit from full formal verification: [list candidates]
```

Fill in the bracketed items based on the contracts identified in Step 2.

### Step 6: Upgrade Path

Present guidance for upgrading to full formal verification later:

- Run `/spec-iterate` with the extracted contracts as a starting point
- Pre/postconditions translate directly to Dafny `requires`/`ensures` clauses
- Loop invariants map to Dafny `invariant` annotations
- Full formal verification additionally proves:
  - **Termination**: the function always completes (no infinite loops)
  - **Absence of runtime errors**: no out-of-bounds access, no division by zero
  - **Universal correctness**: properties hold for ALL possible inputs, not just those tested
  - **Static guarantees**: verification happens at build time, zero runtime cost

## Arguments

The user passes a function description or existing code, plus an optional target language.

Examples:
- `/lightweight-verify "function that returns the maximum element of a non-empty integer list" python`
- `/lightweight-verify "binary search on a sorted array returning the index or -1" go`
- `/lightweight-verify` (with Dafny spec or code already in the conversation context)
