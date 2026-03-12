# Paper Review: Abductive Vibe Coding vs. Semiformal Plugin

**Paper**: "Abductive Vibe Coding (Extended Abstract)" — Murphy, Babikian, Chechik (University of Toronto, 2026)
**Source**: https://arxiv.org/abs/2601.01199
**Date**: 2026-03-12

## Paper Summary

The paper proposes a framework for validating AI-generated code ("vibe-coded artifacts") that sits between two extremes:

- **Naive approach**: unstructured natural language explanations (wide applicability, manual validation)
- **Deductive approach**: formal proofs of correctness (narrow applicability, automatic validation)

The key idea is **abductive reasoning** (Peircean hypothesis-generation): assume the LLM is _entirely untrustworthy_, and instead of asking it to prove correctness, ask it to generate **hypotheses which, if true, would imply adequacy**. These are organized as:

1. **Rationales** — structured, hierarchical trees of claims (inspired by Goal Structuring Notation / structured assurance cases)
2. **Claims** decompose into subclaims via inferences: `(C1 ∧ C2 ∧ ... ∧ Cn) ⟹ C`
3. **Conjectures** — leaf claims that require external verification (static analysis, SMT solvers, or human review)
4. **Checklist** — unverified conjectures + failed inferences returned to the user, with the property that _if all items hold, adequacy is proven_
5. **Analysis engine** — checks inference validity (e.g., via SMT) and attempts to verify conjectures via program analysis tools

Claims can be **formal** (first-order logic) or **semiformal** (containing uninterpreted predicates like `Accusatory(s)`). The framework is being implemented in Lean.

## Comparison

| Dimension | Abductive Vibe Coding (Paper) | Semiformal Plugin |
|---|---|---|
| **Core reasoning model** | Abductive — generate hypotheses that _would imply_ adequacy if true | Deductive-empirical — gather premises from code, derive claims, reach conclusions |
| **Trust model** | Agent is _entirely untrustworthy_; every claim is a conjecture until externally verified | Agent is trusted to read code accurately; evidence is file:line citations, but verification is by the LLM itself |
| **Structure** | Tree of claims with formal decomposition rules `(∧Ci) ⟹ C` | Linear certificate: Premises → Claims → Alternative Hypothesis → Conclusion |
| **Verification** | External: SMT solvers, static analysis tools, human review of checklist | Internal: the LLM verifies its own premises by reading code |
| **Formality level** | Mixed formal/semiformal — some claims in first-order logic, some with uninterpreted predicates | Semiformal — structured natural language with file:line evidence, no formal logic |
| **Output to user** | Checklist of items to verify (with property that if all hold, adequacy is proven) | Reasoning certificate with confidence level (HIGH/MEDIUM/LOW) |
| **Scope** | Validating _generated_ code against (possibly informal) specifications | Analyzing _existing_ code — fault localization, patch comparison, execution tracing, Q&A |
| **Iteration** | Proposed but not yet implemented — user refines conjectures iteratively | Single-pass with orchestrator quality gates |
| **External tooling** | Lean proof assistant, SMT solvers, static analysis | None — pure prompt-based reasoning |

## Ideas Worth Adopting

### 1. Checklist-as-contract output

The paper's strongest contribution is the **checklist property**: decompose a top-level claim into subclaims such that if the user validates every leaf item, the top-level claim is _proven_ by construction. The semiformal plugin currently ends with a confidence level (HIGH/MEDIUM/LOW), which is a subjective assessment. Adding a structured checklist of "things the user must verify for this conclusion to hold" would make the output far more actionable.

**Adaptation**: Add a `## Verification Checklist` section to each skill's output template listing: (a) premises the user should spot-check, (b) assumptions made about framework behavior, (c) alternative hypotheses ruled out but revisitable.

### 2. Claim classification by verification type

The paper distinguishes claims that can be automatically verified (static analysis) from those requiring human judgment (domain knowledge). The semiformal plugin treats all premises equally.

**Adaptation**: Tag each premise/claim with a verification class — `[STATIC]` (verifiable by reading code), `[SEMANTIC]` (requires domain knowledge), `[BEHAVIORAL]` (requires running code). This helps users focus review effort.

### 3. Dependency annotations between claims

The paper's tree structure makes logical dependencies explicit. However, deeply nested trees are impractical in a CLI context.

**Adaptation**: Keep the flat structure but add dependency annotations — e.g., `C[3] (from P[1], P[4])` — giving traceability without the readability cost of deep nesting.

### 4. Uninterpreted predicate marking

When a claim involves subjective judgment (e.g., "this naming is idiomatic"), explicitly mark it as `[UNINTERPRETED]` — the user must supply the judgment.

## Pushback: What Doesn't Fit

### The paper's trust model is impractical for Claude Code

The paper assumes the agent is "entirely untrustworthy" and delegates verification to external tools (SMT solvers, Lean). This is impractical for the semiformal plugin:

1. **No external verification toolchain.** The plugin is pure prompts — no MCP server, no Docker, no external tools. Adding Lean/SMT would make it a fundamentally different plugin (more like `crosscheck/`). The lightweight, zero-dependency nature is a core strength.

2. **The verification bottleneck shifts, not shrinks.** Requiring users to validate every leaf conjecture replaces "trust the LLM's reasoning" with "trust the LLM's decomposition." For a developer using `/locate-fault`, a checklist of formal conjectures is more work than the current certificate format.

3. **Abductive framing is epistemologically interesting but operationally equivalent.** Both systems have the LLM read code, make claims, and structure them for review. The paper adds formal semantics but the actual reasoning quality depends on prompt engineering, not on the philosophical framing.

### Better fit: crosscheck-semiformal integration

The abductive framework would naturally extend the **crosscheck** plugin (which has Dafny as a verification backend):

- Use Dafny to verify "automatically checkable" conjectures
- Use semiformal reasoning for "human review" conjectures
- Produce a unified checklist showing which items were machine-verified vs. needing human review

This would be a **crosscheck-semiformal integration** rather than changes to semiformal alone.

## Recommended Changes

| # | Idea | Adaptation | Effort |
|---|---|---|---|
| 1 | Checklist output | Add `## Verification Checklist` to skill outputs | Low |
| 2 | Claim classification | Tag claims as `[STATIC]`/`[SEMANTIC]`/`[BEHAVIORAL]` | Low |
| 3 | Dependency tracking | Add dependency annotations to claims | Low |
| 4 | Uninterpreted predicates | Mark subjective claims as `[UNINTERPRETED]` | Low |
| 5 | Iterative refinement | Allow users to reject/refine premises after review | Medium |
| 6 | Crosscheck integration | New `/verify-rationale` skill using Dafny for `[STATIC]` claims | High |

Items 1–4 are worth doing now. Item 5 is a good roadmap item. Item 6 is the real payoff but requires cross-plugin architecture work.
