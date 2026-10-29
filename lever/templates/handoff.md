# Hand-off

Use this at L2, when a human or the owning team lands the work.

## Summary
<one paragraph: what changed, measured effect, why it's safe>

## Merge order
| Batch | PRs | Depends on | Notes |
|---|---|---|---|
| 1 | #a, #b | none | independent; queue together |
| 2 | #c | #a | |

## Per-PR evidence
| PR | Change | Metric before → after | CI | Review threads | Risk |
|---|---|---|---|---|---|
| #a | | | ✅ on <sha> | all resolved | low |

## Not included (negative results)
| Idea | Result | PR |
|---|---|---|

## After landing
What to watch, for how long, and how to roll back.
