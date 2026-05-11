# crosscheck/JOURNAL.md

Journal for the Crosscheck plugin. Decisions that affect skills, agents, the MCP server, or the plugin's overall shape land here. Narrower work (a specific skill, a Docker image, the Lean pipeline) may earn its own deeper shard when there's enough to say; for now this is the only journal under `crosscheck/`. Entries newest first. The repo-root [AGENTS.md](../AGENTS.md) walk-up rule sends agents through this file before touching anything below it.

---

## 2026-05-11 — /rationale snapshot: defer STATIC citation post-process

**Type:** correction
**Touches:** docs/specs/rationale-2026-05-11.md (§4 STATIC, §8 deferred-until-field-evidence)
**Why:** No fabricated `STATIC` citations observed in `/rationale` invocations to date. The snapshot's §4 declared the deterministic read-and-string-search post-process as part of the design; this is premature surface area against an unobserved failure mode. §8 already establishes the right pattern for this exact reflex — deferred until field evidence — applied there to tree completeness.
**Links:** [snapshot](docs/specs/rationale-2026-05-11.md), parent snapshot PR (#169), cascade PR (#170)

The §4 STATIC paragraph drops the post-process language and adds a one-line pointer to §8. §8 gains a second deferred concern ("STATIC citation honesty") with the same shape as the existing tree-completeness block — observed-failure trigger, on-shelf option (read-and-string-search), explicit cost framing. Strict v2 reading would make this a new dated snapshot superseding 2026-05-11; chose the lighter-touch amendment within the same week as merge, since this is correction of premature scope rather than design evolution. Downstream effect: the planned SKILL.md catch-up PR series drops from three to two (C0 trust-boundary branch and FORMAL Layer 1 vs Layer 4 routing remain).

---

## 2026-05-11 — /rationale Layer-4 docs cascade + status flip

**Type:** propagated-discovery
**Touches:** docs/specs/rationale-2026-05-11.md (status field), agents/byfuglien.md, agents/hellebuyck.md, docs/agents.md, docs/skills.md, docs/assurance-hierarchy.md
**Why:** Downstream from the snapshot merged at 3da376d. The snapshot reassigned `/rationale` from byfuglien (Layers 1–3) to hellebuyck (Layer 4 — semi-formal rationales); the agent pages, skill catalogue, agent overview, and assurance-hierarchy mapping all still pointed the other way. Also flips the snapshot's frontmatter from `Status: Draft` to `Status: Snapshot` per v2 methodology (`docs/add/.retrospective/findings-and-methodology-v2.md:218-223`) — Status: Snapshot = committed, and merge is the ratification signal.
**Links:** [snapshot](docs/specs/rationale-2026-05-11.md), [v2 methodology](docs/add/.retrospective/findings-and-methodology-v2.md), parent PR (#169)

Single PR, low risk. `/rationale` removed from byfuglien's skill tables, classification, skill-readme list, and Phase 4 quality gates; added to hellebuyck's new "Adequacy (Layer 4 — semi-formal rationales)" subsection with matching classification, skill-readme entry, and validate-output gates (claim tree soundness, classification accuracy, actionable output). `docs/agents.md` moves `/rationale` from byfuglien's "Spec management" bullet to hellebuyck's "Layer 4 (impl–spec alignment)" bullet and rewords the handoff seam — the `/rationale` → `/spec-adversary` chain is now intra-hellebuyck rather than byfuglien→hellebuyck; the seam stands. `docs/skills.md` reflows the "Spec management & adequacy" section into "Spec management" (byfuglien) and folds `/rationale` into the existing "Layer 4 (impl–spec alignment, semi-formal rationales)" section. `docs/assurance-hierarchy.md` extends the Layer 4 row with `/rationale` and adds a "When to use what" pointer. Snapshot text is left untouched apart from the status flip — its byfuglien.md:147 cross-reference becomes stale, but the snapshot is a dated artefact (v2 §3.3); stale line numbers are expected, content stands. SKILL.md catch-up (C0 trust branch, FORMAL Layer 1 vs Layer 4 routing, STATIC citation post-process) ships separately per the snapshot's own callout.

---

## 2026-05-11 — /rationale design-intent snapshot

**Type:** propagated-discovery
**Touches:** docs/specs/rationale-2026-05-11.md (new), docs/specs/ (new directory)
**Why:** Driving the v2 retrospective methodology against a real skill. `/rationale` shipped via SKILL.md a while back; the design intent behind it had never been written down. Snapshotting it now so future drift can be checked against something.
**Links:** [spec snapshot](docs/specs/rationale-2026-05-11.md), [v2 retrospective](docs/add/.retrospective/findings-and-methodology-v2.md), [SKILL.md](skills/rationale/SKILL.md)

First snapshot of `/rationale`'s design intent — what the skill is for, the seams with `/spec-iterate`, the Lean pipeline, `/spec-adversary`, `/intent-check`, and `/assurance-probe`, and what's deliberately not in scope. Seven design decisions land in this snapshot: (1) the skill sits at Layer 4 (semi-formal rationales) and moves from byfuglien to hellebuyck ownership; (2) FORMAL claims split into two discharge routes — Layer 1 → `/spec-iterate` for pure code shipping to production, Layer 4 → `/lean-spec`/`/lean-impl`/`/drt-oracle` for impl-vs-model verification (Lean is the model used as a DRT oracle, not shipped code); (3) STATIC citations get a fast deterministic post-process check (no LLM in the loop) that catches fabricated `file:line` references; (4) trust boundaries promote from a final-checklist bullet to a `C0` top-level branch so they're part of the argument structure rather than a footnote; (5) no on-disk persistence yet; (6) no in-skill chaining — orchestrators compose skills; (7) multi-target applicability is design intent — the goal-structured argument plus four-class leaf taxonomy are target-agnostic, but decomposition templates and FORMAL discharge routes are target-specific and extend together (implementation verification works today; spec analysis, acceptance-scenario adequacy, and others are downstream extensions). One question is *deferred until field evidence*: tree completeness — `/rationale-adversary` (sibling skill) and an in-skill completeness pass are both on the shelf; neither lands until invocations show the gap matters in practice. The Layer-4 reassignment implies cascading updates to `agents.md`, `skills.md`, `byfuglien.md`, and `hellebuyck.md` — downstream work, not part of this PR. First thing under `crosscheck/docs/specs/`, which lands the v2 §3.3 dated-snapshot pattern here as a side-effect.

---

## 2026-05-11 — v1 assurance-driven development stack archived [ADR-0001]

**Type:** retraction
**Touches:** docs/add/ (v1 archived, v3 starts), .assurance/ (one v1 output archived)
**Why:** The v1 design-doc stack inside `docs/add/` reproduced the failure mode it was meant to prevent, so we pulled it after one retrospective.
**Links:** [ADR-0001](../docs/decisions/0001-sharded-journal-architecture.md), [v2 retrospective](docs/add/.retrospective/findings-and-methodology-v2.md)

The v1 install was docs-only — no skill behaviour changed, no MCP-server code changed, nothing in `agents/` or `skills/` was rewired. So pulling it is cheap and contained to documentation. The retrospective under `docs/add/.retrospective/` is the place to read about what happened and where the design is heading; the archive under `.retrospective/v1-archive/` is for anyone hitting a stale link. The plugin runtime is otherwise unchanged.
