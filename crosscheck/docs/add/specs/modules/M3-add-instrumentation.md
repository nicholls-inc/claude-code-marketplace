# M3-add-instrumentation — Functional Spec

```yaml
---
id: M3-add-instrumentation
mode: add
phase: 1
status: Drafted
consumes: [IC8, S4.1, S4.2, ADR-002, M3-add-instrumentation/B1, M3-add-instrumentation/B2, M3-add-instrumentation/B3, M3-add-instrumentation/B4]
produces: [F3.1, F3.2, F3.3, F3.4, F3.5, I1, I2, I3, I4, T3.1..T3.10]
last-attested: N/A (Drafted)
---
```

## Purpose

Per-operation functional specs for the **add-instrumentation** module: the deterministic tool that computes structured signals from git history and the linkage graph, plus the contract by which the Auditor agent (M4) consumes that output. Per ADR-002, the instrumentation layer detects signals; LLMs render judgments. The discipline is enforced structurally — this module contains *no* LLM dependencies.

The tool runs on demand (manually or via M4's workflow) and on schedule (CI cron). It is invoked as either a script or a SKILL.md per S4.1's deferred choice; per the M2 architectural note, if implemented as a skill it is plugin-level and owned by Hellebuyck (the Auditor invokes but does not own it).

---

## Data shapes

### `Signal`

```
Signal := {
  signal_id: String matching ^signal:[a-z-]+:[^:]+@\d+d=\S+$,    // e.g., signal:edit-frequency:docs/add/intent.md@30d=12
  signal_kind: SignalKind,
  artifact_ref: ArtifactRef,
  window_days: PositiveInteger,
  value: SignalValue,
  computed_at: Timestamp
}

SignalKind := EditFrequency | ChangeCoupling | LinkageOrphan | LinkageDangling | CascadePending | DiffShape

SignalValue :=
  | IntegerValue { count: Integer }
  | CouplingValue { artifact_a: ArtifactRef, artifact_b: ArtifactRef, edits_a: Integer, edits_b: Integer, ratio: Float }
  | OrphanValue { artifact: ArtifactRef, missing_edge: ConsumesOrProduces }
  | DanglingValue { artifact: ArtifactRef, dangling_edge: ConsumesOrProduces, target_id_referenced: String }
  | CascadeValue { upstream: ArtifactRef, upstream_commit: GitSha, downstream: ArtifactRef, last_attested_commit: GitSha }
  | DiffShapeValue { artifact: ArtifactRef, new_clauses: Integer, modified_clauses: Integer, deleted_clauses: Integer }
```

### `RunMetadata`

```
RunMetadata := {
  schema_version: SemVer,             // e.g., "1.0.0"
  invoked_at: Timestamp,
  git_commit: GitSha,                 // HEAD at invocation
  window_days: PositiveInteger,
  configuration_fingerprint: String   // hash of all relevant env vars
}
```

### `InstrumentationOutput`

```
InstrumentationOutput := JSONLines {
  line 1: RunMetadata
  lines 2..n: Signal
}
```

Stable schema for v1; additions follow `B3`.

---

## F3.1 — `compute-edit-frequency(workspace: WorkspacePath, paths: List<PathGlob>, window_days: Integer) → List<Signal>`

```yaml
---
id: F3.1
status: Drafted
implementation: manual
consumes: [M3-add-instrumentation/B1, M3-add-instrumentation/B2, IC8, S4.1]
produces: [I1, I2, T3.1, T3.2]
---
```

### Signature
`compute-edit-frequency(workspace, paths, window_days) → List<Signal>`

### Preconditions
- `workspace` is a git working tree.
- `paths` is the set of glob patterns to scan (default: `docs/add/**/*.md`, `docs/invariants/**/*.md`).
- `window_days >= 1`.

### Postconditions

For each file matching `paths`, compute the count of commits in `[HEAD-window_days, HEAD]` that modified the file. Return one `Signal` per file with non-zero count:
- `signal_kind = EditFrequency`
- `signal_id = "signal:edit-frequency:" + relative_path + "@" + window_days + "d=" + count`
- `value = IntegerValue { count }`

The computation invokes `git log` and parses output; no LLM is involved.

### Frame conditions
- Read-only access to the git tree.
- No mutation of any artifact.

### Module invariants preserved
- I1 (no LLM dependency).
- I2 (deterministic — same git state, same window, same paths produces identical signal list).

### Test linkage
- T3.1 — workspace with one spec file edited 5 times in last 30 days; expect single signal with `count=5`.
- T3.2 — invoke twice in succession; output is byte-identical (modulo `computed_at`).

---

## F3.2 — `compute-change-coupling(workspace, pair_specs, window_days) → List<Signal>`

```yaml
---
id: F3.2
status: Drafted
implementation: manual
consumes: [M3-add-instrumentation/B1, M3-add-instrumentation/B2, IC8, S4.1]
produces: [I1, I2, T3.3, T3.4]
---
```

### Signature
`compute-change-coupling(workspace, pair_specs: List<PairSpec>, window_days) → List<Signal>`

`PairSpec := { spec_glob: PathGlob, test_glob: PathGlob }`

### Postconditions

For each `PairSpec`, identify pairs `(a, b)` of files where `a` matches `spec_glob` and `b` matches `test_glob` (with the convention that `b`'s name corresponds structurally to `a` — e.g., `docs/invariants/billing.md` ↔ `tests/test_billing.py`). For each pair, compute:
- `edits_a := count of commits in window modifying a`
- `edits_b := count of commits in window modifying b`
- `ratio := if edits_a > 0 then edits_b / edits_a else 0.0`

Emit a `ChangeCoupling` signal when `ratio < 0.2` (the test side is significantly less edited than the spec side; an indication the test may not be tracking the spec). The 0.2 threshold is configurable via env var `CROSSCHECK_INSTRUMENTATION_COUPLING_THRESHOLD`.

### Frame conditions
- Same as F3.1.

### Module invariants preserved
- I1, I2.

### Test linkage
- T3.3 — pair with `edits_a=10, edits_b=0`; emit signal with ratio=0.0.
- T3.4 — pair with `edits_a=10, edits_b=8`; ratio=0.8 → no signal (above threshold).

---

## F3.3 — `compute-linkage-graph-integrity(workspace) → List<Signal>`

```yaml
---
id: F3.3
status: Drafted
implementation: manual
consumes: [M3-add-instrumentation/B1, M3-add-instrumentation/B2, IC8, IC11, S4.1]
produces: [I1, I2, T3.5, T3.6]
---
```

### Signature
`compute-linkage-graph-integrity(workspace) → List<Signal>`

### Preconditions
- `workspace` contains the `docs/add/` tree.

### Postconditions

Parse all artifacts under `docs/add/` and `docs/invariants/` and build the linkage graph from `consumes:` and `produces:` declarations.

Emit signals for:
- **Orphans:** any `S | B | F | I | T` ID that has no `consumes:` edge to upstream OR no `produces:` edge to downstream within the same module's scope. Per ADR-001, IC and ADR are root nodes and exempt from the `consumes:` requirement.
- **Dangling references:** any `consumes:` or `produces:` entry naming an ID that does not exist in the graph.
- **Cycles:** any cycle in the directed graph.
- **Specific to IC11:** `orphan-B` (a `B` with no `IC` ancestor) and `dangling-B` (a `B` with no `F` descendant) per S1.2.

Each violation produces one `Signal` with the corresponding `signal_kind`.

### Frame conditions
- Read-only graph traversal.

### Module invariants preserved
- I1, I2.

### Test linkage
- T3.5 — graph with one `B` lacking IC ancestry → signal with `signal_kind=LinkageOrphan, missing_edge=Consumes`.
- T3.6 — graph with `consumes: [IC99]` where IC99 doesn't exist → signal with `signal_kind=LinkageDangling`.

---

## F3.4 — `compute-cascade-pending(workspace, window_days) → List<Signal>`

```yaml
---
id: F3.4
status: Drafted
implementation: manual
consumes: [M3-add-instrumentation/B1, M3-add-instrumentation/B2, IC8, S4.1]
produces: [I1, I2, T3.7, T3.8]
---
```

### Signature
`compute-cascade-pending(workspace, window_days) → List<Signal>`

### Postconditions

For each Attested-or-Ratified upstream artifact `u` modified in the window:
- Identify all downstream artifacts `d` whose `consumes:` lists include `u.id`.
- Check whether each `d` has been re-attested (i.e., its `last-attested` field references a commit equal-or-after `u`'s modification commit).
- Emit a `CascadePending` signal for every `(u, d)` pair where `d` has not been re-attested since `u`.

Note: the `last-attested` field is read from the artifact's frontmatter or status block, per the format established in M1's frontmatter conventions and the seed artifacts' header fields.

### Frame conditions
- Read-only.

### Module invariants preserved
- I1, I2.

### Test linkage
- T3.7 — upstream IC1 amended at commit C; downstream S2.1 last-attested at C-1 → signal.
- T3.8 — upstream IC1 amended at commit C; downstream S2.1 last-attested at C+1 → no signal.

---

## F3.5 — `compute-diff-shape(workspace, paths, window_days) → List<Signal>`

```yaml
---
id: F3.5
status: Drafted
implementation: manual
consumes: [M3-add-instrumentation/B1, M3-add-instrumentation/B2, IC8, S4.1]
produces: [I1, I2, T3.9, T3.10]
---
```

### Signature
`compute-diff-shape(workspace, paths, window_days) → List<Signal>`

### Postconditions

For each commit in the window touching `paths`, structurally classify the diff into:
- `new_clauses`: count of new top-level Markdown sections (e.g., `### F1.x` headers added).
- `modified_clauses`: count of pre-existing sections with line-level edits.
- `deleted_clauses`: count of pre-existing sections fully deleted.

Emit a `DiffShape` signal per `(artifact, commit)` pair with non-zero counts.

### Frame conditions
- Read-only.

### Module invariants preserved
- I1, I2.

### Test linkage
- T3.9 — commit adds a new `### F2.5` section; signal with `new_clauses=1`.
- T3.10 — commit modifies the body of an existing F section; signal with `modified_clauses=1, new_clauses=0`.

---

## Module invariants — `I1`..`I4`

### I1 — No LLM dependency
The instrumentation tool's binary (or skill manifest, if implemented as a SKILL.md) contains no calls to any LLM API and no embedded prompt templates. Build-time check `tool-has-no-llm-dependencies` verifies this by scanning imports/dependency manifests.

### I2 — Deterministic output
For any fixed `(git_state, configuration_fingerprint, paths, window_days)`, the tool's output is byte-identical across runs (modulo the `invoked_at` timestamp in `RunMetadata`).

### I3 — Forward-compatible schema
A schema version increment that adds a new `SignalKind` or extends `SignalValue` with a new variant does not require existing consumers to upgrade. The Auditor agent's prompt template (M4) treats unknown signal kinds as additional context rather than as malformed input.

### I4 — Verdict citations
Every Auditor verdict cites at least one signal ID emitted by this module. The signal-ID format defined in `Signal.signal_id` is the citation token.

---

## Test linkage stubs — `T3.1`..`T3.10`

| ID | Operation | Stub description |
|---|---|---|
| T3.1 | F3.1 | one spec file edited 5x → single edit-frequency signal |
| T3.2 | F3.1 | run twice → byte-identical output (modulo timestamp) |
| T3.3 | F3.2 | spec edited 10x, test edited 0x → coupling signal at ratio=0.0 |
| T3.4 | F3.2 | spec edited 10x, test edited 8x → no signal |
| T3.5 | F3.3 | B with no IC ancestor → LinkageOrphan signal |
| T3.6 | F3.3 | consumes: IC99 (nonexistent) → LinkageDangling signal |
| T3.7 | F3.4 | downstream not re-attested since upstream amendment → CascadePending |
| T3.8 | F3.4 | downstream re-attested after upstream amendment → no signal |
| T3.9 | F3.5 | commit adds new section → DiffShape with new_clauses=1 |
| T3.10 | F3.5 | commit modifies existing section → DiffShape with modified_clauses=1 |

---

## What this spec deliberately does not specify

- The implementation language (S4.1 leaves to the agent; recommendation: Python or Go for git/parser ergonomics).
- The exact CLI of the tool (e.g., `add-instrumentation --window=30 --output=foo.jsonl`). Operational detail; agent picks during SKILL.md drafting.
- The pair-spec discovery mechanism for change-coupling (heuristic based on file naming; configurable in v1.x).
- Performance bounds (target: under 30s on a 1k-artifact repo per M1/B-future-extension; not enforced as a B invariant in v1).

## Open questions surfaced by this draft

1. **`compute-change-coupling` thresholds.** I committed on 0.2 as the coupling threshold. The number is founder-intuition (mirroring ADR-002's window-default discipline). Worth confirming or adjusting; configurable via env var either way.
2. **Pair-spec discovery.** F3.2 takes pair specs as input. In practice the tool needs a default discovery mechanism (filename matching). I left this to the implementer. Worth picking now vs. deferring to skill drafting.
3. **`compute-diff-shape` granularity.** I picked top-level Markdown headers as the unit. Sub-section edits (e.g., editing a list item) get classified as `modified_clauses=1` for the parent section. Acceptable for v1?
4. **Tool packaging.** Script vs. skill (S4.1's deferred choice). For deterministic computation with no prompt content, script is the natural answer. For runtime invocation by the Auditor (which is an agent), skill packaging is more idiomatic. I'd default to script with a thin SKILL.md wrapper that just invokes it. Worth your direction.
5. **Schema version bumps.** I3 says additions are forward-compatible. The mechanism for *announcing* a schema bump (e.g., a `schema-version` field in `RunMetadata` plus a CHANGELOG-style file) is not specified. v1 starts at `1.0.0` and the agent documents bumps in `references/add-instrumentation-schema.md` per S4.1.
