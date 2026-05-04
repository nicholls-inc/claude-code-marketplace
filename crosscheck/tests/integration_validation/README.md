# Integration Validation Test Scenarios

This directory contains test scenarios for validating the integration validation phases added to `/locate-fault` and `/reason` skills.

## Test Scenario 1: Multi-file Bug (Execution Trace Spanning)

**Files:**
- `moduleA.py` - Contains `validate_input(x)` that calls `moduleB.process(x)`
- `moduleB.py` - Contains `process(x)` with undocumented precondition `x > 0`
- `test_multifile_bug.py` - Test that exposes the bug when `x=0`

**Bug Description:**
The bug demonstrates an interface assumption mismatch:
- `moduleA.validate_input()` looks correct in isolation - it just passes input to `process()`
- `moduleB.process()` has an undocumented precondition that `x > 0` (crashes with division by zero when `x=0`)
- The bug manifests only when tracing across the A→B boundary

**Expected `/locate-fault` Behavior:**
1. Phase 2 should read both `moduleA.py` and `moduleB.py`
2. Phase 3 should cite the precondition violation in `moduleB.process()`
3. Phase 5 should confirm trace crossed the A→B boundary and identify the interface assumption mismatch

**Manual Test:**
```bash
cd /workspace/crosscheck/tests/integration_validation
python3 -c "import moduleA; moduleA.validate_input(0)"  # Should crash with ZeroDivisionError
python3 -c "import moduleA; print(moduleA.validate_input(5))"  # Should return 40.0
```

## Test Scenario 2: Interface-Only Reasoning

**Files:**
- `caller.py` - Contains `divide_safe(x)` that calls `utils.divide_by(x, x)`
- `utils.py` - Contains `divide_by(a, b)` with precondition `b != 0`
- `test_interface_reasoning.py` - Test that exposes the bug when `x=0`

**Bug Description:**
The bug demonstrates a precondition violation at an interface boundary:
- `caller.divide_safe(x)` passes `x` as both numerator and denominator
- `utils.divide_by(a, b)` requires `b != 0`
- When `x=0`, the caller violates the callee's precondition

**Expected `/reason` Behavior (for question "Is `caller.divide_safe(user_input)` safe?"):**
1. Step 2 should read both `caller.py` and `utils.py`
2. Step 3 should trace that caller passes `x` as both `a` and `b`
3. Step 4c integration validation should document the interface crossing and verify callee behavior
4. Step 5 should flag the `x=0` case as violating the precondition

**Manual Test:**
```bash
cd /workspace/crosscheck/tests/integration_validation
python3 -c "import caller; caller.divide_safe(0)"  # Should crash with ZeroDivisionError
python3 -c "import caller; print(caller.divide_safe(5))"  # Should return 1.0
```

## Verification Approach (Semi-formal)

The plan specifies semi-formal verification by:

1. **Trace execution across test cases** - Run `/locate-fault` and `/reason` with these scenarios
2. **Certificate inspection** - Parse output for:
   - Presence of Phase 5 / Step 4c sections
   - Evidence of reading multiple files
   - Claims referencing code in callee implementations
   - Invariant 1: files-read ⊇ files-on-execution-path
   - Invariant 2: trust boundaries documented for unreadable callees
3. **Adequacy check via `/rationale`** - Build an adequacy argument for the multi-file test scenario

These tests serve as regression tests to ensure the integration validation phases work correctly.
