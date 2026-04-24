---
name: assurance-roadmap-check
description: >-
  Parse each roadmap item under docs/assurance/**/*.md, read its Status field,
  and diff the declared state against observed repo/PR state. Flags drift in
  both directions — docs that say Done but lack a merged PR or expected
  artifact, docs that say Not started but whose code already exists, stale
  In progress items with no recent commits, and Deferred items missing a
  Reason. Intended cadence: weekly. Triggers: "check assurance roadmap",
  "roadmap drift", "status drift", "roadmap status check", weekly assurance
  review.
argument-hint: "[optional: horizon filter — immediate|next|medium-term|aspirational]"
---

# /assurance-roadmap-check — Assurance Roadmap Status Drift Detection

## Description

Walk every horizon doc under `docs/assurance/**/*.md`, extract each item's declared `Status` field, and compare it against observed repo and PR state. Report drift in both directions — overclaim (doc says Done, no merged PR) and underclaim (doc says Not started, code already exists) — plus staleness and missing `Reason:` lines on deferred items. Produce a structured table and a suggested maintenance action per row.

This is a **maintenance** skill for repos already onboarded to the assurance hierarchy (see `/assurance-init`). It surfaces ROADMAP rot before it becomes load-bearing — the roadmap documents are the durable record of assurance decisions, so their Status fields must stay truthful. The intended cadence is **weekly**, aligned with the "How the hierarchy is maintained over time" section of the assurance skills plan.

## Instructions

You are an assurance governance expert helping the user keep each `docs/assurance/**/*.md` item's declared `Status` field consistent with the repo's actual state. The roadmap docs are the source of truth for scope, acceptance criteria, and kill criteria; if their Status fields drift from reality, every downstream assurance decision is built on a false premise.

The status vocabulary is fixed (set during `/assurance-init`):

| Status | Meaning |
|---|---|
| **Not started** | No work has begun. |
| **In progress** | Work has started but is not merged. |
| **Blocked** | Work is paused on an external dependency. Expect a pointer to the blocker. |
| **Done** | Work is merged. Expect a PR number and landing date. |
| **Deferred** | Work is intentionally postponed or dropped. **Requires** a `Reason:` line on the next line or inline. |

### Step 1: Locate Roadmap Directory

Look for `docs/assurance/` in the project root.

**If the directory does not exist:**
- Inform the user: "No `docs/assurance/` directory found. This skill operates on repos onboarded to the assurance hierarchy. Run `/assurance-init` first to scaffold the roadmap, or `/assurance-layer-audit` if you haven't scoped the hierarchy for this repo yet."
- Stop here.

**If the directory exists but contains no horizon subdirectories with item docs:**
- Inform the user: "`docs/assurance/` exists but contains no horizon items. The roadmap lives under `docs/assurance/{immediate,next,medium-term,aspirational}/<NN-slug>.md`. Run `/assurance-init` to populate it."
- Stop here.

Otherwise, enumerate the item docs:
- Walk `docs/assurance/**/*.md`.
- **Exclude** `README.md`, `ROADMAP.md`, and `skills-plan.md` at any depth — these are directory-level docs, not roadmap items.
- Respect the optional `argument-hint` filter (e.g., `immediate` narrows to `docs/assurance/immediate/*.md`).

Summarise what was found:
- Total item docs by horizon.
- Path to the roadmap table (`docs/assurance/ROADMAP.md`) if present — cross-reference rows later to catch "roadmap table lists an item but no doc exists" and vice versa.

### Step 2: Parse Declared Status

For each item doc, extract the declared Status and its supporting fields:

1. **Status line** — look for the first `**Status:**` or `Status:` line in the first ~25 lines of the doc. The value is the phrase after the colon and before the next `**` or newline. Normalise trailing context (e.g., `Done — PR #665 merged 2026-04-19` → declared status `Done` plus landing metadata).
2. **PR references** — capture any `#<number>` or `PR #<number>` tokens on the Status line or within the first few lines. These are the strong signal for "Done" verification.
3. **Reason** — for `Deferred`, look for a `Reason:` field on the same line or on the following line. If absent, flag it in Step 4.
4. **Blocker** — for `Blocked`, look for a `Blocker:` field on the same line or on the following line (this is the label scaffolded by `/assurance-init`; tolerate legacy variants `Blocks on:` / `Depends on:` but prefer `Blocker:`). If absent, flag it in Step 4.
5. **Expected artifacts** — scan the doc's **Deliverables** / **Files to touch** / **Scope** sections for explicit file paths. These anchor the "does the artifact exist?" check in Step 3.
6. **Item ID** — the leading `NN-slug` from the filename (e.g., `01-invariant-test-coverage-ci.md` → `01`). Use it in the output table.

**If the Status field is missing entirely:**
- Record the doc under a `MALFORMED` bucket with drift type `Missing Status field`. Do not attempt to infer.

**If the Status value is not in the controlled vocabulary:**
- Record it under `MALFORMED` with drift type `Unknown status value: "<value>"`. Do not attempt to map to a known status.

### Step 3: Gather Observed State

For each well-formed item, gather observed state from the repo. Use read-only commands only; do not mutate anything.

1. **PR search** — if the doc cites `#<number>`, run `git log --all --oneline --grep="#<number>"` (or `gh pr view <number> --json state,mergedAt,title` if `gh` is available) to confirm the PR merged. If no PR number is cited but Status is `Done`, search `git log --oneline` for recent commits referencing the item slug or item number (`grep "01-invariant" | grep -i "merge"`).
2. **Artifact existence** — for each expected artifact path from the doc, check whether it exists on disk and whether it was last modified after an "item began" marker. If the doc cites specific new files (e.g., `scripts/check_invariant_coverage.py`), use `ls` / `git log -- <path>` to confirm presence and recency.
3. **Recency** — for each item claimed `In progress`, run `git log --since="30 days ago" --oneline -- <expected-file-paths>`. If no commits in the last 30 days touch the relevant files, the item is **stale**.
4. **Reverse check** — if Status is `Not started`, run `ls` / `git log` over the expected file paths. If the artifacts already exist, the doc is out of date in the underclaim direction.
5. **ROADMAP table cross-reference** — if `docs/assurance/ROADMAP.md` contains a summary table with per-item Status columns or inline Status phrases, parse those rows and record them alongside each item's own declared Status. Mismatches surface as the `ROADMAP mismatch` drift type in Step 4.

Summarise observed state per item:

- `PR merged: <number> @ <date>` / `PR not found` / `no PR cited`
- `artifact present: <path>` / `artifact absent: <path>`
- `last commit: <date> (<N days ago>)` / `no commits on record`

### Step 4: Classify Drift

For each item, emit one of the following drift classifications. An item may trigger multiple; report all that apply. The **Suggested Action** column below is what the user sees in the Step 5 report — drift reports are advisory; do **not** edit the docs automatically.

| Drift Type | Condition | Suggested Action |
|---|---|---|
| **None** | Declared status matches observed state. | No action. |
| **Overclaim — Done but no PR** | Status `Done`, but no merged PR is discoverable and no cited PR number resolves. | Open `<file>` and change `**Status:** Done` to `**Status:** In progress` (or cite the PR number if one exists but wasn't captured). |
| **Overclaim — Done but artifact absent** | Status `Done`, PR merged, but an expected artifact path from Deliverables/Files-to-touch is missing from the repo. | Investigate whether the artifact was rolled back or renamed; update the doc's Deliverables list or restore the artifact. |
| **Underclaim — Not started but artifact exists** | Status `Not started`, but expected artifact paths are already present in the repo. | Open `<file>` and change `**Status:** Not started` to `**Status:** Done — PR #<N> merged <date>`, using `git log -- <artifact-path>` to find the landing PR. |
| **Stale In progress** | Status `In progress`, no commits touching the relevant files in > 30 days. | Confirm the gap with `git log --since='30 days ago' -- <paths>`. If genuinely paused, change Status to `Blocked` and add a `**Blocker:** <pointer>` line (the label scaffolded by `/assurance-init`); otherwise revert to `Not started` with a rationale. |
| **Missing Reason** | Status `Deferred` with no `Reason:` field. | Open `<file>` and add a `**Reason:** <why this was deferred>` line immediately after the Status line. Link the superseding item if one exists. |
| **Blocked without blocker** | Status `Blocked` with no explicit pointer to the blocker (another item doc, issue, or external dependency). | Open `<file>` and add a `**Blocker:** <pointer>` line naming the blocker (matches the label scaffolded by `/assurance-init`). |
| **Malformed** | Status field missing, misspelled, or not in the controlled vocabulary. | Open `<file>` and replace the Status value with one of: `Not started`, `In progress`, `Blocked`, `Done`, `Deferred`. |
| **ROADMAP mismatch** | The item doc's Status is out of sync with how `ROADMAP.md` summarises it. | Update whichever of the two is stale; the item doc is the source of truth, so `ROADMAP.md` is the more common edit. |

### Step 5: Emit Report

Present a structured table with the columns below. Group rows by drift type, then by horizon, so the user sees concentrations of drift at a glance.

```
## Assurance Roadmap Drift Report

Scanned N item docs across <horizons>. K items have drift; M are clean.

| File | Item | Declared Status | Observed Status | Drift Type | Suggested Action |
|------|------|-----------------|-----------------|------------|------------------|
| immediate/01-invariant-test-coverage-ci.md | 01 | Done (PR #665) | PR #665 merged; artifact `scripts/check_invariant_coverage.py` absent | Overclaim — Done but artifact absent | Investigate whether the script was renamed or rolled back; update doc or restore file. |
| next/06-queue-dafny-kernel.md | 06 | In progress | last commit 47 days ago on `dafny/queue.dfy` | Stale In progress | Confirm status; move to Blocked with pointer to blocker or back to Not started. |
| aspirational/14-spec-adversary-phase.md | 14 | Deferred | — | Missing Reason | Add `Reason: <why>` line citing the deferral rationale. |

## Summary

- Clean: M items
- Overclaim (Done without evidence): X
- Underclaim (Not started but code present): Y
- Stale In progress (> 30 days): Z
- Missing Reason on Deferred: W
- Malformed Status: V
- ROADMAP.md table mismatches: U
```

If no drift is found, emit a short confirmation: `All N roadmap items' Status fields match observed state. Next check recommended in 7 days.`

Close the report by reminding the user of cadence:

> This check is intended to run **weekly**. Consider scheduling it (cron, calendar reminder, or a `weekly-assurance-review` job) so drift is caught before it compounds. If your repo tracks assurance cadence elsewhere (e.g., `docs/assurance/README.md`), update the "last roadmap-check run" timestamp there.

### Verification Checklist

- [ ] Every `docs/assurance/**/*.md` item doc (excluding READMEs and `ROADMAP.md`) has a row in the report
- [ ] Each row's declared Status is exactly one of the controlled vocabulary values or classified as Malformed
- [ ] Each `Done` row has been cross-checked against `git log` for the cited PR (or flagged as overclaim)
- [ ] Each `In progress` row has been checked for commits in the last 30 days
- [ ] Each `Deferred` row has been checked for a `Reason:` field
- [ ] Each `Blocked` row has been checked for a blocker pointer
- [ ] If `docs/assurance/ROADMAP.md` summarises items in a table, rows are cross-checked against the per-item docs
- [ ] The report groups drift by type so concentrations are visible
- [ ] Next-check cadence (weekly) surfaced to the user

## Arguments

Optional horizon filter narrows the scan to a single horizon directory.

Examples:
- `/assurance-roadmap-check` — scan every horizon.
- `/assurance-roadmap-check immediate` — scan only `docs/assurance/immediate/*.md`.
- `/assurance-roadmap-check next` — scan only `docs/assurance/next/*.md`.
- `/assurance-roadmap-check medium-term` — scan only `docs/assurance/medium-term/*.md`.
- `/assurance-roadmap-check aspirational` — scan only `docs/assurance/aspirational/*.md`.
