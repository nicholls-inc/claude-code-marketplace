"""
Duration string parser for command scheduling.

Parses human-friendly duration strings like '1h30m', '45s', or plain
seconds into millisecond values. Used by the 'later' command to schedule
delayed command execution.

SWE-bench_Pro: qutebrowser__qutebrowser-96b997802e942937e81d2b8a32d08f00d3f4bc4e
"""

from __future__ import annotations

import re


def parse_duration(duration: str) -> int:
    """Parse a duration string into milliseconds.

    Accepted formats:
        - Plain integer: treated as seconds (e.g., '60' -> 60000ms)
        - XhYmZs format: hours/minutes/seconds (e.g., '1h30m' -> 5400000ms)
        - Units can appear in any order (e.g., '1s1h' == '1h1s')
        - Each unit (h, m, s) may appear at most once

    Invalid inputs return -1:
        - Negative values ('-1s')
        - Fractional values ('60.4s')
        - Duplicate units ('34ss')
        - Empty string
        - Non-numeric content

    Args:
        duration: The duration string to parse

    Returns:
        Duration in milliseconds, or -1 if the format is invalid
    """
    if not duration or not isinstance(duration, str):
        return -1

    # Plain integer = seconds
    if re.match(r'^[0-9]+$', duration):
        return int(duration) * 1000

    # Validate: only digits and unit chars, in valid groupings
    if not re.match(r'^([0-9]+[hms])+$', duration):
        return -1

    # Extract components
    hours = 0
    minutes = 0
    seconds = 0

    match = re.search(r'([0-9]+)h', duration)
    if match:
        hours = int(match.group(1))

    match = re.search(r'([0-9]+)m', duration)
    if match:
        minutes = int(match.group(1))

    match = re.search(r'([0-9]+)s', duration)
    if match:
        seconds = int(match.group(1))

    return (hours * 3600 + minutes * 60 + seconds) * 1000


def schedule_command(duration_str: str, command: str) -> dict:
    """Schedule a command to run after the given duration.

    Args:
        duration_str: Duration to wait (e.g., '30s', '1h5m')
        command: The command to execute after the delay

    Returns:
        Dict with scheduling details

    Raises:
        ValueError: If the duration format is invalid
    """
    ms = parse_duration(duration_str)
    if ms < 0:
        raise ValueError(
            f"Invalid duration format: {duration_str!r}. "
            "Expected XhYmZs or plain seconds."
        )
    return {
        'delay_ms': ms,
        'command': command,
        'duration_input': duration_str,
    }
