Goal: 61

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-5, 23.6k tokens in / 4.5k out
- **plan**: succeeded
  - Model: claude-sonnet-4-5, 11.3k tokens in / 2.0k out
  - Files: /workspace/plan.md
- **review**: succeeded
  - Model: claude-sonnet-4-5, 6.2k tokens in / 3.0k out
- **plan_revise**: succeeded
  - Model: claude-sonnet-4-5, 5.8k tokens in / 2.7k out
  - Files: /workspace/plan.md

## Context
- analysis_classification: refactor
- concerns_addressed: 7
- concerns_rejected: 0
- files_involved: ["crosscheck/agents/byfuglien.md","crosscheck/skills/reason/SKILL.md","crosscheck/skills/locate-fault/SKILL.md"]
- intent_gap: aligned
- issue_ref: nicholls-inc/claude-code-marketplace#61
- issue_title: crosscheck: component-correct verification misses end-to-end integration gaps
- plan_revised: true
- plan_step_count: 3
- plan_track: semi-formal
- review_concerns: 1. Step 1 checklist item 'Traced across at least N component boundaries' is unactionable without defining what a 'component boundary' is. Provide a precise definition (e.g., file-to-file boundary, module import boundary) or heuristic.
2. Step 2 assumes callees have readable implementations but doesn't integrate the termination condition from risk mitigation. Clarify: when tracing 2+ files is 'mandatory' but a library call is reached at file boundary, does validation skip or stop? Reconcile Step 2's mandate with the risk mitigation's termination rule.
3. Step 2 and risk mitigation conflict on whether integration validation is mandatory for 2+ files or skippable when reaching library calls. Specify precedence: is the 2-file rule absolute, or do termination conditions override?
4. Test 'Execution trace spanning test' should specify a bug type that proves integration validation works—use a bug that appears correct in moduleA but fails due to violated assumptions at A→B interface (e.g., precondition violation), not just any bug in moduleB.
5. Test 'Interface-only reasoning test' should provide a concrete example (e.g., caller passes x=0 to divide_by(x) where callee requires x != 0) rather than abstract description.
6. [MISSING INVARIANT] For any multi-component analysis, the set of files read during integration validation must be a superset of files containing code on the actual execution path of the failing test. Without this, 'integration validation' doesn't guarantee completeness.
7. [MISSING INVARIANT] When integration validation reaches an unreadable callee (library, extern, proprietary), output must explicitly document the trust boundary and state assumptions about callee behavior. Prevents silent premise gaps.
- review_verdict: revise
- verification_track: semi-formal


# Stage 4 — Implement (with verification artifacts)

You are the implementer. Read the approved plan and execute it
step-by-step. Do exactly what the plan says — no more, no less. The
verification artifacts the plan named are part of "done" — if they
don't pass, this stage fails.

## Read

1. Open `plan.md`. This is the source of truth for what to do.
2. Note the **verification track** declared in the plan. Your
   implementation must produce passing artifacts on that track before
   you finish.

## Execute

### For every plan step

- Make the file edits the step describes.
- Run any relevant local checks (typecheck, lint, fast unit tests) as
  you go. Don't accumulate a giant uncommitted blast radius.
- If a step turns out to be wrong (file doesn't exist, API is
  different from what the plan assumed), **stop and report**. The
  plan/review loop produced a bad plan; surface that, don't silently
  rewrite it.

### Verification artifacts (HARD GATE)

Per the plan's declared track:

- **`formal`** — Write Dafny spec(s) under the path the plan named.
  Call `mcp__plugin_crosscheck_dafny__dafny_verify` and quote the
  exit status verbatim in your output. If verification reports
  unproven obligations, this stage **fails**. Use
  `mcp__plugin_crosscheck_dafny__dafny_compile` if the plan requires
  extracting verified Python/Go.
- **`lightweight`** — Add the contract assertions and property-based
  tests the plan named. Run them locally. Quote the last 30 lines of
  test output verbatim. If any test fails, this stage **fails**.
- **`semi-formal`** — Add the execution-trace anchors / regression
  tests the plan named. Run them. If any regression test fails, this
  stage **fails**.

This is non-negotiable. The downstream verify stage attests coverage,
not correctness — it trusts your artifacts. If you hand it broken
artifacts, the gate is wrong.

### Tests

Add the tests `plan.md → ## Tests / properties to add` listed. Run
the project's full test suite (`npm test`, `pytest`, `go test ./...`,
etc. — look at the repo's CONTRIBUTING / package files to choose).
Tests must pass before you finish.

### Commit

Make commits as you go (one logical commit per plan step is fine, or
one combined commit at the end if changes are tightly coupled). Use
Conventional Commits — this repo's CLAUDE.md notes that `docs:` is
blocked for behavioural artifacts.

The Fabro project config has draft-PR auto-creation enabled, so
committed work surfaces as a PR after the workflow completes.

## Output

Respond with:

1. A bulleted list of plan steps and what you actually did for each.
2. **Verification evidence** — the verbatim Dafny exit status / test
   output tail / regression test output, depending on track.
3. A summary of any deviations from the plan and *why*.

End with one of:

Pass:

```json
{
  "outcome": "succeeded",
  "context_updates": {
    "implementation_complete": true,
    "files_changed": ["path/a", "path/b", "..."],
    "tests_added": ["test_name", "..."],
    "verification_track": "<the track you actually implemented to>",
    "verification_artifact_paths": ["path/to/spec.dfy", "..."],
    "verification_status": "green"
  }
}
```

Fail (artifacts didn't pass):

```json
{
  "outcome": "failed",
  "failure_reason": "<one-paragraph triage suitable for a CI summary, citing the failing artifact and exit status>",
  "context_updates": {
    "files_changed": ["..."],
    "verification_status": "red"
  }
}
```

## Discipline

- The plan is the spec. Drift kills accountability.
- Verification artifacts are part of done. A passing implementation
  with a failing Dafny check is a failed stage, not a partial success.
- If you genuinely cannot finish a step (broken environment, missing
  dependency), commit what you have and emit `outcome: "failed"` with
  the environmental cause named.
