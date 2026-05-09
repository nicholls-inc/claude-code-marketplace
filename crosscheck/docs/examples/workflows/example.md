# Worked example — assurance hierarchy on a work queue

This walkthrough wires the reference workflows in `tier-a/` and `tier-b/`
into a single hypothetical repo: an in-memory work queue with retry
semantics and a dead-letter path. The domain is small enough to fit in
one document, and the invariants are exactly the kind that LLM agents
silently break — FIFO ordering, dedup, and the no-redelivery rule for
poison messages.

Nothing in this example is proprietary. Every code snippet is hand-rolled
for documentation. If you want a real codebase to study, find your own.

## The repo at a glance

```
work-queue/
├── queue/
│   ├── __init__.py
│   └── work_queue.py            — implementation (Layer 1–3 candidate)
├── tests/
│   └── test_work_queue.py       — property tests with invariant comments
├── docs/
│   ├── invariants/
│   │   └── queue.md             — invariant prose (Class B protected)
│   └── assurance/
│       ├── ROADMAP.md           — assurance roadmap
│       ├── acceptance/
│       │   ├── scenarios.yaml   — Layer 5 acceptance scenarios
│       │   └── run_acceptance.py
│       └── attestations/        — populated by the PR-Gate over time
├── scripts/
│   └── check_invariant_coverage.py
├── .github/workflows/
│   ├── assurance.yml            — Layer 4 static gate
│   ├── assurance-pr-gate.md     — Layer 5 gh-aw gate
│   ├── assurance-recheck.md     — Layer 5 force-recheck
│   ├── assurance-squad.md       — daily Layer 4–6 progressive runner
│   └── scripts/
│       ├── assurance_pr_gate_plan.py
│       └── assurance_squad_select.py
├── .pre-commit-config.yaml
└── .claude/rules/protected-surfaces.md
```

## Layer 4 — the invariant doc and its covering tests

`docs/invariants/queue.md` (excerpt):

```markdown
# WorkQueue invariants

The work queue is a FIFO buffer with at-most-`max_retries` redelivery and
a separate dead-letter store. Three properties are load-bearing.

**Q1. FIFO_ORDER.** Items are dequeued in the order they were enqueued.
Reordering would break ordered-processing consumers.

**Q2. DEAD_LETTER_TERMINAL.** Once an item has failed `max_retries`
times, it moves to the dead-letter store and never returns to the main
queue. Redelivering a poison item silently retries forever.

**Q3. NO_DUPLICATE_DELIVERY.** Each item is delivered to a consumer at
most once before either being acknowledged or moving to dead-letter.
Duplicate delivery would force consumers to dedup, defeating the point
of having a queue.
```

`tests/test_work_queue.py` (excerpt):

```python
from queue.work_queue import WorkQueue


# Invariant Q1: FIFO_ORDER.
def test_fifo_order_preserved():
    q = WorkQueue()
    for x in ("A", "B", "C"):
        q.enqueue(x)
    assert [q.dequeue() for _ in range(3)] == ["A", "B", "C"]


# Invariant Q2: DEAD_LETTER_TERMINAL.
def test_poison_item_does_not_redeliver():
    q = WorkQueue(max_retries=3)
    q.enqueue("P")
    for _ in range(4):
        item = q.dequeue()
        q.fail(item)
    assert "P" in q.dead_letter()
    for _ in range(10):
        assert q.dequeue(timeout=0.0) != "P"


# Invariant Q3: NO_DUPLICATE_DELIVERY.
def test_no_duplicate_delivery_before_ack():
    q = WorkQueue()
    q.enqueue("X")
    first = q.dequeue()
    second = q.dequeue(timeout=0.0)
    assert first == "X"
    assert second is None  # nothing else in queue, X is in-flight
```

The pre-commit hook and `assurance.yml` both invoke
`check_invariant_coverage.py`, which:

1. Scans `docs/invariants/queue.md` for `**Q1.`, `**Q2.`, `**Q3.`
2. Scans `tests/test_*.py` for `# Invariant Q1:`, `# Invariant Q2:`,
   `# Invariant Q3:`
3. Reports any declared-but-uncovered or covered-but-undeclared IDs

If a contributor deletes `test_poison_item_does_not_redeliver` to fix a
"flaky test" complaint, both the local commit and the CI build fail with:

```
Missing coverage — add `# Invariant <ID>: <Name>` above the property test:
  - queue/Q2 declared at docs/invariants/queue.md:9 (no covering test)
```

## Layer 5 — acceptance scenarios

`docs/assurance/acceptance/scenarios.yaml` (excerpt — see
`scenarios.template.yaml` for the full template):

```yaml
scenarios:
  - id: QUEUE-001
    flow: work-queue
    name: FIFO ordering preserved across enqueue/dequeue
    target:
      module: queue.work_queue
      function: WorkQueue.dequeue
    inputs:
      enqueue_order: ["A", "B", "C"]
    assertions:
      - id: QUEUE-001-A1
        description: Dequeue produces items in enqueue order
        check: assert dequeued == ["A", "B", "C"]
        verifiable: true
    layer: "Layer 5 (empirical)"
    status: skeleton
```

`docs/assurance/acceptance/run_acceptance.py` (excerpt — see the full
template):

```python
def _run_queue_001() -> list[AssertionResult]:
    from queue.work_queue import WorkQueue

    q = WorkQueue()
    for item in ("A", "B", "C"):
        q.enqueue(item)
    dequeued = [q.dequeue() for _ in range(3)]

    return [AssertionResult(
        "QUEUE-001", "QUEUE-001-A1", "FIFO order",
        passed=(dequeued == ["A", "B", "C"]),
        error=None if dequeued == ["A", "B", "C"] else f"got {dequeued}",
    )]


_RUNNERS = {"QUEUE-001": _run_queue_001, …}
```

The acceptance runner is the user-perspective proxy: it doesn't replace
the property tests, it answers "does the queue actually behave the way a
caller expects?" — which catches flow-level regressions that pass
property tests run in isolation. Wire it into `assurance.yml` as a
non-blocking job until you trust the scenarios.

## Layer 5 — the PR-Gate in flight

A contributor opens a PR that edits `queue/work_queue.py` and rewrites
the test for Q2 because "max_retries was off-by-one". The PR also adds a
single line to `docs/invariants/queue.md`:

```diff
- **Q2. DEAD_LETTER_TERMINAL.** Once an item has failed `max_retries`
- times, it moves to the dead-letter store and never returns to the main
- queue. Redelivering a poison item silently retries forever.
+ **Q2. DEAD_LETTER_TERMINAL.** Once an item has failed `max_retries + 1`
+ times, it moves to the dead-letter store and never returns to the main
+ queue. Redelivering a poison item silently retries forever.
```

This trips the PR-Gate. `assurance-pr-gate.md` runs:

1. `assurance_pr_gate_plan.py` computes the content hash for Q2 (its
   prose changed, the covering test changed, and the module source
   changed). No cache hit. `action: run_intent_check`.
2. `assurance-pr-gate.md` invokes `/crosscheck:intent-check` with:
   - The new Q2 prose (includes "max_retries + 1")
   - The new covering test (`test_poison_item_does_not_redeliver`)
   - The diff to `queue/work_queue.py`
3. The skill's blind back-translator reads only the code + test and
   describes the behaviour: "the queue moves an item to dead-letter
   after the n+1th failure, where n is the max_retries parameter".
4. The diff-checker compares this back-translation against the prose:
   "max_retries + 1" appears in both — they agree.
5. Verdict: **PASS**. Confidence 84%.

The sticky comment on the PR:

```markdown
<!-- assurance-pr-gate:42 -->
**Assurance PR-Gate summary**
Layer 5 intent-check: 1 fresh, 0 cached, 0 skipped
Attestations pushed: 1
Amendment reminder: yes
Dafny hand-off: no
Kill criterion: inactive

<details>
<summary>Intent-check results (click to expand)</summary>

> **Layer 5 — Intent-check (probabilistic)**
>
> Invariant: `Q2` (`DEAD_LETTER_TERMINAL`) in `docs/invariants/queue.md`
> Verdict: **PASS**
> Attestation: `sha256:7f4f9fb…` (`docs/assurance/attestations/7f4f9fb….json`)
> FP-tracker rolling 14 d: 12% (n=8)
>
> Back-translation aligned with prose: both describe the n+1th-failure
> threshold and the no-redelivery property. The off-by-one rewording in
> the prose matches the off-by-one fix in `work_queue.py:dequeue`.
>
> _Layer 5 is probabilistic. Verdict is non-binding; reviewers may
> accept, reject, or defer with rationale. Force a fresh re-run with
> `/assurance-recheck Q2`._

</details>

<details>
<summary>Governance amendment reminder (click to expand)</summary>

> **Governance amendment required**
>
> This PR touches protected-surface files but does not include a
> `## Governance Amendment` block in the PR body.
>
> Files touched: `docs/invariants/queue.md`, `tests/test_work_queue.py`
>
> Add this block to the PR body (use
> `/crosscheck:protected-surface-amend` to generate it).

</details>

<details>
<summary>Dafny hand-off (click to expand)</summary>

No Dafny candidates in this PR.

</details>

---
_Last updated: 2026-05-05T14:32Z · run [#218](…) · head `a3f9b2c`_
```

The contributor adds the governance amendment block, the PR-Gate re-fires
on `synchronize`, the amendment reminder block now reads "No amendment
required.", and the sticky updates in place.

## Layer 5 — force-recheck

A reviewer reads the back-translation summary and isn't sure the
intent-check actually understood the off-by-one fix. They comment on the
PR:

```
/assurance-recheck Q2
```

`assurance-recheck.md` fires. The pre-step parses `Q2` out of the
comment body, the recheck workflow ignores the cache, runs
`/crosscheck:intent-check` fresh, and posts:

```markdown
> **Layer 5 — Force-rechecked (cache bypassed)**
>
> Invariant: `Q2` (`DEAD_LETTER_TERMINAL`) in `docs/invariants/queue.md`
> Verdict: **PASS**
> Attestation: `sha256:c1d8e2a…` (`docs/assurance/attestations/c1d8e2a….json`)
> Triggered by: `/assurance-recheck` on `@reviewer`
> FP-tracker rolling 14 d: 12% (n=9)
>
> Fresh back-translation aligned with prose. The new attestation has a
> different content hash from the cached one because the LLM produced
> slightly different phrasing — both describe the same behaviour.
>
> _Layer 5 is probabilistic; this re-run bypassed the cache by request._
```

## Layer 4–6 — the daily squad

The Squad runs at 19:00 UTC. On day 1 of the queue project,
`task_selection.json` looks like:

```json
{
  "phase_signals": {
    "has_audit": false,
    "has_roadmap": false,
    "invariant_modules": 0,
    "covered_modules": 0,
    "fp_count_14d": 0,
    "kill_criterion_active": false
  },
  "weights": {
    "T1_layer_audit": 50.0,
    "T2_governance_scaffold": 1.0,
    …
  },
  "selected": ["T1_layer_audit"]
}
```

The squad opens a PR with `docs/assurance/AUDIT.md` from
`/crosscheck:assurance-layer-audit`. On day 2, with `has_audit: true`:

```json
{
  "weights": {
    "T2_governance_scaffold": 50.0,
    …
  },
  "selected": ["T2_governance_scaffold"]
}
```

…and so on through the 11-task lifecycle. By week 6, the queue module
has invariants, a coverage gate, acceptance scenarios, and 30 days of
intent-check FP-tracker data. The squad's weights have flattened into
maintenance mode: roadmap-drift checks, FP-tracker reviews, and
spec-adversary rotations across modules.

If the FP rate creeps above 30 % over a rolling 14-day window with ≥3
classified samples, the kill-criterion fires:

```json
{
  "phase_signals": {"kill_criterion_active": true, …},
  "weights": {"T9_kill_criterion": 100.0, "T3_draft_invariants": 0.0, …},
  "selected": ["T9_kill_criterion"]
}
```

The squad opens a high-priority issue listing the offending invariants
and recommending humans clear the criterion via
`.assurance/kill-criterion.json`. Until cleared, T3/T5/T7 (drafting,
acceptance, adversary) are zeroed out — the LLM-generative tasks halt
while the deterministic ones (governance, ROADMAP drift, Dafny promotion
detection) keep running.

## Layer 4 → 1–3 — Dafny hand-off

A contributor adds `dafny_candidate: true` to the frontmatter of
`docs/invariants/queue.md` because the queue's pure sequential logic
fits Dafny's verification model. On the next squad run, T10 fires:

```markdown
**Module `queue` is a Dafny candidate — hand to byfuglien**

Layer 4 → 1–3 (hand-off).

`docs/invariants/queue.md` sets `dafny_candidate: true`. The module's
core dequeue/fail/dead-letter logic is pure sequential code with three
quantified properties (Q1, Q2, Q3) that Dafny can express directly.

Recommend the byfuglien chain:

1. `/crosscheck:spec-iterate` — draft the formal Dafny spec from
   docs/invariants/queue.md.
2. `/crosscheck:generate-verified` — generate a verified
   implementation in Dafny.
3. `/crosscheck:extract-code` — compile to Python and replace the
   current `queue/work_queue.py`.

Hellebuyck's Layer 5/6 output is best-effort; Dafny's Layer 1–3
verification is deterministic. Use the right tool for the layer.
```

The squad does **not** attempt the Dafny work itself — the spec chain is
byfuglien's domain. The hand-off issue is the seam.

## Putting it together

Across one development cycle, the queue project's protections look like:

- **Layer 4 (every push/PR):** `assurance.yml` + pre-commit hook prove
  the invariants in `queue.md` have property tests in `tests/`.
- **Layer 5 (every protected-surface PR):** `assurance-pr-gate.md` runs
  `/crosscheck:intent-check` with a content-hash cache so unchanged
  invariants don't pay the LLM cost on every push.
- **Layer 5 (on demand):** `/assurance-recheck Q2` bypasses the cache
  when a reviewer wants a fresh take.
- **Layer 5 (daily):** `assurance-squad.md` audits, scaffolds, drafts,
  reviews FPs, alerts on kill-criterion, proposes Dafny promotions.
- **Layer 6 (rotated daily):** `assurance-squad.md` task T7 invokes
  `/crosscheck:spec-adversary` on one invariant module per cycle.
- **Layer 1–3 (when promoted):** byfuglien's chain takes the queue's
  pure-logic core and produces verified Python.

The whole thing fits in ~1500 lines of YAML + Python + agentic
markdown, none of which executes Claude in your prod path. The LLM
costs are bounded: the PR-Gate runs only on protected-surface PRs and
caches verdicts, the squad runs once a day, and the kill-criterion is
the parachute when the FP rate proves the LLM isn't earning its keep.

That is the assurance hierarchy as code, on a domain small enough to
audit by reading.
