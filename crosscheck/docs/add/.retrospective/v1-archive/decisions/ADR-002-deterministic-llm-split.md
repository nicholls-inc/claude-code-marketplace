# ADR-002: Deterministic Tools Detect Signals; LLMs Render Judgments

**Status:** Attested (Phase 2 closure 2026-05-09 by nicholls-inc)
**Date:** 2026-05-09
**Consumes:** IC8
**Produces:** S4.1 (instrumentation layer), S4.2 (auditor input contract)

## Context

The methodology requires periodic consolidation passes that produce per-artifact verdicts (Settled / Active / Drifted). Two implementation paths are available:

- **Path A — Pure LLM.** Hand the auditor agent the full repo and a prompt asking it to render verdicts. The agent decides what is settled, active, or drifted by reasoning about the artifacts directly.
- **Path B — Pure deterministic.** Define rules over git history and the linkage graph (e.g., "if this artifact has been edited >5 times in the last 30 days while its tests have been edited 0 times, it is Drifted"). No LLM in the loop.

Neither is sufficient alone. Path A is unreliable for *counting and graph integrity* — these are the things LLMs are reliably bad at, and the consolidation pass depends on them. Path B is unreliable for *judgment over natural language* — whether a prose intent matches a prose code description, whether a spec diff is propagated discovery or drift, whether an adversarially-generated invariant exposes a real gap. These are exactly the things LLMs are good at and deterministic tools cannot do at all.

The forces in tension:

- **Verdicts must be reproducible across runs.** Two consolidation passes with the same repo state should produce the same verdicts. Pure-LLM verdicts have non-trivial run-to-run variance.
- **Verdicts must reflect content meaning.** A Drifted verdict that turns out to be a false alarm (artifacts merely renamed) is expensive in attestation budget. Pure-deterministic verdicts cannot distinguish renames from drift, vacuous changes from material ones.
- **Costs must scale.** A pure-LLM consolidation pass over a large repo is expensive in tokens and time. A pure-deterministic pass is cheap but generates noise.
- **The split must be legible to humans adjudicating verdicts.** "The signal said X; the agent's judgment is Y; here is the trace" is reviewable. "The agent reasoned holistically" is not.

## Decision

The consolidation pass operates in two layers:

1. **Deterministic instrumentation.** A scripted layer (no LLM) computes structured signals from git history and the linkage graph. Signals include but are not limited to:
   - Edit frequency per artifact over a configurable window (Tornhill-style hotspot analysis applied to spec files).
   - Change-coupling between artifact pairs (spec edited, test not edited; or vice versa).
   - Linkage-graph integrity: orphans (no `consumes` or no `produces`), dangling references (`consumes: ICX` where `ICX` does not exist), cycle detection.
   - Cascade-pending: upstream artifact amended at commit C; downstream artifacts that consume it have not been re-attested since C.
   - Diff-shape: structural classification of recent diffs (new clause / modified clause / deleted clause).

   Output is structured (JSON or YAML) at a stable schema documented in `S4.1`. The script is the deterministic ground truth.

2. **LLM-mediated judgment.** The auditor agent consumes the deterministic output as primary input. Its job is to render natural-language verdicts on artifacts the deterministic layer flagged as suspect. It does not "discover" drift directly; it reasons about *whether the deterministic signal points to real drift*.

The order is fixed: deterministic first, LLM second. The auditor agent prompt template includes the deterministic output verbatim and instructs the agent to ground each verdict in one or more of the structured signals.

The deterministic layer is also consumed by humans directly. A human reading the consolidation report sees signals first and verdicts second.

## Alternatives considered

**A1 — Pure LLM.** Rejected for reasons above: unreliable counting, unreproducible verdicts, expensive at scale.

**A2 — Pure deterministic.** Rejected for reasons above: cannot distinguish meaningful from vacuous changes, cannot judge prose-vs-prose alignment, generates verdict noise that consumes attestation budget.

**A3 — LLM-first, with deterministic verification.** Rejected: the LLM still has to scan everything to decide what to verify, and the LLM's framing biases the verification step. Putting deterministic first means the LLM is constrained to reason about flagged items in the language of the signals.

**A4 — Two parallel passes (deterministic and LLM independently), reconciled at the end.** Rejected as a v1 choice: reconciliation logic adds complexity without clear payoff. May reconsider once we have field data on where each layer's verdicts diverge.

## Consequences

- The architectural spec must specify the deterministic instrumentation tool: where it lives, what schema it emits, when it runs (`S4.1`).
- The auditor agent's prompt must take the deterministic output as a structured input and reference signal IDs in its verdicts (`S4.2`).
- A small number of signal kinds is preferable to a large number, especially for v1. The instrumentation tool ships with the five signals listed above; new signals are added through a SKILL.md amendment, not a freeform prompt change.
- The instrumentation tool must be language-agnostic where possible (git, the linkage graph) and language-aware only where necessary (e.g., test-edit detection per language). The Tornhill `xray` skill is a relevant precedent — "Works on any git repository, regardless of programming language. Zero external dependencies beyond git."
- Future signals (e.g., complexity-trend on spec files, or coupling between mode-tagged modules) are extension points, not part of v1.

## Notes on prior art

The user's colleague's `xray-skill` (https://github.com/iledzka/xray-skill, based on Tornhill's *Software Design X-Rays*) is directly applicable to spec files rather than code files. Hotspot analysis on `docs/add/specs/` and `docs/invariants/` reveals which specs are churning. Coupling analysis across `docs/invariants/<module>.md` ↔ test files reveals when an invariant is drifting from its tests without being touched. The architectural spec (`S4.1`) recommends adapting these techniques rather than reimplementing.

## Open questions deferred

- Whether the deterministic layer is implemented as a new skill, a script, or a sidecar daemon. Architectural-spec call.
- The window default (30 days? 14 days?). Configurable in v1; defaults will be founder-intuition until field data exists, per the same discipline applied to the `/intent-check` 30% FP threshold.
