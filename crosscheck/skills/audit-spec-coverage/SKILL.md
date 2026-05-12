---
name: audit-spec-coverage
description: >-
  Layer 6 spec-coverage probe. Given a prose spec and a glob of invariant
  docs, emits two coverage matrices (spec section → invariant IDs;
  audit-finding ID → invariant IDs) plus a prioritised gap list and 4-path
  triage blocks. Mirrors /spec-adversary's discipline (capped output, kill
  criteria, "what this does NOT catch" honesty). Independently useful as a
  standalone audit; also consumed by add-orchestrator as the coverage leg of
  the spec-driven fast path. Triggers: "spec coverage", "coverage matrix",
  "which spec sections lack invariants", "audit-finding coverage".
argument-hint: "<spec-path> [invariant-docs-glob]"
---

# /audit-spec-coverage — Spec-Coverage Audit (Layer 6)

## Description

Layer 6 of the assurance hierarchy — *spec completeness* — has two failure
modes that this skill probes:

1. **Sections of the prose spec with no invariant doc covering them.** The
   spec asserts a constraint; no invariant captures it; the constraint
   silently disappears from the enforceable surface.
2. **Audit-finding IDs (the §14.3-style traceability tables in mature
   specs) with no invariant covering them.** The spec audit caught a class
   of failure; no invariant prevents its recurrence.

Both are coverage gaps, both are silent unless someone runs this audit.

The skill is the mirror image of `/spec-adversary`:

| Skill              | Question it answers                                           |
|--------------------|---------------------------------------------------------------|
| `/spec-adversary`  | "Given an invariant doc, what is it missing about the code?"  |
| `/audit-spec-coverage` | "Given a prose spec, what is the invariant doc set missing about the spec?" |

`/spec-adversary` probes the code-vs-doc gap. This skill probes the
spec-vs-doc gap. Use both for full Layer 6 coverage.

This is **best-effort, like all Layer 6 work.** Coverage by section and
audit-finding ID is a structural check; it does not catch cross-module
semantic gaps where the same constraint hides under different domain nouns,
and it does not catch constraints the spec itself omits but the code
enforces. See "What this does NOT catch" below.

## When to invoke

Trigger phrases: `"spec coverage"`, `"coverage matrix"`, `"which spec
sections lack invariants"`, `"audit-finding coverage"`, `"spec section
coverage"`, `"audit which spec sections are uncovered"`.

Typical invocations:

- `/crosscheck:audit-spec-coverage docs/design/<repo>-spec.md` — default
  invariant glob `docs/invariants/*.md`.
- `/crosscheck:audit-spec-coverage docs/design/<repo>-spec.md docs/invariants/*.md`
  — explicit glob.
- Called by `add-orchestrator` step 7 as the coverage leg of the spec-driven
  fast path.

## Methodology (execute in order)

### Step 1: Resolve inputs

- **Spec path** (required, positional 1). Refuse with a clear error if the
  file does not exist or is empty.
- **Invariant-docs glob** (optional, positional 2; default
  `docs/invariants/*.md`). If the glob matches zero files, refuse with
  "no invariant docs found — run `/crosscheck:draft-invariants` first".

If the orchestrator marker file `.assurance/add-session-*/session.json`
exists in cwd or ancestors AND its `spec_path` matches the supplied spec,
write output to `.assurance/add-session-<id>/findings-coverage.md` instead
of the default cwd `findings-coverage.md`. Detect via filesystem; do not
require a flag.

### Step 2: Enumerate the spec

Read the spec end to end. Build a section index:

- For each `## <N>` / `### <N.M>` / `#### <N.M.K>` heading, record:
  - Section number (e.g. `§3.2`, `§6.4.1`)
  - Section title
  - RFC-2119 keyword density (count of `MUST`, `MUST NOT`, `SHALL`,
    `SHALL NOT`, `SHOULD`, `SHOULD NOT`, `MAY`)
  - Domain nouns (proper nouns + bolded terms + glossary entries within
    the section body)
  - Byte range (start/end line numbers) — needed for evidence citations

Detect any audit-finding traceability table (commonly §14, §15, or an
appendix; identifiable by columns like `ID | Severity | Section | Status`
or RFC-2119-keyworded rows mapping finding IDs to sections):

- Enumerate every audit-finding ID (`C1`, `C2`, `I1`, `I7`, `M3`, `V12`,
  etc. — the exact ID scheme varies by spec).
- For each ID, record the linked spec section and a one-line description
  from the table.
- If no audit-finding table is found, record this as a deliberate
  observation in the output ("no audit-finding table found in this spec";
  the audit-finding-coverage matrix is then empty).

### Step 3: Enumerate the invariants

For each invariant doc:

- Identify every `IN.` heading (`I1.`, `I2.`, ..., `I7.`, plus sub-IDs
  like `I1a.` if present).
- For each invariant:
  - Statement (the one-paragraph English body)
  - `Why:` rationale (which often cites a spec section or audit-finding
    ID)
  - `Test:` sketch
  - Domain nouns it uses

Build a flat list of `(module, IN_id, statement_snippet, why_text)` rows
across all docs.

### Step 4: Section-coverage matrix

For each spec section, list the invariant IDs that cover it. Coverage
signals:

- **Direct citation.** Invariant `Why:` text contains the section number
  (`§3.2`, `Section 3.2`, etc.) or an unambiguous title reference.
- **Domain-noun overlap.** Invariant uses domain nouns the section
  introduces (above a noun-overlap threshold; default ≥ 2 shared
  domain-specific nouns, excluding generic terms).
- **Audit-finding back-link.** Invariant cites an audit-finding ID from the
  §14.3 table, and that ID maps to the section.

A section is **UNCOVERED** if no invariant satisfies any of these signals.
A section is **PARTIALLY COVERED** if at least one signal fires but the
section has multiple distinct constraints (RFC-2119 keyword count > 3) and
not all of them are addressed.

Emit the matrix as a markdown table:

```
| Spec section | RFC-2119 weight | Covering invariants | Coverage status |
|---|---|---|---|
| §3.2 Webhook handler | 4× MUST | secrets.I1, secrets.I3 | COVERED |
| §6.4 Cancellation propagation | 2× MUST, 1× SHOULD | (none) | UNCOVERED |
| §9.1 Failure-budget tripping | 3× MUST | budget.I1 | PARTIAL — 3 MUSTs, 1 covered |
| ... | ... | ... | ... |
```

### Step 5: Audit-finding-ID coverage matrix

If §3 found an audit-finding table:

For each audit-finding ID, list the invariant IDs whose `Why:` mentions
that ID (case-sensitive exact match) or whose statement clearly addresses
the failure class the audit caught.

Emit the matrix:

```
| Audit-finding ID | Description | Linked spec section | Covering invariants | Coverage status |
|---|---|---|---|---|
| I7 | _FILE-wins disambiguation | §12.2 | secrets.I7 | COVERED |
| C3 | Webhook signature replay | §3.4 | (none) | UNCOVERED |
| ... | ... | ... | ... | ... |
```

If no audit-finding table was found, state this explicitly:

> No audit-finding traceability table detected in `<spec-path>`. Audit-finding-ID coverage matrix is empty — this is normal for specs without an embedded audit catalogue.

### Step 6: Prioritised gap list

Collect all `UNCOVERED` items (from both matrices). Order by load-bearing
weight:

1. Sections with `MUST` / `MUST NOT` / `SHALL` keywords (descending count).
2. Sections with `SHOULD` / `SHOULD NOT` keywords.
3. Audit-finding IDs (ordered by their `Severity` column if the table has
   one; otherwise by source order).
4. Sections with `MAY` keywords (informational).

**Cap output at 15 gaps.** If more candidates exist, drop the lowest-weight
items and record `... and N more lower-weight items omitted; re-run with a
narrower invariant glob to see them` in the output.

This cap mirrors `/spec-adversary`'s ≤3 cap rationale: reviewer fatigue is
the dominant failure mode. 15 is the scaled-up coverage analog — high
enough to capture the load-bearing surface, low enough to triage in one
sitting.

### Step 7: Emit findings-coverage.md

Write the output file. Schema:

```markdown
---
session: <id or "standalone">
category: coverage
generated_at: <YYYY-MM-DDTHH:MM:SSZ>
spec_path: <path>
invariant_glob: <glob>
total_findings: <n>
---

# Findings: spec coverage

## Summary

- Spec sections enumerated: <N>
- Audit-finding IDs enumerated: <N or "no table found">
- Invariants enumerated: <N across M modules>
- Uncovered sections: <N>
- Uncovered audit-finding IDs: <N>
- Findings emitted (capped at 15): <N>
- Findings omitted as lower-weight: <N or "0">

## Section coverage matrix

<table from Step 4>

## Audit-finding-ID coverage matrix

<table from Step 5, or "No audit-finding table detected">

## Prioritised gaps

### F1: <short name, e.g. "§6.4 Cancellation propagation uncovered">
**Severity:** Blocker | High | Medium | Low
**Category:** spec-section-uncovered | audit-finding-uncovered | section-partially-covered
**Evidence:**
- Spec: `<spec-path>:<line-range>` — <quote of the relevant RFC-2119 constraint>
- Invariants: (none match) | <module>/<doc>:<IN_id> matches partially because <reason>
**Why this matters:** <2-3 sentences naming the failure class this gap admits>
**Proposed resolution:** <one sentence; usually "add an invariant covering <X> to <module>.md">

**Triage (mark exactly one):**
- [ ] Accept (fix invariant) — <one-line fix description>
- [ ] Accept (amend spec via /protected-surface-amend) — <one-line note on spec edit>
- [ ] Reject — <reason>
- [ ] Defer — <revisit condition>

### F2: ...
...
```

Severity rubric:
- **Blocker** — RFC-2119 `MUST` / `MUST NOT` constraint uncovered, OR audit-finding ID labelled `critical` / `high` uncovered.
- **High** — RFC-2119 `SHALL` constraint uncovered, OR audit-finding ID labelled `medium`.
- **Medium** — RFC-2119 `SHOULD` constraint uncovered, OR audit-finding ID labelled `low`.
- **Low** — RFC-2119 `MAY` constraint uncovered, OR a section with no RFC-2119 keywords but explicit invariant-shaped prose ("the system always ...", "for all X ...").

Sort gaps by severity (Blocker first); within severity, sort by spec section number.

### Step 8: Required "What this does NOT catch" section

Mirror `/spec-adversary`'s honesty discipline. Append this section to the
output verbatim, then add any spec-specific caveats discovered during the
run:

```markdown
## What this does NOT catch

This skill is a structural coverage probe. It detects gaps in the
section → invariant and audit-finding → invariant mappings. It cannot
detect:

1. **Cross-module semantic gaps where domain nouns differ.** If `module_a`
   names a concept `expired` and `module_b` names the same concept
   `past-deadline`, this skill will see both as covered when in fact one
   module has imported a different definition. Use
   `/audit-invariant-consistency` for the cross-module pass.
2. **Constraints the spec itself omits but the code enforces.** If the
   spec is silent on a property but the code defends it, this skill
   reports the spec as authoritative. Use `/spec-adversary` for the
   code-vs-doc gap probe.
3. **Gaps inside the audit-finding table itself.** If the table is
   incomplete (the audit was partial), the matrix will reflect the
   incomplete table. The skill does not re-audit the spec from scratch.
4. **Subtle constraint composition.** Two `SHOULD` constraints that
   together imply a `MUST` are reported as two `SHOULD`s. Compositional
   reasoning is not in scope.
5. **Stale invariant doc citations.** If an invariant cites `§3.2` but
   §3.2 has been renumbered to §3.3, the citation may match the old
   number and miss the new section. Re-run after spec renumbering.

When in doubt, treat a `COVERED` row with low evidence (single domain-noun
overlap, no direct citation) as `PARTIAL` and probe further.
```

### Step 9: Kill criteria

This skill is Layer 6 best-effort. Track its signal-to-noise:

- **Signal-to-noise < 1:5 after 4 runs** (fewer than 1 accepted gap per
  5 emitted) → the audit is mostly false positives; recalibrate domain-noun
  threshold or re-run with a different invariant glob.
- **Zero findings on a spec with ≥ 20 RFC-2119 keywords across multiple
  modules with no invariant docs** → suspect a bug; the skill is missing
  the section-extraction logic, not finding genuine coverage.

If a tracker file `.assurance/audit-spec-coverage-tracker.md` exists,
append a one-line summary of this run (counts, date). If not, do not
create one — tracker creation is the user's choice.

## Output structure

A single `findings-coverage.md` file at one of:

- Orchestrator mode: `.assurance/add-session-<id>/findings-coverage.md`
- Standalone mode: `<cwd>/findings-coverage.md`

The skill writes the file and prints its path. It does NOT modify the spec
or any invariant doc. Triage and apply are downstream tasks (manual, or
via `add-orchestrator` step 10).

## Checklist before handing off

- [ ] Spec path validated; file exists and is non-empty
- [ ] Invariant glob resolved to at least one file
- [ ] Every spec section enumerated by number
- [ ] RFC-2119 keyword density recorded per section
- [ ] Audit-finding table detected (or absence stated explicitly)
- [ ] Every invariant doc's `IN.` IDs enumerated
- [ ] Section-coverage matrix complete
- [ ] Audit-finding-coverage matrix complete (or stated empty)
- [ ] Gaps prioritised by RFC-2119 weight
- [ ] ≤ 15 gaps emitted (with overflow note if applicable)
- [ ] Each gap has 4-path triage block with `Accept (amend spec)` as a
      first-class option
- [ ] "What this does NOT catch" section present
- [ ] Output written to correct path (orchestrator vs standalone)

## Arguments

1. **Spec path** (required) — absolute or repo-relative path to the prose
   spec to audit.
2. **Invariant docs glob** (optional, default `docs/invariants/*.md`) —
   glob pattern for the invariant docs to audit against.

Examples:

- `/audit-spec-coverage docs/design/fabricator-spec.md` — audit the ngst
  spec against all invariant docs in `docs/invariants/`.
- `/audit-spec-coverage docs/specs/queue.md docs/invariants/queue*.md` —
  narrower glob for a single module's invariant set.
- `/audit-spec-coverage` — prompt for the spec path; do not guess.
