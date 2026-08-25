# Gate: Tier-Layer Gate (CI Failure)

## What this gate protects

Not every change carries the same risk. A typo fix and a change to the logic that decides whether other changes are safe are not the same kind of event, and they should not require the same paperwork. Crosscheck sorts every pull request into one of three **tiers** — routine, standard, or critical — and each tier requires a different **artefact** (a committed document proving the right groundwork was done) before a human reviewer is asked to approve anything. The full definition lives in `docs/assurance/TIER-LAYER-MAP.md`; this gate is what enforces it.

The **tier-gate CI job** (`scripts/ci/tier-gate.mjs`) runs on every pull request and checks two things: that a tier was declared, and that the artefact its tier requires is actually present. It runs deterministically, in CI — the same way for every PR, with no judgement calls — and it runs **before the review gate opens**. A failing tier gate means review has not started yet, not that a reviewer raised a concern.

## How to declare a tier

A pull request declares its tier in one of two ways: a `Tier: N` line in the PR body (e.g. `Tier: 2`), or a `tier:N` label on the PR. If neither is present, the gate fails immediately and asks you to add one.

## Which tier applies

- **Tier 1 — routine.** Documentation, tests, and non-behavioural code (formatting, renames, build plumbing that changes no output). Requires a reference to the governing `intent.md` — the "why we're doing this" document from the Plan stage — cited by path in the PR body or added under `intent/`.
- **Tier 2 — standard.** Behavioural code changes: anything that alters what the software does for a user or caller. Requires a committed `spec.md` — a design document generated from the intent that must flag open concerns rather than silently resolving them.
- **Tier 3 — critical/protected.** Protected surfaces (see below), gate logic, hooks, CI enforcement, and invariants — anything that changes how the project decides whether other changes are safe. Requires a committed `plan.md` (detailed enough that someone who never saw the discussion could implement the change from it alone), an **intent-check attestation** (a record that the change's invariant tests were run and classified), and, for any protected file in the diff, a **governance-note block** naming it.

## Why a protected path forces Tier 3

`.claude/rules/protected-surfaces.md` lists paths — skill definitions, agent prompts, invariants, hooks, CI enforcement scripts, and the rules file itself — that define how Crosscheck decides what is correct. If a diff touches any of these, the gate imposes a **floor of Tier 3** regardless of what the PR declares. Declaring Tier 1 or Tier 2 on such a diff is not a judgement call the gate defers to you on — it is a straightforward failure, because a change to how safety is decided cannot be waved through as routine. You may always declare a tier *above* the floor; you may never declare one below it.

## What the incident-eval check adds

When a pull request fixes a bug that caused a production incident, the tier gate additionally expects a durable regression test: an **eval** under `evals/` that reproduces the incident and stays in the test suite permanently, so the same failure cannot silently recur. This follows the playbook's principle that every incident earns a permanent test, not just a one-off patch. If your PR resolves an incident and no matching eval is present, add one before the gate will pass.

## What each decision means

- **Approving (declaring the correct tier and supplying its artefacts)**: the pull request now carries an auditable record matching its actual risk level, and the tier gate passes, opening the review gate.
- **Declining (leaving the tier or artefacts as they are)**: the change stays blocked. Nothing is merged and no reviewer is asked to look at it until the tier and artefacts are corrected.

## How long this takes

For Tier 1 and Tier 2, usually a few minutes — citing or writing a short intent or spec document. For Tier 3, expect tens of minutes, since a genuine `plan.md`, an intent-check attestation, and (for protected files) a governance-note block from `/protected-surface-amend` all take real drafting time.

## Who to ask if unsure

The Crosscheck maintainers, via a GitHub issue on this repository.
