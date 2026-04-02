"""
Tests for FQCN validation — demonstrates the keyword hole.

Run: pytest test_validate.py -v
"""

import keyword
import pytest
from validate_name_bug import is_valid_collection_name, CollectionRegistry


# ============================================================
# Tests that PASS — basic validation works
# ============================================================

class TestValidNames:
    """Valid FQCNs — all pass."""

    @pytest.mark.parametrize('name', [
        'ansible.builtin',
        'community.general',
        'my_namespace.my_collection',
        'ns1.coll2',
        '_private.collection',
    ])
    def test_valid_fqcn(self, name):
        assert is_valid_collection_name(name) is True


class TestInvalidNames:
    """Obviously invalid FQCNs — all pass (correctly rejected)."""

    @pytest.mark.parametrize('name', [
        '',                    # Empty
        'no_dot',              # Missing dot
        'too.many.dots',       # Extra dot
        '.leading_dot',        # Leading dot
        'trailing_dot.',       # Trailing dot
        '1invalid.name',       # Starts with digit
        'ns.2bad',             # Name starts with digit
        'ns.na-me',            # Hyphen not valid in identifier
        'ns.na me',            # Space not valid
    ])
    def test_invalid_fqcn(self, name):
        assert is_valid_collection_name(name) is False


class TestNonStringInput:
    """Non-string inputs are rejected."""

    @pytest.mark.parametrize('value', [
        None, 42, 3.14, [], {}, True,
    ])
    def test_non_string_rejected(self, value):
        assert is_valid_collection_name(value) is False


class TestRegistry:
    """Collection registry validates on insertion."""

    def test_register_valid(self):
        registry = CollectionRegistry()
        registry.register('ansible.builtin', {'version': '2.14'})
        assert registry.is_registered('ansible.builtin')

    def test_register_invalid_raises(self):
        registry = CollectionRegistry()
        with pytest.raises(ValueError):
            registry.register('not-valid')


# ============================================================
# Tests that FAIL — Python keywords accepted as valid names
# ============================================================

class TestKeywordRejection:
    """Python keywords must not be accepted as namespace or collection names.

    Collections map to Python packages at import time:
        import ansible_collections.{namespace}.{name}

    If namespace='def' or name='return', the import statement becomes:
        import ansible_collections.def.return

    This is a SyntaxError. The validator MUST reject these.

    BUG: str.isidentifier() returns True for keywords like 'def', 'class',
    'return', 'import'. The validator needs an additional keyword check.
    """

    @pytest.mark.parametrize('name', [
        'def.collection',     # 'def' is a keyword
        'ns.return',          # 'return' is a keyword
        'class.module',       # 'class' is a keyword
        'import.utils',       # 'import' is a keyword
        'assert.test',        # 'assert' is a keyword
        'lambda.tools',       # 'lambda' is a keyword
        'from.source',        # 'from' is a keyword
        'yield.data',         # 'yield' is a keyword
    ])
    def test_keyword_as_namespace_or_name(self, name):
        assert is_valid_collection_name(name) is False, (
            f"{name!r} was accepted but contains Python keyword "
            f"'{name.split('.')[0]}' or '{name.split('.')[1]}' — "
            f"this would cause 'import ansible_collections.{name}' "
            f"to be a SyntaxError"
        )

    @pytest.mark.parametrize('name', [
        'True.collection',    # 'True' is a keyword (soft in 2.x, hard in 3.x)
        'False.utils',        # 'False' is a keyword
        'None.tools',         # 'None' is a keyword
    ])
    def test_builtin_constants_as_names(self, name):
        assert is_valid_collection_name(name) is False


class TestSoftKeywords:
    """Python 3.10+ soft keywords (match, case, type) are valid identifiers
    and NOT in keyword.kwlist. These are context-dependent and CAN be used
    as names — so they should be ACCEPTED.

    This is the subtle part: a naive fix that checks keyword.iskeyword()
    handles hard keywords but correctly allows soft keywords. A broken
    fix that checks against a hardcoded list might reject these.
    """

    @pytest.mark.parametrize('name', [
        'match.pattern',      # 'match' is a soft keyword — valid!
        'case.handler',       # 'case' is a soft keyword — valid!
        'type.checker',       # 'type' is a soft keyword — valid!
    ])
    def test_soft_keywords_accepted(self, name):
        """Soft keywords are valid identifiers and should be accepted."""
        assert is_valid_collection_name(name) is True
