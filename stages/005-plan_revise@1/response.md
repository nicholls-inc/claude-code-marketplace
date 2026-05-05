The plan looks good. Here is the concern-by-concern resolution table:

---

## Concern-by-Concern Resolution Table

| # | Concern | Status | Affected `plan.md` section |
|---|---------|--------|---------------------------|
| 1 | SKILL.md must specify whether `/assurance-probe` requires the hellebuyck onboarding gate or document what it emits on an unonboarded repo — an empty table is indistinguishable from "all tests are strong" | **addressed** | Step 1 (added explicit onboarding gate spec with verbatim refusal message); Tests (added "Trace: unonboarded repo gate"); Verification approach step 5; Risk register (added "Unonboarded repo emits falsely-green table") |
| 2 | SKILL.md must state a deterministic aggregation rule for a single invariant ID covered by more than one test file — two runs reading files in different orders can produce different scores | **addressed** | Step 1 (added "weakest-wins rule" — minimum score, order-independent, all covering files listed); Tests (added "Trace: multi-file weakest-wins"); Verification approach step 4; Risk register (added "Multi-file coverage produces non-deterministic scores") |
| 3 | Every rubric dimension must be operationalized as specific, grep-able, language-agnostic keywords — "mutation probe hint" is named but never concretized | **addressed** | Step 1 (all 5 dimensions now list exact grep-able keyword lists: boundary markers, property-framework imports, assertion keyword list with threshold, mutation tool names `mutmut`/`pitest`/`stryker`/`#mutant`/`mutpy`, test-function patterns); Risk register updated |
| 4 | The patch comparison verification must check semantic routing conflicts (trigger-phrase non-overlap), not just byte-identity of existing rows | **addressed** | Tests (added "Trigger-phrase non-overlap check" property); Verification approach step 7 (explicit check of all new trigger phrases against existing rows, naming the high-risk pairs); Step 2 note on trigger distinctness; Risk register entry updated to reference step 7 |

## Summary of Material Changes

The revised plan adds four concrete additions to the original: (1) an explicit onboarding gate in Step 1 with a verbatim refusal message and a new verification trace for the unonboarded case; (2) a weakest-wins aggregation rule for multi-file invariant coverage with a new determinism trace; (3) full operationalization of all 5 rubric dimensions as specific grep-able keyword lists (mutation probe hint now lists `mutmut`, `pitest`, `stryker`, `#mutant`, `mutpy`); (4) a new trigger-phrase non-overlap verification step (step 7) that explicitly checks the high-risk routing pairs. The risk register gains two new entries. No scope was added beyond what the concerns required.

```json
{
  "context_updates": {
    "plan_revised": true,
    "concerns_addressed": 4,
    "concerns_rejected": 0,
    "plan_track": "semi-formal"
  }
}
```