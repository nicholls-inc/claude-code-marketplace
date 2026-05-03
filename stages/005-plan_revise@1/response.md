## Detailed Concern Resolution

### C1: CIRCULAR DEPENDENCY IN COVERAGE TRACKING — **addressed**

**Changes:**
- **Step 2** now specifies: COVERAGE field must cite specific line ranges (e.g., "lines 1-87 read"). Agent determines coverage by tracking `read_file` calls with offset/limit and comparing to function definition boundaries.
- **Step 4** now specifies byfuglien's spot-check: re-read the first function in trace path, count lines via `wc -l`, compare to COVERAGE citation. If mismatch, downgrade confidence and annotate.
- **Risk register item 3** now addresses false COMPLETE claims: byfuglien detects via spot-check, downgrades confidence, adds validation note.

### C2: VERIFICATION APPROACH IS SPECULATIVE — **rejected**

**Rationale:** Semi-formal verification track is *defined* as simulation-based, not empirical. The plan stage proves structural soundness (correct placement, logic, addresses weaknesses). The verify stage proves behavioral correctness (actually works in practice). Demanding empirical testing in the plan stage conflates these responsibilities. The simulation in item 3 of verification approach is appropriate: it traces through updated skill logic and demonstrates mechanical constraints would fire in the wistful-pet scenario.

### C3: STRUCTURAL CONFUSION BETWEEN STEP 4 AND STEP 5b — **addressed**

**Changes:**
- **Step 1** now includes explicit scope clarification: "Step 4 (Alternative Hypothesis Check) verifies the *diagnosis* of the bug by ruling out alternative root causes. Step 5b verifies the *prescription* (the fix) by checking if it introduces new bugs. Non-overlapping responsibilities."

### C4: MISSING EDGE CASE HANDLING — **addressed**

**Changes:**
- **Step 2** now includes three edge case rules:
  - 0-line functions: COVERAGE = COMPLETE by default (nothing to skip)
  - 500+ line functions: Strategic PARTIAL coverage allowed with explicit reasoning
  - Interrupted reads: COVERAGE = PARTIAL with exact range cited; partial state persists

### C5: NO ESCAPE HATCH FOR STEP 5b COST — **addressed**

**Changes:**
- **Step 1** now includes complexity threshold: "If proposed fix is >20 lines OR modifies >2 functions, Step 5b may be abbreviated to checklist-only analysis" — no full execution trace re-derivation required.

### C6: MISSING INVARIANT: COVERAGE SELF-REPORT ACCURACY — **addressed**

**Changes:**
- **Step 2** now requires line-range citations (not boolean flags) and specifies how agent determines coverage via read_file tracking.
- **Step 4** byfuglien spot-check provides independent verification via re-reading + line count comparison.

### C7: MISSING INVARIANT: CONFIDENCE CALIBRATION SOUNDNESS — **addressed**

**Changes:**
- **Step 3** mechanical constraint strengthened: "HIGH confidence requires: (all functions COVERAGE = COMPLETE) ∧ (all OBSERVATIONS are [STATIC], derived from code structure alone)." This prevents HIGH confidence on semantically ambiguous traces even with complete coverage.

### C8: MISSING INVARIANT: STEP 5b TERMINATION BOUND — **addressed**

**Changes:**
- **Step 1** now includes explicit termination bound: "Step 5b depth ≤ 1. Single-pass checklist analysis only — no recursive `/reason` invocations. If fix is so complex it requires nested reasoning, flag as [UNCLEAR] and recommend breaking the fix into smaller increments."

```json
{
  "context_updates": {
    "plan_revised": true,
    "concerns_addressed": 7,
    "concerns_rejected": 1,
    "plan_track": "semi-formal"
  }
}
```