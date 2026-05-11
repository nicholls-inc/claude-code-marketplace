---
name: draft-invariants
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

Use this format exactly:

```
**IN. Name — short imperative.**
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

### 7. Translate to property tests

Only after sign-off, generate property tests. Detect the language from the
repo and use its idiomatic framework:

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

### 8. Governance

Output a concrete governance plan covering all four items:

1. **Spec location.** Default: `docs/invariants/<module>.md`. Adjust if the
   repo has an existing `docs/` convention (check `docs/architecture/`,
   `docs/adr/`).
2. **Test location.** Same package as the module, in `*_invariants_test.go`
   / `test_<module>_invariants.py` / equivalent.
3. **Protected surfaces.** Search the repo for:
   - `.claude/rules/protected-surfaces.md` — append the spec doc and test
     files (Class B per the standard partition).
   - `CODEOWNERS` — add entries requiring human review for the spec and
     tests.
   - `.github/branch-protection` / ruleset configs.
   If none exist, recommend running `/crosscheck:assurance-init` first, or
   creating `.claude/rules/protected-surfaces.md` with the standard block
   listing both paths.
4. **CI enforcement.** Confirm the property tests run in CI. If CI uses
   `go test ./...` / `pytest` / equivalent they are already covered —
   verify and say so. If not, recommend running
   `/crosscheck:invariant-coverage-scaffold` to install the bidirectional
   invariant ↔ test coverage gate.

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
      were pre-filled with file:line citations
- [ ] Step 1b (cold elicitation) used only when 1a found no spec
- [ ] Each invariant cites a real failure (incident, audit-finding ID) or
      is explicitly marked speculative
- [ ] 5-8 invariants, each one short paragraph
- [ ] "Not covered" section names the owning layer for each omission
- [ ] Gap analysis references file:line locations
- [ ] User has explicitly signed off on English before any test code is
      written
- [ ] Each property test carries a `// Invariant I<N>: <Name>` comment
      matching the spec heading (bidirectional link enforced by
      `/crosscheck:invariant-coverage-scaffold`)
- [ ] Governance section specifies spec path, test path, protected-surface
      mechanism, and CI path
