## Summary

I've created a plan that addresses the end-to-end integration gap issue in crosscheck's semi-formal reasoning skills. The core problem is that `/locate-fault` and `/reason` can produce component-correct analyses that miss bugs lurking across interface boundaries—when a test exercises multiple modules but the analysis stops at the caller without reading the callee.

**The solution adds mandatory integration validation phases:**

1. **Phase 5 to `/locate-fault`**: After ranking predictions, verify the trace crossed all component boundaries in the test's execution path
2. **Step 4c to `/reason`**: For multi-file questions, mandate documenting each interface crossing and reading callee implementations
3. **Enhanced byfuglien validation**: Enforce integration checks in the orchestrator—if output cites only interface-level code without through-layer verification, reject and re-execute

**Verification via semi-formal reasoning**: The plan includes execution traces on contrived multi-file bugs, certificate inspection to verify cross-boundary tracing, and `/rationale` adequacy checks to confirm the integration validation itself is sound.

**Risks are managed** through skip conditions (single-file bugs exempt), termination criteria (stop at primitives/unrelated code), and active enforcement by byfuglien to prevent users bypassing the new phases.

```json
{
  "context_updates": {
    "plan_track": "semi-formal",
    "plan_step_count": 3
  }
}
```