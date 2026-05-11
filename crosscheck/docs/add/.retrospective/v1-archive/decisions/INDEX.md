# ADR Index

**Status:** Attested v1.1 (ADR-006 added 2026-05-09 by nicholls-inc; user policy declaration after Phase 2 re-attestation)
**Purpose:** Registry of architecture decisions for ADD-in-Crosscheck. Decisions are numbered monotonically and never deleted; supersession creates a new ADR with a back-reference.

ADRs follow the Nygard format: Title / Status / Context / Decision / Alternatives Considered / Consequences. ADRs at this layer encode decisions that constrain the architectural and behavioral specs; lower-level decisions (e.g., "use this prompt template") belong in skill SKILL.md files or in subsequent ADRs.

## Registry

| ID | Title | Status | Consumes | Produces |
|---|---|---|---|---|
| [ADR-001](./ADR-001-operating-modes.md) | Three operating modes (bootstrap / ADD / transitional) | Attested | IC5, IC9 | S1.1, S1.2 |
| [ADR-002](./ADR-002-deterministic-llm-split.md) | Deterministic-tools-detect-signals, LLMs-render-judgments split | Attested | IC8 | S4.1, S4.2 |
| [ADR-003](./ADR-003-auditor-agent.md) | Auditor as a third agent role | Attested | IC6, TM4 | S5.1, S5.2 |
| [ADR-004](./ADR-004-greenfield-skill-set.md) | Minimal greenfield skill set for v1 | Attested | IC1, IC2, IC3, IC4 | S2.1, S2.2, S2.3, S2.4 |
| [ADR-005](./ADR-005-diff-classification.md) | Mandatory diff classification on spec-changing commits | Attested | IC7, TM2 | S6.1 |
| [ADR-006](./ADR-006-pr-level-attestation-approval.md) | PR-level human approval gates attestation merges | Attested | IC1, ADR-005 | M2/I1 (amended), M5/F5.6 (new) |

## Adding a new ADR

1. Pick the next monotonic number (no reuse, even if a previous ADR was retracted).
2. Create `ADR-NNN-<slug>.md` with the standard format.
3. Update this INDEX with the new row.
4. Cross-reference from any spec section that consumes the new decision.
5. Status starts as Drafted; transitions per `glossary.md` § Status field.

## Superseding an ADR

1. Create a new ADR (next monotonic number) that supersedes the old one.
2. Mark the old ADR as `Superseded-by-ADR-NNN`.
3. Add a "Supersedes" reference at the top of the new ADR.
4. Update this INDEX: leave the old row in place with the new status, and add a row for the new ADR.

The old ADR is retained because, per the Nygard rule, "it's still relevant to know that it was the decision, but is no longer the decision."
