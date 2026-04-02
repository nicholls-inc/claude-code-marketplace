"""
Test suite for parse_duration() — appears comprehensive but has gaps.

Run: pytest test_parse_duration.py -v

This test suite achieves 100% line coverage of parse_duration() and all
tests pass. But it misses critical properties that /rationale will find.
"""

import pytest
from parse_duration_bug import parse_duration, schedule_command


# ============================================================
# "Comprehensive" test suite — all pass, 100% line coverage
# ============================================================

class TestPlainSeconds:
    """Plain integer input = seconds."""

    def test_zero(self):
        assert parse_duration('0') == 0

    def test_one_second(self):
        assert parse_duration('1') == 1000

    def test_sixty_seconds(self):
        assert parse_duration('60') == 60000

    def test_large_value(self):
        assert parse_duration('3600') == 3600000


class TestUnitFormats:
    """Standard unit formats."""

    def test_seconds(self):
        assert parse_duration('30s') == 30000

    def test_minutes(self):
        assert parse_duration('5m') == 300000

    def test_hours(self):
        assert parse_duration('2h') == 7200000

    def test_hours_minutes(self):
        assert parse_duration('1h30m') == 5400000

    def test_hours_seconds(self):
        assert parse_duration('1h15s') == 3615000

    def test_minutes_seconds(self):
        assert parse_duration('2m30s') == 150000

    def test_all_units(self):
        assert parse_duration('1h1m1s') == 3661000

    def test_large_compound(self):
        assert parse_duration('10h1m10s') == 36070000


class TestInvalidInputs:
    """Rejected inputs."""

    def test_empty_string(self):
        assert parse_duration('') == -1

    def test_negative_seconds(self):
        assert parse_duration('-1s') == -1

    def test_negative_plain(self):
        assert parse_duration('-1') == -1

    def test_fractional_seconds(self):
        assert parse_duration('60.4s') == -1

    def test_no_digits(self):
        assert parse_duration('abc') == -1

    def test_none_input(self):
        assert parse_duration(None) == -1


class TestScheduleCommand:
    """Integration with schedule_command()."""

    def test_valid_schedule(self):
        result = schedule_command('30s', 'echo hello')
        assert result['delay_ms'] == 30000
        assert result['command'] == 'echo hello'

    def test_invalid_raises(self):
        with pytest.raises(ValueError, match="Invalid duration"):
            schedule_command('-1s', 'echo hello')


# ============================================================
# THE GAP: These tests are MISSING from the suite above.
# They expose properties that the "comprehensive" suite doesn't check.
# /rationale would identify these gaps.
# ============================================================

class TestMissedProperties:
    """Tests the original suite should have included but didn't.

    These are NOT about exotic edge cases — they're about fundamental
    properties of a duration parser that any verification analysis would flag.
    """

    def test_unit_order_invariance(self):
        """Property: unit order should not affect the result.
        '1s1h' and '1h1s' should be identical.

        The regex-then-search approach handles this correctly by accident,
        but the test suite never verified it. If someone refactors to a
        sequential parser, this property could silently break.
        """
        assert parse_duration('1s1h') == parse_duration('1h1s')
        assert parse_duration('30s5m') == parse_duration('5m30s')
        assert parse_duration('1s1m1h') == parse_duration('1h1m1s')

    def test_duplicate_units_rejected(self):
        """Property: each unit should appear at most once.
        '5s10s' has two seconds values — which one wins?

        The regex search approach takes the FIRST match, silently
        ignoring the second. '5s10s' returns 5000 instead of error.
        This is a correctness bug: the input is ambiguous and should
        be rejected.
        """
        assert parse_duration('5s10s') == -1, \
            "Duplicate 's' unit should be rejected, not silently use first match"
        assert parse_duration('1h2h') == -1
        assert parse_duration('3m7m') == -1

    def test_zero_delay_accepted(self):
        """Property: '0s', '0m', '0h' should all be valid (0ms delay).

        The test suite checks '0' (plain seconds) but not unit-prefixed zeros.
        """
        assert parse_duration('0s') == 0
        assert parse_duration('0m') == 0
        assert parse_duration('0h') == 0

    def test_single_unit_h_m(self):
        """Property: single-unit inputs should work for all units.

        The suite tests '30s', '5m', '2h' but never with value 1.
        With '1m', the parser must distinguish 'm' from 'ms' (not a thing
        here, but a common parser bug).
        """
        assert parse_duration('1h') == 3600000
        assert parse_duration('1m') == 60000
        assert parse_duration('1s') == 1000

    def test_overflow_values(self):
        """Property: very large values should not cause unexpected behavior.

        99999h * 3600 * 1000 = 359,996,400,000 — fits in Python int but
        would overflow a 32-bit integer. If this code is ever ported or
        the result is passed to a C extension, this matters.
        """
        result = parse_duration('99999h')
        assert result == 99999 * 3600 * 1000

    def test_leading_zeros(self):
        """Property: leading zeros in values should be handled consistently.

        '007s' — is this 7 seconds or an error? The current implementation
        accepts it (int('007') == 7), but the test suite never checks.
        """
        assert parse_duration('007s') == 7000
        assert parse_duration('007') == 7000
