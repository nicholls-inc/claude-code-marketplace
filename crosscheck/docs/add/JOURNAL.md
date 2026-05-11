# crosscheck/docs/add/JOURNAL.md

Journal for the assurance-driven development work specifically. This is the deepest shard in the current layout — it captures decisions about the methodology itself, the artifacts that support it, and the iteration cycles as they happen. Entries newest first. The retrospective at `.retrospective/findings-and-methodology-v2.md` is the long-form companion; entries here are short, link out.

---

## 2026-05-11 — v1 stack out, v3 starts here [ADR-0001]

**Type:** retraction
**Touches:** methodology.md, glossary.md, intent.md, acceptance.md, decisions/ (INDEX + 6 ADRs), specs/ (architectural + behavioral + M1–M6), phase-2-seam-validation report — all moved under `.retrospective/v1-archive/`. README rewritten. This journal started.
**Why:** v1 reproduced its own failure mode after one shipped iteration.
**Links:** [ADR-0001](../../../docs/decisions/0001-sharded-journal-architecture.md), [v2 retrospective](.retrospective/findings-and-methodology-v2.md)

This directory is now small on purpose. The README points readers at the retrospective and at this journal; the retrospective sketches a working hypothesis (sharded journals, a root walk-up rule, PR-merge as the human signal, a deterministic-only rule for any instrumentation tool, the five-class diff taxonomy moving from commit trailers to journal-entry types); nothing else lives here yet.

The honest state is that none of the v3 shape is validated. The retrospective's §5.3 names two spec sessions that have to happen before there's any case for writing a methodology doc — one mature-repo, one greenfield. Until both have been driven, treat the v3 hypothesis as exactly that, and read the §5.5 list in the retrospective for the symptoms that mean the form is wrong again.
