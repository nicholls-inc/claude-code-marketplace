# main-red-triage automation

**Purpose.** When the default branch goes red, classify the failure within minutes (flake, regression or infrastructure), take the cheapest action that restores green, and leave one clear record. A red main blocks everyone's merges; the cost compounds by the minute.

## Triggers

| Trigger | Workflow config | Why |
|---|---|---|
| `workflow_run` | `workflows: ["<CI workflow>"]`, `types: [completed]`, job `if:` on `conclusion == 'failure'` and `head_branch == default_branch` | The event. The prompt re-checks the branch, so a missing filter costs runs, not correctness. |
| `schedule` | `cron: "17 * * * *"` (hourly) | Sweep: catches missed events and closes the loop on earlier actions (did the rerun pass, did the fix PR land). |
| `workflow_dispatch` | inputs `mode: dry_run\|act`, `resource_url` | Dry runs on historical failed runs. |

Do not add a `workflow_run` on the lever workflows themselves for this worker. It would double its runs without adding signal.

## Permissions and tools (least privilege)

Two layers, both required: the job's `permissions:` block and `claude_args: --allowedTools`.

| Need | `permissions:` | `--allowedTools` |
|---|---|---|
| Read | `contents: read`, `pull-requests: read` | `Read`, `Grep`, `Glob`, `Bash(gh run list:*)`, `Bash(gh run view:*)`, `Bash(git log:*)`, `Bash(git show:*)`, `Bash(git diff:*)`, `Bash(gh pr list:*)`, `Bash(gh pr view:*)`, `Bash(gh issue list:*)`, `Bash(gh issue view:*)` |
| Notify | `issues: write` | `Bash(gh issue create:*)`, `Bash(gh issue comment:*)` |
| `ACTION: notify+rerun` (default) | `actions: write` | + `Bash(gh run rerun:*)` |
| `ACTION: fix_pr` | `contents: write`, `pull-requests: write` | + `Edit`, `Write`, `Bash(git checkout:*)`, `Bash(git add:*)`, `Bash(git commit:*)`, `Bash(git push:*)`, `Bash(gh pr create:*)` |
| Granted test-only fixes on main (L3) | `pull-requests: write` | + `Bash(gh pr merge:*)` |

No `Bash(gh workflow enable:*)` or `disable`, and no write to `.github/workflows/`. The default `GITHUB_TOKEN` can't push workflow-file changes, which is the right floor for a triage worker.

## Dials

| Dial | Default | Options |
|---|---|---|
| `ACTION` | `notify+rerun` | `notify` (file/update one issue), `fix_pr` (also open a fix PR) |
| `AUTONOMY` | L1 (fix PR, never merged by the automation) | L3 only with a standing grant such as `test-only-main-fix` in `.github/lever.yml` |
| `QUARANTINE` | `propose` (the issue includes the exact skip/quarantine diff) | `pr` (open the quarantine PR) |
| `MAX_RERUNS` | 1 per failed run | 0–2 |
| `ATTRIBUTION_WINDOW` | Commits since the last green run on the default branch | Integer commit cap |
| Model tier | Standard (`sonnet`) | Premium (`opus`) if log reading is the bottleneck; record why in the comment at the top of the workflow |

## Outcome metric

- **Primary:** minutes the default branch is red per week (first red run → next green run on the default branch).
- **Also:** time to first classification, flake rate (reruns that passed / reruns), share of regressions attributed to the right commit (a human confirms or corrects the issue).
- **Guardrail:** zero tests skipped or deleted without a linked issue.

## Install

1. Preflight: `gh auth status` has `workflow` scope; the repo has an `ANTHROPIC_API_KEY` (or `CLAUDE_CODE_OAUTH_TOKEN`) secret (`/install-github-app` sets this up); no existing CI-triage workflow (`gh workflow list`); confirm the CI workflow's `name:` with `gh run list`; create labels `main-red` and `flaky-test` if missing.
2. Copy [prompt.md](prompt.md) to `.github/lever/automations/main-red-triage/prompt.md` and fill PARAMETERS with `MODE: dry_run`.
3. Copy [workflow.yml](workflow.yml) to `.github/workflows/lever-main-red-triage.yml` and set `workflows: ["<CI workflow>"]`. Commit and push, then create it paused: `gh workflow disable lever-main-red-triage.yml`.
4. Verify with `gh workflow view lever-main-red-triage.yml` and read the committed YAML back: `on:`, `if:`, `permissions:`, `--allowedTools` and `--model` match this README, and no `gh workflow enable|disable` or `gh pr merge` entry is present.
5. Dry run on 3 historical failed runs on the default branch: `gh workflow enable lever-main-red-triage.yml` (events stay skipped while `LEVER_MAIN_RED_TRIAGE_LIVE` is unset), then `gh workflow run lever-main-red-triage.yml -f mode=dry_run -f resource_url=<run URL>` for each: one known flake, one real regression, one infrastructure failure. Compare each classification with what actually happened.
6. Go live on approval with `MODE: act` in PARAMETERS: `gh variable set LEVER_MAIN_RED_TRIAGE_LIVE --body true`. Pair with an [improver](../improver/README.md): `GOOD_OUTCOME: classification matched the eventual fix`, `BAD_OUTCOME: a human relabeled the issue or the blamed commit was wrong`.
