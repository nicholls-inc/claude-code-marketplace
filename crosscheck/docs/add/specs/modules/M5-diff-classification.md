# M5-diff-classification — Functional Spec

```yaml
---
id: M5-diff-classification
mode: add
phase: 1
status: Drafted
consumes: [IC7, S6.1, ADR-005, TM2, M5-diff-classification/B1, M5-diff-classification/B2, M5-diff-classification/B3, M5-diff-classification/B4]
produces: [F5.1, F5.2, F5.3, F5.4, F5.5, I1, I2, I3, I4, T5.1..T5.10]
last-attested: N/A (Drafted)
---
```

## Purpose

Per-operation functional specs for the **diff-classification** module: pre-commit hook, CI gate, log emission, and the protected-paths predicate. The module enforces the fifth-class-extended ADR-005 taxonomy (`propagated-discovery | intent-refinement | drift | retraction | status-transition`) on every commit modifying protected paths.

The module is the structural mitigation for TM2 (silent spec drift). All five operations are deterministic; no LLM calls. Per S6.1 and the existing dual-track principle, the hook runs in under 5 seconds; the CI gate is the durable enforcement.

---

## Data shapes

### `ClassificationClass`

```
ClassificationClass := { propagated-discovery, intent-refinement, drift, retraction, status-transition }
```

Closed enumeration of five values. New values require a supersession ADR per ADR-005 § Consequences.

### `Trailer`

```
Trailer := {
  classification: ClassificationClass,
  justification: String,    // mandatory for drift, retraction, status-transition; optional otherwise
  raw_lines: List<String>   // the trailer block as it appeared in the commit message
}
```

### `LogEntry`

```
LogEntry := {
  timestamp: ISO8601,
  commit_sha: GitSha,
  author: AuthorIdent,
  classification: ClassificationClass,
  justification: String | None,
  modified_files: NonEmptyList<RelativePath>,
  related_ids: List<ID>    // IC, S, B, F, I, T, ADR ids parsed from commit body
}
```

### `EnforceResult`

```
EnforceResult := Allowed | Rejected { reason: String, exit_code: Integer, actionable_hint: String }
```

### `ProtectedPathPredicateResult`

```
ProtectedPathPredicateResult := Protected { path_class: PathClass } | NotProtected
PathClass := DocsAdd | DocsAddAudit | DocsInvariants | Agents | Skills | ClaudeRules
```

`DocsAddAudit` is treated separately because it carries the Auditor-only authorship constraint per ADR-005's authorship section.

---

## F5.1 — `parse-classification-trailer(commit_message: String) → Option<Trailer>`

```yaml
---
id: F5.1
status: Drafted
implementation: manual
consumes: [M5-diff-classification/B1, M5-diff-classification/B4, IC7, ADR-005]
produces: [I1, I4, T5.1, T5.2]
---
```

### Signature
`parse-classification-trailer(commit_message: String) → Option<Trailer>`

### Postconditions

Parses the commit message body for trailer lines matching:
- `Spec-Diff-Classification: <one of five values>`
- `Spec-Diff-Justification: <text>` (optional unless the classification requires it)

Returns `Some(Trailer)` iff:
1. A `Spec-Diff-Classification:` line is present.
2. Its value is one of the five legal values.
3. If the classification is `drift | retraction | status-transition`, a non-empty `Spec-Diff-Justification:` is also present.

Returns `None` if any of the above fails (the absence of trailer or a malformed trailer is treated identically — no commit is committed).

### Frame conditions
- Pure function of `commit_message`.

### Module invariants preserved
- I1 (trailer ubiquity on protected-path commits).
- I4 (the five classes are the only legal values).

### Test linkage
- T5.1 — message with `Spec-Diff-Classification: propagated-discovery` → `Some(Trailer { propagated-discovery, ... })`.
- T5.2 — message with `Spec-Diff-Classification: drift` and no justification → `None` (justification required for drift).

### Discipline note
Trailer parsing is deliberately strict. A typo (e.g., `intent_refinement` instead of `intent-refinement`) produces `None`, which the pre-commit hook surfaces as a rejection with an actionable hint. Lenient parsing would let drift hide behind typos.

---

## F5.2 — `is-protected-path(path: RelativePath, mode_resolver: ModeOf) → ProtectedPathPredicateResult`

```yaml
---
id: F5.2
status: Drafted
implementation: manual
consumes: [M5-diff-classification/B1, IC7, ADR-005, ADR-001]
produces: [I1, T5.3, T5.4]
---
```

### Signature
`is-protected-path(path: RelativePath, mode_resolver: ModeOf) → ProtectedPathPredicateResult`

`ModeOf := (ModuleRef → ModeTag)`    // typically M1's F1.3

### Postconditions

Returns `Protected { path_class }` iff `path` matches one of:
- `docs/add/**` (excluding `docs/add/audit/**` which is `DocsAddAudit`).
- `docs/add/audit/**` → `DocsAddAudit`.
- `docs/invariants/<module>.md` where `mode_resolver(<module>).mode == add` → `DocsInvariants`.
- `agents/**` → `Agents`.
- `skills/**` → `Skills`.
- `.claude/rules/**` → `ClaudeRules`.

Returns `NotProtected` otherwise.

The predicate consumes `mode_resolver` as a dependency injection point; in production this is `M1-mode-governance/F1.3 (mode-of)`. Bootstrap-mode modules' invariant docs do **not** require classification per ADR-001.

### Frame conditions
- Pure function of `path` and the resolved mode.

### Module invariants preserved
- I1.

### Test linkage
- T5.3 — `docs/add/intent.md` → Protected { DocsAdd }.
- T5.4 — `docs/invariants/billing.md` where billing is bootstrap-mode → NotProtected.

---

## F5.3 — `pre-commit-hook(modified_paths: List<RelativePath>, commit_message: String, author: AuthorIdent) → EnforceResult`

```yaml
---
id: F5.3
status: Drafted
implementation: manual
consumes: [M5-diff-classification/B1, M5-diff-classification/B3, IC7, S6.1]
produces: [I1, T5.5, T5.6]
---
```

### Signature
`pre-commit-hook(modified_paths, commit_message, author) → EnforceResult`

### Preconditions
- The hook is invoked by the configured pre-commit framework (pre-commit.com, lefthook, or husky per S3.2's detection).
- The hook completes in < 5s wall time per S6.1's dual-track principle.

### Postconditions

For each `path ∈ modified_paths`:
- Compute `is-protected-path(path, mode_resolver)`.

If any path is `Protected`:

1. Parse the commit message via F5.1. If the trailer is `None`, return `Rejected` with an actionable hint:
   ```
   ERROR: Commit modifies protected path(s) without Spec-Diff-Classification trailer.

   Add this trailer block to your commit message:

       Spec-Diff-Classification: <propagated-discovery | intent-refinement | drift | retraction | status-transition>
       Spec-Diff-Justification: <required for drift, retraction, status-transition; optional for others>

   See crosscheck/docs/add/decisions/ADR-005-diff-classification.md for guidance.
   ```
2. If any path has `path_class == DocsAddAudit`, check that `author` is in `.assurance/audit-authors.allowlist`. If not, return `Rejected` citing the authorship constraint.
3. **Verify attestation files for any LLM-gated work the modified paths require** (per Phase 2 seam validation A-10; mirrors `/intent-check`'s § Step 7 companion-hook pattern). If any modified path falls under a *covered protected surface* — i.e., a surface for which an attestation file is the SSOT proving the heavy LLM work ran — recompute the SHA-256 over the sorted protected files and compare against the attestation:
   - If `docs/invariants/<module>.md` is touched (Class B governance), check `.assurance/intent-check-attestation.json` exists, its `verdict == "pass"`, and its `content_hash` matches the recomputed hash. If absent / stale / mismatched → reject with hint pointing at `/intent-check`.
   - If `docs/add/specs/architectural.md` or `docs/add/intent.md` is touched (ADD-mode spec-stack), check `.assurance/intent-check-prose-attestation.json` similarly. If absent / stale / mismatched → reject with hint pointing at `/intent-check-prose`.
   - The hook does NOT invoke an LLM during this check. It only reads files, recomputes hashes, and compares. Per `/assurance-init`'s ROADMAP block: *"Pre-commit hooks are fast attestation checks only — they must never invoke LLMs or run slow test suites. Heavy verification lives in CI and in dedicated binaries that the pre-commit hook verifies were run."*

4. Otherwise, return `Allowed`.

If no paths are `Protected`, return `Allowed` unconditionally.

The hook is **fast** (no LLM, no network). Wall-time budget < 5 s. It runs locally on every commit attempt.

### Frame conditions
- Reads `modified_paths`, `commit_message`, `author`, the `audit-authors.allowlist` file, and the relevant `.assurance/*-attestation.json` files when the modified paths require attestation verification. Reads protected-file content only to recompute hashes.
- No mutation of the working tree or commit.

### Module invariants preserved
- I1 (trailer ubiquity).
- I9 (attestation verification on protected surfaces; pre-commit catches commits that bypass the heavy LLM step).

### Test linkage
- T5.5 — modify `docs/add/intent.md` with no trailer → Rejected, exit_code != 0, hint contains the trailer template.
- T5.6 — modify `docs/add/audit/foo.md` with valid trailer but author not in allowlist → Rejected with authorship-constraint hint.
- T5.6b — modify `docs/add/intent.md` with valid trailer but `intent-check-prose-attestation.json` content_hash mismatched → Rejected with `/intent-check-prose` hint.

---

## F5.4 — `ci-gate-validate-and-log(commit_range: CommitRange, log_path: RelativePath) → EnforceResult`

```yaml
---
id: F5.4
status: Drafted
implementation: manual
consumes: [M5-diff-classification/B1, M5-diff-classification/B2, M5-diff-classification/B3, IC7, S6.1]
produces: [I1, I2, T5.7, T5.8, T5.9]
---
```

### Signature
`ci-gate-validate-and-log(commit_range: CommitRange, log_path: RelativePath) → EnforceResult`

### Preconditions
- The CI job runs on every PR or merge group event in the CI system detected by S3.2.
- `commit_range` is the set of commits in the PR.
- For squash-merge events, `commit_range` is the single squashed commit (per Phase 2 A-6).

### Postconditions

For each commit `c ∈ commit_range`:
1. Identify `c.modified_paths`.
2. Check `is-protected-path` for each.
3. If any path is protected, parse F5.1 on `c.message`.
4. If parse fails, return `Rejected` listing every offending commit.

For squash-merge events:
- The single squashed commit must carry a **summary trailer** classifying the merged range (per A-6). The summary trailer's `Spec-Diff-Classification` must be the most-significant class among the pre-squash commits' classifications, where significance order is:
  `drift > retraction > intent-refinement > propagated-discovery > status-transition`
  (drift dominates; status-transition is least; per the dominance rule, "if any commit was drift, the squashed commit is drift").

If all commits validate, append each to `log_path` (`.assurance/diff-classification-log.jsonl` per S6.1) as `LogEntry`s. Append-only — never overwrite or delete existing lines.

For force-pushes that rewrite history on a protected branch, the CI gate rejects the push (per S6.1 amendment): the dual-track principle requires a server-side pre-receive hook to mirror this; the CI gate is the soft mirror.

### Frame conditions
- Reads `commit_range` and the existing log.
- Appends to `log_path` only.

### Module invariants preserved
- I1 (trailer ubiquity).
- I2 (log append-only).

### Test linkage
- T5.7 — PR with 3 commits all carrying valid trailers → 3 log entries appended.
- T5.8 — PR with one commit missing trailer → Rejected listing offending commit; no log entries written.
- T5.9 — PR squash-merge where pre-squash commits include one drift; squashed commit's classification ≠ drift → Rejected with dominance-rule explanation.

---

## F5.5 — `log-append-only-check(log_path: RelativePath, base_commit: GitSha, head_commit: GitSha) → EnforceResult`

```yaml
---
id: F5.5
status: Drafted
implementation: manual
consumes: [M5-diff-classification/B2, IC7, S6.1]
produces: [I2, T5.10]
---
```

### Signature
`log-append-only-check(log_path, base_commit, head_commit) → EnforceResult`

### Postconditions

Compute the diff between `base_commit:log_path` and `head_commit:log_path`. The diff must be **strictly additive**:
- Existing lines must not be modified.
- Existing lines must not be deleted.
- Reordering existing lines is not permitted.
- New lines may only be appended at the end.

If the diff violates any of the above, return `Rejected` with an actionable hint quoting the offending hunk.

### Frame conditions
- Reads two git revisions of `log_path`.

### Module invariants preserved
- I2 (log is append-only).

### Test linkage
- T5.10 — diff that deletes line 3 of the log → Rejected with "log line deleted at L3" message.

---

## Module invariants — `I1`..`I5`

### I1 — Trailer ubiquity
For every commit `c` whose `modified_paths` includes any protected path, `c.message` includes a valid `Spec-Diff-Classification` trailer (parsed by F5.1). The pre-commit hook (F5.3) and CI gate (F5.4) are the layered enforcement.

### I2 — Log append-only
`.assurance/diff-classification-log.jsonl` admits only appended lines. F5.5 enforces in CI.

### I3 — Squash dominance
For squash-merge events, the squashed commit's classification is the most-significant class among the pre-squash range, per the dominance rule. The CI gate (F5.4) enforces.

### I4 — Five-class taxonomy
The set of legal `Spec-Diff-Classification` values is exactly `{propagated-discovery, intent-refinement, drift, retraction, status-transition}`. F5.1's parser rejects any other value.

### I5 — Attestation verification on protected surfaces (per A-10)
For every commit `c` whose `modified_paths` touch a covered protected surface (`docs/invariants/<module>.md` for `/intent-check`; `docs/add/intent.md` or `docs/add/specs/architectural.md` for `/intent-check-prose`), the corresponding `.assurance/*-attestation.json` exists with `verdict == "pass"` and `content_hash` matching the recomputed SHA-256 over sorted protected files. F5.3 enforces at pre-commit time without invoking an LLM, mirroring `/intent-check`'s § Step 7 companion-hook pattern.

---

## Test linkage stubs — `T5.1`..`T5.11`

| ID | Operation | Stub description |
|---|---|---|
| T5.1 | F5.1 | propagated-discovery trailer → Some |
| T5.2 | F5.1 | drift trailer with no justification → None |
| T5.3 | F5.2 | docs/add/intent.md → Protected { DocsAdd } |
| T5.4 | F5.2 | docs/invariants/billing.md (bootstrap) → NotProtected |
| T5.5 | F5.3 | modify intent.md no trailer → Rejected with template hint |
| T5.6 | F5.3 | modify docs/add/audit/foo.md as non-allowlisted author → Rejected |
| T5.6b | F5.3 | modify intent.md with valid trailer but stale/mismatched intent-check-prose-attestation → Rejected with /intent-check-prose hint |
| T5.7 | F5.4 | 3-commit PR all valid → 3 log entries |
| T5.8 | F5.4 | 3-commit PR one missing trailer → Rejected, no log writes |
| T5.9 | F5.4 | squash-merge with drift in range, squashed != drift → Rejected |
| T5.10 | F5.5 | diff deleting log line → Rejected |

---

## What this spec deliberately does not specify

- The exact framework integration (pre-commit.com vs lefthook vs husky). S3.2's detection determines this; the hook is generated for the detected framework.
- The exact CI configuration syntax (GitHub Actions vs GitLab CI vs CircleCI). S3.2's detection determines this.
- The `audit-authors.allowlist` file format beyond the structural requirement (one author identity per line, comments allowed via `#`).
- The wording of every rejection hint beyond the templates above.

## Open questions surfaced by this draft

1. **Squash dominance order.** I committed on `drift > retraction > intent-refinement > propagated-discovery > status-transition`. The intuition: drift is the most consequential (require explicit attestation); status-transition is the least. Worth confirming.
2. **`audit-authors.allowlist` location.** I placed it at `.assurance/audit-authors.allowlist`. Alternative: `crosscheck/docs/add/audit/AUTHORS` (closer to the protected path). Worth picking.
3. **Force-push enforcement.** The CI gate rejects force-pushes per S6.1, but the actual mechanism (server-side pre-receive hook vs branch protection rules) is platform-dependent. I left this to S3.2's CI-system detection. Worth confirming this is acceptable.
4. **`log_path` location.** I committed on `.assurance/diff-classification-log.jsonl` per A-13. Some teams prefer per-repo logs at a different path. The default is here; configurable via env var.
5. **Author identity in trailer.** F5.4's log entry includes `author: AuthorIdent`. The git author email may not match the actual human/agent identity for sandboxed runs. I left this as a TODO for the implementer; the simple option is to use `git log --format=%ae`.
