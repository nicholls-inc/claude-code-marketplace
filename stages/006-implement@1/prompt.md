Goal: 140

## Completed stages
- **analyze**: succeeded
  - Model: claude-sonnet-4-6, 48.0k tokens in / 3.9k out
- **plan**: succeeded
  - Model: claude-sonnet-4-6, 32.7k tokens in / 2.6k out
  - Files: /workspace/plan.md
- **review**: succeeded
  - Model: claude-sonnet-4-6, 33.8k tokens in / 3.9k out
- **plan_revise**: succeeded
  - Model: claude-sonnet-4-6, 24.6k tokens in / 4.5k out
  - Files: /workspace/plan.md

## Context
- analysis_classification: refactor
- concerns_addressed: 4
- concerns_rejected: 0
- files_involved: ["crosscheck/skills/spec-adversary/SKILL.md","crosscheck/skills/invariant-coverage-scaffold/SKILL.md","crosscheck/skills/intent-check/SKILL.md","crosscheck/skills/assurance-status/SKILL.md","crosscheck/agents/hellebuyck.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/docs/skills.md","crosscheck/README.md","crosscheck/.claude-plugin/plugin.json"]
- intent_gap: aligned
- issue_ref: nicholls-inc/claude-code-marketplace#140
- issue_title: crosscheck: assurance-probe — deterministic test-strength layer (design discussion)
- plan_revised: true
- plan_step_count: 5
- plan_track: semi-formal
- review_concerns: 1. The SKILL.md for /assurance-probe must specify whether it requires the hellebuyck onboarding gate (docs/invariants/ present, ROADMAP, protected-surfaces rules) or explicitly documents what it emits on an unonboarded repo. Without this, a user on an unonboarded repo receives a silently empty strength table, which is indistinguishable from 'all tests are strong.' Every other hellebuyck skill either enforces the gate or documents the exception — /assurance-probe must do the same. (Probe 1, Missing invariant 1)
2. The SKILL.md must state a deterministic aggregation rule for the case where a single invariant ID is covered by more than one test file. Without a specified rule (weakest / strongest / list-all-separately), two runs reading files in different orders can produce different strength tables, directly violating the plan's claim of determinism. (Probe 1, Missing invariant 2)
3. Every rubric dimension must be operationalized as specific, grep-able, language-agnostic keywords or structural markers in the SKILL.md. 'Mutation probe hint' is named as a dimension but never concretized (e.g., presence of 'mutmut', 'pitest', 'stryker', '#mutant' in the file, or zero score otherwise). A dimension that requires LLM judgment to evaluate is not a deterministic rubric — the plan's own risk register flags this but does not resolve it in the skill design. (Probe 1, Missing invariant 3)
4. The 'patch comparison' verification test for hellebuyck.md only verifies byte-identity of existing rows (catching deletion/mutation), not semantic routing conflicts. The new row's trigger phrases ('test strength', 'how strong are the tests', 'probe invariant coverage', 'weak tests') must be checked against existing trigger signals for semantic overlap — particularly against 'invariant coverage', 'coverage gate', 'scaffold invariant check' (which routes to /invariant-coverage-scaffold) and 'assurance status' patterns. The plan must add a trigger-phrase non-overlap check to the verification section, not just rely on the risk register statement that 'trigger phrases are distinct.' (Probe 3, Concern 5)
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
