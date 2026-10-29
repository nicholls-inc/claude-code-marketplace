# Loop Card

Fill this in **before** iterating. If a row is blank, you don't have a loop yet. `find-the-loop` produces this card.

| Field | Answer |
|---|---|
| **Outcome** (what the user will feel) | e.g. "scrolling a 5k-line diff feels smooth" |
| **Metric** + guards | e.g. p50 frame time; guards: memory, test pass rate |
| **Instrument** (how the agent measures it *itself*) | command / test / script / eval, plus its path |
| **Noise floor** (A/A) | spread of the metric over N unchanged runs |
| **Baseline** | value on the current head, with N and date |
| **Keep bar** (pre-registered) | e.g. "≥10% better at p50 over 5 paired runs, guards within noise" |
| **Generalization check** | holdout set, second machine, real traffic after merge |
| **Leash / budget** | timebox, max iterations, API / compute budget |
| **Stop rule** | when to stop, re-plan, or ask |
| **Autonomy level** | L0–L5 (see `docs/autonomy.md`) and its source |
| **Ledger** | where accepted *and rejected* attempts are recorded |
