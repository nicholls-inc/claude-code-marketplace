# A Six-Layer Assurance Hierarchy for AI-Assisted Software Development

## Scope

This hierarchy addresses the question: **given a specification, can we be confident the implementation is correct?** It explicitly excludes whether the specification solves the right problem — that is a product discovery concern, not an engineering assurance concern.

## Framing: layered assurance

The 6-layer hierarchy described below is *one structural element* of a broader framework, not the framework itself. The framework is layered formal verification engines + probabilistic complements + stochastic complements, composed and routed per module:

- **Formal verification engines.** Dafny operationalised (Layer 1 verify-and-extract). Lean joining as a peer Layer-1 engine for the executable-model + DRT-oracle role (different from Dafny's role; see Layer 1 below). TLA+/P/Alloy as Layer 4 enrichment for rule-dense modules (state machines, workflows with branches, role hierarchies, invariant-rich data) — pending an ADR for the Layer 4 redefinition.
- **Probabilistic complements.** Property-based testing and differential random testing exercise large input spaces. The Cedar paper's bug taxonomy (Disselkoen et al. 2024) shows ~15/21 of DRT-found bugs were general implementation bugs (parsing, dependencies, error handling), supporting broad applicability of DRT-as-technique.
- **Stochastic complements.** `/intent-check` round-trip informalization, `/spec-adversary` heuristic probing, and the four semi-formal reasoning skills cover surfaces formal methods don't reach.

**Verification-Guided Development (VGD)** is one methodology under this framework — Amazon's process for Cedar — that applies *where its four prerequisites are met at the module level*: (1) deterministic algebraic semantics, (2) provable properties, (3) tractable input generation for DRT, (4) resources for dual development. We hypothesise AI-augmented development substantially reduces the marginal cost of (4) in 2026; no empirical baseline exists for AI-augmented dual development (Cedar 2024 used human Lean + human Rust). Treat (4) as untested working assumption. (1)–(3) still bind — they are properties of the *problem*, not the team. `/assurance-layer-audit` and `/assurance-init` operationalise per-module prerequisite assessment as the routing primitive (pending Phase 3d implementation).

**Adoption disposition (Brooker, diagnostic only).** Brooker's *hubris / humility / laziness* triad — believing software can be correct, accepting one's own fallibility, refusing to fix the same bug twice — is useful as self-assessment for teams considering whether VGD-style discipline will stick. It is **not a gate, checklist, or precondition** in any Crosscheck skill. Teams without the disposition will tend to let governance scaffolding decay; that is a known operational risk, not a hiring criterion.

## The Hierarchy

### Layer 1: Formally Verified Pure, Functional Code

Business logic and core algorithms are written in a formally verifiable language such as Dafny, which compiles to target languages including JavaScript and TypeScript. The formal verification toolchain (Dafny's verifier, or Lean 4 via tools like Axiom or Harmonic's Aristotle) provides mathematical proof that the code satisfies its specification for all possible inputs — not just tested inputs.

This is deterministic assurance: the proof either passes or it doesn't.

**Two engines, two roles.** Dafny and Lean both sit at Layer 1 but in complementary roles, not as alternatives. Dafny is *verify-and-extract*: code is generated from the verified spec and compiled to Python or Go via the Dafny backends. Lean is the *executable model + DRT oracle*: production code is hand- or AI-written (or Dafny-extracted, for partial-verification cases), and a Lean 4 model serves as the oracle for differential random testing. Lean has no production-grade compiler to mainstream languages — the pattern is hand-write production code, validate via DRT, not generate from Lean. Producing usable Lean models is itself a multi-step pipeline (informal spec → Lean spec stub → Lean impl → correspondence review → DRT), well-precedented by GitHub Next's *Lean Squad*. Dafny gives compile-time correctness on the pure-function slice (~22–27% reach band); Lean + DRT gives sample-based correctness on the much larger non-pure slice. Both are needed; Phase 3b lands the Lean side of the engine pair (pending implementation).

### Layer 2: Compilation Correctness

Formal verification proves properties of source code, but the deployed system runs compiled output. If the compiler has a bug, the emitted code may not preserve the properties proven in the source language. This layer provides assurance that the compilation or transpilation step preserves verified properties.

Approaches include verified compilation (e.g., CompCert for C), translation validation (verifying each specific compilation output against its source), or proof-carrying code where proofs travel with the compiled artifact.

Without this layer, a formally verified Dafny program and its compiled JavaScript output are connected only by trust in the compiler — an unverified link in an otherwise verified chain.

### Layer 3: Contract Graph Verification

Individual verified units must compose correctly. The contract graph verifier checks interface contracts at integration boundaries — between verified units and between verified code and third-party libraries. Critically, this verification operates end-to-end across subgraphs, not just at pairwise boundaries.

This distinction matters because two units can each satisfy their boundary contracts while producing emergent behavior that neither unit's specification anticipated — ordering dependencies, resource contention, or feedback loops that only manifest in the full assembly. End-to-end subgraph verification catches these composition failures.

This layer targets the approximately 75% of code surface area that sits at integration boundaries, which resists formal proof of individual units and is where the majority of production bugs originate.

### Layer 4: Implementation–Specification Alignment

Layers 1–3 prove that code satisfies a specification. This layer asks: does the specification actually describe the implementation's behavior? Tools like Midspiral's lemmafit integrate Dafny verification into the development workflow, where code that cannot be proven correct against the specification does not compile.

This is still deterministic — the verifier accepts or rejects.

### Layer 5: Specification–Intent Alignment

A verified proof guarantees the code is correct relative to a specification, but does the specification capture what was actually intended? This is the gap between "verified" and "intended."

Midspiral's claimcheck addresses this via round-trip informalization: translate the formal specification back into plain English without seeing the original requirement, then compare the back-translation against the original intent. In testing, this caught both planted errors and unexpected gaps — for example, a requirement stating "adding a ballot can't decrease a tally" where the lemma merely proved counts are non-negative, a tautology masquerading as a monotonicity guarantee.

This layer is probabilistic. Claimcheck reports approximately 96.3% accuracy using structural separation (two models, with the informalizer blind to the original requirement), though this is acknowledged as a development benchmark rather than a formal evaluation.

### Layer 6: Specification Completeness

Even if every specified property is correctly implemented and aligned with intent, the specification itself may be incomplete — it may fail to enumerate properties that matter. Traditional user stories test the happy path and some error cases, getting nowhere near the exhaustive coverage that formal verification provides.

The concept of formally verified user stories addresses this: rather than writing a handful of test cases from a user story, an LLM systematically enumerates candidate formal properties — invariants, pre/post conditions, boundary behaviors, commutativity, monotonicity, conservation laws. A human reviews the property list, each property is formally verified, and claimcheck validates intent alignment.

This layer is best-effort. There is no theorem that can prove a specification is complete — the question "have I missed any important properties?" is inherently a human judgment call. However, adversarial property discovery — where one agent proposes properties and another agent tries to find scenarios the property set doesn't cover — could make this search significantly more structured than the status quo.

## Two Chains, One Gradient

The hierarchy contains two distinct chains with different failure modes and verification methods:

**The Implementation Chain (Layers 1–3):** Is the code correct and do the pieces fit together? This chain is deterministically verifiable via formal proofs and contract checks.

**The Specification Chain (Layers 4–6):** Does the spec match the code, the intent, and the full problem surface? This chain degrades from deterministic (Layer 4) to probabilistic (Layer 5, ~96%) to best-effort (Layer 6).

Residual risk concentrates at the top of the specification chain — Layer 6 — which is where the research opportunity lies.

## Applying the hierarchy to a real codebase

How far each layer reaches in practice depends on the host language, the maturity of its verification tooling, and the shape of the system being built. The notes below describe what tends to be reachable in mainstream ecosystems (Go, Python, TypeScript, Rust); the crosscheck plugin's skills are referenced where they directly support a layer.

**Layer 1 (formally verified pure code).** Reachable today for sequential, pure logic via Dafny, with extraction to Python or Go. The crosscheck plugin's `/spec-iterate`, `/generate-verified`, and `/extract-code` skills cover this path. The Lean-side pipeline is partial as of sub-phase 3b-α: `/informal-spec` (prose-spec extraction with hard human sign-off) and `/lean-spec` (signed-off prose → Lean 4 stub with `sorry` proof bodies, gated on `lake build` clean) ship the first two of five planned steps; `/lean-impl`, `/correspondence-review`, and `/drt-oracle` remain pending sub-phase 3b-β. Concurrent and effectful code falls outside the verifiable surface and must be handled at higher layers.

**Layer 2 (compilation correctness).** Aspirational outside niche stacks. CompCert exists for C; nothing equivalent exists for Go, Python, TypeScript, or Rust. In mainstream ecosystems, the production compiler is part of the trusted computing base.

**Layer 3 (contract graph verification).** Pairwise interface contracts are reachable in some languages (e.g., Gobra for Go, refinement types in Liquid Haskell, contract decorators in Python). End-to-end subgraph verification across a real service remains a research frontier — viable for distributed protocols (Veil on Lean) but not yet a turnkey practice for application code.

**Layer 4 (implementation-spec alignment).** Reachable wherever Layer 1 is reachable. Enforced via CI gates that run `dafny_verify` on touched specs and via the `/invariant-coverage-scaffold` skill, which generates a bidirectional invariant-to-test coverage gate so every documented invariant has a covering test and every "Invariant <ID>" comment points at a real doc.

**Layer 5 (spec-intent alignment).** Reachable as a probabilistic check using LLM round-trip informalization. The `/intent-check` skill implements this for invariant-prose / covering-test / code-diff triples, with a false-positive tracker and a configurable kill criterion (default 30% rolling FP rate over a 14-day window with `n ≥ 3` minimum sample). See "Calibration of Layer-5 thresholds" below for the rationale and the env vars that override the defaults. Expect ~96% accuracy on curated benchmarks; real-PR accuracy will only be known once it has run on protected-surface diffs for some time.

**Layer 6 (spec completeness).** Best-effort. The `/spec-adversary` skill probes a module's invariant doc for missing properties, and `/acceptance-oracle-draft` generates mechanically-verifiable user-perspective scenarios as an empirical complement to invariant coverage. No theorem proves spec completeness; both are iterative practices, not gates.

## Calibration of Layer-5 thresholds

The Layer 5 kill criterion as currently shipped (30% rolling FP rate, 14-day window, `n ≥ 3` minimum sample size) is **founder intuition, not labelled-pilot data**. There is no calibration trace of the form "threshold set by N-day pilot, M human verdicts, trip line at distribution elbow" backing it. The numbers were chosen so that (a) Layer 5 is taken offline before it gates more than ~1 in 3 commits incorrectly, (b) the window is long enough to absorb a single bad day without tripping, and (c) the minimum sample blocks single-row noise. None of those bounds are empirically validated yet.

The implementation surfaces three environment variables so each adopter can override the defaults once they have their own labelled trace:

- `CROSSCHECK_FP_TRIPPED_THRESHOLD` (default `0.30`) — kill threshold.
- `CROSSCHECK_FP_AT_RISK_THRESHOLD` (default `0.20`) — escalation threshold.
- `CROSSCHECK_FP_WINDOW_DAYS` (default `14`) — rolling window length.

The same env vars are read by `/intent-check` (Step 0 pre-check) and `/assurance-status` (Step 2.4 dashboard) so the FP rate shown to a contributor is computed identically across the two skills. The reference squad workflows in `crosscheck/docs/examples/workflows/` use the same values; tune all three together when calibrating. The Noisy-but-Valid framework (Feng et al., 2026) is one published protocol for deriving such thresholds from a labelled pilot.

## What this hierarchy is not good for

Modeled on Newcombe et al.'s *"What Formal Specification Is Not Good For"* (CACM 2015) — explicit methodological scope-limit, distinct from the README's *Known limitations* section which covers technical Dafny constraints.

- **Modules without deterministic algebraic semantics** (heavy framework callbacks, side effects, framework conventions over types) — Layer 1 doesn't reach. Route to Layers 2–5; Layer 1 verification will not be useful here.
- **Modules without provable properties** — the obstacle is spec design, not tooling. `/spec-iterate` and `/intent-check` apply, but the bottleneck is human articulation of intent.
- **Modules where input generation is intractable** — DRT may not apply because no random sampling strategy exists. Lean on PBT with hand-curated strategies plus invariant docs.
- **Sustained emergent performance degradation, networked failure modes under partition, security in adversarial settings** — out of scope for the hierarchy entirely. Performance regressions need profiling and tracing; security needs threat modeling and adversarial review; networked behavior under partition needs TLA+-style behavioral analysis at Layer 4 (pending ADR — see Phase 3c).

## Supporting Workflow Elements

Two additional practices sit alongside the hierarchy rather than within it:

**External Acceptance Oracle.** Deterministic verification steps written from the perspective of a user, exercising the feature to determine if the user story is satisfied. These exist outside the repository so the coding agent cannot write code to pass them, and are written before development starts to force upfront intent specification. Requirements must be mechanically verifiable — subjective criteria like "the page feels responsive" must be quantified. The oracle provides empirical, user-perspective assurance that the service works, complementing the exhaustive assurance of formal verification.

**Test Theatre Detection.** Critical review of automated tests to eliminate tests that look productive but verify nothing meaningful. This runs after the implementation has stabilized — not as a sequential pipeline step during development, but as a quality gate once there is confidence the code has stopped changing.

## References

- Axiom. AXLE - Axiom Lean Engine. https://axle.axiommath.ai/
- Axiom. https://axiommath.ai/
- Harmonic. Aristotle API. https://aristotle.harmonic.fun/
- Harmonic. "Aristotle: IMO-level Automated Theorem Proving." arXiv, March 2026. https://arxiv.org/html/2510.01346v1
- Midspiral. https://midspiral.com/
- Midspiral. "lemmafit: Make agents prove that their code is correct." GitHub. https://github.com/midspiral/lemmafit
- Midspiral. "claimcheck: Narrowing the Gap between Proof and Intent." https://midspiral.com/blog/claimcheck-narrowing-the-gap-between-proof-and-intent/
- Midspiral. "From Intent to Proof: Dafny Verification for Web Apps." https://midspiral.com/blog/from-intent-to-proof-dafny-verification-for-web-apps/
- Kleppmann, M. "Prediction: AI will make formal verification go mainstream." December 2025. https://martin.kleppmann.com/2025/12/08/ai-formal-verification.html
- de Moura, L. "When AI Writes the World's Software, Who Verifies It?" February 2026. https://leodemoura.github.io/blog/2026/02/28/when-ai-writes-the-worlds-software.html
