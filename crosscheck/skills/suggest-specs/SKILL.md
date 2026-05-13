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

### Step 4: Emit Proposal Queue Artifact

Proposals are an artifact for orchestrator or human review, not a chat dispatch list. Write the structured queue to `.assurance/suggest-specs-queue.json` (creating `.assurance/` if missing). The file is the deliverable; the chat table below is a summary, not the primary output.

Schema (see also `crosscheck/docs/orchestrator-coordination.md` §2 on findings-as-artifacts):

```json
{
  "schema_version": 1,
  "generated_at": "<YYYY-MM-DDTHH:MM:SSZ>",
  "scope": "<file path | function name | directory | recent-changes>",
  "total_proposals": <n>,
  "proposals": [
    {
      "id": "P1",
      "function": "split_energy",
      "location": "billing/calc.py:42",
      "value_tier": "HIGH | MEDIUM | LOW",
      "queued_for": "/spec-iterate | /lightweight-verify | skip",
      "status": "proposed",
      "summary": "<one-line spec summary>",
      "preconditions": ["<bullet>", ...],
      "postconditions": ["<bullet>", ...],
      "loop_invariants": ["<bullet>", ...],
      "implicit_invariants": ["<bullet>", ...],
      "inferred_from": ["<docstring | test path | call-site evidence>", ...],
      "trust_boundary_notes": "<floats | IO | concurrency caveats, or null>"
    }
  ]
}
```

`status: "proposed"` is the only initial value. Downstream (orchestrator triage or human review via PR) transitions it to `approved`, `rejected`, or `processed` as the proposal is acted on.

Also emit a chat-readable summary table (this is the human-facing recap; the JSON is the artifact):

```
## Specification Proposals (queued at .assurance/suggest-specs-queue.json)

| ID | Function | Location | Proposed Spec | Value | Queued for |
|---|----------|----------|---------------|-------|------------|
| P1 | split_energy() | billing/calc.py:42 | `period1 + period2 == total` (energy conservation) | HIGH | `/spec-iterate` |
| P2 | merge_intervals() | utils/intervals.py:15 | Output intervals are non-overlapping and cover all input intervals | HIGH | `/spec-iterate` |
| P3 | validate_token() | auth/tokens.py:88 | Returns true iff token is well-formed and not expired | MEDIUM | `/lightweight-verify` |
| P4 | format_date() | display/format.py:12 | Output matches ISO 8601 pattern | LOW | skip |
```

The "Queued for" column reflects the routing decision; it is **not** an instruction to the user to invoke that skill. An orchestrator driving the verification chain reads the JSON, takes proposals with `value_tier in {HIGH, MEDIUM}` and `status: "proposed"`, and dispatches `/spec-iterate` or `/lightweight-verify` itself. The user's role is to review the queue (typically in a PR alongside the code change that prompted the run) and approve/reject entries by editing their `status` field — not to retype skill invocations.

### Step 5: Hand Off

Report the queue location and the high-level distribution:

```
Wrote .assurance/suggest-specs-queue.json
- HIGH-value proposals (→ /spec-iterate): <N>
- MEDIUM-value proposals (→ /lightweight-verify): <N>
- LOW-value proposals (→ skip): <N>

Review the queue (typically by editing status fields to approved/rejected). An orchestrator driving the verification chain consumes the JSON directly; the user does not need to invoke /spec-iterate or /lightweight-verify by hand for each proposal.
```

Do **not** ask the user which proposals to act on, and do **not** auto-invoke `/spec-iterate` or `/lightweight-verify` from this skill. Auto-invocation would just punt the elicitation problem one skill downstream (both target skills have their own decision points). The clean separation: this skill proposes, the orchestrator dispatches, the human reviews via the queue file.

### Step 6: Verification Checklist

```
## Verification Checklist

- [ ] All high-value functions have been assessed for formal verification fitness
- [ ] Proposed specs accurately reflect the function's documented and tested behavior
- [ ] Implicit invariants (caller assumptions, loop properties) have been surfaced
- [ ] Trust boundary notes included where Dafny limitations apply (IO, concurrency, floats)
- [ ] `.assurance/suggest-specs-queue.json` written with every field populated per the schema
- [ ] Chat summary references the queue file location; no "now run /spec-iterate" instructions aimed at the user
```

## Arguments

Target function, file, module, or empty for recent changes.

Examples:
- `/suggest-specs src/billing/calc.py` — analyze all functions in a file
- `/suggest-specs merge_intervals` — analyze a specific function
- `/suggest-specs src/` — scan a directory for high-value targets
- `/suggest-specs` — analyze recent git changes
