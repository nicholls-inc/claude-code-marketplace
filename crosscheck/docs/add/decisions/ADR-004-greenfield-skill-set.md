# ADR-004: Minimal Greenfield Skill Set for v1

**Status:** Drafted
**Date:** 2026-05-09
**Consumes:** IC1, IC2, IC3, IC4
**Produces:** S2.1, S2.2, S2.3, S2.4 (per-skill specs in the architectural spec)

## Context

Crosscheck's existing skills assume an existing codebase. Five frictions arise when a user has only a written vision:

1. `/assurance-layer-audit` scans manifests and existing tooling; on an empty repo it produces empty diagnostics rather than recognising the empty state.
2. `/assurance-init` asks the user to "name 1-3 load-bearing modules"; with no modules yet, the question is unanswerable.
3. `/intent-check` requires an `(invariant prose, covering test, code diff)` triple; pre-code, none of these exist.
4. `/spec-adversary` requires a ratified invariant doc to probe; with no specs, there is nothing to probe.
5. `/acceptance-oracle-draft` instructs the agent to "detect surfaces" by scanning the file tree; an empty file tree yields nothing.

The recommended fix is *not* to add modes to every skill — that doubles the surface area of the catalogue and breaks the small-skill discipline Crosscheck values. The recommended fix is to add a small set of *new* skills that cover Phase 0 through Phase 2 of ADD, and to make minimal targeted adaptations to a small number of existing skills (covered separately under `S3` in the architectural spec).

The forces in tension:

- **Skill catalogue size has cognitive cost.** The current catalogue of ~20 skills is already cited as a complexity concern in prior synthesis docs.
- **Each existing skill that gains a "mode" branch becomes harder to reason about.** Mode-conditioned skills are a documented anti-pattern.
- **Phase 0 through Phase 2 work is genuinely different in shape from existing skills' work.** Eliciting intent from a vision, deriving a spec stack from intent, and validating prose-vs-prose are not slight variants of any existing skill.
- **v1 must ship a *minimum* viable greenfield path, not the full ADD catalogue.** Greater coverage is a Phase 4 concern after v1 lands.

## Decision

Four new skills constitute the minimum greenfield skill set for v1:

### S2.1 — `/intent-elicit`

Walks a user from a written vision to a Phase 0 intent doc. Produces `docs/add/intent.md` (or equivalent) with numbered `IC` claims, the explicit out-of-scope list, and the threat model. Each claim is ADR-formatted (Context / Decision / Alternatives / Consequences) and carries observable-signal language so downstream Phase 2 can validate.

The skill is conversational and cannot complete in one shot. It elicits, drafts, asks the user to confirm or amend, and iterates until the user attests. The output is a Drafted intent doc; the human's explicit confirmation in the final exchange is what moves it to Attested.

### S2.2 — `/spec-derive`

Takes an Attested intent doc as input and produces an architectural spec at `docs/add/specs/architectural.md` with `S` IDs, `consumes:` declarations against `IC` IDs, and `produces:` declarations for the lower spec tiers. The skill is structurally Phase 1; it does not produce behavioral or functional specs (those are derived in subsequent skill calls or, in v1, drafted by Hellebuyck directly from the architectural spec).

The skill is *not* a one-shot generator. It produces a draft, surfaces alternatives it considered (per the ADR-style discipline), and asks the user to confirm or amend. It refuses to complete if any `IC` is unconsumed — every intent claim must thread into at least one architectural section.

### S2.3 — `/intent-check-prose` (or extension of `/intent-check`)

Performs Phase 2 prose-vs-prose intent alignment. Takes an Attested intent doc and a Drafted spec stack as inputs; produces a structured report identifying gaps, missing intent claims, and spec sections that consume no intent. The skill structurally mirrors the existing `/intent-check` but with different inputs (no test, no code diff) and a different back-translation target (the spec, not the test).

The decision deferred to the architectural spec is whether this is implemented as a new skill (`/intent-check-prose`) or as a mode of the existing `/intent-check`. The latter has fewer skills in the catalogue but a more complex single skill; the former is cleaner but adds one more skill. The architectural spec resolves this in `S2.3`.

### S2.4 — `/spec-adversary-prose` (or extension of `/spec-adversary`)

Performs adversarial completeness probing on a Drafted spec without requiring a covering test. Takes a spec section as input; produces a structured list of behaviors the spec leaves unconstrained, edge cases the spec is silent on, and questions the spec does not answer. The skill mirrors the existing `/spec-adversary` but operates on prose specs in `docs/add/specs/` rather than on `docs/invariants/<module>.md` post-ratification.

Same architectural-spec decision as S2.3: new skill or extension of existing.

## What we are *not* shipping in v1

- **A `/spec-iterate-from-intent` chain.** The existing `/spec-iterate` flow (intent → Dafny spec → verified implementation → Python/Go) is reachable from the spec stack: an architectural section that consumes `IC4` may produce a functional spec that the user passes to `/spec-iterate`. We do not need a new skill that wraps this; the architectural spec just declares the seam.
- **A `/diff-classify` skill.** Diff classification (per ADR-005) is a commit-time discipline enforced by hook + CI, not an interactive skill. Humans and Byfuglien/Hellebuyck classify at commit; the Auditor verifies during consolidation. No standalone skill.
- **A `/consolidation-pass` skill.** This is the Auditor agent's primary work, defined under `S5.2`. It is not a skill; it is an agent workflow.

## Alternatives considered

**A1 — Add modes to existing skills (`/assurance-init --add-mode`, `/intent-check --prose-only`).** Rejected: doubles the cognitive surface of each touched skill, makes documentation more confusing, and tangles the skill's prompt template with conditional branches. Worse: it would not actually cover S2.1 (`/intent-elicit`) or S2.2 (`/spec-derive`) — those have no existing skill to mode-extend.

**A2 — Ship one mega-skill `/add-init` that does everything from intent to spec stack.** Rejected: violates the small-skill discipline and concentrates risk. Also forces sequential execution where iteration between phases is the natural pattern.

**A3 — Defer S2.3 and S2.4 to a future iteration.** Rejected: Phase 2 spec validation is a load-bearing piece of the methodology (per IC4), and adversarial probing of Drafted specs catches gaps cheaply. Skipping them in v1 leaves ADD without its self-validation step.

**A4 — Build all four as one skill family with shared infrastructure.** Acceptable internally (e.g., shared prompt fragments under `references/`), but not an organising principle. Each skill is a separate SKILL.md file with its own trigger phrases and arguments.

## Consequences

- The architectural spec must produce a SKILL.md spec for each of the four skills (`S2.1` through `S2.4`).
- For `S2.3` and `S2.4`, the architectural spec must decide between new skill and existing-skill extension. Both are acceptable; the decision is local to those sections.
- The agent and orchestrator ownership of these new skills must be specified. The natural placement is Hellebuyck (specification chain), since all four are spec-stack work. The architectural spec confirms this.
- New skills must follow existing Crosscheck SKILL.md conventions: trigger phrases, argument hints, step-numbered Instructions, Verification Checklist at end.
- The Quickstart in the README must include the ADD-mode entry path (likely starting at `/intent-elicit`); IC10 covers the documentation surface.

## Open questions deferred to architectural spec

- Whether `/intent-elicit` ships a worked example walking through a sample vision.
- The exact handoff format between `/spec-derive` and the human attestation step.
- Whether `/intent-check-prose` produces a JSON attestation analogous to `/intent-check`'s `.assurance/intent-check-attestation.json`, and how that integrates with the existing FP tracker.
- The kill criteria for `/intent-check-prose` — the existing 30% rolling FP threshold may not apply directly without test signal; the architectural spec works through this.
