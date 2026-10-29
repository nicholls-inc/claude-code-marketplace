# audit-lane automation

**Purpose.** A standing loop that finds problems on its own. A read-only auditor wakes on a schedule. If its inputs moved, it audits one bounded slice against an explicit baseline and files a few reproducible issues with evidence. A separate fix chain (implementer → review → merge → release) does the fixing. Auditors never change code.

This is [loop mode](../../docs/modes.md) running in the cloud (a scheduled workflow or a cloud routine). The audit is the instrument, the issues are the signal, and closed issues are the memory.

## Two variants

| | Parity audit | Log / performance audit |
|---|---|---|
| Signal | Your source vs a reference implementation, read with a sparse git clone | Aggregate-only queries over production logs or metrics |
| Baseline | Reference behavior: states, accessibility, responsive layout, loading and error | A trailing window (for example 72 h) vs the latest window (for example 6 h), per stage, outcome and role |
| Slice per run | One of ~10 rotating surfaces, plus any surface changed since the watermark | One window |
| Evidence | `file:line` on both sides | The query, the window, and baseline vs current numbers |
| Cadence | Hourly (`cron`), but only when either side's default-branch SHA moved | Every few hours |

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
| `CADENCE` | Hourly (parity), 6-hourly (logs) | Any `cron`; prefer an event when one exists |
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

## Runner

| Part | Kind | Runs as |
|---|---|---|
| Watermark check: read the `## lever state` block, compare the stored source and reference SHAs with the current default-branch SHAs, end silently if neither moved (parity variant) | Deterministic | Plain GitHub Actions job `gate` (`gh` + `jq`). No `claude-code-action`, no model, no API key. The logs variant has no SHA to compare: set the repo variable `LEVER_AUDIT_LANE_GATE=off`. |
| Pick the slice, audit against the baseline, keep reproducible findings, dedup, file at most `MAX_NEW_ISSUES`, rewrite state | LLM-driven | `RUNNER: gha` (job `audit`, only when `gate` says something moved) or `RUNNER: routine`. `automate.runner` in `~/.claude/lever.yml` can preset it; otherwise ask once. |

**`RUNNER: gha`**: install steps below. A silent hour costs one cheap job. Reading a private `REFERENCE_REPO` needs a token that can read it (the default `GITHUB_TOKEN` is scoped to this repository): use a read-only PAT secret for the `gate` call and the sparse clone.

**`RUNNER: routine`**: an hourly cloud routine does both parts; it reads the watermark first and ends `NOOP(no change)` quietly. Nothing is committed to `.github/workflows/`.
1. In Claude Code run `/schedule` (alias `/routines`) and create a routine named `lever-audit-lane`, attached to this repository (and the reference repository for the parity variant) with a GitHub connector that can read code and issues and file and comment on issues. Give it nothing that writes code.
2. Trigger: **schedule, hourly** (the minimum interval; use 6-hourly for the logs variant).
3. Prompt: `MODE: dry_run` / `TRIGGER: schedule` / `Follow .github/lever/automations/audit-lane/prompt.md exactly. Read the reference with a sparse git clone, not the REST API.`
4. Paused = create it disabled. Dry run = "run now" with `MODE: dry_run` on 3 slices; check every finding by hand against the evidence. Go live: set `MODE: act` and enable.
5. Least privilege: select only the repositories and connectors above. A routine has no `--allowedTools` list, so the CONTRACT (never change code, open PRs, close issues or touch credentials) is the control, plus a read-only identity.

## Triggers and permissions

| Trigger | Workflow config | Why |
|---|---|---|
| `schedule` | `cron: "29 * * * *"` (parity), `"29 */6 * * *"` (logs) | The loop. `schedule` is best-effort and can start late. |
| `workflow_dispatch` | input `mode` | Dry runs. |

| Job | `permissions:` | `--allowedTools` |
|---|---|---|
| `gate` | `contents: read`, `issues: read` | n/a: plain job, no model |
| `audit` | `contents: read`, `pull-requests: read`, `issues: write` (file issues, comment, the state block and existing audit issues only) | `Read`, `Grep`, `Glob`, `Bash(git clone --filter=blob:none --sparse:*)`, `Bash(git sparse-checkout:*)`, `Bash(git -C:*)`, `Bash(git log:*)`, `Bash(git diff:*)`, `Bash(gh issue list/view/create/comment/edit:*)`, `Bash(gh pr list:*)`, `Bash(gh pr view:*)`; logs variant: + the read-only aggregate query tool |

No code writes, no PRs, no closes.

## Install

1. Run [`automate-it`](../../skills/automate-it/SKILL.md). Copy [prompt.md](prompt.md) to `.github/lever/automations/audit-lane/prompt.md` and fill PARAMETERS with `MODE: dry_run`.
2. Create the pinned state issue with an empty `## lever state` section (a fenced `json` block holding `{}`) and set `STATE_ISSUE` in PARAMETERS and in the workflow's `env:`. Set `REFERENCE_REPO` and `REFERENCE_BRANCH` (parity). Copy [workflow.yml](workflow.yml) to `.github/workflows/lever-audit-lane.yml`, commit and push, then create it paused: `gh workflow disable lever-audit-lane.yml`.
3. Verify with `gh workflow view lever-audit-lane.yml` and read the YAML back: `contents: read` only, no code-write tool, no `gh workflow enable|disable`. Dry-run 3 slices: `gh workflow enable lever-audit-lane.yml` (the schedule stays skipped while `LEVER_AUDIT_LANE_LIVE` is unset), then `gh workflow run lever-audit-lane.yml -f mode=dry_run`. Check every finding by hand against the evidence.
4. Go live on approval with `MODE: act`: `gh variable set LEVER_AUDIT_LANE_LIVE --body true`. Make sure the fix chain picks up the audit label: [`keep-the-pool-full`](../../skills/keep-the-pool-full/), an implementer lane, or `autonomous-delivery`.
5. Pair it with an [improver](../improver/README.md): `GOOD_OUTCOME: issue fixed`, `BAD_OUTCOME: closed not planned`.
