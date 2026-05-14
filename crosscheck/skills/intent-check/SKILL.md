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

The skill is a portable analog of Midspiral's `claimcheck` and an internal Go-binary precursor. It is structurally adversarial by construction: the back-translator is **blind to the original intent prose**, so it cannot tautologically restate it, and the diff-checker is forced to scan for carve-outs and rationale comments before rendering a verdict. These two prompt-level guardrails are the Phase-1 calibration fixes that drive the false-positive rate below the configurable kill criterion (default 30%, see Configuration).

This is a methodology skill. It does not ship a binary — Claude drives the pipeline itself, reading files, invoking the two prompts in the required order, writing the tracker row and the attestation, and running the kill-criterion math before committing. The companion pre-commit hook (described in `references/attestation-schema.md`) is fast and LLM-free: it only checks hashes and verdicts.

Cross-references:
- `references/round-trip-prompt.md` — verbatim two-prompt template for back-translator + diff-checker, including rationale extraction and carve-out taxonomy.
- `references/fp-tracker-schema.md` — CSV schema, append logic, and the kill-criterion computation.
- `references/attestation-schema.md` — JSON attestation format, SHA-256 computation, and pre-commit hook pseudocode.

## Configuration

The kill-criterion thresholds in Step 0 are configurable via environment variables. Read these once at the start of the skill run; if unset, use the documented defaults.

| Env var | Default | Meaning |
|---|---|---|
| `CROSSCHECK_FP_TRIPPED_THRESHOLD` | `0.30` | Rolling FP rate at which Step 0 refuses to run (Layer 5 offline) |
| `CROSSCHECK_FP_AT_RISK_THRESHOLD` | `0.20` | Rolling FP rate at which the verdict reports `AT RISK` |
| `CROSSCHECK_FP_WINDOW_DAYS` | `14` | Rolling-window length used to compute the FP rate |

The defaults (30% / 20% / 14 days, with `n ≥ 3` minimum sample size) are **founder intuition, not labelled-pilot data**. Tune them for your tolerance once you have ≥30 classified human verdicts. See `docs/research/assurance-hierarchy.md` for the calibration rationale and `docs/examples/workflows/README.md` for how the same numbers are surfaced in the reference squad workflows. Schema parity matters: any consumer that reads `.assurance/intent-check-fp-tracker.csv` (e.g. `/assurance-status`) must use the same env vars so the user sees the same rate everywhere.

## Instructions

You are running a two-LLM round-trip informalization pipeline over one protected-surface change. The structural separation between back-translator and diff-checker is the entire point — do not shortcut it, do not let the same reasoning thread produce both outputs, and do not allow the back-translator to see the invariant prose.

### Step 0: Kill-Criterion Pre-Check

Before doing any LLM work, open `.assurance/intent-check-fp-tracker.csv` (create it if missing; see `references/fp-tracker-schema.md` for the exact header).

Read the threshold env vars from the Configuration section: `tripped = CROSSCHECK_FP_TRIPPED_THRESHOLD` (default `0.30`) and `window = CROSSCHECK_FP_WINDOW_DAYS` (default `14`). The defaults are founder intuition, not labelled-pilot data — see the Configuration section.

Compute the rolling false-positive rate over the last `window` days of entries (rows where `date` is within `window` days of today **AND `human_verdict` is non-empty** — empty cells are awaiting review and are excluded from both numerator and denominator; see the pseudocode in `references/fp-tracker-schema.md`):

- FP rate = count(`human_verdict` == `spurious`) / count(rows in window with non-empty `human_verdict`)
- If the window has fewer than 3 classified rows, treat the rate as unknown and proceed with a warning.
- If the FP rate **> `tripped`**, refuse to run. Tell the user:

  > The Layer-5 round-trip pipeline's rolling false-positive rate is `<rate>%` over the last `<window>` days (threshold: `<tripped>%`, default 30%). The kill criterion in the assurance hierarchy says this layer's strategy needs rework before it keeps gating commits. Do not re-enable until (a) the prompt or model is revised, or (b) human review has reclassified enough entries to drop the rate below the threshold. See `references/fp-tracker-schema.md` for the exact computation and the Configuration section above for the threshold env vars.

  Stop. Do not proceed to Step 1.

### Step 1: Gather the Triple

Determine the three inputs by inference; ask only when inference fails.

1. **Invariant prose.** Default location: `docs/invariants/<module>.md`. If the user named a specific module or file, use that. If multiple invariant docs are touched by the diff, **fan out** — run the pipeline in parallel, one invocation per (invariant, test, diff) triple. The orchestrator (or this skill itself when invoked under `add-orchestrator`) dispatches the parallel runs and aggregates verdicts. Do not block on a single-pick AskUserQuestion.
2. **Covering property test.** Auto-resolve by grepping the test directories for `// Invariant <ID>:` (or the repo's equivalent comment convention) matching the invariant IDs touched by the diff. Cross-check with `git diff --staged --name-only` to prefer tests in the staged set. If no covering test is found after both passes, stop and tell the user — the pipeline is undefined without one.
3. **Code diff.** Staged changes in the current working tree: `git diff --staged` plus any relevant already-committed changes on the branch. Scope the diff to files implicated by the invariant — not the entire PR.

Ask the user to supply a path inline only when (a) the staged diff is empty (nothing to check), or (b) covering tests exist but none are linked to the invariant ID via comments. In all other cases, the agent resolves the triple from repo state and reports its inferences in the Step 8 summary.

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

### Step 7: Companion Pre-Commit Hook (draft to disk, do not install)

The skill does not auto-install the hook — installing a hook is a Class A protected-surface decision per `.claude/rules/protected-surfaces.md`, so it must land via `/protected-surface-amend` rather than being silently added. But the skill can do everything *up to* installation: it writes a draft hook script to disk that the user reviews and applies.

Detect the user's pre-commit framework once (mirror `/invariant-coverage-scaffold`'s detection: `.pre-commit-config.yaml` → pre-commit.com; `lefthook.yml` → lefthook; `.husky/` → husky; else `none`). Emit a draft file at one of:

- `.assurance/intent-check-hook.draft.pre-commit.yaml` (for pre-commit.com — a snippet the user merges into their config).
- `.assurance/intent-check-hook.draft.lefthook.yml` (lefthook).
- `.assurance/intent-check-hook.draft.husky.sh` (husky — full executable hook script).
- `.assurance/intent-check-hook.draft.md` (none / unknown — pseudocode reference for a hand-rolled hook).

Each draft contains the < 1 s, no-LLM logic from `references/attestation-schema.md`: scan staged files against the protected-surface patterns; if any match, read `.assurance/intent-check-attestation.json`, recompute the content hash, and reject the commit unless the attestation exists, its verdict is `pass`, and its hash matches.

Report the written draft path. The user reviews and applies via `/protected-surface-amend` (Class A amendment). The skill itself does not modify the user's existing hook configuration.

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

Rolling FP rate (last <window> days): <rate>%  (threshold: <tripped>%, default 30%)
```

If `match == false`, tell the user exactly what the back-translator perceived vs. what the invariant prose claimed, so they can decide whether to fix the code, fix the test, or amend the invariant prose via `/protected-surface-amend`.

### Step 9: What this does NOT catch

The round-trip pipeline is Layer 5 best-effort. It is probabilistic and has well-characterised blind spots. Surface them in the report so the user knows the coverage boundary:

```markdown
## What this does NOT catch

This skill is a two-LLM round-trip probe over (invariant prose, covering test, code diff). It cannot detect:

1. **Code-test-diff inconsistencies the back-translator's vocabulary cannot express.** If the back-translator is missing the domain noun, the round-trip produces a false `match`. Use `/spec-adversary` for the code-vs-doc gap probe.
2. **Cross-module invariant contradictions.** This skill scopes to one invariant doc at a time. Use `/audit-invariant-consistency` for cross-module passes.
3. **Spec sections an invariant should cover but doesn't.** Use `/audit-spec-coverage` for the section → invariant coverage matrix.
4. **Behaviour not reachable from the staged diff.** The pipeline is diff-scoped; properties tested by code outside the diff but invalidated by the diff via call graph propagation are not probed.
5. **Property-test brittleness.** A `pass` result does not mean the test is strong; use `/assurance-probe` (mutation + vacuity + generator probes) for test-strength.

The 30% FP kill criterion is the operational floor — if more than 30% of `pass` verdicts are reclassified as `spurious` on human review, the prompt or model is the problem, not the user's invariant docs.
```

### Verification Checklist

```
## Evidence Summary (agent-verified during this run)

- Kill-criterion pre-check ran before any LLM call; rolling FP rate over the configured window is below the configured tripped threshold (defaults: 14 days, 30%, sourced from Configuration env vars).
- Triple resolved from repo state where possible: invariant doc <auto-detected | provided>, covering test <grep-resolved | provided>, code diff <git-staged | provided>.
- Multi-invariant runs (if applicable): fanned out to <N> parallel invocations; this report aggregates verdicts.
- Back-translator received only {code, test} — never the invariant prose.
- Back-translation contains both Section 1 (behavioural guarantees) and Section 2 (rationale comments, verbatim with file:line).
- Diff-checker performed the mandatory Step 1 carve-out scan.
- Diff-checker output passed semantic validation (no contradictory match/reason, reason >= 20 chars when non-empty).
- FP-tracker row appended with the exact schema (date, invariant_touched, phase_verdict, human_verdict).
- Attestation emitted with sorted protected_files, SHA-256 content_hash, verdict, RFC3339 checked_at, and pipeline_output.
- Draft pre-commit hook written to .assurance/intent-check-hook.draft.<framework> — user reviews and applies via /protected-surface-amend (Class A).
- "What this does NOT catch" section emitted with the five blind spots enumerated.

## Decisions for Review (if verdict is fail)

- [ ] Pick remediation: fix the code, fix the test, or amend the invariant prose via /protected-surface-amend.
```

## Arguments

Optional invariant-doc path and covering-test path. Omitted arguments prompt the skill to locate them itself.

Examples:
- `/intent-check` — locate the triple from staged changes and recent context.
- `/intent-check docs/invariants/queue.md` — run against all covering tests for that invariant doc.
- `/intent-check docs/invariants/queue.md internal/queue/queue_invariants_prop_test.go` — explicit triple; code diff taken from `git diff --staged`.
