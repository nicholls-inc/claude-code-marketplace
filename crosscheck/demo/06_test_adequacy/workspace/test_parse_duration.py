"""
Test suite for parse_duration().

Run: pytest test_parse_duration.py -v
"""

import pytest
from parse_duration import parse_duration, schedule_command


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
