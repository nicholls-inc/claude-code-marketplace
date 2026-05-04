"""
Module A - contains validate_input which calls moduleB.process.
"""

try:
    from . import moduleB
except ImportError:
    import moduleB

def validate_input(x):
    """
    Validate and process input.
    
    This function looks "correct" on the surface - it accepts any integer
    and passes it to process(). However, it violates moduleB.process's
    undocumented precondition that x > 0.
    """
    # Looks correct - just passing through to process
    # Bug: assumes process() accepts any integer
    return moduleB.process(x)