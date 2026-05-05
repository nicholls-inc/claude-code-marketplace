---
name: assurance-probe
description: >-
  Deterministic test-strength probe for covered invariants. Parses
  docs/invariants/<module>.md, locates the test files that cover each
  non-aspirational invariant via "// Invariant <ID>:" comments, applies a
  5-dimension strength rubric based solely on observable file content, and
  emits a structured strength table sorted weakest-first. No LLM inference is
  used during rubric evaluation — every dimension maps to grep-able keywords
  or integer counts with explicit thresholds. Triggers: "test strength", "how
  strong are the tests", "probe invariant coverage", "weak tests".
argument-hint: "<module-name> [--top=N]  (N defaults to 5)"
---

# /assurance-probe — Deterministic Test-Strength Probe

## Description

Measure how strong the test coverage is for each invariant documented in
`docs/invariants/<module>.md`. Produces a structured strength table and a
"weakest-first" action list. All scoring is deterministic: the rubric maps
only to observable file content — keyword presence and integer counts — with
no LLM reasoning step during evaluation.

The skill is read-only. It does not modify files, run tests, or scaffold
anything. Remediation is delegated to sibling skills.

---

## Instructions

You are the test-strength probe. Follow the phases below exactly. Never skip
Phase 1.

---

### Phase 1 — Onboarding Gate (deterministic, no LLM reasoning)

This phase is a mechanical file-existence check. Do not interpret, infer, or
reason about file contents in Phase 1 — only check for the presence of the
three required artifacts. If any are missing, emit the verbatim refusal
message and stop.

#### Step 1.1: Check invariant doc

Verify that `docs/invariants/<module>.md` exists (where `<module>` is the
argument the user supplied).

- **Hard fail** if missing.

#### Step 1.2: Check ROADMAP

Verify that `docs/assurance/ROADMAP.md` exists.

- **Hard fail** if missing.

#### Step 1.3: Check protected-surfaces rules

Verify that `.claude/rules/protected-surfaces.md` exists.

- **Hard fail** if missing.

#### Step 1.4: Gate decision

If any check in 1.1–1.3 hard-failed, collect the names of every failing
artifact and emit verbatim:

```
Repo not onboarded. Missing: <comma-separated list of missing artifacts>.
Next: /assurance-init.
```

Then **stop**. Do not emit a strength table. An empty strength table is
indistinguishable from "all tests are strong" — the refusal message is the
only safe output for an unonboarded repo.

---

### Phase 2 — Parse Invariant Doc

Read `docs/invariants/<module>.md`. Extract every invariant ID declared in
the document.

**Extraction rules:**

1. An invariant ID is any token matching the pattern `I[0-9]+` or
   `IN[0-9]+` (e.g. `I1`, `I12`, `IN3`) that appears in a heading or
   definition line (not in prose sentences mid-paragraph).
2. **Exclude aspirational invariants:** any invariant ID that appears on a
   line containing the annotation `<!-- aspirational -->` (as an HTML
   comment, anywhere on the line, case-insensitive) is excluded from the
   probe scope. Do not emit a strength row for an aspirational invariant —
   it has not been committed to and its absence from test files is expected.
3. Record the set of active (non-aspirational) invariant IDs. If the set is
   empty after exclusion, emit: `No active invariants found in
   docs/invariants/<module>.md after excluding aspirational items.` and stop.

---

### Phase 3 — Locate Covering Tests

For each active invariant ID, search the repo's test files for covering
comments.

**Covering comment pattern:**

```
// Invariant <ID>:
```

(Whole-line or inline comment; the colon is required; `<ID>` is the exact
invariant ID string, e.g. `// Invariant I3:`.)

Treat any file containing this pattern for the given ID as a covering file
for that invariant.

**Zero-coverage invariants:** If no test file contains the covering comment
for an invariant, record the invariant in the output table with:

- test file(s): `—` (em dash)
- strength score: `0`
- strength label: `uncovered`
- all per-dimension scores: `0`
- gap description: `no covering test found`

Zero-coverage invariants are included in the output table and ranked highest
priority in the action list.

---

### Phase 4 — Apply the Strength Rubric

For each active invariant that has at least one covering test file, score
the test(s) using the 5-dimension rubric below.

#### Multi-file coverage: weakest-wins aggregation

When an invariant is covered by more than one test file:

1. Compute the 5-dimension score for **each** covering file independently.
2. Identify the **minimum total score** across all covering files.
3. Use that minimum as the emitted strength score.
4. The "test file(s)" column lists **all** covering files as a
   comma-separated list (sorted alphabetically for determinism).
5. Per-dimension evidence (gap descriptions) is sourced from the
   weakest-scoring file (i.e. the file that produced the minimum total).
   If two files tie for the minimum, use the lexicographically earlier
   filename.

This rule is order-independent: the minimum of a set of integers does not
depend on the order in which the files are read.

#### Dimension 1 — Boundary/edge-case coverage (0–1 pt)

Score = 1 if the test file body contains at least one of the following as a
whole-word or token match (case-insensitive unless noted):

```
min, max, empty, zero, nil, null, boundary, edge, -1, [], {}
```

Score = 0 otherwise.

Gap description (when 0): `no boundary/edge-case markers found`

#### Dimension 2 — Property-based vs example-based (0–1 pt)

Score = 1 if the test file contains at least one of the following as a
literal string match (case-sensitive):

```
@given, @example, fc., fast-check, gopter, rapid.Make, quickcheck, prop.ForAll
```

Score = 0 otherwise.

Gap description (when 0): `no property-based testing framework markers found`

#### Dimension 3 — Assertion post-condition scope (0–1 pt)

Count occurrences of the following assertion keywords anywhere in the test
file (case-sensitive):

```
assert, assertEqual, assertRaises, expect(, should., Must, t.Error, t.Fatal
```

Score = 1 if count ≥ 3, else 0.

Gap description (when 0): `fewer than 3 assertion keywords found (count: <N>)`

When the count is zero, the gap description is: `no assertions found`

#### Dimension 4 — Mutation probe hint (0–1 pt)

Score = 1 if the test file contains at least one of the following as a
whole-word match anywhere in the file (case-insensitive):

```
mutmut, pitest, stryker, #mutant, # mutant, @mutant, mutpy
```

Score = 0 otherwise.

This dimension is a **hint**, not a disqualifier. A zero score on this
dimension reduces the total by 1 point but does not prevent a score of 4
(strong). Gap description (when 0): `no mutation probe markers found`

#### Dimension 5 — Composite scenario breadth (0–1 pt)

Count the number of distinct test functions or test methods in the file by
counting occurrences of any of the following patterns:

```
def test_        (Python)
func Test        (Go)
it(              (JavaScript/TypeScript)
test(            (JavaScript/TypeScript)
describe(        (JavaScript/TypeScript)
```

Score = 1 if the count ≥ 2, else 0.

Gap description (when 0): `fewer than 2 distinct test functions/methods found`

#### Total score and strength label

Sum dimensions 1–5. Map to strength label:

| Score | Label |
|-------|-------|
| 0 | uncovered |
| 1 | minimal |
| 2 | weak |
| 3 | moderate |
| 4 | strong |
| 5 | comprehensive |

---

### Phase 5 — Emit Strength Table

Emit the strength table, sorted **weakest first** (ascending by total score,
then alphabetically by invariant ID within the same score).

```
## Test Strength Report — <module> — <today's date>

| Invariant | Test file(s) | Score | Label | D1 | D2 | D3 | D4 | D5 | Gap |
|-----------|-------------|-------|-------|----|----|----|----|----|-----|
| I1        | tests/foo.py | 2    | weak  | 1  | 0  | 1  | 0  | 0  | no property-based testing framework markers found; no mutation probe markers found |
| I2        | —            | 0    | uncovered | 0 | 0 | 0 | 0 | 0 | no covering test found |
| ...
```

Column meanings:
- **D1**: Boundary/edge-case coverage (0–1)
- **D2**: Property-based vs example-based (0–1)
- **D3**: Assertion post-condition scope (0–1)
- **D4**: Mutation probe hint (0–1)
- **D5**: Composite scenario breadth (0–1)
- **Gap**: Semicolon-separated list of gap descriptions for all dimensions that scored 0.

If all invariants score 5 (comprehensive), emit: `All active invariants score 5/5 — no gaps found.`

---

### Phase 6 — Emit Action List

Emit the top-N weakest invariants (default N=5, override with `--top=<N>`)
as a prioritised action list:

```
## Weakest-First Action List (top <N>)

1. **I2** — uncovered — no covering test found
   Recommended: add a test with `// Invariant I2:` comment.

2. **I1** — weak (2/5) — no property-based testing framework markers found; no mutation probe markers found
   Recommended: add property-based tests (@given / fc. / gopter) and consider running a mutation probe.

...
```

Skip invariants that score 5; they need no action. If fewer than N
invariants score below 5, list all of them.

---

### Step 7: Verification Checklist

```
## Verification Checklist

- [ ] Phase 1 gate was run before any Phase 2–6 output was generated
- [ ] All three gate artifacts were checked by file existence only, not interpretation
- [ ] If any gate artifact was missing, the verbatim refusal message was emitted and no strength table was produced
- [ ] Aspirational invariants (lines containing <!-- aspirational -->) were excluded from the probe scope
- [ ] Zero-coverage invariants are included in the table with score=0 and label="uncovered"
- [ ] Multi-file coverage used the weakest-wins (minimum total score) aggregation rule
- [ ] All covering files are listed in the test file(s) column, sorted alphabetically
- [ ] Per-dimension evidence was sourced from the weakest-scoring file (lexicographically earlier on a tie)
- [ ] Each dimension score was derived from observable keyword presence or integer counts only — no LLM inference was used
- [ ] Strength table is sorted weakest first, then alphabetically by invariant ID within the same score
- [ ] Action list covers top-N invariants below score 5, where N is the argument (default 5)
- [ ] No files were modified — this skill is read-only
```

---

## Arguments

- `<module-name>` (required) — the module whose invariant doc is at
  `docs/invariants/<module-name>.md`.
- `--top=<N>` (optional, default 5) — how many invariants to include in
  the action list.

Examples:

- `/assurance-probe auth` — probe the `auth` module.
- `/assurance-probe payments --top=10` — probe payments, show top 10
  weakest invariants.
