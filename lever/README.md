# lever

**Feedback loops for coding agents.** A Claude Code plugin.

```text
/lever <verb> <your words>     the verb picks the playbook, the rest is you talking normally
```

Install from the [nicholls](https://github.com/nicholls-inc/claude-code-marketplace) marketplace:

```text
/plugin marketplace add nicholls-inc/claude-code-marketplace
/plugin install lever@nicholls
```

The skill is invoked as `/lever`. If the bare name collides with another command, use `/lever:lever`.

Start a new session (or `/reload-plugins`) and try one that only reads:

```text
/lever loop our test suite feels slow. just the card, don't change anything
```

You get a Loop Card back: the metric, how to measure it, how noisy it is, and what counts as a win. Nothing gets pushed, filed or merged.

## Why

Agents get really good once they can check their own work. Give one a number, a way to measure it and a bar for "better", and it'll grind on that for hours without you. Without that, you're the loop: pasting screenshots, eyeballing scroll jank, hitting rerun on CI at midnight.

Setting up the loop is the hard part. What do you measure? How noisy is it? What's a real win and what's a lucky run? When do you stop? Most people skip it, which is fair, because it's annoying. lever asks you those questions, writes the answers down, and then lets agents iterate against it.

It comes from how some of the most AI-forward engineers at GitHub work day to day (this is a Claude Code port; see below): CI speedups, a five-day crate split run from three terminals, weeks of [unattended delivery](https://gist.github.com/EvanBoyle/8c1a97f682d92dab8c09d0e1ea73f4a0), and the automations that took over the chores. We mined our own sessions for what kept working, then put most of the effort into making it one command. Same workflow, no PhD required.

## What it looks like

Plain-words asks routed to a verb, a Loop Card, the autonomy dial: see the **[examples gallery](docs/examples.md)** for text transcripts of each verb: experiment arms, a brief, the three-terminal refactor, the PR shepherd, cross-model review, an automation, and the verbs that tune lever to you.

<details><summary>Copy-pasteable examples</summary>

```text
/lever our CI is slow and I'm tired of hitting rerun
/lever loop the editor feels janky when I scroll big files
/lever arms try the 6 ideas on the card
/lever brief add offline mode to the editor. ship it by tomorrow
/lever refactor src/core is 400k lines. cut it in half
/lever shepherd keep my stack green till it lands
/lever review this PR
/lever automate make the stack shepherd an automation for this repo
/lever mirror last 30 days
/lever steer push immediately, then verify
/lever autonomy agents can merge docs PRs in this repo, nothing else
```

</details>

## Commands

```text
/lever <what's bugging you>      not sure? describe it. lever picks the verb.

BUILD   make something new
  brief <ask>                    turn ~20 words into a brief an agent can run unattended
  deliver [brief|issues]         run lanes in parallel and land the work
  refactor <scope> <goal>        huge mechanical change: producer, packager, shepherd
  backlog                        keep enough ready issues in the pool
  shepherd [pr|stack]            keep PRs green until they land
  land [prs]                     merge plan for a stack or a batch
  status                         progress you never have to ask for

LOOP    make something measurably better
  loop <problem>                 find the feedback loop first: a Loop Card
  arms [ideas]                   one agent per idea, against a control
  review [pr|plan]               several models, blind; only agreed findings

RUN     make it happen without you
  automate <task>                turn a chore into a paused automation (Actions or routine)
  health                         why did it stall? one fix, or "no change"

YOU     tune lever to how you work
  mirror [30d]                   a scorecard of how you prompt, from your Claude Code transcripts
  steer <correction>             say it once, never again
  autonomy [L0-L5|<words>]       who merges. see or set your dial
  spend [standard|high|max]      how much model to throw at it
  setup                          first run: set your dials
  help [verb]                    this list, or one verb in detail
```

Each verb is a skill in [`skills/`](skills/). The `/` menu shows only `/lever`. Skills that only read and advise, like `loop`, can still load on their own when your ask fits. Anything that pushes, files, opens PRs or schedules runs waits for `/lever <verb>` or an explicit ask.

## Your dials

- **[Autonomy](docs/autonomy.md)**, L0 to L5: who merges. L0 only proposes. L2 gets PRs green and you merge. L5 lands, releases and watches. It's your setting, saved in `~/.claude/lever.yml`, and your repo's branch protection still wins. Start at L2 and turn it up as you trust it.
- **[Spend](docs/spend.md)**, `standard` · `high` · `max`: model tier, parallel lanes, review depth and how often things run. A repo can cap it in `.github/lever.yml`.

`/lever setup` asks three questions and writes both ([example](config/personal.example.yml)). For a team, copy [`config/autonomy.example.yml`](config/autonomy.example.yml) to `.github/lever.yml`.

Work lever starts is tracked as an issue assigned to you with the [`lever` label](docs/tracking.md), so `/lever status` can list everything in flight, needs-you first.

## Other ways to install

**Per repo:** use the plugin. The skills reference `${CLAUDE_PLUGIN_ROOT}` for docs, templates, config and scripts, so copying `skills/` alone into `.claude/skills/` breaks those paths; if you copy anyway, you have to replace that variable with your own path. In GitHub Actions, load the plugin into [`claude-code-action`](https://github.com/anthropics/claude-code-action) with its `plugin_marketplaces` and `plugins` inputs instead:

```yaml
- uses: anthropics/claude-code-action@v1
  with:
    anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
    plugin_marketplaces: https://github.com/nicholls-inc/claude-code-marketplace.git
    plugins: lever@nicholls
```

**Hacking on lever:** clone it and run `claude --plugin-dir ./lever`. It loads straight from your checkout, so `/reload-plugins` picks up edits.

**Automations:** the recipes in [`automations/`](automations/) are installed with `/lever automate`. Deterministic steps (reruns, metrics, lint, bookkeeping) become plain GitHub Actions with no model. LLM-driven steps run on the runner you pick (`automate.runner`): a GitHub Actions workflow running `claude-code-action`, or a Claude Code cloud routine (`/schedule`). Each one starts paused (workflow disabled, events gated by a `LEVER_<JOB>_LIVE` repo variable; or the routine paused), dry-runs on past events, and only then gets turned on. You need `gh` authenticated with the `workflow` scope, and either `/install-github-app` or an `ANTHROPIC_API_KEY` (or `CLAUDE_CODE_OAUTH_TOKEN`) repo secret.

## Make it yours

These skills fit how we work. Yours will be different. `/lever mirror` reads your own Claude Code transcripts (`~/.claude/projects/`) and shows you what you keep repeating, and `/lever steer` turns those repeated corrections into rules that stick. Fork it and bend it.

## Ported from github/lever

This is a Claude Code port of [github/lever](https://github.com/github/lever) (MIT, LICENSE kept), which was originally built for GitHub Copilot. Credit to GitHub for the skills, techniques and rules. What changed:

- Automations became plain GitHub Actions for deterministic steps, and a `claude-code-action` workflow or a Claude Code cloud routine for LLM-driven ones. The paused-first, dry-run, least-privilege and ACTED/NOOP/BLOCKED rules carry over.
- Session mining (`/lever mirror`) reads Claude Code transcripts under `~/.claude/projects/`.
- Cross-model review uses a Claude model panel (`opus`, `sonnet`, `haiku` subagents, plus the `lever:blind-reviewer` agent) and optional `codex` / `gemini` CLIs when installed. Same-provider panels are weaker evidence, and lever says so.
- `/every` became `/loop`.
- `copilot-instructions.md` and `.instructions.md` files became `CLAUDE.md` and `.claude/rules/`.

No equivalent: Copilot's `/goal` (put the target in the kickoff prompt and restate it in the `/loop` prompt), and the Copilot Automations improver API (improvers open an issue with a proposed prompt diff; you apply it by PR).

## Status

Early. Issues and PRs welcome.

## License

MIT. See [LICENSE](LICENSE).
