# Gate: audit-spec-coverage triage

## What this gate protects

The `/audit-spec-coverage` skill compares a prose spec against the
"invariants" written for it — the machine-checkable rules ("a queue's
length is never negative") that are supposed to enforce what the spec
promises. It looks for spec sections, and known audit-finding IDs (entries
in a spec's own traceability table of previously-caught problems), that no
invariant covers. An uncovered constraint is one that the spec asserts but
nothing actually defends — a gap that stays invisible until this audit
finds it. This gate is where a human decides what to do about each gap the
audit surfaces.

## What you are being asked to decide

For each gap (each numbered finding, capped at 15 per run so the list
stays reviewable in one sitting), you choose exactly one of four
resolutions.

## What each decision means

- **Accept (fix invariant)** — the gap is real and the fix belongs in the
  invariant set. Someone writes or extends an invariant doc to cover it;
  the spec itself is unchanged.
- **Accept (amend spec via `/protected-surface-amend`)** — the gap is real
  but the fix belongs in the spec instead (for example, the spec should
  never have implied this constraint, or should state it more precisely).
  Invariant docs and specs are both "protected surfaces" — files this repo
  treats as sensitive enough to require a recorded justification before
  editing — so the change is routed through `/protected-surface-amend`
  rather than edited directly.
- **Reject** — the finding is a false positive (for example, the
  constraint is genuinely covered under different wording elsewhere).
  Nothing changes; record the reason so a future run does not re-raise it
  without context.
- **Defer** — real or plausible, but not worth acting on now. Record the
  condition under which it should be revisited.

Nothing in the spec or invariant set is changed by the audit itself —
these four choices are a record of what a human decided to do next, not
an automatic action. The findings live in `findings-coverage.md`, and the
follow-through (writing the invariant, or the spec amendment) is separate,
manual or orchestrator-driven work.

## Where this shows up

Besides the file above, this same gate content is posted as a PR comment
by the repository's automated spec-audit CI check, so reviewers see the
triage prompt directly on the pull request without needing to open a
generated file.

## How long this takes

A few minutes per finding, typically all findings in one sitting since the
list is capped at 15.

## Who to ask if unsure

The Crosscheck maintainers via a GitHub issue on this repository.
