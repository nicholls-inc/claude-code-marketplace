# Gate: draft-invariants red-pen review

## What this gate protects

The `/draft-invariants` skill writes down the rules ("invariants") that a
piece of code must always satisfy — for example, "a queue's length is never
negative". These invariants become the basis for property tests, so a wrong
or sloppy invariant produces tests that pass for the wrong reasons and give
false confidence. This gate is the point where a human reads the invariants
before any test is generated, catching mistakes while they are still cheap
to fix.

## What you are being asked to decide

The skill presents three things as prose: the proposed invariants, a gap
analysis (which invariants might already be violated by the current code,
and which are defended), and a scope-honesty section (what the invariants
deliberately do not cover, and which other part of the system is
responsible for that instead). You are asked to read this and decide
whether it is ready to become the basis for generated tests.

## What each decision means

- **Strike an invariant** — you disagree that it should hold, or it
  describes behaviour outside this module's responsibility. The skill drops
  it and nothing is generated for it.
- **Reword** — the statement is unclear or imprecise. The skill rewrites it
  and shows you the revised version before moving on.
- **Add a failure anchor** — you know of a case the gap analysis missed. The
  skill adds it and re-runs the gap analysis for that case.
- **Sign off** ("ship it", "looks good", "proceed to tests") — you accept
  the invariants as written. The skill moves on to generating property
  tests from exactly what you approved.

Nothing is generated until you sign off. You can iterate through strike/
reword/add as many times as needed first.

## How long this takes

Typically a few minutes — reading a short list of invariants and a gap
analysis, and replying with any changes plus a sign-off phrase.

## A note on batched review

When `/draft-invariants` is run by an orchestrator across many modules at
once (an "orchestrator marker mode" — a repeatable non-interactive run
signalled by a marker left for this module), this per-module gate is
deferred. Instead the orchestrator collects a batched cross-module review
downstream, and property tests are generated only after that batched
review is signed off. The invariant doc is marked `Status: Draft` in the
meantime so it is clear no human has reviewed it yet.

## Who to ask if unsure

The Crosscheck maintainers via a GitHub issue on this repository.
