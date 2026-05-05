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
