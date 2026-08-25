## Protected-Surface Amendment

**Target file(s):** crosscheck/skills/assurance-init/SKILL.md (+ 24 others, see Diff Plan)
**Class:** A — Harness/workflow definitions
**Matched rule:** `crosscheck/skills/*/SKILL.md`, `crosscheck/agents/*.md`
**Date:** 2026-08-25

### Change Description

Additive, presentation-only integration of the AI-native SDLC playbook: each listed skill and agent gains a succinct human-facing gate message in the standard "Action needed" format at its existing sign-off, triage, refusal or verdict delivery point, linking to a new explainer under `docs/gates/`; `assurance-init` additionally gains playbook-side scaffolding steps (intent/, REVIEW.md, hook wiring, evals/, framework docs). No phase logic, threshold, kill criterion, triage option set or sign-off requirement is altered anywhere.

### Rationale

Gate prompts currently assume familiarity with Crosscheck's assurance hierarchy; reviewers outside the project cannot tell what they are approving or what declining costs. Governing roadmap item PB-1 adopts the playbook's rule that human approval concentrates at defined gates and requires each gate to be self-explanatory at its point of delivery.

### Governing Roadmap Item

- **Path:** `docs/assurance/ROADMAP.md` (immediate horizon, item PB-1)
- **Title:** Adopt the AI-native SDLC playbook as the development framework and make every human gate self-explanatory
- **Scope coverage:** PB-1 explicitly authorises additive gate-message and scaffolding changes to every skill and agent that surfaces a human decision point.

### Authority

- **Authoriser:** REQUIRES HUMAN VERIFICATION: Authoriser draft is unverified. Reviewer must confirm or replace with named human (branch is agent-authored; expected authoriser: harry-nicholls as PR author).
- **Role:** PR author

### Diff Plan

| # | File | Lines | Invariant ID / Stage | Action |
|---|------|-------|----------------------|--------|
| 1 | crosscheck/skills/informal-spec/SKILL.md | sign-off prompt (~173–191) | sign-off gate output | added (gate message) |
| 2 | crosscheck/skills/draft-invariants/SKILL.md | §6 red-pen (~309–337) | red-pen gate output | added (gate message) |
| 3 | crosscheck/skills/intent-check/SKILL.md | Step 0 refusal (~46–60); verdict summary (~198, 235–237) | kill-criterion + verdict outputs | added (gate messages) |
| 4 | crosscheck/skills/protected-surface-amend/SKILL.md | refusal (~100–116); Step 7 template | refusal + PR-review outputs | added (gate messages) |
| 5 | crosscheck/skills/assurance-probe/SKILL.md | issue template (~74–107) | triage output | added (gate message) |
| 6 | crosscheck/skills/audit-spec-coverage/SKILL.md | findings header (~238–241) | 4-path triage output | added (gate message) |
| 7 | crosscheck/skills/audit-invariant-consistency/SKILL.md | findings header (~293–338) | 4-path triage output | added (gate message) |
| 8 | crosscheck/skills/spec-adversary/SKILL.md | findings block (~192–196) | triage output | added (gate message) |
| 9 | crosscheck/skills/assurance-init/SKILL.md | collision prompt (~36–43); Q1–Q3 (~59–98); new scaffolding steps | prompts + scaffolding | added (gate messages + playbook scaffolding) |
| 10 | crosscheck/agents/lowry.md | drift packet (~174–186, 221–223) | drift-packet output | added (gate message) |
| 11 | crosscheck/agents/add-orchestrator.md | steps 4/9 sign-offs (~96–109) | batch sign-off output | added (gate message) |
| 12 | crosscheck/agents/auditor.md | report template | verdict output | added (gate message) |
| 13 | docs/assurance/DEVELOPMENT-FRAMEWORK.md | new file | PB-1 framework doc | added |
| 14 | docs/assurance/ROADMAP.md | new file | PB-1 roadmap | added |
| 15 | docs/assurance/TIER-LAYER-MAP.md | new file | PB-1 tier definition | added |
| 16 | .claude/rules/protected-surfaces.md | new file | protection policy | added |
| 17 | .claude/hooks/protected-surface-guard.mjs | new file | deterministic hook | added |
| 18 | .claude/hooks/settings-snippet.json | new file | hook wiring fragment | added |
| 19 | .github/workflows/spec-audit.yml | new file | CI enforcement | added |
| 20 | .github/workflows/protected-surface-check.yml | new file | CI enforcement | added |
| 21 | .github/workflows/tier-gate.yml | new file | CI enforcement | added |
| 22 | .github/workflows/incident-eval-check.yml | new file | CI enforcement | added |
| 23 | scripts/ci/tier-gate.mjs | new file | CI enforcement | added |
| 24 | scripts/ci/incident-eval-check.mjs | new file | CI enforcement | added |
| 25 | evals/README.md | new file | evals scaffolding | added |

### Test / Coverage Impact

- No Class B invariant is touched; no covering test changes required.
- Class A: intent-check baseline refresh queued as a PB-1 follow-up once the new messages land (attestation content unaffected — messages are template additions only).
- REQUIRES HUMAN VERIFICATION: Reviewer must confirm per file that each amended output preserves the same decision options and thresholds (the evidence that this change is presentation-only).

### Review Checklist

- [ ] Rationale is anchored to a concrete trigger (not "cleanup" or "robustness").
- [ ] Authoriser is a named human (not a bot, not an agent).
- [ ] Governing roadmap item exists and actually covers this change.
- [ ] Diff plan enumerates every affected file, line range, and invariant ID / stage.
- [ ] Every added Class B invariant has a covering property test in this PR (or `<!-- aspirational -->` + linked issue). — n/a (no Class B changes)
- [ ] Every removed Class B invariant has its covering test removed or re-pointed in this PR. — n/a (no Class B changes)
- [ ] No invariant is being weakened purely to make a failing test pass.
- [ ] Class A edits: downstream attestation / intent-check baseline refresh is queued.
- [ ] This amendment block appears in the PR body **and** on the relevant invariant doc / governance section.
- [ ] All `REQUIRES HUMAN VERIFICATION:` markers above have been resolved.
