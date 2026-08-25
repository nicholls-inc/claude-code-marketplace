# Tier–Layer Map

This document is Crosscheck's definition of what the AI-Native SDLC playbook calls
**"regulated and critical code"**. The playbook concentrates human approval at defined
gates but leaves each repository to say which of its own surfaces are critical. That
answer lives here: the tier a change falls into determines which artefact must be
committed before a human is asked to approve anything.

Enforcement is deterministic, not advisory. The **tier-gate CI job**
(`scripts/ci/tier-gate.mjs`, run by `.github/workflows/tier-gate.yml`) evaluates every
pull request against the rules below and must pass **before the review gate opens**.
A failing tier gate is not a review comment — the review has not started yet.

## Declaring a tier

Every pull request declares its tier in one of two ways:

- a `Tier: N` line in the PR body (`Tier: 2`), or
- a `tier:N` label on the PR (`tier:2`).

If both are present they must agree. If neither is present, the tier gate fails.

**Protected-path floor.** If the diff touches any path listed in
`.claude/rules/protected-surfaces.md`, the change is at least Tier 3 regardless of what
it declares. A declaration of Tier 1 or Tier 2 on such a diff is a gate failure, not a
judgement call. Declaring a tier *above* the floor is always permitted.

## The tiers

### Tier 1 — routine

**Scope:** documentation, tests, and non-behavioural code (formatting, renames,
comments, build plumbing that changes no output).

**Artefact required:** a reference to the governing `intent.md` — its path under
`intent/`, cited in the PR body.

**Worked example.** A PR fixes three typos in `crosscheck/README.md` and adds a missing
assertion to an existing vitest case. Nothing in the diff is a protected path, no
behaviour changes. The author declares `Tier: 1` and cites
`intent/2026-08-docs-tidy.md`. The tier gate checks the declaration and that the cited
intent file exists; review then covers the diff itself.

### Tier 2 — standard

**Scope:** behavioural code changes — anything that alters what the software does for a
user or a caller. New MCP tool behaviour, changed parsing, changed thresholds in
non-protected code.

**Artefact required:** a committed `spec.md` for the change (in addition to the
`intent.md` it derives from). The spec must flag unresolved concerns rather than quietly
settle them.

**Worked example.** A PR changes how `dafny_verify` reports timeouts so callers can
distinguish a timeout from a verification failure. No protected surface is touched, but
the observable behaviour of a tool changes. The author declares `Tier: 2`, commits
`spec.md` describing the new result shape and the open question about backwards
compatibility, and cites the intent. The tier gate confirms a `spec.md` is present in
the diff or already committed and referenced.

### Tier 3 — critical / protected

**Scope:** protected surfaces (per `.claude/rules/protected-surfaces.md`), gate logic,
hooks, CI enforcement, and invariants. In short: anything that changes how the project
decides whether other changes are safe.

**Artefacts required:**

1. a committed `plan.md` — files that change, order of work, risks, and the proof or
   tests that will demonstrate correctness, written so that an engineer who never saw
   the conversation could implement it;
2. an **intent-check attestation** (the record that the change's invariant tests were
   run and classified, per the false-positive tracker); and
3. for any edit to a protected path, a **governance-note block** — the
   `## Protected-Surface Amendment` block produced by `/protected-surface-amend`,
   naming each protected file it authorises, pasted into the PR description.

**Worked example.** A PR tightens the abort threshold inside
`crosscheck/skills/reason/SKILL.md`. That path matches `crosscheck/skills/*/SKILL.md`,
so the Tier 3 floor applies even though the author initially thought of it as a wording
change. The PR must carry `Tier: 3`, a committed `plan.md`, the intent-check
attestation, and a governance-note block naming that `SKILL.md`. With any one of the
three missing, the tier gate fails and no reviewer is asked to approve.

## Reading the map

The tier is a floor on scrutiny, never a ceiling. A reviewer who believes a Tier 2
change is really Tier 3 should say so in review; the correct remedy is to raise the
declaration and add the missing artefacts, not to argue the path list. Changes to the
path list itself are Tier 3 by construction, because `.claude/rules/**` is protected.

If you are unsure which tier applies, ask the Crosscheck maintainers via a GitHub issue
on this repository before opening the PR. The full explanation of the gate's failure
message is in `docs/gates/tier-layer-gate.md`.
