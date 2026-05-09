# Crosscheck Phase Roadmap — May-2026 Review Remediation

## Context

The May-2026 review (`./crosscheck-review-may-2026.md`) and its in-repo reassessment (`../reports/crosscheck-critical-analysis-reassessment.md`) collectively identified ~15 distinct improvement items spanning README register, skill-level configuration, methodology gaps, and longer-horizon empirical calibration work. This doc tracks the phased delivery of those items: what shipped, what's open, and how each phase verified its own outputs.

The roadmap is delivery-tracking, not strategy. The strategy lives in the review and reassessment docs above; revisit those before adding a new phase.

## Phase index

| Phase | Scope | Status | Outcome |
|---|---|---|---|
| 0 | Import 12 squad reference workflows + harmonise FP-tracker schema between squad scripts and `/intent-check` | **Done locally; awaiting push** | Two commits sitting on local `main` only: `docs(crosscheck): add reference workflows for assurance hierarchy` and `docs(crosscheck): harmonise squad FP-tracker schema with intent-check`. Files live at `crosscheck/docs/examples/workflows/`. Will land in a follow-up push to `origin/main`. |
| 1 | B1 chain-of-trust, B3 persona/orthogonal-axis disclosure, B5 evaluations-first default workflow, A4 threshold framing | **Done** | PR [#143](https://github.com/nicholls-inc/claude-code-marketplace/pull/143), squash-merged 2026-05-05. README rewrite + Configuration env vars + founder-intuition disclaimer across `/assurance-status`, `/intent-check`, and the FP-tracker reference. |
| 2 | B4 catalogue consolidation decision (collapse 20 skills into ~5 Phoenix-style primitives, or keep flat) | **Deferred** | Explicitly deferred during Phase 1 sign-off. Phase-1 audit found 12–14 of 20 skills could plausibly merge; the `assurance-*` and `verify-*` clusters are clean subcommand candidates, but the semi-formal reasoning cluster's intent-driven triggers don't route via subcommand without losing signal. Future work needs its own ADR. |
| 3a-i | Foundation corrections — TLA+/VGD addendum + May review §5.6 recast | **Done** | Local commit `45c2d07`. Layer 1b proposal expunged → Layer 4 placement; Cedar broad-applicability split into D7a (DRT generalises) / D7b (VGD-as-methodology does not); DRT scope expanded to 5 cases. Verified by `crosscheck:hellebuyck` self-review on both files. |
| 3a-ii | Lamport blueprint analogy in `/acceptance-oracle-draft` | **Done** | Local commit `8172fd9`. 3-line analogy, not a rename — the Layer-5 user-perspective semantics differ from Lamport's spec-as-thinking-tool framing per A4. |
| 3a-iii | Layered-assurance framing + downstream doc updates | **Done** | Local commit `42877f1`. New "What Crosscheck is: layered assurance" section in README + parallel section in `assurance-hierarchy.md`; "What Crosscheck is not good for" four-point scope-limit; module-level VGD-prerequisite description with D6 hedged as hypothesis; Brooker as diagnostic only; Lean position note; literature additions. ~86 cumulative lines (well under K1 cap of 150). Verified by per-item hellebuyck self-reviews + final integration sweep + byfuglien `/reason` on the D5 reframe. |
| 3b-α | Lean MCP engine + `/informal-spec` + `/lean-spec` (Lean pipeline steps 1–2 of 5) | **Done** | Local commits `ef6c4f6`, `486512b`, `5008034`. Two-engine Docker harness (Mathlib pre-warmed via `lake exe cache get`); shared `runDocker` core with `runDafny` and `runLean` wrappers; three new MCP tools (`lean_check`, `lean_run`, `lean_test`); regex-keyed sign-off contract between the two skills (`Human sign-off: <YYYY-MM-DD>`). Verified by hellebuyck self-review (APPROVE after 2 REDs + 5 YELLOWs applied) and byfuglien `/reason` on (A) two-skills-not-one architecture (HOLDS, HIGH) and (B) Lean engine architecture (HOLDS WITH CAVEATS — caveats addressed inline). 132 tests passing (15 new). |
| 3b-β | `/lean-impl` + `/correspondence-review` + `/drt-oracle` + K3 smoke test (Lean pipeline steps 3–5 of 5) | **Done** | Three new SKILL.md files (lean-impl 221 lines, correspondence-review 215, drt-oracle 271) plus byfuglien orchestration table + Phase 3 routing entry; assurance-hierarchy.md Layer 1 flipped from "partial as of 3b-α" to full pipeline shipped; CLAUDE.md and README.md persona table updated. MCP-server comments at `index.ts:90` and `leanTest.ts:23` reconciled with the chosen scope (3b-β kept `lean_test` as the `lake build` `#guard` alias and routed `/drt-oracle` through `lean_run` + an external Python harness against per-def `<Name>Runner.lean` files, rather than wiring the `lake test` driver originally hypothesised in 3b-α). K3 smoke test under `formal-verification/tests/power/` caught the planted off-by-one in the production Python with a minimised witness `(base=2, exp=0)` (oracle=1, SUT=2) and 128/200 divergences. The Lean Docker image was *not* built end-to-end in this session (sandbox restriction on `~/.docker/buildx/activity/`); the smoke ran with a Python-faithful `oracle_reference.py` standing in for the Lean runner — pipeline architecture and harness logic exercised, Lean MCP execution path deferred to a session with the carve-out. Verified by `crosscheck:byfuglien` `/reason` certificates on (A) five-skills-not-one architecture (HOLDS, HIGH), (B) Lean engine architecture vs the abandoned `lake test` plan (HOLDS WITH CAVEATS, MEDIUM — forward limitation: PBT-in-Lean would need `lean_test` to evolve), (C) DRT input-space partition cases a–e (HOLDS, HIGH), (D) correspondence-doc-as-DRT-prerequisite (HOLDS, HIGH); plus `crosscheck:hellebuyck` self-reviews on each new SKILL.md (lean-impl + drt-oracle APPROVE WITH NITS; correspondence-review REVISE → both REDs fixed: pipeline-diagram label + missing "What this skill does NOT do" block). |
| 3c | Layer 4 ADR — TLA+/P/Alloy at Layer 4 | **Done** | Local commit `4ca00dd`. ADR-0001 at `crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md`. Decision: redefine Layer 4 (broaden flat) rather than introduce 4a/4b sublayers. Four alternatives explicitly rejected. Verified by byfuglien `/reason` on the layer redefinition with `file:line` citations to the existing Layer 4 description. |
| 3d | Per-module VGD-prerequisite assessment in `/assurance-layer-audit` + `/assurance-init` | **Done** | Local commit `8dbf733`. Step 4.5 inserted in `/assurance-layer-audit` (no renumbering); Step 6.5 inserted in `/assurance-init` with Step 1.3 overwrite-gate handling, Class B exemption explicit, and a cached-output read pattern that avoids duplicating Step 4.5's work. Verified by byfuglien `/reason` on routing logic + the cached-output pattern; hellebuyck self-review on both SKILL.md edits. |

## Per-phase detail

### Phase 0 — Squad workflow imports + FP-tracker schema harmonisation

**What shipped.** Twelve sanitised reference workflow files imported under `crosscheck/docs/examples/workflows/`, covering tier-A static enforcement (assurance.yml CI workflow, pre-commit hook, invariant-coverage Python script, acceptance scenarios template + runner) and tier-B agentic workflows (PR-Gate, Recheck, Squad). The squad's kill-criterion check, the PR-Gate sticky comment, the Recheck verdict, and the squad's status dashboard now all read the same FP tracker that `/intent-check` writes (`.assurance/intent-check-fp-tracker.csv`, `spurious` marker, 14-day window, n ≥ 3 floor, empty rows excluded).

**Initial drift.** The first import had schema drift between the squad scripts and the skill — different file path, different verdict marker, different window. The second commit harmonised six files so the contract is single-source.

**Pre-Phase-1 closure work.** Before Phase 1 began, three founder-intuition questions were resolved:

- Q2: workflows live downstream of the plugin (the marketplace install path doesn't pull them in); narrative weight in the README's intro was the actual gap, not routing in `byfuglien.md`.
- Q3: the 30 % / 14-day / n ≥ 3 numbers are founder intuition, not labelled-pilot data. Now configurable in the squad runner script and surfaced in `crosscheck/docs/examples/workflows/README.md`.
- Reference workflows are checked in so subsequent plan items have something concrete to point at.

**Status.** Both commits are on local `main` but not on `origin/main`. They will be pushed in a follow-up — see the open question "Phase 0 push" at the bottom of this doc.

### Phase 1 — README narrative reshape + A4 threshold framing

**What shipped.**

- **B1** (chain-of-trust): replaced "provably correct Python/Go" with "code is verified against its spec, then compiled to Python or Go via the Dafny backends." Added an explicit Layer 2 / trusted-computing-base hedge.
- **B3** (persona asymmetry + orthogonal-axis disclosure): fixed the wrong "Layers 1–3" claim for Byfuglien (actual ownership: Layer 1 + the regression-detection slice of Layer 4 + four orthogonal semi-formal reasoning skills). Inserted a 3-column persona/orthogonal-axis table.
- **B5** (evaluations-first default workflow): replaced the "What you can run right now" layer-list with a numbered Recommended order putting `/acceptance-oracle-draft` early and Dafny Layer 1 as "optional, when the code shape supports it" with the empirical 22–27 % reach band cited.
- **A4** (threshold framing): added Configuration sections + `CROSSCHECK_FP_TRIPPED_THRESHOLD` / `CROSSCHECK_FP_AT_RISK_THRESHOLD` / `CROSSCHECK_FP_WINDOW_DAYS` env vars to `/assurance-status` and `/intent-check`. Propagated the "founder intuition pending operational data" disclaimer from the squad reference workflows (Phase 0) to the plugin's own SKILL.md files. Added a "Calibration of Layer-5 thresholds" section to `./assurance-hierarchy.md`.

**Verification gates run during the work.** Each item passed at least one orchestrator-agent self-review before its edits were finalised. See "Methodology" below for the pattern.

**Outcome.** PR [#143](https://github.com/nicholls-inc/claude-code-marketplace/pull/143), squash-merged 2026-05-05.

### Phase 2 — B4 catalogue consolidation decision (deferred)

**Question.** Should the 20-skill flat catalogue collapse into ~5 Phoenix-style primitives (`verify-core-logic`, `maintain-invariants`, `maintain-evaluations`, `govern-protected-surfaces`, `probe-specs`) with subcommands, or stay flat?

**Phase-1 audit (kept here so the reviver doesn't redo it).** Of the 20 skills:

- **`assurance-*` quadruplet** (`/assurance-init`, `/assurance-layer-audit`, `/assurance-status`, `/assurance-roadmap-check`) — clean subcommand candidates. Triggers are atomic and pipeline-ordered (audit → init → status → check). A `/govern audit|init|status|check-roadmap` wrapper would route trivially.
- **`verify-*` quintuplet** (`/spec-iterate`, `/generate-verified`, `/extract-code`, `/lightweight-verify`, `/suggest-specs`) — also clean. Pipeline-ordered with distinct triggers.
- **Semi-formal reasoning cluster** (`/reason`, `/compare-patches`, `/locate-fault`, `/trace-execution`, `/rationale`) — **bad** subcommand candidates. Triggers are user-intent-driven ("is this correct?", "what does this do?") rather than action-scoped. Subcommand routing would require a disambiguation step that loses the direct trigger-to-action mapping.

**When this is revived.** Scope a separate ADR + follow-up plan. Realistic outcomes:

- Decide-now: defer permanently (keep flat, document in the README)
- Decide-now: scope partial consolidation (yes for `/govern` and `/verify`; no for semi-formal reasoning; ship ADR only, separate phase for implementation)
- Decide-now: full consolidation (highest risk because the routing layer for the intent-driven cluster is the unsolved part)

### Phase 3 — TLA+/VGD addendum operationalisation (all 6 sub-phases shipped)

Phase 3 took the May-2026 review's TLA+/VGD addendum (`./crosscheck-tla-vgd-addendum.md`) plus the conversation that followed it and turned the resulting decisions into shipped artefacts. The plan for the phase is at `~/.claude/plans/make-all-these-changes-ancient-shell.md`; it splits cleanly into six sub-phases with explicit dependency edges, kill criteria (K1: ≤150 cumulative lines on README + assurance-hierarchy; K2: >2 RED in 3a-iii after one revision; K3: 3b smoke test must catch a planted divergence), and per-sub-phase verification gates.

**What shipped (all on local `main`, not yet on `origin/main`):**

- **3a-i — Foundation corrections.** Two source documents corrected: the addendum (Layer 1b → Layer 4; Cedar narrowed to safety-critical/TCB; DRT scope expanded; D7a/D7b split made explicit) and the May review §5.6 (functional-vs-behavioral specification framing replacing the prior I/O / concurrency / integration framing). Single PR-equivalent commit.
- **3a-ii — Lamport blueprint analogy.** Three-line analogy added to `/acceptance-oracle-draft`'s SKILL.md, deliberately not introduced as an alias. The hellebuyck A4 review caught and prevented a rename that would have collapsed the Layer-5 user-perspective semantics into Lamport's spec-as-thinking-tool framing.
- **3a-iii — Layered-assurance framing + downstream prose.** New top-level section in README and parallel section in `assurance-hierarchy.md` positioning Crosscheck as layered formal verification + probabilistic complements (PBT, DRT) + stochastic complements (`/intent-check`, structured reasoning), with VGD as one methodology applied per-module. New "What Crosscheck is not good for" four-point structure (modelled on Newcombe et al.) tied to the four VGD prerequisites. D6 (AI weakening prerequisite #4) explicitly hedged as a *hypothesis*, not a claim — Cedar 2024 used human Lean + human Rust, no AI-augmented baseline exists. Brooker hubris/humility/laziness as *diagnostic* language, never as adoption gate. Per-module prerequisite assessment described as the routing primitive that 3d operationalises. ~86 cumulative lines (K1 cap was 150).
- **3b-α — Lean MCP engine + spec arm.** Mirrored the Dafny pattern: `runDocker` core shared by `runDafny` and `runLean`; three new MCP tools (`lean_check` / `lean_run` / `lean_test`); Mathlib-pre-warmed Docker image (`lake exe cache get` then `lake build` baked into the image). `/informal-spec` extracts prose specs with a hard human sign-off gate; `/lean-spec` translates signed-off prose into Lean 4 stubs with `sorry` proof bodies, gated on `lake build` clean. Sign-off marker is a regex (`^Human sign-off:\s*\d{4}-\d{2}-\d{2}\s*$`) the consuming skill keys off — that file-artefact contract is what makes the two-skill split load-bearing rather than ceremonial.
- **3c — Layer 4 ADR.** Decided to redefine Layer 4 (broaden the existing definition flat) rather than introduce 4a/4b sublayers. ADR explicitly rejects four alternatives. Routing heuristic for behavioural specs uses standard TLA+ scope criteria (state machines, workflows with branches, rule-interaction surfaces, invariant-rich data) — not "Cedar implies broad."
- **3d — Per-module VGD-prerequisite skills.** Step 4.5 in `/assurance-layer-audit` (per-module pass/partial/fail with one-line evidence per prerequisite, prerequisite #4 flagged as untested under D6); Step 6.5 in `/assurance-init` that reads Step 4.5's output if present (cached) rather than recomputing, honours Step 1.3's overwrite decision, and exempts onboarding writes from `/protected-surface-amend` (the doc is being created, not amended).

**3b-β shipped (closes the five-skill pipeline).** The remaining three Lean-pipeline skills landed in this session:

- **`/lean-impl`** — Lean Squad Task 4 analogue. Translates source implementation into a Lean 4 functional model with explicit modelling-strategy choice (pure transliteration / pure model of imperative / pure model of effectful), `-- src:` comments tying every non-trivial Lean def to a source file:line range, partial-source wrapping in `Option`/`Except` (no fake totality via `panic!`), and a hard `lake build`-clean gate. Seeds the correspondence stub `/correspondence-review` consumes — the stub deliberately does not pre-classify entries.
- **`/correspondence-review`** — Lean Squad Task 6 analogue. Classifies each Lean definition against source as `exact / abstraction / approximation / mismatch` with file:line citations and rubric-step traceability. Step 5 hard-stop on any `mismatch`; if zero mismatches, hands off to `/drt-oracle` with the skip list (every `approximation` ident) explicit.
- **`/drt-oracle`** — Lean Squad Task 8 Route B analogue. Differential random testing scoped by the correspondence doc: runs on `exact` and `abstraction` regions, skips `approximation` regions with a flag, refuses to run while any region is `mismatch`. Five-case D4 input-space partition explicit in Step 0.5. Four-class divergence taxonomy in the report (general impl bug / spec gap / production gap / correspondence error), with the fourth class routed back to `/correspondence-review` as a feedback signal. Aeneas / Charon (Route A) explicitly out of scope as a known gap not an oversight. The MCP scope decision: rather than wire the `lake test` driver hypothesised in 3b-α's `index.ts:90` comment, `/drt-oracle` invokes `lean_run` against per-def `<Name>Runner.lean` files driven by an external Python harness — `lean_test` stays as the compile-time `#guard` path. Server-side comments updated to reflect the chosen scope.

**K3 smoke (kill criterion).** Self-contained fixture under `formal-verification/tests/power/` exercises the full five-step pipeline against `power(base, exp): nat × nat → nat` with a planted off-by-one in the production Python (`for _ in range(exp + 1)` instead of `range(exp)`). The DRT harness caught 128 / 200 divergences with minimised witness `(base=2, exp=0)` returning `1` (oracle) vs `2` (SUT) — exactly the "general implementation bug" class D7a's bug taxonomy predicts DRT catches. Smoke-test scope honest about its reach: the Lean Docker image was not built in this session (sandbox blocks `~/.docker/buildx/activity/`); the harness used `oracle_reference.py` — a Python implementation faithful to `CrosscheckModel.Power.power`'s recursion — as the smoke-time oracle. Pipeline architecture, harness logic, divergence minimisation, and witness reporting all exercised end-to-end; Lean MCP execution path deferred to a session with the sandbox carve-out.

**Verification gates run.** `crosscheck:byfuglien` `/reason` certificates on the four architectural claims (premise-tagged with `file:line` citations and alternative-hypothesis ruling): pipeline architecture (HOLDS, HIGH), Lean engine architecture (HOLDS WITH CAVEATS, MEDIUM — forward limitation: PBT-in-Lean would need `lean_test` to evolve), DRT input-space partition (HOLDS, HIGH), correspondence-as-DRT-prerequisite (HOLDS, HIGH). `crosscheck:hellebuyck` self-reviews on each new SKILL.md (`/lean-impl` and `/drt-oracle` APPROVE WITH NITS; `/correspondence-review` REVISE on first pass → both REDs fixed: pipeline-diagram label drift and missing "What this skill does NOT do" block).

**Verification gates run.** Methodology section below describes the patterns; Phase 3 used both gates extensively. Highlights:

- `crosscheck:byfuglien` `/reason` certificates on five architectural claims with `file:line` citations — D5 reframe (3a-iii.1), Lean engine architecture (3b.1), two-skills-not-one pipeline architecture (3b.2/3b.3 split), Layer 4 redefinition (3c.1), per-module routing logic (3d).
- `crosscheck:hellebuyck` self-reviews on every README, SKILL.md, ADR, and `assurance-hierarchy.md` edit, plus a final integration sweep on the 3a-iii bundle. The 3b-α prose review surfaced 2 RED + 5 YELLOW findings (rename-vs-sign-off coupling, protected-surface partition note, D6 hedging in a table cell, etc.); all addressed before commit.

**Local-only state.** All Phase-3 work plus the 3b-β edits remain local to the working tree (Phase-3 commits 5 from prior sub-phases + 3 from 3b-α, plus the 3b-β changes still uncommitted as of this update). Push to a feature branch + PR(s) is a follow-up choice; the prior Phase-0 commits are in the same condition (see Open questions).

## Outstanding items by tier

Mapped one-to-one against the May-2026 review's section 4 (Categories A/B/C). Tier numbering here matches the review's section-6 impact-to-effort ranking.

| Tier | ID | Item | Status | Notes |
|---|---|---|---|---|
| 1 | B1 | README chain-of-trust framing | **Done** (Phase 1) | |
| 1 | B2 | README reach-ceiling disclosure | **Done** (Phase 3a-iii.2) | Phase 3a-iii.2 added a "What Crosscheck is not good for" four-point section (modelled on Newcombe et al.) to both README and `assurance-hierarchy.md`, tied to the four VGD prerequisites. The 22–27 % reach band remains cited in B5; the structural reach-ceiling disclosure now lives in the scope-limit section. |
| 1 | B3 | Persona / orthogonal-axis disclosure | **Done** (Phase 1) | |
| 2 | B4 | Skill-catalogue consolidation | **Deferred** (Phase 2) | See Phase-2 detail above. The review puts B4 at Tier 2 (high impact, medium effort) — listed under Phase 2 here because it has its own ADR scope, not because it's higher priority than Tier-2 peers. |
| 1 | B5 | Default-workflow re-centring on evaluations | **Done** (Phase 1) | |
| 1 | A4 | Calibration of the 30 % kill threshold | **Done** (Phase 1) | Configurable + documented as founder intuition. Empirical re-calibration on a labelled trace remains open as a Tier-4 item. |
| 1 | — | Cite Lahiri "Intent Formalization" essay + FMCAD 2024 in literature review | **Partial** (Phase 3a-iii.5) | Phase 3a-iii.5 added Disselkoen 2024, Newcombe 2014/2015, Brooker 2022, and Lamport 2015 entries with the D7a/D7b split made explicit. Lahiri "Intent Formalization" + FMCAD 2024 are still pending. |
| 1 | — | Lean pipeline completion (3b-β) — `/lean-impl` + `/correspondence-review` + `/drt-oracle` + K3 smoke test | **Done** (Phase 3b-β) | Five-step Lean pipeline shipped. K3 caught the planted off-by-one with a minimised witness; Lean MCP execution itself is deferred until the Lean Docker image is built in a session that can write to `~/.docker/`. |
| 2 | A1 | Cross-family back-translation in `/intent-check` | Open | Run the back-translator under a different model family than the spec author to partially decorrelate drift. Composes naturally with `/informal-spec` (Phase 3b.2) — `/intent-check` now has a structured prose source to back-translate against. |
| 2 | A2 | Optional MutDafny-derived mutation kicker on `/spec-adversary` | Open | Five-operator scoped mode (relational swap, off-by-one, conditional negation, return-value zeroing, default substitution) with a 60 s budget. |
| 2 | A3 | Post-mismatch sub-translation alignment in `/intent-check` | Open | One follow-up call after a `match=false` to localise the divergence as `(prose-span, formal-fragment)` pairs. |
| 2 | C2 | TiCoder-style interactive yes/no/undefined disambiguation | Open | New skill. The empirically validated unlock for intent formalization at scale; Crosscheck currently has no analogue. Phase 3b.2's `/informal-spec` ambiguity section is a precursor — it surfaces the questions but doesn't drive the resolution loop. |
| 3 | C1 | `/spec-eval` — Lahiri-FMCAD soundness/completeness metrics | Open | Surface alongside the FP rate, not as a replacement. |
| 3 | C3 | Continuous verification hooks (Type-III sidecar mode) | Open (out of scope for Phase 3) | A lightweight CI mode or daemon that runs invariant coverage, `/intent-check` on protected surfaces, and acceptance oracles on a schedule rather than only on manual invocation. Phase 3 explicitly deferred this. |
| 3 | C4 | SPOTs-inspired spec gap probe | Open (out of scope for Phase 3) | Generate small proof-oriented tests for Dafny modules / property tests; complements `/spec-adversary` with deterministic gap detection. Phase 3 explicitly deferred this. |
| 4 | — | Empirical 6-layer hierarchy calibration data | Open | The single largest gap between Crosscheck-as-documented and Crosscheck-as-justified per the review's section 6. Instrument the plugin to publish "which layer caught what" / "which kill criteria fired and when" / "what fraction of code reached each layer". Phase 3 sharpened the question by introducing per-module VGD-prerequisite assessment (Phase 3d): the calibration target is now "did Step 4.5's prerequisite verdict actually predict Layer-1 reach for that module?" |

## Methodology — verification gates

The Phase-1 pattern that should be reproduced for every future phase: each item runs through one or two crosscheck-skill or orchestrator-agent self-reviews **after the edit lands** but **before the PR is opened**. A gate that flags a finding pushes the item back to redrafting.

The gates that proved load-bearing in Phase 1:

| Gate | Use it for | Why |
|---|---|---|
| `crosscheck:hellebuyck` self-review | Edits to README narrative, SKILL.md governance text, agents/hellebuyck.md guidelines, `docs/research/assurance-hierarchy.md`, governance scaffolding | Hellebuyck owns Layer 4–6 + governance. It is the right reviewer for whether new prose drifts from the docs' register or contradicts each SKILL's own "when to use" semantics. Phase 1 caught 2 REDs + 9 AMBERs across A4, B1, B5, and the final integration sweep — every one of which would otherwise have shipped. |
| `crosscheck:byfuglien` `/reason` certificate | Architectural claims about implementation-chain skill ownership, layer mapping, semi-formal reasoning placement | Byfuglien owns Layer 1 and the four orthogonal semi-formal skills. `/reason` produces a structured premises-with-`file:line`-citations + alternative-hypothesis-check certificate. Phase 1 used this to verify every cell of the new persona table against `byfuglien.md`, `assurance-hierarchy.md`, and `skills.md` (PASS, HIGH confidence; no counterexamples found across six dimensions of the alternative-hypothesis search). |
| Final integration sweep via `crosscheck:hellebuyck` | After all phase items are individually green, before commit | Catches cross-edit drift that per-item reviews miss. Phase 1's sweep surfaced the 96 % accuracy hedge (implicit overclaim in the Recommended order block) and two stray hardcoded "30 %" mentions in the catalogue / agent-guidelines that the per-item A4 review hadn't reached. |

Two gates that were **considered and dropped** in Phase 1, with reasoning future phases can reuse:

- `/intent-check` round-trip on README ↔ docs prose. The skill is scaffolded for `(invariant prose, covering test, code diff)`, not for `(README, docs, no diff)`. Generalising it is itself a work item (a candidate Tier-2 follow-up). The Hellebuyck self-review covers the same drift surface for documentation reshape.
- `/check-regressions` on SKILL.md edits. The skill targets Dafny specs whose source has moved, not SKILL.md prose; misuse here would muddle its semantics.

## How to use this doc

- **Picking up a new phase.** Read the relevant tier item below, then the source review section it points at. Reproduce the verification-gate pattern: pick the gate(s) appropriate to the artifact (governance prose → hellebuyck; impl-chain claim → byfuglien `/reason`; cross-cutting reshape → integration sweep).
- **Closing a phase.** Update the phase index + the tier table. Add a short "what shipped + verification gates run" subsection. Cross-link the PR.
- **Adding a new item.** Cite the source paragraph in the May-2026 review or its reassessment. Don't add free-floating items; the constraint is "every item in this doc maps back to a source paragraph."

## References

- May-2026 review: [`./crosscheck-review-may-2026.md`](./crosscheck-review-may-2026.md)
- TLA+/VGD addendum (Phase 3 source): [`./crosscheck-tla-vgd-addendum.md`](./crosscheck-tla-vgd-addendum.md)
- Reassessment of the prior critical analysis: [`../reports/crosscheck-critical-analysis-reassessment.md`](../reports/crosscheck-critical-analysis-reassessment.md)
- Assurance hierarchy onboarding doc: [`./assurance-hierarchy.md`](./assurance-hierarchy.md)
- Logic distribution analysis (the 22–27 % reach band): [`./logic-distribution-analysis.md`](./logic-distribution-analysis.md)
- ADR-0001 (Phase 3c — Layer 4 redefinition): [`./adr/0001-behavioral-specs-at-layer-4.md`](./adr/0001-behavioral-specs-at-layer-4.md)
- Phase 1 PR: [#143](https://github.com/nicholls-inc/claude-code-marketplace/pull/143)

## Open questions

- **Phase 0 + Phase 3 push.** Phase 0's two commits and Phase 3's eight committed sub-phases all sit on local `main` only, plus the 3b-β edits remain uncommitted in the working tree. The aggregate should be pushed (one PR per phase, or a single rollup PR for review) so the work is visible to anyone reading `origin`. Until then, the squad reference workflows, the layered-assurance framing, the ADR, and the new Lean MCP engine + five-skill pipeline are invisible upstream.
- **Lean image build verification.** `crosscheck/scripts/build-lean-docker.sh` includes a baseline-timing block (byfuglien C-B1) that runs after the image build to confirm `lake build` on a 2-line Mathlib-importing file completes in <10 seconds. The image was attempted in the 3b-β session but failed at `docker build` due to a sandbox restriction on `~/.docker/buildx/activity/` (the harness lacks write access there). First build must happen on a developer machine with `~/.docker/` writable; subsequent rebuilds are cheap if the lean-toolchain + Mathlib pin are stable. The K3 smoke test ran with a Python-faithful reference oracle in the meantime — adequate to exercise the pipeline architecture and harness logic but not the Lean MCP execution path.
- **Phase 3b-β Lean MCP execution.** Tied to the previous bullet. Once the Lean image builds, the `/drt-oracle` smoke under `formal-verification/tests/power/` should re-run with `--oracle lean-runner` (after authoring `formal-verification/lean/CrosscheckModel/PowerRunner.lean` exposing a `main : IO Unit` CLI). That re-run exercises `lean_run` against the runner file, closing the deferred reach of the K3 smoke.
- **Frequency of roadmap refresh.** This doc is delivery-tracking, so it ages quickly. Worth pairing with `/assurance-roadmap-check` semantics and refreshing on the same cadence (weekly / per-phase close).
- **Whether to dogfood `/assurance-init` for the meta-plugin itself.** Considered during Phase 1 sign-off; deferred. The meta-plugin currently has no `docs/assurance/` directory — adopting its own scaffolding would be the strongest "we use what we ship" signal but creates ~10 new files of governance overhead. Revisit when the open Tier-2/3 items make a formal phase plan necessary.
