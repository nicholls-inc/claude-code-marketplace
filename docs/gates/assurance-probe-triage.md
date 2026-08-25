# Gate: assurance-probe triage

## What this gate is

Crosscheck is a plugin for Claude Code that checks whether code claims and tests actually hold up, using a mix of formal verification and structured reasoning. One of its checks, `/assurance-probe`, asks a narrower question: not "is the code correct?" but "would the tests notice if it broke?"

An **invariant** is a written-down rule a piece of code must always satisfy (for example, "this function never returns a negative index"). Crosscheck stores these as invariant documents, each with a **Failure condition** clause describing the exact condition that would violate the rule.

`/assurance-probe` measures whether the tests covering an invariant would actually catch a violation, using three checks:

- **Mutation probe**: deliberately breaks the code in a small, targeted way (e.g. changes `x < 0` to `x <= 0`) and checks whether any test fails. If nothing fails, the test didn't notice — a genuine weak spot.
- **Vacuity probe**: deletes a test and checks whether code coverage actually drops. If it doesn't, the test wasn't testing anything load-bearing.
- **Generator probe**: checks whether the test's randomised inputs can even reach the risky cases in the first place.

When one of these probes turns something up, the skill opens a **GitHub issue** listing up to three findings and asks a human to triage each one.

## Why the gate exists

None of these probes can tell, on their own, whether a "finding" reflects a real gap in test coverage or an artefact of how the probe works (for example, a mutation the test's input generator could never have produced anyway). Only a human who understands the invariant and the code can make that call. The gate stops the probe's output from being treated as an automatic verdict.

## What you are being asked to decide

For each finding in the issue, choose one of three options:

- **Accept** — the finding is real. The test (or the invariant's Failure condition) needs fixing, and someone should do that as follow-up work.
- **Reject** — the finding is a false positive: the mutation was unreachable, the deleted test wasn't meant to cover that code, or similar. Nothing changes.
- **Defer** — you can't tell yet. Often because confirming it needs a later-phase probe (for example, generator reachability analysis) that hasn't run. Revisit later.

There is no "approve everything" or "block the pipeline" outcome here — `/assurance-probe` runs on a rotation, not per pull request, so nothing else is waiting on your answer. Accepting creates follow-up work; rejecting or deferring simply records the outcome.

## What happens after each decision

- **Accept**: the test or invariant gets fixed in ordinary follow-up work (not blocked by this gate). The tracker records one more "accepted" row.
- **Reject**: no code changes. The tracker records one more "rejected" row — and that is useful, not wasted, effort (see below).
- **Defer**: the finding is logged and left open for the next probe phase or a future run. The tracker records one more "deferred" row.

## Why a rejected finding still matters: the SNR kill criterion

Every triage decision — including every "Reject" — feeds a rolling **signal-to-noise ratio (SNR)**: accepted findings (signal) against rejected findings (noise), tracked in `.assurance/probe-tracker.csv`. If a module's SNR falls below 1:5 over four weeks (minimum 20 runs), the probe retires itself for that module and records why.

This means "Reject" is not a wasted or throwaway decision. It is exactly the data the probe needs to notice when it is producing more noise than value for a given module, and to stand down gracefully rather than keep generating findings nobody can act on. Honest rejects protect the signal-to-noise ratio for everyone downstream.

## How long this takes

A few minutes per finding — read the invariant, the mutation or coverage delta shown, and the proposed test command; decide which of the three boxes applies. There's no urgency: nothing else in the pipeline is blocked while findings sit untriaged, but stale findings make the SNR tracker less useful, so triage promptly when you see the issue.

## Who to ask if unsure

The Crosscheck maintainers via a GitHub issue on this repository.
