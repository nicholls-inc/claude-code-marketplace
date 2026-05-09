# Crosscheck Addendum: TLA+, VGD, and Blueprints

*Comparative report against four new sources. Builds on `crosscheck/docs/research/crosscheck-review-may-2026.md`. Distinguishes verified claims from elicited assumptions.*

---

## Reading of intent

The synthesis doc already concluded that Crosscheck's largest gap is empirical layer-attribution and that Dafny Layer 1 reaches only the pure-functional-core fraction (~22-27%) of typical codebases. All four new sources are about a different stratum of formal methods than Dafny: TLA+ and Lean operating on *executable behavioral models* rather than on *code-level functional correctness*. The cleanest split is *functional* vs *behavioral* specification. Dafny answers "given these inputs to this function, what does it return, and does it satisfy these pre/post-conditions?" TLA+, P, and Alloy answer "across all possible orderings and interactions of these operations, what states can the system reach, and which of those states violate my safety or liveness properties?" Both classes of tool are valuable for the same systems — they catch different bug classes. The Cedar paper is the only one of the four sources that ties model-level proofs to code-level confidence via a third mechanism (differential random testing).

So the question this report tries to answer is narrower than "are these good methodologies": it is "do these expose a missing layer or axis in Crosscheck, and what would the targeted addition look like?" The reading throughout is that they expose two: a behavioral-specification stratum that Crosscheck currently has no analogue for, and a model-to-code bridge mechanism (DRT) that Crosscheck's documentation explicitly leaves as an open problem ("Verified in Dafny … embedding correctness is your responsibility").

---

## What each source actually says

**Newcombe et al. 2014 (AWS / TLA+ paper).** Verified from text: TLA+ is used at AWS to specify system *designs and behaviors*, not to verify code. The paper documents 10 systems, six bugs in the public table, including a DynamoDB replication bug requiring a 35-step trace that "had passed unnoticed through extensive design reviews, code reviews, and testing." Engineers learn TLA+ in 2-3 weeks. Although the paper foregrounds distributed systems, it is explicit that TLA+'s reach is broader: "TLA+ is an excellent tool for data modeling, e.g. designing the schema for a relational or 'No SQL' database … with semantic invariants over the data that were much richer than standard multiplicity constraints and foreign key constraints." The paper is unusually honest about scope: TLA+ does not address "sustained emergent performance degradation" (timeout-cascade type failures), and the answer to "How do we know that the executable code correctly implements the verified design?" is explicitly "we don't." The framing line that maps directly onto Crosscheck's vocabulary: TLA+ exists to eliminate "plausible hand-waving" in design documents — which is the *plausible vs correct* distinction at the behavioral level rather than the code level.

**Brooker 2022 (adoption blog).** Verified from text: Brooker's path was Alloy → Spin → TLA+ over several years driven by EBS control-plane bugs that came in bursts after network partitions. The piece is primarily about adoption mechanics, not technique. The operative formula is "hubris (software can be correct), humility (I can't write correct software), and laziness (I don't want to fix this again)" — engineers without all three don't take to formal specification. Brooker is explicit that TLA+'s applicability is not limited to distributed systems: "Tricky business logic (like the volume state merge I mentioned) can definitely benefit." He now reaches for TLA+ "every couple months" and notes the field has expanded: P language, Shuttle, Dafny, Kani, and S3's lightweight formal methods work all coexist. The implicit thesis: no single tool covers the assurance space; the right tool depends on the problem shape.

**Lamport 2015 (CACM "Who Builds a House Without Drawing Blueprints?").** I could not retrieve the full article (CACM returned 403), but verified the central claims from a high-quality excerpt and the bibliographic record. The argument: a specification is a blueprint; the value of blueprints is not that buildings can be automatically generated from them, but that "we think in order to understand what we are doing" and "writing is nature's way of letting you know how sloppy your thinking is." The article was written by Lamport as a framing companion to the Newcombe et al. paper in the same April 2015 CACM issue. The substantive philosophical claim — *thinking precedes coding, specifications enable thinking* — is the design-first stance that TLA+, P, and Alloy operationalise.

**Disselkoen et al. 2024 (Cedar / VGD, arXiv 2407.01688).** Verified from text and Amazon Science blog: AWS Cedar uses *Verification-Guided Development* (VGD) — three activities. (1) Executable model in Lean (the team migrated from Dafny to Lean during development), with mechanical proofs of seven properties including type soundness. (2) Differential Random Testing: generate millions of inputs, run both the Lean model and the Rust production code, assert outputs match. (3) Property-Based Testing for code paths the model doesn't cover. Reported results: 4 bugs found via proofs, 21 via DRT/PBT. Proof-to-code ratio 3.4:1. Lean model executes in 6μs vs Rust's 10μs — the model is fast enough to be a runtime oracle.

*Notable for Crosscheck's positioning, with two distinct claims that should not be conflated*:

- **DRT-as-technique generalises.** The 21 bugs DRT/PBT found break down ~15/21 as *general implementation bugs* (parser didn't unescape raw strings; pretty-print AST crash; comments dropped on records; Rust IP-address dependency bug; `ipaddr` vs `IPAddr` naming mismatch; validator didn't typecheck `has` expressions; etc.) and ~6/21 as *authorization-specific rule-interaction bugs*. Most of the catch is in the same bug classes any non-trivial codebase has — parsing, error handling, third-party dependency drift, naming inconsistencies. This is strong evidence that DRT applies far beyond authorization.
- **VGD-as-methodology Amazon does not generalise.** The paper scopes VGD to TCB / safety-critical systems and lists four prerequisites: (1) domain amenable to formal modeling (deterministic, algebraic semantics); (2) properties provable about the model; (3) ability to generate comprehensive random test inputs; (4) resources for dual development. The paper's only generalisation gesture is Related Work's conditional "Verification-guided development *could be used* to produce a dependability case…" — explicitly conditional, not a claim of broad applicability.

Earlier framings that read Cedar's stateless single-node nature as the "existence proof" for behavioral methods broadly are an overreach. Cedar evidences DRT-as-technique generalising; the behavioral-methods case rests on the AWS/TLA+ literature (Newcombe et al., Brooker, Lamport), not on Cedar.

---

## Comparison to Crosscheck

The four sources collectively define a verification stack that Crosscheck partly maps onto and partly does not.

**Where Crosscheck and the AWS/Lamport tradition agree, narratively.** The "plausible vs correct" framing at the top of Crosscheck's commitment list is exactly Lamport's "plausible hand-waving" objection. The spec-first orientation, the invariant docs, `/acceptance-oracle-draft`, and the protected-surface partition all sit in the Lamport-blueprint tradition philosophically. Both traditions argue that the durable artefact is the specification, not the code. The synthesis already establishes this alignment via Phoenix.

**Where Crosscheck does not have an analogue.** Crosscheck has no behavioral-specification stratum. Its six layers are: Dafny (code-level functional correctness), tests, lints, invariant docs (prose + property tests), `/intent-check` (back-translation), `/spec-adversary` (prompt-driven probing). None of these checks an *executable behavioral model* against *safety and liveness properties* across all reachable states of a system. This is the bug class TLA+ catches at AWS, evidenced by the AWS / Newcombe et al. literature (35-step DynamoDB replication trace, EBS control-plane bursts after network partitions, etc.). The applicability criterion is the standard TLA+ scope criterion: any system where behavior emerges from interactions of rules a human can't enumerate by inspection — multi-step workflows with branches and rollback (subscription lifecycles, checkout flows with stacked discounts, document review state machines), permission and access systems with role hierarchies and overrides, pricing or billing engines, configuration systems that combine multiple sources, schedulers and retry/backoff logic even single-threaded, data schemas with semantic invariants richer than FK and multiplicity constraints, and UI state machines with non-trivial transitions. The criterion is *rule-density / state-explosion*, not "non-distributed code generally." Most CRUD endpoints don't cross this bar; complex billing engines, auth systems with role hierarchies, and workflow state machines do. The synthesis flagged Layer 1's reach as ~22-27% of typical codebases. For the remaining ~75% of code, Crosscheck's coverage above Layer 1 is *prose-and-tests*, not formal — and prose-and-tests catches a different bug class than executable-model verification does, in the rule-dense slice that meets the criterion.

**Where the Cedar paper directly contradicts a documented Crosscheck position.** The synthesis records Crosscheck's deliberate decision to *defer* the Lean backend from "Abductive Vibe Coding" — the documented reason was that prompt-driven `/rationale` was preferred. Cedar shows a production deployment of an executable Lean model used as both a proof artefact *and* a differential-testing oracle. This doesn't make Crosscheck's deferral wrong (Cedar is one authorization language; Crosscheck is generic), but it removes one rebuttal: "executable model + DRT" is not theoretical, it has shipped, and it found 25 bugs in a real product.

**Where the Brooker blog illuminates Crosscheck's adoption surface.** Brooker's hubris/humility/laziness criterion is a useful diagnostic for Crosscheck's target user. A team without all three will not maintain protected surfaces, will not respond to FP-rate kill criteria, and will treat invariant docs as overhead. The synthesis already names governance-decay as a Category-B threat to validity (item 4); Brooker's framing makes the failure mode concrete. Crosscheck's onboarding does not currently say "you need a motivating example before this is worth installing" the way the AWS/TLA+ adoption story uniformly does.

---

## The genuinely new ideas

Three additions, in descending order of how cleanly they slot into Crosscheck's existing architecture.

**1. Differential Random Testing as the model-to-code bridge.** Cedar shows the empirical mechanism that Crosscheck's docs currently leave as user responsibility. *Verified*: the Cedar paper reports DRT/PBT found 21 bugs that proofs alone missed; the bug taxonomy is ~15/21 general implementation bugs (parsing, dependencies, error handling, naming) and ~6/21 authorization-specific. This is strong evidence that DRT-as-technique generalises beyond Cedar's domain.

The DRT input space — for Crosscheck's purposes — is broader than the Cedar pattern alone:

- (a) Hand or AI-written production code with no formal verification (Cedar's exact pattern: Lean model vs Rust);
- (b) Systems composing Dafny-verified kernels with non-verified glue (per-kernel proofs don't cover the composition);
- (c) Chains of independently-verified kernels (end-to-end semantics not proven by per-kernel proofs);
- (d) Cross-extract validation: Python-extract vs Go-extract from the same Dafny (validates the extraction backends, addresses synthesis item B2's reach-ceiling concern);
- (e) Per-method partial verification: a function with verified preconditions but unverified postconditions (DRT exercises only the unverified surface).

Excluded: fully Dafny-verified slices (DRT redundant; the slice already has compile-time correctness). It does not require a new verifier, only a harness plus an executable model. The model is non-trivial: Lean is the natural choice, and producing a usable Lean model is itself a multi-step pipeline (informal spec → Lean spec stub → Lean impl → correspondence review), well-precedented by GitHub Next's *Lean Squad* (https://github.com/githubnext/agentics/blob/main/docs/lean-squad.md), which decomposes the same problem into 11 phase-weighted tasks. The addendum recommends lifting Lean Squad's load-bearing tasks (informal spec extraction, formal spec writing, implementation extraction, correspondence review, runnable correspondence tests) rather than a single fuzzy `/lean-model` skill.

**2. Behavioral-specification model checking as a Layer 4 enrichment.** TLA+, P, and Alloy verify behaviors that Dafny cannot reach. The relevant criterion is rule-density / state-explosion: any module where the reachable-state space is too large to enumerate by hand. That covers subscription state machines, permission systems with role hierarchies, multi-step checkout with rollback, pricing rule interactions, configuration merge logic, and rich data-model invariants.

The fix is not to add TLA+ to the default workflow — that would violate the friction-vs-value calibration the synthesis established — and not to introduce a "Layer 1b" peer to Dafny. TLA+/P/Alloy verify *the spec*, not deployed code; on Crosscheck's impl/spec seam they sit on the spec side. The natural home is **Layer 4 enrichment**: the formal upgrade path for `docs/invariants/*.md`, which is currently prose + property tests. An executable behavioral model is the formal form of the same artefact.

This avoids the renumbering churn a Layer 1a/1b split would cause, and matches the existing Hellebuyck (Layer 4-6) ownership of governance and spec artefacts. A separate ADR (referenced from the phase roadmap) resolves the question of whether the existing Layer 4 definition broadens to encompass executable behavioral models or whether 4a/4b sublayers are needed; the recommendation is broaden the definition, keep the layer flat. `/assurance-init` and `/assurance-layer-audit` would detect rule-dense modules and route them to the behavioral-spec scaffold; pure transformations route to Dafny as before; many modules need both.

**3. The "blueprint" reframing of `/acceptance-oracle-draft`.** Lamport's blueprint metaphor is more rhetorically powerful than "acceptance oracle" and aligns with terminology a software-architecture-literate audience already recognises. The synthesis recommended (B5) re-centring the default workflow on `/acceptance-oracle-draft`. Lamport gives the language to do that: *acceptance oracles are blueprints in Lamport's sense — they are the durable artefact the implementation must conform to, and they exist to force precise thinking before code is written*. This is documentary and doesn't require code changes. *Verified*: the Lamport piece makes this argument explicitly.

---

## Recommendations for Crosscheck

The synthesis already enumerated four tiers of work. These are additions and amendments to that list, anchored to the new sources.

**Tier-1 additions** (documentation, low effort).

The `/acceptance-oracle-draft` re-centring (synthesis B5) should cite Lamport's blueprint argument explicitly and use the term "blueprint" or "design specification" alongside "acceptance oracle." This raises the legibility of the skill to anyone with an architecture background.

The "Honest Map" at the top of the README (synthesis Tier 1) should include a row labelled *behavioral / state-machine specification* with the explicit note that Crosscheck currently does not address this stratum, and that teams whose modules contain non-trivial state machines, workflows, or interacting rule systems should pair Crosscheck with TLA+, P, or Alloy. The AWS paper's "What Formal Specification Is Not Good For" section is the model for this honesty; Crosscheck's equivalent would be a "What Crosscheck Is Not Good For" section.

The Brooker hubris/humility/laziness diagnostic should appear in the onboarding doc as an adoption-readiness check. Engineers without the right disposition won't maintain Crosscheck's governance scaffolding regardless of how good the engine is.

**Tier-2 additions** (implementation, medium effort).

Add a Differential Random Testing skill (`/diff-test` or similar) for modules where Dafny code and production-language code coexist. Generate inputs via property-test strategies, run both implementations, assert equivalence, fail loudly on divergence. This converts synthesis item B1 from a documentation hedge into an actual mechanism. It uses no new dependencies — Crosscheck already runs Dafny in a Docker sandbox, and property-test frameworks exist in every supported language.

Add a `/behavioral-spec-init` skill (or `/design-spec-init`) that, for modules containing a state machine, workflow, or rule-interaction surface, scaffolds a TLA+, P, or Alloy specification from the prose invariant doc. The skill should not attempt to *verify* the spec; that belongs to the model-checker tooling outside Crosscheck. It should produce an editable starting point and route the user to the appropriate model checker. This is the minimum viable behavioral-stratum integration.

**Tier-3 additions** (architectural, higher effort).

Enrich Layer 4 with executable behavioral specs (TLA+, P, or Alloy). Layer 4 currently houses prose invariant docs + property tests; the enrichment is the formal upgrade path. This requires updating `/assurance-init` to detect rule-dense modules (state machines, workflows with branches, interacting rule sets) and route them to the behavioral-spec scaffold path in addition to the existing prose-invariant path. It also requires choosing whether Crosscheck embeds model checkers (heavyweight, multi-language) or orchestrates them (lighter — preserves the Dafny-Docker pattern as one engine among several). Recommend orchestration for the first iteration; embed only if a tangible need emerges. The 1a/1b split the prior draft proposed at Layer 1 is rejected in favour of this Layer 4 placement; an ADR captures the layer-definition broadening required.

Add Lean as a peer Layer 1 engine alongside Dafny, with a different role. Dafny is verify-and-extract: code is generated from the spec. Lean is the executable-model + DRT-oracle role: production code is hand- or AI-written (or extracted from Dafny for partial-verification cases), and a Lean model serves as the oracle for differential random testing. The Cedar trajectory (Dafny → Lean), the Murphy et al. paper Crosscheck cites (Lean-backed claim trees), and Lean's growing position in formal verification all support this. The two engines are complementary, not redundant: Dafny gives compile-time correctness on the pure-function slice; Lean + DRT gives sample-based correctness on the much larger non-pure slice. Keeping both means Crosscheck covers the verify-and-extract niche where Dafny is the right tool *and* the executable-model niche where Lean is the right tool.

**Tier-4 amendment.**

The synthesis Tier-4 item — instrument the plugin to publish which layer caught what — should be expanded to *also* publish which bug class. AWS publishes a six-row table of "system / component / line count / benefit"; Cedar publishes "4 bugs from proofs, 21 from DRT/PBT." This is the genre of empirical artefact that converts an assurance framework from theory to evidence. Crosscheck's equivalent table would be the single largest credibility-enhancing publication it could ship.

---

## Elicited assumptions

Things this report and the underlying synthesis treat as true without textual proof; flagged so they can be challenged.

**Assumption 1 (recast).** Crosscheck's target user base contains *some* modules where the modal bug is *behavioral* — emergent from rule interactions, state transitions, or workflow branching — rather than *functional* (single-input/single-output transformation). This is a weaker assumption than "users are building distributed systems," but it is also weaker than the prior draft of this addendum claimed. Cedar (a rule-dense authorization engine) is *not* an existence proof for behavioral methods applying to most application code; it is one rule-dense data point. The standard TLA+ scope criterion — rule-density / state-explosion — applies to a slice of typical codebases (auth systems, billing engines, complex workflow state machines, configuration merge logic), not to most CRUD endpoints or pure transformations. Plausibly 15-25% of typical codebases, distinct from Dafny's 22-27% functional-core slice. *Test*: audit a sample of modules where users have run `/intent-check` and classify each as "pure functional transformation," "rule-dense / state-explosion," or "neither." Only the second category supports the Layer 4 enrichment case.

**Assumption 2.** Crosscheck's user base has the discipline Brooker's framework requires. The synthesis already lists this as a threat to validity. The Brooker addition makes it specific: without hubris/humility/laziness present in the team, the governance scaffolding decays regardless of tool quality. *Test*: the field-report Django billing case is one data point; more like it would falsify or confirm.

**Assumption 3.** The `/intent-check` round-trip is a sufficient prose-to-code bridge. The Cedar DRT mechanism suggests it is not — when you can run two implementations on millions of inputs and compare, you find 21 bugs that the four-bug proof effort missed. Crosscheck's implicit position is that the back-translation comparison is structurally adversarial enough. The Cedar evidence pressures that position. *Test*: on any module with both Dafny and production code, run DRT for a week and count divergences `/intent-check` did not surface.

**Assumption 4.** Invariants-as-prose plus property tests is qualitatively equivalent to invariants-as-executable-model. The TLA+ tradition holds that an executable spec is a different kind of artefact — it can be model-checked, used as a documentation source, used as a what-if tool for proposed changes. Crosscheck's invariant docs are none of these. *Test*: pick a Crosscheck module's invariant doc, attempt to translate it to PlusCal in two days, and see whether anything new surfaces.

**Assumption 5.** Crosscheck's adoption model — install plugin, follow defaults — is the right shape. The AWS/Cedar pattern is *organisation-wide process change* with management endorsement and dedicated engineering time. The plugin shape is more lightweight, but if formal-methods-as-process is what actually works at AWS scale, then Crosscheck's plugin shape may be optimising for adoption over impact. *Test*: this requires data Crosscheck does not yet have; Tier-4 publication of layer-attribution data would inform it.

---

## Net assessment

The four sources do not invalidate any conclusion in the synthesis doc. They sharpen three. First, the synthesis's "I/O, concurrency, integration" line was too narrow — the real distinction is *functional* vs *behavioral* specification, and the gap is that Crosscheck has no analogue for the latter. The behavioral-methods case rests on the AWS / TLA+ literature (Newcombe et al., Brooker, Lamport), where the standard scope criterion is rule-density / state-explosion. Cedar evidences DRT-as-technique generalising — *not* behavioral-methods applying broadly to non-distributed code; the addendum's earlier draft conflated those claims. Second, the model-to-code bridge that the synthesis flagged as user-responsibility (B1) has a named, productionised mechanism (DRT) that Cedar shows works in practice — finding 21 bugs in the studied case vs 4 from the proof effort alone, noting that the proofs also shaped the model DRT exercised — with a bug taxonomy (~15/21 general implementation bugs) that supports broad applicability. Third, the adoption-discipline assumption that the synthesis listed as a Category-B threat acquires a specific operative diagnostic from Brooker (hubris/humility/laziness) — useful as diagnostic language only, never as an adoption gate.

The single highest-leverage addition is a Differential Random Testing pipeline for production-code-vs-executable-model equivalence. It is the smallest scope change that addresses the largest documented gap in Crosscheck's own assurance story (B1: model-to-code chain-of-trust). It does require new architecture: Lean joins as a peer Layer-1 engine, and producing usable Lean models is itself a multi-step pipeline (informal spec → Lean spec stub → Lean impl → correspondence review → DRT) — substantially more involved than a single `/diff-test` skill, but well-precedented by *Lean Squad* (referenced above). The Layer 4 behavioral-specification enrichment (TLA+/P/Alloy) is the second opportunity, narrower in applicability (rule-dense modules only) and requiring its own ADR for the Layer 4 redefinition. VGD-as-overarching-frame is rejected: VGD is one methodology that applies where its four prerequisites are met at the module level, not the umbrella for Crosscheck. The umbrella is layered formal verification + probabilistic + stochastic complements, applied per module.

## References

### Primary sources reviewed for the addendum

Newcombe, C., Rath, T., Zhang, F., Munteanu, B., Brooker, M., Deardeuff, M. (2014). *Use of Formal Methods at Amazon Web Services*. Internal report, 29 September 2014. Published as: Newcombe, C., et al. (2015). "How Amazon Web Services Uses Formal Methods." *Communications of the ACM* 58(4), 66-73. https://cacm.acm.org/magazines/2015/4/184701-how-amazon-web-services-uses-formal-methods/fulltext

Brooker, M. (2022). *Getting into formal specification, and getting my team into it too*. Marc's Blog, 29 July 2022. https://brooker.co.za/blog/2022/07/29/getting-into-tla.html

Lamport, L. (2015). "Who Builds a House without Drawing Blueprints?" *Communications of the ACM* 58(4), 38-41. https://cacm.acm.org/opinion/who-builds-a-house-without-drawing-blueprints/ — DOI: 10.1145/2736348

Disselkoen, C., Eline, A., He, S., Headley, K., Hicks, M., Hietala, K., Kastner, J., Mamat, A., McCutchen, M., Rungta, N., Shah, B., Torlak, E., Wells, A. (2024). *How We Built Cedar: A Verification-Guided Approach*. arXiv:2407.01688 [cs.SE]. Also: FSE Companion '24 (Industry Track), 15-19 July 2024, Porto de Galinhas, Brazil. https://arxiv.org/abs/2407.01688

---

### Supporting AWS / Cedar material referenced

Amazon Science (2024). *How we built Cedar with automated reasoning and differential testing*. Amazon Science blog, 6 April 2024. https://www.amazon.science/blog/how-we-built-cedar-with-automated-reasoning-and-differential-testing — Pre-Lean version of the VGD process; the team migrated to Lean during development.

Brooker, M., Chen, T., Ping, F. (2020). "Millions of Tiny Databases." *NSDI '20 (USENIX)*. https://www.usenix.org/conference/nsdi20/presentation/brooker — Background context for the EBS Physalia work Brooker references in the 2022 blog.

Brooker, M. (2024). *Formal Methods: Just Good Engineering Practice?* Marc's Blog, 17 April 2024. https://brooker.co.za/blog/2024/04/17/formal.html — Companion piece; not cited directly in the addendum but in the same argumentative line.

---

### Foundational TLA+ and formal-methods works named

Lamport, L. *The TLA Home Page*. https://lamport.azurewebsites.net/tla/tla.html

Lamport, L. (2002). *Specifying Systems: The TLA+ Language and Tools for Hardware and Software Engineers*. Addison-Wesley.

Lamport, L. (2006). "Fast Paxos." *Distributed Computing* 19(2), 79-103. http://research.microsoft.com/pubs/64624/tr-2005-112.pdf

Wayne, H. *Learn TLA+*. https://learntla.com/

Zave, P. (2012). "Using lightweight modeling to understand Chord." *ACM SIGCOMM Computer Communication Review* 42(2). http://www.pamelazave.com/chord-ccr.pdf — The Alloy/Chord work that motivated Brooker's and Newcombe's first attempts at formal methods.

McKeeman, W. M. (1998). "Differential testing for software." *Digital Technical Journal* 10(1), 100-107. — Original DRT paper cited by the Cedar team.

de Moura, L., Ullrich, S. (2021). "The Lean 4 theorem prover and programming language." *CADE 28*, 625-635. — Cited by Cedar for their executable-model language choice.

---

### Crosscheck-internal sources referenced from the synthesis

Crosscheck repository, `docs/research/literature-review.md`, `docs/research/assurance-hierarchy.md`, `docs/research/logic-distribution-analysis.md`, `docs/reports/crosscheck-field-report.md`, `docs/reports/abductive-vibe-coding-review`, `docs/reports/vibe-reasoning-paper-review`, `docs/invariants/`, `.claude/agents/`. Privileged in-repo review summarised in `crosscheck/docs/research/crosscheck-review-may-2026.md`.

Graciolli, R., Amin, K. (2026). *Midspiral `claimcheck` development benchmark*. Source for the 96.3% accuracy figure on 108 comparisons across 36 requirement-lemma pairs in 5 domains, 8 deliberately planted bogus lemmas. (Self-disclaimed by authors as "a development benchmark rather than a formal evaluation.")

---

### Cited research grounding for Crosscheck (named in the synthesis)

Ugare, S., Chandra, S. (2026). *Agentic Code Reasoning*. arXiv:2603.01896 — Source for premise-tagged certificate format used in `/rationale`, `/reason`, `/compare-patches`, `/locate-fault`, `/trace-execution`.

Murphy, A., Babikian, A., Chechik, M. (2026). *Abductive Vibe Coding*. arXiv:2601.01199 — Lean-backed claim trees; backend deferred by Crosscheck in favour of prompt-driven `/rationale`. Cedar's Lean migration is directionally relevant to this deferral decision.

Mitchell, B., Shaaban, K. (2025). *Vibe Coding Needs Vibe Reasoning*. arXiv:2511.00202 — TypeScript Type-III sidecar; rejected by Crosscheck as architecturally incoherent with a Dafny backend.

Lahiri, S. K. (2026). *Intent Formalization* (essay). arXiv:2603.17150 — Spectrum framing (tests → code contracts → logical contracts → DSLs); recommended for more prominent citation in Crosscheck's literature review.

Endres, M., et al. (2024). "Can Large Language Models Write Good Property-Based Tests?" *FSE 2024* — Postcondition generation results on Defects4J.

Lahiri, S. K., et al. *FMCAD 2024* spec evaluation work — Soundness/completeness reporting framework for Hoare-triple-encoded specs; the basis for the synthesis's proposed `/spec-eval` skill (C1).

Goldweber, J., et al. (2024). *IronSpec*. *OSDI '24* — Mutation-based deterministic discrimination signals for Dafny specs.

*MutDafny*. arXiv:2511.15403 — 5-operator mutation set proposed for the optional `--mutate` mode on `/spec-adversary` (synthesis A2).

Ko, S., et al. (2022). *TiCoder*. — Interactive yes/no/undefined disambiguation on candidate tests; +22 to +37 percentage points pass@1 on MBPP with one user query. Basis for the synthesis's proposed interactive disambiguation skill (C2).

Hahn, C., et al. (2022). *nl2spec*. — Fragment-level sub-translation alignment between natural-language prose and formal fragments. Basis for the synthesis's post-mismatch sub-translation alignment proposal (A3).

Swamy, N., Lahiri, S. K., et al. (2026). *SPOTs (Small Proof-Oriented Tests)*. — Deterministic gap detection for Dafny modules; basis for synthesis proposal C4.

Feng, Y., et al. (2026). *Noisy-but-Valid framework*. arXiv:2601.20913 — Published protocol for deriving threshold values from labelled-pilot data; relevant to calibrating the 30% FP kill threshold (synthesis A4).

---

### Methodology and framing references

Fowler, M. *The Deletion Test* — Heuristic invoked in the synthesis for distinguishing durable artefacts from regenerable ones.

Phoenix Project (referenced in synthesis) — Source of the "non-deletable primitives" thesis and the regeneration-speed pace-layer framing.

Intent Stack reference framework. https://intentstack.org — Named as a peer framework in the synthesis open questions.
