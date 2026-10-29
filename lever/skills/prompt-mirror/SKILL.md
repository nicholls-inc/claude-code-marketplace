---
name: prompt-mirror
user-invocable: false
description: Mine your own local Claude Code session history and return a scorecard of how you direct agents, plus three suggestions illustrated with your own before/after messages. It reads your private history, so it runs only on `/lever mirror` or an explicit ask ("how am I prompting?", "coach me", "mine my sessions", "what do I keep repeating?", a weekly self-retro).
---

# prompt-mirror

A coach that reads only your own session history. The extraction runs on your machine. It separates what you typed from the far larger volume of subagent, cross-session, headless and scheduled traffic, scores how you direct agents (length, steering, polling, repeated corrections, missing done-when and stop conditions, stuck loops, fan-out), and returns three concrete suggestions using your own messages as the before. It may draft instruction or skill changes; it never applies them. Applying is [`steer-once`](${CLAUDE_PLUGIN_ROOT}/skills/steer-once/SKILL.md)'s job.

## When to use

- On demand: "how am I prompting", "coach me", "what do I keep correcting", "self-retro".
- Weekly, run locally on your own machine: on demand when a weekly reminder fires, or from a local cron job running `claude -p`. A `/loop 7d` is too long: `/loop` tasks expire after 7 days.
- After a run where you typed "still broken", "status?" or "why did you stop" more than twice.

Do not use it to evaluate someone else, to rank teammates, or from a cloud scheduled task (`/schedule`), Claude Code on the web or a GitHub Actions run. Cloud runners cannot and should not read local session history.

## Privacy rules (non-negotiable defaults; each is also a dial)

1. **Extraction is local. Analysis goes through your model.** The script reads `~/.claude` (or `$CLAUDE_CONFIG_DIR`) and writes to `~/.claude/lever/prompt-mirror/<date>/` with `0700`/`0600` permissions, and nothing it writes is uploaded. But any redacted sample Claude reads to score you goes to the Anthropic API (or your configured model provider: Bedrock, Vertex, Foundry), same as anything else in the conversation. So before reading message text, ask once: "Read redacted samples (they go to your model service), or numbers only?" Numbers only means `--no-text`: build the scorecard from counts, and skip the sample reads in steps 3 and 5 and the before/after quotes. Never paste history into an issue, PR, gist, cloud session, scheduled task or workflow.
2. **Quote sparingly.** At most one quote per suggestion, at most 25 words, redacted, and only from the "before" side.
3. **Never quote** messages flagged `heated` or `sensitive` (profanity, HR, performance reviews, compensation, health, legal). Use aggregate counts only.
4. **Redact before display:** secrets and tokens, emails, URLs, home paths, `@handles`, UUIDs and long hex, plus every term in the user's `--redact-terms` file (people, codenames, internal hosts). Never show session IDs beyond an 8-character prefix, and never in anything shared.
5. Anything leaving the machine (for example a steer-once PR) is re-shown to the user first, as the exact text that will be published.

## Where the data lives

`<home>` is `$CLAUDE_CONFIG_DIR` if set, else `~/.claude`. The format is internal to Claude Code and drifts between releases; Claude Code deletes transcripts older than `cleanupPeriodDays` (default 30).

| Source | Format | Use |
|---|---|---|
| `<home>/projects/<encoded-cwd>/<session-id>.jsonl` | One JSON line per entry: `{type, uuid, parentUuid, sessionId, timestamp, cwd, gitBranch, isSidechain, userType, entrypoint, message: {role, content}, ...}`. `type` is `user`, `assistant`, `attachment`, `queue-operation`, `system` and others. The directory name is the cwd with non-alphanumerics replaced by `-` | High confidence. Has turn origin and delivery. |
| `<home>/projects/<encoded-cwd>/<session-id>/subagents/agent-*.jsonl` | Same line format, `isSidechain: true`, `agentId` set | Machine traffic only. Counted, never typed. |
| `<home>/history.jsonl` | One line per prompt typed at the prompt box: `{display, pastedContents, timestamp (epoch ms), project (cwd), sessionId?}`. May not exist (cloud and SDK sessions do not write it) | Medium confidence. Backfills sessions whose transcript was cleaned up. No source tag. |

How to tell your words from machine traffic in a transcript:

| Signal | Meaning |
|---|---|
| `type == "user"`, `turnOrigin == "human"` or `origin.kind == "human"` | Typed message |
| `type == "user"`, neither field present (older builds), text content, none of the signals below | Candidate typed message |
| `toolUseResult` present, or `message.content` blocks of type `tool_result` | Tool result, not a message |
| `isMeta: true` (skill bodies starting "Base directory for this skill:", "Stop hook feedback:", `<system-reminder>` injections) | Harness-injected context |
| `<command-name>`, `<command-message>`, `<local-command-stdout>`, `<bash-input>` in content | Slash command, `!` bash mode or its output: counted as `command`, not scored |
| `isCompactSummary: true` ("This session is being continued from a previous conversation…") | Compaction summary |
| text starts `[Request interrupted by user` | Esc abort (`aborts_user_initiated`), not a message |
| `turnOrigin` other than `human` (for example `sdk`), or `origin.kind` other than `human` | Headless, scheduled or harness turn |
| content contains `<agent-message`, `<task-notification>`, `<child-session-event`, `<webhook-payload` | Cross-session message, subagent hand-back, background-task notice or webhook |
| content contains `<wake` or `<scheduled-task` | Scheduled or `/loop` wake |
| `isSidechain: true` or `agentId` present, or any line in `subagents/agent-*.jsonl` | Subagent traffic |
| `type == "attachment"`, `attachment.type == "queued_command"`, human origin | Typed mid-turn: delivery `steering` |
| `type == "queue-operation"`, `operation == "enqueue"` well before the matching user turn | Typed while a turn ran, delivered after it: delivery `queued`; otherwise `idle` |

Use the typed text in `message.content`; strip `<system-reminder>` and similar injected blocks. History rows need heuristics: drop `/` and `!` commands, envelopes and templates, and long third-person briefs ("User request verbatim: ..."); skip rows from a session whose transcript is still present. Drop sessions whose cwd is under `/tmp` or `/var/folders`. A 60+ character prompt whose opening recurs in 4+ sessions is usually an eval harness or pasted fan-out, so drop it when the source is uncertain. If it's clearly you typing it (the same morning prompt every day), keep it: that's an automation candidate, not noise. If the format has drifted, run the script with `--debug-keys` (key and value histograms, never content) and update the filters before scoring.

## Steps

1. **Set the dials** (table below) in one line back to the user: window, scope, tone, output. Ask only if the user named a scope you cannot infer.
2. **Extract.** Run the bundled stdlib script:
   ```bash
   python3 "${CLAUDE_PLUGIN_ROOT}/skills/prompt-mirror/scripts/mine_sessions.py" --since 30d [--repo owner/name] [--redact-terms ~/.config/prompt-mirror/terms.txt]
   ```
   It writes `stats.json`, `messages.jsonl` and `messages.csv` (redacted text, flags, correction classes). Use `--no-text` for numbers only: no message text and no text-derived phrase stats. If you read transcripts any other way, apply the same filters.
3. **Sanity-check the extraction before scoring.** Typed count and sessions plausible? `machine_per_typed` usually ≫ 1 for heavy agent users. If the user allowed samples, read 20 random redacted messages: if any read like an agent brief or report, tighten filters and re-run. Never score unfiltered history; you would be measuring your agents.
4. **Build the scorecard** from `stats.json`, compared with the previous run if one exists:

   | Metric | Field | Healthy direction |
   |---|---|---|
   | Median / p90 typed length | `typed.words` | Context, not a target |
   | Mid-turn steering share | `delivery.steering_pct` | Context; high is fine if outcomes are stated |
   | Status-ask rate | `rates_pct_of_typed.status_ask` | Down; each ask is a poll you ran by hand |
   | Repeated corrections | `correction_classes`, `repeated_correction_phrases` (≥3 sessions) | Down per class |
   | Kickoffs missing done-when | `100 − kickoffs.has_done_when_pct` | Down |
   | Kickoffs missing a stop condition | `100 − kickoffs.has_stop_condition_pct` | Down |
   | Stuck without measurement (loop smells) | `loop_smells`, `rates.stuck` vs `rates.measurement` | Fewer sessions |
   | Fan-out share | `rates.fanout`, `kickoffs.fanout_pct` | Context |
   | Same prompt typed again and again | `automation_candidates` (rows flagged `repeated_prompt`) | Each one is an `/lever automate` candidate |
   | Agent concessions ("You're right…") | `assistant.concessions_per_100` | A concession without a durable fix predicts a repeat |
   | Machine messages per typed message | `machine_per_typed` | Context: how much work follows each message |

5. **Validate each headline signal by reading.** The classifiers are regexes. For every metric you will cite, open at least 5 flagged messages and discard false positives; report the corrected number and say it was hand-checked.
6. **Pick three suggestions,** ranked by frequency × cost (time lost, sessions affected). Each one has:
   - the signal and its number (for example `status asks: 14% of typed messages, 212 in 9 sessions`);
   - **before:** one of your own messages (rules above);
   - **after:** a rewrite, or the brief field or skill that removes the need for the message;
   - the lever piece that fixes it, for example status polling → [`progress-contract`](${CLAUDE_PLUGIN_ROOT}/skills/progress-contract/SKILL.md); no done-when/stop → [`brief`](${CLAUDE_PLUGIN_ROOT}/skills/brief/SKILL.md); stuck ×2 with no measurement → [`find-the-loop`](${CLAUDE_PLUGIN_ROOT}/skills/find-the-loop/SKILL.md); same correction in ≥3 sessions → [`steer-once`](${CLAUDE_PLUGIN_ROOT}/skills/steer-once/SKILL.md); same prompt typed daily → [`automate-it`](${CLAUDE_PLUGIN_ROOT}/skills/automate-it/SKILL.md);
   - the metric that should move next run.
7. **Optionally draft** instruction or skill text for the top repeated correction, as a diff in the report. Do not write to any instruction, skill or config file. Offer `steer-once` to land it.
8. **Save** `report.md` and `suggestions.json` (suggestion, metric, baseline, date) next to the extraction. Show the report in chat.
9. **On the next run,** load the last `suggestions.json` and report each metric's change. Call a change real only if it exceeds the week-to-week spread of the previous 4 weeks (`weekly`); otherwise say "within noise".

## Dials

| Dial | Default | Options |
|---|---|---|
| Window | 30 days | any `--since/--until` |
| Scope | all local sessions | `--repo` substring |
| Redaction | strict + user term file | strict; `--keep-heated` / `--keep-sensitive` for the user's eyes only |
| Quoting | ≤1 quote per suggestion, ≤25 words | counts only (`--no-text`) |
| Text sent to the model | ask once before reading samples | numbers only (`--no-text`); redacted samples |
| Tone | direct, specific, no scolding | gentle; blunt |
| Output | report + drafted diffs | report only |
| Comparison | your own trend only | none; opt-in anonymous team aggregate (counts only, never text) |
| Cadence | on demand | weekly, run locally (reminder or local cron with `claude -p`); never a cloud scheduled task |
| Retention | keep last 8 runs | delete after report (`rm -rf ~/.claude/lever/prompt-mirror`) |

## Outputs

- `~/.claude/lever/prompt-mirror/<date>/stats.json`, `messages.jsonl`, `messages.csv`: local, redacted, private.
- `report.md`: scorecard table, three suggestions with before/after, drafted diffs, trend versus last run.
- `suggestions.json`: what was suggested and which metric should move.

## This skill's own feedback loop

Every suggestion names a metric and a baseline. Next run, the skill reports the fraction of last run's suggestions whose metric moved beyond noise. If that falls below one in three across two runs, the suggestions are generic; ground them harder in the user's repeated phrases and loop smells. Secondary: user status asks per day and repeated-correction rate, trending down.

## Failure modes it prevents

- Measuring your agents instead of you (machine traffic is often 15–20× the typed volume).
- Generic prompting advice; every suggestion carries your own before.
- Polish-anxiety: length and typos are context, not faults. Outcome, evidence and boundaries matter.
- Private text leaking into a shared artifact.
- Trusting a regex rate you never read.

## Why this exists

- This suite was built this way: one heavy user's ~9.5k typed messages across ~1.8k sessions were mined to find what to encode. Separating typed messages from agent traffic was the first and hardest step; in one month each typed message produced about 21 machine messages.
- Kickoffs had a median of 18 words. Only 6% stated a done-when, and stop conditions were the weakest field of agent-written briefs at 14%.
- As autonomy grew, status and ETA asks rose from about 2% to 13% of typed messages: polling done by hand. 40% of messages landed mid-turn.
- When agents were stuck, the effective move was "build a feedback loop" (23 explicit rescues), not more instructions. The mirror flags "stuck twice, no measurement" so you make that move sooner.
