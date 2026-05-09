# ADR-0001: Behavioral specs (TLA+ / P / Alloy) at Layer 4

**Status:** Accepted (decision) / Implementation deferred (Phase 3c does not ship code; see "Deferred work").
**Date:** 2026-05-08
**Supersedes:** The "Layer 1b" proposal in earlier drafts of `crosscheck/docs/research/crosscheck-tla-vgd-addendum.md` (since corrected).

## Context

The May-2026 review identified that Crosscheck has no analogue for *behavioral* specification — verifying safety and liveness properties of an executable model over its reachable state space. Layers 1–3 verify that code implements a spec; Layers 4–6 verify spec quality, intent, and completeness. None of the layers as currently defined check whether a spec, considered as a behavioural model, is internally consistent over all reachable states.

The TLA+/P/Alloy tradition addresses exactly this: Newcombe et al. (CACM 2015) report a 35-step DynamoDB replication bug that "had passed unnoticed through extensive design reviews, code reviews, and testing"; Brooker (2022) traces EBS control-plane bursts after network partitions to design-level state-space gaps. Lamport (CACM 2015) frames specifications as blueprints — durable thinking artefacts that surface imprecision before code is written. The applicability criterion is rule-density / state-explosion, not concurrency or distribution: Cedar (stateless, single-node, rule-dense) is one data point; multi-step workflows with branches, role-hierarchy permission systems, billing engines, configuration-merge logic, and invariant-rich data models are others.

An earlier draft of the addendum proposed a "Layer 1b" peer to Dafny at Layer 1. This was rejected on review:

- Layer 1 verifies *deployed code* (Dafny ingests source, proves properties of the compiled artefact). TLA+/P/Alloy do not ingest deployed code; they verify *the spec*. Placing them at Layer 1 conflates deployed-code verification with spec verification.
- The 1a/1b split would force a renumbering across every doc, agent, skill, and roadmap entry that references "Layer 2 = compilation correctness" etc. Renumbering cost is high and the value is low — the layer ladder is well-known to current users.

Two alternatives remained: Layer 4 enrichment, or Layer 6 augmentation. This ADR documents the decision.

## Decision

**TLA+/P/Alloy enter as Layer 4 enrichment.** Layer 4 is the formal upgrade path for `docs/invariants/*.md` (which is currently prose + property tests). Executable behavioral models are the formal form of the same artefact. Specifically:

- A module's behavioral spec lives at `docs/invariants/<module>.tla` (or `.p` / `.als`) alongside its prose invariant doc.
- Model-checking the spec produces a deterministic accept/reject signal (TLC, P checker, or Alloy Analyzer).
- The skill that scaffolds these specs — `/behavioral-spec-init` or similar, deferred to a later phase — emits the TLA+/P/Alloy file plus a small documentation block linking it to the prose invariant doc.

### Layer 4 redefinition

The existing Layer 4 description (`assurance-hierarchy.md:45-49`) says:

> Layer 4: Implementation–Specification Alignment. Layers 1–3 prove that code satisfies a specification. This layer asks: does the specification actually describe the implementation's behavior?

This wording is *too narrow* once executable behavioral specs are included. The question "does the spec describe the implementation's behavior?" is one alignment direction; "does the spec, considered as a behavioural model, satisfy its own invariants over reachable states?" is a second deterministic question that lives at the same layer. Both are spec-side; both produce accept/reject signals; both are owned by Hellebuyck.

**Recommended redefinition (to be applied in `assurance-hierarchy.md` when the implementing skill ships):**

> Layer 4: Specification Soundness. Two deterministic questions about the specification: (a) does the specification align with the implementation's actual behaviour (prose invariants + property tests + Dafny verification with `/check-regressions` and `/invariant-coverage-scaffold`)? (b) does the specification, considered as an executable behavioural model, satisfy its own invariants over all reachable states (TLA+/P/Alloy model checking)? Both are accept/reject; both are spec-side. The layer is *broader than the original definition* and includes the spec's internal consistency, not only its alignment with code.

The redefinition is **flat** — no 4a/4b sublayering. Sublayering creates the same renumbering problem the Layer 1b proposal had at one layer down. Broadening the definition keeps the ladder stable while accommodating the new check.

### Routing heuristic (per-module applicability)

`/assurance-init` and `/assurance-layer-audit` (extended in Phase 3d) detect whether a module is a behavioral-spec candidate using the standard TLA+ scope criterion. A module is a candidate if it contains *any* of:

- A state machine (explicit transitions between named states with guard conditions)
- A workflow with branching and rollback (multi-step process where steps can fail and unwind)
- A rule-interaction surface where multiple rules combine non-trivially (auth role hierarchies, pricing rule stacks, configuration-merge logic)
- An invariant-rich data model (semantic invariants over the data shape that exceed FK / multiplicity constraints)
- A concurrency or distribution model (the original TLA+ niche; now one of many)

Modules that are pure transformations from input to output are *not* behavioral-spec candidates — they fit Layer 1 (Dafny) instead. Many real modules need both engines.

Crucially: **the routing criterion is rule-density / state-explosion, not "non-distributed code generally."** The earlier addendum overreached on this point (since corrected). Cedar evidences DRT-as-technique generalising; the behavioral-methods case rests on the AWS / TLA+ literature, where the operative criterion has always been state-space explosion that exceeds human inspection.

### Tooling stance: orchestrate, not embed

Crosscheck orchestrates the chosen model checker (TLC, P, Alloy Analyzer) by writing the spec file and invoking the user's local toolchain via documented commands — same Docker pattern as Dafny if desired, but the user provides the model checker installation. Embedding three additional toolchains (each with JVM or other runtime dependencies) is rejected for the first iteration on the grounds that the value/friction trade favours orchestration; embedding can be revisited if a tangible need emerges.

This preserves the Dafny pattern (engine in Docker, results back to skill) as the single embedded engine, with TLA+/P/Alloy as orchestrated peers.

## Consequences

- **Layer 4's role broadens.** The `assurance-hierarchy.md` Layer 4 description must be updated when the implementing skill ships. The redefinition above is the recommended wording.
- **Layer 6 (`/spec-adversary`) gains a deterministic peer.** `/spec-adversary` is heuristic ("propose up to 3 candidate invariants the spec is failing to document"); model checking is exhaustive over reachable states. They are complementary: `/spec-adversary` proposes properties that might be missing; model checking confirms (or finds counterexamples to) the properties already named.
- **`/assurance-init` Step 6 grows a behavioral-spec scaffold path.** Detected behavioral-spec candidates get a `docs/invariants/<module>.tla` (or `.p` / `.als`) skeleton in addition to the prose invariant doc.
- **The protected-surface partition extends.** TLA+/P/Alloy specs are Class B governance artefacts (same partition as `docs/invariants/*.md`).
- **Hellebuyck owns the new artefact.** Layer 4 is already on the spec side of the impl/spec seam.

## Deferred work

This ADR documents the *decision*; it does not ship code. Implementation phases follow:

1. **Choose the first model checker to support.** TLA+ has the largest user base (Newcombe et al. case at AWS). P has better tooling for asynchronous protocols. Alloy is lighter for invariant exploration. Recommendation: support TLA+ first (broadest applicability), with P and Alloy as later additions if user demand surfaces. This decision can be made when the implementing skill is scoped.
2. **Build `/behavioral-spec-init` (or similar) skill.** Scaffold a TLA+/P/Alloy file from the prose invariant doc + module signature. Mirror the Phase 3b Lean-pipeline pattern: scaffold first, leave proof obligations as TODOs, gate on parser success not on model-check success. Detection logic uses the routing heuristic above.
3. **Update `assurance-hierarchy.md`** with the redefinition wording when the implementing skill ships.
4. **Update `/spec-adversary` and `/check-regressions`** to be aware of behavioral specs alongside prose invariants (low-priority — the existing skills don't need to know unless they read the new files).

## Alternatives considered

### Alternative A: Layer 1b peer to Dafny

Rejected. Reasoning above; documented for historical context. The key argument: Layer 1 verifies *deployed code*; TLA+/P/Alloy verify *the spec*. Different verification target, different layer.

### Alternative B: Layer 6 augmentation

Rejected. Layer 6 is best-effort and heuristic; model checking is deterministic and exhaustive over reachable states. Misclassifying a deterministic check as best-effort would weaken the layer's operational meaning.

### Alternative C: New "Layer 4.5" or sublayering (4a / 4b)

Rejected. Sublayering creates renumbering cost (every reference to Layer 5 / Layer 6 in skills, docs, agents, roadmaps would propagate). Broadening Layer 4's definition achieves the same conceptual placement without churn.

### Alternative D: Don't add behavioral specs at all

Rejected. The threats-to-validity item in `crosscheck-review-may-2026.md` §5.6 (recast in Phase 3a-i.2 to "functional vs behavioral specification") names behavioral methods as a real gap with documented evidence (Newcombe et al., Brooker, Lamport). Leaving the gap unaddressed is acceptable only if the framework is honest about it; that honesty already shipped in 3a-iii.2 ("What this hierarchy is not good for"), but the longer-term position is to fill the gap, not perpetuate it.

## References

- `/Users/harry.nicholls/repos/formal-verify/crosscheck/docs/research/crosscheck-review-may-2026.md` — May synthesis (§5.6 recast in Phase 3a-i.2).
- `/Users/harry.nicholls/repos/formal-verify/crosscheck/docs/research/crosscheck-tla-vgd-addendum.md` — addendum (3a-i.1 corrections expunged the Layer 1b proposal).
- `/Users/harry.nicholls/repos/formal-verify/crosscheck/docs/research/assurance-hierarchy.md` — current Layer 4 definition (`:45-49`).
- `/Users/harry.nicholls/repos/formal-verify/crosscheck/docs/research/literature-review.md` — Newcombe / Brooker / Lamport entries (added in Phase 3a-iii.5).
- Newcombe, C., et al. "How Amazon Web Services Uses Formal Methods." *CACM* 58(4), 66–73 (2015).
- Brooker, M. "Getting into formal specification." Marc's Blog, 2022. https://brooker.co.za/blog/2022/07/29/getting-into-tla.html
- Lamport, L. "Who Builds a House without Drawing Blueprints?" *CACM* 58(4), 38–41 (2015).
