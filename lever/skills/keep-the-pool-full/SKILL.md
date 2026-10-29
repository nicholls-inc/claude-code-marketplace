---
name: keep-the-pool-full
user-invocable: false
description: Backlog steward that keeps a pool of ready, independently mergeable issues above a target by mining real sources (failing tests, TODOs, gaps, audits, user reports), deduplicating, sizing and labeling them ready. It files issues, so it runs only on `/lever backlog`, inside a delivery run whose idle policy is refill, on a schedule the user set up, or on an explicit ask ("file more issues", "keep them busy", "fill the backlog", "we have nothing to do").
---

# keep-the-pool-full

Once implementation, review and merging are automated, the bottleneck moves upstream to supply. Agents with nothing well-defined to pick up either sit idle or invent work. This skill keeps a pool of issues that an agent can take, finish and land on its own: each one is real, independently mergeable, small, checkable and not a duplicate. The human job shifts from "build X" to setting the pool's target and its sources.

## When to use

- The count of ready issues is below the target, or will be within a day at current throughput.
- A delivery loop or a set of worker automations reports `NOOP(no eligible work)` repeatedly.
- On a schedule (daily, or every few hours during an active run).

## When not to use

- No one or nothing is consuming the pool. Filing issues nobody picks up is noise.
- The work needs product decisions that haven't been made. File one decision issue for a human instead.
- The repo's maintainers haven't agreed to agent-filed issues. Ask once; on other people's repos default to `DRY_RUN`.

## Steps

1. **Measure the pool.** Count open issues matching `POOL_QUERY` (default `is:open is:issue label:ready no:assignee`). Measure consumption: ready issues closed or assigned per day over the last 7 days. Pool coverage = pool size / daily consumption, in days.
2. **Decide whether to file.** If coverage ≥ `TARGET_DAYS`, end `NOOP(pool healthy: N issues, C days)` and post nothing. Otherwise the deficit is `ceil(TARGET_DAYS × consumption) − pool`, capped at `MAX_FILED_PER_RUN`.
3. **Mine sources, in `SOURCES` order.** Each candidate must cite evidence a reader can check:
   - failing or quarantined tests, and flake-tracking issues;
   - `TODO`/`FIXME` with an owner-free, local fix;
   - gaps between a spec, doc or reference implementation and the code;
   - audit and report outputs (perf, accessibility, security, dependency) already in the repo or its issues;
   - user reports that are confirmed but lack an actionable task;
   - follow-ups promised in merged PR descriptions or review replies.
4. **Shape each candidate into an issue** using the checklist below. Split anything over `SIZE_CAP` into independent issues, not a dependency chain. If a candidate can't be made independently mergeable, drop it or file it as one human-decision issue.
5. **Deduplicate.** Search open and recently closed issues and open PRs for the same file, symbol, test name or error string. Embed a source fingerprint in the body (`<!-- pool-source: <kind>:<path-or-test>:<hash> -->`) and search for it before filing. If a match exists, comment the new evidence on it once (if allowed) instead of filing.
6. **File and label.** Create the issue with `READY_LABEL` and the area label. Never assign it; consumers claim work by assignment or linked PR.
7. **Report.** One line per run: `pool: before → after (target T, coverage C days), filed n, skipped-duplicate d, dropped k`. Silent when nothing was filed.

### Issue-quality checklist

- **Title** says the change, not the symptom ("Make X cache key include Y", not "X is slow").
- **Why**: one sentence plus the source evidence link (test run, file and line, report, user issue).
- **Done when**: an observable acceptance check, ideally a command or test that fails now and passes after. If you can't write one, the issue isn't ready.
- **Loop**: how the implementer verifies it (see `find-the-loop`).
- **Scope hint**: the files or area likely touched, and what is out of scope.
- **Size**: expected diff under `SIZE_CAP`; one PR; no ordering dependency on other open issues.
- **Safety**: no secrets, credentials, internal links or personal data copied from the source.

## Dials

| Dial | Default | Options |
|---|---|---|
| `POOL_QUERY` | `is:open is:issue label:ready no:assignee` | Any issue search |
| `TARGET_DAYS` | 2 days of consumption | Integer; or a fixed count |
| `MAX_FILED_PER_RUN` | 5 | Integer; keep low until the invalid rate is known |
| `SIZE_CAP` | One PR, about 200 changed lines | Lines or files |
| `SOURCES` | tests, todos, gaps, audits, reports, follow-ups | Any subset, in priority order |
| `READY_LABEL` | `ready` | Repo label |
| Mode | `file` on repos you own; `dry_run` elsewhere | `dry_run` (write a candidate list only), `file` |
| Autonomy | L0 (issues only) | This skill never writes code; see [docs/autonomy.md](${CLAUDE_PLUGIN_ROOT}/docs/autonomy.md) |

## Running it as an automation

Use `automate-it`. Writing well-formed issues takes judgement, so this is LLM-driven: the runner follows `automate.runner` (`gha`: a workflow running `anthropics/claude-code-action@v1`; `routine`: a Claude Code cloud routine via `/schedule`, hourly minimum). The duplicate and label checks can be a deterministic first step. Trigger: `schedule` with a daily cron (or hourly during an active delivery run). Tools (`--allowedTools` on `gha`; use `gh` via `Bash(gh ...:*)` entries, or the `mcp__github__*` equivalents where a GitHub MCP server is configured): `gh search issues`, `gh search prs`, `gh issue view`, repo file reads, code search, `gh run list`, `gh run view --log-failed`, `gh label list`, `gh issue create`, `gh issue edit` (labels), plus `gh issue comment` if dedupe-comments are on. Job `permissions:` `contents: read`, `issues: write`, `actions: read` (there is no triggering resource). Create it paused (`gha`: commit, `gh workflow disable`, schedule gated by the `LEVER_<JOB>_LIVE` repo variable; `routine`: created paused) and dry-run it once (`gh workflow run ... -f mode=dry_run`, or **Run now** with `MODE: dry_run`); read the candidate list before enabling.

## Outputs

- New issues labeled ready, each with evidence, acceptance check, loop and size.
- A per-run line in the run summary or ledger.

## This skill's own feedback loop

Track weekly, per source:
- **Pick-up rate**: filed issues claimed within `TARGET_DAYS`.
- **Merged-per-filed**: filed issues closed by a merged PR.
- **Invalid rate**: filed issues closed as not planned, duplicate or invalid. Above 20% for a source → tighten or drop that source.
- **Pool coverage** over time. If the pool drains to zero every day, raise `TARGET_DAYS` or `MAX_FILED_PER_RUN`; if issues age past 14 days unclaimed, lower them.

## Failure modes it prevents

- Idle workers, or workers inventing their own work.
- Big, coupled issues that stall review and CI.
- Duplicate issues from multiple agents mining the same failure.
- "Ready" issues with no way to tell when they're done.

## Why this exists

- In one unattended delivery loop, once implement, review and merge were automated, supply became the bottleneck. The human's prompts shifted from "build X" to "file real, independently mergeable issues and keep the pool full."
- "The answer can never be that we have nothing to do": idle was fixed with a standing backlog, not with more agents.
- Small, independently mergeable units are what let review, CI and merge keep up without a human; large units stalled.
- A backlog curation job run every 30–60 minutes also served as recovery memory when the coordinating session went down.
