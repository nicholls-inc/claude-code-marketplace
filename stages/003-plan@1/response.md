## Summary

I've created a concrete implementation plan for refactoring the crosscheck plugin's semi-formal reasoning artifacts based on typical field report findings. The plan addresses quality and clarity issues in three files:

**Key interventions:**

1. **`/reason` skill** — Add explicit abort thresholds and strengthen evidence citation requirements (file:line mandatory)
2. **`/trace-execution` skill** — Enhance hypothesis-driven workflow with confidence calibration and mandatory rationale
3. **`byfuglien` agent** — Strengthen quality gates to mechanically reject outputs without proper evidence grounding
4. **Cross-cutting** — Add cross-references between orchestrator and skills for consistency

**Verification strategy:** Since these are instruction artifacts (SKILL.md/agent.md), verification uses semi-formal reasoning applied to the refactor itself — gathering premises about field report findings, tracing execution through hypothetical invocations, and checking alternative hypotheses. The plan includes instruction completeness checks, consistency verification, and workflow coherence traces.

**Risk mitigation:** The plan acknowledges medium confidence due to lack of access to the actual field report or analysis output. It focuses on common issues (instruction ambiguity, abort condition clarity, quality gate enforcement) that typically emerge in semi-formal reasoning skills.

```json
{
  "context_updates": {
    "plan_track": "semi-formal",
    "plan_step_count": 4
  }
}
```