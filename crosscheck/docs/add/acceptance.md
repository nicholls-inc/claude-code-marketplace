# Acceptance Criteria and Phase 2 Validation Protocol

**Status:** Drafted v1.0 (awaiting human attestation; Phase 2 protocol is the agent's first task)
**Consumes:** every IC (IC1–IC11) and every ADR (ADR-001 through ADR-005)

This document serves two functions: it describes what "v1 done" means for the ADD-in-Crosscheck project (acceptance criteria), and it specifies the Phase 2 self-validation protocol the agent runs *before* drafting any further specs or code.

---

## Phase 2 validation protocol (run this first)

The agent must complete this protocol before drafting `docs/add/specs/behavioral.md` or any per-module functional spec. Phase 2 is non-negotiable per the methodology. The protocol exists to surface gaps, contradictions, or under-specifications in the human-authored intent + architectural seed *before* they propagate into the agent's downstream work.

### Step 1 — Cold read

The agent reads, in order:
- `methodology.md` (full)
- `glossary.md` (full)
- `intent.md` (full)
- `decisions/INDEX.md` and all five ADRs
- `specs/architectural.md` (full)

The agent should *not* read any other Crosscheck files yet. The cold read tests whether the seed is self-contained.

### Step 2 — Back-translation (blind)

The agent writes a fresh document at `.assurance/phase-2-back-translation-<timestamp>.md` describing, in its own words and *without quoting* the input artifacts:

1. What system this directory is asking us to build.
2. What problem it is intended to solve.
3. Who the users are.
4. What the success criteria are.
5. What is explicitly out of scope.
6. What threats to validity the design rules out.
7. What the agent (Claude Code) is supposed to do, and what the human is supposed to do.

The agent does not consult `intent.md` or any other artifact while writing this. It writes from its understanding alone.

### Step 3 — Comparison

The agent then opens `intent.md` and `specs/architectural.md` side-by-side with its own back-translation. It produces a structured report at `.assurance/phase-2-comparison-<timestamp>.md` with the following sections:

- **Matches.** For each `IC`, what the back-translation says about the corresponding intent. State of agreement: full / partial / divergent / silent.
- **Gaps in intent.** Things the back-translation expected but the intent doc does not address. These may be missing intent claims or things the architectural spec assumes that intent does not state.
- **Gaps in spec.** Things the intent doc states that the architectural spec does not cover. Each unconsumed `IC` is a gap; surface it explicitly.
- **Contradictions.** Statements in the intent doc and the architectural spec that cannot both be true.
- **Under-specifications.** Architectural sections that are vague enough to admit incompatible implementations. The agent names the ambiguity and proposes which interpretation it would adopt absent guidance.
- **Out-of-scope drift candidates.** Things the back-translation thinks ought to be in scope but the negative-space (`N`) list excludes. These are candidate intent-refinement items the human may want to reconsider.

### Step 4 — Adversarial probing

Adversarially probe the spec stack for gaps the comparison missed. For each `S` section:

- What edge case does this section not address?
- What failure mode does this section silently assume away?
- What downstream artifact does this section need to produce that is not declared in `produces:`?

Surface the top 5–10 highest-leverage gaps (the SKILL pattern from `/spec-adversary`). Append to the comparison report.

### Step 5 — Verdict

The agent produces a Phase 2 verdict at the bottom of the comparison report:

- **PASS** — no contradictions, no IC unconsumed, no high-severity gaps. Ready for human attestation. The agent recommends transitioning intent.md, the ADRs, and architectural.md from Drafted to Attested.
- **PASS-WITH-AMENDMENTS** — small gaps or ambiguities exist but the spec is substantially correct. The agent proposes specific amendments the human reviews.
- **HOLD** — material gaps, contradictions, or under-specifications. The agent recommends amending before any Phase 1 lower-tier work begins.

### Step 6 — Hand off

The agent stops authoring. It does not draft `behavioral.md` or any module functional spec until the human:

1. Reads the comparison report.
2. Either attests Drafted → Attested (no amendments needed) or amends and re-attests.
3. Explicitly authorises the agent to proceed.

Phase 2 attestation is the first hard gate in the methodology being applied to itself.

---

## v1 acceptance criteria

These are the conditions under which v1 is considered complete. The Auditor agent (once running) will check these on subsequent consolidation passes.

### A1 — Greenfield empty-repo flow works end-to-end

A user with only a written vision and an empty git repository can:

1. Run `/intent-elicit` (S2.1) and produce a Drafted `docs/add/intent.md` through interactive elicitation.
2. Attest the intent doc (mark Status=Attested in a human-authored commit).
3. Run `/spec-derive` (S2.2) and produce a Drafted `docs/add/specs/architectural.md`.
4. Run `/intent-check-prose` (S2.3) and receive a Phase 2 report.
5. Attest the architectural spec.
6. Proceed to behavioral and functional spec authoring (out of v1 scope as a polished flow, but reachable).

The flow must work without the user invoking any non-ADD skill first. No "name your load-bearing modules" question appears.

### A2 — Existing bootstrap-mode flow is unchanged

On a representative existing codebase (e.g., the Crosscheck repo itself, treated as bootstrap-mode), the existing recommended order in the README still works:

- `/assurance-layer-audit` produces a sensible projection.
- `/assurance-init` creates the same governance scaffolding it did pre-this-work.
- `/intent-check`, `/invariant-coverage-scaffold`, `/spec-adversary` all behave as before.

Mechanical signal: every test in the repo that existed before the ADD work continues to pass. No existing skill SKILL.md file loses functionality (additions are permitted; subtractions are not).

### A3 — Mode tagging is enforceable

The deterministic linkage-graph integrity check (S4.1) reads frontmatter mode tags and applies the appropriate per-mode rules (S1.2). Manually crafted test cases:

- A bootstrap-mode module without an `IC` trace passes integrity.
- An ADD-mode module without an `IC` trace fails integrity.
- A mode-untagged module is treated as `mode: bootstrap` (default per S1.1).

### A4 — Diff classification is enforced

A commit that modifies any file under `docs/add/` without a `Spec-Diff-Classification` trailer is rejected by the pre-commit hook. The CI job appends classified commits to the log and fails the build for missing classifications.

Mechanical test: a deliberately malformed commit (touching `docs/add/intent.md` without trailer) is rejected. A correctly classified commit succeeds and appears in `.assurance/diff-classification-log.csv`.

### A5 — Auditor agent runs and produces verdicts

The auditor agent (S5.1) can be invoked. On a repo with the ADD seed in place, it:

1. Calls the deterministic instrumentation (S4.1).
2. Reads the structured signal output.
3. Produces a Markdown verdict report at `docs/add/audit/<date>.md` plus the JSON sidecar.
4. Each verdict cites at least one signal ID.
5. Does not modify any artifact in `docs/add/`, `docs/invariants/`, `agents/`, `skills/`, or `.claude/rules/`.

The agent's tool set must demonstrably exclude write access to those paths (test: a command that attempts a write fails with a clear error).

### A6 — Documentation surfaces ADD honestly

The plugin README contains an "Operating modes" section per S7.1. The recommended-order section distinguishes bootstrap-mode order from ADD-mode order. The methodology's hypothesis status is acknowledged.

`docs/skills.md` lists all four new greenfield skills (S2.1–S2.4) plus the auditor agent.

`docs/agents.md` lists the auditor as a peer to Byfuglien and Hellebuyck.

### A7 — The seed artifacts in this directory are themselves Ratified

Once v1 ships, `methodology.md`, `glossary.md`, `intent.md`, all five ADRs, the architectural spec, and this acceptance doc all transition from Drafted/Attested to Ratified through one consolidation pass. This is ADD eating its own dogfood: the methodology successfully audits itself.

### A8 — All Phase 2 PASS criteria from the agent's protocol report were satisfied

The agent's Phase 2 verdict must have been PASS or PASS-WITH-AMENDMENTS at the moment v1 ships. A HOLD verdict that was not addressed before shipping is grounds for delaying v1.

---

## What "Ratified" means for this directory

These seed artifacts move through three transitions:

1. **Drafted → Attested.** Triggered by human review after the agent's Phase 2 protocol completes. The human reads the agent's comparison report, addresses any HOLD or PASS-WITH-AMENDMENTS items, and explicitly attests in a commit. Pre-this-step, the artifacts are subject to amendment by anyone.

2. **Attested → Ratified.** Triggered by the first successful consolidation pass during Phase 4 (after some non-trivial usage of ADD-mode skills on a real project). The auditor verdict on each artifact is "Settled" and the human confirms.

3. **Ratified → Superseded-by-N.** Any future material change to a Ratified artifact requires a supersession ADR. In-place edits are not permitted on Ratified artifacts.

## The diff classification on commits creating this directory

Per ADR-005, every commit modifying `docs/add/` requires a classification. The initial commit creating this directory is classified as `propagated-discovery` (the entire directory is a new addition; nothing pre-existed to drift from).

Subsequent commits during the agent's Phase 2 work will be classified per the actual nature of each change.
