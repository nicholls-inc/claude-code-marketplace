---
name: check-regressions
description: >-
  Scan the spec registry for verified Dafny specs whose associated source files have
  changed. Re-verify affected specs and report which properties still hold. Use for
  regression detection after code changes, pre-commit checks, or spec maintenance.
  Triggers: "check regressions", "did my changes break specs", "re-verify",
  "spec status", pre-commit review.
argument-hint: "[optional: spec-id or --hard-only]"
---

# /check-regressions — Spec Registry Regression Detection

## Description

Scan the spec registry for verified Dafny specifications whose associated source files have changed since last verification. Re-verify affected specs and report which properties still hold and which need attention.

## Instructions

You are a formal verification expert helping the user detect when code changes have potentially invalidated previously-verified properties. The spec registry (`.crosscheck/specs.json`) tracks which functions have verified Dafny specs and when they were last checked.

### Step 1: Load Registry

Look for `.crosscheck/specs.json` in the project root.

**If the registry does not exist:**
- Inform the user: "No spec registry found. The registry is created when you use `/extract-code` to extract verified Dafny code. Run `/spec-iterate` → `/generate-verified` → `/extract-code` to verify a function and register its spec."
- Stop here.

**If the registry exists but is empty (no specs):**
- Inform the user: "The spec registry exists but contains no entries. Use `/extract-code` to extract verified code — it will offer to register the spec."
- Stop here.

Read the registry and summarize what's tracked:
- Number of specs registered
- Languages covered (Python, Go)
- How many are hard vs. soft constraints

### Step 2: Detect Changes

For each spec entry in the registry, check whether the extracted code file has been modified since `lastVerified`:

1. **Use git diff** — Run `git diff --name-only` comparing the file's state at `lastVerified` to HEAD. If the extracted code file appears in the diff, it has changed.
2. **Check Dafny source** — Compare the current file's SHA-256 hash against `dafnySourceHash`. If the spec itself changed, flag it separately (the spec may need re-review, not just re-verification).

Classify each spec into one of:

| Status | Meaning |
|--------|---------|
| **UNCHANGED** | Neither the extracted code nor the Dafny source has changed |
| **CODE_CHANGED** | The extracted code file was modified; Dafny spec is the same |
| **SPEC_CHANGED** | The Dafny source file was modified (hash mismatch) |
| **BOTH_CHANGED** | Both the extracted code and the Dafny source have changed |
| **MISSING** | The extracted code file or Dafny source file no longer exists |

Present a summary table:

```
## Change Detection

| Spec ID | Function | Status | Extracted Code | Last Verified |
|---------|----------|--------|----------------|---------------|
| max-of-array | MaxOfArray | CODE_CHANGED | src/utils.py | 2026-03-10 |
| split-energy | SplitEnergy | UNCHANGED | billing/calc.py | 2026-03-12 |
| validate-token | ValidateToken | MISSING | auth/tokens.py (deleted) | 2026-03-08 |
```

### Step 3: Re-verify Affected Specs

For each spec with status CODE_CHANGED, SPEC_CHANGED, or BOTH_CHANGED:

**Hard constraints (`constraint: "hard"`):**
1. Read the Dafny source file
2. Call `dafny_verify` with the Dafny source
3. If verification **passes**: the property still holds despite code changes
4. If verification **fails**: generate a structured diagnostic (see Step 4)

**Soft constraints (`constraint: "soft"`):**
1. Note that soft constraints are verified via property-based tests, not Dafny
2. Check if the property-based test file still exists
3. Suggest the user run the property-based tests to confirm

**SPEC_CHANGED or BOTH_CHANGED entries:**
- Warn: "The Dafny specification itself has changed. This may indicate an intentional spec evolution — verify that the new spec still captures the intended properties."
- Re-verify with the updated Dafny source

**MISSING entries:**
- Warn: "The extracted code file or Dafny source has been deleted. If the function was removed intentionally, consider removing this entry from the registry."

### Step 4: Generate Diagnostics for Failures

For each spec that fails re-verification, produce a structured diagnostic:

```
### Regression: [spec-id]

**Function:** `function_name` at `file:line`
**Failed postcondition:** `ensures result[i] <= result[i+1]` (or specific failing condition)
**What changed:** Brief description of the relevant diff in the extracted code
**Suggested action:**
- Re-run `/spec-iterate` to update the spec for the new behavior, OR
- Revert the code change if the original property should be preserved
```

### Step 5: Update Registry (write-by-default)

For specs that passed re-verification:
- Update `lastVerified` to the current timestamp.
- Update `dafnySourceHash` if the spec changed.

**Write the updated registry to `.crosscheck/specs.json` directly.** Persisting the regression-check result is the whole point of the run; pausing to ask for permission to save the result is admin theater. The user can `git diff` / `git revert` if they don't want the update committed.

If the user passed `--dry-run`, skip the write and report the diff that would have been applied.

### Step 6: Actions Taken and Decisions for Review

Two blocks — admin work the agent performed (Actions Taken) and judgment items the human owns (Decisions for Review). Per byfuglien's rule, never mix them.

```
## Regression Check Summary

- **Total specs:** N
- **Unchanged:** N (no action needed)
- **Re-verified (pass):** N (registry updated)
- **Re-verified (fail):** N (see diagnostics above)
- **Missing:** N (deletion candidates)

## Actions Taken (agent did these during the run)

- Re-verified <N> CODE_CHANGED specs via dafny_verify; each has a per-spec diagnostic in Step 4 above.
- Updated `lastVerified` timestamps for <N> passing specs.
- Updated `dafnySourceHash` for <N> SPEC_CHANGED entries that re-verified clean.
- Wrote `.crosscheck/specs.json` (skip with `--dry-run`).

## Decisions for Review (human owns these)

- [ ] Re-verified-fail specs: pick remediation per Step 4 diagnostic (update spec OR revert code OR document as intentional behavior change).
- [ ] SPEC_CHANGED entries that re-verified clean: confirm the spec evolution was intentional, not accidental drift.
- [ ] MISSING entries: remove from registry if the function was intentionally deleted; otherwise restore the source file.
- [ ] Soft-constraint entries: confirm their property-based tests still pass (the agent does not run them in this skill).
```

## Arguments

Optional filter for which specs to check.

Examples:
- `/check-regressions` — check all registered specs
- `/check-regressions max-of-array` — check only the `max-of-array` spec
- `/check-regressions --hard-only` — check only hard constraint specs
