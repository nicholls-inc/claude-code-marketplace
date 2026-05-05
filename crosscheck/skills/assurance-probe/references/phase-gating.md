# Phase-gating for /assurance-probe

This document defines the phase progression for the assurance-probe skill,
including entry criteria, success metrics, and kill criteria for each phase.

## Overview

Assurance-probe implements test strength measurement in three phases:

1. **Phase 1** (current): Mutation probe (Python-only)
2. **Phase 2** (gates on Phase 1 SNR ≥ 1:3): Vacuity probe
3. **Phase 3** (gates on Phase 2 success): Generator probe (Hypothesis strategy inspection)

Each phase must demonstrate practical value (signal-to-noise ratio) before
the next phase is enabled.

## Phase 1: Mutation Probe

**Status**: Experimental (active)

### What it does
- Parses `Failure condition` clauses from invariant documentation
- Generates 1-3 targeted source mutations per invariant
- Runs covering tests against mutations
- Reports verdict: `killed` (strong test), `survived` (weak test), `errored` (bad mutation)

### Supported grammar
Simple predicates only: `<var> <op> <literal>` where `<op>` ∈ {`<`, `>`, `<=`, `>=`, `==`, `!=`, `in`, `not in`}

**Examples**:
- `x < 0` → mutations: `x >= 0`, `x == -1`
- `len(arr) > MAX_SIZE` → mutations: `len(arr) <= MAX_SIZE`, `len(arr) == MAX_SIZE`
- `key not in cache` → mutations: `key in cache`

### Constraints
- Python-only (Go, TypeScript deferred to Phase 2+)
- Simple predicates only (multi-line, nested, compound boolean skipped)
- Mutation must target AST node in `Failure condition`
- ≤3 findings per run (bounded output)
- Bit-identical reproducers (locked to commit SHA + Python version + dependencies)

### Success metrics
**Signal-to-noise ratio (SNR)**:
```
SNR = accepted_findings / max(rejected_findings, 1)
```

Where:
- `accepted` = mutation revealed a real test weakness
- `rejected` = false positive (mutation unreachable, syntax error, etc.)
- `deferred` = requires Phase 3 to confirm

**Phase 1 success criterion**: SNR ≥ 1:3 over last 20 probe runs for a module

### Kill criterion
**Retire probe for a module if**: SNR <1:5 over 4 weeks (minimum 20 runs)

**Rationale**: If signal-to-noise drops below 1:5, the probe is generating too much
noise to be useful. Document the reason and disable probe for that module.

### Known limitations
1. **Mutation soundness**: Generated mutations may not be reachable by test's input
   generator (no symbolic execution in Phase 1). Accept false negatives as inherent
   to deterministic mutation testing.

2. **Reproducer environment lockdown**: Requires exact match on Git SHA + Python
   version + dependencies. May hinder adoption if environment varies across team.

## Phase 2: Vacuity Probe

**Status**: Planned (gates on Phase 1 SNR ≥ 1:3)

### Entry criteria
Phase 1 must achieve SNR ≥ 1:3 over last 20 probe runs for a module.

Checked via:
```python
def can_enable_phase2(module: str) -> bool:
    tracker = read_csv('.assurance/probe-tracker.csv')
    last_20 = tracker[tracker['module'] == module].tail(20)
    if len(last_20) < 20:
        return False
    signal = last_20['accepted'].sum()
    noise = last_20['rejected'].sum()
    return signal / max(noise, 1) >= 1/3
```

If not met:
```
Phase 2 vacuity probe not available: Phase 1 SNR = 0.25 (3 accepted / 12 rejected over last 20 runs).
Required: SNR ≥ 0.33. Continue with Phase 1 mutation probe only.
```

### What it does
- Deletes covering test temporarily (via Git worktree)
- Measures branch-coverage delta for protected module
- Zero delta = test not load-bearing for that module
- Restores test after probe (no working-directory mutation)

### Prerequisites
- `pytest-cov` installed (checked via `importlib.util.find_spec("pytest_cov")`)
- Git repository (for worktree isolation)

### Success metrics
**Phase 2 success criterion**: ≥5 vacuities caught, SNR ≥ 1:3 over 20 runs

## Phase 3: Generator Probe

**Status**: Planned (gates on Phase 2 success)

### Entry criteria
Phase 2 must demonstrate:
- ≥5 vacuities caught (across all modules)
- SNR ≥ 1:3 over 20 probe runs

### What it does
- Static inspection of Hypothesis strategies
- Checks if strategy can produce inputs in failure-condition region
- Reports unreachable failure regions

### Implementation requirements
- Symbolic constraint solving (Z3, CVC4)
- Hypothesis strategy AST analysis
- Integration with `/suggest-specs` for strategy recommendations

### Success metrics
**Phase 3 success criterion**: ≥10 generator gaps caught, SNR ≥ 1:3 over 20 runs

## SNR Tracking

All probe runs update `.assurance/probe-tracker.csv`:

```csv
date,module,proposed,accepted,rejected,deferred,skipped
2026-05-05,validator,3,1,1,1,0
2026-05-12,validator,2,2,0,0,0
2026-05-19,validator,1,1,0,0,0
```

**Columns**:
- `date`: Probe run date (YYYY-MM-DD)
- `module`: Module probed
- `proposed`: Total findings reported
- `accepted`: Findings triaged as real weaknesses
- `rejected`: Findings triaged as false positives
- `deferred`: Findings requiring Phase 3 to confirm
- `skipped`: Invariants with unparseable `Failure condition` clauses

**SNR calculation**:
```python
def calculate_snr(module: str, window: int = 20) -> float:
    tracker = read_csv('.assurance/probe-tracker.csv')
    last_n = tracker[tracker['module'] == module].tail(window)
    signal = last_n['accepted'].sum()
    noise = last_n['rejected'].sum()
    return signal / max(noise, 1)
```

**Kill criterion enforcement**:
```python
def should_retire(module: str) -> bool:
    # Require minimum 20 runs before SNR is enforceable
    tracker = read_csv('.assurance/probe-tracker.csv')
    module_runs = tracker[tracker['module'] == module]
    if len(module_runs) < 20:
        return False
    
    # Check last 4 weeks
    four_weeks_ago = date.today() - timedelta(weeks=4)
    recent = module_runs[module_runs['date'] >= four_weeks_ago]
    
    if len(recent) < 20:
        return False
    
    signal = recent['accepted'].sum()
    noise = recent['rejected'].sum()
    snr = signal / max(noise, 1)
    
    return snr < 1/5
```

## Reverting to Earlier Phases

If SNR degrades in a later phase, revert to the previous phase:

- Phase 3 SNR <1:5 → revert to Phase 2
- Phase 2 SNR <1:5 → revert to Phase 1
- Phase 1 SNR <1:5 → retire probe for module

Document the reversion reason in tracker CSV or governance notes.

## Module-Specific Kill Criteria

Some modules may not be suitable for mutation testing:

1. **IO-heavy modules**: Mutations may introduce side effects that break tests
2. **Non-deterministic modules**: Random number generation, time-based logic
3. **External dependencies**: Mocked API calls, database queries

If a module consistently shows SNR <1:5, retire the probe for that module
and document the reason:

```markdown
## Assurance-probe retirement for module `api_client`

**Reason**: Mutation probe generates false positives due to mocked HTTP responses.
Mutations that change request parameters are not caught by mocks, but this doesn't
indicate weak tests — the mock layer is intentionally simplified.

**Decision**: Retire mutation probe for this module. Continue with integration tests
that exercise real API endpoints.
```

## Experimental Label Removal

Once Phase 1 demonstrates SNR ≥ 1:3 over 20 runs across ≥3 modules, remove
"experimental" label from:

- `/assurance-probe` SKILL.md
- Byfuglien agent routing
- README and assurance-hierarchy.md documentation

Mark as "stable" and recommend for production use.
