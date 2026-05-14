# crosscheck/JOURNAL.md

Journal for the Crosscheck plugin. Decisions that affect skills, agents, the MCP server, or the plugin's overall shape land here. Narrower work (a specific skill, a Docker image, the Lean pipeline) may earn its own deeper shard when there's enough to say; for now this is the only journal under `crosscheck/`. Entries newest first. The repo-root [AGENTS.md](../AGENTS.md) walk-up rule sends agents through this file before touching anything below it.

---

## 2026-05-14 — release pipeline: drain the 2.4.0 → 2.5.0 backlog and harden the commit convention

**Type:** release-process / governance
**Touches:** [CLAUDE.md](../CLAUDE.md) (commit conventions), [.husky/commit-msg](../.husky/commit-msg) (enforcement), this entry triggers `Release-As: 2.5.0` for the crosscheck component.
**Why:** Sixteen commits landed after the 2.4.0 release without producing a release PR. The Release workflow ran every time and reported `No user facing commits found since c62ea3c... - skipping`. Cause: every one of those commits was `refactor(crosscheck): …` — release-please's default semver rules only treat `feat:` and `fix:` (and `feat!:` / `BREAKING CHANGE`) as user-facing. `refactor:` is silently ignored for versioning. The previous convention allowed `refactor:` for behavioral changes to `SKILL.md` / `agents/*.md`, which is exactly the failure mode that produced the backlog.

The fix is two-track. **Convention:** behavior changes to behavioral artifacts must be `feat:` (new behavior) or `fix:` (corrective). `refactor:` is reserved for structural changes that genuinely do not alter behavior — and when they touch a behavioral artifact, they must be split into a separate commit that does not. **Enforcement:** `.husky/commit-msg` now blocks `refactor:` on behavioral artifacts in the same way it already blocks `docs:`, with an error message that names release-please as the reason. Without enforcement, the convention is just prose and the same drift recurs.

This entry is also the carrier for the `Release-As: 2.5.0` trailer on the merging PR — it touches `crosscheck/` so release-please attributes the trailer to the crosscheck component, and it documents the rationale in the place a future maintainer will look when the next release misbehaves.

---

## 2026-05-11 — /rationale: FORMAL routing Layer 1 vs Layer 4

**Type:** propagated-discovery
**Touches:** skills/rationale/SKILL.md (Steps 3, 4, 6)
**Why:** Lands the snapshot's §4 FORMAL design into the operational prompt. Before this PR, every FORMAL leaf was routed through a single discharge path — *"draft Dafny spec → offer `/spec-iterate`"* — with no distinction between code that's a Layer 1 candidate (pure, ships as Dafny extraction) and code that needs Layer 4 (effectful/networked/concurrent, Lean as DRT oracle, production code stays as-is). The snapshot named both routes and the picking heuristic; SKILL.md was still pointing only at Layer 1.
**Links:** [snapshot §4](docs/specs/rationale-2026-05-11.md), parent snapshot PR (#169), C0 branch PR (#173)

Step 3's classification table widens the `[FORMAL]` verification-method cell from a single Dafny route into two layer-tagged routes (Layer 1 Dafny, Layer 4 Lean pipeline). Classification guidelines gain a *FORMAL routing* line that names the purity/effect-profile heuristic — pure-functional shape → Layer 1, effectful/networked/concurrent/shipping-floats → Layer 4.

Step 4's `[FORMAL]` section is restructured around the two discharge routes. *Layer 1* keeps the existing draft-Dafny-spec mechanics. *Layer 4* names the full Lean pipeline (`/lean-spec` → `/lean-impl` → `/correspondence-review` → `/drt-oracle`), is explicit that the Lean model is **not** shipped, and marks the leaf verified only after `/drt-oracle` reports clean. A new *Picking the route* paragraph encodes the heuristic and instructs the skill to ask the user when ambiguous rather than guess.

Step 6's verification-checklist `[FORMAL]` bullet widens accordingly — routing-by-profile is now an explicit gate the skill self-checks before delivery.

Step 5's worked example is intentionally untouched: the sort case is a clean Layer 1 example, so the existing verification results (`dafny_verify` discharge) still match. A worked Layer 4 example would be a follow-up addition once a candidate effectful module is in hand; not part of this PR.

---

## 2026-05-11 — /rationale: promote trust boundaries to C0 top-level branch

**Type:** propagated-discovery
**Touches:** skills/rationale/SKILL.md (Steps 2, 5, 6)
**Why:** Lands the snapshot's §2 design decision into the operational prompt. Before this PR, trust boundaries were a single footnote in the final verification checklist (*"Trust boundaries noted (Dafny limitations, extern methods, float precision)"*), buried below the structural soundness check. The snapshot reframed them as a first-class C0 branch because they bound what every other leaf can verify — a verified `sort.py` proof is meaningless if its comparison operator is `{:extern}`. SKILL.md was still pointing the other way.
**Links:** [snapshot §2](docs/specs/rationale-2026-05-11.md), parent snapshot PR (#169)

Step 2 gains a C0 trust-boundary branch alongside C1/C2/C3, with three generic leaf templates (external dependencies enumerated; domain limitations documented; trust assumptions stated). A new *Why C0 is first-class* paragraph names the conditioning relationship — every downstream claim is conditional on the C0 leaves, and that conditioning is now visible in the tree rather than implicit. Step 5's worked sort example gains two C0 leaves (no extern/IO/network as STATIC; `<=` totality + transitivity as SEMANTIC), and the summary table updates accordingly. Step 6's standalone *"Trust boundaries noted"* footnote bullet is replaced by *"C0 trust-boundary branch enumerated"* — making the check structural rather than ad-hoc.

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
