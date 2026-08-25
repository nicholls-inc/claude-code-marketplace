# Assurance Roadmap

## Purpose

This directory tracks the execution of the 6-layer assurance hierarchy as
applied pragmatically to this repository. It exists so the plan survives across
long timeframes without depending on conversation context or ephemeral plan
files. Each numbered item below is the source of truth for its own scope,
acceptance criteria, and kill criteria; larger items get a standalone doc under
`immediate/`, `next/`, `medium-term/`, or `aspirational/`.

This roadmap is deliberately minimal. Crosscheck dogfoods the tooling it ships,
so the first item is the framework by which every later item will be built.

## Strategic context

The long-term goal is deterministic AI-driven software development: formally
verified kernels for critical pure logic, contract graphs for integration
boundaries, and spec-intent alignment checks for everything else. Crosscheck
already implements the layers; what it has lacked is a documented development
framework that applies them to itself, and human gates a newcomer can act on
without asking anyone.

Current projection across the six layers: `TODO: fill from
/assurance-layer-audit output`.

## Dual-track enforcement principle

> Every deterministic assurance check added by this roadmap must produce two
> enforcement points:
>
> 1. **Pre-commit hook** — cheap, fast (< 5 s), lightweight. Blocks the commit
>    locally. Must emit a human-readable error that includes the exact command
>    to resolve the failure. LLMs acting as coding agents will see this error
>    and must be able to fix it by following the instruction without human
>    intervention.
> 2. **CI job** — slower, more comprehensive, runs on every PR regardless of
>    how the code change was authored. Must also emit actionable fix
>    instructions. Can perform work too expensive for pre-commit (full test
>    suites, container-based verification).
>
> Workflow phases are **not a substitute** for either enforcement point.
>
> Pre-commit hooks are fast attestation checks only — they must never invoke
> LLMs or run slow test suites. Heavy verification lives in CI and in dedicated
> binaries that the pre-commit hook verifies were run.

## Horizon index

### Immediate (start now)

| # | Item | Cost | Doc |
|---|---|---|---|
| PB-1 | Adopt the AI-native SDLC playbook as the development framework and make every human gate self-explanatory | M | [`DEVELOPMENT-FRAMEWORK.md`](DEVELOPMENT-FRAMEWORK.md) |

**PB-1 — Status: In progress.** Scope: adopt the playbook artefact chain
(intent → spec → plan → diff + tests → PR with review findings → incident
record + eval) as this repository's own development framework, with a tier map,
a protected-surface hook, `REVIEW.md`, and the four enforcing CI workflows; and
give every human gate a plain-language explainer under `docs/gates/` reached
from a fixed three-sentence gate message. Acceptance: a new contributor can
trace a change from `intent/<slug>.md` to a merged PR using
`DEVELOPMENT-FRAMEWORK.md` alone, and every gate in the inventory names its
explainer. Later artefacts cite PB-1 as their governing roadmap item.

### Next (4–8 weeks)

| # | Item | Cost | Doc |
|---|---|---|---|
| TODO | TODO | TODO | TODO |

### Medium-term (2–3 months)

| # | Item | Cost | Doc |
|---|---|---|---|
| TODO | TODO | TODO | TODO |

### Aspirational (scope and commit later)

| # | Item | Cost | Doc |
|---|---|---|---|
| TODO | TODO | TODO | TODO |

## Kill criteria

Stop and re-plan the entire roadmap if any of the following becomes true.

- The formal-verification pipeline remains unusable for a first kernel after a
  sustained attempt, so no verified kernel is reachable.
- The spec-alignment false-positive rate breaches the 30% ceiling enforced by
  `/intent-check`, meaning the alignment signal costs more than it earns.
- No immediate-horizon item merges within four weeks, meaning the roadmap is
  aspirational in practice.

## How to use this directory

Every item carries a `Status:` field: `Not started`, `In progress`, `Blocked`
(with a `Blocker:` line), `Done` (cross-referencing the landing PR), or
`Deferred` (with a required `Reason:` line and a link to any superseding item).
Items are never deleted — only marked `Deferred`. IDs are stable: once an item
has an ID, that ID is never reused or renumbered, because other artefacts cite
it as their governing roadmap item.

## References

- [`DEVELOPMENT-FRAMEWORK.md`](DEVELOPMENT-FRAMEWORK.md) — the artefact chain
  and its triggers.
- [`TIER-LAYER-MAP.md`](TIER-LAYER-MAP.md) — change tiers and required
  artefacts.
- `../../REVIEW.md` — review passes and finding severities.
