---
description: |
  Mandatory Layer 5 gate for protected-surface diffs. Runs /intent-check
  on each changed invariant, with content-hashed attestations as a cache.
  Pushes attestation JSON back to the PR branch so the cache warms over
  time. Posts governance-amendment reminders and Dafny hand-off comments.

  Layer 4 coverage enforcement is owned by .github/workflows/assurance.yml
  (static, runs on every push/PR). This workflow does NOT duplicate
  coverage checking; it focuses on Layer 5 and governance comments.

  Cache short-circuit: per-invariant content hash =
    sha256(invariant_id || prose || covering_test_sources || module_source)
  On hit (and kill-criterion not active), reuse
  docs/assurance/attestations/<hash>.json with explicit "(cached,
  originally checked YYYY-MM-DD)" labelling.

  Force-recheck: contributors can run `/assurance-recheck <ID>` on a PR
  comment to bypass the cache. That trigger lives in a sibling workflow
  (assurance-recheck.md) — gh-aw forbids slash_command and pull_request
  in the same workflow.

  Trigger: pull_request opened/synchronize/reopened/ready_for_review,
  paths covering protected surfaces.

on:
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review]
    paths:
      - 'docs/invariants/**'
      - 'docs/assurance/**'
      - '.claude/rules/protected-surfaces.md'
      - 'tests/**/*invariant*'
      - 'tests/**/*property*'
      - '**/*_property_test.py'
      - '**/*_property_test.go'
      - '**/*.property.test.ts'
      - '**/*.property.test.tsx'

timeout-minutes: 30

# gh-aw strict mode: write operations must go through safe-outputs (see
# below), not via raw permissions. Job runs read-only; safe-output side
# jobs handle: add-comment (Layer 5 verdicts), create-issue (kill-criterion
# alert), and push-to-pull-request-branch (attestation JSON).
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

  - name: Compute per-invariant plan (deterministic pre-step)
    env:
      # PR body via env (NEVER interpolated into a `run` script body).
      # The Python script reads via os.environ and uses it only for
      # substring detection of "## Governance Amendment".
      GH_AW_PR_BODY: ${{ github.event.pull_request.body }}
      GITHUB_BASE_REF: ${{ github.event.pull_request.base.ref || 'main' }}
      ASSURANCE_WORK_DIR: ${{ runner.temp }}/gh-aw
    run: |
      python3 .github/workflows/scripts/assurance_pr_gate_plan.py
      mkdir -p /tmp/gh-aw
      cp "${ASSURANCE_WORK_DIR}/pr_gate_plan.json" /tmp/gh-aw/pr_gate_plan.json

engine:
  id: claude
  args:
    - "--plugin-dir"
    - "${{ runner.temp }}/gh-aw/marketplace/crosscheck"

safe-outputs:
  mentions: false
  add-comment:
    max: 2
  create-issue:
    title-prefix: "[Assurance PR-Gate] "
    labels: [assurance, "pr-gate"]
    max: 1   # only the kill-criterion alert path opens an issue
  push-to-pull-request-branch:
    # Allow the agent to push attestation JSON back to the PR branch so
    # the cache warms over time. Defence-in-depth: protected-files set to
    # fallback-to-issue so any accidental write to a sensitive file
    # (workflows, ROADMAP, invariant docs themselves, settings) escalates
    # to human review rather than landing on the branch.
    target: "triggering"
    title-prefix: "[Assurance Squad] "
    protected-files: fallback-to-issue
    max: 1
---

# Assurance PR-Gate

You are the mandatory Layer 5 gate for protected-surface diffs. Three jobs:

1. **Intent-check** (Layer 5, probabilistic): for each *changed* invariant,
   run `/crosscheck:intent-check` — but only if the content-hash isn't
   already in the attestation cache. Push the new attestation JSON back to
   the PR branch so the cache warms over time. Post a clearly-labelled
   comment per invariant.
2. **Governance amendment reminder**: if the PR touches a Class A/B
   protected file without a `## Governance Amendment` block in the PR body,
   post a comment with the `/crosscheck:protected-surface-amend` template.
3. **Dafny hand-off**: if any changed invariant doc sets
   `dafny_candidate: true`, post a comment recommending byfuglien.

You do **not** draw stochastically. Every applicable check runs. The only
gating is the cache short-circuit.

**Layer 4 coverage gate is NOT your job.** That's owned by
`.github/workflows/assurance.yml` (static, runs on every push/PR). If
coverage is failing, the gate workflow has already surfaced it.

The workflow has already run a deterministic Python pre-step that computed
a per-invariant plan. **Read `/tmp/gh-aw/pr_gate_plan.json`** for the plan
— it contains `invariants[]` (with `action: use_cached | run_intent_check`),
`kill_active`, `amendment_reminder_needed`, `dafny_handoff_needed`, and
`protected_files_touched`.

---

## Phase 0 — Read context

```bash
cat /tmp/gh-aw/pr_gate_plan.json
gh pr view "${PR_NUMBER:-}" --json body,headRefName,headRefOid 2>/dev/null || true
```

Read the PR body. Note any existing `## Governance Amendment` block.

Also search for the existing sticky comment so it can be edited in place later.
Guard against an unset `PR_NUMBER` (possible on `reopened`/`ready_for_review`
events depending on event resolution):

```bash
EXISTING_COMMENT_ID=""
if [ -n "${PR_NUMBER:-}" ]; then
  EXISTING_COMMENT_ID=$(
    gh api --paginate \
      "repos/$GITHUB_REPOSITORY/issues/$PR_NUMBER/comments" \
      --jq ".[] | select(.body | contains(\"<!-- assurance-pr-gate:$PR_NUMBER -->\")) | .id" \
    2>/dev/null | head -n1
  )
fi
```

`--paginate` handles PRs with >100 comments. `head -n1` takes the first match
(oldest), which is the canonical sticky. `EXISTING_COMMENT_ID` is empty string
if not found or if `PR_NUMBER` is unset — used in Phase 4.

---

## Phase 1 — Intent-check (Layer 5, probabilistic)

For each plan entry where `action == "run_intent_check"`, invoke
`/crosscheck:intent-check` with:
- The invariant prose (slice of the doc).
- The covering test sources (paths in `covering_tests`).
- The module source diff (`git diff origin/<base>...HEAD -- <module-paths>`).

The skill will append a row to `.assurance/fp-tracker.csv` and emit a JSON
attestation. **Save the attestation to**
`docs/assurance/attestations/<content_hash>.json` (use the `content_hash`
from the plan).

**Push the attestation back to the PR branch** so the cache warms over
time. The safe-output `push-to-pull-request-branch` is configured to allow
exactly `docs/assurance/attestations/*.json` and nothing else.

**Accumulate the per-invariant result blocks** for the sticky comment
(Phase 4). Do **not** post individual comments — all content goes into the
single sticky.

For each `run_intent_check` entry, format the result block as:

> **Layer 5 — Intent-check (probabilistic)**
>
> Invariant: `<ID>` (`<name>`) in `docs/invariants/<module>.md`
> Verdict: **<PASS|FLAG|UNCERTAIN>**
> Attestation: `sha256:<content_hash>` (`docs/assurance/attestations/<hash>.json`)
> FP-tracker rolling 30 d: <rate>% (n=<count>)
>
> <one-paragraph summary of the back-translation diff>
>
> _Layer 5 is probabilistic. Verdict is non-binding; reviewers may
> accept, reject, or defer with rationale. Force a fresh re-run with
> `/assurance-recheck <ID>`._

For each `use_cached` entry, format as:

> **Layer 5 — Intent-check (cached verdict)**
>
> Invariant: `<ID>` (`<name>`)
> Verdict: **<from cached.verdict>** _(originally checked
> <cached.checked_at>)_
> Attestation: `sha256:<content_hash>` _(unchanged inputs)_
>
> Inputs hash matches the prior attestation; no LLM call. To force a
> re-check, push a whitespace change or comment
> `/assurance-recheck <ID>`.

If `kill_active == true`, prepend every invariant block with:

> **Layer 5 degraded** — kill criterion active (FP rate ≥ 30 %).
> Cache is bypassed; this is a fresh run. Verdicts should be treated
> as advisory until humans clear the criterion via
> `.assurance/kill-criterion.json`.

If `module_source_resolvable == false` for an invariant, note in the
block that the cache key is best-effort (module source could not be
located) and treat the verdict as fresh.

---

## Phase 2 — Governance amendment reminder

**Accumulate the amendment block** for the sticky comment (Phase 4).
Do **not** post a separate comment.

If `plan.amendment_reminder_needed == true`, the block content is:

> **Governance amendment required**
>
> This PR touches protected-surface files but does not include a
> `## Governance Amendment` block in the PR body. Protected surfaces
> partition into Class A (harness/workflow definitions) and Class B
> (module invariants & tests). Every amendment must declare which class
> it belongs to.
>
> Files touched: `<list from protected_files_touched>`
>
> Add this block to the PR body (use
> `/crosscheck:protected-surface-amend` to generate it):
>
> ```markdown
> ## Governance Amendment
>
> **Class:** A (Harness/workflow) | B (Module invariants & tests)
> **Change:** <one-sentence description>
> **Rationale:** <why this change is necessary>
> **Authority:** <ROADMAP item, prior attestation, or human decision>
> **Diff plan:** <bullet list of files and intent>
> **Test/coverage impact:** <none | new tests added | existing updated>
> **Review checklist:**
> - [ ] Class declared
> - [ ] Authority cited (not "agent judgement")
> - [ ] Diff plan matches actual diff
> ```

If `plan.amendment_reminder_needed == false`, use "No amendment required."
as the block content.

---

## Phase 3 — Dafny hand-off announcement

**Accumulate the Dafny block** for the sticky comment (Phase 4).
Do **not** post a separate comment.

If `plan.dafny_handoff_needed == true`, the block content is:

> **Dafny candidate detected — hand-off recommended**
>
> One or more invariant docs in this PR set `dafny_candidate: true`.
> This module fits byfuglien's spec chain better than hellebuyck's.
> After merging the invariant draft, run:
>
> 1. `/crosscheck:spec-iterate` — draft the formal Dafny spec.
> 2. `/crosscheck:generate-verified` — generate a verified
>    implementation.
> 3. `/crosscheck:extract-code` — compile the verified Dafny to
>    Python or Go.
>
> Hellebuyck's Layer 5/6 output is best-effort; Dafny's Layer 1–3
> verification is deterministic. Use the right tool for the layer.

If `plan.dafny_handoff_needed == false`, use "No Dafny candidates in this
PR." as the block content.

---

## Phase 4 — Assemble and post the sticky comment

Compute the run metadata:

```bash
RUN_TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%MZ")
RUN_URL="${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"
HEAD_SHA_SHORT=$(git rev-parse --short HEAD)
```

Assemble the full sticky comment body in this exact structure. The summary
lines (Layer 5 counts, amendment, Dafny, kill criterion) appear immediately
after the `<!-- assurance-pr-gate:${PR_NUMBER} -->` marker and before the
collapsible detail sections — they are always visible without expanding any
section.

```
<!-- assurance-pr-gate:${PR_NUMBER} -->
**Assurance PR-Gate summary**
Layer 5 intent-check: <N fresh, M cached, K skipped>
Attestations pushed: <N>
Amendment reminder: <yes|no>
Dafny hand-off: <yes|no>
Kill criterion: <inactive|active>

<details>
<summary>Intent-check results (click to expand)</summary>

<accumulated Phase 1 blocks>

</details>

<details>
<summary>Governance amendment reminder (click to expand)</summary>

<Phase 2 block>

</details>

<details>
<summary>Dafny hand-off (click to expand)</summary>

<Phase 3 block>

</details>

---
_Last updated: ${RUN_TIMESTAMP} · run [#${GITHUB_RUN_NUMBER}](${RUN_URL}) · head `${HEAD_SHA_SHORT}`_
```

All three `<details>` sections are always rendered. Use the "none" placeholder
text from the corresponding phase if a section has no content.

Post or update the sticky comment using the PATCH / `add_comment` fallback
pattern. **Never use a bare `|| fallback` construct — always capture
`PATCH_EXIT_CODE` explicitly** (bare `||` swallows the exit code, which is
a silent-failure analogue and violates Operating Principle 5):

```bash
PATCH_EXIT_CODE=0
if [ -n "${EXISTING_COMMENT_ID:-}" ]; then
  gh api --method PATCH \
    "repos/$GITHUB_REPOSITORY/issues/comments/$EXISTING_COMMENT_ID" \
    --field body="<assembled body>" || PATCH_EXIT_CODE=$?
  if [ $PATCH_EXIT_CODE -ne 0 ]; then
    echo "WARNING: Sticky-comment PATCH failed (id=$EXISTING_COMMENT_ID, exit=$PATCH_EXIT_CODE); falling back to add_comment"
  fi
fi

if [ -z "${EXISTING_COMMENT_ID:-}" ] || [ $PATCH_EXIT_CODE -ne 0 ]; then
  add_comment <assembled body>
fi
```

Rules:
- Log the comment id and exit code verbatim when `PATCH_EXIT_CODE` is non-zero
  (Operating Principle 5 — no silent failures).
- The fallback **must** use the `add_comment` safe-output — not a raw
  `gh api POST` — so the threat-detection pipeline tracks it.
- Never post a second sticky comment when a PATCH would suffice
  (Operating Principle 6).

---

## Operating principles

1. **Mandatory, not stochastic.** Every applicable check runs.
2. **Cache is load-bearing.** Content-hashed attestations as cache keys
   — labelled cached when reused. Bypass on kill-criterion or
   `/assurance-recheck`.
3. **Layered honesty.** Every comment declares its layer.
4. **Gate is advisory at Layer 5.** Verdicts inform reviewers; they do
   not auto-fail the build. Layer 4 (coverage) auto-fails — but that's
   `assurance.yml`'s job.
5. **No silent failures.** If you cannot compute a content hash (module
   source missing, parse error), conservatively run the LLM check and
   log why caching was skipped.
6. **One sticky comment per PR.** On every run, search for the prior
   sticky comment (marker: `<!-- assurance-pr-gate:{PR_NUMBER} -->`). If
   found, PATCH it: capture `PATCH_EXIT_CODE` explicitly, log id and exit
   code if non-zero, then fall back to `add_comment` (safe-output). If no
   prior comment exists (or `PR_NUMBER` is unset), use `add_comment`
   (safe-output). Never post a second comment when an edit would suffice;
   never silently discard a PATCH error.
7. **Push attestations narrowly.** Only `docs/assurance/attestations/*.json`
   files. Never anything else.

Generated by Assurance PR-Gate. Hellebuyck owns Layers 4–6.
