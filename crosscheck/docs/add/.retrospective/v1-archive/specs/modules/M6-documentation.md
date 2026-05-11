# M6-documentation — Functional Spec

```yaml
---
id: M6-documentation
mode: add
phase: 1
status: Drafted
consumes: [IC10, S7.1, S7.2, S7.3, S7.4, TM6, M6-documentation/B1, M6-documentation/B2, M6-documentation/B3]
produces: [F6.1, F6.2, F6.3, F6.4, I1, I2, I3, T6.1..T6.8]
last-attested: N/A (Drafted)
---
```

## Purpose

Per-operation functional specs for the **documentation** module: the README "Operating modes" section, the bootstrap-vs-ADD recommended-order split, the hypothesis-status disclaimer, and the catalogue-sync CI check. This module is structurally simpler than M1–M5 (no runtime predicates beyond a CI check) but is the user-facing surface for IC10 and TM6.

The operations here are **content audits** — they take Markdown files as input and assert structural properties (presence of named sections, absence of misleading copy). The implementation is a small CI script, not a SKILL.md.

---

## Data shapes

### `ReadmeAuditResult`

```
ReadmeAuditResult := Pass | Fail { violations: NonEmptyList<DocViolation> }

DocViolation :=
  | MissingSection { expected_heading: String }
  | MergedRecommendedOrder { evidence_excerpt: String }
  | MissingHypothesisDisclaimer
  | MissingOpenProblemsLink
  | StaleSkillCatalogue { skill_id: String, status: Missing | Stale }
  | StaleAgentRegistry { agent_id: String, status: Missing | Stale }
```

### `MarkdownDoc`

```
MarkdownDoc := {
  path: AbsolutePath,
  raw: String,
  parsed_headings: List<Heading>,
  parsed_sections: List<Section>
}
```

Standard Markdown AST; the implementation choice is the agent's.

---

## F6.1 — `audit-readme-operating-modes-section(readme: MarkdownDoc) → ReadmeAuditResult`

```yaml
---
id: F6.1
status: Drafted
implementation: manual
consumes: [M6-documentation/B1, IC10, S7.1]
produces: [I1, T6.1, T6.2]
---
```

### Signature
`audit-readme-operating-modes-section(readme: MarkdownDoc) → ReadmeAuditResult`

### Postconditions

Returns `Pass` iff ALL of:
1. `readme.parsed_headings` contains a heading at level 2 or deeper exactly matching one of: "Operating modes", "Operating Modes", "Modes", or "Operating mode" (case-insensitive equality after trimming whitespace).
2. The body of that section mentions both `bootstrap` and `add` (or `ADD`) as named modes.
3. The body either describes both modes inline or links to descriptions.

Returns `Fail { violations }` listing each unmet condition with an excerpt for context.

### Frame conditions
- Reads `readme` only.

### Module invariants preserved
- I1 (README structurally distinguishes bootstrap and ADD).

### Test linkage
- T6.1 — README with `## Operating modes` containing both modes → Pass.
- T6.2 — README without an Operating Modes section → Fail with `MissingSection { "Operating modes" }`.

---

## F6.2 — `audit-recommended-order-split(readme: MarkdownDoc) → ReadmeAuditResult`

```yaml
---
id: F6.2
status: Drafted
implementation: manual
consumes: [M6-documentation/B1, IC10, S7.1]
produces: [I1, T6.3, T6.4]
---
```

### Signature
`audit-recommended-order-split(readme: MarkdownDoc) → ReadmeAuditResult`

### Postconditions

Returns `Pass` iff:
1. `readme.parsed_headings` contains either:
   - A "Recommended order — bootstrap mode" subsection (or `Bootstrap mode order`, `Bootstrap-mode recommended order`, etc.) AND a "Recommended order — ADD mode" subsection (or equivalent), each at level 2 or 3, OR
   - A single "Recommended order" subsection that explicitly contains a `bootstrap-mode:` and an `add-mode:` sub-heading (level 4 or deeper).
2. The bootstrap subsection's first non-empty line does not reference ADD-mode-only commands (`/intent-elicit`, `/spec-derive`, `/intent-check-prose`, `/spec-adversary-prose`).
3. The ADD subsection's first non-empty line does not exclusively reference bootstrap-mode-only commands (`/assurance-init` for the modules question, etc.).

Returns `Fail { violations: [MergedRecommendedOrder { excerpt }] }` if a single un-split "Recommended order" lists commands from both modes without distinguishing them.

### Frame conditions
- Reads `readme` only.

### Module invariants preserved
- I1.

### Test linkage
- T6.3 — README with split bootstrap/ADD sections → Pass.
- T6.4 — README with merged "Recommended order" listing both `/assurance-init` and `/intent-elicit` without subsection split → Fail with `MergedRecommendedOrder`.

---

## F6.3 — `audit-hypothesis-disclaimer(readme: MarkdownDoc) → ReadmeAuditResult`

```yaml
---
id: F6.3
status: Drafted
implementation: manual
consumes: [M6-documentation/B2, IC10, TM6, S7.4]
produces: [I2, T6.5, T6.6]
---
```

### Signature
`audit-hypothesis-disclaimer(readme: MarkdownDoc) → ReadmeAuditResult`

### Postconditions

Returns `Pass` iff ALL of:
1. The README contains the substring `hypothesis` (case-insensitive) within 200 characters of any mention of "Assurance Driven Development" or "ADD mode".
2. The README either contains a list of open problems (matched as a Markdown list under a heading containing "open problems" or "limitations") OR a link to `crosscheck/docs/add/methodology.md` § Open problems (matched as `[...](.../methodology.md#open-problems)` or similar anchor).
3. No phrase in the README claims ADD is "proven", "validated", "field-tested", or "evidence-backed" (these strings are forbidden when describing ADD's status).

Returns `Fail { violations }` listing each unmet condition.

### Frame conditions
- Reads `readme` only.

### Module invariants preserved
- I2 (hypothesis honesty).

### Test linkage
- T6.5 — README explicitly says "ADD is a hypothesis" and links to open problems → Pass.
- T6.6 — README says "ADD is field-tested and validated" → Fail (forbidden phrase).

---

## F6.4 — `audit-catalogue-sync(skills_md: MarkdownDoc, agents_md: MarkdownDoc, skills_dir: DirectoryPath, agents_dir: DirectoryPath) → ReadmeAuditResult`

```yaml
---
id: F6.4
status: Drafted
implementation: manual
consumes: [M6-documentation/B3, IC10, S7.2, S7.3]
produces: [I3, T6.7, T6.8]
---
```

### Signature
`audit-catalogue-sync(skills_md, agents_md, skills_dir, agents_dir) → ReadmeAuditResult`

### Postconditions

For each subdirectory `s` of `skills_dir` containing a `SKILL.md`:
- Extract the slash-command identifier (e.g., `/intent-elicit` from `skills/intent-elicit/SKILL.md`).
- Verify that `skills_md.raw` contains a non-stub mention of the identifier (a line listing the skill with at least a one-sentence description).
- If missing → emit `StaleSkillCatalogue { skill_id, status: Missing }`.
- If listed but the description is empty/placeholder → emit `StaleSkillCatalogue { skill_id, status: Stale }`.

For each subdirectory `a` of `agents_dir` containing an `<agent>.md`:
- Same check against `agents_md.raw` for the agent identifier.
- Emit `StaleAgentRegistry { agent_id, status }` for missing/stale.

Returns `Pass` iff no violations; otherwise `Fail { violations }`.

### Frame conditions
- Reads the four input artifacts.

### Module invariants preserved
- I3 (catalogues stay in sync with directories).

### Test linkage
- T6.7 — `skills/intent-elicit/SKILL.md` exists; `skills.md` does not list `/intent-elicit` → Fail with `StaleSkillCatalogue { /intent-elicit, Missing }`.
- T6.8 — `agents/auditor.md` exists; `agents.md` does not list the Auditor → Fail with `StaleAgentRegistry { auditor, Missing }`.

---

## Module invariants — `I1`..`I3`

### I1 — README structurally distinguishes bootstrap and ADD
The plugin README contains an "Operating modes" section AND a split (or sub-headed) "Recommended order" that gives bootstrap-mode users a path that does not reference ADD-mode-only commands and vice versa. F6.1 + F6.2 enforce.

### I2 — Hypothesis honesty
The README acknowledges ADD's hypothesis status, links to the methodology's open-problems list, and contains no copy claiming ADD is proven/validated/field-tested/evidence-backed. F6.3 enforces.

### I3 — Catalogue sync
Every skill in `skills/<skill>/SKILL.md` has a non-stub entry in `docs/skills.md`. Every agent in `agents/<agent>.md` has a non-stub entry in `docs/agents.md`. F6.4 enforces in CI.

---

## Test linkage stubs — `T6.1`..`T6.8`

| ID | Operation | Stub description |
|---|---|---|
| T6.1 | F6.1 | README with Operating Modes section → Pass |
| T6.2 | F6.1 | README without it → Fail (MissingSection) |
| T6.3 | F6.2 | split bootstrap/ADD recommended order → Pass |
| T6.4 | F6.2 | merged recommended order → Fail (MergedRecommendedOrder) |
| T6.5 | F6.3 | "ADD is a hypothesis" + open-problems link → Pass |
| T6.6 | F6.3 | "ADD is field-tested" → Fail (forbidden phrase) |
| T6.7 | F6.4 | skill exists, not in docs/skills.md → Fail (StaleSkillCatalogue) |
| T6.8 | F6.4 | agent exists, not in docs/agents.md → Fail (StaleAgentRegistry) |

---

## What this spec deliberately does not specify

- The exact wording of the README's "Operating modes" section. Editorial concern; humans write the prose, the audit checks structure.
- The Markdown parser implementation (any standard CommonMark parser is fine).
- The CI integration shape (GitHub Action vs GitLab CI job vs CircleCI orb). S3.2's detection determines.
- The list of forbidden phrases beyond `proven | validated | field-tested | evidence-backed`. New phrases require a propagated-discovery amendment.

## Open questions surfaced by this draft

1. **Forbidden-phrase list completeness.** Four phrases is a starting point; reasonable to extend (e.g., `production-ready`, `battle-tested`). Worth your judgment on the full list.
2. **Heading-name flexibility in F6.1 / F6.2.** I allowed several variants of "Operating modes". If you'd prefer a single canonical heading, easy to tighten.
3. **F6.4's "non-stub" criterion.** I committed on "at least a one-sentence description." Could be stricter (e.g., "≥3 sentences") or looser (e.g., "any non-empty description"). Worth picking.
4. **CI placement.** The four operations naturally bundle into one CI job (`docs-audit`). Worth confirming this is the right packaging vs. four separate checks.
5. **Bootstrap vs ADD command lists.** F6.2 references specific commands as mode-specific. The mapping is in the architectural spec; if commands change, F6.2's predicate must be updated. Worth flagging that this couples M6 to the skill catalogue evolution.
