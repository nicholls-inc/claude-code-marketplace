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

## Context
- analysis_classification: refactor
- concerns_addressed: 7
- concerns_rejected: 1
- files_involved: ["crosscheck/skills/reason/SKILL.md","crosscheck/skills/trace-execution/SKILL.md","crosscheck/agents/byfuglien.md"]
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
- verification_track: semi-formal


# Stage 4 — Implement

You are the implementer. Read the approved plan and execute it
step-by-step. Do exactly what the plan says — no more, no less.

## Read

1. Open `plan.md`. This is the source of truth for what to do.
2. Note the **verification track** declared in the plan. Your
   implementation must produce the artifacts the verify stage expects.

## Execute

### For every plan step

- Make the file edits the step describes.
- Run any relevant local checks (typecheck, lint, fast unit tests) as you
  go. Don't accumulate a giant uncommitted blast radius.
- If a step turns out to be wrong (the file doesn't exist, the API is
  different from what the plan assumed), **stop and report**, don't
  improvise. The plan/review loop produced a bad plan; the right move is
  to surface that, not to silently rewrite it.

### Verification artifacts

Per the plan's declared track:

- **`formal`** — Write Dafny spec(s) under the path the plan named (often
  next to the implementation, with a `.dfy` extension). Use the
  `mcp__plugin_crosscheck_dafny__dafny_verify` MCP tool to confirm the spec
  verifies. Use `mcp__plugin_crosscheck_dafny__dafny_compile` if the plan
  requires extracting verified Python/Go.
- **`lightweight`** — Add the contract assertions and property-based tests
  the plan named. Run them locally before finishing.
- **`semi-formal`** — Add the execution-trace anchors / regression tests
  the plan named.

### Tests

Add the tests `plan.md → ## Tests / properties to add` listed. Run the
project's test suite (`npm test`, `pytest`, `go test ./...`, etc. — look
at the repo's CONTRIBUTING / package files to choose). Tests must pass
before you finish.

### Commit

Make commits as you go (one logical commit per plan step is fine, or one
combined commit at the end if changes are tightly coupled). Use
Conventional Commits — this repo's CLAUDE.md notes that `docs:` is blocked
for behavioural artifacts.

The Fabro project config has draft-PR auto-creation enabled, so
committed work surfaces as a PR after the workflow completes.

## Output

Respond with:

1. A bulleted list of plan steps and what you actually did for each.
2. The output of the final test run (a short tail is fine).
3. A summary of any deviations from the plan and *why* (this becomes
   evidence for the verify stage).

End with:

```json
{
  "context_updates": {
    "implementation_complete": true,
    "files_changed": ["path/a", "path/b", "..."],
    "tests_added": ["test_name", "..."],
    "verification_track": "<the track you actually implemented to>"
  }
}
```

## Discipline

- The plan is the spec. Drift kills accountability.
- If you genuinely cannot finish a step (broken environment, missing
  dependency), commit what you have and emit
  `{"outcome": "partially_succeeded", ...}` so the verify stage can
  judge what was achieved.
