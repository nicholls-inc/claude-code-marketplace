"""Reproduction of SWE-bench sympy__sympy-13177.

SymPy's Mod function incorrectly simplifies x**n % x to 0 for ALL x,
but this is only valid when x is a non-zero integer and n is a
positive integer.

The bug is in the simplification rule inside Mod.eval's doit() helper.
The original code checks:

    p.is_Pow and p.exp.is_Integer and p.base == q

Problems:
  1. Uses .is_Integer (capital I = "is a SymPy Integer type") instead
     of .is_integer (lowercase = "has the mathematical property of
     being an integer"). Symbols with assume integer=True fail the check.
  2. Missing check: q.is_integer — the modulus must also be an integer.
     Mod(1.5**2, 1.5) = 0.75, not 0.
  3. Missing check: p.exp.is_positive — negative exponents break it.
     Mod(2**(-2), 2) = 0.25, not 0.

Source: https://github.com/sympy/sympy/issues/13169
Fix: https://github.com/sympy/sympy/pull/13177
"""


def can_simplify_mod_power(base_is_integer, exp_value, modulus_is_integer):
    """Determine if base**exp % modulus can be simplified to 0.

    This is the BUGGY version — extracted from SymPy's Mod.eval logic.
    It only checks that the exponent is an integer, missing the checks
    for modulus being integer and exponent being positive.
    """
    # Original buggy condition (simplified):
    #   p.is_Pow and p.exp.is_Integer and p.base == q
    # Assumes base == modulus (which is the pattern x**n % x)
    exp_is_integer = isinstance(exp_value, int)
    return exp_is_integer  # Missing: modulus_is_integer and exp_value > 0


def can_simplify_mod_power_fixed(base_is_integer, exp_value, modulus_is_integer):
    """Correct version — checks all three conditions."""
    exp_is_integer = isinstance(exp_value, int)
    exp_is_positive = isinstance(exp_value, int) and exp_value > 0
    return base_is_integer and modulus_is_integer and exp_is_positive
