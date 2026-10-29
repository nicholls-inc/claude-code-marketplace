---
name: steer-once
user-invocable: false
description: >-
  Turn a correction you keep repeating into a durable fix: mine your
  corrections, classify them, choose the cheapest layer that would have
  prevented them (instruction, skill, lint/CI check, test, automation, or
  product bug report), open a PR with an evidence block, then re-measure after
  K days and keep, escalate or revert. Opens PRs, so it runs only on
  `/lever steer` or an explicit ask ("mine my corrections", "why do I keep
  telling you X", "didn't I give you a skill for this"), or when prompt-mirror
  reports the same correction class in 3+ sessions and you say go.
---

# steer-once

Correct an agent once; the repo remembers. An apology from the agent changes nothing next session. A merged rule does. This skill takes corrections you have typed more than once, finds the layer that would have prevented them, lands the smallest durable change there with evidence, and checks a week later whether the correction actually stopped. If it didn't, it climbs one layer; if the rule made things worse, it proposes a revert.

## When to use

- **Fast path (live):** the user just repeated a correction mid-session ("again", "I told you", "why did you stop", "didn't I give you a skill"). Handle that one correction now.
- **Batch:** "mine my corrections", a weekly retro, or a `prompt-mirror` report listing a class with ≥N hits across ≥3 sessions.
- **Team variant:** recurring reviewer feedback on agent PRs in a repo (see Sources).

Not for one-off preferences, a single bad turn, or anything already handled by an open steer-once PR.

## Sources

1. **Your typed messages** via [`prompt-mirror`](${CLAUDE_PLUGIN_ROOT}/skills/prompt-mirror/SKILL.md)'s extractor (local, redacted, source-filtered):
   `python3 "${CLAUDE_PLUGIN_ROOT}/skills/prompt-mirror/scripts/mine_sessions.py" --since 21d [--repo owner/name]`.
   Use `correction_classes`, `repeated_correction_phrases`, and each message's `classes`/`flags`. Read `messages.jsonl` for the instances.
2. **The agent message before each correction** (the cause) and **after** (the response). Count concession openers ("You're right", "Agreed", "Good catch") on the same class; a concession followed by a repeat is the strongest signal that a rule is missing or not loaded.
3. **Team variant:** `changes_requested` reviews and review comments on Claude-authored PRs (for example `gh pr list --search "author:app/claude" --state all`, or PRs whose head branch starts with `claude/` from Claude Code on the web or `claude-code-action`; then `gh api repos/{owner}/{repo}/pulls/{n}/reviews` and `gh api repos/{owner}/{repo}/pulls/{n}/comments`). Same pipeline, no local history needed.

## Steps

1. **Read the dials** (table below) and [`docs/autonomy.md`](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md). State the scope and autonomy level in one line.
2. **Collect** corrections for the window. Exclude subagent, cross-session, headless, scheduled, command and template traffic (the extractor does this). Keep delivery mode: a mid-turn steer is a stronger correction than an idle follow-up.
3. **Classify** each correction. Start from the extractor's classes, then confirm by reading:

   | Class | The agent… | Typical correction |
   |---|---|---|
   | gate | invented approvals, locks, preflights, "exact GO" phrases | "you don't need authorization" |
   | over_ask | asked for what it already had | "don't ask, just do it" |
   | premature_stop | treated a milestone as done | "why did you stop?" |
   | polling | waited or slept instead of doing parallel work | "why are you waiting?" |
   | status_gap | made the human poll for state | "status? ETA?" |
   | verification_theatre | let checking replace shipping | "what did you spend 3 hours on?" |
   | stale_claim | reasoned from a proxy (stale clone, cache) | "that's not true, check the live source" |
   | shallow_fix | suppressed the symptom | "actually fix it instead of disabling lint" |
   | letter_of_law | shrank an ambitious ask | "I asked for the orchestrator pattern" |
   | scope_creep | did unrequested side quests | "why did you open a PR there?" |
   | process_default | lacked a team default procedure | "breaking changes are always two-phase" |
   | runtime_gap | hit something no prompt can fix | "rekick the stalled sessions", quota, leaked processes |

4. **Find recurrences:** same class and same underlying expectation, ≥N hits across ≥2 sessions in the window (default N=3). Merge near-duplicates; one rule per expectation.
5. **Check for an existing rule.** Search `CLAUDE.md` and `.claude/CLAUDE.md` (and nested `CLAUDE.md` in subdirectories), `CLAUDE.local.md`, `.claude/rules/*.md` (`paths:` scope), `AGENTS.md`, `.claude/skills/*/SKILL.md`, hooks in `.claude/settings.json`, `.github/lever.yml`, and personal `~/.claude/CLAUDE.md` and `~/.claude/rules/`. Run `/memory` to see which files actually loaded in a session. If a rule exists and the class still recurs, the finding is **"rule exists but is weak, misplaced or not loaded"**. Fix placement or wording, or escalate a layer, before adding more prose.
6. **Pick the cheapest durable layer** that would have prevented these specific instances:

   | The expectation is… | Layer | Example |
   |---|---|---|
   | A personal posture (tone, verbosity, how you like updates) | Personal instructions (`~/.claude/CLAUDE.md` or `~/.claude/rules/*.md`) | "Report as `x → y (Δz%)`; no preamble" |
   | A repo-wide default posture | `CLAUDE.md` (one line, with the why) | "Pre-launch service: ship and monitor; don't add approval gates" |
   | Authority or autonomy | A dial in `.github/lever.yml`, not prose | `autonomy: L3` with the grant recorded |
   | A task-specific procedure | Scoped rule (`.claude/rules/*.md` with `paths:`, or a nested `CLAUDE.md`) or a skill rule (`.claude/skills/*/SKILL.md`) | expand → migrate → contract for breaking changes |
   | Mechanically detectable | Lint or CI check | no new `eslint-disable` without a linked issue |
   | Behavioural | A test that fails today | the service recovers from a stuck lease by itself |
   | Must happen on every tool call, prompt or stop, regardless of what the model remembers | Hook in `.claude/settings.json` (`PreToolUse`, `PostToolUse`, `Stop`, `SessionStart`, `UserPromptSubmit`) | `Stop` hook blocks ending the turn while `gh pr checks` is red |
| Timing or liveness | Automation (see [`automate-it`](${CLAUDE_PLUGIN_ROOT}/skills/automate-it/SKILL.md)) | resume stalled lanes on a schedule |
   | Outside the agent's control | Product or infra bug report | sessions don't auto-resume after sleep |

   Corrections that cluster on one dial (merge authority, stop policy, status cadence, validation depth) become a dial change, not a new rule.
7. **Write the rule.** Positive, concrete, with the why and the scope, at most 3 lines. Name the dial it sets. No ALL CAPS, no "NEVER" walls, and never add a gate to prevent gates. Example: "Unattended runs never idle: when the current item lands, take the next ready item. Why: 'why did you stop' recurred in 4 sessions."
8. **Open the PR** (or file the issue, for bugs) with the evidence block below. For personal instructions there is no PR: show the diff and apply only when the user says so, or under a recorded grant. Show the user the exact evidence text before it is published to any shared repo.
9. **Land per the autonomy dial.** Default L2: a ready PR the human lands. Instruction-only PRs are a common L3 grant class ("docs and instructions PRs may self-merge when green"). Record which level applied.
10. **Record** in the ledger: class, rule, layer, PR, baseline rate, re-measure date.
11. **Re-measure after K days** (and at least M typed messages in scope). Re-run the extractor for the same scope. Compare the class rate per 100 typed messages with the baseline, and with the week-to-week spread of that class over the previous 4 weeks.
    - Fell beyond noise → **keep**; comment the result on the PR.
    - Didn't fall → **escalate one layer** (instruction → skill rule → hook or check/test → automation → bug report) and link the new PR to the old one.
    - A neighbour class rose (for example over_ask fell but premature_stop rose) → **propose a revert** or rewording, with numbers.

## Evidence block (PR body)

```markdown
### steer-once evidence
- **Class:** premature_stop: agent treated a milestone as done while ready work remained
- **Window / scope:** 2026-10-01 → 2026-10-21, this repo, typed messages only
- **Count:** 9 corrections in 4 sessions = 1.6 per 100 typed messages (baseline)
- **Agent concessions on this class:** 5 ("You're right…"), followed by repeats
- **Existing rule:** none | `path:line` exists but recurred (placement/wording issue)
- **Quotes** (redacted, dated, ≤20 words each, 3–5):
  - 2026-10-04 "why did you stop? there is more in the queue"
  - 2026-10-09 "if you're idle, take the next ready item"
- **Layer:** repo instruction. Cheaper layer insufficient because: … Costlier layer unnecessary because: …
- **Keep bar:** rate ≤ 50% of baseline and outside 4-week noise, re-measured on 2026-10-28 (K=7) or after 150 typed msgs
- **If not met:** escalate to <next layer>
```

## Dials

| Dial | Default | Options |
|---|---|---|
| Detection threshold | N=3 hits across ≥2 sessions in 21 days | any N / window |
| Mode | open PR (L2, human lands) | propose in chat; draft PR (L1); self-merge per [`docs/autonomy.md`](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md) grant |
| Destination | repo for shared expectations, personal for posture | repo only; personal only |
| Evidence | redacted quotes + counts, shown to user before publishing | counts only; paraphrase |
| Re-measure | K=7 days and ≥150 typed messages in scope | any K / M |
| Keep bar | ≤50% of baseline and outside noise | any threshold |
| Max open steer-once PRs | 3 | any |
| Severity weighting | off | weight mid-turn steers and heated messages higher (never quote heated ones) |

## Outputs

- One small PR or issue per recurring expectation, labelled `steer-once`, with the evidence block. The label is the team ledger.
- A personal ledger entry (default `~/.claude/lever/steer-once.ledger.md`, format per [`templates/ledger.md`](${CLAUDE_PLUGIN_ROOT}/templates/ledger.md)): keep, escalate or revert, with before/after rates.
- A re-measure comment on the original PR.

## This skill's own feedback loop

Per rule: class rate per 100 typed messages, before and after. Across rules: share kept versus escalated versus reverted; concession openers per 100 agent replies (should fall); time from first occurrence to merged fix (should shrink). If most rules escalate, the skill is choosing layers too cheaply. If any rule raises a neighbour class, the skill is writing over-broad rules.

## Failure modes it prevents

- Accepting "You're right" as a fix.
- Prose rules for things a check, test, dial or bug report should own.
- Rule sprawl: duplicate, contradictory or ALL-CAPS instructions nobody re-measures.
- One person's posture hard-coded into a shared repo; posture goes to personal instructions or a dial.
- Private text published as "evidence" without the user seeing it first.

## Why this exists

- Across one heavy user's sessions, agents opened 75 replies with "You're right" or an equivalent, and the same corrections recurred for days. They stopped recurring only once they became a merged instruction or skill change.
- The fastest path, from a frustrated message to a merged operating-model PR, took 16 minutes. One correction that stayed verbal took about 18 days to become a default.
- After the user had an agent mine his last ~200 typed messages into a skill rewrite: "why did you stop" nudges 27 → 0, gate language in agent messages 31% → 3%, status asks ~24% → 7%, while his message volume rose from about 50 to 87 per day. "The delivery skill is ~200 of my own corrections, compressed."
- Some corrections are bug reports. "Restart the stalled sessions" recurred 8 times after the instruction existed, because interrupted sessions were a runtime gap no prompt could fix.
