"""
Mutation framework for assurance-probe skill.

Parses `Failure condition` clauses from invariant documentation and generates
targeted source mutations. Phase 1 supports simple predicates only.
"""

import ast
import re
from dataclasses import dataclass
from typing import List, Optional, Tuple


@dataclass
class Mutation:
    """Represents a single mutation to apply to source code."""
    
    original: str
    mutated: str
    line_number: int
    mutation_type: str  # "boundary", "operator", "literal"
    description: str


class FailureConditionParser:
    """
    Parses `Failure condition` clauses from invariant documentation.
    
    Phase 1 grammar: <var> <op> <literal>
    where <op> ∈ {<, >, <=, >=, ==, !=, in, not in}
    """
    
    # Simple predicate pattern
    SIMPLE_PREDICATE_PATTERN = re.compile(
        r'^\s*(.+?)\s+(<=|>=|<|>|==|!=|not\s+in|in)\s+(.+?)\s*$'
    )
    
    # Operator mappings for boundary mutations
    BOUNDARY_MUTATIONS = {
        '<': '>=',
        '>': '<=',
        '<=': '>',
        '>=': '<',
        '==': '!=',
        '!=': '==',
        'in': 'not in',
        'not in': 'in',
    }
    
    @classmethod
    def parse(cls, failure_condition: str) -> Optional[Tuple[str, str, str]]:
        """
        Parse a failure condition clause.
        
        Args:
            failure_condition: The failure condition string from invariant doc
            
        Returns:
            (variable, operator, literal) tuple if parseable, None otherwise
        """
        # Clean up whitespace and common markdown artifacts
        condition = failure_condition.strip()
        condition = condition.strip('`')  # Remove backticks
        
        # Check for multi-line or complex conditions
        if '\n' in condition or '&&' in condition or '||' in condition or 'and' in condition or 'or' in condition:
            return None
        
        match = cls.SIMPLE_PREDICATE_PATTERN.match(condition)
        if not match:
            return None
        
        var, op, literal = match.groups()
        
        # Normalize operator
        op = op.strip()
        
        return (var.strip(), op, literal.strip())
    
    @classmethod
    def generate_mutations(cls, failure_condition: str) -> List[Tuple[str, str, str]]:
        """
        Generate mutation candidates from a failure condition.
        
        Returns list of (original_expr, mutated_expr, mutation_type) tuples.
        
        Example:
            "x < 0" -> [
                ("x < 0", "x >= 0", "boundary"),
                ("x < 0", "x == -1", "literal")
            ]
        """
        parsed = cls.parse(failure_condition)
        if not parsed:
            return []
        
        var, op, literal = parsed
        original_expr = f"{var} {op} {literal}"
        mutations = []
        
        # Boundary mutation (flip operator)
        if op in cls.BOUNDARY_MUTATIONS:
            boundary_op = cls.BOUNDARY_MUTATIONS[op]
            mutations.append((
                original_expr,
                f"{var} {boundary_op} {literal}",
                "boundary"
            ))
        
        # Literal mutation (try to generate an example violation)
        literal_mutation = cls._generate_literal_mutation(var, op, literal)
        if literal_mutation:
            mutations.append((
                original_expr,
                literal_mutation,
                "literal"
            ))
        
        # Sort mutations by type for determinism
        mutations.sort(key=lambda m: (m[2], m[1]))
        
        return mutations
    
    @classmethod
    def _generate_literal_mutation(cls, var: str, op: str, literal: str) -> Optional[str]:
        """
        Generate a literal mutation that violates the failure condition.
        
        For numeric comparisons, try boundary +/- 1.
        For equality/membership, try a different value.
        """
        # Try to parse as numeric
        try:
            num = int(literal)
            if op == '<':
                # x < 0 -> x == -1 (example violation)
                return f"{var} == {num - 1}"
            elif op == '>':
                # x > 10 -> x == 11
                return f"{var} == {num + 1}"
            elif op == '<=':
                # x <= 10 -> x == 10
                return f"{var} == {num}"
            elif op == '>=':
                # x >= 0 -> x == 0
                return f"{var} == {num}"
            elif op == '==':
                # x == 5 -> x == 4
                return f"{var} == {num - 1}"
            elif op == '!=':
                # x != 5 -> x == 5
                return f"{var} == {num}"
        except ValueError:
            pass
        
        # For membership operators, no literal mutation in Phase 1
        # (would require knowledge of collection contents)
        if op in ['in', 'not in']:
            return None
        
        # For string/identifier comparisons, try a different identifier
        if op == '==':
            # state == READY -> state == PENDING
            if literal.isupper():
                return f"{var} == PENDING"
        
        return None


class MutationApplicator:
    """
    Applies mutations to Python source code using AST manipulation.
    """
    
    @staticmethod
    def find_ast_node(source_code: str, target_expr: str) -> Optional[int]:
        """
        Find the line number of an expression in source code.
        
        Args:
            source_code: The Python source code
            target_expr: The expression to find (e.g., "x >= 0")
            
        Returns:
            Line number (1-indexed) if found, None otherwise
        """
        try:
            tree = ast.parse(source_code)
        except SyntaxError:
            return None
        
        # Normalize target expression
        target_normalized = target_expr.replace(' ', '')
        
        for node in ast.walk(tree):
            if isinstance(node, ast.Compare):
                # Reconstruct the comparison expression
                try:
                    node_expr = ast.unparse(node).replace(' ', '')
                    if node_expr == target_normalized:
                        return node.lineno
                except:
                    continue
        
        return None
    
    @staticmethod
    def apply_mutation(
        source_code: str,
        original_expr: str,
        mutated_expr: str,
        line_number: Optional[int] = None
    ) -> Optional[str]:
        """
        Apply a mutation to source code.
        
        Args:
            source_code: The Python source code
            original_expr: The expression to replace
            mutated_expr: The replacement expression
            line_number: Optional line number hint for replacement
            
        Returns:
            Mutated source code if successful, None if mutation failed
        """
        if line_number is None:
            line_number = MutationApplicator.find_ast_node(source_code, original_expr)
            if line_number is None:
                return None
        
        # Split into lines
        lines = source_code.split('\n')
        
        # Validate line number
        if line_number < 1 or line_number > len(lines):
            return None
        
        # Replace on the target line (1-indexed -> 0-indexed)
        target_line = lines[line_number - 1]
        
        # Normalize expressions for matching
        original_normalized = original_expr.replace(' ', '')
        target_normalized = target_line.replace(' ', '')
        
        if original_normalized not in target_normalized:
            return None
        
        # Replace while preserving whitespace structure
        mutated_line = target_line.replace(original_expr, mutated_expr)
        
        # Handle cases where spaces differ
        if mutated_line == target_line:
            # Try with different spacing
            original_parts = original_expr.split()
            mutated_parts = mutated_expr.split()
            
            if len(original_parts) == 3 and len(mutated_parts) == 3:
                # Try to match pattern: var op literal
                pattern = re.escape(original_parts[0]) + r'\s*' + re.escape(original_parts[1]) + r'\s*' + re.escape(original_parts[2])
                mutated_line = re.sub(pattern, mutated_expr, target_line)
        
        lines[line_number - 1] = mutated_line
        return '\n'.join(lines)


def parse_and_mutate(failure_condition: str) -> List[Tuple[str, str, str]]:
    """
    Convenience function for parsing and generating mutations.
    
    Args:
        failure_condition: The failure condition string
        
    Returns:
        List of (original, mutated, type) tuples
    """
    return FailureConditionParser.generate_mutations(failure_condition)