# Crosscheck ledger-gap roadmap (the "epic")

This is the tracked home for the **known gaps** recorded in the conformance
oracle's ledger ([`../../conformance/claims.json`](../../conformance/claims.json)).
The oracle classifies a narrative claim as `known-gap` when the docs over-promise
and the work is pending. A gap that is "known" but tracked nowhere is exactly the
`plausible ≠ correct` failure mode the oracle exists to catch — so every
`known-gap` entry in `claims.json` carries a `tracked_in` link pointing at a
section below, and the oracle **fails CI** if a `known-gap` has no such link (see
`analyze` in [`../../conformance/main.go`](../../conformance/main.go)).

Each section records: the claim, current reality, the disposition / next step,
and — where one exists — the self-invalidating `present_artifact` check that
flips the oracle from PASS to FAIL the day the gap is closed (so the ledger
re-fires and forces re-triage rather than silently going stale).

These gaps are intentionally **not** scheduled by this documentation pass; this
doc gives them a durable in-repo home and makes the ledger's `tracked_in` links
resolve to something real (they previously pointed at `ISSUE#…` placeholders that
were never filed). Closing any gap is separate, release-triggering work.

---

## CLAIM-PHASE4

- **Claim:** ADD includes a Phase 4 gated-implementation loop that drives code to
  green within three legal commit shapes.
- **Reality:** Designed, not shipped. No implementing agent exists — `agents/`
  holds only `byfuglien`, `hellebuyck`, and `add-orchestrator`.
- **Self-invalidating check:** `present_artifact` watches `agents/lowry.md` with
  `expect_present: false`. The day a Phase 4 agent lands at that path the oracle
  goes RED, forcing this claim to be re-reviewed and re-classified.
- **Disposition:** Tracked, unscheduled. When picked up: ship the Phase 4 agent
  (`agents/lowry.md`), wire it after `add-orchestrator`'s routing recommendation
  (`CLAIM-ADDORCH-TERMINAL` is accurate today — the orchestrator stops at the
  recommendation and disclaims implementation closure), and flip this entry to
  `reviewed-accurate`.

## CLAIM-MODES

- **Claim:** Three operating modes (bootstrap / add / transitional) with
  module-level mode tags honoured by skills.
- **Reality:** Not wired. `add-orchestrator` hardwires the signed-off-spec path
  (Step 1 requires ≥10 RFC-2119 keywords) and reads no mode tags; legacy
  hardening and empty-repo greenfield have no orchestrated entrypoint.
- **Self-invalidating check:** none — no single artifact cleanly represents the
  mode system. Re-review manually when the orchestrator gains mode selection.
- **Disposition:** Tracked, unscheduled. When picked up: introduce mode selection
  in `agents/add-orchestrator.md`, document the three modes, and flip this entry
  to `reviewed-accurate`. Until then it stays a disclosed gap.

## CLAIM-METHODOLOGY-COMMITTED

- **Claim:** The canonical ADD methodology (`methodology.md`, `glossary.md`, the
  ADRs) lives in the repo.
- **Reality:** Never committed. Only `docs/add/README.md`, `docs/add/JOURNAL.md`,
  and `docs/add/orchestrator-improvements.md` shipped (plus this roadmap).
- **Self-invalidating check:** `present_artifact` watches `docs/add/methodology.md`
  with `expect_present: false`. Note this roadmap is **not** `methodology.md`: it
  tracks the gaps, it is not the canonical methodology, so it neither satisfies
  this claim nor trips the check.
- **Disposition:** Tracked, unscheduled. When picked up: commit
  `docs/add/methodology.md` as the canonical methodology, then flip this entry to
  `reviewed-accurate`.

## CLAIM-AUDITOR

- **Claim:** A Phase 5 Auditor agent renders settled / active / drifted verdicts
  on every artifact.
- **Reality:** Designed, not shipped. No auditor agent exists.
- **Self-invalidating check:** `present_artifact` watches `agents/auditor.md` with
  `expect_present: false`. Lands RED the day an auditor agent ships at that path.
- **Disposition:** Tracked, unscheduled. When picked up: ship the auditor agent
  (`agents/auditor.md`) and flip this entry to `reviewed-accurate`.

## CLAIM-SELF-COVERAGE

- **Claim:** Crosscheck applies its own assurance machinery to itself.
- **Reality:** Only the leaves are verified — `docs/invariants/` covers three
  helper functions (`parseDafnyOutput`, `shouldExclude`, `extractDifficultyMetrics`).
  The conformance oracle is the **first** trunk-level self-check; orchestrators,
  gates, and the workflow remain unverified.
- **Self-invalidating check:** none — coverage is a spectrum, not a single
  artifact. Re-review manually as trunk coverage grows.
- **Disposition:** Tracked, unscheduled. The conformance oracle is the first
  increment. As orchestrator/gate self-checks are added, narrow this claim's
  `reality` toward `reviewed-disclosed` and eventually `reviewed-accurate`.
