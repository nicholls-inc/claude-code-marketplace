---
name: suggest-specs
description: >-
  Analyze code to propose candidate formal specifications. Identifies functions that
  would benefit from verification and generates natural-language preconditions,
  postconditions, and invariants. Lowers the barrier to entry for /spec-iterate.
  Triggers: "suggest specs", "what should I verify", "find verification targets",
  "propose specifications".
argument-hint: "[optional: file path, function name, or directory]"
---

# /suggest-specs — Specification Discovery

## Description

Analyze code to propose candidate formal specifications. Identifies functions that would benefit from verification and generates natural-language preconditions, postconditions, and invariants — lowering the barrier to entry for `/spec-iterate`.

## Instructions

You are a formal verification expert helping users discover what is worth verifying. Most developers don't know where to start with formal specs. This skill bridges that gap by reading code and proposing specifications the user can approve, edit, or reject before entering the verification pipeline.

### Step 1: Identify Targets

Determine what code to analyze based on the user's input:

**If the user specifies a function or file:**
- Read the specified function(s) and surrounding context

**If the user specifies a module or directory:**
- Scan for functions that match high-value indicators (see Step 2)

**If no target specified:**
- Check recent git changes: `git diff --name-only HEAD~5..HEAD`
- Focus on recently modified functions as the most likely candidates

Read each target function's:
- Signature, type hints, docstring
- Function body (logic, control flow, assertions)
- Call sites (how is this function used? what assumptions do callers make?)
- Existing tests (what properties are already tested?)

### Step 2: Assess Verification Value

For each function, assess whether formal verification adds value over simpler approaches. Use these indicators:

**High-value indicators (recommend `/spec-iterate`):**
- Quantified properties: "for all elements", "there exists", permutation preservation
- Non-obvious invariants: loop invariants, state machine transitions, accumulator correctness
- Safety-critical logic: access control, financial calculations, crypto operations
- Subtle edge cases: off-by-one risks, empty input handling, overflow potential
- Complex data structure operations: tree balancing, graph traversal, priority queues

**Medium-value indicators (recommend `/lightweight-verify`):**
- Simple transformations with clear contracts (map, filter, format)
- Functions with good existing test coverage but no formal properties
- Business logic with well-defined rules but no safety criticality

**Low-value indicators (recommend skipping):**
- Pure CRUD operations
- Thin wrappers or delegation functions
- IO-heavy functions (cannot be formally verified)
- Concurrency coordination (Dafny limitation)

### Step 3: Propose Specifications

For each high-value and medium-value function, propose candidate specs in natural language:

**Preconditions** — infer from:
- Input validation or guard clauses in the function body
- Type hints and their constraints
- Assertions or `raise` statements
- Assumptions made by callers

**Postconditions** — infer from:
- Return statements and what they guarantee
- Docstring promises ("returns the sorted...", "finds the maximum...")
- Test assertions (what do tests check about the output?)
- Mathematical properties (idempotency, monotonicity, associativity)

**Loop invariants** — infer from:
- Loop patterns: accumulation, search, partitioning
- Variables modified in the loop and their relationships
- The gap between the loop's "obvious" behavior and what it actually maintains

**Implicit invariants** — flag non-obvious properties:
- "This function is called in a loop — should the accumulated result satisfy X?"
- "The caller assumes the output is sorted, but nothing enforces this"
- "This modifies shared state — should the state transition be monotonic?"

### Step 4: Present Proposals

Present proposals in a structured table:

```
## Specification Proposals

| # | Function | Location | Proposed Spec | Value | Recommended Skill |
|---|----------|----------|--------------|-------|-------------------|
| 1 | split_energy() | billing/calc.py:42 | `period1 + period2 == total` (energy conservation) | HIGH | `/spec-iterate` |
| 2 | merge_intervals() | utils/intervals.py:15 | Output intervals are non-overlapping and cover all input intervals | HIGH | `/spec-iterate` |
| 3 | validate_token() | auth/tokens.py:88 | Returns true iff token is well-formed and not expired | MEDIUM | `/lightweight-verify` |
| 4 | format_date() | display/format.py:12 | Output matches ISO 8601 pattern | LOW | Skip |
```

For each HIGH-value proposal, expand the spec:

```
### Proposal 1: split_energy()

**Location:** billing/calc.py:42
**Current behavior:** Splits a total energy value into two billing periods based on a date ratio

**Proposed preconditions:**
- `total >= 0` (energy cannot be negative)
- `0.0 <= ratio <= 1.0` (ratio represents a fraction of the billing period)

**Proposed postconditions:**
- `period1 + period2 == total` (energy conservation — no energy created or lost)
- `period1 == total * ratio` (proportional split)
- `period1 >= 0 and period2 >= 0` (non-negative outputs)

**Inferred from:**
- Docstring: "Split total energy proportionally across two periods"
- Test at tests/test_billing.py:67: `assert p1 + p2 == total`
- No existing test for negative inputs or ratio boundaries

**Trust boundary note:** Uses floating-point arithmetic — Dafny `real` type provides exact rational arithmetic, so the formal spec will be stronger than the runtime behavior. Add epsilon-tolerance property-based tests after extraction.
```

### Step 5: User Selection

Ask the user which proposals to act on. For each selected proposal:

- **HIGH-value** → "Ready to formalize. Run `/spec-iterate` with this proposal as the starting point."
- **MEDIUM-value** → "Run `/lightweight-verify` to generate design-by-contract assertions and property-based tests."
- **Declined** → Record the decision. The user may revisit later.

If the user selects multiple proposals, process them sequentially, starting with the highest-value ones.

### Step 6: Report

```
## Verification Checklist

- [ ] All high-value functions have been assessed for formal verification fitness
- [ ] Proposed specs accurately reflect the function's documented and tested behavior
- [ ] Implicit invariants (caller assumptions, loop properties) have been surfaced
- [ ] Trust boundary notes included where Dafny limitations apply (IO, concurrency, floats)
- [ ] Declined proposals recorded with reasoning for future review
- [ ] Selected proposals have a clear next step (`/spec-iterate` or `/lightweight-verify`)
```

## Arguments

Target function, file, module, or empty for recent changes.

Examples:
- `/suggest-specs src/billing/calc.py` — analyze all functions in a file
- `/suggest-specs merge_intervals` — analyze a specific function
- `/suggest-specs src/` — scan a directory for high-value targets
- `/suggest-specs` — analyze recent git changes
