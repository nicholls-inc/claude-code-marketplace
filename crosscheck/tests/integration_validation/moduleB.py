"""
Module B - contains process function with undocumented precondition.
"""

def process(x):
    """
    Process an integer value.
    
    Undocumented precondition: x > 0
    Bug: crashes when x=0 due to division
    """
    # This will crash when x=0
    result = 100 / x
    return result * 2
