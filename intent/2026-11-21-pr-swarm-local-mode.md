# Intent: pr-swarm local mode (`--local`)

## Problem statement
Codeman runs pr-swarm on every review on its Reviews page. Many of those reviews are branches with no GitHub pull request, and the owner wants results kept in Codeman only. Both skills are written around GitHub: `/pr-swarm` reads the PR and posts comments, and `/review-triage` triages review threads, replies, resolves and arms auto-merge. A caller that tells the agent to skip half of each skill is fragile, so the skills need a mode of their own (Codeman spec `intent/2026-11-20-pr-swarm-reviews.md`, D2, ticket t2).

## Proposed outcome
- `/pr-swarm --local --base <ref>` reviews `git diff <base>...HEAD`, makes no `gh` call, skips posting, and ends with the Step 7a JSON contract.
- `/review-triage --local --base <ref> [--push-ref <refspec>] [--push-remote <remote>]` runs unattended, triages pr-swarm's findings as its units, commits fixes with the `PR-Swarm-Automated: true` trailer, pushes only where told, never arms a merge, and ends with a JSON result carrying the verdict.

## Constraints
- Zero GitHub writes and zero `gh` reads under `--local`. The caller may blank `GH_TOKEN` and `GITHUB_TOKEN`.
- INV-1 and the greppable proof obligation are unchanged, byte for byte, and the grep still returns no match.
- INV-8 (commit trailer) holds unchanged.
- Small diff: one local-mode section per skill and a marked `--local` line at each GitHub-touching step. No rewrite of the GitHub path.

## Open questions
None. The R-3 step (`--spec`, `--poteto-skill`) is a separate change (Codeman ticket t7).
