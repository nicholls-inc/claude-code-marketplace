Goal: 140

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-5, 30.5k tokens in / 4.1k out
- **plan**: succeeded
  - Model: claude-sonnet-4-5, 32.0k tokens in / 3.7k out
  - Files: /workspace/plan.md
- **review**: succeeded
  - Model: claude-sonnet-4-5, 19.9k tokens in / 6.4k out
- **plan_revise**: succeeded
  - Model: claude-sonnet-4-5, 10.6k tokens in / 5.5k out
  - Files: /workspace/plan.md

## Context
- analysis_classification: algorithmic
- concerns_addressed: 15
- concerns_rejected: 0
- files_involved: ["crosscheck/skills/spec-adversary/SKILL.md","crosscheck/skills/assurance-init/SKILL.md","crosscheck/skills/invariant-coverage-scaffold/SKILL.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/README.md","crosscheck/demo/06_test_adequacy/SCRIPT.md"]
- intent_gap: aligned
- issue_ref: nicholls-inc/claude-code-marketplace#140
- issue_title: crosscheck: assurance-probe — deterministic test-strength layer (design discussion)
- plan_revised: true
- plan_step_count: 12
- plan_track: formal
- review_concerns: 1. Hidden assumption (Step 2, line 17): Define the grammar/pattern for parseable 'Failure condition' clauses, or cite example invariant docs that demonstrate the expected format.
2. Hidden assumption (Step 3, line 23-24): Specify behavior when coverage tooling (pytest-cov) is absent — fail-fast with actionable error or auto-install?
3. Hidden assumption (Step 5, line 38, 111): Define 'bit-identical on same commit' — which variables are locked (git SHA, Python version, dependencies, OS)? Add explicit environment capture to reproducer template.
4. Hidden assumption (Step 11, line 71): Define 'rotation-based' operationally — who triggers the probe, how often, via what mechanism (manual, cron, /assurance-status recommendation)?
5. Hidden assumption (Verification line 119-122): Tracker integrity property assumes single writer. Address concurrent execution: prevent it, or prove append-only semantics under contention.
6. Missing edge cases: Handle empty/missing Failure condition clauses (skip, warn, or error?), zero-invariant modules, zero-finding runs (SNR 0/0), test framework syntax errors vs execution errors, reproducer scripts referencing rebased-away commits.
7. Test adequacy (line 81-84): Mutation parser test 'determinism' is tautological. Add correctness oracle — reference table mapping Failure condition examples to expected mutations.
8. Test adequacy (line 90-92): Reproducer integration test should include negative case — run reproducer on different commit or with mutation reverted, assert output differs.
9. Test adequacy (line 94-98): E2E test must use real executable Python test + real killable mutation, not mocked/scaffolded stubs (verify this is not test theatre).
10. Track terminology (line 3): Clarify 'formal' means 'Layer 4 deterministic property testing', not 'Layer 1 Dafny proof', to avoid confusion.
11. Reversibility (Step 11): Document byfuglien.md routing addition as additive-only (no modification of existing routes) to preserve backward compatibility if skill is retired.
12. Reversibility (Steps 8-9): Add risk register note that README/assurance-hierarchy.md updates should be marked experimental (e.g., 'Layer 4 (Phase 1)') until SNR ≥ 1:3 demonstrated.
13. Reversibility (line 119-122): Add backup/restore or checksum validation to tracker CSV writes to recover from corruption.
14. Missing invariant (ACCEPT): Mutation soundness — every generated mutation must violate a Failure condition reachable by the covering test's input generator. Address in Step 2 or risk register as known Phase 1 limitation (Phase 3 generator probe partially mitigates).
15. Missing invariant (ACCEPT): Reproducer environment capture — template must verify Python version, dependency versions, emit clear error on mismatch. Add to Step 5 spec.
- review_verdict: revise
- verification_track: formal


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
