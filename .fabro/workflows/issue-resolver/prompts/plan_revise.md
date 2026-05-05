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
