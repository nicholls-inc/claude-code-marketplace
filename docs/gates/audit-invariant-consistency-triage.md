# Gate: audit-invariant-consistency triage

## What this gate protects

Invariants are the machine-checkable rules written down for a codebase
(for example, "a queue's length is never negative"). Different invariant
documents are usually written at different times, by different people, and
can end up disagreeing with each other, or with the spec they are meant to
enforce — one document allows a state transition another forbids, or two
modules define the same shared term two different ways. A contradiction
like this is worse than a gap: it means the invariant set itself cannot be
trusted, and anything checked against it inherits the confusion. The
`/audit-invariant-consistency` skill finds these contradictions by running
three passes over the invariant set: within a single module, across
modules, and (when a spec is supplied) invariants against that spec. This
gate is where a human decides what to do about each contradiction it
surfaces.

## What you are being asked to decide

For each finding (each numbered item, capped at 15 per run so the list
stays reviewable in one sitting, drawn from whichever of the three passes
found it), you choose exactly one of four resolutions.

## What each decision means

- **Accept (fix invariant)** — the contradiction is real and the fix
  belongs in the invariant wording itself. Someone rewrites or splits the
  offending invariant; the spec is unchanged.
- **Accept (amend spec via `/protected-surface-amend`)** — the
  contradiction is real but traces back to the spec, not the invariants
  (for example, the spec itself is ambiguous or wrong). Invariant docs and
  specs are both "protected surfaces" — files this repo treats as
  sensitive enough to require a recorded justification before editing — so
  the change is routed through `/protected-surface-amend` rather than
  edited directly.
- **Reject** — the finding is a false positive (for example, the two
  invariants use the same word but are not actually in tension once read
  in context). Nothing changes; record the reason so a future run does not
  re-raise it without context.
- **Defer** — real or plausible, but not worth acting on now. Record the
  condition under which it should be revisited.

Nothing in the invariant set or spec is changed by the audit itself —
these four choices are a record of what a human decided to do next, not an
automatic action. The findings live in `findings-consistency.md`, and the
follow-through (rewriting the invariant, or the spec amendment) is
separate, manual or orchestrator-driven work.

## How long this takes

A few minutes per finding, typically all findings in one sitting since the
list is capped at 15.

## Who to ask if unsure

The Crosscheck maintainers via a GitHub issue on this repository.
