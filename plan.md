# Plan: Field Report: Crosscheck Plugin Performance in wistful-pet Session

## Verification track
semi-formal

## Steps

1. **Add "Verify the Fix" step to `/reason` skill** — `crosscheck/skills/reason/SKILL.md`
   - Insert new Step 5b between Step 5 (Formal Conclusion) and Step 6 (Summary)
   - Title: "Step 5b: Verify the Proposed Fix"
   - **Scope clarification** (addresses C3): Step 4 (Alternative Hypothesis Check) verifies the *diagnosis* of the bug by ruling out alternative root causes. Step 5b verifies the *prescription* (the fix) by checking if it introduces new bugs. Non-overlapping responsibilities.
   - Mandate checking if the proposed solution introduces new instances of the same bug pattern (race conditions, data integrity issues, side-effect bypasses)
   - Requires structured analysis: "Does this fix introduce: [RACE CONDITIONS] / [DATA INTEGRITY ISSUES] / [SIDE-EFFECT BYPASSES] / [OTHER BUGS]?"
   - **Complexity threshold** (addresses C5): If the proposed fix is >20 lines OR modifies >2 functions, Step 5b may be abbreviated to checklist-only analysis: "Does this fix class [RACE/INTEGRITY/BYPASS/OTHER]? [YES/NO/UNCLEAR]" without full execution trace re-derivation.
   - **Termination bound** (addresses C8): Step 5b depth ≤ 1. Single-pass checklist analysis only — no recursive `/reason` invocations for nested fix verification. If a fix is so complex it requires nested reasoning, flag as [UNCLEAR] and recommend breaking the fix into smaller increments.

2. **Add exhaustive reading instruction to `/trace-execution` skill** — `crosscheck/skills/trace-execution/SKILL.md`
   - Extend Step 2 (Structured File Exploration) with explicit exhaustive-read instruction
   - Add rule: "Read the ENTIRE function body, line by line. Pay specific attention to filter/guard clauses that constrain which inputs reach later stages."
   - Add to OBSERVATIONS template: "COVERAGE: [COMPLETE - lines 1-N read] / [PARTIAL - lines X-Y read, M-P skipped]"
   - **Coverage mechanics** (addresses C1a, C6): COVERAGE field must cite specific line ranges. COMPLETE requires continuous range from function start to end (verified by line count). PARTIAL explicitly lists skipped ranges. Agent determines coverage by tracking `read_file` calls with offset/limit and comparing to function definition boundaries from grep/glob results.
   - **Edge case rules** (addresses C4):
     - 0-line functions (abstract methods, pure delegation): COVERAGE = COMPLETE by default (nothing to skip)
     - 500+ line functions: Agent may declare strategic PARTIAL coverage with explicit reasoning: "COVERAGE: PARTIAL - lines 1-50, 200-230 read (entry + critical path); lines 51-199, 231-500 skipped (error handlers, logging)" — incompleteness is acceptable if declared
     - Interrupted reads (e.g., reads lines 1-50 of 100): COVERAGE = PARTIAL with exact range cited. Partial state persists in OBSERVATIONS output; agent does not retroactively upgrade to COMPLETE.

3. **Add confidence calibration to `/trace-execution` skill** — `crosscheck/skills/trace-execution/SKILL.md`
   - Extend Step 6 (Execution Summary) with confidence level similar to `/reason`
   - Add mandatory field: "CONFIDENCE: [HIGH / MEDIUM / LOW] — HIGH requires all functions in path read in entirety; MEDIUM if any function partially read; LOW if external boundaries dominate"
   - **Mechanical constraint** (addresses C7): "If COVERAGE for any function in the path shows PARTIAL, confidence MUST be MEDIUM or below. If any OBSERVATION is tagged [SEMANTIC] or [BEHAVIORAL] (vs [STATIC]), confidence MUST be MEDIUM or below. HIGH confidence requires: (all functions COVERAGE = COMPLETE) ∧ (all OBSERVATIONS are [STATIC], derived from code structure alone)."
   - This prevents HIGH confidence on semantically ambiguous traces even when coverage is complete.

4. **Update byfuglien's validation rules for `/trace-execution` output** — `crosscheck/agents/byfuglien.md`
   - In Phase 4 (Validate Output), under "For semi-formal reasoning output", add new bullet for `/trace-execution` specifically:
   - "**Completeness check** — if execution summary shows PARTIAL completeness, confidence must be MEDIUM or below; reject HIGH confidence claims from partial traces"
   - **Coverage verification** (addresses C1b, C6): Spot-check that entry point functions were read in entirety. Re-read the first function in the trace path, count lines via `wc -l`, compare to the line range cited in COVERAGE field. If mismatch detected (agent claimed COMPLETE but line count shows unread ranges), downgrade confidence to MEDIUM and append validation note: "[byfuglien: coverage incomplete, confidence adjusted]"
   - **False COMPLETE handling** (addresses C1c): If agent incorrectly reports COVERAGE = COMPLETE when lines were skipped, byfuglien's spot-check detects the mismatch and downgrades confidence. Agent does not re-run skill; output is annotated with correction.

## Tests / properties to add

No automated tests required — this is a refactor of behavioral artifacts (skill methodologies and agent validation rules). Verification is manual via session testing.

## Verification approach

**Semi-formal reasoning via `/compare-patches`** (deferred to verify_byfuglien stage):

1. **Structural integrity check**: Does each modified skill still have all required sections (Description, Instructions, Arguments)? Are step numbers sequential?

2. **Patch comparison anchor**: The field report identifies three critical failures:
   - W1: Missed filter in entry function → addressed by Step 2 (exhaustive read instruction + coverage tracking)
   - W2: Overconfident assertion on incomplete trace → addressed by Step 3 (confidence calibration with mechanical constraint)
   - W3: Proposed fix reintroduced TOCTOU → addressed by Step 1 (verify the fix step)

3. **Execution trace simulation**: Given the wistful-pet scenario (TOCTOU race in Django model's `enroll_user`), would the updated skills force the behaviors that were missing?
   - `/reason` Step 5b: Would it mandate checking if the proposed `leave() + create()` fix introduces a new race? **YES** — explicit check for race conditions in proposed fix.
   - `/trace-execution` Step 2 coverage rule: Would it prevent skimming the sync task entry function? **YES** — "read ENTIRE function body, line by line" is explicit; COVERAGE field requires line-range citation.
   - `/trace-execution` Step 6 confidence calibration: Would it prevent the "definitively possible" claim on a partial trace? **YES** — mechanical constraint ties confidence to coverage completeness AND observation type (STATIC vs SEMANTIC).

4. **Validation cascade check**: Does byfuglien's Phase 4 enforcement actually reject violations?
   - New rule: "reject HIGH confidence claims from partial traces" — would catch the W2 scenario.
   - New rule: "spot-check that entry point functions were read in entirety (re-read + line count comparison)" — would catch the W1 scenario.

5. **No regression to other skills**: `/reason` and `/trace-execution` are independent skills. Changes to one don't affect the other's execution, and both are routed independently by byfuglien. No cross-skill dependencies introduced.

**Note on C2 (VERIFICATION APPROACH IS SPECULATIVE)**: This concern is rejected. The verification track is semi-formal, meaning verification is simulation-based, not empirical. The plan stage's job is to prove the changes are *structurally sound* (correct placement, correct logic, addresses field report weaknesses). The verify stage's job is to prove they *behaviorally work* (actually prevents the failures in practice). Demanding empirical testing in the plan stage conflates the two. The simulation above (item 3) is appropriate for semi-formal verification: it traces through the updated skill logic and shows the mechanical constraints would fire in the wistful-pet scenario. That's sufficient to proceed to implementation.

## Risk register

- **Risk**: Adding Step 5b to `/reason` could make sessions too long if the proposed fix is complex
  - **Mitigation**: Step 5b abbreviated for fixes >20 lines or >2 functions (addresses C5). The verification is scoped to "does this fix introduce the same bug class?" — not a full re-analysis. Estimated 1-2 minutes added to 4-minute reasoning sessions for simple fixes; <30 seconds for complex fixes using checklist mode.

- **Risk**: Exhaustive read instruction in `/trace-execution` could cause agents to read hundreds of lines in large functions
  - **Mitigation**: The instruction is "read ENTIRE function body" for functions *in the trace path*, not all functions in the codebase. Execution traces typically involve 3-5 key functions. Also, large functions are a code smell — exhaustive reading pressure may surface refactoring opportunities. Edge case rule for 500+ line functions allows strategic PARTIAL coverage if declared explicitly (addresses C4).

- **Risk**: Confidence calibration could be gamed (agent claims COMPLETE coverage when it was PARTIAL)
  - **Mitigation**: The COVERAGE field requires specific line-range citations (e.g., "lines 1-87 read"), not just boolean flags. Byfuglien's Phase 4 validation spot-checks entry point functions by re-reading and comparing line counts. False COMPLETE claims are detected and corrected via confidence downgrade + validation annotation. This is not a cryptographic proof of coverage, but it's sufficient for internal reasoning artifacts where the agent has no adversarial incentive to lie (addresses C1c, C6).

- **Risk**: Changes to behavioral artifacts might have unintended effects on agent behavior
  - **Mitigation**: All changes are additive (new steps, new fields, new validation rules). No existing steps are removed or semantically altered. The `/reason` and `/trace-execution` skills remain structurally compatible with existing byfuglien routing logic.
