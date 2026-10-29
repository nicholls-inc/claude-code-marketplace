# Modes

Not every piece of work is a loop. Before you pick skills, pick the mode the work is in. Most projects move through all three, and many run two at once. Not sure which mode you're in? Type [`/lever`](../skills/lever/) and say what's wrong; it picks for you.

| Mode | The work looks like | "Done" means | Human job | Start with |
|---|---|---|---|---|
| **Build** | Something new and large: a prototype, a new surface, a rewrite. There's no signal to climb yet. | The behavior exists, is merged and deployed, and works on the real path | Set direction, keep the backlog full, unblock | [`brief`](../skills/brief/) → [`keep-the-pool-full`](../skills/keep-the-pool-full/) → [`autonomous-delivery`](../skills/autonomous-delivery/) |
| **Loop** | Making an existing thing measurably better: speed, flakes, cost, quality, parity with a reference | A metric moved past its keep bar and stayed there on real traffic | Pick the metric and the keep bar | [`find-the-loop`](../skills/find-the-loop/) → [`experiment-arms`](../skills/experiment-arms/) |
| **Run** | Repeatable work that should just happen: review, triage, repro, red-main, status | Each event gets the right output, or correctly gets none | Approve the contract once, review the improver's proposals | [`automate-it`](../skills/automate-it/) → [`automations/`](../automations/) |

## Build mode

Coordination, not a feedback loop. The process is:
1. File well-formed issues.
2. Let independent sessions take them in parallel, one owner per outcome.
3. Merge, deploy, and file the next issues.

What matters is the throughput of verified, landed work: keeping enough ready issues in the pool, keeping owners from colliding, landing without serializing everyone, and reporting without being asked. Don't invent a metric to climb when there isn't one yet. Forcing a loop onto a prototype slows it down.

Skills: `brief`, `keep-the-pool-full`, `autonomous-delivery`, `landing-plan`, `pr-shepherd`, `progress-contract`.

## Loop mode

There is a signal, and the agent can read it without you. The job is to make the signal trustworthy (noise floor, paired control), then let the agent climb it. When you're the one pasting screenshots or asking "is it faster now?", you're the test runner. Run `find-the-loop`.

Loops also run as standing lanes in the cloud. The shape: a read-only auditor wakes, checks a watermark, audits one slice against a baseline, files at most a few issues with evidence, and hands them to a separate fix chain.
- **Parity audit** ([`audit-lane`](../automations/audit-lane/)): compare your UI source with a reference implementation, one surface per run. In one project, about 175 issues were filed and about 97% fixed, most the same day, with under 1% false positives.
- **Log or performance audit** ([`audit-lane`](../automations/audit-lane/)): compare a recent window of production logs to a trailing baseline. In the same project, about half of the issues were fixed and about 1 in 5 was noise. A sharper signal gives a better loop.
- **CI speed** ([`ci-speed`](../automations/ci-speed/)): a cheap daily watch files a goal, and the goal wakes an experiment lane.

Skills: `find-the-loop`, `experiment-arms`, `cross-model-review`.

## Run mode

The work repeats, and you've done it by hand at least three times. Turn it into a GitHub Actions automation (`claude-code-action`) that starts paused, holds least-privilege tools, ends every run as ACTED, NOOP or BLOCKED, and has an improver beside it. The best run often posts nothing.

Skills: `automate-it`, `run-health-review`, `automations/`.

## Moving between modes

- **Build → Loop:** once the prototype exists and people use it, find its loops (latency, errors, parity), and climb them.
- **Loop → Run:** once a loop has worked by hand a few times, move it to the cloud so it runs on every change or every night.
- **Run → Build:** standing lanes file issues, and those issues feed the next build.

Choose the [spend](spend.md) and [autonomy](autonomy.md) dials per mode. A prototype build might run L4 at `max` spend, while a standing review lane runs L1 at `standard`.
