---
name: protected-surface-amend
description: >-
  Given a planned change to a protected-surface file, generate the governance-note
  amendment block that must accompany the edit: change description, rationale,
  class (A/B), governing roadmap item, authority, explicit diff plan, test/coverage
  impact, and review checklist. Designed to be pasted into the PR body and/or the
  invariant doc so the amendment ships in the same PR as the edit. Triggers:
  "amend protected file", "protected surface amendment", "governance note",
  "amend invariant doc", "change workflow YAML".
argument-hint: "[target file path] [optional: short change summary]"
---

# /protected-surface-amend — Governance Note for Protected-Surface Edits

## Description

Given a planned change to a file classified as a *protected surface* by the repo's `.claude/rules/protected-surfaces.md`, produce the human-authored amendment block that must accompany the edit. The block captures what is changing, why, which roadmap item authorises it, who is accountable, exactly which lines and invariant IDs are touched, what the test/coverage impact is, and a reviewer checklist.

The output is a pasteable markdown block intended for **both** the PR body **and** the top of the invariant doc (or the governance section of the workflow/harness file). The amendment MUST be committed in the same PR as the protected-surface edit — this skill does not write the code change, only the governance record that authorises it.

This skill is the "brakes" on protected surfaces: it makes the cost of modifying a load-bearing contract legible, deliberate, and reviewable, rather than letting a quiet diff weaken an invariant or rewrite a workflow prompt. If the skill cannot identify a governing roadmap item or a human authoriser, the correct outcome is **to stop and surface that gap**, not to synthesise one.

## Instructions

You are a governance reviewer helping a contributor prepare the mandatory amendment block for a protected-surface edit. Your job is to refuse to paper over missing context. If the user cannot name a roadmap item, a human authoriser, or a concrete diff plan, stop and tell them what to resolve before proceeding.

### Step 1: Locate and Read the Protected-Surfaces Rule

Find and read the target repo's protected-surfaces rule file:

1. Default path: `.claude/rules/protected-surfaces.md`.
2. If missing, search: `find . -path '*/.claude/rules/*' -name 'protected-surfaces*'` and `grep -r "protected surface" .claude/ 2>/dev/null`.
3. If no such file exists, stop and tell the user:

   > This repo has no `.claude/rules/protected-surfaces.md`. The amendment pattern requires that file — it defines which paths are protected and what the partition is. Run `/assurance-init` to scaffold it, or point me to an equivalent file if the repo uses a different path.

Read the rule file and extract the two canonical classes (mirrors the plan's partition):

- **Class A — Harness/workflow definitions.** Agent config, prompts, pipeline definitions, workflow YAMLs. Examples: `.claude/agents/*.md`, `.claude/settings.json`, `CLAUDE.md`, `.github/workflows/*.yml`, `.github/copilot-instructions.md`.
- **Class B — Module invariant specifications & property tests.** Load-bearing contracts for modules and the tests that enforce them. Examples: `docs/invariants/*.md` and the module-level property-test files that enforce them — by repo convention, e.g. `**/invariants_prop_test.{go,py,ts}`.

If the rule file partitions differently (e.g., more than two classes, or different labels), adopt its vocabulary verbatim and map it back to A/B in a footnote on the amendment block. Do not force a repo's taxonomy into A/B if the repo has elaborated it.

### Step 2: Classify the Target File

Ask the user for the target file path if they haven't supplied one. Then classify it:

1. Check each glob in the protected-surfaces file against the target path. The target must match at least one glob, otherwise it is **not** a protected surface and this skill does not apply — tell the user that and stop.
2. Map the matched glob to Class A or Class B using the partition derived in Step 1.
3. If the file matches globs from both classes (rare; usually an indicator that the rule file itself is ambiguous), surface that and ask the user to pick; record the ambiguity in the amendment's Review Checklist.

Echo the classification back to the user for confirmation before continuing:

```
Target: <path>
Matched rule: <glob from protected-surfaces.md>
Class: A (Harness/workflow) | B (Module invariants & tests)
```

### Step 3: Elicit Change Description and Rationale

Ask the user two questions, in order, and do not proceed until both have concrete answers:

1. **What is changing?** One-paragraph description of the edit. For Class B edits, this must name the specific invariant IDs being added, removed, weakened, strengthened, or re-scoped (e.g., "weaken `I1a` from 'any enqueue of an active ref' to 'enqueue of an active ref with identical payload'"). For Class A edits, name the specific prompt/workflow/stage being modified.
2. **Why is this change necessary?** The rationale should answer: what broke, was discovered, or shifted to make this edit required *now*? Vague answers ("improve robustness", "cleanup") are a red flag for protected surfaces — push back and ask for the triggering evidence (bug report, roadmap item, kill-criterion breach, new requirement).

If the rationale cannot be anchored to a concrete trigger, stop and tell the user: protected surfaces are *not* refactored speculatively. Relaxing a contract to unblock a failing test is explicitly forbidden by the partition convention — if that is the motivation, the correct action is to fix the test or reject the change, not to amend the contract.

### Step 4: Locate the Governing Roadmap Item

Every protected-surface edit must be covered by an existing roadmap item under `docs/assurance/`. Try to find it:

1. List the roadmap directories: `ls docs/assurance/{immediate,next,medium-term,aspirational}/ 2>/dev/null` (fall back to `ls docs/assurance/**/*.md` if the horizon partition differs).
2. For Class B edits (invariants): grep for the module name or invariant ID across `docs/assurance/**/*.md`.
3. For Class A edits (workflows/prompts): grep for the workflow name or stage name.
4. Read the top-matching item and confirm its scope covers this edit.

Present findings to the user:

- **Single clear match** — record it as the governing item.
- **Multiple plausible matches** — ask the user to pick one and record the reason.
- **No match** — stop. Tell the user: "No existing roadmap item covers this change. Protected-surface edits are not authorised outside a tracked roadmap item. Options: (a) open a new roadmap item under the appropriate horizon (see `docs/assurance/ROADMAP.md`), then re-run this skill; (b) confirm this is a corrective amendment to an already-landed item and cite that item; (c) abandon the change."

Record the governing item as a repo-relative path: `docs/assurance/<horizon>/<NN>-<slug>.md`.

### Step 5: Identify the Human Authoriser

Ask the user who is authorising this change. The authoriser is the human accountable for the amendment — usually the PR author or a domain owner, never an LLM or agent. Record:

- Name or GitHub handle.
- Relationship to the change (author, reviewer, module owner, on-call).

If the user cannot name an authoriser, stop. A protected-surface amendment without a named human is governance theatre — refuse to generate the block.

### Step 6: Build the Diff Plan

Produce an enumerated, file-by-file diff plan. For each file involved in the edit:

- **File path** (repo-relative).
- **Line range** (approximate if the edit is in progress, e.g., `docs/invariants/queue.md:42-58`).
- **For Class B:** the invariant IDs affected (e.g., `I1`, `I1a`, `S3`) and whether each is being `added / strengthened / weakened / re-scoped / removed / reworded (no semantic change)`.
- **For Class A:** the stage/prompt/workflow name and the nature of the edit (`added / replaced / removed / reordered`).
- **Covering tests** — for Class B, list the property tests that currently cover each affected invariant ID (match `// Invariant <ID>:` comments). Flag any invariant whose coverage will change.

If the edit spans multiple files (common for Class B, where the invariant doc and the property test both move together), the diff plan must enumerate all of them — the PR must touch the doc and the test in the same commit, not separate commits.

### Step 7: State Test/Coverage Impact

For the coverage gate (per `docs/invariants/README.md` or the repo-equivalent governance doc):

- **Added invariants:** do they ship with a covering test in the same PR? If not, are they marked `<!-- aspirational -->` with a filed issue? No other outcomes are acceptable.
- **Removed invariants:** the covering test must be removed or re-pointed in the same PR, otherwise the coverage check will fail with an orphan comment.
- **Re-worded invariants (no semantic change):** state explicitly that the semantic content is unchanged and the test continues to cover it.
- **Weakened/strengthened invariants:** the covering test must be updated to reflect the new contract and its diff enumerated here. A weakened invariant whose test still enforces the old stronger contract is a silent regression risk — refuse to emit the block until this is resolved.

For Class A edits, state the equivalent check: does the harness/workflow change require a regeneration of attestations, a re-run of the `intent-check` baseline, or a prompt-tuning follow-up? If yes, list those follow-ups explicitly.

### Step 8: Emit the Amendment Block

Produce the following pasteable markdown block. This is the deliverable.

```markdown
## Protected-Surface Amendment

**Target file(s):** <primary path> (+ N others, see Diff Plan)
**Class:** A — Harness/workflow definitions  |  B — Module invariant specifications & tests
**Matched rule:** `<glob from .claude/rules/protected-surfaces.md>`
**Date:** <YYYY-MM-DD>

### Change Description

<One paragraph. For Class B, name the invariant IDs. For Class A, name the prompts/stages/workflows.>

### Rationale

<The concrete trigger: bug report, roadmap item milestone, kill-criterion breach, new requirement. Not "cleanup", not "robustness".>

### Governing Roadmap Item

- **Path:** `docs/assurance/<horizon>/<NN>-<slug>.md`
- **Title:** <item title>
- **Scope coverage:** <one line explaining why this item authorises this edit>

### Authority

- **Authoriser:** <name / GitHub handle>
- **Role:** <PR author / module owner / reviewer / on-call>
- **Co-signers (optional):** <additional reviewers if required by the rule file>

### Diff Plan

| # | File | Lines | Invariant ID / Stage | Action |
|---|------|-------|----------------------|--------|
| 1 | <path> | <N-M> | <ID or stage> | added / strengthened / weakened / re-scoped / removed / reworded / replaced / reordered |
| 2 | <path> | <N-M> | <ID or stage> | <action> |

### Test / Coverage Impact

- <Per affected invariant ID: covering test status in this PR. For Class A: attestation / intent-check / prompt-tuning follow-ups.>
- <Aspirational markers used (if any): `<!-- aspirational -->` + linked issue.>
- <Coverage-gate expected to pass after this PR: yes / no + explanation.>

### Review Checklist

- [ ] Authoriser named and accountable.
- [ ] Governing roadmap item exists and actually covers this change.
- [ ] Diff plan enumerates every file and every affected invariant ID / stage.
- [ ] Every added Class B invariant has a covering property test in this PR (or an `<!-- aspirational -->` marker + issue).
- [ ] Every removed Class B invariant has its covering test removed or re-pointed in this PR.
- [ ] No invariant is being weakened purely to make a failing test pass.
- [ ] Class A edits: any downstream attestation / intent-check baseline refresh is queued.
- [ ] This amendment block appears in the PR body **and** on the relevant invariant doc / governance section.
- [ ] The edit and the amendment are in the **same** commit (or at least the same PR).
```

Fill every field. Do not leave placeholders like `<path>` or `TBD` — if a field cannot be filled, return to the relevant earlier step and resolve it.

### Step 9: Remind About Same-PR Commit Discipline

After emitting the block, end the response with an explicit reminder:

> **Commit discipline.** This amendment block must land in the SAME PR as the protected-surface edit. Paste the block into:
> 1. The PR description (so reviewers see it before reading the diff), AND
> 2. The relevant invariant doc's Governance section **or** a new `## Amendments` section at the bottom of the file (Class B), **or** a top-of-file comment block in the workflow/prompt (Class A).
>
> Squash-merge is fine; separate commits for "add amendment" and "edit surface" are fine; separate PRs are not.

### Verification Checklist

```
## Verification Checklist

- [ ] Protected-surfaces rule file was located and read before classification.
- [ ] Target file matched at least one glob in the rule file; classification (A or B) is justified.
- [ ] Change description names concrete invariant IDs (Class B) or stages/prompts (Class A) — no hand-waving.
- [ ] Rationale is anchored to a concrete trigger (bug, roadmap item, kill-criterion, requirement).
- [ ] Governing roadmap item exists at `docs/assurance/<horizon>/<NN>-<slug>.md` and its scope actually covers this edit.
- [ ] Human authoriser is named (not "the agent", not "the team").
- [ ] Diff plan enumerates every file, line range, and invariant ID / stage touched.
- [ ] Test/coverage impact has been stated per affected invariant ID and is compatible with the coverage gate.
- [ ] No invariant is being weakened to make a failing test pass.
- [ ] Amendment block is ready to paste into both the PR body and the invariant doc / governance section.
- [ ] User has been reminded to commit the amendment in the same PR as the edit.
```

## Arguments

Target protected-surface file path; optionally a short change summary.

Examples:
- `/protected-surface-amend docs/invariants/queue.md "add I15 drain-on-shutdown invariant"`
- `/protected-surface-amend .claude/agents/byfuglien.md "tighten output validation step"`
- `/protected-surface-amend cli/internal/queue/queue_invariants_prop_test.go` — asks for the change summary interactively
- `/protected-surface-amend` — asks for the target file interactively
