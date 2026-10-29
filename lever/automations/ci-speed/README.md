# ci-speed automation

**Purpose.** Keep CI fast and trustworthy continuously, not in a quarterly cleanup sprint. CI gets worse every week: the code grows, tuning such as compile locks and page-file size drifts, and each bottleneck you fix exposes the next one. This recipe turns [`experiment-arms`](../../skills/experiment-arms/SKILL.md) into a standing lane.

The recipe has two jobs in one workflow, `lever-ci-speed.yml`:

| Job | Trigger | Tier | What it does |
|---|---|---|---|
| `ci-speed-watch` (`watch`) | `schedule` daily (off-peak) | Standard (`sonnet`) | Measures time-to-green p50/p90, first-try pass rate and the critical path. When a metric misses its target, it opens or updates **one** goal issue. Otherwise it stays silent. |
| `ci-speed-lane` (`lane`) | `issues: [labeled]` with job `if:` on `github.event.label.name == 'ci-goal'` | Premium (`opus`; judging paired-run evidence is the whole job, and the watch, the label, one goal at a time and the daily limit gate it) | Owns one goal at a time. Runs paired arms on real CI within hard caps, hands a stacked PR set with evidence to the owners, measures again after merge, then closes the goal or files the next bottleneck. |

The cheap watch decides when the expensive lane wakes. **Triggers decide cost.**

The watch's label is applied by the Claude GitHub App token, which does start workflow runs (the default `GITHUB_TOKEN` would not). So the lane job is the one place that does not carry the usual `claude[bot]` actor guard, and its step sets `allowed_bots: "claude"`. Turn that on there and nowhere else. Three things keep the chain from looping: the lane dedupes on the goal (one open goal at a time; a second label event on an owned goal is `NOOP(owned)`), a daily run limit (`LANE_RUNS_PER_DAY`, start at 2, checked in the prompt with `gh run list`), and a `concurrency:` group that serializes lane runs. If you would rather not allow bot-caused events at all, have the watch only open the issue and apply the label yourself, or dispatch the lane with `gh workflow run lever-ci-speed.yml -f role=lane -f resource_url=<goal>`.

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
| `watch` | `contents: read`, `actions: read`, `pull-requests: read`, `issues: write` | `Read`, `Grep`, `Glob`, `python3 .../ci_arm_metrics.py`, `gh run list/view`, `gh pr list/view`, `gh issue list/view/create/comment/edit` (labels and body) |
| `lane` | `contents: write` (pushes `ci-arm/*` branches only), `pull-requests: write`, `issues: write`, `actions: write` (rerun failed arm jobs) | Watch tools + `Write`, `Edit`, the `experiment-arms` scripts, `git checkout/add/commit`, `git push origin ci-arm/*`, `gh pr create/edit/ready/close/comment`, `gh run rerun`, `gh issue close` |
| Standing grant only | | + `Bash(gh pr merge:*)` |

There's no `gh run cancel`, so the lane never relies on cancelling a run; it pushes a new commit or waits. A lane run is bounded by `timeout-minutes: 360` (the hosted-runner maximum) and `--max-turns`; it writes progress to the goal issue, and `gh workflow run lever-ci-speed.yml -f role=lane -f resource_url=<goal>` resumes a goal whose lane run timed out.

The prompt follows [`experiment-arms`](../../skills/experiment-arms/SKILL.md): copy the lever plugin's `skills/experiment-arms/` to `.claude/skills/experiment-arms/` in the repository (the workflow's script paths assume it), or load the plugin in the action.

## Install

1. Run [`automate-it`](../../skills/automate-it/SKILL.md). Start with a manual baseline: give [`experiment-arms`](../../skills/experiment-arms/SKILL.md) to an agent and ask for time-to-green p50/p90 and the critical path. Set the targets from it.
2. Create the label `ci-goal`. Copy [prompt.md](prompt.md) to `.github/lever/automations/ci-speed/prompt.md` and fill PARAMETERS with `MODE: dry_run`.
3. Copy [workflow.yml](workflow.yml) to `.github/workflows/lever-ci-speed.yml`, commit and push, then create it paused: `gh workflow disable lever-ci-speed.yml`. Add `Bash(gh pr merge:*)` to the lane only with a standing grant.
4. Verify with `gh workflow view lever-ci-speed.yml` and read the committed YAML back: the watch has no PR or contents write; the lane has no `gh run cancel`, no merge and no `gh workflow enable|disable`.
5. Dry run: `gh workflow enable lever-ci-speed.yml` (events stay skipped while `LEVER_CI_SPEED_LIVE` is unset), then `gh workflow run lever-ci-speed.yml -f role=watch -f mode=dry_run` on the last 7 days and check its numbers against `skills/experiment-arms/scripts/ci_arm_metrics.py`. Dry-run the lane on one goal issue (`-f role=lane -f mode=dry_run -f resource_url=<goal>`), with arms capped at 1.
6. Go live on approval with `MODE: act` in PARAMETERS: `gh variable set LEVER_CI_SPEED_LIVE --body true`. Pair the lane with an [improver](../improver/README.md): `GOOD_OUTCOME: a kept arm held its gain after merge`, `BAD_OUTCOME: a kept arm regressed on real traffic or an owner reverted it`.
