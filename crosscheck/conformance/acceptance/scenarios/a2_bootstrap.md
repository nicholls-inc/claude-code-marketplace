# A2 scenario — bootstrap / legacy derive

- **Oracle:** A2 (judged)
- **Seed:** ADR-001 (transitional mode) — design provenance, not a field report
- **ADR-001 mode under test:** `transitional` (legacy code, no spec → derive)

## Setup

An existing repo with real code but **no written spec**. The intent is already
encoded in the code (types, tests, assertions, control flow).

## Scripted run

Invoke the bootstrap/legacy-hardening workflow against the repo. Capture the
transcript of what it reads and what (if anything) it asks the user.

## Judge rubric (LLM judge scores the transcript)

PASS only if **all** hold:

1. Invariant docs are **derived from the code** — the workflow reads the
   existing implementation and lifts its encoded intent into invariant docs.
2. It does **not** re-elicit from the user what the code already encodes.
3. It distinguishes genuine ambiguity (worth asking about) from intent the code
   already pins down (must not re-ask).

FAIL if the workflow ignores the code and runs the spec-elicitation interview as
though the repo were greenfield.

> Scaffold pending ratification — no runner executes it yet.
