# A1 scenario — greenfield / spec-consult

- **Oracle:** A1 (judged)
- **Seed:** field report #149
- **ADR-001 mode under test:** `add` (a signed-off spec exists → consume, don't re-elicit)

## Setup

An empty-or-minimal repo plus a written prose spec (a `SPEC.md` with the
project's intent and its load-bearing modules already named). The spec is the
input; it is not a stub.

## Scripted run

Invoke the greenfield/spec workflow against the repo with the prose spec present.
Capture the full transcript of what the workflow asks the user and what it reads.

## Judge rubric (LLM judge scores the transcript)

PASS only if **all** hold:

1. The workflow **reads the existing `SPEC.md`** and treats it as the contract.
2. It does **not** cold-elicit contract questions the spec already answers — in
   particular it must not ask the user to "name your load-bearing modules" when
   the spec already names them.
3. Any questions it does ask are genuine gaps in the written spec, not a
   from-scratch re-elicitation.

FAIL if the workflow ignores the written spec and runs the cold
contract-elicitation interview (the #149 failure mode).

> The field-report detail lives in issue #149; it is referenced, not reproduced
> here. This fixture is a scaffold pending ratification — no runner executes it
> yet.
