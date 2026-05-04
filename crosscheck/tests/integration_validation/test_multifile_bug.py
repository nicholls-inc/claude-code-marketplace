"""
Test for multi-file bug scenario.

This test demonstrates a failure that occurs across component boundaries:
- moduleA.validate_input looks correct in isolation
- moduleB.process has an undocumented precondition (x > 0)
- The bug manifests when validate_input calls process with x=0

The fault localization should:
1. Read both moduleA.py and moduleB.py (Phase 2)
2. Cite the precondition violation in moduleB (Phase 3)
3. Confirm trace crossed the A→B boundary (Phase 5)
4. Identify the interface assumption mismatch
"""

import pytest
from . import moduleA

def test_validate_input_with_zero():
    """
    Test that exposes the interface assumption mismatch.
    
    Expected: Should handle x=0 gracefully
    Actual: Crashes with ZeroDivisionError in moduleB.process
    """
    # This should trigger the bug
    with pytest.raises(ZeroDivisionError):
        moduleA.validate_input(0)

def test_validate_input_with_positive():
    """
    Test with valid input (x > 0) - should pass.
    """
    result = moduleA.validate_input(5)
    assert result == 40.0  # (100 / 5) * 2 = 40
