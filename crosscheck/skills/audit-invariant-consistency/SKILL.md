---
name: audit-invariant-consistency
description: >-
  Layer 5 + 6 hybrid consistency audit. Given a glob of invariant docs and
  an optional spec, runs three passes (within-module contradictions,
  cross-module contradictions, invariant-vs-spec contradictions) and emits
  a capped, prioritised findings file with 4-path triage blocks
  (`Accept (amend spec)` is first-class). Doc-wide pre-test probe — distinct
  from /intent-check's single-invariant-with-test post-test probe.
  Independently useful as a standalone audit; also consumed by
  add-orchestrator as the consistency leg of the spec-driven fast path.
  Triggers: "are these invariants consistent", "find contradictions",
  "consistency audit", "invariant contradictions".
argument-hint: "<invariant-docs-glob> [spec-path]"
---

# /audit-invariant-consistency — Cross-Doc Consistency Audit (Layer 5 + 6)

## Description

Three failure modes hide in any set of invariant docs:

1. **Within-module contradictions.** Two invariants in the same module
   doc constrain the same object incompatibly. The doc compiles to prose
   that reads cleanly; the constraints fight on the page.
2. **Cross-module contradictions.** Two invariants in different module
   docs use the same domain noun with incompatible meanings, or describe
   the same constraint with disagreeing strength. Each module reads fine
   in isolation; the system as a whole is incoherent. **This is the
   load-bearing case.**
3. **Invariant-vs-spec contradictions.** An invariant's English statement
   contradicts the prose of its anchoring spec section. The PR #94 `_FILE`-wins
   resolution between the ngst issue spec and `CONVENTIONS.md` is the
   canonical worked example: the invariant draft surfaced a contradiction
   that neither doc had noticed.

All three are silent until someone audits cross-doc. This skill runs that
audit.

### Layer classification

This is a Layer 5 + Layer 6 hybrid skill:

- The invariant-vs-spec pass is **Layer 5** (spec–intent alignment) —
  doc-wide pre-test, distinct from `/intent-check`'s single-invariant +
  covering-test + diff post-test probe.
- The within-module and cross-module passes are **Layer 6** (spec
  completeness — invariant set internal consistency).

Layer 5 vs Layer 6 distinction matters for kill criteria and best-effort
honesty: see Step 9 below.

### Distinct from /intent-check

| Skill | Inputs | Trigger time | Pass shape |
|---|---|---|---|
| `/intent-check` | One invariant + one covering test + one code diff | Post-test, per-PR | Two-LLM round-trip informalisation (probabilistic, FP-tracked, 30% kill criterion) |
| `/audit-invariant-consistency` | Glob of invariant docs + optional spec | Pre-test, doc-wide | Three-pass static cross-reference (best-effort, severity-ranked) |

Use `/intent-check` to verify *one* invariant captures intent of *one*
test/diff. Use this skill to verify the *set* of invariants is
self-consistent and consistent with the spec.

## When to invoke

Trigger phrases: `"are these invariants consistent"`, `"find contradictions"`,
`"consistency audit"`, `"invariant contradictions"`, `"audit invariants
for inconsistencies"`, `"do my invariants contradict each other"`.

Typical invocations:

- `/crosscheck:audit-invariant-consistency docs/invariants/*.md` —
  within-module + cross-module passes only; no spec supplied.
- `/crosscheck:audit-invariant-consistency docs/invariants/*.md docs/design/<repo>-spec.md`
  — all three passes including invariant-vs-spec.
- Called by `add-orchestrator` step 7 as the consistency leg of the
  spec-driven fast path.

## Methodology (execute in order)

### Step 1: Resolve inputs

- **Invariant-docs glob** (required, positional 1). Refuse if zero
  matches.
- **Spec path** (optional, positional 2). If supplied, validate the file
  exists. If not supplied, skip the invariant-vs-spec pass and note in the
  output that the third pass was skipped.

If the orchestrator marker file `.assurance/add-session-*/session.json`
exists AND its `spec_path` matches the supplied spec, write to
`.assurance/add-session-<id>/findings-consistency.md`; otherwise write
to `<cwd>/findings-consistency.md`. Detect via filesystem.

### Step 2: Build the vocabulary map

For every invariant doc:

- Enumerate every `IN.` heading (`I1`, `I2`, `I1a`, etc.).
- For each invariant, extract:
  - Statement (the one-paragraph body)
  - `Why:` text (often cites a spec section or audit-finding ID)
  - Domain nouns used (proper nouns, bolded terms, terms defined in the
    doc's `Contract` or glossary section)
  - Quantifiers and modal verbs (`for all`, `there exists`, `must`,
    `should`, `never`, `always`)

Build:

- **Per-module noun set** — `{module → {noun: [IN_ids that use it]}}`
- **Cross-module shared-noun set** — `{noun: [(module, IN_id) pairs]}`
  for every noun that appears in ≥ 2 modules
- **Statement index** — `{(module, IN_id): statement_text}` for evidence
  citation

This map drives Steps 3–5; do not skip it.

### Step 3: Within-module pass

For each invariant doc, find pairs of invariants `(IN_a, IN_b)` in the
same doc where:

- They reference the same domain object (same noun in the statement).
- Their constraints are incompatible:
  - Different quantifiers on the same property (`for all X` vs
    `there exists X`).
  - Disagreeing modal strength on overlapping conditions (`must always`
    vs `must never`).
  - Overlapping but contradictory state transitions (`terminal states
    are immutable` vs `failed states can transition to pending`).

For each contradictory pair, classify the sub-category:

- `incompatible_quantifiers`
- `disagreeing_modals`
- `state_transition_conflict`
- `overlapping_scope_disagreement`

Confidence levels:

- **HIGH** — both statements literally contain the contradicting
  language; no interpretation needed.
- **MEDIUM** — the contradiction depends on a domain-noun definition that
  both invariants share but neither states.
- **LOW** — plausible contradiction under one reading; not under
  another.

Skip LOW contradictions in the output unless fewer than 5 HIGH/MEDIUM
findings exist across all passes.

### Step 4: Cross-module pass (LOAD-BEARING)

For every shared noun in the cross-module noun set:

- Collect every `(module, IN_id)` that uses the noun.
- Read each invariant's statement; classify how the noun is used:
  - **Definition** — the invariant defines a property of the noun.
  - **Constraint** — the invariant constrains how the noun behaves.
  - **Reference** — the invariant mentions the noun but doesn't constrain
    it.

Compare definitions across modules:

- If two modules define the same noun differently (different states,
  different boundaries, different lifecycle), flag as a
  `cross-module-definition-disagreement`.
- If one module constrains the noun in a way that another module's
  constraint forbids, flag as a `cross-module-constraint-conflict`.

This pass is the load-bearing case for the skill. Most cross-doc bugs
land here: each module's author was correct in isolation; the system
becomes incoherent in composition.

Example (illustrative, not literal):

```
Cross-module-definition-disagreement: `expired`

- dispatcher.md I3: "A run is `expired` if its deadline has passed."
- canceller.md I7: "A run is `expired` if its failure-budget has been
  tripped."

The same noun names two different conditions. Code reading
`dispatcher.IsExpired(r)` and `canceller.IsExpired(r)` will get
different answers; tests written against one definition will not catch
violations of the other.
```

Same confidence levels (HIGH/MEDIUM/LOW) as Step 3. Same skip-LOW rule.

### Step 5: Invariant-vs-spec pass (skip if no spec)

For each invariant:

- Locate its anchoring spec section. Signals (in priority order):
  - `Why:` text contains a section number (`§3.2`, `Section 3.2`,
    `spec §3.2`).
  - `Why:` text contains an audit-finding ID that maps to a section
    (require the spec's audit-finding table to make this mapping).
  - Domain-noun overlap with a single section (high-confidence match
    only — ≥ 3 shared domain nouns).
- If no anchoring section can be identified, skip the invariant for this
  pass and record `(module, IN_id)` in the "unmapped invariants" list at
  the bottom of the output.

For mapped invariants:

- Read the anchoring section in full.
- Check whether the invariant's statement is consistent with the
  section's prose:
  - **Contradiction.** Invariant asserts `MUST X`; section asserts `MUST
    NOT X`. Flag as `invariant-contradicts-spec`.
  - **Over-strengthening.** Invariant asserts `MUST X`; section asserts
    `SHOULD X`. Flag as `invariant-stronger-than-spec` (often
    intentional, but worth surfacing for review).
  - **Under-strengthening.** Invariant asserts `SHOULD X`; section
    asserts `MUST X`. Flag as `invariant-weaker-than-spec` (usually a
    bug).
  - **Scope mismatch.** Invariant constrains a sub-case the section
    does not address, or skips a sub-case the section requires. Flag
    as `invariant-scope-mismatch`.

Cite paired evidence in every finding:

- Invariant: `<module>/<doc>:<IN_id>` plus quote.
- Spec: `<spec-path>:<line-range>` plus quote.

### Step 6: Aggregate and cap

Combine findings from all three passes. **Cap total output at 15
findings** — same reviewer-fatigue rationale as `/audit-spec-coverage`
and `/spec-adversary`.

Prioritisation rules (descending priority):

1. Invariant-vs-spec contradictions (Layer 5, highest signal).
2. Cross-module-constraint-conflict (the load-bearing Layer 6 case).
3. Cross-module-definition-disagreement.
4. Within-module state-transition conflicts.
5. Within-module incompatible quantifiers.
6. All other categories at HIGH confidence.
7. Invariant-stronger-than-spec / invariant-weaker-than-spec.
8. MEDIUM confidence findings.
9. Scope mismatches.
10. LOW confidence findings (only if total < 5).

If more than 15 candidates pass the priority filter, drop the lowest and
record: `... and N more lower-priority findings omitted; re-run with a
narrower glob to see them`.

### Step 7: Emit findings-consistency.md

Schema:

```markdown
---
session: <id or "standalone">
category: consistency
generated_at: <YYYY-MM-DDTHH:MM:SSZ>
invariant_glob: <glob>
spec_path: <path or "(none supplied)">
total_findings: <n>
passes_run: within-module, cross-module, invariant-vs-spec
---

# Findings: invariant consistency

## Summary

- Invariant docs audited: <N>
- Invariants enumerated: <N>
- Shared domain nouns (≥ 2 modules): <N>
- Within-module findings: <N>
- Cross-module findings: <N>
- Invariant-vs-spec findings: <N or "pass skipped (no spec)">
- Total findings (capped at 15): <N>
- Findings omitted as lower-priority: <N or "0">

## Within-module contradictions

### F1: <short name, e.g. "queue.md I3 vs I7 — terminal-state immutability">
**Severity:** Blocker | High | Medium | Low
**Sub-category:** incompatible_quantifiers | disagreeing_modals | state_transition_conflict | overlapping_scope_disagreement
**Confidence:** HIGH | MEDIUM | LOW
**Evidence:**
- `<module>/<doc>:<IN_a>` — "<verbatim statement quote>"
- `<module>/<doc>:<IN_b>` — "<verbatim statement quote>"
**Why this matters:** <2-3 sentences naming the bug class this admits>
**Proposed resolution:** <one sentence; typically "reword <IN_a> to clarify <X>" or "split <IN_b> into sub-invariants">

**Triage (mark exactly one):**
- [ ] Accept (fix invariant) — <one-line fix description>
- [ ] Accept (amend spec via /protected-surface-amend) — <one-line note on spec edit>
- [ ] Reject — <reason>
- [ ] Defer — <revisit condition>

### F2: ...

## Cross-module contradictions

### F<n>: <short name>
**Severity:** ...
**Sub-category:** cross-module-definition-disagreement | cross-module-constraint-conflict
**Confidence:** ...
**Evidence:**
- `<module_a>/<doc>:<IN_id>` — "<quote>"
- `<module_b>/<doc>:<IN_id>` — "<quote>"
- Shared noun: `<noun>` used in <module_a> as `<sense_a>`, in <module_b> as `<sense_b>`
**Why this matters:** ...
**Proposed resolution:** ...

**Triage (mark exactly one):**
- [ ] Accept (fix invariant) — <one-line fix description>
- [ ] Accept (amend spec via /protected-surface-amend) — <one-line note on spec edit>
- [ ] Reject — <reason>
- [ ] Defer — <revisit condition>

## Invariant-vs-spec contradictions

(If no spec supplied: "Pass skipped — no spec path provided.")

### F<n>: <short name>
**Severity:** ...
**Sub-category:** invariant-contradicts-spec | invariant-stronger-than-spec | invariant-weaker-than-spec | invariant-scope-mismatch
**Confidence:** ...
**Evidence:**
- Invariant: `<module>/<doc>:<IN_id>` — "<quote>"
- Spec: `<spec-path>:<line-range>` — "<quote>"
**Why this matters:** ...
**Proposed resolution:** <one sentence; for `invariant-contradicts-spec`, options are "fix invariant" or "amend spec — sometimes the spec is the one that's wrong; see PR #94 _FILE-wins resolution">

**Triage (mark exactly one):**
- [ ] Accept (fix invariant) — <one-line fix description>
- [ ] Accept (amend spec via /protected-surface-amend) — <one-line note on spec edit>
- [ ] Reject — <reason>
- [ ] Defer — <revisit condition>

## Unmapped invariants

(Invariants whose anchoring spec section could not be identified; review
manually if the invariant-vs-spec pass was expected to cover them.)

- `<module>/<doc>:<IN_id>` — no spec section identified (cite reason: no
  section citation in Why, no audit-finding ID, insufficient noun overlap)
```

Severity rubric:

- **Blocker** — `invariant-contradicts-spec` at HIGH confidence on a
  spec `MUST` constraint, OR `cross-module-constraint-conflict` at HIGH
  confidence where both modules ship to the same binary.
- **High** — `cross-module-definition-disagreement` at HIGH confidence,
  OR within-module `state_transition_conflict` at HIGH confidence.
- **Medium** — MEDIUM confidence findings of any sub-category, OR
  `invariant-stronger-than-spec`.
- **Low** — `invariant-scope-mismatch`, `invariant-weaker-than-spec` on
  `SHOULD` constraints, LOW confidence findings.

Sort within each pass by severity (Blocker first); secondary sort
alphabetical by module name.

### Step 8: Required "What this does NOT catch" section

Append verbatim:

```markdown
## What this does NOT catch

This skill probes structural consistency. It cannot detect:

1. **Semantic-equivalence contradictions where the prose uses different
   words for the same constraint.** If two invariants assert the same
   property in different vocabulary, the skill sees them as unrelated
   and does not flag redundancy. This is a coverage probe failure, not
   a consistency probe failure, but it is worth knowing.
2. **Contradictions internal to the spec itself.** If the spec is
   self-contradictory (e.g. §3.2 says `X MUST` and §7.4 says `X MUST
   NOT`), this skill will compare invariants to each section
   independently and see consistency. Detecting spec-internal
   contradictions is out of scope — the spec is treated as authoritative
   for each section. Run a separate spec-internal-consistency review if
   the spec is large or has been edited by multiple authors.
3. **Contradictions that depend on running the code.** If two invariants
   are statically consistent but produce conflicting runtime behaviour
   under specific inputs, this skill will not catch them. Use
   `/intent-check` post-test or `/spec-adversary` for code-vs-doc
   probes.
4. **Cross-pair compositional contradictions.** A chain `I1 → I2 → I3`
   may be inconsistent at the chain level even if each pairwise check
   passes. Chains of length > 2 are not analysed.
5. **Domain-noun aliasing.** If two modules use different nouns for
   the same concept (`run` vs `job`), the cross-module pass will not
   detect that they refer to the same thing. Aliasing detection is
   beyond static text analysis.

The skill is best-effort by design. Layer 6 has no theorem for
"completeness of consistency checking"; the goal is high-signal
findings, not exhaustive coverage.
```

### Step 9: Kill criteria

This skill is Layer 5/6 best-effort:

- **Signal-to-noise < 1:5 after 4 runs** → the cross-module domain-noun
  threshold may be too loose; recalibrate.
- **Zero findings on an invariant set with ≥ 20 invariants across ≥ 3
  modules sharing visible domain nouns** → suspect a bug; the
  vocabulary map is probably not detecting shared nouns.

If a tracker file `.assurance/audit-invariant-consistency-tracker.md`
exists, append a one-line summary (date, counts, accepted count once
filled in by review). If not, do not create — tracker creation is the
user's choice.

## Output structure

Single `findings-consistency.md` file at:

- Orchestrator mode: `.assurance/add-session-<id>/findings-consistency.md`
- Standalone mode: `<cwd>/findings-consistency.md`

The skill writes the file and prints its path. It does NOT modify any
invariant doc or the spec. Triage and apply are downstream (manual or via
`add-orchestrator` step 10, including `/protected-surface-amend` invocation
when `Accept (amend spec)` triages).

## Checklist before handing off

- [ ] Invariant glob resolved to at least one file
- [ ] Spec path validated if supplied; pass skipped explicitly if not
- [ ] Vocabulary map built; per-module and cross-module noun sets
      populated
- [ ] Within-module pass executed for every doc
- [ ] Cross-module pass executed for every shared-noun pair
- [ ] Invariant-vs-spec pass executed (or stated skipped)
- [ ] Each finding has paired file:line evidence
- [ ] 4-path triage block per finding (with `Accept (amend spec)` as a
      first-class option, especially for invariant-vs-spec contradictions)
- [ ] ≤ 15 findings emitted (with overflow note if applicable)
- [ ] Unmapped invariants list present (or stated empty)
- [ ] "What this does NOT catch" section present
- [ ] Output written to correct path

## Arguments

1. **Invariant docs glob** (required) — glob pattern matching the
   invariant docs to audit.
2. **Spec path** (optional) — repo-relative or absolute path. Enables the
   invariant-vs-spec pass when supplied.

Examples:

- `/audit-invariant-consistency docs/invariants/*.md` — within-module +
  cross-module passes only.
- `/audit-invariant-consistency docs/invariants/*.md docs/design/fabricator-spec.md`
  — all three passes against the ngst spec.
- `/audit-invariant-consistency docs/invariants/queue*.md` — narrow glob
  for the queue module family.
- `/audit-invariant-consistency` — prompt for the glob; do not guess.
