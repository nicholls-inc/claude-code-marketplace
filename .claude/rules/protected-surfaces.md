# Protected surfaces

This repository partitions its protected surfaces into two classes. Both classes
require an explicit human-authored amendment when modified — a machine-authored
change to either class, landed without that amendment, is always a governance
violation.

A *protected surface* is a file whose content silently governs the behaviour of
future runs or defines a load-bearing correctness contract. Editing one changes
what every downstream agent, gate, or test believes to be true, so the edit must
be legible in the pull request rather than buried in a diff.

Enforcement is dual-track:

- **Deterministic layer:** `.claude/hooks/protected-surface-guard.mjs`, a
  `PreToolUse` hook that blocks writes to any path matching the machine-readable
  list below unless a governance-note block naming that file exists in the
  working tree. The hook parses its glob list from this file, so this document
  is the single source of truth for what is protected.
- **Advisory layer:** `/protected-surface-amend`, the skill that drafts the
  governance-note block (change description, rationale, governing roadmap item,
  authority, diff plan, test/coverage impact, review checklist). Skills make the
  policy likely; the hook makes it always; the human reviewer decides.

## Class A — Harness / workflow definitions

These files define how agents, skills, pipelines, and gates behave. A silent
change here alters every downstream run without leaving an obvious trace in the
code diff.

Protect:

- Skill behaviour definitions — `crosscheck/skills/*/SKILL.md`
- Agent and orchestrator definitions — `crosscheck/agents/*.md`
- Harness rules and deterministic hooks — `.claude/rules/**`, `.claude/hooks/**`
- Governance and roadmap documents the gates read — `docs/assurance/**`
- Evaluation suites that encode regression expectations — `evals/**`
- CI enforcement — `.github/workflows/**`, `scripts/ci/**` (the tier gate and
  incident-eval checks live here; editing them can silently widen what merges)

## Class B — Module invariant specifications and tests

These files are the behavioural contracts for the repository's core modules. A
weakening here silently lowers the correctness floor.

Protect:

- `crosscheck/docs/invariants/**` and `docs/invariants/**` — the module
  invariant specifications
- The property tests that cover those invariants: `*.test.ts` under
  `crosscheck/mcp-server/tests/` and `*_test.go` under `tools/`

## Amendment pattern

When a change to any protected file is proposed:

1. **Name the authority.** A named human reviewer's approval is required.
   Automated agents must *propose* the amendment, never self-authorise it.
2. **Link to a roadmap item.** Every amendment must cite a governing item in
   `docs/assurance/ROADMAP.md`. If no item covers the change, open one first —
   `/protected-surface-amend` refuses to synthesise governance.
3. **Produce a governance-note block.** Run `/protected-surface-amend` to
   generate the `## Protected-Surface Amendment` block mechanically. It lands in
   `.assurance/protected-surface-amend/<slug>-<date>.md`, in the pull request
   body, and in the governance section of any affected invariant document — in
   the *same* pull request as the edit itself.
4. **Resolve every marker.** `REQUIRES HUMAN VERIFICATION:` markers in the block
   are the reviewer's red-pen list. Merging with an unresolved marker is a
   governance violation.
5. **Never weaken an invariant to make a failing test pass.** A failing test is
   evidence that either the code or the invariant is wrong — either way an
   amendment is required, and the direction of the change must be argued, not
   assumed.

If you are unsure whether a file is protected, treat it as protected and ask the
Crosscheck maintainers via a GitHub issue on this repository.

## Machine-readable path list

The deterministic hook parses the fenced block below. One glob per line, no
comments, no blank lines. Adding or removing a line here is itself a Class A
amendment.

```
crosscheck/skills/*/SKILL.md
crosscheck/agents/*.md
crosscheck/docs/invariants/**
docs/invariants/**
docs/assurance/**
.claude/rules/**
.claude/hooks/**
evals/**
.github/workflows/**
scripts/ci/**
```
