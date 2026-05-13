---
name: informal-spec
description: >-
  Extract a precise prose specification for a module before any Lean code is
  written — preconditions, postconditions, invariants, edge cases, concrete
  examples, and ambiguities flagged for the user. Step 1 of the Lean pipeline
  (informal spec → Lean spec stub → Lean impl → correspondence review →
  DRT oracle); pairs with `/intent-check` as the spec input to the round-trip
  triple. Hard human sign-off gate before `/lean-spec` runs. Triggers:
  "informal spec", "extract intent", "write prose spec", "specify module
  before Lean", "lean pipeline step 1", "draft prose spec", "informalize a
  module".
argument-hint: "[module path or name | invariant doc path | dafny spec path]"
---

# /informal-spec — Prose Specification for the Lean Pipeline

## Description

Sub-phase 3b.2 of the Crosscheck Lean pipeline. Extracts a precise *informal* (prose) specification for a single module from one of three input shapes:

1. An invariant doc (`docs/invariants/<module>.md`) plus the module signature.
2. An existing Dafny spec (`*.dfy`) you want to lift into Lean.
3. A bare function/method signature plus prose intent supplied by the user.

The output is **prose, not Lean** — preconditions, postconditions, invariants, edge cases, worked concrete examples, and an explicit list of ambiguities for the user to resolve. The skill writes one file: `formal-verification/specs/<module>_informal.md`. It then **stops and asks for sign-off** before any downstream Lean skill runs.

This is the Lean-side analogue of the contract-first methodology used elsewhere in the plugin (see the broader pattern in `/acceptance-oracle-draft` and the contract-elicitation discipline of `/draft-invariants`): lock the prose first, then translate. Racing to Lean before the prose is signed off compounds extraction errors with translation errors and is the single most common Lean-pipeline failure mode (see GitHub Next *Lean Squad*, Task 2).

**Pipeline position.** This skill is step 1 of 5:

```
informal spec  →  formal spec stub (sorry)  →  implementation model  →  correspondence review  →  DRT
   THIS              /lean-spec (3b.3)          /lean-impl (3b.4)         /correspondence-review     /drt-oracle
                                                                          (3b.5)                    (3b.6)
```

`/lean-spec` is the immediate downstream consumer; the full five-step pipeline is shipped as of sub-phase 3b-β.

**Composition with `/intent-check`.** Layer 5 round-trip checking validates a triple of (invariant prose, covering test, code diff). `/informal-spec` produces the *prose* arm of that triple in a form that is consumable by `/intent-check` without further editing — same vocabulary, same scope discipline, same explicit carve-out language. The two skills compose cleanly: `/informal-spec` *writes* the prose; `/intent-check` *audits* the prose against the test and code.

## Instructions

You are extracting a precise prose specification for one module. Your job is to produce a document so unambiguous that translating it to Lean is mechanical, and to stop before any Lean is written.

### Step 0: Assess Lean-Pipeline Fit

Before extracting any spec, assess whether the module is a fit for the Lean pipeline at all. The Lean side of Layer 1 is the executable-model + DRT oracle; it requires the same four module-level prerequisites that the broader VGD methodology requires (see `crosscheck/docs/research/assurance-hierarchy.md` "Framing: layered assurance" for the full list).

Check each prerequisite against the candidate module:

| Prerequisite | Pass / Partial / Fail | Why |
|---|---|---|
| **#1 Deterministic algebraic semantics** | ? | Module behaves as a pure function of its inputs (or can be modelled that way). Heavy framework callbacks, hidden global state, non-determinism in error paths, or behaviour that depends on wall-clock time → fail. |
| **#2 Provable properties** | ? | The module has properties expressible as quantified statements (forall / exists / equality). "Does what the user wanted" is not a provable property; "the output list is a permutation of the input" is. |
| **#3 Tractable input generation** | ? | A random-input strategy can cover the input space well enough that DRT divergences are informative. Unbounded recursive structures, schema-coupled inputs without generators, or external-resource handles → fail. |
| **#4 Dual-development resources** | flagged (D6 hypothesis) | Treat as an untested working assumption under the D6 hypothesis: AI-augmented dual development is plausibly cheap in 2026, but no empirical baseline exists. Mark as such; do not gate on it. |

Output:

- **Mostly pass.** Note the assessment in one paragraph and continue to Step 1.
- **One or more fail (excluding #4).** Present the failure, recommend an alternative path:
  - #1 fail → route to Layer 2–5 (`/lightweight-verify`, `/intent-check`, property-based testing).
  - #2 fail → spec-design problem; recommend `/spec-iterate` or `/draft-invariants`-style prose elicitation first.
  - #3 fail → DRT will not apply downstream; the Lean *spec* may still be valuable as documentation, but `/drt-oracle` will skip the module. Tell the user.
- Never block. The user can override and proceed; record the override in the output file's "Fitness assessment" section.

**Output paths are not protected surfaces (yet).** `formal-verification/specs/` is a new directory introduced by sub-phase 3b. It is not Class A or Class B per `crosscheck/.claude/rules/protected-surfaces.md` (the file is the authoritative partition; if it has not yet been added to the repo, treat this directory as unprotected). Do not invoke `/protected-surface-amend` for edits to it. If sub-phase 3b-β reclassifies the directory, that ADR will state so explicitly.

### Step 1: Identify the Input Shape and Module Boundary

Determine which of the three input shapes you have:

- **(a) Invariant doc + signature** — read `docs/invariants/<module>.md` plus the module's public surface (function/method signatures). Treat the invariant doc as authoritative for *intent*; treat the signatures as authoritative for *types*.
- **(b) Dafny spec** — read the `.dfy` file. Treat each `requires` as a candidate precondition, each `ensures` as a candidate postcondition, each loop `invariant` as a candidate loop invariant, and each `decreases` clause as a termination argument worth restating in prose.
- **(c) Signature + prose intent** — the user supplies the signature and a paragraph of intent. Treat both as draft inputs, not authoritative; you will iterate with the user before locking them.

Record the **module boundary** explicitly: which functions are in scope, which are out of scope (e.g., helpers used by the module but not part of its contract), and what the module depends on (callees whose own contracts are *assumed*, not re-specified, by this skill).

### Step 2: Extract Preconditions, Postconditions, Invariants

For each in-scope function, extract:

- **Preconditions.** What must be true of the inputs and the surrounding state for the function to be defined? State each as a quantified prose statement, with concrete examples of inputs that satisfy it and inputs that violate it.
- **Postconditions.** What does the function guarantee about its output and any state it modifies? Distinguish *functional* postconditions ("the result is sorted") from *frame conditions* ("only field `x` of `s` is mutated").
- **Invariants.** What must hold across iterations of any loop, across recursive calls, or across the lifetime of any data structure the module owns? Number the invariants (`I1`, `I2`, …) so they can be cross-referenced in tests and Lean theorems.
- **Termination.** For recursive or loop-bearing functions, name the well-founded measure that decreases. If you cannot name one, flag it in Step 5 — the module may not terminate on all inputs.

Each item must be expressible as a quantified prose statement; if you find yourself writing "usually" or "typically", you have a property-based test, not an invariant. Move it to a different section or reject it.

### Step 3: Enumerate Edge Cases

List the boundary inputs the spec must address explicitly. Common categories:

- Empty / singleton / two-element collections.
- Numeric extremes (zero, negative, max-int, NaN, infinity if floats are in scope).
- Aliasing (same reference passed twice, output aliasing input).
- Identity inputs (no-op cases — the spec should say what the no-op outputs *are*, not just that they exist).
- Failure inputs (precondition violation — does the function panic, return an error, or is calling it with violated preconditions undefined behaviour?).

For each edge case, state the expected behaviour in one sentence. If the source material does not say what should happen, **do not invent it** — flag it in Step 5.

### Step 4: Worked Concrete Examples

Write 3–5 concrete worked examples. Each example is a tuple of (input, expected output, which preconditions/postconditions/invariants the example exercises). Examples are not tests; they are sanity checks that the prose spec reads correctly when applied to specific inputs.

Pick examples that *together* exercise every numbered invariant from Step 2 at least once. If an invariant has no example, either the invariant is dead or the example set is incomplete.

### Step 5: Ambiguities and Open Questions

This is the most important section. List every ambiguity you encountered while writing Steps 2–4. Categories:

- **Source silence.** The source material does not say what should happen in some case (typically an edge case from Step 3).
- **Source conflict.** Two parts of the source material disagree (e.g., the invariant doc says one thing, the Dafny spec says another).
- **Imprecise language.** A condition is stated in prose that does not translate cleanly to a quantified Lean statement ("reasonably fast", "most inputs", "approximately").
- **Scope question.** Whether something is the responsibility of this module or a caller.

Each ambiguity gets a numbered entry (`A1`, `A2`, …) with a one-sentence question phrased so the user can answer it directly. Do not guess. Do not propose a default. The whole point of the sign-off gate is to force the user to resolve these before Lean code is written.

### Step 6: Write the Output File

Write `formal-verification/specs/<module>_informal.md` with this structure:

```
# Informal specification: <module>

## Fitness assessment
<Step 0 output: per-prerequisite verdict + paragraph + any override notes>

## Module boundary
<in-scope / out-of-scope / assumed callee contracts>

## Input shape
<(a), (b), or (c) + the source artefacts read>

## Preconditions
P1. …
P2. …

## Postconditions
Q1. …
Q2. …

## Invariants
I1. …
I2. …

## Termination
<well-founded measure per recursive / looping function, or flagged in Ambiguities>

## Edge cases
<one bullet per case from Step 3>

## Worked examples
1. input: …, expected: …, exercises: I1, P1, Q2
2. …

## Ambiguities
A1. <question>
A2. <question>

## Pipeline forward references
- /lean-spec (sub-phase 3b.3) — translates this prose into a Lean 4 spec stub with `sorry` proof bodies.
- /lean-impl (sub-phase 3b.4) — Lean functional model of the implementation.
- /correspondence-review (sub-phase 3b.5) — classifies model-vs-source correspondence.
- /drt-oracle (sub-phase 3b.6) — differential random testing against the model.
- /intent-check (Layer 5) — composes with this prose as the spec arm of the (prose, test, code-diff) triple.
```

Do not write any Lean code into this file. Do not write any test code into this file. Prose only.

### Step 7: Sign-Off Gate (HARD STOP)

Once the file is written, present a summary to the user and **stop**. Do not invoke `/lean-spec`. Do not propose Lean translations. Do not pre-emptively answer ambiguities. The sign-off prompt is verbatim:

> The informal spec for `<module>` is written to `formal-verification/specs/<module>_informal.md`. It contains `<n>` preconditions, `<n>` postconditions, `<n>` invariants, `<n>` edge cases, `<n>` worked examples, and `<n>` open ambiguities. Before `/lean-spec` runs, please review the spec end-to-end and confirm: (1) the preconditions/postconditions/invariants are complete and correctly stated, (2) the edge cases enumerate every boundary you care about, (3) every ambiguity in the Ambiguities section has been resolved (either by editing the spec directly or by replying here). Reply `signed off` to release the spec to `/lean-spec`, `revise` with notes for me to incorporate, or `abandon` to drop the module from the pipeline.

If the user replies `signed off`, append a marker line to the spec file's last section in this exact format and on its own line:

```
Human sign-off: <YYYY-MM-DD>
```

The format is load-bearing — `/lean-spec`'s Step 0 prerequisite check uses the literal regex `^Human sign-off:\s*\d{4}-\d{2}-\d{2}\s*$` to gate execution. Use today's date in ISO-8601 (`YYYY-MM-DD`). Do not deviate from this format; "Signed off: ...", "approved", or freeform language will not pass the gate.

Once the marker is written, the file's presence + sign-off date is the orchestrator-readable handoff: byfuglien (or any caller driving the chain) detects the marker and advances to `/lean-spec` automatically. The skill does not need to tell the user "`/lean-spec` is ready" — that treats the user as the workflow driver. Emit a one-line confirmation that the marker was written and the spec is now eligible for pipeline-step 2.

If the user replies `revise`, incorporate the notes and re-present the same prompt — looping is expected and correct.

If the user replies `abandon`, leave the file in place (it is still useful documentation) but mark its first line `Status: abandoned — not advanced to /lean-spec`.

### Verification Checklist

```
## Verification Checklist

- [ ] Step 0 fitness assessment ran against all four VGD prerequisites; failures (other than #4) presented to the user with a routing recommendation
- [ ] Module boundary recorded explicitly (in-scope / out-of-scope / assumed callee contracts)
- [ ] Every precondition, postcondition, and invariant is stated as a quantified prose statement (no "usually" / "typically")
- [ ] Every numbered invariant is exercised by at least one worked example
- [ ] Every edge case from Step 3 has an expected behaviour or is escalated to Ambiguities
- [ ] Every ambiguity is phrased as a direct question to the user; no defaults, no guesses
- [ ] Output file written to `formal-verification/specs/<module>_informal.md` with the Step 6 structure
- [ ] No Lean code, no test code, no implementation in the output file — prose only
- [ ] Forward references to /lean-spec, /lean-impl, /correspondence-review, /drt-oracle present with their sub-phase ids (3b.3 / 3b.4 / 3b.5 / 3b.6)
- [ ] Sign-off gate presented verbatim; the skill stopped and did not invoke /lean-spec
```

## Arguments

A module path, module name, invariant doc path, or Dafny spec path identifying the module to specify. If omitted, the skill asks the user which module to focus on before doing anything else.

Examples:
- `/informal-spec internal/queue` — extract a prose spec for the `queue` module from `docs/invariants/queue.md` plus its Go signature.
- `/informal-spec specs/MaxOfArray.dfy` — lift an existing Dafny spec into prose suitable for the Lean pipeline.
- `/informal-spec` — ask the user which module.

## References

- `crosscheck/docs/research/assurance-hierarchy.md` — "Framing: layered assurance" for the four VGD prerequisites used in Step 0.
- GitHub Next *Lean Squad* (Task 2) — pipeline precedent for separating intent extraction from Lean translation.
- `/intent-check` SKILL.md — downstream consumer of the prose produced here as the spec arm of the round-trip triple.
