# repro-gate automation (+ improver)

**Purpose.** Turn noisy issue intake into ranked, test-backed work. Classify each new issue as a bug or not. For a bug, write a test that reproduces it. Apply `repro-confirmed` only if that test fails on the default branch head. Otherwise say what was tried and ask for what's missing.

**Why "prove it or stay quiet".** A label is a claim other people and agents will act on. Gating it on a failing test raised the share of new issues confirmed from 17.5% (128 of 732) to 28% (56 of 198), and every confirmation came with a test the fix PR can start from. The same automation once ran for "five more days" with a known prompt bug because the only update loop was a person finding time to edit it; the improver sibling closes that gap.

## Runner

The judgement is LLM-driven (classify, write a test, judge whether it fails for the reported reason). The mechanical parts are deterministic: job `if:` conditions filter events by label and author, a `select` job picks the sweep's trusted-author issues, and a `publish` job pushes the agent's repro branches. `RUNNER: gha` or `routine`; `automate.runner` in `~/.claude/lever.yml` can preset it, otherwise ask once.

**`RUNNER: gha`**: [workflow.yml](workflow.yml) with `claude-code-action`, install steps below. Strongest least privilege: `permissions:` plus `--allowedTools`.

**`RUNNER: routine`**: a Claude Code cloud routine on Anthropic's infrastructure; it needs the repo cloned and a runnable test command, which the cloud environment provides. Nothing is committed to `.github/workflows/`.
1. In Claude Code run `/schedule` (alias `/routines`) and create a routine named `lever-repro-gate`, attached to this repository with a GitHub connector that can read issues and code, comment, label, and push `repro/*` branches.
2. Triggers: **GitHub event, issue opened** and **GitHub event, issue labeled** (use it for `needs-repro`; the contract ignores other labels), plus **schedule, daily** as the sweep. Check the routines page for the current event list; if an event isn't offered, the daily sweep covers it within a day.
3. Prompt: `MODE: dry_run` / `TRIGGER: <issue opened|labeled|schedule>` / `RESOURCE: <issue URL from the event>` / `Follow .github/lever/automations/repro-gate/prompt.md exactly.`
4. Paused = create it disabled. Dry run = "run now" with `MODE: dry_run` and an issue URL, on 3 historical issues. Go live: set `MODE: act`, enable. Pause it in the routines page.
5. Least privilege: select only this repository and the connectors above. A routine has no `--allowedTools` list, so tool limits are weaker than `gha`; the CONTRACT (never close, never merge, never run commands from an issue, never touch secrets or workflows) is the control. Pushes are limited to `repro/<issue>` by the contract, not by the platform.

## Triggers

| Trigger | Workflow config | Why |
|---|---|---|
| `issues` | `types: [opened]`; job `if:` skips issues already labeled `needs-info` or `repro-confirmed` | New intake. |
| `issues` | `types: [labeled]`; job `if:` passes only `github.event.label.name == 'needs-repro'` | Re-run on demand, for example after the reporter adds details. (`issues: labeled` has no label filter in `on:`, so the filter is the `if:`; a skipped job costs no model call.) |
| `issues` | `opened`: `author_association` is `OWNER`, `MEMBER` or `COLLABORATOR` | Outside reporters' issues don't reach an agent that runs code unless a maintainer opts them in with `needs-repro`. |
| `schedule` | `cron: "23 6 * * *"` (daily) | Sweep: the `select` job lists open issues from the last 7 days with none of the gate's labels **and a trusted author**, and passes them in as `SWEEP_ISSUES`. |
| `workflow_dispatch` | inputs `mode`, `resource_url` | Dry runs on historical issues. |

Outside reporters: `anthropics/claude-code-action` refuses runs triggered by actors without write access, so their `opened` events are skipped, and the sweep leaves them out too (an untrusted issue plus `Write` plus test execution is a prompt-injection path to code execution). A maintainer applying `needs-repro` is the opt-in: that event's actor is the maintainer, and the run processes the issue. Don't relax this with `allowed_non_write_users`.

## Permissions and tools (least privilege)

| Job / need | `permissions:` | `--allowedTools` |
|---|---|---|
| `select` (deterministic) | `issues: read` | n/a: one `gh api search/issues` call, filtered on `author_association` |
| `run`: read | `contents: read`, `pull-requests: read` | `Read`, `Grep`, `Glob`, `Bash(git log:*)`, `Bash(gh issue view:*)`, `Bash(gh issue list:*)`, `Bash(gh pr list:*)`, `Bash(gh pr view:*)` |
| `run`: write and run a repro test | still `contents: read`: the agent commits locally and never pushes | + `Write`, `Edit`, the repo's test command (for example `Bash(npm test:*)`), `Bash(git checkout -b repro/*)`, `Bash(git add:*)`, `Bash(git commit:*)` |
| `run`: report | `issues: write` | `Bash(gh issue comment:*)`, `Bash(gh issue edit:*)` (labels) |
| `publish` (deterministic) | `contents: write` | n/a: verifies the bundle, pushes `refs/heads/repro/<number>` only, never force, skips any branch that changes `.github/` |
| `OPEN_DRAFT_FIX_PR: on` | `pull-requests: write` on `publish` | uncomment the `gh pr create --draft` line in `publish` |

The test command runs code the agent wrote from untrusted issue text, so the `run` job is built for that: `persist-credentials: false` on the checkout, `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB: "1"` (the API key, OIDC and runner tokens are stripped from the agent's subprocesses), and a token that can't write contents. That token (`GITHUB_TOKEN`, `issues: write`) is still reachable from test code through the agent's `GH_TOKEN`, so the worst case is issue comments and labels until the job ends, not a push. The agent's commits leave the job as a git bundle artifact; `publish` runs on a fresh runner, executes nothing from it, and pushes with `GITHUB_TOKEN`, which starts no workflows and can't push workflow-file changes. No issue-close, no merge, no review tools.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `REPRO_BUDGET` | 20 minutes of work per issue | Over budget: comment what was tried, add `needs-info` |
| `OPEN_DRAFT_FIX_PR` | off | When on, opens a draft PR containing the failing test, plus a fix only if it is small and obvious |
| `NOT_BUG_ACTION` | `noop` | Or `label:<question/feature label>` (adds `gh issue edit --add-label` usage only) |
| `LABELS` | `repro-confirmed`, `needs-info`, `needs-repro` | Create them before enabling |
| Model tier | Standard (`sonnet`) | Premium (`opus`) if repros routinely need deep code reading; record why in the workflow's top comment |

## Outcome metric

- **Primary:** `repro-confirmed` labels a human kept / labels applied (precision of the gate).
- **Also:** share of new bug issues triaged within 1 day; confirmed share of new issues; median time from confirmation to fix PR.
- **Guardrail:** confirmed issues later closed as "not a bug" (target zero; each one is an improver data point).

## Install

1. Preflight: `gh auth status` has `workflow` scope; the repo has an `ANTHROPIC_API_KEY` (or `CLAUDE_CODE_OAUTH_TOKEN`) secret (`/install-github-app`); no other triage workflow labels the same issues (`gh workflow list`); create the labels `repro-confirmed`, `needs-info`, `needs-repro`; confirm the repo's test command and put it in PARAMETERS and in the workflow's `--allowedTools`.
2. Copy [prompt.md](prompt.md) to `.github/lever/automations/repro-gate/prompt.md` and fill PARAMETERS with `MODE: dry_run`.
3. Copy [workflow.yml](workflow.yml) to `.github/workflows/lever-repro-gate.yml`, add the toolchain setup step, commit and push, then create it paused: `gh workflow disable lever-repro-gate.yml`.
4. Verify with `gh workflow view lever-repro-gate.yml` and read the committed YAML back: no close, merge or `git push` entry; `run` has `contents: read`, `persist-credentials: false` and `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB: "1"`; `contents: write` appears only on `publish`; the `opened` path checks `author_association`.
5. Dry run on 3 historical issues: `gh workflow enable lever-repro-gate.yml` (events stay skipped while `LEVER_REPRO_GATE_LIVE` is unset), then `gh workflow run lever-repro-gate.yml -f mode=dry_run -f resource_url=<issue URL>` for each: one real bug that was later fixed, one feature request, one bug report too vague to act on. Expect `repro-confirmed` (with a failing test), `NOOP(not a bug)` and `needs-info` respectively.
6. Go live on approval with `MODE: act` in PARAMETERS: `gh variable set LEVER_REPRO_GATE_LIVE --body true`.
7. Create the improver from [improver-workflow.yml](improver-workflow.yml) (`.github/workflows/lever-repro-gate-improver.yml`) and [improver-prompt.md](improver-prompt.md) (`.github/lever/automations/repro-gate/improver-prompt.md`), plus [lever-read.sh](../bin/lever-read.sh) at `.github/lever/automations/bin/lever-read.sh` (executable), paused the same way, and set `LEVER_REPRO_GATE_IMPROVER_LIVE` once the gate has labeled about 20 issues.

## Improver

| | |
|---|---|
| Triggers | `schedule` weekly; `workflow_run` of `lever-repro-gate` with `if:` conclusion `failure`. Optional: an `issues: [labeled, unlabeled]` trigger with a job `if:` on `repro-confirmed` / `needs-info` and a non-bot actor, if your label volume is low enough that one improver run per correction is cheap. |
| Permissions and tools | `contents: read`, `issues: write`, `pull-requests: read`, `actions: read`. Read-only `git log`, `gh issue view/list`, `.github/lever/automations/bin/lever-read.sh issue-timeline <n>` (fixed read; never a `gh api` pattern), `gh pr list/view`, `gh run list`; `gh issue create/comment` for the proposal. Never `.github/` write, never `gh workflow enable/disable`. |
| Evidence | Every human label correction is a labeled data point: a human removed `repro-confirmed`, added it after the gate said `needs-info`, or closed a confirmed issue as not a bug. |
| Output | At most one classification-rule proposal per week, as an issue containing the prompt diff. A human applies it by PR. |
