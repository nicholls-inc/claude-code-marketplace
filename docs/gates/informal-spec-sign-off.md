# Gate: informal spec sign-off

## What this gate protects, and why it exists

Crosscheck is a plugin that helps Claude prove code is correct using formal verification tools, rather than relying only on tests or trust. One of its pipelines turns a plain-English description of what a piece of code should do — an "informal spec" — into a machine-checked model, and eventually compares that model against the real implementation.

That whole chain only produces a trustworthy result if the starting description is right. If the informal spec is wrong or incomplete, every later formal step will faithfully prove the wrong thing. This gate is the point where a human, not the model, confirms the informal spec actually says what they mean before Crosscheck spends further effort building on it.

## What you are being asked to decide

The `/informal-spec` skill writes a structured document describing a module's preconditions (what must be true before it runs), postconditions (what must be true after it runs), invariants (facts that stay true throughout), edge cases, and worked examples. It also lists any "ambiguities" — open questions it could not resolve on its own.

You are asked to read that document end-to-end and decide whether it is ready to hand to the next stage of the pipeline (`/lean-spec`, which turns the prose into a formal Lean model).

## What each decision means

- **`signed off`** — The spec is accurate and complete, and every ambiguity has been resolved (either by editing the spec directly or by answering in your reply). The skill stamps the file with a machine-readable sign-off marker and the pipeline advances automatically to `/lean-spec`. No further action is needed from you at this step.
- **`revise`** — The spec needs changes. Include your notes; the skill will incorporate them and show you the updated spec again for the same decision. This can loop as many times as needed — that is expected, not a failure.
- **`abandon`** — The module is dropped from the formal-verification pipeline. The spec file is kept as documentation, but marked abandoned, and nothing downstream is triggered.

## How long this takes

Reading and signing off a spec typically takes a few minutes to half an hour, depending on the module's complexity and how many ambiguities it raises. Modules with several open ambiguities usually need at least one `revise` round.

## Who to ask if unsure

If anything about the spec, the gate, or the pipeline is unclear, raise it with the Crosscheck maintainers via a GitHub issue on this repository.
