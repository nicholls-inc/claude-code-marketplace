---
name: correspondence-review
description: >-
  Review each Lean definition produced by `/lean-impl` against the actual
  source code and classify correspondence as exact, abstraction, approximation,
  or mismatch — with file:line citations. Documents divergences and assesses
  impact on any proved theorems. Step 4 of the Lean-side pipeline; downstream
  of `/lean-impl`, upstream of `/drt-oracle`. The classification scopes which
  surface `/drt-oracle` can validly test. Triggers: "correspondence review",
  "lean-vs-source check", "model matches production", "classify lean
  correspondence", "lean pipeline step 4".
argument-hint: "[module name — e.g. RateLimiter; resolves to formal-verification/lean/CrosscheckModel/<Name>.lean and formal-verification/correspondence/<Name>.md]"
---

# /correspondence-review — Lean Model vs Source Correspondence Classification

## Description

For each Lean definition produced by `/lean-impl`, classify how faithfully the model corresponds to the actual source code: **exact**, **abstraction**, **approximation**, or **mismatch**. Document divergences with file:line citations and assess whether each divergence invalidates any of the theorems stated by `/lean-spec`.

Per the Layer 1 architecture in `docs/research/assurance-hierarchy.md`, the Lean model is the *oracle* for differential random testing: production code is the system under test, the Lean definition is the spec it is being checked against. An oracle is only valid where the model and source actually correspond. A model classified `approximation` over some region cannot be a DRT oracle for that region — divergences will be model-vs-source mismatches, not bugs. The correspondence document this skill produces is therefore a *prerequisite input* to `/drt-oracle`: it scopes which surface DRT can validly test.

**Pipeline position.** Step 4 of 5:

```
informal spec  →  formal spec stub (sorry)  →  implementation model  →  correspondence review  →  DRT
   /informal-spec   /lean-spec (3b.3)            /lean-impl (3b.4)         /correspondence-review     /drt-oracle
                                                                           (3b.5) ← THIS              (3b.6)
```

`/lean-impl` is the immediate upstream; the classified document is consumed by `/drt-oracle`, which uses it to scope, skip, or block.

**Classification semantics (Lean Squad Task 6 categories, adopted verbatim).**

| Class | Meaning | Effect on /drt-oracle |
|---|---|---|
| **exact** | The Lean definition is a one-for-one transliteration of the source's behaviour. Same control flow, same arithmetic, same edge-case behaviour. Any divergence DRT finds is a real bug. | DRT runs; divergences are bugs. |
| **abstraction** | The Lean definition deliberately omits some source behaviour because it is irrelevant to the property under test (e.g., logging, telemetry, cache layer over a deterministic computation). The model still computes the same output for the same input. | DRT runs; divergences are bugs in the *non-abstracted* behaviour. |
| **approximation** | The Lean definition computes something *similar* to the source but not identical (e.g., real-number model of floating-point source, abstract domain over concrete one). Divergences may be model artefacts, not bugs. | DRT skipped; the region is flagged. |
| **mismatch** | The Lean definition computes something different from the source. The model is broken or the source has been edited since `/lean-impl` ran. | DRT blocked; an issue is opened against the upstream skill. |

**Why this matters for DRT.** Without the classification, `/drt-oracle` cannot distinguish "model bug" from "production bug" when a divergence is found. The Lean Squad pipeline this is lifted from (Task 6) is explicit on this: "any mismatch opens an issue" — DRT against a `mismatch` region is wasted compute and noisy signal.

**Why this is a separate skill from `/lean-impl`.** `/lean-impl` writes the model; this skill audits it adversarially. Same code, different cognitive frame: impl mode trusts the source and chases `lake build` clean; review mode distrusts the model and chases divergences. Bundling these into one skill blurs which kind of error caused a finding; Lean Squad draws the same boundary.

**What this skill does NOT do.** It does not classify with optimistic bias — uncertain entries are downgraded, not promoted. It does not propose fixes for the mismatches it finds; whether a divergence should be repaired in the model, the source, or the informal spec is the user's call. It does not run differential testing — that is `/drt-oracle`, gated on this skill's verdict. It does not discharge `theorem` `sorry` bodies or audit any proof-side artefact other than impact assessment of theorem statements against the classifications produced here.

## Instructions

You are auditing a Lean 4 functional model against its source implementation. The source is authoritative; your job is to find every place the model diverges and classify each divergence by impact. Do not paper over divergences with optimistic classifications; if you are uncertain whether a definition is `exact` or `abstraction`, classify it `abstraction` and document the uncertainty.

### Step 0: Prerequisite Check

1. **Resolve the module name.** If the user supplied a module name, use it. Otherwise, ask: *"Which module should I review the correspondence for? Expected at `formal-verification/lean/CrosscheckModel/<Name>.lean` with a stub at `formal-verification/correspondence/<Name>.md`."*

2. **Confirm the Lean impl exists** at `formal-verification/lean/CrosscheckModel/<Name>.lean`, has its header set to "Pipeline step: 3 of 5", and contains no `:= sorry` definition bodies (theorem `sorry` bodies are expected and acceptable). If any `def` body is still `sorry`, refuse:

   > The Lean impl at `formal-verification/lean/CrosscheckModel/<Name>.lean` still has unfilled definition bodies. Run `/lean-impl` to completion before invoking `/correspondence-review`.

3. **Confirm the correspondence stub exists** at `formal-verification/correspondence/<Name>.md` with the structure seeded by `/lean-impl` (Modelling decisions / Definitions to classify / Open questions / Source files). If the stub is absent, refuse and direct the user back to `/lean-impl`.

4. **Confirm `lake build` is clean.** Call `lean_check` against `CrosscheckModel/<Name>.lean`. If the file does not currently build, refuse — correspondence review against a non-building model is meaningless.

5. **Confirm the source files referenced in the stub still exist** at the paths recorded by `/lean-impl`. If any source path is missing or has been moved, the model is stale; recommend re-running `/lean-impl`.

6. **Output paths are not protected surfaces (yet).** `formal-verification/correspondence/` is introduced by sub-phase 3b. Same partition note as `/lean-impl`.

Do not proceed to Step 1 until all six checks pass.

### Step 1: Pair Each Lean Definition with its Source

For each `def` in `CrosscheckModel/<Name>.lean`:

- Locate the `-- src:` comment `/lean-impl` left immediately above it. This points at `<path>:<start>-<end>`.
- Open the source file at that range. Read the lines.
- Confirm the source-side identifier matches the description in the correspondence stub. If a `def` has no `-- src:` comment, or the cited range no longer matches the named identifier, that is the first finding to record (candidate `mismatch`).

If a Lean definition has no source-side counterpart at all, that is also `mismatch`: the model has gained behaviour the source does not have. Examples: a helper function the impl skill invented; a Mathlib-flavoured definition introduced because the source pattern did not translate cleanly.

If a source-side function in the informal spec's Module boundary has no Lean counterpart, that is *also* a finding — the model is incomplete. Record it as a candidate `mismatch` (an absent definition cannot be DRT-tested, and the model does not cover the spec's surface).

### Step 2: Classify Each Definition

For each Lean–source pair, work through the classification rubric in order:

1. **Could this Lean def be replaced with a literal transliteration of the source?** If yes, and the existing def *is* such a transliteration (same operations in the same order over the same data shape), the class is **exact**.

2. **If not exact, what differs?** Walk through every place the Lean control flow, arithmetic, error path, or data layout differs from the source.
   - **Side-channels the model does not represent.** Logging, telemetry, caching, retry loops, observability hooks. If removing these from the source would not change its observable input/output behaviour for in-spec inputs, the class is **abstraction**. Document the side-channel.
   - **Imperative-vs-functional restructuring with same input/output.** Source uses a `for` loop, model uses `List.foldl`. Same elements visited in the same order, same accumulator. Class is **exact** if the result is provably the same on all inputs; class is **abstraction** if the model assumes the source's loop never short-circuits in a way the fold cannot model.
   - **Mathematical idealisation.** Source uses `float64`, model uses `Real`. Source uses bounded `int32`, model uses `Int`. Class is **approximation**: divergences may be IEEE-754 vs real-number mismatches, not bugs.
   - **Behavioural disagreement on at least one in-spec input.** The model and source compute different outputs on some input the informal spec admits. Class is **mismatch**.

3. **Edge cases.**
   - **Source has a bug the model does not.** This is `mismatch` — the model "fixes" the source. DRT should not run; the bug should be filed and the model brought into sync. Note explicitly: it is not `/correspondence-review`'s job to decide whether the source bug should be fixed in source or in model.
   - **Model has a bug the source does not.** Also `mismatch`. The Lean impl should be revised before DRT runs.
   - **Source is non-deterministic; model is deterministic.** Class is `approximation` if the non-determinism is over a parameter the model takes as an explicit input (e.g., RNG seed), `mismatch` otherwise.

Record the rubric step that drove each classification — do not collapse the reasoning to a single word.

### Step 3: Assess Theorem Impact

For each `theorem` declared by `/lean-spec` and still bearing a `sorry` body:

- Identify which `def`s the theorem statement quantifies over.
- For each such `def`, look up its classification from Step 2.
- If any covered `def` is `mismatch`, the theorem is *vacuously unprovable* against the current model — `sorry` masks a structural problem, not a missing proof. Note this; downstream proof skills will need the model fixed first.
- If any covered `def` is `approximation`, the theorem may not be provable in the strict sense even if the source is correct (e.g., a postcondition stated in `Real` over a `float64` source). Note the soft impact.
- If all covered `def`s are `exact` or `abstraction`, the theorem is a well-formed proof obligation against the model — no correspondence-driven blocker.

This is impact assessment, not proof. The skill does not discharge `sorry` bodies.

### Step 4: Write the Classified Document

Replace `formal-verification/correspondence/<Name>.md` with the classified version:

```
# Correspondence: <Name>

Pipeline step: 4 of 5 — classified by /correspondence-review.

## Verdict summary

| Class | Count | DRT effect |
|---|---|---|
| exact | <n> | runs |
| abstraction | <n> | runs (modulo abstracted side-channels) |
| approximation | <n> | skipped, flagged |
| mismatch | <n> | blocks DRT until resolved |

Overall DRT readiness: <READY / READY-WITH-SKIPS / BLOCKED>.

## Definitions

### <LeanIdent> — class: <exact / abstraction / approximation / mismatch>

- **Source.** `<path>:<start>-<end>` — `<source identifier>`.
- **Behaviour.** <one paragraph: what the source does, what the Lean def does, the precise relationship between them>.
- **Rubric step that drove the class.** <reference Step 2.X>.
- **Divergences.** <bullet list of every place the Lean def differs from a literal transliteration; cite source line numbers>.
- **Theorem impact.** <list of theorems that quantify over this def, with the impact verdict from Step 3>.
- **DRT note.** <what /drt-oracle should do with this def>.

(repeat for every definition)

## Modelling decisions (carried forward from /lean-impl)

<the Modelling decisions section from the stub, lightly edited if Step 2 surfaced new context>

## Mismatch issues opened

For every `mismatch` classification, open or reference a tracking issue. The issue must capture:

- The Lean ident
- The source file:line
- A one-paragraph summary of the disagreement
- A reproducible witness if one is obvious by reading (otherwise mark "DRT will witness")
- Owner: `/lean-impl` (model bug) or `/spec-iterate`-style spec-revision (source bug) or "needs human triage"

(no `mismatch` ⇒ "No mismatch issues. DRT may run on all `exact` and `abstraction` regions.")

## Open questions for /drt-oracle

- Which `abstraction` regions need the abstracted side-channels held constant during DRT (i.e., the harness must not vary them).
- Which `approximation` regions should be skipped silently vs flagged in the DRT report.
- Any cross-definition coupling that DRT cannot test by exercising one definition at a time.
```

### Step 5: Mismatch Gate

If any definition is classified `mismatch`, **stop the pipeline**:

> Correspondence review found `<n>` mismatch classification(s). `/drt-oracle` does not run while mismatches are open — testing against a mismatched model produces noise, not signal. Resolve each mismatch by either:
>
> 1. Re-running `/lean-impl` with corrected modelling assumptions (model bug), or
> 2. Filing a source-side fix and re-running `/lean-impl` after the fix lands (source bug), or
> 3. Adjusting the informal spec via `/informal-spec` if the divergence reveals an unstated assumption (spec bug).
>
> Do not invoke `/drt-oracle` until every mismatch is resolved. The blocked classifications are listed in `formal-verification/correspondence/<Name>.md` under "Mismatch issues opened".

If there are no mismatches, present the verdict summary and hand off:

> Correspondence review complete. `<n>` definitions classified: `<m>` exact, `<k>` abstraction, `<j>` approximation, 0 mismatch. `/drt-oracle` is unblocked; it will run on the `exact` and `abstraction` regions and skip the `approximation` regions with a flag.

### Step 6: Verification Checklist

```
## Verification Checklist

Before /drt-oracle runs:
- [ ] Step 0 prerequisite check passed: Lean impl exists, builds clean, no `def := sorry`, source paths resolve
- [ ] Every `def` in CrosscheckModel/<Name>.lean has a Lean–source pair recorded (or is recorded as a model-only `mismatch`)
- [ ] Every source-side function in the informal spec's Module boundary has a Lean counterpart (or is recorded as an absent-counterpart `mismatch`)
- [ ] Every classification cites the rubric step that drove it (no naked one-word verdicts)
- [ ] Every divergence cites source `file:line`
- [ ] Theorem impact assessed for every theorem with a `sorry` body
- [ ] Verdict summary table emitted with counts and DRT-readiness
- [ ] Mismatch issues opened (or "No mismatch issues" recorded) in the doc
- [ ] If any mismatch present, the skill stopped and did not hand off to /drt-oracle
- [ ] If no mismatch, the user has been told /drt-oracle is unblocked and what it will skip
```

## Arguments

The module name (e.g. `RateLimiter`). The skill resolves the Lean impl at `formal-verification/lean/CrosscheckModel/<Name>.lean` and the seeded correspondence doc at `formal-verification/correspondence/<Name>.md`.

Example: `/correspondence-review RateLimiter`

## References

- `docs/research/assurance-hierarchy.md` — Layer 1 "Two engines, two roles"; constrains the Lean side as the *oracle*, which is only valid where the model corresponds to source.
- `crosscheck/skills/lean-impl/SKILL.md` — upstream skill that produces the model and seeds the correspondence stub.
- `crosscheck/skills/drt-oracle/SKILL.md` — downstream skill that consumes the classified document; runs only on `exact` / `abstraction` regions.
- `crosscheck/docs/research/crosscheck-tla-vgd-addendum.md` — D7a (DRT generalises ~15/21 general bugs) and D7b (VGD-as-methodology Amazon scopes narrowly); the correspondence-as-DRT-prerequisite claim is what makes the D7a generalisation operationally usable.
- GitHub Next, *Lean Squad* (Task 6 analogue): https://github.com/githubnext/agentics/blob/main/docs/lean-squad.md — pipeline source for the four-class rubric and the mismatch-opens-issue gate.
