# Plan: Adopt the AI-native SDLC playbook as Crosscheck's development framework + self-explanatory human gates

## Context

The uploaded task brief asks for two work packages in this repo (Crosscheck lives in `crosscheck/`):

- **Package A** — adopt Anthropic's AI-Native SDLC playbook (artefact chain: intent.md → spec.md → plan.md → diff+tests → PR review → incident record; controls: skills=advisory, hooks=deterministic, human approval at defined gates) as the framework for Crosscheck's own development, wired to Crosscheck's assurance skills.
- **Package B** — inventory every human gate in Crosscheck and give each one (a) a plain-language explainer in `docs/gates/` and (b) a succinct three-sentence message in the exact prescribed format at its point of delivery.

**Key reality check from exploration** (this shapes the whole plan): the brief cites `docs/assurance/`, repo-root `docs/invariants/` and `.claude/rules/protected-surfaces.md` as existing — **none exist in this repo**. They are what `/assurance-init` scaffolds into *target* repos; Crosscheck has never been dogfooded on itself. So Package A is partly a dogfooding exercise: lay down that governance layout at the repo root for Crosscheck's own development, matching the templates `assurance-init` already defines (`crosscheck/skills/assurance-init/SKILL.md`).

Also: the user asked me to orchestrate implementation with **Workflow** fan-outs using opus/sonnet/haiku, conserving my own context for orchestration. Constraints from the brief: additive only (no gate weakened, no semantics changed except A's new enforcement), each new doc under two pages, **British spelling**, and this plan itself becomes the committed `plan.md` artefact.

Branch: `claude/multi-model-orchestration-w7qhni` (clean, even with main). Note: the existing repo-root `plan.md` is a committed artefact of a *previous* change (assurance-probe design). Per the framework being adopted, `plan.md` reflects the current change; the new plan supersedes it (old one stays in git history). No PR will be created unless asked.

## Package A — playbook as primary development framework

### A1. Artefact chain (docs + templates)
- `intent/README.md` + `intent/TEMPLATE.md` — template with the playbook's fields: problem statement, proposed outcome, affected users and systems, constraints, open questions.
- `docs/assurance/DEVELOPMENT-FRAMEWORK.md` — the full flow: which artefact each stage commits (intent.md → spec.md → plan.md → diff+tests → PR+review findings → incident record/eval), which commit triggers the next stage, and where each Crosscheck skill sits (`/informal-spec`, `/draft-invariants`, `/audit-spec-coverage`, `/audit-invariant-consistency`, `/intent-check`, `/assurance-probe`, `/protected-surface-amend`, orchestrator agents). Under two pages.
- Repo-root `plan.md` — replaced with the approved plan for this change (first dogfooded artefact).
- `CLAUDE.md` — short addition pointing at the framework doc and the artefact convention.

### A2. CI wiring (`.github/workflows/`)
- `spec-audit.yml` — PR touching `**/spec.md` or `formal-verification/specs/**` → non-interactive `/audit-spec-coverage` + `/audit-invariant-consistency` via `anthropics/claude-code-action@v1`, reusing the skills' existing **orchestrator marker mode** (documented in `crosscheck/docs/orchestrator-coordination.md`) as the non-interactive path — no skill degraded; findings posted as a PR comment in the Package B message format. Requires `ANTHROPIC_API_KEY` secret; the workflow states this and fails with a clear message if absent.
- `protected-surface-check.yml` — PR touching any protected path → non-interactive `/intent-check`; attestation JSON published as a check run (via `actions/github-script` check-run creation) with a Package-B-format summary.
- `incident-eval-check.yml` — deterministic script (no model): on merged PRs whose body/commits reference an incident (`Fixes-Incident:` trailer or `incident` label), verify a matching eval exists under `evals/` and a candidate invariant under `docs/invariants/`; open/annotate a failing check otherwise.
- `tier-gate.yml` — deterministic script implementing A3 below.

### A3. Tier-to-layer map
- `docs/assurance/TIER-LAYER-MAP.md` — three tiers:
  - **Tier 1 (routine)** — docs, tests, non-behavioural code: requires `intent.md` reference only.
  - **Tier 2 (standard)** — behavioural code, skills' non-gate sections: requires committed `spec.md` (+ intent).
  - **Tier 3 (critical/protected)** — protected surfaces, gate logic, hooks, CI enforcement, invariants: requires `plan.md`, an intent-check attestation, and a governance-note block where a protected surface changes.
- `tier-gate.yml` reads the PR's declared tier (a `Tier: N` line in the PR body, or `tier:N` label), infers a floor from the diff (any protected path ⇒ at least Tier 3), and verifies the required artefacts exist on the branch. Failure message uses the Package B format. This document is Crosscheck's definition of the playbook's "regulated and critical code".

### A4. Extend `assurance-init`
Extend `crosscheck/skills/assurance-init/SKILL.md` (not a new sibling skill — its existing pre-flight collision table and ≤3-question flow absorb the additions without becoming unwieldy) to scaffold, in the same pass: `intent/` + template, `CLAUDE.md` additions, `REVIEW.md`, `.claude/settings.json` PreToolUse hook wiring + hook script, `evals/` + CI workflow stubs, and target-repo copies of DEVELOPMENT-FRAMEWORK.md and TIER-LAYER-MAP.md. Behavioural artefact ⇒ `feat(crosscheck)` commit.

### A5. Deterministic protected-surface hook
- `.claude/rules/protected-surfaces.md` — created at repo root using assurance-init's own template, with a machine-readable path list protecting: `crosscheck/skills/*/SKILL.md`, `crosscheck/agents/*.md`, `docs/invariants/**`, `crosscheck/docs/invariants/**`, `docs/assurance/**`, `.claude/rules/**`, `.claude/hooks/**`, `evals/**`.
- `.claude/hooks/protected-surface-guard.mjs` (Node, matching the repo's toolchain) + `.claude/settings.json` PreToolUse hook on `Edit|Write|NotebookEdit`: reads the tool-input JSON on stdin, matches the target path against the protected list, and blocks (exit 2) unless the working tree contains a `## Protected-Surface Amendment` governance-note block (under `.assurance/protected-surface-amend/` or `.assurance/add-session-*/`, as generated by `/protected-surface-amend`) naming that file. Block message on stderr in the exact Package B format; repo URL derived from `git remote get-url origin` at runtime, linking to the explainer path on the default branch.

### A6. `REVIEW.md`
Repo-root `REVIEW.md`: four passes — bugs/logic, security, compliance against `spec.md` + `plan.md`, and the Crosscheck pass (does the diff touch a protected surface; does the declared tier match the diff). Important vs Nit definitions, "at most five nits per review; summarise the rest as a count", excluded paths (`crosscheck/mcp-server/dist/**`, lockfiles, generated changelogs, `CHANGELOG.md`).

## Package B — self-explanatory gates

### Gate inventory (from exploration; full detail with file:line already gathered)
`docs/gates/README.md` lists all of these; one explainer file each under `docs/gates/`:

| # | Gate (explainer file) | Delivery point to update |
|---|---|---|
| 1 | `informal-spec-sign-off.md` | `informal-spec/SKILL.md` sign-off prompt |
| 2 | `draft-invariants-red-pen.md` | `draft-invariants/SKILL.md` §6 red-pen prompt |
| 3 | `intent-check-kill-criterion.md` (30% FP trip) | `intent-check/SKILL.md` Step 0 refusal message |
| 4 | `intent-check-verdict.md` (fail remediation) | `intent-check/SKILL.md` verdict summary |
| 5 | `protected-surface-amendment.md` (PR-review verdict incl. REQUIRES HUMAN VERIFICATION markers) | `protected-surface-amend/SKILL.md` Step 7 PR-description template |
| 6 | `protected-surface-roadmap-refusal.md` (no-roadmap-item hard refusal) | `protected-surface-amend/SKILL.md` refusal message |
| 7 | `assurance-probe-triage.md` (accept/reject/defer) | `assurance-probe/SKILL.md` GitHub-issue template |
| 8 | `audit-spec-coverage-triage.md` (4-path) | `audit-spec-coverage/SKILL.md` findings-file header |
| 9 | `audit-invariant-consistency-triage.md` (4-path) | `audit-invariant-consistency/SKILL.md` findings-file header |
| 10 | `spec-adversary-triage.md` | `spec-adversary/SKILL.md` findings block |
| 11 | `assurance-init-prompts.md` (collision skip/overwrite/abort + Q1–Q3) | `assurance-init/SKILL.md` prompts |
| 12 | `lowry-drift-packet.md` | `agents/lowry.md` drift-packet commit/report |
| 13 | `orchestrator-batch-sign-off.md` | `agents/add-orchestrator.md` steps 4/9 prompts |
| 14 | `auditor-verdicts.md` | `agents/auditor.md` report template |
| 15 | `protected-surface-hook.md` (new, A5) | hook stderr message |
| 16 | `tier-layer-gate.md` (new, A3) | `tier-gate.yml` failure output |

Each explainer (written for a Crosscheck newcomer, < 2 pages, defines terms like oracle independence / protected surface / false-positive tracker on first use): what the gate protects and why, what you're deciding, what each option means and what happens next, typical decision time, who to ask.

### Message format (verbatim shape, added at every delivery point)
> **Action needed: [imperative, under ten words]**
> You are being asked to [decision] because [reason]. Approving means [consequence]; declining means [consequence]. Full explanation: [link].

Three sentences plus link; links point to `docs/gates/<file>.md` on the default branch, repo URL derived from the git remote (skills instruct the agent to run `git remote get-url origin`; the hook and CI scripts do it in code). SKILL.md edits change *presentation templates only* — no verification logic, thresholds or kill criteria touched.

### Self-referential governance
After A5 lands, `SKILL.md`/`agents/*.md` become protected surfaces — so the Package B edits to them follow `/protected-surface-amend`'s own process: generate the governance-note block into `.assurance/protected-surface-amend/` in the same commits (which also keeps the new hook from blocking the work), for later inclusion in any PR description.

## Orchestration (per user instruction: Workflow with opus/sonnet/haiku, conserve orchestrator context)

- **Stage 1 (Workflow "package-a-core")** — parallel authoring: **opus** agents for the judgment-heavy prose (DEVELOPMENT-FRAMEWORK.md, TIER-LAYER-MAP.md, REVIEW.md, protected-surfaces.md, assurance-init extension); **sonnet** agents for code (hook script + settings.json, four CI workflows, intent template + CLAUDE.md addition). Each agent returns file contents; I write files and run local verification.
- **Stage 2 (Workflow "package-b-gates")** — pipeline over the 16 gates: **sonnet** writes each explainer from the inventory's file:line detail (haiku is tempting for these but they need careful plain-language accuracy; **haiku** is used for the mechanical link/format/British-spelling lint pass instead), then **sonnet** edits each emitting SKILL.md/agent template (surgical, presentation-only), then a **haiku** format-compliance check per gate (exact message shape, ≤3 sentences, link path valid) and an **opus** adversarial constraint check across the whole diff (no gate weakened, no semantics changed, page limits).
- **Stage 3** — I verify locally (hook simulation via `echo '<json>' | node .claude/hooks/protected-surface-guard.mjs`, YAML parse of workflows, commitlint dry-run), fix residuals, and commit in conventional-commit slices: `feat(crosscheck)` for behavioural artefacts, `ci:`/`docs:`/`chore:` for the rest; then push with `git push -u origin claude/multi-model-orchestration-w7qhni`.

## Verification

- **Hook**: simulate PreToolUse JSON for (a) a protected path without a governance block → exit 2 with correctly-formatted message and working link path; (b) same path with block present → allowed; (c) unprotected path → allowed.
- **CI**: `node -e` YAML parse of each workflow; run the deterministic tier-gate and incident-eval scripts locally against fixture inputs (missing artefact → fail with Package B message; present → pass).
- **Messages**: haiku lint pass asserts every gate message matches the exact shape and its link target exists in `docs/gates/`.
- **Acceptance walkthrough**: trace a hypothetical change intent→PR using only DEVELOPMENT-FRAMEWORK.md; confirm `docs/gates/README.md` covers all 16 gates with explainer + updated delivery point; confirm each doc ≤ 2 pages and British spelling (haiku pass).
- Existing checks: `npx commitlint` on commit messages; repo tests untouched (no mcp-server code changes) but `npm test` in `crosscheck/mcp-server` run once as a regression guard.
