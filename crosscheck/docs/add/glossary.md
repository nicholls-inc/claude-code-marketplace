# ADD Glossary and ID Conventions

**Status:** Ratified v1.0
**Purpose:** A ubiquitous language (Evans, DDD) for ADD. Terms here are used consistently across all artifacts in `docs/add/`, in skill SKILL.md files, in commit messages, and in agent-authored specs. If a term shifts meaning, this file is amended first; the cascade then updates dependent artifacts.

## ID conventions

All ADD artifacts carry stable, monotonic identifiers. Identifiers are never reused after retraction.

| Prefix | Artifact kind | Format | Example |
|---|---|---|---|
| `IC` | Intent Claim | `IC<n>` | `IC4` |
| `S` | Architectural Spec section | `S<n>[.<sub>]` | `S2.3` |
| `B` | Behavioral spec invariant | `B<n>` per module | `B1` (within module M2) |
| `F` | Functional spec section | `F<n>[.<sub>]` per module | `F1.2` |
| `I` | Functional invariant | `I<n>` per module | `I3` |
| `T` | Test linkage stub | `T<n>.<m>` | `T1.7` |
| `M` | Module | `M<n>-<slug>` | `M3-billing` |
| `ADR` | Architecture Decision Record | `ADR-<nnn>-<slug>` | `ADR-001-operating-modes` |

Within a module's directory, `B`/`F`/`I`/`T` IDs are scoped to that module. Cross-module references use the qualified form `M3-billing/I3`.

## Linkage declarations

Every spec section, invariant, and test carries `consumes:` and `produces:` declarations in its frontmatter or top-of-file metadata block.

```yaml
---
id: S2.3
status: Ratified
consumes: [IC1, IC4, ADR-002]
produces: [B1, B2, F1.1, F1.2]
---
```

`consumes:` lists IDs of upstream artifacts the section depends on. `produces:` lists IDs of downstream artifacts the section generates or constrains. A spec section with no `consumes` is suspicious (no anchor to intent); one with no `produces` is also suspicious (no downstream effect).

## Status field

Every artifact carries one of the following statuses, recorded at the top:

- **Drafted** — written but not yet attested. Not part of the linkage graph until attested.
- **Attested** — read and confirmed by the responsible human or agent. Now part of the linkage graph.
- **Ratified** — has been pressure-tested through one or more consolidation passes; high confidence.
- **Superseded-by-N** — replaced by another artifact (named). The original is retained, never deleted. The supersession reason is logged in the replacement.
- **Retracted-with-Reason** — abandoned without replacement. Reason is recorded; downstream artifacts that consumed this one enter Drifted state until they amend.

Status transitions are monotonic in one of two senses: forward (Drafted → Attested → Ratified) or terminal (any → Superseded-by-N or Retracted-with-Reason). Attested can transition back to Drafted only through an explicit *Re-drafting* event triggered by upstream change; the prior Attested state is recorded.

## Core terms

**Attestation.** A narrowly-scoped human (or designated agent) confirmation of an artifact, committed to the repo alongside the artifact. Attestations are not "approvals" in a process-overhead sense; they are claims of the form "I have read this and confirm it states what I meant." Attestations are durable artifacts in their own right.

**Auditor agent.** A distinct agent role, separate from authoring agents, that runs consolidation passes. Does not propose changes; produces verdicts on artifact state. See `decisions/ADR-003-auditor-agent.md`.

**Behavioral spec.** A specification of system behavior across reachable states. The TLA+/Alloy/P stratum. Distinct from *functional spec*, which specifies per-operation pre/post conditions.

**Bootstrap mode.** Operating mode where ADD is applied to an existing codebase by recovering intent and retrofitting governance. Distinct from *ADD mode* (clean slate) and *transitional mode* (mixed).

**Cascade.** The downstream re-derivation triggered by an upstream change. A change to `IC4` cascades to architectural sections that consume it, then to behavioral specs that consume those, then to functional specs and tests. The cascade is mechanical at the integrity-check level (which artifacts now need re-attestation) and judgment-mediated at the content level (whether the downstream artifact actually needs to change).

**Consolidation pass.** A scheduled review by the auditor agent of all artifacts in the repo, producing per-artifact verdicts of Settled / Active / Drifted. Distinct from continuous assurance (which is event-driven).

**Diff classification.** Mandatory tag on every spec-changing commit: Propagated discovery / Intent refinement / Drift / Retraction / Status transition. See `decisions/ADR-005-diff-classification.md`.

**Drift.** A spec change that weakens the spec to match what got built, rather than the implementation matching the spec. Not always wrong, but never silent.

**Functional spec.** A specification of per-operation pre- and post-conditions. The Dafny-or-equivalent stratum. Distinct from *behavioral spec*.

**Governance amendment.** A commit that modifies a protected-surface artifact (spec, invariant doc, harness config). Requires explicit human attestation and a written justification.

**Intent claim.** A numbered, ADR-formatted statement of what success looks like or what is out of scope. The atomic unit of Phase 0. `IC` IDs are global to the project.

**Intent refinement.** A spec change driven by improved human understanding of what the system should do. Distinguished from *propagated discovery* (driven by implementation feedback) and *drift* (driven by implementation pressure).

**Linkage graph.** The typed graph of `consumes:` / `produces:` edges across all ADD artifacts. Mechanically traversable. The integrity check (no orphans, no dangling references) is a deterministic gate.

**Negative space.** The explicit out-of-scope list captured in Phase 0. Often where post-hoc spec drift originates if not made explicit.

**Orphan.** An artifact with no `consumes:` or no `produces:` edge to other artifacts in the linkage graph. Mechanically detected. Each orphan is a governance violation until resolved (either by adding the missing link or retracting the artifact).

**Phase boundary.** A point at which the methodology requires human attestation before proceeding. Phase 0 → 1, Phase 1 → 2, Phase 2 → 3 are all attestation boundaries; Phase 4 commits attest at commit time per the diff classification rule.

**Propagated discovery.** A spec change driven by implementation revealing something that should have been known earlier. Healthy and expected, especially in early projects.

**Protected surface.** An artifact whose modification requires human-authored amendment with a governance note. ADD artifacts in `docs/add/` are protected surfaces from the moment they reach Attested status.

**Reasoning trace.** The portion of an artifact that records *why* the decision was made: forces in tension, alternatives considered, rejection rationale. ADRs are the canonical reasoning-trace format. ADD requires reasoning traces at every spec layer, not just architectural decisions.

**Retraction.** A previously-made claim that is being abandoned without replacement. Distinct from supersession (where a replacement exists).

**Supersession.** Replacement of one artifact by another, with the original retained and marked Superseded-by-N. The replacement records the supersession reason.

**Three gates.** The conditions that determine which assurance tools are available against an artifact: *vintage* (maturity), *coverage* (what surrounding artifacts exist), *confidence* (pressure-testing survived). Tool availability falls out of artifact state.

**Transitional mode.** A repo where some modules originated in bootstrap mode and others in ADD mode, with appropriate per-module governance. The common case in practice.

**Ubiquitous language.** Evans' term: a shared vocabulary across humans, agents, specs, tests, and code. This file *is* the ubiquitous language for ADD.

## Anti-terms (deliberately avoided)

- **"Approval"** — too process-loaded; we use *attestation* instead.
- **"Done"** — context-dependent; we use *Attested* or *Ratified* with explicit criteria.
- **"Spec change"** without a classification — never permitted; every spec change is one of the four diff classifications.
- **"Sign off"** — implies hierarchical authority; attestations are claims of personal confirmation, not authority.

## Amendment rule

This glossary is itself a protected surface. Adding a term is a low-friction change. Renaming or redefining a term cascades to every artifact that uses the old term and is treated as a governance amendment.
