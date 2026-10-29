# Examples

Illustrative exchanges, abbreviated. The numbers are made up; the shapes are real. Each card is one command and roughly what comes back.

Everything is `/lever <verb> <your ask>`. Don't know the verb? `/lever <your ask>`.

## Front door

One command, a handful of verbs. The verb picks the playbook; the rest is you, talking normally.

```text
> /lever our CI is slow and I'm tired of hitting rerun

That's a Loop problem: something that exists should get measurably better.
Running /lever loop, then /lever arms.

  Build  make something new               brief · deliver · refactor · shepherd · land
  Loop   make something better            loop · arms · review
  Run    make it happen without you       automate · health
  You    tune lever to how you work       mirror · steer · autonomy · spend

Next time: /lever loop our CI is slow. All verbs: /lever help.
```

## Loop: make an existing thing measurably better

### `/lever loop`: how would we know it worked? Before touching a single file.

```text
> /lever loop our CI is slow

Loop Card, ci-time-to-green. No code changed yet.

  Outcome        PR time to green, push -> all required checks
  Instrument     paired reruns on the same commit, read with gh run
  Baseline       p50 31 min · p90 52 min (n=12, main)
  Critical path  test-windows. Everything else is done 9 min earlier.
  Noise floor    ±2.4 min at p50 (6 unchanged runs)
  Keep bar       any improvement >= 1 min on the critical path, flake rate not worse

Edit the keep bar, or say go.

> bar's good. go
```

### `/lever arms`: one agent per idea. Keep what clears the bar. Write down the rest.

```text
> /lever arms try the 6 ideas on the card

6 arms and 1 control, 3 rounds each, same commit.
An arm is kept only if it beats control in every round.

  shard tests 4 -> 8           -6.1 min        KEEP
  treeless git fetch           -3.6 min        KEEP
  bigger runner                -2.0 min        DROP  2x cost
  cache the package registry   -0.4 min        DROP  under the bar
  split the lint job            0 on crit path DROP
  the thing from that tweet    +0.2 min        DROP  lol

2 kept, stacked into 2 PRs for review. 4 negatives in the ledger so nobody
burns an afternoon on them again.
```

### `/lever review`: several models, reviewing blind. Only what survives gets posted.

```text
> /lever review this PR

  raw findings, 3 reviewers               11
  at least 2 reviewers agree               3
  survived a proof attempt                 1

Posted 1 inline comment: cache key ignores the lockfile, so a dependency bump
serves stale builds. Repro attached.
Dropped 10. If your bot comments on every PR, people mute your bot.
```

## Build: something new, no signal yet

### `/lever brief`: you type 12 words. It writes the brief an agent can run without you.

```text
> /lever brief add offline mode to the editor. ship it by tomorrow

  Your words   "add offline mode to the editor. ship it by tomorrow"
  Done when    edit with the network off -> reconnect -> nothing lost.
               Proven by an E2E test, not a vibe.
  Budget       standard · 18 h · 4 lanes
  Autonomy     L2: agents open PRs, you merge
  Stop if      it needs a schema migration, or 2 lanes are blocked at once
  Reports      only when something changed

Filed #142, assigned to you, label lever. 4 lanes as sub-issues:
storage, sync queue, conflict UI, E2E.

Who runs it? here · cloud · later

> here

Coordinating #142 from this session. /loop 30m keeps it going; status goes on
the issue. Go to bed.
Tomorrow, from anywhere: /lever status. Make it the default: brief.runner: here.
```

### `/lever refactor`: three terminals, one job each. They never talk to each other.

```text
> /lever refactor src/core is 400k lines. cut it in half

Paste one block into each terminal. They coordinate only through the branch
and the PR stack.

  Producer   one split per commit, tests green, until core is <= 200k.
             /loop 30m report lines vs. baseline
  Packager   /loop 3h turn new commits into PRs under 2k lines each
  Shepherd   /loop 20m walk the stack: conflicts, red CI, review comments

Lesson baked in: push immediately, then verify. You merge.
```

## Build: getting work landed

### `/lever shepherd`: stop watching CI. It wakes on events and fixes the root cause.

```text
> /lever shepherd keep my stack green till it lands

wake 7
  PR 1   green, approved                                    READY FOR YOU
  PR 2   red: not a flake. Upload timeout too low for
         big files.                                         FIXED · PUSHED
  PR 3   conflict with main                                 REBASED · PUSHED
  reviews  3 threads: 2 fixed, 1 answered with why not      RESOLVED

Next wake on a CI or review event. Nothing changes, nothing posted.
```

## Run: work that should happen without you

### `/lever automate`: did it by hand three times? Now it's a GitHub Actions workflow. Paused.

```text
> /lever automate make the stack shepherd an automation for this repo

  Created    lever-pr-shepherd · PAUSED (workflow disabled)
  Trigger    checks completed, PR updated. Drafts skipped.
  Tools      6 allowed tools. Only what it needs.
  Improver   proposes edits to its own prompt as an issue. You apply by PR.

Dry run on 3 past events:
  PR updated, checks green                  NOOP
  review comment: "nit: rename this"        NOOP  author's call
  main moved, stack conflicts               REBASE + PUSH

Turn it on?

> ship it
```

## You: tune lever to how you work

### `/lever mirror`: how do you actually steer agents? Your own sessions know.

```text
> /lever mirror last 30 days

412 prompts, 61 sessions. Extracted locally; only redacted samples went to
the model.

  "status?" / "any update?"       54x
  CI logs pasted by hand          22x
  "push, then verify"              9x

1. You're the status page. Add /lever status and it reports when something changes.
2. You're the CI reader. Give it gh run and let it read logs itself.
3. Same correction 9 times. That's a rule: /lever steer.
```

### `/lever steer`: say it once. It writes the rule, then checks the rule worked.

```text
> /lever steer so which bit about push immediately THEN verify did you not understand

Fair. You've said this 9 times in 3 weeks. The cheapest layer that would have
stopped all 9 is an instruction, not a CI check.

  Opened PR  CLAUDE.md +2 lines: push the commit before local validation,
             then verify the pushed head.
  Evidence   the 9 messages, quoted, with dates
  Re-check   in 7 days. Keep if repeats drop to 0; otherwise escalate a layer.
```

### `/lever autonomy`: who merges is your call. Set it once. Turn it up as trust grows.

```text
> /lever autonomy

L2 here: I get PRs green and ready, you land them. The lowest layer wins:

  You        L2, your default. Your side project is L4.
  This repo  max L4 · .github/workflows/** capped at L2
  GitHub     required checks, no required approval · ceiling L5

> /lever autonomy agents can merge docs PRs in this repo, nothing else

~/.claude/lever.yml
  autonomy:
    repos:
+     acme/payments:
+       default: L2
+       paths: { "docs/**": L4 }

Docs PRs land themselves once green. Everything else stays with you.
Your words go in history. Save?

> yep
```
