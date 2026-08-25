# REVIEW.md — how changes are reviewed in this repository

Every pull request gets four passes, in this order. Later passes assume the earlier ones
found nothing blocking. A reviewer who runs out of time should say which passes they
completed rather than implying all four.

The tier gate (`scripts/ci/tier-gate.mjs`) must be green before review starts. If it is
red, the change is not ready to review — see `docs/assurance/TIER-LAYER-MAP.md`.

## Pass (a) — bugs and logic

Read the diff for defects that make the code do the wrong thing: off-by-one and boundary
errors, unhandled error paths, incorrect state transitions, wrong operator precedence,
resource leaks, race-prone assumptions, and anything that silently swallows a failure.
For each finding, state a concrete failure scenario — inputs or state in, wrong output or
crash out. A claim with no failure scenario is a Nit at best.

## Pass (b) — security

Look for authorisation and ownership gaps, injection paths (shell, SQL, prompt), unsafe
deserialisation, secrets or tokens committed or logged, over-broad file or network
access, and any weakening of a sandbox. In this repository, pay particular attention to
the Docker isolation flags for the Dafny and Lean containers and to anything that would
let verification input reach a shell.

## Pass (c) — compliance against `spec.md` and `plan.md`

Check the diff against the artefacts the change committed. Does the implementation do
what `spec.md` says, and only that? Does it follow the order of work and the proof
strategy in `plan.md`, or has the plan been quietly abandoned mid-change? Undocumented
scope — a behaviour in the diff that no artefact asked for — is an Important finding,
because it means the audit trail no longer describes the software. Concerns that
`spec.md` explicitly flagged as open are not findings; they are known and recorded.

## Pass (d) — Crosscheck pass

Two questions, both answered explicitly in the review:

1. **Does the change touch a protected surface?** Compare every changed path against
   `.claude/rules/protected-surfaces.md`. If any path matches, the PR must carry the
   governance-note block (`## Protected-Surface Amendment`) naming that file, and the
   reviewer must resolve every `REQUIRES HUMAN VERIFICATION` marker in it before
   approving.
2. **Does the declared tier match the diff?** Apply
   `docs/assurance/TIER-LAYER-MAP.md`: confirm the `Tier: N` line or `tier:N` label is
   present, that the artefacts that tier requires are committed, and that a protected
   path in the diff has forced the Tier 3 floor. A tier declared below the floor is
   always an Important finding, even if the code itself is correct.

## Important vs Nit

**Important** — the change is wrong, unsafe, or undocumented in a way that would matter
after merge. Anything in passes (a)–(d) with a concrete consequence: a reachable bug, a
security weakness, a divergence from `spec.md` or `plan.md`, a missing governance note, a
mis-declared tier. Important findings block approval until resolved or explicitly
accepted by the author with a reason recorded in the PR.

**Nit** — style, naming, phrasing, or preference. Correct as written, could be nicer.
Nits never block approval and the author may decline them without justification.

**Nit cap.** Report at most **five** nits per review. Summarise the remainder as a count,
for example: "5 nits below, plus 11 further nits (naming and comment wording) not
itemised." The cap exists so that Important findings are not buried; it is not a licence
to promote nits to Important in order to report more of them.

## Excluded paths

Do not review, and do not raise findings in, generated or vendored files:

```
crosscheck/mcp-server/dist/**
package-lock.json
CHANGELOG.md
```

Also excluded: any other generated output — build directories, lockfiles, release notes
produced by tooling, and `.assurance/` tracker files written by skills rather than by
hand. If a generated file looks wrong, the finding belongs on the generator, not on the
output.

## If you are unsure

Ask the Crosscheck maintainers via a GitHub issue on this repository. A review that
records an honest "I could not judge pass (b) here" is more useful than one that
approves by silence.
