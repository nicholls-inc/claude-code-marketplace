---
name: lean-impl
description: >-
  Translate the source implementation into a Lean 4 functional definition that
  the propositions from `/lean-spec` can be connected to. For imperative or
  effectful production code, build a pure functional model and explicitly
  document what the model abstracts away. Step 3 of the Lean-side pipeline;
  downstream of `/lean-spec`, upstream of `/correspondence-review`. Hard gate
  on `lake build` clean. Triggers: "lean impl", "implementation model",
  "translate source to lean", "fill in the lean stub", "discharge sorry
  bodies", "lean pipeline step 3".
argument-hint: "[module name — e.g. RateLimiter; resolves to formal-verification/lean/CrosscheckModel/<Name>.lean and the source it models]"
---

# /lean-impl — Lean 4 Functional Model of the Source Implementation

## Description

Translate the source implementation of a module into a Lean 4 *functional definition* that the `theorem` declarations produced by `/lean-spec` can be connected to. The Lean impl is the executable side of the engine; together with the spec stub it is the model that `/correspondence-review` classifies and `/drt-oracle` uses as the oracle for differential random testing.

Per the Layer 1 architecture in `docs/research/assurance-hierarchy.md` ("Two engines, two roles"), Lean is the *executable model + DRT oracle* — not a code generator. There is no production-grade Lean-to-Python/Go compiler, and this skill does not produce production code. The pattern is: production code is hand- or AI-written (or Dafny-extracted, for partial-verification cases) in a mainstream language, and a Lean 4 model serves as the oracle. This skill builds the model side of that pair.

**Pipeline position.** Step 3 of 5:

```
informal spec  →  formal spec stub (sorry)  →  implementation model  →  correspondence review  →  DRT
   /informal-spec   /lean-spec (3b.3)            /lean-impl (3b.4) ← THIS   /correspondence-review     /drt-oracle
                                                                            (3b.5)                    (3b.6)
```

`/lean-spec` is the immediate upstream; the file produced here is consumed by `/correspondence-review`, which scopes which surface `/drt-oracle` can validly test.

**Hard gate.** The skill returns success only when the appended file `lake build`s cleanly — every type error fixed, every import resolved, every `def` body closed. The whole point of `/lean-impl` is to *replace* the `:= sorry` bodies for definitions left by `/lean-spec`. Theorem `sorry` bodies remain — those are proof obligations, not implementation gaps. The skill iterates against `lean_check` until the build is clean or until a 5-attempt retry budget exhausts.

**Why a separate skill from `/lean-spec`.** The spec stub fixes the *shape* of the model (types, signatures, theorem statements). The impl fills in the *behaviour*. These are different cognitive activities: spec drafting demands fidelity to the prose contract; impl drafting demands fidelity to the source code. Bundling them into one skill blurs which kind of error caused a build failure. The Lean Squad pipeline this is lifted from (Task 4) draws the same boundary for the same reason.

**What this skill does NOT do.** It does not write Lean *proofs* of the theorems from `/lean-spec` — those `sorry` bodies stay as `sorry`. It does not extract the Lean impl back to Python or Go. It does not run differential testing — that is `/drt-oracle`. It does not classify how faithfully the model corresponds to the source — that is `/correspondence-review`, which this skill seeds with a stub.

## Instructions

You are translating a source implementation into a Lean 4 functional model. The source is authoritative for behaviour; your job is to produce a Lean definition the theorems from `/lean-spec` can be applied to. Do not invent behaviour the source does not exhibit; do not silently smooth over imperative or effectful corners — document them.

### Step 0: Prerequisite Check

Before writing any Lean, verify the upstream artefacts exist and are in the expected state.

1. **Resolve the module name.** If the user supplied a module name as the argument, use it directly. Otherwise, ask: *"Which module's Lean spec should I extend with an implementation? Expected at `formal-verification/lean/CrosscheckModel/<Name>.lean`."*

2. **Confirm the Lean spec stub exists** at `formal-verification/lean/CrosscheckModel/<Name>.lean`. If it does not, refuse and direct the user to run `/lean-spec` first:

   > The Lean impl extends a Lean spec stub at `formal-verification/lean/CrosscheckModel/<Name>.lean`. That file is not present. Run `/lean-spec` first to produce it, and then re-run `/lean-impl`.

3. **Confirm the spec stub is buildable.** Read the file. It should contain `def <name> ... := sorry` lines (definitions left for this skill to fill) and `theorem <name> ... := by sorry` lines (proof obligations left for downstream). Call `lean_check` against the current file contents. If the stub does not build cleanly, refuse:

   > The Lean spec stub at `formal-verification/lean/CrosscheckModel/<Name>.lean` does not currently `lake build` cleanly. `/lean-impl` only extends a clean stub; if the spec is broken, fix it via `/lean-spec` first.

4. **Confirm the informal spec sign-off marker is present and unchanged.** Read `formal-verification/specs/<module>_informal.md` and confirm it carries a `Human sign-off: <YYYY-MM-DD>` line matching the regex `^Human sign-off:\s*\d{4}-\d{2}-\d{2}\s*$`. The Lean file's header records this date — confirm the two match. If they disagree, the informal spec was edited after `/lean-spec` last ran:

   > The informal spec's sign-off date `<X>` does not match the Lean stub's recorded sign-off `<Y>`. Re-run `/lean-spec` so the stub picks up the revised informal spec, then re-run `/lean-impl`.

5. **Locate the source implementation.** Ask the user for the path to the source file(s) implementing the module — the production code Lean is modelling. Acceptable shapes:
   - **(a) Single file in a mainstream language** (Python, Go, TypeScript, Rust). Common case.
   - **(b) Dafny-extracted code.** The implementation was generated by `/extract-code`. Note this in the correspondence stub — the model should match the *Dafny source*, not the extraction artefact, unless the user explicitly wants to validate the extraction backend (DRT case (d)).
   - **(c) Multiple files / spread across a package.** Identify the public surface mentioned in the informal spec; ignore helpers not in scope per the informal spec's "Module boundary" section.

6. **Output paths are not protected surfaces (yet).** `formal-verification/lean/` and `formal-verification/correspondence/` are introduced by sub-phase 3b. They are not Class A or Class B per `crosscheck/.claude/rules/protected-surfaces.md` (the file is the authoritative partition; if it has not yet been added to the repo, treat these directories as unprotected). Do not invoke `/protected-surface-amend` for edits to these paths.

Do not proceed to Step 1 until all six checks pass.

### Step 1: Inventory the Source Implementation

Read the source file(s) end-to-end. Produce a structured inventory:

- **Public surface.** Functions/methods named in the informal spec's Module boundary, with their actual signatures (input types, output type, error/option/result wrapping).
- **Computation pattern.** For each function: pure / loop / recursion / mutation / IO / framework callback. Note the pattern explicitly — the impl strategy depends on it.
- **Side-channels.** Any state outside the function arguments that the implementation reads or writes: globals, instance fields, environment, files, network, RNG, clock. List each.
- **Error paths.** What does the implementation do on precondition violation? Does it raise / panic / return an error sentinel / return undefined behaviour?
- **Identifier mapping.** Note where source naming (`snake_case`, `camelCase`, `PascalCase`) will need to translate to Lean's `camelCase` for definitions and `PascalCase` for types.

Present the inventory to the user before writing Lean. They should confirm the inventory matches what they understand the source to do — discrepancies here are usually source-code-reading bugs that compound into translation bugs.

### Step 2: Decide the Modelling Strategy

For each function, decide how the Lean model represents the source's behaviour. Three patterns cover almost all cases:

- **Pure transliteration.** The source is already a pure function of its inputs. Translate one-for-one: `def f (x : T) : U := ...` mirrors the source's control flow with Lean's `match` / `if` / `let` / recursion.
- **Pure model of imperative code.** The source uses loops or mutation but is *behaviourally* a pure function of its inputs. Replace loops with structural recursion or `List.foldl` / `Nat.rec`; replace mutation with rebinding. The Lean impl is the function the imperative code computes — not a step-by-step trace of how it computes.
- **Pure model of effectful code.** The source touches state, IO, RNG, or framework callbacks. Choose the *abstraction line*: which side-channels are passed as explicit inputs to the model, and which are abstracted away. Document every abstraction explicitly in Step 5's correspondence stub. Examples:
  - Wall-clock dependency → take time as an explicit `Nat` input.
  - Filesystem read → take the file's content as a `String` input.
  - RNG → take the random seed as an explicit input, or model as `List Nat` of pre-drawn values.
  - Framework callback → fold callback effects into an explicit input/output pair.

If the abstraction line cannot be drawn cleanly (i.e., you would need to model the entire framework), this is a finding: stop and tell the user the module is not Lean-pipeline-tractable. Recommend `/lightweight-verify` (Layer 2) or property-based testing instead. Do not proceed by silently weakening the model — `/correspondence-review` will catch it but at higher cost.

### Step 3: Plan Mathlib + Stdlib Surface

Inherit the imports already declared by `/lean-spec`. Add only what the impl needs. Common impl-side additions over the spec-side imports:

| Surface | Likely imports |
|---|---|
| Recursive functions on `List`, `Array`, `Nat` | `Mathlib.Data.List.Basic` (likely already present) |
| `List.foldl` / `List.foldr` / `Array.mapIdx` | `Mathlib.Data.List.Basic` / `Mathlib.Data.Array.Basic` |
| Hash-/tree-map analogues for dictionary code | `Mathlib.Data.HashMap.Basic` (or a `Std.HashMap` substitute) |
| Pattern matching on derived ADTs | already covered by `deriving DecidableEq` from spec stub |
| Termination via well-founded recursion | `decreasing_by` tactics; no extra import needed in Lean 4 |
| `Option` / `Except` plumbing | core; no import needed |

Do not micro-optimise imports. The Mathlib pre-warmed Docker image makes import cost trivial; a typecheck-fail loop is far more expensive than a too-broad import.

### Step 4: Append the Implementation to the Lean File

Edit `formal-verification/lean/CrosscheckModel/<Name>.lean` in place. Replace each `def <name> ... := sorry` body with a real definition. Leave every `theorem ... := by sorry` body untouched — those are proof obligations, not impl gaps.

Update the file header to record the pipeline step:

```lean
/-
Module: <Name>
Source informal spec: formal-verification/specs/<module>_informal.md
Source implementation: <path/to/source/file>
Sign-off: <YYYY-MM-DD as recorded in the informal spec>
Pipeline step: 3 of 5 (/lean-impl). Next: /correspondence-review (3b.5).
-/
```

Rules for this step:

- **One definition body at a time.** Replace `:= sorry` for one `def`, then call `lean_check`. Iterating one body at a time keeps build failures small and the repair targeted.
- **Match the source's control flow.** If the source uses a `for` loop over a list, the Lean impl should be a `List.foldl` or structural recursion that visits the same elements in the same order. Behavioural fidelity is the goal; algorithmic fidelity is the means.
- **Termination must be obvious.** If Lean cannot infer termination automatically, supply `decreasing_by` with a clear measure. If you cannot supply one, the source may not actually terminate on all inputs — flag this as a correspondence finding for Step 5.
- **Total functions only.** If the source is partial (raises on some inputs), wrap the Lean output in `Option` or `Except` and document the wrapping in Step 5. Never insert a `panic!`, `unreachable!`, or default-return in the Lean impl just to make it total — that hides correspondence gaps.
- **No `sorry` in `def` bodies.** A leftover `:= sorry` for a definition means the impl is incomplete. The skill cannot return success while any `def` body remains `sorry`.
- **Source-code line citations.** For each non-trivial Lean definition, leave a one-line Lean comment of the form `-- src: <path>:<start>-<end>` immediately above it, pointing at the lines of the source the Lean transliterates. `/correspondence-review` keys off these.

After every body update, call `lean_check` and apply the same failure-class policy that `/lean-spec` uses (`parse-error` / `typecheck-error` / `build-error` are must-fix; `success` is done; `sorry`-related warnings on theorems are expected). Retry budget: 5 attempts per body, plus 5 attempts for whole-file integration after all bodies are filled.

### Step 5: Seed the Correspondence Document

Write `formal-verification/correspondence/<Name>.md` as a *stub*. This is the input artefact for `/correspondence-review`; do not pre-classify the entries — that is the next skill's job. Provide the structure and the raw evidence the next skill needs.

```
# Correspondence: <Name>

Pipeline step: 3 of 5 — seeded by /lean-impl, classified by /correspondence-review (3b.5).

## Modelling decisions

For each abstraction made in Step 2 of /lean-impl, record:

- **Decision.** What aspect of the source is abstracted (loop → fold, RNG → explicit seed, IO → explicit input).
- **Why.** Why this abstraction is faithful to the source's *behaviour* even though it changes the source's *form*.
- **Risk.** What the abstraction would hide if it were wrong (e.g., "if the source RNG is biased, the explicit-seed model would not catch it").

## Definitions to classify

For each Lean `def` in CrosscheckModel/<Name>.lean, list:

- Lean identifier
- Source identifier + file:line range
- One-line description of the source behaviour
- Modelling pattern from Step 2 (pure transliteration / pure model of imperative / pure model of effectful)
- /correspondence-review will classify this as: exact / abstraction / approximation / mismatch

## Open questions for /correspondence-review

- Any abstractions where you (the /lean-impl run) were uncertain whether the model is faithful enough.
- Any place where the Lean impl had to weaken a definition to make `lake build` pass — these are candidate "approximation" or "mismatch" classifications.
- Any source path where you could not locate a definition for a `def` you implemented — these are candidate "mismatch" classifications.

## Source files modelled

- `<path/to/source/file>` (lines `<start>`–`<end>` per definition; full file if module-wide)
```

Do not assign classifications. Do not declare correspondence "exact" — that judgment requires reading the source against the Lean impl with adversarial intent, which is `/correspondence-review`'s job.

### Step 6: Final Build + Hand Off

Once every `def` body is filled and the file builds cleanly, call `lean_check` one more time on the whole file to confirm the integration. Optionally, call `lean_run` on a tiny `main : IO Unit` smoke test that exercises the impl on a literal input from the informal spec's Worked Examples section — this catches "compiles but doesn't compute" cases that pure typechecking misses. Skipping the `lean_run` smoke is acceptable for purely-typed surfaces; for any module with `Nat`/`String`/`List` arithmetic, run it.

Present:

- **The file path:** `formal-verification/lean/CrosscheckModel/<Name>.lean`.
- **The build status:** `lake build` clean; warnings limited to `sorry`-uses on theorems (not definitions), count `<N>`.
- **The definitions filled:** every `def` from `/lean-spec`, with one-line description and source-line citation.
- **The correspondence stub:** path to `formal-verification/correspondence/<Name>.md`, summary of how many definitions are pending classification.
- **Pipeline next step: `/correspondence-review` (sub-phase 3b.5).** State explicitly: *"`/correspondence-review` classifies each Lean definition's correspondence to source as exact / abstraction / approximation / mismatch. `/drt-oracle` (3b.6) only runs on regions classified `exact` or `abstraction`; it skips `approximation` and blocks on `mismatch`. Do not invoke `/drt-oracle` until `/correspondence-review` has run."*

### Verification Checklist

```
## Verification Checklist

Before proceeding to /correspondence-review:
- [ ] Step 0 prerequisite check passed: spec stub exists, builds clean, sign-off date matches informal spec
- [ ] Source-implementation inventory presented to user and confirmed
- [ ] Modelling strategy chosen explicitly per function (pure transliteration / pure model of imperative / pure model of effectful)
- [ ] Every `def := sorry` from /lean-spec replaced with a real definition
- [ ] No `sorry` remains in any `def` body (theorem `sorry` bodies untouched is correct)
- [ ] Every non-trivial definition carries a `-- src:` comment citing the source file:line range
- [ ] Termination handled: every recursive `def` either terminates obviously or has an explicit `decreasing_by` measure
- [ ] Partial source functions wrapped in `Option`/`Except`; no fake totality via `panic!` / unreachable / default-return
- [ ] `lake build` clean on the whole file; only `sorry`-uses on theorems remain as warnings
- [ ] Correspondence stub at `formal-verification/correspondence/<Name>.md` written with definitions listed but unclassified
- [ ] Lean file header updated to "Pipeline step: 3 of 5"
```

## Arguments

The module name (e.g. `RateLimiter`). The skill resolves the Lean stub at `formal-verification/lean/CrosscheckModel/<Name>.lean` and asks the user for the source-implementation path(s).

Example: `/lean-impl RateLimiter`

## References

- `docs/research/assurance-hierarchy.md` — Layer 1 "Two engines, two roles" section; constrains `/lean-impl` to the *executable model*, not a code generator.
- `crosscheck/skills/lean-spec/SKILL.md` — upstream skill that produces the buildable stub this skill extends.
- `crosscheck/skills/correspondence-review/SKILL.md` — downstream skill that classifies the model produced here.
- `crosscheck/skills/extract-code/SKILL.md` — sister Dafny-side skill; extraction is fundamentally different (Dafny generates production code, Lean does not).
- GitHub Next, *Lean Squad* (Task 4 analogue): https://github.com/githubnext/agentics/blob/main/docs/lean-squad.md — pipeline source for the impl-as-pure-functional-model pattern, with explicit documentation of what is abstracted away.
