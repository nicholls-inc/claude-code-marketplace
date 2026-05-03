## Plan Summary

I've created a focused refactor plan to address the three critical weaknesses identified in the field report from the wistful-pet session:

**Problem**: The crosscheck plugin performed deep analysis but missed critical details in execution tracing and didn't verify proposed fixes.

**Solution** (4 steps):
1. Add "Verify the Fix" step to `/reason` to catch when proposed solutions reintroduce the same bug pattern (addresses W3: TOCTOU fix reintroduced TOCTOU)
2. Add exhaustive reading instruction to `/trace-execution` with coverage tracking (addresses W1: missed critical filter)
3. Add confidence calibration to `/trace-execution` with mechanical constraints tying confidence to coverage completeness (addresses W2: overconfident assertion)
4. Update byfuglien's validation rules to enforce the new coverage and confidence requirements

All changes are additive and scoped to semi-formal reasoning skills. The verification approach uses execution trace simulation against the original wistful-pet scenario to confirm each change would have prevented the observed failure. No automated tests needed—these are behavioral artifact changes verified through manual session testing.

The plan addresses the root causes identified in the field report while keeping changes minimal and reversible.

```json
{
  "context_updates": {
    "plan_track": "semi-formal",
    "plan_step_count": 4
  }
}
```