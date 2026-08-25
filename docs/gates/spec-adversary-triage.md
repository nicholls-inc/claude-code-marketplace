# Gate: spec-adversary triage

## What this gate protects

`/spec-adversary` is a Crosscheck skill that adversarially reads a module's
**invariant doc** (a markdown file under `docs/invariants/<module>.md` that
records the properties a module's code is agreed to uphold) and proposes up
to three properties the doc does not currently capture. The skill only
*proposes* — it never edits the invariant doc itself. This gate is the point
where a human decides what to do with each proposal.

The invariant doc is a **protected surface**: a file the repository treats
as governed, so any change to it must go through a deliberate amendment
process rather than a casual edit. This gate exists so that no adversarial
proposal — however plausible-sounding — enters the ratified spec without a
person judging whether it is actually true, actually useful, and worth the
review cost of a separate pull request.

## What you are being asked to decide

For each proposal in the findings file (`.assurance/spec-adversary/<module>-<date>.md`),
tick exactly one box:

- **Accept** — the proposal describes a real property worth documenting.
  Nothing happens automatically: you (or another contributor) must later run
  `/protected-surface-amend` in a **separate pull request** to actually add
  the property to `docs/invariants/<module>.md`, complete with the
  governance note that amendment process requires. Accepting here only
  records the decision — it does not touch the invariant doc.
- **Reject** — the proposal is wrong, already covered, or not worth
  documenting. Write a one-line reason. The proposal is discarded; no
  further action is needed.
- **Defer** — the proposal might be worth documenting later, but not now
  (for example, it depends on other work landing first). Write a one-line
  note on when to revisit it. No file changes happen until someone
  reconsiders it.

You may also receive **zero** proposals for a run. That is a legitimate,
expected outcome — it means the adversary found nothing worth flagging this
time, not that the process failed. No decision is required when there is
nothing to triage.

The skill is capped at three proposals per run by design, so you are never
asked to triage more than three at once.

## How long this takes

Each proposal includes its supporting code lines and reasoning, so triage is
usually a five- to fifteen-minute read per proposal — mostly checking that
the cited code actually behaves as claimed and that the proposal does not
duplicate an existing invariant.

## What happens after you decide

Your ticked boxes stay in the findings file as a record. A maintainer later
copies the accepted/rejected/deferred counts into
`.assurance/spec-adversary-tracker.md`, which tracks this skill's long-run
signal-to-noise ratio (how many proposals turn out to be worth promoting).
That ratio is used to judge whether the adversary is still earning its
review time for this module — it does not change anything about how your
individual triage decision is acted on.

## Who to ask if you're unsure

Raise a GitHub issue on this repository addressed to the Crosscheck
maintainers if a proposal is ambiguous, if you are unsure whether something
is already covered by an existing invariant, or if you are unsure whether a
proposal should go through `/protected-surface-amend` at all.
