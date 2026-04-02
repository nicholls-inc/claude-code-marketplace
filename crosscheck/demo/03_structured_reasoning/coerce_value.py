from __future__ import annotations

import os
import tempfile
from collections.abc import Mapping, Sequence
from decimal import Decimal, InvalidOperation


class TaggedValue:
    """A value annotated with origin/trust metadata.

    Tags track where a configuration value came from (e.g., 'env', 'file',
    'cli', 'default') and whether it should be trusted. Tags MUST survive
    type coercion — a value loaded from an untrusted source doesn't become
    trusted just because it was converted from string to int.
    """

    def __init__(self, value, tag: str):
        self.value = value
        self.tag = tag

    def __repr__(self):
        return f"TaggedValue({self.value!r}, tag={self.tag!r})"

    def __eq__(self, other):
        if isinstance(other, TaggedValue):
            return self.value == other.value and self.tag == other.tag
        return NotImplemented

    def __hash__(self):
        return hash((self.value, self.tag))


def _unwrap(value):
    """Extract raw value, preserving tag reference for re-wrapping."""
    if isinstance(value, TaggedValue):
        return value.value, value.tag
    return value, None


def _rewrap(value, tag):
    """Re-apply tag if one was present on the original value."""
    if tag is not None:
        return TaggedValue(value, tag)
    return value


# Valid boolean truthy/falsy strings
BOOLEANS_TRUE = frozenset(("yes", "on", "1", "true"))
BOOLEANS_FALSE = frozenset(("no", "off", "0", "false"))


def coerce_value(value, expected_type: str):
    """Coerce a value to the expected type, preserving any metadata tag.

    Supported types:
        'boolean'  - Convert to bool (accepts strings 'yes'/'no'/'true'/'false'/etc.)
        'integer'  - Convert to int
        'float'    - Convert to float
        'string'   - Convert to str
        'list'     - Convert to list (split comma-separated strings)
        'dict'     - Convert to dict
        'path'     - Expand ~ and env vars, convert to str
        'temppath' - Create a temp directory path

    Args:
        value: The value to coerce (may be TaggedValue-wrapped)
        expected_type: Target type name

    Returns:
        Coerced value, re-wrapped in TaggedValue if input was tagged

    Raises:
        ValueError: If the value cannot be coerced to the expected type
    """
    raw, tag = _unwrap(value)

    if expected_type == "boolean":
        if isinstance(raw, bool):
            return _rewrap(raw, tag)
        if isinstance(raw, str):
            s = raw.lower().strip()
            if s in BOOLEANS_TRUE:
                return _rewrap(True, tag)
            if s in BOOLEANS_FALSE:
                return _rewrap(False, tag)
        # Try membership test for non-string types
        if raw in BOOLEANS_TRUE:
            return _rewrap(True, tag)
        if raw in BOOLEANS_FALSE:
            return _rewrap(False, tag)
        raise ValueError(f"Cannot coerce {raw!r} to boolean")

    elif expected_type == "integer":
        if isinstance(raw, bool):
            # bool is a subclass of int in Python — but True/False
            # should stay as 1/0 respectively
            return _rewrap(raw, tag)
        if isinstance(raw, int):
            return _rewrap(raw, tag)
        if isinstance(raw, (str, float)):
            try:
                d = Decimal(str(raw))
                return _rewrap(int(d), tag)
            except (InvalidOperation, ValueError):
                raise ValueError(f"Cannot coerce {raw!r} to integer")
        raise ValueError(f"Cannot coerce {raw!r} to integer")

    elif expected_type == "float":
        if isinstance(raw, (int, float)):
            return _rewrap(float(raw), tag)
        if isinstance(raw, str):
            try:
                return _rewrap(float(raw), tag)
            except ValueError:
                raise ValueError(f"Cannot coerce {raw!r} to float")
        raise ValueError(f"Cannot coerce {raw!r} to float")

    elif expected_type == "string":
        return _rewrap(str(raw), tag)

    elif expected_type == "list":
        if isinstance(raw, str):
            items = [s.strip() for s in raw.split(",")]
            return _rewrap(items, tag)
        if isinstance(raw, Sequence):
            return _rewrap(list(raw), tag)
        raise ValueError(f"Cannot coerce {raw!r} to list")

    elif expected_type == "dict":
        if isinstance(raw, str):
            # Try key=value parsing
            pairs = {}
            for item in raw.split(","):
                if "=" not in item:
                    raise ValueError(f"Cannot coerce {raw!r} to dict")
                k, v = item.split("=", 1)
                pairs[k.strip()] = v.strip()
            return _rewrap(pairs, tag)
        if isinstance(raw, Mapping):
            return _rewrap(dict(raw), tag)
        raise ValueError(f"Cannot coerce {raw!r} to dict")

    elif expected_type == "path":
        if isinstance(raw, str):
            expanded = os.path.expanduser(os.path.expandvars(raw))
            return _rewrap(expanded, tag)
        raise ValueError(f"Cannot coerce {raw!r} to path")

    elif expected_type == "temppath":
        # Temp paths are always freshly created — no tag preservation
        return tempfile.mkdtemp()

    else:
        raise ValueError(f"Unknown type: {expected_type}")
