# M2-greenfield-skills — Functional Spec

```yaml
---
id: M2-greenfield-skills
mode: add
phase: 1
status: Drafted (re-drafted v1.1 per Phase 2 seam validation, Bucket A)
consumes: [IC1, IC2, IC3, IC4, IC7, IC11, S2.1, S2.2, S2.3, S2.4, S2.5, ADR-004, ADR-005, "skills/intent-check/SKILL.md (inherited)", "skills/spec-adversary/SKILL.md (inherited)", M2-greenfield-skills/B1, M2-greenfield-skills/B2, M2-greenfield-skills/B3, M2-greenfield-skills/B4, M2-greenfield-skills/B5, M2-greenfield-skills/B6]
produces: [F2.1, F2.2, F2.3, F2.4, F2.5, F2.6, F2.7, F2.8, I1, I2, I3, I4, I5, I6, I7, T2.1..T2.16]
last-attested: N/A (Drafted)
---
```

## Purpose

Per-operation functional specs for the **greenfield-skills** module: the four new skills (`/intent-elicit`, `/spec-derive`, `/intent-check-prose`, `/spec-adversary-prose`) plus the `/spec-iterate` seam declared in S2.5. The module owns the agent-facing surface of Phase 0 → 1 → 2 in the ADD lifecycle.

Operations consume `M1-mode-governance/F1.3 (mode-of)` whenever a skill needs to know whether the surrounding repo is in ADD mode. The shared mode resolver guarantees `I3` of M1 (default uniformity).

Skills produced here are SKILL.md files at `skills/<skill-name>/SKILL.md`. The operations below specify the *behavior contracts* of those skills, not the prompt strings — prompts are a SKILL-md-tier concern the agent drafts when it produces each SKILL.md (next phase of work after this functional spec is attested).

---

## Data shapes

### `ICRecord`

```
ICRecord := {
  id: String matching ^IC[1-9][0-9]*$,
  status: { Drafted, Attested, Ratified, "Superseded-by-N", "Retracted-with-Reason" },
  observable_signal: NonEmptyString,
  context: String,
  decision: String,
  alternatives: List<{ label, rationale }>,
  consequences: String
}
```

### `SRecord`

```
SRecord := {
  id: String matching ^S[1-9][0-9]*(\.[0-9]+)*$,
  status: ...,
  consumes: List<ID-of-IC-or-ADR-or-S>,
  produces: List<ID-of-B-or-F>,
  body: String,
  alternatives: List<{ label, rationale }>
}
```

### `IntentDoc`

```
IntentDoc := {
  vision: String,
  intent_claims: List<ICRecord>,
  out_of_scope: List<{ id: String matching ^N[1-9][0-9]*$, body: String }>,
  threat_model: List<{ id: String matching ^TM[1-9][0-9]*$, body: String }>,
  status: ...
}
```

### `BackTranslation`

Two-section output, inherited from `skills/intent-check/SKILL.md` § Step 2.

```
BackTranslation := {
  written_at: Timestamp,
  spec_stack_inputs: List<ID-with-content-hash>,
  section_1_behavioural: NonEmptyString,         // prose: system the spec describes
  section_2_carve_outs: String,                  // verbatim quotes from "Not covered" / negative-space sections; "None." if absent
  intent_doc_in_context: false                   // mandatory; see I3
}
```

Section 2 quotes: scope markers (`Not covered`, `caller-responsibility`, `precondition`, `aspirational`, `known violation`, `privileged`, `exempt`, `out of scope`, `does not apply`) plus the intent doc's `N1`–`Nn` negative-space items if visible in the spec stack passed to the back-translator. If none, the section says `None.`.

### `DiffCheckerOutput`

JSON schema inherited verbatim from `/intent-check`.

```json
{
  "match": true | false,
  "mismatch_reason": "string",
  "mismatch_category": "spec_scope_mismatch | weaker_guarantee | missing_property | missing_coverage | rationale_explains | carve_out_applies | clean_match",
  "confidence_pct": 0-100,
  "confidence_basis": "carve-out-found | rationale-found | rationale-absent | spec-ambiguous | code-ambiguous"
}
```

### `ComparisonReport`

Markdown rendering for humans; pairs with the JSON attestation (see F2.8).

```
ComparisonReport := {
  matches: List<{ ic_id, status: Full | Partial | Divergent | Silent }>,
  gaps_in_intent: List<Finding>,
  gaps_in_spec: List<Finding>,
  contradictions: List<Finding>,
  underspecifications: List<Finding>,
  drift_candidates: List<Finding>,
  verdict: PASS | PASS_WITH_AMENDMENTS | HOLD,
  fp_tracker_path: AbsolutePath,
  attestation_path: AbsolutePath
}
```

### `FPRecord`

Schema inherited verbatim from `skills/intent-check/SKILL.md` § Step 5 (`references/fp-tracker-schema.md`). Stable across consumers (`/intent-check`, `/intent-check-prose`, `/assurance-status`) so cross-consumer rates are computed identically.

```
FPRecord := {
  date: Date,                                                  // YYYY-MM-DD
  intent_doc_or_section: NonEmptyString,                       // short label, e.g., "intent.md IC4 vs S2.3 (back-translation gap)"
  phase_verdict: pass | fail,                                  // pass iff match==true AND confidence_pct>=80
  human_verdict: "" | genuine | genuine-planted | partial | spurious  // empty at skill-run time; human fills in later
}
```

Note: column 2 is `intent_doc_or_section` for the prose variant (parallel to `invariant_touched` in `/intent-check`'s tracker). Column count and types are byte-identical so consumers can union the two CSVs for cross-mode rate analysis.

### `Attestation`

Schema inherited verbatim from `skills/intent-check/SKILL.md` § Step 6.

```json
{
  "protected_files": ["…sorted…"],
  "content_hash": "<64-hex-chars-sha256>",
  "verdict": "pass" | "fail",
  "checked_at": "<RFC3339>",
  "pipeline_output": {
    "back_translation": "Section 1 verbatim\\nSection 2 verbatim",
    "diff_result": { ...DiffCheckerOutput post-validation... }
  }
}
```

### `CommitContext`

```
CommitContext := {
  modified_paths: List<AbsolutePath>,
  proposed_classification: ClassificationClass,    // one of the five
  justification: String | None,
  related_ids: List<ID>
}
```

---

## F2.1 — `intent-elicit-attestation-guard(human_message: String, current_status: Status) → AttestationDecision`

```yaml
---
id: F2.1
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B1, IC2, S2.1, TM5]
produces: [I1, T2.1, T2.2]
---
```

### Signature
`intent-elicit-attestation-guard(human_message: String, current_status: Status) → AttestationDecision`

`AttestationDecision := AllowAttest | RefuseAttest { reason: String }`

### Preconditions
- `current_status` is the current `Status:` field of `docs/add/intent.md`.
- `human_message` is the most recent user message in the same exchange.

### Postconditions

Returns `AllowAttest` iff ALL of:
1. `current_status` is `Drafted`.
2. `human_message` matches at least one of the explicit-confirmation phrases enumerated below (case-insensitive substring match against the human's last message):
   - `"i attest"`
   - `"attested"` (when used by the human themselves; the skill must distinguish the human's voice from quoted agent output)
   - `"flip status to attested"`
   - `"mark attested"`
   - `"phase 0 attestation"` (when paired with affirmative language in the same message)
3. The skill's previous turn included a "ready for attestation" prompt offering this transition.

Returns `RefuseAttest { reason }` otherwise. The `reason` is human-readable and explains which of (1), (2), (3) failed.

### Frame conditions
- Reads only the human's most recent message and the current intent doc Status.
- Does not modify the intent doc.

### Module invariants preserved
- I1 (human-attestation-required for status promotion).

### Test linkage
- T2.1 — human says "i attest after reading IC1..IC10", expect `AllowAttest`.
- T2.2 — human says nothing about attestation (just answers the previous question), expect `RefuseAttest { reason: "no attestation phrase in human message" }`.

### Implementation discipline note
The phrase list is deliberately narrow. False positives (auto-attesting on a confused affirmative) are worse than false negatives (the human re-asks). Per ADR-004's open question on `/intent-elicit`'s human-attestation commit shape, the skill emits a Drafted intent doc only; the actual Drafted → Attested transition is committed *by the human* via a status-transition commit per ADR-005. The skill never makes that commit.

---

## F2.2 — `spec-derive-IC-coverage-check(intent: IntentDoc, spec: List<SRecord>) → CompletionVerdict`

```yaml
---
id: F2.2
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B2, IC3, IC11, S2.2]
produces: [I2, T2.3, T2.4]
---
```

### Signature
`spec-derive-IC-coverage-check(intent: IntentDoc, spec: List<SRecord>) → CompletionVerdict`

`CompletionVerdict := Complete | Incomplete { unconsumed_ics: List<String> }`

### Preconditions
- `intent.status` is `Attested` (the skill refuses to derive specs from a Drafted intent doc).
- `spec` is the list of all `S` sections in the Drafted architectural spec.

### Postconditions

Let `consumed_ics := union over s in spec of (filter s.consumes for entries matching ^IC[0-9]+$)`.

Returns `Complete` iff `consumed_ics ⊇ { ic.id for ic in intent.intent_claims }`.

Returns `Incomplete { unconsumed_ics }` otherwise, where `unconsumed_ics := { ic.id | ic ∈ intent.intent_claims, ic.id ∉ consumed_ics }`.

### Frame conditions
- Reads `intent` and `spec`. No external I/O.

### Module invariants preserved
- I2 (every IC consumed by ≥1 S in any emitted architectural spec).

### Test linkage
- T2.3 — intent has IC1..IC11; spec consumes IC1..IC10 only; expect `Incomplete { unconsumed_ics: ["IC11"] }`.
- T2.4 — intent has IC1..IC11; spec consumes all; expect `Complete`.

### Behavior contract
The skill `/spec-derive`:
- Calls this predicate before emitting the spec.
- On `Incomplete`, returns the spec marked `Status: Drafted` with a top-of-file notice listing `unconsumed_ics`. Does not commit.
- On `Complete`, the spec is emitted with the `Status: Drafted` marker; the human attests separately (cf. F2.1's pattern).

---

## F2.3 — `intent-check-prose-back-translate(spec: List<SRecord>) → BackTranslation` (blindness + two-section contract)

```yaml
---
id: F2.3
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B3, IC4, S2.3, "skills/intent-check/SKILL.md § Step 2"]
produces: [I3, T2.5, T2.6, T2.7]
---
```

### Signature
`intent-check-prose-back-translate(spec: List<SRecord>) → BackTranslation`

### Preconditions
- `spec` is the Drafted architectural spec (and any deeper-tier specs the run is checking).
- The skill's invocation is gated by a build-time check (lint rule on the prompt template source) that the back-translator template does not include `intent.md` content references.

### Postconditions

Returns a `BackTranslation` value with all required fields populated:
- `written_at = now`.
- `spec_stack_inputs` lists each `S | B | F` ID and its content hash.
- `section_1_behavioural` is non-empty prose describing the system the spec specifies (parallel to `/intent-check`'s "behavioural guarantees" section).
- `section_2_carve_outs` quotes verbatim every "Not covered" / negative-space / scope-marker passage in the spec stack, with file:section references. If none exist, the section says exactly `None.`.
- `intent_doc_in_context = false` is guaranteed by the build-time check.

If either section is missing or empty (other than the explicit `None.`), re-invoke the prompt once. If still malformed on the second try, fail the run and surface the malformed output.

### Frame conditions
- Reads only `spec`.
- The implementer MUST partition prompts: the back-translator prompt receives `spec` only; the diff-checker prompt (F2.4 below) receives `(intent, back_translation)`. The two MUST run in separate model invocations to prevent context-bleed.

### Module invariants preserved
- I3 (back-translation prompt is blind to intent).
- I7 (two-section structure inherited from `/intent-check`).

### Test linkage
- T2.5 — invoke back-translator with a spec that mentions "auditor agent"; `section_1_behavioural` contains "auditor" or synonym.
- T2.6 — build-time check: a code path that adds intent to the back-translator's context fails the check (violation detected at skill-compile-time).
- T2.7 — invoke against a spec containing a `## Negative space (out of scope)` block; `section_2_carve_outs` quotes those lines verbatim.

### Implementation discipline note
The blindness contract is enforced **structurally** (separate prompts, separate model invocations) not by convention. Build-time enforcement means a developer modifying the skill's prompt template cannot accidentally regress this property; the build fails. The two-section discipline mirrors `/intent-check`'s existing pattern verbatim (per S2.3 inheritance declaration); changes to the section structure require a follow-up ADR.

---

## F2.4 — `intent-check-prose-compare(intent: IntentDoc, back_translation: BackTranslation) → DiffCheckerOutput`

```yaml
---
id: F2.4
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B3, M2-greenfield-skills/B4, IC4, S2.3, "skills/intent-check/SKILL.md § Step 3 and § Step 4"]
produces: [I4, I7, T2.8, T2.9]
---
```

### Signature
`intent-check-prose-compare(intent: IntentDoc, back_translation: BackTranslation) → DiffCheckerOutput`

### Preconditions
- `back_translation` was produced by F2.3 in a prior, separate model invocation (the two prompts MUST NOT share state).
- The diff-checker prompt's first internal step is the **mandatory carve-out scan**: scan both the intent doc's `N1`–`Nn` negative-space items AND `back_translation.section_2_carve_outs` for scope markers (`Not covered`, `caller-responsibility`, `precondition`, `aspirational`, `known violation`, `privileged`, `exempt`, `out of scope`, `does not apply`). Classify each found clause by the scope-modifier taxonomy. Only after the scan does the prompt evaluate apparent gaps.

### Postconditions

Returns a `DiffCheckerOutput` value satisfying the inherited JSON schema. After parsing the raw diff-checker output, apply two fail-closed semantic-validation rules verbatim from `/intent-check` § Step 4:

1. **Contradictory output.** If `match == true` AND `mismatch_reason` is non-empty and non-trivial, flip to:
   ```
   match = false
   confidence_pct = 40
   confidence_basis = "spec-ambiguous"
   mismatch_category = "missing_property"
   ```
   Record a note that the raw output was internally contradictory.
2. **Truncated reason.** If `match == false` AND `len(strip(mismatch_reason)) < 20`, reject the output as truncation. Do not write a tracker row or attestation. Ask the user to re-run.

### Frame conditions
- Reads `intent` and `back_translation` only. No mutation.

### Module invariants preserved
- I4 (FP-rate kill criterion is computable from this output).
- I7 (mandatory carve-out scan + fail-closed validation; inherited from `/intent-check`).

### Test linkage
- T2.8 — diff-checker raw output `{match: true, mismatch_reason: "but the spec misses the audit case"}` → semantic validation flips to `{match: false, confidence_pct: 40, ...}`.
- T2.9 — diff-checker raw output `{match: false, mismatch_reason: "yes"}` → rejected as truncated; no tracker row, no attestation.

---

## F2.4b — `intent-check-prose-fp-tracker-update(diff_result: DiffCheckerOutput, intent_doc_or_section: String, fp_csv_path: AbsolutePath) → FPState`

```yaml
---
id: F2.4b
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B4, IC4, S2.3, "skills/intent-check/SKILL.md § Step 0 and § Step 5"]
produces: [I4, T2.10, T2.11]
---
```

### Signature
`intent-check-prose-fp-tracker-update(diff_result, intent_doc_or_section, fp_csv_path) → FPState`

`FPState := { rolling_rate: Float in [0.0, 1.0], status: ACTIVE | AT_RISK | TRIPPED, classified_count: Integer, window_days: Integer }`

### Configuration (inherited verbatim from `/intent-check` § Configuration)

| Env var | Default | Meaning |
|---|---|---|
| `CROSSCHECK_FP_TRIPPED_THRESHOLD` | `0.30` | Rolling FP rate at which Step 0 of `/intent-check-prose` refuses to run |
| `CROSSCHECK_FP_AT_RISK_THRESHOLD` | `0.20` | Rolling FP rate at which the verdict reports `AT RISK` |
| `CROSSCHECK_FP_WINDOW_DAYS` | `14` | Rolling-window length in days |

Minimum sample: `n ≥ 3` classified rows in the window. Below that, treat the rate as unknown and proceed with a warning. The same env vars are read by `/intent-check`, `/intent-check-prose`, and `/assurance-status`; do not introduce parallel env vars.

### Preconditions
- `diff_result` is the output of F2.4 (post-validation).
- `fp_csv_path` defaults to `.assurance/intent-check-prose-fp-tracker.csv`. Created if missing with header `date,intent_doc_or_section,phase_verdict,human_verdict`.

### Postconditions

Append exactly one row to `fp_csv_path` with columns:
- `date` — today in `YYYY-MM-DD`.
- `intent_doc_or_section` — short label identifying what was checked (e.g., `intent.md IC4 vs S2.3`).
- `phase_verdict` — `pass` if `diff_result.match == true` AND `diff_result.confidence_pct >= 80`; `fail` otherwise.
- `human_verdict` — empty at skill-run time.

Compute the rolling rate over the last `CROSSCHECK_FP_WINDOW_DAYS` days of entries with non-empty `human_verdict`:
- FP rate = count(`human_verdict == spurious`) / count(rows in window with non-empty `human_verdict`).

Status (mirrors `/intent-check` Step 0 semantics):
- `TRIPPED` iff rate `>` tripped threshold AND classified_count `>= 3`. Skill refuses to run on next invocation.
- `AT_RISK` iff rate `>` at-risk threshold AND `<=` tripped threshold AND classified_count `>= 3`.
- `ACTIVE` otherwise (including unknown-rate cases when classified_count `< 3`).

No marker file is written (per `/intent-check`'s pattern: the CSV itself is the source of truth; refusal is computed at skill invocation time).

### Frame conditions
- Appends to `fp_csv_path`. No other side effects.

### Module invariants preserved
- I4 (kill criterion enforced via Step 0 pre-check at next invocation).

### Test linkage
- T2.10 — append rows totalling 4 spurious / 12 classified within 14 days; rolling_rate = 0.33; status = TRIPPED.
- T2.11 — append rows totalling 2 spurious / 12 classified within 14 days; rolling_rate = 0.17; status = ACTIVE (below at-risk).

---

## F2.5 — `spec-adversary-prose-probe(spec_section: SRecord) → AdversaryReport`

```yaml
---
id: F2.5
status: Drafted
implementation: manual
consumes: [S2.4, M2-greenfield-skills/B6, "skills/spec-adversary/SKILL.md (inherited)"]
produces: [T2.12, T2.13]
---
```

### Signature
`spec-adversary-prose-probe(spec_section: SRecord) → AdversaryReport`

### Inheritance from `/spec-adversary`

This operation inherits `/spec-adversary`'s pipeline structure (Steps 1–7 of `skills/spec-adversary/SKILL.md`) verbatim, with one substitution: input is a Drafted spec section (`S | B | F`) rather than a ratified module's invariant doc + code.

Inherited verbatim:
- **Cap of 3 proposals per run** (per `/spec-adversary` § Step 3: *"Cap the output at 3 proposals. Reviewer fatigue is the dominant failure mode of this pattern; 3 high-signal proposals beat 15 noisy ones."*)
- **Confidence labels** `HIGH | MEDIUM | LOW`.
- **Category enum** (verbatim): `missing_property | tighter_bound | missing_precondition | missing_postcondition | missing_interaction`.
- **Radio-block triage format** for each proposal (Accept / Reject / Defer with one-line reason or revisit condition).
- **Kill criteria** (verbatim from `/spec-adversary` § Step 7):
  - Signal-to-noise < 1:5 after 4 weeks (fewer than 1 accepted proposal per 5 proposed) → scale back cadence or retire for this spec section.
  - No ratified proposals land within 8 weeks → strategy needs rework.
- **Tracker file** `.assurance/spec-adversary-prose-tracker.md` (parallel to `/spec-adversary`'s `.assurance/spec-adversary-tracker.md`); same per-run section format:
  ```
  ## <YYYY-MM-DD> — <spec_section.id>
  Proposed: N
  Accepted: N
  Rejected: N
  Deferred: N

  ### Accepted / Rejected / Deferred
  - <name> — <reason>.
  ```

### Output: `AdversaryReport`

```
AdversaryReport := {
  proposals: List<Proposal>,                    // length 0..3
  signal_to_noise_self_check: SnrSelfCheckResult,
  tracker_kill_criteria_status: Active | At-Risk | Tripped
}

Proposal := {
  short_name: String,
  candidate_invariant_prose: NonEmptyString,
  category: missing_property | tighter_bound | missing_precondition | missing_postcondition | missing_interaction,
  confidence: HIGH | MEDIUM | LOW,
  rationale: NonEmptyString,                    // 2-4 sentences tying to spec content
  supporting_spec_lines: List<{ ref: SectionRef, reason: String }>,
  adjacent_artifacts: List<ID>,                 // sibling S/B/F sections; declare deltas
  triage_block: RadioBlockMarkdown              // verbatim radio format
}
```

### Preconditions
- `spec_section.status == Drafted` (the skill refuses to probe Ratified sections; per S2.4 and `/spec-adversary` § Step 1, those go through full consolidation).

### Postconditions

Returns an `AdversaryReport` containing:
- 0–3 proposals, each carrying full triage block.
- The signal-to-noise self-check explaining why each proposal is signal not noise.
- The kill-criteria status read from the tracker file's recent runs.

The skill **does not modify any artifact**. It produces probing output the human or Hellebuyck uses to amend (via `/protected-surface-amend` for the spec stack). Promotion is a separate PR (mirrors `/spec-adversary` § Step 6).

### Frame conditions
- Reads `spec_section` only. Reads the tracker file for kill-criteria status. Does not write the tracker — that happens later, after human triage (per `/spec-adversary` § Step 5).

### Module invariants preserved
- (probe-only-no-amend remains structural to the operation; not a separately tracked invariant.)

### Test linkage
- T2.12 — probe a deliberately under-specified spec section; expect ≥1 proposal at MEDIUM-or-higher confidence; ≤3 proposals total.
- T2.13 — probe a Ratified section; expect refusal (skill returns an error, not a report).

---

## F2.6 — `layer1-seam-honored(F_section: FSection) → IntegrityVerdict`

```yaml
---
id: F2.6
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B6, IC3, S2.5]
produces: [I5, T2.14, T2.15]
---
```

### Signature
`layer1-seam-honored(F_section: FSection) → IntegrityVerdict`

`FSection := SRecord-like with frontmatter including 'implementation:' and optional 'implementation-status:'`

### Preconditions
- `F_section.frontmatter.implementation` is one of `spec-iterate | lean-pipeline | manual | external`, or undeclared (defaults to `manual`).

### Postconditions

If `F_section.frontmatter.implementation == "spec-iterate"` (Dafny verify-and-extract):
- Returns `Ok` iff one of:
  - A `.dfy` artifact exists at `<module-dir>/verification/<F_section.slug>.dfy`, OR
  - `F_section.frontmatter.implementation_status` matches `^deferred-to-phase-[3-5]$`.
- Returns `Violation { kind: SpecIterateSeamUnhonored, location: F_section.id }` otherwise.

If `F_section.frontmatter.implementation == "lean-pipeline"` (Lean executable-model + DRT-oracle):
- Returns `Ok` iff one of:
  - ALL THREE artifacts exist:
    - `<module-dir>/lean/<F_section.slug>.lean`, AND
    - `.assurance/correspondence/<module>/<F_section.slug>.json` with `verdict: exact | abstraction | approximation` (any of these three; `mismatch` is a hard violation), AND
    - A DRT-oracle run record under `.assurance/drt-oracle/<module>/` referencing the same slug.
  - OR `F_section.frontmatter.implementation_status` matches `^deferred-to-phase-[3-5]$`.
- Returns `Violation { kind: LeanPipelineSeamUnhonored, location: F_section.id, missing: [<list-of-missing-artifacts>] }` otherwise.
- Returns `Violation { kind: LeanCorrespondenceMismatch, location: F_section.id }` if the correspondence verdict is `mismatch`.

If `F_section.frontmatter.implementation == "manual"` or `"external"`:
- Returns `Ok` (no Layer-1 seam to verify; standard integrity rules — invariant covered by test — still bind).

### Frame conditions
- Reads `F_section` and the module's verification directory listing.

### Module invariants preserved
- I5 (Layer-1 seam declared and integrity-checked, covering both spec-iterate and lean-pipeline).

### Test linkage
- T2.14 — F section with `implementation: spec-iterate` and matching .dfy → `Ok`. F section with `implementation: lean-pipeline` and all three Lean artifacts (with `verdict: exact`) → `Ok`.
- T2.15 — F section with `implementation: lean-pipeline` and `verdict: mismatch` in correspondence-review → `Violation::LeanCorrespondenceMismatch`. F section with `implementation: spec-iterate` and no .dfy, no deferred-to-phase note → `Violation::SpecIterateSeamUnhonored`.

---

## F2.7 — `greenfield-skill-emit-trailer(commit_ctx: CommitContext) → CommitMessage`

```yaml
---
id: F2.7
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B5, IC7, ADR-005]
produces: [I6, T2.13, T2.14]
---
```

### Signature
`greenfield-skill-emit-trailer(commit_ctx: CommitContext) → CommitMessage`

### Preconditions
- The greenfield skill (`/intent-elicit`, `/spec-derive`, `/intent-check-prose`, or `/spec-adversary-prose`) is producing a commit on behalf of the agent.
- `commit_ctx.modified_paths` includes at least one path under `docs/add/`.

### Postconditions

Returns a `CommitMessage` where:
- The body ends with the structured trailer block:
  ```
  Spec-Diff-Classification: <one of the five classes>
  Spec-Diff-Justification: <commit_ctx.justification or "" if not drift>
  ```
- For `propagated-discovery` and `intent-refinement` classifications, `Spec-Diff-Justification` is optional but if non-empty must be a single line.
- For `drift`, `Spec-Diff-Justification` is mandatory and must answer the canonical question (per ADR-005 § Canonical question).
- For `retraction` and `status-transition`, `Spec-Diff-Justification` is required and describes the rationale.

If the trailer would be malformed (missing classification, value not in legal set), the skill refuses to emit the commit and returns an error.

### Frame conditions
- Pure transformation of `commit_ctx` to a `CommitMessage` string.

### Module invariants preserved
- I6 (every greenfield-skill commit carries trailer).

### Test linkage
- T2.16 — commit_ctx with valid `propagated-discovery` classification → trailer present in commit message. commit_ctx with empty `classification` → skill refuses; no commit.

### Behavior contract
This is a **shared** operation across all four greenfield skills. Each skill calls `greenfield-skill-emit-trailer` rather than reimplementing trailer formatting. A duplicated implementation is an integrity violation analogous to M1's mode-of pattern.

---

## F2.8 — `intent-check-prose-write-attestation(diff_result: DiffCheckerOutput, back_translation: BackTranslation, protected_files: List<AbsolutePath>) → Attestation`

```yaml
---
id: F2.8
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B4, IC4, S2.3, "skills/intent-check/SKILL.md § Step 6"]
produces: [I8, T2.7]
---
```

### Signature
`intent-check-prose-write-attestation(diff_result, back_translation, protected_files) → Attestation`

### Preconditions
- `diff_result` is the post-validation output of F2.4.
- `back_translation` is the F2.3 output that drove the diff-check.
- `protected_files` is the union of:
  - `docs/add/intent.md`
  - All spec-stack files visible to the back-translator (architectural.md plus any deeper-tier specs included in this run)
  - Files explicitly named as protected by the intent doc

### Postconditions

Compute `content_hash` (mirrors `/intent-check` § Step 6 verbatim):
1. Sort `protected_files` alphabetically.
2. Read each file as raw bytes in that order; concatenate with no delimiter.
3. SHA-256 the concatenated byte stream; hex-encode lowercase.

Write `.assurance/intent-check-prose-attestation.json` matching the inherited `Attestation` schema:
```json
{
  "protected_files": ["...sorted..."],
  "content_hash": "<64-hex>",
  "verdict": "pass" | "fail",
  "checked_at": "<RFC3339>",
  "pipeline_output": {
    "back_translation": "Section 1 verbatim\n\nSection 2 verbatim",
    "diff_result": { ...diff_result post-validation... }
  }
}
```

`verdict` mirrors `phase_verdict`: `pass` iff `diff_result.match == true` AND `diff_result.confidence_pct >= 80`.

The attestation MUST be written before any commit that touches `protected_files`. The companion pre-commit hook (M5/F5.3) recomputes the hash, rejects the commit if the attestation is absent / stale / hash-mismatched / verdict-not-pass.

### Frame conditions
- Reads `protected_files` content. Writes the attestation file.

### Module invariants preserved
- I8 (content-hashed attestation; pre-commit hook can verify without invoking an LLM).

### Test linkage
- T2.7 — supply protected_files with known SHA-256; attestation file contains the expected hash and the verbatim diff_result.

### Implementation discipline note
The pre-commit hook does NOT invoke an LLM. It re-reads the protected files, recomputes the hash, and matches against the attestation. This preserves the dual-track principle (per `/assurance-init`'s ROADMAP block) — heavy LLM verification lives in the skill; the gate only checks that the heavy work ran with valid inputs.

---

## Module invariants — `I1`..`I8`

### I1 — PR-approved Status promotion for Attested-tier artifacts (per ADR-006)
For every commit that flips `Status` of an Attested-tier artifact (currently `docs/add/intent.md`, `docs/add/specs/architectural.md`, any `docs/add/decisions/ADR-*.md`, `docs/add/methodology.md`, `docs/add/glossary.md`, `docs/add/acceptance.md`) into `Attested`, the PR carrying the commit MUST receive at least one approving review by a GitHub identity in `.assurance/audit-authors.allowlist`, posted *after* the latest commit in the PR, by an identity *other than* the PR author. The agent MAY author the commit; the PR review is the human signal. F2.1's guard prevents the agent from synthesising an in-skill `Attested` flip *without* the user's explicit in-session direction; ADR-006's PR gate (enforced by M5/F5.6) is the additional structural check at merge time. This invariant supersedes the v1.0 wording (which forbade agent-authored attestation commits outright); see ADR-006 § Context for the rationale.

### I2 — Architectural-spec IC coverage
For every Drafted-or-later architectural spec produced by `/spec-derive`, every `IC` from the consumed intent doc is in the `consumes:` of at least one `S` section. F2.2 is the integrity predicate.

### I3 — Back-translation blindness
Every back-translation produced by `/intent-check-prose` carries `intent_doc_in_context: false` and is the output of a model invocation whose context window did not include the intent doc. F2.3's build-time check enforces this structurally.

### I4 — FP-rate kill criterion (inherited semantics)
`/intent-check-prose` reads the same `CROSSCHECK_FP_*` env vars as `/intent-check`, computes the rolling rate over `CROSSCHECK_FP_WINDOW_DAYS` (default 14), refuses to run when the rate exceeds `CROSSCHECK_FP_TRIPPED_THRESHOLD` (default 0.30) with `n ≥ 3` classified rows. F2.4 + F2.4b compute and enforce. This was incorrectly drafted as a 30-attestation rolling window in v1.0; corrected per Phase 2 seam validation A-1.

### I5 — Layer-1 seam declared and verified
Every `F` section with `implementation: spec-iterate` or `implementation: lean-pipeline` either has the corresponding chain artifacts (per F2.6's per-implementation rules) or an explicit `implementation-status: deferred-to-phase-<n>` note. F2.6 is the integrity predicate. v1.0 covered only spec-iterate; lean-pipeline added per Phase 2 seam validation B-2.

### I6 — Greenfield trailer ubiquity
Every commit produced by the four greenfield skills carries a valid `Spec-Diff-Classification` trailer per ADR-005. F2.7 is the shared trailer-emission operation.

### I7 — Two-section back-translator + carve-out scan + fail-closed validation (inherited)
The prose-vs-prose pipeline inherits three structural disciplines from `/intent-check`: (a) back-translator emits two sections (behavioural guarantees + carve-out quotes), (b) diff-checker performs a mandatory carve-out scan before evaluating gaps, (c) two fail-closed semantic-validation rules (contradictory-output flip, truncated-reason rejection). F2.3 + F2.4 enforce. Added per Phase 2 seam validation A-3, A-4, A-5.

### I8 — Content-hashed attestation
Each `/intent-check-prose` run writes `.assurance/intent-check-prose-attestation.json` with a SHA-256 hash over sorted, concatenated raw bytes of the protected file set. The pre-commit hook verifies without invoking an LLM. F2.8 is the attestation operation. Added per Phase 2 seam validation A-6.

---

## Test linkage stubs — `T2.1`..`T2.16`

| ID | Operation | Stub description |
|---|---|---|
| T2.1 | F2.1 | human says "i attest..." → AllowAttest |
| T2.2 | F2.1 | human's message has no attestation phrase → RefuseAttest |
| T2.3 | F2.2 | spec missing IC11 in any S consumes → Incomplete([IC11]) |
| T2.4 | F2.2 | spec consumes all IC1..IC11 → Complete |
| T2.5 | F2.3 | back-translation Section 1 captures key concept from spec |
| T2.6 | F2.3 | build-time check fails when intent.md is added to back-translator context |
| T2.7 | F2.3 + F2.8 | spec contains "## Negative space (out of scope)" block → back-translation Section 2 quotes those lines verbatim. Attestation file contains expected SHA-256 over sorted protected files. |
| T2.8 | F2.4 | diff-checker raw `{match: true, mismatch_reason: "..."}` → semantic validation flips to fail with confidence_pct=40 |
| T2.9 | F2.4 | diff-checker raw `{match: false, mismatch_reason: "yes"}` → rejected as truncated; no tracker row written |
| T2.10 | F2.4b | append rows totalling 4 spurious / 12 classified within 14 days → rolling=0.33, status=TRIPPED |
| T2.11 | F2.4b | append rows totalling 2 spurious / 12 classified within 14 days → rolling=0.17, status=ACTIVE |
| T2.12 | F2.5 | probe an under-specified section → ≥1 proposal at MEDIUM+ confidence; ≤3 proposals total |
| T2.13 | F2.5 | probe a Ratified section → error (skill refuses) |
| T2.14 | F2.6 | F with `implementation: spec-iterate` + matching .dfy → Ok. F with `implementation: lean-pipeline` + Lean artifacts + verdict=exact → Ok |
| T2.15 | F2.6 | F with `implementation: lean-pipeline` + verdict=mismatch → Violation::LeanCorrespondenceMismatch |
| T2.16 | F2.7 | valid commit context → trailer in message; empty classification → skill refuses; no commit |

---

## What this spec deliberately does not specify

- The exact prompt strings for any of the four greenfield skills. SKILL-md-tier concern.
- The wording of the "ready for attestation" prompt that gates F2.1.
- The number of independent back-translations per `/intent-check-prose` invocation (single in v1; multi-shot is a future extension).
- The internals of `/intent-check`'s Steps 0–8. Inherited verbatim per S2.3; this functional spec parameterises the substitutions only.

## Open questions surfaced by this draft

1. **Phrase list in F2.1.** I committed on a small set of explicit-confirmation phrases. The list is narrow on purpose (false positives are worse than false negatives) but may be too narrow in practice — humans may say "yes, I attest IC1..IC10" or "I confirm and attest" or other variants. Worth reviewing the list against your typical attestation language.
2. **F2.7 ubiquity guarantees.** The "every greenfield-skill commit" rule depends on the agent following discipline. There's no harness-level enforcement that the agent calls F2.7 from inside the skill — only convention via SKILL.md. The pre-commit hook (M5) is the harness-level safety net. Worth confirming the layered enforcement is acceptable.

(Open questions Q2–Q4 from v1.0 — build-time check mechanism, degraded-mode marker file, confidence floor — were resolved by the seam validation pass per Bucket C; resolutions are reflected in the inheritance from `/intent-check` and `/spec-adversary` and need no separate adjudication.)
