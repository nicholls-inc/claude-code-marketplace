# ADR-003: Auditor as a Third Agent Role

**Status:** Attested (Phase 2 closure 2026-05-09 by nicholls-inc)
**Date:** 2026-05-09
**Consumes:** IC6, TM4
**Produces:** S5.1 (auditor agent definition), S5.2 (consolidation pass workflow)

## Context

ADD's continuous-assurance phase requires periodic consolidation passes that produce per-artifact verdicts. The natural candidate to run these is one of the existing orchestrator agents — Byfuglien (implementation chain) or Hellebuyck (specification chain). Either could plausibly be extended to render verdicts on artifacts within their domain.

The problem with that path: the agent that authored an artifact is the wrong agent to audit it. Hellebuyck is responsible for drafting and amending specs in the ADD-mode flow; if Hellebuyck also produces the consolidation verdict on the same specs, the audit becomes self-confirming. The auditor's incentive to flag drift is in tension with its role of having authored the spec under audit. This is a textbook conflict-of-interest pattern.

The forces in tension:

- **Audit and authoring must be separable for the verdict to be trustworthy.** TM4 names this directly as a threat to validity.
- **Adding agent roles costs cognitive overhead** for users learning the system. A third agent is a real complexity tax.
- **The auditor's work is genuinely different in shape from authoring.** It consumes deterministic signals (per ADR-002), produces verdicts not changes, and has no write authority on the artifacts it audits.
- **The auditor must be able to render judgments on artifacts produced by either Byfuglien or Hellebuyck**, so it cannot be subordinate to either.
- **The user's existing `assurance-squad.md` workflow is a partial precedent** but does not yet distinguish "settled" from "drifted" — exactly the gap the auditor closes.

## Decision

A third agent role, **Auditor**, is added as a peer to Byfuglien and Hellebuyck. The Auditor:

1. **Owns consolidation passes.** A scheduled or on-demand pass over all ADD artifacts in the repo, producing per-artifact verdicts of Settled / Active / Drifted.
2. **Has read-only access to artifacts.** It cannot author or modify any artifact. Its output is a verdict report, optionally accompanied by *proposed remediations* the human adjudicates.
3. **Consumes deterministic signals as primary input** (per ADR-002). Its prompt template includes the structured signal output verbatim and instructs the agent to ground every verdict in one or more cited signal IDs.
4. **Renders judgments only on artifacts the deterministic layer flagged.** The Auditor does not scan everything — that work is done by the instrumentation script. The Auditor's job is to convert "this artifact has signal X" into a natural-language judgment of severity and a proposed remediation.
5. **Honours operating-mode tags.** A bootstrap-mode module is not flagged Drifted for lacking Phase 0 attestation; an ADD-mode module is.
6. **Produces a written, repo-resident report at each pass.** The report is itself an ADD artifact, with a stable ID, Status, and links to the signal data and the artifacts it audited.

The Auditor is not subordinate to Byfuglien or Hellebuyck and does not consume their output directly. Its inputs are the artifacts under audit plus the deterministic signal output. Its outputs are the verdict report.

## Alternatives considered

**A1 — Extend Hellebuyck to run consolidation passes.** Rejected: violates audit/author separation. Hellebuyck owns spec authoring under ADD; auditing its own work is a direct conflict.

**A2 — Run consolidation passes "as the user" rather than as an agent.** Rejected: defeats the purpose of automation. The whole point of the auditor is to do work the user cannot afford to do at the cadence required.

**A3 — A skill rather than an agent.** Rejected as the primary form: the consolidation pass is multi-step (read signals → judge artifacts → propose remediations → write report), spans many artifacts, and benefits from the agent-orchestrator pattern. A *skill* invoked by the Auditor agent is fine; the agent role is the unit of trust.

**A4 — Hire a fourth role for "remediation."** Rejected as v1: the auditor *proposes* remediations as part of its verdict; humans adjudicate; if humans approve, Byfuglien or Hellebuyck *executes* the remediation. Adding a fourth agent for the proposal step is unwarranted complexity.

**A5 — Single shared "governance" agent that handles both authoring and auditing for any module.** Rejected: collapses the trust separation that motivated this ADR.

## Consequences

- The architectural spec must define the Auditor agent (`S5.1`) including its name, scope statement, prompt template, and tools available to it.
- The architectural spec must define the consolidation pass workflow (`S5.2`) including invocation points (manual, scheduled, on-cadence) and the report format.
- The Auditor must not have write tools in its toolset that allow modifying any artifact under `docs/add/`, `docs/invariants/`, `agents/`, `skills/`, or `.claude/rules/`. Its write authority is limited to its own report directory.
- Tooling that humans use to adjudicate Auditor verdicts (accept proposed remediation → route to Byfuglien/Hellebuyck for execution) is a separate workflow, not part of the Auditor's surface.
- Documentation must place the Auditor as a peer to the existing two agents; the agent table in the README is updated (per IC10).
- A naming follows the existing Crosscheck pattern (named after a hockey figure for continuity with Byfuglien and Hellebuyck) but the specific name is left open for the agent and human to choose. The slug `auditor` is the placeholder identifier in this directory.

## On naming

The existing two orchestrators are named after the Winnipeg Jets' Dustin Byfuglien and Connor Hellebuyck. Continuing the convention is desirable but not load-bearing on the architecture. The Auditor's role — review, not action — suggests a referee or a defensive specialist. The agent and human can choose; this ADR ratifies the role, not the name.

## Open questions deferred

- The exact format of the verdict report. Architectural-spec call (`S5.2`).
- Whether there is a separate "deep audit" mode (slower, more thorough) and a "fast audit" mode (deterministic-only, no LLM judgment) for use as a pre-commit or CI gate. v1 ships only the full pass; modes are an extension point.
- Whether the Auditor is permitted to flag *itself* (the report directory) for review by a human or by another Auditor pass. Self-review feels valuable but adds complexity; deferred to a later iteration.
