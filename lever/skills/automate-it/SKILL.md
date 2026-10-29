---
name: automate-it
user-invocable: false
description: Turns a task you keep doing by hand into a paused automation, as a plain GitHub Actions workflow for deterministic jobs, or for LLM-driven jobs a GitHub Actions workflow running anthropics/claude-code-action or a Claude Code cloud routine (/schedule), with an event trigger, least-privilege permissions and tools, a PARAMETERS + CONTRACT prompt, a quiet exit and a propose-only improver sibling. It creates automations, so it runs only on `/lever automate` or an explicit ask ("automate this", "run this on every PR/issue", "do this nightly", "make this run in the cloud"). When the same task has been done 3 or more times by hand, or a local `/loop` has fired 5 or more times for the same job, suggest it in one line instead of running it.
---

# automate-it

Repeatable work belongs in the cloud, on an event, with nobody babysitting it. This skill moves a task you already know how to do (a prompt you keep retyping, a local template, a `/loop` that polls) into an automation that lives with the repository. Work with no judgement in it becomes a plain GitHub Actions workflow with no model at all. Work that needs judgement runs on the runner the user picks: a workflow running `anthropics/claude-code-action@v1` (`gha`) or a Claude Code cloud routine created with `/schedule` (`routine`). Either way the automation starts **paused**, holds only the permissions and tools its allowed actions need, ends every run with an explicit outcome, and gets a sibling that proposes improvements for a human to apply. You stop being the motor and the update loop; you become the approver.

## When to use

- The same task shape appears 3 or more times in your session history.
- A local `/loop` has fired 5 or more times for the same job (polling that an event could replace).
- The user asks to automate, schedule, or "run this on every X".

## When not to use

- One-off tasks, or tasks with no checkable outcome. Run `find-the-loop` first if you can't name the metric.
- Tasks that need a human judgment call on every run. Automate the evidence-gathering and leave the call to the human.
- You have never done the task by hand. Do it manually at least once; the corrections you make become the contract.

## Steps

1. **Name the outcome and the metric.** One sentence for "done" (a comment posted, a label applied, a PR landed, a digest read). A metric with a denominator: `useful outputs / eligible events`, for example "opted-in PRs merged within 24 h of first green / opted-in PRs". A duration is not a finish condition. Refuse to continue without a denominator.
2. **Mine prior runs.** Find the last 5–20 times the task was done by hand or by a local template (`~/.claude/projects/<encoded-cwd>/*.jsonl`, `~/.claude/history.jsonl`; the format is internal and may drift). Extract the decision rules, the actions actually taken, and **every correction the user made**. Each correction becomes a numbered contract rule. Example: "returning failures is lame, why can't it just return nothing to do" became the rule `NOOP is success`.
3. **Classify the job, step by step.** Mark each step **deterministic** (no judgement: rerun a failed job, apply a label from a rule, compute time-to-green, a no-op rate or API-call counts, lint workflow permissions, sweep for stale PRs, rewrite a state block) or **LLM-driven** (reading diffs, classifying failures, writing findings, briefs, fixes, proposals). Use the "Kind" column in [reference.md](${CLAUDE_SKILL_DIR}/reference.md#trigger-decision-table).
   - All deterministic → a plain GitHub Actions workflow: shell, `gh`, `actions/github-script` or a stdlib Python script. **No `claude-code-action`, no model, no API key.** Cheaper, reproducible, testable. Skip the model-tier dial.
   - Any LLM-driven step → pick the runner (step 4).
   - **Hybrid** (a deterministic watch decides whether an LLM run is needed) → the watch is a plain workflow job; it hands off only on signal. With `gha`: a second job, `needs:` the watch and gated by `if:` on its output (or a `workflow_call`). With `routine`: the watch job calls the routine's API trigger, or files/labels an issue the routine's GitHub-event trigger listens to. Cheap watch first, expensive model only on signal.
4. **Pick the runner for the LLM-driven part.** Read `automate.runner` from `~/.claude/lever.yml` (honour `$CLAUDE_CONFIG_DIR`): `gha`, `routine` or `ask` (the default when unset). On `ask`, ask once, with this trade-off, and record the answer as `RUNNER` in PARAMETERS:
   - `gha`: GitHub Actions + `anthropics/claude-code-action@v1`. Strongest least privilege (job `permissions:` plus `--allowedTools`), runs on every repo event, billed to the API key or OAuth token in repo secrets. Prefer it when the tool list is what keeps a write out (push, merge).
   - `routine`: a Claude Code cloud routine via `/schedule` (alias `/routines`), managed at claude.ai/code/routines. Triggers on a schedule (hourly minimum), GitHub events, or an API call. Runs on Anthropic infrastructure under the user's account (subscription usage, their connectors); nothing is committed to `.github/workflows/`. Tool limits are weaker than `--allowedTools`: say so, and keep the CONTRACT's allowed actions explicit.
5. **Pick the trigger** from the decision table in [reference.md](${CLAUDE_SKILL_DIR}/reference.md#trigger-decision-table). Prefer an event over polling. Add a `schedule` sweep as a fallback for missed events. Add a filter (`types:`, `branches:`, `paths:`, and a job-level `if:` such as `github.event.pull_request.draft == false` or a label check) on every PR and issue trigger. **Don't wake on your own writes.** Exclude events whose actor is the automation's own identity (`github.actor != 'claude[bot]' && github.actor != 'github-actions[bot]'`), and events for a head SHA the automation just pushed. Pushes with the default `GITHUB_TOKEN` don't start new workflow runs; pushes with an App token do. In one long-running coordinator, about 1 in 6 wakes was the agent reacting to its own push, PR or merge, and about 6 in 10 wakes did nothing. Filter at the trigger; a job skipped by `if:` costs no model call, a prompt rule still pays for the run. **Triggers decide cost**: an unfiltered per-push trigger can multiply runs per PR without adding signal.
6. **Derive the minimal permissions and tools from the allowed actions.** List the actions the contract allows, then map each one with the least-privilege table in [reference.md](${CLAUDE_SKILL_DIR}/reference.md#least-privilege-tool-table): a job `permissions:` scope and the `--allowedTools` entries (narrow `Bash(gh ...:*)` patterns; `mcp__github_inline_comment__create_inline_comment` for inline review comments). Read tools are always fine; every write needs a matching allowed action. A comment-only reviewer needs about 8 tools, not 49. Pass `github_token: ${{ github.token }}` to the action so `permissions:` is the ceiling, unless the job must push and have CI run on the push. Never grant `Bash(gh workflow enable:*)`, `Bash(gh workflow disable:*)` or write access to `.github/workflows/` to a worker or an improver. On `routine`, the equivalent is selecting only the repos and connectors the job needs.
7. **Write the prompt** as a PARAMETERS block plus a stable CONTRACT using the [template](${CLAUDE_SKILL_DIR}/reference.md#prompt-template), saved at `.github/lever/automations/<job>/prompt.md` for both runners so revisions diff like code. PARAMETERS is the only section that varies between repos and revisions. The CONTRACT ends every run with exactly one outcome — `ACTED(summary)`, `NOOP(reason)` or `BLOCKED(reason)` — and writes one record per run. Deterministic jobs get no prompt; their script still prints one outcome line and the record.
8. **Define the quiet exit.** State exactly when the run posts nothing: draft PR, stale head SHA, duplicate of an earlier run, nothing above the confidence floor, no change since the last digest. Silence is the default; a no-op is a successful run (the job exits 0), not a failure. After writing the run record, **end the run**: set `--max-turns` and `timeout-minutes:`. Don't wait for a timeout. Hosted runs that did 1–2 minutes of work routinely held a runner for 15–20 minutes.
9. **Preflight (read-only).**
   - `gh auth status` shows a login with the `workflow` scope (`gh auth refresh -s workflow` if not); pushing a file under `.github/workflows/` needs it.
   - `gha`: `gh secret list` shows `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN`. If neither, stop and tell the user to run `/install-github-app` (or add the secret). Don't pretend you created anything. Deterministic jobs need no secret.
   - `gh workflow list` and `ls .github/workflows/`: no existing workflow with the same name or the same job on the same trigger. `routine`: the routines page has no routine with the same name.
   - `.github/lever.yml` caps (autonomy, budgets) and repo rulesets.
10. **Create paused.**
    - `gha` and deterministic: the job's event path is gated by `vars.LEVER_<JOB>_LIVE == 'true'` in `if:`; `workflow_dispatch` (inputs `mode: dry_run|act`, `resource_url`) always runs. Commit the workflow at `.github/workflows/lever-<job>.yml` and the prompt, push, then `gh workflow disable lever-<job>.yml` straight away.
    - `routine`: commit and push the prompt, then `/schedule` the routine with a one-line prompt ("Follow .github/lever/automations/<job>/prompt.md exactly", plus `MODE` and `RESOURCE` lines), only the repos and connectors it needs, and create it paused (or pause it on the routines page at once).
    Run on public repos only for actors with write access: `author_association` in `[OWNER, MEMBER, COLLABORATOR]` in `if:`.
11. **Verify.** `gha`: `gh workflow view lever-<job>.yml` shows it disabled, and the committed YAML must read back the exact `on:`, `if:`, `permissions:`, `--allowedTools`, `--model` and `concurrency:` of your spec. Diff it against the spec. `routine`: the routines page shows it paused, with the exact trigger, repos, connectors and prompt.
12. **Dry-run on 3 historical items.** Pick one item that should ACT, one that should NOOP and one edge case. Show the user the three records.
    - `gha`: `gh workflow enable lever-<job>.yml` (events stay skipped while the variable is unset; GitHub won't dispatch a disabled workflow), then `gh workflow run lever-<job>.yml -f mode=dry_run -f resource_url=<item>` for each. This is a real run with real permissions; the prompt writing nothing in `dry_run` is what keeps it dry.
    - Parity option (strictest): run the same prompt locally, `claude -p "MODE: dry_run\nRESOURCE: <item>\nFollow .github/lever/automations/<job>/prompt.md exactly." --allowedTools "<the same list minus the write entries>"`. Manual and automated runs should be the same thing.
    - `routine`: set the routine prompt's `MODE: dry_run` and `RESOURCE: <item>`, then **Run now**, once per item.
13. **Enable on approval.** Ask once, with the dry-run records attached. Then `gh variable set LEVER_<JOB>_LIVE --body true` (`gha`), or restore `MODE: act` and resume the routine. Don't ask again for permission you already have (see [autonomy](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md)).
14. **Pair with an improver sibling.** Create it from [automations/improver](${CLAUDE_PLUGIN_ROOT}/automations/improver/README.md), on the same runner, also paused: `lever-<job>-improver`, weekly `schedule` plus `workflow_run` of the worker with `if: conclusion == 'failure'`, `contents: read` and `issues: write` only. It proposes by opening an issue with the prompt diff; a human applies it by PR. The `lever-<job>` / `lever-<job>-improver` names are the grouping. Deterministic jobs get one only if they have thresholds worth tuning.
15. **Qualify before you trust it.** Treat the automation as migrated only once it has produced real outputs *and* kept picking up work from events, with no local prompting, over an agreed window (for example a week). Stop any local `/loop` doing the same work so the two never compete. One good run isn't a migration.
16. **Hand off.** Report the kind (deterministic, LLM-driven, hybrid), the runner, the workflow file or routine name, the revision (commit hash), model tier and why, how to pause it (`gh workflow disable lever-<job>.yml`, or pause the routine), where its records live (job summaries, the routine's transcripts, or the ledger issue), and the metric's baseline.

## Dials

| Dial | Default | Options |
|---|---|---|
| Runner (`automate.runner`, `RUNNER`) | `ask` | `gha` (claude-code-action), `routine` (`/schedule`); deterministic steps are always plain GitHub Actions |
| Trigger breadth | One event trigger + filter + daily or hourly `schedule` sweep | Event only; sweep only; more event types (each adds runs) |
| Trigger filter | `if: github.event.pull_request.draft == false` on PR triggers; label or branch filter where possible | None (costly), `paths:`, label opt-in |
| Wake filter | Exclude self-actor (`claude[bot]`, `github-actions[bot]`) and self-pushed SHAs | Include (only for loop-safe, capped jobs) |
| Model tier | `sonnet` (standard) | `haiku` (cheap), `opus` (premium; several times standard per call). Record why in the workflow's top comment; let the improver re-test the choice |
| Token | `github_token: ${{ github.token }}` (job `permissions:` is the ceiling) | Claude GitHub App token via `id-token: write`, when pushes must start CI |
| Autonomy level (`AUTONOMY`) | The level the recipe states, never above the repo default; L0 if the recipe doesn't say | Up to L5 per [docs/autonomy.md](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md), capped by repo rules; the tool list enforces it |
| Writes per run | 1 | Integer; plus open-automation-PR cap (5) |
| Evidence budget | 3 items assessed, 10 logs read, 20 records scanned | Integers |
| API budget | 75 REST calls per useful output, shared across automations | Integer; see `run-health-review` |
| Bot-triggered runs | Excluded in `if:` | Included with a `concurrency:` cap and `--max-turns` (it can trigger itself) |
| Rollout | paused → dry_run → act on one repo | → more repos → org |
| Improver authority | Propose-only | There is no "apply" setting; a human applies |

The permissions and tool list are the enforcement; PARAMETERS is the intent. Below L3 no merge command is in the list at all.

## Outputs

- A paused workflow (`.github/workflows/lever-<job>.yml`) or routine (`lever-<job>`) whose definition matches the spec; for hybrids, both halves.
- A prompt file with PARAMETERS + CONTRACT at `.github/lever/automations/<job>/prompt.md`, so revisions diff like code. Start from a [recipe](${CLAUDE_PLUGIN_ROOT}/automations/README.md) when one fits.
- Three dry-run records and the metric baseline.
- A paused improver sibling.

## This skill's own feedback loop

- **Metric:** cost per useful output = (runs × model tier cost) / useful outputs, tracked weekly with the post rate and NOOP mix. Deterministic runs count at runner minutes only.
- **Active ratio:** tool-active minutes / wall minutes per run. Below 0.3 means runs idle out instead of ending. **Self-echo rate:** wakes caused by the automation's own writes / all wakes; the target is 0.
- **Check at 7 days:** runs/day vs eligible events (are triggers too broad?), skipped-by-`if:` vs run, BLOCKED reasons, failures that were really no-ops, tools never used (remove them), LLM steps that turned out deterministic (move them to a plain step).
- Every change is a new commit with one reason. "No change" is a valid weekly outcome.

## Failure modes it prevents

- Review-only bots holding push, issue-close and PR-create rights.
- No-op runs reported as failures, waking the improver on noise.
- Per-push triggers without a draft filter multiplying cost.
- A known prompt bug running for days because a human is the only update loop.
- Silent automations: running all day and producing no measurable data point.
- Paying a model to rerun a job or count runs that a ten-line script does the same way every time.
- Editing the tool list and accidentally dropping an entry: the read-back in step 11 diffs the whole list.

## Why this exists

- Code review ran as a local template for about 3.3k turns across 6 repos before it became 2 cloud automations, which were then graded about 230 times. Automate what you have already done by hand.
- Those review-only automations held 46–49 tools. The allowed actions (read, comment, review) need about 8.
- "I was the update loop": the review automations took 7 hand revisions in two months, and an issue-repro automation ran five more days with a known prompt bug before a human edited it.
- "I didn't add more agents, I added more pushes": runs per PR went up ~2–3× because a per-push trigger had no draft filter.
- One user typed the same morning-brief prompt by hand 28 days in a row. "Monitor CI and push fixes until green" was hand-typed 109 times. Both are one trigger each.
