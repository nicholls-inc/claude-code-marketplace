"""
Unit tests for vacuity probe (Phase 2).

Tests coverage delta computation and Git worktree isolation.
"""

import pytest
from pathlib import Path
from unittest.mock import Mock, patch
from crosscheck.skills.assurance_probe.lib.vacuity import VacuityProbe, VacuityResult


class TestVacuityPrerequisites:
    """Tests for prerequisite checks."""
    
    def test_check_prerequisites_success(self):
        """Test that check_prerequisites succeeds when pytest-cov is available."""
        with patch('importlib.util.find_spec') as mock_find:
            mock_find.return_value = Mock()  # pytest-cov found
            VacuityProbe.check_prerequisites()  # Should not raise
    
    def test_check_prerequisites_missing_pytest_cov(self):
        """Test that check_prerequisites raises when pytest-cov is missing."""
        with patch('importlib.util.find_spec') as mock_find:
            mock_find.return_value = None  # pytest-cov not found
            with pytest.raises(RuntimeError, match="pytest-cov not found"):
                VacuityProbe.check_prerequisites()


class TestCoverageMeasurement:
    """Tests for coverage measurement."""
    
    def test_measure_coverage_parses_output(self):
        """Test that measure_coverage parses pytest-cov output correctly."""
        mock_stdout = """
        ============================= test session starts ==============================
        collected 1 item
        
        tests/test_validator.py .                                              [100%]
        
        ----------- coverage: platform linux, python 3.11.0-final-0 -----------
        Name                    Stmts   Miss Branch BrPart  Cover
        -----------------------------------------------------------------
        src/validator.py            10      2      4      1    85%
        -----------------------------------------------------------------
        TOTAL                       10      2      4      1    85%
        
        ============================== 1 passed in 0.05s ===============================
        """
        
        with patch('subprocess.run') as mock_run:
            mock_run.return_value = Mock(stdout=mock_stdout, stderr="", returncode=0)
            
            coverage = VacuityProbe.measure_coverage(
                "src/validator.py",
                "tests/test_validator.py"
            )
            
            assert coverage == 85.0
    
    def test_measure_coverage_handles_timeout(self):
        """Test that measure_coverage returns 0 on timeout."""
        with patch('subprocess.run') as mock_run:
            mock_run.side_effect = Exception("Timeout")
            
            coverage = VacuityProbe.measure_coverage(
                "src/validator.py",
                "tests/test_validator.py"
            )
            
            assert coverage == 0.0
    
    def test_measure_coverage_excludes_test(self):
        """Test that measure_coverage can exclude a specific test."""
        with patch('subprocess.run') as mock_run:
            mock_run.return_value = Mock(stdout="TOTAL 50%", stderr="", returncode=0)
            
            VacuityProbe.measure_coverage(
                "src/validator.py",
                "tests/test_validator.py",
                exclude_test="test_specific"
            )
            
            # Verify -k flag was passed
            call_args = mock_run.call_args[0][0]
            assert "-k" in call_args
            assert "not test_specific" in call_args


class TestVacuityProbe:
    """Tests for vacuity probe logic."""
    
    def test_probe_vacuous_test(self):
        """Test probe detects vacuous test (zero coverage delta)."""
        with patch.object(VacuityProbe, 'check_prerequisites'):
            with patch.object(VacuityProbe, 'measure_coverage') as mock_measure:
                # Both with and without test: same coverage
                mock_measure.side_effect = [85.0, 85.0]
                
                result = VacuityProbe.probe(
                    "src/validator.py",
                    "tests/test_validator.py",
                    "test_vacuous"
                )
                
                assert result.is_vacuous is True
                assert result.delta == pytest.approx(0.0)
    
    def test_probe_load_bearing_test(self):
        """Test probe detects load-bearing test (positive coverage delta)."""
        with patch.object(VacuityProbe, 'check_prerequisites'):
            with patch.object(VacuityProbe, 'measure_coverage') as mock_measure:
                # With test: 85%, without test: 60%
                mock_measure.side_effect = [85.0, 60.0]
                
                result = VacuityProbe.probe(
                    "src/validator.py",
                    "tests/test_validator.py",
                    "test_load_bearing"
                )
                
                assert result.is_vacuous is False
                assert result.delta == pytest.approx(25.0)
    
    def test_probe_small_delta_tolerance(self):
        """Test that small delta (<1%) is treated as vacuous."""
        with patch.object(VacuityProbe, 'check_prerequisites'):
            with patch.object(VacuityProbe, 'measure_coverage') as mock_measure:
                # With test: 85.0%, without test: 84.5%
                mock_measure.side_effect = [85.0, 84.5]
                
                result = VacuityProbe.probe(
                    "src/validator.py",
                    "tests/test_validator.py",
                    "test_small_delta"
                )
                
                # Delta is 0.5%, which is < 1.0% threshold
                assert result.is_vacuous is True


class TestWorktreeIsolation:
    """Tests for Git worktree isolation."""
    
    def test_probe_with_worktree_creates_and_removes(self):
        """Property: worktree is created and removed, leaving repo unchanged."""
        with patch.object(VacuityProbe, 'check_prerequisites'):
            with patch('subprocess.run') as mock_run:
                with patch.object(VacuityProbe, 'measure_coverage') as mock_measure:
                    mock_measure.side_effect = [85.0, 60.0]
                    mock_run.return_value = Mock(returncode=0)
                    
                    VacuityProbe.probe_with_worktree(
                        "src/validator.py",
                        "tests/test_validator.py",
                        "test_example",
                        Path("/fake/repo")
                    )
                    
                    # Verify worktree add and remove were called
                    calls = [call[0][0] for call in mock_run.call_args_list]
                    
                    # Should have at least one 'git worktree add' and one 'git worktree remove'
                    add_calls = [c for c in calls if 'worktree' in c and 'add' in c]
                    remove_calls = [c for c in calls if 'worktree' in c and 'remove' in c]
                    
                    assert len(add_calls) >= 1
                    assert len(remove_calls) >= 1
