---
name: cross-model-review
user-invocable: false
description: Reviews a PR, diff, plan or skill with 2–3 models (different Claude tiers, plus another provider's CLI when installed) working blind and independently, and surfaces only the findings they agree on or that survive a contradiction-first proof; a precision mode runs one automated reviewer that posts at most two proven findings and stays silent otherwise. Use when asked to "review this with opus, codex and gemini" (or "opus and sonnet"), "rubber duck this plan", "second opinion", "only show me issues the models agree on", when a plan is about to drive a day or more of work, when work has been stuck for hours, or when setting up review-on-ready automation.
---

# Cross-model review

The goal is a review people keep reading, which means one with few, correct findings rather than many plausible ones. Different models make different mistakes. Run them blind and independent, and agreement between them becomes evidence. A finding that only one model raised has to earn its place by surviving an attempt to disprove it. The same idea applies to plans: before you spend a day executing one, have a different model tear it apart.

The skill has two modes:
- **Panel** runs locally, with 2–3 models, each a `lever:blind-reviewer` subagent or an external CLI. Use it for PRs, diffs, plans and skills, whenever someone asks for a review.
- **Precision** runs as one automated reviewer (a GitHub Actions workflow) on every ready PR. It allows at most 2 findings, is comment-only, and stays silent by default.

## When to use / not use

Use it:
- before handing a PR to its owner (L2), or before gated self-merge (L4); agreed findings and thread dispositions are the evidence those levels need;
- before executing a plan that will take a day or more, or fan out to many workers;
- when work has made no progress for about 2 hours: write a simpler plan and run it through panel mode;
- to run review automatically on a repo where noisy bots get muted.

Don't use it:
- for style and formatting. That's the linter's job.
- as a merge gate on its own. Review is evidence for the autonomy dial ([`docs/autonomy.md`](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md)), not a substitute for CI.
- on drafts, unless someone explicitly asks.

## Panel mode: PRs and diffs

1. **Gather the same packet for every reviewer:** the diff, the linked issue or brief, the touched files' surrounding code, and the repo's review rules. Leave out the author's opinion of the change, any other reviewer's output and any hint of the expected answer. That's what *blind* means. Write the packet to one file (e.g. `review-packet.md` in a scratch dir) so every seat gets byte-identical input.
2. **Fan out** to 2–3 models, in parallel, using one neutral prompt: "Find defects that would cause wrong behavior, data loss, security exposure, or a broken contract. For each one, give the file and line, the mechanism, a concrete failing input or sequence, and your confidence. No style comments." Split large diffs by area and give each area to every model.
   - **Claude seats:** one Agent tool call per seat, all in the same message so they run in parallel, each with `subagent_type: lever:blind-reviewer`, a different `model` (default roster: `opus` + `sonnet`), and a prompt that is only the packet (or its path). The agent carries the neutral prompt and output format. If the plugin agent isn't available, use `general-purpose` with the body of `${CLAUDE_PLUGIN_ROOT}/agents/blind-reviewer.md` prepended to the packet.
   - **Another provider's seat:** check `command -v codex` and `command -v gemini`. If either is on PATH, add it as a seat with the same packet and the same prompt: `codex exec "<prompt>" < review-packet.md` or `gemini -p "<prompt>" < review-packet.md` (or name the file in the prompt). Ask for the same JSON-lines format.
   - **Be honest about the roster.** Different Claude tiers share training and blind spots. If every seat is Claude, label agreement in the output as **same-provider agreement**, and run the proof step (5) on *every* finding, agreed ones included. Only cross-provider agreement skips the proof.
3. **Normalize the output** into `findings.json` with one entry per finding: `{model, file, lines, mechanism, failing_case, severity P1–P3, confidence}`.
4. **Bucket the findings.** Two findings match when they point to the same location *and* the same mechanism. A separate pass does the matching (a fresh subagent given only `findings.json`, or you, after all seats have returned); the reviewers never see each other.
   - **Agreed:** raised by at least 2 models (tag it same-provider or cross-provider).
   - **Single:** raised by one model.
   - **Contradicted:** another model explicitly argued the opposite.
5. **Try to disprove every single and contradicted finding** (every finding, on an all-Claude panel) with the proof table below. Keep a finding only if it survives with high confidence.
6. **Rank** what's left: agreed findings first, then by severity, capped at the max-findings dial. When models disagree on severity, use the lower one.
7. **Report.** Locally, give the operator the ranked list plus counts ("3 agreed, 1 survived proof, 9 dropped"). On a PR, post **one** review with inline comments for the surviving findings only, and skip anything that duplicates an existing thread. Use the GitHub MCP tools if present (`mcp__github__pull_request_review_write` to create a pending review, `mcp__github__add_comment_to_pending_review` per finding, then submit it as a comment), else `gh api repos/{owner}/{repo}/pulls/{n}/reviews -f event=COMMENT` with a `comments[]` array, or `gh pr review <n> --comment --body` when there are no inline comments. Read existing threads first with `gh api repos/{owner}/{repo}/pulls/{n}/comments`. Put the dropped count in a single collapsed note, or leave it out.
8. **Triage existing comments on follow-up.** Weight human reviewers' comments highly. Verify bot comments before acting on them. Close every thread either with a fix or with a reason for declining.

## Panel mode: plans and skills ("rubber duck")

1. Give 1–2 *different* models the plan and its Loop Card, with no leading framing (same fan-out as above: `lever:blind-reviewer` seats on a model other than the one that wrote the plan, plus `codex`/`gemini` if installed). Ask: "What assumption is wrong? What will we be unable to measure? Which constraints are unnecessary? What is the simplest plan that reaches the same outcome?"
2. Bucket the answers the same way as for PRs. Agreed objections must be addressed. A single objection gets addressed if it survives the proof.
3. Revise the plan and record what changed and why. The author decides; the reviewers only advise.
4. When you're stuck, rubber-duck a **simpler** plan rather than the one that stalled.

## Precision mode: one automated reviewer

1. **Trigger:** `pull_request: types: [ready_for_review]`, plus `synchronize` with the job-level filter `if: github.event.pull_request.draft == false`. Never review a draft on every push.
2. Read the diff and its context. List candidate defects privately.
3. Build a **proof table** for each candidate and drop any that fails a row:

   | Premise | Counter-evidence searched | Before → after behavior | Reachability | Consequence | Remedy | Confidence |
   |---|---|---|---|---|---|---|
   | what you claim is wrong | where you looked for proof it's fine | concrete input, old vs new result | real caller or path that hits it | what breaks, for whom | smallest fix | high / medium |

4. Post **at most 2** findings (P1–P3) as inline comments. Never approve, never request changes, never nitpick, never speculate.
5. **If nothing survives, post nothing.** Silence is the expected outcome on most runs.
6. End every run with a record: `ACTED` (n findings), `NOOP` (nothing survived), or `BLOCKED` (reason). Use [automate-it](${CLAUDE_PLUGIN_ROOT}/skills/automate-it/SKILL.md) to create the automation from the recipe in `${CLAUDE_PLUGIN_ROOT}/automations/precision-review/`: a GitHub Actions workflow (`.github/workflows/lever-precision-review.yml`) running `anthropics/claude-code-action@v1`. Start it paused (`gh workflow disable`), give it the minimal tool set (`permissions:` with `contents: read`, `pull-requests: write`; `--allowedTools` limited to reading the PR, reading files and posting review comments), and pair it with an improver that proposes prompt revisions in an issue for a human to apply.

## Dials

| Dial | Default | Options |
|---|---|---|
| Mode | panel (local), precision (automation) | either, anywhere |
| Models | `opus` + `sonnet` seats, plus `codex` / `gemini` when on PATH | any roster; record the reason for each choice and its cost tier |
| Blind | on | off only for debugging the reviewers |
| Agreement threshold | 2 models | all N |
| Single-model findings | proof-checked; post only if they survive | summarize only · drop |
| Max findings posted | panel 5, precision 2 | any |
| Minimum confidence | high | medium |
| Severity floor | P3 | P1 only for noisy repos |
| Output | local `findings.json` | PR review · both |
| Drafts | skipped | review on request |
| Trigger breadth (precision) | ready_for_review + synchronize with `draft == false` | ready_for_review only (cheapest) |
| Comment-source trust | human high, bot verify-first | per-bot weights from history |
| Autonomy | L0 (comment-only) on others' repos | at L1+ the author's agent may fix agreed findings itself |
| Model tier | standard (`sonnet`) | `haiku` (cheap) · `opus` (premium) on high-risk paths; premium costs several times standard per call, so say so |

## Outputs

- `findings.json`, listing every finding with its model, provider, bucket and proof result.
- One PR review with only the surviving findings, or nothing.
- For plans: a critique, a revised plan, and a short "what changed" note.
- A run record: `ACTED` / `NOOP` / `BLOCKED`, plus counts.

## This skill's own feedback loop

- **Acted-on rate:** the share of posted findings fixed by the author or answered with "valid". This is the primary metric. If it falls below about 50%, tighten the confidence or severity dial.
- **Post rate and inline rate** per run. A rising post rate with a flat acted-on rate means the reviewer is getting noisy.
- **Per-model contribution:** agreed findings each model took part in, against findings only it raised that were later dropped. Prune models that add only noise.
- **Escapes:** bugs found after merge in reviewed code. Use them to recalibrate the severity floor.
- **Plans:** did the critique change the plan, and did the revised plan ship?

## Failure modes it prevents

- The bot gets muted because it posts nits and maybes on every push.
- One model's hallucination ends up posted as fact.
- Same-provider agreement mistaken for independent confirmation.
- Anchoring: reviewers converge because they read each other's output.
- Approving, or requesting changes, on a repo where the agent should only comment.
- Review cost multiplied by a per-push trigger with no draft filter.
- A stalled plan retried as-is instead of simplified.

## Why this exists

- About 25 times over several months, the same PR was sent to three models from different vendors (opus, codex and gemini) with "post findings to the PR". The instruction that stuck was "only feed me issues where the models are in agreeance". The habit later became a built-in review command.
- In one set of review automations on teammates' repos, about 2,200 runs a day read a PR. 37% posted a review and 9.5% left an inline finding. Defaulting to silence is what keeps teammates from muting the bot.
- Bot review comments were judged roughly 50% noise, while human comments were usually right. Hence the source-weighted triage.
- Rubber-ducking plans and skills with a second model became a standing habit, and it caught skills that were over-constrained before they shipped.
- A per-push trigger with no draft filter raised review runs ~2–3× per PR while PR volume barely moved. Triggers decide cost.
