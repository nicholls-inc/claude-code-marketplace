# Architectural Spec — ADD in Crosscheck

**Status:** Attested v1.0 (Phase 2 closure 2026-05-09 by nicholls-inc; comparison report § 9.4)
**Phase:** 1 (Specification — Architectural tier)
**Consumes:** IC1, IC2, IC3, IC4, IC5, IC6, IC7, IC8, IC9, IC10, IC11, ADR-001, ADR-002, ADR-003, ADR-004, ADR-005
**Produces:** the behavioral spec (`docs/add/specs/behavioral.md`, agent-authored) and the per-module functional specs (`docs/add/specs/modules/*.md`, agent-authored)

## Purpose

This spec names *what changes* in the Crosscheck plugin to support ADD. It is structurally architectural: it identifies the artifacts to add, the artifacts to modify, the data shapes that flow between them, and the linkage rules. It does *not* specify the prompts inside new skills or the exact Dafny patterns those skills emit — those are functional-spec-tier concerns for the agent to draft.

Sections are numbered hierarchically (`S1`, `S1.1`, `S2`, ...). Each section declares its consumed and produced artifacts.

---

## S1 — Operating-mode tagging

**Consumes:** IC5, IC9, IC11, ADR-001
**Produces:** behavioral spec section "B-modes"; constraints on every existing skill that touches governance; integrity rules covering B-tier linkage quality (S1.2)

### S1.1 — Mode tag location and format

Every module's invariant doc (`docs/invariants/<module>.md`) carries a frontmatter block at the top of the file:

```yaml
---
mode: add | bootstrap
phase: 0 | 1 | 2 | 3 | 4 | 5
attested-at: <commit SHA where mode was attested, or "draft">
---
```

The `phase:` field records the module's current ADD phase per `methodology.md` § Phase structure. Bootstrap-mode modules use `phase: 5` (continuous assurance) as the default since they enter Crosscheck post-construction; ADD-mode modules increment monotonically as the module advances. The integrity rules in S1.2 read this field.

For ADD-mode modules whose invariants live in `docs/add/specs/modules/<module>.md` (the dual location accommodates the source-of-truth difference between the two modes), the same frontmatter format applies in that file.

Modules without a frontmatter mode tag default to `bootstrap` for backwards compatibility (per IC9).

### S1.2 — Linkage-graph integrity rules per mode

The deterministic instrumentation (S4.1) checks linkage-graph integrity. Per ADR-001, the rules differ by mode:

- **Bootstrap-mode modules:** require `docs/invariants/<module>.md` exists, has at least one `I` invariant, and each `I` has a covering test. No requirement to trace back to an `IC`.
- **ADD-mode modules:** require `docs/add/specs/modules/<module>.md` exists, has at least one `F` functional spec section, each `F` traces via `consumes:` to an `S` architectural section, and each `S` traces to one or more `IC` claims. Tests are required once Phase 4 begins (the integrity check is mode-aware about which phase the module is in, recorded in the same frontmatter).
- **B-tier linkage quality (per IC11):** every `B` invariant in a module's behavioral spec traces via `consumes:` to at least one `IC` (possibly via an `S` intermediate) and via `produces:` to at least one `F` within its module. Violations are reported as:
  - `orphan-B` — `B` with no `IC` ancestor. Hard violation; integrity check fails.
  - `dangling-B` — `B` with no `F` descendant. Soft violation in Phase 1 (legitimate during derivation); hard violation from Phase 3 onwards.

Cross-module references use the qualified form `M3-billing/I3` per `glossary.md`.

---

## S2 — Greenfield skill set (new skills)

**Consumes:** IC1, IC2, IC3, IC4, ADR-004
**Produces:** four new SKILL.md files at `skills/<skill-name>/SKILL.md`

### S2.1 — `/intent-elicit`

**Trigger phrases:** "elicit intent", "capture vision", "phase zero", "draft intent doc", "start ADD project".

**Argument hint:** `[optional: path to a vision document or rough notes]`

**Owner:** Hellebuyck (specification chain)

**Inputs:** A natural-language vision from the user, either inline in the conversation, in an attached document, or in a draft text file in the repo. The skill is interactive and tolerant of starting from minimal input.

**Outputs:** `docs/add/intent.md` (or whatever path the user prefers; default is `docs/add/intent.md`) containing:
- Vision paragraph
- Numbered `IC` claims, each in ADR-style format (Context / Decision / Alternatives / Consequences) with explicit observable signals
- Out-of-scope list (numbered `N1`, `N2`, ...)
- Threat model (numbered `TM1`, `TM2`, ...)
- A Status field (Drafted on first emit; transitions to Attested only via human confirmation)

**Behavior contract:**
- Multi-turn conversational. The skill does not produce the full intent doc in one shot; it elicits, drafts, asks for amendment, iterates.
- Refuses to mark Status=Attested without explicit human confirmation in the same exchange.
- Each `IC` includes the observable-signal language so downstream Phase 2 can validate.
- Every commit it produces carries the `Spec-Diff-Classification` trailer (see S6.1) — typically `intent-refinement` for amendments, `propagated-discovery` for additions.

**Interactions:**
- Recommended next skill on completion: `/spec-derive`.
- May call `/intent-check-prose` (S2.3) on completion as a self-check before requesting attestation.

### S2.2 — `/spec-derive`

**Trigger phrases:** "derive spec", "draft architectural spec", "phase one architectural", "build spec from intent".

**Argument hint:** `[optional: path to attested intent doc, defaults to docs/add/intent.md]`

**Owner:** Hellebuyck

**Inputs:** An Attested intent doc.

**Outputs:** `docs/add/specs/architectural.md` containing:
- Numbered `S` sections, each with `consumes:` and `produces:` declarations
- Each section in ADR-style format (Context / Decision / Alternatives / Consequences)
- A "Coverage table" mapping every `IC` to the `S` sections that consume it; every `IC` must appear

**Behavior contract:**
- Refuses to complete (returns Drafted with a notice) if any `IC` from the intent doc is not consumed by at least one `S` section.
- Surfaces alternatives considered for each major architectural choice; the agent must produce at least one alternative per `S` section or explain why no alternative was viable.
- Conversational; does not produce the full spec in one shot.
- Emits the `Spec-Diff-Classification` trailer in commits (typically `propagated-discovery` for new sections, `intent-refinement` after intent amendment).
- Recommended next skill: `/intent-check-prose` for Phase 2 validation.

### S2.3 — `/intent-check-prose` (or `/intent-check --mode=prose`)

**Decision required for the agent:** implement as a new skill or as a mode of the existing `/intent-check`. Recommendation: a new skill, because the inputs and back-translation differ structurally enough that a mode flag would tangle the existing prompt template. The agent may overrule this if the implementation is genuinely cleaner as a mode; document the choice in a follow-up ADR.

**Trigger phrases:** "intent check prose", "phase two validation", "validate spec against intent".

**Argument hint:** `[optional: path to intent doc] [optional: path to architectural spec]`

**Owner:** Hellebuyck

**Inputs:** An Attested intent doc and a Drafted architectural spec.

**Outputs:** A structured report at `.assurance/intent-check-prose-report-<timestamp>.json` (and a Markdown rendering for humans) containing:
- For each `IC`: whether at least one `S` section consumes it, and whether the consuming section's substance plausibly satisfies the claim
- For each `S`: which `IC`s it consumes, and whether the back-translation by a fresh agent reading only the spec recovers a description plausibly matching the intent doc
- Gaps: `IC`s not consumed; `S` sections that consume no `IC`; substantive divergences between back-translation and intent

**Behavior contract:**
- Mirrors the structural-separation pattern of `/intent-check`: one agent (or one prompt) produces the back-translation blind to the intent doc; a second prompt compares.
- Does *not* require a covering test or code diff (this is the structural difference from `/intent-check`).
- Emits a JSON attestation; pre-commit hook may consume the attestation hash.
- Outputs go to `.assurance/intent-check-prose-fp-tracker.csv` for the FP rate computation. The 30% kill criterion applies, configurable per the existing `/intent-check` env-var pattern.
- **FP definition:** a False Positive is a flagged divergence between the back-translation and the intent doc that the human reviewer attests is spurious (e.g., wording difference but semantic equivalence). The human marks the flag spurious in the FP-tracker CSV; the rolling 30% threshold is computed over those judgments.

### S2.4 — `/spec-adversary-prose` (or `/spec-adversary --mode=prose`)

**Decision required for the agent:** as for S2.3 — new skill or mode of existing `/spec-adversary`. Recommendation: extend the existing skill with a mode flag, because adversarial probing is structurally similar across modes (the difference is the input artifact and the lack of a covering test).

**Trigger phrases:** "adversary prose", "probe spec", "phase two adversary", "find spec gaps".

**Argument hint:** `[optional: path to spec section to probe]`

**Owner:** Hellebuyck

**Inputs:** A Drafted spec section (`S` or `B` or `F`).

**Outputs:** A report enumerating:
- Behaviors the spec leaves unconstrained
- Edge cases the spec is silent on
- Questions the spec does not answer
- Candidate invariants or sections that, if added, would close the gap

**Behavior contract:**
- Same kill-criterion discipline as `/spec-adversary` (HIGH/MEDIUM/LOW confidence labels; signal-to-noise self-check).
- Does *not* propose changes to the spec; produces probing output the human or Hellebuyck can use to amend.
- Operates on Drafted specs; refuses to probe Ratified ones (those go through full consolidation, not adversarial probing).

### S2.5 — Seam to `/spec-iterate`

**Consumes:** IC3, ADR-004
**Produces:** integrity rule covering the spec ↦ implementation handoff

ADR-004 says the existing `/spec-iterate` flow is reachable from the spec stack — but the seam between a per-module functional spec section and a `/spec-iterate` invocation must be declared, not implicit, so the linkage-graph integrity check (S1.2) can verify it.

**Declaration form.** A functional spec section that is intended to be implemented via `/spec-iterate` records the intent in its frontmatter:

```yaml
---
id: F1.2
status: Drafted
consumes: [S2.1, IC3]
produces: [I1, T1.2]
implementation: spec-iterate
---
```

The `implementation:` field takes one of: `spec-iterate` (Layer-1 verified-Dafny chain), `manual` (agent or human writes code directly without `/spec-iterate`), `external` (the implementation lives outside this repo). Other values require a follow-up ADR.

**Integrity rule.** For any `F` section with `implementation: spec-iterate`, the integrity check (S1.2) requires that the corresponding module directory contain either:
- a `.dfy` artifact under the module's verification directory matching the `F` section's slug, *or*
- a Status note in the `F` section explaining why the artifact is not yet present (e.g., `implementation-status: deferred-to-phase-4`).

**What this section deliberately does not specify.**
- The internals of `/spec-iterate` itself (out of scope; existing skill).
- The exact Dafny patterns the agent emits when invoking `/spec-iterate` from an `F` section (functional-spec-tier concern).
- The behaviour when an `F` section's `implementation:` value is `manual` or `external` — those cases follow the standard integrity rule (test must cover invariant; no spec-iterate seam to verify).

---

## S3 — ADD-mode adaptations to existing skills

**Consumes:** IC1, IC9, ADR-001
**Produces:** modifications (additive only, never removing existing behavior) to a small number of existing SKILL.md files

For each existing skill listed below, the agent must produce a *delta spec* — a description of the smallest change required to make the skill mode-aware without breaking IC9.

### S3.1 — `/assurance-layer-audit`

**Change:** Detect empty-repo state (no manifests, no source files). On empty repo, produce a report explaining that this is an ADD-mode candidate and recommend `/intent-elicit` as the entry point. Existing behavior (manifest-driven layer projection) is unchanged for non-empty repos.

### S3.2 — `/assurance-init`

**Change:** Two additive responsibilities.

1. *Detect ADD-mode state* (`docs/add/` already exists with an Attested intent doc). On ADD-mode state, the "name 1-3 load-bearing modules" question is replaced with "name 1-3 architectural modules from `docs/add/specs/architectural.md`". Modules thus seeded inherit `mode: add` and `phase: <current-phase>` in their frontmatter.
2. *Detect surrounding governance frameworks* — the pre-commit framework in use (pre-commit.com, lefthook, husky, or none) and the CI system in use (GitHub Actions, GitLab CI, CircleCI, or none). Detection is conservative: presence of `.pre-commit-config.yaml`, `lefthook.yml`, or `.husky/` for pre-commit; presence of `.github/workflows/`, `.gitlab-ci.yml`, or `.circleci/config.yml` for CI. The detection result is recorded at `.assurance/governance-detection.json` and is consumed by S6.1 when installing the diff-classification gates.

Existing behavior (load-bearing-module elicitation, governance scaffolding) is unchanged for repos without `docs/add/` and is unchanged for the framework-detection step (the step is purely additive — its output sits alongside existing scaffolding rather than replacing any).

### S3.3 — `/intent-check`

**Change:** No behavioral change to existing `/intent-check`. The new `/intent-check-prose` (S2.3) is a sibling, not a replacement. The existing skill's documentation should reference the prose variant for users in Phase 2.

### S3.4 — `/spec-adversary`

**Change:** If implemented as a mode (per S2.4 architectural recommendation), add the `--mode=prose` flag and route to the prose variant when the input is a Drafted spec. Otherwise, no change; the new skill is sibling.

### S3.5 — `/acceptance-oracle-draft`

**Change:** Detect empty-source-tree state. On empty source tree but with an Attested intent doc, derive surfaces from the *intent doc's observable-signal language* rather than from file-tree scanning. Existing surface-detection behavior unchanged when source files exist.

---

## S4 — Deterministic instrumentation

**Consumes:** IC8, ADR-002
**Produces:** behavioral spec sections "B-instrumentation"; auditor agent input contract

### S4.1 — Instrumentation tool

A new tool, `scripts/add-instrumentation/` or `skills/add-instrumentation/SKILL.md` (the agent decides; if it has substantive prompt content, it is a skill; if it is purely deterministic, it is a script). The tool:

1. Reads git history over a configurable window (default 30 days; env var `CROSSCHECK_INSTRUMENTATION_WINDOW_DAYS`).
2. Computes the five signals enumerated in ADR-002:
   - Edit frequency per artifact
   - Change-coupling between artifact pairs
   - Linkage-graph integrity (orphans, dangling refs, cycles)
   - Cascade-pending detection
   - Diff-shape analysis
3. Emits structured output (JSON-lines preferred for ease of consumption) at `.assurance/add-instrumentation-<timestamp>.jsonl`.

The schema is documented in a `references/add-instrumentation-schema.md` file the agent authors as part of S4.1. Schema columns/keys, once committed, are stable.

The tool runs on demand (manually or via the auditor's workflow) and on schedule (CI cron). It does not invoke an LLM. It must be language-agnostic where possible — the Tornhill `xray-skill` precedent is recommended reading.

### S4.2 — Auditor input contract

The Auditor agent's prompt template (S5.1) takes the deterministic-instrumentation output as a structured input. The contract:

- The auditor's prompt receives the output verbatim, not summarised.
- Every verdict the auditor renders cites at least one signal ID from the instrumentation output.
- A verdict with no signal grounding is a malformed output and the agent must re-attempt.

---

## S5 — Auditor agent

**Consumes:** IC6, ADR-003
**Produces:** the auditor agent definition file and the consolidation-pass workflow doc

### S5.1 — Auditor agent definition

A new file `agents/<auditor-name>.md` (slug to be chosen by the agent and human, following the existing hockey-figure naming convention). The file structure mirrors `agents/byfuglien.md` and `agents/hellebuyck.md`:

- Scope statement: read-only access; consumes deterministic signals; produces verdicts.
- Skills owned (none in v1). The Auditor *invokes* `/add-instrumentation` and renders verdicts directly. If `/add-instrumentation` is implemented as a skill (per S4.1's script-vs-skill choice), the skill is plugin-level and *owned by Hellebuyck* (specification chain); the Auditor invokes but does not own it. This preserves the audit/author separation: the Auditor cannot modify the instrumentation it depends on.
- Tool restrictions: explicitly no write tools that allow modification of `docs/add/` (except the Auditor's own report directory `docs/add/audit/`, per ADR-005 § Boundary), `docs/invariants/`, `agents/`, `skills/`, `.claude/rules/`. The restriction is **declared in the Auditor's `agents/<auditor-name>.md` frontmatter** as a `tool-allowlist:` block enumerating the permitted tools (read tools, the `/add-instrumentation` invocation, and write tools constrained to `docs/add/audit/<date>.md` and its JSON sidecar). The plugin loader honours the frontmatter at agent-spawn time; agent runs with restricted tools cannot bypass via prompt injection because the constraint is harness-level, not prompt-level.
- Prompt template: a structured prompt that takes the deterministic output, walks the auditor through per-artifact verdict rendering, and emits the report.

### S5.2 — Consolidation-pass workflow

A new file `docs/examples/workflows/consolidation-pass.md` describing:

- When the pass runs (manual on-demand; scheduled weekly in early projects, monthly later).
- What inputs it requires (the instrumentation output, the linkage graph, the diff-classification log).
- What outputs it produces (a Markdown report at `docs/add/audit/<date>.md` plus a JSON sidecar for tooling).
- The verdict vocabulary (Settled / Active / Drifted) with examples of each.
- The remediation-proposal format for Drifted artifacts.
- The human adjudication workflow that follows.

The user's existing `assurance-squad.md` workflow at `ev-energy/ev.shapes` is a partial precedent the agent can reference; the v1 consolidation-pass workflow extends it with the settled/active/drifted distinction.

---

## S6 — Diff classification enforcement

**Consumes:** IC7, ADR-005
**Produces:** the pre-commit hook stub, the CI job stub, the diff-classification log schema

### S6.1 — Enforcement gates and log

Three artifacts:

1. **Pre-commit hook**, integrated with the existing pre-commit framework detected by `/assurance-init` (pre-commit.com, lefthook, or husky). The hook checks the structured commit-message trailer per ADR-005. It is fast (< 5s per Crosscheck's existing dual-track principle) and does not invoke an LLM.

2. **CI job**, added to the CI system detected by `/assurance-init` (GitHub Actions, GitLab CI, or CircleCI). The job re-verifies the trailer and appends to `.assurance/diff-classification-log.csv`.

3. **Log schema**, documented in `references/diff-classification-log-schema.md`. Format: **JSON-lines** at `.assurance/diff-classification-log.jsonl`. Each line is a JSON object with keys: `timestamp`, `commit_sha`, `author`, `classification`, `justification`, `modified_files` (array), `related_ids` (array, parsed from commit body — IC, S, B, F, I, T, ADR identifiers).

4. **Squash and rebase handling.** Squash-merging into a protected branch is permitted only when the squashed commit carries a summary `Spec-Diff-Classification` trailer covering the merged range. The CI gate runs on the *final* commit set on the merge target, not the pre-squash commits. Force-pushes that rewrite history on protected branches are rejected at the CI gate; the dual-track principle applies (a fast pre-receive hook on the remote, mirrored by the CI job).

Implementation should follow the precedent of `/invariant-coverage-scaffold` (which generates similar dual-track artifacts).

---

## S7 — Documentation surface updates

**Consumes:** IC10
**Produces:** edits to `README.md`, `docs/skills.md`, `docs/agents.md`, and possibly new files under `docs/`

### S7.1 — README updates

Add a new section "Operating modes" to the plugin README describing bootstrap and ADD modes as peers, with brief descriptions and pointers. Update the "Recommended order" section to distinguish bootstrap-mode order (existing) from ADD-mode order (new). The "Honest Map" recommendation from prior synthesis docs is realised: a one-line summary of layer reach, mode availability, and known limitations.

### S7.2 — Skill catalogue updates

`docs/skills.md` gains an "ADD mode" section listing the four greenfield skills (S2.1–S2.4) and noting the mode-aware adaptations to existing skills (S3.1–S3.5).

### S7.3 — Agent registry updates

`docs/agents.md` gains an entry for the Auditor agent.

### S7.4 — Methodology pointer

A short paragraph in the main README pointing readers at `docs/add/methodology.md` and stating ADD's hypothesis status. Honest about open problems per `methodology.md` § Open problems.

---

## S8 — Phase boundaries between human and agent work in this project

**Consumes:** the README's "What the human authored, what the agent authors" section
**Produces:** the boundary the agent honors

This section makes the boundary explicit at the architectural-spec level so the agent has no ambiguity:

- The human authors `methodology.md`, `glossary.md`, `intent.md`, this architectural spec, ADR-001 through ADR-005, and `acceptance.md`.
- The agent authors `docs/add/specs/behavioral.md`, the per-module functional specs, all SKILL.md files for new skills (S2), the delta specs for existing skills (S3), the deterministic-instrumentation tool (S4.1), the Auditor agent definition (S5.1), the consolidation-pass workflow doc (S5.2), the enforcement gates (S6.1), and the documentation updates (S7).
- Both human and agent author additional ADRs as new decisions arise; per `decisions/INDEX.md`, IDs are monotonic.

The agent does not modify any human-authored Ratified artifact. To propose a change, the agent files a supersession ADR per the `decisions/INDEX.md` procedure. While Status is Drafted, the human and agent collaborate openly.

---

## What this spec deliberately does not specify

- The wording of any prompt template inside any skill. Functional-spec-tier concern; the agent drafts.
- The exact Dafny patterns or the spec-iterate flow internals. Out of scope; existing.
- The Auditor's prompt language or the verdict-report Markdown format. Functional-spec-tier; the agent drafts within the constraints of S5.1 and S5.2.
- Implementation language for the deterministic instrumentation. The agent picks something appropriate (likely Python or a small Go binary) and documents the choice.
- A roadmap or timeline. Premature; the agent and human scope the work after Phase 2 validation.

## Coverage table — IC ↦ S

| IC | Consumed by | Notes |
|---|---|---|
| IC1 (Empty-repo entrypoint) | S2.1, S3.1 | `/intent-elicit` is the entry; layer-audit detects empty state |
| IC2 (Phase 0 explicit) | S2.1 | `/intent-elicit` is the producer |
| IC3 (Phase 1 derives from intent) | S2.2, S2.5 | `/spec-derive` produces the spec stack; S2.5 declares the seam to `/spec-iterate` |
| IC4 (Phase 2 prose-vs-prose) | S2.3, S2.4 | the two prose-mode skills |
| IC5 (Three operating modes) | S1.1, S1.2 | mode tag and per-mode integrity |
| IC6 (Auditor) | S5.1, S5.2 | agent definition and workflow |
| IC7 (Diff classification) | S6.1 | gates and log |
| IC8 (Deterministic instrumentation) | S4.1, S4.2 | tool and auditor input |
| IC9 (Existing flows unchanged) | S1.1, S3.* (additive only) | default-to-bootstrap in S1.1; each S3 adaptation is gated on mode/state |
| IC10 (Documentation surfaces ADD) | S7.* | README, skills.md, agents.md |
| IC11 (B-tier linkage quality) | S1.2 | `orphan-B` and `dangling-B` integrity rules; agent-authored behavioral.md will extend with module-level invariants |

Every `IC` is consumed by at least one `S`. (The Phase 2 validation in `acceptance.md` will check this mechanically.)
