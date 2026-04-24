# Attestation schema — JSON format, SHA-256 computation, pre-commit hook

`/intent-check` writes `.assurance/intent-check-attestation.json` on every run. The attestation is the deterministic, LLM-free record that a companion pre-commit hook uses to decide whether a commit touching protected-surface files is allowed.

The pattern is lifted from xylem (`/Users/harry.nicholls/repos/xylem/docs/assurance/next/07-intent-check-phase.md`, lines 34–58) and generalised so any repo can adopt it without a Go binary.

## Why an attestation (not just a CI job)

A CI-only enforcement leaves a window in which a developer can push a broken spec-intent pairing and only learn about it minutes later, blocking their merge. A **pre-commit attestation** enforces the rule at the earliest possible moment, and — crucially — it does so without invoking an LLM. LLMs are slow, non-deterministic, and expensive; pre-commit hooks must be all three opposite.

The trick is the content-hash binding: the attestation records a SHA-256 of the protected files at the moment the LLM pipeline ran, and the pre-commit hook recomputes that hash from the current file contents. Any unexplained change to a protected file between `/intent-check` and `git commit` invalidates the attestation and rejects the commit.

## Schema

```json
{
  "protected_files":  ["<sorted-paths-relative-to-repo-root>"],
  "content_hash":     "<lowercase-hex-sha256>",
  "verdict":          "pass" | "fail",
  "checked_at":       "<RFC3339 timestamp>",
  "pipeline_output":  {
    "back_translation": "<Section 1 + Section 2 verbatim>",
    "diff_result": {
      "match":              true | false,
      "mismatch_reason":    "<string>",
      "mismatch_category":  "<enum>",
      "confidence_pct":     0-100,
      "confidence_basis":   "<enum>"
    }
  }
}
```

| Field                            | Required | Description                                                                                                                                                       |
|----------------------------------|----------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `protected_files`                | yes      | Alphabetically sorted list of repo-relative paths touched by this pipeline run. Forms the input to the content hash.                                              |
| `content_hash`                   | yes      | Lowercase hex SHA-256 of `concat(read_bytes(f) for f in protected_files)` with **no** delimiter between files. Order matches `protected_files`.                   |
| `verdict`                        | yes      | `pass` if the diff-checker returned `match=true` AND `confidence_pct>=80`; otherwise `fail`. Low-confidence matches are not passes.                                |
| `checked_at`                     | yes      | RFC3339 timestamp in UTC (e.g. `2026-04-24T14:32:10Z`). Used by external tooling to detect stale attestations beyond pre-commit scope.                            |
| `pipeline_output.back_translation` | yes    | Section 1 + Section 2 from the blind back-translator, verbatim. Making this human-readable is a deliberate forgery-cost decision — see "why the pipeline output is a field".     |
| `pipeline_output.diff_result`    | yes      | Full JSON object from the diff-checker after semantic validation (not the raw pre-validation output).                                                            |

Unknown/extra fields are tolerated by the pre-commit hook for forward compatibility but SHOULD NOT be added without updating this doc and the skill's verification checklist.

## Example

```json
{
  "protected_files": [
    "docs/invariants/queue.md",
    "internal/queue/queue.go",
    "internal/queue/queue_invariants_prop_test.go"
  ],
  "content_hash": "9c3b1f1e2d1b4a3a6c4b9e5d2c1b4a3a6c4b9e5d2c1b4a3a6c4b9e5d2c1b4a3a",
  "verdict": "pass",
  "checked_at": "2026-04-24T14:32:10Z",
  "pipeline_output": {
    "back_translation": "### Section 1: Behavioural guarantees\nThe test enforces that enqueue followed by dequeue preserves payload ordering under single-writer concurrency… \n\n### Section 2: Design rationale comments\nqueue_invariants_prop_test.go:452-457\n> Clock values are zeroed before equality comparison because wall-clock drift between reference run and crash run is a test-infrastructure artefact.",
    "diff_result": {
      "match": true,
      "mismatch_reason": "",
      "mismatch_category": "clean_match",
      "confidence_pct": 92,
      "confidence_basis": "rationale-found"
    }
  }
}
```

## SHA-256 computation (exact)

The hash binds the attestation to the specific byte contents of the protected files at the time of the run. Any subsequent edit to any protected file invalidates the attestation until `/intent-check` is re-run.

Algorithm:

```python
import hashlib

def content_hash(repo_root: str, protected_files: list[str]) -> str:
    h = hashlib.sha256()
    for rel_path in sorted(protected_files):  # sort BEFORE hashing, always
        with open(f"{repo_root}/{rel_path}", "rb") as f:
            h.update(f.read())  # NO delimiter, NO newline padding
    return h.hexdigest()
```

Invariants of the algorithm:

1. Sort `protected_files` alphabetically using locale-independent ordering (bytewise). The sorted list is what goes into the attestation AND into the hash. The sort is what makes the hash independent of the order files were discovered in.
2. Concatenate raw file bytes with **no delimiter**. Adding a delimiter (newline, null byte, filename prefix) would be fine, but the choice must stay fixed — the pre-commit hook must use the same algorithm. We pick "no delimiter" for simplicity; the file boundaries are implicit in the manifest (`protected_files`) and the hash only needs to be unique per (file-set, byte-content) pair.
3. Hex-encode the digest in lowercase. Case matters for hook equality comparison.

## Pre-commit hook (pseudocode; fast, no LLM)

Design target: < 1 second wall-clock on a typical repo. The hook runs on every commit, so even a few hundred milliseconds is perceptible.

```bash
#!/usr/bin/env bash
set -euo pipefail

ATTESTATION=".assurance/intent-check-attestation.json"
PATTERNS_FILE=".claude/rules/protected-surfaces.md"   # or a .gitignore-style pattern list

# Step 1: Identify staged protected files.
staged="$(git diff --cached --name-only --diff-filter=ACMR)"
protected_staged="$(echo "$staged" | grep_against_patterns "$PATTERNS_FILE" || true)"

# Step 2: Fast exit if no protected files were touched.
if [[ -z "$protected_staged" ]]; then
    exit 0
fi

# Step 3: Attestation must exist.
if [[ ! -f "$ATTESTATION" ]]; then
    echo "intent-check attestation missing. Run /intent-check and stage ${ATTESTATION} before committing." >&2
    exit 1
fi

# Step 4: Verdict must be pass.
verdict="$(jq -r '.verdict' "$ATTESTATION")"
if [[ "$verdict" != "pass" ]]; then
    echo "intent-check attestation verdict is '${verdict}' (expected 'pass'). Re-run /intent-check and resolve the mismatch." >&2
    exit 1
fi

# Step 5: Recompute content hash over the attestation's protected_files.
claimed_files="$(jq -r '.protected_files[]' "$ATTESTATION" | sort)"
claimed_hash="$(jq -r '.content_hash' "$ATTESTATION")"
actual_hash="$(cat $claimed_files | sha256sum | awk '{print $1}')"

if [[ "$actual_hash" != "$claimed_hash" ]]; then
    echo "intent-check attestation is stale — protected files have changed since the last run." >&2
    echo "  claimed: $claimed_hash" >&2
    echo "  actual:  $actual_hash" >&2
    echo "Re-run /intent-check and re-stage ${ATTESTATION}." >&2
    exit 1
fi

# Step 6: Ensure every staged protected file is in the attestation's set.
for f in $protected_staged; do
    if ! grep -qx "$f" <<<"$claimed_files"; then
        echo "intent-check attestation does not cover staged file: $f" >&2
        echo "Re-run /intent-check with the full set of protected files staged." >&2
        exit 1
    fi
done

exit 0
```

Notes on the pseudocode:

- `grep_against_patterns` is a placeholder for whatever the repo uses to match a path list against its `protected-surfaces.md` patterns. Most repos parse the section headers + bullet paths from that file; some use a dedicated `.protected-surfaces` list.
- `cat $claimed_files | sha256sum` replicates the "concat with no delimiter" rule. On repos with large protected files, use a streamed `sha256sum` loop to avoid memory spikes.
- The hook only checks files already in the attestation — if a staged protected file is missing from `protected_files`, Step 6 rejects the commit. This prevents partial-coverage exploits.
- The hook must not invoke `/intent-check` itself, must not call network, must not shell out to any LLM tool. It is deterministic by design.

## Registering the hook

The skill **describes** the hook but does **not install** it — installing hooks is a governance decision per repo. Tell the user to wire it up via their existing hook manager:

- **pre-commit (Python):** add a local repo stanza in `.pre-commit-config.yaml`:
  ```yaml
  - repo: local
    hooks:
      - id: intent-check-attestation
        name: intent-check attestation
        entry: scripts/check-intent-attestation.sh
        language: system
        pass_filenames: false
        stages: [commit]
  ```
- **lefthook:** add under `pre-commit.commands.intent-check-attestation`.
- **husky / simple-git-hooks:** add to the `pre-commit` command in `package.json`.
- **bare `.git/hooks/pre-commit`:** the simplest path. Copy the script to `.git/hooks/pre-commit` and `chmod +x` it. Not portable across fresh clones — prefer a managed hook.

## Why `pipeline_output` is a field (forgery cost)

An attentive reader will notice the `pipeline_output` field is semantically redundant — the pre-commit hook doesn't read it, only the hash + verdict. It lives in the attestation deliberately:

- **Forgery cost.** For an LLM (or a human) to forge a passing attestation without running the pipeline, they would need to (a) compute the correct SHA-256 over the exact file contents in the correct sort order, (b) produce a plausible `pipeline_output` matching what the back-translator would actually generate, and (c) JSON-encode it correctly. Part (b) is the load-bearing friction — it is cheaper to re-run `/intent-check` than to fabricate believable back-translation prose.
- **Human review.** When a reviewer audits whether a verdict is right, they can read the back-translation + diff result without re-running the pipeline.
- **Tracker reconciliation.** When a human fills the `human_verdict` column in the FP tracker, they need the verdict context. The attestation + tracker row together give them enough to decide.

## Interaction with `/protected-surface-amend`

If a reviewer decides that a `fail` verdict is actually the result of an intentional spec evolution rather than a drift, the correct remediation is NOT to hand-edit the attestation to `pass`. It is to:

1. Run `/protected-surface-amend` to produce the governance-note amendment block.
2. Amend the invariant prose (or the covering test) so that the next `/intent-check` run legitimately returns `match=true`.
3. Re-run `/intent-check` to get a fresh, legitimately-passing attestation.

Hand-editing the attestation is a governance bypass and should be blocked by the pre-commit hook's hash check in any case (the hash won't match).
