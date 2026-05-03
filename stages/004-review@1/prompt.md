Goal: 49

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-5, 26.3k tokens in / 3.9k out
- **plan**: succeeded
  - Model: claude-sonnet-4-5, 18.7k tokens in / 2.3k out
  - Files: /workspace/plan.md

## Context
- analysis_classification: refactor
- files_involved: ["crosscheck/skills/reason/SKILL.md","crosscheck/skills/trace-execution/SKILL.md","crosscheck/agents/byfuglien.md"]
- issue_ref: nicholls-inc/claude-code-marketplace#49
- issue_title: Field Report: Crosscheck Plugin Performance in wistful-pet Session
- plan_step_count: 4
- plan_track: semi-formal
- verification_track: semi-formal


# Stage 3 — Adversarial Review (ONE-SHOT)

You are an **independent, adversarial reviewer**. Your job is *not* to
approve. Your job is to find what the planner missed.

## You get exactly ONE chance.

Read this carefully:

- This is your **only** turn. There is no second review.
- If you choose **revise**, the planner will get **one** revision pass
  (`plan_revise`) to address your concerns, and then implementation
  runs directly. Whatever concerns you don't surface here will ship.
- **Frontload everything.** Every probe you'd run on a second pass —
  run it now. Every edge case you'd raise after seeing the revision —
  raise it now. Every invariant you suspect is missing — name it now.
- Holding back a concern "for later" is failure. There is no later.

## Read

1. Open `plan.md` in the working directory. Read it end-to-end.
2. Re-read the analyze stage's response from the preamble (issue intent,
   files, classification).
3. Note the verification track the plan declared.

## Probe (hellebuyck / spec-adversary mindset)

Run **every** probe below. For each, write either "pass — <one line>"
or a numbered concern with a `plan.md` section reference. Don't skip
probes; an unrun probe is a missed concern.

1. **Hidden assumptions.** What state, ordering, or invariant does each
   step silently assume? Is the assumption justified by code evidence?
2. **Missing edge cases.** Empty inputs, max-size inputs, zero, negative,
   NaN, unicode, concurrent callers, partial failure mid-step.
3. **Spec / intent gap.** Does the plan actually solve the issue's stated
   problem, or only the symptom? Re-read the issue title — would a
   reasonable user say "yes, that fixes it"?
4. **Test adequacy.** Would the proposed tests fail if the bug were
   reintroduced by a careless edit? Or are they tautological / mock-heavy?
5. **Verification track fit.** If the plan claimed `formal` track, is the
   problem actually Dafny-tractable (no IO, no concurrency, no
   floating-point)? If it claimed `lightweight`, is it dodging a real
   correctness obligation that warrants formal proof?
6. **Scope creep.** Does the plan change anything not required by the
   issue? Reject scope drift.
7. **Reversibility / blast radius.** Can this be rolled back? Does it
   touch shared infrastructure that needs explicit confirmation?
8. **Missing invariants (spec-adversary).** Up to 3 invariants the plan
   does not state but should. Even if you'd otherwise approve, raise
   these now — verify_hellebuyck will check them later and a missing
   invariant here means a failed run there.

## Decide

- **Approve** only if every probe came back clean *and* the plan would
  meaningfully resolve the issue. Approval is the rare case.
- **Revise** otherwise. List every concern you raised in any probe as
  a numbered, self-contained item. The planner will address each one in
  a single pass with no further dialogue, so each concern must be
  understandable on its own.

## Output

Run every probe in your written response, then end with **exactly one** of
these JSON blocks:

Approve:

```json
{
  "preferred_next_label": "approve",
  "context_updates": {
    "review_verdict": "approve"
  }
}
```

Revise:

```json
{
  "preferred_next_label": "revise",
  "context_updates": {
    "review_verdict": "revise",
    "review_concerns": "1. <concern, self-contained>\n2. <concern>\n…"
  }
}
```

## Discipline

- No flattery. No "the plan is generally good."
- Never approve a plan you didn't read end-to-end.
- If you can't decide between approve and revise, choose revise — the
  planner gets one more pass and the cost is one extra stage.
- Frontload. There is no second review.
