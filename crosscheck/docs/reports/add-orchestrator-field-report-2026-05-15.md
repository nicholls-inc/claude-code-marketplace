# add-orchestrator — Field Report

**Task:** Apply the ADD pipeline (`add-orchestrator`) to a multi-module backend feature that had produced four production bugs in succession with no behavioural contract.
**Date:** 2026-05-13 → 2026-05-15
**Outcome:** Pipeline ran end-to-end; the feature shipped; concrete orchestrator gaps surfaced. The actionable backlog lives at [docs/add/orchestrator-improvements.md](../add/orchestrator-improvements.md).

## Context

A Python web application feature — admin-mediated user-access approval with email-driven invitation handoff to OAuth signup — had shipped four production bugs in sequence. Each fix was narrow and correct; each unlocked the next latent bug. Working diagnosis: *the feature has no behavioural contract, only a set of weakly-coupled implementation surfaces.*

Two narrow fixes preceded the pivot:

- **Fix 1 — a Jinja2 silent-undefined.** A template referenced an attribute the underlying dict didn't carry; Jinja2's default `Undefined` rendered to an empty string and produced a malformed URL path. Five-line template fix. The reviewer flagged out-of-scope latent risk: the same pattern could recur on any of ~40 similarly-keyed tables.
- **Fix 2 — a two-arm CSRF wiring bug.** The URL fix in #1 finally let requests reach the handler, which then rejected them for missing CSRF tokens. The meta tag was absent from one admin template *and* the JS module lacked an `X-CSRF-Token` header. Two-file fix. The reviewer again flagged out-of-scope latent risk: a `getCsrfToken()` helper that silently returned `""` when the meta tag was missing, and a `email_sent=false` field never surfaced to admin.

The third bug arrived predictably: a swallowed `IntegrityError` in the invitation store. A row was being inserted with FK-violating values that PostgreSQL caught and SQLite (dev) hid because `PRAGMA foreign_keys` defaulted to `OFF`. *The shape of each bug matched the reviewer's latent-risk flag on the previous PR: side effects of letting a failure surface look like success.*

The pivot decision: don't fix the third bug narrowly. Write the spec, write the invariants, write the failing tests, then implement against them.

## What the pipeline produced

- Spec drafted (~30KB, RFC-2119) and signed off.
- Glossary + module map, content-hashed into a session marker so dispatched subagents could verify they were drafting against a consistent spec version.
- **7 modules of invariants drafted in one parallel turn** via `draft-invariants`: 62 invariants produced in ~3 minutes of wall-clock time.
- Three audit skills run in parallel: `audit-spec-coverage`, `audit-invariant-consistency`, an invariant-quality probe. ~5 minutes wall-clock.
- Two adversarial probes (`spec-adversary`) on the two highest-risk modules.
- 38 findings triaged 1:1: 29 fix-invariant, 6 amend-spec, 3 defer, 0 reject. Triage produced one retrofitted invariant and two adversarial-probe additions, bringing the total to 65.
- Failing regression tests committed *before* the implementation commit. The scaffolding (spec + invariants + failing tests) was a single commit.
- Implementation done in a separate session.
- Multi-agent review, post-merge findings closure, a `protected-surface-amend` invocation, and a latent-fixture remediation pass.

End state: 65 invariants in `docs/invariants/`, property tests citing them via `# Invariant <ID>:` comments, and a pre-commit bidirectional coverage gate enforcing both directions.

## What worked

**Parallel fan-out compressed agent-only time to ~15 minutes.** Drafting seven modules' worth of invariants plus three audits cost roughly 15 minutes of wall-clock time because the work was structurally parallel. This is the part of ADD that runs at machine speed.

**The audits caught spec holes the human author had missed.** Six of 38 findings were `accept-amend-spec` — the audit pipeline found real contradictions in the freshly-written spec. The most consequential: a glossary term contradicted between two rows; an invariant naming three tables when the actual transaction touched five. Without the audits, those holes would have shipped encoded into the invariant tests.

**Cross-cutting concerns surfaced that the narrow fixes had no path to discovering.** A dev/prod database-engine FK-parity concern was infrastructure, not feature code — the narrow PRs couldn't have touched it. ADD made it a named module with its own invariant doc and `Governance:` hook.

**The recovery property.** Mid-implementation, the agent destroyed roughly two hours of production-code work via an incorrect `git restore` loop that reverted source files back to master HEAD. The recovery was: commit the durable artifacts (spec, invariants, failing tests, per-file plan) that had already been written, hand off to a fresh session with a verbatim brief. **From disaster to working implementation: ~36 minutes.** Without the durable scaffolding, the cost would have been everything from the start of the ADD session. This is the strongest empirical argument the pipeline produced: the contract survived the conversation crashing.

**Prompt-shape change after scaffolding.** Pre-scaffolding user prompts were narrative and ambiguous (*"we keep finding bugs, this is frustrating, apply ADD"*). Post-scaffolding prompts were short and concrete (*"systematically address findings F1/F2/F3"*; *"diagnose failing tests on PR X"*; *"fix this CI error: [paste]"*) because the contract did the heavy lifting — the user no longer had to explain what right looks like. **The ADD investment paid back partly in agent-friendliness of future prompts.**

## What didn't work

**No checkpoint discipline.** The scaffolding commit happened only because the user manually committed it minutes before the catastrophic `git restore`. Had the bash error fired ~30 minutes earlier, the entire spec + invariants would have been lost. The orchestrator should commit the scaffolding before attempting implementation in the same session, every time, not as a happy-path artifact.

**Bidirectional coverage-gate retrofit was missing from the apply step.** New invariants were produced but pre-existing tests that should have claimed `# Invariant <ID>:` coverage weren't retrofitted. The gate was technically broken on merge and required a post-merge commit to fix. Partly a temporal artifact (the gate convention was younger than the audits in this run) but as the convention stabilises, retrofit must be in scope.

**No pre-flight for destructive invariants.** Closing one invariant required adding a SQLite `PRAGMA foreign_keys=ON` listener, which immediately surfaced ~170 latent FK violations across pre-existing test fixtures plus a real production seed-order bug elsewhere in the codebase. This is a *feature* of the methodology — the listener is what made the latent class visible — but it arrived as a mid-PR surprise that consumed ~70 minutes of remediation. A worktree-based dry-run that reports affected tests would have made this a planning input, not a merge-time crisis.

**Friction concentrated in the 38-finding human-triage step.** The parallel fan-out worked at machine speed. The triage step was where the user articulated the load-bearing feedback: *keep ADD in force, but auto-close findings where the action is mechanical (e.g. "add an invariant covering this missing endpoint") and only red-pen findings that need genuine spec judgement.*

**Adversarial-probe routing was retroactive.** The closing journal entry recommended which two of seven modules deserved `spec-adversary` probes — but only after the audit step had already chosen them implicitly. The recommendation should surface during the audit step.

**Post-merge findings.** Three findings landed *after* merge: the coverage-gate retrofit; one spec glossary inconsistency; one cross-cutting fix deferred to a separate PR. The audit passes ran on the invariant docs but missed the gate-retrofit work — the bidirectional gate convention was younger than the audits and not yet a check in scope.

## Honest costs

- **Total wall-clock human-in-loop time: ~5h** across ~28h, spread over 6 sessions of [~5.5h, ~36m, ~22m, ~70m, ~3m, ~8m].
- **Total agent-only work: ~2.5h** across the same sessions.
- **Lost work from the bash-error catastrophe: ~2h** of production-code effort, recovered in ~36 minutes via handoff.
- **Scope-discovery surprise: ~70 minutes** of fixture remediation arrived mid-PR after a destructive invariant landed.
- **Reviewer surface on the merged PR:** roughly 50 files, ~10K added / ~3K deleted lines. Multi-agent review flagged 4 items pre-merge, 3 post-merge.

The honest framing is that the *first* ADD pass on a feature is significantly more expensive than the narrow-fix path it replaces. Amortisation depends on whether the feature is the kind that produces recurring bug classes — when it is, the contract that survives the conversation is the load-bearing benefit. When it isn't, narrow fix is still correct. See [docs/add/orchestrator-improvements.md §6](../add/orchestrator-improvements.md) for the trigger criterion.

## The single load-bearing finding

Out of all of the above, one observation is the strongest empirical argument for the methodology and should drive orchestrator changes more than any other:

> **When the agent destroyed two hours of production-code work, recovery cost was a third of an hour because the contract was already on disk.**

Every orchestrator change that makes the contract durable earlier pays back in this property. Every change that defers the scaffolding commit makes the methodology brittle in exactly the failure mode that almost killed this session. Improvement target #1 in the backlog is the response.

## Limitations of this report

- The retrospective the report draws from was anonymised at the domain level: technologies are named, but the feature, modules, invariant IDs, PR/commit identifiers, and the source repository are not. Numeric specifics (finding counts, invariant counts, session durations) are exact; PR size and lost-work durations are rounded.
- The report covers one mature-repo field test. The v3 README's §5.3 listed two preconditions for writing a methodology doc — one mature-repo and one greenfield. This is the mature-repo half. The greenfield half is still pending.
- Subjective load (frustration intensity, context-switching cost across the 28h window) is inferred from durable artifacts and transcripts; the operator was not separately interviewed beyond their participation in the session.
