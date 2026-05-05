"""
Unit tests for mutation parser.

Tests the parsing of Failure condition clauses and generation of mutations.
"""

import pytest
from crosscheck.skills.assurance_probe.lib.mutations import (
    FailureConditionParser,
    parse_and_mutate,
)


class TestFailureConditionParser:
    """Tests for FailureConditionParser."""
    
    def test_parse_simple_less_than(self):
        """Test parsing 'x < 0' pattern."""
        result = FailureConditionParser.parse("x < 0")
        assert result == ("x", "<", "0")
    
    def test_parse_simple_greater_than(self):
        """Test parsing 'len(arr) > MAX_SIZE' pattern."""
        result = FailureConditionParser.parse("len(arr) > MAX_SIZE")
        assert result == ("len(arr)", ">", "MAX_SIZE")
    
    def test_parse_membership(self):
        """Test parsing 'key not in cache' pattern."""
        result = FailureConditionParser.parse("key not in cache")
        assert result == ("key", "not in", "cache")
    
    def test_parse_with_backticks(self):
        """Test parsing with markdown backticks."""
        result = FailureConditionParser.parse("`x >= 100`")
        assert result == ("x", ">=", "100")
    
    def test_parse_multiline_returns_none(self):
        """Test that multi-line conditions return None."""
        result = FailureConditionParser.parse("x < 0 and\ny > 10")
        assert result is None
    
    def test_parse_compound_returns_none(self):
        """Test that compound conditions return None."""
        result = FailureConditionParser.parse("x < 0 and y > 10")
        assert result is None
    
    def test_parse_complex_boolean_returns_none(self):
        """Test that complex boolean conditions return None."""
        result = FailureConditionParser.parse("x < 0 or y > 10")
        assert result is None


class TestMutationGeneration:
    """
    Tests for mutation generation with correctness oracle.
    
    Reference table maps Failure condition examples to expected mutations.
    """
    
    # Correctness oracle: (failure_condition, expected_mutations)
    ORACLE = [
        (
            "x < 0",
            [
                ("x < 0", "x >= 0", "boundary"),
                ("x < 0", "x == -1", "literal"),
            ]
        ),
        (
            "len(arr) > MAX_SIZE",
            [
                ("len(arr) > MAX_SIZE", "len(arr) <= MAX_SIZE", "boundary"),
                # Note: literal mutation for non-numeric literals not supported in Phase 1
            ]
        ),
        (
            "key not in cache",
            [
                ("key not in cache", "key in cache", "boundary"),
            ]
        ),
        (
            "balance >= 100",
            [
                ("balance >= 100", "balance < 100", "boundary"),
                ("balance >= 100", "balance == 100", "literal"),
            ]
        ),
        (
            "state == READY",
            [
                ("state == READY", "state != READY", "boundary"),
                ("state == READY", "state == PENDING", "literal"),
            ]
        ),
    ]
    
    @pytest.mark.parametrize("condition,expected", ORACLE)
    def test_mutation_correctness_oracle(self, condition, expected):
        """Test that parse_and_mutate matches oracle for known examples."""
        actual = parse_and_mutate(condition)
        # Sort both for comparison
        actual_sorted = sorted(actual, key=lambda x: (x[2], x[1]))
        expected_sorted = sorted(expected, key=lambda x: (x[2], x[1]))
        assert actual_sorted == expected_sorted
    
    def test_mutation_determinism(self):
        """Property: mutations are deterministic (same input → same output)."""
        condition = "x < 0"
        
        # Run multiple times
        results = [parse_and_mutate(condition) for _ in range(10)]
        
        # All results should be identical
        first = results[0]
        for result in results[1:]:
            assert result == first


class TestBoundaryMutations:
    """Tests for boundary mutation generation."""
    
    def test_less_than_to_greater_equal(self):
        """Test < → >="""
        mutations = parse_and_mutate("x < 0")
        boundary = [m for m in mutations if m[2] == "boundary"]
        assert len(boundary) == 1
        assert boundary[0][1] == "x >= 0"
    
    def test_greater_than_to_less_equal(self):
        """Test > → <="""
        mutations = parse_and_mutate("x > 10")
        boundary = [m for m in mutations if m[2] == "boundary"]
        assert len(boundary) == 1
        assert boundary[0][1] == "x <= 10"
    
    def test_less_equal_to_greater(self):
        """Test <= → >"""
        mutations = parse_and_mutate("x <= 10")
        boundary = [m for m in mutations if m[2] == "boundary"]
        assert len(boundary) == 1
        assert boundary[0][1] == "x > 10"
    
    def test_greater_equal_to_less(self):
        """Test >= → <"""
        mutations = parse_and_mutate("x >= 0")
        boundary = [m for m in mutations if m[2] == "boundary"]
        assert len(boundary) == 1
        assert boundary[0][1] == "x < 0"
    
    def test_equal_to_not_equal(self):
        """Test == → !="""
        mutations = parse_and_mutate("state == READY")
        boundary = [m for m in mutations if m[2] == "boundary"]
        assert len(boundary) == 1
        assert boundary[0][1] == "state != READY"
    
    def test_not_in_to_in(self):
        """Test 'not in' → 'in'"""
        mutations = parse_and_mutate("key not in cache")
        boundary = [m for m in mutations if m[2] == "boundary"]
        assert len(boundary) == 1
        assert boundary[0][1] == "key in cache"


class TestLiteralMutations:
    """Tests for literal mutation generation."""
    
    def test_less_than_literal(self):
        """Test literal mutation for x < 0 generates x == -1"""
        mutations = parse_and_mutate("x < 0")
        literal = [m for m in mutations if m[2] == "literal"]
        assert len(literal) == 1
        assert literal[0][1] == "x == -1"
    
    def test_greater_than_literal(self):
        """Test literal mutation for x > 10 generates x == 11"""
        mutations = parse_and_mutate("x > 10")
        literal = [m for m in mutations if m[2] == "literal"]
        assert len(literal) == 1
        assert literal[0][1] == "x == 11"
    
    def test_less_equal_literal(self):
        """Test literal mutation for x <= 10 generates x == 10"""
        mutations = parse_and_mutate("x <= 10")
        literal = [m for m in mutations if m[2] == "literal"]
        assert len(literal) == 1
        assert literal[0][1] == "x == 10"
    
    def test_greater_equal_literal(self):
        """Test literal mutation for x >= 0 generates x == 0"""
        mutations = parse_and_mutate("x >= 0")
        literal = [m for m in mutations if m[2] == "literal"]
        assert len(literal) == 1
        assert literal[0][1] == "x == 0"


class TestEmptyAndInvalid:
    """Tests for empty and invalid inputs."""
    
    def test_empty_string(self):
        """Test empty string returns empty list."""
        assert parse_and_mutate("") == []
    
    def test_invalid_syntax(self):
        """Test invalid syntax returns empty list."""
        assert parse_and_mutate("not a valid condition") == []
    
    def test_unparseable_condition(self):
        """Test unparseable condition returns empty list."""
        assert parse_and_mutate("x <> 0") == []  # Invalid operator
