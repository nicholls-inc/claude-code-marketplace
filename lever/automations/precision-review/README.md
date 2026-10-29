# precision-review automation (+ improver)

**Purpose.** A code reviewer that comments only when it can prove a defect. At most two findings per PR, each backed by a proof table, posted as inline COMMENT review comments. Silence is the default and the most common outcome.

**Why it looks like this.** Review automations tend to drift toward noise. In one set of about 2,200 review runs a day, 37% posted a review and only 9.5% left an inline comment, so most runs paid for reading a PR and produced nothing anyone acted on. The reviewers in that set had 46–49 tools each; about 8 are needed. Two review automations needed 7 hand revisions between them before they stabilised; the improver sibling exists so a person isn't the update loop.

Files: [prompt.md](prompt.md), [workflow.yml](workflow.yml), [improver-prompt.md](improver-prompt.md), [improver-workflow.yml](improver-workflow.yml).

## Triggers

| Event (`on:`) | Filter (job `if:`) | Why |
|---|---|---|
| `pull_request: types: [ready_for_review]` | — | Drafts are not reviewed. Ready is the first review. |
| `pull_request: types: [opened]` | `github.event.pull_request.draft == false` | PRs opened ready. |
| `pull_request: types: [synchronize]` | `github.event.pull_request.draft == false` | New commits. The contract reviews only the delta since the last reviewed SHA. |
| `workflow_dispatch` (`mode`, `resource_url`) | — | Dry runs and manual re-runs. |

The draft filter matters. Without it, an agent that pushes many small commits to a draft triggers a review on each one and multiplies cost without adding signal. Uncomment `paths-ignore:` in the workflow to skip generated or vendored files if your repo has them. `concurrency:` keyed on the PR number with `cancel-in-progress: true` means a fast second push cancels the review of the first.

The `if:` also skips runs whose actor is `claude[bot]` or `github-actions[bot]`, so agent-authored pushes aren't reviewed. Remove those two conditions only if you want agent-authored PRs reviewed, and keep the draft filter so agents' work-in-progress isn't reviewed on every push.

## Tools (least privilege: 11 entries, 1 write)

Job `permissions:` `contents: read`, `pull-requests: write`. The workflow passes `github_token: ${{ github.token }}` to the action, so that block is the ceiling and comments post as `github-actions[bot]`. (Without it, the action exchanges an OIDC token for the Claude GitHub App's token, whose installation permissions apply instead.)

`--allowedTools`: `Read`, `Grep`, `Glob`, `Bash(gh pr view:*)`, `Bash(gh pr diff:*)`, `Bash(<RUNNER_TEMP>/lever-bin/lever-read.sh:*)`, `Bash(git log:*)`, `Bash(git show:*)`, `Bash(git diff:*)`, `Bash(git fetch origin:*)`, `mcp__github_inline_comment__create_inline_comment`.

[`lever-read.sh`](../bin/lever-read.sh) is a fixed read-only wrapper (`pr-review-comments <pr>`: one GET endpoint, validated number, no pass-through flags). The workflow copies it from the default branch to `$RUNNER_TEMP/lever-bin/` (the checkout is the PR head, which may predate the install) and the allowlist names that copy as `Bash(${{ runner.temp }}/lever-bin/lever-read.sh:*)`.

No push, no approve, no request-changes, no merge, no issue writes. The contract asks for COMMENT-only inline comments; leaving out every other write tool makes that hard to get wrong. `contents: read` means the token can't merge or push even if a command slipped through. `pull-requests: write` would let a raw API call approve, which is why there is no `gh api` entry at all (not even `Bash(gh api --method GET:*)`: gh honours the last `--method`, so `--method GET --method POST …/reviews` would post a review) and `gh pr review` isn't listed.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `MAX_FINDINGS` | 2 | Per PR head. Not per run. |
| `MIN_PRIORITY` | P2 | P1 = wrong behaviour or data loss reachable in production; P2 = reachable bug with a workaround; P3 = latent risk. P3 posts only when `MIN_PRIORITY: P3`. |
| `MIN_CONFIDENCE` | high | `high` = traced end to end; `medium` is never posted by default |
| `REVIEW_SCOPE` | Delta since last reviewed SHA | Or full diff on `ready_for_review` |
| `SKIP_PATHS` | lockfiles, generated, vendored | Glob list; mirror it in `paths-ignore:` so skipped PRs cost nothing |
| Model tier | Premium (`--model claude-opus-5-5`) | Proof quality is the product here. Record why in the workflow's top comment; premium costs several times a standard call, so the trigger filters carry the budget. |
| Turn and time cap | `--max-turns 30`, `timeout-minutes: 20` | The run ends instead of idling. |

## Outcome metric

- **Primary:** findings addressed before merge / findings posted (a commit touched the cited lines, or the thread was resolved after a fix).
- **Also:** runs that posted anything / runs (expect it to be low; that's the point), runs per PR. **Review latency vs merge**: the share of reviews that finished after the PR merged ("retrospective"). Above 10% means the trigger fires too late or the queue is too long. Fix the trigger or concurrency, not the prompt. Also track **runs per PR** (`gh run list --workflow lever-precision-review.yml`), with an alert above 5. One PR drew 24 review sessions in 3 days.
- **Guardrail:** bugs later traced to a reviewed PR that the reviewer missed (sample them; don't chase zero).

## Runner

Every step is LLM-driven (reading a diff and proving a defect is judgement), so there is no deterministic half. Pick `gha` or `routine` (`automate.runner` in `~/.claude/lever.yml`, or answer when `automate-it` asks).

- **`gha` (recommended).** The shipped [workflow.yml](workflow.yml). The tool list and `contents: read` are what keep approve, push and merge out, and the draft filter in `if:` means drafts cost nothing.
- **`routine`.** A Claude Code cloud routine with a GitHub-event trigger on this repo's pull requests. No workflow file and no API key in repo secrets; runs on your account's usage. Weaker enforcement: there is no `--allowedTools`, so "COMMENT only, never approve or request changes" rests on CONTRACT rule 7 and ALLOWED ACTIONS rather than on a missing tool, and comments post under the routine's GitHub identity. Set `SELF_LOGIN` to that login.

### Install: `gha`

1. Preflight: `gh auth status` shows the `workflow` scope (`gh auth refresh -s workflow` if not); `gh secret list` shows `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN` (run `/install-github-app` in Claude Code if neither); `gh workflow list` for existing reviewers. If one exists, revise it instead of adding a second voice on every PR. Note the identity it posts as (`github-actions[bot]` with the shipped workflow); you need it for `SELF_LOGIN` and the improver's `REVIEWER_LOGIN`.
2. Fill PARAMETERS in [prompt.md](prompt.md). Leave `MODE: act` in the file; dispatch runs pass `mode` explicitly.
3. Create paused:

```sh
mkdir -p .github/lever/automations/precision-review .github/lever/automations/bin
cp prompt.md .github/lever/automations/precision-review/prompt.md
cp ../bin/lever-read.sh .github/lever/automations/bin/lever-read.sh   # shared with the improvers
cp workflow.yml .github/workflows/lever-precision-review.yml
git add .github/lever/automations/precision-review .github/lever/automations/bin .github/workflows/lever-precision-review.yml
git commit -m "ci: add lever-precision-review (paused)"
git push
gh workflow disable lever-precision-review.yml
```

4. Verify: `gh workflow view lever-precision-review.yml` shows it disabled; read the committed YAML back and check the three `pull_request` types, the draft and actor conditions in `if:`, exactly the 11 `--allowedTools` entries above, `contents: read`, and `--model`. `gh variable list` must not show `LEVER_PRECISION_REVIEW_LIVE`.
5. Dry run. `MODE: dry_run` is a read-only replay: it accepts merged PRs, reviews the head they merged at, and posts nothing. `gh workflow enable lever-precision-review.yml` (events are still skipped while the variable is unset), then run it on 3 merged PRs: one that later needed a bug-fix follow-up, one clean, one large refactor.

```sh
gh workflow run lever-precision-review.yml -f mode=dry_run -f resource_url=https://github.com/<owner>/<repo>/pull/<n>
gh run watch   # then read the job summary
```

   Expect a finding on the first, silence on the second, and at most two findings on the third. Each finding in the record must carry its proof table. For a strict local parity run instead, from a checkout: `claude -p "MODE: dry_run
RESOURCE: <PR url>
Follow .github/lever/automations/precision-review/prompt.md exactly." --model opus --allowedTools "<the list above, minus mcp__github_inline_comment__create_inline_comment>"`.
6. Enable on approval: `gh variable set LEVER_PRECISION_REVIEW_LIVE --body true`.
7. Create the improver from [improver-prompt.md](improver-prompt.md) and [improver-workflow.yml](improver-workflow.yml) the same way (prompt to `.github/lever/automations/precision-review/improver-prompt.md`, workflow to `.github/workflows/lever-precision-review-improver.yml`, push, `gh workflow disable`), plus `gh label create lever-proposal`. Enable it (`gh workflow enable`, then `gh variable set LEVER_PRECISION_REVIEW_IMPROVER_LIVE --body true`) after the reviewer has about 20 PRs of output.

Pause either one with `gh workflow disable <file>`.

### Install: `routine`

1. Preflight: no existing reviewer on the routines page or in `gh workflow list`; note the login the routine will post as.
2. Fill PARAMETERS in [prompt.md](prompt.md), commit it to `.github/lever/automations/precision-review/prompt.md` and push.
3. Create it paused. In Claude Code:

```text
/schedule Create a routine named lever-precision-review on <owner>/<repo>, triggered by GitHub pull_request events ready_for_review, opened and synchronize, skipping draft PRs and PRs whose latest push came from a bot. Prompt:
MODE: act
RESOURCE: the triggering pull request
Follow .github/lever/automations/precision-review/prompt.md exactly. MODE above overrides the file's MODE for this run.
Give it access to <owner>/<repo> only and no other connectors. Create it paused.
```

4. Verify on claude.ai/code/routines: paused, one repo, no extra connectors, the three event types, the draft filter (if the trigger can't filter drafts, CONTRACT rule 1 ends drafts `NOOP(not reviewable)`, but each one still costs a run).
5. Dry run: edit the prompt to `MODE: dry_run` and `RESOURCE: <merged PR url>`, then **Run now**, once for each of the same 3 PRs as the `gha` path. Read the transcripts' records.
6. Enable on approval: restore `MODE: act` and `RESOURCE: the triggering pull request`, then resume it.
7. The improver: `/schedule` a weekly routine `lever-precision-review-improver` whose prompt is `Follow .github/lever/automations/precision-review/improver-prompt.md exactly.`, paused, same single repo. Run it once with **Run now** after about 20 PRs of output, then resume it.

## Improver

| | |
|---|---|
| Triggers | `schedule` weekly (`cron: '17 15 * * 1'`, pick your own); `workflow_run` of `lever-precision-review` with `if: conclusion == 'failure'` |
| Permissions | `contents: read`, `issues: write`, `pull-requests: read`, `actions: read` |
| Tools | `Read`, `Grep`, `Glob`, `Bash(git log:*)`, `Bash(git show:*)`, `Bash(gh pr list:*)`, `Bash(gh pr view:*)`, `Bash(gh search prs:*)`, `Bash(.github/lever/automations/bin/lever-read.sh:*)` (`pr-review-threads`, `pr-review-comments`; read-only, in place of `gh api`), `Bash(gh run list:*)`, `Bash(gh issue list:*)`, `Bash(gh issue create:*)`, `Bash(gh issue comment:*)` |
| Output | At most one proposal issue per week (label `lever-proposal`), for example "raise the floor for P3 style findings: 7 of 9 were dismissed", with the prompt diff. A human applies it by PR to `.github/lever/automations/precision-review/prompt.md` or closes it with a reason. |
| Metric | The reviewer's addressed-before-merge rate, per revision (per commit to the prompt file) |

A `workflow_run` trigger on every completed reviewer run is deliberately filtered to failures only: the reviewer runs on every push, and an improver run per review would double the cost to learn nothing new before the week's sample is in.
