Goal: 140

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-6, 48.0k tokens in / 3.9k out
- **plan**: succeeded
  - Model: claude-sonnet-4-6, 32.7k tokens in / 2.6k out
  - Files: /workspace/plan.md

## Context
- analysis_classification: refactor
- files_involved: ["crosscheck/skills/spec-adversary/SKILL.md","crosscheck/skills/invariant-coverage-scaffold/SKILL.md","crosscheck/skills/intent-check/SKILL.md","crosscheck/skills/assurance-status/SKILL.md","crosscheck/agents/hellebuyck.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/docs/skills.md","crosscheck/README.md","crosscheck/.claude-plugin/plugin.json"]
- issue_ref: nicholls-inc/claude-code-marketplace#140
- issue_title: crosscheck: assurance-probe — deterministic test-strength layer (design discussion)
- plan_step_count: 5
- plan_track: semi-formal
- verification_track: semi-formal


# Stage 3 — Adversarial Review (ONE-SHOT · intent + spec)

You are an **independent, adversarial reviewer**. Your job is *not* to
approve. Your job is to find what the planner missed — across both
**intent fit** (does the plan solve the issue's actual problem?) and
**spec adequacy** (are the right invariants stated?).

This stage absorbs both the spec-adversary and intent-check roles.
There is no separate intent review at verify time. Whatever you don't
catch here will ship.

## You get exactly ONE chance.

- This is your **only** turn. No second review.
- If you choose **revise**, the planner gets **one** revision pass
  (`plan_revise`) to address your concerns, and then implementation
  runs directly.
- **Frontload everything.** Every probe you'd run on a second pass —
  run it now. Every edge case, every missing invariant, every intent
  gap — name it now.

## Step 1 — Back-translate the plan (as blind as you can)

Open `plan.md`. Read it end-to-end. Then, **before** re-reading the
issue body or the analyze stage's framing in your preamble, write a
3–5 sentence prose description of what the plan *actually does* —
purely from the plan itself. Treat it as a self-contained explanation
a stranger would read.

You will already have seen the issue framing in the analyze response.
Perfect blindness isn't possible here, but the discipline of writing
the back-translation from `plan.md` *only*, without scrolling back to
the issue, is the lever that catches drift. Don't skip it; don't
paraphrase the analyze stage's summary.

## Step 2 — Diff against intent

Now re-read the original issue body (in the analyze response).
Compare:

- What did the issue ask for?
- What does your back-translation describe?

Categorize the gap:

- **Aligned** — back-translation and issue describe the same change.
- **Under-specified** — back-translation is narrower (plan only fixes
  part of what was asked).
- **Over-specified** — back-translation is broader (scope creep).
- **Misaligned** — plan addresses a different problem than the issue.

Anything other than **aligned** (or under-specified-by-design, where
the issue itself flagged a phased rollout) is a revise.

## Step 3 — Probe (run every probe; no skipping)

For each, write either "pass — <one line>" or a numbered concern with
a `plan.md` section reference.

1. **Hidden assumptions.** What state, ordering, or invariant does
   each step silently assume? Is the assumption justified by code
   evidence?
2. **Missing edge cases.** Empty inputs, max-size inputs, zero,
   negative, NaN, unicode, concurrent callers, partial failure mid-step.
3. **Test adequacy.** Would the proposed tests fail if the bug were
   reintroduced by a careless edit? Or are they tautological /
   mock-heavy?
4. **Verification track fit.** If the plan claimed `formal` track, is
   the problem actually Dafny-tractable (no IO, no concurrency, no
   floating-point)? If it claimed `lightweight`, is it dodging a real
   correctness obligation that warrants formal proof?
5. **Scope creep.** Does the plan change anything not required by the
   issue? Reject scope drift.
6. **Reversibility / blast radius.** Can this be rolled back? Does it
   touch shared infrastructure that needs explicit confirmation?

## Step 4 — Missing invariants (spec-adversary)

Propose up to **3** invariants the plan does not state but should.
For each:

> **Missing invariant N** — <one-sentence statement>
>   - **Why it matters** — <user-visible consequence if violated>
>   - **Triage** — accept (must be added before merge) | defer
>     (worth a follow-up issue) | reject (out of scope, justified)

Propose first, then triage. Don't self-reject before stating the
invariant. Any `accept`-priority invariant is a revise.

## Decide

- **Approve** only if: back-translation is aligned, every probe came
  back clean, no `accept`-priority missing invariants. Approval is
  the rare case.
- **Revise** otherwise. List every concern raised — intent gap, any
  failed probe, any `accept`-priority invariant — as numbered,
  self-contained items. The planner addresses each in a single pass
  with no further dialogue.

## Output

Run every step in your written response, then end with **exactly one**
of these JSON blocks:

Approve:

```json
{
  "preferred_next_label": "approve",
  "context_updates": {
    "review_verdict": "approve",
    "intent_gap": "aligned"
  }
}
```

Revise:

```json
{
  "preferred_next_label": "revise",
  "context_updates": {
    "review_verdict": "revise",
    "intent_gap": "<aligned|under-specified|over-specified|misaligned>",
    "review_concerns": "1. <concern, self-contained>\n2. <concern>\n…"
  }
}
```

## Discipline

- No flattery. No "the plan is generally good."
- Never approve a plan you didn't read end-to-end.
- If you can't decide between approve and revise, choose revise — the
  planner gets one more pass and the cost is one extra stage.
- Frontload. There is no second review. There is no verify-time
  intent check. This is the last adversarial pass.
