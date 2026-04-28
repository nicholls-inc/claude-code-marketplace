---
name: intent-check
description: >-
  Portable round-trip informalization over (invariant prose, covering test,
  code diff) — Layer 5 of the assurance hierarchy. Runs a two-LLM pipeline
  (blind back-translator then diff-checker), appends outcomes to a
  false-positive tracker, emits a content-addressed JSON attestation, and
  enforces a 30% FP kill criterion. Triggers: "intent check", "round-trip
  check", "spec-intent alignment", "verify invariant against test",
  protected-surface diffs touching `docs/invariants/` or property tests.
argument-hint: "[optional: invariant doc path] [optional: covering test path]"
---

# /intent-check — Round-Trip Intent Verification

## Description

Operationalise Layer 5 (spec-intent alignment) of the assurance hierarchy as a portable, repo-agnostic Claude Code skill. The input is a triple — an **invariant prose description**, a **covering property test**, and the **code diff** the user wants to ship. The output is a structured verdict (match / mismatch + confidence + reason), an appended row in a per-repo false-positive tracker CSV, and a content-hashed JSON attestation that a companion pre-commit hook can check without invoking an LLM.

The skill is a portable analog of Midspiral's `claimcheck` and an internal Go-binary precursor. It is structurally adversarial by construction: the back-translator is **blind to the original intent prose**, so it cannot tautologically restate it, and the diff-checker is forced to scan for carve-outs and rationale comments before rendering a verdict. These two prompt-level guardrails are the Phase-1 calibration fixes that drive the false-positive rate below the 30% kill criterion.

This is a methodology skill. It does not ship a binary — Claude drives the pipeline itself, reading files, invoking the two prompts in the required order, writing the tracker row and the attestation, and running the kill-criterion math before committing. The companion pre-commit hook (described in `references/attestation-schema.md`) is fast and LLM-free: it only checks hashes and verdicts.

Cross-references:
- `references/round-trip-prompt.md` — verbatim two-prompt template for back-translator + diff-checker, including rationale extraction and carve-out taxonomy.
- `references/fp-tracker-schema.md` — CSV schema, append logic, and the 30% kill-criterion computation.
- `references/attestation-schema.md` — JSON attestation format, SHA-256 computation, and pre-commit hook pseudocode.

## Instructions

You are running a two-LLM round-trip informalization pipeline over one protected-surface change. The structural separation between back-translator and diff-checker is the entire point — do not shortcut it, do not let the same reasoning thread produce both outputs, and do not allow the back-translator to see the invariant prose.

### Step 0: Kill-Criterion Pre-Check

Before doing any LLM work, open `.assurance/intent-check-fp-tracker.csv` (create it if missing; see `references/fp-tracker-schema.md` for the exact header).

Compute the rolling false-positive rate over the **last 2 weeks of entries** (rows where `date` is within 14 days of today **AND `human_verdict` is non-empty** — empty cells are awaiting review and are excluded from both numerator and denominator; see the pseudocode in `references/fp-tracker-schema.md`):

- FP rate = count(`human_verdict` == `spurious`) / count(rows in window with non-empty `human_verdict`)
- If the window has fewer than 3 classified rows, treat the rate as unknown and proceed with a warning.
- If the FP rate **> 30%**, refuse to run. Tell the user:

  > The Layer-5 round-trip pipeline's rolling false-positive rate is `<rate>%` over the last 14 days (threshold: 30%). The kill criterion in the assurance hierarchy says this layer's strategy needs rework before it keeps gating commits. Do not re-enable until (a) the prompt or model is revised, or (b) human review has reclassified enough entries to drop the rate below 30%. See `references/fp-tracker-schema.md` for the exact computation.

  Stop. Do not proceed to Step 1.

### Step 1: Gather the Triple

Determine the three inputs:

1. **Invariant prose.** Default location: `docs/invariants/<module>.md`. If the user named a specific module or file, use that. If multiple invariant docs are touched by the diff, ask the user which one to focus on — run the pipeline once per (invariant, test, diff) triple.
2. **Covering property test.** The test file(s) that exercise the invariant. Prefer tests whose comments reference the invariant ID (e.g. `// I2:` or `// invariant I2`). If you cannot identify a covering test, stop and tell the user — the pipeline is undefined without one.
3. **Code diff.** Staged changes in the current working tree: `git diff --staged` plus any relevant already-committed changes on the branch. Scope the diff to files implicated by the invariant — not the entire PR.

If any of the three is missing, ask the user to supply it (path or inline) before proceeding. Do not guess.

Record the set of **protected files** touched by this run — the union of the invariant doc paths, the covering test paths, and any code files that the invariant doc explicitly protects. This set is later written into the attestation as `protected_files`.

### Step 2: Run the Back-Translator (BLIND Prompt)

Use the `back_translate` prompt verbatim from `references/round-trip-prompt.md`. Fill the `{code}` and `{test}` placeholders only. **Do not include the invariant prose** — passing it through contaminates the back-translator and voids the round-trip property.

The back-translator must output **two sections**:

- **Section 1: Behavioural guarantees** — a plain-text paragraph describing what the code + test enforce.
- **Section 2: Design rationale comments** — every 3+ line comment block and every single-line comment whose text contains rationale markers (`because`, `since`, `artefact`, `workaround`, `zeroed`, `intentional`, `skipping`, `ignore`), quoted verbatim with file and line references. If none exist, the section must say `None.`.

Both sections are mandatory. If either is missing or empty (other than the explicit `None.`), re-invoke the prompt once. If it is still malformed on the second try, stop and report the failure.

### Step 3: Run the Diff-Checker

Use the `diff_check` prompt verbatim from `references/round-trip-prompt.md`. Supply:

- `{invariant_prose}` — the full relevant section of the invariant doc (the specific invariant(s) the test covers, not the entire doc).
- `{back_translation}` — both sections from Step 2, unedited.

The prompt forces the diff-checker to do a **mandatory Step 1 carve-out scan** before evaluating any gap. The scan looks for scope markers — `Not covered`, `caller-responsibility`, `precondition`, `aspirational`, `known violation`, `privileged`, `exempt`, `out of scope`, `does not apply` — and classifies each found clause by the scope-modifier taxonomy documented in the prompt template. Only after the scan does the prompt evaluate apparent gaps.

The diff-checker returns a JSON object matching this schema:

```json
{
  "match": true | false,
  "mismatch_reason": "string",
  "mismatch_category": "spec_scope_mismatch | weaker_guarantee | missing_property | missing_coverage | rationale_explains | carve_out_applies | clean_match",
  "confidence_pct": 0-100,
  "confidence_basis": "carve-out-found | rationale-found | rationale-absent | spec-ambiguous | code-ambiguous"
}
```

### Step 4: Semantic Validation (fail-closed hardening)

After parsing the diff-checker JSON, apply these two defensive rules (they are independent of the prompt so they catch model contradictions):

1. **Contradictory output.** If `match == true` AND `mismatch_reason` is non-empty and non-trivial → flip to:
   ```
   match = false
   confidence_pct = 40
   confidence_basis = "spec-ambiguous"
   mismatch_category = "missing_property"
   ```
   Record a note that the raw output was internally contradictory. Fail-closed is safer than trusting either half of the contradiction.
2. **Truncated reason.** If `match == false` AND `len(strip(mismatch_reason)) < 20` → reject the output as truncation. Do not write a tracker row or an attestation. Ask the user to re-run (network transient / model returned malformed result).

If either rule trips, surface the raw pre-rule output alongside the post-rule decision so the user can audit.

### Step 5: Append to the FP Tracker

Append exactly one row to `.assurance/intent-check-fp-tracker.csv` with columns:

```
date,invariant_touched,phase_verdict,human_verdict
```

- `date` — today in `YYYY-MM-DD`.
- `invariant_touched` — a short label identifying the invariant under test, e.g. `queue.md I2 (6-of-19 field mutation coverage)`. Match the style described in `references/fp-tracker-schema.md`.
- `phase_verdict` — `pass` if `match == true` **AND** `confidence_pct >= 80`; `fail` otherwise. This must match the `verdict` written to the attestation in Step 6 — low-confidence matches are tracked as `fail` so the kill criterion picks up prompt weakness, and the attestation hook refuses the commit.
- `human_verdict` — leave **empty** at skill-run time. A human reviewer fills this in later as `genuine`, `genuine-planted`, `partial`, or `spurious`. The kill-criterion math ignores empty cells (see `references/fp-tracker-schema.md` for the append logic and the rolling-window computation).

The schema is fixed — do not add columns and do not rename columns. Stability across repos is what makes the rows directly concatenatable for cross-repo calibration analysis.

### Step 6: Write the Attestation

Compute the `content_hash` of the protected-file set:

1. Sort `protected_files` alphabetically.
2. Read each file as raw bytes in that order and concatenate with no delimiter.
3. SHA-256 the concatenated byte stream; hex-encode lowercase.

Write `.assurance/intent-check-attestation.json` with the schema documented in `references/attestation-schema.md`:

```json
{
  "protected_files": ["…sorted…"],
  "content_hash": "<64-hex-chars>",
  "verdict": "pass" | "fail",
  "checked_at": "<RFC3339 timestamp>",
  "pipeline_output": {
    "back_translation": "…Section 1 + Section 2 verbatim…",
    "diff_result": { …full JSON from Step 4 post-validation… }
  }
}
```

`verdict` mirrors `phase_verdict`: `pass` if `match == true` **and** `confidence_pct >= 80`, otherwise `fail`. Low-confidence matches do not count as clean passes — the attestation says `fail` and the user can override with a documented governance note (use `/protected-surface-amend`).

The attestation must be written **before** the commit that touches the protected files. The companion pre-commit hook (described in `references/attestation-schema.md`) re-reads the protected files, recomputes the hash, and fails the commit if the attestation is absent, stale, or the verdict is not `pass`.

### Step 7: Companion Pre-Commit Hook (describe, don't install)

The skill does not install the hook for the user — installing hooks is a repo-level governance decision. But it must **describe** the hook so the user can wire it up. Point the user at `references/attestation-schema.md`, which contains the full pseudocode. Summarise in two lines:

> A fast pre-commit hook (< 1 s, no LLM) that scans staged files for any match to the protected-surface patterns, and — if found — reads `.assurance/intent-check-attestation.json`, recomputes the content hash, and rejects the commit unless the attestation exists, its verdict is `pass`, and its hash matches. The hook is installed once per repo and calls `/intent-check` only when the user re-runs it manually.

### Step 8: Report

Present a single summary block to the user:

```
## /intent-check verdict: <pass|fail>

- Invariant: <label>
- Match: <true|false>
- Confidence: <pct>% (basis: <basis>)
- Category: <mismatch_category>
- Reason: <mismatch_reason or "clean match">

Tracker row appended: .assurance/intent-check-fp-tracker.csv
Attestation written:   .assurance/intent-check-attestation.json

Rolling 14-day FP rate: <rate>% (threshold 30%)
```

If `match == false`, tell the user exactly what the back-translator perceived vs. what the invariant prose claimed, so they can decide whether to fix the code, fix the test, or amend the invariant prose via `/protected-surface-amend`.

### Verification Checklist

```
## Verification Checklist

- [ ] Kill-criterion pre-check ran before any LLM call (rolling 14-day FP rate < 30%)
- [ ] Back-translator prompt received only {code, test} — never the invariant prose
- [ ] Back-translation contains both Section 1 (behavioural guarantees) and Section 2 (rationale comments, verbatim with file:line)
- [ ] Diff-checker performed the mandatory Step 1 carve-out scan before rendering a verdict
- [ ] Diff-checker output passes semantic validation (no contradictory match/reason, reason >= 20 chars when non-empty)
- [ ] FP-tracker row appended with the exact schema (date, invariant_touched, phase_verdict, human_verdict)
- [ ] Attestation emitted with sorted protected_files, SHA-256 content_hash, verdict, RFC3339 checked_at, and pipeline_output
- [ ] Pre-commit hook behaviour described (not installed) so the user can wire it up via their hook config
- [ ] User given the remediation path if verdict is fail (fix code / fix test / amend invariant with /protected-surface-amend)
```

## Arguments

Optional invariant-doc path and covering-test path. Omitted arguments prompt the skill to locate them itself.

Examples:
- `/intent-check` — locate the triple from staged changes and recent context.
- `/intent-check docs/invariants/queue.md` — run against all covering tests for that invariant doc.
- `/intent-check docs/invariants/queue.md internal/queue/queue_invariants_prop_test.go` — explicit triple; code diff taken from `git diff --staged`.
