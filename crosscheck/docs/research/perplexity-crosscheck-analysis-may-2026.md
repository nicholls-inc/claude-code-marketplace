# Crosscheck, in my own words

Crosscheck is a Claude Code plugin that wraps a multi-layer assurance model around AI-assisted coding, with Dafny-backed formal verification at the bottom, intent/provenance tooling at the top, and governance scaffolding tying them together.
It treats **LLM-generated code as inherently plausible but untrusted**, and responds by inserting machine-checkable gates (formal, semi-formal, and empirical) along the path from informal intent to shipped code.

At a high level, Crosscheck’s design rests on the following commitments:

- **Plausible vs. correct.** The core failure mode of AI-generated code is *silent semantic drift*, not syntax errors, so the system focuses on surfacing violations of intent rather than parser-level issues.
- **A two-orchestrator split.** One orchestrator (Byfuglien) owns *implementation-side assurance* (Dafny verification, unit/property tests, static reasoning), while a second (Hellebuyck) owns *intent/spec/governance* (invariants, adversarial spec review, acceptance oracles, provenance).
- **A 6-layer assurance hierarchy.** Layers 1–3 harden implementation; layers 4–6 harden intent, specifications, and governance, with explicit kill-criteria per layer.
- **Dafny as Layer 1.** Verified core logic is written (or synthesized) in Dafny and compiled to Python/Go, with known trust boundaries (I/O, concurrency, FFI, and generic erasure for Go).
- **Round‑trip informalization.** `/intent-check` uses a two-LLM pipeline—blind back-translation from code+tests to English, then diff-checking against invariant prose—to detect drift between intent, tests, and implementation, with a claimed ~96% accuracy inherited from Midspiral’s `claimcheck` benchmark.
- **Adversarial spec probing.** `/spec-adversary` scans code and invariant docs to propose missing invariants, with human triage and tracker files rather than automatic promotion.
- **Governance scaffolding against drift.** Invariant coverage gates, protected-surface amendments, FP/coverage trackers, and pre-commit/CI hooks are intended to keep specs, tests, and code aligned over time.
- **Semi-formal reasoning with evidence certificates.** `/rationale` builds structured adequacy arguments whose leaves are classified by verification method (formal, behavioral, static, semantic), inspired by semi-formal and abductive assurance-case work.
- **Research-grounded prompts.** The skill catalogue draws explicitly on Agentic Code Reasoning (semi-formal templates), Abductive Vibe Coding (structured rationales and checklists), and Vibe Coding Needs Vibe Reasoning (sidecar verification for long-horizon vibe coding).

**Scope of problems.** Crosscheck is aimed at *verification-aware AI coding* in mainstream languages: it assumes you are already using Claude Code for implementation-level work and want stronger guarantees for a small set of load-bearing modules.
Layer 1 is explicitly scoped to non-I/O, single-threaded, side-effect-free core logic; the governance skills are scoped to repos that are willing to adopt invariant docs, acceptance scenarios, and protected surfaces as first-class artifacts.

**Unsupported or weakly supported claims.**

- The README and docs repeatedly quote a **“~96% accuracy”** figure for `/intent-check`, but this number comes from Midspiral’s `claimcheck` dev benchmark on a different codebase and pipeline; there is no evidence yet that Crosscheck’s portable, all-in-LLM implementation reproduces this accuracy on user repos.
- The 6-layer hierarchy is inspired by assurance hierarchies and the Intent Stack but is not yet backed by empirical calibration (e.g., no data on which layers actually catch bugs or which kill-criteria are realistic in practice).
- End-to-end workflows for combining Dafny Layer 1 with Hellebuyck’s Layer 4–6 skills are sketched but not fully demonstrated on realistic multi-module systems; there is no published case study showing cross-layer interaction over time.

The rest of this report tests these commitments against three lenses: **Microsoft RiSE’s intent-formalization work**, **nl2spec and related NL→spec tooling**, and **Chad Fowler’s Phoenix Architecture for regenerative software**, plus the newer semi-formal reasoning literature.

***

## Per-resource analysis

### Crosscheck docs (README, agents, skills, assurance hierarchy)

**Thesis.** Crosscheck operationalizes a **6-layer assurance hierarchy** inside Claude Code: Dafny verification (L1), unit/property tests and static checks (L2–3), invariants and semi-formal rationales (L4), intent–spec alignment via round-trip informalization and acceptance oracles (L5), and spec completeness/adversarial probing plus governance scaffolding (L6).
Byfuglien orchestrates Layers 1–3 for concrete code; Hellebuyck orchestrates Layers 4–6 for intent, specs, and governance.

**Methods.**

- **Layer 1 (Dafny).**
  - Verify core logic in Dafny, then compile to Python/Go.
  - Explicitly excludes I/O, concurrency, and external libraries; externs and wrappers are treated as trust boundaries.
  - Go generics are handled via type erasure, with the usual caveats.
- **Layers 2–3 (tests & static reasoning).**
  - Conventional unit/property tests, generated and maintained by Claude.
  - Static reasoning skills (e.g., `/rationale`) classify claims by evidence type and try to force the model to cite file/line-level support.
- **Layer 4 (semi-formal rationales).**
  - `/rationale` builds claim trees, tagging leaves as FORMAL/BEHAVIORAL/STATIC/SEMANTIC.
  - Users can request Dafny specs for FORMAL leaves, tests for BEHAVIORAL leaves, and checklists for SEMANTIC leaves.
- **Layer 5 (intent alignment).**
  - `/intent-check` performs blind back-translation from code+tests to English, then diff-checks that against invariant prose.
  - A rolling false-positive tracker and a 30% FP “kill criterion” are used to decide whether this layer should gate commits.
  - `/acceptance-oracle-draft` generates mechanically-checkable acceptance scenarios and a runner stub, with an explicit ban on subjective assertions.
- **Layer 6 (spec completeness & governance).**
  - `/spec-adversary` proposes up to three missing invariants, with evidence and a triage block; accepted proposals are manually promoted via `/protected-surface-amend`.
  - `/invariant-coverage-scaffold` wires a bidirectional invariants↔tests coverage gate as both a pre-commit hook and CI job.
  - `/protected-surface-amend` requires structured governance notes (rationale, roadmap item, authority, diff-plan, coverage impact) for edits to prompts, workflows, and invariants/tests.

**Limitations and assumptions.**

- The **hierarchy is largely normative**, not validated: there is no evidence yet that a 6-layer split is the right granularity or that kill-criteria like a 30% FP rate are achievable outside toy repos.
- **Dafny’s expressiveness and runtime limitations** (no effects, no concurrency, limited library ecosystem) mean only a small fraction of real systems can live in Layer 1; most behavior will sit in Layers 2–5.
- `/intent-check`’s **96% figure is inherited from Midspiral’s `claimcheck` dev benchmark**, not re-measured for Crosscheck’s all-LLM implementation, and no soundness/completeness metrics are reported.
- `/spec-adversary` is explicitly “best-effort”; it does not attempt to measure completeness (in Lahiri’s sense) and instead relies on human reviewers to decide whether proposed invariants matter.
- Governance skills assume a **high-discipline team** willing to maintain `.claude/rules/protected-surfaces.md`, explicit roadmap docs, and invariant coverage gates; this is closer to a research prototype in governance than a drop-in tool for typical teams.

***

### RiSE MSR blog (Agentic Proof-Oriented Programming, A3, Intent Formalization, AutoCLRS, Spotting Specs)

**Overall thesis.** The RiSE work argues that in the age of AI coding, the bottleneck is no longer producing code, but **formalizing and validating intent** so that verification can scale to mainstream development.
The focus shifts from “can we verify code?” to “what are we verifying *against*, and how do we know those specs match user intent?”[1][2]

**Key posts.**

- **Agentic Proof-Oriented Programming (A3).**
  - Demonstrates an agentically generated verifier (`a3-python`) that uses a “kitchen-sink” of verification strategies (barrier certificates, dataflow, symbolic execution) plus an LLM triage layer.[3]
  - Emphasizes *translation validation* and “fighting AI slop with AI slop”: LLMs propose theory and code, but everything is run through adversarial testing and static checks.[3]
  - Highlights that the verifier itself becomes an object of verification and adversarial stress, not a static trusted base.

- **How to Train Your Program Verifier.**
  - Describes using agentic loops over codebases to iteratively build verifiers, guided by concolic oracles and barrier certificates.[3]
  - Stresses that **noise and false positives** are the main user-facing failure modes; quantitative semantics (“distance to safety”) and prioritization are key.

- **Intent Formalization: A Grand Challenge.**
  - Frames **intent formalization** as the central problem: automatically translating informal user intent into checkable formal specs.[2][1]
  - Positions a spectrum from lightweight tests → code contracts → rich postconditions/logical specs → full DSLs and synthesis.
  - Argues that **spec validation is the bottleneck**, not spec generation; metrics like correctness and completeness (Endres et al., Lahiri FMCAD) are needed to evaluate specs without an oracle.[2]

- **Formalizing Data Structures and Algorithms with Agents (AutoCLRS).**
  - Uses agents to formalize textbook algorithms and data structures in F*/Dafny-like settings.
  - Shows that LLMs can produce useful specs, but **proof automation and spec debugging loops** are crucial; users must see where specs fail and why.[4]

- **Spotting Specification Gaps with Small Proof-Oriented Tests.**
  - Proposes “SPOTs”: small proof-oriented tests whose verification outcomes reveal spec gaps even when code passes conventional tests.[5]
  - Emphasizes **mutation and adversarial testing of specs themselves**, not just code, and metrics over soundness/completeness.

**Stated limitations and relevance.**

- RiSE work repeatedly notes that **fully formal methods are too heavy** for most developers; the value-per-friction sweet spot is often tests, postconditions, and lightweight interactive workflows (e.g., TiCoder) rather than deep logical specs for everything.[2]
- The blog explicitly calls for **benchmarks and metrics for specification quality**, especially for verification-aware languages like Dafny and F*.[2]
- Intent formalization is framed as *complementary* to agentic code reasoning: the verifying pipeline is only as good as the specs fed into it.

**Bearing on Crosscheck.**

- Validates Crosscheck’s focus on intent formalization and on Dafny-backed postconditions for core logic.
- Puts pressure on Crosscheck to **measure spec quality** (soundness/completeness) rather than treating invariant docs and Dafny specs as axiomatically correct.
- Suggests that **interactive, test-driven formalization (TiCoder) and SPOT-like spec probes** are empirically effective; Crosscheck’s skills partially mirror these, but are missing key metrics and user-in-the-loop flows.

***

### Microsoft intent-formalization repo, TiCoder, and FMCAD 2024 spec evaluation

**Repo thesis.** The `nl-2-postcond`/intent-formalization repo aggregates artifacts around **LLM-driven intent formalization**: TiCoder for test-driven code generation, LLM-generated postconditions on Defects4J, and formal metrics for specification quality in Dafny and F*.[3]

**TiCoder (Interactive Test-Driven Code Generation).**

- **Workflow.**
  - Given an NL intent and function signature, TiCoder generates candidate code and tests via LLMs, then **interacts with the user** by asking whether proposed tests should pass or fail.[4]
  - User feedback is used to prune and rank code suggestions; tests become a *weak formal specification* shaping the search space.[4]
- **Results.**
  - On MBPP and HumanEval, TiCoder improves pass@1 by ~22–38 percentage points on MBPP and ~25–54 points on HumanEval with 1–5 user queries, compared to pure LLM baselines.[4]
  - It can generate at least one test consistent with user intent within ~1.5–1.7 queries for the vast majority of tasks.[4]
- **Limitations.**
  - Requires user time in the loop; tested primarily on Python; still relies on hidden test suites as the ultimate oracle.

**FSE 2024 postconditions and FMCAD 2024 spec evaluation.**

- **Postconditions on Defects4J (Endres et al.).**
  - LLM-generated postconditions can catch real bugs when evaluated against mutated implementations and test suites, using correctness/completeness metrics.[5]
  - The key is **mutation-based evaluation** of specs to avoid vacuity.

- **Lahiri FMCAD 2024.**
  - Proposes automated **correctness and completeness metrics** for specs in verification-aware languages like Dafny by symbolically testing postconditions with mutated outputs.[6]
  - Correctness: spec must hold for all known input-output tests; completeness: fraction of mutated outputs the spec rejects, approximated via Dafny-encoded Hoare triples and SMT.[6]
  - Shows these metrics correlate well with human labels of {WRONG, WEAK, STRONG} specs, but also catch cases where human labels were overly generous.[6]

**Bearing on Crosscheck.**

- Confirms that **semantically evaluating specs** (postconditions and invariants) is both feasible and useful, especially via mutation and symbolic execution.
- Highlights the **value of interactive user feedback** on tests as a low-friction form of intent formalization; Crosscheck’s `/intent-check` and `/acceptance-oracle-draft` sit in this space but currently lack interactive TiCoder-style loops.
- Provides a concrete template for **spec quality metrics** that Crosscheck’s Layer 5/6 could adopt instead of relying solely on FP rates for `/intent-check`.

***

### nl2spec (NL-to-LTL specs)

**Thesis.** `nl2spec` is a framework and tool for translating unstructured natural language requirements into temporal-logic specifications (LTL/STL) using LLMs, with a focus on **interactive disambiguation via sub-translations**.[7][8]

**Methods.**

- Decomposes an NL requirement into **sub-translations**: fragments of English mapped to sub-formulas (e.g., “do not hold at the same time” → `!(g0 & g1)`).[8]
- Uses few-shot prompting and an LLM backend to propose these sub-translations, with **confidence scores and alternatives** for each.[8]
- Provides a web interface where users can **add, edit, or delete sub-translations**, making corrections at the fragment level rather than editing the entire formula.[8]
- Supports multiple temporal logics (LTL, STL) and model backends via a pluggable architecture.[7]

**Results and limitations.**

- A user study shows that interactive sub-translation makes it easier for experts to converge on correct LTL specs than from-scratch editing, but **human supervision remains essential**.[8]
- The tool does not provide formal metrics for correctness/completeness of the resulting specs; the main emphasis is usability and ambiguity resolution.

**Bearing on Crosscheck.**

- Validates Crosscheck’s instinct that **round-trip and structured natural-language views of specs** are critical for users.
- Suggests that Crosscheck’s `/intent-check` could benefit from **fine-grained sub-intent mapping** (e.g., mapping individual invariant clauses to specific code/test fragments) rather than one big prose block.
- Shows a well-tested pattern for **interactive correction of spec translations**, whereas Crosscheck currently uses `/intent-check` and `/spec-adversary` primarily as one-shot analyses with human review, not interactive spec-edit loops.

***

### TiCoder arXiv paper and TSE 2024 user study

Beyond the repo README, the TiCoder papers deepen the argument that **lightweight, test-based intent formalization** is both usable and effective.[4]

- The arXiv paper formalizes **Interactive Test-Driven Code Generation (ITDCG)**: a workflow where the agent proposes tests and code, and users respond with YES/NO/UNDEFINED labels.[4]
- The user study in TSE 2024 shows that developers can **effectively guide code generation via test judgments**, and that the tests themselves become reusable regression suites.[4]
- Importantly, TiCoder measures both **pass@k@m** (code correctness after m test queries) and **accept@m** (probability of getting at least one user-approved test after m queries), giving a clear picture of the tradeoff between user effort and reliability.

**Bearing on Crosscheck.**

- Strongly supports Crosscheck’s emphasis on **tests as a weak but practical spec language**, especially near Layer 2/3.
- Suggests that Crosscheck should **measure and expose similar interaction metrics** for any test-driven flows it adds (e.g., number of `/intent-check` runs or `/acceptance-oracle-draft` iterations required to catch a regression).

***

### Phoenix Architecture essays (Fowler)

Because we could not reliably fetch full post bodies for each essay, this analysis relies on the homepage listing plus secondary commentary (e.g., daily.dev, LinkedIn, and Jason Goecke’s “The Ashes Have Intent”).[9]
The core Phoenix thesis is nevertheless clear and consistent across these sources.

**Overall thesis.** In the AI-coding era, **code is disposable** and should be designed to be burned and regenerated, while **intent, evaluations, architectural shape, and provenance are the durable assets**.
The architecture’s job is to make regeneration safe: define what can be deleted, what must persist, and how trust gradients control regeneration.

**Tier-1 essays (read/triaged via secondary sources).**

- **Regenerative Software.**
  - Introduces “Phoenix Architecture”: systems designed so components can be destroyed and rebuilt without losing identity.[9]
  - The most durable systems will be built from code “meant to die”; the durable artifacts are specifications, evaluations, and architectural boundaries.

- **Code Was Never the Asset.**
  - Argues that the real asset is *the system’s ability to keep working*, not the accumulated codebase.[10]
  - Emphasizes that when LLMs can regenerate competent implementations quickly, the value shifts to problem framing, success criteria, and deletion safety.

- **The Gradient of Trust.**
  - Proposes a shape-first view: **better architectural shapes beat better prompts**; the system should encode where regeneration is safe vs. dangerous.[11]

- **Evaluations Are the Real Codebase.**
  - Claims evaluations (tests, checks, acceptance criteria) *are* the real codebase, because they define behavior independent of any specific implementation.[9]

- **Relocating Rigor.**
  - Rigor does not disappear; it moves from handwritten proofs into **feedback loops and architectural constraints** that make regeneration safe.[11]

- **Provenance Is the New Version Control.**
  - When code can be recreated on demand, **provenance and conversational history** become central; the commit is the *conversation* and its evidence, not the diff.[9]

- **The Deletion Test.**
  - A system passes the deletion test if you can safely delete large swathes of code knowing they can be regenerated from preserved intent and evaluations.[10]

- **The Conversation Is the Commit.**
  - Argues that in AI coding, the negotiation between human and agent (prompts, clarifications, rationales) is the real commit record; preserving it is essential for provenance and regeneration.[9]

**Tier-2/Tier-3 essays (skimmed via summaries).**

- **Pace Layers and AI Integration.** Different layers of a system should change at different speeds; regeneration should target fast layers and protect slow ones.[9]
- **The Phoenix Primitives.** The architecture is defined by what you *cannot* delete; primitives codify these non-deletable artifacts.[9]
- **Production Is a Compiler Input.** Observability and production behavior feed back into regeneration; production becomes part of the “compiler pipeline.”[9]
- Other essays (Compaction, Immutable Infrastructure, System Is the Asset, UI as Conservation Layer, The Regenerative Grain, The Generative Stack) elaborate cost/compaction, stability of UI, and how much you can delete at once.[9]

**Bearing on Crosscheck.**

- Strongly supports Crosscheck’s emphasis on **intent, evaluations, and provenance** as first-class protected surfaces: invariant docs, acceptance scenarios, attestation JSONs, and governance notes look very much like Phoenix “primitives.”
- Puts pressure on Crosscheck’s instinct to **treat certain code surfaces as protected**; under Phoenix, the code implementing an invariant should be regenerable, while the invariant itself, its acceptance oracle, and the governance record should be the durable assets.
- Suggests that Crosscheck’s extensive skill catalogue must genuinely encode **architectural shapes** (gradients of trust, regeneration boundaries), not just more elaborate prompts.

***

### Agentic Code Reasoning, Abductive Vibe Coding, Vibe Coding Needs Vibe Reasoning

**Agentic Code Reasoning (Ugare & Chandra).**

- Introduces **semi-formal reasoning** as a middle ground between informal chain-of-thought and fully formal proofs: structured templates with explicit premises, traced execution paths, and formal conclusions.[3]
- Shows that semi-formal reasoning improves patch-equivalence verification, fault localization, and code Q&A accuracy compared to unstructured reasoning, while remaining execution-free.[3]
- Certificates act as **machine-checkable evidence** without fully formalizing language semantics.

**Abductive Vibe Coding (Murphy et al.).**

- Proposes abductive vibe coding: structured, hierarchical rationales (inspired by Goal Structuring Notation) that explain *why* an AI-generated artifact is adequate.[4]
- Rationales decompose a top-level adequacy claim into subclaims and conjectures, some of which can be checked automatically, others forming a **checklist for human investigation**.[4]
- Emphasizes that the agent’s claims should be treated as **hypotheses**, not facts, and that the rationale must clearly state what would need to be true for the artifact to be considered adequate.

**Vibe Coding Needs Vibe Reasoning (Mitchell & Shaaban).**

- Analyzes long-horizon **vibe coding** workflows and shows that LLMs struggle to reconcile accumulating constraints, leading to technical debt, state-machine divergence, and “pit of despair” codebases.[5]
- Argues for **Vibe Reasoning**, a Type-III sidecar that:
  1. Autoformalizes verification targets from evolving natural-language constraints.
  2. Continuously verifies only critical invariants with the lightest effective techniques.
  3. Feeds actionable verification results back to the LLM.
  4. Keeps the developer in control as a collaborator, not an oracle.[5]
- Distinguishes Type I (formal methods filter LLM outputs) and Type II (formal methods post-process outputs) systems, and advocates a Type III system that keeps specifications and verification aligned with evolving intent.[5]

**Bearing on Crosscheck.**

- Validates Crosscheck’s `/rationale` skill: a structured adequacy argument with tagged leaves is essentially an **abductive rationale** in the sense of Murphy et al.
- Supports Crosscheck’s overall architecture as a **Type-III sidecar** for vibe coding: Dafny, invariant coverage, `/intent-check`, and `/spec-adversary` collectively form a Vibe Reasoning-like system attached to Claude Code.
- Highlights gaps: Crosscheck is still **largely one-shot and manual** in its flows; Vibe Reasoning calls for continuous, prioritized verification and automated autoformalization of evolving constraints.

***

## Cross-cutting themes across lenses

Several themes recur across the RiSE work, nl2spec, TiCoder, Phoenix, and the semi-formal reasoning papers:

1. **Intent formalization as the bottleneck.**
   - Lahiri and the intent-formalization repo argue that spec generation is tractable; **spec validation** is the hard part.[2][6]
   - TiCoder and nl2spec show that users can interactively refine tests and specs when given good tooling.[4][8]
   - Phoenix reframes this as deciding **what must persist across regenerations**: the durable specs and evaluations are the main bottleneck.[9]

2. **Value-per-friction sweet spot at the “lighter” end.**
   - TiCoder’s large gains from a handful of test queries and the Defects4J postcondition work suggest that **tests + postconditions** are often the best tradeoff.[5][4]
   - Lahiri’s FMCAD metrics show you can get meaningful spec-quality signals with **symbolic testing** rather than full proofs.[6]
   - Phoenix emphasizes evaluations and architectural boundaries over universal deep formalization.[9]

3. **Semi-formal reasoning and assurance cases.**
   - Agentic Code Reasoning and Abductive Vibe Coding converge on **structured, certificate-like reasoning templates** that can be partially machine-checked.[3][4]
   - Crosscheck’s `/rationale` and evidence-certificates are in exactly this space.

4. **Sidecar verification for long-horizon agentic workflows.**
   - Vibe Reasoning and RiSE’s agentic verifiers both imagine **sidecar systems** that continuously check invariants and specs as the code evolves.[3][5]
   - Crosscheck is explicitly such a sidecar for Claude Code, but its flows are more discrete than continuous.

5. **Governance, provenance, and kill-criteria.**
   - Phoenix’s “Provenance is the new version control” and the Intent Stack’s governance layers echo Crosscheck’s emphasis on **attestations, trackers, and protected surfaces**.[12][9]
   - Lahiri’s FMCAD work implicitly defines **kill-criteria for specs** via low completeness scores.[6]
   - Crosscheck adopts explicit kill-criteria (e.g., FP rate > 30% retires `/intent-check` as a gate), which is relatively rare in tooling.

Overall, the **academic lens and Phoenix lens apply pressure in the same direction**: center tests, postconditions, and evaluations; treat formal verification as a targeted tool for the hardest invariants; and invest heavily in governance and provenance.
Crosscheck’s design is broadly aligned with this direction, but its emphasis on Dafny as Layer 1 sometimes overshadows the lighter-weight, evaluation-centric layers where the evidence suggests most value lies.

***

## The seven cross-examinations

### 1. Is Crosscheck operating on the right slice of the intent-formalization spectrum?

Lahiri’s spectrum runs from **tests → code contracts → logical contracts/postconditions → DSLs and synthesis**.[2]
The evidence from TiCoder, Endres et al., and Lahiri’s own FMCAD work is that the **highest value-per-friction** is obtained in the middle: tests plus postconditions generated and evaluated automatically, with interactive user feedback where needed.[4][5][6]

Crosscheck’s architecture anchors Layer 1 on **Dafny logical specs**, with tests and acceptance oracles higher up the stack.
For highly critical, well-isolated core logic (pure functions, no I/O), this is defensible and synergistic with Lahiri’s Dafny-centric spec evaluation work.
But as a **Claude Code plugin aimed at mainstream workflows**, the cost of getting into Dafny (new language, verification failures, modeling work) is high relative to the incremental assurance gained over robust tests and postconditions.

Both RiSE and Phoenix push in the same direction:

- RiSE emphasizes that **intent formalization is the bottleneck** and advocates for **lightweight, interactive methods** (TiCoder, SPOTs) rather than universal deep verification.[5][2]
- Phoenix argues that **evaluations and provenance are the durable assets**, and code should be cheap and replaceable.[9]

On that spectrum, Crosscheck currently **over-weights the Dafny end** relative to its evaluation-centric layers.
A more balanced design would:

- Treat **Dafny Layer 1 as an optional, high-rigor path** for a very small set of core modules.
- Put **acceptance oracles, tests, and invariant docs at the center of the default workflow**, aligning with both TiCoder and Phoenix.

### 2. Does `/intent-check` correspond to validated techniques in the literature?

`/intent-check` is a **round-trip informalization pipeline**: it back-translates code+tests into English, then diffs that text against invariant prose to detect drift.
Structurally, this mirrors patterns in nl2spec (back-translating sub-formulas), Doc2Spec-style frameworks, and Endres/Lahiri’s round-trip evaluation of specs, but with important differences.

- **What’s validated elsewhere.**
  - nl2spec validates **human usability** of sub-translation and back-translation, but does not claim high precision for automated drift detection.[8]
  - Endres et al. and Lahiri’s FMCAD work validate **correctness/completeness metrics** based on tests and mutations, not purely on NL back-translation.[5][6]
  - Clover and related closed-loop systems in the literature rely on **execution-based or symbolic metrics** (tests, oracles, counterexamples) rather than NL comparisons.

- **What `/intent-check` actually does.**
  - Uses **two separate LLM prompts** (back-translator blind to prose; diff-checker blind to code) and a **semantic validation layer** to catch contradictions.
  - Tracks a **rolling false-positive rate** via a CSV and imposes a **kill-criterion at 30% FP**, which is novel and healthy.
  - **Does not mutate tests or specs**, and does not measure correctness/completeness in the sense of Lahiri.

- **The 96% claim.**
  - The ~96% figure is explicitly inherited from Midspiral’s `claimcheck` dev benchmark, which measured accuracy of round-trip checks on a specific internal corpus.
  - Crosscheck has **not yet demonstrated equivalent accuracy on user code or invariant docs**, and the docs themselves acknowledge that real-PR accuracy remains to be measured.

Relative to the literature, `/intent-check` is **directionally aligned** with round-trip and back-translation methods but **weaker on guarantees**:

- It lacks **mutation-based stress-testing** of specs/tests.
- Its only quantitative metric is **false-positive rate**, not spec soundness/completeness.
- It relies on NL similarity judgments, which the RiSE work explicitly warns are insufficient as the sole spec-quality signal.

If `/intent-check` is used as a **soft advice tool**, this is acceptable.
If it is used as a **hard merge gate**, it currently promises more than the literature can support.

### 3. How does `/spec-adversary` compare to adversarial/mutation/completeness-checking methods?

The RiSE “Spotting Specification Gaps” post and Lahiri’s FMCAD paper both advocate **actively attacking specifications** via small proof-oriented tests or symbolic mutations of tests and outputs.[5][6]
The common pattern is: generate adversarial examples; run them through a verifier; measure how often specs still hold when they shouldn’t.

`/spec-adversary` instead takes a **human-centered, NL-first approach**:

- It reads the invariant doc, code, and tests, then proposes up to **three candidate missing invariants**, each with a category, confidence, and supporting code references.
- It explicitly refuses to auto-promote these; humans must triage and then update invariant docs via `/protected-surface-amend`.
- It defines **kill-criteria in terms of signal-to-noise** (e.g., <1 accepted per 5 proposed over 4 weeks → reconsider).

Compared to the literature:

- `/spec-adversary` is **less formal and weaker as an oracle**: it does not produce counterexamples or failures, only proposed additional invariants.
- It is **stronger as a governance pattern**: the triage blocks, tracker files, and kill-criteria are explicit design to keep human review sustainable.

So `/spec-adversary` is best understood as a **pragmatic, human-in-the-loop adversary**, not a completeness metric.
It does not subsume SPOTs or Lahiri’s symbolic spec testing; if Crosscheck wants true spec-completeness signals, it should add a **SPOT-style or Lahiri-style spec-eval skill** alongside `/spec-adversary`.

### 4. Does the two-agent split align with how RiSE frames the problem?

RiSE’s framing is clear: **intent formalization is the central bottleneck**, with verification downstream of it.[1][2]
They also emphasize that spec generation is often easier than spec validation, and that much of the benefit comes from interactive, test-driven flows.

Crosscheck’s two-orchestrator split—Byfuglien for implementation, Hellebuyck for specs/governance—**recognizes this division in principle**:

- Byfuglien is responsible for Dafny, tests, static analysis, and low-level assurance.
- Hellebuyck is responsible for invariants, `/intent-check`, `/spec-adversary`, `/acceptance-oracle-draft`, and governance.

In practice, however, the **docs and narrative weight are skewed toward Byfuglien**:

- The README leads with Dafny-backed verification and Layer 1 as the headline feature.[13]
- Many skills under Hellebuyck are framed as “governance extras” rather than the primary interface.

RiSE’s work and Phoenix both suggest the **spec/governance side should be heavier**:

- Intent formalization and spec validation are where novel research is happening.
- Evaluations, invariants, and provenance are the durable Phoenix primitives.

So the split is **principled**, but Crosscheck’s **center of gravity is misaligned** with the RiSE diagnosis.
Rebalancing defaults and documentation to put Hellebuyck’s flows **front and center** would better reflect the state of the art.

### 5. What does Crosscheck not do that prior art has shown matters?

Several capabilities are notably missing or underdeveloped relative to the literature:

1. **Spec soundness/completeness metrics.**
   - Lahiri FMCAD demonstrates concrete, automatable metrics for spec quality using symbolic testing, which Crosscheck does not yet exploit.
   - Crosscheck tracks `/intent-check`’s false-positive rate but does not measure how often specs are too weak, vacuous, or incomplete.

2. **Interactive user-in-the-loop spec/test disambiguation.**
   - TiCoder shows that asking users **yes/no about tests** is highly effective and relatively low friction.[4]
   - Crosscheck has no analogous interactive loop; `/intent-check` and `/acceptance-oracle-draft` are primarily LLM-driven with human review after the fact.

3. **Change-intent compositionality.**
   - RiSE and Vibe Reasoning emphasize that **evolving requirements** require compositional reasoning about how changes propagate.
   - Crosscheck’s governance tools ensure co-editing of docs and tests, but there is no explicit model of **how multiple intent changes compose** over time.

4. **Spec-to-code translation and proof-feedback loops.**
   - AutoCLRS and tools like Auto-Verus use proof failures to iteratively refine both specs and implementations.
   - Crosscheck supports Dafny verification but does not yet provide a **tight loop** where proof obligations feed directly back into prompts and governance decisions.

5. **Quantitative evaluation of the assurance hierarchy.**
   - There is no data on how often each layer catches real issues, how often kill-criteria trigger, or how much user effort each layer costs.

Some of these omissions are **defensible scoping decisions** (e.g., not reimplementing full Auto-Verus inside a plugin).
Others—especially **lack of spec-quality metrics and interactive test-driven disambiguation**—look like real gaps given the weight of prior evidence.

### 6. Where does Crosscheck genuinely advance the state of the art?

Crosscheck’s main contributions are **architectural and operational**, not algorithmic:

- **Governance scaffolding as first-class tooling.**
  - The combination of `.claude/rules/protected-surfaces.md`, `/protected-surface-amend`, invariant coverage gates, and Layer 5/6 trackers is a sophisticated governance model that goes well beyond most research prototypes.
  - It operationalizes Phoenix-style ideas (non-deletable primitives, provenance as VC) into concrete workflows.

- **Assurance-hierarchy framing for practitioners.**
  - The 6-layer model is a digestible way to explain how Dafny, tests, semi-formal rationales, acceptance oracles, and governance relate.
  - This framing could help teams reason about **where to invest effort** before picking specific tools.

- **Deployment as a Claude Code plugin.**
  - Packaging Dafny verification, semi-formal reasoning, and governance scaffolding as a plugin integrated into an AI IDE is a meaningful step toward **productionization** of research ideas.

- **Evidence certificates adapted to code reasoning.**
  - `/rationale` is a concrete, developer-facing instantiation of abductive, semi-formal reasoning about code adequacy, bridging Agentic Code Reasoning and Abductive Vibe Coding.

It is fair to say that Crosscheck **does not advance the algorithmic state of the art** in verification or intent formalization.
Instead, it **integrates and operationalizes** existing ideas (Dafny verification, spec evaluation, semi-formal reasoning, Phoenix governance) into a coherent plugin with explicit kill-criteria and governance rules.
That is still non-trivial and valuable.

### 7. Is Crosscheck protecting the right artefact in the Phoenix sense?

Phoenix argues that in the AI-coding era, **code is disposable** and the durable artifacts are **intent, evaluations, architectural shape, and provenance**.[9]
Crosscheck’s design partially aligns and partially conflicts with this thesis.

- **Alignment.**
  - **Protected surfaces** focus on:
    - Invariant docs and property tests (intent + evaluations).
    - Agent configs, prompts, and workflow files (architectural shape for the AI coding system).
    - Attestations and trackers (`/intent-check`, `/spec-adversary` trackers, coverage docs) (provenance).
  - `/acceptance-oracle-draft` treats **evaluations as the real codebase**: scenarios live outside the agent’s write-scope and become an external oracle.
  - `/intent-check` and `/protected-surface-amend` embed **provenance into the development workflow** via attestation JSONs and governance notes.

- **Tensions.**
  - `/protected-surface-amend` can cause **code and spec surfaces to ossify** if too many files are classified as protected, making regeneration harder than Phoenix would recommend.
  - The emphasis on Dafny Layer 1 can tempt teams to treat verified code as **less disposable**, even though Phoenix would argue that code can still be regenerated as long as specs and evaluations persist.

- **Deletion Test and Crosscheck.**
  - Under the Deletion Test, Crosscheck passes if you can delete the Dafny code and regenerated Python/Go while preserving invariant docs, acceptance scenarios, and governance artifacts.
  - The plugin’s architecture is consistent with this: Dafny is an implementation detail; invariants and acceptance oracles are the durable assets.

- **“Better shapes vs. better prompts.”**
  - Many Crosscheck skills (invariant coverage gates, protected-surface governance, acceptance-oracle scaffolding) are genuine **architectural shapes**, not just prompts.
  - However, the **proliferation of skills** risks becoming “better prompts dressed as architecture” if they are not organized into a small set of Phoenix-like primitives.

In summary, Crosscheck is **mostly protecting the right artefacts** in Phoenix’s sense but needs to be vigilant that protected-surface policies do not accidentally make large chunks of implementation code undeletable.
The healthiest interpretation is: **protect invariants, evaluations, and provenance, not the particular implementation of the core logic**.

***

## Alignment and divergence ledger

Here is how the nine theoretical commitments line up against the three lenses (RiSE+intent-formalization, nl2spec/TiCoder, Phoenix/semi-formal reasoning).

For each commitment, I mark whether it is **confirmed**, **contradicted**, **refined**, or **orthogonal**, and by which sources.

1. **Plausible vs. correct framing.**
   - **Confirmed** by RiSE (Agentic Code Reasoning shows plausible-but-wrong reasoning; TiCoder and Defects4J postconditions show many plausible but incorrect implementations/specs).[3][5]
   - **Refined** by semi-formal reasoning work, which shows that structured certificates reduce “plausible hallucinations.”[4][3]
   - **Aligned** with Phoenix: plausibility is cheap; **trustworthy behavior** requires evaluations and governance.[9]

2. **Two-orchestrator split (Byfuglien/Hellebuyck).**
   - **Principled** and **aligned** with RiSE’s separation of intent formalization vs. verification, but **RiSE suggests the spec orchestrator should be heavier-weight**.[2]
   - **Refined** by Vibe Reasoning, which advocates a sidecar verifier that continuously reconciles constraints, suggesting Hellebuyck should own more continuous monitoring.[5]

3. **6-layer assurance hierarchy.**
   - **Conceptually aligned** with assurance hierarchies and the Intent Stack’s layered governance.[12]
   - **Unvalidated empirically**; no source confirms that six is the right number of layers or that the proposed partition matches where bugs actually appear.
   - **Refined** by Phoenix, which would likely group layers by **regeneration vs. conservation** rather than by verification technique.

4. **Dafny-as-Layer-1.**
   - **Confirmed in principle** by RiSE and Lahiri’s work on Dafny spec evaluation; Dafny is an excellent vehicle for formal postconditions and symbolic spec testing.[6]
   - **Constrained** by Vibe Reasoning and Phoenix: Dafny should be used **surgically** for the hardest invariants, not as a universal foundation.
   - **Limited** by practical constraints (no IO, concurrency, extern libs), which Crosscheck acknowledges but does not fully integrate into its trust model.

5. **Round-trip informalization (`/intent-check`, ~96% figure).**
   - **Partially confirmed**: back-translation and round-tripping are standard techniques (nl2spec, Doc2Spec), but **no literature supports a 96% accuracy claim for NL-only drift detection across arbitrary repos**.[8][14]
   - **Contradicted** in spirit by spec-evaluation work that insists on **test/mutation-based metrics** for spec soundness/completeness.[5][6]
   - **Refined** by Vibe Reasoning: NL back-translation is one input to autoformalization, not a standalone gate.[5]

6. **Adversarial spec probing (`/spec-adversary`).**
   - **Aligned** with the idea of attacking specs rather than only code (SPOTs, FMCAD), but **weaker** than mutation-based methods because it produces proposals rather than counterexamples.[6][5]
   - **Refined** by governance insights: the tracker and triage pattern is a strong addition that the literature largely leaves to future work.

7. **Governance scaffolding against drift.**
   - **Strongly confirmed** by Phoenix (which treats governance, provenance, and pace layers as central) and by the Intent Stack’s governance layers.[9][12]
   - **Novel in practice**: few tools operationalize kill-criteria, protected surfaces, and coverage gates as concretely as Crosscheck does.

8. **Semi-formal reasoning with evidence certificates.**
   - **Directly confirmed** by Agentic Code Reasoning and Abductive Vibe Coding, which both advocate structured, semi-formal rationales.[3][4]
   - Crosscheck’s `/rationale` is a faithful adaptation of these ideas into a developer-facing skill.

9. **Cited research grounding overall.**
   - Crosscheck generally **applies the cited work faithfully** at the level of ideas (semi-formal templates, abductive rationales, Type-III sidecars), but often **stops one step short** of the strongest techniques (e.g., it cites Lahiri but does not implement spec correctness/completeness metrics).

***

## Threats to validity (for Crosscheck’s assumptions)

1. **Assuming Dafny-verified cores dominate risk.** In many systems, the most serious bugs live in **I/O, concurrency, and integration logic**—precisely where Dafny cannot easily reach.
   Treating Layer 1 as “the hard part” risks leaving large, high-risk surfaces in Layers 2–3 with weaker guarantees.

2. **Treating `/intent-check` as a strong gate without spec-quality metrics.** Without mutation-based or test-based spec evaluation, `/intent-check` can be **fooled by vacuous or incomplete invariants**, and the 96% accuracy claim is not yet justified for arbitrary projects.

3. **Assuming teams will maintain governance artifacts.** The value of protected surfaces, invariant coverage gates, and trackers depends on **consistent, disciplined updates**.
   For many teams, this is a big cultural shift; without strong automation and clear payoffs, governance scaffolding may decay.

4. **Skill catalogue complexity.** A large catalogue of skills increases **cognitive load** and risks turning shapes into “prompt soup.”
   Without a small set of Phoenix-style primitives, users may not know which skill to use when or how they compose.

5. **Lack of empirical evaluation of the 6-layer model.** There is no data on **how many bugs each layer catches, how often kill-criteria fire, or what the false-negative rates are**.
   Without this, the hierarchy is an elegant theory but not yet an evidence-based engineering practice.

6. **Assuming spec docs are correct by construction.** Invariant docs are treated as ground truth for `/intent-check` and `/spec-adversary`.
   The RiSE work shows that **specs themselves can be wrong, weak, or incomplete**; until Crosscheck integrates spec-quality metrics, it risks building strong machinery on top of shaky specs.

***

## Improvements implementable in Crosscheck

Below are concrete changes, each with: (a) research/essay backing, (b) affected component, (c) proposed change, and (d) expected benefit.
Ordered roughly by **impact vs. effort**.

### 1. Add spec soundness/completeness metrics for Dafny specs and invariants

- **(a) Evidence.** Lahiri FMCAD’s correctness/completeness metrics for Dafny specs via symbolic testing.[6]
- **(b) Component.** New skill (e.g., `/spec-eval`) plus integration with `/intent-check` and invariant docs.
- **(c) Change.**
  - Implement a skill that, given a Dafny spec and its test cases, computes correctness and completeness scores à la Lahiri (Hoare-triple encoding with mutated outputs).
  - For modules with invariant docs but no Dafny code, approximate via property tests and mutation testing.
  - Surface these metrics in `/intent-check` reports and in governance docs, with thresholds that can act as kill-criteria for specs themselves.
- **(d) Benefit.** Turns invariant docs and Dafny specs from **assumed ground truth** into **graded artifacts** with measurable quality, aligning Layer 5/6 with the intent-formalization literature.

### 2. Introduce a TiCoder-style interactive test clarification loop

- **(a) Evidence.** TiCoder’s large pass@1 improvements from 1–5 user queries; accept@m metrics showing test agreement is easy to obtain.[4]
- **(b) Component.** New skill (e.g., `/intent-tests`) and extensions to `/acceptance-oracle-draft`.
- **(c) Change.**
  - Add a mode where Crosscheck proposes tests (or acceptance scenarios) and asks the user **yes/no/undefined** about whether they match intent.
  - Use user responses to prune or generate new scenarios and to refine invariants.
  - Track metrics similar to pass@k@m and accept@m in a tracker file.
- **(d) Benefit.** Aligns Crosscheck with proven low-friction intent formalization techniques, and gives concrete numbers for **user effort vs. reliability** of the test suite.

### 3. Strengthen `/intent-check` by integrating mutation-based evaluation

- **(a) Evidence.** Endres et al. and Lahiri show that **mutation-based evaluation** is key to avoiding vacuous specs.[5][6]
- **(b) Component.** `/intent-check` skill and its reference docs.
- **(c) Change.**
  - Extend `/intent-check` to optionally run a **mutation phase**:
    - For property tests, mutate assertions or input generators.
    - For Dafny specs, mutate postconditions or outputs, then rerun verification.
  - Use failures to adjust the **confidence basis** and to flag specs as too weak even when NL alignment looks good.
- **(d) Benefit.** Reduces over-trust in NL-based back-translation, making `/intent-check` a **hybrid NL+semantic evaluator** closer to the literature.

### 4. Re-center the default workflow on evaluations and invariants (Phoenix-aligned)

- **(a) Evidence.** Phoenix’s “Evaluations Are the Real Codebase” and “Code Was Never the Asset,” plus RiSE’s focus on tests/postconditions.[2][9][10]
- **(b) Component.** README, docs, `/assurance-init`, and agent routing logic.
- **(c) Change.**
  - Reorder documentation so that the **default Crosscheck path** is:
    1. `/assurance-init` → invariant docs + protected surfaces.
    2. `/acceptance-oracle-draft` → external acceptance scenarios.
    3. `/intent-tests` (TiCoder-like) → refine tests.
    4. Optional: Dafny Layer 1 for selected modules.
  - Update Byfuglien/Hellebuyck orchestration so new users are steered **first** to evaluations and governance, then to Dafny.
- **(d) Benefit.** Aligns user mental model with both Phoenix and the intent-formalization literature; reduces friction by making heavy formal methods opt-in rather than the perceived default.

### 5. Add a SPOTs-inspired spec gap probe for Dafny and property tests

- **(a) Evidence.** RiSE’s SPOTs work (“Spotting Specification Gaps with Small Proof-Oriented Tests”).[5]
- **(b) Component.** New skill (e.g., `/spot-spec-gaps`) plus optional integration into Layer 6.
- **(c) Change.**
  - Implement a skill that generates **small proof-oriented tests** for Dafny modules or property tests, then measures whether invariants/specs fail when they should.
  - Use failures to suggest missing invariants or to signal incomplete specs.
- **(d) Benefit.** Gives Layer 6 a **concrete, automated gap-detection mechanism**, complementing `/spec-adversary`’s human-focused proposals.

### 6. Simplify and canonicalize the skill catalogue into Phoenix-like primitives

- **(a) Evidence.** Phoenix’s emphasis on a small set of primitives (“what you can’t delete”) and Gradient of Trust shapes.[9][11]
- **(b) Component.** Skill catalogue and agent-selection rules.
- **(c) Change.**
  - Group skills into a handful of **primitives**: e.g., *Verify Core Logic*, *Maintain Invariants*, *Maintain Evaluations*, *Govern Protected Surfaces*, *Probe Specs*.
  - Provide a **single entrypoint per primitive**, with subcommands or options instead of separate skills.
- **(d) Benefit.** Reduces cognitive load, emphasizes **architectural shapes** over prompts, and makes it easier to reason about coverage and responsibilities.

### 7. Add minimal continuous verification hooks (Vibe Reasoning / A3 style)

- **(a) Evidence.** Vibe Reasoning’s Type-III sidecar and A3’s agentic verifier loops.[3][5]
- **(b) Component.** `/invariant-coverage-scaffold`, `/intent-check`, and CI integration.
- **(c) Change.**
  - Introduce a **lightweight daemon or CI mode** that runs key checks (invariant coverage, `/intent-check` on protected surfaces, acceptance oracles) on a **schedule or per-branch basis**, not only on manual invocation.
  - Prioritize which specs/tests to check based on change impact and past failures.
- **(d) Benefit.** Moves Crosscheck closer to a **live sidecar** that continuously enforces invariants, as advocated by Vibe Reasoning and A3, without requiring full-blown agents.

### 8. Instrument and publish metrics for the assurance hierarchy

- **(a) Evidence.** RiSE’s emphasis on benchmarks and metrics; the need to treat the hierarchy as a hypothesis.[2]

Sources
[1] Intent Formalization: A Grand Challenge for Reliable Coding in the ... https://risemsr.github.io/blog/2026-03-05-shuvendu-intent-formalization/
[2] A Grand Challenge for Reliable Coding in the Age of AI Agents - arXiv https://arxiv.org/html/2603.17150v1
[3] Claude Code Plugin Marketplace Skill | Validate & Manage https://mcpmarket.com/tools/skills/claude-code-plugin-marketplace-manager
[4] cc-agents-md #1673 - hesreallyhim/awesome-claude-code - GitHub https://github.com/hesreallyhim/awesome-claude-code/issues/1673
[5] Agent Skills Overview - Agent Skills https://agentskills.io/home
[6] Customize Claude Code with plugins - Hacker News https://news.ycombinator.com/item?id=45530150
[7] nl2spec/Readme.md at main · realChrisHahn2/nl2spec https://github.com/realChrisHahn2/nl2spec/blob/main/Readme.md
[8] nl2spec: Interactively Translating Unstructured https://arxiv.org/pdf/2303.04864.pdf
[9] The Ashes Have Intent: Phoenix Architecture and Spec Driven ... https://jasongoecke.substack.com/p/the-ashes-have-intent-phoenix-architecture
[10] Rida Al Barazi's Post - The Phoenix Architecture https://www.linkedin.com/posts/rbarazi_the-death-and-rebirth-of-programming-the-activity-7409354029522300928-YxlG
[11] The Phoenix Architecture | daily.dev https://app.daily.dev/posts/the-phoenix-architecture-gtk7ny4id
[12] Clause 8 — The Seven Layers – The Intent Stack https://www.intentstack.org/docs/specification/8-seven-layers/
[13] Formally Verifying the Easy Part - by Harry - Brainflow - Substack https://brainflow.substack.com/p/formally-verifying-the-easy-part
[14] Synthesizing Formal Programming Specifications from ... https://arxiv.org/abs/2602.04892
[15] Your AGENTS.md is a Liability - Emergent Minds | paddo.dev https://paddo.dev/blog/your-agents-md-is-a-liability/
[16] Claude Skills and Claude MD: Complete Guide to Anthropic's New ... https://www.gend.co/blog/claude-skills-claude-md-guide
[17] Agentic Dev #1: Setting Up a Claude Code Plugin Marketplace https://www.youtube.com/watch?v=xlnsWvk3D4A
[18] AGENTS.md outperforms skills in our agent evals - Vercel https://vercel.com/blog/agents-md-outperforms-skills-in-our-agent-evals
[19] How to build your first Claude Skill in 30 minutes - Reddit https://www.reddit.com/r/automation/comments/1rx98j0/how_to_build_your_first_claude_skill_in_30/
[20] claude-plugins-official is broken due to commit bd04149 15 hours ago https://www.reddit.com/r/ClaudeCode/comments/1rqul0v/claudepluginsofficial_is_broken_due_to_commit/
[21] AGENTS.md Specification: A Research-Backed Guide - ASDLC.io https://asdlc.io/practices/agents-md-spec/
[22] Built a linter for SKILL.md files that catches cross-agent issues - Reddit https://www.reddit.com/r/ClaudeAI/comments/1rrdv9i/built_a_linter_for_skillmd_files_that_catches/
[23] Nicholls Inc - GitHub https://github.com/nicholls-inc
[24] A two-layer AGENTS.md template for cross-project and ... - Reddit https://www.reddit.com/r/codex/comments/1sway79/a_twolayer_agentsmd_template_for_crossproject_and/
[25] Digital Phenotyping for Adolescent Mental Health: Feasibility Study ... https://pmc.ncbi.nlm.nih.gov/articles/PMC12871944/
[26] CrossCheck - AI-Reviewed Code, Human-Level Quality - GitHub https://github.com/sburl/CrossCheck
[27] GitHub - pnnl/crosscheck https://github.com/pnnl/crosscheck
[28] Dafny https://dafny.org
[29] There are 28 official Claude Code plugins most people don't know ... https://www.reddit.com/r/ClaudeAI/comments/1r4tk3u/there_are_28_official_claude_code_plugins_most/
[30] CrossCheck command - github.com/Splinter0/CrossCheck https://pkg.go.dev/github.com/Splinter0/CrossCheck
[31] Dafny is a verification-aware programming language - GitHub https://github.com/dafny-lang/dafny
[32] double-check - Claude Code Plugin | ClaudePluginHub https://www.claudepluginhub.com/plugins/ananddtyagi-double-check-plugins-double-check
[33] CrossCheck https://github.com/Crosscheck
[34] [PDF] The Dafny Integrated Development Environment - Microsoft https://www.microsoft.com/en-us/research/wp-content/uploads/2016/12/krml236.pdf
[35] Claude Code Action Official - GitHub Marketplace https://github.com/marketplace/actions/claude-code-action-official
[36] nicholls-inc/claude-code-marketplace: Claude Code ... - GitHub https://github.com/nicholls-inc/claude-code-marketplace
[37] Assignment 7b -- Dafny https://users.cs.northwestern.edu/~robby/courses/396-2023-spring/assignment7b.html
[38] ccplugins/awesome-claude-code-plugins - GitHub https://github.com/ccplugins/awesome-claude-code-plugins
[39] Issues · nicholls-inc/claude-code-marketplace · GitHub https://github.com/nicholls-inc/claude-code-marketplace/issues
[40] Verification Optimization | Dafny Documentation https://dafny.org/latest/VerificationOptimization/VerificationOptimization
[41] nl2spec: Interactively Translating Unstructured Natural ... - GitHub https://github.com/realChrisHahn2/nl2spec
[42] realChrisHahn2 - GitHub https://github.com/realChrisHahn2
[43] [PDF] nl2spec: Interactively Translating Unstructured Natural ... https://finkbeiner.groups.cispa.de/publications/CHMST23.pdf
[44] NL2TL-dataset/datasets-nl2spec/Readme.md · tt-dart/NL2HLTL at e91a58c80944132b726700803a5891ae92a9484b https://huggingface.co/tt-dart/NL2HLTL/blob/e91a58c80944132b726700803a5891ae92a9484b/NL2TL-dataset/datasets-nl2spec/Readme.md
[45] nl2spec/.gitignore at main - GitHub https://github.com/realChrisHahn2/nl2spec/blob/main/.gitignore
[46] FORMAL SPECIFICATIONS FROM NATURAL LANGUAGE https://openreview.net/pdf?id=ywAjQw-spmY
[47] Security - realChrisHahn2/nl2spec - GitHub https://github.com/realChrisHahn2/nl2spec/security
[48] Releases · realChrisHahn2/nl2spec - GitHub https://github.com/realChrisHahn2/nl2spec/releases
[49] Interactively Translating Unstructured Natural Language ... - SIGARCH https://www.sigarch.org/interactively-translating-unstructured-natural-language-to-temporal-logics-with-nl2spec/
[50] Pull requests · realChrisHahn2/nl2spec - GitHub https://github.com/realChrisHahn2/nl2spec/pulls
[51] nl2spec: Interactively Translating Unstructured Natural ... https://finkbeiner.groups.cispa.de/publications/CHMST23.html
[52] Verifiable Natural Language to Linear Temporal Logic Translation https://arxiv.org/html/2507.00877v2
[53] microsoft/WaveCoder: Advancing LLM with Diverse Coding ... - GitHub https://github.com/microsoft/WaveCoder
[54] Microsoft - GitHub https://github.com/microsoft
[55] GitHub - microsoft/intent-formalization: Artefacts for evaluating user ... https://github.com/microsoft/nl-2-postcond
[56] Training for GitHub - Microsoft Learn https://learn.microsoft.com/en-us/training/github/
[57] Activity · microsoft/intent-formalization - GitHub https://github.com/microsoft/nl-2-postcond/activity
[58] GitHub - microsoft/ExeCoder: Train an LLM specifically designed for ... https://github.com/microsoft/ExeCoder
[59] microsoft/TiCoder - GitHub https://github.com/microsoft/TiCoder
[60] Introduction to GitHub | Microsoft Learn https://learn.microsoft.com/en-us/shows/start-dev-change-start-dev-change/introduction-to-github
[61] Shuvendu Lahiri's Post - Intent Formalization - LinkedIn https://www.linkedin.com/posts/shuvendu-lahiri-9a35151_intent-formalization-a-grand-challenge-for-activity-7435490798206603264-2NCM
[62] NextCoder: Robust Adaptation of Code LMs to Diverse ... - GitHub https://github.com/microsoft/NextCoder
[63] Intent Formalization: A Grand Challenge for Reliable Coding in the ... https://www.microsoft.com/en-us/research/publication/intent-formalization-a-grand-challenge-for-reliable-coding-in-the-age-of-ai-agents/
[64] Isn't it a little contradictory that Github, the world's most popular open ... https://www.reddit.com/r/opensource/comments/1h5z9ai/isnt_it_a_little_contradictory_that_github_the/
[65] MicrosoftResearch/Intent-based-Task-Representation-Learning https://github.com/MicrosoftResearch/Intent-based-Task-Representation-Learning/issues
[66] Pere Villega's Post - The Phoenix Architecture https://www.linkedin.com/posts/perevillega_the-death-and-rebirth-of-programming-the-activity-7409503463463727104-u_N8
[67] Intimacy Gradient and Other Lessons from Architectur... https://www.lifewithalacrity.com/article/intimacy-gradient-and-other-lessons-from-architecture/
[68] Stop Maintaining Your Code. Start Replacing It - Tessl https://tessl.io/podcast/98/
[69] Stop Maintaining Your Code. Start Replacing It - Apple Podcasts https://podcasts.apple.com/in/podcast/stop-maintaining-your-code-start-replacing-it/id1756073806?i=1000757049511
[70] pdd_hp_pdf_00205.pdf https://www.phoenix.gov/content/dam/phoenix/pddsite/documents/hp/pdd_hp_pdf_00205.pdf
[71] The new architecture of trust https://www.pwc.com/gx/en/issues/trust/new-architecture-of-trust.html
[72] Chad Fowler - Atmosphere Conference https://atmosphereconf.org/profile/chadfowler.com
[73] The Ashes Have Intent: Phoenix Architecture and Spec Driven ... https://www.linkedin.com/pulse/ashes-have-intent-phoenix-architecture-spec-driven-jason-goecke-7nzcc
[74] Nitya Narasimhan, PhD's Post - LinkedIn https://www.linkedin.com/posts/nityan_just-discovering-the-phoenix-architecture-activity-7420370846038364160-DFKS
[75] Regenerate Software Nightly for Decoupling Legacy Code - LinkedIn https://www.linkedin.com/posts/heathermdowning_ive-been-talking-about-this-same-approach-activity-7442313703426961408-XQ2a
[76] Simon Vart's Post - The Phoenix Architecture https://www.linkedin.com/posts/simon-vart-67b28025_the-death-and-rebirth-of-programming-the-activity-7417958875254988802-ivcu
[77] Phoenix: Architecture Art Regeneration - All Books https://web.archive.org/web/20110708012456/http:/blackdogonline.com/all-books/phoenix.html
[78] Assurance hierarchies in B2C electronic commerce https://nvlpubs.nist.gov/nistpubs/Legacy/IR/nistir6713.pdf
[79] ninthspace/claude-code-marketplace https://github.com/ninthspace/claude-code-marketplace
[80] Common Criteria https://www.ipa.go.jp/security/jisec/about/gmcbt80000005r7p-att/CCPART3V3.1R4.pdf
[81] A Multi-layered Approach to Security in High Assurance Systems1 https://www.uidaho.edu/~/media/UIdaho-Responsive/Files/engr/research/csds/publications/2004/A%20Multi-Layered%20Approach%20to%20Security%20in%20High%20Assurance%20Systems%202004.ashx
[82] Developers Are SLEEPING on Claude Code Plugins https://www.youtube.com/watch?v=vKgfoQZgkYY
[83]  https://docbox.etsi.org/MTS/MTS/10-PromotionalMaterial/MBS-20111118/Referenced%20Documents/iso15408-3%20CC%20Assurance.pdf
[84] Skill Reviewer: Audit & Validate Claude Code Skills - MCP Market https://mcpmarket.com/tools/skills/skill-reviewer-auditor
[85] Cross-Check, Verification Using Different Methods, and Quality ... https://www.tarmacview.com/glossary/cross-check/
[86] [PDF] Security assurance components April 2017 Version 3.1 Revision 5 https://www.commoncriteriaportal.org/files/ccfiles/CCPART3V3.1R5.pdf
[87] Plugin marketplaces https://code.claude.com/docs/en/plugin-marketplaces
[88] [PDF] Security assurance components September 2012 Version 3.1 https://www.commoncriteriaportal.org/files/ccfiles/CCPART3V3.1R4.pdf
[89] Dev-GOM/claude-code-marketplace https://github.com/Dev-GOM/claude-code-marketplace
[90] České vysoké učení technické https://petamaj.github.io/other/crosscheck.pdf
[91] Security Assurance Requirements March 2004 Version 2.4 ... https://www.commoncriteriaportal.org/files/ccfiles/ccpart3v2.4r256.pdf
[92] Alignment Layer: Mechanisms in AI & Materials - Emergent Mind https://www.emergentmind.com/topics/alignment-layer
[93] The ultimate Claude Code marketplace hub https://www.reddit.com/r/ClaudeCode/comments/1o3txls/the_ultimate_claude_code_marketplace_hub/
[94] [PDF] Three-Layer Agents and Alignment Verification - PhilArchive https://philarchive.org/archive/MAHTAA-5
[95] BadgeApp Security: Its Assurance Case - GitHub https://github.com/coreinfrastructure/best-practices-badge/blob/main/docs/assurance-case.md
[96] 2 Security assurance requirements - Page Notes https://pagenotes.com/writings/ccToolbox6f/CCManual/PART3/PART32.HTM
[97] Cross-Layer Feature Alignment and Steering in Large Language ... https://www.alignmentforum.org/posts/feknAa3hQgLG2ZAna/cross-layer-feature-alignment-and-steering-in-large-language-2
[98] Using CrossCheck to Police Your Defined Terms and Look for Other ... https://www.adamsdrafting.com/crosscheck-qa-with-steven-gullion/
[99] Cross-Layer Feature Alignment and Steering in Large Language Model https://www.greaterwrong.com/posts/feknAa3hQgLG2ZAna/cross-layer-feature-alignment-and-steering-in-large-language-2
[100] CrossCheck: Claude Code skill for automation - Shyft https://shyft.ai/skills/crosscheck
[101] 2 LLMs instead of one - Facebook https://www.facebook.com/groups/claudeaicommunity/posts/1265085218991976/
[102] GitHub - Beneficial-AI-Foundation/dafny-autopilot https://github.com/Beneficial-AI-Foundation/dafny-autopilot
[103] Inside a five-year-old startup's rapid AI makeover https://newsletter.pragmaticengineer.com/p/ai-first-makeover-craft
[104] Dafny Verifier MCP server for AI agents - Playbooks https://playbooks.com/mcp/namin-dafny-verifier
[105] What tools and techniques are you using to verify AI-generated code ... https://www.reddit.com/r/ExperiencedDevs/comments/1rzq738/what_tools_and_techniques_are_you_using_to_verify/
[106] Harry Nicholls (@HarryNicholls) / Posts / X - Twitter https://x.com/HarryNicholls
[107] The gap between what technical and non-technical people get from ... https://www.reddit.com/r/ClaudeAI/comments/1spnb80/the_gap_between_what_technical_and_nontechnical/
[108] nicholls-inc/xylem: Turns labeled GitHub issues into ... - GitHub https://github.com/nicholls-inc/xylem
[109] code-review | Skills Marketplace · LobeHub https://lobehub.com/it/skills/iimuz-dotfiles-code-review
[110] AI thinks your code is correct, but it can not prove it - hackernews ... https://hackernews.pink/story/47361931
[111] Vibe coded plugins - Page 5 - Instruments Forum - KVR Audio https://www.kvraudio.com/forum/viewtopic.php?t=629251&start=60
