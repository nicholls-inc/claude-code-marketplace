# add-orchestrator — Improvements Backlog

Improvement targets surfaced by the first mature-repo field test of the ADD pipeline. See [reports/add-orchestrator-field-report-2026-05-15.md](../reports/add-orchestrator-field-report-2026-05-15.md) for the underlying evidence. Numbered in rough priority order; "load-bearing" items are the ones the field test produced the strongest empirical case for.

## 1. Commit the scaffolding before attempting implementation in the same session [load-bearing]

**Status:** unimplemented
**Affects:** `agents/add-orchestrator.md`

**The change.** After the failing regression tests are written and the triage log is closed, the orchestrator commits `spec + invariants + failing tests + per-file plan` as a single commit *before* dispatching the implementation step. The two-commit pattern (scaffolding, then implementation) is the default, not the recovery path from a session crash.

**Evidence.** In the field test, the implementation session destroyed ~2h of production-code work via an incorrect bash loop. Recovery cost ~36 minutes because the scaffolding had been manually committed minutes before the disaster. Had the bash error fired earlier, the entire spec + invariants would have been lost.

**Why this is load-bearing.** The recovery property is the strongest empirical argument for the methodology. The orchestrator should make it cheaper and more automatic, not contingent on the operator noticing the right moment to commit.

## 2. Auto-close mechanical findings; only red-pen findings that need spec judgement [load-bearing]

**Status:** unimplemented; significant
**Affects:** `agents/add-orchestrator.md`, `skills/audit-spec-coverage/`, `skills/audit-invariant-consistency/`, `skills/spec-adversary/`

**The change.** Classify each audit finding by the kind of action it requires:

- **Mechanical** — the action is determined by the finding text. Examples: *"the spec covers endpoint X but no invariant references it; add an invariant"*; *"two invariants overlap; merge their wording"*; *"invariant Y is missing a `Governance:` hook"*. The orchestrator closes these autonomously.
- **Judgement** — the action requires choosing between defensible options. Examples: *"the spec's glossary contradicts an invariant — which definition is correct?"*; *"this invariant rejects a valid scenario the spec implies — narrow the invariant or amend the spec?"*. The orchestrator re-engages the user.

The orchestrator runs the mechanical closures autonomously and only red-pens the judgement-class findings. Auto-closed findings appear in the triage log with a `[auto]` marker; the operator reviews the full closure list at PR review time.

**Evidence.** The load-bearing operator feedback in the field test was a frustration with the 38-finding triage step *despite* the parallel fan-out and parallel audits running at machine speed. The operator's explicit target was *"the ADD framework should remain in force; I want it more automated."* Auto-closing mechanical findings moves the triage step closer to the speed of the rest of the pipeline.

**Risk.** Over-eager auto-closure encodes wrong invariants. Mitigation: each auto-closed finding logs its rationale; the closure batch is reviewed at PR-review time (the same surface the operator already sees); the `[auto]` marker makes it cheap to spot-check.

## 3. Pre-flight any invariant whose closure changes test-suite behaviour repo-wide

**Status:** unimplemented
**Affects:** `agents/add-orchestrator.md`; possibly a new helper skill

**The change.** During triage, flag any invariant whose `Governance:` hook plausibly affects the full test suite — a new pre-commit listener, a new lint rule, a new fixture-validation step. Before the apply step, run the invariant's hook in a probe worktree and report the affected test count alongside the triage decision.

**Evidence.** Closing one invariant added a SQLite-FK-pragma listener that surfaced ~170 latent FK violations across pre-existing test fixtures plus one real production seed-order bug elsewhere in the codebase. The operator had no advance warning that the invariant would gate the entire PR on fixture cleanup; the work arrived as a mid-PR surprise that consumed ~70 minutes.

**Why this isn't a bug.** The listener is what made the latent class visible — that's a methodology *feature*, not a defect. The fix is to move the visibility forward in time so it's a planning input, not a merge-time crisis.

## 4. Add a coverage-gate retrofit pass to the apply step

**Status:** unimplemented
**Affects:** `agents/add-orchestrator.md`; possibly `skills/invariant-coverage-scaffold/`

**The change.** When new invariants are produced, the orchestrator scans existing tests in scope of each new invariant and retrofits `# Invariant <ID>:` comments to satisfy the bidirectional coverage gate. New tests claim coverage by construction; pre-existing tests need an explicit retrofit pass.

**Evidence.** A post-merge finding flagged that four pre-existing admin auth tests should have carried `# Invariant <ID>:` comments to satisfy the gate. The bidirectional gate convention was younger than the audits in this run, so this is partly a temporal artifact — but as the convention stabilises across plugin work, the retrofit step must be in the orchestrator's scope.

## 5. Surface adversarial-probe routing during the audit step, not after

**Status:** unimplemented
**Affects:** `agents/add-orchestrator.md`

**The change.** After the three audits run, the orchestrator explicitly names which 1–2 of the N modules deserve a `spec-adversary` probe — based on audit findings density, declared module risk in the spec, and a per-run probe cap (two probes for seven modules in the field test was the right shape). This recommendation appears in the assistant turn that completes the audit step, before the probes are launched, not in the closing journal entry.

**Evidence.** In the field test the two highest-risk modules were probed correctly, but the routing logic was only documented after the fact. Making it explicit during the audit step turns an implicit decision into a reviewable one.

## 6. Tighten the trigger criterion for invoking ADD

**Status:** documentation-only
**Affects:** `agents/add-orchestrator.md` (preamble / when-to-use), possibly an entry in this directory's `JOURNAL.md`

**The change.** Document that ADD is for *features with recurring-bug-class symptoms*, not for individual bugs. The trigger phrase: *"this feature has produced N distinct bugs in succession with no behavioural contract, and reviewers keep flagging the same shape of out-of-scope latent risk."* One-shot bugs in features with a coherent contract are still narrow-fix work.

**Evidence.** The field-test feature met the bar (four bugs in succession; reviewer-flagged latent-risk classes recurring across the previous two PRs; the third bug matched the shape of those flags exactly). The cost shape made it clear ADD-per-bug would be wildly miscalibrated; ADD-per-feature-post-incident is the right frame.

## Notes on what didn't break

These should not be optimised; they worked.

- **Parallel fan-out.** Drafting 7 modules of invariants plus running 3 audits in ~15 minutes of wall-clock time is the part of the pipeline that runs at machine speed. Keep the shape.
- **The content-hashed session marker.** Locking dispatched subagents to a consistent spec version via a sha256-tagged marker file worked. Keep it.
- **Single-commit scaffolding shape.** When the scaffolding commit happened, the right artifacts were in it (spec, invariants, failing tests, per-file plan). Improvement #1 is about making it earlier and automatic, not about changing the artifact set.
- **Audit calibration.** A 0/38 reject rate suggests the audits were calibrated about right for this codebase. Re-evaluate after the next field test rather than tightening on a sample of one.
