# Why `/intent-check` sometimes refuses to run

## What this gate protects

`/intent-check` is a Crosscheck skill that checks whether a piece of code, its
covering test, and the prose description of an **invariant** (a rule about
the code's behaviour that must always hold, written down in a `docs/invariants/`
file) all agree with each other. It does this by asking two separate AI
prompts to reason about the change independently, then comparing what they
say — a "round trip". This is one layer in Crosscheck's assurance hierarchy,
and like any automated check it can be wrong.

To keep track of how reliable it is, every run appends a row to a
**false-positive tracker** — a CSV file at
`.assurance/intent-check-fp-tracker.csv` that records each verdict alongside
a later human classification. When a human reviews a row, they mark it
`genuine` (the tool was right to flag or pass it), `genuine-planted` (right,
on a deliberately seeded test case), `partial` (partly right), or `spurious`
(the tool was wrong — a false positive).

Before doing any work, Step 0 of the skill computes the rolling
false-positive rate over the last 14 days of classified rows. If that rate
is 30% or higher — and at least 3 rows have been classified, so the number
means something — the skill refuses to run at all. This is a **kill
criterion**: a pre-agreed threshold at which a check is judged to be doing
more harm than good and is switched off until it is fixed, rather than
being trusted to limp along.

## What you are being asked to decide

When the refusal fires, a human needs to look at the tracker and decide
what happens next. Concretely: review the classified rows in the tracker
CSV, and either

- **recalibrate** — conclude the round-trip prompts or the underlying model
  need improving, fix them, and let the rate fall back under 30% naturally
  as new, better-classified rows enter the rolling window; or
- **retire** — conclude this check is not worth running for this repository
  or this class of change, and stop relying on it (removing it from any
  required gates) rather than continuing to patch it.

There is no third option of just overriding the refusal — the whole point
of a kill criterion is that it is not optional. If you believe the tracker
itself is wrong (rows misclassified, wrong window), fix the tracker data
and re-run Step 0; the maths will re-evaluate automatically.

## How long this takes

Classifying a handful of tracker rows is usually a 15–30 minute task per
batch, since it means re-reading the original diff and verdict for each
row. Deciding recalibrate vs retire is a judgement call that may need a
short discussion, but rarely more than one meeting.

## Who to ask if unsure

The Crosscheck maintainers via a GitHub issue on this repository.
