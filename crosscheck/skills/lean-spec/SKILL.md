---
name: lean-spec
description: >-
  Translate a signed-off informal specification into a Lean 4 specification
  stub: type definitions mirroring the source, function signatures, key theorem
  declarations with `sorry` proof bodies, and the Mathlib imports needed to make
  the file `lake build` cleanly. Drives the Lean toolchain in a retry loop
  against `lean_check` until the file parses and typechecks. Step 2 of the
  five-step Lean-side pipeline; downstream of `/informal-spec`, upstream of
  `/lean-impl`.
  Triggers: "lean spec", "lean spec stub", "lean 4 specification", "translate
  informal spec to lean", "lean pipeline step 2".
argument-hint: "[module name — e.g. RateLimiter; resolves to formal-verification/specs/<module>_informal.md]"
---

# /lean-spec — Lean 4 Specification Stub from Informal Spec

## Description

Translate a signed-off informal specification (the prose artefact produced by `/informal-spec`) into a Lean 4 *specification stub* that the rest of the Lean pipeline can build on.

The stub is not a proof. It is a type-checked surface that pins down:

- **Type definitions** mirroring the source domain (records, datatypes, structures).
- **Function signatures** for every operation the informal spec names — bodies omitted.
- **`theorem` declarations** stating the preconditions, postconditions, and invariants from the informal spec, with `sorry` in the proof body.
- **Mathlib imports** needed for the types and tactics the stub references.

Per the Lean-side architecture in `docs/research/assurance-hierarchy.md` ("Two engines, two roles"), Lean is the *executable model + DRT oracle* — it is not a verify-and-extract engine like Dafny, and there is no production-grade Lean-to-Python/Go compiler. The stub produced here is the formal-spec end of the model; `/lean-impl` supplies the executable side that the theorems connect to, and `/correspondence-review` then classifies how faithfully the model corresponds to source before `/drt-oracle` runs.

**Hard gate.** The skill returns success only when the file `lake build`s cleanly — every type error fixed, every import resolved. A `sorry` in a proof body is fine; a parse error or typecheck error is not. The skill iterates against the `lean_check` MCP tool until the build is clean or until a 5-attempt retry budget exhausts (matching `/spec-iterate`).

**Why a separate skill from `/informal-spec`.** The Lean-Squad pipeline this is lifted from treats intent extraction and Lean translation as distinct phases because writing Lean before locking the intent compounds errors. The informal spec is the contract; this skill mechanically translates a *signed-off* contract into a buildable Lean surface.

## Instructions

You are translating a signed-off informal specification into a Lean 4 stub. The informal spec is authoritative; your job is mechanical translation plus enough Mathlib knowledge to make the file build. Do not invent properties that are not in the informal spec; do not silently drop properties that are.

### Step 0: Prerequisite Check

Before doing any Lean work, verify the input artefact exists and has been signed off.

1. **Resolve the module name.** If the user supplied a module name as the argument, use it directly. Otherwise, ask: *"Which module's informal spec should I translate? Expected at `formal-verification/specs/<module>_informal.md`."*

2. **Confirm the file exists** at `formal-verification/specs/<module>_informal.md`. If it does not, refuse and direct the user to run `/informal-spec` first:

   > The Lean spec stub depends on a signed-off informal spec at `formal-verification/specs/<module>_informal.md`. That file is not present. Run `/informal-spec` first to produce it, get human sign-off, and then re-run `/lean-spec`.

3. **Confirm sign-off.** Read the file's last section. The `/informal-spec` skill terminates the document with a sign-off line that this skill keys off:

   ```
   Human sign-off: <YYYY-MM-DD>
   ```

   Accept any line in the document's final section that matches the regex `^Human sign-off:\s*\d{4}-\d{2}-\d{2}\s*$` (trailing whitespace allowed). If the line is absent, refuse:

   > The informal spec at `formal-verification/specs/<module>_informal.md` is present but lacks a human sign-off marker. Re-run `/informal-spec` and complete its sign-off step before continuing. The expected marker on the last section is a line matching `Human sign-off: <YYYY-MM-DD>`.

   Do not attempt to fall back to other sign-off conventions (`Approved-by:`, GitHub-style trailers, etc.). The contract between `/informal-spec` and `/lean-spec` is exactly the marker above; either it's there or the upstream skill hasn't done its job.

4. **Confirm the output path is writable; default policy is auto-rename.** The target is `formal-verification/lean/CrosscheckModel/<Name>.lean`, where `<Name>` is the module name in `PascalCase`. If a file already exists at that path, **auto-rename the existing file** to `<Name>_v<n>.lean` (where `<n>` is the lowest unused integer ≥ 2) and proceed with a fresh write at the original path. Report the rename in the Step 5 hand-off so the user sees it. Do not pause to ask — the deterministic policy is "old file renamed, new file at canonical path"; if the user wanted the old file preserved at the canonical path they can revert via git. **A renamed v<n> stub still consumes the same informal spec at `formal-verification/specs/<module>_informal.md` and inherits the same sign-off date** — do not allow a renamed stub to skip Step 0.3 or to be paired with an older informal-spec revision than what is currently signed off. If the user wants the renamed file to follow a different informal spec, that is a fresh `/informal-spec` run, not a `/lean-spec` rename.

5. **Output paths are not protected surfaces (yet).** `formal-verification/specs/` and `formal-verification/lean/` are new directories introduced by sub-phase 3b. They are not Class A or Class B per `crosscheck/.claude/rules/protected-surfaces.md` (the file is the authoritative partition; if it has not yet been added to the repo, treat these directories as unprotected). Do not invoke `/protected-surface-amend` for edits to these paths. If sub-phase 3b-β reclassifies them, that ADR will state so explicitly.

Do not proceed to Step 1 until all five checks pass.

### Step 1: Read and Inventory the Informal Spec

Open `formal-verification/specs/<module>_informal.md` and extract a structured inventory:

- **Domain types.** Records, enums, datatypes the spec names. For each, record the field/constructor list and any cardinality or non-emptiness constraints.
- **Operations.** Function signatures the spec names. For each, record the input types, output type, and any partiality (does it return `Option`/`Except`?).
- **Properties.** Preconditions, postconditions, invariants, and edge-case clauses. Each becomes a candidate `theorem`.
- **Examples.** Concrete worked examples in the spec — these become docstring fixtures, not theorems, but flag any where the example reveals a property the spec text didn't state explicitly.
- **Ambiguities flagged for later.** The `/informal-spec` skill marks these inline; carry them forward as `-- TODO(spec ambiguity):` comments in the Lean file rather than guessing.

Record this inventory in the Lean file header (Step 3 includes it as a structured comment block). Do **not** stop to ask the user to confirm it before writing Lean — the inventory is mechanically derived from the signed-off informal spec, so a re-confirmation gate just duplicates the sign-off that already happened in `/informal-spec`. The inventory becomes reviewable as part of the Lean file in PR review; if the user spots an inventory error then, that signals the informal spec needs revision (re-run `/informal-spec`, then re-run `/lean-spec`).

### Step 2: Plan the Mathlib Surface

For each domain type and property, decide which Mathlib namespaces you need. Common starters:

| Surface | Likely imports |
|---|---|
| Lists, sequences | `import Mathlib.Data.List.Basic` |
| Finite sets / multisets | `import Mathlib.Data.Finset.Basic`, `import Mathlib.Data.Multiset.Basic` |
| Natural-number arithmetic, ordering, `Nat.lt_irrefl` etc. | `import Mathlib.Data.Nat.Defs` |
| Linear arithmetic in proof obligations | `import Mathlib.Tactic.Linarith` |
| Decidability of order / equality on derived types | `import Mathlib.Tactic.DeriveDecidableEq` |
| Real-number / measure-theoretic claims | `import Mathlib.Analysis.SpecialFunctions.Basic` (only if the informal spec explicitly uses reals) |

The harness Docker image pre-builds Mathlib, so `lake build` on a small file completes in seconds. Do not micro-optimise imports — pulling in `Mathlib.Data.List.Basic` is cheap and saves a typecheck-fail loop. If in doubt, import.

### Step 3: Draft the Lean Stub

Write the file at `formal-verification/lean/CrosscheckModel/<Name>.lean`. Structure:

```lean
/-
Module: <Name>
Source informal spec: formal-verification/specs/<module>_informal.md
Sign-off: <YYYY-MM-DD as recorded in the informal spec>
Pipeline step: 2 of 5 (/lean-spec). Next: /lean-impl (3b.4).
-/

import Mathlib.Data.List.Basic
-- ... other imports planned in Step 2

namespace CrosscheckModel.<Name>

-- == Types ===============================================================

structure <DomainType> where
  -- field declarations mirroring the informal spec
  deriving Repr, DecidableEq

-- == Signatures ==========================================================

/-- <one-line description from the informal spec> -/
def <operation> (x : <InputType>) : <OutputType> := sorry

-- == Properties ==========================================================

/-- <property name from informal spec> — <one-line restatement>. -/
theorem <property_name> (x : <InputType>) (h : <precondition>) :
    <postcondition> := by
  sorry

end CrosscheckModel.<Name>
```

Rules for this step:

- **Mirror the source domain in types.** If the informal spec names a `RateLimiterState` with three fields, the Lean structure has those three fields with the same names, transliterated to Lean naming conventions (`camelCase`).
- **Bodies are `sorry`.** Definitions get `:= sorry` (the implementation lands in `/lean-impl`); theorems get `by sorry` proof bodies.
- **One `theorem` per spec property.** Do not bundle multiple properties into a single conjunction — `/lean-impl` and `/correspondence-review` will reference these by name, and granular theorems give DRT scoping that bundled theorems do not.
- **Preserve names.** The theorem name should be a transliteration of the property name from the informal spec, so a reviewer comparing the two files can match line-for-line.
- **No invented properties.** If a property is not in the inventory from Step 1, it does not appear here.

### Step 4: Build and Iterate

Call the `lean_check` MCP tool with the file. The tool runs `lake build` against the file in the Docker harness and returns `{ success, errors, warnings, rawOutput, kind }` where `kind` is one of `parse-error`, `typecheck-error`, `build-error`, `success`.

**Failure-class policy:**

| `kind` | Treat as | Behaviour |
|---|---|---|
| `parse-error` | Must-fix | Iterate. The file cannot be parsed; this is always a translation bug. |
| `typecheck-error` | Must-fix | Iterate. Either a missing import, a wrong type signature, or a malformed `theorem` statement. |
| `build-error` | Must-fix | Iterate. Usually a Mathlib version skew or a missing transitive import; resolvable by adjusting imports. |
| `success` | Done | Even if `warnings` are present. |

**`sorry`-related warnings are NOT failures.** Lean emits `declaration uses 'sorry'` warnings for every `sorry`-bodied theorem and definition. These are *expected proof obligations*, not errors. The whole point of this skill is to land a buildable stub with `sorry`s; downstream skills (`/lean-impl`, the eventual proof skills) discharge them. Do not iterate on `sorry` warnings.

Do **not** call `lean_run`. That tool is for `/lean-impl` (downstream) and runs executable code; this skill only needs `lake build`.

For each `must-fix` failure, apply these repair strategies before retrying:

| Error pattern | Repair |
|---|---|
| `unknown identifier 'X'` from Mathlib | Add the import that defines `X`. |
| `type mismatch ... expected Y got Z` in a theorem statement | Re-read the informal spec — the property's actual statement may have been mis-translated. |
| `unexpected token` / `function expected at` | Parse-level Lean syntax bug; fix and retry. |
| `failed to synthesize instance Decidable ...` | Add `deriving DecidableEq` to the relevant structure, or import `Mathlib.Tactic.DeriveDecidableEq`. |
| `cannot find file ... in search path` | Lake / Mathlib import path issue — check the import string against Mathlib's current module layout via the harness. |

**Retry budget: 5 attempts.** On each attempt:

1. Show the current `errors` (filtered to must-fix kinds).
2. State the repair strategy you are applying.
3. Apply the change to the file.
4. Re-call `lean_check`.

If the budget exhausts, present the best version with remaining errors. Ask the user:

> The Lean stub did not build cleanly within the 5-attempt budget. Remaining errors: `<summary>`. Do you want to (a) adjust the informal spec at `formal-verification/specs/<module>_informal.md` to remove the property that is forcing this failure, or (b) adjust the Lean translation strategy (e.g., model a domain type differently, weaken a theorem statement)?

Do not silently weaken theorems to make the build pass. If a property cannot be stated in Lean as written in the informal spec, that is a finding, not a bug to paper over.

### Step 5: Present and Hand Off

Once `lean_check` returns `success`, present:

- **The file path:** `formal-verification/lean/CrosscheckModel/<Name>.lean`.
- **The build status:** `lake build` clean; warnings limited to `sorry`-uses, count `<N>`.
- **The theorem inventory:** every `theorem` declared, with a one-line restatement of what each one claims (cross-referenced to the informal spec property name).
- **Pipeline step 3 of 5 ready (`/lean-impl`).** State the contract: *"`/lean-impl` translates the source implementation into a Lean functional definition that connects to these theorems and seeds the correspondence document `/correspondence-review` (3b.5) classifies."* The hand-off is mediated by file presence (clean-building Lean stub at the canonical path); whoever drives the pipeline next — orchestrator (`byfuglien`) or human — picks it up. This skill does not assume the human is the driver.

### Step 6: Verification Checklist

Present this checklist alongside the stub:

```
## Verification Checklist

Before proceeding to /lean-impl:
- [ ] Every property in the informal spec has a matching `theorem` in the Lean stub.
- [ ] Every `theorem` name is traceable back to a property in the informal spec by name.
- [ ] No properties were invented during translation.
- [ ] All `-- TODO(spec ambiguity):` markers from the informal spec carried through.
- [ ] `lake build` clean; only `sorry`-related warnings remain.
- [ ] Sign-off date in the file header matches the date in the informal spec.
```

Fill in concrete counts where applicable (e.g., "12 properties translated; 12 theorems declared"). The checklist surfaces what this skill *didn't* do — the proof bodies, the implementation, and the correspondence to source code all lie downstream.

## Arguments

The module name (e.g. `RateLimiter`). The skill resolves the informal spec to `formal-verification/specs/<module>_informal.md` (lowercase / snake_case as written by `/informal-spec`) and writes the Lean stub to `formal-verification/lean/CrosscheckModel/<Name>.lean` (PascalCase).

Example: `/lean-spec RateLimiter`

## References

- `docs/research/assurance-hierarchy.md` — Layer 1 "Two engines, two roles" section. Lean's role as executable-model + DRT oracle, distinct from Dafny's verify-and-extract.
- `crosscheck/skills/spec-iterate/SKILL.md` — sister skill on the Dafny side; same retry budget and verifier-loop pattern.
- `crosscheck/skills/informal-spec/SKILL.md` — upstream skill that produces the input artefact and writes the sign-off marker this skill keys off.
- GitHub Next, *Lean Squad* (Task 3 analogue): https://github.com/githubnext/agentics/blob/main/docs/lean-squad.md — pipeline source for the prose-to-stub-with-`sorry`-bodies pattern.
