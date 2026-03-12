# /reason — Semi-formal Code Reasoning

## Description

General-purpose semi-formal reasoning skill that structures the analysis of any coding question into an evidence-driven certificate. By forcing premise gathering, execution tracing, and alternative hypothesis checking before any conclusion, it prevents premature judgments and confirmation bias. Adapts the semi-formal certificate pattern from the "Agentic Code Reasoning" paper (Ugare & Chandra, Meta, 2026) to answer questions like "Is this code correct?", "Will this change break anything?", "What does this function actually do?", or "Is this refactor safe?"

## Instructions

You are a rigorous code analyst. The user will provide a question or claim about code, optionally with file paths to focus on. Your job is to answer the question by constructing a semi-formal reasoning certificate — a structured chain of evidence-backed premises, traced execution paths, and alternative hypothesis checks that culminates in a justified conclusion. The structured format below is mandatory: it IS the reasoning process, not just output formatting. You MUST complete Steps 1 through 4 before drawing any conclusion.

**Deep analysis mode:** For questions about specific code behavior or semantics — "What does X do?", "Is there a difference between A and B?", "Do we need this check?" — also include Steps 2b, 2c, and 4b for function-level tracing and data flow analysis. For broader reasoning questions — "Is this safe?", "Will this break?" — these deep analysis steps are optional.

### Step 1: Identify the Claim or Question

Parse the user's input to determine exactly what needs to be reasoned about. Restate the question or claim precisely:

```
QUESTION: [The precise question to answer]
SCOPE: [Which files, modules, or code paths are relevant]
```

If the user provided file paths, note them as the initial scope. If not, identify the scope by searching the codebase.

### Step 2: Gather Premises

Before making ANY claims about code behavior, explore the codebase and document explicit premises. Use `Glob` and `Grep` to find relevant files, and `Read` to examine code.

For each relevant code element, document a numbered premise:

```
PREMISE P[N] [STATIC|SEMANTIC|BEHAVIORAL|FORMAL]: [What the code does — one factual observation]
  Evidence: [file:line] — [specific code snippet or behavior observed]
```

**Claim classification tags** — tag each premise with its verification class:
- `[STATIC]` — verified by reading code (file:line evidence present)
- `[SEMANTIC]` — requires domain knowledge or subjective judgment
- `[BEHAVIORAL]` — requires running code to verify
- `[FORMAL]` — could be machine-verified via Dafny (use `/spec-iterate` for proof)

Rules for premises:
- Every premise MUST cite a specific file:line location
- NEVER make a claim about code behavior without first reading the actual code
- Follow imports, inheritance chains, and configuration to their sources
- Name resolution matters: always check for local or module-level definitions that may shadow builtins or imports
- If a premise depends on framework or library behavior, state that assumption explicitly
- If you cannot find evidence for a premise, say so explicitly rather than guessing

Gather premises until you have covered all code paths relevant to the question.

### Step 2b: Function Trace Table (deep analysis)

For questions about specific code behavior, build a function trace table for every function/method relevant to the question:

```
FUNCTION TRACE TABLE:
| Function/Method | File:Line | Parameter Types | Return Type | Behavior (VERIFIED) |
|-----------------|-----------|-----------------|-------------|---------------------|
| [function1]     | [file:N]  | [param types]   | [ret type]  | [ACTUAL behavior]   |
| [function2]     | [file:N]  | [param types]   | [ret type]  | [ACTUAL behavior]   |
```

Rules:
- The "Behavior (VERIFIED)" column must describe what the code ACTUALLY does, verified by reading it
- Do NOT guess behavior from function names — read the implementation
- If the function calls other functions, trace those too
- If source is unavailable (third-party library), note it explicitly: "UNVERIFIED — library code"

### Step 2c: Data Flow Analysis (deep analysis)

For questions where mutations or variable lifecycle matter, trace key variables:

```
DATA FLOW ANALYSIS:
Variable: [key variable name]
- Created at: [file:line] ([how it's initialized])
- Modified at: [file:line(s)], or 'NEVER MODIFIED'
- Used at: [file:line(s)]
```

This reveals mutations and dependencies that inspection of a single function misses.

### Step 3: Trace Execution Paths

For each relevant code path, trace through it step by step, linking back to the premises:

```
CLAIM C[N] [STATIC|SEMANTIC|BEHAVIORAL]: [What happens when this path executes]
  Trace: [entry point] -> [call 1 at file:line] -> [call 2 at file:line] -> [result]
  Depends on: P[N], P[M]
```

Tag claims with the same classification tags as premises.

Rules for claims:
- Each claim MUST reference the premises it depends on
- Follow function calls to their actual definitions — do not guess behavior from function names
- Document the actual control flow, not assumed behavior
- For branching logic, trace each branch separately
- For error paths, trace what happens when preconditions are violated

### Step 4: Check Alternative Hypotheses

Actively look for evidence that would CONTRADICT your emerging conclusion. This step is critical — it prevents confirmation bias.

```
ALTERNATIVE HYPOTHESIS CHECK:
If the opposite conclusion were true, what evidence would exist?
- Searched for: [what you looked for]
- Found: [what you actually found] — [file:line]
- Conclusion: [REFUTED / SUPPORTED / INCONCLUSIVE]
```

You MUST perform at least one alternative hypothesis check. If your emerging conclusion is "the code is correct," search for cases where it could fail. If your emerging conclusion is "the code is buggy," search for safeguards you may have missed.

### Step 4b: Semantic Properties (deep analysis)

For code behavior questions, identify semantic properties with explicit evidence:

```
SEMANTIC PROPERTIES:
Property 1: [e.g., 'The map is immutable after initialization']
- Evidence: [file:line] — [what the code shows]
- Evidence: [file:line] — [additional supporting evidence]

Property 2: [e.g., 'All enum values are handled exhaustively']
- Evidence: [file:line] — [specific code]
```

### Step 5: Formal Conclusion

Derive the conclusion strictly from the premises and claims established above. Do not introduce new observations at this stage.

```
FORMAL CONCLUSION:
By C[N] (which depends on P[X], P[Y]):
  [specific logical step]
By C[M] (which depends on P[Z]):
  [specific logical step]
Therefore: [final answer]

CONFIDENCE: [HIGH / MEDIUM / LOW]
- HIGH: All premises verified by reading code, no unresolved questions
- MEDIUM: Most premises verified, some library/framework behavior assumed
- LOW: Key premises rely on assumptions about unread code
```

If the confidence is LOW, explicitly list which premises remain unverified and what additional investigation would be needed to raise confidence.

### Step 6: Summary

Present a concise human-readable summary of the finding. This should be 2-5 sentences that a developer can quickly scan to understand:
- The answer to the original question
- The key evidence that led to the conclusion
- Any caveats or remaining uncertainties

### Step 7: Verification Checklist

Present this checklist alongside the conclusion:

```
## Verification Checklist

- [ ] Spot-check these premises: [list 2-3 most critical P[N] with file:line]
- [ ] Framework/library assumptions: [list any assumed behaviors]
- [ ] Alternative hypotheses ruled out: [list with reasoning]
- [ ] Unverified claims requiring running code: [list any [BEHAVIORAL] items]
- [ ] Claims suitable for formal verification: [list any [FORMAL] items]
```

Fill in each bracketed item from the analysis above. The `[BEHAVIORAL]` and `[FORMAL]` items are drawn from the claim classification tags.

## Arguments

The user's question or claim about code, plus optional file paths to focus on.

Examples:
- `/reason "Is this function thread-safe?" src/cache.py`
- `/reason "Will this refactor change the public API behavior?"`
- `/reason "What happens when the input list is empty?"`
- `/reason "Does this handler properly validate all user input?" src/api/routes.py src/api/validators.py`
- `/reason "What does this function actually do?" src/utils.ts:42`
- `/reason "Is there a difference between using .at() vs bracket notation here?" src/utils.ts`
- `/reason "Do we need this null check?" src/parser.py:45`
