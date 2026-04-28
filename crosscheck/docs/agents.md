# Crosscheck agents

Crosscheck ships two complementary orchestrators, both named after Winnipeg Jets. **Byfuglien** — the bruising defenceman whose crosscheck gave the plugin its name — enforces implementation correctness. **Hellebuyck** — the goaltender, last line of defence — takes over when formal proof is clean but the spec itself might be the wrong spec. Use byfuglien to prove code matches a spec; use hellebuyck to interrogate whether that spec is the right one and stays mechanically enforced.

## Byfuglien — implementation chain

**Role.** Implementation chain orchestrator: formal verification (Dafny) and semi-formal reasoning. Owns Layers 1–3 of the [6-layer assurance hierarchy](./assurance-hierarchy.md) — verified pure code, compilation correctness, and contract graphs. See [`../agents/byfuglien.md`](../agents/byfuglien.md).

**Skills it routes to.**

- Formal verification: [`/spec-iterate`](../skills/spec-iterate/SKILL.md), [`/generate-verified`](../skills/generate-verified/SKILL.md), [`/extract-code`](../skills/extract-code/SKILL.md), [`/lightweight-verify`](../skills/lightweight-verify/SKILL.md)
- Semi-formal reasoning: [`/reason`](../skills/reason/SKILL.md), [`/compare-patches`](../skills/compare-patches/SKILL.md), [`/locate-fault`](../skills/locate-fault/SKILL.md), [`/trace-execution`](../skills/trace-execution/SKILL.md)
- Spec management: [`/check-regressions`](../skills/check-regressions/SKILL.md), [`/suggest-specs`](../skills/suggest-specs/SKILL.md), [`/rationale`](../skills/rationale/SKILL.md)

**When to invoke.** "I want to verify this code." "Is this patch equivalent to the old one?" "Why does this test fail?" "What does this function do?"

## Hellebuyck — specification chain

**Role.** Specification chain orchestrator: Layers 4–6 of the [assurance hierarchy](./assurance-hierarchy.md) — implementation–spec alignment, spec–intent alignment, and spec completeness — plus the governance scaffolding and status reporting that keeps the spec→test→code chain mechanically enforced. Last line of defence when formal proof is clean but the spec might not capture what you actually meant. See [`../agents/hellebuyck.md`](../agents/hellebuyck.md).

**Skills it routes to.**

- Onboarding & status: [`/assurance-layer-audit`](../skills/assurance-layer-audit/SKILL.md), [`/assurance-init`](../skills/assurance-init/SKILL.md), [`/assurance-status`](../skills/assurance-status/SKILL.md), [`/assurance-roadmap-check`](../skills/assurance-roadmap-check/SKILL.md)
- Layer 4 (impl–spec alignment): [`/invariant-coverage-scaffold`](../skills/invariant-coverage-scaffold/SKILL.md), [`/protected-surface-amend`](../skills/protected-surface-amend/SKILL.md)
- Layer 5 (spec–intent alignment): [`/intent-check`](../skills/intent-check/SKILL.md), [`/acceptance-oracle-draft`](../skills/acceptance-oracle-draft/SKILL.md)
- Layer 6 (spec completeness): [`/spec-adversary`](../skills/spec-adversary/SKILL.md)

**When to invoke.** "Onboard this repo to the assurance hierarchy." "What is the spec missing?" "Did my changes break any invariants?" "Is the spec actually capturing what we mean?" "Show me the assurance status of this repo."

## Handoff seam

Byfuglien proves the code is correct against the spec. Hellebuyck checks the spec is the right spec and that the spec→test→code chain stays mechanically enforced. Concrete escalation patterns:

- Byfuglien proves your sort is correct — hellebuyck's [`/intent-check`](../skills/intent-check/SKILL.md) confirms the spec actually says "sorted permutation" and not just "monotonically increasing".
- Byfuglien's [`/rationale`](../skills/rationale/SKILL.md) produces an adequacy argument for a CRUD endpoint — hellebuyck's [`/spec-adversary`](../skills/spec-adversary/SKILL.md) probes for missing invariants the rationale didn't anticipate.
- A protected-surface file is about to change — use byfuglien for code reasoning about the diff, then hellebuyck's [`/protected-surface-amend`](../skills/protected-surface-amend/SKILL.md) to generate the governance note that ships in the same PR.

## Invocation

```
Use the byfuglien agent to verify your bug fix against the spec
Use the hellebuyck agent to check what invariants this module is missing
```

## See also

- [`./skills.md`](./skills.md) — full skill catalogue
- [`./assurance-hierarchy.md`](./assurance-hierarchy.md) — the 6-layer model and how the two agents partition it
