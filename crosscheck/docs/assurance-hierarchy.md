# Assurance hierarchy — onboarding guide

## TL;DR

Crosscheck organises correctness into six layers of assurance. Layers 1–3 prove the implementation is correct against a specification — deterministic, machine-checkable. Layers 4–6 prove the specification is the right specification — deterministic at Layer 4, probabilistic at Layer 5 (~96% accuracy on round-trip checks), best-effort at Layer 6. Confidence degrades as you climb: a verified function at Layer 1 is mathematically proved; a "complete" spec at Layer 6 is as complete as adversarial probing has so far been able to show. For the full treatment — motivation, semantics, and trade-offs — see [`./research/assurance-hierarchy.md`](./research/assurance-hierarchy.md).

## Skill → layer mapping

| Layer | Concern | Crosscheck skill(s) | Confidence | Owner |
|-------|---------|---------------------|------------|-------|
| 1 | Formally verified pure code | [`/spec-iterate`](../skills/spec-iterate/SKILL.md), [`/generate-verified`](../skills/generate-verified/SKILL.md), [`/extract-code`](../skills/extract-code/SKILL.md), [`/lightweight-verify`](../skills/lightweight-verify/SKILL.md) | Deterministic | byfuglien |
| 2 | Compilation correctness | (Not addressed — trust your toolchain) | n/a | n/a |
| 3 | Contract graph verification | (Future — see research doc) | n/a | n/a |
| 4 | Implementation–spec alignment | [`/invariant-coverage-scaffold`](../skills/invariant-coverage-scaffold/SKILL.md), [`/protected-surface-amend`](../skills/protected-surface-amend/SKILL.md), [`/check-regressions`](../skills/check-regressions/SKILL.md) | Deterministic | hellebuyck (4) / byfuglien (regressions) |
| 5 | Specification–intent alignment | [`/intent-check`](../skills/intent-check/SKILL.md), [`/acceptance-oracle-draft`](../skills/acceptance-oracle-draft/SKILL.md) | Probabilistic (~96%) | hellebuyck |
| 6 | Specification completeness | [`/spec-adversary`](../skills/spec-adversary/SKILL.md) | Best-effort | hellebuyck |

## Getting started — onboarding flow

1. Run [`/assurance-layer-audit`](../skills/assurance-layer-audit/SKILL.md) to scope which layers are reachable in your repo (language, tooling, ecosystem maturity).
2. Run [`/assurance-init`](../skills/assurance-init/SKILL.md) to scaffold the governance skeletons (ROADMAP, protected surfaces, skeleton invariant docs for 1–3 modules).
3. Run [`/invariant-coverage-scaffold`](../skills/invariant-coverage-scaffold/SKILL.md) once per supported language to install the pre-commit + CI gate that ties invariant docs to property tests.
4. On every protected-surface change: run [`/protected-surface-amend`](../skills/protected-surface-amend/SKILL.md) to generate the governance-note amendment block; on every invariant-related change: run [`/intent-check`](../skills/intent-check/SKILL.md) to verify the spec→test alignment survived the diff.
5. Run [`/assurance-status`](../skills/assurance-status/SKILL.md) weekly to surface drift, FP rate, and kill-criterion triggers.
6. Run [`/spec-adversary`](../skills/spec-adversary/SKILL.md) on stable modules to probe for invariants the spec is missing — Layer 6 is iterative, not deterministic.

## When to use what

- Writing new business logic that should be provably correct? → [`/spec-iterate`](../skills/spec-iterate/SKILL.md) → [`/generate-verified`](../skills/generate-verified/SKILL.md) → [`/extract-code`](../skills/extract-code/SKILL.md) (byfuglien).
- Code is correct but you suspect the spec might be wrong? → [`/intent-check`](../skills/intent-check/SKILL.md) (hellebuyck).
- Adding a new feature with user-observable behaviour? → [`/acceptance-oracle-draft`](../skills/acceptance-oracle-draft/SKILL.md) to lock down the scenarios upfront (hellebuyck).
- Changing a file that's already governed (e.g. an invariant doc, an agent, a workflow)? → [`/protected-surface-amend`](../skills/protected-surface-amend/SKILL.md) (hellebuyck).
- Module has been stable for a while — what might its spec be missing? → [`/spec-adversary`](../skills/spec-adversary/SKILL.md) (hellebuyck).
- Forgot whether your repo's onboarded? → [`/assurance-status`](../skills/assurance-status/SKILL.md) (hellebuyck).

## Closing pointers

- [`./agents.md`](./agents.md) — which agent owns what.
- [`./skills.md`](./skills.md) — full skill catalogue with trigger phrases.
- [`./research/assurance-hierarchy.md`](./research/assurance-hierarchy.md) — full research treatment.
- [`./research/literature-review.md`](./research/literature-review.md) — academic prior art.
