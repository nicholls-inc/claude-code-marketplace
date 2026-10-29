# Ledger

Append one row for every attempt, **including rejects**. Negative results are data: they stop the next agent from retrying them.

| # | Date | Hypothesis / change | Arm / PR | Metric (control → treatment, Δ%) | n | Noise floor | Verdict | Why |
|---|---|---|---|---|---|---|---|---|
| 1 | | | | | | | keep / reject / inconclusive | |

Verdict rules: **keep** only if the pre-registered keep bar is met. **inconclusive** if Δ is inside the noise floor; that's not a win. Closed PRs for rejected arms are labeled `[negative result]` and linked here.
