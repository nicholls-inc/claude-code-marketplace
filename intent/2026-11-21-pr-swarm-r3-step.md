# Intent: R-3 spec review step in `/review-triage` (`--spec`, `--poteto-skill`)

## Problem statement
Codeman merges and deploys work attributed to an approved spec only when two reviews pass: pr-swarm's verdict and R-3, a check of the work against its spec in poteto-mode. The owner decided R-3 is a step of `/review-triage` (Codeman spec `intent/2026-11-20-pr-swarm-reviews.md`, D4, Outcome 8, invariant I9, ticket t7). The skill has no such step, so Codeman has no R-3 verdict to read.

## Proposed outcome
- `/review-triage` takes `--spec <path>` and `--poteto-skill <path>`. With both, Step 5b runs once after the loop: it reads poteto-mode's `SKILL.md`, follows it as a reviewer of the final diff against the spec, and edits nothing.
- The result gains `r3: {verdict, trigger?, findings[], deviations[], tip}`. `verdict` is `pass`, `changes`, `escalate` or `unavailable`. `escalate` names one trigger from a fixed list.
- An unreadable spec or `SKILL.md`, one flag without the other, or a tip that moves during the review gives `unavailable`. The run still ends with its full report.
- It works with `--local` and in GitHub mode. In neither does it write to GitHub. In GitHub mode, Step 6 does not arm auto-merge unless `r3` is `pass` on the HEAD being armed.

## Constraints
- R-3 never edits code (new INV-9, checked by reading `HEAD` and `git status --porcelain` before and after the step).
- The plugin hard-codes no path into another plugin. The caller passes both paths.
- poteto-mode is read from its `SKILL.md`, not reimplemented. An ask becomes a `deferred` finding, and any subagent runs in the foreground.
- INV-1 to INV-8 and the INV-1 grep are unchanged.
- A crosscheck finding feeds R-3's judgment and never escalates on its own.

## Open questions
None. Two triggers of the Codeman lifecycle's list (builder and R-3 still disagreeing after 2 rounds, a failed deploy fix) need history a single run does not have, so the caller owns them.
