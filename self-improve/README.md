# self-improve — an evidence-grounded Claude Code routine

A daily controller for how you work with Claude Code. It **measures** your recent
usage, checks whether its **own past changes** moved the metric, applies what you
accept in Slack, and proposes a few new evidence-backed improvements. Borrowed from
Karpathy-style autoresearch and GitHub Next's Autoloop: propose → measure against a
pre-registered metric → keep only what improves it. The model proposes; the data
disposes.

## Why it won't drift

The usual failure of self-improvement loops is running on the model's taste instead
of a metric, so conjecture compounds. Four mechanisms prevent that here:

1. **No measurement, no suggestion.** Every grounded suggestion must cite a specific
   quantified observation from the digest (`evidence`) and a `falsifier` operationalised
   as a metric the engine measures next week. The engine *drops* anything ungrounded.
   Pure speculation is allowed only in the one clearly-labelled `[challenge]` slot.
2. **Recurrence thresholds.** A command must recur `min_command_recurrence`+ times
   before it's actionable; one-offs are noise, not constraints.
3. **A closed loop.** When you accept a change, its baseline metric is snapshotted.
   After `eval_after_days`, the routine re-measures and, if the metric didn't move,
   proposes reverting its own change. It cannot keep a change on vibes.
4. **A version-controlled lessons ledger.** `lessons.md` records every rejection and
   every failed change as a hard constraint it reads each run — and you can hand-edit
   it. Git-tracking `~/.claude` makes every applied change a one-command revert.

## Install (macOS)

1. `cp -R self-improve ~/.claude/skills/self-improve` (keep `engine.py` under `scripts/`).
2. Set `slack_channel` in `config.json`.
3. Confirm Slack can **post**: in a normal session, ask Claude to post a test message
   to that channel and read your reply. If the connector is read/search-only, enable
   posting in your Slack connector settings.
4. Put your config under version control so changes are auditable and reversible:
   ```bash
   cd ~/.claude && git init
   printf 'projects/\n*.db\nshell-snapshots/\nstatsig/\n*.log\n' > .gitignore
   git add -A && git commit -m "baseline"
   ```

## Register as a Local routine

**Claude Code Desktop → Routines → New routine → Local.** Folder: any trusted folder
(`~/.claude/skills/self-improve` is fine — the skill reads `~/.claude` globally).
Schedule: daily, when your Mac is awake and Desktop is open. Prompt:
`Run the self-improve skill's daily cycle.`

## Tradeoffs to know

- A **local** routine fires only while Desktop is open and the Mac is awake — the cost
  of local transcript access. Cloud routines can't see `~/.claude/projects`.
- Transcripts auto-delete after ~30 days; `state.db` is the durable record. Raise
  `cleanupPeriodDays` in `settings.json` for longer raw history.
- The proxy metric (mainly your correction/intervention rate) is noisier than a training
  loss, so the loop is deliberately conservative: it suggests less and reverts readily.
  That conservatism is the design, not a shortfall.

## config.json

| key | meaning |
|---|---|
| `slack_channel` | where it posts / reads your replies |
| `lookback_days` | analysis window |
| `max_suggestions` | hard cap per day |
| `challenge_slots` | max speculative (un-metered) picks per day |
| `min_command_recurrence` | a command must recur this many times to be actionable |
| `eval_after_days` | wait this long before judging an applied change |

## Files

```
self-improve/
├── SKILL.md            the controller logic (the "genome")
├── config.json         channel + cadence + thresholds
├── lessons.md          human-editable hard constraints (auto-appends)
├── scripts/engine.py   measurement, evidence gate, metric snapshot/compare, state
├── state.db            created on first run (suggestions + baselines + verdicts)
└── staged/             high-blast-radius changes parked for your review
```

## engine.py subcommands

```
digest    measure + inventory + state + pending evaluations (JSON)
record    validate (evidence gate) and store new suggestions
resolve   accepted | rejected | applied (captures baseline) | deferred
evaluate  improved | nochange | regressed  (closes the loop, logs lessons)
commit    git-commit an applied change for audit + one-step revert
```
