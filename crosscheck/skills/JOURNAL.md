# crosscheck/skills/JOURNAL.md

Journal for skills under the Crosscheck plugin. Decisions that affect a specific skill, or that change the shape of the skill tree, land here. A single skill may earn its own deeper shard later if it accumulates enough to say; for now this is the only journal under `crosscheck/skills/`. Entries newest first. The repo-root [AGENTS.md](../../AGENTS.md) walk-up rule sends agents through this file before touching anything below it.

---

## 2026-05-11 — `/journal-context` skill shipped

**Type:** propagated-discovery
**Touches:** skills/JOURNAL.md (new), skills/journal-context/ (new — SKILL.md, scripts/walk.sh, docs/invariants/journal-context.md, tests/run_tests.sh)
**Why:** Skills under `crosscheck/` have accumulated enough distinct design weight that decisions affecting one skill (or the skill tree's shape) shouldn't have to land alongside Lean-pipeline and MCP-server context at `crosscheck/JOURNAL.md`. The first skill that motivated a dedicated journal here is `/journal-context` itself — its design forced the question of where its own introducing entry should go, which is what the v2 retrospective `§3.8` bootstrap-journal rule is for.
**Links:** [SKILL.md](journal-context/SKILL.md), [invariants](journal-context/docs/invariants/journal-context.md), [v2 §3.4, §5.2](../docs/add/.retrospective/findings-and-methodology-v2.md)

`/journal-context` walks the directory tree from a given path upward to the enclosing git repository root, dumping every `JOURNAL.md` it finds in walk order — deepest first, root last. The walk is deterministic (no LLM, no clock, no random source, no network) and read-only (no filesystem or git mutation). Agents and humans invoke it to load the historical context above a file before substantial design work; the walk-up rule already lives in the root [`AGENTS.md`](../../AGENTS.md), and this is how the rule actually fires without having to invent the loader each time.

The implementation is a bash script (`scripts/walk.sh`) under 100 lines. The invariant doc pins seven properties: walk shape (path-dir up to git toplevel inclusive, no crossing repo boundaries), ordering (deepest-first), determinism (byte-identical output for byte-identical inputs), read-only (no FS or git mutation), symlink handling (walk literal parents, terminate canonically), an explicit empty-case message instead of zero bytes, and a fixed `=== <path-relative-to-repo-root> ===` delimiter between files. Each invariant has a covering bash test under `tests/run_tests.sh` tagged with `# Invariant Ix: <Name>`; 19 assertions, all green at merge.

What's deliberately out for v1:

- **`/journal-lint`** for content-shape checks (date / type / why / links integrity, orphan `Supersedes:` links, contradictions between recent entries). Named in v2 `§3.4` as later enforcement; not needed until journals grow large enough that contradictions become plausible.
- **Pre-commit warning** when a touched directory's `JOURNAL.md` wasn't also edited. v2 `§3.4` layer 3; build when usage shows where drift creeps in.
- **Bidirectional coverage gate** wiring `# Invariant Ix:` test comments to the invariant doc. The convention is already in place; mechanical enforcement is not wired in this repo. If the gap starts mattering in practice, [`/invariant-coverage-scaffold`](invariant-coverage-scaffold/SKILL.md) is the on-shelf path.

This is the only shard under `crosscheck/skills/` for now. It exists because the skill tree has acquired distinct design weight, not because every directory needs a journal — empty shards add noise. The next shard (if one ever earns its place) appears when there's enough to say there too.
