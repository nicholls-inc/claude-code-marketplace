#!/usr/bin/env python3
"""
Basic test runner for assurance-probe (no pytest required).

Runs core determinism and correctness tests.
"""

import sys
from pathlib import Path

# Add lib directory to path
lib_dir = Path(__file__).parent.parent / "lib"
sys.path.insert(0, str(lib_dir))

from mutations import (
    FailureConditionParser,
    parse_and_mutate,
)


def test_parse_simple_conditions():
    """Test parsing of simple failure conditions."""
    print("Testing failure condition parsing...")
    
    # Test 1: Simple less-than
    result = FailureConditionParser.parse("x < 0")
    assert result == ("x", "<", "0"), f"Expected ('x', '<', '0'), got {result}"
    print("  ✓ Parse 'x < 0'")
    
    # Test 2: Greater-than with function call
    result = FailureConditionParser.parse("len(arr) > MAX_SIZE")
    assert result == ("len(arr)", ">", "MAX_SIZE"), f"Unexpected result: {result}"
    print("  ✓ Parse 'len(arr) > MAX_SIZE'")
    
    # Test 3: Membership
    result = FailureConditionParser.parse("key not in cache")
    assert result == ("key", "not in", "cache"), f"Unexpected result: {result}"
    print("  ✓ Parse 'key not in cache'")
    
    # Test 4: Complex (should return None)
    result = FailureConditionParser.parse("x < 0 and y > 10")
    assert result is None, f"Expected None for complex condition, got {result}"
    print("  ✓ Reject complex condition")


def test_mutation_generation():
    """Test mutation generation with correctness oracle."""
    print("\nTesting mutation generation...")
    
    # Oracle test 1: x < 0
    mutations = parse_and_mutate("x < 0")
    mutations_sorted = sorted(mutations, key=lambda x: (x[2], x[1]))
    
    expected = [
        ("x < 0", "x == -1", "literal"),
        ("x < 0", "x >= 0", "boundary"),
    ]
    expected_sorted = sorted(expected, key=lambda x: (x[2], x[1]))
    
    assert mutations_sorted == expected_sorted, \
        f"Expected {expected_sorted}, got {mutations_sorted}"
    print("  ✓ Mutations for 'x < 0'")
    
    # Oracle test 2: balance >= 100
    mutations = parse_and_mutate("balance >= 100")
    boundary = [m for m in mutations if m[2] == "boundary"]
    assert len(boundary) == 1
    assert boundary[0][1] == "balance < 100"
    print("  ✓ Boundary mutation for 'balance >= 100'")


def test_determinism():
    """Test that mutations are deterministic."""
    print("\nTesting determinism...")
    
    condition = "x < 0"
    
    # Run 10 times
    results = [parse_and_mutate(condition) for _ in range(10)]
    
    # All should be identical
    first = results[0]
    for i, result in enumerate(results[1:], 1):
        assert result == first, \
            f"Run {i+1} produced different result: {result} vs {first}"
    
    print(f"  ✓ Determinism verified over 10 runs")


def test_boundary_mutations():
    """Test boundary mutation operators."""
    print("\nTesting boundary mutations...")
    
    tests = [
        ("x < 0", "x >= 0"),
        ("x > 10", "x <= 10"),
        ("x <= 10", "x > 10"),
        ("x >= 0", "x < 0"),
        ("state == READY", "state != READY"),
        ("key not in cache", "key in cache"),
    ]
    
    for condition, expected_mutation in tests:
        mutations = parse_and_mutate(condition)
        boundary = [m for m in mutations if m[2] == "boundary"]
        assert len(boundary) == 1, f"Expected 1 boundary mutation for '{condition}'"
        assert boundary[0][1] == expected_mutation, \
            f"For '{condition}', expected '{expected_mutation}', got '{boundary[0][1]}'"
        print(f"  ✓ {condition} → {expected_mutation}")


def main():
    """Run all tests."""
    print("=" * 70)
    print("Assurance-Probe Basic Tests")
    print("=" * 70)
    
    try:
        test_parse_simple_conditions()
        test_mutation_generation()
        test_determinism()
        test_boundary_mutations()
        
        print("\n" + "=" * 70)
        print("✓ All tests passed")
        print("=" * 70)
        return 0
        
    except AssertionError as e:
        print(f"\n✗ Test failed: {e}")
        return 1
    except Exception as e:
        print(f"\n✗ Unexpected error: {e}")
        import traceback
        traceback.print_exc()
        return 1


if __name__ == "__main__":
    sys.exit(main())