---
name: protected-surface-amend
add-mode: bootstrap
description: >-
  Given a planned change to a protected-surface file, generate the governance-note
  amendment block that must accompany the edit: change description, rationale,
  class (A/B), governing roadmap item, authority, explicit diff plan, test/coverage
  impact, and review checklist. Primarily agent-invocable — the implementer agent
  making the protected-surface edit calls this skill, which reads the staged diff,
  drafts every section, and emits the block into the PR description. The human
  renders the governance verdict via PR review, not via runtime prompts. Triggers:
  "amend protected file", "protected surface amendment", "governance note",
  "amend invariant doc", "change workflow YAML".
argument-hint: "[target file path] [optional: short change summary]"
---

# /protected-surface-amend — Governance Note for Protected-Surface Edits

## Description

Given a planned change to a file classified as a *protected surface* by the repo's `.claude/rules/protected-surfaces.md`, produce the amendment block that must accompany the edit. The block captures what is changing, why, which roadmap item authorises it, who is accountable, exactly which lines and invariant IDs are touched, what the test/coverage impact is, and a reviewer checklist.

This skill is the "brakes" on protected surfaces: it makes the cost of modifying a load-bearing contract legible, deliberate, and reviewable, rather than letting a quiet diff weaken an invariant or rewrite a workflow prompt.

**The human's governance moment is the PR review, not the skill invocation.** The skill drafts every field from the staged diff, the commit messages, and the repo's roadmap directory. Fields that genuinely require human judgement (rationale-anchor, authoriser, roadmap-item disambiguation) are drafted with the best available evidence and emitted into the PR description with explicit `REQUIRES HUMAN VERIFICATION:` markers where the draft is uncertain. The reviewer red-pens those at merge time.

**Primary caller: an implementer agent** finishing a protected-surface edit. The agent invokes this skill after staging the edit; the skill writes the amendment artifact and the agent paste it into the PR description.

**Secondary caller: a human** running `/protected-surface-amend` directly. In this mode the skill falls back to interactive prompts only when the staged diff is empty or no commit context is available.

If the skill cannot identify a governing roadmap item, the correct outcome is **to stop and surface that gap**, not to synthesise one. That refusal is the only irreducible governance gate; everything else is draft-and-review.

## Instructions

You are drafting the mandatory amendment block for a protected-surface edit. Your job is to do the analysis the implementer agent would otherwise push at the user, and to surface only the irreducibly-human decisions in the PR description for review-time red-pen.

### Step 1: Detect mode

Look for a marker file per `crosscheck/docs/orchestrator-coordination.md` §1: walk from cwd to git repo root looking for `.assurance/add-session-*/session.json`. If found, validate per the protocol there and operate in **orchestrator-marker mode** (the orchestrator owns the downstream governance gate; this skill writes the artifact and returns without prompting).

If no marker is found, check whether a staged diff exists (`git diff --staged --name-only`). If non-empty, operate in **agent mode** (an implementer agent has staged a change and is calling this skill to draft the amendment). If empty, operate in **interactive mode** (a human invoked the skill directly; prompt for missing context).

Record the mode for subsequent steps. In orchestrator-marker and agent modes the skill is non-interactive: no `AskUserQuestion` calls. In interactive mode, fall back to the prompts in Steps 2–5 only when a field cannot be drafted from repo state.

### Step 2: Read the protected-surfaces rule and identify the target file

1. Locate `.claude/rules/protected-surfaces.md` (default path; search `.claude/rules/` if missing).
2. If the rule file does not exist, stop and emit:
   > This repo has no `.claude/rules/protected-surfaces.md`. The amendment pattern requires that file — it defines which paths are protected and what the partition is. Run `/assurance-init` to scaffold it, or point me to an equivalent file if the repo uses a different path.
3. Extract the canonical classes:
   - **Class A — Harness/workflow definitions.** Agent config, prompts, pipeline definitions, workflow YAMLs.
   - **Class B — Module invariant specifications & property tests.** Load-bearing contracts and the tests that enforce them.

   If the rule file uses a different partition, adopt its vocabulary verbatim and map back to A/B in a footnote on the amendment block.

4. Identify the target file(s):
   - **Orchestrator-marker or agent mode:** derive from `git diff --staged --name-only` filtered against the protected-surfaces globs. If the staged diff touches multiple protected files, treat them as one logical amendment (one block, multi-file diff plan).
   - **Interactive mode:** use the supplied argument if present; otherwise ask once.

5. Classify each target file by matching against the protected-surfaces globs. If no match, the file is not a protected surface and this skill does not apply — surface that and stop. If a file matches globs from both classes, note the ambiguity in the amendment's Review Checklist; pick the stricter class for the block.

### Step 3: Auto-fill change description, diff plan, and coverage impact

Draft these three sections directly from repo state. Do not ask the user.

**Change description** (one paragraph):
- For Class B targets: parse the staged diff for added/removed/changed invariant IDs (look for `## I<N>:` or `### I<N>:` headings and their context). Classify each as `added | strengthened | weakened | re-scoped | removed | reworded (no semantic change)`. State the IDs and the actions explicitly.
- For Class A targets: identify modified stages/prompts/workflows by inspecting the staged hunks. Name them and state the nature of the edit (`added | replaced | removed | reordered`).

**Diff plan** (table):
- Use `git diff --staged --stat` for the file list and `git diff --staged` for the line ranges and invariant IDs.
- For Class B: find covering property tests by grepping `// Invariant <ID>:` (or the repo's equivalent comment style) across the test directories. Flag any invariant whose coverage changes.
- For Class A: identify any downstream attestation/intent-check baseline refresh needed.

**Coverage impact**:
- For added Class B invariants: verify they ship with a covering test in the same staged diff (grep the staged test files for `// Invariant <new-ID>:`). If not, flag as `<!-- aspirational -->` candidate.
- For removed Class B invariants: verify the covering test is also being removed or re-pointed.
- For weakened/strengthened invariants: verify the test is updated. A weakened invariant whose test still enforces the old stronger contract is a silent regression risk — emit a **BLOCKING:** line in the output (see Step 6).
- For Class A: state explicitly whether attestation regeneration, intent-check baseline refresh, or prompt-tuning follow-up is required.

If any of these cannot be derived (e.g., the staged diff is empty in agent mode), this is a genuine error: report it and stop. The amendment cannot be drafted from no evidence.

### Step 4: Draft rationale (the irreducible governance content)

Rationale is the brake. It must anchor the change to a concrete trigger: a bug report, a roadmap item milestone, a kill-criterion breach, a new requirement. Vague rationales ("cleanup", "improve robustness") are explicitly rejected.

**In orchestrator-marker and agent modes:**

1. Extract candidate rationale from:
   - Staged commit messages (`git log <upstream>..HEAD --pretty=%B`).
   - The current branch name (often encodes the issue or motivation).
   - If a PR exists for the branch (`gh pr view --json body 2>/dev/null`), the PR body.
   - Any `docs/incidents/`, `docs/postmortems/`, or recent merged PRs that touch the same module.
2. Score the candidate against the anchor test: does it name a *concrete trigger* (issue ID, audit-finding ID, postmortem date, kill-criterion name, RFC-2119 requirement)?
3. If anchored: include verbatim in the rationale field.
4. If not anchored or no candidate: emit `REQUIRES HUMAN VERIFICATION: Rationale draft below is not anchored to a concrete trigger. Reviewer must replace with an anchored rationale or escalate to governance.` and include the best draft + a one-line note on what's missing.

**In interactive mode:** ask the user once. If their answer is vague, push back with the anchor test verbatim and stop until they provide an anchored answer.

### Step 5: Locate the governing roadmap item (hard refusal if missing)

Every protected-surface edit must be covered by an existing roadmap item under `docs/assurance/`. This is the one non-negotiable gate.

1. List roadmap directories: `ls docs/assurance/{immediate,next,medium-term,aspirational}/ 2>/dev/null` (fall back to `find docs/assurance -name '*.md'`).
2. For Class B edits: grep for the module name and any affected invariant IDs.
3. For Class A edits: grep for the workflow/stage/prompt name.
4. Read the top-matching item and confirm scope.

Outcomes:

- **Single clear match** — record as the governing item.
- **Multiple plausible matches** — orchestrator-marker / agent mode: emit `REQUIRES HUMAN VERIFICATION: Multiple roadmap items match. Reviewer must select one.` with the candidates listed. Interactive mode: ask the user.
- **No match** — stop in all modes. Emit:
  > No existing roadmap item covers this change. Protected-surface edits are not authorised outside a tracked roadmap item. Options: (a) open a new roadmap item under the appropriate horizon (see `docs/assurance/ROADMAP.md`), then re-run; (b) confirm this is a corrective amendment to an already-landed item and cite that item explicitly; (c) abandon the change.

The "no roadmap item" stop is unconditional: it is not relaxed in agent mode. The whole point of this skill is to refuse synthetic governance.

### Step 6: Draft authoriser

The authoriser is the human accountable for the amendment — usually the PR author or a domain owner, never an LLM or agent.

**Orchestrator-marker / agent mode:**

1. Default to the PR author if a PR exists (`gh pr view --json author -q .author.login`).
2. Otherwise default to the commit author of the most recent staged commit (`git log -1 --pretty=%an`).
3. If neither is available or the candidate looks bot-like (matches `*-bot`, `*[bot]*`, `claude-*`), emit `REQUIRES HUMAN VERIFICATION: Authoriser draft is unverified. Reviewer must confirm or replace with named human.`

**Interactive mode:** ask the user. Refuse to proceed if they cannot name a human.

### Step 7: Emit the amendment block

Produce the following pasteable markdown block. Every field must be filled from Steps 2–6, with `REQUIRES HUMAN VERIFICATION:` markers where draft confidence is low. Do not leave `<placeholder>` text in the output — if a field genuinely cannot be drafted, that is an error and the skill should have stopped earlier.

```markdown
## Protected-Surface Amendment

**Target file(s):** <primary path> (+ N others, see Diff Plan)
**Class:** A — Harness/workflow definitions  |  B — Module invariant specifications & tests
**Matched rule:** `<glob from .claude/rules/protected-surfaces.md>`
**Date:** <YYYY-MM-DD>

### Change Description

<One paragraph drafted from the staged diff. Class B: invariant IDs + action verbs. Class A: stage/prompt names + action verbs.>

### Rationale

<Anchored rationale, or REQUIRES HUMAN VERIFICATION: line + best draft.>

### Governing Roadmap Item

- **Path:** `docs/assurance/<horizon>/<NN>-<slug>.md`
- **Title:** <item title>
- **Scope coverage:** <one line on why this item authorises this edit>

### Authority

- **Authoriser:** <name / GitHub handle> (or REQUIRES HUMAN VERIFICATION:)
- **Role:** <PR author / module owner / reviewer / on-call>

### Diff Plan

| # | File | Lines | Invariant ID / Stage | Action |
|---|------|-------|----------------------|--------|
| 1 | <path> | <N-M> | <ID or stage> | added / strengthened / weakened / re-scoped / removed / reworded / replaced / reordered |

### Test / Coverage Impact

- <Per affected invariant ID: covering test status in this PR. Class A: attestation / intent-check / prompt-tuning follow-ups.>
- <BLOCKING: lines for any silent-regression risks detected in Step 3.>

### Review Checklist

- [ ] Rationale is anchored to a concrete trigger (not "cleanup" or "robustness").
- [ ] Authoriser is a named human (not a bot, not an agent).
- [ ] Governing roadmap item exists and actually covers this change.
- [ ] Diff plan enumerates every affected file, line range, and invariant ID / stage.
- [ ] Every added Class B invariant has a covering property test in this PR (or `<!-- aspirational -->` + linked issue).
- [ ] Every removed Class B invariant has its covering test removed or re-pointed in this PR.
- [ ] No invariant is being weakened purely to make a failing test pass.
- [ ] Class A edits: downstream attestation / intent-check baseline refresh is queued.
- [ ] This amendment block appears in the PR body **and** on the relevant invariant doc / governance section.
- [ ] All `REQUIRES HUMAN VERIFICATION:` markers above have been resolved.
```

### Step 8: Output destination

Write the amendment block as a file artifact (per `crosscheck/docs/orchestrator-coordination.md` §2). Do not return only as a chat block.

- **Orchestrator-marker mode:** write to `.assurance/add-session-<id>/protected-surface-amend-<target-slug>.md`. The orchestrator's apply step consumes it.
- **Agent mode (no marker):** write to `.assurance/protected-surface-amend/<target-slug>-<YYYY-MM-DD>.md`. The implementer agent reads it and pastes the block into the PR description.
- **Interactive mode:** also emit the block in chat for immediate human paste, *and* write to the agent-mode path so it persists.

Report the written path and a one-line summary. Do not paste the full block in chat as the primary output (chat-only output is the state-carrier anti-pattern this refactor removes).

### Step 9: Same-PR commit discipline

The amendment block must land in the **same PR** as the protected-surface edit. In agent mode this is automatic — the implementer agent is staging both. In interactive mode, surface the reminder:

> Commit this amendment block in the same PR as the protected-surface edit. Paste it into the PR description AND into the relevant invariant doc's governance section (or a top-of-file comment block for Class A). Separate PRs for "add amendment" and "edit surface" are not acceptable.

## Arguments

Target protected-surface file path (optional in orchestrator-marker / agent mode where the staged diff is the source of truth); optionally a short change summary.

Examples:

- `/protected-surface-amend` — orchestrator-marker / agent mode; reads staged diff, no further prompts.
- `/protected-surface-amend docs/invariants/queue.md` — interactive mode targeting a specific file.
- `/protected-surface-amend docs/invariants/queue.md "add I15 drain-on-shutdown invariant"` — interactive mode with a change summary hint.
