# Plan: crosscheck: assurance-probe — deterministic test-strength layer (design discussion)

## Verification track
formal

## Steps

1. **Create skill definition** — `crosscheck/skills/assurance-probe/SKILL.md`
   - Layer 4 (deterministic strength) skill scaffolding
   - Mutation probe, vacuity probe, and generator probe instructions (phase-gated)
   - Issue-only output pattern matching `/spec-adversary`
   - Bounded output (≤3 findings), rotation-based, kill criterion from day 1
   - Integration with `.assurance/probe-tracker.csv` tracking

2. **Create mutation framework** — `crosscheck/skills/assurance-probe/lib/mutations.py`
   - Parse `Failure condition` clause from invariant docs
   - Generate 1-3 targeted source mutations per invariant
   - Apply mutation, run covering test, capture killed/survived/errored verdict
   - Deterministic, reproducible results (bit-identical on re-run)
   - Phase 1: Python-only support

3. **Create vacuity probe** — `crosscheck/skills/assurance-probe/lib/vacuity.py`
   - Delete covering test temporarily, run pytest with `--cov`
   - Compute branch-coverage delta for protected module
   - Zero delta = test not load-bearing for that module
   - Restore test after probe
   - Phase 2 (deferred until Phase 1 demonstrates SNR ≥ 1:3)

4. **Create generator probe** — `crosscheck/skills/assurance-probe/lib/hypothesis_probe.py`
   - Static inspection of Hypothesis strategies
   - Check if strategy can produce inputs in failure-condition region
   - Report unreachable failure regions
   - Phase 3 (deferred)

5. **Create reproducer template** — `crosscheck/skills/assurance-probe/templates/reproducer.py.template`
   - Standalone script that re-runs probe findings
   - Committed to adopter repo at `scripts/probe/<module>_<YYYYMMDD>.py`
   - Ensures bit-identical results on same commit
   - Opt-in flag for in-tree placement vs issue-body-only

6. **Create tracker template** — `crosscheck/skills/assurance-probe/templates/probe-tracker.csv.template`
   - Mirrors `.assurance/spec-adversary-log.csv` shape
   - Tracks: date, module, proposed, accepted, rejected, deferred
   - SNR calculation for kill-criterion enforcement

7. **Create GitHub issue template** — `crosscheck/skills/assurance-probe/templates/issue.md.template`
   - At most 3 findings per run
   - Each finding: mutation diff, test command, observed result
   - Accept/reject/defer triage block per finding
   - Link to reproducer script

8. **Update assurance hierarchy docs** — `crosscheck/docs/assurance-hierarchy.md`
   - Add `/assurance-probe` to Layer 4 row in skill→layer mapping table
   - Update "Getting started" section to mention probe in rotation workflow
   - Add probe to "When to use what" decision tree

9. **Update crosscheck README** — `crosscheck/README.md`
   - Add `/assurance-probe` to Layer 4 bullet in "What you can run right now"
   - Add to Skills overview section under "Assurance hierarchy & governance"
   - Add worked example to "Worked examples" section

10. **Create demo script** — `crosscheck/demo/07_test_strength/SCRIPT.md`
    - Demonstrate mutation probe on a real invariant
    - Show killed/survived/errored verdicts
    - Show reproducer script in action
    - Timing budget: ~5 minutes

11. **Update byfuglien agent** — `crosscheck/agents/byfuglien.md`
    - Add `/assurance-probe` to skill registry
    - Add routing rule for "test strength", "mutation testing", "invariant probe" triggers
    - Note: probe is rotation-based, not per-PR

12. **Create reference documentation** — `crosscheck/skills/assurance-probe/references/phase-gating.md`
    - Phase 1: mutation probe (Python-only)
    - Phase 2: vacuity probe (gates on Phase 1 SNR ≥ 1:3)
    - Phase 3: generator probe (gates on Phase 2 success)
    - Kill criterion: SNR <1:5 over 4 weeks → retire for that module

## Tests / properties to add

- **Unit test for mutation parser** — `crosscheck/skills/assurance-probe/tests/test_mutations.py`
  - Parse `Failure condition` clause from invariant doc
  - Generate expected mutations for common patterns
  - Property: mutations are deterministic (same input → same mutations)

- **Unit test for vacuity detector** — `crosscheck/skills/assurance-probe/tests/test_vacuity.py`
  - Coverage delta computation
  - Property: deleting test and restoring leaves repo unchanged

- **Integration test for reproducer** — `crosscheck/skills/assurance-probe/tests/test_reproducer.py`
  - Run reproducer script twice, assert bit-identical output
  - Property: reproducer is truly deterministic

- **E2E test for probe workflow** — `crosscheck/skills/assurance-probe/tests/test_e2e.py`
  - Scaffold fake adopter repo with invariant doc + test
  - Run probe, verify issue output
  - Verify tracker CSV update
  - Property: ≤3 findings per run

## Verification approach

### Formal verification (Layer 1)
Not applicable for this skill — the skill orchestrates test execution and mutation, not pure business logic.

### Deterministic verification (Layer 4)
1. **Mutation determinism proof**: Given the same `Failure condition` clause, the mutation generator must produce the same set of mutations every time. Test via property-based tests with Hypothesis:
   - `∀ clause: parse(clause) == parse(clause)`
   - `∀ (clause, seed): mutations(clause, seed) == mutations(clause, seed)`

2. **Reproducer bit-identical property**: The reproducer script must produce identical output on the same commit:
   - Run script twice on commit `C`
   - Assert `output_1 == output_2`
   - Formalized as a pytest parametrized test over 5 real invariant docs

3. **Bounded output enforcement**: The skill must never emit >3 findings per run:
   - Property test: `∀ (module, invariants): len(probe(module, invariants).findings) ≤ 3`
   - Test with modules having 5, 10, 20 invariants

4. **Tracker integrity**: Each probe run must append exactly one row to `.assurance/probe-tracker.csv`:
   - Before: `wc -l tracker.csv` → N
   - After: `wc -l tracker.csv` → N+1
   - Checksum: SHA256 of rows 1..N unchanged

### Semi-formal verification (Layer 5)
Use `/reason` to verify the probe's output on a demo invariant from `demo/07_test_strength/`:
- Does the mutation probe correctly identify a vacuous test?
- Does the reproducer script accurately capture the mutation?
- Does the SNR calculation correctly track findings over time?

Produce an evidence log (similar to `/rationale` output) showing:
- Mutation is derived from `Failure condition` clause
- Test execution verdict matches expected behavior
- Reproducer script matches committed artifact

## Risk register

- **Risk**: Mutation framework generates non-deterministic mutations due to Python's dict iteration order
  - **Mitigation**: Sort mutations by a canonical key (e.g., line number + mutation type) before returning

- **Risk**: Probe runs take >5 minutes on large modules, blocking rotation workflow
  - **Mitigation**: Hard timeout at 5 minutes per module; emit partial results with warning; add to kill criteria

- **Risk**: Vacuity probe deletes test but fails to restore, leaving repo in dirty state
  - **Mitigation**: Use Git worktree or copy; never modify working directory directly. Property test: restore always succeeds.

- **Risk**: Reproducer script committed to repo clutters `scripts/probe/` with stale artifacts
  - **Mitigation**: Document cleanup policy in skill instructions (e.g., delete reproducers >90 days old). Add `.gitignore` entry for `scripts/probe/` in template.

- **Risk**: SNR <1:5 kill criterion triggers prematurely on new adopters (small sample size)
  - **Mitigation**: Require minimum 20 probe runs before SNR is enforceable; document in phase-gating.md

- **Risk**: Phase-gating creates confusion (users try to run Phase 2/3 before Phase 1 succeeds)
  - **Mitigation**: Skill checks `.assurance/probe-tracker.csv` SNR before allowing Phase 2. Emit clear error with SNR value.

- **Risk**: Probe output conflicts with `/spec-adversary` output (both propose changes to same invariant)
  - **Mitigation**: Probe does NOT propose new invariants; it only tests strength of existing ones. Document clear boundary.

- **Risk**: Mutation parser fails on complex `Failure condition` clauses (nested conditions, multi-line)
  - **Mitigation**: Phase 1 supports simple conditions only (single predicate). Complex conditions emit "unsupported" warning and skip.

- **Risk**: Generator probe (Phase 3) reports false positives on Hypothesis strategies with complex dependencies
  - **Mitigation**: Phase 3 deferred until Phase 1/2 demonstrate value. Require human review on all generator findings.
