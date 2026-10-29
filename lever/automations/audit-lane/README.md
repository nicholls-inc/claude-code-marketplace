# audit-lane automation

**Purpose.** A standing loop that finds problems on its own. A read-only auditor wakes on a schedule. If its inputs moved, it audits one bounded slice against an explicit baseline and files a few reproducible issues with evidence. A separate fix chain (implementer → review → merge → release) does the fixing. Auditors never change code.

This is [loop mode](../../docs/modes.md) running in the cloud. The audit is the instrument, the issues are the signal, and closed issues are the memory.

## Two variants

| | Parity audit | Log / performance audit |
|---|---|---|
| Signal | Your source vs a reference implementation, read with a sparse git clone | Aggregate-only queries over production logs or metrics |
| Baseline | Reference behavior: states, accessibility, responsive layout, loading and error | A trailing window (for example 72 h) vs the latest window (for example 6 h), per stage, outcome and role |
| Slice per run | One of ~10 rotating surfaces, plus any surface changed since the watermark | One window |
| Evidence | `file:line` on both sides | The query, the window, and baseline vs current numbers |
| Cadence | Hourly, but only when either side's default-branch SHA moved | Every few hours |

## What's possible

In one run of each variant over a few weeks:
- **Parity audit:** about 175 issues filed and about 97% fixed, most the same day. Under 1% were false positives.
- **Log audit:** about 45 issues filed, with roughly half fixed. About 1 in 5 was noise.

The difference is the signal. A source diff with `file:line` on both sides is close to deterministic. Latency on runs that succeeded anyway is noisy. Start with the sharpest signal you have, and expect to tune a metrics auditor before it's trusted.

## Contract highlights

- **Watermark first.** The watermark and rotation live in a `## lever state` section of one pinned issue's body (see [durable state](../README.md#conventions-shared-by-every-recipe)). Every run reads it first and rewrites it last. If no input moved, the run updates state and stays silent.
- **Cap the output:** at most 2–3 new issues per run (and a few comments). The fix chain's capacity is the limit, not the auditor's.
- **Dedup:** search open issues and PRs first. Update an existing issue instead of filing a new one. Skip a finding when an open PR already touches those files.
- **Read-only:** use a read-only, narrowly scoped identity. Never paste secrets or raw log lines into issues; use aggregates and redacted examples.
- **False positives are memory:** a maintainer closes them as *not planned*, and the auditor never refiles a key that was closed that way.

## Dials

| Dial | Default | Options |
|---|---|---|
| `CADENCE` | Hourly (parity), 6-hourly (logs) | Any; prefer an event when one exists |
| `MAX_NEW_ISSUES` | 3 | 1–5 |
| `MAX_COMMENTS` | 4 | 0–10 |
| `SURFACES` | ~10 rotating | Your list |
| `WINDOW` / `BASELINE` | 6 h / 72 h | Any |
| `NOISE_FLOOR` | Ignore keys with fewer than N samples, and success-only slowness unless it's large | Per metric |
| `AUTONOMY` | L1 for the auditor (issues only); the fix chain has its own level | — |
| [Spend](../../docs/spend.md) | Premium model, high effort: judgment is the whole job. About one session per non-silent run. | Lower the tier for a sharp, rule-based signal |

## Outcome metric

- **Primary:** issues fixed / issues filed. Track not-planned / filed as the false-positive rate.
- **Also:** median time from filed to fixed, and the share of runs that stay silent.
- **Health:** if the fix chain never picks up audit issues, check the label wiring first. That has broken before.

## What broke while tuning, so you can skip it

- Reading the reference over the REST API drained a shared rate-limit budget. Read it with git instead.
- Silent no-change runs had to be made explicit; early versions posted "nothing found".
- Duplicate filings stopped only after the open-PR overlap check was added.
- A label mismatch meant the fix chain never saw an audit's issues.
- The audit environment lacked a CLI it assumed, so sign-in needed a fallback. Preflight the tools.
- Rules against secrets and raw logs had to be written down, not assumed.

## Install via GitHub MCP

1. Run [`automate-it`](../../skills/automate-it/SKILL.md). Fill PARAMETERS in [prompt.md](prompt.md) with `MODE: dry_run`.
2. Create the pinned state issue with an empty `## lever state` section and set `STATE_ISSUE`. Create the automation paused, with an `interval` trigger (`write_scope: repository`, since there's no triggering resource) and read tools plus `github/create_issue`, `github/add_issue_comment` and `github/update_issue_body` (for the state section and existing audit issues only). Give it no write tools for code.
3. Dry-run 3 slices. Check every finding by hand against the evidence.
4. Enable on approval. Make sure the fix chain picks up the audit label: [`keep-the-pool-full`](../../skills/keep-the-pool-full/), an implementer lane, or `autonomous-delivery`.
5. Pair it with an [improver](../improver/README.md): `GOOD_OUTCOME: issue fixed`, `BAD_OUTCOME: closed not planned`.
