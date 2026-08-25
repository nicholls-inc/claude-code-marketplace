# Gate: add-orchestrator batched sign-off

## What this gate protects

`add-orchestrator` drives the ADD (Assurance-Driven Development) fast path:
it takes a signed-off spec, splits it into modules, and dispatches several
subagents in parallel to draft "invariants" (rules a piece of code must
always satisfy, e.g. "a queue's length is never negative") for each module.
Because the work happens in parallel across many modules at once, the
orchestrator does not stop for a human decision after every subagent step.
Instead it batches the review into two checkpoints, so a human still reads
and approves the substantive decisions before they become the basis for
generated tests, without being asked to sign off on each module separately.

## What you are being asked to decide

There are two separate sign-offs in one ADD session, at different points in
the workflow:

1. **Module-partition draft (step 4).** The orchestrator proposes how the
   spec should be split into modules — which sections belong to which
   module, and which modules depend on each other — written to
   `module-map.md`. You are asked whether this split is sensible before any
   invariants are drafted against it.
2. **Findings triage (step 9).** After invariants are drafted for every
   module, three automated audits check them for spec coverage gaps,
   cross-module inconsistencies, and quality issues (vague wording, missing
   rationale, and similar). Each issue found ("finding") is written to one
   of three findings files. You are asked to mark, for each finding, which
   of four outcomes applies: fix the invariant, amend the spec instead,
   reject the finding, or defer it for later.

## What each decision means

**At the module-map gate:**
- **Edit the file** (merge modules, split rows, fix section references) —
  the orchestrator re-reads your edit and asks again.
- **Sign off** (`proceed`, `ship it`, `looks good`) — the orchestrator
  writes a session marker and dispatches invariant-drafting for every
  module exactly as partitioned in the file you approved.

**At the findings-triage gate**, for each finding:
- **Accept (fix invariant)** — the orchestrator edits the invariant doc as
  you describe.
- **Accept (amend spec)** — the finding shows the invariant was right and
  the spec needs to change instead; this routes through the separate
  `/protected-surface-amend` gate, which produces its own pull request.
- **Reject** — the finding is recorded as not applicable, with your reason,
  and nothing changes.
- **Defer** — the finding is recorded with a condition for revisiting it
  later; nothing changes now.

Only after every finding in a file has exactly one outcome marked does the
orchestrator apply that file's changes.

## What happens next

Signing off on the module map lets invariant-drafting proceed; it is not
final approval of the invariants themselves, only of how the work is
divided. Signing off on findings triage causes the triaged changes to be
applied to the invariant documents on disk. Neither sign-off commits or
merges anything — the orchestrator opens a pull request afterwards, and
**merging that pull request is what ratifies the whole batch** of decisions
made at both gates.

## How long this takes

The module-map review is usually a few minutes — it is one table. The
findings-triage review takes longer, roughly proportional to the number of
findings, since each needs an individual outcome marked; expect anywhere
from ten minutes to a longer session on a large spec.

## Who to ask if unsure

The Crosscheck maintainers via a GitHub issue on this repository.
