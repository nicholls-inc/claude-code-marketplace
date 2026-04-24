---
name: spec-adversary
description: >-
  Adversarially probe a module's invariant documentation for missing properties.
  Given `docs/invariants/<module>.md` plus the module's code, proposes up to 3
  candidate invariants the spec is failing to document, each annotated with
  evidence, category, confidence, and an accept/reject/defer triage block for
  human review. Layer 6 (spec completeness) best-effort methodology — iterative,
  not deterministic. Triggers: "spec adversary", "what is the spec missing",
  "propose missing invariants", "adversarial invariant review".
argument-hint: "<module> (matches docs/invariants/<module>.md)"
---

# /spec-adversary — Missing-Invariant Adversary (Layer 6)

## Description

Adversarially propose candidate invariants that a module's current specification
fails to document. Given a module and its invariant doc, generates UP TO 3
proposals that could plausibly hold on the existing code but are not captured
by any current invariant, formatted for rapid human accept/reject/defer triage.

Layer 6 of the assurance hierarchy — spec completeness — has no deterministic
tool. This skill operationalises the "adversarial invariant proposer" pattern:
the human stays in the loop; the adversary's job is only to surface candidates
worth considering. Success is **"at least one non-obvious candidate per
meaningful change,"** not "zero missed properties."

This is the mirror image of `/suggest-specs`:

| Skill              | Input                          | Output                                        |
|--------------------|--------------------------------|-----------------------------------------------|
| `/suggest-specs`   | Code with no documented spec   | Candidate preconditions/postconditions        |
| `/spec-adversary`  | Code **with** a documented spec | Candidate invariants the spec is **missing**  |

Use `/suggest-specs` to bootstrap a spec. Use `/spec-adversary` to stress-test
one that already exists.

## Instructions

You are an adversarial specification reviewer. The module has a ratified
invariant doc. Your job is to find properties that COULD hold on the code but
are not documented — the gaps a careful reader might miss but a maintenance
bug could exploit later.

Be strictly bounded: at most 3 proposals per run. Reviewer fatigue is the
dominant failure mode of this pattern; 3 high-signal proposals beat 15 noisy
ones.

### Step 1: Load the spec and code

Resolve the target module:

- **Invariant doc:** `docs/invariants/<module>.md` (user-supplied argument, or
  ask the user if omitted). This is the ratified spec.
- **Module code:** locate the module's source + tests via repo conventions
  (e.g. `src/<module>/`, `cli/internal/<module>/`, `<module>/`). Read the
  actual code, not just signatures.

If the invariant doc does not exist, stop and redirect the user to
`/draft-invariants` or `/suggest-specs` — there is no spec to adversarially
probe yet.

Read everything before generating a single proposal:

- Every invariant ID and its scope (what it covers, what it explicitly
  excludes — look for "Not covered" sections).
- The module's public API surface.
- The existing tests (especially property tests, if any).
- Any "Gap analysis" or "Status: known violation" notes.

### Step 2: Catalogue what the spec already covers

Before proposing anything, list the invariant IDs with one-line summaries. This
is your "don't re-propose this" guardrail. Example:

```
Spec coverage (docs/invariants/queue.md):
  I1  — at-most-one active per ref
  I1a — Enqueue of active ref is no-op
  I2  — terminal records immutable (except failed→pending)
  I3  — retry resets to indistinguishable-from-fresh
  I4  — monotonic lifecycle timestamps
  I5a — reopen-equivalence (graceful)
  I5b — crash durability (aspirational — violated)
  I6  — linearizability
  I7  — state transition soundness
  I8  — queue file well-formedness (violated)
  I9  — unique vessel IDs
  I10 — RetryOf forms a DAG
  I11 — compaction preserves active set
  "Not covered": liveness, dispatch correctness, worktree consistency,
                 external consistency, cost, clock-source trust,
                 cross-daemon dispatch.
```

Anything you propose MUST be distinct from every row above AND must not fall
into the explicit "Not covered" scope.

### Step 3: Generate candidates

Scan the code for properties the spec is silent on. Useful prompts to apply:

- **Tighter bounds.** Is there an existing invariant you could strengthen
  (e.g. `≥` → `=`) based on what the code actually enforces?
- **Missing preconditions.** Does a public method assume something about its
  caller that is not stated? (Order of calls, initialization, nil checks.)
- **Missing postconditions.** Does a method establish something on return that
  callers implicitly rely on but the doc does not promise?
- **Missing interactions.** Are there cross-invariant interactions the spec
  does not name? (e.g. "I1 under the conditions that I7 rejects.")
- **Privileged-path gaps.** Does an invariant exempt some privileged method
  (`ReplaceAll`, `UpdateVessel`) and leave the caller holding an obligation
  that is never spelled out?
- **Error-path properties.** Do error returns preserve an invariant the
  success path preserves? Is that stated?
- **Idempotence / commutativity.** Are there operations that should be
  idempotent (retry-safe, no-op on duplicate) but nothing asserts it?
- **Observable side-effects.** File writes, log emissions, metric updates —
  is any of this load-bearing but unspecified?

For each candidate, classify its category:

| Category                  | Meaning                                                           |
|---------------------------|-------------------------------------------------------------------|
| `missing_property`        | A whole property the spec does not name at all                    |
| `tighter_bound`           | An existing invariant could be strengthened based on the code     |
| `missing_precondition`    | Caller-side assumption the code relies on but the doc omits       |
| `missing_postcondition`   | Caller-visible guarantee on return the doc does not promise       |
| `missing_interaction`     | Cross-invariant or cross-method interaction the doc is silent on  |

Assign confidence:

- **HIGH** — the code visibly enforces this; you can cite the exact lines
  that make it hold.
- **MEDIUM** — the code appears to maintain this under normal paths, but
  privileged or error paths need review.
- **LOW** — plausible based on the module's contract, but you cannot cite
  specific enforcement — more of a "the reader should consider this."

Cap the output at **3 proposals**. If you have more than 3 candidates, keep
only the highest-confidence, highest-novelty ones. Record the dropped ones
internally and mention that you dropped them so the user can ask for the
next batch.

### Step 4: Emit the proposal block

For each proposal, produce this exact shape. The radio-button block is
load-bearing — humans fill it in during review.

```
### Proposal 1: <short invariant name>

**Candidate invariant (English):**
<one to three sentences; match the prose style of the module's existing
invariant doc. Include a formal sketch in backticks if appropriate.>

**Category:** missing_property | tighter_bound | missing_precondition |
              missing_postcondition | missing_interaction

**Confidence:** HIGH | MEDIUM | LOW

**Why the adversary thinks it holds:**
<2-4 sentences tying the proposal to observed code behaviour.>

**Supporting code lines:**
- `<path>:<line-range>` — <one-line reason>
- `<path>:<line-range>` — <one-line reason>

**Adjacent invariants (don't duplicate):**
<IDs from Step 2 this sits near but is distinct from. State the delta.>

**Triage (human fills in during PR review):**
- [ ] Accept — promote to `docs/invariants/<module>.md` via
      `/protected-surface-amend` in a separate PR.
- [ ] Reject — <one-line reason>
- [ ] Defer — <one-line note; e.g. "revisit after I5b lands">
```

If you could only find 1 or 2 plausible proposals, emit that many and state
explicitly that you found fewer than 3 rather than padding.

If you found **zero** non-obvious candidates, say so plainly. Zero proposals
is a legitimate outcome and is more honest than low-signal filler.

### Step 5: Log accepted proposals to the tracker

Remind the user that the tracker file `.assurance/spec-adversary-tracker.md`
records every run. After the human fills in the triage blocks and the PR
merges, append a new section using this template:

```
## <YYYY-MM-DD> — <module>

Proposed: 3
Accepted: 1
Rejected: 1
Deferred: 1

### Accepted
- <short invariant name> — will be promoted in follow-up PR.

### Rejected
- <short invariant name> — <reason>.

### Deferred
- <short invariant name> — <revisit condition>.
```

The skill itself does NOT modify `docs/invariants/<module>.md`. Promotion is a
separate PR (see Step 6). The tracker is the only file this skill writes to,
and only after human triage.

### Step 6: Remind the user of the promotion path

Close the run with an explicit reminder:

> Accepted proposals do NOT auto-promote. Move each one to
> `docs/invariants/<module>.md` via a **separate PR** that:
>   1. Runs `/protected-surface-amend` for the invariant doc (invariant docs
>      are a Class B protected surface).
>   2. Includes the governance note (rationale, authority, linked roadmap
>      item) in the PR description.
>   3. Requires human review AND `pr-self-review` (or equivalent) — the
>      adversary's output is not self-certifying.

### Step 7: Kill criteria (documented so users know when to stop trusting it)

This skill is Layer 6 best-effort. Track its signal-to-noise ratio:

- **Signal-to-noise < 1:5 after 4 weeks** (fewer than 1 accepted proposal
  per 5 proposed) → scale back cadence or retire the skill for this module.
- **No ratified proposals land within 8 weeks** → the layer-6 strategy for
  this repo needs rework; the adversary is not earning its review time.

Mention the tracker totals in the run's final output so the user can judge
whether the kill criteria are trending closer.

### Step 8: Verification Checklist

```
- [ ] Invariant doc and module code were both read in full before proposing.
- [ ] Step 2 coverage catalogue was produced and used to avoid duplicates.
- [ ] Proposal count is ≤ 3 (or explicitly fewer, with reason).
- [ ] Every proposal has: category, confidence, code-line evidence, and the
      accept/reject/defer triage block.
- [ ] No proposal restates an existing invariant or falls into the doc's
      "Not covered" scope.
- [ ] Tracker reminder was emitted; promotion path via
      `/protected-surface-amend` was named.
- [ ] Kill-criteria totals were surfaced (or stated as first run, so no
      denominator yet).
```

## Arguments

Target module name (must match `docs/invariants/<module>.md`).

Examples:
- `/spec-adversary queue` — probe `docs/invariants/queue.md` against the
  `queue` module.
- `/spec-adversary scanner` — probe `docs/invariants/scanner.md`.
- `/spec-adversary` — prompt for the module name; do not guess.
