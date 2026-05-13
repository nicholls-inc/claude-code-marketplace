---
name: drt-oracle
description: >-
  Differential random testing between a Lean 4 model (the oracle) and a
  production implementation (the system under test), scoped by the
  classification document produced by `/correspondence-review`. Step 5 of the
  Lean-side pipeline; final step. Generates random inputs, executes both
  implementations, compares outputs, and reports divergences with witness
  inputs and minimal repros. Triggers: "drt", "differential random testing",
  "drt oracle", "fuzz against the lean model", "lean as oracle", "lean
  pipeline step 5".
argument-hint: "[module name — e.g. RateLimiter; resolves to formal-verification/lean/CrosscheckModel/<Name>.lean and formal-verification/correspondence/<Name>.md]"
---

# /drt-oracle — Differential Random Testing with a Lean Model as Oracle

## Description

Differential random testing (DRT) between a Lean 4 model and a production implementation: generate random inputs, execute both implementations on each input, compare outputs, and report any divergence with the witness input and a minimal repro. The Lean model is the *oracle*; the production code is the system under test.

Per the Layer 1 architecture in `docs/research/assurance-hierarchy.md` ("Two engines, two roles"), Lean is the *executable model + DRT oracle*, distinct from Dafny's verify-and-extract role. The pattern is: hand- or AI-write production code in a mainstream language, build a Lean model alongside via `/informal-spec` → `/lean-spec` → `/lean-impl`, classify correspondence via `/correspondence-review`, then DRT here. Lean has no production-grade compiler to mainstream languages — the model never becomes the production code, it only ever serves as the oracle.

**Pipeline position.** Step 5 of 5 (final):

```
informal spec  →  formal spec stub (sorry)  →  implementation model  →  correspondence review  →  DRT
   /informal-spec   /lean-spec (3b.3)            /lean-impl (3b.4)         /correspondence-review     /drt-oracle
                                                                           (3b.5)                    (3b.6) ← THIS
```

The classified correspondence document is a **prerequisite input**; this skill runs *scoped by* the classifications, not over the whole model.

## When DRT applies (D4 input-space partition)

DRT is operationally meaningful in five distinct cases. For each module under test, classify which case applies — the harness construction depends on it. (These are the D4 cases from `docs/research/crosscheck-tla-vgd-addendum.md`; quoted near-verbatim.)

- **(a) Hand- or AI-written production code with no formal verification** (Cedar's exact pattern: Lean model vs Rust). The most common case. Production code is the SUT; the Lean model from `/lean-impl` is the oracle.
- **(b) Systems composing Dafny-verified kernels with non-verified glue.** Per-kernel proofs do not cover the composition. DRT exercises the glue and the kernel-glue interfaces.
- **(c) Chains of independently-verified kernels.** End-to-end semantics are not proven by per-kernel proofs. DRT exercises the chain.
- **(d) Cross-extract validation: Python-extract vs Go-extract from the same Dafny.** Validates the extraction backends. Addresses the "reach ceiling" concern about Dafny extraction (review item B2). Both arms are SUT; the Dafny source (or its Lean transliteration) is the oracle.
- **(e) Per-method partial verification: a function with verified preconditions but unverified postconditions.** DRT exercises only the unverified surface. Inspect Dafny verification scope at method granularity.

**Excluded: fully Dafny-verified slices.** DRT is redundant — the slice already has compile-time correctness. The skill flags these and skips them.

## D7a / D7b — what this skill claims and what it does not

DRT-as-technique generalises beyond Cedar's authorization domain. Cedar's bug taxonomy (Disselkoen et al. 2024) shows ~15/21 of DRT-found bugs were *general implementation bugs* — parsing, dependencies, error handling, naming inconsistencies — not authorization-specific (D7a). This skill operationalises that broad applicability.

It does not claim the *VGD methodology* generalises broadly. Amazon scopes Verification-Guided Development to safety-critical systems and lists four prerequisites (deterministic algebraic semantics, provable properties, tractable input generation, dual-development resources); the paper's only generalisation gesture is conditional ("could be used"). This skill does not assume those prerequisites for adopters; the per-module routing in `/assurance-init` Step 6.5 and `/assurance-layer-audit` Step 4.5 is where the prerequisite check happens (D7b).

## When DRT may not apply (the four VGD prerequisites)

Even within the five D4 cases, DRT is only useful when the module clears all four prerequisites the informal-spec skill checked at module-fitness time:

1. **Deterministic algebraic semantics.** If the module's behaviour depends on wall-clock time, framework callbacks, hidden global state, or environmental non-determinism that the model cannot hold constant, DRT divergences will be non-determinism artefacts, not bugs.
2. **Provable properties.** If the module's spec is "does what the user wanted" without quantified properties, there is no oracle behaviour to compare against. Lean Squad's pattern of `def`-based oracles requires the spec to be expressible as a function.
3. **Tractable input generation.** If the input space cannot be sampled in a way that covers the in-spec inputs (e.g., schema-coupled inputs without generators, external resource handles, unbounded recursive structures), DRT will hit shallow surfaces only.
4. **Dual-development resources.** Hypothesised under D6 to be substantially reduced by AI-augmented development; treat as untested working assumption.

If `/informal-spec`'s Step 0 fitness assessment recorded a fail on (1), (2), or (3), `/drt-oracle` should refuse and route the user back to property-based testing or `/lightweight-verify` (Layer 2).

## Why correspondence-scoped

The Lean model is an oracle only where it actually corresponds to the source. A region classified `approximation` by `/correspondence-review` will produce divergences that are model artefacts, not bugs (e.g., a `Real`-domain Lean model over a `float64` source). A region classified `mismatch` will produce divergences that are model bugs, not source bugs. DRT against either is wasted compute and noisy signal.

This skill therefore:

- **Runs DRT** on regions classified `exact`.
- **Runs DRT** on regions classified `abstraction`, holding the abstracted side-channels constant per the correspondence doc's "Open questions for /drt-oracle" section.
- **Skips** regions classified `approximation`, with the skip reported in the divergence report so the user knows what was not tested.
- **Refuses to run** if any region is classified `mismatch`. Mismatches must be resolved upstream (re-run `/lean-impl` or fix the source) before DRT is meaningful.

## What this skill does NOT do

- It does not generate code. The harness it produces is a fuzz driver, not a compiled artefact.
- It does not prove anything. Witness-finding is a sound bug-finding technique; *not finding* a witness is not a soundness guarantee.
- It does not classify model-vs-source correspondence — that is `/correspondence-review`'s job; this skill consumes the classification.
- It does not implement the Aeneas / Charon Route A pattern (mechanically deriving Lean from Rust). Aeneas requires Rust source and the Charon + Aeneas toolchain; the Crosscheck plugin does not currently ship the integration. Adopters with Rust codebases should be aware this is a known gap, not an oversight. See "Aeneas alternative" in References.

## Instructions

You are running differential random testing with a Lean model as oracle and production code as SUT. Your job is to construct a harness, drive both implementations on the same random inputs, and report divergences with witnesses. Do not interpret divergences yourself — report them; classification of "real bug vs model artefact" was already done by `/correspondence-review`.

### Step 0: Prerequisite Check

1. **Resolve the module name.** If the user supplied a module name, use it. Otherwise, ask: *"Which module should I run DRT on? Expected at `formal-verification/lean/CrosscheckModel/<Name>.lean` with a classified correspondence doc at `formal-verification/correspondence/<Name>.md`."*

2. **Confirm the Lean impl exists** at `formal-verification/lean/CrosscheckModel/<Name>.lean`, builds clean (call `lean_check`), and has its header set to "Pipeline step: 4 of 5" or later. If the impl is missing or does not build, refuse and route the user to the upstream skill.

3. **Confirm the classified correspondence doc exists** at `formal-verification/correspondence/<Name>.md`, has its pipeline-step header set to "4 of 5", and contains a "Verdict summary" table. If the doc is unclassified (still "3 of 5") or absent, refuse:

   > DRT requires a classified correspondence document. Run `/correspondence-review <Name>` first.

4. **Mismatch gate.** Read the verdict summary. If `mismatch > 0`, refuse:

   > Correspondence review found `<n>` mismatch classification(s). DRT does not run against mismatched models — divergences would be model bugs, not source bugs. Resolve the mismatches per the doc's "Mismatch issues opened" section, then re-run.

5. **D4 case selection — infer from upstream artifacts.** Determine which of the five D4 cases (a–e above) applies by reading the correspondence doc and the source paths it records. Do not cold-ask the user; ask only when inference produces an ambiguous result.

   Inference rules (apply in order; first match wins):

   - The correspondence doc's "Source files modelled" section lists exactly one path under a mainstream-language directory (`*.py`, `*.go`, `*.ts`, `*.rs`) and no `_dafny` imports are present → **(a)**.
   - The recorded source path is annotated as Dafny-extracted (presence of `_dafny` imports, or `// dafny.dtr` annotations, or the correspondence doc's input shape says "(b) Dafny-extracted code") → **(e)** per-method partial verification, *unless* a second extract exists (Python and Go both present for the same Dafny source) in which case → **(d)** cross-extract validation.
   - Multiple source paths under different kernel directories with composition annotations in the correspondence doc → **(b)** or **(c)** depending on whether the doc names the composition explicitly.
   - All other cases → emit a `REQUIRES HUMAN VERIFICATION:` line in the divergence report header and default to **(a)** with the assumption documented.

   The user may override via an explicit `--d4-case=<a|b|c|d|e>` argument. The skill records the inferred case + its evidence in the divergence report's header.

6. **Locate the production implementation.** This is the SUT path the user gave to `/lean-impl`. Confirm the file still exists at the path recorded in the correspondence doc. For case (d), there are two SUT paths.

7. **Skip-list construction.** From the verdict summary, build the skip list: every Lean ident classified `approximation`. The harness will report these as "not tested (approximation)" rather than running them.

8. **Output paths are not protected surfaces (yet).** `formal-verification/tests/` and `formal-verification/correspondence/` are introduced by sub-phase 3b. Same partition note as the upstream Lean-pipeline skills.

Do not proceed to Step 1 until all eight checks pass.

### Step 1: Generate the Lean Runner

For each Lean `def` that will be DRT-tested (i.e., classified `exact` or `abstraction` and not on the skip list), generate a thin `main : IO Unit` runner that:

- Reads one input from stdin in a stable JSON-like serialisation (literal Lean `Repr` is acceptable for in-tree types; for richer surfaces, use `Mathlib.Data.Json` or a hand-written parser).
- Calls the Lean def on the parsed input.
- Prints the result to stdout in a stable form (`IO.println (toString result)` or equivalent).
- Exits 0 on success, non-zero on a parse error or a Lean runtime exception.

Write the runner(s) to `formal-verification/lean/CrosscheckModel/<Name>Runner.lean`. Confirm it builds and runs by calling `lean_run` with a literal input from the informal spec's Worked Examples section.

The runner is a CLI surface. The harness in Step 3 will invoke it once per test input via `lake exe` (or equivalent).

> **Implementation note.** The MCP server's `lean_test` tool is currently a `lake build` alias (per `crosscheck/mcp-server/src/tools/leanTest.ts`); it is suitable for compile-time `#guard` checks against literal fixtures but not for the random-input loop this skill runs. Use `lean_run` against the `<Name>Runner.lean` file via the harness rather than expecting `lean_test` to fuzz. Compile-time `#guard` fixtures over the worked-example inputs are still useful as a sanity check on the runner; emit them in `<Name>Runner.lean` if helpful.

### Step 2: Generate the SUT Adapter

Mirror the runner shape on the production side: one CLI per def-under-test, taking the same JSON input on stdin, producing the same output on stdout, exiting non-zero on failure.

Cases:

- **(a)** SUT adapter is a thin wrapper around the production code in its own language (Python script that imports the function; Go binary that calls it; etc.). Write it under `formal-verification/tests/<name>/sut/`.
- **(b), (c)** Adapter exercises the composed/chained system end-to-end. May require fixtures or test-harness setup the user's repo already provides.
- **(d)** Two adapters: one for each extract. The Lean runner from Step 1 stays in place but plays a tertiary cross-check role.
- **(e)** Adapter only exercises the post-region of each method; the harness asserts the pre-region holds before invoking.

Stable serialisation is critical. Both arms must agree on input encoding and output encoding before the harness runs. If the production language's natural serialisation differs from Lean's `Repr` output, write an adapter that reads/emits Lean-compatible JSON.

### Step 3: Generate the DRT Harness

Write `formal-verification/tests/<name>/drt_harness.<ext>` (Python is the recommended default; the user may pick a different driver language). The harness is a small program that:

1. **Reads the correspondence doc** at `formal-verification/correspondence/<Name>.md` to confirm the skip list. If the doc has been edited since DRT last ran, re-read.
2. **For each `def` to test** (i.e., not skipped):
   1. Generate `N` random inputs using a generator suitable for the input type. Default `N = 1000`; configurable via a `--count` flag. Use a seedable RNG; record the seed in the report.
   2. For each input:
      1. Pipe to the Lean runner; capture stdout, stderr, exit code.
      2. Pipe to the SUT adapter; capture stdout, stderr, exit code.
      3. Compare outputs. If they differ, record the divergence with: input (verbatim), Lean output, SUT output, exit codes, RNG seed and iteration index.
      4. If either side errors (non-zero exit) where the other does not, that is also a divergence.
3. **Minimisation.** For each divergence, apply one of:
   - Shrink integer inputs by halving toward zero.
   - Shrink list inputs by deleting one element at a time.
   - Shrink string inputs by truncating from the right.
   - Stop shrinking when the divergence no longer reproduces.
   Record both the original witness and the minimised witness.
4. **Emit a divergence report** to `formal-verification/tests/<name>/drt_report.md` with the structure in Step 4.
5. **For each `approximation` skip**, emit a one-line entry: "skipped: `<LeanIdent>` (approximation; see correspondence doc)".
6. **For each `exact`/`abstraction` def with zero divergences**, emit a one-line entry: "passed: `<LeanIdent>` (`<N>` inputs, seed `<S>`)".

Generators, count, and seed are user-configurable. Defaults sized to run in seconds, not minutes — DRT in this skill is for development and CI, not exhaustive sweeps.

### Step 4: Divergence Report Structure

```
# DRT report: <Name>

Run: <timestamp>
RNG seed: <S>
Inputs per def: <N>
D4 case: <a / b / c / d / e>
Correspondence doc revision: <git SHA or mtime>

## Verdict summary

| Status | Count |
|---|---|
| Passed (no divergence) | <m> |
| Failed (divergence found) | <k> |
| Skipped (approximation) | <j> |
| Excluded (fully Dafny-verified) | <i> |

## Failures

### <LeanIdent> — <k> divergences

- **D4 case applied.** <(a)–(e)>.
- **Witness (minimised).** Input `<minimised input>` produced Lean output `<X>` and SUT output `<Y>`.
- **Witness (original).** Input `<original input>` (RNG iteration `<i>`).
- **Divergence classification.** One of:
  - **general implementation bug** (~15/21 of Cedar's findings; D7a — parsing, dependency drift, naming, error handling)
  - **spec-modelling gap** (the informal spec did not specify behaviour on this input shape)
  - **production-code gap** (the source has a logic error)
  - **correspondence error** (the divergence is in an `exact` region, suggesting `/correspondence-review` mis-classified — feeds back to /correspondence-review)
- **Reproducer.** Shell command to re-run the harness pinned to the failing seed and iteration.

(repeat per divergence)

## Passes

- `<LeanIdent>`: <N> inputs, no divergence (seed <S>).

## Skips (approximation)

- `<LeanIdent>` — class `approximation` per correspondence doc; not DRT-tested. Reason: <one-line rationale from /correspondence-review's classification>.

## Exclusions (fully Dafny-verified slices)

- `<LeanIdent>` — Dafny-verified per `crosscheck/specs/<spec>.dfy:<line>`; DRT redundant.
```

The classification taxonomy is load-bearing: "general implementation bug" maps directly to D7a, the broad-applicability finding. Reporting the four-class taxonomy (rather than a single "divergence" verdict) lets adopters track the proportion of catches by class — analogous to Cedar's ~15/21 split.

### Step 5: Feedback to /correspondence-review

The fourth divergence classification — **correspondence error** — is the feedback signal back to `/correspondence-review`. If DRT finds divergences in a region classified `exact`, the classification was wrong: either the model and source are not exact, or the divergence is a real bug that the classifier should have predicted. Surface this explicitly in the report:

> `<n>` correspondence-error divergence(s) found. The Lean defs `<list>` were classified `exact` by `/correspondence-review` but DRT found divergences. Recommend re-running `/correspondence-review` with these defs flagged for adversarial re-examination.

This makes DRT a feedback loop, not a one-shot. Future runs of `/correspondence-review` may downgrade `exact` to `abstraction` or `mismatch` based on DRT evidence.

### Step 6: Hand-off

Single-paragraph handoff. The divergence report at `formal-verification/tests/<name>/drt_report.md` is the deliverable; do not re-present Step 4 content here.

Emit one of two lines:

- **Zero failures:** "DRT pass at `<report path>`: `<n>` defs tested, `<N>` inputs/def, seed `<S>`, zero divergences. `<j>` skip(s) (approximation). Absence of divergences is evidence, not proof."
- **One or more failures:** "DRT report at `<report path>`: `<k>` failure(s) across `<n>` defs. Witnesses minimised in the report. Re-run upstream skill per the per-failure classification — `/correspondence-review` for correspondence-errors, `/lean-impl` or `/spec-iterate`-style spec revision for spec/source/model gaps. This skill does not propose fixes."

The report is the artifact; the chat handoff is one line.

### Verification Checklist

```
## Verification Checklist

- [ ] Step 0 prerequisite check passed: Lean impl builds clean; correspondence doc classified; mismatch count is zero
- [ ] D4 case selected and recorded in the report
- [ ] Skip list constructed from `approximation` classifications
- [ ] Lean runner generated per def-under-test, builds clean, and runs on a worked-example input
- [ ] SUT adapter generated, runs on a worked-example input, produces stable output
- [ ] DRT harness runs `<N>` inputs per def with a recorded RNG seed
- [ ] Each divergence carries a minimised witness, original witness, RNG iteration index, and one of four classifications (general impl bug / spec gap / production gap / correspondence error)
- [ ] `approximation` skips emitted as one-line entries with rationale
- [ ] Fully Dafny-verified slices excluded with citation
- [ ] Correspondence-error divergences (if any) routed back to `/correspondence-review`
- [ ] Report at `formal-verification/tests/<name>/drt_report.md` has the Step 4 structure
- [ ] No fixes proposed — failures handed back to the user with the upstream skill named
```

## Arguments

The module name (e.g. `RateLimiter`). The skill resolves the Lean impl, the classified correspondence doc, and the production source path recorded in the correspondence doc.

Example: `/drt-oracle RateLimiter`

## References

- `docs/research/assurance-hierarchy.md` — Layer 1 "Two engines, two roles". Lean is the oracle, not the SUT.
- `crosscheck/skills/correspondence-review/SKILL.md` — upstream skill; produces the classified document this skill consumes.
- `crosscheck/skills/lean-impl/SKILL.md` — upstream skill; produces the model this skill drives.
- `crosscheck/docs/research/crosscheck-tla-vgd-addendum.md` — D4 input-space partition (cases a–e); D7a (DRT generalises ~15/21 general bugs); D7b (VGD-as-methodology Amazon scopes narrowly); the four VGD prerequisites.
- GitHub Next, *Lean Squad* (Task 8 Route B analogue): https://github.com/githubnext/agentics/blob/main/docs/lean-squad.md — pipeline source for "executable correspondence tests" with executed evidence rather than hand-wavy similarity claims.
- **Aeneas alternative (Lean Squad Route A) explicitly out of scope.** Aeneas + Charon mechanically derives Lean 4 from Rust source and is "optional but valuable" in Lean Squad's own framing. Rust-only; requires the Charon + Aeneas toolchain; not currently integrated in Crosscheck. Adopters with Rust codebases should know this is a known gap, not an oversight.
