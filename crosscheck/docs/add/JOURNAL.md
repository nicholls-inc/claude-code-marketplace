# crosscheck/docs/add/JOURNAL.md

Journal for the assurance-driven development work specifically. This is the deepest shard in the current layout — it captures decisions about the methodology itself, the artifacts that support it, and the iteration cycles as they happen. Entries newest first. The retrospective at `.retrospective/findings-and-methodology-v2.md` is the long-form companion; entries here are short, link out.

---

## 2026-05-15 — first mature-repo field test of v3: pipeline shipped, six concrete improvement targets

**Type:** field-test
**Touches:** [field report](../reports/add-orchestrator-field-report-2026-05-15.md), [orchestrator improvements backlog](orchestrator-improvements.md)
**Why:** The v3 README said the shape was unvalidated until at least one mature-repo and one greenfield spec session had been driven against it. This entry records the first mature-repo run. The pipeline shipped the feature; six concrete orchestrator gaps surfaced and are now in the backlog.

The load-bearing finding is the **recovery property**. Mid-implementation, an agent destroyed ~2h of production-code work via an incorrect `git restore` loop, and recovery cost ~36 minutes because the spec + invariants + failing tests + per-file plan were already on disk. Every orchestrator change that makes the contract durable earlier pays back in this property; every change that defers it makes the methodology brittle in exactly that failure mode. Improvement target #1 (commit scaffolding before implementation) is the response.

The load-bearing operator feedback during the run is the second-most-important input: *keep ADD in force; automate more of finding triage where the action is mechanical.* The 38-finding triage step was where friction concentrated despite the parallel fan-out and parallel audits running at machine speed (~15 minutes total). Improvement target #2 (mechanical-vs-judgement finding classification) is the response.

One of two field tests the v3 README named as preconditions for writing a methodology doc. The other (greenfield) is still pending — until both have run, the v3 hypothesis stays a hypothesis.

---

## 2026-05-11 — v1 stack out, v3 starts here [ADR-0001]

**Type:** retraction
**Touches:** methodology.md, glossary.md, intent.md, acceptance.md, decisions/ (INDEX + 6 ADRs), specs/ (architectural + behavioral + M1–M6), phase-2-seam-validation report — all moved under `.retrospective/v1-archive/`. README rewritten. This journal started.
**Why:** v1 reproduced its own failure mode after one shipped iteration.
**Links:** [ADR-0001](../../../docs/decisions/0001-sharded-journal-architecture.md), [v2 retrospective](.retrospective/findings-and-methodology-v2.md)

This directory is now small on purpose. The README points readers at the retrospective and at this journal; the retrospective sketches a working hypothesis (sharded journals, a root walk-up rule, PR-merge as the human signal, a deterministic-only rule for any instrumentation tool, the five-class diff taxonomy moving from commit trailers to journal-entry types); nothing else lives here yet.

The honest state is that none of the v3 shape is validated. The retrospective's §5.3 names two spec sessions that have to happen before there's any case for writing a methodology doc — one mature-repo, one greenfield. Until both have been driven, treat the v3 hypothesis as exactly that, and read the §5.5 list in the retrospective for the symptoms that mean the form is wrong again.
