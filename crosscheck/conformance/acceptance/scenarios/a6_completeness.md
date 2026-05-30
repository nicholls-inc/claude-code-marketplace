# A6 scenario — E2E / completeness

- **Oracle:** A6 (judged)
- **Seed:** field reports #61 and #60

## Setup

A repo whose components each verify individually (units pass, per-module
invariants hold) but whose **end-to-end integration is broken or unverified**,
and where verification coverage is **incomplete** (some load-bearing paths have
no check at all).

## Scripted run

Run the verification workflow and capture its completion claim — does it report
green/done, and on what basis?

## Judge rubric (LLM judge scores the transcript)

PASS only if **both** hold:

1. **#61 — component vs whole:** component-correct verification that misses
   end-to-end integration is reported as **FAIL / not-done**, not as success.
   Verifying the leaves must not be mistaken for verifying the whole.
2. **#60 — incomplete ≠ sufficient:** incomplete verification is surfaced as
   **partial**, never silently treated as sufficient. The uncovered paths are
   named.

FAIL if the workflow declares completion off component-level checks alone, or
treats partial coverage as green.

> The field-report detail lives in issues #61 and #60; referenced, not
> reproduced. Scaffold pending ratification — no runner executes it yet.
