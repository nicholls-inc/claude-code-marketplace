# stall-watchdog automation

**Purpose.** An hourly check on unattended delivery. It measures time since last merge, finds sessions and PRs that stopped moving, names the stall class, and proposes one change or none. It stays silent when delivery is healthy. It's the hourly cloud form of [skills/run-health-review](../../skills/run-health-review/SKILL.md); [automation-health](../automation-health/README.md) is the weekly one.

The human should never have to notice a stall, compute time since last merge by eye, or rekick a session by hand.

## Triggers

| Trigger | Config | `write_scope` | Why |
|---|---|---|---|
| `interval` | `hourly` | `repository` | The watchdog. Run it only while agents work unattended. |

## Tools (least privilege)

| Need | Tools |
|---|---|
| Read PRs, runs and automations | `github/search_pull_requests`, `github/pull_request_read`, `github/list_commits`, `github/actions_list`, `github/list_automations`, `github/get_automation` |
| Read sessions | Your host's session read tools, if it has them |
| Report | `github/add_issue_comment` (one comment on `REPORT_ISSUE`) |
| Rekick (optional) | Your host's session-message tool. Add it only when you raise `TAKEOVER` above `propose` |

No merge, push or automation-edit tools.

## Dials

| Dial | Default | Notes |
|---|---|---|
| `STALL_THRESHOLD` | 2 h without a merge during active hours | Below it, and with no other class firing, the run is `NOOP(healthy)` |
| `IDLE_AFTER` | 30 min with no new turn or tool call | Busy-but-idle threshold |
| `ORPHAN_AFTER` | 2 h without commits and no live owner | Takeover candidates |
| `TAKEOVER` | `propose` | `continue`, `continue_and_close`; see the takeover protocol in `run-health-review` |
| `NOOP_STORM` | 50% no-op wakes | Self-echo / no-op storm threshold |
| `REPORT_ISSUE` | A pinned "Stall watchdog" issue | |
| Model tier | Standard | |

## Outcome metric

- **Primary:** human rekicks per week, target 0.
- **Also:** busy-but-idle minutes per day, and stall minutes per day (time since last merge above `STALL_THRESHOLD`).
- **Guardrail:** share of hourly runs that post anything. Most should be silent.

## Install via GitHub MCP

1. Run [`automate-it`](../../skills/automate-it/SKILL.md). Create the pinned report issue and fill PARAMETERS in [prompt.md](prompt.md) with `MODE: dry_run`.
2. Create it paused (`"disabled": true`) with the hourly `interval` trigger and the tools above.
3. Verify with `get_automation`.
4. Dry-run against 3 past windows: one healthy, one real stall, one rate-limited. Check each class against what you saw.
5. Enable on approval with `MODE: act`. Keep `TAKEOVER: propose` until its proposals have been right for a week.
