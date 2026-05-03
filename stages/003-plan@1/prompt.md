Goal: 49

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-5, 19.7k tokens in / 3.9k out

## Context
- analysis_classification: refactor
- files_involved: ["crosscheck/skills/reason/SKILL.md","crosscheck/skills/trace-execution/SKILL.md","crosscheck/agents/byfuglien.md"]
- issue_ref: nicholls-inc/claude-code-marketplace#49
- issue_title: Field Report: Crosscheck Plugin Performance in wistful-pet Session
- verification_track: semi-formal


# Stage 2 — Plan

You are the planner. Turn the analysis from the previous stage into a
concrete, sequenced implementation plan that the implement stage will
execute step-by-step.

## What you have

Your preamble already contains:

- The full **analyze** stage response (issue summary, files, classification,
  verification track).
- `analysis_classification` and `verification_track` in context.
- **If this is your second visit:** the **review** stage's response *and*
  `review_concerns` in context. The review may have rejected your previous
  plan — incorporate every concern.

## What to do

### If `review_concerns` is present (revision pass)

1. Read the full review response from the preamble.
2. List each concern explicitly in your output ("Concern 1: …", "Concern 2: …").
3. For each concern, describe how the revised plan addresses it. Do not
   wave concerns away. If a concern is invalid, say *why* it is invalid —
   don't just ignore it.

### Always

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
   <How verify_byfuglien will check this. For `formal` track, name the
   Dafny spec(s) that will be written. For `lightweight`, name the
   property-based tests / contract assertions. For `semi-formal`, name
   the execution traces / patch-comparison anchors.>

   ## Risk register
   - <known risk — mitigation>
   ```

3. Keep the plan small and reversible. A bug fix is one or two changes,
   not a refactor. Don't smuggle in unrelated cleanups.

## Output

Respond with a brief summary (under 250 words) of what you wrote to
`plan.md`. The implement stage reads `plan.md` directly; you don't need
to re-paste it in your response.

If revising, lead with: "Revision pass — addressed N review concerns:
…". This makes the iteration auditable in the run history.

End with this JSON block:

```json
{
  "context_updates": {
    "plan_track": "<verification track from your plan>",
    "plan_step_count": <integer>
  }
}
```
