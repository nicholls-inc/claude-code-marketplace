# Intent — Build ADD into Crosscheck

**Status:** Attested v1.1
**Phase:** 0 (Intent capture)
**Project:** Add Assurance Driven Development support to the Crosscheck plugin
**Last attested:** 2026-05-09 by nicholls-inc (Phase 2 re-attestation; re-drafting cascade from seam validation B-3, B-4 — IC4 re-framed as `/intent-check` parameterisation, IC10 references existing dual-track principle. Authorisation for the agent-authored re-attestation commit recorded in commit body.)
**Prior attestation:** v1.0 — 2026-05-09 by nicholls-inc (Phase 2 closure on branch claude/validate-add-phase-2-bFneS, comparison report § 9.4)

## Vision

Crosscheck today supports **bootstrap mode** — applying assurance machinery over an existing codebase. We are extending it to support **ADD mode** — applying assurance from a clean slate, where humans define vision and govern specs while AI agents perform construction. Both modes coexist; module boundaries carry an origin-mode tag; the existing bootstrap-mode flow is unchanged for users who do not opt in to ADD.

The motivating use case: a developer (human) starts a new repository with only a vision in mind. They use Crosscheck-with-ADD to capture intent, derive specs, install governance, and *then* let agents implement against the spec stack. Today this user experiences friction because Crosscheck's recommended order presupposes existing modules, manifests, and tests.

## Intent claims

Each claim is numbered and stable. Supersession of a claim creates a new IC; the old IC is retained and marked Superseded-by.

---

**IC1 — Empty-repo entrypoint.**
A user with only a written vision (no code, no manifests, no tests) can run a Crosscheck command and begin Phase 0 of ADD without the command failing or producing empty diagnostics. The entrypoint either (a) recognises the empty-repo state and routes to ADD-mode skills, or (b) is a new ADD-mode skill the user invokes directly.

*Observable signal:* the existing `/assurance-layer-audit` and `/assurance-init` skills no longer produce nonsensical output (empty per-layer projection, "name 1-3 load-bearing modules" with no modules) when run in an empty repo. Either they detect the empty state and recommend the ADD path, or a new skill is the documented first step for empty repos.

---

**IC2 — Phase 0 intent capture is an explicit, repo-resident artifact.**
ADD-mode users produce a `docs/add/intent.md` (or equivalent path determined by the agent) with numbered intent claims, explicit out-of-scope items, and a threat model. The artifact is created interactively with skill assistance, not from a blank page.

*Observable signal:* a new (or extended) skill walks the user through Phase 0, eliciting intent claims with the required ADR-style structure (Context / Decision / Alternatives / Consequences) and writing them to disk with stable IC IDs and Status fields.

---

**IC3 — Phase 1 spec stack derives from intent, not from code.**
ADD-mode users produce architectural, behavioral, and functional specs derived top-down from `IC` IDs. Every spec section declares `consumes:` and `produces:` linkage against the IDs it depends on and generates.

*Observable signal:* a skill (or skills) produces architectural and behavioral specs from an attested intent doc, with the linkage graph integrity check passing (no orphans). For functional specs in the Layer 1 reach band, the existing `/spec-iterate` flow is reachable from the spec stack — but it is invoked *because* the architectural spec said so, not as a standalone entrypoint.

---

**IC4 — Phase 2 spec validation reuses `/intent-check`'s pipeline with prose-vs-prose inputs.**
Crosscheck's existing `/intent-check` skill operates on `(invariant prose, covering test, code diff)`. Its load-bearing disciplines — two-prompt structural separation (back-translator blind to original intent), kill-criterion pre-check (FP rate over rolling window), mandatory carve-out scan, fail-closed semantic validation, content-hashed attestation, FP-tracker CSV with stable schema — all transfer to the prose-vs-prose case. ADD's Phase 2 step is therefore the existing pipeline parameterised on a different input shape: instead of `(invariant prose, covering test, code diff)`, the inputs are `(intent doc, spec stack)`. No code or test is required.

*Observable signal:* a new skill (or new mode of `/intent-check`) reuses the existing pipeline structure end-to-end — same env vars (`CROSSCHECK_FP_TRIPPED_THRESHOLD` / `_AT_RISK_THRESHOLD` / `_WINDOW_DAYS`), same tracker-CSV schema, same SHA-256-hashed JSON attestation, same two-section back-translator output, same carve-out scan — substituting only the input artifacts. The architectural spec (S2.3) declares the inheritance explicitly.

---

**IC5 — Three operating modes; per-module tags carry the originative two; transitional is a repo-level descriptor.**
Modules in a Crosscheck-governed repo carry a tag indicating their origin mode, drawn from `{bootstrap, add}`. *Transitional* is not a per-module tag — it is the *repo-level* descriptor that applies when modules disagree (some originated bootstrap-mode, others ADD-mode). Per ADR-001, there is no `mode: transitional` value on any individual module. Governance applied to each module is mode-appropriate. A bootstrap-mode module is not expected to have an intent-attestation trail back to Phase 0; an ADD-mode module is not expected to have its specs treated as recoverable from code.

*Observable signal:* each module's invariant doc (`docs/invariants/<module>.md`) or equivalent metadata records its origin mode. Skills consulting governance (e.g., the consolidation pass) honour the mode tag.

---

**IC6 — The auditor agent runs consolidation passes.**
A third agent role, distinct from Byfuglien and Hellebuyck, runs scheduled consolidation passes that produce per-artifact verdicts (Settled / Active / Drifted) without modifying artifacts. The auditor consumes deterministic signals from the linkage graph and renders natural-language judgments on suspect artifacts.

*Observable signal:* a new agent definition file (`agents/auditor.md` or equivalent) plus a skill or workflow that runs the consolidation pass on demand or on schedule. The user-facing workflow document `docs/examples/workflows/` (e.g. an updated `assurance-squad.md`) describes when and how the auditor runs.

---

**IC7 — Diff classification is enforced on spec-changing commits.**
Every commit that modifies an artifact under `docs/add/` (and, when extended, `docs/invariants/` for ADD-mode modules) carries one of four classifications: Propagated discovery / Intent refinement / Drift / Retraction. The classification is recorded mechanically (commit-message convention or a sidecar log) and the auditor agent verifies the classification during consolidation passes.

*Observable signal:* a pre-commit hook or CI gate detects spec-changing commits and requires a classification. A small log (e.g., `.assurance/diff-classification-log.csv`) accumulates the history; consolidation passes consume it.

---

**IC8 — Deterministic instrumentation complements LLM judgments.**
Crosscheck-with-ADD ships at least minimal deterministic instrumentation derivable from git history and the linkage graph: edit-frequency hotspots on spec files (Tornhill-style), change-coupling between specs and tests, orphan detection, cascade-pending detection, and diff-shape analysis (new clause / modified clause / deleted clause). The auditor agent consumes these signals before rendering verdicts.

*Observable signal:* a script or skill produces these signals as structured output. The auditor agent's prompt explicitly takes the structured output as input. The output format is stable enough that future skills can consume it.

---

**IC9 — Existing bootstrap-mode flows are unchanged for non-opting users.**
A user who does not invoke any ADD skill or set an ADD-mode flag experiences Crosscheck exactly as it was before this work. The recommended order in the README continues to work for existing-codebase users; no existing skill loses functionality.

*Observable signal:* every existing test in the repo passes after the ADD work merges. The pre-ADD recommended order in the README still produces sensible output on a representative existing codebase.

---

**IC10 — Documentation surfaces ADD as a peer to bootstrap mode, not as a replacement; the dual-track enforcement principle carries forward.**
The plugin README and `docs/skills.md` and `docs/agents.md` (or successors) describe ADD as an additional operating mode with its own recommended order. The relationship between bootstrap and ADD is honestly described, including the open problems listed in `methodology.md` § Open problems.

ADD's diff-classification gates (IC7) are an instance of the dual-track enforcement principle that `/assurance-init` already writes into every onboarded repo's `docs/assurance/ROADMAP.md` (verbatim block: pre-commit hook + CI job + LLM-free attestation check). The README and skill catalogue should frame ADD as continuing this discipline rather than introducing it.

*Observable signal:* README contains an "Operating modes" section. The recommended-order sections distinguish bootstrap-mode order from ADD-mode order. The "Honest Map" recommendation from prior synthesis docs is realised, with ADD reach honestly marked. The dual-track-enforcement principle is referenced when describing the diff-classification gates, with a pointer back to the existing `/assurance-init` template.

---

**IC11 — Behavioral-spec prose carries linkage quality.**
The behavioral spec (`B`-tier) authored by the agent in Phase 1 satisfies a minimum linkage quality: every `B` invariant traces via `consumes:` to at least one `IC` (intent claim, possibly via an intermediate `S` section) and via `produces:` to at least one `F` (functional spec section) within its module. A `B` with no `IC` ancestor is *orphaned* (governance violation). A `B` with no `F` descendant has no implementation seam; the integrity check flags it as `dangling-B`. Both conditions are mechanically detectable from the linkage graph.

This claim addresses the methodology's "compositional gap" open problem at the prose-quality level (without requiring a model checker, which is N4): if the `B`-tier has correctly-shaped traces, the prose layer is at least *structurally* a behavioral spec rather than a free-form essay. A model checker integration remains future work; structural integrity is v1.

*Observable signal:* the deterministic linkage-graph integrity check (S1.2 extended; S4.1) reports `orphan-B` and `dangling-B` counts; both must be zero for a Phase-3-or-later module to satisfy IC11. The auditor agent's verdict on a behavioral.md file with non-empty `orphan-B` count is Drifted; with non-empty `dangling-B` count is Active-with-warning. Phase-1 modules in mid-derivation may legitimately carry `dangling-B` until the F-tier is drafted.

---

## Out of scope (negative space)

These items are explicitly *not* part of the v1 ADD integration. Surfacing them here protects against drift toward them.

- **N1.** *Empirical layer attribution* — instrumenting which assurance layer catches what bug class. This is the largest open problem in `methodology.md`, but addressing it requires field data from at least one full ADD-mode project. Out of scope until such a project exists.
- **N2.** *Attestation batching* — ADD's likely mitigation for attestation fatigue. Premature; do not optimise. v1 attestations are per-artifact.
- **N3.** *Auto-promotion of artifacts from Drafted to Attested.* All attestation in v1 is human-driven. Even if an agent passes Phase 2 validation cleanly, the human attests.
- **N4.** *A behavioral-spec model checker integration* (TLA+, Alloy, P). The architectural spec acknowledges the `B`-tier exists; v1 produces behavioral specs as prose with cross-references but does not embed a model checker. The next iteration may add this.
- **N5.** *Differential random testing implementation* (Cedar/VGD-style). The architectural spec accommodates it as a future skill; v1 does not ship it.
- **N6.** *A full TiCoder-style interactive disambiguation skill.* Recommended in prior synthesis but separately scoped; v1 of ADD does not require it.
- **N7.** *Replacing the existing Byfuglien/Hellebuyck split.* The auditor agent is added as a peer, not by carving out work from the existing two. Their surfaces are unchanged.
- **N8.** *Marketing copy or external announcement.* v1 ships with documentation honest about ADD's hypothesis status; no external claims of validation are appropriate yet.

## Threat model

Failure modes the v1 design must rule out, and the rationale for each:

- **TM1 — ADD doc proliferation that humans cannot maintain.** If `docs/add/` grows faster than humans can attest, attestation fatigue collapses the methodology. Mitigation: keep the seed artifact set small (this directory at ~12 files), require explicit attestation, do not auto-promote.
- **TM2 — Silent spec drift in early projects.** If the diff classification rule is not enforced from day one, drift becomes invisible and the methodology fails the hardest case it was designed for. Mitigation: enforce diff classification on `docs/add/` commits from v1 (IC7); do not defer this.
- **TM3 — Bootstrap-mode users surprised by ADD-mode behavior.** If existing skills change behavior under ADD, users running existing flows break. Mitigation: ADD changes are additive (IC9). Existing skills detect mode via tagging; default behavior for un-tagged repos is bootstrap-mode.
- **TM4 — The auditor agent authoring artifacts.** If the auditor agent gains write authority on artifacts under audit, the audit/author separation collapses and audit verdicts become self-confirming. Mitigation: auditor produces verdicts only; remediations are *proposals* humans adjudicate. Codified in ADR-003.
- **TM5 — Premature optimisation of attestation cadence.** If we batch attestations, gate them on heuristics, or make them implicit, we lose the high-leverage human-judgment moments ADD depends on. Mitigation: v1 is explicit per-artifact attestation; batching is N2 (out of scope).
- **TM6 — The methodology drifting from its hypothesis status.** If documentation describes ADD as proven, users adopt it under wrong expectations. Mitigation: methodology.md explicitly enumerates open problems; README honesty is IC10.

## Attestation

This intent doc moves from Drafted to Attested when the human reviewer (nicholls-inc) confirms that:

1. Each `IC` captures a real intent rather than a process artifact.
2. The negative space (N items) is what is genuinely out of scope.
3. The threat model surfaces the failure modes worth ruling out.
4. Nothing material is missing.

Once Attested, supersession requires a new IC plus the old IC marked Superseded-by; no in-place rewrites.
