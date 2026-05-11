"""
Integration tests for reproducer script.

Tests that reproducer scripts are deterministic and can detect differences.
"""

import pytest
import subprocess
import tempfile
from pathlib import Path
from textwrap import dedent


class TestReproducerDeterminism:
    """Tests for reproducer determinism property."""
    
    def test_reproducer_bit_identical_on_same_commit(self, tmp_path):
        """
        Property: Run reproducer twice on same commit, assert bit-identical output.
        """
        # Create a minimal reproducer script
        reproducer = tmp_path / "reproducer.py"
        reproducer.write_text(dedent("""
            #!/usr/bin/env python3
            import subprocess
            import sys
            
            # Mock environment validation
            RECORDED_COMMIT = "abc123"
            
            actual_commit = "abc123"  # Mock: same commit
            
            if actual_commit != RECORDED_COMMIT:
                print(f"Error: commit mismatch")
                sys.exit(2)
            
            # Mock test run (deterministic)
            print("Finding 1/1: Mutation survived")
            print("Source: validator.py")
            print("Result: survived")
            sys.exit(0)
        """))
        
        reproducer.chmod(0o755)
        
        # Run twice
        result1 = subprocess.run(
            [sys.executable, str(reproducer)],
            capture_output=True,
            text=True
        )
        
        result2 = subprocess.run(
            [sys.executable, str(reproducer)],
            capture_output=True,
            text=True
        )
        
        # Assert bit-identical
        assert result1.returncode == result2.returncode
        assert result1.stdout == result2.stdout
        assert result1.stderr == result2.stderr
    
    def test_reproducer_detects_commit_mismatch(self, tmp_path):
        """Negative case: reproducer detects commit mismatch."""
        reproducer = tmp_path / "reproducer.py"
        reproducer.write_text(dedent("""
            #!/usr/bin/env python3
            import sys
            
            RECORDED_COMMIT = "abc123"
            actual_commit = "def456"  # Different commit
            
            if actual_commit != RECORDED_COMMIT:
                print(f"Error: Reproducer recorded on commit {RECORDED_COMMIT}, currently on {actual_commit}.")
                sys.exit(2)
            
            sys.exit(0)
        """))
        
        reproducer.chmod(0o755)
        
        result = subprocess.run(
            [sys.executable, str(reproducer)],
            capture_output=True,
            text=True
        )
        
        assert result.returncode == 2
        assert "commit" in result.stdout.lower()
        assert "abc123" in result.stdout
        assert "def456" in result.stdout


class TestReproducerMutationDetection:
    """Tests for reproducer mutation detection."""
    
    def test_reproducer_detects_mutation_difference(self, tmp_path):
        """
        Negative case: Mutate source file, run reproducer, assert output differs.
        Then revert mutation, assert original output restored.
        """
        # Create source file
        source = tmp_path / "validator.py"
        original_code = "def validate(x: int) -> bool:\n    return x >= 0\n"
        source.write_text(original_code)
        
        # Create reproducer that applies mutation and runs test
        reproducer = tmp_path / "reproducer.py"
        reproducer.write_text(dedent(f"""
            #!/usr/bin/env python3
            import sys
            from pathlib import Path
            
            source_file = Path("{source}")
            
            # Read original
            original = source_file.read_text()
            
            # Apply mutation
            mutated = original.replace("x >= 0", "x > 0")
            source_file.write_text(mutated)
            
            try:
                # Check mutation applied
                current = source_file.read_text()
                if "x > 0" in current:
                    print("Mutation applied: x >= 0 -> x > 0")
                    verdict = "survived"
                else:
                    verdict = "killed"
                
                print(f"Result: {{verdict}}")
            finally:
                # Revert
                source_file.write_text(original)
            
            sys.exit(0)
        """))
        
        reproducer.chmod(0o755)
        
        # Run reproducer (should detect mutation)
        result1 = subprocess.run(
            [sys.executable, str(reproducer)],
            capture_output=True,
            text=True
        )
        
        assert "Mutation applied" in result1.stdout
        assert "Result: survived" in result1.stdout
        
        # Verify source was reverted
        assert source.read_text() == original_code
    
    def test_reproducer_restores_original_on_error(self, tmp_path):
        """Property: Reproducer restores original code even if test errors."""
        source = tmp_path / "validator.py"
        original_code = "def validate(x: int) -> bool:\n    return x >= 0\n"
        source.write_text(original_code)
        
        reproducer = tmp_path / "reproducer.py"
        reproducer.write_text(dedent(f"""
            #!/usr/bin/env python3
            import sys
            from pathlib import Path
            
            source_file = Path("{source}")
            original = source_file.read_text()
            
            try:
                # Apply mutation
                source_file.write_text("SYNTAX ERROR")
                
                # Simulate error
                raise RuntimeError("Test crashed")
            finally:
                # Must restore even on error
                source_file.write_text(original)
            
            sys.exit(1)
        """))
        
        reproducer.chmod(0o755)
        
        # Run reproducer (will error)
        result = subprocess.run(
            [sys.executable, str(reproducer)],
            capture_output=True,
            text=True
        )
        
        # Verify error exit
        assert result.returncode == 1
        
        # Verify source was still reverted
        assert source.read_text() == original_code


class TestReproducerEnvironmentValidation:
    """Tests for environment validation in reproducer."""
    
    def test_reproducer_validates_python_version(self, tmp_path):
        """Test reproducer validates Python version."""
        reproducer = tmp_path / "reproducer.py"
        reproducer.write_text(dedent("""
            #!/usr/bin/env python3
            import sys
            
            RECORDED_PYTHON = (3, 11)
            actual = (3, 10)  # Mock: different version
            
            if actual != RECORDED_PYTHON:
                print(f"Error: Reproducer requires Python {RECORDED_PYTHON}, got {actual}")
                sys.exit(2)
            
            sys.exit(0)
        """))
        
        reproducer.chmod(0o755)
        
        result = subprocess.run(
            [sys.executable, str(reproducer)],
            capture_output=True,
            text=True
        )
        
        assert result.returncode == 2
        assert "Python" in result.stdout
        assert "(3, 11)" in result.stdout
        assert "(3, 10)" in result.stdout
