# Assurance Driven Development (ADD)

**Status:** Attested v1.0
**Supersedes:** none
**Last attested:** 2026-05-09 by nicholls-inc (Phase 2 closure on branch claude/validate-add-phase-2-bFneS)

## Definition

Assurance Driven Development is a methodology for AI-agent-executed software construction in which humans define the vision, govern the spec, and adjudicate ambiguity, while AI agents perform construction within bounds the spec makes explicit. The thesis: in an agent-executed SDLC, the *durable artifact* is the spec stack and its reasoning trace; code is a regenerable optimisation pass over it. ADD is the methodology for building that stack from a clean slate and keeping it honest as agents iterate against it.

ADD inverts how an assurance hierarchy is conventionally read. Code-first workflows climb the layers post-hoc: code exists, so we verify it; specs exist somewhere, so we check alignment; intent is implicit. ADD reads top-down from the start. Intent is captured first, specs are derived to serve the intent, and code is the *last* artifact to enter the picture, gated against everything above it.

## Lineage

ADD is a synthesis, not a novel invention. Six prior practices, each with traceable origins, supply its discipline:

- **The IETF RFC tradition** (Steve Crocker, RFC 1, April 1969). Written, distributed, numbered, never deleted.
- **Architecture Decision Records** (Michael Nygard, "Documenting Architecture Decisions," November 2011). ADRs encode forces and tradeoffs, not just conclusions; they are repo-resident, numbered monotonically, and superseded rather than deleted.
- **Domain-Driven Design / Ubiquitous Language** (Eric Evans, *Domain-Driven Design*, 2003). A common, rigorous language between developers and domain experts that lives all the way into the source code.
- **Behavior-Driven Development / Specification by Example** (Dan North, "Introducing BDD," March 2006). Given/When/Then scenarios as executable acceptance criteria; specifications double as tests.
- **Design docs at scale** (Google, Uber, Amazon, and others, codified through the 2010s). Written before code, reviewed across teams, serve as organisational memory.
- **Formal-methods stack** (Lamport's TLA+ "blueprints" framing; AWS Cedar's Verification-Guided Development; the Crosscheck assurance hierarchy). Specifications can be model-checked, proven, mutated, and adversarially probed.

The human-powered conventions were optimised for *durability across time and personnel turnover* — the same constraint agent-executed development faces. The forms (written, repo-resident, numbered, status-tracked, cross-referenced, force-encoding) port directly. What requires adaptation: surfacing reasoning traces humans could leave implicit, formalising cross-reference graphs into typed IDs agents can traverse, and adding deterministic instrumentation where humans previously substituted judgment.

## Operating principles

**Humans own intent and governance; agents own construction.** Humans articulate vision, negative space (what is out of scope), threat model, and the moments of "stop, this isn't the system we wanted." Agents draft specs, derive lower-level specs, write code, run model checkers, perform adversarial probing, and iterate. The interface is a small set of high-leverage *attestation* points — narrowly-scoped questions whose answers are committed alongside the artifact.

**The repository is the source of truth.** Nothing of consequence lives in chat. Every claim, decision, alternative considered, rejection rationale, and supersession is a repo-resident artifact. Agents have no synchronous channel; the audit trail is the only channel.

**Specs encode forces and tradeoffs, not just conclusions.** An ADR-style structure — Context, Alternatives Considered, Decision, Rejected Approaches, Consequences — is mandatory at every spec layer. The downstream agent needs the *why* to make compatible local decisions; without it, it reinvents or contradicts the original reasoning. This is the cross-agent intent-transmission solution the methodology depends on.

**Cross-references are a typed graph, not prose links.** Stable IDs across all artifacts make auditing tractable. Orphans (an invariant with no test, a test with no spec, a spec with no intent claim) are mechanically detectable and prohibited. See `glossary.md` for the canonical ID conventions.

**Status is explicit and monotonic.** Every artifact carries one of: Drafted, Attested, Ratified, Superseded-by-N, Retracted-with-Reason. Transitions are recorded; nothing is deleted.

**Deterministic tools detect signals; LLMs render judgments.** Deterministic tools count, track linkage integrity, detect coupling, and compute time-series. LLMs compare prose against prose, classify diffs as discovery vs drift vs refinement, adversarially probe, and back-translate. Neither is sufficient alone; together they make consolidation tractable. See `decisions/ADR-002-deterministic-llm-split.md`.

**Governance applies progressively, not uniformly.** Three gates determine which assurance tools are available against an artifact: vintage (how mature), coverage (what surrounding artifacts exist — `intent-check`'s prose-vs-test triple is unavailable when no test exists yet, so prose-vs-prose is the substitute), and confidence (how much pressure-testing the artifact has survived). Tool availability falls out of artifact state rather than being enumerated by phase.

## Operating modes

Three modes coexist; modules carry a tag indicating their origin, and governance is mode-appropriate. See `decisions/ADR-001-operating-modes.md` for the full discussion.

**Bootstrap mode.** Existing codebase. Recover intent from code, retrofit governance over already-running implementation. Risk: rationalisation of what got built rather than what was wanted. Crosscheck pre-ADD lives here.

**ADD mode.** Clean slate. Start with intent, derive specs, prefigure governance, gate code. Risk: pristine intent that turns out to be unbuildable.

**Transitional mode.** A *repo-level* descriptor — not a per-module tag — for a partially-built system where early modules are bootstrap-mode (governance retrofitted) and new modules are ADD-mode (governance prefigured). The common case in practice. Per-module mode tags draw from `{bootstrap, add}` only; the repo is in transitional mode whenever modules disagree. Governance applied to each module is mode-appropriate. A bootstrap-mode module is not expected to have an intent-attestation trail back to Phase 0; an ADD-mode module is not expected to have its specs treated as recoverable from code. See `decisions/ADR-001-operating-modes.md` for the rationale.

## Phase structure

ADD has five phases. Phases 0 through 3 are pre-code; Phase 4 is gated implementation; Phase 5 is continuous assurance.

**Phase 0 — Intent capture.** Articulate three things, each numbered (`IC1`, `IC2`, ...) and ADR-formatted:
- Success criteria as observable behaviors
- The explicit out-of-scope list (negative space)
- The threat model (which failure modes the design rules out, which it tolerates)

The intent doc is a protected surface from day one. Humans attest before Phase 1 begins.

**Phase 1 — Specification.** Three tiers, derived top-down. Every spec section declares `consumes:` (which intent claims and upstream sections it depends on) and `produces:` (which lower-level constraints it generates).

- *Architectural* (`S` IDs): modules, responsibilities, flows between them
- *Behavioral* (`B` IDs): per-module invariants across reachable states — TLA+/Alloy/P territory, valuable wherever modules have non-trivial state, workflow branching, or rule interactions
- *Functional* (`F` IDs, with associated `I` invariants): per-operation pre/post conditions — Dafny-or-equivalent

Granularity rule: a spec is detailed enough when an independent implementer (human or agent) could read it and produce conformant code without asking clarifying questions. Anything more is implementation in disguise.

**Phase 2 — Spec validation.** Three independent checks, run before any code exists:

- *Self-consistency*: the spec compiles, model-checks, or Dafny-verifies with stub bodies
- *Intent alignment*: a fresh agent reads the spec cold, describes what system it specifies, and the description is compared to the Phase 0 intent doc — prose-vs-prose, since no tests exist yet
- *Adversarial completeness probing*: what behaviors does the spec leave unconstrained?

Phase 2 ends when Phase 0 acceptance scenarios are derivable from the spec without further interpretation. Humans attest that derived specs still serve higher-level intent.

**Phase 3 — Skeleton and governance install.** Materialise the file structure: one module directory per architectural module, invariant doc populated, property-test files with one stub per invariant (currently failing), implementation files with type-only signatures and unimplemented bodies. Install governance: protected-surface rules, linkage-graph integrity check (no orphans), diff classifier, audit log. Phase 3 ends with fully red CI: zero passing tests, all governance live, every spec artifact ratified.

**Phase 4 — Gated implementation.** Code is written module by module. Three commit shapes are legal:

- *Implementation*: turns one or more tests green without modifying any spec or invariant
- *Governance amendment*: modifies a spec and the tests covering it together, with an attached classification (see Diff classification below) and the cascade re-derivation through any lower spec layers that consumed the changed section
- *New invariant*: goes through Phase 1-and-2-equivalent probing before merging

Anything else is rejected. Crosscheck-style round-trip checks, regression re-verification, and adversarial spec probing become well-defined here because the triples (prose, test, code) now exist. Differential random testing becomes available wherever both a model-side oracle and an implementation exist — and ADD makes this case the default rather than the exception, because the model is built before the implementation.

**Phase 5 — Continuous assurance with consolidation passes.** Drift detection, FP tracking, kill criteria, regression re-verification, *plus* a named consolidation pass at regular cadence (weekly in early projects, monthly or quarterly later). The consolidation pass is owned by an *auditor agent*, distinct from the agents that author specs or write code (see `decisions/ADR-003-auditor-agent.md`). It does not propose changes; it produces a verdict on every artifact:

- *Settled* — linkage graph intact, no recent edits, no upstream amendments since last attestation
- *Active* — recent legitimate work in progress
- *Drifted* — signals indicate divergence (upstream intent amended N edits ago, downstream spec not re-derived; or test modified but invariant prose not)

For drifted artifacts, the auditor agent renders a natural-language judgment of severity and proposes remediation. Humans adjudicate.

## Diff classification

The single structural element that protects ADD from collapsing into waterfall on one side and into drift on the other. Every spec change is classified as one of:

- **Propagated discovery** — implementation revealed something that should have been known earlier and now is. The spec is amended; the cascade re-runs; lower specs are re-derived; the discovery is logged. Healthy and expected, especially in early projects.
- **Intent refinement** — the human's understanding of what they want has improved. Phase 0 is amended; Phase 1 cascade re-runs. The most significant kind of change; requires explicit human attestation.
- **Drift** — the spec is being weakened to match what got built. Flagged; requires human approval and a written justification answering *"did we want this behavior or did the implementation drift?"*. Drift is not always wrong, but it is never silent.
- **Retraction** — a previously-made claim is being abandoned. Logged with reason; the linkage graph is updated; downstream artifacts that consumed the retracted claim enter Drifted state until amended.
- **Status transition** — an artifact's Status field flipped (Drafted → Attested, Attested → Ratified, anything → Superseded-by-N or Retracted-with-Reason) without content change. Isolated from the four content classes above so the audit log does not conflate status flips with substantive iteration.

Forcing the agent to *classify the diff* — and the auditor agent to *verify the classification* — is what catches drift early. Healthy convergence shows decreasing rate of cascade-triggering diffs over time, increasing fraction classified as Intent Refinement rather than Propagated Discovery, and stable-or-decreasing rejection rate at human attestation. Thrash shows the opposite. Both signals are computable from the diff log when metadata is recorded structurally. See `decisions/ADR-005-diff-classification.md`.

## Tooling and instrumentation

ADD assumes deterministic and LLM tools layered against the same artifacts.

**Deterministic instrumentation**, derivable from git history and the linkage graph:
- Edit frequency per spec/invariant/test (Tornhill-style hotspot analysis applied to specs, not code)
- Change-coupling between spec and test files (an invariant edited 12 times in 6 weeks while its tests were edited 0 times is a signal)
- Orphan detection across the linkage graph
- Cascade-pending detection (upstream intent claim was amended at commit C; downstream specs that consumed it have not been re-attested since C)
- Diff-shape analysis (new clause / modified clause / deleted clause)

**LLM-mediated work**, layered on top of those signals:
- Prose intent vs prose code description comparison
- Adversarial completeness probing
- Back-translation
- Diff classification
- Consolidation-pass judgment

Crosscheck's existing engines (round-trip prompt structure, protected-surface partition, FP tracker, kill criteria, certificate format with premise tagging, Dafny verification, mutation kickers) are largely usable without modification. What is missing for ADD: empty-repo entrypoints, prose-vs-prose intent checking, spec-first adversarial probing, the auditor agent role, and the deterministic linkage-graph layer.

## Roles

Three agent roles, two pre-existing in Crosscheck and one new:

- **Byfuglien** (implementation chain) — pre-existing. Owns Layer 1 verification and the four orthogonal semi-formal reasoning skills. ADD does not change Byfuglien's surface area.
- **Hellebuyck** (specification chain) — pre-existing. Owns Layers 4-6 plus governance scaffolding. ADD extends Hellebuyck's surface to include the spec-first / empty-repo workflow (Phase 0 through Phase 3).
- **Auditor** — new, introduced by ADD. Owns consolidation passes. Does not author or modify artifacts. See `decisions/ADR-003-auditor-agent.md`.

Humans own intent capture, attestation at phase boundaries, and adjudication of auditor verdicts.

## Open problems

ADD is a hypothesis, not an evidence-backed practice. Its claims are testable but not yet tested. The most significant open problems, in approximate order of severity:

1. **Empirical layer attribution.** No data yet exists on which assurance layer catches what bug class in an ADD-mode project, or whether the upfront cost is recovered downstream. Tier-4 instrumentation (publishing which layer caught what, with bug-class taxonomy) is the single most credibility-enhancing artifact ADD could ship.
2. **The fidelity gap.** Specs that have never been implemented are aspirational. The methodology depends on tight Phase 4 ↔ Phase 1-2 iteration to keep them grounded. Without iteration, ADD becomes waterfall.
3. **The granularity question.** Specs too detailed reproduce implementation in spec form; specs too coarse don't constrain enough. The right granularity is domain-dependent and usually only learned by doing it badly once. A team's first ADD project should expect to throw away or substantially rewrite specs as it discovers the value-per-friction peak. The diff classification rule is what makes this rewriting healthy rather than chaotic.
4. **Attestation fatigue.** If every artifact requires human attestation and agents produce artifacts orders of magnitude faster than humans review, attestation becomes the new bottleneck. Batching is the likely mitigation, but the right batching shape is empirical. Do not premature-optimise.
5. **The compositional gap.** Per-module functional specs verify each module in isolation. Cross-module behaviors require behavioral specs (the `B`-tier). ADD without a behavioral-spec layer is mostly Dafny-flavoured TDD.
6. **The spec-as-code problem.** Specifications are themselves artifacts that can be buggy, vacuous, or contradictory. Phase 2 is the partial answer; the full answer is iteration plus the auditor agent.

## Net positioning

ADD is not a replacement for the assurance hierarchy; it is the methodology for *applying the hierarchy from a clean slate*, in a world where humans are the source of intent and judgment and agents perform construction. The hierarchy itself becomes a *toolkit* whose tool availability is determined by artifact state rather than by phase. The substantive claim is that intent is the foundation, specs are load-bearing, and code is the surface that conforms. Layers 4–6 become the construction order; Layer 1 becomes the optimisation pass.

## References

- Crocker, S. *Host Software*. RFC 1, IETF, April 1969.
- Nygard, M. *Documenting Architecture Decisions*. Cognitect blog, November 2011. https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions
- Evans, E. *Domain-Driven Design: Tackling Complexity in the Heart of Software*. Addison-Wesley, 2003.
- North, D. *Introducing BDD*. Better Software magazine, March 2006. https://dannorth.net/introducing-bdd/
- Ubl, M. *Design Docs at Google*. Industrial Empathy, 2020. https://www.industrialempathy.com/posts/design-docs-at-google/
- Lamport, L. "Who Builds a House Without Drawing Blueprints?" *Communications of the ACM* 58(4), April 2015.
- Newcombe, C. et al. "How Amazon Web Services Uses Formal Methods." *Communications of the ACM* 58(4), April 2015.
- Disselkoen, C. et al. *How We Built Cedar: A Verification-Guided Approach*. arXiv:2407.01688, 2024.
- Tornhill, A. *Software Design X-Rays: Fix Technical Debt with Behavioral Code Analysis*. Pragmatic Bookshelf, 2018.
- Crosscheck `docs/research/assurance-hierarchy.md` and `docs/research/literature-review.md` (this repo).
