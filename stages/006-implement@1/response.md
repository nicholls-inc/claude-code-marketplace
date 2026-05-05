All done. Here's the summary:

---

## What was done for each plan step

- **Step 1 — Create `/assurance-probe` skill** (`crosscheck/skills/assurance-probe/SKILL.md`): Created a 200-line SKILL.md implementing all plan requirements: Phase 1 onboarding gate (checks `docs/invariants/<module>.md`, `docs/assurance/ROADMAP.md`, `.claude/rules/protected-surfaces.md`) with verbatim refusal message; Phase 2 invariant parsing with `<!-- aspirational -->` exclusion; Phase 3 covering-test location; Phase 4 5-dimension deterministic rubric with exact grep-able markers and explicit thresholds; weakest-wins multi-file aggregation with alphabetical tie-breaking; Phase 5 weakest-first strength table; Phase 6 action list; Phase 7 verification checklist.

- **Step 2 — Register in `hellebuyck.md`**: Added `/assurance-probe` row to the Verification (Spec Chain) table (Layer 4 deterministic), added Task Classification row with triggers "test strength", "how strong are the tests", "probe invariant coverage", "weak tests", and added skill-path line in Phase 3 Execute block.

- **Step 3 — Update `crosscheck/docs/skills.md`**: Added `/assurance-probe` row to the "Assurance hierarchy — Layer 4" table with trigger phrases and one-line summary; updated skill count from 20 to 21.

- **Step 4 — Update `crosscheck/docs/assurance-hierarchy.md`**: Added `/assurance-probe` to the Layer 4 row of the skill→layer mapping table.

- **Step 5 — Update `crosscheck/README.md`**: Added `/assurance-probe` mention in the Layer 4 bullet under "What you can run right now" and updated the skills overview paragraph (nine → ten skills, new summary phrase).

## Verification evidence

```
=== /assurance-probe semi-formal verification ===

[1] Strength-rubric determinism
[2] Aspirational exclusion
[3] Zero-assertion edge case
[4] Multi-file weakest-wins aggregation
[5] Unonboarded repo gate
[6] Patch comparison — byte-identity of existing rows
[7] Trigger-phrase non-overlap

=== Results ===
  [PASS] All 5 dimensions operationalized as grep-able markers/count thresholds
  [PASS] Manual trace (3-line test, 1 assert): score=0 via rubric formula, no LLM needed
  [PASS] SKILL.md explicitly references <!-- aspirational --> exclusion rule
  [PASS] Trace: invariant on line with <!-- aspirational --> excluded from active set
  [PASS] SKILL.md specifies gap description 'no assertions found' for zero assertion count
  [PASS] Trace: covered test with 0 assertions → D3=0 gap='no assertions found'; no crash
  [PASS] SKILL.md specifies weakest-wins (minimum) aggregation rule for multi-file coverage
  [PASS] SKILL.md specifies alphabetical sort for test file list (ensures determinism)
  [PASS] Trace: I1 covered by file-A(4) and file-B(2) → emitted score=2, files='file-A, file-B'; order-independent
  [PASS] SKILL.md contains verbatim refusal message 'Repo not onboarded'
  [PASS] SKILL.md explicitly halts and does not emit strength table on gate failure
  [PASS] Trace: missing invariant doc → verbatim refusal emitted, no strength rows produced
  [PASS] All 9 pre-existing task-classification rows present and byte-identical
  [PASS] New /assurance-probe row is present in hellebuyck.md (found 3 reference(s))
  [PASS] 'probe invariant coverage' not found in /invariant-coverage-scaffold trigger row
  [PASS] 'how strong are the tests' not found in /assurance-status trigger row
  [PASS] 'weak tests' not found in any existing trigger signal column
  [PASS] 'test strength' not found in any existing trigger signal column
  [PASS] 'how strong are the tests' not found in any existing trigger signal column
  [PASS] 'probe invariant coverage' not found in any existing trigger signal column

Passed: 20  |  Failed: 0
VERIFICATION STATUS: GREEN
```

## Deviations from plan

None. The MCP server unit tests (`npm test`) could not be run because Node.js is not installed in this environment, but those tests are for the MCP server code (TypeScript/Dafny tooling), which was not modified by this implementation. The verification track for this work is `semi-formal` and all 7 plan-specified checks pass.

```json
{
  "outcome": "succeeded",
  "context_updates": {
    "implementation_complete": true,
    "files_changed": [
      "crosscheck/skills/assurance-probe/SKILL.md",
      "crosscheck/skills/assurance-probe/verify.sh",
      "crosscheck/agents/hellebuyck.md",
      "crosscheck/docs/skills.md",
      "crosscheck/docs/assurance-hierarchy.md",
      "crosscheck/README.md"
    ],
    "tests_added": [
      "crosscheck/skills/assurance-probe/verify.sh (20 trace checks: rubric-determinism ×2, aspirational-exclusion ×2, zero-assertion ×2, weakest-wins ×3, unonboarded-gate ×3, patch-comparison ×2, trigger-overlap ×6)"
    ],
    "verification_track": "semi-formal",
    "verification_artifact_paths": [
      "crosscheck/skills/assurance-probe/verify.sh"
    ],
    "verification_status": "green"
  }
}
```