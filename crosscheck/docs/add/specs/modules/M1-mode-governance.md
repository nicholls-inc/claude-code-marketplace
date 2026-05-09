# M1-mode-governance — Functional Spec

```yaml
---
id: M1-mode-governance
mode: add
phase: 1
status: Drafted
consumes: [IC1, IC5, IC9, S1.1, S1.2, S3.1, S3.2, S3.3, S3.4, S3.5, M1-mode-governance/B1, M1-mode-governance/B2, M1-mode-governance/B3, M1-mode-governance/B4, M1-mode-governance/B5]
produces: [F1.1, F1.2, F1.3, F1.4, F1.5, I1, I2, I3, I4, I5, T1.1..T1.10]
last-attested: N/A (Drafted)
---
```

## Purpose

Per-operation functional specs for the **mode-governance** module: the data shapes, predicates, and invariants that implement the mode and phase tagging discipline. This file consumes the M1 behavioural invariants in `specs/behavioral.md` and produces the operation-level pre/post-conditions an implementer (agent or human) can compile against.

The module exposes five operations, each numbered `F1.<n>`, plus five invariants `I1`–`I5` that all operations preserve, plus ten test linkage stubs `T1.1`–`T1.10`.

Implementation note (per S2.5 and ADR-005's open question on `implementation:`): the operations in this module are **deterministic** and best implemented in the same language as `S4.1`'s deterministic instrumentation tool. They do not require `/spec-iterate` and are tagged `implementation: manual`. Each `F` carries its own `implementation:` declaration in the frontmatter block at the start of the section.

---

## Data shapes

These types are referenced by every operation below. They are not themselves operations; they are the shared vocabulary. Renaming a shape cascades to every operation that consumes it.

### `Mode`

```
Mode := { bootstrap, add }
```

Closed enumeration. New values require a supersession ADR to ADR-001.

### `Phase`

```
Phase := { 0, 1, 2, 3, 4, 5 }
```

Closed enumeration. New values require a supersession ADR to the methodology.

### `ModuleRef`

```
ModuleRef := {
  slug: String matching ^[a-z][a-z0-9-]*$,
  invariant_doc_path: AbsolutePath,
  module_id: String matching ^M[0-9]+-[a-z][a-z0-9-]*$
}
```

The `module_id` is `M<n>-<slug>` per glossary § ID conventions.

### `ModeTag`

```
ModeTag := {
  mode: Mode,
  phase: Phase,
  attested_at: GitSha | "draft",
  source: ExplicitFrontmatter | ImpliedDefault
}
```

`source` distinguishes a module that explicitly declared its mode tag from one that received the default.

### `IntegrityVerdict`

```
IntegrityVerdict := Ok | Violation { kind: ViolationKind, location: ArtifactRef, signal: SignalRef }
ViolationKind := ModeTagFlipped | PhaseRegressed | RequiredArtifactMissing | DanglingB | OrphanB
```

Used as the output type of every integrity predicate in this module.

---

## F1.1 — `mode-tag-monotonic(module: ModuleRef) → IntegrityVerdict`

```yaml
---
id: F1.1
status: Drafted
implementation: manual
consumes: [M1-mode-governance/B1, IC5, S1.1, S1.2]
produces: [I1, T1.1, T1.2]
---
```

### Signature
`mode-tag-monotonic(module: ModuleRef) → IntegrityVerdict`

### Preconditions
- `module.invariant_doc_path` exists and is a readable Markdown file with a YAML frontmatter block at line 1.
- The module's git history is reachable (i.e., the predicate is invoked inside a git working tree).

### Postconditions

Returns `Ok` iff one of:
1. The mode tag has never changed across the module's git history (the only commit that introduced `mode:` is the file's creation commit).
2. The mode tag has changed at most once, AND the change-commit's parent commit history contains a `decisions/ADR-NNN-*.md` referenced in the commit message under a `Supersedes-mode-of:` trailer.

Returns `Violation { kind: ModeTagFlipped, ... }` iff:
3. The mode tag has changed in the module's history with no Supersedes-mode-of trailer in the change-commit message, OR
4. The mode tag has changed more than once (regardless of trailer presence).

### Frame conditions
- Reads only the module's git log and frontmatter.
- Does not modify the working tree; does not create or delete files.

### Module invariants preserved
- I1 (mode tag monotonicity).

### Test linkage
- T1.1 — set mode to `bootstrap`, then to `add` without trailer, expect `Violation::ModeTagFlipped`.
- T1.2 — set mode to `bootstrap`, then to `add` with `Supersedes-mode-of: ADR-NNN`, expect `Ok`.

---

## F1.2 — `phase-monotonic(module: ModuleRef) → IntegrityVerdict`

```yaml
---
id: F1.2
status: Drafted
implementation: manual
consumes: [M1-mode-governance/B2, IC5, S1.1, S1.2]
produces: [I2, T1.3, T1.4]
---
```

### Signature
`phase-monotonic(module: ModuleRef) → IntegrityVerdict`

### Preconditions
- Same as F1.1.

### Postconditions

Returns `Ok` iff one of:
1. The module's `phase:` field has been set in a strictly increasing sequence across its git history (each phase-change commit advances `phase` by exactly 1).
2. The module's `phase:` field has been retracted via a `Status: Retracted-with-Reason` transition (terminal; no further phase claims expected).
3. The module's `phase:` field has been re-drafted backward by an explicit re-drafting event triggered by an upstream change. The re-drafting commit must reference the upstream commit SHA in a `Re-drafting-cause:` trailer.

Returns `Violation { kind: PhaseRegressed, ... }` iff:
4. Any phase-decrement commit lacks a `Re-drafting-cause:` trailer, OR
5. Any phase-jump commit advances `phase` by more than 1 (skipping phases is not permitted; a phase advance requires the prior phase to have been recorded).

### Frame conditions
- Same as F1.1.

### Module invariants preserved
- I2 (phase monotonicity, with the re-drafting and supersession exceptions).

### Test linkage
- T1.3 — phase 0 → 1 → 2, expect `Ok`.
- T1.4 — phase 2 → 1 with no trailer, expect `Violation::PhaseRegressed`.

---

## F1.3 — `mode-of(module: ModuleRef) → ModeTag` (dual-location aware)

```yaml
---
id: F1.3
status: Drafted (re-drafted per Phase 2 seam validation A-11)
implementation: manual
consumes: [M1-mode-governance/B3, IC9, S1.1, "skills/assurance-init/SKILL.md § Step 6.5 (VGD prereq summary)"]
produces: [I3, T1.5, T1.6, T1.5b]
---
```

### Signature
`mode-of(module: ModuleRef) → ModeTag`

### Preconditions
- The module's invariant doc exists at one of two locations:
  - `docs/invariants/<module>.md` (bootstrap-mode convention; pre-existing repo state).
  - `docs/add/specs/modules/<module>.md` (ADD-mode convention; per S1.1 § dual location).

### Postconditions

The predicate inspects the module's invariant doc in this priority order:

1. **YAML frontmatter present at line 1** with both `mode:` and `phase:` keys and valid values:
   Returns `ModeTag { mode = <parsed>, phase = <parsed>, attested_at = <parsed or "draft">, source = ExplicitFrontmatter }`.
2. **Prose Status field** matching the existing bootstrap-mode convention from `/assurance-init` (e.g., `Status: Skeleton`, `Status: Active`):
   Returns `ModeTag { mode = bootstrap, phase = inferred-from-status-line, attested_at = "draft", source = ProseStatus }`. The phase inference: `Skeleton → 0`, `Active or any other → 5`. This branch preserves IC9 — existing bootstrap-mode invariant docs do not need YAML frontmatter retrofit.
3. **Neither frontmatter nor prose Status**:
   Returns `ModeTag { mode = bootstrap, phase = 5, attested_at = "draft", source = ImpliedDefault }`.

The location-resolution rule: try `docs/invariants/<module>.md` first, then `docs/add/specs/modules/<module>.md`. If both exist, this is a dual-location violation; return `ModeTag` with an additional `dual_location_violation: true` flag the integrity check (S1.2) reports.

### Frame conditions
- Reads the module's invariant doc only.
- No git history is read by this predicate.

### Module invariants preserved
- I3 (default tag is bootstrap, phase 5 — uniformly applied across all governance-consulting skills).

### Test linkage
- T1.5 — module with explicit YAML `mode: add, phase: 1` frontmatter, expect `ModeTag { mode=add, phase=1, source=ExplicitFrontmatter }`.
- T1.5b — module with `Status: Skeleton` prose line (existing bootstrap convention), expect `ModeTag { mode=bootstrap, phase=0, source=ProseStatus }`.
- T1.6 — module with no frontmatter and no Status line, expect `ModeTag { mode=bootstrap, phase=5, source=ImpliedDefault }`.

### Implementation discipline note

This predicate is the single source of truth for mode resolution. Per the seam-validation A-9 finding, the *existing* set of governance-consulting skills that MUST call `mode-of` rather than re-implementing the read is broader than initially enumerated:

- Existing Hellebuyck-owned: `/assurance-layer-audit`, `/assurance-init`, `/assurance-status`, `/assurance-roadmap-check`, `/intent-check`, `/spec-adversary`, `/acceptance-oracle-draft`, `/protected-surface-amend`, `/invariant-coverage-scaffold`.
- New ADD-mode skills (Hellebuyck): `/intent-elicit`, `/spec-derive`, `/intent-check-prose`, `/spec-adversary-prose`.
- Auditor agent (post-S5).
- Other skills consulting governance (e.g., the lean pipeline's `/correspondence-review` when it consults invariant docs).

Operationally, the discipline is: any skill that reads a module's `mode:` or `phase:` MUST go through `F1.3`. A duplicated read is an integrity violation analogous to other "single source of truth" patterns in the repo. The architectural-spec list (S3.1–S3.8 plus S2.* plus S5.1) names the specific known consumers; future skills inherit the rule.

### VGD-prerequisite handoff (per Phase 2 seam validation A-12)

The module's invariant doc may also carry a VGD-prerequisite summary block (per `/assurance-init` § Step 6.5: a 4-row table `#1 Deterministic algebraic semantics`, `#2 Provable properties`, `#3 Tractable input generation`, `#4 Dual-development resources` with verdicts). `F1.3` does not parse this block — that's a separate F operation deferred to v1.x — but its presence MUST NOT break parsing. The parser tolerates content following the YAML frontmatter / Status line.

---

## F1.4 — `integrity-rules(mode: Mode, phase: Phase) → RuleSet`

```yaml
---
id: F1.4
status: Drafted
implementation: manual
consumes: [M1-mode-governance/B4, IC5, IC9, IC11, S1.2, ADR-001]
produces: [I4, T1.7, T1.8]
---
```

### Signature
`integrity-rules(mode: Mode, phase: Phase) → RuleSet`

`RuleSet := Set<Rule>` where `Rule := RequiresArtifact { kind, predicate } | RequiresLinkage { from, to }`.

### Preconditions
- `mode` and `phase` are valid values from their respective enumerations (enforced at the type level).

### Postconditions

Returns the rule set per the table below. The function is total (every (mode, phase) pair has a defined rule set).

| mode | phase | RuleSet (informal) |
|---|---|---|
| bootstrap | 0..4 | RequiresArtifact { invariant doc with at least one I and a covering test } |
| bootstrap | 5 | RequiresArtifact { invariant doc with at least one I, a covering test, and a continuous-assurance instrumentation hook } |
| add | 0 | RequiresArtifact { intent.md with at least one IC and a Status field } |
| add | 1 | RequiresArtifact { architectural.md with each IC consumed by ≥1 S }, plus RequiresLinkage { every B → ≥1 IC } (per IC11) |
| add | 2 | All of phase-1 rules, plus RequiresArtifact { intent-check-prose report with PASS or PASS-WITH-AMENDMENTS } |
| add | 3 | All of phase-2 rules, plus RequiresArtifact { skeleton: type-only signatures, failing test stubs } |
| add | 4 | All of phase-3 rules, plus RequiresArtifact { covering test for every I }, plus RequiresLinkage { every F → matching .dfy or `implementation-status: deferred-to-phase-N` } (per S2.5) |
| add | 5 | All of phase-4 rules, plus RequiresArtifact { continuous-assurance instrumentation hook } |

### Frame conditions
- Pure function of `(mode, phase)` — no I/O, no side effects.

### Module invariants preserved
- I4 (rule set is a function of (mode, phase) — not of skill identity, not of caller context).

### Test linkage
- T1.7 — `integrity-rules(add, 1)` returns the phase-1 rule set defined above.
- T1.8 — `integrity-rules(bootstrap, 5)` returns the bootstrap-phase-5 rule set.

### Why prose-only and not Dafny

Per N4 and the methodology, behavioural specs are prose with cross-references in v1. The rule-table here is a literal representation of `integrity-rules`'s semantics; an implementer (human or agent) producing a Python or Go function from this table can do so without further clarification. A Dafny encoding of the rule set is future work for the iteration after a model checker is integrated.

---

## F1.5 — `is-empty-repo(workspace: WorkspacePath) → Boolean`

```yaml
---
id: F1.5
status: Drafted
implementation: manual
consumes: [M1-mode-governance/B5, IC1, S3.1, S3.5]
produces: [I5, T1.9, T1.10]
---
```

### Signature
`is-empty-repo(workspace: WorkspacePath) → Boolean`

### Preconditions
- `workspace` exists and is the root of a git working tree.

### Postconditions

Returns `true` iff BOTH of:

1. None of the source manifests in the layer-audit's standard manifest list exist anywhere in the working tree. The list is inherited verbatim from `skills/assurance-layer-audit/SKILL.md` § Step 2 (per Phase 2 seam validation A-8); `is-empty-repo` reads `/assurance-layer-audit`'s configuration at runtime rather than carrying its own copy:

   | Manifest | Language |
   |---|---|
   | `go.mod`, `go.sum` | Go |
   | `pyproject.toml`, `setup.py`, `setup.cfg`, `requirements*.txt`, `Pipfile`, `poetry.lock` | Python |
   | `package.json`, `tsconfig.json`, `pnpm-lock.yaml`, `yarn.lock` | TypeScript / JavaScript |
   | `Cargo.toml`, `Cargo.lock` | Rust |
   | `Gemfile`, `*.gemspec` | Ruby |
   | `pom.xml`, `build.gradle`, `build.gradle.kts` | Java / Kotlin |
   | `*.csproj`, `*.sln`, `*.fsproj` | C# / .NET |
   | `mix.exs` | Elixir |
   | `*.cabal`, `stack.yaml` | Haskell |

   Future additions to the manifest list (e.g., a new ecosystem) update both `/assurance-layer-audit` and `is-empty-repo` simultaneously by virtue of the shared source.

2. `<workspace>/docs/add/intent.md` does not exist.

Otherwise returns `false`.

### Frame conditions
- Reads only the working tree (not git history); no I/O outside the workspace.

### Module invariants preserved
- I5 (empty-repo classification is conservative — both manifest absence AND intent.md absence required).

### Test linkage
- T1.9 — workspace with only `.gitignore`, `LICENSE`, `README.md`, expect `true` (none of those count as manifests).
- T1.10 — workspace with `package.json` only, expect `false`. Workspace with `docs/add/intent.md` only, expect `false`. Workspace with both, expect `false`.

### Implementation discipline note

`/assurance-layer-audit` and `/acceptance-oracle-draft` both MUST call `is-empty-repo` rather than reimplementing the predicate. Per `B3` (M1) and `I3`, divergent implementations of the same predicate are an integrity violation.

---

## Module invariants — `I1`..`I5`

These are the durable invariants the operations preserve. Each is referenced by one or more `F` operations above.

### I1 — Mode tag monotonicity
The set `{ commit C | C is in module's git history ∧ C set the mode tag to a different value than the prior commit }` either is empty, contains exactly one commit whose message has a `Supersedes-mode-of:` trailer, or the integrity predicate F1.1 returns a Violation.

### I2 — Phase monotonicity (with re-drafting and supersession exceptions)
The sequence of `phase:` values across the module's git history is strictly increasing by 1, except that decrements are permitted when accompanied by an explicit `Re-drafting-cause:` trailer naming the upstream commit, and any sequence terminates at a `Status: Retracted-with-Reason` transition.

### I3 — Default uniformity (extended scope per A-9)
For every governance-consulting skill `s` and every module `m` lacking explicit `mode:`/`phase:` frontmatter, the value computed by `s` for `(mode, phase)` of `m` equals `(bootstrap, 5)`. The set of governance-consulting skills includes (post-seam-validation): `/assurance-layer-audit`, `/assurance-init`, `/assurance-status`, `/assurance-roadmap-check`, `/intent-check`, `/spec-adversary`, `/acceptance-oracle-draft`, `/protected-surface-amend`, `/invariant-coverage-scaffold`, the four greenfield ADD skills, the lean pipeline (where it consults invariants), and the Auditor agent. The discipline is enforced structurally: every such skill MUST call `F1.3 (mode-of)` rather than computing locally. A duplicated read is an integrity violation.

### I4 — Rule-set determinism
For all valid `(mode, phase)`, the rule set returned by `F1.4 (integrity-rules)` is fixed. No skill or caller can alter the rule set without modifying `F1.4`'s table, which requires a propagated-discovery or intent-refinement classification per ADR-005.

### I5 — Conservative empty-repo classification
`F1.5 (is-empty-repo)` returns `true` only when manifest absence AND `docs/add/intent.md` absence both hold. False positives (classifying a non-empty repo as empty) are ruled out by the conjunction; false negatives (classifying an empty repo as non-empty) are tolerated as the safer-error direction (the user with no real artifacts who happens to have a `docs/add/intent.md` is already on the ADD path anyway).

---

## Test linkage stubs — `T1.1`..`T1.10`

Each stub references exactly one operation above. The stubs are *failing* until Phase 4 implementation lands (per the methodology's "fully red CI" rule for Phase 3). Stub locations are at `tests/M1-mode-governance/test_<F-id>.py` (or whatever the agent's chosen language requires).

| ID | Operation | Stub description |
|---|---|---|
| T1.1 | F1.1 | mode flip without trailer → Violation::ModeTagFlipped |
| T1.2 | F1.1 | mode flip with trailer → Ok |
| T1.3 | F1.2 | phase 0→1→2 → Ok |
| T1.4 | F1.2 | phase 2→1 without trailer → Violation::PhaseRegressed |
| T1.5 | F1.3 | explicit YAML frontmatter → ExplicitFrontmatter source |
| T1.5b | F1.3 | prose `Status: Skeleton` line (existing bootstrap convention) → ProseStatus, (bootstrap, 0) |
| T1.6 | F1.3 | no frontmatter and no Status line → ImpliedDefault, (bootstrap, 5) |
| T1.7 | F1.4 | (add, 1) → phase-1 ADD rule set |
| T1.8 | F1.4 | (bootstrap, 5) → bootstrap-phase-5 rule set |
| T1.9 | F1.5 | workspace with only `.gitignore`/`LICENSE`/`README.md` → true |
| T1.10 | F1.5 | workspace with manifests → false; with intent.md → false |

---

## What this spec deliberately does not specify

- The implementation language (per S4.1's deferred choice).
- The exact format of git-trailer parsing (any robust trailer parser will do; no commitment here).
- The error rendering for `IntegrityVerdict::Violation` beyond the structured fields. The auditor agent's report and the pre-commit hook will format these for humans separately.
- The internal structure of `/assurance-layer-audit`'s manifest-list configuration; F1.5 reads from it at runtime per A-8.
- The format of the VGD-prerequisite summary block; `F1.3` tolerates its presence but does not parse it (deferred to v1.x).

## Open questions surfaced by this draft

1. **Git-trailer naming.** I chose `Supersedes-mode-of:` and `Re-drafting-cause:` as trailer names. They follow git-trailer conventions and read clearly but were invented here rather than ratified upstream. If you'd prefer different names (or want a single consolidated trailer), flag.
2. **Phase-skip semantics.** I-2 forbids skipping phases on advance (a `phase: 0 → 2` jump is rejected). This is stricter than the methodology's monotonic-forward rule, which permits skips. I went stricter because the auditor's "Settled" verdict relies on phase-by-phase attestation history; permitting skips would let modules claim phase 4 without a phase-2 attestation trail. Worth your judgment.
3. **`integrity-rules`'s "continuous-assurance instrumentation hook" rule for phase 5.** I assumed every phase-5 module needs an instrumentation hook. The architectural spec doesn't explicitly require this; I extrapolated from the methodology's "Phase 5 — Continuous assurance" section. If you'd rather wait until S4.1 specifies the hook contract before formalising this rule, easy to weaken to a soft check.

(Open question Q4 from v1.0 — manifest-list ownership — was resolved by the seam-validation pass per Bucket C: F1.5 consumes `/assurance-layer-audit`'s configuration at runtime; coupling acknowledged and acceptable.)
