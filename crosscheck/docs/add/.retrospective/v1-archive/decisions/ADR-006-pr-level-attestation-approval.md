# ADR-006: PR-Level Human Approval Gates Attestation Merges

**Status:** Attested (2026-05-09 by nicholls-inc; user policy declaration in agent-driven exchange after Phase 2 re-attestation cycle)
**Date:** 2026-05-09
**Consumes:** IC1, ADR-005
**Produces:** Amendment to M2/I1; new M5/F5.6 (PR-merge attestation gate)

## Context

Phase 2 of ADD's own development surfaced a tension between two desirable properties:

1. **Authorial discipline.** The intent doc's IC1 calls for human-authored Status promotions. The Drafted M2/I1 invariant operationalised this as "the agent never authors the attestation commit." If an agent could mint an `Attested` flip from inside a single skill invocation, the human-in-the-loop check that gates spec promotion would be illusory.

2. **Operational reality.** Both Phase 2 attestation events to date (`f7b69c0` for v1.0, `407d5c9` for v1.1) were agent-authored under explicit in-session human authorisation. The reason is mechanical: Claude Code's web UX does not surface a per-artifact attestation control to the human reviewer during an agent-driven exchange. Asking the human to context-switch into a terminal to author the commit themselves breaks the agent flow and defeats the purpose of the pairing.

The literal reading of M2/I1 forbids the operational pattern; the operational pattern is the only one that has actually worked. Continuing this contradiction silently would corrode the discipline. Either the discipline must change or the operational pattern must.

The forces in tension:

- **Authorship as identity (rejected).** Constraining who *types* the commit is a weak proxy for human review. A human running `git commit` themselves can still rubber-stamp. Worse, requiring it adds friction without adding signal.
- **Authorship as authorisation (chosen).** What matters is that a human approved the change in a reviewable medium. The PR is that medium: the diff is visible, comments are addressable, the approval is on-the-record, and the merge is gated by the platform.
- **Branch protection is the existing tool.** GitHub / GitLab / Bitbucket all support per-branch rules requiring review approvals before merge. The platform-level gate is harder to bypass than a pre-commit hook.
- **In-session approval still has value.** When the user types "I approve, make the attestation," that is the substantive human signal. The PR review is the formal record; the in-session signal is the working agreement that produces the PR.

## Decision

**Agent-authored attestation commits are permitted.** The discipline shifts from authorship-of-the-commit to approval-of-the-PR.

For any PR whose commit range includes one or more commits with `Spec-Diff-Classification: status-transition` *that touch an Attested-tier artifact* (currently: `docs/add/intent.md`, `docs/add/specs/architectural.md`, `docs/add/decisions/ADR-*.md`, `docs/add/methodology.md`, `docs/add/glossary.md`, `docs/add/acceptance.md`), the merge MUST be gated on:

1. **At least one approving review** posted by a GitHub identity present in `.assurance/audit-authors.allowlist`.
2. **The approving review post-dates** the most recent commit in the PR. Force-push or new commits invalidate prior approvals; re-approval is required.
3. **The approving reviewer is not the PR author.** Self-approval is not a review.

`status-transition` commits to Drafted artifacts (e.g., flipping a Drafted module spec into Drafted v1.1) do NOT trigger the gate. The gate is exclusively about Attested-tier promotions, retractions, and supersessions.

### Implementation surfaces

- **Branch protection rule** on the default branch requiring approving review for PRs that touch any Attested-tier path. The rule is configured per-repo in the host platform; the configuration is not enforceable from inside the repo, so an in-repo audit script is the secondary check.
- **CI check** (M5/F5.6, new) that runs on every PR event, walks the commit range, identifies `status-transition` commits touching Attested-tier paths, and verifies the approval criteria above. Fails the PR check if any criterion is unmet.
- **`.assurance/audit-authors.allowlist`** is the single source of truth for who can approve. The same file is consumed by the existing `docs/add/audit/` author check (per ADR-005 Authorship constraint).

### Why per-class enforcement, not per-path

ADR-005 established the five-class taxonomy (`propagated-discovery / intent-refinement / drift / retraction / status-transition`). The `status-transition` class isolates lifecycle events from content changes. Tying the PR-merge gate to that class means content-only commits to Attested artifacts (which should be `propagated-discovery` or `intent-refinement` and thus already trigger re-drafting → Drafted state) don't double-block. Only the lifecycle event itself — the act of promoting back to Attested — gates merge.

## Alternatives considered

### Require human-typed attestation commits (M2/I1 as drafted, rejected)

The agent would refuse to author the commit; the human would run `git commit` themselves. Pros: literal adherence to "human attestation." Cons: adds friction the human will route around (using their own scripts, copy-pasting the agent's diff into their terminal); does not actually verify approval — a human running `git commit -m "Attested"` in haste is no better than an agent doing it; the operational pattern shows the friction is real (both attestations to date have routed around it).

### Branch-protection only, no in-repo CI check (rejected)

Pros: simpler. Cons: branch-protection rules are platform-configured and not visible in the repo. A misconfiguration silently disables the gate. The CI check provides a redundant in-repo signal that catches branch-protection drift.

### Require multiple approvers (deferred)

Could require 2+ approving reviews. Pros: stronger discipline. Cons: not all teams have multiple humans available; ADD-in-Crosscheck is currently a single-human project. Decision: start with `>=1`; revisit if the project grows.

### Allow agent-authored attestations only when the commit body cites a specific user message (considered)

The commit body could be required to quote the human's authorising message verbatim. Pros: ties the commit to an audit-trail moment. Cons: the message text is in agent-controlled territory; the agent could fabricate a quote. The PR review step is the harder-to-fake check.

## Consequences

### Positive

- The actual operational pattern (agent commits + human PR review) is now the documented discipline. No silent contradiction.
- Branch protection is the gate, which is platform-enforced and reviewable.
- The `status-transition` class earns its keep — it specifically gates the high-stakes promotions, and `propagated-discovery` / `intent-refinement` content commits flow at the normal pace.
- Force-push invalidation closes a subtle loophole: a stale approval cannot be carried forward across a rewrite.

### Negative

- The PR step adds latency to attestation cycles. A human must explicitly approve the PR before merge; previously, the agent's local commit + push would have been sufficient (in practice this latency already existed because the user reviewed the work before authorising the attestation, but it is now made explicit).
- The branch-protection rule must be configured per-repo and is invisible to the in-repo discipline. Drift between the rule and the in-repo CI check is a new failure mode the M5/F5.6 implementation must address (cross-check the in-platform rule against an in-repo declaration).
- M2/I1 must be amended (cascade); M5 gains a new F section (cascade); see propagated-discovery commits accompanying this ADR.

### Neutral

- The existing `docs/add/audit/` Authorship constraint (ADR-005) is unchanged — the auditor agent is still the only legitimate writer of `docs/add/audit/`. ADR-006 is about Attested-tier *promotions*, which is a different surface.
- The methodology document does not need amendment; the Status field semantics are agent/human-agnostic and the new policy is an enforcement detail for the lifecycle event.

## Implementation cascade

1. **M2/I1** is amended to: "For every commit that flips Status of an Attested-tier artifact, the commit's PR carries an approving review by a human in `.assurance/audit-authors.allowlist`, posted after the latest commit in the PR, by an identity other than the PR author. The agent MAY author the commit; the PR review is the human signal."
2. **M5 gains F5.6** (`pr-merge-attestation-gate(pr_ref) → EnforceResult`) running in CI on PR events.
3. **New module invariant M5/I6**: PR-merge gate enforced for `status-transition` commits touching Attested-tier paths.
4. **No change to ADR-005** — the diff classification taxonomy is unchanged; this ADR adds enforcement *layered on top of* the classification, it does not redefine the classes.

## References

- M2/I1 (the original Drafted invariant being amended).
- M5/F5.4, F5.6 (the CI gate; F5.6 is the new addition).
- ADR-005 § Authorship constraint (the existing pattern this extends).
- The two attestation commits this ADR is grounded in: `f7b69c0` (Phase 2 v1.0 closure) and `407d5c9` (Phase 2 v1.1 re-attestation).
