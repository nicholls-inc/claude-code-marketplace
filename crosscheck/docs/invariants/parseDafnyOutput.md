# Invariants: `parseDafnyOutput`

**Source:** `mcp-server/src/tools/verify.ts`
**Covering tests:** `mcp-server/src/__tests__/property/parseDafnyOutput.prop.test.ts`
**Dafny spec:** `mcp-server/specs/parseDafnyOutput.dfy`
**Layer:** 4 (formal spec)

## Purpose

`parseDafnyOutput(stdout, stderr)` classifies non-empty lines from the combined
Dafny output stream into two disjoint lists: `errors` and `warnings`. It is a
pure function — no IO, no side effects, deterministic.

## Invariants

### I1 — Error membership

Every string in `errors` contains the substring `"error"` (case-insensitive).

```
∀ e ∈ result.errors → ContainsErrorCI(e)
```

**Covering test:** `"every string in errors contains 'Error' (case-insensitive)"`

---

### I2 — Warning membership

Every string in `warnings` contains the substring `"warning"` (case-insensitive).

```
∀ w ∈ result.warnings → ContainsWarningCI(w)
```

**Covering test:** `"every string in warnings contains 'Warning' (case-insensitive)"`

---

### I3 — Disjointness

No line appears in both `errors` and `warnings`. The two output lists are disjoint.

```
∀ s ∈ result.errors → s ∉ result.warnings
```

**Covering test:** `"no line appears in both errors and warnings"`

---

### I4 — Error takes precedence over Warning

A line matching both `Error` (CI) and `Warning` (CI) is placed in `errors` only,
never in `warnings`. The error-path check runs first and short-circuits.

```
∀ line: ContainsErrorCI(line) ∧ ContainsWarningCI(line)
    → line ∈ result.errors ∧ line ∉ result.warnings
```

**Covering test:** `"lines with both Error and Warning go to errors only (Error takes precedence)"`

**Note:** I4 is a stronger form of I3 restricted to mixed-marker lines. It is
separately valuable because it encodes the parsing priority rule.

---

### I5 — Verifier summary exclusion

Lines whose trimmed prefix matches `/^Dafny program verifier/` are excluded from
`errors` even when they contain the substring `"Error"`. These lines are the
Dafny verifier summary (e.g., `"Dafny program verifier finished with 1 error"`).

```
∀ e ∈ result.errors → ¬IsVerifierSummaryLine(e)
```

**Covering test:** `"'Dafny program verifier' lines are excluded from errors"`

**Rationale:** The verifier summary line is metadata about the verification run,
not a source-level error. Conflating it with errors causes false positives in the
`success` flag computation.

---

### I6 — Empty input gives empty output

When both `stdout` and `stderr` are the empty string, both output lists are empty.

```
stdout = "" ∧ stderr = "" → result.errors = [] ∧ result.warnings = []
```

**Covering test:** `"empty input gives empty arrays"`

---

### I7 — Categorization is bounded by input

The total number of categorized lines (errors + warnings) never exceeds the total
number of non-empty lines in the combined input.

```
|result.errors| + |result.warnings| ≤ NonEmptyLineCount(stdout + "\n" + stderr)
```

**Covering test:** `"total categorized lines never exceed total non-empty input lines"`

**Note:** Some lines (e.g., clean lines with no markers) are intentionally uncategorized.
I7 guarantees the parser never invents lines that do not appear in the input.

## Carve-outs / known scope limits

- **Not covered:** the function does not validate that error messages are
  syntactically well-formed Dafny diagnostics. Structural parsing of
  `file.dfy(line,col): Error` is out of scope for I1–I7.
- **Not covered:** line deduplication. The same error string can appear in
  `errors` multiple times if the combined output contains it multiple times.
- **Caller responsibility:** `dafnyVerify` is responsible for concatenating
  `stdout` and `stderr` in the correct order before calling this function.
