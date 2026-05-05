Goal: 140

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-6, 48.0k tokens in / 3.9k out
- **plan**: succeeded
  - Model: claude-sonnet-4-6, 32.7k tokens in / 2.6k out
  - Files: /workspace/plan.md
- **review**: succeeded
  - Model: claude-sonnet-4-6, 33.8k tokens in / 3.9k out

## Context
- analysis_classification: refactor
- files_involved: ["crosscheck/skills/spec-adversary/SKILL.md","crosscheck/skills/invariant-coverage-scaffold/SKILL.md","crosscheck/skills/intent-check/SKILL.md","crosscheck/skills/assurance-status/SKILL.md","crosscheck/agents/hellebuyck.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/docs/skills.md","crosscheck/README.md","crosscheck/.claude-plugin/plugin.json"]
- intent_gap: aligned
- issue_ref: nicholls-inc/claude-code-marketplace#140
- issue_title: crosscheck: assurance-probe — deterministic test-strength layer (design discussion)
- plan_step_count: 5
- plan_track: semi-formal
- review_concerns: 1. The SKILL.md for /assurance-probe must specify whether it requires the hellebuyck onboarding gate (docs/invariants/ present, ROADMAP, protected-surfaces rules) or explicitly documents what it emits on an unonboarded repo. Without this, a user on an unonboarded repo receives a silently empty strength table, which is indistinguishable from 'all tests are strong.' Every other hellebuyck skill either enforces the gate or documents the exception — /assurance-probe must do the same. (Probe 1, Missing invariant 1)
2. The SKILL.md must state a deterministic aggregation rule for the case where a single invariant ID is covered by more than one test file. Without a specified rule (weakest / strongest / list-all-separately), two runs reading files in different orders can produce different strength tables, directly violating the plan's claim of determinism. (Probe 1, Missing invariant 2)
3. Every rubric dimension must be operationalized as specific, grep-able, language-agnostic keywords or structural markers in the SKILL.md. 'Mutation probe hint' is named as a dimension but never concretized (e.g., presence of 'mutmut', 'pitest', 'stryker', '#mutant' in the file, or zero score otherwise). A dimension that requires LLM judgment to evaluate is not a deterministic rubric — the plan's own risk register flags this but does not resolve it in the skill design. (Probe 1, Missing invariant 3)
4. The 'patch comparison' verification test for hellebuyck.md only verifies byte-identity of existing rows (catching deletion/mutation), not semantic routing conflicts. The new row's trigger phrases ('test strength', 'how strong are the tests', 'probe invariant coverage', 'weak tests') must be checked against existing trigger signals for semantic overlap — particularly against 'invariant coverage', 'coverage gate', 'scaffold invariant check' (which routes to /invariant-coverage-scaffold) and 'assurance status' patterns. The plan must add a trigger-phrase non-overlap check to the verification section, not just rely on the risk register statement that 'trigger phrases are distinct.' (Probe 3, Concern 5)
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
