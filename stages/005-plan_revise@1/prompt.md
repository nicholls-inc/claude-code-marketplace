Goal: 49

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-5, 26.3k tokens in / 3.9k out
- **plan**: succeeded
  - Model: claude-sonnet-4-5, 18.7k tokens in / 2.3k out
  - Files: /workspace/plan.md
- **review**: succeeded
  - Model: claude-sonnet-4-5, 19.4k tokens in / 4.0k out

## Context
- analysis_classification: refactor
- files_involved: ["crosscheck/skills/reason/SKILL.md","crosscheck/skills/trace-execution/SKILL.md","crosscheck/agents/byfuglien.md"]
- issue_ref: nicholls-inc/claude-code-marketplace#49
- issue_title: Field Report: Crosscheck Plugin Performance in wistful-pet Session
- plan_step_count: 4
- plan_track: semi-formal
- review_concerns: 1. CIRCULAR DEPENDENCY IN COVERAGE TRACKING (Steps 2-4): The plan requires agents to self-report COVERAGE as COMPLETE vs PARTIAL, then uses this to calibrate confidence and trigger byfuglien spot-checks. But the field report's W1 shows agents already fail to recognize when they've missed critical code (the filter). The mechanical constraint in Step 3 ('if COVERAGE shows PARTIAL, confidence MUST be MEDIUM') only works if the COVERAGE field is accurate. Specify: (a) how an agent determines it has read 'all N lines' (line count? diff-check?), (b) what byfuglien's 'spot-check that entry point functions were read in entirety' actually does (re-read? verify line count? how?), and (c) what happens when an agent incorrectly reports COMPLETE coverage.

2. VERIFICATION APPROACH IS SPECULATIVE, NOT EMPIRICAL (plan.md verification approach, item 3): The plan's verification includes 'Execution trace simulation: Given the wistful-pet scenario... would the updated skills force the behaviors that were missing? YES' — but this is a thought experiment, not an actual trace. Specify a concrete test: run the updated `/trace-execution` skill on a representative codebase (e.g., create a minimal Django function with a filter), verify that COVERAGE tracking actually fires and that confidence calibration prevents HIGH confidence claims on partial traces. Without empirical validation, we're guessing that the changes work.

3. STRUCTURAL CONFUSION BETWEEN STEP 4 AND STEP 5b (/reason skill, Steps 4-5): The field report's W3 says the EXISTING 'Alternative Hypothesis Check' (Step 4) should have verified the proposed fix. The plan adds a NEW Step 5b for fix verification instead. This creates two overlapping steps: when does an agent check the diagnosis (Step 4) vs the prescription (Step 5b)? Clarify the boundary: is Step 4 only for ruling out alternative diagnoses of the BUG, while Step 5b checks the FIX? Or do they have overlapping responsibilities? If overlapping, why not strengthen Step 4 instead of adding Step 5b?

4. MISSING EDGE CASE HANDLING (Step 2 exhaustive read instruction): What happens when: (a) a function is 0 lines (pure delegation / abstract method) — does COVERAGE = COMPLETE or PARTIAL? (b) a function is 500+ lines (e.g., Django admin class) — is 'read ENTIRE function body' still mandatory, or is there a cost/benefit threshold? (c) the agent is interrupted mid-read (reads lines 1-50 of 100) — does COVERAGE reset to PARTIAL or maintain partial state? Without edge case handling, the instruction 'read ENTIRE function body' is ambiguous in degenerate cases.

5. NO ESCAPE HATCH FOR STEP 5b COST (plan.md risk register, first risk): The risk register notes Step 5b could make sessions too long for complex fixes, mitigated by '1-2 minutes added to 4-minute sessions'. But what if a fix is a 50-line refactor requiring nested reasoning? Step 5b is now mandatory for ALL /reason invocations — users can't opt out even if they judge the cost unacceptable for their use case. Add either: (a) a complexity threshold (e.g., 'if proposed fix is >20 lines or modifies >2 functions, Step 5b may be abbreviated to checklist only'), or (b) explicit opt-out mechanism.

6. MISSING INVARIANT: COVERAGE SELF-REPORT ACCURACY (Steps 2-3, theoretical gap): The plan introduces COVERAGE = COMPLETE | PARTIAL as a self-reported field, but provides no mechanism for the agent to verify its own claim. Establish invariant: 'COVERAGE = COMPLETE ⟹ agent has line-by-line reading record OR byfuglien has independently verified completeness'. Without this, COMPLETE is just an assertion, not a verified fact. The field report shows assertions fail (W1: agent missed filter, presumably while believing it had read the whole function).

7. MISSING INVARIANT: CONFIDENCE CALIBRATION SOUNDNESS (Step 3, theoretical gap): The mechanical constraint 'if COVERAGE = PARTIAL, confidence MUST be MEDIUM or below' is necessary but not sufficient. An agent could read all lines (COVERAGE = COMPLETE) but misinterpret them (e.g., read the filter but think it only applies to tier, not enrollment target). Establish invariant: 'confidence = HIGH ⟹ (all functions have COVERAGE = COMPLETE) ∧ (all OBSERVATIONS are [STATIC], not [SEMANTIC] or [BEHAVIORAL])'. This prevents HIGH confidence on semantically ambiguous traces even when coverage is complete.

8. MISSING INVARIANT: STEP 5b TERMINATION BOUND (Step 1, theoretical gap): Step 5b checks if the proposed fix introduces new bugs. But what if the fix itself is complex (e.g., a 50-line refactor)? Does Step 5b invoke a nested /reason call? Apply the full 7-step process recursively? Or just run a checklist? Establish invariant: 'Step 5b depth ≤ 1 (single-pass checklist analysis, no recursive reasoning)'. Without a termination bound, Step 5b could trigger unbounded recursion on complex fixes.
- review_verdict: revise
- verification_track: semi-formal


# Stage 2b — Plan · revise (one-shot)

You are the planner, on your single revision pass. The adversarial reviewer
has flagged concerns with the original `plan.md`. Your job is to address
**every** concern in **one** pass.

**There is no second review.** After this stage, the implementer reads
`plan.md` and executes it directly. If you skip a concern, it ships.

## What you have

Your preamble contains:

- The full **plan** stage response (your original plan).
- The full **review** stage response (the reviewer's verdict and concerns).
- `review_concerns` in context — the itemised list the reviewer wrote.
- `plan.md` already exists in the working directory; it is the original plan.

## What to do

1. Re-read `plan.md` so you know what you wrote originally.
2. List every concern from `review_concerns` explicitly. Number them.
3. For each concern, state plainly:
   - **How the revised plan addresses it** — concrete change to a step, a
     test, a verification approach, or the risk register. Cite which
     section of `plan.md` you're editing.
   - **Or — why the concern is invalid** — with code-level evidence
     (`path/to/file.py:LINE`). If you reject a concern, say so and justify
     it; do not silently ignore it.
4. **Update `plan.md` in place.** Overwrite the file with the revised plan.
   Same structure as the original (Verification track / Steps / Tests /
   Verification approach / Risk register), now incorporating the changes.
5. If the reviewer's concerns reveal that the analysis itself was wrong
   (e.g. classification was off, key files missed), update the relevant
   sections of `plan.md` accordingly. The plan can change track if needed.

## Output

Respond with:

1. **Concern-by-concern resolution table.** For each numbered review concern:
   - The concern text (one line)
   - Status: `addressed` | `rejected (with rationale)`
   - Affected `plan.md` section
2. A brief summary (under 150 words) of what materially changed in the plan.

The implement stage reads `plan.md` directly; you don't need to re-paste it.

End with this JSON block:

```json
{
  "context_updates": {
    "plan_revised": true,
    "concerns_addressed": <integer count>,
    "concerns_rejected": <integer count>,
    "plan_track": "<verification track in the revised plan>"
  }
}
```

## Discipline

- One pass. Every concern gets a status. No hand-waving.
- If you reject a concern, the implementer and the verify stage will see
  your rationale in the run log — make it bulletproof.
- Don't expand scope to address concerns. Address them with the minimum
  surgical change. The reviewer's job was to catch missed essentials,
  not to volunteer a redesign.
