# Demo 3: Structured Reasoning — SWE-bench_Pro ansible ensure_type

**"Five interacting bugs in one function. Can you find them all?"**

## The bug

A type coercion function converts values between types (string→int,
string→bool, etc.) while preserving metadata tags that track value
origin. The function has 5 interacting bugs:

1. **bool→int loses the tag**: `True` stays as `bool`, not converted to `int(1)`
2. **Unhashable types crash boolean conversion**: `[1,2,3]` in `frozenset` → `TypeError`
3. **bytes treated as Sequence**: `list(b'hello')` → `[104, 101, ...]` instead of error
4. **Floats silently truncate to int**: `Decimal('1.7')` → `1` instead of error
5. **String floats silently truncate**: `'3.14'` → `3` instead of error

Source: SWE-bench_Pro `ansible__ansible-d33bedc48fdd933b5abd65a77c081876298e2f07`

## Setup (before the demo)

- No Docker needed
- Open Claude Code in this directory

## Step 1: Show the Passing Tests (30s)

```bash
pytest test_coerce.py::TestBasicCoercion -v
```

All 11 tests pass. String→int, string→bool, list passthrough — everything looks fine.

> "This type coercion function is from a configuration management system.
> Values carry metadata tags tracking where they came from — 'env', 'cli',
> 'file'. The basic tests all pass. The function looks correct."

## Step 2: Ask Vanilla Claude to Review (1-2 min)

```
Look at coerce_value_bug.py. This function converts values between types
while preserving metadata tags. The basic test suite passes. Are there any
bugs in this function?
```

**Expected vanilla response:** Claude typically catches 1-2 issues:
- Might notice the bool/int issue (since `isinstance(True, int)` is True)
- Might mention bytes as a Sequence
- Rarely catches the float truncation via Decimal
- Almost never identifies ALL 5 bugs systematically

**If Claude finds all 5:** Rare. Pivot to: "Claude found them, but
notice there's no structured evidence — no invariant statement, no
systematic path enumeration. The /reason certificate makes the analysis
auditable."

## Step 3: Use /reason (2-3 min)

```
/reason "The coerce_value() function in coerce_value_bug.py must satisfy
these properties:
1. Tags are preserved through ALL type conversion paths
2. Type conversion never silently loses precision
3. Invalid inputs raise ValueError, never TypeError or unexpected results
4. The output type matches the expected_type parameter

Is the function correct with respect to these properties?"
```

**What the audience sees — structured reasoning:**

**Premises** (gathered with file:line evidence):
- P1: `_unwrap` extracts tag at line 48, `_rewrap` re-applies at line 53
- P2: `boolean` path has `raw in BOOLEANS_TRUE` at line 96 — `in` operator calls `__hash__`
- P3: `integer` path for `bool` at line 103 returns `_rewrap(raw, tag)` where `raw` is still `bool`
- P4: `integer` path for `float` at line 110 does `int(Decimal(str(raw)))` — truncates
- P5: `list` path at line 126 does `isinstance(raw, Sequence)` — `bytes` is a `Sequence`

**Execution trace through each property:**

Property 1 (tag preservation):
- string→bool: ✓ (line 91-95 rewrap correctly)
- string→int: ✓ (line 110-112 rewrap correctly)
- **bool→int: ✗** — line 103 `isinstance(raw, bool)` matches, returns `_rewrap(raw, tag)` but `raw` is still `True`/`False`, not converted to `int`
- Result: bool stays bool, tag preserved but TYPE is wrong

Property 2 (no silent precision loss):
- **float→int: ✗** — `int(Decimal('1.7'))` returns `1`. No check for nonzero remainder
- **string→int: ✗** — `int(Decimal('3.14'))` returns `3` via same path

Property 3 (invalid inputs → ValueError):
- **list→bool: ✗** — `[1,2,3] in frozenset(...)` throws `TypeError`, not `ValueError`
- **bytes→list: ✗** — `list(b'hello')` returns `[104, 101, 108, 108, 111]`, should be rejected

**Alternative hypotheses checked:**
- "Maybe bool→int is intentional?" — No, the function's contract says "convert to expected type". `bool` is not `int`.
- "Maybe bytes→list of ints is useful?" — No, the function is for config values. A byte string is not a list of integers.

**Conclusion:** 5 bugs found across 4 properties. Each with file:line evidence.

**Key moment:** When the premises enumerate all code paths:

> "Look at this: the /reason certificate forced Claude to enumerate
> EVERY type conversion path and check EVERY property against it.
> That's how it found the Decimal truncation — by tracing the float→int
> path through str→Decimal→int and checking if precision is preserved.
> Without the structure, Claude stops after finding the obvious bool bug."

## Step 4: Show All 5 Failing Tests (30s)

```bash
pytest test_coerce.py -v -x
```

> "Five bugs, five failing tests. The /reason certificate found them all
> because it was forced to check every property against every code path.
> That's structured reasoning — not 'look at the code and tell me what's
> wrong', but 'here are the invariants, prove they hold for every path.'"

## Timing Budget

| Step | Duration |
|------|----------|
| Show passing tests | 0:30 |
| Vanilla Claude | 1:30 |
| /reason | 2:30 |
| Show failing tests | 0:30 |
| **Total** | **~5:00** |

## Contingencies

**Claude finds all 5 bugs without /reason:**
"Good catch — but count the evidence citations. How many file:line
references? How many alternative hypotheses checked? The /reason
certificate isn't just about the answer — it's about showing your
work in a way that's auditable."

**/reason misses one bug:**
"Even structured reasoning isn't perfect. But notice it found 4 out
of 5 bugs versus Claude's typical 1-2. And each finding comes with
the exact line number and the property it violates."
