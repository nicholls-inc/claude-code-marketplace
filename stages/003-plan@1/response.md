The plan is written. Here's a brief summary:

**What's in `plan.md`:**

The plan implements `/assurance-probe` — a new **deterministic** Layer 4 skill that measures test strength for covered invariants. Unlike `/intent-check` (probabilistic, Layer 5) or `/spec-adversary` (best-effort, Layer 6), this skill operates entirely on observable test-file structure: assertion counts, property-based framework presence, boundary-case coverage, negative-path checks, and composite scenario breadth. No LLM round-trip judgment — scores derive from counts and keyword presence only.

**5 steps:**
1. Create `crosscheck/skills/assurance-probe/SKILL.md` — full skill definition with a 5-dimension rubric, output table format, and "weakest first" action list
2. Register in `hellebuyck.md` — task-classification row + Phase 3 skill-path entry
3. Update `crosscheck/docs/skills.md` — add to Layer 4 table
4. Update `crosscheck/docs/assurance-hierarchy.md` — add to Layer 4 skill→layer mapping
5. Update `crosscheck/README.md` — mention in Layer 4 bullet and skills overview

The verification approach uses three semi-formal execution traces (rubric determinism, aspirational exclusion, zero-assertion edge case) plus one patch comparison on `hellebuyck.md` to confirm no existing routing rows were disturbed. The risk register calls out the key threat — rubric subjectivity leaking in — and mitigates it by constraining all dimensions to observable counts.

```json
{
  "context_updates": {
    "plan_track": "semi-formal",
    "plan_step_count": 5
  }
}
```