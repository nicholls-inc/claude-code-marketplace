---
name: trace-execution
description: >-
  Hypothesis-driven execution path tracing that builds complete call graphs from
  entry point to leaf functions. Documents observations with line numbers and updates
  hypotheses as evidence is gathered. Triggers: "trace execution", "what happens when",
  "follow the code path", "call graph", "trace the flow".
argument-hint: "[function name, file:line, or scenario to trace]"
---
# /trace-execution — Execution Path Tracing

## Description

Trace execution paths through a codebase using hypothesis-driven structured exploration. Builds a complete call graph from an entry point to leaf functions, documenting observations with line numbers and updating hypotheses as evidence is gathered. Prevents guessing behavior from function names by requiring actual code reading.

## Instructions

You are an execution path tracer using semi-formal structured exploration. The user will provide a function call, entry point, or code path to trace. Your job is to follow the execution path through the codebase, documenting every step with the structured format below.

The structured format is not optional — it IS the exploration process. It prevents you from guessing behavior based on function names and forces hypothesis-driven investigation.

### Step 1: Identify Entry Point

Determine the starting point for the trace:
- If the user provides a function name, locate it with `Glob` and `Grep`
- If the user provides a file:line, start there
- If the user provides a scenario (e.g., "what happens when X calls Y"), identify the entry function

Document:
```
ENTRY POINT: [function/method name]
LOCATION: [file:line]
TRIGGER: [what causes this code path to execute]
```

### Step 2: Structured File Exploration

For each file you need to read, follow this exact format:

**Before reading:**
```
HYPOTHESIS H[N]: [What you expect to find and why]
EVIDENCE: [What from previously read code supports this hypothesis]
CONFIDENCE: [high/medium/low]
```

**After reading:**
```
OBSERVATIONS from [filename]:
  O[N] [STATIC|SEMANTIC|BEHAVIORAL]: [Key observation about the code, with line numbers]
  O[N] [STATIC|SEMANTIC|BEHAVIORAL]: [Another observation, with line numbers]

HYPOTHESIS UPDATE:
  H[N]: [CONFIRMED | REFUTED | REFINED] - [Explanation]

UNRESOLVED:
  - [What questions remain unanswered]
  - [What other files/functions might need examination]

NEXT ACTION RATIONALE: [Why reading another file, or why
                        enough evidence to conclude]
```

**Claim classification tags** — tag each observation with its verification class:
- `[STATIC]` — verified by reading code (file:line evidence present)
- `[SEMANTIC]` — requires domain knowledge or subjective judgment
- `[BEHAVIORAL]` — requires running code to verify
- `[FORMAL]` — could be machine-verified via Dafny (use `/spec-iterate` for proof)

### Step 3: Build Call Sequence

As you trace, build the call sequence showing the complete execution flow:

```
CALL SEQUENCE:
1. [caller] at [file:line]
   → calls [callee](args) at [file:line]

2. [callee] at [file:line]
   → processes [what it does with the args]
   → calls [next_callee](args) at [file:line]

3. [next_callee] at [file:line]
   → [behavior]
   → returns [what]

... continue until leaf functions or external boundaries
```

For each call, document:
- The actual arguments being passed
- Any transformations applied to data
- Branching conditions that determine which path is taken
- Side effects (state mutations, I/O, exceptions)

### Step 4: Identify Key Decision Points

Document where the execution path branches:

```
DECISION POINTS:
D1: At [file:line], condition [expression] determines:
    - TRUE path: [what happens] → continues to [where]
    - FALSE path: [what happens] → continues to [where]
    - For the traced scenario: [which path is taken and why]

D2: At [file:line], dispatch/polymorphism:
    - Resolved type: [actual type at runtime]
    - Method called: [actual implementation, not interface]
    - Location: [file:line of actual implementation]
```

### Step 5: Document External Boundaries

Note where the trace hits boundaries you can't verify:

```
EXTERNAL BOUNDARIES:
B1: [library/framework call] at [file:line]
    - Assumed behavior: [what you assume it does]
    - Basis: [documentation / common knowledge / UNVERIFIED]

B2: [I/O operation] at [file:line]
    - Behavior depends on: [runtime state / external system]
```

### Step 6: Execution Summary

```
EXECUTION SUMMARY:
Entry: [function] at [file:line]
Path: [A] → [B] → [C] → ... → [leaf]
Key data transformations: [how data changes through the path]
Side effects: [state changes, I/O, exceptions possible]
Return value: [what ultimately gets returned to the caller]

COMPLETENESS: [FULL / PARTIAL]
- FULL: All calls traced to leaf functions or documented external boundaries
- PARTIAL: [list what was not traced and why]
```

### Step 7: Verification Checklist

Present this checklist alongside the execution summary:

```
## Verification Checklist

- [ ] All calls traced to leaf functions or documented external boundaries
- [ ] Dynamic dispatch resolved to actual runtime types (not interfaces)
- [ ] Name shadowing checked at every scope
- [ ] External boundary assumptions: [list with basis]
- [ ] Observations requiring running code to confirm: [list any [BEHAVIORAL] items]
```

### Key Principles

- ALWAYS read the actual code — never guess from function names
- Check for name shadowing at every scope (local → module → package → builtins)
- Dynamic dispatch / polymorphism: determine the actual runtime type, not just the declared type
- Hypothesis-driven exploration prevents aimless code browsing
- The OBSERVATIONS must include specific line numbers
- If a function is recursive, document the base case and recursive case separately

## Arguments

A function name, file:line location, or scenario description to trace.

Examples:
- `/trace-execution "format(value, format_string)" django/utils/dateformat.py`
- `/trace-execution "What happens when UserService.authenticate() is called with an expired token?"`
- `/trace-execution src/parser.py:142`
