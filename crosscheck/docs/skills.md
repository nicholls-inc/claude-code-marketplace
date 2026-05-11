# Crosscheck Skill Catalogue

Exhaustive index of all 27 skills in the crosscheck plugin, grouped by category. See [`../README.md`](../README.md) for the plugin overview, and [`./agents.md`](./agents.md) for the orchestrator agent pages (`byfuglien`, `hellebuyck`).

## Formal verification (Dafny verify-and-extract)

| Skill | Trigger phrases | One-line summary | Owner |
|-------|----------------|------------------|-------|
| [`/spec-iterate`](../skills/spec-iterate/SKILL.md) | "specify", "formal spec", "preconditions", "postconditions" | Draft and verify a Dafny formal specification from a natural-language description. | byfuglien |
| [`/generate-verified`](../skills/generate-verified/SKILL.md) | "implement the spec", "generate verified code", "prove the implementation" | Generate a Dafny implementation body that satisfies a verified spec. | byfuglien |
| [`/extract-code`](../skills/extract-code/SKILL.md) | "extract to python", "extract to go", "compile dafny" | Compile verified Dafny to Python or Go with runtime boilerplate stripped. | byfuglien |
| [`/lightweight-verify`](../skills/lightweight-verify/SKILL.md) | "lightweight verify", "add contracts", "property-based tests", "assertions" | Generate design-by-contract assertions, property-based tests, or runtime invariants when full Dafny verification is overkill. | byfuglien |
| [`/assurance-probe`](../skills/assurance-probe/SKILL.md) | "assurance probe", "test strength", "mutation probe", "vacuity probe" | Measure test strength via mutation, vacuity, and generator probes (rotation-based; Phase 1 — experimental). | byfuglien |

## Formal verification (Lean executable-model + DRT-oracle pipeline)

The five-step Lean pipeline. Run sequentially; each consumes the previous step's artefact.

| Skill | Trigger phrases | One-line summary | Owner |
|-------|----------------|------------------|-------|
| [`/informal-spec`](../skills/informal-spec/SKILL.md) | "informal spec", "prose spec", "extract specification" | Extract a precise prose specification with hard human sign-off (step 1 of 5). | byfuglien |
| [`/lean-spec`](../skills/lean-spec/SKILL.md) | "lean spec", "lean spec stub", "translate spec to lean" | Translate signed-off prose into a Lean 4 stub with `sorry` proof bodies (step 2 of 5). | byfuglien |
| [`/lean-impl`](../skills/lean-impl/SKILL.md) | "lean impl", "lean implementation", "executable lean model" | Translate source implementation into a Lean 4 functional model (step 3 of 5). | byfuglien |
| [`/correspondence-review`](../skills/correspondence-review/SKILL.md) | "correspondence review", "lean correspondence", "model fidelity" | Classify each Lean def vs source as exact / abstraction / approximation / mismatch (step 4 of 5). | byfuglien |
| [`/drt-oracle`](../skills/drt-oracle/SKILL.md) | "drt oracle", "differential random testing", "lean drt", "fuzz against lean" | Differential random testing between the Lean model and production code (step 5 of 5). | byfuglien |

## Semi-formal reasoning

| Skill | Trigger phrases | One-line summary | Owner |
|-------|----------------|------------------|-------|
| [`/reason`](../skills/reason/SKILL.md) | "reason about", "is this correct", "will this break", "what does this do" | General-purpose evidence-grounded code reasoning with execution traces and alternative-hypothesis checks. | byfuglien |
| [`/compare-patches`](../skills/compare-patches/SKILL.md) | "compare patches", "are these equivalent", "same behavior" | Determine whether two patches are semantically equivalent by tracing execution through the test suite. | byfuglien |
| [`/locate-fault`](../skills/locate-fault/SKILL.md) | "locate fault", "find the bug", "why does this fail", "root cause" | Locate the root cause of a failing test using 4-phase structured analysis. | byfuglien |
| [`/trace-execution`](../skills/trace-execution/SKILL.md) | "trace execution", "what happens when", "follow the code path", "call graph" | Hypothesis-driven execution-path tracing that builds complete call graphs. | byfuglien |

## Spec management

| Skill | Trigger phrases | One-line summary | Owner |
|-------|----------------|------------------|-------|
| [`/check-regressions`](../skills/check-regressions/SKILL.md) | "check regressions", "did my changes break specs", "re-verify" | Re-verify Dafny specs whose source files have changed. | byfuglien |
| [`/suggest-specs`](../skills/suggest-specs/SKILL.md) | "suggest specs", "what should I verify", "find verification targets" | Analyse code to propose candidate formal specifications. | byfuglien |

## Assurance hierarchy — Layer 4 (impl–spec alignment, semi-formal rationales)

| Skill | Trigger phrases | One-line summary | Owner |
|-------|----------------|------------------|-------|
| [`/invariant-coverage-scaffold`](../skills/invariant-coverage-scaffold/SKILL.md) | "invariant coverage", "coverage gate", "scaffold invariant check" | Generate a pre-commit + CI gate linking invariant docs to property tests (Go/Python/TypeScript in v1). | hellebuyck |
| [`/protected-surface-amend`](../skills/protected-surface-amend/SKILL.md) | "amend protected file", "protected surface amendment", "governance note" | Generate the governance-note amendment block required when editing a protected-surface file. | hellebuyck |
| [`/rationale`](../skills/rationale/SKILL.md) | "build rationale", "is this code adequate", "adequacy argument" | Build a hierarchical claim tree arguing code adequately satisfies its requirements; leaves classified by verification method. See [snapshot](./specs/rationale-2026-05-11.md). | hellebuyck |

## Assurance hierarchy — Layer 5 (spec–intent alignment)

| Skill | Trigger phrases | One-line summary | Owner |
|-------|----------------|------------------|-------|
| [`/intent-check`](../skills/intent-check/SKILL.md) | "intent check", "round-trip check", "spec-intent alignment" | Round-trip informalisation over (invariant prose, covering test, code diff) with FP tracking and a configurable kill criterion (default 30%, see SKILL Configuration). | hellebuyck |
| [`/acceptance-oracle-draft`](../skills/acceptance-oracle-draft/SKILL.md) | "acceptance oracle", "draft scenarios", "user-observable flows" | Draft mechanically-verifiable user-flow acceptance scenarios; rejects subjective criteria. | hellebuyck |

## Assurance hierarchy — Layer 6 (spec completeness)

| Skill | Trigger phrases | One-line summary | Owner |
|-------|----------------|------------------|-------|
| [`/spec-adversary`](../skills/spec-adversary/SKILL.md) | "spec adversary", "what is the spec missing", "propose missing invariants" | Adversarially probe a module's invariant docs for properties the spec is failing to capture. | hellebuyck |

## Onboarding & status (governance)

| Skill | Trigger phrases | One-line summary | Owner |
|-------|----------------|------------------|-------|
| [`/assurance-layer-audit`](../skills/assurance-layer-audit/SKILL.md) | "layer audit", "assurance audit", "hierarchy reach" | Entry-point diagnostic; emits a per-layer reach projection. Run before `/assurance-init`. | hellebuyck |
| [`/assurance-init`](../skills/assurance-init/SKILL.md) | "assurance init", "onboard to assurance hierarchy", "scaffold assurance" | Interactive bootstrap of governance scaffolding (ROADMAP, protected surfaces, invariant docs). | hellebuyck |
| [`/draft-invariants`](../skills/draft-invariants/SKILL.md) | "draft invariants", "invariants for X", "property tests for X", "lock down X against drift" | Spec-aware module invariant drafting; reads prose specs first, falls back to user elicitation, produces `docs/invariants/<module>.md` + property tests. | hellebuyck |
| [`/assurance-status`](../skills/assurance-status/SKILL.md) | "assurance status", "status dashboard" | Onboarding-gated status dashboard surfacing drift, FP rate, kill criteria. | hellebuyck |
| [`/assurance-roadmap-check`](../skills/assurance-roadmap-check/SKILL.md) | "check assurance roadmap", "roadmap drift" | Weekly roadmap drift detector (Status field vs observed state). | hellebuyck |

---

For the skill→layer mapping and the recommended onboarding flow, see [`./assurance-hierarchy.md`](./assurance-hierarchy.md). For the full research treatment behind the 6-layer model, see [`./research/assurance-hierarchy.md`](./research/assurance-hierarchy.md).
