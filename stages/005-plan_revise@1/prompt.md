Goal: 61

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-5, 23.6k tokens in / 4.5k out
- **plan**: succeeded
  - Model: claude-sonnet-4-5, 11.3k tokens in / 2.0k out
  - Files: /workspace/plan.md
- **review**: succeeded
  - Model: claude-sonnet-4-5, 6.2k tokens in / 3.0k out

## Context
- analysis_classification: refactor
- files_involved: ["crosscheck/agents/byfuglien.md","crosscheck/skills/reason/SKILL.md","crosscheck/skills/locate-fault/SKILL.md"]
- intent_gap: aligned
- issue_ref: nicholls-inc/claude-code-marketplace#61
- issue_title: crosscheck: component-correct verification misses end-to-end integration gaps
- plan_step_count: 3
- plan_track: semi-formal
- review_concerns: 1. Step 1 checklist item 'Traced across at least N component boundaries' is unactionable without defining what a 'component boundary' is. Provide a precise definition (e.g., file-to-file boundary, module import boundary) or heuristic.
2. Step 2 assumes callees have readable implementations but doesn't integrate the termination condition from risk mitigation. Clarify: when tracing 2+ files is 'mandatory' but a library call is reached at file boundary, does validation skip or stop? Reconcile Step 2's mandate with the risk mitigation's termination rule.
3. Step 2 and risk mitigation conflict on whether integration validation is mandatory for 2+ files or skippable when reaching library calls. Specify precedence: is the 2-file rule absolute, or do termination conditions override?
4. Test 'Execution trace spanning test' should specify a bug type that proves integration validation works—use a bug that appears correct in moduleA but fails due to violated assumptions at A→B interface (e.g., precondition violation), not just any bug in moduleB.
5. Test 'Interface-only reasoning test' should provide a concrete example (e.g., caller passes x=0 to divide_by(x) where callee requires x != 0) rather than abstract description.
6. [MISSING INVARIANT] For any multi-component analysis, the set of files read during integration validation must be a superset of files containing code on the actual execution path of the failing test. Without this, 'integration validation' doesn't guarantee completeness.
7. [MISSING INVARIANT] When integration validation reaches an unreadable callee (library, extern, proprietary), output must explicitly document the trust boundary and state assumptions about callee behavior. Prevents silent premise gaps.
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
