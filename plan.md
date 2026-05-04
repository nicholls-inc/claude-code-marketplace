# Plan: crosscheck: component-correct verification misses end-to-end integration gaps

## Verification track
semi-formal

## Steps

1. **Add integration validation phase to `/locate-fault`** (`crosscheck/skills/locate-fault/SKILL.md`)
   - After Phase 4 (Ranked Predictions), add new **Phase 5: Integration Validation**
   - Mandate cross-cutting checks: verify that traced code paths span all components mentioned in the test, flag if divergence analysis stopped at interface boundaries without crossing into callee implementations
   - Add checklist item: "[ ] Traced across at least N component boundaries (where N = number of components test exercises)"
   - Add to Step 6 (Verification Checklist): "[ ] Integration validation performed: traced beyond interface boundaries"

2. **Add integration validation phase to `/reason`** (`crosscheck/skills/reason/SKILL.md`)
   - After Step 4 (Alternative Hypothesis Check), add new **Step 4c: Integration Validation**
   - For questions spanning multiple functions/modules, mandate: "Document each interface crossing (caller → callee), verify assumptions about callee behavior by reading its implementation"
   - Add to Step 7 (Verification Checklist): "[ ] Integration validation: all interface crossings verified by reading callee implementations"
   - Add to "Deep analysis mode" guidance: integration validation is mandatory when tracing execution across 2+ files

3. **Update byfuglien Phase 4 validation** (`crosscheck/agents/byfuglien.md`)
   - In "Phase 4: Validate Output" → "For semi-formal reasoning output" section, add new bullet after "Alternative hypothesis check":
     - **Integration validation** — for multi-component analysis, verify that evidence gathering crossed interface boundaries; if analysis cites "caller calls X" but didn't read X's implementation, flag as incomplete
   - Add to rejection criteria: "If a claim about end-to-end behavior cites only interface-level code (function signatures, API contracts) without reading through-layer implementations, re-execute with explicit instructions to trace the full call chain"

## Tests / properties to add

- **Execution trace spanning test**: Create a contrived multi-file scenario (e.g., `test_multifile_bug.py` calling `moduleA.foo()` which calls `moduleB.bar()` where the bug lives). Run `/locate-fault` and verify Phase 2 reads both `moduleA.py` and `moduleB.py`, Phase 3 cites the bug in `moduleB`, and Phase 5 confirms trace crossed the A→B boundary.

- **Interface-only reasoning test**: Ask `/reason` a question answerable only by reading callee code (e.g., "Is this safe?" where caller looks safe but callee has preconditions). Verify Step 4c integration validation forces reading the callee and flags the issue.

## Verification approach

**Semi-formal reasoning** — execution trace analysis:

1. **Trace execution across test cases**: Run the newly added tests with `/locate-fault` and `/reason`. Capture the structured output (Phase/Step sections).

2. **Certificate inspection**: Parse the output for:
   - Presence of Phase 5 / Step 4c sections (structural completeness)
   - Evidence of reading multiple files (grep for "file:line" references spanning 2+ files)
   - Claims that reference code in callee implementations, not just caller sites

3. **Adequacy check via `/rationale`**: For the multi-file test scenario, run `/rationale` to build an adequacy argument. The claim tree should reveal:
   - Root claim: "The fault localization is complete"
   - Leaf claim: "All component boundaries in the test execution path were traced" [STATIC — verified by Phase 5 checklist]
   - Leaf claim: "Divergence analysis cited the actual buggy line in moduleB" [STATIC — verified by Phase 3 output]

If the `/rationale` tree holds (all leaves verified, root follows), the integration validation is adequate.

## Risk register

- **Risk**: New phases add overhead to simple single-file bugs where integration validation is trivial
  - **Mitigation**: Phase 5 / Step 4c explicitly state "Skip if analysis is single-file or single-function; integration validation only applies to multi-component traces"

- **Risk**: Users skip reading the skill instructions and don't perform the new phases
  - **Mitigation**: Byfuglien's Phase 4 validation actively checks for integration validation in the output certificate. If missing when expected, it forces re-execution with explicit instructions

- **Risk**: Integration validation could recurse indefinitely (every function calls another)
  - **Mitigation**: Document termination condition: "Trace until you reach code that (a) is demonstrably correct, (b) is a primitive/library call, or (c) is unrelated to the test's concerns"

- **Risk**: False positives — flagging interface boundaries that don't need tracing (e.g., well-known library calls)
  - **Mitigation**: Integration validation language says "interface boundaries *relevant to the question*" and "if premise depends on callee behavior, verify by reading callee"
