# Assurance-Probe Specifications

This directory contains **formal specifications** (not executable verification artifacts) that document the mathematical properties verified by the Python test suite.

## Files

### `mutation_determinism.dfy`

**Role**: Specification document (NOT executable verification)

This Dafny file formalizes the key invariants that ensure probe results are deterministic and reproducible:

1. **Mutation determinism**: Same input → same mutations
2. **Bounded output**: ≤3 findings per run
3. **Tracker integrity**: Each probe run appends exactly one row
4. **Mutation soundness**: Mutations target AST nodes from failure conditions
5. **SNR calculation**: Signal-to-noise ratio is well-defined
6. **Phase gating**: Phase 2 requires Phase 1 SNR ≥ 1:3
7. **Reproducer bit-identical property**: Same environment → same output
8. **Atomic tracker updates**: Prevent corruption

**Why not executable?**

The spec contains trivially-true assertions (e.g., `assert BoundedFindings(run) || true;`) because:
- The properties are verified by **Layer 4 property-based tests** in `../tests/`
- The mutation framework orchestrates external test execution (Python/pytest), which cannot be modeled in Dafny
- Dafny cannot verify IO/process spawning without `{:extern}` trust boundaries

**Actual verification**: See `../tests/run_basic_tests.py` (13 property-based tests covering all 8 properties above)

## Verification Track

As stated in `/workspace/plan.md` (line 6):

> "formal" here means **Layer 4 deterministic property testing** (reproducible, bounded, rotation-based), NOT Layer 1 Dafny proofs. The skill orchestrates external test execution and mutation analysis; it has no provable business logic suitable for formal verification.

The Dafny spec is a **design document** that formalizes the properties; the Python tests are the **executable verification**.
