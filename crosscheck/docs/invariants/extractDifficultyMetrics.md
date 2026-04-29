# Invariants: `extractDifficultyMetrics`

**Source:** `mcp-server/src/tools/verify.ts`
**Covering tests:** `mcp-server/src/__tests__/property/extractDifficultyMetrics.prop.test.ts`
**Dafny spec:** `mcp-server/specs/extractDifficultyMetrics.dfy`
**Layer:** 4 (formal spec)

## Purpose

`extractDifficultyMetrics(source, rawOutput)` is a pure function. Given a Dafny
source string and the raw verifier output, it extracts five proof-difficulty
signals: solver wall-clock time, resource count, proof hint count, empty lemma
body count, and a derived `trivialProof` boolean.

These signals are downstream metadata used by callers to reason about proof
strength. The function does not mutate state and makes no IO calls.

## Invariants

### I1 — Non-negative counts

`proofHintCount` and `emptyLemmaBodyCount` are always non-negative integers.

```
result.proofHintCount  ≥ 0
result.emptyLemmaBodyCount ≥ 0
```

**Covering test:** `"counts are always non-negative"`

---

### I2 — Optional solver time is non-negative

`solverTimeMs` is either `null` (no timing data found) or a non-negative integer
representing milliseconds.

```
result.solverTimeMs = null  ∨  result.solverTimeMs ≥ 0
```

**Covering test:** `"solverTimeMs is null or >= 0"`

**Note:** The value is derived from `Math.round(parseFloat(match[1]) * 1000)`.
Since the regex only matches `\d+(?:\.\d+)?s` (a non-negative decimal), the
result is always ≥ 0 when present.

---

### I3 — Optional resource count is non-negative

`resourceCount` is either `null` (no resource count found in output) or a
non-negative integer.

```
result.resourceCount = null  ∨  result.resourceCount ≥ 0
```

**Covering test:** `"resourceCount is null or >= 0"`

---

### I4 — Lemma count requires lemma keyword

If the source string contains no occurrence of the substring `"lemma"`, then
`emptyLemmaBodyCount` is exactly 0. The empty-lemma regex (`/lemma\s+\w+[^{]*\{\s*\}/g`)
can only match when `"lemma"` appears in the source.

```
¬Contains(source, "lemma") → result.emptyLemmaBodyCount = 0
```

**Covering test:** `"no lemma keyword in source means emptyLemmaBodyCount === 0"`

---

### I5 — `trivialProof` is false when hints exist

`trivialProof` is `true` only when `proofHintCount == 0`. Equivalently, a non-zero
proof hint count always implies `trivialProof = false`.

```
result.trivialProof = true → result.proofHintCount = 0
result.proofHintCount > 0 → result.trivialProof = false
```

**Covering tests:**
- `"trivialProof is consistent: true only when proofHintCount === 0"`
- `"trivialProof is false when proofHintCount > 0"`

**Rationale:** The `trivialProof` flag is defined as:
`(proofHintCount === 0 && emptyLemmaBodyCount > 0) || (proofHintCount === 0 && solverTimeMs < 2000)`.
Both disjuncts require `proofHintCount === 0`.

## Carve-outs / known scope limits

- **Not covered:** the relationship between `solverTimeMs` and `trivialProof`
  boundary condition (`solverTimeMs < 2000`). Testing this requires generating
  rawOutput strings that contain timing data, which the property tests do not
  exhaustively cover.
- **Not covered:** I4 does not constrain `emptyLemmaBodyCount` for sources
  containing `"lemma"` as a substring of another identifier (e.g., `"dilemma"`).
  The regex uses `\w+` after `lemma` which may produce unexpected matches.
  This is an implementation-level caveat, not a spec relaxation.
- **Caller responsibility:** `dafnyVerify` trims the `rawOutput` before passing
  it to this function. The function does not normalise whitespace internally.
