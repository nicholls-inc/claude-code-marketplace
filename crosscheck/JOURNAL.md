# crosscheck/JOURNAL.md

Journal for the Crosscheck plugin. Decisions that affect skills, agents, the MCP server, or the plugin's overall shape land here. Narrower work (a specific skill, a Docker image, the Lean pipeline) may earn its own deeper shard when there's enough to say; for now this is the only journal under `crosscheck/`. Entries newest first. The repo-root [AGENTS.md](../AGENTS.md) walk-up rule sends agents through this file before touching anything below it.

---

## 2026-05-11 — v1 assurance-driven development stack archived [ADR-0001]

**Type:** retraction
**Touches:** docs/add/ (v1 archived, v3 starts), .assurance/ (one v1 output archived)
**Why:** The v1 design-doc stack inside `docs/add/` reproduced the failure mode it was meant to prevent, so we pulled it after one retrospective.
**Links:** [ADR-0001](../docs/decisions/0001-sharded-journal-architecture.md), [v2 retrospective](docs/add/.retrospective/findings-and-methodology-v2.md)

The v1 install was docs-only — no skill behaviour changed, no MCP-server code changed, nothing in `agents/` or `skills/` was rewired. So pulling it is cheap and contained to documentation. The retrospective under `docs/add/.retrospective/` is the place to read about what happened and where the design is heading; the archive under `.retrospective/v1-archive/` is for anyone hitting a stale link. The plugin runtime is otherwise unchanged.
