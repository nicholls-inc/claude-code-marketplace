# Demo 6: Test Adequacy — SWE-bench_Pro qutebrowser parse_duration

**"100% line coverage. Zero duplicate-unit detection."**

## The bug

A duration string parser converts `'1h30m'` → milliseconds. The test
suite achieves 100% line coverage and all 20 tests pass. But it misses
a critical property: duplicate units like `'5s10s'` are silently accepted.
The regex-then-search approach takes the FIRST match (`5s`), ignoring
the second (`10s`). The user thinks they scheduled a 15-second delay
but got 5 seconds.

Source: SWE-bench_Pro `qutebrowser__qutebrowser-96b997802e942937e81d2b8a32d08f00d3f4bc4e`

## Setup (before the demo)

- No Docker needed
- Open Claude Code in this directory

## Step 1: Show the Green Test Suite (30s)

```bash
pytest test_parse_duration.py -v -k "not MissedProperties"
```

All 20 tests pass. Point out the coverage:
- Plain seconds (0, 1, 60, 3600)
- Unit formats (30s, 5m, 2h, compound)
- Invalid inputs (empty, negative, fractional, non-numeric)
- Integration with schedule_command()

> "Twenty tests, all green. Valid inputs, invalid inputs, integration
> tests. This looks comprehensive. Is it?"

## Step 2: Ask Vanilla Claude to Evaluate (1-2 min)

```
Look at parse_duration_bug.py and the test suite in test_parse_duration.py
(only the first 4 test classes — TestPlainSeconds, TestUnitFormats,
TestInvalidInputs, TestScheduleCommand).

The suite has 100% line coverage. Is it adequate? Are there any properties
of a duration parser that these tests don't verify?
```

**Expected vanilla response:** Claude typically says:
- "The test suite looks comprehensive"
- May suggest a few additional edge cases (empty string, very large values)
- Rarely identifies the DUPLICATE UNIT property as a gap
- Almost never connects the regex-then-search implementation to the
  specific vulnerability (first-match-wins on duplicate units)

**The real gap:** The tests never verify:
1. **Duplicate unit rejection**: `'5s10s'` should be invalid but returns 5000
2. **Order invariance**: `'1s1h'` == `'1h1s'` is never tested (works by accident)
3. **Unit-prefixed zeros**: `'0s'`, `'0m'`, `'0h'` are never tested

## Step 3: Use /rationale (2-3 min)

```
/rationale "Evaluate whether the test suite for parse_duration() in
test_parse_duration.py (classes TestPlainSeconds, TestUnitFormats,
TestInvalidInputs, TestScheduleCommand) adequately verifies the
correctness of parse_duration() in parse_duration_bug.py.

The function must:
1. Parse valid duration strings correctly
2. Reject all invalid inputs with -1
3. Handle edge cases without silent data loss"
```

**What the audience sees — claim tree:**

**Root claim:** "parse_duration is correctly implemented and tested"

**Leaf 1:** "Valid inputs produce correct millisecond values"
- Verification: behavioral (tests exist) — ✓ COVERED
- Tests: 12 parametrized cases covering all unit combinations

**Leaf 2:** "Invalid inputs are rejected with -1"
- Verification: behavioral (tests exist) — ✓ COVERED
- Tests: 6 cases (empty, negative, fractional, non-numeric, None)

**Leaf 3:** "Ambiguous inputs are rejected"
- Verification: **GAP** — NO TEST COVERAGE
- The regex `^([0-9]+[hms])+$` accepts `'5s10s'` (valid syntax)
- `re.search('([0-9]+)s', '5s10s')` returns the FIRST match (`5`)
- The second `10s` is silently discarded
- **This is silent data loss** — violates requirement 3

**Leaf 4:** "Unit order does not affect result"
- Verification: **GAP** — NO TEST COVERAGE
- `'1s1h'` should equal `'1h1s'` (commutativity property)
- Works correctly by accident (regex search is order-independent)
- But no test verifies this — a refactor could break it silently

**Leaf 5:** "Boundary values are tested for all unit types"
- Verification: **PARTIAL GAP**
- `'0'` is tested but `'0s'`, `'0m'`, `'0h'` are not
- These exercise different code paths (unit format vs plain integer)

**Key moment:** When the claim tree reaches Leaf 3:

> "Here's the killer gap: the regex accepts `'5s10s'` as valid syntax.
> Then `re.search` grabs the FIRST `s` match — `5s` — and ignores
> `10s`. The function returns 5000 instead of -1.
>
> That's not an exotic edge case. That's a user typing `'5s10s'` and
> getting a 5-second delay instead of an error. The test suite has
> 100% line coverage but ZERO duplicate-unit tests."

## Step 4: Demonstrate the Bug (30s)

```bash
python -c "from parse_duration_bug import parse_duration; print(parse_duration('5s10s'))"
# Output: 5000  (should be -1)

python -c "from parse_duration_bug import parse_duration; print(parse_duration('1h2h'))"
# Output: 3600000  (1h, ignores 2h — should be -1)
```

```bash
pytest test_parse_duration.py::TestMissedProperties::test_duplicate_units_rejected -v
```

> "The /rationale certificate didn't just say 'add more tests.' It
> identified a SPECIFIC gap — duplicate unit handling — connected it
> to the regex implementation, and showed that it causes silent data
> loss. That's the difference between 'improve coverage' and 'here's
> the exact property your tests don't verify.'"

## Timing Budget

| Step | Duration |
|------|----------|
| Show green tests | 0:30 |
| Vanilla Claude | 1:30 |
| /rationale | 2:30 |
| Demonstrate bug | 0:30 |
| **Total** | **~5:00** |

## Contingencies

**Claude identifies the duplicate-unit gap:**
"Good catch. But the /rationale certificate is a STRUCTURED analysis —
it builds a claim tree, classifies each leaf by verification method,
and identifies ALL gaps, not just the most obvious one. The order
invariance gap and the boundary gap are also real."

**/rationale focuses on coverage metrics instead of properties:**
"Coverage measures which lines ran. /rationale measures which
PROPERTIES are verified. 100% line coverage with 0% duplicate-unit
coverage is a false sense of security. The claim tree makes the
difference visible."
