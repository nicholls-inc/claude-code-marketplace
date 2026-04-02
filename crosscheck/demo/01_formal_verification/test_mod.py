"""Tests demonstrating the Mod simplification bug.

No SymPy dependency needed — pure Python demonstrates the underlying
mathematical property that SymPy's simplification rule got wrong.
"""
import pytest


# --- The mathematical property under test ---
# When is x**n % x == 0?

class TestModPowerIntegers:
    """x**n % x == 0 for non-zero integer x and positive integer n."""

    def test_positive_base_positive_exp(self):
        assert 3**2 % 3 == 0   # 9 % 3 = 0
        assert 5**3 % 5 == 0   # 125 % 5 = 0
        assert 7**1 % 7 == 0   # 7 % 7 = 0

    def test_negative_base_positive_exp(self):
        assert (-2)**4 % (-2) == 0  # 16 % -2 = 0
        assert (-3)**3 % (-3) == 0  # -27 % -3 = 0

    def test_large_exponent(self):
        assert 2**100 % 2 == 0


class TestModPowerCounterexamples:
    """Cases where x**n % x != 0 — the conditions SymPy failed to check."""

    def test_non_integer_base(self):
        """Mod(1.5**2, 1.5) should be 0.75, not 0. This is the bug."""
        result = 1.5**2 % 1.5  # 2.25 % 1.5
        assert result == pytest.approx(0.75)
        assert result != 0  # SymPy incorrectly returned 0

    def test_another_non_integer(self):
        result = 2.5**2 % 2.5  # 6.25 % 2.5
        assert result == pytest.approx(1.25)

    def test_negative_exponent(self):
        """Mod(2**(-2), 2) should be 0.25, not 0."""
        result = 2**(-2) % 2  # 0.25 % 2
        assert result == pytest.approx(0.25)
        assert result != 0

    def test_zero_exponent(self):
        """x**0 % x = 1 % x, which is 1 for |x| > 1."""
        assert 5**0 % 5 == 1  # 1 % 5 = 1, not 0


class TestCanSimplify:
    """Test the extracted guard function."""

    def test_buggy_version_wrong_on_non_integer(self):
        from mod_bug import can_simplify_mod_power
        # Buggy: says we can simplify even when modulus isn't integer
        assert can_simplify_mod_power(
            base_is_integer=False, exp_value=2, modulus_is_integer=False
        ) == True  # WRONG — should be False

    def test_buggy_version_wrong_on_negative_exp(self):
        from mod_bug import can_simplify_mod_power
        # Buggy: says we can simplify with negative exponent
        assert can_simplify_mod_power(
            base_is_integer=True, exp_value=-2, modulus_is_integer=True
        ) == True  # WRONG — should be False

    def test_fixed_version_correct(self):
        from mod_bug import can_simplify_mod_power_fixed
        # Correct: rejects non-integer modulus
        assert can_simplify_mod_power_fixed(False, 2, False) == False
        # Correct: rejects negative exponent
        assert can_simplify_mod_power_fixed(True, -2, True) == False
        # Correct: accepts valid case
        assert can_simplify_mod_power_fixed(True, 2, True) == True
