# `docs/add/` — Assurance Driven Development for Crosscheck

**Status:** Attested v1.0 (Phase 2 closure 2026-05-09 by nicholls-inc)

## What is this directory

This directory contains the seed artifacts for adding **Assurance Driven Development (ADD)** to the Crosscheck plugin. It is *itself* an ADD-mode project: ADD applied to its own integration into Crosscheck.

ADD is a methodology for AI-agent-executed software construction. Humans define the vision, govern the spec, and adjudicate ambiguity; agents perform construction within bounds the spec makes explicit. The full methodology is in `methodology.md`.

## Audience

Two audiences read this directory:

- **The Claude Code agent** building ADD into Crosscheck. The agent has only repo context plus web access; this directory is its primary brief.
- **Humans** governing the work — reviewing intent claims, attesting at phase boundaries, adjudicating drift.

Both audiences should start at `methodology.md`, then `glossary.md`, then `intent.md`.

## File map

```
docs/add/
├── README.md                           # this file
├── methodology.md                      # canonical ADD reference
├── glossary.md                         # ubiquitous language and ID conventions
├── intent.md                           # IC1...ICn — what success means for this project
├── acceptance.md                       # how we know v1 is done; Phase 2 protocol
├── specs/
│   └── architectural.md                # S1...Sn — what gets built/changed in Crosscheck
└── decisions/
    ├── INDEX.md                        # ADR registry
    ├── ADR-001-operating-modes.md
    ├── ADR-002-deterministic-llm-split.md
    ├── ADR-003-auditor-agent.md
    ├── ADR-004-greenfield-skill-set.md
    └── ADR-005-diff-classification.md
```

## What the human authored, what the agent authors

This directory at v1.0 contains only the layers humans uniquely own under ADD:

- The methodology reference itself (`methodology.md`)
- The ubiquitous language (`glossary.md`)
- Intent (`intent.md`)
- Architectural spec (`specs/architectural.md`)
- Five foundational ADRs (`decisions/`)
- Acceptance criteria (`acceptance.md`)

The agent will author, with human attestation:

- `specs/behavioral.md` — per-module behavioral invariants (the `B` tier)
- `specs/functional.md` (or one file per module under `specs/modules/`) — per-operation pre/post conditions (the `F` and `I` tiers)
- New SKILL.md files for skills introduced under `specs/architectural.md` §S2 (greenfield skill set)
- Modified SKILL.md files for existing skills extended under `specs/architectural.md` §S3 (ADD mode adaptations)
- The auditor agent prompt and SKILL surface
- Test stubs and the linkage-graph integrity script
- Subsequent ADRs as new decisions are made

Boundary rule: agents draft, humans attest. The agent should not modify any artifact in `docs/add/` whose Status is Ratified except by proposing a supersession ADR. Drafted artifacts authored by the human can be amended by the agent before attestation, but only with the human's explicit confirmation in the same exchange.

## Where this fits in the existing Crosscheck repo

Crosscheck pre-ADD operates exclusively in *bootstrap mode* (see `methodology.md` § Operating modes). Existing skills assume an existing codebase: `/assurance-layer-audit` scans manifests, `/assurance-init` asks for "load-bearing modules," `/intent-check` requires an `(invariant, covering-test, code-diff)` triple.

ADD adds the *clean-slate* operating mode and the *transitional* mode without removing bootstrap mode. Module boundaries carry an origin-mode tag; governance applied to each module is mode-appropriate.

The agent should read these existing Crosscheck artifacts before starting:

- `README.md` — plugin overview and recommended order
- `docs/research/assurance-hierarchy.md` and `docs/research/literature-review.md` — the assurance hierarchy
- `docs/research/logic-distribution-analysis.md` — empirical reach data for Layer 1
- `docs/skills.md` and `docs/agents.md` — current skill catalogue and agent ownership
- `agents/byfuglien.md`, `agents/hellebuyck.md` — orchestrator agent definitions
- `skills/assurance-layer-audit/SKILL.md`, `skills/assurance-init/SKILL.md`, `skills/intent-check/SKILL.md`, `skills/spec-adversary/SKILL.md`, `skills/acceptance-oracle-draft/SKILL.md`, `skills/spec-iterate/SKILL.md` — the skills most affected by ADD
- `docs/reports/crosscheck-field-report.md` — prior failure mode that motivated complexity gating

## Recommended order for the agent

1. **Read** `methodology.md` end-to-end. This is the canonical reference.
2. **Read** `glossary.md`. Internalise the ID conventions and Status field.
3. **Read** the ADR INDEX and all five ADRs in `decisions/`.
4. **Read** `intent.md`. Note: each `IC` is a numbered claim with explicit out-of-scope.
5. **Read** `specs/architectural.md`. Note its `consumes:` declarations against `IC` IDs.
6. **Read** `acceptance.md` and execute the **Phase 2 validation protocol** described there before drafting any further specs. Phase 2 is non-negotiable.
7. After Phase 2 attestation by the human, draft `specs/behavioral.md` and the per-module functional specs as scoped in the architectural spec.
8. After Phase 1 attestation, proceed to Phase 3 (skeleton + governance install) and Phase 4 (gated implementation) as defined in `methodology.md`.

## Recommended order for humans reviewing

1. `methodology.md` — confirm the methodology captures what we ratified through dialogue.
2. `intent.md` — confirm the intent claims match what you actually want from this project.
3. `decisions/` (five ADRs) — confirm the decisions and their rejected alternatives.
4. `specs/architectural.md` — confirm the proposed architectural changes to Crosscheck are right-sized.
5. `acceptance.md` — confirm the v1-done criteria are tight enough.

If anything is wrong, amend in place while Status is Drafted; once attested, propose a supersession ADR instead.

## Ratification of this directory

This directory's artifacts move from Drafted to Attested when the human signs off after agent Phase 2 validation has surfaced any gaps. They move from Attested to Ratified after at least one consolidation pass during Phase 4.

## Conversational origin

The artifacts in this directory are the synthesis of a multi-turn conversation between the human collaborator (nicholls-inc) and Claude. The conversation is not itself in the repo — per ADD's "repository is the source of truth" principle, the durable artifacts are these files. If the artifacts are wrong, the fix is to amend the artifacts, not to consult the conversation.
