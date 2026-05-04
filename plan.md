# Plan: crosscheck: component-correct verification misses end-to-end integration gaps

## Verification track
semi-formal

## Steps

1. **Add integration validation phase to `/locate-fault`** (`crosscheck/skills/locate-fault/SKILL.md`)
   - After Phase 4 (Ranked Predictions), add new **Phase 5: Integration Validation**
   - **Component boundary definition**: A boundary is crossed when execution moves from one file to another via function call, method invocation, or module import followed by invocation. Intra-file function calls are not boundaries.
   - Mandate cross-cutting checks: verify that traced code paths span all files on the test execution path; flag if divergence analysis stopped at interface boundaries without crossing into callee implementations
   - Add checklist item: "[ ] Traced across at least N component boundaries (where N = count of distinct file-to-file calls on test execution path)"
   - Add to Step 6 (Verification Checklist): "[ ] Integration validation performed: traced beyond interface boundaries, or documented trust boundary with callee assumptions"
   - **Trust boundary documentation**: If validation reaches an unreadable callee (library, extern, proprietary), output must state: "Trust boundary at <caller>:<line> → <callee>. Assumption: <expected behavior>."

2. **Add integration validation phase to `/reason`** (`crosscheck/skills/reason/SKILL.md`)
   - After Step 4 (Alternative Hypothesis Check), add new **Step 4c: Integration Validation**
   - For questions spanning multiple functions/modules, mandate: "Document each interface crossing (caller → callee), verify assumptions about callee behavior by reading its implementation"
   - **Termination integrated with 2+ file rule**: When tracing execution across 2+ files, integration validation is mandatory. Stop tracing when reaching: (a) demonstrably correct code, (b) library/primitive, or (c) code unrelated to the question. At stop points (b), document trust boundary: "Trust boundary at <caller>:<line> → <library_function>. Assumption: <expected behavior>."
   - Add to Step 7 (Verification Checklist): "[ ] Integration validation: all interface crossings verified by reading callee implementations OR documented trust boundaries"
   - Add to "Deep analysis mode" guidance: integration validation is mandatory when tracing execution across 2+ files, continuing until termination condition is met

3. **Update byfuglien Phase 4 validation** (`crosscheck/agents/byfuglien.md`)
   - In "Phase 4: Validate Output" → "For semi-formal reasoning output" section, add new bullet after "Alternative hypothesis check":
     - **Integration validation** — for multi-component analysis, verify that evidence gathering crossed interface boundaries; if analysis cites "caller calls X" but didn't read X's implementation, flag as incomplete unless trust boundary is documented with explicit assumptions
   - Add to rejection criteria: "If a claim about end-to-end behavior cites only interface-level code (function signatures, API contracts) without reading through-layer implementations or documenting trust boundaries, re-execute with explicit instructions to trace the full call chain"

## Tests / properties to add

- **Execution trace spanning test**: Create a contrived multi-file scenario:
  - `test_multifile_bug.py` calls `moduleA.validate_input(x)` which calls `moduleB.process(x)`
  - Bug: `moduleA` assumes `process` accepts any integer, but `moduleB.process` has undocumented precondition `x > 0` and crashes on `x=0`
  - Test passes `x=0`, triggering failure in `moduleB` despite `moduleA` looking "correct"
  - Run `/locate-fault` and verify: Phase 2 reads both `moduleA.py` and `moduleB.py`, Phase 3 cites the precondition violation in `moduleB`, and Phase 5 confirms trace crossed the A→B boundary and identified the interface assumption mismatch

- **Interface-only reasoning test**: Ask `/reason` "Is `caller.divide_safe(user_input)` safe?" where:
  - `caller.py`: `def divide_safe(x): return utils.divide_by(x, x)`
  - `utils.py`: `def divide_by(a, b): return a / b  # requires b != 0`
  - Caller passes `x` as both numerator and denominator; if `x=0`, violates callee's precondition
  - Verify Step 4c integration validation forces reading `utils.py` and flags the `x=0` case

## Verification approach

**Semi-formal reasoning** — execution trace analysis:

1. **Trace execution across test cases**: Run the newly added tests with `/locate-fault` and `/reason`. Capture the structured output (Phase/Step sections).

2. **Certificate inspection**: Parse the output for:
   - Presence of Phase 5 / Step 4c sections (structural completeness)
   - Evidence of reading multiple files (grep for "file:line" references spanning 2+ files)
   - Claims that reference code in callee implementations, not just caller sites
   - **Invariant 1 check**: For the multi-file test, extract set of files read during analysis. Verify files-read ⊇ {files on actual test execution path}. If test executes A→B→C but analysis only read A and B, fail.
   - **Invariant 2 check**: For any unreadable callee encountered, verify output contains "Trust boundary at <location> → <callee>. Assumption: <behavior>." If library call crossed but no trust boundary documented, fail.

3. **Adequacy check via `/rationale`**: For the multi-file test scenario, run `/rationale` to build an adequacy argument. The claim tree should reveal:
   - Root claim: "The fault localization is complete"
   - Leaf claim: "All component boundaries in the test execution path were traced OR documented as trust boundaries" [STATIC — verified by Phase 5 checklist and Invariant 2]
   - Leaf claim: "Divergence analysis cited the actual buggy line in moduleB" [STATIC — verified by Phase 3 output]
   - Leaf claim: "Files read during analysis cover the execution path" [STATIC — verified by Invariant 1]

If the `/rationale` tree holds (all leaves verified, root follows), the integration validation is adequate.

## Risk register

- **Risk**: New phases add overhead to simple single-file bugs where integration validation is trivial
  - **Mitigation**: Phase 5 / Step 4c explicitly state "Skip if analysis is single-file or single-function; integration validation only applies to multi-component traces"

- **Risk**: Users skip reading the skill instructions and don't perform the new phases
  - **Mitigation**: Byfuglien's Phase 4 validation actively checks for integration validation in the output certificate. If missing when expected, it forces re-execution with explicit instructions

- **Risk**: Integration validation could recurse indefinitely (every function calls another)
  - **Mitigation**: Termination conditions integrated into mandatory validation (see Step 2): stop at (a) demonstrably correct code, (b) library/primitive, or (c) unrelated code. At stop point (b), document trust boundary with explicit assumptions about callee behavior

- **Risk**: False positives — flagging interface boundaries that don't need tracing (e.g., well-known library calls)
  - **Mitigation**: Integration validation language says "interface boundaries *relevant to the question*" and "if premise depends on callee behavior, verify by reading callee OR document trust boundary with assumptions"
