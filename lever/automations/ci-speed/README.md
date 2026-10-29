# ci-speed automation

**Purpose.** Keep CI fast and trustworthy continuously, not in a quarterly cleanup sprint. CI gets worse every week: the code grows, tuning such as compile locks and page-file size drifts, and each bottleneck you fix exposes the next one. This recipe turns [`experiment-arms`](../../skills/experiment-arms/SKILL.md) into a standing lane.

The recipe has two jobs in one workflow, `lever-ci-speed.yml`:

| Job | Trigger | Tier | What it does |
|---|---|---|---|
| `ci-speed-watch` (`watch`) | `schedule` daily (off-peak) | None: deterministic script, no model | Measures time-to-green p50/p90, first-try pass rate and the critical path with [`ci_watch.py`](ci_watch.py). When a metric misses its target, it opens or updates **one** goal issue. Otherwise it stays silent. |
| `ci-speed-lane` (`lane`) | The watch opened a goal (`needs:` hand-off), or `issues: [labeled]` with job `if:` on `github.event.label.name == 'ci-goal'` (a human), or dispatch | Premium (`opus`; judging paired-run evidence is the whole job, and the watch, the label, one goal at a time and the daily limit gate it) | Owns one goal at a time. Runs paired arms on real CI within hard caps, hands a stacked PR set with evidence to the owners, measures again after merge, then closes the goal or files the next bottleneck. |

The cheap watch decides when the expensive lane wakes. **Triggers decide cost.**

In the `gha` runner the hand-off is a `needs:` dependency inside `lever-ci-speed.yml`, not a label event: the watch creates its issue with the default token (which starts no workflow runs), and the lane job runs because the watch output a goal issue number. So no job needs a bot-actor exception. Three things keep the chain from looping: the lane dedupes on the goal (one open goal at a time; a second wake on an owned goal is `NOOP(owned)`), a daily run limit (`LANE_RUNS_PER_DAY`, start at 2, checked in the prompt with `gh run list`), and a `concurrency:` group that serializes lane runs. The script cannot read logs, so the slowest steps, top 3 flake signatures and hypothesis list on the goal issue are written by the lane.

## Runner

| Part | Kind | Runs as |
|---|---|---|
| `watch`: measure p50/p90, first-try pass, flake rate, critical path; compare with targets; open or update one goal issue | Deterministic | Plain GitHub Actions job running `ci_watch.py` (python3 stdlib + `gh`). No `claude-code-action`, no model, no API key. Always this, whichever runner you choose for the lane. |
| `lane`: hypotheses, paired arms, PR stack, re-measure | LLM-driven | Your choice, `RUNNER: gha` or `routine` (`automate.runner` in `~/.claude/lever.yml` can preset it; otherwise ask once). |

**`RUNNER: gha`** (default shipped): the `lane` job in [workflow.yml](workflow.yml) runs `anthropics/claude-code-action@v1`, with `permissions:` and `--allowedTools` as in the table below. Install steps are under Install.

**`RUNNER: routine`**: a Claude Code cloud routine runs the lane on Anthropic's infrastructure under your account; nothing but the watch is committed to `.github/workflows/`.
1. Delete the `lane` job from your copy of `workflow.yml`. Keep `watch`. Replace nothing else.
2. In Claude Code run `/schedule` (alias `/routines`) and create a routine named `lever-ci-speed-lane`, attached to this repository and only the GitHub connector it needs.
3. Trigger: **GitHub event, issue labeled `ci-goal`** (the lane then wakes on a human-applied label). To wake it from the watch instead, add an **API trigger** to the same routine and add a last step to the `watch` job that POSTs the new goal issue URL to the routine's API trigger URL with its token (store both as repository secrets; the routine's page shows the exact request). Check the routine's trigger options in the routines page for the current event list.
4. Prompt: `ROLE: lane` / `MODE: dry_run` / `RESOURCE: <goal issue URL from the trigger>` / `Follow .github/lever/automations/ci-speed/prompt.md exactly (ROLE: lane section).`
5. Paused = create it disabled. Dry run = "run now" with `MODE: dry_run` in the prompt, on one goal issue. Go live: set `MODE: act`, enable it.
6. Least privilege: choose only this repository and the connectors it needs. A routine has no `--allowedTools` list, so tool limits are weaker than the `gha` path; the prompt's CONTRACT (no merge, no `gh run cancel`, no workflow edits, arms capped) is the control. Give it no merge access without a standing grant.

## What a run looks like

1. **Measure what people feel:** time from push until every required check is green, at p50 and p90. Only the slowest required lane matters, so find the critical path.
2. **One change per PR,** small and revertible, labeled do-not-merge while it's being measured.
3. **Prove it on real CI** with paired runs on the same SHA, at least 6 per arm, with the same test counts and verdicts. Fewer is INCONCLUSIVE, and a failed treatment run counts against the arm. Keep a change only if it moves the critical path by a minute or more, or clearly cuts cost or flakes. Write down the arms that fail.
4. **Run arms in parallel, decide the order later.**
5. **Hand the stack to the owners** with the evidence in each description, landed at the resolved [autonomy level](../../docs/autonomy.md). Validate in parallel, merge in batches, and serialize only PRs that touch the same thing ([`landing-plan`](../../skills/landing-plan/SKILL.md)).
6. **Measure again on real traffic after merge,** and post the numbers.

## Dials

| Dial | Default | Options |
|---|---|---|
| `TARGET_P50_MIN` / `TARGET_P90_MIN` | Set from the baseline | Minutes |
| `TARGET_FIRST_TRY_PASS` | Set from the baseline | Percent |
| `KEEP_BAR` | Critical path −1 min, or a clear cost or flake cut | Your bar, written before measuring |
| `RUNS_PER_ARM` | 6 paired | 6–12; under 6 is INCONCLUSIVE |
| `MAX_CONCURRENT_ARMS` | 4 | Never above your spend tier's lane cap or `budgets.max_parallel_arms`. Keep it low enough not to starve teammates' runners |
| `MAX_OPEN_PRS` | 8 | Review capacity is the real limit; a stack that outruns its owners rots |
| `CI_MINUTES_PER_GOAL` | Set per repo | Hard stop; report Actions minutes spent per minute saved |
| `API_RATE_FLOOR` | Pause below 20% of the hourly quota | Agent polling can burn a full quota and break unrelated tools |
| `RUN_WINDOW` | Off-peak hours | Any |
| `LANE_RUNS_PER_DAY` | 2 | Daily run limit for the lane |
| `AUTONOMY` | L1: PRs for owners. A safe floor for a team new to agents. | Raise it as the team trusts the evidence, up to the repo ceiling; L4 with a standing grant for CI-config changes (which also needs a token allowed to push `.github/workflows/`; the default `GITHUB_TOKEN` can't) |

## Guardrails

These are fixed and are not dials:
- No speedups that weaken the signal: no skipped tests, blanket retries or lower coverage.
- Never widen `permissions:`, add `pull_request_target` or touch secrets. Keep code-owner review on `.github/workflows/`.
- Flakes become issues for the owning team with a repro, not retries.
- Close every measurement PR at the end of a goal.
- Re-baseline whenever the runner images change.

## Outcome metric

- **Primary:** p90 time-to-green on real PR traffic. The long tail is what makes CI feel unpredictable.
- **Also:** p50, first-try pass rate, flake rate, billed Actions minutes.
- **Cost:** Actions minutes spent per minute of p50 saved.
- **Guardrail:** test count and coverage never drop.

## What's possible

In one run of this method on two busy repos (100–250 PRs a day each), p50 time-to-green fell **43%** in one repo and **34%** in the other, in about a day and a half. The top flake went to zero after it was fixed in product code rather than retried. Some of the largest wins were not speed at all: cache scoping (restore on PRs, save only on the default branch), memory pressure on build runners, shallow fetches, and timeouts on hung jobs. Several "obvious" wins turned out to be noise once they were measured in pairs.

## Permissions and tools (least privilege)

| Job | `permissions:` | `--allowedTools` |
|---|---|---|
| `watch` | `contents: read`, `actions: read`, `pull-requests: read`, `issues: write` | n/a: a plain job, no model and no `--allowedTools`; the script calls `gh run list` and `gh issue list/create/comment` |
| `lane` | `contents: write` (pushes `ci-arm/*` branches only), `pull-requests: write`, `issues: write`, `actions: write` (rerun failed arm jobs) | Watch tools + `Write`, `Edit`, the `experiment-arms` scripts, `git checkout/add/commit`, `git push origin ci-arm/*`, `gh pr create/edit/ready/close/comment`, `gh run rerun`, `gh issue close` |
| Standing grant only | | + `Bash(gh pr merge:*)` |

There's no `gh run cancel`, so the lane never relies on cancelling a run; it pushes a new commit or waits. A lane run is bounded by `timeout-minutes: 360` (the hosted-runner maximum) and `--max-turns`; it writes progress to the goal issue, and `gh workflow run lever-ci-speed.yml -f role=lane -f resource_url=<goal>` resumes a goal whose lane run timed out.

The prompt follows [`experiment-arms`](../../skills/experiment-arms/SKILL.md): the workflow loads the lever plugin with the action's `plugin_marketplaces` and `plugins: lever@nicholls` inputs, and sparse-checks out the plugin's `skills/experiment-arms/` at `.lever-plugin/` so the scripts have a fixed path for `--allowedTools` (`ARMS_SCRIPTS` in the prompt). Don't copy skills into `.claude/skills/`. Add `.lever-plugin/` to `.gitignore` so the lane never commits it.

## Install

1. Run [`automate-it`](../../skills/automate-it/SKILL.md). Start with a manual baseline: give [`experiment-arms`](../../skills/experiment-arms/SKILL.md) to an agent and ask for time-to-green p50/p90 and the critical path. Set the targets from it.
2. Create the label `ci-goal`. Copy [prompt.md](prompt.md) to `.github/lever/automations/ci-speed/prompt.md` and fill PARAMETERS with `MODE: dry_run`.
3. Copy [workflow.yml](workflow.yml) to `.github/workflows/lever-ci-speed.yml` and [ci_watch.py](ci_watch.py) to `.github/lever/automations/ci-speed/ci_watch.py`; set the `REQUIRED_WORKFLOWS` and `TARGET_*` env values in the `watch` job from your baseline. Commit and push, then create it paused: `gh workflow disable lever-ci-speed.yml`. Add `Bash(gh pr merge:*)` to the lane only with a standing grant.
4. Verify with `gh workflow view lever-ci-speed.yml` and read the committed YAML back: the watch has no PR or contents write; the lane has no `gh run cancel`, no merge and no `gh workflow enable|disable`.
5. Dry run: `gh workflow enable lever-ci-speed.yml` (events stay skipped while `LEVER_CI_SPEED_LIVE` is unset), then `gh workflow run lever-ci-speed.yml -f role=watch -f mode=dry_run` on the last 7 days and check its numbers against a manual run of `skills/experiment-arms/scripts/ci_arm_metrics.py`. Dry-run the lane on one goal issue (`-f role=lane -f mode=dry_run -f resource_url=<goal>`), with arms capped at 1.
6. Go live on approval with `MODE: act` in PARAMETERS: `gh variable set LEVER_CI_SPEED_LIVE --body true`. Pair the lane with an [improver](../improver/README.md): `GOOD_OUTCOME: a kept arm held its gain after merge`, `BAD_OUTCOME: a kept arm regressed on real traffic or an owner reverted it`.
