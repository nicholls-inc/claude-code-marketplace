# What to do when `/intent-check` reports a fail verdict

## What this gate protects

`/intent-check` checks that a code change, its covering test, and the prose
description of an **invariant** (a rule about the code's behaviour that must
always hold, recorded in a `docs/invariants/` file) are all telling the same
story. It runs two separate AI prompts — one that reads only the code and
test and describes what they seem to guarantee, and a second that compares
that description against the invariant prose — and combines their output
into a verdict: `pass` or `fail`.

A `fail` verdict means the tool believes there is a mismatch somewhere in
that trio: the code does something the invariant doesn't claim, the
invariant claims something the code and test don't cover, or the test is
too weak to prove what the invariant says. `pass` requires both agreement
and high model confidence — a low-confidence match is still recorded as
`fail`, because a shaky "probably fine" should not silently authorise a
protected change. This attestation is written to
`.assurance/intent-check-attestation.json` and can gate a commit through a
companion pre-commit hook.

## What you are being asked to decide

When you see a fail verdict, the skill's report tells you what the code
and test appear to guarantee versus what the invariant prose claims. From
there you choose one of three remediations:

- **Fix the code** — the invariant and test are correct; the implementation
  needs to change to actually satisfy the invariant.
- **Fix the test** — the invariant and code are correct; the test doesn't
  yet exercise the behaviour the invariant describes, so it needs
  strengthening.
- **Amend the invariant** — the code and test are correct and the invariant
  prose is out of date, too strict, or was wrong from the start. This
  route goes through `/protected-surface-amend`, a separate Crosscheck
  skill that requires a written governance note (why the invariant is
  changing and under what authority) before the invariant file — a
  **protected surface**, meaning a file that cannot be edited without that
  extra scrutiny — can be touched.

Only the third option changes what is being verified; the first two change
the thing being checked against a verification that stays fixed. This
matters because it is the difference between "the code doesn't yet meet
the bar" and "the bar itself needs to move" — the second one always needs
the extra paper trail so a reviewer can see why the guarantee changed.

## How long this takes

Reading the mismatch report and picking a route is usually a few minutes.
The remediation itself — fixing code, strengthening a test, or writing a
governance note for an amendment — takes as long as that kind of change
normally takes in this repository.

## Who to ask if unsure

The Crosscheck maintainers via a GitHub issue on this repository.
