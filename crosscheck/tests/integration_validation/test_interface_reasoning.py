"""
Test for interface-only reasoning scenario.

This test verifies that /reason performs integration validation:
- caller.divide_safe(x) appears to safely divide x by x
- utils.divide_by requires b != 0
- When x=0, the caller violates the callee's precondition

The reasoning should:
1. Read both caller.py and utils.py (Step 2)
2. Identify the precondition b != 0 in utils.divide_by
3. Trace that caller passes x as both a and b (Step 3)
4. Document the interface crossing and verify callee behavior (Step 4c)
5. Flag the x=0 case as violating the precondition
"""

import pytest
from . import caller

def test_divide_safe_with_zero():
    """
    Test that exposes the precondition violation.
    
    Expected: Should handle x=0 gracefully
    Actual: Crashes with ZeroDivisionError
    """
    with pytest.raises(ZeroDivisionError):
        caller.divide_safe(0)

def test_divide_safe_with_nonzero():
    """
    Test with valid input (x != 0) - should return 1.0.
    """
    result = caller.divide_safe(5)
    assert result == 1.0  # 5 / 5 = 1
