# improver (generic sibling template)

**Purpose.** Every worker automation gets a sibling that watches its outcomes and drafts one revision at a time for a human to apply. Without it, the human is the update loop: a known prompt bug keeps running until someone finds time to edit it. The improver turns that lag into a review queue. It never updates the worker itself.

**Pairs with:** any worker in this folder. Filled-in versions: [precision-review/improver-prompt.md](../precision-review/improver-prompt.md), [repro-gate/improver-prompt.md](../repro-gate/improver-prompt.md).

Files: [prompt.md](prompt.md), [workflow.yml](workflow.yml). Installed as `.github/lever/automations/<worker>/improver-prompt.md` and `.github/workflows/lever-<worker>-improver.yml`, next to the worker it improves.

## Triggers

| Event (`on:`) | Filter (job `if:`) | Why |
|---|---|---|
| `schedule: - cron: '17 15 * * 1'` | — | Main path, weekly. Enough samples per revision to judge. Cron is best-effort, so the exact minute doesn't matter. |
| `workflow_run: workflows: [lever-<worker>], types: [completed]` | `github.event.workflow_run.conclusion == 'failure'` | React to real failures quickly. NOOP runs exit 0, so they never wake the improver. |
| the same `workflow_run`, unfiltered | — | **Only for low-volume workers** (a few runs a day). On a busy worker this doubles the run count; leave the failure filter in. |
| `workflow_dispatch` | — | One hand-run before enabling. |

## Tools (least privilege)

Job `permissions:` `contents: read`, `issues: write`, `pull-requests: read`, `actions: read`; `github_token: ${{ github.token }}` so that block is the ceiling.

`--allowedTools`: `Read`, `Grep`, `Glob`, `Bash(git log:*)`, `Bash(git show:*)`, `Bash(gh pr list:*)`, `Bash(gh pr view:*)`, `Bash(gh search prs:*)`, `Bash(.github/lever/automations/bin/lever-read.sh:*)`, `Bash(gh run list:*)`, `Bash(gh issue list:*)`, `Bash(gh issue create:*)`, `Bash(gh issue comment:*)`. Add `Bash(gh search issues:*)` and `Bash(gh issue view:*)` for workers whose outputs are issues.

[`lever-read.sh`](../bin/lever-read.sh) stands in for raw API reads: `pr-review-comments`, `pr-review-threads`, `issue-timeline`, `issue-comments`, each one fixed GET or GraphQL read on a validated number. Never `Bash(gh api --method GET:*)` (gh honours the last `--method`, so `--method GET --method DELETE` deletes) or a mid-pattern wildcard such as `Bash(gh api repos/*/issues/*/timeline:*)` (the `*` can swallow a different endpoint and `-X DELETE`).

`gh issue comment` also covers the optional weekly summary on a tracking issue (`SUMMARY_TARGET`).

Never: `contents: write`, `actions: write`, `Bash(gh workflow:*)`, or anything that writes to `.github/`. A human applies every proposal by PR.

## Dials

| Dial | Default | Options |
|---|---|---|
| `MIN_SAMPLES` | 20 eligible worker outputs on the current revision | Integer; 30 for noisy metrics |
| `MAX_PROPOSALS` | 1 per run | 2 |
| Proposal scope | One PARAMETERS value or one CONTRACT rule | Model or trigger-filter change, with evidence |
| May propose | Narrowing: fewer tools, tighter filters, lower caps, stricter quiet exit | Widening tools, autonomy or write scope is human-only: describe it, don't propose it |
| Model tier | Standard | Premium only if proposals are being rejected for poor reasoning |
| Runner | Same as the worker | `gha` or `routine` |

## Outcome metric

Accepted proposals that moved the worker's metric in the predicted direction / proposals made. Also track time from evidence to applied fix (the lag it removes) and the share of runs ending `NOOP(no new evidence)`, which should be most of them.

## Runner

LLM-driven: deciding which single change the evidence supports is judgement. The counting in step 4 is deterministic and can move into a plain workflow step that writes the numbers to a file the prompt reads, once you trust the formulas. Use the same runner as the worker.

- **`gha`.** The shipped [workflow.yml](workflow.yml). `contents: read` means it can't push to `.github/` even if asked.
- **`routine`.** A weekly Claude Code cloud routine. Weaker enforcement: nothing but the CONTRACT keeps it from pushing, so give it access to the one repo and keep "never push, never edit .github/" explicit. It can't see a `gha` worker's failures as an event; the weekly run reads them from `gh run list`.

### Install: `gha`

1. Preflight: `gh auth status` shows the `workflow` scope; `gh secret list` shows `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN`; `gh workflow view lever-<worker>.yml` exists; `git log -- .github/lever/automations/<worker>/` shows its current revision. `gh label create lever-proposal` if missing.
2. Fill PARAMETERS in [prompt.md](prompt.md) (worker workflow, prompt path, metric, evidence query). Replace `<worker>` and `WORKER` in [workflow.yml](workflow.yml).
3. Create paused:

```sh
cp prompt.md .github/lever/automations/<worker>/improver-prompt.md
mkdir -p .github/lever/automations/bin && cp ../bin/lever-read.sh .github/lever/automations/bin/   # if not installed yet
cp workflow.yml .github/workflows/lever-<worker>-improver.yml
git add .github/lever/automations/<worker>/improver-prompt.md .github/lever/automations/bin .github/workflows/lever-<worker>-improver.yml
git commit -m "ci: add lever-<worker>-improver (paused)"
git push
gh workflow disable lever-<worker>-improver.yml
```

4. Verify: `gh workflow view lever-<worker>-improver.yml` shows it disabled; the committed YAML has `contents: read`, no `gh workflow` or `git push` in `--allowedTools`, and `workflows:` naming the worker.
5. Run once by hand after the worker has some history: `gh workflow enable lever-<worker>-improver.yml` (events stay skipped until the variable is set), then `gh workflow run lever-<worker>-improver.yml -f mode=dry_run`. Read the proposal in the job summary.
6. Enable on approval: `gh variable set LEVER_<WORKER>_IMPROVER_LIVE --body true`.

The `lever-<worker>` / `lever-<worker>-improver` names are the grouping.

### Install: `routine`

1. Commit the filled-in prompt to `.github/lever/automations/<worker>/improver-prompt.md` and push. `gh label create lever-proposal`.
2. In Claude Code:

```text
/schedule Create a routine named lever-<worker>-improver on <owner>/<repo>, weekly on Monday at 15:17 UTC. Prompt:
MODE: act
Follow .github/lever/automations/<worker>/improver-prompt.md exactly. Never push, open a PR, or edit .github/; propose by issue only.
Give it access to <owner>/<repo> only and no other connectors. Create it paused.
```

3. Verify on claude.ai/code/routines: paused, one repo, weekly.
4. Set the first prompt line to `MODE: dry_run`, **Run now**, read the proposal in the transcript, then restore `MODE: act` and resume it on approval.

## Applying a proposal (the human step)

Read the evidence and the predicted delta. Apply it, reject it with a one-line reason, or ask for a smaller change. Apply it as a PR to the worker's `prompt.md` (or workflow) that links the proposal issue; the merge commit is the new dated revision with its one reason, and the improver reads it from `git log` next week. Close the issue either way.
