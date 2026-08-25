# Gate: auditor verdicts

## What this gate protects

The `auditor` agent runs a read-only "consolidation pass" over a repository's
specs, invariants (the machine-checkable rules a spec's promises are supposed
to enforce, e.g. "a queue's length is never negative"), tests, and journal
entries. For each artefact it renders one verdict — `settled` (no drift),
`active` (legitimate work-in-progress), or `drifted` (a deterministic signal
of divergence that needs a human's attention) — and, for every `drifted`
verdict, a severity and a proposed remediation.

Crucially, `auditor` has no authority to act on its own findings. It writes
one file, its own consolidation report, and nothing else. It does not edit
specs, invariants, tests, or code, and it does not flip a `Status` field on
any artefact it audits. This separation exists because the agent that wrote
an artefact is the wrong agent to judge whether that artefact still matches
its spec — if the same hand authored and audited, the audit becomes
self-confirming. This gate is the point where a human, not the auditor, takes
each proposed remediation and decides whether it happens.

## What you are being asked to decide

For every `drifted` verdict in the report, accept or reject the auditor's
proposed remediation. There is no cap on the number of `drifted` findings in
a single report; work through the table row by row.

## What each decision means

- **Accept** — the remediation is sound. You (or whoever you hand it to)
  route it to the named downstream agent the report specifies — `byfuglien`
  for code-touching fixes, `hellebuyck` for spec re-derivation or intent
  re-attestation, or `lowry` for a run-to-green loop. Only once that
  downstream work actually lands does anyone update the artefact's `Status`
  field — the auditor never does this, and accepting the remediation does
  not do it either.
- **Reject** — the auditor misread the signals, or the divergence is
  acceptable as-is. Record why, so a future pass does not re-raise the same
  finding without that context.

Verdicts of `settled` and `active` carry no remediation and need no
decision — they are informational. One caveat worth knowing: on an artefact
with no tracking metadata (no `consumes:` field, no `add-mode` tag, no
attestation timestamp), several `drifted` criteria simply cannot fire. A
`settled` verdict there means "no signal was computable," not "checked
against everything and found clean" — the report flags this distinction
explicitly wherever it applies.

## Where this shows up

The auditor writes its consolidation report to
`.assurance/audit/<date>-audit.md`. Each report is dated and immutable —
a new pass writes a new file rather than editing an old one, so the
adjudication history for past reports stays intact.

## How long this takes

A few minutes per `drifted` finding. Reports with no `drifted` verdicts need
no action beyond reading them.

## Who to ask if unsure

The Crosscheck maintainers via a GitHub issue on this repository.
