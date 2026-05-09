# M2-greenfield-skills — Functional Spec

```yaml
---
id: M2-greenfield-skills
mode: add
phase: 1
status: Drafted
consumes: [IC1, IC2, IC3, IC4, IC7, IC11, S2.1, S2.2, S2.3, S2.4, S2.5, ADR-004, ADR-005, M2-greenfield-skills/B1, M2-greenfield-skills/B2, M2-greenfield-skills/B3, M2-greenfield-skills/B4, M2-greenfield-skills/B5, M2-greenfield-skills/B6]
produces: [F2.1, F2.2, F2.3, F2.4, F2.5, F2.6, F2.7, I1, I2, I3, I4, I5, I6, T2.1..T2.14]
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

```
BackTranslation := {
  written_at: Timestamp,
  spec_stack_inputs: List<ID-with-content-hash>,
  prose: String,                 // the agent's blind summary
  intent_doc_in_context: false   // mandatory; see I3
}
```

### `ComparisonReport`

```
ComparisonReport := {
  matches: List<{ ic_id, status: Full | Partial | Divergent | Silent }>,
  gaps_in_intent: List<Finding>,
  gaps_in_spec: List<Finding>,
  contradictions: List<Finding>,
  underspecifications: List<Finding>,
  drift_candidates: List<Finding>,
  verdict: PASS | PASS_WITH_AMENDMENTS | HOLD,
  fp_tracker_path: AbsolutePath
}
```

### `FPRecord`

```
FPRecord := {
  finding_id: String,
  human_adjudication: { spurious: Boolean, justification: String },
  recorded_at: Timestamp,
  rolling_window_position: Integer
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

## F2.3 — `intent-check-prose-back-translate(spec: List<SRecord>) → BackTranslation` (blindness contract)

```yaml
---
id: F2.3
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B3, IC4, S2.3]
produces: [I3, T2.5, T2.6]
---
```

### Signature
`intent-check-prose-back-translate(spec: List<SRecord>) → BackTranslation`

### Preconditions
- `spec` is the Drafted architectural spec.
- The skill's invocation is gated by a build-time check that the back-translation prompt's context window does not include `intent.md` content.

### Postconditions

Returns a `BackTranslation` value with:
- `written_at = now`.
- `spec_stack_inputs` lists each `S` ID and its content hash.
- `prose` is the agent's natural-language description of the system the spec describes.
- `intent_doc_in_context = false` is guaranteed by the build-time check; an output with `intent_doc_in_context = true` is malformed and the skill must refuse to ship it.

### Frame conditions
- Reads only `spec`.
- The implementer MUST partition prompts: the back-translator prompt receives `spec` only; the comparator prompt (F2.7 below — see also next section) receives intent and back-translation. The two MUST run in separate model invocations to prevent context-bleed.

### Module invariants preserved
- I3 (back-translation prompt is blind to intent).

### Test linkage
- T2.5 — invoke back-translator with a spec that mentions "auditor agent"; back-translation contains "auditor" or synonym.
- T2.6 — build-time check: a code path that adds intent to the back-translator's context fails the check (violation detected at skill-compile-time).

### Implementation discipline note
The blindness contract is enforced **structurally** (separate prompts, separate model invocations) not by convention. Build-time enforcement means a developer modifying the skill's prompt template cannot accidentally regress this property; the build fails. This mirrors the M3/B1 discipline (deterministic instrumentation has no LLM dependency).

---

## F2.4 — `intent-check-prose-fp-tracker(report: ComparisonReport, adjudication: HumanAdjudication) → FPState`

```yaml
---
id: F2.4
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B4, IC4, S2.3]
produces: [I4, T2.7, T2.8]
---
```

### Signature
`intent-check-prose-fp-tracker(report: ComparisonReport, adjudication: HumanAdjudication) → FPState`

`FPState := { rolling_rate: Float in [0.0, 1.0], degraded: Boolean, recent_records: List<FPRecord> }`

`HumanAdjudication := { finding_id: String, spurious: Boolean, justification: String }`

### Preconditions
- A `ComparisonReport` has been produced by an earlier invocation of `intent-check-prose-compare` (F2.7).
- The human has reviewed at least one `Finding` from that report and provided their adjudication.

### Postconditions

The function appends an `FPRecord` to `.assurance/intent-check-prose-fp-tracker.csv`:
```
recorded_at, finding_id, spurious, justification, rolling_window_position
```

It then computes:
- `rolling_rate := count(fp_records[-30:].spurious == true) / min(30, len(fp_records))`.
- `degraded := rolling_rate > 0.30`.
- `recent_records := fp_records[-30:]`.

When `degraded` flips from `false` to `true`, the skill writes a marker file at `.assurance/intent-check-prose-degraded.flag` containing the rolling-rate value and the timestamp; the marker is removed when `degraded` flips back to `false`.

The 30% threshold is configurable via env var `CROSSCHECK_INTENT_CHECK_PROSE_FP_THRESHOLD` per the existing `/intent-check` precedent.

### Frame conditions
- Appends to the FP-tracker CSV; writes/removes the degraded marker. No other side effects.

### Module invariants preserved
- I4 (FP-rate kill criterion enforced).

### Test linkage
- T2.7 — append 31 records, 10 spurious; rolling_rate = 0.33; degraded = true; marker file present.
- T2.8 — append further records bringing rolling spurious count to 8/30; degraded flips false; marker removed.

### FP definition note (per Phase 2 A-5)
A False Positive is a flagged divergence the human reviewer attests is spurious (e.g., wording difference but semantic equivalence). The human marks `spurious: true` in the adjudication; the rolling rate counts those.

---

## F2.5 — `spec-adversary-prose-probe(spec_section: SRecord, confidence_floor: ConfidenceLevel) → AdversaryReport`

```yaml
---
id: F2.5
status: Drafted
implementation: manual
consumes: [S2.4, M2-greenfield-skills/B6]
produces: [T2.9, T2.10]
---
```

### Signature
`spec-adversary-prose-probe(spec_section: SRecord, confidence_floor: ConfidenceLevel) → AdversaryReport`

`ConfidenceLevel := LOW | MEDIUM | HIGH`

`AdversaryReport := { gaps: List<Gap>, signal_to_noise_self_check: SnrSelfCheckResult }`

`Gap := { description: String, confidence: ConfidenceLevel, gap_kind: BehaviorUnconstrained | EdgeCaseSilent | UnansweredQuestion }`

### Preconditions
- `spec_section.status == Drafted` (the skill refuses to probe `Ratified` sections; per S2.4 those go through full consolidation).

### Postconditions

Returns an `AdversaryReport` containing:
- `gaps`: a list of identified gaps in the spec section, each labelled with confidence `LOW | MEDIUM | HIGH`. Only gaps at `confidence >= confidence_floor` are returned.
- `signal_to_noise_self_check`: the agent's self-assessment of whether the gaps are real or noise. Mirrors the existing `/spec-adversary` discipline.

The skill **does not propose changes** to the spec section. It produces probing output that the human or Hellebuyck can use to amend.

### Frame conditions
- Reads `spec_section` only. No mutation of spec, intent, or any other artifact.

### Module invariants preserved
- (no new module invariant; the probe-only-no-amend property is structural to the operation, not a separately tracked invariant.)

### Test linkage
- T2.9 — probe a deliberately under-specified spec section; expect at least one gap at MEDIUM-or-higher confidence.
- T2.10 — probe a Ratified section; expect refusal (skill returns an error, not a report).

---

## F2.6 — `spec-iterate-seam-honored(F_section: FSection) → IntegrityVerdict`

```yaml
---
id: F2.6
status: Drafted
implementation: manual
consumes: [M2-greenfield-skills/B6, IC3, S2.5]
produces: [I5, T2.11, T2.12]
---
```

### Signature
`spec-iterate-seam-honored(F_section: FSection) → IntegrityVerdict`

`FSection := SRecord-like with frontmatter including 'implementation:' and optional 'implementation-status:'`

### Preconditions
- `F_section.frontmatter.implementation` is one of `spec-iterate | manual | external`, or undeclared (defaults to `manual`).

### Postconditions

If `F_section.frontmatter.implementation == "spec-iterate"`:
- Returns `Ok` iff one of:
  - A `.dfy` artifact exists at `<module-dir>/verification/<F_section.slug>.dfy`, OR
  - `F_section.frontmatter.implementation_status` matches `^deferred-to-phase-[3-5]$`.
- Returns `Violation { kind: SpecIterateSeamUnhonored, location: F_section.id }` otherwise.

If `F_section.frontmatter.implementation == "manual"` or `"external"`:
- Returns `Ok` (no seam to verify).

### Frame conditions
- Reads `F_section` and the module's verification directory listing.

### Module invariants preserved
- I5 (spec-iterate seam declared and integrity-checked).

### Test linkage
- T2.11 — F section with `implementation: spec-iterate` and matching .dfy → `Ok`.
- T2.12 — F section with `implementation: spec-iterate` and no .dfy, no deferred-to-phase note → `Violation::SpecIterateSeamUnhonored`.

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
- T2.13 — commit_ctx with valid `propagated-discovery` classification → trailer present in commit message.
- T2.14 — commit_ctx with empty `classification` → skill refuses; no commit.

### Behavior contract
This is a **shared** operation across all four greenfield skills. Each skill calls `greenfield-skill-emit-trailer` rather than reimplementing trailer formatting. A duplicated implementation is an integrity violation analogous to M1's mode-of pattern.

---

## Module invariants — `I1`..`I6`

### I1 — Human-attestation-required for IC promotion
For every commit that flips `Status` of `docs/add/intent.md` from `Drafted` to `Attested`, the commit author MUST be a human (by `.assurance/audit-authors.allowlist`); the agent never authors such a commit. F2.1's guard prevents the agent from synthesising the flip.

### I2 — Architectural-spec IC coverage
For every Drafted-or-later architectural spec produced by `/spec-derive`, every `IC` from the consumed intent doc is in the `consumes:` of at least one `S` section. F2.2 is the integrity predicate.

### I3 — Back-translation blindness
Every back-translation produced by `/intent-check-prose` carries `intent_doc_in_context: false` and is the output of a model invocation whose context window did not include the intent doc. F2.3's build-time check enforces this structurally.

### I4 — FP-rate kill criterion
The rolling 30-attestation FP rate for `/intent-check-prose` stays below 30%; otherwise the skill enters degraded mode and refuses to ship clean attestations until the rate recovers. F2.4 computes and enforces.

### I5 — Spec-iterate seam declared and verified
Every `F` section with `implementation: spec-iterate` either has a corresponding `.dfy` artifact or an explicit deferred-to-phase note. F2.6 is the integrity predicate.

### I6 — Greenfield trailer ubiquity
Every commit produced by the four greenfield skills carries a valid `Spec-Diff-Classification` trailer per ADR-005. F2.7 is the shared trailer-emission operation.

---

## Test linkage stubs — `T2.1`..`T2.14`

| ID | Operation | Stub description |
|---|---|---|
| T2.1 | F2.1 | human says "i attest..." → AllowAttest |
| T2.2 | F2.1 | human's message has no attestation phrase → RefuseAttest |
| T2.3 | F2.2 | spec missing IC11 in any S consumes → Incomplete([IC11]) |
| T2.4 | F2.2 | spec consumes all IC1..IC11 → Complete |
| T2.5 | F2.3 | back-translation captures key concept from spec |
| T2.6 | F2.3 | build-time check fails when intent.md is added to back-translator context |
| T2.7 | F2.4 | 31 records, 10 spurious → rolling=0.33 degraded marker present |
| T2.8 | F2.4 | further records bring spurious to 8/30 → degraded flips false marker removed |
| T2.9 | F2.5 | probe an under-specified section → ≥1 gap at MEDIUM+ confidence |
| T2.10 | F2.5 | probe a Ratified section → error (skill refuses) |
| T2.11 | F2.6 | F with `implementation: spec-iterate` + matching .dfy → Ok |
| T2.12 | F2.6 | F with `implementation: spec-iterate`, no .dfy, no deferred note → Violation |
| T2.13 | F2.7 | valid commit context → trailer in message |
| T2.14 | F2.7 | empty classification → skill refuses; no commit |

---

## What this spec deliberately does not specify

- The exact prompt strings for any of the four greenfield skills. SKILL-md-tier concern.
- The wording of the "ready for attestation" prompt that gates F2.1.
- The list of `Gap.kind` values beyond the three enumerated (extension via SKILL.md amendment is the path).
- The number of independent back-translations per `/intent-check-prose` invocation (single in v1; multi-shot is a future extension).

## Open questions surfaced by this draft

1. **Phrase list in F2.1.** I committed on a small set of explicit-confirmation phrases. The list is narrow on purpose (false positives are worse than false negatives) but may be too narrow in practice — humans may say "yes, I attest IC1..IC10" or "I confirm and attest" or other variants. Worth reviewing the list against your typical attestation language.
2. **F2.3's build-time check mechanism.** I committed on "build-time check" without specifying *how*: lint rule scanning the prompt source for intent.md mentions? Test that runs the skill against a sample input and asserts the back-translator's prompt didn't contain intent? Both work; the former is faster and less brittle. Worth your call.
3. **F2.4's degraded-mode marker file.** A file under `.assurance/` is the SSOT for degraded state. The pre-commit hook would need to inspect this file to gate emissions of `/intent-check-prose` reports while degraded. Alternative: degraded mode is a warning only, not a gate. The latter is closer to the existing `/intent-check` discipline; the former is stricter. Worth your judgment.
4. **F2.5's confidence floor.** The signature takes `confidence_floor` as input but the skill's default value isn't fixed here. Recommend `MEDIUM` as the default per the existing `/spec-adversary` precedent. Flag if you'd prefer `HIGH` or `LOW`.
5. **F2.7 ubiquity guarantees.** The "every greenfield-skill commit" rule depends on the agent following discipline. There's no harness-level enforcement that the agent calls F2.7 from inside the skill — only convention via SKILL.md. The pre-commit hook (M5) is the harness-level safety net. Worth confirming the layered enforcement is acceptable.
