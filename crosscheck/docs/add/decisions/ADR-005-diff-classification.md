# ADR-005: Mandatory Diff Classification on Spec-Changing Commits

**Status:** Drafted
**Date:** 2026-05-09
**Consumes:** IC7, TM2
**Produces:** S6.1 (commit-time enforcement gate and audit log)

## Context

The methodology requires every spec-changing commit to carry one of four classifications: Propagated discovery / Intent refinement / Drift / Retraction. The classifications are not just documentation — they are the load-bearing mechanism that distinguishes healthy iteration (which is expected and valuable) from silent spec weakening (which is the failure mode ADD exists to prevent, per TM2).

Without enforcement at commit time, classification becomes optional and decays. The methodology depends on it being unavoidable.

The forces in tension:

- **Friction at commit time is friction users feel directly.** A heavy classification ritual will be skipped, gamed, or routed around.
- **Lack of enforcement collapses the value.** A classification log that captures only the diligent commits is useless because the silent drift commits never appear.
- **The classification must be on a *per-commit* basis**, not per-PR, because a single PR may contain mixed classifications (one commit refines intent, another fixes a typo). Per-commit granularity matches the unit of change.
- **Classification is a judgment call** — the agent or human deciding which of the four classes applies must reason about the change. We should not push that judgment onto the deterministic gate.
- **Auditing requires durable, queryable history.** A classification recorded only in a commit message is queryable only by `git log` parsing; one in a CSV log is queryable directly.

## Decision

A two-track enforcement, matching the dual-track principle Crosscheck already uses for invariant coverage:

### Pre-commit enforcement (fast)

A pre-commit hook detects whether the commit modifies any artifact under `docs/add/` (including `docs/add/audit/`, with the authorship constraint below), `docs/invariants/<module>.md` for ADD-mode modules (per ADR-001), `agents/`, `skills/`, or `.claude/rules/`. If yes, the hook requires:

1. The commit message contains a structured trailer:
   ```
   Spec-Diff-Classification: <propagated-discovery | intent-refinement | drift | retraction>
   Spec-Diff-Justification: <one-line summary; required for drift, optional otherwise>
   ```
2. For `drift` classification, the justification must answer the canonical question: *"did we want this behavior or did the implementation drift?"* The hook does not parse the answer for content but requires non-empty text.
3. The hook *does not* invoke an LLM. It is a fast attestation check only.

The hook fails the commit (with a clear, actionable error message) if these conditions are not met, in line with Crosscheck's existing dual-track principle that pre-commit hooks "must emit a human-readable error that includes the exact command to resolve the failure."

### CI enforcement (thorough)

A CI job runs on every PR:

1. Verifies the structured trailer is present on every commit that modifies a protected ADD/governance artifact.
2. Appends each classified commit to `.assurance/diff-classification-log.csv` (or equivalent) with columns: timestamp, commit SHA, author, classification, justification, modified files, related IDs (parsed from commit body).
3. Fails the build if any classification is malformed or missing.

The log is the durable artifact. It is consumed by:
- The Auditor agent during consolidation passes (per ADR-003): the agent reads the log to compute *Drift fraction over the last N days* as an instrument signal.
- Humans inspecting governance health.
- Future tooling that may compute long-term trends.

### Boundary

Classification applies to artifacts named above. It does *not* apply to:
- Application source code (this is implementation, not spec change).
- Documentation files outside the listed paths (e.g., README.md changes do not require classification unless they constitute a governance amendment per the protected-surfaces partition).
- Auto-generated artifacts (e.g., the diff-classification log itself, lockfiles).

### Authorship constraint on `docs/add/audit/`

The Auditor agent's report directory `docs/add/audit/` is a protected path *with an additional authorship rule*: only the Auditor agent (and humans, for adjudication-driven amendments) may write there. Authoring agents (Byfuglien, Hellebuyck) must not write to `docs/add/audit/` even when the trailer is correct, because the report is the audit trail of the auditor's verdicts and authoring-agent writes would compromise the audit/author separation that ADR-003 establishes.

Enforcement: Byfuglien and Hellebuyck have no tool-allowlist entries for writing under `docs/add/audit/` (see `agents/byfuglien.md`, `agents/hellebuyck.md` frontmatter). The pre-commit hook additionally rejects commits modifying `docs/add/audit/` whose author identity does not match the Auditor agent or a human reviewer (configured via `.assurance/audit-authors.allowlist`).

## Alternatives considered

**A1 — Classification only at PR level, not per commit.** Rejected: a PR with mixed-class commits hides drift inside otherwise-routine refactoring. Per-commit granularity preserves the signal.

**A2 — Classification recorded in PR description, not commit message.** Rejected: PR descriptions are mutable and not part of the git history that travels with the repo. Commit-message trailers are durable and parseable by tooling without GitHub access.

**A3 — Classification inferred by tooling rather than declared.** Rejected: the four classes encode *intent of the change*, which is not reliably inferable from the diff alone. A spec weakening that is genuinely Propagated Discovery looks identical in diff to one that is Drift; only the author/agent knows which.

**A4 — Optional classification with reminders.** Rejected: TM2 says drift becomes invisible if classification is optional. The whole point of mandatory enforcement is to make drift undeniable.

**A5 — Five or more classes (e.g., adding "Refinement-of-form" for typo fixes).** Rejected for v1: four classes are already at the edge of what authors can keep straight. Typo fixes do not modify protected artifacts in any meaningful way; the rule "classification applies only to material changes" handles this implicitly.

## Consequences

- The architectural spec must define the pre-commit hook and CI job (`S6.1`) and how they integrate with each supported pre-commit framework (pre-commit.com, lefthook, husky) and CI system (GitHub Actions, GitLab CI, CircleCI). The existing `/invariant-coverage-scaffold` skill is the closest precedent.
- The log schema must be stable enough that consolidation-pass tooling can rely on it. v1 of the schema is committed in the architectural spec; later additions can extend, but column meanings cannot change without an ADR.
- The four classes are the *only* legal values. New classes require a supersession ADR.
- The `/spec-derive` skill (ADR-004) and any other ADD-mode skill that produces spec-changing commits must emit the trailer in the commits they help author. The skill SKILL.md files include this requirement.
- The Auditor agent's consolidation-pass workflow consumes the log; the Auditor's verdict on a "Settled" artifact requires the artifact's edit history in the log to be free of unclassified entries within the consolidation window.

## On the canonical question

The phrase *"did we want this behavior or did the implementation drift?"* is the canonical drift-detection prompt the methodology adopts verbatim. It is repeated in the pre-commit hook error message, in the `/spec-derive` skill instructions, in the auditor agent's prompt, and in the methodology doc. Asking the same question consistently across surfaces reduces the chance the discipline decays under pressure.

## Open questions deferred

- ~~Whether the classification log is a CSV, a JSON-lines file, or something queryable like SQLite.~~ **Resolved (Phase 2 amendment A-13):** JSON-lines at `.assurance/diff-classification-log.jsonl`. Schema in S6.1.
- Whether older commits (pre-this-feature) get retroactively classified or are exempted. v1 default: exempt; the log starts at the feature's introduction.
- Whether the pre-commit hook enforces classification on Drafted-status artifacts (i.e., before they're attested). v1 default: yes — classification discipline begins from first commit, not from first attestation.
