Now I have a complete picture of the codebase. Let me produce the analysis.

---

## Analysis: nicholls-inc/claude-code-marketplace#140

---

### 1. Issue Summary

**Title:** `crosscheck: assurance-probe — deterministic test-strength layer (design discussion)`

The issue identifies a gap in the crosscheck plugin's 6-layer assurance hierarchy: while Layer 4 currently checks that a covering-test *comment* (`# Invariant <ID>:`) exists above a test function, it does not verify that the test would actually *fail* if the documented invariant were violated. A vacuous test (one that imports the module but never exercises the failure path) silently passes the current Layer 4 gate. The issue proposes a new `assurance-probe` skill — scoped to Layer 4 — that applies targeted source mutations derived from each invariant's `Failure condition` clause, re-runs the covering tests, and reports `killed | survived | errored` per (invariant, mutation, test). The three open questions embedded in the issue have already been answered by the author (commit reproducers in-tree under an isolated directory; yes include a SKILL.md; yes open a follow-up implementation issue).

---

### 2. Files Involved

- `crosscheck/skills/invariant-coverage-scaffold/SKILL.md` — **entire file** — the existing Layer 4 presence gate this new skill strengthens; establishes the bidirectional ID-comment link that `assurance-probe` builds on.
- `crosscheck/skills/spec-adversary/SKILL.md` — **entire file** — defines the run output shape (`≤3 findings, issue-only, triage blocks, tracker CSV, SNR kill criterion`) that `assurance-probe` should mirror.
- `crosscheck/skills/assurance-status/SKILL.md` — **entire file** — dashboard consumer; will need a new section for `probe-tracker` when the skill ships.
- `crosscheck/docs/assurance-hierarchy.md:9–16` — Layer/skill mapping table; `assurance-probe` needs a new row under Layer 4.
- `crosscheck/docs/skills.md:31–37` — Layer 4 skill catalogue table; new row required.
- `crosscheck/agents/hellebuyck.md` — the hellebuyck agent owns Layer 4–6 skills; routing table will need `assurance-probe` added.
- `crosscheck/.claude-plugin/plugin.json` — plugin metadata (version bump expected on ship, not a blocker).
- `crosscheck/README.md` — Layer 4 description block will need a bullet for the new skill.

**Notably absent:** There is no existing `crosscheck/skills/assurance-probe/` directory at any path (`find` and `grep` confirm zero matches for `assurance-probe`, `probe-tracker`, `mutation probe`, `vacuity probe`). This is a net-new skill; no code exists to modify.

---

### 3. Evidence Trace

This is a feature-design issue, not a regression, so the four-phase locate-fault methodology does not apply. Instead, a plain evidence trace of the gap:

**Phase A — Existing Layer 4 coverage check (the presence gate)**

`crosscheck/skills/invariant-coverage-scaffold/SKILL.md:17–20` establishes what the gate checks:

> 1. **Silent test drop.** An invariant exists in the doc but no test references it → coverage is claimed that does not exist.
> 2. **Orphan test comment.** A test references an invariant ID that is not declared in any module doc → stale or typo'd reference.

The gate only verifies *comment existence* (`// Invariant <ID>: <Name>` appearing above a test function), not *test effectiveness*. This is confirmed by `invariant-coverage-scaffold/SKILL.md:126–144` (Step 7: Wire In The Pre-commit Hook), which hooks on file changes to `docs/invariants/**/*.md` and the test glob — a static text match, not a run.

**Phase B — The asserted gap**

The issue (`body`) states the uncovered scenario directly: "A vacuous test (one that imports the module but never exercises the failure condition) passes the Layer 4 gate today." No code anywhere in the plugin tests for mutation-kill — confirmed by searching for `mutation`, `vacuity`, `Failure condition`, and `probe-tracker` across all files (zero matches).

**Phase C — The proposed artifact shape**

From the issue body:
- Output file: `scripts/probe/<module>_<YYYYMMDD>.py` in the adopter repo, committed deterministically (Q1 answer: yes, commit it).
- Tracker: `.assurance/probe-tracker.csv` (mirrors `.assurance/spec-adversary-log.csv` shape — **note**: the existing spec-adversary skill uses `.assurance/spec-adversary-tracker.md`, `spec-adversary/SKILL.md:189`; the issue names it `probe-tracker.csv` in CSV format, matching `intent-check-fp-tracker.csv`'s shape more than the Markdown tracker).
- Trigger: rotation, not per-PR; gated on non-aspirational invariants, modules not probed in the last 30 days.
- Kill criterion: SNR < 1:5 over 4 weeks → retire for that module.

---

### 4. Root-Cause Hypothesis

**Primary (confirmed):** The gap is structural — the existing Layer 4 gate was intentionally designed as a *presence* check only (`invariant-coverage-scaffold/SKILL.md:17–20`). The strength of the invariant–test link was never in scope for that gate. No code exists to strengthen it. The new skill is additive with no functional overlap.

**Alternative 1 (alive but lower):** The issue might be partially solvable by extending `/intent-check` (Layer 5) rather than a new Layer 4 skill. `/intent-check` already runs a round-trip back-translation to check whether the covering test semantically matches the invariant prose (`intent-check/SKILL.md:59–89`). If the test is vacuous, the back-translator would likely produce a back-translation that diverges from the invariant's `Failure condition`. However, the issue correctly notes this is a different concern (semantic prose alignment vs. deterministic kill/survive from code mutation) and places the new skill at Layer 4 (deterministic). The round-trip check is probabilistic (~96%) and runs on PR diffs — not on rotation with reproducers.

**Alternative 2 (low):** The tracker format discrepancy (`probe-tracker.csv` CSV vs. `spec-adversary-tracker.md` Markdown) could indicate the issue intends a different file layout. Reading `spec-adversary/SKILL.md:186–207` confirms the adversary tracker is Markdown, not CSV. The issue explicitly says "mirrors the existing `.assurance/spec-adversary-log.csv` shape" (referencing a CSV that does not appear to exist in the plugin — the actual file is `.assurance/spec-adversary-tracker.md`). The implementation will need to pick one format and document it clearly; the CSV shape is more machine-readable and mirrors `intent-check-fp-tracker.csv`, so it's the better choice.

---

### 5. Classification

| Classification | Verification track | Rationale |
|---|---|---|
| `refactor` | `semi-formal` | This is a **behaviour-preserving structural addition** to the crosscheck plugin's skill catalogue. No existing behaviour changes; the new `assurance-probe` skill is additive. The implementation work is writing a new `SKILL.md` (a behavioral artifact) plus updating the routing tables in `hellebuyck.md`, `docs/assurance-hierarchy.md`, `docs/skills.md`, and `README.md`. The correctness question is: does the new skill's design faithfully implement the issue's specification without contradicting or overlapping the existing Layer 4/5/6 skill boundaries? That is a `compare-patches`/semi-formal reasoning question, not a Dafny formalization target. |

---

```json
{
  "context_updates": {
    "issue_ref": "nicholls-inc/claude-code-marketplace#140",
    "issue_title": "crosscheck: assurance-probe — deterministic test-strength layer (design discussion)",
    "analysis_classification": "refactor",
    "verification_track": "semi-formal",
    "files_involved": [
      "crosscheck/skills/spec-adversary/SKILL.md",
      "crosscheck/skills/invariant-coverage-scaffold/SKILL.md",
      "crosscheck/skills/intent-check/SKILL.md",
      "crosscheck/skills/assurance-status/SKILL.md",
      "crosscheck/agents/hellebuyck.md",
      "crosscheck/docs/assurance-hierarchy.md",
      "crosscheck/docs/skills.md",
      "crosscheck/README.md",
      "crosscheck/.claude-plugin/plugin.json"
    ]
  }
}
```