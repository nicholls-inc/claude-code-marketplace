---
name: assurance-init
description: >-
  Interactive bootstrap that scaffolds the governance skeletons a repo needs to
  adopt the 6-layer assurance hierarchy. Creates docs/assurance/ROADMAP.md + horizon
  directories, .claude/rules/protected-surfaces.md (two-class partition), and
  skeleton docs/invariants/ docs for 1-3 user-chosen modules. Asks dual-track
  enforcement questions (pre-commit framework, CI system, seed modules) as it goes.
  Creates exactly what /assurance-status Phase 1 checks for. Triggers: "assurance
  init", "onboard to assurance hierarchy", "scaffold assurance", "bootstrap
  governance".
argument-hint: "[optional: comma-separated seed module names]"
---

# /assurance-init — Assurance Hierarchy Bootstrap

## Description

Interactive bootstrap for onboarding a repository to the 6-layer assurance hierarchy. Scaffolds a strategic ROADMAP with horizon directories, a protected-surfaces policy partitioning harness config from module invariants, and skeleton invariant docs for 1–3 seed modules. Asks dual-track enforcement questions (pre-commit framework, CI system) and emits hook/CI stubs to match.

The artifacts created here are exactly what `/assurance-status` Phase 1 checks for — the two skills are tightly coupled. The real invariant ↔ test coverage gate is installed afterwards by `/invariant-coverage-scaffold`; this skill only prepares the ground.

## Instructions

You are a governance scaffolding assistant. The user wants to onboard their repository to the assurance hierarchy. Your job is to interactively gather the minimum set of answers needed, then write the skeleton files into the *target repo* (the user's current working directory) — not into the crosscheck plugin repo.

Work through the steps in order. Ask questions one block at a time — do not dump the whole questionnaire at once. Write files only after all answers in a block are collected.

### Step 1: Confirm Context and Prerequisites

Before asking anything, orient yourself:

1. Read the repo root: list top-level files, detect the primary language(s), and note whether `docs/`, `.claude/`, `.pre-commit-config.yaml`, `lefthook.yml`, `.husky/`, `.github/workflows/`, `.gitlab-ci.yml`, or `.circleci/` already exist.
2. If `docs/assurance/ROADMAP.md` already exists, **stop**. Emit: "Repo already has `docs/assurance/ROADMAP.md`. Re-running `/assurance-init` would overwrite governance. If you want to refresh scaffolding, delete the existing files first or run `/assurance-status` to see current state." Then exit.
3. **Pre-flight overwrite check.** Even when the ROADMAP is absent, any of the other artifacts this skill writes may already exist from a prior partial run or unrelated tooling. Before proceeding, stat every target path (exact list below) and build a `pre_existing` set:
   - `docs/assurance/{immediate,next,medium-term,aspirational}/README.md`
   - `.claude/rules/protected-surfaces.md`
   - `docs/invariants/README.md`
   - `docs/invariants/<module>.md` for each module candidate (from `$ARGUMENTS` if supplied, otherwise deferred until Step 2)
   - `.pre-commit-config.yaml`, `lefthook.yml`, `.husky/pre-commit-assurance-placeholder`
   - `.github/workflows/assurance.yml`, `.gitlab-ci.yml`, `.circleci/config.yml`
   If the set is non-empty, list the colliding paths to the user and ask: "These files already exist and would be modified or overwritten. Skip (keep existing), overwrite, or abort? (skip/overwrite/abort)". Default to `skip` on ambiguous answers. Record the decision and honour it in every write step below: when the decision is `skip`, leave the existing file untouched and note the skip in the Step 8 summary; when `overwrite`, replace the file; `abort` exits immediately with no writes.
4. If `/assurance-layer-audit` has not been run yet in this session (no prior layer-projection output visible), recommend running it first:

   > Tip: `/assurance-layer-audit` produces a per-layer reach projection that informs which modules are worth seeding as invariants. Running it first takes ~15 min and makes the answers below more grounded. Proceed without the audit? (yes/no)

   If the user says yes, continue. Otherwise wait for them to run the audit.

### Step 2: Gather Dual-Track Enforcement Answers

Ask three questions. Collect all three before writing any files.

**Q1 — Pre-commit framework:**

> Which pre-commit framework does this repo use?
> 1. `pre-commit.com` (the Python-based framework, `.pre-commit-config.yaml`)
> 2. `lefthook` (`lefthook.yml`)
> 3. `husky` (Node-based, `.husky/`)
> 4. None — repo has no pre-commit hooks today

Record the answer. If the user picks "None", note that the dual-track enforcement principle requires a fast local gate; `/invariant-coverage-scaffold` will later recommend installing one, but this skill will not force it.

**Q2 — CI system:**

> Which CI system does this repo use?
> 1. GitHub Actions (`.github/workflows/`)
> 2. GitLab CI (`.gitlab-ci.yml`)
> 3. CircleCI (`.circleci/`)
> 4. Other — please name it

Record the answer.

**Q3 — Seed modules for `docs/invariants/`:**

> Which 1–3 modules should seed `docs/invariants/`? Pick the modules whose behaviour is most load-bearing (a regression here would cascade). These will receive skeleton invariant docs you will fill out with `/draft-invariants` next.

If the user passed a comma-separated list as `$ARGUMENTS`, use those names and confirm. Otherwise ask, then confirm by echoing the module names back. Reject the list if it has more than 3 entries — the onboarding discipline is deliberately narrow.

### Step 3: Write `docs/assurance/ROADMAP.md`

Create the file with the following sections (filled with generalised language — strip any crosscheck- or xylem-specific references):

1. **Purpose** — one paragraph: "This directory tracks the execution of the 6-layer assurance hierarchy as applied pragmatically to this repository. It exists so the plan survives across long timeframes without depending on conversation context or ephemeral plan files. Each numbered item below has its own standalone doc under `immediate/`, `next/`, `medium-term/`, or `aspirational/`. That doc is the source of truth for scope, acceptance criteria, and kill criteria."
2. **Strategic Context** — state the long-term goal: "deterministic AI-driven software development": formally verified kernels for critical pure logic, contract graphs for integration boundaries, and spec-intent alignment checks for everything else. Include a "current projection" placeholder table with the six layers and `TODO: fill from /assurance-layer-audit output` in each reach cell.
3. **Dual-Track Enforcement Principle** — verbatim block:

   > Every deterministic assurance check added by this roadmap must produce two enforcement points:
   >
   > 1. **Pre-commit hook** — cheap, fast (< 5 s), lightweight. Blocks the commit locally. Must emit a human-readable error that includes the exact command to resolve the failure. LLMs acting as coding agents will see this error and must be able to fix it by following the instruction without human intervention.
   > 2. **CI job** — slower, more comprehensive, runs on every PR regardless of how the code change was authored. Must also emit actionable fix instructions. Can perform work too expensive for pre-commit (full test suites, container-based verification).
   >
   > Workflow phases are **not a substitute** for either enforcement point.
   >
   > Pre-commit hooks are fast attestation checks only — they must never invoke LLMs or run slow test suites. Heavy verification lives in CI and in dedicated binaries that the pre-commit hook verifies were run.

4. **Horizon Index** — four empty tables titled `Immediate (start now)`, `Next (4–8 weeks)`, `Medium-term (2–3 months)`, `Aspirational (scope and commit later)`, each with columns `# | Item | Cost | Doc`. Seed each table with a single `TODO` placeholder row so the format is obvious.
5. **Kill Criteria** — paragraph explaining: "Stop and re-plan the entire roadmap if any of the following becomes true." Seed with three editable TODO bullets the user must replace, covering (a) a first-kernel failure criterion (e.g., formal-verification pipeline unusable after N weeks), (b) a spec-alignment false-positive ceiling (default 30%, enforced later by `/intent-check`), and (c) an immediate-horizon delivery criterion (e.g., no item merged within 4 weeks).
6. **How to use this directory** — short block describing the `Status` vocabulary (see Step 4) and the rule that items are never deleted, only marked `Deferred` with `Reason:`.
7. **References** — empty bullet list with a `TODO` note.

### Step 4: Write Horizon Directory READMEs

Create four directories with a README.md in each:

- `docs/assurance/immediate/README.md`
- `docs/assurance/next/README.md`
- `docs/assurance/medium-term/README.md`
- `docs/assurance/aspirational/README.md`

Each README must contain:

1. A one-paragraph description of the horizon's purpose:
   - **Immediate**: "Items that are actionable now. Fits in the current sprint; blockers are known and small."
   - **Next**: "Items coming in the following 4–8 weeks. Scoped but not yet in progress; dependencies are mostly resolved."
   - **Medium-term**: "Items 2–3 months out. Scope is understood but execution is deferred behind current priorities."
   - **Aspirational**: "Items whose value is clear but whose scope or dependencies are not. Do not commit resources without a re-scoping pass."
2. **Status field vocabulary** — verbatim block:

   > Each item's standalone doc must carry a `Status:` field at the top. Valid values:
   >
   > - `Not started` — no work has begun; the item is waiting in the queue.
   > - `In progress` — at least one PR has been opened or is in draft.
   > - `Blocked` — work has started but is waiting on an external dependency. Blocked items must name the blocker on the next line (`Blocker: <what>`).
   > - `Done` — the item has shipped. Cross-reference the landing PR number.
   > - `Deferred` — the item has been intentionally postponed or cancelled. A `Reason:` line is **required** explaining why, and a link to the superseding item if applicable.
   >
   > Items are never deleted. Marking `Deferred` is how decisions are retired without losing the record.

3. A TODO bullet reminding the user to add items here using standalone docs (`<N>-<slug>.md`).

### Step 5: Write `.claude/rules/protected-surfaces.md`

Create the file with the two-class partition template. Use the following body verbatim, substituting `<language-property-test-glob>` with the repo's primary test-file pattern detected in Step 1 (e.g., `*_test.go`, `test_*.py`, `*.spec.ts`); if unsure, leave a `TODO` placeholder and flag it.

```markdown
# Protected surfaces

This repo partitions its protected surfaces into two classes. Both classes
require explicit human-authored amendments when modified — a machine-authored
change to either class is always a governance violation.

## Class A — Harness / workflow definitions

These files define how agents, pipelines, and automated workflows behave. A
silent change here can alter the behaviour of every downstream run without
leaving an obvious trace in the code diff.

Protect:

- Agent and orchestrator configuration (`.claude/agents/**`, `.claude/rules/**`)
- Pipeline/workflow YAMLs (`.github/workflows/**`, `.gitlab-ci.yml`, `.circleci/**`,
  any in-repo workflow definitions)
- Prompt templates consumed by agents at runtime
- Any file the harness interprets as "ground truth" for agent behaviour

## Class B — Module invariant specifications and tests

These files are the load-bearing behavioural contracts for the repo's core
modules. A weakening here silently lowers the correctness floor.

Protect:

- `docs/invariants/*.md` — the module invariant specifications
- Property-test files that cover those invariants, matching the repo's test
  convention (e.g., `<language-property-test-glob>`; see CONTRIBUTING if unsure)

## Amendment pattern

When a change to any protected file is proposed:

1. Name the authority — a human reviewer's approval is required. Automated
   agents must *propose* the amendment, not apply it.
2. Link the amendment to a Roadmap item (see `docs/assurance/ROADMAP.md`).
   Changes that don't trace to a roadmap item should be rejected; if no item
   exists, create one first.
3. Produce a governance-note block in the PR body explaining: (a) what is
   changing, (b) why, (c) which downstream behaviours are affected. Use
   `/protected-surface-amend` to generate this block mechanically.
4. Never weaken an invariant to make a failing test pass. Failing tests are
   evidence that either the code is wrong or the invariant is wrong — either
   way, a governance-note is required.
```

### Step 6: Write `docs/invariants/README.md` and Seed Module Docs

First write `docs/invariants/README.md`:

```markdown
# Module invariant specifications

This directory holds the load-bearing behavioural contracts for this repo's
core modules. Each file captures the invariants one module must preserve,
numbered (`I1`, `I2`, …) and paired with property tests that enforce them.

These documents and their property tests are **protected surfaces** (see
[`.claude/rules/protected-surfaces.md`](../../.claude/rules/protected-surfaces.md)):
they must not be modified, deleted, or weakened without an explicit
human-authored amendment.

## Seeded modules

<one bullet per module chosen in Step 2, with a relative link to the file>

## Coverage enforcement

Invariant ↔ test coverage is enforced mechanically by a coverage gate (to be
installed via `/invariant-coverage-scaffold`). Until that gate is wired in, the
mapping from invariant prose to covering test is maintained by convention only.

## Adding a new invariant

1. Write the prose in the module's spec with a `**IN. Name.**` header.
2. Add a property test with a `// Invariant IN: Name.` comment above it.
3. Commit both files together.

## Removing an invariant

Removing an invariant is a human-authored amendment, not a mechanical change.
Propose the removal in a PR description, justify it against the module's
guarantees, and obtain human review before merging.
```

Then, for each module the user named in Step 2, write `docs/invariants/<module>.md`:

```markdown
# <Module> invariants

Status: Skeleton — populate via `/draft-invariants <module>`.

## Purpose

<one-paragraph TODO describing what this module is responsible for>

## Invariants

**I1. TODO: first invariant name.** <!-- aspirational -->

<prose describing the property this invariant guarantees>

**Governance:** covered by `<path/to/test/file>::<test_name>` — update this
line when the property test lands.

---

**I2. TODO: second invariant name.** <!-- aspirational -->

<prose>

**Governance:** covered by `<path/to/test/file>::<test_name>`.

## References

- `.claude/rules/protected-surfaces.md` — amendment policy for this file.
- `docs/assurance/ROADMAP.md` — strategic context.
```

The `<!-- aspirational -->` markers indicate invariants that are declared but
not yet covered by a property test — this keeps the future coverage gate from
hard-failing on a freshly scaffolded repo.

### Step 7: Emit Pre-commit and CI Stubs (Dual-Track)

Using the answers from Step 2, write **two** stub files that the user will fill in when `/invariant-coverage-scaffold` runs next. Do not attempt to implement the coverage check itself — that's the next skill's job. These stubs exist only to anchor the dual-track shape.

When appending to an existing YAML file (pre-commit, lefthook, gitlab, circleci), first grep the file for the literal token `assurance-gate-placeholder`. If already present, skip the append and record that the stub is already in place — do NOT emit a second entry. If the target file exists but the Step 1 overwrite decision was `skip`, do not append; instead surface a note in the summary that the user must add the placeholder manually.

**Pre-commit stub.** Based on the Q1 answer:

- `pre-commit.com` → append a placeholder repo entry to `.pre-commit-config.yaml` (create the file if missing, with a minimal `repos:` list). Use hook id `assurance-gate-placeholder` with a `language: fail` and a message instructing the user to run `/invariant-coverage-scaffold`. Preserve existing entries — parse the file as YAML and append under the existing `repos:` key rather than blindly concatenating text.
- `lefthook` → append a placeholder command to `lefthook.yml` under `pre-commit.commands.assurance-gate-placeholder`. Parse existing YAML and merge; do not clobber other `pre-commit.commands.*` entries.
- `husky` → create `.husky/pre-commit-assurance-placeholder` (an executable shell script that echoes the instruction and exits non-zero). Note in the summary that husky does not auto-invoke this filename — the user must wire it into `.husky/pre-commit` manually, or `/invariant-coverage-scaffold` will do it later.
- `none` → write no file, but state in the summary: "No pre-commit framework configured. The dual-track principle requires a fast local gate; install one (pre-commit.com, lefthook, or husky) before `/invariant-coverage-scaffold`."

**CI stub.** Based on the Q2 answer:

- GitHub Actions → write `.github/workflows/assurance.yml` with a single job `assurance-gate-placeholder` that runs `echo "run /invariant-coverage-scaffold to install the real gate" && exit 1` on push and pull_request. If the file already exists from Step 1, honour the overwrite decision.
- GitLab CI → append a `assurance-gate-placeholder` job to `.gitlab-ci.yml` (parse YAML, preserve existing jobs).
- CircleCI → append the equivalent to `.circleci/config.yml` (parse YAML, preserve existing jobs and workflows).
- Other → write a `docs/assurance/ci-stub.md` describing the expected job shape (name, trigger, failing step) and ask the user to port it to their CI.

Annotate each stub with `# TODO(/invariant-coverage-scaffold): replace this placeholder with the real coverage check` so the intent is traceable.

### Step 8: Summarise and Point at the Next Two Skills

Emit a final report:

```
## Assurance init complete

Created:
- docs/assurance/ROADMAP.md
- docs/assurance/{immediate,next,medium-term,aspirational}/README.md
- .claude/rules/protected-surfaces.md
- docs/invariants/README.md
- docs/invariants/<module>.md  (one per seed module)
- <pre-commit stub path>  (or noted absence)
- <CI stub path>

Next steps — run these in order:
1. `/draft-invariants <module>` on each seeded module. This expands the I1/I2
   skeletons into real prose + governance blocks.
2. `/invariant-coverage-scaffold` to replace the placeholder hook and CI job
   with the real invariant ↔ test coverage gate.

After both skills run, `/assurance-status` Phase 1 should pass.
```

### Step 9: Verification Checklist

```
## Verification Checklist

- [ ] `docs/assurance/ROADMAP.md` exists with Strategic Context, Dual-Track
      Enforcement principle, Horizon Index, and Kill Criteria sections
- [ ] All four horizon directories exist, each with a README.md documenting
      the Status vocabulary (Not started / In progress / Blocked / Done /
      Deferred with Reason)
- [ ] `.claude/rules/protected-surfaces.md` exists with Class A + Class B
      partition and an amendment pattern
- [ ] `docs/invariants/README.md` exists and links to each seeded module doc
- [ ] One `docs/invariants/<module>.md` per user-selected module (1–3 total)
      with `Invariants` section skeleton
- [ ] Pre-commit stub file matches the framework answer from Q1 (or the
      "none" branch is flagged in the summary)
- [ ] CI stub file matches the CI answer from Q2
- [ ] Placeholder hook/CI fail loudly (so no one mistakes the skeleton for
      the real gate)
- [ ] Summary names the next two skills: `/draft-invariants` then
      `/invariant-coverage-scaffold`
- [ ] No xylem- or crosscheck-specific references leaked into the generated
      files (text should read as repo-agnostic)
- [ ] Pre-flight overwrite check (Step 1.3) ran and any pre-existing target
      files were either skipped or overwritten per the user's decision — no
      silent clobbering occurred
```

## Arguments

Optional comma-separated list of seed module names (1–3). If omitted, the skill asks interactively in Step 2.

Examples:
- `/assurance-init` — fully interactive, asks all questions
- `/assurance-init queue,scanner,runner` — pre-seeds the three module names
- `/assurance-init billing` — pre-seeds a single module

If more than 3 names are passed, the skill rejects the list and asks the user to narrow it — onboarding discipline requires a narrow starting set.
