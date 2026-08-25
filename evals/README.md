# evals/

This directory holds the eval suite that grows out of production incidents. Each production incident gets an eval that stays in the suite as a regression test, so the same failure cannot recur unnoticed. Evals also run when `CLAUDE.md`, a skill, or a hook changes, since those are the surfaces most likely to shift agent behaviour silently.

## Naming convention

`evals/<incident-or-behaviour>.md` for a written eval (scenario, expected behaviour, and how to check it), or an executable format (e.g. a script or fixture file) with the same base name where one already exists for that incident or behaviour. Keep the name descriptive rather than referencing a ticket number alone, so the eval is legible without cross-referencing another system.

## CI linkage

`incident-eval-check.yml` verifies that a merged PR referencing an incident (via a `Fixes-Incident:` trailer or an `incident` label) is accompanied by a matching eval under `evals/` and, where applicable, a candidate invariant under `docs/invariants/`. A PR referencing an incident without a matching eval fails this check.

## Full picture

See `docs/assurance/DEVELOPMENT-FRAMEWORK.md` for how incident records and evals feed back into a new intent file at stage 1, closing the loop from Maintain back to Plan.
