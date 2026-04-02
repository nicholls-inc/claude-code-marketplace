"""
Tests for coerce_value() — demonstrates the bugs.

Run: pytest test_coerce.py -v
"""

import pytest
from coerce_value import TaggedValue, coerce_value

# ============================================================
# Tests that PASS (Claude sees these and thinks things are fine)
# ============================================================


class TestBasicCoercion:
    """Basic type coercion — all pass, giving false confidence."""

    def test_string_to_bool_true(self):
        assert coerce_value("yes", "boolean") is True

    def test_string_to_bool_false(self):
        assert coerce_value("no", "boolean") is False

    def test_int_passthrough(self):
        assert coerce_value(42, "integer") == 42

    def test_string_to_int(self):
        assert coerce_value("42", "integer") == 42

    def test_string_to_float(self):
        assert coerce_value("3.14", "float") == 3.14

    def test_string_to_list(self):
        assert coerce_value("a, b, c", "list") == ["a", "b", "c"]

    def test_list_passthrough(self):
        assert coerce_value([1, 2, 3], "list") == [1, 2, 3]

    def test_dict_from_string(self):
        assert coerce_value("k1=v1, k2=v2", "dict") == {"k1": "v1", "k2": "v2"}

    def test_dict_passthrough(self):
        assert coerce_value({"a": 1}, "dict") == {"a": 1}

    def test_path_expansion(self):
        result = coerce_value("~/test", "path")
        assert "~" not in result

    def test_tagged_string_to_int(self):
        result = coerce_value(TaggedValue("42", "env"), "integer")
        assert isinstance(result, TaggedValue)
        assert result.value == 42
        assert result.tag == "env"


# ============================================================
# Tests that FAIL — these expose the bugs
# ============================================================


class TestTagPreservation:
    """Tag preservation through all conversion paths."""

    def test_bool_to_int_preserves_tag(self):
        """BUG 1: bool-to-int returns raw bool, tag is lost.

        The code returns _rewrap(raw, tag) where raw is still True/False.
        It should convert True->1, False->0 first.
        Also: the value should become an actual int, not remain bool.
        """
        result = coerce_value(TaggedValue(True, "cli"), "integer")
        assert isinstance(result, TaggedValue), "Tag was lost!"
        assert result.value == 1
        assert type(result.value) is int, "Should be int, not bool"
        assert result.tag == "cli"

    def test_bool_false_to_int_preserves_tag(self):
        result = coerce_value(TaggedValue(False, "default"), "integer")
        assert isinstance(result, TaggedValue), "Tag was lost!"
        assert result.value == 0
        assert type(result.value) is int
        assert result.tag == "default"


class TestBooleanEdgeCases:
    """Boolean conversion edge cases."""

    def test_unhashable_value_to_bool(self):
        """BUG 2: Unhashable values (lists, dicts) cause TypeError.

        The code does `raw in BOOLEANS_TRUE` which calls __hash__.
        Lists and dicts are unhashable, so this throws TypeError
        instead of ValueError.
        """
        with pytest.raises(ValueError, match="Cannot coerce"):
            coerce_value([1, 2, 3], "boolean")


class TestSequenceHandling:
    """Sequence type handling edge cases."""

    def test_bytes_to_list(self):
        """BUG 3: bytes is a Sequence but list(bytes) gives ints.

        b'hello' is isinstance(bytes, Sequence) == True, so the code
        calls list(b'hello') which gives [104, 101, 108, 108, 111]
        instead of raising ValueError.
        """
        with pytest.raises(ValueError, match="Cannot coerce"):
            coerce_value(b"hello", "list")


class TestIntegerConversion:
    """Integer conversion edge cases."""

    def test_float_with_fractional_part_to_int(self):
        """BUG 4: Decimal(str(1.7)) silently truncates to 1.

        The code does int(Decimal(str(raw))) which truncates 1.7 to 1.
        It should verify the fractional part is zero before converting.
        """
        with pytest.raises(ValueError, match="Cannot coerce"):
            coerce_value(1.7, "integer")

    def test_float_string_with_fraction_to_int(self):
        """Same bug via string input: '3.14' should not become 3."""
        with pytest.raises(ValueError, match="Cannot coerce"):
            coerce_value("3.14", "integer")

    def test_float_integer_value_to_int(self):
        """2.0 SHOULD convert to 2 (zero fractional part)."""
        assert coerce_value(2.0, "integer") == 2


class TestMappingConversion:
    """Mapping type handling."""

    def test_ordered_dict_to_dict(self):
        """Mapping subclasses should convert to plain dict with tag."""
        from collections import OrderedDict

        od = OrderedDict([("b", 2), ("a", 1)])
        result = coerce_value(TaggedValue(od, "file"), "dict")
        assert isinstance(result, TaggedValue)
        assert result.value == {"b": 2, "a": 1}
        assert result.tag == "file"


class TestListElementValidation:
    """BUG 5: List coercion doesn't validate element types for path lists."""

    def test_sequence_with_non_string_elements(self):
        """When coercing a sequence to list, non-string mixed types
        are silently accepted. For pathspec/pathlist types this can
        cause downstream failures when the values are used as paths.

        The existing tests don't check this at all — they only test
        string splitting and list passthrough.
        """
        result = coerce_value([1, None, True, "ok"], "list")
        # This passes — no validation. The list contains non-strings
        # that will break if used as file paths downstream.
        assert result == [1, None, True, "ok"]
