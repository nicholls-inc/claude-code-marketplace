# Audit of `findings-and-methodology-v2.md` against a broader session corpus

**Status:** Hand-off report. For the agent who authored `findings-and-methodology-v2.md` (hereafter "v2") to action.
**Date:** 2026-05-11.
**Author:** A separate agent commissioned to stress-test v2 against sessions outside its evidence base.

This report exists because v2 was drafted from five transcripts (webhook spec, CKG-initial-spec, CGV-system-design, CKG-design-refinement, CKG-MVP-spec), three of which are CKG-family and all of which are mature-repo freeform design conversations. The corpus was narrow. Harry asked for an audit against other spec-writing sessions, with the explicit instruction to find counter-evidence rather than confirmation. The findings below come from that audit.

The recommended revisions to v2 are listed inline against each finding, with cited evidence the v2 author can re-verify.

---

## 1. Scope

**In scope.** Human-agent freeform spec-definition sessions and the artifacts they produced. Spec lifecycle from greenfield to mature repo. ID semantics in spec docs. Adversarial review patterns. Artifact location and lifecycle. The ngst session as the originating case for ADD.

**Out of scope (per Harry, 2026-05-11).**
- Pipeline-orchestrated spec work (xylem worker chain — Harry has abandoned that pipeline for now and has no plans to use pipelines for spec work in the near term). v2 should not be revised to accommodate it.
- The mechanics of any individual existing skill (e.g. `/draft-invariants`'s elicitation cadence). Skill audits live elsewhere; this report only mentions them where they affect ADD's role.
- The pickbrain search methodology itself.

**A caveat on evidence depth.** Most findings cite session IDs and file paths. Some claims are anchored on file-content excerpts and session-turn fragments from `pickbrain`'s semantic index rather than full session reads. Where a finding would benefit from a full session dump (e.g. the May 5 adversarial-review session), the report says so and the v2 author should dump the session before acting.

---

## 2. Reframing finding — ngst is the originating case for ADD

Harry's stated motivation for ADD: the existing crosscheck plugin skills are not set up to work for an empty repository. The `ngst` field report (`/Users/harry.nicholls/repos/ngst/field-reports/crosscheck--2ff1cde8-3ff--2026-05-09.md`) documents the failure mode end-to-end: `/draft-invariants` ran 9 sequential AskUserQuestion prompts in a session where 5 of the 6 substantive contract questions had answers in `docs/design/fabricator-spec.md` §3.2 / §14.3, because the skill was designed to elicit contracts cold rather than read existing specs first. The complaint reached the crosscheck team because the workflow is perceived as unitary.

v2's working pattern (§3.5 step 1, "Read available context silently") implicitly assumes the kind of context that exists in `ev.shapes`, `xylem`, or `formal-verify` — a ROADMAP, protected-surfaces.md, an invariants directory, prior design docs to anchor against. None of that exists in greenfield. In `ngst`'s case the spec *did* exist (`docs/design/fabricator-spec.md`) but the skills didn't read it; in a true empty repo there is nothing to read at all. v2 does not address either case.

**Recommendation.** Add a new §4.9 "Greenfield as the ADD origin case" naming this scope. The methodology must be evaluated on whether it works on a fresh repo from message one without reproducing the `/draft-invariants` 9-prompt friction band. v2's "throw out / keep" lists in §3.1–§3.2 should be re-checked against the greenfield scenario specifically, since several discarded items (per-artifact Status, phase model) may behave differently when there's no prior artifact to attach Status to.

This is the highest-leverage revision in this audit. The other findings are calibrations; this is a scope-of-applicability gap.

---

## 3. Finding — §3.2 over-discards IDs

v2 §3.2 lists "IC# / S# / F# / M# / B# numbering as user-facing structure" among the items to throw out. Harry's clarification (2026-05-11): the problem with v1 ADD was the *number of simultaneous ID schemes*, not IDs themselves. IDs in the spec doc, used to refer to sections, are fine. IDs for documenting and managing invariants are fine and Harry intends to keep using them.

The audit found four productive uses of user-facing IDs in Harry's own freeform spec corpus that v2's blanket rule would mistakenly proscribe:

| Artifact | ID scheme | Anchors to |
|---|---|---|
| `piano-play/docs/PRD.md` | `OQ-1`…`OQ-9` Open Questions table with Answered/Pending status | Real open questions, each row has a resolution prose column |
| `xylem/docs/plans/sota-gap-implementation-2026-04-11.md` | `C-1`, `G-1`, `H-1` issue umbrellas | GitHub issues (each block is a GitHub issue template body) |
| `~/.claude/plans/agile-nibbling-peacock.md` (ev.shapes, May 9) | `SPEC-01`…`SPEC-30` | GitHub issues / per-issue spec files |
| `ev.shapes/docs/invariants/secure-logger.md`, `scheduler.md`, `ai-knowledge.md` | `I1`, `I2`, … (invariant IDs) | Property tests via `// Invariant <ID>:` comment convention; load-bearing for the coverage gate |

The distinguishing property: every working ID scheme anchors to **something external to the doc** — a GitHub issue, an open question with a resolution, a property test. The v1 ADD scheme failed because IC#/S#/F#/M#/B# were five simultaneous ID schemes anchoring to *invented internal partition* (intent cluster / spec section / finding / module / bucket), each one organising the doc's own structure rather than referring out to a referent that exists independently.

**Recommendation.** Revise §3.2 to remove the blanket "throw out IDs" line. Replace with a principle in §3.3 along the lines of:

> **IDs in the spec doc are fine when they anchor to an external referent — a GitHub issue, an open-question row with a resolution column, a property test, or another spec doc — and when there is at most one ID scheme active in any single doc. Multiple simultaneous ID schemes (IC# *and* S# *and* F# *and* M# *and* B#, all in the same project) is the broken-form failure. One scheme that points outward is the working form. Invariant IDs (e.g. `I1`, `I2`) are preserved as-is; they anchor to property tests and are governed by the invariant-coverage gate.**

This preserves what Harry wants to keep (invariant IDs; section refs in the spec; OQ-/SPEC-/C- IDs in the broader corpus) without re-opening the door to the v1 ceremony stack.

---

## 4. Finding — §3.5 step 1 assumes context that doesn't exist in greenfield

v2 §3.5 step 1: "Read available context silently. Project files relevant to the request. Prior conversations if the platform supports retrieval and they exist."

In greenfield (ngst at session start, or any fresh repo) there is nothing to read. The agent's default opening pattern breaks here. The transcripts in v2's evidence base do not include a greenfield session; v2 cannot have observed how the pattern degrades.

The risk this exposes: with no project files to anchor on, the agent either (a) elicits from the user with the same fixed-cadence interview that broke `/draft-invariants` in ngst, or (b) cargo-cults conventions from the agent's training set, importing structure from mature repos by inertia. Either failure mode produces exactly the ceremony v2 is meant to avoid.

**Recommendation.** Add a §3.5a or extend §3.5 step 1 with a greenfield variant:

> **Greenfield variant.** When there are no project files to read (no `docs/`, no `README.md`, no prior spec), the agent does not default into the elicitation cadence used by existing crosscheck skills. Instead:
> - Ask one open-ended question about the project's purpose and audience. Not a structured multi-question interview.
> - Produce a single short draft (`Status: Draft`) from the answer. The draft is the elicitation device — Harry red-pens it rather than answering questions cold.
> - Resist importing conventions from `ev.shapes`/`xylem`/`formal-verify` by default. Those are mature-repo conventions and they don't fit a fresh repo.

The v2 author should treat this as the v2 doc's most important open question, not a minor revision. The ngst friction band is what motivates ADD; if v2's opening pattern reproduces it, the methodology has not solved the problem it was commissioned to solve.

---

## 5. Finding — adversarial-subagent-review is a working pattern v2 omits

Session `4d7542e6-d0de-4431-aa5d-c857014811f8` (formal-verify, May 5 18:35): Harry's prompt was *"Use a couple of subagents to adversarially review your proposal. Then refine it based on their output."* The agent dispatched two reviewers in parallel — one to pressure-test sequencing/dependency logic, one to pressure-test agent-implementability — absorbed their outputs, and refined the proposal. The pattern is observable in the index and corroborated by the v2 doc itself listing the same session under the inventory of the ev.shapes-assurance-import work.

v2 §3.5 describes the agent surfacing tensions on its own turn boundary in three loci. That captures *unsolicited* surfacing. Adversarial-subagent-review is a *solicited* surfacing mechanism: Harry explicitly asks the agent to spawn adversaries, the adversaries produce structured critique, the agent reconciles. Different mechanism, stronger signal, demonstrably productive. It is not the same as v2's §3.5 closing-observation cadence and should not be folded into it.

**Recommendation.** Add a new §3.7 or extend §3.5:

> **Adversarial review on demand.** When Harry requests adversarial review ("send subagents to pressure-test this"), the agent dispatches parallel reviewers with focused mandates (sequencing, implementability, hidden-assumption hunt, etc.), absorbs the outputs into the existing spec doc as targeted in-place edits, and reports what changed. The adversarial review is not a separate findings doc; it is a refinement pass on the living spec. This is a Harry-initiated pattern, not an agent default — the agent does not spawn adversaries spontaneously.

The v2 author should dump session `4d7542e6` in full before drafting §3.7 to verify the mechanism (parallel dispatch, focus areas, synthesis-back-into-spec) and check whether any specific subagent-instruction shape is worth preserving.

---

## 6. Finding — v2 has no stance on artifact location and lifecycle

v2 says "git is the history" (§3.3) and proposes commit trailers as the audit trail. Both presume the spec doc is under version control in a repository with a working `git log`. The audit found Harry's freeform spec artifacts in four different locations:

| Location | Examples | Lifecycle |
|---|---|---|
| `docs/` in a committed repo | `piano-play/docs/PRD.md`, ev.shapes design docs | Versioned. v2's trailer + PR ratification applies. |
| `.idea/` (JetBrains scratchpad) | `ev.shapes/.idea/Production-Readiness-Plan.md` | Sometimes committed, sometimes not. v2's trailer story partially applies. |
| `~/.claude/plans/` | `agile-nibbling-peacock.md`, `immutable-giggling-gem.md` | Per-user, ephemeral, never in git. v2's trailer story does not apply. |
| `/tmp/claude-*` | Various worker-generated issue files | Truly ephemeral. v2's trailer story does not apply. |

Most of v2's working pattern (Status: Draft / Ratified, PR-merge as ratification signal, `Spec-Diff-Classification:` trailer) is reachable only for the first row. v2 does not say which row ADD applies to, nor what happens when a spec begins in `.idea/` or `~/.claude/plans/` and migrates to `docs/`.

**Recommendation.** Add a short §3.8 or extend §5 with a scope statement:

> **Where ADD applies.** ADD's audit trail (commit trailers, PR-merge ratification) requires the spec doc to be under version control in a repo with a working `git log`. Specs that begin in `.idea/`, `~/.claude/plans/`, or `/tmp/` are pre-ADD drafting surfaces — they are not subject to the trailer discipline. ADD begins when the spec is committed to a tracked path in a real repo. The migration from drafting surface to tracked path is itself an event worth a `Spec-Diff-Classification: propagated-discovery` trailer on the introducing commit.

This is a smaller revision than §3 or §4 but worth doing — it pre-empts a class of "do trailers apply here?" questions that will otherwise consume future sessions.

---

## 7. Findings v2 has right — confirmed by the wider corpus

For completeness, the audit corroborates the following v2 claims against the new evidence:

- **§2.1 process vocabulary is absent in freeform spec sessions.** None of the new freeform sessions reach for IC#/Bucket A/seam-validation/attestation vocabulary. The pattern holds beyond the original five transcripts. (It does *not* hold for skill-driven sessions like `/spec-iterate` and `/draft-invariants`, but those are out of scope per §1 above.)
- **§2.2 artifact form is variable.** `piano-play/docs/PRD.md` and `ev.shapes/.idea/Production-Readiness-Plan.md` confirm the "common skeleton, variable structure" picture. Both use front-matter, numbered sections, risks/tradeoffs at the back, and avoid per-section process metadata.
- **§2.3 tensions surface on the agent's own turn boundary.** The new sessions show the three-loci pattern v2 describes.
- **§2.5 prompting idioms.** "Verify your claims, no conjecture" and "think about my intention" recur in the wider corpus. They are durable preferences.
- **§3.1 substantive design discoveries to keep.** No new evidence against any of them. The deterministic-only instrumentation, PR-merge ratification, M2-reuses-`/intent-check`, and skill-adaptation-list items survive the wider corpus.

These do not need revision.

---

## 8. Suggested order of operations for the v2 author

1. **Read this report end-to-end first.** Do not start patching v2 before reading the full audit.
2. **Dump session `4d7542e6-d0de-4431-aa5d-c857014811f8` in full** (e.g. `pickbrain --dump 4d7542e6-d0de-4431-aa5d-c857014811f8`) to verify the adversarial-review mechanism before drafting §3.7.
3. **Read `ngst/field-reports/crosscheck--2ff1cde8-3ff--2026-05-09.md` in full.** It is the single best document of the failure mode ADD is meant to solve. v2 currently does not cite it.
4. **Draft the §4.9 greenfield insert first.** It is the most consequential revision and may force re-thinking earlier sections.
5. **Re-revise §3.2** with the "one ID scheme that anchors outward" calibration. Preserve invariant IDs explicitly.
6. **Add §3.5a greenfield variant of the opening pattern.**
7. **Add §3.7 adversarial review on demand.**
8. **Add §3.8 or §5.0 stance on artifact location.**
9. **Re-read v2 §5.5 "what failure looks like"** and check whether any of the new findings change the list. (The current list catches the v1 ceremony stack; it does not catch a `/draft-invariants`-style elicitation firehose. Worth a new failure-mode bullet: *"The agent runs a fixed-cadence interview in greenfield instead of producing a short draft as the elicitation device."*)

---

## 9. Index of cited sessions and artifacts

For verification:

| Reference | Path or session ID | Repo | Date |
|---|---|---|---|
| ngst field report | `~/repos/ngst/field-reports/crosscheck--2ff1cde8-3ff--2026-05-09.md` | ngst | 2026-05-09 |
| Production readiness plan | `~/repos/ev.shapes/.idea/Production-Readiness-Plan.md` | ev.shapes | indexed 2026-05-09 |
| Piano PRD | `~/repos/piano-play/docs/PRD.md` | piano-play | indexed 2026-04-17 |
| SOTA gap implementation plan | `~/repos/xylem/docs/plans/sota-gap-implementation-2026-04-11.md` | xylem | 2026-04-11 |
| Agile-nibbling-peacock plan | `~/.claude/plans/agile-nibbling-peacock.md` | ev.shapes (per-user plan) | indexed 2026-05-09 |
| Immutable-giggling-gem plan | `~/.claude/plans/immutable-giggling-gem.md` | piano-play (per-user plan) | indexed 2026-04-17 |
| Adversarial subagent review session | `4d7542e6-d0de-4431-aa5d-c857014811f8` | formal-verify | 2026-05-05 |
| Invariant docs (sample) | `~/repos/ev.shapes/docs/invariants/secure-logger.md` (`I1`–`I3`) | ev.shapes | indexed 2026-05-09 |

---

## 10. What's deliberately not in this report

- A revised v2 draft. That is the v2 author's job. This report identifies what to change and points at the evidence; it does not pre-empt the revision.
- An assessment of the substantive design discoveries in v2 §3.1. Harry has stated those are sound and the audit found no counter-evidence.
- Field-report-style recommendations for `/draft-invariants` or other existing skills. Those belong in skill-specific audits.
- Multi-person-team scenarios. v2 §4.6 already defers these; the audit found no new evidence to change the deferral.

If anything in this report is unclear, dump the cited session and re-check before patching v2.
