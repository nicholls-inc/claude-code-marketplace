---
name: self-improve
description: >
  Daily self-improvement controller for how the user works with Claude Code. Use
  this skill when running the self-improve routine, the /self-improve command, or
  a scheduled "improvement"/"self-improve" task. It measures the user's recent
  Claude Code usage, checks whether its own past changes moved the metric, applies
  accepted changes, and posts up to a few evidence-backed suggestions to Slack. It
  can also propose improvements to itself. Trigger it for any "review how I use
  Claude Code", "suggest improvements to my setup", or scheduled self-improve run.
---

# self-improve

A conservative, evidence-grounded controller — not a suggestion box. The
operating principle, borrowed from Karpathy-style autoresearch and GitHub Next's
Autoloop: **propose, measure against a pre-registered metric, keep only what
improves it.** The model proposes; the data disposes.

The hard truth this design accepts: "working well with Claude Code" has no clean
loss function. So the rule is **no measurement, no suggestion** — and a deliberate
bias toward suggesting less and reverting readily on weak signal. The genome is
this file; deterministic measurement + state live in `scripts/engine.py`.

## The daily cycle (run in order)

**1 — Check your own past work first (close the loop).**
Run `python3 scripts/engine.py digest`. For each item in
`state.pending_evaluations`, compare `baseline_value` → `followup_value` against
that item's pre-registered `falsifier`. Issue the verdict it earns — not the one
you'd like:
`engine.py evaluate --id N --verdict improved|nochange|regressed --note "<numbers>"`.
If `nochange` or `regressed`, the change failed: add a **revert** suggestion to
today's batch (category `revert`, evidence = the before/after numbers). A change
that didn't move its metric does not get to stay on vibes.

**2 — Harvest yesterday's Slack decisions.** For each `state.pending` item with a
`slack_ts`, read that thread via the Slack connector and interpret the reply in
plain language. Then:
`resolve --status accepted|rejected|deferred`. **Apply** accepted items (see
*Applying changes*), then `resolve --status applied` (this captures the metric
baseline) and `engine.py commit --id N --message "<what>"` for the audit trail.

**3 — Load the constraints.** `state.lessons` (the contents of `lessons.md`) and
`state.suppressed_categories` are **hard limits**. Never propose anything a lesson
rules out. Lessons are hand-editable by the user and outrank your judgement.

**4 — Generate suggestions (≤ `max_suggestions`).** Every *grounded* suggestion
MUST carry all of:
- `evidence` — a specific quantified observation copied from the digest (a count
  from `actionable_commands`, a named `correction_snippet`, a `skills_never_invoked`
  entry). Not "this seems useful." If you can't point to a number, don't propose it.
- `falsifier` — the observation that will later show it worked, and what would show
  it failed. State direction explicitly.
- `metric_kind` + `metric_key` — the falsifier, operationalised so the engine can
  measure it next week. Pick: `correction_rate` (changes meant to reduce
  redirections, e.g. CLAUDE.md rules/scaffolding); `skill_triggers` + skill name
  (did a created/refined skill actually get used); `command_freq` + command prefix
  (did toil around a command change — say so in the falsifier).

Recurrence discipline (anti-premature-optimisation): only propose automating a
command that appears in `actionable_commands` (already filtered to
`min_command_recurrence`+). Only act on a redirection pattern if the snippets show
it **recurring**, not a one-off. One observation is an anecdote, not a constraint.

The engine **drops** any grounded suggestion missing evidence/falsifier — but
build them right regardless; the gate is a backstop, not a crutch.

**5 — Post to Slack** in the format below; capture the `ts`.
**6 — Record:** pipe the suggestions as a JSON array to
`engine.py record --slack-channel <ch> --slack-ts <ts>`. Read its `dropped` field
— if it dropped something, you proposed conjecture; learn from it. Stop.

## The challenge slot (kept, but quarantined)

Up to `challenge_slots` suggestion may be a higher-variance pick **without** a
metric (`is_challenge: 1`, `metric_kind: none`). This preserves useful variance,
but it is clearly labelled `[challenge]` in Slack and never mixed in with the
grounded picks. Include one when a genuinely interesting idea exists; don't
manufacture one to fill the slot.

## Applying changes

- **Apply on accept** (low blast radius, reversible): files under
  `~/.claude/skills/`, rules in `~/.claude/CLAUDE.md`, subagents in
  `~/.claude/agents/`. Always `engine.py commit` after, so revert is one `git`
  command.
- **Stage, never auto-edit** (changes what Claude is *permitted* to do):
  `settings.json` permissions/hooks, MCP config, dynamic workflows. Write the exact
  diff to `staged/<id>-<slug>.md` and tell the user in-thread it awaits their hand.

When in doubt, stage. Never widen permissions or add a command-running hook yourself.

## Guardrails (hard rules)

1. **Measure before you propose; the metric, not acceptance, is ground truth.**
   Do not optimise for what gets accepted — that is the Goodhart trap. Optimise for
   what the follow-up metric shows actually helped.
2. **No evidence + falsifier ⇒ not a grounded suggestion.** Concrete and testable
   only: name the file, the count, the command. Honest critique over validation.
3. **Self-modification is human-gated.** You may propose edits to `SKILL.md` /
   `engine.py` as `meta_self` items, but only apply them when accepted. No
   unattended self-rewrites.
4. **Honour `lessons.md` and `suppressed_categories` absolutely.** Never re-propose
   a pending or recently-rejected item.
5. **Bias to less.** A quiet week with no strong signal is a valid result — post
   fewer suggestions, or none, rather than padding with plausible-sounding noise.

## Slack message format

```
🌱 self-improve — <date>

1. [permissions] Allowlist `go test ./...`
   evidence: approved 6× this week (actionable_commands)
   change: add to settings.json allow-list
   check (next week): manual approvals of go test → ~0

2. [challenge] <speculative pick, clearly flagged>

Reply in-thread: "do 1, skip 2" / "all" / "none" / "1 later".
```

## Commands

```bash
python3 scripts/engine.py digest                       # measure + inventory + state
echo '[ {...} ]' | python3 scripts/engine.py record --slack-channel C --slack-ts TS
python3 scripts/engine.py resolve  --id N --status accepted|rejected|applied|deferred --note "..."
python3 scripts/engine.py evaluate --id N --verdict improved|nochange|regressed --note "..."
python3 scripts/engine.py commit   --id N --message "..."   # git audit + revert point
```

Paths honour `$CLAUDE_CONFIG_DIR` (default `~/.claude`).
