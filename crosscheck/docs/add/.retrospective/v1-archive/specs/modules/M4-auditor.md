# M4-auditor — Functional Spec

```yaml
---
id: M4-auditor
mode: add
phase: 1
status: Drafted
consumes: [IC6, S5.1, S5.2, ADR-002, ADR-003, ADR-005, TM4, M4-auditor/B1, M4-auditor/B2, M4-auditor/B3, M4-auditor/B4]
produces: [F4.1, F4.2, F4.3, F4.4, F4.5, I1, I2, I3, I4, T4.1..T4.10]
last-attested: N/A (Drafted)
---
```

## Purpose

Per-operation functional specs for the **auditor** module: the third agent role (peer to Byfuglien and Hellebuyck), its read-only access enforcement, the consolidation-pass workflow, and the verdict-report shape. The Auditor consumes M3's deterministic signals and renders natural-language judgments. It does not author or modify the artifacts it audits.

The Auditor agent definition lives at `agents/<auditor-name>.md`. The slug `auditor` is a placeholder until a hockey-figure-named decision is made (per ADR-003 § On naming). This functional spec is independent of the chosen name.

---

## Data shapes

### `Verdict`

```
Verdict := Settled | Active | Drifted | ActiveWithWarning

VerdictRecord := {
  verdict_id: String matching ^V[0-9]{8}T[0-9]{6}Z-[0-9]+$,    // e.g., V20260509T143830Z-0001
  artifact: ArtifactRef,
  verdict: Verdict,
  rationale: String,
  cited_signals: NonEmptyList<SignalId>,           // per I1; at least one
  proposed_remediation: Option<RemediationProposal>,
  pass_id: PassId
}

PassId := String matching ^pass-[0-9]{4}-[0-9]{2}-[0-9]{2}-[0-9]{4}$    // e.g., pass-2026-05-09-1438
```

### `RemediationProposal`

```
RemediationProposal := {
  description: String,
  routed_to: { Byfuglien, Hellebuyck, Human },     // who would execute if approved
  estimated_diff_classification: ClassificationClass    // hint for the eventual implementer
}
```

### `ConsolidationPassReport`

```
ConsolidationPassReport := {
  pass_id: PassId,
  invoked_at: Timestamp,
  invoked_by: AgentRef,
  instrumentation_run_metadata: RunMetadata,    // from M3
  verdicts: List<VerdictRecord>,
  empty_signal_set: Boolean,    // see F4.4
  json_sidecar_path: AbsolutePath    // .json sidecar for tooling consumption
}
```

### `AuditorAgentDefinition`

```
AuditorAgentDefinition := {
  name: String,
  scope_statement: String,
  tool_allowlist: NonEmptyList<ToolSpec>,    // per I3
  prompt_template: PromptTemplateRef,
  skills_owned: List = []                     // empty in v1; see ADR-003 / S5.1
}

ToolSpec :=
  | ReadTool { paths: List<PathGlob> }
  | InvokeSkill { skill_id: String }
  | WriteTool { paths: List<PathGlob> }    // restricted to docs/add/audit/ for the auditor
```

---

## F4.1 — `auditor-render-verdict(artifact, signals, deterministic_output) → VerdictRecord`

```yaml
---
id: F4.1
status: Drafted
implementation: manual
consumes: [M4-auditor/B2, M4-auditor/B4, IC6, S5.1, S4.2]
produces: [I1, T4.1, T4.2]
---
```

### Signature
`auditor-render-verdict(artifact: ArtifactRef, signals: NonEmptyList<Signal>, deterministic_output: InstrumentationOutput) → VerdictRecord`

### Preconditions
- `signals` is non-empty (the Auditor only renders verdicts on artifacts the deterministic layer flagged; per F4.4).
- All signals reference `artifact`.
- `deterministic_output.git_commit` matches the workspace HEAD at the time of the Auditor's invocation.

### Postconditions

Returns a `VerdictRecord` where:
- `cited_signals` is non-empty and a subset of `signals` (the LLM's judgment; not all signals must be cited, but at least one must be).
- `verdict` is one of `Settled | Active | Drifted | ActiveWithWarning`.
- `rationale` is a natural-language explanation grounded in the cited signals.
- For `Drifted` verdicts, `proposed_remediation` is non-empty; for other verdicts, optional.

A `VerdictRecord` with `cited_signals = []` is malformed and the Auditor MUST retry. Per ADR-002, ungrounded LLM judgment is the failure mode this rule rules out.

### Frame conditions
- Read-only on the artifact and the deterministic output.
- The Auditor's only write operation is appending to its own report directory (`docs/add/audit/`); F4.1 does not write — it produces a `VerdictRecord` that F4.3 will collect into the report.

### Module invariants preserved
- I1 (every verdict cites ≥1 signal).

### Test linkage
- T4.1 — invoke with one EditFrequency signal; verdict cites it; verdict.rationale references the count.
- T4.2 — invoke with empty signal list; F4.1 raises a precondition violation (Auditor's caller is responsible for not invoking with empty signals; the F4.4 scope rule normally prevents this).

---

## F4.2 — `auditor-propose-remediation(verdict: VerdictRecord) → RemediationProposal`

```yaml
---
id: F4.2
status: Drafted
implementation: manual
consumes: [M4-auditor/B2, IC6, S5.1, ADR-003]
produces: [I2, T4.3, T4.4]
---
```

### Signature
`auditor-propose-remediation(verdict: VerdictRecord) → RemediationProposal`

### Preconditions
- `verdict.verdict == Drifted` (proposals are only generated for Drifted artifacts; Settled/Active/ActiveWithWarning carry no proposal).

### Postconditions

Returns a `RemediationProposal` with:
- `description`: a one-paragraph natural-language proposal grounded in the cited signals.
- `routed_to`: one of `Byfuglien | Hellebuyck | Human` based on the artifact kind:
  - Spec stack changes (intent.md, ADRs, architectural.md, behavioural.md, per-module functional specs) route to **Hellebuyck**.
  - Implementation/test changes route to **Byfuglien**.
  - Status flips, attestations, governance amendments require **Human** adjudication.
- `estimated_diff_classification`: the Auditor's best guess at which of the five classes the eventual amendment would carry.

The Auditor does **not** execute the remediation. Per ADR-003 alternative-A4 rejection, proposing-and-executing collapses the trust separation.

### Frame conditions
- Read-only on `verdict` and the artifact's history.

### Module invariants preserved
- I2 (Auditor proposes; never executes).

### Test linkage
- T4.3 — Drifted verdict on a spec section; proposal.routed_to == Hellebuyck.
- T4.4 — Drifted verdict on test code; proposal.routed_to == Byfuglien.

---

## F4.3 — `consolidation-pass-run(workspace, deterministic_output) → ConsolidationPassReport`

```yaml
---
id: F4.3
status: Drafted
implementation: manual
consumes: [M4-auditor/B2, M4-auditor/B4, IC6, S5.2]
produces: [I2, T4.5, T4.6, T4.7]
---
```

### Signature
`consolidation-pass-run(workspace, deterministic_output: InstrumentationOutput) → ConsolidationPassReport`

### Preconditions
- `deterministic_output` was produced by M3's instrumentation tool against the same workspace within the last 24 hours.
- The Auditor agent's `tool_allowlist` does not contain write tools targeting paths outside `docs/add/audit/`.

### Postconditions

For each artifact identified in `deterministic_output.signals` (grouped by `artifact_ref`):
- Invoke `auditor-render-verdict` (F4.1) to produce a `VerdictRecord`.
- For Drifted records, invoke `auditor-propose-remediation` (F4.2).
- Append the record to `report.verdicts`.

If `deterministic_output.signals == []`:
- Set `report.empty_signal_set = true`.
- Emit a single notice in the report explaining "no signals; all artifacts considered Settled by absence-of-signal" — distinguished from "instrumentation did not run."

The report is written to:
- `docs/add/audit/<pass_id>.md` — Markdown rendering for humans.
- `<json_sidecar_path>` — JSON sidecar for tooling consumption (auditor's input on subsequent passes).

### Frame conditions
- Reads workspace and `deterministic_output`.
- Writes only to `docs/add/audit/<pass_id>.{md,json}`.

### Module invariants preserved
- I2 (proposes; doesn't execute).

### Test linkage
- T4.5 — instrumentation output has 5 signals across 3 artifacts; report has 3 verdict records; markdown file at expected path.
- T4.6 — instrumentation output has 0 signals; report has empty_signal_set=true with explicit notice.
- T4.7 — Auditor invocation with a tool that attempts to write outside docs/add/audit/ → rejected at agent-spawn time (precondition violation).

---

## F4.4 — `auditor-scope-determination(deterministic_output) → List<ArtifactRef>`

```yaml
---
id: F4.4
status: Drafted
implementation: manual
consumes: [M4-auditor/B4, IC6, S5.1, ADR-002, ADR-003]
produces: [I4, T4.8, T4.9]
---
```

### Signature
`auditor-scope-determination(deterministic_output: InstrumentationOutput) → List<ArtifactRef>`

### Postconditions

Returns the deduplicated list of artifacts referenced by signals in `deterministic_output`. The Auditor's per-pass attention is limited to this set. Artifacts not in this set are *not* re-evaluated by the Auditor in this pass.

### Frame conditions
- Pure transformation of `deterministic_output`.

### Module invariants preserved
- I4 (Auditor renders verdicts only on artifacts the deterministic layer flagged).

### Test linkage
- T4.8 — instrumentation output references 5 distinct artifacts → scope list has 5 elements.
- T4.9 — instrumentation output is empty → scope list is empty (Auditor produces empty-signal-set report per F4.3).

### Discipline note
This operation is the structural mechanism by which the deterministic-then-LLM order is enforced. An Auditor implementation that bypassed F4.4 and scanned all artifacts would slip back into Path A (pure-LLM) territory per ADR-002.

---

## F4.5 — `auditor-tool-allowlist-enforce(definition: AuditorAgentDefinition) → EnforceVerdict`

```yaml
---
id: F4.5
status: Drafted
implementation: manual
consumes: [M4-auditor/B1, IC6, S5.1, ADR-003, ADR-005, TM4]
produces: [I3, T4.10]
---
```

### Signature
`auditor-tool-allowlist-enforce(definition: AuditorAgentDefinition) → EnforceVerdict`

`EnforceVerdict := Allowed | Denied { tool: ToolSpec, violation_reason: String }`

### Preconditions
- `definition` is the Auditor agent definition loaded from `agents/<auditor-name>.md`.
- The harness is about to spawn an Auditor agent invocation.

### Postconditions

For every `tool` in `definition.tool_allowlist`:
- If `tool` is a `WriteTool` whose `paths` glob matches anything under `docs/add/` (other than `docs/add/audit/`), `docs/invariants/`, `agents/`, `skills/`, or `.claude/rules/`, return `Denied` with the offending tool and the violation reason.
- Otherwise allow.

If all tools pass, return `Allowed`.

The harness invokes this predicate **at agent-spawn time**, not at first-tool-invocation time. A definition that fails this predicate cannot spawn an Auditor invocation at all. This is the structural mechanism that enforces TM4.

### Frame conditions
- Reads `definition` only.

### Module invariants preserved
- I3 (Auditor cannot write to protected paths — harness-enforced).

### Test linkage
- T4.10 — definition with `WriteTool { paths: ["docs/add/**"] }` (overly broad) → Denied with violation reason citing protected paths.

### Implementation discipline note
The harness-level enforcement is not optional. A future change that moved this check to "first write attempt" would create a TOCTOU window during which the Auditor could write before being checked. Per the discipline, the check is at spawn time.

---

## Module invariants — `I1`..`I4`

### I1 — Verdict citations
Every `VerdictRecord` produced by F4.1 has `cited_signals` non-empty. Verdicts without a cited signal are malformed and the Auditor retries.

### I2 — Verdicts not executions
The Auditor produces `VerdictRecord`s and `RemediationProposal`s. It does not execute remediations; the actual amendment is committed by Byfuglien, Hellebuyck, or a human after adjudication. The Auditor's tool allowlist (I3) prevents it from authoring artifacts even if its prompt encouraged that.

### I3 — Tool-allowlist as harness contract
The Auditor's `tool_allowlist` (declared in `agents/<auditor-name>.md` frontmatter) is a hard contract enforced at agent-spawn time by F4.5. A definition that violates the allowlist cannot spawn an Auditor invocation. Convention-only enforcement is rejected.

### I4 — Scope determined by deterministic layer
The set of artifacts the Auditor renders verdicts on equals exactly the set of artifacts referenced by signals in the deterministic output. F4.4 is the function; the Auditor's prompt template MUST be structured so that the agent cannot reason its way past the scope (e.g., "I notice this other artifact looks suspicious" should not produce a verdict on that artifact).

---

## Test linkage stubs — `T4.1`..`T4.10`

| ID | Operation | Stub description |
|---|---|---|
| T4.1 | F4.1 | one EditFrequency signal → verdict cites it |
| T4.2 | F4.1 | empty signals → precondition violation |
| T4.3 | F4.2 | Drifted spec verdict → proposal routes to Hellebuyck |
| T4.4 | F4.2 | Drifted test verdict → proposal routes to Byfuglien |
| T4.5 | F4.3 | 5 signals 3 artifacts → 3 verdict records, MD report at expected path |
| T4.6 | F4.3 | 0 signals → empty_signal_set=true, explicit notice |
| T4.7 | F4.3 | tool attempting unprotected write → spawn-time rejection |
| T4.8 | F4.4 | 5 distinct artifacts → 5-element scope list |
| T4.9 | F4.4 | empty input → empty scope list |
| T4.10 | F4.5 | overly-broad WriteTool → Denied with violation reason |

---

## What this spec deliberately does not specify

- The Auditor's name (placeholder `auditor`; per ADR-003 the agent and human choose).
- The exact prompt template the Auditor uses (SKILL-md-tier; agent drafts when authoring `agents/<auditor>.md`).
- The verdict-report Markdown layout beyond the structural fields. The auditor agent's own SKILL.md / agent definition specifies this.
- The harness mechanism for tool-allowlist enforcement (Claude Code permission system, plugin loader hook, etc.). The interface is `auditor-tool-allowlist-enforce`; the implementation is plugin-architecture-dependent.

## Open questions surfaced by this draft

1. **`PassId` format.** I used `pass-YYYY-MM-DD-HHMM` for human readability. Alternative: monotonic integer `pass-0001`, `pass-0002`. The former tells you when at a glance; the latter survives clock skew. Worth picking.
2. **Routing rules in F4.2.** I committed on three buckets (Hellebuyck, Byfuglien, Human). The architectural spec doesn't enumerate the artifact → agent mapping; I extrapolated from ADR-003. Worth confirming.
3. **F4.3 location of JSON sidecar.** The Markdown report goes at `docs/add/audit/<pass_id>.md`. The JSON sidecar I left at the same directory with `.json` extension. Alternatives: `docs/add/audit/<pass_id>.json` (current) or `.assurance/audit-<pass_id>.json` (separate directory). Current is simplest; flag if you'd prefer separation.
4. **F4.5 enforcement of `docs/add/audit/` write authority.** I treated this as the Auditor's only legitimate write target. Per ADR-005's authorship constraint, humans can also write there during adjudication. The pre-commit hook from M5 handles author identity; the harness allowlist handles the agent. Layered enforcement is the discipline.
5. **Empty-signal-set semantics.** I committed on "no signals → all Settled by absence-of-signal." An alternative read: "no signals → instrumentation suspect; flag for human review." The former is closer to "trust the deterministic layer"; the latter is closer to "verify the deterministic layer." Worth picking.
