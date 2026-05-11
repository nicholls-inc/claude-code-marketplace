# ADR-001: Three Operating Modes (Bootstrap / ADD / Transitional)

**Status:** Attested (Phase 2 closure 2026-05-09 by nicholls-inc)
**Date:** 2026-05-09
**Consumes:** IC5, IC9
**Produces:** S1.1, S1.2 (constraints on the architectural spec)

## Context

Crosscheck pre-this-work assumes a single operating mode: bootstrap. Skills detect existing manifests, scan for "load-bearing modules," require `(invariant, covering-test, code-diff)` triples, and treat the codebase as the source of truth from which intent is recovered. This works well when applying assurance to an existing system but produces friction or empty diagnostics when a user has only a written vision.

The natural alternative — flip Crosscheck to a spec-first / clean-slate mode — would either replace bootstrap mode (breaking existing users) or duplicate the entire skill catalogue under a new flag. Both are unattractive. A third option is to support multiple modes simultaneously, with module boundaries carrying mode metadata.

The forces in tension:

- **Existing users must not break.** Crosscheck has a real installed base operating in bootstrap mode. Their workflow continues to be legitimate and supported.
- **New users with only a vision must not hit the friction documented in prior assessments.** The recommended order in the README presupposing existing modules is the chief offender.
- **Mixed repos are the common case in practice.** A team adopting ADD on an existing project will retrofit governance over their old code (bootstrap) while writing new modules ADD-mode. Forcing the team to choose at repo level forces an uncomfortable choice.
- **Mode-conditional governance must not become a maze of branching logic.** Each mode's governance should be locally legible at the module it applies to.

## Decision

Crosscheck supports three operating modes. Modules carry a tag indicating their origin mode; governance applied to each module is mode-appropriate.

- **Bootstrap mode (`mode: bootstrap`).** Applied to existing code. Governance is recovered from code: invariants are extracted from behavior + tests, protected surfaces are retrofitted, intent is recovered post-hoc. The pre-this-work Crosscheck experience.
- **ADD mode (`mode: add`).** Applied to clean-slate work. Governance is prefigured: intent is captured first, specs derive top-down, code is gated against the spec stack. Phase structure is per `methodology.md`.
- **Transitional mode (`mode: transitional`).** Refers to repo-level state, not module-level. A repo containing a mix of bootstrap-mode and ADD-mode modules is in transitional mode. There is no `mode: transitional` tag on any individual module.

The mode tag lives in each module's invariant doc (`docs/invariants/<module>.md`) frontmatter, or in equivalent metadata for ADD-mode modules under `docs/add/specs/modules/<module>.md`. Modules without an explicit tag default to `bootstrap` for backwards compatibility — IC9 requires unchanged behavior for non-opting users.

Skills consulting governance must honor the mode tag. Examples:

- Consolidation passes do not flag a bootstrap-mode module as Drifted for lacking an intent-attestation trail.
- The diff classification rule applies to all `docs/add/` commits regardless of repo state, but applies to `docs/invariants/` commits only for ADD-mode modules.
- The empty-repo entrypoint (IC1) applies only when no modules exist; once modules exist, the entrypoint inspects mode tags.

## Alternatives considered

**A1 — Replace bootstrap mode with ADD mode.** Rejected: breaks IC9 (existing users unchanged), and bootstrap mode is the right answer for some real use cases (assurance retrofit on legacy code where intent is genuinely unrecoverable).

**A2 — Repo-level flag (`crosscheck.mode = add | bootstrap`).** Rejected: forces a single-mode commitment per repo, which is unrealistic for teams adopting ADD on an existing project. Also produces a discontinuity at the repo level rather than the module level, where governance actually applies.

**A3 — Implicit mode detection by skills.** Rejected: makes governance behavior depend on heuristics rather than declarations. A bootstrap-mode module that happens to have an invariant doc would be misclassified as ADD-mode. Worse, the heuristics drift over time.

**A4 — A separate plugin for ADD.** Rejected: separates the assurance hierarchy from its application, duplicates skill code, and prevents transitional mode entirely.

## Consequences

- The architectural spec must specify how mode tags are recorded, defaulted, and consulted (`S1.1`, `S1.2`).
- Each existing skill that consults governance must be audited for mode-awareness; the architectural spec enumerates which (`S3`, the ADD mode adaptations).
- Documentation must describe both modes and the transitional case (IC10).
- The auditor agent's verdict logic must take mode into account (ADR-003 will reference this).
- Future features that "make sense" only in ADD mode (e.g., Phase 2 prose-vs-prose validation) gate on the module's mode tag rather than being unconditionally available.

## Open questions deferred to later ADRs or specs

- The exact frontmatter format for mode tags (decided in S1.1 of architectural spec).
- Whether bootstrap-mode modules can be migrated to ADD mode in place, and what the governance-amendment ritual looks like for that migration. Deferred until at least one such migration is attempted.
