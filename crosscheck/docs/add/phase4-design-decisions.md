# Phase 4 gated-implementation agent — resolved design decisions

**Status:** Design record (resolved). The operational agent is `crosscheck/agents/lowry.md`.
**Date:** 2026-06-01
**Pass condition:** the ratified A1–A6 acceptance oracles (`crosscheck/conformance/acceptance/`, RATIFY.md, #226). A4 (`ClassifyCommitShape`) and A5 (drift-stop) are the load-bearing ones; A4 ships its deterministic mechanism in this work.
**Grounding:** the `ngst` greenfield field test ([`reports/add-greenfield-field-report-ngst.md`](reports/add-greenfield-field-report-ngst.md)), whose `cli` module was a *hand-run* of exactly this loop.

This doc records the design decisions behind Phase 4 — the run-to-green loop that turns *approved invariants* into *green, governed code*. `add-orchestrator` owns the front half (spec → approved invariants); byfuglien owns the verification proofs; **Phase 4 (`lowry`) owns the loop between them**: drive the build to green within strict commit shapes, and refuse to cheat by weakening the contract.

## D1 — Entry contract

**Decision.** `lowry` refuses to start unless all hold: (1) ratified/approved invariant docs exist for the target module(s); (2) the build is red (there is work to do); (3) the gate bundle (D2) is wired and runnable; (4) the operating mode is known (from `add-orchestrator` Step 0 — `add`/`bootstrap`). Missing any → stop and name what's missing.

**Why.** The ngst recovery-property finding: a durable on-disk contract (spec + invariants + failing tests + plan) let an agent recover ~2h of destroyed work in ~36min. The entry contract makes that durability a precondition, not an afterthought.

## D2 — The gate bundle

**Decision.** The loop drives this set to green, and reports each independently: **build**, **tests**, the **bidirectional invariant-coverage gate** (run *early* — before review), the **conformance oracle**, and — once their harness exists — the **A1–A6 acceptance oracles**. Today only the deterministic gates are enforceable: build, test, coverage, conformance, plus A4 (commit-shapes). A3 (mode-tags) is **not yet** enforceable — its acceptance oracle is RED by design because the mode system is unwired (CLAIM-MODES), so `lowry` treats A3 as a checklist item alongside the judged oracles. The judged oracles (A1/A2/A5/A6) need a scenario-runner + LLM judge that **does not exist yet**; `lowry` treats their pass condition as a checklist a human/`hellebuyck` confirms, and says so rather than claiming green.

**Why.** ngst's `cli` module: `hellebuyck` caught a coverage-gate *module-derivation* blocker (tests under `cmd/ngst/` tagged `module=ngst` while `cli.md` declared `module=cli`) — invisible to code review, caught only by running the coverage gate. Running it *early* is the fix.

## D3 — Commit-shape classifier

**Decision.** Every commit `lowry` emits is exactly one of three shapes; anything else is rejected:

| Shape | Subject grammar | Extra requirement |
|---|---|---|
| `implementation` | `implementation: <summary>` | none — fast-path, no interrupt |
| `new-invariant` | `new-invariant: <summary>` | human checkpoint before merge |
| `governance-amendment` | `governance-amendment: <summary>` | body MUST carry `amendment-kind: <propagated-discovery\|intent-refinement\|drift\|retraction>` |

The canonical classifier is `ClassifyCommitShape(subject, body)` in `crosscheck/conformance/acceptance/oracles.go` (the A4 seam, now implemented). `lowry` applies its grammar; a commit it cannot classify legally is not authored.

**Why.** A4's ratified pass condition. ADR-005's per-commit classification is what catches drift early.

## D4 — The single batched human interrupt

**Decision.** `implementation` commits flow without interrupt. Governance-class decisions (drift, new-invariant, intent-refinement) are **accumulated and surfaced once** as a batch, not drip-fed per commit. The batch is one `AskUserQuestion`-style checkpoint listing each staged governance amendment with its proposed classification + justification.

**Why.** ngst's `add-orchestrator` run concentrated friction in serial triage of 26 findings; the operator feedback was "automate more of finding triage where the action is mechanical." Batching the irreducible (judgment) decisions and auto-flowing the mechanical (implementation) ones is the response (cf. #205–#208).

## D5 — The defer path (granularity)

**Decision.** Each governance amendment requiring human judgment is **deferred as a single commit** — one classification, one justification — even if it spans multiple `docs/add/` files. `lowry` stages the change, prepares the commit message with the trailer, and surfaces it at the D4 batch; it does not author the commit until the classification is approved.

**Why.** ADR-005 per-commit granularity: a PR may mix classifications, so the unit of classification is the commit, not the PR.

## D6 — Two-tier completion contract

**Decision.** `passes-oracles ≠ matches-intent`. Reaching green (all D2 gates pass) is **necessary, not sufficient**. `lowry` reports *"passes oracles"* and explicitly **does not** claim *"matches intent"* — that is a human / `hellebuyck` (`/intent-check`, `/rationale`) judgment. This disclaimer is load-bearing and mirrors `add-orchestrator`'s CLAIM-ADDORCH-TERMINAL.

**Why.** The whole point of the assurance hierarchy: a green build over a wrong spec is false confidence.

## D7 — Kill criterion + checkpoint discipline (A5 — load-bearing)

**Decision.** When the **only** path to green weakens an invariant `I`, `lowry` **STOPS** and emits a **drift packet** — a staged `governance-amendment` with `amendment-kind: drift` and a justification answering *"did we want this behaviour, or did the implementation drift?"* — for the D4 batch. It **never** silently relaxes, deletes, or edits `I` or its check to reach green. A silent stall (stopping without emitting the drift packet) is equally a failure. `lowry` also commits the scaffolding (contract on disk) early and at every checkpoint.

**Why.** This is A5 verbatim — the critical guardrail that makes D6 real. An agent that reaches green by quietly relaxing the contract defeats the entire framework.

## D8 — Acceptance behaviours from #205–#208

**Decision.** `lowry` honours these as acceptance behaviours (it does not re-file them): auto-close *mechanical* findings and red-pen only *judgement* findings (#205/#206); pre-flight invariants whose closure changes test-suite behaviour repo-wide (#207); a coverage-gate retrofit pass in the apply step (#208). It also tracks reviewer verdicts (byfuglien/hellebuyck) and **blocks merge on unresolved dissent** — the ngst `cli` run had no such quorum gate, which this closes.

## Deferred / not in this work

- The **judged-oracle harness** (scenario runner + LLM judge for A1/A2/A5/A6) — A5's *behavioural* validation waits on it; the drift-stop *discipline* is specified here and in `lowry.md` regardless.
- A standalone **deferred-coverage lifecycle** tool (follow-up tickets keyed to the blocking module) — `lowry` records the gap in the JOURNAL; full automation is a follow-up.
- Bootstrap-mode Phase 4 (run-to-green over code-derived invariants) reuses the same loop; the worked examples here are `add`-mode.
