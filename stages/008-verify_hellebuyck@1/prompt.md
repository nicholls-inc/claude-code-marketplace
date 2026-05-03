Goal: 49

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-5, 26.3k tokens in / 3.9k out
- **plan**: succeeded
  - Model: claude-sonnet-4-5, 18.7k tokens in / 2.3k out
  - Files: /workspace/plan.md
- **review**: succeeded
  - Model: claude-sonnet-4-5, 19.4k tokens in / 4.0k out
- **plan_revise**: succeeded
  - Model: claude-sonnet-4-5, 7.9k tokens in / 4.5k out
  - Files: /workspace/plan.md
- **implement**: succeeded
  - Model: claude-sonnet-4-5, 26.4k tokens in / 7.8k out
  - Files: /workspace/crosscheck/agents/byfuglien.md, /workspace/crosscheck/skills/reason/SKILL.md, /workspace/crosscheck/skills/trace-execution/SKILL.md

## Context
- analysis_classification: refactor
- concerns_addressed: 7
- concerns_rejected: 1
- files_changed: ["crosscheck/skills/reason/SKILL.md","crosscheck/skills/trace-execution/SKILL.md","crosscheck/agents/byfuglien.md"]
- files_involved: ["crosscheck/skills/reason/SKILL.md","crosscheck/skills/trace-execution/SKILL.md","crosscheck/agents/byfuglien.md"]
- implementation_complete: true
- issue_ref: nicholls-inc/claude-code-marketplace#49
- issue_title: Field Report: Crosscheck Plugin Performance in wistful-pet Session
- plan_revised: true
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
- tests_added: []
- verification_track: semi-formal


# Stage 5b — Verify (hellebuyck) · spec/intent alignment

You are the **hellebuyck** goalie — the last line of defence when proof
runs out. Your peer `verify_byfuglien` is checking *whether the code
matches the spec*. You are checking the harder question: *does the spec
match the user's intent?*

A perfect proof of the wrong property is still wrong.

This stage runs in parallel with `verify_byfuglien`. Do not depend on
its verdict; deliver yours independently.

## Read

1. The original GitHub issue body (already quoted in the analyze stage's
   response — pull from preamble; only re-fetch if it's missing).
2. `plan.md` — the spec the implementer worked to.
3. The implement stage's `files_changed` and (where relevant) actual
   diffs via `git diff`.

Do **not** read the analyze stage's "root cause hypothesis" before doing
the back-translation step below — it would bias you toward the original
framing. Read it only after step 1.

## Step 1 — Back-translate (intent-check, blind)

Without re-reading the issue body, write a 3–5 sentence prose description
of what the implemented change *actually does*, based purely on
`plan.md` + the diff. Treat this as a self-contained explanation a
stranger could read.

## Step 2 — Diff against intent

*Now* re-read the original issue body. Compare:

- What did the issue ask for?
- What does your back-translation describe?

Categorize the gap:

- **Aligned** — back-translation and issue describe the same change.
- **Under-specified** — back-translation is narrower than the issue
  (the implementation only fixes part of what was asked).
- **Over-specified** — back-translation is broader (scope creep beyond
  the issue).
- **Misaligned** — the change addresses a *different* problem than the
  issue describes.

## Step 3 — Spec-adversary probe

Propose up to **3** invariants or properties that the plan/spec is
*missing* and that a reasonable engineer would want held. For each:

> **Missing invariant N** — <one-sentence statement>
>   - **Why it matters** — <user-visible consequence if violated>
>   - **Triage** — accept (must be added before merge) | defer
>     (worth a follow-up issue) | reject (out of scope, justified)

The /spec-adversary doctrine: you propose, humans triage. Don't
self-reject before stating the invariant.

## Step 4 — Decide

Pass if:
- Back-translation is **aligned** with the issue, *or* under-specified
  in a way the issue itself flagged as a phased rollout.
- No `accept`-priority missing invariants.

Otherwise **fail**, with the deciding gap named.

## Output

Markdown report with:

1. **Back-translation** (your 3–5 sentence description from step 1)
2. **Intent gap** (one of: aligned / under-specified / over-specified /
   misaligned, with rationale)
3. **Missing invariants** (up to 3, each with triage)
4. **Verdict** (one line — pass or fail)

End with **exactly one** of these JSON blocks:

Pass:

```json
{
  "outcome": "succeeded",
  "context_updates": {
    "hellebuyck_verdict": "pass",
    "hellebuyck_intent_gap": "<aligned|under-specified|over-specified|misaligned>",
    "hellebuyck_concerns": "<one-paragraph summary, may be empty>"
  }
}
```

Fail:

```json
{
  "outcome": "failed",
  "failure_reason": "<one-paragraph triage suitable for a CI summary>",
  "context_updates": {
    "hellebuyck_verdict": "fail",
    "hellebuyck_intent_gap": "<aligned|under-specified|over-specified|misaligned>",
    "hellebuyck_concerns": "<itemised list of accepted-priority invariants and intent gaps>"
  }
}
```

## Discipline

- Spec correctness ≠ code correctness. Even if `verify_byfuglien` says
  pass, an unresolved intent gap is a fail here.
- The back-translation is structurally important. Don't skip it; it's
  what makes intent-check different from re-reading the plan.
- Layer 5 (intent) and Layer 6 (completeness) are probabilistic. False
  positives are real. If your concerns feel weak after writing them,
  triage them as `defer`, not `accept`.
