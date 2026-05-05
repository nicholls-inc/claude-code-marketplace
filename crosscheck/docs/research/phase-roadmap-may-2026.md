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

## Outstanding items by tier

Mapped one-to-one against the May-2026 review's section 4 (Categories A/B/C). Tier numbering here matches the review's section-6 impact-to-effort ranking.

| Tier | ID | Item | Status | Notes |
|---|---|---|---|---|
| 1 | B1 | README chain-of-trust framing | **Done** (Phase 1) | |
| 1 | B2 | README reach-ceiling disclosure | **Partial** | The 22–27 % empirical reach band is now cited inline in B5's "Optional Dafny" step. A standalone reach-ceiling section/table is not yet shipped — keep open as a potential single-paragraph follow-up. |
| 1 | B3 | Persona / orthogonal-axis disclosure | **Done** (Phase 1) | |
| 2 | B4 | Skill-catalogue consolidation | **Deferred** (Phase 2) | See Phase-2 detail above. The review puts B4 at Tier 2 (high impact, medium effort) — listed under Phase 2 here because it has its own ADR scope, not because it's higher priority than Tier-2 peers. |
| 1 | B5 | Default-workflow re-centring on evaluations | **Done** (Phase 1) | |
| 1 | A4 | Calibration of the 30 % kill threshold | **Done** (Phase 1) | Configurable + documented as founder intuition. Empirical re-calibration on a labelled trace remains open as a Tier-4 item. |
| 1 | — | Cite Lahiri "Intent Formalization" essay + FMCAD 2024 in literature review | Open | One-line additions to `./literature-review.md`; small. |
| 2 | A1 | Cross-family back-translation in `/intent-check` | Open | Run the back-translator under a different model family than the spec author to partially decorrelate drift. |
| 2 | A2 | Optional MutDafny-derived mutation kicker on `/spec-adversary` | Open | Five-operator scoped mode (relational swap, off-by-one, conditional negation, return-value zeroing, default substitution) with a 60 s budget. |
| 2 | A3 | Post-mismatch sub-translation alignment in `/intent-check` | Open | One follow-up call after a `match=false` to localise the divergence as `(prose-span, formal-fragment)` pairs. |
| 2 | C2 | TiCoder-style interactive yes/no/undefined disambiguation | Open | New skill. The empirically validated unlock for intent formalization at scale; Crosscheck currently has no analogue. |
| 3 | C1 | `/spec-eval` — Lahiri-FMCAD soundness/completeness metrics | Open | Surface alongside the FP rate, not as a replacement. |
| 3 | C3 | Continuous verification hooks (Type-III sidecar mode) | Open | A lightweight CI mode or daemon that runs invariant coverage, `/intent-check` on protected surfaces, and acceptance oracles on a schedule rather than only on manual invocation. |
| 3 | C4 | SPOTs-inspired spec gap probe | Open | Generate small proof-oriented tests for Dafny modules / property tests; complements `/spec-adversary` with deterministic gap detection. |
| 4 | — | Empirical 6-layer hierarchy calibration data | Open | The single largest gap between Crosscheck-as-documented and Crosscheck-as-justified per the review's section 6. Instrument the plugin to publish "which layer caught what" / "which kill criteria fired and when" / "what fraction of code reached each layer". |

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
- Reassessment of the prior critical analysis: [`../reports/crosscheck-critical-analysis-reassessment.md`](../reports/crosscheck-critical-analysis-reassessment.md)
- Assurance hierarchy onboarding doc: [`./assurance-hierarchy.md`](./assurance-hierarchy.md)
- Logic distribution analysis (the 22–27 % reach band): [`./logic-distribution-analysis.md`](./logic-distribution-analysis.md)
- Phase 1 PR: [#143](https://github.com/nicholls-inc/claude-code-marketplace/pull/143)

## Open questions

- **Phase 0 push.** The two Phase-0 commits sit on local `main` only. They should be pushed to `origin/main` (either by a fast-forward push or by opening a small retroactive PR for review). Until that lands, the squad reference workflows under `crosscheck/docs/examples/workflows/` aren't visible to anyone reading `origin`.
- **Frequency of roadmap refresh.** This doc is delivery-tracking, so it ages quickly. Worth pairing with `/assurance-roadmap-check` semantics and refreshing on the same cadence (weekly / per-phase close).
- **Whether to dogfood `/assurance-init` for the meta-plugin itself.** Considered during Phase 1 sign-off; deferred. The meta-plugin currently has no `docs/assurance/` directory — adopting its own scaffolding would be the strongest "we use what we ship" signal but creates ~10 new files of governance overhead. Revisit when the open Tier-2/3 items make a formal phase plan necessary.
