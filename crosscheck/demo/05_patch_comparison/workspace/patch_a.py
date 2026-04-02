"""
Patch A: Add keyword.iskeyword() check to the validator.

Uses Python's built-in keyword module. Clean, minimal fix.
"""

from __future__ import annotations

import keyword
import re


_FQCN_RE = re.compile(r'^[a-zA-Z_]\w*\.[a-zA-Z_]\w*$')


def _is_python_identifier(name: str) -> bool:
    """Check if a string is a valid Python identifier AND not a keyword."""
    return (
        isinstance(name, str)
        and name.isidentifier()
        and not keyword.iskeyword(name)
    )


def is_valid_collection_name(name: str) -> bool:
    """Validate a Fully Qualified Collection Name (FQCN).

    A valid FQCN must:
    1. Contain exactly one dot separating namespace and name
    2. Have both segments be valid Python identifiers
    3. Neither segment may be a Python keyword
    """
    if not isinstance(name, str):
        return False

    parts = name.split('.')
    if len(parts) != 2:
        return False

    namespace, collection = parts
    return _is_python_identifier(namespace) and _is_python_identifier(collection)
