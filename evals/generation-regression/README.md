# Generation regression eval

Companion to `evals/pr-review-regression/`: instead of asking whether crosscheck *detects* known bugs in review, this asks whether a crosscheck-disciplined agent *avoids writing them* when implementing the same task from scratch.

## Method

Each case is a **counterfactual re-implementation**: a task spec (`specs/spec_<pr>.md`) reconstructed from the original bug-introducing PR's title/body/files — describing what to build, never hinting at the bug — plus the parent commit (`specs/parents.tsv`) to check out. An agent implements the spec; an adversarial judge decides whether the generated diff contains the historical defect mechanism (BUG_PRESENT / BUG_AVOIDED).

Arms: `bare` (implement directly), `delib` (design-first, no verification machinery — isolates "we made it think first"), `cc` (invariant-first contract discipline), plus a real-plugin arm run via the Claude Agent SDK invoking the actual crosscheck skills.

Two protocols:
- **Agentic** (`gen_case.sh <pr> <arm> [run]`): headless `claude -p` in a worktree at the parent commit; edits files; diff captured. This is the protocol that matters.
- **One-shot batch** (`batch_submit.py` → `judge_submit.py`): context assembled deterministically, submitted via the Anthropic Batch API at 50% cost. ~10-20× cheaper; good for case mining, NOT a proxy for agentic behavior (see baselines).

## Baselines (2026-08-27, opus)

- `baselines/oneshot_batch_verdicts.json` — 7 cases × 3 arms × k=3 (63 runs): every case BUG_AVOIDED in one-shot mode **except #16635 (9/9 BUG_PRESENT)**. No arm separation in one-shot mode.
- `baselines/agentic_pilot_verdicts.json` — 2 cases × 2 arms agentic pilot: #10279 reproduced in BOTH arms agentically (though avoided 9/9 one-shot — protocol changes outcomes); on #9688 the bare arm introduced a cross-tenant data leak the cc arm guarded.
- `baselines/agentic_16635_verdicts.json` — gold case, agentic, k=3: bare 0/3 avoided, delib 0/3, cc 1/3 (the one avoidance subclassed aiokafka's `StickyPartitionAssignor` per consumer, citing its class-level state).
- `baselines/agentic_16635_plugin_verdicts.json` — real-plugin arm (Agent SDK, actual `crosscheck:draft-invariants` skill loaded, verified in transcripts): 1/3 avoided — same rate as the prompt-approximated cc arm; the avoidance run built a per-consumer `IsolatedStickyPartitionAssignor` subclass and asserted instance isolation in tests. Combined cc-style arms: 2/9 avoided vs 0/6 bare+delib.

## Key findings

1. **#16635 is a near-universal attractor**: 17/18 runs across protocols reproduced the DEVOPS-1551 production bug (stock `StickyPartitionAssignor` shared between two consumer groups). Use it as the primary generation regression case.
2. Protocol dominates discipline: one-shot generation with the right files in context avoids bugs that agentic runs commit (#10279), and washes out arm differences. Regression thresholds must be per-protocol.
3. Crosscheck-style discipline shows a weak preventive signal agentically (1/3 vs 0/6 on #16635; leak guard on #9688). n is too small for claims — treat as hypothesis, per the eval mandate's k-runs/variance requirements.

## Thresholds (agentic protocol, case #16635, k=3)

- bare arm: expected 0/3 avoided (baseline). A harness change is *interesting* if its arm beats 1/3.
- Judge criterion: BUG_PRESENT iff both consumer groups can share mutable assignor state (stock sticky class for both, no per-group isolation).

## Requirements

`claude` CLI (agentic), `ANTHROPIC_API_KEY` (batch + Agent SDK plugin arm), gh + local ev-energy/core clone, `pip install anthropic claude-agent-sdk`.
