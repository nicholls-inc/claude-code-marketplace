"""
Fully Qualified Collection Name (FQCN) validator.

Collections use dotted namespace.name format (e.g., 'ansible.builtin',
'community.general'). Both segments must be valid Python identifiers
since they map to Python package/module names at import time.
"""

from __future__ import annotations

import re

# Legacy validation pattern: allows dotted names with alphanumerics and underscores
_FQCN_RE = re.compile(r'^[a-zA-Z_]\w*\.[a-zA-Z_]\w*$')


def _is_python_identifier(name: str) -> bool:
    """Check if a string is a valid Python identifier.

    Used to validate that collection namespace and name segments
    can be used as Python package/module names.
    """
    return isinstance(name, str) and name.isidentifier()


def is_valid_collection_name(name: str) -> bool:
    """Validate a Fully Qualified Collection Name (FQCN).

    A valid FQCN must:
    1. Contain exactly one dot separating namespace and name
    2. Have both namespace and name be valid Python identifiers
    3. Not be empty on either side of the dot

    Args:
        name: The collection name to validate (e.g., 'ansible.builtin')

    Returns:
        True if the name is a valid FQCN, False otherwise

    Examples:
        >>> is_valid_collection_name('ansible.builtin')
        True
        >>> is_valid_collection_name('community.general')
        True
        >>> is_valid_collection_name('bad')
        False
        >>> is_valid_collection_name('too.many.dots')
        False
    """
    if not isinstance(name, str):
        return False

    parts = name.split('.')
    if len(parts) != 2:
        return False

    namespace, collection = parts
    return _is_python_identifier(namespace) and _is_python_identifier(collection)


# --- Collection registry using the validator ---

class CollectionRegistry:
    """Registry of installed collections, validated on insertion."""

    def __init__(self):
        self._collections: dict[str, dict] = {}

    def register(self, fqcn: str, metadata: dict | None = None) -> None:
        """Register a collection by its FQCN.

        Raises:
            ValueError: If the FQCN is invalid
        """
        if not is_valid_collection_name(fqcn):
            raise ValueError(f"Invalid collection name: {fqcn!r}")
        self._collections[fqcn] = metadata or {}

    def is_registered(self, fqcn: str) -> bool:
        return fqcn in self._collections

    def list_collections(self) -> list[str]:
        return sorted(self._collections.keys())
