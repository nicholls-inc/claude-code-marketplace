# Gate explainers

A **gate** is any point in Crosscheck's workflow where a human is asked to make a judgement call, or where an automated check refuses to proceed without one — a sign-off, a triage decision, a kill-criterion refusal, a blocked commit. Every gate in this repository presents the same four-line message so a reader always knows, at a glance, what decision is being asked of them and what each option costs: a bolded **Action needed** line (an imperative, under ten words), one sentence naming the decision and its reason, one sentence stating what approving and declining each lead to, and a `Full explanation:` link. That link always points here, to the explainer file for that specific gate, committed under `docs/gates/` on the default branch — so whoever hits the gate can read the full background in plain language before deciding, without needing prior familiarity with Crosscheck. Each explainer defines any Crosscheck-specific term (protected surface, oracle independence, false-positive tracker, and so on) the first time it is used.

The table below inventories all sixteen gates: which explainer covers it, where in the codebase the gate actually fires and how the message reaches the reader, and what decision it asks for.

| # | Explainer | Surfaces at | Delivery | Decision asked |
|---|-----------|-------------|----------|-----------------|
| 1 | [informal-spec-sign-off.md](informal-spec-sign-off.md) | `crosscheck/skills/informal-spec/SKILL.md` (~173–191) | Skill output to the user | Signed off / revise / abandon the informal spec |
| 2 | [draft-invariants-red-pen.md](draft-invariants-red-pen.md) | `crosscheck/skills/draft-invariants/SKILL.md` §6 (~309–337) | Skill output to the user | Strike / reword / add / ship-it on drafted invariants before tests are generated |
| 3 | [intent-check-kill-criterion.md](intent-check-kill-criterion.md) | `crosscheck/skills/intent-check/SKILL.md` Step 0 (~46–60) | Skill auto-refusal to the user | Recalibrate or retire `/intent-check` once its rolling false-positive rate hits the kill criterion |
| 4 | [intent-check-verdict.md](intent-check-verdict.md) | same `SKILL.md` (~198, 235–237) | Skill output to the user | Fix code / fix test / amend invariant via `/protected-surface-amend` on a failed verdict |
| 5 | [protected-surface-amendment.md](protected-surface-amendment.md) | `crosscheck/skills/protected-surface-amend/SKILL.md` Step 7 template | PR description + governance-note block | Resolve REQUIRES HUMAN VERIFICATION markers before merging |
| 6 | [protected-surface-roadmap-refusal.md](protected-surface-roadmap-refusal.md) | same `SKILL.md` (~100–116) | Skill auto-refusal to the user | Open a new roadmap item / cite an existing one / abandon the amendment |
| 7 | [assurance-probe-triage.md](assurance-probe-triage.md) | `crosscheck/skills/assurance-probe/SKILL.md` (~74–107) | GitHub issue (one per finding, ≤3/run) | Accept / reject / defer each test-strength finding |
| 8 | [audit-spec-coverage-triage.md](audit-spec-coverage-triage.md) | `crosscheck/skills/audit-spec-coverage/SKILL.md` (~238–241) | `findings-coverage.md` file | 4-path triage per gap: accept-fix-invariant / accept-amend-spec / reject / defer (cap 15) |
| 9 | [audit-invariant-consistency-triage.md](audit-invariant-consistency-triage.md) | `crosscheck/skills/audit-invariant-consistency/SKILL.md` (~293–338) | Skill output across 3 passes | Same 4-path triage per finding |
| 10 | [spec-adversary-triage.md](spec-adversary-triage.md) | `crosscheck/skills/spec-adversary/SKILL.md` (~192–196) | Skill output to the user | Accept (promote via separate PR) / reject / defer each proposal (cap 3) |
| 11 | [assurance-init-prompts.md](assurance-init-prompts.md) | `crosscheck/skills/assurance-init/SKILL.md` (~36–43, ~59–98) | Interactive skill prompts | Skip / overwrite / abort on file collisions; answers to scaffolding questions Q1–Q3 |
| 12 | [lowry-drift-packet.md](lowry-drift-packet.md) | `crosscheck/agents/lowry.md` (~174–186, 221–223) | Agent output to the user | Accept a governance amendment or send the implementation back |
| 13 | [orchestrator-batch-sign-off.md](orchestrator-batch-sign-off.md) | `crosscheck/agents/add-orchestrator.md` (~96–109) | Agent output, batched at steps 4 and 9 | Sign off a batch of decisions; ratified on PR merge |
| 14 | [auditor-verdicts.md](auditor-verdicts.md) | `crosscheck/agents/auditor.md` report | Agent report to the user | Accept or reject each proposed remediation |
| 15 | [protected-surface-hook.md](protected-surface-hook.md) | new PreToolUse hook (`.claude/hooks/`) | Blocked-write message on stderr | Add a governance-note block before editing a protected surface (or abandon the edit) |
| 16 | [tier-layer-gate.md](tier-layer-gate.md) | new tier-gate CI job (`.github/workflows/`) | Failed CI check message | Declare the correct `Tier: N` and supply the artefact the tier requires, or reduce the diff's scope |

Anyone unsure how to respond to a gate, or who thinks a gate is miscalibrated, should raise it with the Crosscheck maintainers via a GitHub issue on this repository.
