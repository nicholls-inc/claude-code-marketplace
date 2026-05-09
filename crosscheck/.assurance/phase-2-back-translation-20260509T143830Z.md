# Phase 2 Back-Translation — ADD into Crosscheck

**Author:** Claude Code agent (cold-read back-translation)
**Date (UTC):** 2026-05-09T14:38:30Z
**Inputs consulted while writing:** None of `intent.md` or `specs/architectural.md`. Written from memory of the cold read of `methodology.md`, `glossary.md`, `decisions/INDEX.md`, the five ADRs, `intent.md`, and `specs/architectural.md` (the latter two were read once during the cold read but are *not* open while drafting this).
**Status:** Drafted — input to the comparison report; not itself an attestation.

---

## 1. What system this directory is asking us to build

The directory `crosscheck/docs/add/` is the seed for adding a new operating mode to the **Crosscheck** plugin called **Assurance Driven Development (ADD) mode**. The deliverable is *not* a new plugin; it is an extension of the existing Crosscheck plugin so that its skill catalogue, agent registry, and governance machinery accommodate three distinct operating modes:

- **Bootstrap mode** — the way Crosscheck works today: assurance machinery applied over an already-existing codebase, with intent recovered post-hoc from code and tests.
- **ADD mode** — clean-slate / spec-first: the user starts with only a written vision; intent is captured first, specs are derived top-down from intent, governance is prefigured before any code, and code is the *last* artifact to enter the picture, gated against the spec stack.
- **Transitional mode** — a *repo-level* state that arises naturally when some modules originated in bootstrap mode and others were authored in ADD mode. Per-module mode tags carry the distinction; transitional itself is not a per-module tag.

Concretely, v1 ships:

- A small set of **new skills** (the "greenfield skill set") covering Phase 0 (intent elicitation), Phase 1 (architectural-spec derivation from intent), and Phase 2 (prose-vs-prose intent-vs-spec validation, plus adversarial probing of pre-code specs).
- **Additive adaptations** to a small number of existing Crosscheck skills so they detect empty-repo or ADD-mode state and route appropriately, without changing behavior for repos that have not opted in.
- A **third agent role**, peer to the existing Byfuglien (implementation chain) and Hellebuyck (specification chain) — an *Auditor* that runs scheduled consolidation passes over ADD artifacts. The Auditor is read-only on the artifacts it audits; its job is to render per-artifact verdicts of *Settled / Active / Drifted*, grounded in deterministic signals.
- A **deterministic instrumentation** layer (a script or skill) that computes structured signals from git history and the linkage graph: edit-frequency hotspots on spec files (Tornhill-style), change-coupling between specs and tests, linkage-graph integrity (orphans, dangling refs, cycles), cascade-pending detection, and diff-shape analysis. The Auditor consumes this output as primary input.
- A **mandatory diff-classification gate** on every commit that touches an ADD or governance artifact: each such commit must carry a structured trailer assigning the diff to one of four classes — *Propagated discovery / Intent refinement / Drift / Retraction*. Enforcement is dual-track: a fast pre-commit hook (no LLM) and a CI job that appends to a durable, queryable log (`.assurance/diff-classification-log.csv` or equivalent).
- **Documentation updates** that introduce ADD honestly: an "Operating modes" section in the plugin README, a recommended-order section that distinguishes bootstrap-mode order from ADD-mode order, additions to the skill catalogue and agent registry, and a methodology pointer that acknowledges ADD's hypothesis (rather than evidence-backed) status.

The directory is also itself an ADD-mode project. The seed artifacts (methodology, glossary, intent, ADRs, architectural spec, acceptance) were authored by the human; the agent is expected to validate them via the Phase 2 protocol *before* drafting any further specs or code.

## 2. What problem this is intended to solve

Crosscheck today assumes the user already has a codebase. Its skills:

- Scan manifests and source files to project an "assurance hierarchy" view.
- Ask the user to "name 1-3 load-bearing modules" — unanswerable when no modules exist.
- Require an `(invariant prose, covering test, code diff)` triple for `/intent-check` validation — none of which exist before code.
- Detect "surfaces" by scanning the file tree — useless on an empty tree.
- Adversarially probe ratified invariant docs — there are none on a fresh repo.

A user whose *only* artifact is a written vision therefore experiences friction or empty diagnostics from the recommended order in the Crosscheck README. The opportunity is real: in an agent-executed SDLC, the durable artifact is the spec stack and its reasoning trace; code is a regenerable optimisation pass over it. Crosscheck has the assurance hierarchy and most of the engines (round-trip prompts, FP tracker, kill criteria, Dafny verification, mutation kickers); what it is missing is the *empty-repo entrypoint*, the *prose-vs-prose intent-validation* step, *spec-first adversarial probing*, the *auditor agent* role, and the *deterministic linkage-graph layer* that consolidation passes need.

The deeper problem ADD as a methodology addresses: when humans articulate vision and agents execute construction at machine speed, the bottleneck shifts from typing code to keeping intent honest as agents iterate. Diff classification, attestation gates, and the auditor role are the structural mechanisms that prevent silent spec weakening (drift) and keep the human's high-leverage judgments on the audit trail rather than in chat.

## 3. Who the users are

Three audiences, each with distinct touch points:

- **Greenfield human developers** — humans starting a new repository with only a vision, who want to use Crosscheck to capture intent, derive specs, install governance, and then let agents implement. Today they hit friction; v1 of ADD-in-Crosscheck unblocks them.
- **Existing Crosscheck users on existing codebases** — humans whose repos already contain manifests, modules, tests, and so on. ADD must not break their flows. Default mode for an un-tagged module is `bootstrap`. Recommended order in the README continues to work.
- **AI agents executing under ADD** — Claude Code agents (or successors) drafting specs, deriving lower tiers, writing implementation code, classifying their own diffs, and submitting to governance gates. Agents draft; humans attest. Construction is delegated; intent and adjudication are not.

A fourth, narrower audience: **humans governing ADD-mode projects** — the same people as audience 1 in many cases, but in their *attesting* role at phase boundaries, *adjudicating* role on Auditor verdicts, and *amending* role on Drafted artifacts.

## 4. Success criteria (what v1 done looks like)

I recall roughly ten numbered intent claims (the `IC1`–`IC10` series). My recall of each, in summary form:

- **IC1 — Empty-repo entrypoint works.** A user with only a vision and an empty git repo can begin Phase 0 of ADD without any Crosscheck command failing or producing empty diagnostics.
- **IC2 — Phase 0 intent capture is an explicit, repo-resident artifact.** Numbered `IC` claims, an explicit out-of-scope list, a threat model. Created interactively with a skill, not from a blank page.
- **IC3 — Phase 1 spec stack derives top-down from intent, with `consumes:` / `produces:` linkage.** Architectural, behavioral, and functional tiers. The linkage graph integrity check passes (no orphans).
- **IC4 — Phase 2 validation is prose-vs-prose, not prose-vs-test.** A fresh agent reads the spec cold, back-translates it, and the back-translation is compared against intent. Available before any test or code exists.
- **IC5 — Three operating modes with per-module mode tags.** Mode tag in the module's invariant doc (or equivalent ADD-mode doc); skills consulting governance honour the tag.
- **IC6 — The Auditor agent runs consolidation passes.** A new agent definition file, plus a workflow that runs the pass on demand or on schedule.
- **IC7 — Diff classification is enforced on spec-changing commits.** Pre-commit hook + CI gate; durable log.
- **IC8 — Deterministic instrumentation complements LLM judgments.** At minimum: edit-frequency hotspots, change-coupling, orphan detection, cascade-pending. Auditor consumes these as primary input.
- **IC9 — Existing bootstrap-mode flows are unchanged for non-opting users.** Every existing test continues to pass. No existing skill loses functionality. Recommended order in the README still works for existing-codebase users.
- **IC10 — Documentation surfaces ADD honestly as a peer to bootstrap mode, not as a replacement.** README contains an "Operating modes" section. Hypothesis status is acknowledged. Open problems are surfaced.

## 5. What is explicitly out of scope (negative space)

I recall about eight `N` items:

- **N1 — Empirical layer attribution** — instrumenting which assurance layer catches what bug class. The largest open problem in the methodology, but it requires field data from at least one full ADD-mode project. Out of scope until such data exists.
- **N2 — Attestation batching** — premature optimisation; v1 attestations are per-artifact.
- **N3 — Auto-promotion of artifacts from Drafted to Attested.** Attestation is human-driven, even if Phase 2 passes cleanly.
- **N4 — Behavioral-spec model checker integration** (TLA+, Alloy, P). v1 acknowledges the `B`-tier exists but produces behavioral specs as prose with cross-references; no embedded model checker.
- **N5 — Differential random testing implementation** (Cedar/VGD-style). v1 accommodates it as a future skill but does not ship it.
- **N6 — Full TiCoder-style interactive disambiguation skill.** Recommended elsewhere but separately scoped; v1 of ADD does not require it.
- **N7 — Replacing the Byfuglien/Hellebuyck split.** The Auditor is added as a *peer*; existing roles are not carved up. Their surfaces are unchanged.
- **N8 — Marketing copy / external announcement.** v1 documentation is honest about hypothesis status; no external claims of validation.

## 6. Threats to validity the design rules out (threat model)

I recall six `TM` items:

- **TM1 — Doc proliferation that humans cannot maintain.** Mitigation: keep the seed artifact set small (~12 files), explicit attestation, no auto-promotion.
- **TM2 — Silent spec drift in early projects.** Mitigation: enforce diff classification from day one; classification is mandatory on every commit touching an ADD or governance artifact.
- **TM3 — Bootstrap-mode users surprised by ADD-mode behavior.** Mitigation: ADD changes are additive; un-tagged modules default to `bootstrap`; existing flows preserved.
- **TM4 — Auditor authoring artifacts under audit.** Mitigation: Auditor is read-only on audited artifacts; produces verdicts and *proposed* remediations only; humans adjudicate.
- **TM5 — Premature optimisation of attestation cadence.** Mitigation: explicit per-artifact attestation in v1; batching is N2.
- **TM6 — Methodology drifting from its hypothesis status.** Mitigation: methodology.md explicitly enumerates open problems; README honesty is IC10.

## 7. What the agent (Claude Code) is supposed to do, and what the human is supposed to do

**Boundary rule:** humans own intent and governance; agents own construction. Drafted artifacts authored by the human can be amended by the agent only with the human's explicit confirmation in the same exchange. Ratified artifacts cannot be modified in place; a supersession ADR is the only path.

**The human authors:**

- `methodology.md` — the canonical ADD reference.
- `glossary.md` — the ubiquitous language and ID conventions.
- `intent.md` — the Phase 0 intent doc (IC claims, negative space, threat model).
- `decisions/ADR-001` through `decisions/ADR-005` — the foundational ADRs (operating modes; deterministic-vs-LLM split; Auditor as third role; minimal greenfield skill set; mandatory diff classification).
- `specs/architectural.md` — the Phase 1 architectural spec.
- `acceptance.md` — the Phase 2 protocol and v1 acceptance criteria.

**The agent authors (after Phase 2 attestation):**

- `specs/behavioral.md` — the per-module behavioral invariants (`B` tier).
- The per-module functional specs (`F` tier with associated `I` invariants), one file per module under `specs/modules/` or a single `specs/functional.md`.
- New SKILL.md files for the four greenfield skills (Phase 0 elicitation, architectural derivation, prose-vs-prose validation, prose-mode adversarial probing).
- Delta specs (additive only) for the small number of existing skills that need to detect empty-repo or ADD-mode state.
- The Auditor agent definition file.
- The consolidation-pass workflow document.
- The deterministic-instrumentation tool (script or skill) and its schema doc.
- The pre-commit hook stub, the CI job stub, the diff-classification log schema doc.
- Updates to the Crosscheck README, skill catalogue, and agent registry.
- Subsequent ADRs as new decisions arise.

**The agent's first task (this Phase 2 protocol):**

Cold-read the seed; write a blind back-translation; open intent and architectural side-by-side and produce a structured comparison covering matches, gaps in intent, gaps in spec, contradictions, under-specifications, out-of-scope drift candidates, and adversarial probes of the spec stack. Emit a verdict — PASS / PASS-WITH-AMENDMENTS / HOLD — and *stop*. Do not draft any new artifact, modify any Drafted seed artifact without confirmation, or proceed to Phase 1 lower-tier work until the human attests.

## 8. Things I am uncertain about from the cold read

Honest admissions, written before opening intent.md or architectural.md to compare:

- I am unsure whether **transitional mode** has its own per-module tag or is purely a repo-level descriptor. I recall the methodology saying "modules carry a tag" but I also recall an ADR ruling out a `mode: transitional` per-module tag. The architectural spec presumably resolves this; I will check.
- I am unsure whether `/intent-check-prose` and `/spec-adversary-prose` are committed as **new skills** or as **modes of existing skills**. ADR-004 leaves the choice to the architectural spec for at least one of them; I recall the spec recommending one of each but not which.
- I am unsure of the exact path of the diff-classification log (`.assurance/diff-classification-log.csv` is what I recall, but the schema spec mentions JSON-lines as acceptable).
- I am unsure whether the Auditor's name has been chosen or whether the slug `auditor` is a placeholder.
- I am unsure of the exact list of existing Crosscheck skills the architectural spec adapts. I recall five (`/assurance-layer-audit`, `/assurance-init`, `/intent-check`, `/spec-adversary`, `/acceptance-oracle-draft`) — but I want to verify whether `/spec-iterate` is in or out of the adaptation list.
- I am unsure whether the integrity-check rule for ADD-mode modules requires tests at all phases or only from Phase 4 onwards. The architectural spec is what makes that mode-aware; I recall a "phase-aware" qualifier but want to verify.
- I am unsure whether the Phase 2 protocol explicitly authorises the agent to read other Crosscheck files (`README.md`, `agents/byfuglien.md`, etc.) before producing the verdict. The protocol I recall says the cold read is *only* the seed; reading existing Crosscheck files is for *after* Phase 2 attestation. (The user's external task brief contradicted this somewhat by listing existing Crosscheck files to read; I am following the protocol as stated in `acceptance.md` since the protocol takes precedence.)

These uncertainties are flagged here so the comparison report can address each one explicitly rather than letting them leak silently into the verdict.
