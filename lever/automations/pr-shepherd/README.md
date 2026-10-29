# pr-shepherd automation

**Purpose.** Drive opted-in PRs to landed without anyone babysitting them: green on the exact head SHA, every review thread dispositioned, landed at the autonomy level you set. This is the cloud form of [skills/pr-shepherd](../../skills/pr-shepherd/SKILL.md); both use the same contract, so a manual run and an automated run behave the same.

It's the loop people hand-type as "monitor CI and push fixes until green, then merge" and the 20-minute self-wakes that poll for it.

Files: [prompt.md](prompt.md), [workflow.yml](workflow.yml), [shepherd.sh](shepherd.sh), [pr-state.graphql](pr-state.graphql).

## Triggers

| Event (`on:`) | Filter (job `if:`) | Why |
|---|---|---|
| `pull_request: types: [labeled]` | `github.event.label.name == 'shepherd'` | Opt-in. The first run reads state and acts. |
| `workflow_run: workflows: [<required CI workflow>], types: [completed]` | `conclusion == 'failure'`, `event == 'pull_request'` | Red CI is the main reason to act. The run maps the failed run's head branch to an opted-in PR or ends `NOOP(not opted in)`. The shepherd's own pushes are kept here on purpose (its fix can fail too); `MAX_FIX_PUSHES` caps that loop. |
| `pull_request_review: types: [submitted]` | label `shepherd`, `review.state` in `changes_requested`, `commented`; actor not `claude[bot]` or `github-actions[bot]` | New threads to disposition. The actor filter matters: the shepherd's own thread replies arrive as `commented` reviews. |
| `schedule: - cron: '23 * * * *'` | — | Sweep: lands PRs that went green (there is no "CI succeeded" trigger for a PR), picks up merge conflicts (Actions has no merge-conflict event, so the sweep checks mergeability), and catches missed events. |
| `workflow_dispatch` (`mode`, `resource_url`) | — | Dry runs and manual kicks. |

`pull_request: synchronize` is left out on purpose. A push only matters once CI reports, and `workflow_run` plus the sweep cover both outcomes. Adding it would fire a run on every push that ends `NOOP(ci pending)`. Triggers decide cost.

If your PRs go green slowly and you want landing sooner than hourly, add `synchronize` to the `pull_request` types with `contains(github.event.pull_request.labels.*.name, 'shepherd') && github.event.pull_request.draft == false` in `if:` and accept the extra no-op runs, or ask the PR author's session to keep a `/loop` on it.

If you change `SHEPHERD_LABEL`, change the two `'shepherd'` strings in the workflow's `if:` in the same commit. Change `workflows: [CI]` to your `REQUIRED_WORKFLOWS`.

## Tools (least privilege, by level)

Job `permissions:` `contents: write`, `pull-requests: write`, `issues: write`, `actions: write` (rerun), `checks: read`, `id-token: write`. The workflow uses the Claude GitHub App token (no `github_token` input) because a fix pushed with `GITHUB_TOKEN` would not start CI, and the shepherd needs CI on its own pushes. That makes the App installation's permissions, not this block, the agent's ceiling; the tool list does the rest.

Every GitHub write that `gh` has no narrow subcommand for goes through [shepherd.sh](shepherd.sh). The workflow copies it and [pr-state.graphql](pr-state.graphql) to a read-only directory under `$RUNNER_TEMP` before the agent starts and allowlists only that copy, so an `Edit` can't change what it runs. Each subcommand takes fixed, validated arguments:

| `shepherd.sh` subcommand | Replaces | What it refuses |
|---|---|---|
| `state <pr>` | `gh api graphql` with the batched read | Anything but the fixed `pr-state.graphql` query |
| `resolve-thread <thread id>` | `gh api graphql` (`resolveReviewThread`) | Threads outside this repo, or on a PR that is closed, from a fork or not labeled `SHEPHERD_LABEL` |
| `reply <pr> <comment id> <body file>` | `gh api --method POST repos/*/pulls/*/comments/*/replies` | Non-numeric ids, PRs that aren't open, opted-in, same-repo |
| `push <pr>` | `git push origin HEAD:*` | Any ref but that PR's own head branch; the default branch; protected branches; fork PRs; commits that change `.github/`; `--force` |

| Level | `--allowedTools` entries |
|---|---|
| Read (all levels) | `Read`, `Grep`, `Glob`, `Bash(gh pr view:*)`, `Bash(gh pr list:*)`, `Bash(gh pr diff:*)`, `Bash(gh pr checks:*)`, `Bash(gh run view:*)`, `Bash(gh run list:*)`, `Bash(<RUNNER_TEMP>/lever-bin/shepherd.sh:*)` (`state`), `Bash(gh issue list:*)`, `Bash(gh search issues:*)`, `Bash(git status:*)`, `Bash(git log:*)`, `Bash(git show:*)`, `Bash(git diff:*)`, `Bash(git fetch:*)` |
| L1 (fix and push) | + `Edit`, `Write`, `Bash(git checkout:*)`, `Bash(git merge:*)`, `Bash(git add:*)`, `Bash(git commit:*)`, `Bash(<TEST_COMMAND>:*)`, `Bash(gh run rerun:*)`, `shepherd.sh push` and `reply`, `Bash(gh pr comment:*)`, `Bash(gh issue create:*)`, `Bash(gh issue comment:*)` (one flake issue per test) |
| L2 (default) | + `shepherd.sh resolve-thread`, `Bash(gh pr ready:*)` |
| L3 / L4 | + `Bash(gh pr merge:*)`, always with `--match-head-commit <sha>` |

The shipped workflow lists `shepherd.sh` once, as `Bash(${{ runner.temp }}/lever-bin/shepherd.sh:*)`. Below L2, delete the `resolve-thread` case from your copy of the script rather than adding a second pattern. Never add `Bash(gh api:*)`, `Bash(gh api --method GET:*)` (gh honours the last `--method`, so `--method GET --method DELETE` sends DELETE), `Bash(gh api graphql:*)` (any mutation, including `mergePullRequest`), or a `git push` pattern: a wildcard in `HEAD:*` matches `HEAD:main`. Add `gh pr merge` only when you set `AUTONOMY` to L3 or L4. Never add `Bash(gh workflow:*)` even though `actions: write` would allow it.

Fork PRs are skipped: the job `if:` requires the PR's head repository to be this repository, and `shepherd.sh` refuses cross-repository PRs on the sweep.

### What the tool list does not contain

The allowlist stops a misled agent from calling a write it wasn't given. It does not contain code the agent runs. `TEST_COMMAND` executes the PR's code, and `Edit` lets the agent change that code (and git hooks) first, so treat everything `TEST_COMMAND` runs as having the agent's token:

- `claude-code-action` puts the App installation token in the agent's `GH_TOKEN` and in the `origin` remote URL in `.git/config`. Code run by `TEST_COMMAND` can read both and use the App's full installation permissions until the token expires (about an hour).
- What the workflow does remove: `persist-credentials: false` keeps the job's own `GITHUB_TOKEN` out of `.git/config`, and `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB: "1"` strips the Anthropic API key, the OIDC request token (with it, code could mint fresh App tokens) and the runner's tokens from every subprocess the agent starts.
- So the real ceiling is the App installation plus your rulesets. Before going live: a ruleset on the default branch (and release branches) that requires a pull request with an approving human review and blocks force pushes, with no bypass for the Claude App; and label `shepherd` only on PRs whose code you'd run with the App's write access. Install the App on this repository only.
- If that ceiling is too high, run below L1: drop `Edit`, `Write` and `TEST_COMMAND` from the list, and the shepherd reruns, replies and hands off without executing PR code.

## Dials

| Dial | Default | Options |
|---|---|---|
| `AUTONOMY` | L2 (mark ready + hand-off; a human lands) | L0–L4, capped by rulesets, CODEOWNERS, merge queue and `.github/lever.yml` |
| `SHEPHERD_LABEL` | `shepherd` | Any label; or scope by author |
| `MAX_FIX_PUSHES` | 3 per PR | Integer |
| `FLAKE_POLICY` | `rerun_once_then_file` | `rerun_once`, `never_rerun` |
| `RESOLVE_POLICY` | Resolve threads you fixed; leave declined ones for the reviewer | Resolve all once dispositioned; reviewer resolves all |
| `UPDATE_STRATEGY` | Merge base into head | Rebase (own branches only) |
| `API_BUDGET` | ≤10 calls per run; ≤75 REST calls per merged PR | Integers |
| Model tier | Standard (`--model claude-sonnet-5-5`) | Premium (`opus`) for repos where fixes are subtle; record why in the workflow's top comment |
| Turn and time cap | `--max-turns 60`, `timeout-minutes: 30` | The run ends instead of idling |
| `STOP_AFTER` | 3 consecutive `BLOCKED` with the same reason | Integer; then remove nothing, escalate once |

## Outcome metric

- **Primary:** opted-in PRs landed (or handed off at L2) within 24 h of first green / opted-in PRs.
- **Also:** time to green, time to merge, reruns per PR (and how often a rerun actually passed), API calls per merged PR.
- **Guardrail:** reverted shepherd merges (target zero); human overrides of thread dispositions.

## Runner

LLM-driven: classifying a red check, writing a root-cause fix and dispositioning a thread are judgement. Pick `gha` or `routine` (`automate.runner` in `~/.claude/lever.yml`, or answer when `automate-it` asks). Optional hybrid: make the hourly sweep a plain `gh pr list --label shepherd --json number,mergeable,statusCheckRollup` step and start the model job only when an opted-in PR is green-and-unlanded, conflicting or red; a healthy hour then costs no model call.

- **`gha` (recommended).** The shipped [workflow.yml](workflow.yml). The tool list keeps merge out below L3 and the per-PR `concurrency:` group keeps runs from racing.
- **`routine`.** A Claude Code cloud routine with GitHub-event triggers (label, CI failure, review) plus an hourly schedule. No workflow file and no API key in repo secrets; runs under your account, so its pushes and replies appear as the routine's GitHub identity, and its pushes start CI. Weaker enforcement: there is no `--allowedTools`, so "no merge below L3" rests on CONTRACT rule 7 and a branch ruleset that requires a human approving review. Use `routine` only at L0–L2 with that ruleset in place. Check whether the routine may push to non-`claude/` branches before you rely on it to push fixes to someone's PR branch.

### Install: `gha`

1. Preflight: `gh auth status` shows the `workflow` scope (`gh auth refresh -s workflow` if not); `gh secret list` shows `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN` and the Claude GitHub App is installed (`/install-github-app` sets up both); `gh workflow list` shows no existing shepherd. Create the label if missing: `gh label create shepherd`.
2. Fill PARAMETERS in [prompt.md](prompt.md). Start at `AUTONOMY: L2`. Set `workflows:` in the workflow to your CI and replace `Bash(make test:*)` with your `TEST_COMMAND`. Add the default-branch ruleset described under "What the tool list does not contain" if you don't have one.
3. Create paused:

```sh
mkdir -p .github/lever/automations/pr-shepherd
cp prompt.md shepherd.sh pr-state.graphql .github/lever/automations/pr-shepherd/
chmod +x .github/lever/automations/pr-shepherd/shepherd.sh
cp workflow.yml .github/workflows/lever-pr-shepherd.yml
git add .github/lever/automations/pr-shepherd .github/workflows/lever-pr-shepherd.yml
git commit -m "ci: add lever-pr-shepherd (paused)"
git push
gh workflow disable lever-pr-shepherd.yml
```

4. Verify: `gh workflow view lever-pr-shepherd.yml` shows it disabled; read the committed YAML back: four event triggers plus dispatch, the label and actor conditions in `if:`, no `gh pr merge` in `--allowedTools` at L2, no `gh api` or `git push` entry (only `shepherd.sh`), `persist-credentials: false`, `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB: "1"`, `--model`. `gh variable list` must not show `LEVER_PR_SHEPHERD_LIVE`.
5. Dry run on 3 historical PRs: one red from a real bug, one red from a flake, one green with open threads. `gh workflow enable lever-pr-shepherd.yml` (events stay skipped while the variable is unset), then for each:

```sh
gh workflow run lever-pr-shepherd.yml -f mode=dry_run -f resource_url=https://github.com/<owner>/<repo>/pull/<n>
```

   Read the three records in the job summaries. For a strict local parity run instead: `claude -p "MODE: dry_run
RESOURCE: <PR url>
SHEPHERD_SH: .github/lever/automations/pr-shepherd/shepherd.sh
Follow .github/lever/automations/pr-shepherd/prompt.md exactly." --allowedTools "<the Read row above, with that path for shepherd.sh>"`.
6. Enable on approval: `gh variable set LEVER_PR_SHEPHERD_LIVE --body true`. Create the [improver](../improver/README.md) paused, with `OUTPUT_QUERY: is:pr label:shepherd updated:>={window_start}`, `GOOD_OUTCOME: merged without a human push or revert`, `BAD_OUTCOME: human reverted, re-opened a thread, or pushed a fix the shepherd missed`.
7. To raise to L3/L4 later: append `,Bash(gh pr merge:*)` to `--allowedTools` and change `AUTONOMY` in the same commit.

Pause with `gh workflow disable lever-pr-shepherd.yml`.

### Install: `routine`

1. Preflight: no existing shepherd on the routines page or in `gh workflow list`; a branch ruleset on the default branch requires a human approving review; `gh label create shepherd`.
2. Fill PARAMETERS in [prompt.md](prompt.md) (`AUTONOMY: L2` at most), commit it to `.github/lever/automations/pr-shepherd/prompt.md` and push.
3. Create it paused. In Claude Code:

```text
/schedule Create a routine named lever-pr-shepherd on <owner>/<repo> with these triggers: GitHub pull_request labeled with "shepherd"; GitHub workflow_run completed with conclusion failure for workflow "<CI>" on pull request branches; GitHub pull_request_review submitted on PRs labeled "shepherd" (skip reviews by bots); and hourly. Prompt:
MODE: act
RESOURCE: the triggering pull request, or "sweep" on the hourly run
Follow .github/lever/automations/pr-shepherd/prompt.md exactly. MODE above overrides the file's MODE for this run.
Give it access to <owner>/<repo> only and no other connectors. Create it paused.
```

4. Verify on claude.ai/code/routines: paused, one repo, no extra connectors, the triggers and filters above. If a filter isn't available on the trigger, CONTRACT rule 1 ends the run `NOOP(not opted in)`, but each one still costs a run.
5. Dry run: set the prompt to `MODE: dry_run` and `RESOURCE: <PR url>`, then **Run now**, once for each of the 3 PRs above. Read the records in the transcripts.
6. Enable on approval: restore `MODE: act` and the `RESOURCE` line, then resume it. Create the improver routine paused, as in [improver](../improver/README.md).

