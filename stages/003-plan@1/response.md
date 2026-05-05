## Summary

I've created a detailed implementation plan for the `/assurance-probe` skill based on issue #140. This is a **Layer 4 deterministic test-strength verification skill** that fills a specific gap in the crosscheck plugin: proving that tests claimed to cover invariants actually fail when those invariants are violated.

**Key design decisions:**

1. **Narrow scope**: Focuses solely on mutation probing, vacuity detection, and generator inspection — does NOT duplicate existing skills like `/spec-adversary` (Layer 6 completeness) or `/intent-check` (Layer 5 accuracy)

2. **Deterministic core**: All outputs must be bit-identical on re-runs. Mutation generation, test execution, and reproducer scripts are fully deterministic.

3. **Issue-based workflow**: Matches `/spec-adversary` pattern — produces GitHub issues with ≤3 findings, human triage blocks, and rotation-based (not per-PR) execution.

4. **Phase-gated rollout**: Phase 1 (mutation probe, Python-only) ships first. Phases 2/3 (vacuity/generator probes) only unlock after Phase 1 demonstrates SNR ≥ 1:3.

5. **Kill criterion from day 1**: Same as `/spec-adversary` — SNR <1:5 over 4 weeks → retire for that module. Tracked in `.assurance/probe-tracker.csv`.

The plan includes 12 implementation steps covering skill definition, mutation framework, templates, documentation updates, and a demo. Verification uses property-based tests for determinism plus semi-formal reasoning to validate the probe's output quality.

```json
{
  "context_updates": {
    "plan_track": "formal",
    "plan_step_count": 12
  }
}
```