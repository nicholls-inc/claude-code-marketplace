"""
Patch B: Hardcoded keyword list + identifier check.

Uses a manually maintained list of "reserved names" that includes
Python keywords, builtins, and some additional restrictions.
"""

from __future__ import annotations

import re


_FQCN_RE = re.compile(r'^[a-zA-Z_]\w*\.[a-zA-Z_]\w*$')

# Hardcoded set of reserved names — no import of keyword module.
# This list was assembled from Python 3.10 keyword.kwlist and builtins.
_RESERVED_NAMES = frozenset({
    # Python keywords
    'False', 'None', 'True', 'and', 'as', 'assert', 'async', 'await',
    'break', 'class', 'continue', 'def', 'del', 'elif', 'else', 'except',
    'finally', 'for', 'from', 'global', 'if', 'import', 'in', 'is',
    'lambda', 'nonlocal', 'not', 'or', 'pass', 'raise', 'return', 'try',
    'while', 'with', 'yield',
    # Soft keywords (match, case, type) — also blocked
    'match', 'case', 'type',
    # Common builtins that could cause confusion
    'print', 'input', 'list', 'dict', 'set', 'str', 'int', 'float',
    'bool', 'tuple', 'range', 'len', 'map', 'filter', 'zip', 'enumerate',
    'object', 'super', 'property', 'staticmethod', 'classmethod',
})


def _is_python_identifier(name: str) -> bool:
    """Check if a string is a valid Python identifier and not reserved."""
    return (
        isinstance(name, str)
        and name.isidentifier()
        and name not in _RESERVED_NAMES
    )


def is_valid_collection_name(name: str) -> bool:
    """Validate a Fully Qualified Collection Name (FQCN).

    A valid FQCN must:
    1. Contain exactly one dot separating namespace and name
    2. Have both segments be valid Python identifiers
    3. Neither segment may be a reserved name
    """
    if not isinstance(name, str):
        return False

    parts = name.split('.')
    if len(parts) != 2:
        return False

    namespace, collection = parts
    return _is_python_identifier(namespace) and _is_python_identifier(collection)
