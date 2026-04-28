# Brief Literature Review: Formal Verification Hierarchies and AI-Assisted Code Assurance

## Purpose

This review was conducted to assess whether the six-layer assurance hierarchy described in the companion report ("A Six-Layer Assurance Hierarchy for AI-Assisted Software Development") has precedent in existing literature. The review is preliminary — conducted over the course of a single conversation, not a systematic literature search — and should be treated as a starting point for a proper review, not a definitive novelty claim.

## Existing Work

### Guaranteed Safe AI (Dalrymple et al., 2024)

The closest structural analog identified. This paper proposes a verification hierarchy spanning from no guarantees (Level 0) through increasingly sophisticated empirical testing methods (Levels 1–5), probabilistic inference with convergence guarantees (Levels 6–8), to formal proofs establishing definitive safety bounds (Levels 9–10). The framework is organized around three core components: a world model (mathematical description of how the AI system affects the outside world), a safety specification (mathematical description of acceptable effects), and a verifier (auditable proof certificate).

The paper also notes that AI capabilities could accelerate creation of good safety specifications — for example, AI systems could suggest new specifications, critique proposed ones, or generate examples of cases where two candidate specifications differ.

**Relevance:** This is the most important prior work to engage with. The confidence gradient (deterministic → probabilistic → best-effort) and the structural decomposition into distinct verification concerns are shared features. The key divergences are: (1) GS AI targets AI system safety broadly, not developer workflows specifically; (2) it does not distinguish an implementation chain from a specification chain; (3) it does not address compilation correctness, contract graph verification at integration boundaries, or the spec-intent alignment problem as distinct layers.

### Kleppmann (2025)

Martin Kleppmann's blog post argues that formal verification is about to become mainstream due to three converging factors: formal verification is becoming vastly cheaper, AI-generated code needs formal verification so human review can be skipped, and the precision of formal verification counteracts the imprecise nature of LLMs. He identifies the specification problem clearly: as verification becomes automated, the challenge moves to correctly defining the specification — knowing that the properties proven are the right ones.

**Relevance:** Kleppmann names the spec completeness problem but does not propose a layered hierarchy or structured approach to addressing it. The observation that "it doesn't matter if [LLMs] hallucinate nonsense, because the proof checker will reject any invalid proof" is foundational context for why LLM-assisted formal verification is viable.

### Midspiral: lemmafit and claimcheck

Midspiral builds open-source tools for what they call "the correctness layer for AI-generated software." Two tools are directly relevant:

**lemmafit** integrates Dafny formal verification into the Claude Code development workflow. Business logic is written in Dafny, mathematically verified, then auto-compiled to TypeScript for use in React applications. A verification daemon watches .dfy files, runs `dafny verify`, and on success compiles to JavaScript/TypeScript. Code that cannot be proven correct does not compile.

**claimcheck** addresses the gap between "verified" and "intended" using round-trip informalization. The technique sends a Dafny lemma to one LLM (blind to the original requirement) to describe what it guarantees in plain English, then sends both the original requirement and the back-translation to a second LLM to assess semantic match. In testing against an election tally system, claimcheck caught both a planted error (a tautology masquerading as a monotonicity guarantee) and an unexpected gap (a lemma proving one specific reordering rather than arbitrary permutation). Structural separation (two different models, informalizer blind to the requirement) outperformed single-model approaches by 10 points. Reported accuracy is 96.3%, acknowledged as a development benchmark rather than a formal evaluation.

**Relevance:** Midspiral addresses spec-intent alignment (Layer 5 of the hierarchy) as a specific tool, and frames it as "one layer in an ongoing attempt to map natural language requirements to formal guarantees." They do not propose a broader hierarchy. Their blog post "From Intent to Proof" notes that a more transparent workflow would surface generated domain obligations back to the user in human-readable form before proof generation begins, allowing iteration on intent before committing to proof search — which aligns with the specification chain concept but is not formalized as such.

### Axiom

Axiom trains AI systems to generate formally verified outputs in Lean. The system uses deterministic proof verifiers to detect incorrect outputs, providing mathematical certainty that code functions return the correct answer for every input. Axiom also verifies that code does not introduce hidden vulnerabilities. The company raised $200M in Series A funding (March 2026) at a $1.6B valuation, and has released AXLE (Axiom Lean Engine) as a public API providing proof verification and manipulation primitives.

Axiom's approach involves a "verified data flywheel" — orders of magnitude more formally verified data than all previously available human-produced sources — which feeds back into training. Because this data is checked by deterministic proof verifiers, it avoids the data pollution and model collapse problems associated with unverified AI-generated training data.

**Relevance:** Axiom provides the tooling substrate for Layer 1 (formally verified code) via Lean. Their focus is on making formal verification cheap and automated, not on defining an assurance hierarchy. The recursive self-improvement loop (verified data improving the model that generates verified data) is relevant infrastructure but does not address the specification chain.

### Harmonic: Aristotle

Harmonic's Aristotle system combines formal verification with informal reasoning, achieving gold-medal-equivalent performance on the 2025 International Mathematical Olympiad with formally verified solutions in Lean 4. The system can autonomously prove and formalize for up to 24 hours without human intervention and leads ProofBench rankings. Harmonic's roadmap includes expansion from mathematics into software verification, targeting safety-critical industries.

The system works by requiring models to produce reasoning as code in Lean 4, which is then checked by an algorithmic process independent of the AI. Every solution is verified down to foundational axioms using the Lean 4 proof assistant.

**Relevance:** Like Axiom, Aristotle provides tooling for Layer 1. Harmonic's stated roadmap into software verification and code correctness suggests these tools will become directly applicable to the assurance hierarchy's implementation chain. The key architectural insight — that the verifier is outside the loop and does not negotiate — aligns with the hierarchy's principle that each layer provides independent assurance.

### de Moura (2026)

Leonardo de Moura's blog post surveys the Lean ecosystem and argues that verification will be a decisive advantage in software development. He notes that AI agents are already building their own proof strategies on top of the Lean platform, and highlights Veil, a distributed protocol verifier built on Lean that combines model checking with full formal proof. Veil generates concrete counterexamples when properties fail and full formal proofs when they hold. During verification of the Rabia consensus protocol, the Veil team discovered an inconsistency in a prior formal verification that had gone undetected across two separate tools.

De Moura also raises the specification problem as not purely technical: specifications for medical devices, voting systems, or AI safety monitors encode values, not just logic. He suggests AI can bridge the accessibility gap by explaining specifications in plain language.

**Relevance:** The Veil work is directly relevant to Layer 3 (contract graph verification), specifically for distributed systems where composition verification is hardest. The observation about specifications encoding values connects to the boundary the hierarchy draws between Layer 6 (spec completeness) and the explicitly excluded "right solution for the right problem" domain.

### Skomarovsky (2026)

A practitioner-oriented post proposing five independent structural dimensions that determine whether AI-generated code is amenable to formal verification. The framework classifies code artifacts by tractability rather than defining layers of assurance.

**Relevance:** Complementary but structurally different. Skomarovsky's framework answers "can we verify this?" while the assurance hierarchy answers "given that we can, what layers of confidence do we have?"

## Assessment of Novelty

The individual concepts in the hierarchy exist separately in the literature:

- **Confidence gradients** in verification are present in the GS AI paper
- **Spec-intent alignment** is addressed by Midspiral's claimcheck
- **Spec completeness** as the limiting problem is named by Kleppmann and Midspiral
- **Contract verification at composition boundaries** is well-studied, particularly in hardware verification and distributed systems (Veil)
- **Compilation correctness** is a known concern (CompCert, translation validation)
- **Formally verified code via LLM-assisted tools** is actively developed by Axiom, Harmonic, and Midspiral

What was not found in this review is a single framework assembling these into a unified hierarchy with the specific structure proposed: the split into an implementation chain (Layers 1–3, deterministic) and a specification chain (Layers 4–6, degrading from deterministic through probabilistic to best-effort), with explicit identification of where residual risk concentrates.

**Caveat:** This review was conducted in a single conversation session, not as a systematic literature search. "Not found" is weaker than "does not exist." A proper literature review — particularly engaging deeply with the GS AI paper, the formal methods in AI-generated code space, and the DO-178C / safety-critical systems literature — would be required before making any novelty claims in publication.

## References

- Dalrymple, D. et al. "Towards Guaranteed Safe AI: A Framework for Ensuring Robust and Reliable AI Systems." 2024. https://arxiv.org/html/2405.06624v1
- Kleppmann, M. "Prediction: AI will make formal verification go mainstream." December 2025. https://martin.kleppmann.com/2025/12/08/ai-formal-verification.html
- Midspiral. "claimcheck: Narrowing the Gap between Proof and Intent." https://midspiral.com/blog/claimcheck-narrowing-the-gap-between-proof-and-intent/
- Midspiral. "From Intent to Proof: Dafny Verification for Web Apps." https://midspiral.com/blog/from-intent-to-proof-dafny-verification-for-web-apps/
- Midspiral. "lemmafit: Make agents prove that their code is correct." GitHub. https://github.com/midspiral/lemmafit
- Axiom. https://axiommath.ai/
- Axiom. AXLE - Axiom Lean Engine. https://axle.axiommath.ai/
- SiliconANGLE. "Verifiable AI startup Axiom raises $200M to prove AI-generated code is safe to use." March 2026. https://siliconangle.com/2026/03/12/verifiable-ai-startup-axiom-raises-200m-prove-ai-generated-code-safe-use/
- Menlo Ventures. "AI Will Write All the Code. Mathematics Will Prove It Works." March 2026. https://menlovc.com/perspective/ai-will-write-all-the-code-mathematics-will-prove-it-works/
- Harmonic. "Aristotle: IMO-level Automated Theorem Proving." arXiv, March 2026. https://arxiv.org/html/2510.01346v1
- Harmonic. News. https://harmonic.fun/news
- de Moura, L. "When AI Writes the World's Software, Who Verifies It?" February 2026. https://leodemoura.github.io/blog/2026/02/28/when-ai-writes-the-worlds-software.html
- Lean FRO. https://lean-lang.org/fro/about/
- Skomarovsky. "Your AI Just Wrote 500 Lines of Code. Can You Prove Any of It Works?" April 2026. https://skomarovsky.github.io/verification/verification_framework.html
