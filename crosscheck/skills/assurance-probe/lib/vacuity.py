"""
Vacuity probe for assurance-probe skill (Phase 2).

Measures whether a test is load-bearing for a module by computing
branch-coverage delta when the test is removed.
"""

import importlib.util
import os
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Optional


@dataclass
class VacuityResult:
    """Result of a vacuity probe."""
    
    module: str
    test_file: str
    baseline_coverage: float  # Coverage with test present
    probe_coverage: float     # Coverage with test removed
    delta: float              # baseline - probe (should be > 0 for load-bearing tests)
    is_vacuous: bool          # True if delta ≈ 0


class VacuityProbe:
    """
    Phase 2 vacuity detector.
    
    Uses pytest-cov to measure branch coverage delta when a test is removed.
    """
    
    @staticmethod
    def check_prerequisites() -> None:
        """
        Verify pytest-cov is installed.
        
        Raises:
            RuntimeError: If pytest-cov is not found
        """
        if importlib.util.find_spec("pytest_cov") is None:
            raise RuntimeError(
                "pytest-cov not found; install via 'pip install pytest-cov' "
                "to enable vacuity probe (Phase 2)."
            )
    
    @staticmethod
    def measure_coverage(
        module_path: str,
        test_file: str,
        exclude_test: Optional[str] = None
    ) -> float:
        """
        Measure branch coverage for a module.
        
        Args:
            module_path: Path to the module to measure (e.g., "src/validator.py")
            test_file: Path to test file (e.g., "tests/test_validator.py")
            exclude_test: Optional test name to exclude (e.g., "test_validate_input")
            
        Returns:
            Branch coverage percentage (0-100)
        """
        # Build pytest command
        cmd = [
            sys.executable, "-m", "pytest",
            test_file,
            f"--cov={module_path}",
            "--cov-report=term-missing",
            "--cov-branch",
            "-q"
        ]
        
        if exclude_test:
            cmd.extend(["-k", f"not {exclude_test}"])
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=30
            )
            
            # Parse coverage from output
            # Look for line like: "TOTAL    100    20    85%"
            for line in result.stdout.split('\n'):
                if 'TOTAL' in line or module_path in line:
                    parts = line.split()
                    for part in parts:
                        if '%' in part:
                            return float(part.strip('%'))
            
            return 0.0
            
        except subprocess.TimeoutExpired:
            return 0.0
        except Exception:
            return 0.0
    
    @staticmethod
    def probe(module_path: str, test_file: str, test_name: str) -> VacuityResult:
        """
        Run vacuity probe on a test.
        
        Args:
            module_path: Path to module under test
            test_file: Path to test file
            test_name: Name of test to probe
            
        Returns:
            VacuityResult with coverage delta
        """
        VacuityProbe.check_prerequisites()
        
        # Measure baseline coverage (with test)
        baseline = VacuityProbe.measure_coverage(module_path, test_file, exclude_test=None)
        
        # Measure coverage without test
        probe = VacuityProbe.measure_coverage(module_path, test_file, exclude_test=test_name)
        
        # Compute delta
        delta = baseline - probe
        
        # Test is vacuous if removing it causes no coverage loss
        # Allow small epsilon for floating-point comparison
        is_vacuous = abs(delta) < 1.0
        
        return VacuityResult(
            module=module_path,
            test_file=test_file,
            baseline_coverage=baseline,
            probe_coverage=probe,
            delta=delta,
            is_vacuous=is_vacuous
        )
    
    @staticmethod
    def probe_with_worktree(
        module_path: str,
        test_file: str,
        test_name: str,
        repo_root: Path
    ) -> VacuityResult:
        """
        Run vacuity probe using Git worktree for isolation.
        
        This ensures the working directory is never modified directly.
        
        Args:
            module_path: Path to module under test (relative to repo root)
            test_file: Path to test file (relative to repo root)
            test_name: Name of test to probe
            repo_root: Path to repository root
            
        Returns:
            VacuityResult with coverage delta
        """
        VacuityProbe.check_prerequisites()
        
        # Create temporary worktree
        with tempfile.TemporaryDirectory() as tmpdir:
            worktree_path = Path(tmpdir) / "worktree"
            
            # Add worktree
            subprocess.run(
                ["git", "worktree", "add", str(worktree_path), "HEAD"],
                cwd=repo_root,
                check=True,
                capture_output=True
            )
            
            try:
                # Measure baseline (with test)
                baseline = VacuityProbe.measure_coverage(
                    str(worktree_path / module_path),
                    str(worktree_path / test_file),
                    exclude_test=None
                )
                
                # Measure without test
                probe = VacuityProbe.measure_coverage(
                    str(worktree_path / module_path),
                    str(worktree_path / test_file),
                    exclude_test=test_name
                )
                
                delta = baseline - probe
                is_vacuous = abs(delta) < 1.0
                
                return VacuityResult(
                    module=module_path,
                    test_file=test_file,
                    baseline_coverage=baseline,
                    probe_coverage=probe,
                    delta=delta,
                    is_vacuous=is_vacuous
                )
                
            finally:
                # Remove worktree
                subprocess.run(
                    ["git", "worktree", "remove", str(worktree_path), "--force"],
                    cwd=repo_root,
                    capture_output=True
                )
