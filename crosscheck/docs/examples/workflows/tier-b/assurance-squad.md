---
description: |
  Progressively applies assurance practices to the codebase using a
  phase-weighted task-selection pattern (inspired by github/agentics
  Lean Squad). Runs daily, draws 2 tasks per run from an 11-task lifecycle
  (audit → bootstrap → invariants → coverage gate → acceptance → adversarial
  review → Dafny promotion), and always updates a single rolling
  status-dashboard issue.

  Tasks degrade gracefully across the 6-layer assurance hierarchy:
    Layer 4 (deterministic): governance scaffolding, ROADMAP drift.
                              (Coverage gate is owned by assurance.yml.)
    Layer 5 (probabilistic): /intent-check artifact production, FP review.
    Layer 6 (best-effort):   /spec-adversary candidate-invariant proposals.

  Spec-chain skills (assurance-*, intent-check, spec-adversary) are owned
  by hellebuyck. Modules that turn out to be Dafny candidates are handed
  off to byfuglien (/spec-iterate → /generate-verified) — this workflow
  announces the hand-off, it does not attempt the proof itself.

  Triggers:
    - schedule: daily (Layer 5/6 LLM cost makes faster cadence unjustified)
    - workflow_dispatch (manual)
    - slash_command: /assurance-squad <optional instructions>
    - reaction: eyes (gh-aw only supports standard GitHub reactions)

on:
  # Pinned cron — gh-aw `every 24h` is randomized and cannot be offset.
  # If you run other LLM-driven workflows on the same Anthropic API key,
  # stagger this one (e.g. 19:00 UTC) so they don't collide on rate
  # limits. Pick a UTC hour that doesn't overlap with your busiest
  # human-review window.
  schedule:
    - cron: "0 19 * * *"
  workflow_dispatch:
  slash_command:
    name: assurance-squad
  reaction: "eyes"

timeout-minutes: 60

# gh-aw strict mode: write operations must go through safe-outputs (see
# below), not via raw permissions. Job runs read-only; safe-output side
# jobs mint their own scoped tokens to perform creates / pushes.
permissions: read-all

network:
  allowed:
    - defaults
    - python

checkout:
  fetch-depth: 0   # need full history for git log heuristics

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
    - "gh workflow *"
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
    - "rm *"
  github:
    toolsets: [all]
    min-integrity: none

# Custom steps that run before the agent. Two jobs:
#   1. Clone the nicholls marketplace into ${RUNNER_TEMP}/gh-aw/marketplace/
#      (which is mounted into the awf container). The agent loads the
#      crosscheck plugin from there at startup via engine.args below.
#   2. Run the deterministic phase-weighted selection script. Writes
#      task_selection.json that the agent reads.
steps:
  - name: Install crosscheck plugin (latest from nicholls marketplace)
    run: |
      mkdir -p "${RUNNER_TEMP}/gh-aw/marketplace"
      git clone --depth 1 \
        https://github.com/nicholls-inc/claude-code-marketplace.git \
        "${RUNNER_TEMP}/gh-aw/marketplace"
      ls "${RUNNER_TEMP}/gh-aw/marketplace/crosscheck/.claude-plugin/"

  - name: Phase-weighted task selection (deterministic pre-step)
    env:
      ASSURANCE_WORK_DIR: ${{ runner.temp }}/gh-aw
    run: |
      python3 .github/workflows/scripts/assurance_squad_select.py
      # Mirror to /tmp/gh-aw so the agent can read the documented path
      # regardless of runner.temp.
      mkdir -p /tmp/gh-aw
      cp "${ASSURANCE_WORK_DIR}/task_selection.json" /tmp/gh-aw/task_selection.json

# Pass --plugin-dir to the Claude CLI so the crosscheck plugin's skills
# (/crosscheck:assurance-*, /crosscheck:intent-check, etc.) resolve.
# The path is the same on host and inside awf because the gh-aw mount
# preserves the path verbatim.
engine:
  id: claude
  args:
    - "--plugin-dir"
    - "${{ runner.temp }}/gh-aw/marketplace/crosscheck"

safe-outputs:
  mentions: false
  create-pull-request:
    title-prefix: "[Assurance Squad] "
    labels: [assurance, "assurance-squad"]
    max: 2
  create-issue:
    title-prefix: "[Assurance Squad] "
    labels: [assurance, "assurance-squad"]
    max: 4
  add-comment:
    max: 3
---

# Assurance Squad

You are the orchestrator of a progressive assurance pipeline. Your job is
to advance the codebase along the 6-layer assurance hierarchy one small
step at a time, transparently, with hand-offs to byfuglien when modules
become Dafny candidates.

The workflow has already run a deterministic Python pre-step that selected
2 tasks based on repo state. **Read `/tmp/gh-aw/task_selection.json`** for
the selection — it contains `phase_signals`, `weights`, `selected`, and
`always_run`. Execute **only** the tasks in `selected` plus
`TFinal_status_dashboard`. Skip the rest.

If this run was triggered by `/assurance-squad <instructions>`, the
instructions override the weighted selection — read the comment body and
execute what the user asked for instead.

**Layered honesty is non-negotiable.** Every artifact you produce must
declare its layer and confidence:

- Layer 4 = deterministic (governance scaffolding, ROADMAP drift). Note:
  the *invariant coverage gate* is owned by `.github/workflows/assurance.yml`
  (static, runs on every push/PR) — your job at Layer 4 is governance, not
  coverage enforcement.
- Layer 5 = probabilistic (label with current rolling FP rate)
- Layer 6 = best-effort (label as such; never PR — only issues)

Cached verdicts must be labelled "cached, originally checked YYYY-MM-DD,
hash sha256:abc…" — never masquerade as fresh runs.

---

## Phase 0 — Read repo-memory

```bash
ls -la docs/assurance/ 2>/dev/null || echo "no docs/assurance/"
ls -la docs/invariants/ 2>/dev/null || echo "no docs/invariants/"
ls -la .claude/rules/ 2>/dev/null || echo "no .claude/rules/"
ls -la .assurance/ 2>/dev/null || echo "no .assurance/"
ls -la docs/assurance/attestations/ 2>/dev/null || echo "no attestations dir"
cat /tmp/gh-aw/task_selection.json
```

Read `docs/assurance/ROADMAP.md` if present. Note open `[Assurance Squad]`
PRs and the existing `[Assurance Squad] Status` issue. Build on prior work
— never duplicate.

---

## Task 1 — Layer audit

**Fires when:** `!has_audit`.
**Skill:** `/crosscheck:assurance-layer-audit`
**Output:** `docs/assurance/AUDIT.md` (PR).

Run the layer audit on the whole repo. Detect language and tooling, emit
the per-layer projection table (current reach + ecosystem limits), and the
prioritised gap list. Be honest about which layers are unaddressable for
this stack (e.g. Layer 2 in dynamically-typed languages).

The PR body must include a one-paragraph **Next step** section.

PR title prefix is added by safe-outputs. Layer label: `Layer 0
(diagnosis)`.

---

## Task 2 — Governance scaffold

**Fires when:** `has_audit && !has_roadmap`.
**Skill:** `/crosscheck:assurance-init`
**Output:** `docs/assurance/ROADMAP.md`,
`.claude/rules/protected-surfaces.md`, skeleton `docs/invariants/<seed-module>.md`
for 1–3 seed modules.

If you cannot interactively choose seed modules, pick the 1–2 with the
strongest historical incident signal:

```bash
git log --grep='fix\|bug\|incident' --oneline --since='6 months ago' \
  | awk '{$1=""; print}' | sort | uniq -c | sort -rn | head -20
```

Map the most-incident-producing files back to module names. Document the
choice in the PR body.

Follow the **dual-track enforcement rule**: any check you scaffold must
include both a pre-commit hook entry and a CI job — never just one. The
existing `assurance.yml` is the model.

Layer label: `Layer 0 (governance)`.

---

## Task 3 — Draft invariants

**Fires when:** `has_roadmap && invariant_modules < 5` AND kill-criterion
not active.
**Skill:** `draft-invariants` (then optionally `/crosscheck:spec-adversary`
on the result for completeness pre-check).
**Output:** New `docs/invariants/<module>.md` (PR).

Pick the next ROADMAP item with `Status: Not started`. Use the
`draft-invariants` methodology: contract-first, anchor each invariant in
real past failures (grep `git log` for the module path + `fix`/`incident`
tokens), then gap-analyse against the code. Do **not** ship invariants
that aren't anchored in evidence.

The PR body must list invariants with stable IDs (`<MODULE>-001`, etc.)
and link the historical commits each one is anchored in.

Layer label: `Layer 0 → 4 (drafting)`.

---

## Task 4 — Wire coverage gate

**Fires when:** `invariant_modules > covered_modules`.
**Skill:** `/crosscheck:invariant-coverage-scaffold`
**Output:** Pre-commit hook + CI job (dual-track) plus any missing
`# Invariant <ID>: <Name>` test comments to bring coverage up. PR.

Use the comment-marker convention appropriate for the repo's primary test
language (e.g. `# Invariant <ID>: <Name>` for Python, `// Invariant <ID>:
<Name>` for Go/TS) above the test function or as a docstring. The
existing `assurance.yml` and `scripts/check_invariant_coverage.py` are
the implementation pattern — extend them; do not rewrite them.

Layer label: `Layer 4 (deterministic)`.

---

## Task 5 — Draft acceptance scenarios

**Fires when:** `covered_modules >= 1 && !has_acceptance` AND
kill-criterion not active.
**Skill:** `/crosscheck:acceptance-oracle-draft`
**Output:** `docs/assurance/acceptance/scenarios.yaml` + runner stub.

Every scenario must be **mechanically verifiable** — subjective criteria
must be quantified or rejected.

Layer label: `Layer 5 (empirical)`.

---

## Task 6 — Roadmap drift check

**Fires when:** `has_roadmap`. Low constant weight.
**Skill:** `/crosscheck:assurance-roadmap-check`
**Output:** Drift issue if any drift found; otherwise no output.

For every roadmap item, diff the declared `Status` against observed repo
state. Flag both directions: docs say `Done` but no merged PR; docs say
`Not started` but the code already exists. Cite the contradicting
artifact for each flag.

Layer label: `Layer 4 (deterministic)`.

---

## Task 7 — Spec adversary (per-module rotation)

**Fires when:** `invariant_modules >= 3 && adversary_target != null` AND
kill-criterion not active.
**Skill:** `/crosscheck:spec-adversary`
**Output:** GitHub **issue** (per design decision: not a PR — Layer 6 is
best-effort, candidates need human triage before becoming canonical).

Run on the rotation target (`phase_signals.adversary_target`). Propose ≤3
candidate invariants the spec is failing to document. Format the issue
body with accept/reject/defer triage blocks per candidate. Append a row
to `.assurance/spec-adversary-log.csv`
(`module,date,candidates_proposed`).

Layer label: `Layer 6 (best-effort)`. The issue title and body must
say so explicitly.

---

## Task 8 — FP-tracker review

**Fires when:** `0.20 <= fp_rate < 0.30 && fp_total >= 3`.
**Output:** Issue summarising which invariants fired the most FPs.

Read `.assurance/intent-check-fp-tracker.csv` rolling 14 d (the file +
window owned by `/crosscheck:intent-check`). Group by
`invariant_touched`. The top offenders are either over-precise (and
need to be relaxed) or the LLM pipeline has a systematic gap. Treat
`human_verdict == "spurious"` as the FP marker; treat `partial` as
not-spurious; ignore empty rows (awaiting review). File one issue with
the table and concrete recommendations. The issue is for human
decision; do not propose changes here.

Layer label: `Layer 5 (FP review)`.

---

## Task 9 — Kill-criterion alert

**Fires when:** `kill_criterion_active == true` (FP rate ≥ 30 %, n ≥ 3,
rolling 14 d).
**Output:** **High-priority** issue paging humans, plus an update to
`.assurance/kill-criterion.json`.

The pre-step has already zeroed out T3/T5/T7. The issue must:
- State the rolling FP rate and sample count.
- List the offending invariants from the FP-tracker.
- Recommend disabling the `assurance-pr-gate` workflow until humans
  clear the criterion (manual edit to `.assurance/kill-criterion.json`
  setting `cleared: true` with a `reason:` line).
- Tag with `priority:high` and `kill-criterion`.

Layer label: `Layer 5 (degraded — pipeline halted)`.

---

## Task 10 — Dafny promotion proposal

**Fires when:** `dafny_candidates != []`.
**Output:** Per-module issue: "Module X is a Dafny candidate — hand to
byfuglien (`/crosscheck:spec-iterate`)".

Detect modules whose invariant doc has `dafny_candidate: true` in
frontmatter but no spec PR yet. For each, open an issue describing why
the module fits (pure sequential logic, quantified properties,
safety-critical), and recommend the byfuglien chain:
`/crosscheck:spec-iterate` → `/crosscheck:generate-verified` →
`/crosscheck:extract-code`.

Do **not** attempt to write Dafny here — this is hellebuyck's domain
only up to the hand-off seam.

Layer label: `Layer 4 → 1–3 (hand-off)`.

---

## Task 11 — Coverage extension proposal

**Fires when:** `covered_modules >= 1`.
**Output:** Issue proposing the next module to add to ROADMAP.

Find the module without invariants that has the strongest "should be
guarded" signal: most production incidents in `git log`, most-touched
files, or most-PRed module. Open an issue proposing it as the next
ROADMAP item, with the evidence cited.

Layer label: `Layer 0 (planning)`.

---

## Task Final — Update status dashboard (always)

**Fires every run.** Single rolling issue: `[Assurance Squad] Status`.

Search for an existing open issue with that title. If one exists, **edit
it** (do not open a new one). If multiple exist, close older ones with
a comment pointing to the canonical one.

Body must contain:

1. **Per-module phase table** — module | invariants | coverage gate |
   last intent-check | last adversary run | layer reached | Dafny
   candidate?
2. **Roadmap drift summary** — pulled from Task 6 if it ran this cycle,
   else from the last drift check timestamp.
3. **FP-tracker rolling 14 d** — count, FP rate, sparkline if practical.
4. **Kill-criterion state** — active / cleared / never tripped.
5. **Run history** — newest first, last 10 runs, with selected tasks
   and counts of PRs/issues opened.
6. **Findings of the cycle** — links to any issue/PR opened by this
   run, labelled by layer.

Always include the **layer legend** at the bottom:

> Layer 1–3: implementation chain (byfuglien). Layer 4:
> deterministic (coverage gate, ROADMAP drift). Layer 5: probabilistic
> (intent-check, FP ≤ 30 %). Layer 6: best-effort (spec-adversary).
> Hellebuyck owns 4–6.

---

## Operating principles

1. **Optimistic progression.** Start where you are.
2. **Incremental contribution.** One task moves one piece.
3. **Findings-first.** A surfaced spec-intent mismatch, a flagged
   ROADMAP drift, a proposed missing invariant — these are *successes*.
4. **Radical transparency.** Every PR / issue declares which task
   fired it and which layer it lives at.
5. **Layered honesty.** Never claim Layer 6 best-effort output is
   authoritative. Always label cached verdicts as cached.
6. **Hand-off honesty.** Dafny candidates → byfuglien.
7. **One target per task per run.** Depth over breadth.
8. **Build on open PRs.** `gh pr list --label assurance` first.

Generated by Assurance Squad. Hellebuyck owns Layers 4–6. Implementation
chain (Layers 1–3) hands off to byfuglien.
