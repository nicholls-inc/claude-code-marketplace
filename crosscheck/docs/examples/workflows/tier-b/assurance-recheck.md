---
description: |
  Force-recheck handler for the Assurance PR-Gate. Contributors can comment
  `/assurance-recheck <INVARIANT-ID>` on a PR to bypass the content-hash
  cache for that invariant and rerun /crosscheck:intent-check fresh.

  This is a standalone workflow because gh-aw forbids `slash_command` and
  `pull_request` triggers in the same workflow. It shares the same plugin
  install + per-invariant plan computation as assurance-pr-gate.md, but
  scopes its work to a single invariant ID extracted from the slash
  command body.

  Trigger: slash_command: /assurance-recheck <INVARIANT-ID>

on:
  slash_command:
    name: assurance-recheck

timeout-minutes: 20

permissions: read-all

network:
  allowed:
    - defaults
    - python

checkout:
  fetch-depth: 0

tools:
  bash:
    - "ls *"
    - "find *"
    - "grep *"
    - "git diff *"
    - "git log *"
    - "git show *"
    - "git rev-parse *"
    - "git status"
    - "git checkout *"
    - "git switch *"
    - "git add *"
    - "git commit *"
    - "git push *"
    - "gh pr *"
    - "gh issue *"
    - "gh api *"
    - "python3 *"
    - "head *"
    - "tail *"
    - "wc *"
    - "sha256sum *"
    - "cat *"
    - "awk *"
    - "sed *"
    - "jq *"
    - "mkdir *"
    - "cp *"
  github:
    toolsets: [all]
    min-integrity: none

steps:
  - name: Install crosscheck plugin (latest from nicholls marketplace)
    run: |
      mkdir -p "${RUNNER_TEMP}/gh-aw/marketplace"
      git clone --depth 1 \
        https://github.com/nicholls-inc/claude-code-marketplace.git \
        "${RUNNER_TEMP}/gh-aw/marketplace"

  - name: Resolve target invariant from slash args
    # gh-aw forbids ${{ github.event.comment.body }} in workflow expressions
    # (injection vector). We pass only the comment ID (a safe integer) and
    # fetch the body via gh api at runtime, then parse it inside the shell.
    # The parsed invariant ID is written to /tmp/gh-aw/recheck_target.json
    # for the agent to read.
    env:
      COMMENT_ID: ${{ github.event.comment.id }}
      ISSUE_NUMBER: ${{ github.event.issue.number }}
    run: |
      mkdir -p /tmp/gh-aw "${RUNNER_TEMP}/gh-aw"
      BODY=$(gh api "repos/$GITHUB_REPOSITORY/issues/comments/$COMMENT_ID" --jq .body)
      # Parse: /assurance-recheck <ID>. ID = uppercase alnum + - _
      INVARIANT_ID=$(printf '%s\n' "$BODY" \
        | grep -oE '/assurance-recheck[[:space:]]+[A-Z0-9_-]+' \
        | head -n1 \
        | awk '{print $2}')
      jq -n \
        --arg id "${INVARIANT_ID:-}" \
        --arg comment "$COMMENT_ID" \
        --arg issue "$ISSUE_NUMBER" \
        --arg body "$BODY" \
        '{invariant_id: $id, comment_id: ($comment|tonumber), pr_number: ($issue|tonumber), raw_body: $body}' \
        > /tmp/gh-aw/recheck_target.json
      cp /tmp/gh-aw/recheck_target.json "${RUNNER_TEMP}/gh-aw/recheck_target.json"
      echo "Parsed invariant_id: ${INVARIANT_ID:-<empty>}"

  - name: Compute per-invariant plan (deterministic pre-step)
    env:
      ASSURANCE_WORK_DIR: ${{ runner.temp }}/gh-aw
    run: |
      python3 .github/workflows/scripts/assurance_pr_gate_plan.py
      cp "${ASSURANCE_WORK_DIR}/pr_gate_plan.json" /tmp/gh-aw/pr_gate_plan.json

engine:
  id: claude
  args:
    - "--plugin-dir"
    - "${{ runner.temp }}/gh-aw/marketplace/crosscheck"

safe-outputs:
  mentions: false
  add-comment:
    max: 2   # one verdict comment + one error/usage comment if mis-formatted
  push-to-pull-request-branch:
    target: "triggering"
    title-prefix: "[Assurance Squad] "
    protected-files: fallback-to-issue
    max: 1
---

# Assurance Recheck

You are the force-recheck handler for the Assurance PR-Gate. A contributor
commented `/assurance-recheck <INVARIANT-ID>` on a PR to bypass the cache
for one invariant.

---

## Phase 0 — Read the parsed slash args

A pre-agent step has already parsed the slash command body. Read the
result:

```bash
cat /tmp/gh-aw/recheck_target.json
```

Fields:
- `invariant_id` — extracted ID (empty string if the comment didn't match
  `/assurance-recheck <ID>`)
- `pr_number` — the PR the comment was made on
- `comment_id` — the originating comment

If `invariant_id` is empty, post a single comment and **stop**:

> **Usage:** `/assurance-recheck <INVARIANT-ID>` (e.g.
> `/assurance-recheck QUEUE-001`).

If `invariant_id` is present, continue.

---

## Phase 1 — Locate the invariant

Look up the invariant ID in the PR's changed `docs/invariants/*.md` files.
The pre-step has already computed the per-invariant plan in
`/tmp/gh-aw/pr_gate_plan.json`.

```bash
cat /tmp/gh-aw/pr_gate_plan.json
```

Find the entry where `invariant_id == <ID>`. If no entry exists (the
invariant isn't in any changed doc), post:

> **Cannot recheck `<ID>`** — that invariant isn't in any
> `docs/invariants/*.md` file changed by this PR. Recheck applies only to
> invariants whose docs / tests / module source are part of the PR diff.

…and stop.

---

## Phase 2 — Run intent-check fresh (bypass cache)

For the located invariant entry, **ignore** the `cache_hit` and
`cached_attestation` fields entirely. Run `/crosscheck:intent-check` fresh
with:

- The invariant prose (from the changed doc).
- The covering test sources (paths in `covering_tests`).
- The module source diff
  (`git diff origin/<base>...HEAD -- <module-paths>`).

The skill will append a row to `.assurance/fp-tracker.csv` and emit a JSON
attestation. **Save it to**
`docs/assurance/attestations/<content_hash>.json` and push it back to the
PR branch via the `push-to-pull-request-branch` safe-output.

If the new content hash matches the cached one but the verdict differs,
that's a notable event — log it in the comment ("Force-recheck produced
a different verdict from the cached attestation; this is rare and worth
investigating").

---

## Phase 3 — Post the verdict

Post a single PR comment:

> **Layer 5 — Force-rechecked (cache bypassed)**
>
> Invariant: `<ID>` (`<name>`) in `docs/invariants/<module>.md`
> Verdict: **<PASS|FLAG|UNCERTAIN>**
> Attestation: `sha256:<content_hash>` (`docs/assurance/attestations/<hash>.json`)
> Triggered by: `/assurance-recheck` on `<commenter handle>`
> FP-tracker rolling 30 d: <rate>% (n=<count>)
>
> <one-paragraph summary of the back-translation diff>
>
> _Layer 5 is probabilistic; this re-run bypassed the cache by request._

If the new verdict differs from the cached one, prepend:

> **Verdict changed on force-recheck.** Cached: `<old>`; fresh:
> `<new>`. Inputs hash unchanged → the LLM produced a different
> back-translation. This is a rare signal — please surface to the
> assurance maintainer.

---

## Operating principles

1. **Single invariant only.** This workflow rechecks exactly one ID. Do
   not run a generic gate pass.
2. **Cache bypass is the whole point.** Never short-circuit. Always run
   intent-check fresh.
3. **Push the attestation.** The fresh attestation must be committed and
   pushed via `push-to-pull-request-branch` so the new verdict becomes
   the canonical one for that hash.
4. **No silent failures.** If the invariant ID can't be located or the
   slash command is mis-formatted, post a clear usage / "not found"
   comment and stop.

Generated by Assurance Recheck. Sibling of Assurance PR-Gate.
