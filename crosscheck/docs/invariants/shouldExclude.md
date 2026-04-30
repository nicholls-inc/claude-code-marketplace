# Invariants: `shouldExclude`

**Source:** `mcp-server/src/tools/compile.ts`
**Covering tests:** `mcp-server/src/__tests__/property/shouldExclude.prop.test.ts`
**Dafny spec:** `mcp-server/specs/shouldExclude.dfy`
**Layer:** 4 (formal spec)

## Purpose

`shouldExclude(filePath, target)` is a pure boolean predicate. It returns `true`
when a file path should be omitted from the compiled output collected after a
Dafny translate run. The exclusion rules are target-specific (Python vs Go) and
eliminate Dafny runtime boilerplate files that are useless — or actively harmful —
in the extracted output.

## Invariants

### I1 — Go: `/dafny/` directory paths are excluded

For the `"go"` target, any path that contains the segment `/dafny/` (i.e., a
directory named exactly `dafny`) returns `true`.

```
∀ filePath: Contains(filePath, "/dafny/") → ShouldExclude(filePath, "go")
```

**Covering test:** `"paths containing /dafny/ are excluded for go target"`

**Rationale:** The Dafny compiler emits a `dafny/` runtime support directory in
the Go output; its contents are boilerplate that callers should not inspect or
ship.

---

### I2 — Go: `/System_/` directory paths are excluded

For the `"go"` target, any path that contains the segment `/System_/` returns `true`.

```
∀ filePath: Contains(filePath, "/System_/") → ShouldExclude(filePath, "go")
```

**Covering test:** `"paths containing /System_/ are excluded for go target"`

---

### I3 — Python: `_dafny.py` paths are excluded

For the `"py"` target, any path whose basename equals `_dafny.py`, or that
contains the substring `_dafny.py`, returns `true`.

```
∀ filePath: Contains(filePath, "_dafny.py") → ShouldExclude(filePath, "py")
```

**Covering test:** `"paths containing _dafny.py are excluded for py target"`

---

### I4 — Python: `__pycache__` paths are excluded

For the `"py"` target, any path that contains the string `__pycache__` returns `true`.

```
∀ filePath: Contains(filePath, "__pycache__") → ShouldExclude(filePath, "py")
```

**Covering test:** `"paths containing __pycache__ are excluded for py target"`

---

### I5 — Safe filenames are never excluded

A filename consisting only of lowercase ASCII letters (no special characters) with
a `.py` or `.go` extension, with no path components matching any exclude pattern,
returns `false` for the corresponding target.

```
∀ name ∈ SafePyFilenames: ¬ShouldExclude(name, "py")
∀ name ∈ SafeGoFilenames: ¬ShouldExclude(name, "go")
```

where `SafePyFilenames = { s + ".py" | s ∈ [a-z]{1,8} }` and
`SafeGoFilenames = { s + ".go" | s ∈ [a-z]{1,8} }`.

**Covering tests:**
- `"safe Python filenames are not excluded for py target"`
- `"safe Go filenames are not excluded for go target"`

---

### I6 — Target isolation

Python exclude patterns do not affect the `"go"` target, and vice versa.
Specifically:
- `_dafny.py` (as a bare filename, not in a path) is **not** excluded for `"go"`.
- `__pycache__` (as a bare filename) is **not** excluded for `"go"`.
- `dafny.go` and `System_.go` (as bare filenames) are **not** excluded for `"py"`.

```
¬ShouldExclude("_dafny.py", "go")
¬ShouldExclude("__pycache__", "go")
¬ShouldExclude("dafny.go", "py")
¬ShouldExclude("System_.go", "py")
```

**Covering tests:**
- `"Python exclude filenames do not affect go target (basename check)"`
- `"Go exclude filenames do not affect py target"`

**Rationale:** The exclusion lists are entirely disjoint by target. A Go output
directory will not contain Python files, and vice versa. Cross-target exclusion
would indicate a bug in the pattern matching.

## Carve-outs / known scope limits

- **Caller responsibility:** `shouldExclude` is called by `collectFiles`, which
  provides the `filePath` relative to the temp directory root. The predicate does
  not validate that the path is well-formed or that the file exists on disk.
- **Not covered:** extension matching. `shouldExclude` only filters by name/path
  patterns; extension filtering (`.py` vs `.go`) is done by `collectFiles` before
  calling `shouldExclude`.
- **Not covered:** recursive directory exclusion for `__pycache__`. The current
  implementation checks `filePath.includes("__pycache__")`, which catches files
  inside `__pycache__/` subdirectories transitively.
