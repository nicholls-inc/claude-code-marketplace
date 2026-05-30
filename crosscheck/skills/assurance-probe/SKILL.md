---
name: assurance-probe
description: >-
  Measure test strength via mutation/vacuity/generator probes (rotation-based).
  Layer 4 deterministic-strength check: parses the Failure condition clause of
  each invariant, generates targeted source mutations, and runs covering
  property-based tests against them, then adds vacuity (branch-coverage delta)
  and generator-reachability probes. Emits a GitHub issue (<=3 findings per run)
  with accept/reject/defer triage. Rotation-based, not per-PR — triggered
  manually or via an /assurance-status recommendation. Triggers: "is this test
  too weak", "run mutation probe", "check test adequacy", "test strength".
argument-hint: "[optional: invariant ID or module to probe; default is the next in rotation]"
---
# /assurance-probe

**Layer**: 4 (Deterministic strength)  
**Phase**: 1 (experimental; gates on SNR ≥ 1:3 over 20 runs)  
**Scope**: Test strength measurement via mutation + vacuity + generator probes  
**Output**: GitHub issue (≤3 findings per run) with accept/reject/defer triage  

## What it does

Measures the strength of property-based tests covering crosscheck invariants through three probes:

1. **Mutation probe** (Phase 1, Python-only):
   - Parses `Failure condition` clause from invariant documentation
   - Generates 1-3 targeted source mutations per invariant (operators, boundaries, literals)
   - Runs covering test against each mutation
   - Reports verdict: `killed` (good), `survived` (weak test), `errored` (bad mutation or test crash)

2. **Vacuity probe** (Phase 2, gates on Phase 1 SNR ≥ 1:3):
   - Deletes covering test, measures branch-coverage delta
   - Zero delta = test not load-bearing for that module

3. **Generator probe** (Phase 3):
   - Inspects Hypothesis strategies
   - Reports failure regions unreachable by current generator

**Zero-finding output**: If no mutations survive and no vacuities detected, emit success message:
```
No test-strength issues found for module `<module>` (3 invariants tested, 9 mutations killed).
```
Do NOT create a GitHub issue. Update tracker CSV with `accepted=0, rejected=0, deferred=0`.

## When to use

**Rotation-based only** (not per-PR). The skill is designed to be dispatched by an orchestrator or a scheduled agent — rotation is mechanical state-tracking, not a user-driven decision.

Triggers (in order of preference):

- **Scheduled agent / cron** — preferred. A scheduled job emits a marker (per `crosscheck/docs/orchestrator-coordination.md` §1) when a module is due, and this skill consumes it. The marker schema includes the module, the last-probe date, and the next-probe target.
- **Orchestrator dispatch** — `add-orchestrator` or `hellebuyck` runs the rotation as part of a broader assurance check.
- **Direct user invocation** — supported when the user explicitly runs `/assurance-probe <module>`. Not the primary path; the rotation cadence is a system property, not a user prompt to remember.

The `/assurance-status` "consider re-running" line is now a structured emit consumed by the rotation scheduler, not a prompt the user is expected to act on by typing the next command.

**Frequency**: Every 2-4 weeks for active modules (≥1 invariant doc added/modified in last 90 days).

**Kill criterion**: If SNR <1:5 over 4 weeks (minimum 20 runs) for a module, retire probe for that module and document reason.

## Input format

```
/assurance-probe <module>
```

Where `<module>` is a Python module with:
- Invariant docs in `invariants/<module>.md`
- Property-based tests in `tests/test_<module>.py`

## Output format

**GitHub issue** (created only if ≥1 finding):
```markdown
# Test strength findings for `<module>`

**Probe run**: <YYYY-MM-DD HH:MM UTC>  
**Commit**: <git SHA>  
**Environment**: Python <version>, pytest <version>, hypothesis <version>  

## Finding 1 of N: Mutation survived

**Invariant**: [Link to invariant doc]  
**Failure condition**: `x < 0`  
**Mutation**:
```diff
- return x >= 0
+ return x > 0
```

**Test command**: `pytest tests/test_validator.py::test_validate_input`  
**Observed**: Mutation survived (test passed on mutated code)

**Reproducer**: `scripts/probe/<module>_<YYYYMMDD>.py` (optional)

**Triage**:
- [ ] Accept (test is too weak; fix test or refine Failure condition)
- [ ] Reject (false positive; mutation unreachable by generator)
- [ ] Defer (requires Phase 3 generator probe to confirm)

---

[Repeat for findings 2-3 if present]

**SNR tracking**: See `.assurance/probe-tracker.csv`
```

**Tracker update**: Append one row to `.assurance/probe-tracker.csv`:
```csv
2026-05-05,validator,3,1,0,2,0
```
(date, module, proposed, accepted, rejected, deferred, skipped)

## Phase 1 constraints

### Mutation grammar
**Simple predicates only**: `<var> <op> <literal>` where `<op>` ∈ {`<`, `>`, `<=`, `>=`, `==`, `!=`, `in`, `not in`}.

**Examples**:
- `x < 0` → mutations: `x >= 0`, `x == -1`
- `len(arr) > MAX_SIZE` → mutations: `len(arr) <= MAX_SIZE`, `len(arr) == MAX_SIZE`
- `key not in cache` → mutations: `key in cache`

**Complex conditions** (multi-line, nested, compound boolean): Skip with warning:
```
Warning: Complex Failure condition in [invariants/cache.md]; Phase 1 supports simple predicates only. Skipped.
```
Increment `skipped` in tracker CSV.

### Empty/unparseable clauses
If invariant doc has no `Failure condition` section or clause is unparseable:
```
Warning: No parseable Failure condition in [invariants/cache.md]; skipping mutation probe.
```
Increment `skipped` in tracker CSV (`proposed=0, skipped=1`).

### Zero-invariant modules
If `invariants/<module>.md` does not exist or contains no invariants:
- Do NOT create GitHub issue
- Log in rotation summary: "Module X: 0 invariants found; skipping probe."
- No tracker CSV row

### Error handling
- **Syntax error** in mutated code → `errored` verdict (not `survived`)
- **Test framework crash** → `errored` verdict
- **Timeout** (>30s per test) → `errored` verdict
- **Coverage tool missing** (pytest-cov): Fail-fast with actionable error:
  ```
  Error: pytest-cov not found; install via 'pip install pytest-cov' to enable vacuity probe (Phase 2).
  ```

### Mutation soundness
**Constraint**: Each mutation must target an AST node (variable, operator, literal) explicitly referenced in the `Failure condition`. Parser validates this; if no AST match, skip with warning:
```
Warning: No AST match for Failure condition in [invariants/X.md]; skipping mutation probe.
```

### Determinism
**Bit-identical on re-run** with same:
- Git commit SHA
- Python `major.minor` version
- `pytest` version
- `hypothesis` version

OS/platform informational only (logged but not enforced).

### Bounded output
**≤3 findings per run**. If >3 mutations survive, prioritize by:
1. Mutations closest to boundary conditions
2. Mutations in most recently modified code (by `git log`)
3. Random selection (seeded by date) for ties

## Integration

### Tracker CSV
`.assurance/probe-tracker.csv` schema:
```csv
date,module,proposed,accepted,rejected,deferred,skipped
2026-05-05,validator,3,1,0,2,0
```

**Concurrency safety**:
- Acquire file lock (`fcntl.flock` on Unix, `msvcrt.locking` on Windows) before append
- Timeout: 10s; if unavailable, write to `.assurance/probe-tracker.csv.pending` with warning
- Atomic write: write to `.assurance/probe-tracker.csv.tmp`, then rename (POSIX atomic)
- Backup before write: copy to `.assurance/probe-tracker.csv.backup`
- Checksum validation: SHA256 of existing rows unchanged

### Reproducer script
**Template**: `crosscheck/skills/assurance-probe/templates/reproducer.py.template`

**Environment capture**:
```python
import subprocess
import sys
import importlib.metadata
import platform

# Validate environment
RECORDED_COMMIT = "abc123..."
RECORDED_PYTHON = (3, 11)
RECORDED_PYTEST = "7.4.0"
RECORDED_HYPOTHESIS = "6.92.0"

actual_commit = subprocess.check_output(['git', 'rev-parse', 'HEAD']).decode().strip()
if actual_commit != RECORDED_COMMIT:
    print(f"Error: Reproducer recorded on commit {RECORDED_COMMIT}, currently on {actual_commit}.")
    print("Findings may not reproduce.")
    sys.exit(2)

if sys.version_info[:2] != RECORDED_PYTHON:
    print(f"Error: Reproducer requires Python {RECORDED_PYTHON}, got {sys.version_info[:2]}")
    sys.exit(2)

# ... run mutation probe ...
```

**Placement**: Opt-in flag for committing to `scripts/probe/<module>_<YYYYMMDD>.py` vs issue-body-only.

**Cleanup**: Document in skill instructions: delete reproducers >90 days old. Template includes `.gitignore` entry.

## Rotation mechanics

**Triggered by**:
- Manual byfuglien query: "run assurance probe on module validator"
- `/assurance-status` recommendation: "Last probe for module X: 28 days ago (threshold: 14-28 days). Consider re-running."

**Recommended frequency**: Every 2-4 weeks for active modules.

**Phase gates**:
- Phase 2 (vacuity probe): Requires Phase 1 SNR ≥ 1:3 over last 20 runs in `.assurance/probe-tracker.csv`
- Phase 3 (generator probe): Requires Phase 2 success (≥5 vacuities caught, SNR ≥ 1:3)

Check gates before running:
```python
def can_run_phase2(module: str) -> bool:
    tracker = read_tracker_csv()
    last_20 = tracker[tracker['module'] == module].tail(20)
    if len(last_20) < 20:
        return False
    signal = last_20['accepted'].sum()
    noise = last_20['rejected'].sum()
    return signal / max(noise, 1) >= 1/3
```

If gate not met:
```
Phase 2 vacuity probe not available: Phase 1 SNR = 0.25 (3 accepted / 12 rejected over last 20 runs).
Required: SNR ≥ 0.33. Continue with Phase 1 mutation probe only.
```

## Examples

### Example 1: Mutation killed (good test)
```
Finding 1 of 1: Mutation killed

Invariant: invariants/validator.md  
Failure condition: `x < 0`  
Mutation: `x >= 0` → `x > 0`  
Test: pytest tests/test_validator.py::test_validate_input  
Observed: Mutation killed (test failed on x=0)

Triage: [Accept] Test is strong; boundary condition well-covered.
```

### Example 2: Mutation survived (weak test)
```
Finding 1 of 1: Mutation survived

Invariant: invariants/validator.md  
Failure condition: `x < 0`  
Mutation: `x >= 0` → `x > 0`  
Test: pytest tests/test_validator.py::test_validate_input  
Observed: Mutation survived (test passed on x > 0)

Triage: [Accept] Test generator does not include x=0. Fix: Add `integers(min_value=0, max_value=0)` strategy.
```

### Example 3: Mutation errored (syntax error)
```
Finding 1 of 1: Mutation errored

Invariant: invariants/parser.md  
Failure condition: `len(tokens) > 0`  
Mutation: `len(tokens) > 0` → `len(tokens) >= 0`  
Test: pytest tests/test_parser.py::test_parse  
Observed: Mutation errored (SyntaxError in mutated code)

Triage: [Reject] Mutation introduced syntax error; not a valid test-strength signal.
```

## Known limitations (Phase 1)

1. **Mutation soundness**: Generated mutations may not be reachable by the test's input generator. Full reachability analysis requires symbolic execution (deferred to Phase 3). Accept false negatives (mutations that survive due to generator gaps, not test weaknesses) as inherent to deterministic mutation testing.

2. **Python-only**: Go, TypeScript mutation probes deferred to Phase 2+.

3. **Simple predicates only**: Complex `Failure condition` clauses (nested, multi-line) skipped with warning.

4. **Reproducer environment lockdown**: Requires exact match on Git SHA + Python version + dependencies. May hinder adoption if environment varies across team members.

## References

- Phase-gating criteria: `crosscheck/skills/assurance-probe/references/phase-gating.md`
- Mutation grammar: `crosscheck/skills/assurance-probe/lib/mutations.py`
- Tracker schema: `.assurance/probe-tracker.csv.template`
