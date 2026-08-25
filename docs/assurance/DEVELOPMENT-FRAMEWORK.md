# Crosscheck Development Framework

Governing roadmap item: **PB-1** (`docs/assurance/ROADMAP.md`).

Crosscheck develops itself with the artefact chain from the AI-native SDLC
playbook. Every stage commits an artefact the next stage reads. Together the
intent, the spec, the plan, the diff, the tests and the review findings are the
audit trail — no stage depends on conversation context that has since scrolled
away.

Three kinds of control appear below. **Skills are advisory** — they make the
right behaviour likely. **Hooks and CI are deterministic** — they make it
always. **Human approval is concentrated at named gates**, each with an
explainer under `docs/gates/`.

## The chain

| Stage | Artefact committed | Where | Triggered by |
|---|---|---|---|
| 1. Plan | `intent/<slug>.md` | repo-root `intent/` | a person deciding to make a change |
| 2. Design | `spec.md` | beside the intent, or `crosscheck/docs/` for module specs | accepted intent committed |
| 3. Build | `plan.md`, then diff + tests | `plan.md` at repo root | accepted spec committed |
| 4. Test | verification logs, attestations | `.assurance/` | diff pushed |
| 5. Deploy | PR body with tier, review findings, approval records | GitHub PR | branch pushed |
| 6. Maintain | incident record + eval | `evals/` | production or dogfood failure |

Stage 6 feeds a fresh `intent/<slug>.md` back into stage 1. That loop is the
framework; each production incident gets an eval, and the eval stays in the
suite as a regression test.

### 1. Plan — `intent/<slug>.md`

Fields: problem statement, proposed outcome, affected users and systems,
constraints, open questions. `/informal-spec` is the skill that extracts this
precisely; it ends at a hard human sign-off gate
(`docs/gates/informal-spec-sign-off.md`) which writes
`Human sign-off: YYYY-MM-DD` into the file. No sign-off, no stage 2.

### 2. Design — `spec.md`

Generated from the accepted intent. A spec must **flag** concerns rather than
silently resolve them. Invariants are drafted from the spec by
`/draft-invariants`, whose red-pen gate
(`docs/gates/draft-invariants-red-pen.md`) lets a human strike, reword or add
invariants *before* any test is generated. Adequacy of the spec is probed by
`/audit-spec-coverage` (spec section → invariant coverage matrices),
`/audit-invariant-consistency` (contradictions within, across, and against the
spec) and `/spec-adversary` (up to three missing invariants). Each emits capped,
prioritised findings with a four-path triage block a human resolves.

For repos onboarding from scratch, `/assurance-init` scaffolds
`docs/assurance/`, the horizon directories and `.claude/rules/`.

### 3. Build — `plan.md`, diff, tests

The bar for `plan.md`: *an engineer who has never seen the conversation could
implement the change from the plan alone.* It lists the files that change, the
order of work, the risks, and the proof or tests that will show it worked.
`lowry` drives the red-to-green loop against the ratified invariant contract; it
refuses to reach green by weakening an invariant and instead stops and emits a
drift packet (`docs/gates/lowry-drift-packet.md`).

Edits to protected surfaces (`SKILL.md`, `agents/*.md`, invariant docs,
`docs/assurance/**`, `.claude/rules/**`, `.claude/hooks/**`, `evals/**`) require
a governance note from `/protected-surface-amend`. That skill refuses
unconditionally without a governing roadmap item
(`docs/gates/protected-surface-roadmap-refusal.md`), and the PreToolUse hook in
`.claude/hooks/` blocks the write until the note exists in the working tree
(`docs/gates/protected-surface-hook.md`).

### 4. Test — verification and alignment

`/intent-check` runs the round-trip triple (invariant prose, covering test, code
diff), appends to the false-positive tracker and emits an attestation. It
auto-refuses when the rolling false-positive rate reaches the kill threshold
(`docs/gates/intent-check-kill-criterion.md`); a failing verdict routes to fix
code / fix test / amend invariant
(`docs/gates/intent-check-verdict.md`). `/assurance-probe` measures test
strength on rotation, not per PR.

### 5. Deploy — the PR

The PR body declares `Tier: N` (see `docs/assurance/TIER-LAYER-MAP.md`) and
carries the review findings produced against `REVIEW.md`: separate passes for
bugs and logic, security, and compliance with the spec and plan; findings marked
Important or Nit; at most five nits reported and the rest summarised as a count;
generated paths excluded. Four CI workflows gate the merge:

- `spec-audit.yml` — runs the spec-coverage and invariant-consistency audits on
  changed invariant docs and specs.
- `protected-surface-check.yml` — fails when the diff touches a protected path
  without a matching governance note.
- `tier-gate.yml` — fails when the declared tier lacks its required artefacts
  (`docs/gates/tier-layer-gate.md`). Any protected path forces a Tier 3 floor.
- `incident-eval-check.yml` — fails when an incident record under `evals/` has
  no accompanying eval.

### 6. Maintain — incidents and evals

An incident record plus its eval lands under `evals/`. `auditor` runs read-only
consolidation passes and renders settled / active / drifted per artefact for
human adjudication (`docs/gates/auditor-verdicts.md`); it never edits what it
audits.

## Which agent runs which stretch

- `add-orchestrator` — spec → bulk-drafted invariants → batched audit → triaged
  findings → approved invariants (stages 1–2), with batched sign-offs.
- `byfuglien` — the implementation and verification chain (stages 3–4).
- `hellebuyck` — specification-chain assurance and governance scaffolding.
- `lowry` — the gated run-to-green loop (stage 3).
- `auditor` — stage 6 consolidation, read-only.

If you are unsure at any gate, ask the Crosscheck maintainers via a GitHub issue
on this repository.
