# Phase 2 Seam-Validation Report — ADD into Crosscheck

**Author:** Claude Code agent (retroactive seam-validation pass)
**Date (UTC):** 2026-05-09T18:24:03Z
**Inputs read in this pass:**
- `crosscheck/README.md`
- `crosscheck/docs/research/assurance-hierarchy.md`
- `crosscheck/docs/research/literature-review.md`
- `crosscheck/docs/skills.md`, `crosscheck/docs/agents.md`
- `crosscheck/agents/byfuglien.md`, `crosscheck/agents/hellebuyck.md`
- `crosscheck/skills/{assurance-layer-audit,assurance-init,intent-check,spec-adversary,acceptance-oracle-draft,spec-iterate}/SKILL.md`

**Status:** Drafted — input to the human's adjudication; not itself an attestation.

**Why this report exists.** The original task brief instructed the agent to read the existing Crosscheck artifacts (README, research, catalogues, agents, six SKILL.md files) so it would understand the seam between ADD and the existing plugin. The agent did not read them in turn 1, having followed the strict reading of `acceptance.md` § Step 1 ("the cold read tests whether the seed is self-contained"). The user surfaced the miss; this pass corrects it.

The pass classifies findings into three buckets, per the user's directive on the recommended option:

- **Bucket A** — findings against Drafted Phase 1 work (M1–M6 functional specs and `behavioral.md`). These are agent-amendable on the next exchange after sign-off.
- **Bucket B** — findings against the Attested seed (intent.md, ADRs, architectural.md). These require either a re-drafting event triggered by this upstream discovery, or a supersession ADR.
- **Bucket C** — resolutions to the ~25 open questions surfaced across the six functional specs. Several are now mechanically answerable from the artifacts.

---

## File map and human/agent boundary — retroactive confirmation

Closing the "confirm out loud" miss from turn 1.

**File map I have now internalised:**

- `crosscheck/docs/add/` — the ADD seed (Attested as of commit `f7b69c0`): methodology, glossary, intent (IC1–IC11), five ADRs, architectural spec (S1–S8 + S2.5), acceptance.
- `crosscheck/docs/add/specs/` — agent-authored Phase 1 spec stack: `behavioral.md` (Drafted) and `modules/M1..M6.md` (Drafted).
- `crosscheck/.assurance/` — Phase 2 outputs: back-translation, comparison, and now this seam-validation report.
- `crosscheck/agents/{byfuglien,hellebuyck}.md` — pre-existing orchestrator agents. ADD adds a third (the Auditor) per ADR-003; the slug is undecided.
- `crosscheck/skills/<26 skills>/` — the existing skill catalogue. ADD adds four greenfield skills (S2.1–S2.4) and adapts a small number of existing ones additively (S3.1–S3.5; the seam-validation surfaces three more).
- `crosscheck/docs/{skills.md,agents.md}` — catalogues that S7 of the architectural spec updates.
- `crosscheck/docs/research/{assurance-hierarchy,literature-review}.md` — research grounding the methodology cites.
- `crosscheck/docs/invariants/` — pre-existing bootstrap-mode invariant docs (currently scaffolded by `/assurance-init`).

**Human/agent boundary I have now internalised:**

- Humans author intent, glossary, methodology, ADRs, the architectural spec (S1–S8), the acceptance doc — and attest at phase boundaries.
- Agents draft behavioral, functional, SKILL.md, agent definitions, the deterministic instrumentation tool, hooks/CI gates, and documentation updates — gated on human attestation.
- Drafted artifacts are amendable by the agent with the human's explicit confirmation in the same exchange (per `crosscheck/docs/add/README.md` boundary rule).
- Attested artifacts cannot be modified in place. Re-drafting (triggered by upstream change) or supersession (via a new ADR) are the two legal paths.
- The Auditor agent has read-only access to artifacts under audit; only `docs/add/audit/` is its write surface (per ADR-005 § Authorship constraint).

---

## Bucket A — findings against Drafted Phase 1 work

These are concrete defects in M1–M6 / `behavioral.md`. Each is amendable as a propagated-discovery commit on agent's next exchange after your sign-off.

### A-1 (severe) — M2/F2.4 uses the wrong env vars and the wrong rolling window

**My draft:** F2.4 uses `CROSSCHECK_INTENT_CHECK_PROSE_FP_THRESHOLD` and a "rolling 30-attestation window."

**Existing convention** (per `skills/intent-check/SKILL.md` § Configuration and `docs/research/assurance-hierarchy.md` § Calibration of Layer-5 thresholds):
- `CROSSCHECK_FP_TRIPPED_THRESHOLD` (default `0.30`) — tripped threshold.
- `CROSSCHECK_FP_AT_RISK_THRESHOLD` (default `0.20`) — escalation threshold (a concept I omitted entirely).
- `CROSSCHECK_FP_WINDOW_DAYS` (default `14`) — rolling-window length **in days, not attestations**, with **n ≥ 3 minimum sample size**.
- The existing `/intent-check` SKILL.md explicitly states: *"any consumer that reads `.assurance/intent-check-fp-tracker.csv` (e.g. `/assurance-status`) must use the same env vars so the user sees the same rate everywhere."*

**Fix:** Adopt the three-env-var pattern. Use 14-day rolling window with `n ≥ 3` minimum. Add the `AT RISK` reporting band. Use the SAME env vars (the prose variant should not introduce a parallel set; the existing skill's discipline says one set, both consumers).

### A-2 (severe) — M2/F2.4's FP-tracker schema doesn't match the existing one

**My draft:** Columns `recorded_at, finding_id, spurious, justification, rolling_window_position`.

**Existing schema** (per `skills/intent-check/SKILL.md` § Step 5 and `references/fp-tracker-schema.md`):
- `date,invariant_touched,phase_verdict,human_verdict`
- `human_verdict` legal values: `genuine | genuine-planted | partial | spurious` — empty is legal at skill-run time and is filled in later by a human reviewer.

**Fix:** Adopt the existing four-column schema. For the prose variant, repurpose `invariant_touched` to identify the intent-doc-or-section under check. The schema must be **byte-identical** so cross-consumer rates are computed identically.

### A-3 (severe) — M2/F2.3 misses /intent-check's two-section back-translator structure

**My draft:** F2.3 specifies a single prose back-translation with a build-time blindness check.

**Existing /intent-check** requires the back-translator to emit:
- **Section 1: Behavioural guarantees** — plain-text paragraph.
- **Section 2: Design rationale comments** — every 3+ line comment block AND every single-line comment containing rationale markers (`because`, `since`, `artefact`, `workaround`, `zeroed`, `intentional`, `skipping`, `ignore`), quoted verbatim with file:line references. If none exist, the section says `None.`.
- Both sections mandatory; missing/empty (other than `None.`) → re-invoke once; fail on second malformed.

**Fix:** Mirror the two-section structure in F2.3. The prose-vs-prose variant doesn't have code comments to extract, but the discipline of structured output still applies — the back-translation's structure should be: Section 1 (system the spec describes) + Section 2 (any explicit "out of scope" / "not covered" markers in the spec stack).

### A-4 (severe) — M2 misses the diff-checker's mandatory carve-out scan

**Existing /intent-check** § Step 3: *"The prompt forces the diff-checker to do a **mandatory Step 1 carve-out scan** before evaluating any gap. The scan looks for scope markers — `Not covered`, `caller-responsibility`, `precondition`, `aspirational`, `known violation`, `privileged`, `exempt`, `out of scope`, `does not apply`."*

**My draft:** F2.3/F2.4 don't reference this scan.

**Fix:** Add a pre-comparison carve-out scan to F2.3's compare-prompt contract. The intent-doc's negative-space (`N1`–`N8`) is a natural carve-out source for the prose variant.

### A-5 (severe) — M2 misses the diff-checker output schema and semantic validation

**Existing /intent-check** § Step 4 has fail-closed hardening:
1. Contradictory output (`match=true` but `mismatch_reason` non-empty/non-trivial) → flip to fail with `confidence_pct=40`, `confidence_basis=spec-ambiguous`, `mismatch_category=missing_property`.
2. Truncated reason (`match=false` and `len(strip(mismatch_reason)) < 20`) → reject as malformed; ask user to re-run.

The diff-checker JSON schema is also fixed:
```json
{ "match": ..., "mismatch_reason": ..., "mismatch_category": ..., "confidence_pct": ..., "confidence_basis": ... }
```

**My draft:** No semantic validation in M2; no diff-checker output schema beyond `ComparisonReport`.

**Fix:** Add the JSON schema and the two fail-closed rules to F2.3 / F2.4.

### A-6 (medium) — M2 misses the attestation hash mechanism

**Existing /intent-check** § Step 6 produces `.assurance/intent-check-attestation.json`:
```json
{
  "protected_files": [...sorted...],
  "content_hash": "<SHA-256 hex of concatenated bytes>",
  "verdict": "pass" | "fail",
  "checked_at": "<RFC3339>",
  "pipeline_output": { "back_translation": "...", "diff_result": {...} }
}
```

The companion pre-commit hook recomputes the hash and rejects the commit if attestation is absent, stale, or `verdict != pass`.

**My draft:** M2 emits a `.json` report at `.assurance/intent-check-prose-report-<timestamp>.json` but doesn't compute a content hash, doesn't pair with a pre-commit hook, doesn't mirror the attestation mechanism.

**Fix:** Add a separate F2.x for `intent-check-prose-attestation` mirroring the existing schema. The path becomes `.assurance/intent-check-prose-attestation.json`. The pre-commit hook (M5/F5.3) verifies it.

### A-7 (medium) — M2/F2.5 doesn't reference /spec-adversary's explicit thresholds

**Existing /spec-adversary** § Step 7 has explicit kill criteria:
- *"Signal-to-noise < 1:5 after 4 weeks (fewer than 1 accepted proposal per 5 proposed) → scale back cadence or retire the skill for this module."*
- *"No ratified proposals land within 8 weeks → the layer-6 strategy for this repo needs rework."*

Plus: cap of 3 proposals per run; tracker file at `.assurance/spec-adversary-tracker.md`; explicit promotion-via-`/protected-surface-amend` discipline.

**My draft:** F2.5 says "same kill-criterion discipline as /spec-adversary" but doesn't quantify; doesn't enforce the 3-proposal cap; uses a different tracker path.

**Fix:** Reference the explicit thresholds; enforce the 3-proposal cap; use the same tracker path with a `.assurance/spec-adversary-prose-tracker.md` separation only if the prose variant tracker is genuinely distinct.

### A-8 (medium) — M1/F1.5 has a too-narrow manifest list

**My draft:** Hardcoded `package.json, requirements.txt, Cargo.toml, go.mod, pom.xml, *.csproj, Gemfile, composer.json, pyproject.toml, setup.py`.

**Existing /assurance-layer-audit** § Step 2 has a richer table:
| Manifest | Language |
|---|---|
| `go.mod`, `go.sum` | Go |
| `pyproject.toml`, `setup.py`, `setup.cfg`, `requirements*.txt`, `Pipfile`, `poetry.lock` | Python |
| `package.json`, `tsconfig.json`, `pnpm-lock.yaml`, `yarn.lock` | TypeScript / JavaScript |
| `Cargo.toml`, `Cargo.lock` | Rust |
| `Gemfile`, `*.gemspec` | Ruby |
| `pom.xml`, `build.gradle`, `build.gradle.kts` | Java / Kotlin |
| `*.csproj`, `*.sln`, `*.fsproj` | C# / .NET |
| `mix.exs` | Elixir |
| `*.cabal`, `stack.yaml` | Haskell |

**Fix:** F1.5 should either consult `/assurance-layer-audit`'s manifest list at runtime (couples M1 → S3.1) OR adopt the full list verbatim and document the duplication as the chosen tradeoff.

### A-9 (medium) — M1/I3's "governance-consulting skills" list is incomplete

**My draft:** `/assurance-layer-audit, /assurance-init, /intent-check, /spec-adversary, /acceptance-oracle-draft, the auditor agent`.

**Existing repo:** Hellebuyck owns nine skills; Byfuglien owns 17; the Auditor adds a third role. Governance-consulting skills (those that read `docs/invariants/` or governance scaffolding) include at least:
- `/assurance-status` (Phase 1 onboarding gate; reads governance state)
- `/assurance-roadmap-check` (drift detector against ROADMAP)
- `/protected-surface-amend` (consults the protected-surface partition)
- All five enumerated above
- The Lean pipeline skills (`/informal-spec` → `/lean-spec` → `/lean-impl` → `/correspondence-review` → `/drt-oracle`) — these consume invariants per the methodology

**Fix:** Extend M1/I3's list. Possibly add the discipline that "every skill consulting governance MUST call F1.3 (mode-of)" rather than enumerating skills explicitly.

### A-10 (medium) — M5/F5.3 doesn't fully embrace the attestation pre-commit pattern

**Existing /assurance-init** § Step 3 verbatim block: *"Pre-commit hooks are fast attestation checks only — they must never invoke LLMs or run slow test suites. Heavy verification lives in CI and in dedicated binaries that the pre-commit hook verifies were run."*

**My draft:** F5.3 says "fast" and "no LLM" but doesn't fully articulate that the pre-commit hook *verifies that the heavy work ran* (by checking attestation files), not that it merely runs a fast check.

**Fix:** Strengthen F5.3 to state explicitly: the pre-commit hook checks for the presence of `Spec-Diff-Classification` trailer AND verifies that the most recent `.assurance/intent-check-attestation.json` (and prose-attestation when applicable) is non-stale. The companion pattern is documented in `references/attestation-schema.md` (existing).

### A-11 (medium) — M1/S1.1 frontmatter retrofit conflicts with bootstrap-mode invariant doc convention

**My A-2 amendment** added YAML frontmatter to S1.1: `mode: add | bootstrap`, `phase: 0..5`, `attested-at: <SHA>`.

**Existing convention:** Bootstrap-mode `docs/invariants/<module>.md` files do NOT carry YAML frontmatter today. They have a prose `Status: Skeleton` / `Status: Active` field plus a per-module VGD-prerequisite summary table (per `/assurance-init` § Step 6.5).

**Fix options:**
1. Retrofit existing invariant docs with frontmatter (adds noise to a stable convention).
2. M1's F1.3 (`mode-of`) handles both prose-Status and YAML-frontmatter cases; default-to-bootstrap when frontmatter absent (already the rule). The VGD prereq table stays in the prose body.
3. ADD-mode docs at `docs/add/specs/modules/<module>.md` carry YAML frontmatter; bootstrap-mode docs at `docs/invariants/<module>.md` keep their prose convention.

Option 3 is what S1.1 actually says ("the dual location accommodates the source-of-truth difference between the two modes"). My M1 functional spec didn't honour the dual-location distinction. **Fix:** Update F1.3 to read either location based on which the module's invariant doc resides at, and tolerate prose-Status in bootstrap-mode docs.

### A-12 (low) — M1 doesn't reference VGD-prerequisite assessment

**Existing /assurance-init** § Step 6.5 produces a per-module VGD prereq table (#1 deterministic, #2 provable, #3 tractable, #4 hypothesis-only). `/assurance-layer-audit` § Step 4.5 produces the same table at audit time.

**My draft:** M1 doesn't mention VGD prereqs. Bootstrap-mode modules already have them; ADD-mode modules should too if we want consistent governance.

**Fix:** Add a frontmatter or section reference for the VGD prereq summary in M1's data-shapes section, or extend `S1.1`'s frontmatter format. Suggest a separate field `vgd-prereqs:` referenceable from M1/F1.4's rule table.

### A-13 (low) — M2/F2.5 missed the existing `/spec-adversary` tracker schema

`/spec-adversary` writes `.assurance/spec-adversary-tracker.md` with this structure:
```
## <YYYY-MM-DD> — <module>

Proposed: 3
Accepted: 1
Rejected: 1
Deferred: 1

### Accepted / Rejected / Deferred
- <name> — <reason>.
```

Plus the per-proposal triage block uses a specific markdown radio format with category, confidence, supporting code lines, adjacent invariants, and accept/reject/defer checkboxes.

**My draft:** F2.5 referenced `Gap.kind` enum but didn't reuse the existing tracker schema or the radio-block format.

**Fix:** F2.5 inherits the existing schema; the prose variant only differs in input shape (Drafted spec section vs ratified invariant doc).

### A-14 (low) — M3 should use the existing assurance-layer-audit tooling-detection pattern as precedent

**Existing /assurance-layer-audit** § Step 3 has a structured tooling-detection table covering test frameworks, property-based testing, formal verification tooling, governance signals, type checking, linters. This is the precedent for any "detect what the repo has" predicate.

**My draft:** M3's instrumentation tool reads git history but doesn't anchor on this precedent.

**Fix:** Note in M3's "What this spec deliberately does not specify" that the deterministic instrumentation tool's input scope (paths to scan) defaults to the artifacts the existing detection logic surfaces; reference the precedent.

---

## Bucket B — findings against the Attested seed

These need your adjudication. Per the methodology, Attested artifacts can be modified via either (a) re-drafting triggered by upstream discovery, or (b) supersession via a new ADR. The seam-validation pass is the upstream discovery; you decide which path applies.

### B-1 (high) — Architectural spec S3 is missing three skill adaptations

**The seed:** S3.1 (`/assurance-layer-audit`), S3.2 (`/assurance-init`), S3.3 (`/intent-check`), S3.4 (`/spec-adversary`), S3.5 (`/acceptance-oracle-draft`). Five skills.

**Discovered missing for v1 of ADD:**
- **S3.6 (proposed) — `/assurance-status`.** The status dashboard's Phase-2 view should be mode-aware: ADD-mode modules don't have a ROADMAP-equivalent; instead, they trace back to `docs/add/intent.md` IC IDs. The dashboard should read mode tags via `M1/F1.3 (mode-of)` and surface ADD-mode artifact statuses (Drafted/Attested/Ratified counts, cascade-pending count from M3's signals) alongside bootstrap-mode statuses.
- **S3.7 (proposed) — `/assurance-roadmap-check`.** Drift detector currently scans `docs/assurance/**/*.md`. ADD-mode modules don't write to that directory; they live under `docs/add/`. The drift detector should additionally scan `docs/add/**/*.md` for status-field drift (e.g., an artifact marked `Status: Attested` whose linkage graph shows it's Drifted in practice).
- **S3.8 (proposed) — `/protected-surface-amend`.** Currently writes governance notes for Class A/Class B partition. ADD adds `docs/add/` and `docs/add/audit/` as additional protected surfaces (per ADR-005). The amendment generator should know about both partitions and emit the right governance note depending on which partition the touched file belongs to.

**Recommendation:** Re-draft S3 to add S3.6, S3.7, S3.8. The discovery is upstream-justified (seam knowledge revealed the gap). A supersession ADR is overkill — re-drafting fits.

### B-2 (medium) — Architectural spec S2.5's `implementation:` enum misses the Lean pipeline

**The seed (post-A3 amendment):** `implementation: spec-iterate | manual | external`.

**Discovered:** The Lean pipeline (`/informal-spec` → `/lean-spec` → `/lean-impl` → `/correspondence-review` → `/drt-oracle`) is a peer Layer-1 path, *not* a path under `/spec-iterate`. Per `agents/byfuglien.md` and `docs/research/assurance-hierarchy.md`, Dafny verify-and-extract and Lean executable-model + DRT-oracle are *complementary* engines at Layer 1.

**Recommendation:** Re-draft S2.5 to extend `implementation:` enum: `spec-iterate | lean-pipeline | manual | external`. The integrity rule for `lean-pipeline` would check for a corresponding `<module>/lean/<F-section.slug>.lean` file plus a `correspondence-verdict` of `exact | abstraction | approximation` (not `mismatch`) plus a DRT-oracle run record in `.assurance/drt-oracle/<module>.json` (or whatever path that skill uses).

This is also a small upstream-discovery; re-drafting is appropriate.

### B-3 (medium) — IC4 and S2.3 inherit /intent-check's discipline implicitly; should reference it explicitly

**The seed:** IC4 says ADD requires "Phase 2 validation step that runs *before* any code or test exists" with prose-vs-prose. S2.3 (`/intent-check-prose`) describes structural mirroring of `/intent-check` but doesn't pin to the existing skill's specific disciplines (carve-out scan, two-section back-translator, attestation hash, kill-criterion env vars).

**Recommendation:** A small re-drafting to S2.3 (and possibly to IC4) explicitly inheriting `/intent-check`'s entire pipeline structure with one substitution (input shape is `(intent doc, spec stack)` instead of `(invariant prose, covering test, code diff)`). This eliminates A-1 through A-5 (Bucket A items) at the architectural-spec level rather than leaving the agent to re-derive at the functional-spec level.

Practical effect: M2 functional specs become much smaller (they reuse most of `/intent-check`'s machinery), and the seam between them and the existing skill is explicit.

### B-4 (low) — IC10 should reference the /assurance-init existing dual-track discipline

**The seed:** IC10 says documentation surfaces ADD honestly. S7.1 says README has "Operating modes" section.

**Discovered:** `/assurance-init` already writes a `Dual-Track Enforcement Principle` block (verbatim) into every onboarded repo's `docs/assurance/ROADMAP.md`. ADD's pre-commit hook + CI gate (S6.1) is a direct application of that principle. IC10 should reference the existing principle so users see ADD as continuing the discipline, not introducing it.

**Recommendation:** Optional re-draft to IC10 or to S7.1. Low priority; cosmetic.

### B-5 (low) — Existing skill catalogue is already drifted; M6/F6.4 will catch it

**Discovered:** `docs/skills.md` claims to be an "Exhaustive index of all 20 skills" but the `skills/` directory contains 26 entries. Missing from the catalogue:
- `/informal-spec`, `/lean-spec`, `/lean-impl`, `/correspondence-review`, `/drt-oracle` (the five Lean pipeline skills, which `agents/byfuglien.md` lists)
- `/assurance-probe` (mentioned in the README but not in `docs/skills.md`)

**Implication:** When M6/F6.4 (`audit-catalogue-sync`) goes live, it would currently fail. This is *not* a Bucket B finding against the seed — it's a pre-existing drift in the existing repo. Worth flagging because it means the catalogue-sync rule has immediate backlog.

**Recommendation:** Independent of ADD; would be addressed when M6 ships. No re-drafting needed.

---

## Bucket C — resolutions to open questions from M1–M6

Many open questions surfaced in turn 1 are now mechanically answerable from the seam.

| Open Q | Module | Resolution from seam |
|---|---|---|
| M1 Q1 (trailer naming) | M1 | Existing `/intent-check` uses `Spec-Diff-Classification:` trailers; my proposed `Supersedes-mode-of:` and `Re-drafting-cause:` are stylistically consistent. Keep as proposed. |
| M1 Q4 (manifest list ownership) | M1 | Existing `/assurance-layer-audit` § Step 2 has a 9-language manifest table. F1.5 should consume that table at runtime (couples M1 → S3.1 — acceptable). Documented in A-8. |
| M2 Q1 (attestation phrase list) | M2 | No existing precedent constrains this. Keep my narrow list; expand only on field-data evidence. |
| M2 Q2 (build-time check mechanism) | M2 | `/intent-check`'s blindness is enforced by *prompt structure* (separate prompts, the back-translator template never receives the invariant prose). Build-time check = lint rule scanning the prompt source for forbidden tokens. Cleaner than runtime check. |
| M2 Q3 (degraded-mode marker file) | M2 | `/intent-check` § Step 0 *refuses to run* when FP rate > tripped threshold; doesn't ship a separate marker file. The CSV itself is the source of truth. Adopt the same: no marker file; the kill-criterion pre-check reads the CSV. |
| M2 Q4 (confidence-floor default) | M2 | `/spec-adversary` doesn't expose a confidence floor; instead it caps proposals at 3 and labels each `HIGH/MEDIUM/LOW`. The floor concept can be removed; cap-at-3 is the discipline. |
| M2 Q5 (F2.7 layered enforcement) | M2 | Existing pattern: skill-side discipline + pre-commit hook safety net + CI gate. Layered, not redundant. Acceptable. |
| M3 Q1 (coupling threshold) | M3 | `/intent-check`'s 30%/20%/14d thresholds are documented as founder intuition with explicit env-var override. M3's coupling threshold should use the same disclosure pattern. |
| M3 Q2 (pair-spec discovery) | M3 | `/assurance-layer-audit` § Step 3 has a tooling-detection table that pairs spec files with test files structurally. Adopt the same precedent. Documented in A-14. |
| M3 Q3 (diff-shape granularity) | M3 | No existing precedent. Top-level Markdown headers as the unit is fine for v1. |
| M3 Q4 (script vs skill) | M3 | Existing pattern: skills carry substantive prompt content; scripts are pure computation. ADR-002 says instrumentation has no LLM. ⇒ **script**. |
| M3 Q5 (schema-bump mechanism) | M3 | Existing `references/*-schema.md` files are the precedent. M3 should produce `references/add-instrumentation-schema.md` per S4.1; CHANGELOG-style additions there. |
| M4 Q1 (PassId format) | M4 | No existing precedent. Date-based is fine. |
| M4 Q2 (routing rules) | M4 | `agents/byfuglien.md` and `agents/hellebuyck.md` make the split explicit (impl-chain vs spec-chain skills). The Auditor's routing should mirror: spec-stack changes → Hellebuyck, code/test → Byfuglien, status flips/governance → human. Confirms my proposal. |
| M4 Q3 (JSON sidecar location) | M4 | Existing `.assurance/intent-check-attestation.json` is at repo-root `.assurance/`. Auditor's report at `docs/add/audit/<pass>.md` is fine; the JSON sidecar should be `docs/add/audit/<pass>.json` (same dir, sibling). |
| M4 Q4 (write-authority layering) | M4 | Existing `.husky/pre-commit-assurance-placeholder` precedent: filesystem permissions + pre-commit hook + CI gate. Layered. Keep. |
| M4 Q5 (empty-signal-set semantics) | M4 | Existing `/assurance-status` § Phase-1-fail rule: if onboarding state is missing, the dashboard refuses Phase 2 and lists what's missing. Apply same: empty signal set with "instrumentation didn't run" branch. Distinguished from "all settled by absence of signal." Adopt my proposed disambiguation. |
| M5 Q1 (squash dominance order) | M5 | No existing precedent. My proposed order is reasonable; adopt. |
| M5 Q2 (allowlist file location) | M5 | `.assurance/audit-authors.allowlist` is consistent with other `.assurance/` artifacts. Adopt. |
| M5 Q3 (force-push enforcement) | M5 | Platform-dependent; `/assurance-init` § Step 7 already detects CI system. Use detected CI's branch-protection mechanism. |
| M5 Q4 (log_path default) | M5 | `.assurance/diff-classification-log.jsonl` is consistent with existing `.assurance/intent-check-fp-tracker.csv`. Adopt. |
| M5 Q5 (author identity) | M5 | `/intent-check` uses git author ident directly. Same here. |
| M6 Q1 (forbidden-phrase list) | M6 | Existing copy avoids `proven`, `validated`, `field-tested`, `evidence-backed`. Add `production-ready`, `battle-tested` as obvious extensions. |
| M6 Q2 (heading-name flexibility) | M6 | No precedent forces a single canonical heading. Keep flexible. |
| M6 Q3 ("non-stub" criterion) | M6 | Existing `docs/skills.md` has "one-line summary" per skill. Adopt: "≥1 sentence non-empty description". |
| M6 Q4 (CI packaging) | M6 | `/assurance-init` writes a single `.github/workflows/assurance.yml` that runs all checks. Bundle the four M6 checks into one job. |
| M6 Q5 (mode-command coupling) | M6 | Couple-to-catalogue is unavoidable because the catalogue IS the source of truth. Acceptable. |

---

## Adversarial probing of the new findings

Per the methodology's Phase 2 § Step 4 discipline, I should probe the seam-validation findings themselves for second-order issues.

- **A-1, A-2, A-3, A-4, A-5 (cluster).** All of these reduce to: M2 should adopt /intent-check's pipeline wholesale and parameterise on input shape. The functional spec for M2 will likely shrink by 30-40% if I do this. Worth confirming that's the right shape.
- **B-1 (S3 extension).** Adding S3.6/S3.7/S3.8 expands the architectural spec's surface area by ~15%. The risk is that each adaptation requires its own delta spec — three more pieces of work. Worth confirming you want the discipline applied to all three or only the highest-leverage one (S3.6 `/assurance-status` is most user-visible).
- **B-2 (Lean pipeline).** Adding `lean-pipeline` to S2.5 implies ADD-mode modules whose F-sections declare `implementation: lean-pipeline` need a place for the correspondence-review verdict. That's not currently in the spec. Implementing this properly means another integrity rule and possibly a 6th value class.
- **A-11 (frontmatter retrofit).** Choosing option 2 or 3 is fine; option 1 (retrofit existing invariant docs) is the most invasive and should be avoided. The choice has implications for M1's F1.3 implementation.

---

## Verdict

### **PASS-WITH-AMENDMENTS** (against the Drafted M1–M6 and `behavioral.md`)

The Drafted Phase 1 work is substantively sound but has 14 concrete defects (A-1..A-14) — most of them traceable to one root cause: I drafted M2 (greenfield-skills) without the mental model of how `/intent-check` currently works. M2 is the largest cluster of fixes; M1, M3, M5, M6 each have 1–2 smaller issues.

### **PASS-WITH-AMENDMENTS** (against the Attested seed)

Five findings (B-1..B-5). B-1 (S3 extension) is the most material — re-drafting the architectural spec to add S3.6, S3.7, S3.8 is appropriate. B-2 (Lean pipeline) is also re-drafting-worthy. B-3 (S2.3 inheritance from /intent-check) is the single highest-leverage edit because it reduces work in M2. B-4 and B-5 are cosmetic/backlog.

### Halting at the verdict

Per the protocol shape (Phase 2 § Step 6), the agent halts after the verdict. I am NOT executing any of the Bucket A or Bucket B amendments without your sign-off.

The Bucket A items live on Drafted artifacts and are safe to amend on agent's next exchange after explicit approval — same shape as the original 13 amendments.

The Bucket B items live on Attested artifacts and require your explicit decision: re-drafting (lighter) vs supersession ADR (heavier). The seam-validation pass *is* the upstream discovery that justifies re-drafting, so re-drafting is the principled choice for B-1, B-2, B-3.

Bucket C resolutions are agent-decided once you give the nod; they don't require per-question sign-off.

---

## What this report deliberately does not do

- Does not amend any artifact (Drafted or Attested).
- Does not propose Phase 3 skeleton work.
- Does not redo turn 1's Phase 2 verdict — that verdict was based on the seed-only cold read, which is what the protocol asked for. The seam-validation is a *complement*, not a replacement.

## Recommended adjudication path

1. Read this report; tick the Bucket A items you want amended (probably all of them).
2. Decide on Bucket B per finding:
   - B-1 (S3 extension): re-draft / supersede / defer?
   - B-2 (Lean pipeline): re-draft / supersede / defer?
   - B-3 (S2.3 inheritance): re-draft / supersede / defer?
   - B-4, B-5: optional cosmetic; decide later.
3. The Bucket C resolutions are noted; agent applies them as part of the Bucket A amendments without separate per-question sign-off, unless you flag specific ones.
4. After amendments commit, you re-attest the affected seed artifacts (the re-drafting flow per `glossary.md` § Cascade).
5. Then we proceed to Phase 3 skeleton work, one module at a time per your previous direction.

I'd recommend tackling Bucket A and the three high-leverage Bucket B re-draftings (B-1, B-2, B-3) as one session. B-3 in particular makes A-1 through A-5 nearly free.
