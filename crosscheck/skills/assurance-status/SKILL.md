---
name: assurance-status
description: >-
  Onboarding-gated status dashboard for a repo that has adopted the 6-layer
  assurance hierarchy. Phase 1 runs a deterministic onboarding check (no LLM);
  Phase 2 emits a status dashboard covering ROADMAP Status drift, invariant
  coverage gaps, protected-surface edits lacking governance notes, intent-check
  FP-tracker rolling rate, last verify-kernel run, and any tripped
  kill-criterion triggers. Triggers: "assurance status", "status dashboard",
  "roadmap drift", "invariant coverage", "FP tracker rate".
argument-hint: "[optional: --phase=1|2 to force a phase; default is auto]"
---

# /assurance-status — Assurance Hierarchy Status Dashboard

## Description

Report the current state of the 6-layer assurance hierarchy in the target repo. Runs in two phases: a deterministic onboarding gate that refuses to progress if the governance scaffolding is missing, followed by a status dashboard that surfaces drift between documented intent and observed state.

The skill is read-only. It does not modify files, write attestations, or kick off verification jobs — it reports. Remediation is delegated to sibling skills (`/assurance-init`, `/assurance-layer-audit`, `/assurance-roadmap-check`, `/intent-check`, `/protected-surface-amend`).

## Instructions

You are the assurance-hierarchy status reporter. Your job is to tell the user whether their repo is onboarded onto the hierarchy and, if it is, what has drifted since the last check. You run in two phases and **never skip Phase 1**.

### Phase 1 — Onboarding Gate (deterministic, no LLM reasoning)

This phase is a mechanical file-existence check. Do not interpret, infer, or reason about the contents of any file in Phase 1 — only check for the presence of the artifacts listed. If any hard-fail check fails, emit the exact refusal message in Step 1.6 and stop before Phase 2.

#### Step 1.1: Check ROADMAP

Verify that `docs/assurance/ROADMAP.md` exists at the repo root.

- **Hard fail** if missing.

#### Step 1.2: Check horizon directories

Verify that all four horizon directories exist:

- `docs/assurance/immediate/`
- `docs/assurance/next/`
- `docs/assurance/medium-term/`
- `docs/assurance/aspirational/`

- **Hard fail** if any of the four is missing.

#### Step 1.3: Check invariant docs

Verify that `docs/invariants/` exists and contains at least one module doc (any `*.md` other than a `README.md`).

- **Hard fail** if the directory is missing or contains no module doc.

#### Step 1.4: Check protected-surfaces rules

Verify that `.claude/rules/protected-surfaces.md` exists.

- **Hard fail** if missing.

#### Step 1.5: Check coverage script wiring

Verify that a coverage script exists and is wired into CI or pre-commit:

- Look for a coverage script under `scripts/` (e.g. `scripts/check-invariant-coverage.*`) or an equivalent path referenced from CI/pre-commit config.
- Confirm the script is referenced from at least one of: `.pre-commit-config.yaml`, `.github/workflows/*.yml`, or an equivalent hook/CI config the repo uses.

- **Warning only** if missing or unwired. This does not block Phase 2.

#### Step 1.6: Gate decision

If any check in 1.1–1.4 hard-failed, emit verbatim:

```
Repo not onboarded. Missing: <comma-separated list of failed items>.
Next: /assurance-init (optionally preceded by /assurance-layer-audit to scope realistically first).
```

Then **stop**. Do not proceed to Phase 2. Do not read file contents. Do not speculate about what the dashboard would have shown.

If all hard-fail checks passed but the coverage-script check (1.5) emitted a warning, record the warning and continue to Phase 2 — surface the warning in the dashboard output.

### Phase 2 — Status Dashboard (only if gate passed)

Phase 2 is observational. You are reporting on state, not changing it. For each section below, gather the evidence, compare it to the documented intent, and flag drift. Do not attempt repairs; point at the sibling skill that handles the remediation.

#### Step 2.1: ROADMAP item Status drift

For each row in `docs/assurance/ROADMAP.md` and each horizon doc under `docs/assurance/{immediate,next,medium-term,aspirational}/`:

1. Parse the `Status:` field. The valid vocabulary is: `Not started`, `In progress`, `Blocked`, `Done`, `Deferred` (with a `Reason:` line on `Deferred`).
2. Compare documented status to observed state using lightweight signals:
   - `Done` — look for a linked PR number; if referenced, verify via `gh pr view <N>` or `git log --grep` that it is merged. If the doc claims Done but no matching merged PR is findable, flag as drift.
   - `In progress` — look for an open PR or recent commits touching the area the doc describes. If nothing is in flight, flag as possible drift.
   - `Not started` — if code, docs, or PRs matching the item already exist, flag as drift (doc lagging reality).
   - `Blocked` — surface the blocking reason if stated; otherwise flag that the blocker is not documented.
   - `Deferred` — confirm a `Reason:` line is present; flag if missing.
3. Present a compact table of items whose documented status does not match observed state. Unchanged items may be summarised as a count.

Recommend `/assurance-roadmap-check` when drift is detected and the user wants a deeper review.

#### Step 2.2: Invariant coverage gap count

Enumerate invariant IDs declared in `docs/invariants/*.md` (convention: IDs of the form `I<digits>` or `IN<digits>`; whatever pattern the repo's coverage script recognises — defer to that pattern if available).

Count the invariant IDs that have no corresponding test-comment coverage. Report the count and a short list of the top-N uncovered IDs (cap N at 10 to keep the dashboard readable).

If the repo has a coverage script, prefer its output; otherwise, do a best-effort grep of test files for `Invariant <ID>` / `// Invariant <ID>:` / language-appropriate equivalents.

Recommend `/invariant-coverage-scaffold` if coverage tooling is missing or incomplete.

#### Step 2.3: Outstanding protected-surface edits without governance notes

Identify recent changes to files the repo marks as protected (per `.claude/rules/protected-surfaces.md`).

1. Enumerate the protected files / globs from the rules doc.
2. Run `git log --since="30 days ago" --name-only` filtered to those paths, or use the repo's attestation record (e.g. `.assurance/intent-check-attestation.json` if present) to cross-reference.
3. For each touched protected file, check whether the governance note required by the rules doc is present in the commit message or linked amendment file.

Report any protected-surface edit from the review window that lacks a governance note. Recommend `/protected-surface-amend` to author the missing amendment block.

#### Step 2.4: intent-check FP tracker summary

Read `.assurance/intent-check-fp-tracker.csv` (columns: `date,invariant_touched,phase_verdict,human_verdict`).

- If the file does not exist, note "FP tracker not initialised" and recommend `/intent-check` to establish the tracker.
- If it exists, use the canonical FP-rate definition from `/intent-check` (see `references/fp-tracker-schema.md` in that skill — the pseudocode there is authoritative):
  1. Filter rows to the rolling 2-week window ending today (`date` within 14 days).
  2. Exclude rows with empty `human_verdict` from both numerator and denominator — they are awaiting review and bias the rate either way.
  3. Let `window_size = count(rows in window with non-empty human_verdict)`. If `window_size == 0`, report sample size 0 and verdict `INSUFFICIENT DATA` — do not compute a rate or compare against the kill criterion.
  4. Otherwise, compute `FP rate = count(human_verdict == "spurious") / window_size`. `partial` does NOT count as spurious (the pipeline was still doing useful work).
  5. Compare against the 30% kill criterion.
  6. Report the rolling rate, `window_size`, and the verdict: `OK` if below 20%, `AT RISK` if 20% ≤ rate < 30%, `TRIPPED` if rate ≥ 30%.

If `TRIPPED`, surface it prominently in Step 2.6.

#### Step 2.5: Last verify-kernel run status

If any `*.dfy` files exist anywhere in the repo (check recursively):

- Look for evidence of a recent verify-kernel run: a CI log reference in the ROADMAP, a `.assurance/verify-kernel-last.json` (or equivalent), or the most recent commit to the Dafny files.
- Report the most recent known status (pass/fail/unknown) and timestamp.
- If no evidence of a recent run can be found, note "verify-kernel status unknown — trigger via CI job or `crosscheck:check-regressions`."

If no `*.dfy` files exist, skip this section and note "No Dafny kernels in repo — section skipped."

#### Step 2.6: Open kill-criterion triggers

Re-read the ROADMAP's kill-criteria section (if present) and any per-item kill criteria in the horizon docs. Cross-reference against the evidence gathered above:

- intent-check FP rate ≥ 30% → kill criterion tripped.
- Any item-level kill criterion that the earlier steps flagged as met.
- Any Status drift that explicitly references a kill-criterion trigger (e.g. "If Immediate item X not landed in 4 weeks, re-plan").

List each tripped criterion with a one-line justification and point at the relevant horizon doc.

#### Step 2.7: Compose the dashboard

Present the full dashboard in this order:

```
## Assurance Status — <repo name> — <today's date>

### Onboarding gate
- [x] ROADMAP present
- [x] Horizon directories present
- [x] Invariant docs present
- [x] Protected-surfaces rules present
- [ ] Coverage script wired  (warning only — see Step 1.5)

### ROADMAP drift
| Item | Documented | Observed | Note |
|---|---|---|---|
| ...

(N items unchanged)

### Invariant coverage
- Total invariant IDs: N
- Uncovered: M
- Top uncovered: I3, I7, I12, ...

### Protected-surface edits lacking governance notes
| File | Last touched | Commit | Missing note |
|---|---|---|---|
| ...

(or: "None in last 30 days.")

### intent-check FP tracker (rolling 2 weeks)
- Sample size: N
- False positives: M
- FP rate: X%  (kill criterion: 30%)
- Verdict: OK | AT RISK | TRIPPED | INSUFFICIENT DATA

### verify-kernel
- Last run: <timestamp or "unknown">
- Status: <pass | fail | unknown | N/A>

### Kill-criterion triggers
- [TRIPPED] intent-check FP rate ≥ 30% — see next/07 (if applicable)
- (or: "None tripped.")

### Recommended next steps
- <skill> — <one-line reason>
```

Keep the dashboard terse. The caller runs this often; verbosity erodes signal.

### Step 3: Verification Checklist

```
## Verification Checklist

- [ ] Phase 1 was run before any Phase 2 output was generated
- [ ] Each of the four hard-fail checks (ROADMAP, horizons, invariants, protected-surfaces) was evaluated by file existence only, not interpretation
- [ ] If any hard-fail check failed, the verbatim refusal message was emitted and Phase 2 was skipped entirely
- [ ] Coverage-script warning (Step 1.5) was surfaced in the dashboard if raised
- [ ] ROADMAP drift was computed against the `Not started / In progress / Blocked / Done / Deferred` vocabulary only — no other status values accepted
- [ ] Invariant coverage used the repo's coverage-script output when available, not ad-hoc grep, before falling back to grep
- [ ] Protected-surface review window is explicit (default 30 days) and stated in the dashboard
- [ ] FP-tracker rate was computed over the rolling 2-week window and compared against the 30% kill criterion
- [ ] verify-kernel section was skipped cleanly if no `*.dfy` files exist
- [ ] Every tripped kill criterion was surfaced in Step 2.6 with a link to its horizon doc
- [ ] Recommended next steps point at sibling skills (`/assurance-init`, `/assurance-layer-audit`, `/assurance-roadmap-check`, `/intent-check`, `/invariant-coverage-scaffold`, `/protected-surface-amend`) rather than proposing ad-hoc fixes
- [ ] No files were modified — this skill is read-only
```

## Arguments

Optional phase override. Default is auto (Phase 1 always runs; Phase 2 runs iff the gate passes).

Examples:
- `/assurance-status` — run both phases (default).
- `/assurance-status --phase=1` — run only the onboarding gate and report pass/fail without the dashboard.
- `/assurance-status --phase=2` — force the dashboard. Use only when you have already verified the gate passes; the skill still runs Phase 1 first and refuses if it fails.
