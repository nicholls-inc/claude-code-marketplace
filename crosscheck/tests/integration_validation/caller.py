"""
Caller module that uses utils.divide_by incorrectly.
"""

try:
    from . import utils
except ImportError:
    import utils

def divide_safe(x):
    """
    Attempt to safely divide x by x.
    
    Bug: When x=0, this violates the callee's precondition (b != 0)
    because we pass x as both numerator and denominator.
    """
    # Passes x as both a and b
    # When x=0, this violates utils.divide_by's precondition
    return utils.divide_by(x, x)