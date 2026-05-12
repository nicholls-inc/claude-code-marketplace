# Crosscheck agents

Crosscheck ships three orchestrators. Two are sequential routers — **Byfuglien** (the bruising defenceman whose crosscheck gave the plugin its name) enforces implementation correctness; **Hellebuyck** (the goaltender, last line of defence) interrogates whether the spec is the right spec and stays mechanically enforced. The third, **add-orchestrator**, is a parallel-workflow runner at the methodology layer: it drives the ADD (Assurance-Driven Development) fast path from signed-off spec to approved invariant docs, dispatching subagents per module in parallel and aggregating a batched audit into per-category findings files for human triage.

Use byfuglien to prove code matches a spec. Use hellebuyck to interrogate whether that spec is the right one. Use add-orchestrator when you have a signed-off spec and want it turned into a vetted invariant set ready to hand off to byfuglien (verification) or hellebuyck (ongoing governance).

## Byfuglien — implementation chain

**Role.** Implementation chain orchestrator: formal verification (Dafny) and semi-formal reasoning. Owns Layers 1–3 of the [6-layer assurance hierarchy](./assurance-hierarchy.md) — verified pure code, compilation correctness, and contract graphs. See [`../agents/byfuglien.md`](../agents/byfuglien.md).

**Skills it routes to.**

- Formal verification: [`/spec-iterate`](../skills/spec-iterate/SKILL.md), [`/generate-verified`](../skills/generate-verified/SKILL.md), [`/extract-code`](../skills/extract-code/SKILL.md), [`/lightweight-verify`](../skills/lightweight-verify/SKILL.md)
- Semi-formal reasoning: [`/reason`](../skills/reason/SKILL.md), [`/compare-patches`](../skills/compare-patches/SKILL.md), [`/locate-fault`](../skills/locate-fault/SKILL.md), [`/trace-execution`](../skills/trace-execution/SKILL.md)
- Spec management: [`/check-regressions`](../skills/check-regressions/SKILL.md), [`/suggest-specs`](../skills/suggest-specs/SKILL.md)

**When to invoke.** "I want to verify this code." "Is this patch equivalent to the old one?" "Why does this test fail?" "What does this function do?"

## Hellebuyck — specification chain

**Role.** Specification chain orchestrator: Layers 4–6 of the [assurance hierarchy](./assurance-hierarchy.md) — implementation–spec alignment, spec–intent alignment, and spec completeness — plus the governance scaffolding and status reporting that keeps the spec→test→code chain mechanically enforced. Last line of defence when formal proof is clean but the spec might not capture what you actually meant. See [`../agents/hellebuyck.md`](../agents/hellebuyck.md).

**Skills it routes to.**

- Onboarding & status: [`/assurance-layer-audit`](../skills/assurance-layer-audit/SKILL.md), [`/assurance-init`](../skills/assurance-init/SKILL.md), [`/assurance-status`](../skills/assurance-status/SKILL.md), [`/assurance-roadmap-check`](../skills/assurance-roadmap-check/SKILL.md)
- Layer 4 (impl–spec alignment): [`/invariant-coverage-scaffold`](../skills/invariant-coverage-scaffold/SKILL.md), [`/protected-surface-amend`](../skills/protected-surface-amend/SKILL.md), [`/rationale`](../skills/rationale/SKILL.md) (semi-formal rationales — see [snapshot](./specs/rationale-2026-05-11.md))
- Layer 5 (spec–intent alignment): [`/intent-check`](../skills/intent-check/SKILL.md), [`/acceptance-oracle-draft`](../skills/acceptance-oracle-draft/SKILL.md)
- Layer 6 (spec completeness): [`/spec-adversary`](../skills/spec-adversary/SKILL.md)

**When to invoke.** "Onboard this repo to the assurance hierarchy." "What is the spec missing?" "Did my changes break any invariants?" "Is the spec actually capturing what we mean?" "Show me the assurance status of this repo."

## add-orchestrator — ADD methodology workflow runner

**Role.** Workflow runner at the methodology layer above the byfuglien/hellebuyck partition. Drives the ADD spec-driven fast path: signed-off spec → bulk-drafted invariants → batched audit → user-triaged findings → approved invariant docs ready for implementation. **Parallel-workflow-runner pattern** — dispatches N subagents per module concurrently in a single assistant turn, runs a parallel audit, aggregates findings across all subagents into per-category files for batched human triage. See [`../agents/add-orchestrator.md`](../agents/add-orchestrator.md).

**Skills it composes** (does not own — it coordinates the workflow that uses these skills, all of which remain owned by hellebuyck):

- Bulk drafting: [`/draft-invariants`](../skills/draft-invariants/SKILL.md) (in marker-aware mode — orchestrator-deferred red-pen)
- Audit: [`/audit-spec-coverage`](../skills/audit-spec-coverage/SKILL.md), [`/audit-invariant-consistency`](../skills/audit-invariant-consistency/SKILL.md)
- Spec amendment: [`/protected-surface-amend`](../skills/protected-surface-amend/SKILL.md) (when a finding's accept-path is "amend spec")

**Hand-off contracts** (closing-recommendation only — never auto-chains):

- byfuglien for verification-chain follow-on: `/lightweight-verify` for IO/concurrency-heavy modules (the dominant case); `/spec-iterate` → `/generate-verified` → `/extract-code` for Dafny-suitable; Lean pipeline for tractable-input modules
- hellebuyck for ongoing governance: `/invariant-coverage-scaffold` (gated on `/assurance-init`), `/spec-adversary` on coverage-thinnest modules, `/intent-check` per protected-surface PR

**When to invoke.** "Drive the ADD fast path on this spec." "Bulk-draft invariants from `<spec-path>`." "Spec to invariants." Discoverability note: the trigger surface is workflow-shaped and disjoint from `awesome-copilot/agents/project-scaffold.md`'s generic project-scaffolding triggers.

## Handoff seam

Byfuglien proves the code is correct against the spec. Hellebuyck checks the spec is the right spec and that the spec→test→code chain stays mechanically enforced. add-orchestrator turns a spec into the invariant set that hellebuyck will govern and that byfuglien will eventually verify against. Concrete escalation patterns:

- Byfuglien proves your sort is correct — hellebuyck's [`/intent-check`](../skills/intent-check/SKILL.md) confirms the spec actually says "sorted permutation" and not just "monotonically increasing".
- Byfuglien handles a CRUD endpoint's code reasoning — hellebuyck's [`/rationale`](../skills/rationale/SKILL.md) builds the adequacy argument, and [`/spec-adversary`](../skills/spec-adversary/SKILL.md) probes for missing invariants the rationale didn't anticipate.
- A protected-surface file is about to change — use byfuglien for code reasoning about the diff, then hellebuyck's [`/protected-surface-amend`](../skills/protected-surface-amend/SKILL.md) to generate the governance note that ships in the same PR.

## Invocation

```
Use the byfuglien agent to verify your bug fix against the spec
Use the hellebuyck agent to check what invariants this module is missing
Use the add-orchestrator agent to drive ADD on this spec
```

## See also

- [`./skills.md`](./skills.md) — full skill catalogue
- [`./assurance-hierarchy.md`](./assurance-hierarchy.md) — the 6-layer model and how the two agents partition it
