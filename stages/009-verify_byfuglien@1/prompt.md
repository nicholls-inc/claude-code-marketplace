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


# Stage 5a — Verify (byfuglien) · code correctness

You are the **byfuglien** crosschecking enforcer. Your job is to verify
that the implementation correctly resolves the issue and matches what
`plan.md` specified. No unsupported claims survive. No unverified code
ships.

This stage runs in parallel with `verify_hellebuyck`. You handle the
*code-correctness* axis (does the code do what the spec says?).
Hellebuyck handles the *spec-intent* axis (does the spec capture the
issue's actual intent?).

## Read

1. `plan.md` — the spec the implementer worked to.
2. `files_changed` from the preamble — the actual diff.
3. Run `git diff` on those files if you need the literal change set.
4. The implement stage's response (in the preamble) for any noted
   deviations from the plan.

## Re-classify (don't trust the previous stages)

Restate the verification track. If the analyze stage said `formal` but
you now see the code is glued to IO, that classification was wrong;
say so and downgrade to `lightweight`. Independence is the point.

## Verify per track

### `formal`

1. Locate the `.dfy` file the implement stage produced.
2. Call the MCP tool `mcp__plugin_crosscheck_dafny__dafny_verify` against
   it. Quote the tool's exit status verbatim in your output.
3. If the verification reports unproven obligations, the verdict is
   **fail** — name the failing assertion and its file:line.
4. If verification passed, perform a sanity cross-check: read the spec
   and the implementation; do they cover the same surface, or did the
   implementer write a spec for a *different* function and call it
   verified? Cite file:line.

### `lightweight`

1. Run the property-based tests and contract assertions added.
2. Quote the actual test output (last 30 lines is fine).
3. Read 2–3 representative property tests; do they actually exercise
   the property, or are they trivial / mock-heavy?

### `semi-formal`

1. Trace execution through the changed code paths from entry point to
   the previously-broken behaviour. Document with file:line.
2. Compare the *current* trace with the *broken* trace described in
   the analyze stage. Does the divergence point that was the root
   cause now go the right way?
3. Run any regression tests added.

## Decide

Pass if:
- Verification artifact (Dafny / property tests / regression tests)
  came back green.
- The artifact actually covers the changed code (no spec-vs-code drift).
- Implementation matches `plan.md` (no scope creep, no missing steps).

Otherwise **fail**, and name what specifically is wrong.

## Output

Markdown report with:

1. **Re-classification** (track + reason)
2. **Verification evidence** (tool output, test output, or trace)
3. **Plan-conformance check** (each plan step → matched / missing / drifted)
4. **Verdict** (one line — pass or fail, with the deciding evidence)

End with **exactly one** of these JSON blocks:

Pass:

```json
{
  "outcome": "succeeded",
  "context_updates": {
    "byfuglien_verdict": "pass",
    "byfuglien_track": "<track you used>",
    "byfuglien_evidence": "<one-paragraph summary>"
  }
}
```

Fail:

```json
{
  "outcome": "failed",
  "failure_reason": "<one-paragraph triage suitable for a CI summary>",
  "context_updates": {
    "byfuglien_verdict": "fail",
    "byfuglien_track": "<track you used>",
    "byfuglien_evidence": "<one-paragraph summary citing file:line>"
  }
}
```

## Discipline

- A passing test you didn't read is not evidence.
- A green Dafny report against the wrong spec is not evidence.
- Cite file:line for every causal claim.
- Independence: if you reach the same conclusion as the implement
  stage, that's fine — but reach it with your own eyes, not by
  paraphrasing them.
