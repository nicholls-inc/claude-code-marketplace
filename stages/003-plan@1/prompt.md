Goal: 140

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-5, 30.5k tokens in / 4.1k out

## Context
- analysis_classification: algorithmic
- files_involved: ["crosscheck/skills/spec-adversary/SKILL.md","crosscheck/skills/assurance-init/SKILL.md","crosscheck/skills/invariant-coverage-scaffold/SKILL.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/README.md","crosscheck/demo/06_test_adequacy/SCRIPT.md"]
- issue_ref: nicholls-inc/claude-code-marketplace#140
- issue_title: crosscheck: assurance-probe — deterministic test-strength layer (design discussion)
- verification_track: formal


# Stage 2 — Plan

You are the planner. Turn the analysis from the previous stage into a
concrete, sequenced implementation plan that the implement stage will
execute step-by-step.

## What you have

Your preamble already contains:

- The full **analyze** stage response (issue summary, files, classification,
  verification track).
- `analysis_classification` and `verification_track` in context.

## What to do

1. **Read** any files mentioned in `files_involved` so your plan is grounded
   in actual code, not the analyzer's summary.
2. **Write `plan.md`** to the working directory. Structure:

   ```markdown
   # Plan: <issue title>

   ## Verification track
   <formal | lightweight | semi-formal>

   ## Steps
   1. <action — file path — what changes>
   2. ...

   ## Tests / properties to add
   - <test name — what it asserts>

   ## Verification approach
   <How the verify stage will check this. For `formal` track, name the
   Dafny spec(s) that will be written. For `lightweight`, name the
   property-based tests / contract assertions. For `semi-formal`, name
   the execution traces / patch-comparison anchors.>

   ## Risk register
   - <known risk — mitigation>
   ```

3. Keep the plan small and reversible. A bug fix is one or two changes,
   not a refactor. Don't smuggle in unrelated cleanups.

## What happens next

The adversarial reviewer reads `plan.md` and gets exactly one chance to
flag concerns. If they revise, you'll get one revision pass via the
`plan_revise` stage to address every concern, and then implementation
runs directly. So write the plan as if it might ship as-is — don't
leave gaps for the reviewer to catch.

## Output

Respond with a brief summary (under 250 words) of what you wrote to
`plan.md`. The reviewer reads `plan.md` directly; you don't need to
re-paste it in your response.

End with this JSON block:

```json
{
  "context_updates": {
    "plan_track": "<verification track from your plan>",
    "plan_step_count": <integer>
  }
}
```
