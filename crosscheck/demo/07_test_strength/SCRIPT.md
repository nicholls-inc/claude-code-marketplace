# Demo: Test Strength Measurement with /assurance-probe

**Timing**: ~5 minutes  
**Layer**: 4 (Deterministic strength)  
**Phase**: 1 (Python mutation probe)

## Setup

Create a minimal Python module with an invariant and a property-based test:

**1. Create module** — `demo_validator.py`:

```python
def validate_input(x: int) -> bool:
    """
    Validates that input is non-negative.
    
    Returns True if x >= 0, False otherwise.
    """
    return x >= 0
```

**2. Create invariant doc** — `invariants/demo_validator.md`:

```markdown
# Invariants for demo_validator

## validate_input

**Failure condition**: `x < 0`

The function must return False when given a negative integer.
```

**3. Create test** — `tests/test_demo_validator.py`:

```python
from hypothesis import given
from hypothesis.strategies import integers
from demo_validator import validate_input


@given(integers(min_value=1, max_value=100))
def test_validate_input(x):
    """Test that validate_input handles positive integers correctly."""
    # This test is deliberately weak — it only tests positive integers
    result = validate_input(x)
    assert result is True
```

## Scenario 1: Mutation killed (strong test)

**Run**: `/assurance-probe demo_validator`

**Expected output**:
```
No test-strength issues found for module `demo_validator` (1 invariant tested, 0 mutations survived).
```

**Why**: The test covers `x >= 1`, which should kill boundary mutations like `x >= 0` → `x > 0`.

**Wait, that's wrong!** Let's fix the test to be actually weak:

```python
@given(integers(min_value=1, max_value=100))
def test_validate_input(x):
    """Test that validate_input handles positive integers correctly."""
    result = validate_input(x)
    # Bug: we never test x=0
    assert result is True
```

Now re-run the probe.

## Scenario 2: Mutation survived (weak test)

**Run**: `/assurance-probe demo_validator`

**Expected output** (GitHub issue created):

```markdown
# Test strength findings for `demo_validator`

**Probe run**: 2026-05-05 14:32 UTC  
**Commit**: abc123...  
**Environment**: Python 3.11, pytest 7.4.0, hypothesis 6.92.0  
**Phase**: 1 (mutation probe)

---

## Finding 1 of 1: Mutation survived

**Invariant**: invariants/demo_validator.md  
**Failure condition**: `x < 0`  
**Mutation**:
```diff
- return x >= 0
+ return x > 0
```

**Test command**: `pytest tests/test_demo_validator.py::test_validate_input`  
**Observed**: Mutation survived (test passed on mutated code)

**Explanation**: The test only generates `x >= 1`, so it never exercises the boundary `x = 0`. The mutation `x >= 0` → `x > 0` changes behavior at `x = 0`, but the test doesn't detect it.

**Reproducer**: `scripts/probe/demo_validator_20260505.py`

**Triage**:
- [x] Accept (test is too weak; fix test generator to include x=0)
- [ ] Reject (false positive; mutation unreachable by generator)
- [ ] Defer (requires Phase 3 generator probe to confirm)

---

**SNR tracking**: See `.assurance/probe-tracker.csv`

Current run: 1 proposed, 1 accepted, 0 rejected, 0 deferred, 0 skipped
```

## Scenario 3: Fix the test

**Edit** `tests/test_demo_validator.py`:

```python
@given(integers(min_value=0, max_value=100))  # Now includes x=0
def test_validate_input(x):
    """Test that validate_input handles non-negative integers correctly."""
    result = validate_input(x)
    assert result is True
```

**Run**: `/assurance-probe demo_validator`

**Expected output**:
```
No test-strength issues found for module `demo_validator` (1 invariant tested, 2 mutations killed).
```

**Mutations killed**:
1. `x >= 0` → `x > 0` (caught by test with x=0)
2. `x >= 0` → `x == 0` (caught by test with x>0)

## Scenario 4: Mutation errored (syntax error)

**Manually corrupt the mutation framework** to generate a bad mutation:

```python
# In mutations.py, temporarily modify _generate_literal_mutation:
if op == '<':
    return f"{var} == {num - 1} SYNTAX_ERROR"  # Intentional syntax error
```

**Run**: `/assurance-probe demo_validator`

**Expected output** (GitHub issue created):

```markdown
## Finding 1 of 1: Mutation errored

**Invariant**: invariants/demo_validator.md  
**Failure condition**: `x < 0`  
**Mutation**:
```diff
- return x >= 0
+ return x == -1 SYNTAX_ERROR
```

**Test command**: `pytest tests/test_demo_validator.py::test_validate_input`  
**Observed**: Mutation errored (SyntaxError in mutated code)

**Triage**:
- [ ] Accept (test is too weak; fix test or refine Failure condition)
- [x] Reject (false positive; mutation introduced syntax error)
- [ ] Defer (requires Phase 3 generator probe to confirm)
```

## Scenario 5: Run reproducer script

**Run**: `python scripts/probe/demo_validator_20260505.py`

**Expected output**:
```
================================================================================
Assurance Probe Reproducer
================================================================================

Validating environment...
✓ Environment matches recorded state

Reproducing 1 finding(s)...

Finding 1/1: Mutation survived
  Source: demo_validator.py
  Mutation: x >= 0 → x > 0
  Test: pytest tests/test_demo_validator.py::test_validate_input

  Applying mutation...
  Running test...
  Result: survived
  ✓ Matches expected verdict: survived
  Reverting mutation...

================================================================================
Reproduction complete
================================================================================
```

## Takeaways

1. **Mutation probe catches weak tests** — even when tests pass, they may not exercise critical boundaries
2. **Bit-identical reproducers** — findings can be reproduced exactly on the same commit/environment
3. **Bounded output** — probe reports ≤3 findings per run, avoiding noise
4. **Triage workflow** — each finding requires accept/reject/defer decision
5. **SNR tracking** — tracker CSV enables kill-criterion enforcement if SNR drops

## Cleanup

```bash
rm -rf demo_validator.py tests/test_demo_validator.py invariants/demo_validator.md
rm -rf scripts/probe/demo_validator_*.py
rm -rf .assurance/probe-tracker.csv
```

## Next steps

- Phase 2 vacuity probe (gates on Phase 1 SNR ≥ 1:3)
- Phase 3 generator probe (Hypothesis strategy inspection)
- Rotation-based probe scheduling via `/assurance-status`
