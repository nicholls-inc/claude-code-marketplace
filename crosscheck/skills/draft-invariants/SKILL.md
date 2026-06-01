---
name: draft-invariants
add-mode: bootstrap
description: >-
  Contract-first methodology for drafting module-level invariants and property
  tests. Reads the project's prose spec first when one exists; falls back to
  user elicitation only when no spec is found. Anchors each invariant in real
  past failures (incidents, audit-findings), gap-analyses against the code, and
  only translates to tests after explicit English sign-off. Triggers: "draft
  invariants for X", "invariants for module X", "property tests for X",
  "formalize the contract of X", "lock down X against drift", "what must X
  guarantee".
argument-hint: "<module> (will populate docs/invariants/<module>.md)"
---

# /draft-invariants — Module Invariant Drafting (spec-aware)

## Purpose

Produce a small set (5-8) of precisely stated, independently testable
module-level invariants for a target module, plus property tests that enforce
them and a governance plan that prevents agents from silently modifying the
spec. Invariants describe what the module MUST guarantee to callers, in the
domain language of the module — not what the current implementation happens to
do.

This skill is the canonical entrypoint named by `/crosscheck:assurance-init`
Step 2 Q3 ("seed modules"). It is the bridge between a ratified prose spec
(or a verbal contract) and the property-test layer, producing the
`docs/invariants/<module>.md` artefact that downstream layer-5 and layer-6
skills (`/crosscheck:intent-check`, `/crosscheck:spec-adversary`) consume.

## When to invoke

Trigger phrases: "draft invariants", "invariants for <module>", "property
tests for <module>", "formalize the contract of <module>", "lock down <module>
against drift", "what must <module> guarantee".

Typical invocation: `/crosscheck:draft-invariants <module>` with the user in
the target repo.

## The core anti-pattern (DO NOT DO THIS)

**Read the implementation first and extract invariants from it.** This
produces tests that encode current behaviour, not intended contract. It is
the vibe-coding failure mode this skill exists to prevent. If you catch
yourself opening the module's `.go` / `.py` / `.ts` files before completing
steps 1-3, STOP and restart from step 1.

**Note on prose specs.** A written prose spec (for example
`docs/design/<repo>-spec.md`, an RFC-2119-keyworded design doc, or a similar
architecture document) is **not** the implementation. Specs are the
externalised contract — the highest-trust source for Step 1 and the default
starting point. Reading and citing the spec is required before any
elicitation. Reading code remains prohibited until Step 4.

Other anti-patterns:
- Writing invariants without user red-pen review.
- Translating to tests before English sign-off.
- Invariants longer than one short paragraph.
- Invariants that reference private field names or internal helpers.
- "The function returns X when Y" phrasing (that's a unit test, not an
  invariant).
- Generating AskUserQuestion options that paraphrase the spec instead of
  quoting it. If the user must confirm a contract framing, options must be
  **verbatim spec snippets with file:line citations**, plus at most one
  "synthesis" option — never reworded paraphrases competing with the spec's
  own language.

## Methodology (execute in order)

### 1. Establish the contract

#### 1a. Spec discovery (do this FIRST, before any elicitation)

Before asking the user anything, search for an externalised contract:

- Glob common spec locations:
  - `docs/design/*spec*.md`, `docs/design/*.md`
  - `docs/specs/*.md`, `docs/SPEC.md`, `SPEC.md`
  - `docs/architecture/*.md`, `docs/adr/*.md`
  - `docs/<module>.md`
  - `RFC*.md`, `docs/RFC*.md`
- Detect RFC-2119 keyword presence (`MUST`, `MUST NOT`, `SHALL`, `SHOULD`,
  `SHOULD NOT`, `MAY`). A non-trivial count of these keywords indicates a
  ratified prose spec.
- Look for an audit-finding-to-section traceability table (often a §14 or
  appendix in larger specs — for example, "audit-finding to spec-section
  mapping"). This table is the canonical catastrophe corpus for Step 2.
- Check the issue tracker for spec-decomposition issues:
  `gh issue list --search <module>` and look for issues that label themselves
  as deriving from a spec section.

If a spec is found:

1. **Read the section(s) covering this module in full.** Not just the table
   of contents. Fetch concrete section text including any audit findings
   cited.
2. **Read the audit-finding traceability table in full** if one exists.
3. **Pre-fill candidate answers** to the four contract questions in 1b from
   the spec, citing exact `<file>:<lines>` references for each. Do not
   paraphrase — quote.
4. **Present the pre-filled answers to the user** as a confirmation pass:
   *"I've extracted the following from `<spec-path>:<lines>`. Confirm,
   red-pen, or extend."* Do **not** elicit cold.
5. Take user-supplied corrections as authoritative over the spec only when
   the user explicitly states the spec is wrong (in which case flag this as
   a spec amendment under `/crosscheck:protected-surface-amend` for a
   separate PR — do not silently let invariants override prose).

The spec is the authoritative contract. The user's verbal answers are the
fallback for repos without one.

**Refuse-and-redirect rule.** If §1a discovers a spec and the skill would
otherwise issue cold `AskUserQuestion` paraphrasing the spec, the skill
MUST instead emit pre-filled candidates with `<file>:<lines>` citations as
a single confirmation pass. Paraphrasing options for multi-select is a
§3.5a violation per the v2 ADD retrospective (see
`crosscheck/docs/add/.retrospective/findings-and-methodology-v2.md`
§3.5a Case B, lines 297–303) — fixed-cadence elicitation cold against
context that already has the answers is exactly the failure mode this
skill was rewritten to prevent. If the skill cannot construct a confirm
pass from the spec (e.g. the spec genuinely does not address one of the
contract questions), it MAY ask a single targeted question for that
specific gap; it MUST NOT issue a generic interview when reading has
already answered most of it.

#### 1b. Cold elicitation (only if 1a found nothing)

If no spec exists, ask the user:

- What is this module responsible for, in one sentence?
- Who are its callers, and what do they rely on?
- What would be catastrophic if it broke silently?
- What properties must hold regardless of call order, concurrency,
  crash/restart?

You MAY skim public API signatures (exported function names, type
definitions) and docstrings. You MAY NOT open function bodies or private
files.

If the user cannot answer, interview. Do not guess.

#### 1c. Orchestrator marker mode (deferred red-pen)

This skill may be invoked directly by a user or dispatched by the
`add-orchestrator` agent as part of the spec-driven fast path. When
dispatched by the orchestrator, the skill's `§6` red-pen loop is
deferred — the orchestrator owns a batched cross-module review
downstream and per-module red-pen would duplicate that gate.

The hand-off is mediated by a content-hashed **marker file**, not a
caller-supplied flag. A flag would be forgeable from a user prompt; a
marker file with a content hash over fixed inputs is at least a
consistency check (see *Coordination mechanism, not tamper-resistance*
below).

**Detection.** Look for `.assurance/add-session-*/session.json` in the
current working directory or any ancestor directory. If found, read its
JSON. The marker has the shape (see also
`crosscheck/agents/add-orchestrator.md` for the canonical schema):

```json
{
  "session_id": "<16-hex-char nonce>",
  "created_at": "YYYY-MM-DDTHH:MM:SSZ",
  "spec_path": "<path>",
  "modules": ["<m1>", "<m2>", ...],
  "hash_inputs": ["<spec_path>", "<glossary_path>", "<module_map_path>"],
  "hash_algorithm": "sha256",
  "hash_value": "<lowercase-hex-sha256>",
  "hash_discipline_ref": "crosscheck/skills/intent-check/references/attestation-schema.md"
}
```

**Validation.** When a marker is detected:

1. Confirm the target module appears in the marker's `modules` array. If
   not, treat as marker-absent (the marker is scoped to a different
   session); fall through to the standard §6 red-pen.
2. Recompute the content hash over `hash_inputs` using the discipline
   documented at
   `crosscheck/skills/intent-check/references/attestation-schema.md`
   lines 76–92 (sorted absolute paths, raw bytes concatenated with no
   delimiter, single SHA-256, lowercase hex). If the recomputed hash
   does NOT match `hash_value`, refuse with a clear error: *"Marker hash
   mismatch — the spec, glossary, or module-map has changed since the
   orchestrator pre-flight. Re-run `add-orchestrator` to regenerate the
   marker."*
3. Confirm the marker JSON parses cleanly and contains all required
   fields. If malformed, refuse with: *"Marker file malformed —
   delete `.assurance/add-session-<id>/session.json` and re-run
   `add-orchestrator`."*

**Effect (when marker is valid).** Suppress the §6 in-skill red-pen
loop. Generate the invariant doc as usual but write:

- `Status: Draft` (unchanged from standard behaviour — the Status
  taxonomy is preserved across the v2 ADD retrospective §3.3
  vocabulary)
- An additional line `Audit: pending session <session_id>` recording
  the marker session

When the orchestrator subsequently applies findings (step 10 of its
workflow), it rewrites the `Audit:` line to `applied session
<session_id> on <YYYY-MM-DD>`. The `Status:` value remains `Draft`
until the human approves the PR; at that point a separate edit can
flip it to `Status: Snapshot` per existing convention.

**Effect (when marker is absent).** The §6 red-pen loop is mandatory
as before. There is no caller-supplied flag to bypass it. Direct users
of `/draft-invariants` get the unchanged v1 behaviour.

**Coordination mechanism, not tamper-resistance.** The marker file is
a coordination mechanism between this skill and the orchestrator. It
ensures the in-skill red-pen and the orchestrator's red-pen do not
both fire on the same module, and it catches typo'd manual marker
files via the hash check. It is **NOT** a security boundary — sub-
agents share filesystem and tool-permission scope with the parent
agent, so a determined actor (malicious user prompt, compromised
sub-agent) can write a forged marker. Future maintainers should not
treat the marker as a tamper-resistant attestation. The actual safety
net is the standard `§6` red-pen, which fires whenever the marker is
absent. Compare `/intent-check`'s `.assurance/intent-check-attestation.json`,
which uses the same hash discipline but for a different purpose (a
fast pre-commit check that an LLM pipeline actually ran).

**Sign-off semantics.** The checklist item "User has explicitly signed
off on English before any test code is written" (see the Checklist
section below) is satisfied by EITHER the in-skill §6 red-pen OR by
the orchestrator's downstream batched review of the per-category
findings files. The orchestrator's apply step (step 10 of its
workflow) records the sign-off via the `Audit:` line update; that line
is the durable evidence the obligation transferred. If a downstream
consumer (e.g. `/invariant-coverage-scaffold`) needs to verify
sign-off, it can read the `Audit:` line plus the orchestrator's
`triage-log.md` in the same `.assurance/add-session-<id>/` directory.

### 2. Anchor in real failures

Search the repo for past incidents touching this module. Look for:

- **Audit-finding traceability tables in the spec** (highest priority if one
  was found in 1a — every audit ID listed there is a real failure class to
  anchor on).
- Issue tracker history (`gh issue list --search <module>`, closed issues,
  postmortems).
- Recent merged PRs that fixed bugs in the module.
- `CLAUDE.md`, `AGENTS.md`, memory files, `docs/postmortems/`,
  `docs/incidents/`.
- Commit messages with "fix:", "bug:", "regression" mentioning the module.
- TODO / FIXME / XXX comments in adjacent modules referencing this one.

Each candidate invariant should ideally cite at least one real failure it
would have prevented. If you cannot cite one, flag it as "speculative" for
the user to confirm.

### 3. Draft candidate invariants (5-8)

**Heading convention (load-bearing — do NOT vary).** Each invariant MUST be introduced by a level-2 heading of the exact form `## I<N>: <Name>`, where `<N>` is a stable monotonic integer (`I1`, `I2`, …) and `<Name>` is a short PascalCase or kebab-style identifier. The heading is the grep anchor that downstream tooling (`/invariant-coverage-scaffold`, `// Invariant Ix: <Name>` test comments) relies on. Examples that are correct: `## I1: RedactionSentinelString`, `## I7: BothSetFileWins`. Examples that are WRONG and MUST NOT be produced: `### I1 RedactionSentinelString` (h3, not h2), `## Engine selection — single embedding API` (prose-section heading with the ID buried in body), `**I1. RedactionSentinelString.**` (bold-prefix in paragraph; this was the legacy v1 style and is grandfathered for already-shipped docs only). Group invariants under a single `## Invariants` parent only if you need to use `### I<N>: <Name>` (h3) consistently for every invariant — never mix h2 and h3 invariant headings in the same doc.

When dispatched by `add-orchestrator` with the orchestrator marker active, every invariant heading in this doc MUST be h2 (`## I<N>: <Name>`). The heading-convention discipline is enforced at the per-subagent quality gate in `add-orchestrator.md` Step 6.

Use this body format exactly under each heading:

```
## I<N>: <Name>

<One or two sentence statement of the property in plain English.>

- *Why:* <real failure class this blocks, tied to specific past incident or audit-finding ID where possible>
- *Test:* <concrete sketch of how to translate this into a property-based or fuzz test>
```

Each invariant MUST be:
- Precisely stated (no weasel words like "usually", "generally")
- Independently testable (can be violated by itself, without another
  invariant also breaking)
- About system-level properties visible to callers, not implementation
  details
- Phrased in the domain language of the module (use the module's nouns)
- One short paragraph — if you need three sentences to state it, it's two
  invariants

### 4. Gap-analyse against implementation

NOW, and only now, read the code. For each invariant, identify specific
sites where the current implementation could violate it. Cite files and line
ranges. This gap analysis is output alongside the invariants — it becomes
the fix backlog.

Format:

```
### Gap analysis

**I1:** potentially violated at `<path>:<lines>` — <one-line reason>.
**I2:** holds; defended by <code location>.
...
```

### 5. Scope honesty

Explicitly list what the invariants do NOT cover. For each omission, name
the layer that property belongs to. Example: "Liveness (drain eventually
makes progress) is not covered here — it belongs to the scheduler, not the
queue."

### 6. Red-pen loop

Present invariants, gap analysis, and scope section to the user as prose.
Do NOT generate tests yet. Wait for the user to:
- Strike invariants they disagree with
- Reword unclear statements
- Add failure anchors you missed
- Sign off explicitly ("ship it", "looks good", "proceed to tests")

Iterate until sign-off.

**Marker-mode skip.** If §1c detected a valid orchestrator marker for
this module, the §6 red-pen loop is deferred — the orchestrator owns a
batched cross-module review downstream. In marker mode, after Step 5
emit the invariant doc with `Status: Draft` plus `Audit: pending
session <session_id>`; skip directly to closing without generating
property tests in Step 7. The orchestrator will dispatch step 7
(property tests) separately after its batched audit completes and the
user signs off the findings files.

### 7. Translate to property tests

Only after sign-off, generate property tests. **In marker mode this step
does not run** — the orchestrator dispatches it after its batched audit
and findings triage. In standard mode, sign-off is the explicit "ship
it" / "looks good" from §6; in marker mode the sign-off is recorded
downstream in `.assurance/add-session-<id>/triage-log.md` and the
`Audit: applied …` line is the durable evidence.

Detect the language from the repo and use its idiomatic framework:

- **Go:** `pgregory.net/rapid` — `func TestProp<Name>(t *testing.T) { rapid.Check(t, func(t *rapid.T) { ... }) }`, file suffix `_prop_test.go` or `_invariants_test.go`
- **Python:** `hypothesis` — `@given(...)` on test functions, stateful tests via `RuleBasedStateMachine`
- **Rust:** `proptest` — `proptest! { #[test] fn ... }` macro
- **TypeScript/JavaScript:** `fast-check` — `fc.assert(fc.property(...))`
- **Haskell:** `QuickCheck`
- **Java/Kotlin:** `jqwik`
- **Ruby:** `rantly` or `minitest-hypothesis`

If the repo already has property tests for other modules, mirror their
style and directory layout. Each test MUST carry a comment `// Invariant
IN: <Name>` linking it back to the spec — this is the bidirectional link
that `/crosscheck:invariant-coverage-scaffold` enforces in CI and
pre-commit.

### 8. Governance — emit artifacts, not user instructions

Inspect repo state directly and emit two agent-authored artifacts. Do not
ask the user, and do not list "run skill X" instructions — orchestrators
or human PR reviewers consume what this step writes.

**8a. PR-pasteable governance fragment.** Emit a markdown fragment named
`docs/invariants/<module>.governance-fragment.md` (or append to the main
output document under a `## Governance` heading, depending on the output
mode in `## Output structure`). The fragment is filled from repo
inspection — every line cites a file path. Schema:

```markdown
## Governance

**Spec location:** docs/invariants/<module>.md  (or repo-specific override
detected at <evidence path>)
**Test location:** <detected path>  (matched against <pattern>)
**Protected-surfaces classification:** Class B  (per
.claude/rules/protected-surfaces.md:<line>; if the rule file is absent,
this row reads "MISSING — protected-surfaces.md must be scaffolded; see
.assurance/draft-invariants-<module>-protected-surfaces.patch for the
proposed entry").
**CODEOWNERS coverage:** <detected entry or "MISSING — see patch artifact">.
**CI enforcement:** <detected CI workflow path + the test command that
runs the property tests, or "MISSING — property tests are not exercised
by any detected CI job; install /crosscheck:invariant-coverage-scaffold's
gate to fix">.
```

The fragment is human-reviewable in the PR alongside the invariant doc
itself; it does not require the user to take separate action.

**8b. protected-surfaces patch artifact (only when needed).** If
`.claude/rules/protected-surfaces.md` does not list the new invariant doc
or the matching property-test glob under Class B, write a patch artifact
at `.assurance/draft-invariants-<module>-protected-surfaces.patch`
containing the exact diff that would add the entries. Schema:

```diff
--- a/.claude/rules/protected-surfaces.md
+++ b/.claude/rules/protected-surfaces.md
@@ <hunk header> @@
 ## Class B — Module invariant specifications and tests
 ...
+- `docs/invariants/<module>.md`
+- `<repo-test-glob-for-module>`
```

If the rule file itself is missing entirely, write the patch as a full
new-file proposal listing the standard two-class partition, plus the new
entries. Either way, the patch is an artifact the PR reviewer applies (or
an orchestrator dispatches via `/protected-surface-amend` for the
Class B amendment) — **not** an instruction telling the user to do it
themselves.

**Repo inspection rules.** When pre-filling the fragment, derive every
field from observation:

- Spec location: default `docs/invariants/<module>.md`; override if the
  repo's `docs/` layout already shows a different convention
  (`docs/architecture/<module>/invariants.md`, etc.) — cite the evidence.
- Test location: pattern-match by language (`*_invariants_test.go`,
  `test_<module>_invariants.py`, `<module>.invariants.spec.ts`), then
  verify by grep that the file actually exists. If multiple candidates
  match, list all and flag the ambiguity in the fragment.
- CODEOWNERS: read `CODEOWNERS` (root, `.github/`, or `docs/`); match the
  spec and test paths against existing rules; emit MISSING when absent.
- CI enforcement: read `.github/workflows/*.yml` /
  `.gitlab-ci.yml` / `.circleci/config.yml`; identify any job that runs
  the language's standard test command (`go test`, `pytest`, etc.);
  confirm coverage and cite the workflow path + job name. If no covering
  job, mark MISSING and reference the patch artifact (`8b`) for the
  amendment route.

A missing protected-surfaces entry, missing CODEOWNERS entry, or missing
CI coverage is **not** a hard stop — it is a finding the artifact
surfaces. The downstream PR review decides whether to land the patch in
the same PR, in a follow-up, or to accept the gap with a documented
reason.

## Output structure

Deliver a single markdown document to the user with these sections in this
order:

```
# Invariants: <module>

## Contract (summary from step 1)
<2-4 sentences; if from a spec, cite the section>

## Invariants
**I1. ...**
**I2. ...**
...

## Not covered
- <property> — belongs to <layer>.
...

## Gap analysis
**I1:** ...
...

## Governance
- Spec: ...
- Tests: ...
- Protected: ...
- CI: ...
```

After sign-off, add a section `## Property tests` with the generated test
code, or write them directly to the test file if the user prefers.

## Gold-standard reference example

This is the quality bar. An invariant should read like this:

> **I1. At-most-one active per ref.**
> For any non-empty `Ref`, at most one vessel is in `{pending, running,
> waiting}` at any time. Enqueue of an already-active ref is a no-op.
> - *Why:* prevents the double-dispatch class of bugs (filed issue #541 for
>   AND-match leaking duplicates; loop 239 cancelled two duplicates;
>   audit-finding I9 in `docs/design/queue-spec.md:842`).
> - *Test:* rapid sequence of arbitrary Enqueue/Update/Cancel ops
>   interleaved; after each step, assert |active vessels with ref == R| ≤ 1
>   for every R.

Note: domain nouns (`Ref`, `vessel`, state names), precise quantifier ("at
most one"), specific past incidents (`#541`, `loop 239`, audit-ID `I9`),
testable sketch.

## Checklist before handing off

- [ ] Step 1a (spec discovery) executed before any AskUserQuestion call
- [ ] If a spec was found, the relevant section was read in full and answers
      were pre-filled with file:line citations (no paraphrasing options for
      multi-select)
- [ ] Step 1b (cold elicitation) used only when 1a found no spec
- [ ] Step 1c (orchestrator marker detection) executed; marker validated or
      explicitly stated absent
- [ ] Each invariant cites a real failure (incident, audit-finding ID) or
      is explicitly marked speculative
- [ ] 5-8 invariants, each one short paragraph
- [ ] "Not covered" section names the owning layer for each omission
- [ ] Gap analysis references file:line locations
- [ ] User has explicitly signed off on English before any test code is
      written — satisfied by EITHER §6 in-skill red-pen OR a valid marker
      file plus the orchestrator's downstream triage being signed off
      (recorded in `.assurance/add-session-<id>/triage-log.md` and via the
      `Audit: applied …` line on the invariant doc)
- [ ] Status line is `Status: Draft` (standard mode) or `Status: Draft` +
      `Audit: pending session <id>` (marker mode); the `Status:` taxonomy
      from retrospective §3.3 is preserved
- [ ] Each property test carries a `// Invariant I<N>: <Name>` comment
      matching the spec heading (bidirectional link enforced by
      `/crosscheck:invariant-coverage-scaffold`) — only emitted in standard
      mode; in marker mode the orchestrator dispatches property-test
      generation separately
- [ ] Governance section specifies spec path, test path, protected-surface
      mechanism, and CI path
