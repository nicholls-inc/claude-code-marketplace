# Where lever keeps state

Sessions are disposable. Issues aren't. Anything lever runs for longer than you're watching lives on a GitHub issue, so any session (yours, a teammate's, a cloud agent's) can pick it up with `/lever deliver #142` and know exactly what's going on.

## One label, one assignee

| What | Means |
|---|---|
| label `lever` | lever manages this. Its body has a brief and a status section. |
| assignee on the parent issue | the **human who owns the outcome**. Usually you. |
| assignee on a lane sub-issue | **whoever is doing the work**: you (a local session acting as you) or Claude (cloud: claude-code-action or a Claude Code on the web session). |
| label `lever:needs-you` | a question only a human can answer is waiting. lever removes it when answered. |

So the queries you actually want are:

```text
is:open label:lever assignee:@me                # everything lever is running for you
is:open label:"lever:needs-you" assignee:@me    # your to-do list
is:open label:lever no:assignee                 # unowned, up for grabs
```

`/lever status` with no args runs the first two across this repo and every repo with a dial in your `~/.claude/lever.yml` and prints `needs-you` first. It only reads: nothing gets merged, rekicked or edited.

**Why not `lever:<username>`?** The assignee already says whose it is, and GitHub search already filters on it. A label per person multiplies, goes stale when people leave, and puts usernames in the label list of public repos. Handing work to someone else should be a reassign, not a relabel.

lever creates both labels the first time it files an issue in a repo. If you can't (no triage permission, someone else's repo), it doesn't file anything: the brief stays in the session and lever says so.

## What's on the issue

- **Body:** the brief from [`templates/brief.md`](../templates/brief.md), your words verbatim at the top, then a `## Status` section that is **edited in place** (no comment-per-tick spam), with the last check time.
- **Sub-issues:** one per lane, each with its "1 of N" brief. GitHub shows progress on the parent for free. Link a lane with the child's `id` (not its number):
  ```bash
  gh api repos/{o}/{r}/issues/{n}/sub_issues -F sub_issue_id=$(gh api repos/{o}/{r}/issues/{child} --jq .id)
  ```
- **Comments:** only for things a human should see in their notifications: a milestone landed, a `needs-you` question, the final done-audit.
- **A one-line marker** in the body, for lever, not for people:
  `<!-- lever: brief · runner=here · autonomy=L2 · coordinator=<who> · checked=<time> -->`

## Who runs it

`/lever brief` ends with one choice. Set a default with `brief.runner` in your `~/.claude/lever.yml` and it stops asking.

| Runner | What happens | What keeps it going |
|---|---|---|
| **here** (default) | This session becomes the coordinator and runs `/lever deliver #142` now. Lanes run as child sessions (app) or background agents (CLI). | **CLI:** `/goal <done-when>` plus `/every 30m /lever deliver #142`. **App:** `/goal` plus a session automation that runs `/lever deliver #142` every 30 min. Each tick is a deliver sweep: rekick, land, refill. Close the laptop and it pauses; the issue says where. |
| **cloud** | Each lane sub-issue is handed to Claude in the cloud (an `@claude` mention or label picked up by `claude-code-action`, or a Claude Code on the web session), which opens a PR per lane on a `claude/` branch. | `/lever automate` offers a GitHub Actions workflow on the repo that shepherds PRs linked to `lever` issues at your autonomy level. It starts paused. |
| **later** | Files the issue and stops. | Nothing yet. `/lever deliver #142` from any session, any time. |

## Two sessions, one issue

The coordinator writes its name and a check time into the marker on every tick. A second session that finds someone else in the marker stays read-only and tells you who's driving.

Taking over:
- **Clean:** an explicit handoff. You or the old coordinator say so on the issue, and the new one writes itself into the marker.
- **Stale:** a check older than 2 hours. The new session may claim it, but a timestamp isn't a lock. Two sessions can both see it stale, and a paused coordinator can wake back up. So every coordinator re-reads the marker right before each write (push, merge, issue edit) and backs off the moment it names someone else.

That narrows the window to seconds. It doesn't close it, so prefer the clean handoff when you can.

`/lever status` also flags a lane as stalled when its last check is older than 2 hours or a workflow run is waiting for approval (first-time contributors and agent-opened PRs often need a click).
