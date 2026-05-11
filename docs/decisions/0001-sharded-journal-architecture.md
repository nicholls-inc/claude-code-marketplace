# ADR-0001 — Sharded JOURNAL.md architecture

**Date:** 2026-05-11
**Status:** Accepted

## Context

The repo had no shared way to record why things look the way they do across plugins, skills, and the MCP server. PR descriptions decay into git noise; design intent lives in the heads of whoever last touched a file. The first attempt at a shared audit trail — the v1 assurance-driven development stack inside `crosscheck/docs/add/` — over-formalised the surface with ten typed artifacts and five parallel ID schemes and was archived after one retrospective.

The retrospective at `crosscheck/docs/add/.retrospective/findings-and-methodology-v2.md` carries the long-form rationale: what survives substantively (deterministic-only signal computation, PR-merge as the human signal, the five-class diff taxonomy as journal-entry types, `/intent-check` covering what M2 was trying to design from scratch), what was wrong (typed artifact stack, parallel ID schemes, governance metadata leaking into reader-facing docs), and the failure-mode checklist in §5.5.

## Decision

Adopt a sharded `JOURNAL.md` architecture:

- **Co-located narrative files.** `JOURNAL.md` lives at meaningful directory boundaries — repo root, plugin root, deep subdirectories that have enough design weight to earn one. Not every directory. New shards earn existence by accumulating content, not by being scaffolded in advance.
- **Entry shape.** Dated heading, a small frontmatter block (`Type:` / `Touches:` / `Why:` / `Links:`), and one short paragraph of product-voice prose. Newest entries at the top of each file. Append-only after merge; freely editable in-PR.
- **Type values** are the five-class diff taxonomy: `propagated-discovery`, `intent-refinement`, `drift`, `retraction`, `status-transition`. They appear in frontmatter only — never in body prose, never in chat.
- **Root `AGENTS.md`** carries one rule: before non-trivial changes in any directory, walk up to the repo root reading every `JOURNAL.md` you encounter. AGENTS.md is the cross-runtime standard honoured by Claude Code, Codex, Cursor, Copilot, Devin, Windsurf, and Gemini CLI with closest-file precedence.
- **Cross-shard decisions** get an ADR at `docs/decisions/<NNNN>-<slug>.md`. Affected shards' `JOURNAL.md` files carry a one-line entry pointing at the ADR; the ADR holds the longer rationale.

## Consequences

- **Portable.** AGENTS.md works across every major agent harness without per-tool rule files.
- **Cheap to adopt.** This bootstrap is the whole install: one AGENTS.md, three `JOURNAL.md` files, this ADR, an archived v1 stack, a rewritten README. No new tooling required.
- **Unvalidated.** The methodology behind the architecture has not been driven against a real spec session. The retrospective's §5.3 names two test cases — one mature-repo, one greenfield. Both are pending. Treat the shape as a working hypothesis until both succeed.
- **Failure modes are known.** The v2 retrospective §5.5 lists what regression looks like (per-section status fields creeping onto artifacts, governance vocabulary in chat, separate findings reports for adversarial review, agents skipping the walk-up rule, specs being re-edited to stay current). That list is the regression check.
- **Enforcement is layered and additive.** AGENTS.md is the v1 mechanism. Additional layers (deterministic skills, pre-commit warnings, lint passes) can follow if usage shows where drift creeps in — not before.
