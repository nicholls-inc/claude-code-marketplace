# Brief

A brief turns a short intent into work that can run unattended. `brief` produces this as the body of an issue labelled `lever` and assigned to the outcome owner (see [`docs/tracking.md`](../docs/tracking.md)).

```
## Ask (verbatim)
> <the user's exact words>

## Outcome
Owner: <who cares about the result>
Done when: <observable condition, checkable by the agent>

## Feedback loop
<Loop Card summary or link: metric, instrument, keep bar>

## Constraints
- <constraint> (source: <who/what>, expires: <when or "end of task">)
Neighbors: <other lanes/PRs/people touching the same area>

## Budget
Timebox: <e.g. 3h> · Iterations: <n> · API/compute: <budget>

## Autonomy
Level: <L0–L5> (ceiling: <from repo>; source: <prompt/config/operator>)
Absence window: <when the human is away; what to do meanwhile>

## Stop conditions
- <success> · <give-up> · <escalate-once question>

## Report contract
Cadence: <on change / hourly / at milestones> · Format: templates/status-digest.md
Fan-out: <1 of N, sibling briefs>

## Status
<edited in place by the coordinator; last check: <time>>

<!-- lever: brief · runner=<here|cloud|later> · autonomy=<Ln> · coordinator=<who> · checked=<time> -->
```
