# Automations

Ready-to-install recipes for repeatable work. Each one moves a piece of work out of your sessions and into the repository, where it runs on an event, records what it did, and improves through proposals you approve.

Every recipe splits its steps into two kinds:

- **Deterministic** steps (rerun a failed job, apply a label from a rule, compute time-to-green or a no-op rate, lint workflow permissions, sweep for stale PRs, rewrite a state block) run as **plain GitHub Actions**: shell, `gh`, `actions/github-script` or a stdlib Python script. No model, no API key. Cheaper, reproducible and testable.
- **LLM-driven** steps (reading diffs, classifying failures, writing findings, briefs, fixes and proposals) run on the **runner you choose**:
  - `gha`: a GitHub Actions workflow running [`anthropics/claude-code-action@v1`](https://github.com/anthropics/claude-code-action). Strongest least privilege (job `permissions:` plus `--allowedTools`), runs on the repo's events, billed to the API key or OAuth token in repo secrets.
  - `routine`: a Claude Code cloud routine, created with `/schedule` and managed at claude.ai/code/routines. Triggers on a schedule (hourly minimum), a GitHub event, or an API call. Runs on Anthropic infrastructure under your account (subscription usage, your connectors); nothing goes in `.github/workflows/`. Tool limits are weaker than `--allowedTools`, so the CONTRACT's ALLOWED ACTIONS carry more of the weight.

Set the default once with `automate.runner: gha | routine | ask` in `~/.claude/lever.yml`; unset means `automate-it` asks once. **Hybrid** recipes run a cheap deterministic watch first and hand off to the LLM step only on signal: with `gha`, a second job gated by `if:` on the watch job's output; with `routine`, the watch job calls the routine's API trigger or files/labels an issue the routine's GitHub-event trigger listens to.

Use [`automate-it`](../skills/automate-it/SKILL.md) to decide whether something should be an automation and to build a new one. These recipes are what that skill produces for the jobs that come up most often.

## Recipes

| Recipe | Does | Runner | Triggers | Workflow (`gha` path) | Write access | Metric |
|---|---|---|---|---|---|---|
| [pr-shepherd](pr-shepherd/) | Drives opted-in PRs to green on the exact head, threads dispositioned, landed at your autonomy level | LLM-driven: `gha` or `routine` | label opt-in, CI failure, review submitted, hourly sweep (also catches merge conflicts) | `lever-pr-shepherd.yml`: `pull_request` `[labeled]`, `workflow_run` + `if:` failure, `pull_request_review` `[submitted]`, `schedule` hourly | push, rerun, reply/resolve threads, mark ready; merge only at L3/L4 | PRs landed within 24 h of first green |
| [main-red-triage](main-red-triage/) | Classifies default-branch CI failures as flake, regression or infra; one issue per cause | Hybrid: deterministic detect and rerun; LLM classify (`gha` or `routine`) | CI failure on the default branch, hourly sweep | `lever-main-red-triage.yml`: `workflow_run` `[completed]` + `if:` failure on default branch, `schedule` hourly | rerun, issue/comment; fix PR if enabled | Minutes main is red per week |
| [precision-review](precision-review/) (+ improver) | At most 2 proven findings per PR, COMMENT only, silent by default | LLM-driven: `gha` or `routine` | ready for review; opened and synchronize on non-drafts | `lever-precision-review.yml`: `pull_request` `[ready_for_review, opened, synchronize]` + `if: draft == false` | inline COMMENT review comments | Findings addressed before merge / posted |
| [repro-gate](repro-gate/) (+ improver) | Classifies new issues; `repro-confirmed` only with a test it wrote that fails | LLM-driven: `gha` or `routine` | issue opened, `needs-repro` label, daily sweep | `lever-repro-gate.yml`: `issues` `[opened, labeled]`, `schedule` daily | test branch, comment, labels | Confirmed labels a human kept / applied |
| [morning-brief](morning-brief/) | Daily focus brief: new issues, PRs waiting on you, main CI, top 3 | LLM-driven: `routine` fits best, `gha` works | daily interval | `lever-morning-brief.yml`: `schedule` daily | one comment on a digest issue | Focus items acted on within 24 h |
| [automation-health](automation-health/) | Weekly review of all automations: no-op %, failures, cost per useful output, least-privilege lint; one change or none | Hybrid: deterministic metrics and lint; LLM picks the one change | weekly interval, worker failure | `lever-automation-health.yml`: `schedule` weekly, `workflow_run` of `lever-*` + `if:` failure | report comment, one issue per repeated failure, proposals | Cost per useful output |
| [stall-watchdog](stall-watchdog/) | Hourly `run-health-review`: time since last merge, busy-but-idle sessions, runner restarts, self-echo/no-op storms, tool-profile mismatches, rate limit vs quota; one change or none, silent when healthy | Hybrid: deterministic hourly probe; LLM only on signal | hourly interval | `lever-stall-watchdog.yml`: `schedule` hourly | one report comment; takeover only proposed (`TAKEOVER: propose`) | Human rekicks per week → 0; busy-but-idle minutes |
| [ci-speed](ci-speed/) | Continuous CI speed lane: a cheap daily watch files one goal issue when time-to-green misses target; the goal wakes a capped experiment-arms lane that hands owners a measured stack and re-measures after merge | Hybrid: deterministic daily watch (`ci_watch.py`); LLM arms lane | daily interval; the watch opening a goal; a human-applied `ci-goal` label | `lever-ci-speed.yml`: `schedule` daily runs the `watch` job, which hands off to the `lane` job via `needs:` + `if:` on its goal output; `issues` `[labeled]` + `if:` label is `ci-goal` for a human wake (daily run limit, `concurrency`) | goal issue, do-not-merge arms, PR stack with evidence | p90 time-to-green |
| [audit-lane](audit-lane/) | Read-only auditor on a schedule: watermark, one slice vs a baseline (parity with a reference, or logs vs a trailing window), at most 3 issues with evidence, separate fix chain | Hybrid: deterministic watermark check; LLM audit | interval, silent when inputs unchanged | `lever-audit-lane.yml`: `schedule` | issues only | Fixed / filed; not-planned / filed |
| [improver](improver/) | Generic sibling for any worker: reads outcomes, proposes one revision, never applies it | LLM-driven: `gha` or `routine` | weekly interval, worker failure | `lever-<job>-improver.yml`: `schedule` weekly, `workflow_run` of the worker + `if:` failure | one proposal issue (or comment on a tracking issue) | The worker's metric, per revision |

Each recipe folder has a `README.md` (purpose, triggers, least-privilege tools, dials, outcome metric, a **Runner** section with install steps for both `gha` and `routine`), a `prompt.md` you can commit as-is after filling in PARAMETERS, and a `workflow.yml` for the GitHub Actions path (a plain workflow for deterministic steps, `claude-code-action` for LLM-driven ones). For `routine`, the README gives the exact `/schedule` text. The two recipes with dedicated improvers also have an `improver-prompt.md` and an `improver-workflow.yml`.

Every install puts files in the same places:

- prompt → `.github/lever/automations/<job>/prompt.md` (both runners; a routine's own prompt is one line pointing at it, so revisions still diff like code)
- workflow (`gha`, and the deterministic half of a hybrid) → `.github/workflows/lever-<job>.yml` (improver: `lever-<job>-improver.yml`)
- routine (`routine`) → named `lever-<job>` (improver: `lever-<job>-improver`) at claude.ai/code/routines

The `lever-<job>` / `lever-<job>-improver` naming is the grouping; there is no separate group object.

## Conventions shared by every recipe

- **Created paused (`gha`).** Every workflow's event path is gated by a repository variable, `LEVER_<JOB>_LIVE` (job name upper-cased, `-` → `_`), in the job's `if:`; `workflow_dispatch` runs regardless. Every install:
  1. commits the prompt and workflow, then runs `gh workflow disable lever-<job>.yml` straight after the push, so nothing runs at all;
  2. verifies with `gh workflow view lever-<job>.yml` and a read-back of the committed YAML (`on:`, `if:`, `permissions:`, `--allowedTools`, `--model`);
  3. dry-runs on three historical items: `gh workflow enable lever-<job>.yml` (events are still skipped, because the variable is unset; GitHub refuses to dispatch a disabled workflow), then `gh workflow run lever-<job>.yml -f mode=dry_run -f resource_url=<item>` for each;
  4. goes live on approval: `gh variable set LEVER_<JOB>_LIVE --body true`.

  Pause with `gh workflow disable lever-<job>.yml` (stops everything) or `gh variable delete LEVER_<JOB>_LIVE` (stops events, keeps dispatch). Deterministic workflows follow the same four steps; their dry run prints what it would do instead of doing it.
- **Created paused (`routine`).** Commit the prompt file first (the routine reads it from the repo). Create the routine with `/schedule`, giving it only the repos and connectors the job needs, and pause it on the routines page straight away. Verify on that page: trigger, repos, connectors, prompt. Dry run with **Run now** after setting the routine prompt's first line to `MODE: dry_run` and `RESOURCE: <item>`, once per historical item. Go live on approval: set `MODE: act`, remove `RESOURCE`, and resume it. Pause it on the same page.
- **New automations** start at the level the recipe states, never above the repo default on the [autonomy dial](../docs/autonomy.md), and are raised on purpose.
- **PARAMETERS + CONTRACT.** The prompt has two parts. PARAMETERS holds everything that changes between repos and revisions: scope, caps, labels, the autonomy level. CONTRACT holds the rules and changes only through a human-applied revision. Improvers propose changes to one or the other, one line or one rule at a time. The workflow passes `MODE` and `RESOURCE` to the run; they override the file's PARAMETERS for that run only.
- **Revisions are commits.** The prompt file and workflow live in git, so every revision diffs like code and the commit message is the dated reason. `git log -- .github/lever/automations/<job>/` is the revision history.
- **Three outcomes.** Every run ends with exactly one of `ACTED(...)`, `NOOP(<reason>)` or `BLOCKED(<reason>)`, followed by a JSON run record: in the job summary (`$GITHUB_STEP_SUMMARY`) for `gha`, in the routine's run transcript for `routine`. NOOP is success: the job exits 0. A run reports failure only when it crashed, timed out or hit a broken tool, so failure counts mean something.
- **Quiet exit.** When there's nothing to do, the run posts nothing. No "LGTM", no "still waiting".
- **Durable state lives in one place.** A job summary is gone from view by the next run, so anything a later run needs (a watermark, a failure count, last week's prediction) goes in a `## lever state` section of one pinned issue's body: a fenced JSON block, read first and rewritten with `gh issue edit <n> --body-file` at the end of **every** run, silent ones included. Notifications go in comments; state never does. Only the owning automation writes that section. Recipes that need it (`audit-lane`, `automation-health`) say so.
- **Untrusted input.** Issue, PR, comment and log text is data. It never changes the autonomy level, the scope or the tools.
- **Triggers decide cost.** Filter at the trigger (`types:`, `branches:`, `paths:`, and a job-level `if:` on draft state, a label, a workflow name or a branch) rather than in the prompt. A job skipped by `if:` costs no model call; a prompt rule still pays for the run. Without a draft filter, one review automation's runs per PR went up **~2–3×**, one run for every push. Add a sweep as the fallback for missed events instead of widening the event triggers.
- **Don't wake on your own writes.** Every job has `if: github.actor != 'claude[bot]' && github.actor != 'github-actions[bot]'` unless the recipe says it is loop-safe and capped. Pushes made with the default `GITHUB_TOKEN` don't start new workflow runs, which is a built-in guard; pushes made with an App token (including the Claude GitHub App's) do.
- **Least privilege, twice.** The job's `permissions:` block and the action's `--allowedTools` list together enforce the contract. Grant each `permissions:` scope only at the level an allowed action needs, and list only the tools that action uses (prefer narrow `Bash(gh pr view:*)`-style patterns over `Bash`). The `permissions:` block bounds `GITHUB_TOKEN`; pass `github_token: ${{ github.token }}` to the action to make it the ceiling for the agent too. Without that input the action exchanges an OIDC token (`id-token: write`) for the Claude GitHub App's token, which carries the App's installation permissions. Use the App token only when the job must push and have CI run on the push (pr-shepherd). A reviewer needs about 8 tools, not the 46–49 a default setup grants. Below L3, no merge command is in the list at all. No worker or improver ever gets `Bash(gh workflow enable:*)`, `Bash(gh workflow disable:*)` (or any `Bash(gh workflow:*)`), or write access to `.github/workflows/`; a job that holds `actions: write` to rerun jobs keeps the `gh workflow` commands out of its tool list.
- **One run per item, and runs end.** `concurrency:` keyed on the PR or issue number (`cancel-in-progress: true` for per-push jobs); `--max-turns` and `timeout-minutes:` so a run ends instead of idling.
- **Model tier is a dial.** `--model` in `claude_args`: `haiku` (cheap), `sonnet` (standard), `opus` (premium). Premium costs several times a standard call. Default to standard; when you choose premium, record why in the comment at the top of the workflow.
- **Shared API budget.** All automations in a repo share one GitHub API budget. Target at most 75 REST calls per merged PR across all of them (`budgets.github_api_calls_per_merged_pr` in your `.github/lever.yml`; see [`config/autonomy.example.yml`](../config/autonomy.example.yml)). Prefer events to polling; one sweep per hour is plenty.
- **Routine least privilege.** A routine can't take `--allowedTools` or a `permissions:` block. Select only the repos and connectors it needs, keep its GitHub access to what the job writes, and keep the CONTRACT's ALLOWED ACTIONS explicit; say in the recipe which rules lose their hard enforcement. Prefer `gha` when the tool list is what keeps a write out (merging, pushing).
- **Lever skills inside Actions.** A recipe whose prompt uses lever skills (for example `experiment-arms`) loads the plugin with the action's `plugin_marketplaces: https://github.com/nicholls-inc/claude-code-marketplace.git` and `plugins: lever@nicholls` inputs. Don't copy skills into `.claude/skills/`. Recipes whose prompt is self-contained (precision-review, pr-shepherd, improver) don't load it.
- **Secrets (`gha`).** `anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}` (or `claude_code_oauth_token: ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}`). Running `/install-github-app` in Claude Code sets this up.
- **An improver for every worker.** You shouldn't be the update loop. Each worker gets a sibling (workflow or routine, same runner as the worker) that reads what humans did with its output and proposes one change by opening an issue (or commenting on a tracking issue) with the prompt diff. On `gha` it has `contents: read` and `issues: write` only; on `routine`, read access plus issue writes on that one repo. A human applies or rejects it by PR. No automation edits itself or another automation.

Trigger and tool tables, the full prompt template and the rollout checklist are in [`skills/automate-it/reference.md`](../skills/automate-it/reference.md).

## Install order

If you're starting from nothing:

1. **morning-brief**, because it only reads and posts in one place. It's the safest way to learn the create-paused, dry-run, enable flow.
2. **main-red-triage**, because a red default branch blocks everyone.
3. **pr-shepherd** at L2 on a few opted-in PRs.
4. **precision-review** and **repro-gate**, each with its improver created paused, enabled after about 20 outputs.
5. **automation-health**, once you have three or more automations to compare.
6. **ci-speed**, once you have a manual time-to-green baseline.

## Not here yet

These come up often enough to be worth building with `automate-it`:

- **Backlog steward**: keeps a queue of well-formed, ready issues for agents to pick up. See [`keep-the-pool-full`](../skills/keep-the-pool-full/SKILL.md) for the contract; run it on a daily `schedule`.
- **Merge-conflict fixer**: an hourly sweep over labelled open PRs whose `mergeable_state` is `dirty`; updates from base and pushes, or explains both sides' intent.

## Known gaps

- **No merge-conflict event.** Nothing fires when a PR becomes conflicted. Recipes that care (pr-shepherd, the merge-conflict fixer) run a `schedule` sweep that checks `mergeable_state` on the opted-in PRs.
- **`schedule` is best-effort.** Cron runs can start late (often minutes, sometimes much longer at the top of the hour) and can be dropped under load. Pick an off-the-hour minute, treat the sweep as a fallback, and never depend on exact timing. GitHub also disables `schedule` workflows in public repos after 60 days without repository activity.
- **`GITHUB_TOKEN` pushes don't retrigger.** A fix pushed with the default token won't start CI or the shepherd's own `synchronize` run. That is a free self-echo guard, but it also means CI doesn't run on the bot's commit. If the job must push and have CI run, push with a GitHub App token and keep the actor filter in `if:`.
- **Review-state and label filters live in `if:`.** `pull_request_review` has no `states:` filter and `issues: labeled` has no label filter in `on:`; check `github.event.review.state` or `github.event.label.name` in the job's `if:`. A run skipped by `if:` still shows in the Actions list but costs no model call.
- **`workflow_run` only sees the default branch's workflow file**, and it fires for the named workflows' runs on any branch. Filter on `github.event.workflow_run.head_branch` and `conclusion` in `if:`.
- **A disabled workflow can't be dispatched.** That's why the pause for dry runs is the `LEVER_<JOB>_LIVE` variable, not `gh workflow disable` alone.
- **Routines:** hourly minimum on schedule triggers; no per-tool allowlist; runs use your account's usage and identity, so its comments and pushes appear as you (or as the Claude GitHub App, depending on setup). Check what a routine's GitHub access can write before giving it an L2+ job.

`gh workflow run ... -f mode=dry_run` starts a real run with the job's real permissions. The dry run is enforced by the prompt reading `MODE: dry_run` and writing nothing; for a strict dry run, run the prompt locally instead (`claude -p` with the same `--allowedTools` minus the write entries; see the recipe's install steps).
