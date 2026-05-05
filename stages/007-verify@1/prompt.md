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
- **implement**: succeeded
  - Model: claude-sonnet-4-6, 43.4k tokens in / 17.0k out
  - Files: /workspace/crosscheck/README.md, /workspace/crosscheck/agents/hellebuyck.md, /workspace/crosscheck/docs/assurance-hierarchy.md, /workspace/crosscheck/docs/skills.md, /workspace/crosscheck/skills/assurance-probe/SKILL.md, /workspace/crosscheck/skills/assurance-probe/verify.sh

## Context
- analysis_classification: refactor
- concerns_addressed: 4
- concerns_rejected: 0
- files_changed: ["crosscheck/skills/assurance-probe/SKILL.md","crosscheck/skills/assurance-probe/verify.sh","crosscheck/agents/hellebuyck.md","crosscheck/docs/skills.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/README.md"]
- files_involved: ["crosscheck/skills/spec-adversary/SKILL.md","crosscheck/skills/invariant-coverage-scaffold/SKILL.md","crosscheck/skills/intent-check/SKILL.md","crosscheck/skills/assurance-status/SKILL.md","crosscheck/agents/hellebuyck.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/docs/skills.md","crosscheck/README.md","crosscheck/.claude-plugin/plugin.json"]
- implementation_complete: true
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
- tests_added: ["crosscheck/skills/assurance-probe/verify.sh (20 trace checks: rubric-determinism ×2, aspirational-exclusion ×2, zero-assertion ×2, weakest-wins ×3, unonboarded-gate ×3, patch-comparison ×2, trigger-overlap ×6)"]
- verification_artifact_paths: ["crosscheck/skills/assurance-probe/verify.sh"]
- verification_status: green
- verification_track: semi-formal


# Stage 5 — Verify · coverage attestation (ONE-SHOT)

You are the **byfuglien** crosschecking enforcer at the gate. Your
job is narrow: confirm that the verification artifacts the implementer
produced **actually cover the change** and **are green**.

You do not re-classify. You do not re-derive. You do not re-run intent
analysis (that lived at review time). You attest.

## You get exactly ONE chance.

- This is your only verification pass.
- If you flag failures, the `fix` stage gets **one** attempt to address
  what you list, and then the workflow exits to a PR. There is no
  second verify.
- **Frontload everything.** Run **all three checks** even if Check 1
  fails — do not short-circuit. The fix stage works one-shot off your
  list, and whatever you don't surface reaches the PR unflagged.
- Every concern must be self-contained: file path, exact failure,
  enough context that the fix stage can act without re-reading verify's
  reasoning chain.

## Read

1. The implement stage's response (in your preamble) — note
   `verification_track`, `verification_artifact_paths`,
   `verification_status`, `files_changed`.
2. `plan.md` — the spec the implementer worked to.
3. The artifacts themselves: the `.dfy` files / property tests /
   regression tests the implementer named.
4. The actual diff (`git diff` on `files_changed`) — you need to know
   what surface changed.

## Three checks. Run all three. Collect every failure.

### Check 1 — Artifacts exist

For each path in `verification_artifact_paths`: does the file exist
on disk? List every missing artifact. Do not stop at the first.

### Check 2 — Artifacts are green

- **`formal`** — Re-run `mcp__plugin_crosscheck_dafny__dafny_verify`
  on each `.dfy` file. Quote each exit status. List every artifact
  that reports unproven obligations, with the failing assertion's
  file:line.
- **`lightweight` / `semi-formal`** — Run the named tests. Quote the
  last 30 lines of output. List every failing test by name.

If implement reported `verification_status: green` but your re-run is
red, **flag it loudly** in the report — the implementer's self-report
was wrong, and that is exactly the independence check this stage
exists for.

### Check 3 — Artifacts cover the changed surface

This is the only judgment call you make. For each file in
`files_changed`, is there at least one verification artifact that
references it (Dafny spec for the function, property test for the
behaviour, regression test for the path)?

Read 2–3 representative artifacts and confirm each names a function,
type, or property that *actually exists in the diff*. Build a coverage
map: every `files_changed` entry → covering artifact, or `UNCOVERED`.
List every `UNCOVERED` entry and every artifact that points at code
the diff didn't change. This is the "passing Dafny spec for a
different function" trap; it is the one thing the implementer cannot
credibly check on themselves.

## Decide

Pass if all three checks passed clean.
Fail if any check produced one or more failures. List **every**
failure across **all three** checks — not just the first.

## Output

Markdown report with:

1. **Check 1** — artifact existence (every path · ok / missing)
2. **Check 2** — artifact green status (verbatim tool output for each)
3. **Check 3** — coverage map (every `files_changed` entry → covering
   artifact, or `UNCOVERED`)
4. **Verdict** (one line — pass or fail)

End with **exactly one** of these JSON blocks:

Pass:

```json
{
  "outcome": "succeeded",
  "preferred_next_label": "pass",
  "context_updates": {
    "verify_verdict": "pass",
    "verify_evidence": "<one-paragraph summary>"
  }
}
```

Fail:

```json
{
  "outcome": "succeeded",
  "preferred_next_label": "fail",
  "context_updates": {
    "verify_verdict": "fail",
    "verify_concerns": "1. <self-contained failure with file:line>\n2. <…>\n…",
    "verify_evidence": "<one-paragraph summary>"
  }
}
```

(Note: `outcome` is `succeeded` even on verdict-fail. The verify agent
succeeded at its job — it found problems. The run continues to `fix`.)

## Discipline

- A passing artifact you didn't read is not evidence.
- A green Dafny report against the wrong spec is the trap. Check 3
  is the only thing standing between that trap and a merged PR.
- Cite `file:line` for every coverage claim.
- Frontload. There is no second verify. Every failure not on your
  list reaches the PR unflagged.
- This stage is thin by design. If you find yourself re-doing the
  implementer's reasoning, stop — that's drift back into the old
  parallel-verify shape.
