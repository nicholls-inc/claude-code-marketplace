Goal: 140

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-5, 30.5k tokens in / 4.1k out
- **plan**: succeeded
  - Model: claude-sonnet-4-5, 32.0k tokens in / 3.7k out
  - Files: /workspace/plan.md
- **review**: succeeded
  - Model: claude-sonnet-4-5, 19.9k tokens in / 6.4k out

## Context
- analysis_classification: algorithmic
- files_involved: ["crosscheck/skills/spec-adversary/SKILL.md","crosscheck/skills/assurance-init/SKILL.md","crosscheck/skills/invariant-coverage-scaffold/SKILL.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/README.md","crosscheck/demo/06_test_adequacy/SCRIPT.md"]
- intent_gap: aligned
- issue_ref: nicholls-inc/claude-code-marketplace#140
- issue_title: crosscheck: assurance-probe — deterministic test-strength layer (design discussion)
- plan_step_count: 12
- plan_track: formal
- review_concerns: 1. Hidden assumption (Step 2, line 17): Define the grammar/pattern for parseable 'Failure condition' clauses, or cite example invariant docs that demonstrate the expected format.
2. Hidden assumption (Step 3, line 23-24): Specify behavior when coverage tooling (pytest-cov) is absent — fail-fast with actionable error or auto-install?
3. Hidden assumption (Step 5, line 38, 111): Define 'bit-identical on same commit' — which variables are locked (git SHA, Python version, dependencies, OS)? Add explicit environment capture to reproducer template.
4. Hidden assumption (Step 11, line 71): Define 'rotation-based' operationally — who triggers the probe, how often, via what mechanism (manual, cron, /assurance-status recommendation)?
5. Hidden assumption (Verification line 119-122): Tracker integrity property assumes single writer. Address concurrent execution: prevent it, or prove append-only semantics under contention.
6. Missing edge cases: Handle empty/missing Failure condition clauses (skip, warn, or error?), zero-invariant modules, zero-finding runs (SNR 0/0), test framework syntax errors vs execution errors, reproducer scripts referencing rebased-away commits.
7. Test adequacy (line 81-84): Mutation parser test 'determinism' is tautological. Add correctness oracle — reference table mapping Failure condition examples to expected mutations.
8. Test adequacy (line 90-92): Reproducer integration test should include negative case — run reproducer on different commit or with mutation reverted, assert output differs.
9. Test adequacy (line 94-98): E2E test must use real executable Python test + real killable mutation, not mocked/scaffolded stubs (verify this is not test theatre).
10. Track terminology (line 3): Clarify 'formal' means 'Layer 4 deterministic property testing', not 'Layer 1 Dafny proof', to avoid confusion.
11. Reversibility (Step 11): Document byfuglien.md routing addition as additive-only (no modification of existing routes) to preserve backward compatibility if skill is retired.
12. Reversibility (Steps 8-9): Add risk register note that README/assurance-hierarchy.md updates should be marked experimental (e.g., 'Layer 4 (Phase 1)') until SNR ≥ 1:3 demonstrated.
13. Reversibility (line 119-122): Add backup/restore or checksum validation to tracker CSV writes to recover from corruption.
14. Missing invariant (ACCEPT): Mutation soundness — every generated mutation must violate a Failure condition reachable by the covering test's input generator. Address in Step 2 or risk register as known Phase 1 limitation (Phase 3 generator probe partially mitigates).
15. Missing invariant (ACCEPT): Reproducer environment capture — template must verify Python version, dependency versions, emit clear error on mismatch. Add to Step 5 spec.
- review_verdict: revise
- verification_track: formal


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
