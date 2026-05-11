# Behavioral Spec — ADD in Crosscheck

**Status:** Drafted v0 (agent-authored Phase 1 cascade; awaiting human review)
**Phase:** 1 (Specification — Behavioral tier)
**Project:** Add Assurance Driven Development support to the Crosscheck plugin
**Consumes:** Attested architectural spec at `docs/add/specs/architectural.md` § S1–S8 plus IC1–IC11
**Produces:** per-module functional specs at `docs/add/specs/modules/<module>.md` (agent-authored next; not yet drafted)
**Last attested:** N/A (Drafted; awaiting human attestation per `README.md` lifecycle)

## Purpose and granularity

This file enumerates per-module behavioral invariants — the `B`-tier of the spec stack. Per the methodology, behavioral specs describe invariants that hold across reachable states of a module: TLA+/Alloy/P territory in principle. Per N4, v1 ships behavioral specs as **prose with cross-references**; no embedded model checker. IC11 (added in Phase 2) imposes the linkage-quality bar: every `B` traces via `consumes:` to at least one `IC` (possibly via an `S` intermediate) and via `produces:` to at least one `F`.

The `B` IDs are scoped per module per `glossary.md` § ID conventions. Cross-module references use the qualified `M<n>-<slug>/B<m>` form.

The granularity rule: a behavioral invariant is detailed enough when an agent reading it could draft a corresponding functional spec without asking clarifying questions. Several invariants below are intentionally tight; a few are intentionally broad and will likely fragment into 2–3 finer invariants once the functional specs are drafted. That is expected; revisit during the first consolidation pass per `acceptance.md` § A7.

## Module decomposition

Six modules are identified for v1, derived from the architectural spec's S sections:

| Module ID | Slug | Architectural sections | Rough scope |
|---|---|---|---|
| M1 | mode-governance | S1, S3.* | mode tagging, phase tracking, integrity rules, mode-aware skill behaviour |
| M2 | greenfield-skills | S2.1, S2.2, S2.3, S2.4, S2.5 | the four new skills plus the spec-iterate seam |
| M3 | add-instrumentation | S4.1, S4.2 | deterministic signal computation; auditor input contract |
| M4 | auditor | S5.1, S5.2 | auditor agent and consolidation-pass workflow |
| M5 | diff-classification | S6.1 | pre-commit hook, CI gate, log |
| M6 | documentation | S7.* | README, skills.md, agents.md, methodology pointer |

`M7-phase-boundaries` (S8) is a project-meta concern — agent/human work boundary — and is enforced by the linkage graph and attestation discipline rather than by per-module invariants. It is out of scope for behavioural-spec coverage.

---

## M1 — mode-governance

**Mode:** add
**Phase:** 1 (Specification — Behavioral tier)
**Consumes:** IC5, IC9, IC11, S1.1, S1.2, S3.1, S3.2, S3.3, S3.4, S3.5, ADR-001
**Produces:** F1.1, F1.2, F1.3, F1.4, F1.5

### B1 — Mode tag is monotonic per module

**Statement.** Once a module's frontmatter `mode:` field is set to a non-default value (i.e., explicitly `bootstrap` or `add` rather than the implicit default), it cannot change without a supersession ADR. In-place edits that flip mode are rejected by the integrity check.

**Consumes:** IC5, S1.1, ADR-001
**Produces:** F1.1 (functional spec: integrity check predicate `mode-tag-monotonic(module)`)

**Rationale.** Mode is a load-bearing property: it determines which governance rules apply. Silent reclassification of a module from bootstrap to ADD would change its expected attestation trail mid-life, voiding earlier integrity verdicts. Explicit supersession preserves the audit trail.

### B2 — Phase progression is monotonic forward, or terminal via supersession

**Statement.** A module's `phase:` field advances only forward (`0 → 1 → 2 → 3 → 4 → 5`) or terminates via supersession/retraction (per `glossary.md` § Status field). Backwards transitions on a module's phase are rejected by the integrity check unless accompanied by a re-drafting event triggered by an upstream change (per `glossary.md` § Status field).

**Consumes:** IC5, S1.1, S1.2
**Produces:** F1.2 (functional spec: integrity check predicate `phase-monotonic(module)`)

**Rationale.** Backwards phase moves indicate either (a) a discovered drift the upstream cascade should re-derive, or (b) an attestation-discipline failure. Making it impossible without explicit re-drafting forces the failure mode into the visible classification path.

### B3 — Default for an untagged module is bootstrap

**Statement.** A module whose invariant doc lacks an explicit `mode:` frontmatter field is treated by all governance-consulting skills as `mode: bootstrap, phase: 5`. This default must apply uniformly across `/assurance-layer-audit`, `/assurance-init`, `/intent-check`, `/spec-adversary`, `/acceptance-oracle-draft`, and the auditor agent.

**Consumes:** IC9, S1.1
**Produces:** F1.3 (functional spec: `default-mode-tag(module) = (bootstrap, 5)` and the read predicate `mode-of(module)` that returns the explicit tag if present, else the default)

**Rationale.** IC9 requires bootstrap-mode users to experience Crosscheck unchanged. Defaulting to `(bootstrap, 5)` for untagged modules is the mechanism that makes pre-ADD repos behave identically post-merge. A non-uniform default (e.g., one skill defaulting to `add` while another defaults to `bootstrap`) would silently break IC9 in subtle cases.

### B4 — Integrity check is mode-aware and phase-aware

**Statement.** For any module under integrity check, the rule set applied depends on `(mode, phase)`. Bootstrap-mode modules at any phase require `(invariant doc, covering test, code)` triples where applicable. ADD-mode modules at phase 0 require only intent.md presence; phase 1 adds architectural-spec coverage; phase 2 adds prose-vs-prose validation; phase 3 adds skeleton presence; phase 4 adds covering tests; phase 5 adds continuous assurance instrumentation. The integrity check does not falsely flag a phase-1 ADD-mode module for lacking a covering test (per ADR-001 alternative-A3 rejection).

**Consumes:** IC5, IC9, S1.2, ADR-001
**Produces:** F1.4 (functional spec: the rule-table `integrity-rules(mode, phase) → required-artifacts`)

**Rationale.** Without phase-awareness, the integrity check would either (a) be too strict (flagging legitimate phase-1 ADD modules) or (b) be too loose (letting phase-5 ADD modules escape requirements). The rule table makes the discipline explicit and auditable.

### B5 — Empty-repo detection is conservative

**Statement.** A repo is classified as "empty" by `/assurance-layer-audit` and `/acceptance-oracle-draft` only when no source manifests are present (per the layer-audit's standard manifest list) AND `docs/add/intent.md` is absent. The presence of `.gitignore`, `LICENSE`, top-level `README.md`, or CI config files alone does not flag the repo as non-empty for ADD-routing purposes.

**Consumes:** IC1, S3.1, S3.5
**Produces:** F1.5 (functional spec: predicate `is-empty-repo(workspace) → bool`)

**Rationale.** False-negative empty-repo detection routes the user away from `/intent-elicit` even when ADD-mode is the right answer. False-positive routes them away from existing-codebase flows. The conservative rule (require both manifest absence AND intent.md absence to classify as empty) errs on the safe side: a repo with `docs/add/intent.md` present already has an ADD entrypoint anyway.

---

## M2 — greenfield-skills

**Mode:** add
**Phase:** 1 (Specification — Behavioral tier)
**Consumes:** IC1, IC2, IC3, IC4, IC11, S2.1, S2.2, S2.3, S2.4, S2.5, ADR-004
**Produces:** F2.1.*, F2.2.*, F2.3.*, F2.4.*, F2.5.*

### B1 — `/intent-elicit` cannot mark Status=Attested without explicit human confirmation in the same exchange

**Statement.** The skill's prompt template includes a guard that refuses to write `Status: Attested` to the intent doc unless the human's last message in the same exchange contains an explicit attestation phrase. The agent does not auto-promote the artifact even when the agent itself believes the elicitation is complete.

**Consumes:** IC2, S2.1, TM5
**Produces:** F2.1.1 (functional spec: the attestation-guard predicate and the exact phrases recognised as explicit confirmation)

**Rationale.** TM5 names premature attestation cadence as a failure mode. If `/intent-elicit` could promote on its own judgment, the human attestation step would decay over time. The guard makes the human in the loop unbypassable.

### B2 — `/spec-derive` refuses to complete with any IC unconsumed

**Statement.** The skill's completion check inspects the produced architectural spec and rejects emission if any `IC` from the input intent doc is not consumed by at least one `S` section. The skill returns a Drafted spec with a notice listing the unconsumed `IC`s; the human or the agent then iterates.

**Consumes:** IC3, S2.2, IC11
**Produces:** F2.2.1 (functional spec: the completion-check predicate `every-IC-consumed(intent, spec) → bool`)

**Rationale.** An unconsumed `IC` is a structural hole — the architectural spec does not address part of what the human asked for. Allowing emission would let the cascade propagate the hole into behavioural and functional specs. Refusing emission forces the gap to be addressed before propagation.

### B3 — `/intent-check-prose`'s back-translation prompt is blind to the intent doc

**Statement.** The two-prompt structural separation pattern from `/intent-check` is preserved: one prompt produces the back-translation, taking only the spec stack as input; a second prompt compares back-translation to intent. The back-translation prompt does not have the intent doc in its context window.

**Consumes:** IC4, S2.3
**Produces:** F2.3.1 (functional spec: the prompt-isolation contract; predicate `back-translation-blindness(prompt) → bool` checked at skill compile-time)

**Rationale.** If the back-translator sees the intent doc, the comparison becomes self-confirming — the back-translator can simply reproduce intent verbatim. Structural blindness is what makes the prose-vs-prose check meaningful.

### B4 — FP rate over the rolling 30-attestation window stays below 30% (kill criterion)

**Statement.** `/intent-check-prose` maintains an FP-tracker CSV that records human adjudications of flagged divergences. When the rolling 30-attestation FP rate exceeds 30%, the skill enters a degraded mode that flags this fact prominently and refuses to ship a clean attestation until the threshold is restored. Per the FP definition committed in S2.3 (A-5): an FP is a flagged divergence the human reviewer attests is spurious.

**Consumes:** IC4, S2.3
**Produces:** F2.3.2 (functional spec: rolling-window computation; degraded-mode predicate; recovery condition)

**Rationale.** Mirrors the existing `/intent-check` discipline. Without a kill criterion, the prose-vs-prose check decays into noise; without an explicit FP definition, the threshold is ill-defined.

### B5 — Each greenfield-skill commit carries a `Spec-Diff-Classification` trailer

**Statement.** Every commit produced by `/intent-elicit`, `/spec-derive`, `/intent-check-prose`, or `/spec-adversary-prose` includes a valid `Spec-Diff-Classification` trailer (one of the five classes per ADR-005). The skill's commit-emission step refuses to commit without one.

**Consumes:** IC7, S2.1, S2.2, S2.3, S2.4, ADR-005
**Produces:** F2.*.5 (functional spec: the per-skill trailer-emission contract)

**Rationale.** The diff-classification gate (M5) enforces the trailer post-hoc. Having each skill emit the trailer pro-actively means the agent does not need to re-classify after the fact. This also ensures the agent's own work is held to the same discipline as the human's.

### B6 — `/spec-iterate` invocations declared in F-frontmatter are matched by an artifact or an explicit deferral

**Statement.** For any functional spec section with `implementation: spec-iterate` in its frontmatter, the integrity check (S1.2) requires either (a) a corresponding `.dfy` artifact in the module's verification directory, or (b) an `implementation-status: deferred-to-phase-<n>` note in the F section. Functional spec sections with no `implementation:` declaration are treated as `manual` and skip this check.

**Consumes:** IC3, S2.5
**Produces:** F2.5.1 (functional spec: the integrity predicate `spec-iterate-seam-honored(F-section) → bool`)

**Rationale.** S2.5 (added in Phase 2 amendment A-3) declares the seam between architectural specs and `/spec-iterate`. Without an integrity rule that checks the seam, agents can declare `implementation: spec-iterate` without ever invoking it.

---

## M3 — add-instrumentation

**Mode:** add
**Phase:** 1 (Specification — Behavioral tier)
**Consumes:** IC8, S4.1, S4.2, ADR-002
**Produces:** F3.1, F3.2, F3.3, F3.4

### B1 — The instrumentation tool does not invoke an LLM

**Statement.** The deterministic instrumentation tool is purely scripted (Python, Go, or shell — agent's choice per S4.1). It contains no calls to any LLM, no embedded prompt templates, no Anthropic/OpenAI/etc. SDK imports. The audit/separation principle from ADR-002 is enforced by absence of LLM dependencies, not just by convention.

**Consumes:** IC8, S4.1, ADR-002
**Produces:** F3.1 (functional spec: build-time predicate `tool-has-no-llm-dependencies(binary) → bool` checked in CI)

**Rationale.** ADR-002's split — deterministic tools detect signals; LLMs render judgments — collapses if the deterministic layer can call an LLM under the hood. A build-time check ensures the discipline survives later refactors.

### B2 — Same git state produces identical structured output

**Statement.** Two invocations of the instrumentation tool against the same git tree (same commit, same working tree, same configuration) produce byte-identical JSON-lines output, modulo the `timestamp` field of the run-metadata header. This is testable by running the tool twice and diffing output (excluding the timestamp line).

**Consumes:** IC8, S4.1
**Produces:** F3.2 (functional spec: determinism-check test; tolerance for timestamp-only diff)

**Rationale.** Auditor verdicts must be reproducible across runs (per ADR-002 § Forces in tension). Non-determinism in the deterministic layer would propagate to verdict variance.

### B3 — Schema additions are forward-compatible

**Statement.** A schema version increment that adds a new field (e.g., a sixth signal kind) does not break existing consumers of the JSONL output. The Auditor agent's prompt template treats unknown fields as additional context rather than as malformed input. The schema version is recorded in the run-metadata header.

**Consumes:** IC8, S4.1
**Produces:** F3.3 (functional spec: schema-version field; consumer-compatibility test cases)

**Rationale.** New signals are an extension point per ADR-002. Strict schema-version checking would force every consumer to upgrade in lockstep — unrealistic in practice.

### B4 — Every Auditor verdict cites at least one signal ID

**Statement.** The Auditor agent's per-artifact verdict in the consolidation-pass report includes at least one structured-signal ID from the instrumentation output (e.g., `signal:edit-frequency:docs/add/intent.md@30d=12`). The signal ID format is part of S4.1's stable schema. Verdicts without a citation are a malformed output and the Auditor is required to retry.

**Consumes:** IC8, S4.2, ADR-002
**Produces:** F3.4 (functional spec: signal-ID format; verdict-validation predicate `verdict-has-signal-citation(verdict) → bool`)

**Rationale.** ADR-002 makes the deterministic-then-LLM order load-bearing. If the Auditor's verdicts could be ungrounded, the LLM layer slips back into Path A (pure-LLM) territory. Mandatory citation keeps the discipline visible.

---

## M4 — auditor

**Mode:** add
**Phase:** 1 (Specification — Behavioral tier)
**Consumes:** IC6, S5.1, S5.2, ADR-003, TM4
**Produces:** F4.1, F4.2, F4.3, F4.4

### B1 — Auditor cannot write to protected paths (harness-enforced)

**Statement.** The Auditor agent's tool-allowlist (declared in `agents/<auditor>.md` frontmatter per S5.1) excludes any tool whose effect can write under `docs/add/` (except `docs/add/audit/`), `docs/invariants/`, `agents/`, `skills/`, or `.claude/rules/`. The plugin loader enforces the allowlist at agent-spawn time. A test case attempts a write to a protected path and asserts the operation fails with a clear error.

**Consumes:** IC6, S5.1, ADR-003, TM4
**Produces:** F4.1 (functional spec: allowlist enforcement contract; test case `auditor-write-rejected(path) → fails-with-error`)

**Rationale.** TM4 names auditor-authored artifacts as a threat to validity. Convention-only enforcement decays under pressure; harness-level enforcement does not.

### B2 — Auditor produces verdicts and remediation proposals; never executes remediation

**Statement.** The Auditor agent's output is a verdict report (Markdown) plus a JSON sidecar. For Drifted verdicts, the output includes a *proposed* remediation. The Auditor does not commit, edit, or otherwise apply the remediation; humans adjudicate; if approved, Byfuglien or Hellebuyck executes.

**Consumes:** IC6, S5.1, ADR-003
**Produces:** F4.2 (functional spec: output-format contract; absence-of-execution predicate)

**Rationale.** ADR-003's audit/author separation is the trust boundary. An Auditor that proposes-and-executes is just an authoring agent with a different name.

### B3 — Auditor's report directory is itself protected and append-only by the Auditor

**Statement.** Per ADR-005's authorship-constraint section, only the Auditor (and humans during adjudication) may write under `docs/add/audit/`. Authoring agents cannot. The pre-commit hook enforces this via the author-allowlist at `.assurance/audit-authors.allowlist`.

**Consumes:** IC6, ADR-003, ADR-005
**Produces:** F4.3 (functional spec: author-allowlist file format and the pre-commit predicate `author-permitted-for-path(author, path) → bool`)

**Rationale.** The Auditor's report is the audit trail. If authoring agents could write to it, the trail could be rewritten by the agent under audit.

### B4 — Auditor renders verdicts only on artifacts the deterministic layer flagged

**Statement.** The Auditor's per-pass scope is the set of artifacts identified by the deterministic instrumentation as carrying at least one signal in the configured window. The Auditor does not scan all artifacts; that work is the instrumentation tool's job. An Auditor pass over a repo with an empty signal set produces a verdict report stating "no signals; all artifacts considered Settled by absence-of-signal" — and explicitly distinguishes this from "no signals because the instrumentation did not run" (the latter is a malformed output and must retry).

**Consumes:** IC6, S5.1, S4.2, ADR-002, ADR-003
**Produces:** F4.4 (functional spec: scope-determination predicate; empty-set-handling rule)

**Rationale.** Without scope-determination, the Auditor either does the instrumentation's job in slow LLM tokens or silently skips drifted artifacts the instrumentation flagged. The empty-set rule exists because silent no-ops are the worst false-negative class — the user can't distinguish "all good" from "instrumentation broken."

---

## M5 — diff-classification

**Mode:** add
**Phase:** 1 (Specification — Behavioral tier)
**Consumes:** IC7, S6.1, ADR-005, TM2
**Produces:** F5.1, F5.2, F5.3, F5.4

### B1 — Every commit modifying protected paths carries a valid trailer

**Statement.** A commit whose diff touches any file under `docs/add/`, `docs/invariants/<module>.md` for ADD-mode modules, `agents/`, `skills/`, or `.claude/rules/` carries a `Spec-Diff-Classification` trailer with one of the five legal values. The pre-commit hook rejects commits without a valid trailer; the CI job double-checks on every PR.

**Consumes:** IC7, S6.1, ADR-005
**Produces:** F5.1 (functional spec: trailer parser; legal-values predicate; rejection-error format)

**Rationale.** TM2 — silent spec drift in early projects — is the failure mode this enforcement exists to rule out. Optional classification decays; mandatory enforcement survives.

### B2 — The classification log is append-only

**Statement.** `.assurance/diff-classification-log.jsonl` admits new lines only; existing lines are never deleted, edited, or reordered. The CI job rejects PRs whose diff includes deletions or modifications of pre-existing lines in the log. New lines may be appended only by the CI job itself or by a deliberate manual append (with its own trailer-classified commit).

**Consumes:** IC7, S6.1
**Produces:** F5.2 (functional spec: log-mutation predicate; CI-side enforcement)

**Rationale.** The log is the audit trail of classifications. If old entries can be edited, the trail rewrites itself under pressure — the classic drift-erasure failure mode.

### B3 — Squash-merges to protected branches require a summary trailer

**Statement.** Per the Phase 2 A-6 amendment to S6.1, when a feature branch is squash-merged to a protected branch, the squashed commit must carry a summary `Spec-Diff-Classification` trailer covering all classifications in the merged range. The CI gate verifies the squashed-commit trailer; pre-squash per-commit trailers are not re-validated post-squash.

**Consumes:** IC7, S6.1
**Produces:** F5.3 (functional spec: squash-detection predicate; summary-trailer format)

**Rationale.** A6 explicitly addresses trailer survival under squash. Without this rule, squash-merging into protected branches would silently strip per-commit classifications.

### B4 — The five classes are the only legal trailer values

**Statement.** A trailer with any value other than `propagated-discovery`, `intent-refinement`, `drift`, `retraction`, or `status-transition` is rejected as malformed. Adding a sixth class requires a supersession ADR that updates ADR-005, the methodology, the glossary, and the trailer parser in lockstep.

**Consumes:** IC7, ADR-005
**Produces:** F5.4 (functional spec: legal-values enumeration; supersession-required predicate)

**Rationale.** Per ADR-005 § Consequences, the legal values are fixed. New classes via prompt-only changes would let the taxonomy drift without the cascade.

---

## M6 — documentation

**Mode:** add
**Phase:** 1 (Specification — Behavioral tier)
**Consumes:** IC10, S7.1, S7.2, S7.3, S7.4
**Produces:** F6.1, F6.2, F6.3

### B1 — README distinguishes bootstrap-mode and ADD-mode entry points

**Statement.** The plugin README contains a section titled "Operating modes" that names bootstrap and ADD as peers. The "Recommended order" section is split into two subsections — one for bootstrap-mode (existing-codebase) and one for ADD-mode (greenfield) — with no merger that elides the difference. A reader scanning the README can identify which path applies to them within 60 seconds.

**Consumes:** IC10, S7.1
**Produces:** F6.1 (functional spec: README structural predicate `has-mode-section(README) ∧ has-split-recommended-order(README)`)

**Rationale.** IC10 names the documentation-honesty requirement. A merged "Recommended order" that tries to serve both audiences fails both.

### B2 — Open problems and hypothesis status are surfaced honestly

**Statement.** The README's ADD section explicitly states that ADD is a hypothesis (per `methodology.md` § Open problems) and lists the six open problems (or links to them). No copy implies ADD is evidence-backed.

**Consumes:** IC10, TM6, S7.4
**Produces:** F6.2 (functional spec: copy-audit predicate `has-hypothesis-disclaimer(README) ∧ links-to-open-problems(README)`)

**Rationale.** TM6 names hypothesis-status drift in documentation as a failure mode. Surface honesty is the mitigation.

### B3 — Skill catalogue and agent registry list all greenfield additions

**Statement.** `docs/skills.md` lists `/intent-elicit`, `/spec-derive`, `/intent-check-prose`, `/spec-adversary-prose` (and any spec-adversary mode flag if S2.4 is implemented as a mode). `docs/agents.md` lists the Auditor as a peer to Byfuglien and Hellebuyck. Both files are kept in sync with the actual `skills/` and `agents/` directories via a CI check.

**Consumes:** IC10, S7.2, S7.3
**Produces:** F6.3 (functional spec: catalogue-sync CI check; missing-entry detection)

**Rationale.** Catalogue drift is a known failure mode in plugin-style repos. The CI check makes the discipline self-enforcing.

---

## Coverage table — IC ↦ B

For Phase 2 A-9 traceability and IC11 compliance, every IC is consumed by at least one B (possibly via an S intermediate already declared in the architectural spec). This table makes the B-tier coverage explicit.

| IC | Consumed by (B-tier) | Notes |
|---|---|---|
| IC1 | M1/B5 | empty-repo detection predicate |
| IC2 | M2/B1 | `/intent-elicit` attestation guard |
| IC3 | M2/B2, M2/B6 | `/spec-derive` IC-coverage check; spec-iterate seam integrity |
| IC4 | M2/B3, M2/B4 | back-translation blindness; FP kill criterion |
| IC5 | M1/B1, M1/B2, M1/B4 | mode tag monotonicity; phase monotonicity; mode-aware integrity |
| IC6 | M4/B1, M4/B2, M4/B3, M4/B4 | auditor write-restriction, output shape, report protection, scope determination |
| IC7 | M5/B1, M5/B2, M5/B3, M5/B4 | trailer enforcement, log append-only, squash, legal values |
| IC8 | M3/B1, M3/B2, M3/B3, M3/B4 | no-LLM, determinism, schema forward-compat, signal citation |
| IC9 | M1/B3 | bootstrap default for untagged modules |
| IC10 | M6/B1, M6/B2, M6/B3 | README structure, honesty, catalogues |
| IC11 | (this file's existence + the linkage rule in S1.2) | every B in this file consumes ≥1 IC and produces ≥1 F placeholder |

Threat-model coverage (consumes lines reference TM2, TM4, TM5, TM6 directly; TM1 and TM3 are covered by structural design choices — small seed, additive S3 adaptations — declared at the architectural tier rather than as B invariants).

## What this spec deliberately does not specify

- The exact prompt strings for any skill (functional-spec-tier; agent drafts in `docs/add/specs/modules/<module>.md`).
- The Dafny patterns the agent emits when invoking `/spec-iterate` from a F section.
- The internal structure of `.assurance/diff-classification-log.jsonl` lines beyond the column list in S6.1 (functional-spec-tier).
- The Auditor's prompt language or the verdict-report Markdown layout (functional-spec-tier).
- Implementation language for the deterministic instrumentation (S4.1 leaves this to the agent).

## Open questions surfaced by this draft

These are flagged for the human to adjudicate, not unilaterally resolved by the agent. They are not currently amendments — just things the agent noticed while drafting and is unsure about.

1. **Module decomposition granularity.** Six modules feels right for v1 but `M2-greenfield-skills` lumps five distinct skills together. Splitting into five modules (one per skill) would push B IDs into per-skill scope and may improve traceability when functional specs are drafted. The downside is more cross-module references in this file. Worth your judgment before functional-spec drafting begins.
2. **`M7-phase-boundaries` exclusion.** The decision to leave S8 (human/agent boundary) without behavioural-spec coverage is defensible but means the boundary is enforced only at attestation discipline, not at integrity check. Worth confirming.
3. **B6 in M2 (spec-iterate seam).** The integrity rule sits between behavioural and architectural tiers. It could equally live in M1 (mode-governance integrity) or M2 (skill behavior). Placing it in M2 reflects that the seam is owned by skills that emit `implementation: spec-iterate` declarations; the alternative placement reflects that the integrity check is mode-aware. Both are defensible; M2 is the agent's choice for now.
4. **TM1 and TM3 explicit invariants.** The threat-model coverage note says these are covered structurally rather than via B invariants. If you want B-tier invariants for them — e.g., "B-doc-budget: docs/add/ does not exceed N artifacts in v1" or "B-noop-bootstrap: every existing test passes after merge" — flag and the agent will draft.
5. **Frontmatter format on this file.** `behavioral.md` itself is a Phase-1 ADD-mode artifact. Per S1.1 it should arguably carry `mode:` and `phase:` frontmatter. The header above uses prose rather than YAML; the agent can convert if you prefer YAML for consistency with module invariant docs.

These five questions are written in the report, not as inline TODO comments, per the project's no-comments-unless-non-obvious convention.
