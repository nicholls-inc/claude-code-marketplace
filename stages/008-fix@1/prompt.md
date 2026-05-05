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
- **implement**: succeeded
  - Model: claude-sonnet-4-5, 56.2k tokens in / 38.2k out
  - Files: /workspace/crosscheck/.gitignore, /workspace/crosscheck/README.md, /workspace/crosscheck/agents/byfuglien.md, /workspace/crosscheck/demo/07_test_strength/SCRIPT.md, /workspace/crosscheck/docs/assurance-hierarchy.md, /workspace/crosscheck/skills/assurance-probe/SKILL.md, /workspace/crosscheck/skills/assurance-probe/__init__.py, /workspace/crosscheck/skills/assurance-probe/lib/__init__.py, /workspace/crosscheck/skills/assurance-probe/lib/hypothesis_probe.py, /workspace/crosscheck/skills/assurance-probe/lib/mutations.py, /workspace/crosscheck/skills/assurance-probe/lib/vacuity.py, /workspace/crosscheck/skills/assurance-probe/references/phase-gating.md, /workspace/crosscheck/skills/assurance-probe/specs/mutation_determinism.dfy, /workspace/crosscheck/skills/assurance-probe/templates/issue.md.template, /workspace/crosscheck/skills/assurance-probe/templates/probe-tracker.csv.template, /workspace/crosscheck/skills/assurance-probe/templates/reproducer.py.template, /workspace/crosscheck/skills/assurance-probe/tests/__init__.py, /workspace/crosscheck/skills/assurance-probe/tests/run_basic_tests.py, /workspace/crosscheck/skills/assurance-probe/tests/test_e2e.py, /workspace/crosscheck/skills/assurance-probe/tests/test_mutations.py, /workspace/crosscheck/skills/assurance-probe/tests/test_reproducer.py, /workspace/crosscheck/skills/assurance-probe/tests/test_vacuity.py
- **verify**: succeeded
  - Model: claude-sonnet-4-5, 40.0k tokens in / 9.0k out

## Context
- analysis_classification: algorithmic
- concerns_addressed: 15
- concerns_rejected: 0
- files_changed: ["crosscheck/README.md","crosscheck/agents/byfuglien.md","crosscheck/demo/07_test_strength/SCRIPT.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/skills/assurance-probe/SKILL.md","crosscheck/skills/assurance-probe/__init__.py","crosscheck/skills/assurance-probe/lib/__init__.py","crosscheck/skills/assurance-probe/lib/hypothesis_probe.py","crosscheck/skills/assurance-probe/lib/mutations.py","crosscheck/skills/assurance-probe/lib/vacuity.py","crosscheck/skills/assurance-probe/references/phase-gating.md","crosscheck/skills/assurance-probe/specs/mutation_determinism.dfy","crosscheck/skills/assurance-probe/templates/issue.md.template","crosscheck/skills/assurance-probe/templates/probe-tracker.csv.template","crosscheck/skills/assurance-probe/templates/reproducer.py.template","crosscheck/skills/assurance-probe/tests/__init__.py","crosscheck/skills/assurance-probe/tests/run_basic_tests.py","crosscheck/skills/assurance-probe/tests/test_e2e.py","crosscheck/skills/assurance-probe/tests/test_mutations.py","crosscheck/skills/assurance-probe/tests/test_reproducer.py","crosscheck/skills/assurance-probe/tests/test_vacuity.py","crosscheck/.gitignore"]
- files_involved: ["crosscheck/skills/spec-adversary/SKILL.md","crosscheck/skills/assurance-init/SKILL.md","crosscheck/skills/invariant-coverage-scaffold/SKILL.md","crosscheck/docs/assurance-hierarchy.md","crosscheck/README.md","crosscheck/demo/06_test_adequacy/SCRIPT.md"]
- implementation_complete: true
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
- tests_added: ["test_parse_simple_conditions","test_mutation_generation","test_determinism","test_boundary_mutations","test_real_mutation_killed","test_bounded_output","test_zero_invariant_module","test_tracker_csv_update","test_skipped_count_for_unparseable","test_mutation_soundness","test_reproducer_bit_identical_on_same_commit","test_reproducer_detects_commit_mismatch","test_reproducer_detects_mutation_difference","test_vacuity_prerequisites","test_coverage_measurement","test_probe_vacuous_test","test_probe_load_bearing_test"]
- verification_artifact_paths: ["crosscheck/skills/assurance-probe/specs/mutation_determinism.dfy","crosscheck/skills/assurance-probe/tests/run_basic_tests.py"]
- verification_status: green
- verification_track: formal
- verify_concerns: 1. Dafny verification artifact not executed — `crosscheck/skills/assurance-probe/specs/mutation_determinism.dfy` is listed in `verification_artifact_paths` but cannot be verified (no Dafny/Docker/Node tooling available). Plan states this is Layer 4 property testing (NOT Layer 1 Dafny proofs) and the spec contains trivially-true assertions (line 274: `assert BoundedFindings(run) || true;`), suggesting it's a specification document rather than executable verification. Resolution: Either execute Dafny verification via available tooling to confirm green status, or clarify the spec is documentation-only and remove from verification artifacts list, retaining only `run_basic_tests.py` (which IS green).
- verify_evidence: Check 1 (existence): Both artifacts exist. Check 2 (green status): `run_basic_tests.py` is GREEN (all 13 tests passed), but Dafny spec cannot be executed due to missing tooling. Implementer reported `verification_status: green` but this cannot be independently confirmed for the Dafny artifact. Check 3 (coverage): Core mutation framework (`lib/mutations.py`) fully covered by `run_basic_tests.py`. Documentation/routing files (6/22) uncovered but contain no executable logic. Primary verification gap is the unexecuted Dafny spec.
- verify_verdict: fail


# Stage 6 — Fix verification failures (ONE-SHOT)

You are the implementer, on your single fix pass. The verify stage
flagged one or more failures. Your job is to address **every** failure
in **one** pass.

## You get exactly ONE chance.

- This is your only fix pass. After this stage, the workflow exits to
  a PR.
- Whatever you don't address — fix or document — reaches the PR
  unflagged.
- No new scope. Address only what verify flagged.

## What you have

Your preamble contains:

- The full **verify** stage response (failures listed across Check 1,
  Check 2, Check 3).
- `verify_concerns` in context — the itemised list verify wrote.
- The **implement** stage's response (`files_changed`, artifact paths,
  declared verification track).
- `plan.md` is on disk; the codebase is in whatever state implement
  left it.

## What to do

1. Re-read `plan.md` and the verify response so you know exactly what
   was promised and what was flagged.
2. List every concern from `verify_concerns`. Number them.
3. For each concern, do exactly **one** of:
   - **Fix** — edit the code, edit the artifact, add a missing
     artifact. Cite which file changed.
   - **Defer** — the concern is a real issue but cannot be addressed
     in this pass without scope creep or external input. State the
     concrete reason. Deferred concerns surface on the PR for human
     triage.

   Do **not** silently ignore a concern. Every numbered item gets a
   status.

4. After all fixes: re-run the verification artifacts on the declared
   track:
   - **`formal`** — `mcp__plugin_crosscheck_dafny__dafny_verify` on
     each `.dfy`.
   - **`lightweight` / `semi-formal`** — the named property tests /
     regression tests.

   Quote the new exit status / test output verbatim. Record the final
   status as `green`, `red`, or `mixed` (some artifacts green, some
   red).

5. Commit fix changes. Conventional Commits — `fix:` prefix. The
   commit message body lists fixed concerns and any deferred ones,
   so the PR description carries the verification status forward.

## Output

Markdown report with:

1. **Concern resolution table** — for each numbered concern: status
   (`fixed` | `deferred`), affected file, one-line evidence.
2. **Re-run verification output** — verbatim Dafny exit status / test
   output tail.
3. **Final status** (`green` | `red` | `mixed`) with a one-paragraph
   summary of what is and isn't verified.

End with this JSON block:

```json
{
  "outcome": "succeeded",
  "context_updates": {
    "fix_attempted": true,
    "fix_concerns_total": <integer>,
    "fix_concerns_resolved": <integer>,
    "fix_concerns_deferred": <integer>,
    "verification_status_after_fix": "<green|red|mixed>"
  }
}
```

## Discipline

- One pass. Every concern gets a status — fixed or deferred, never
  silent.
- Surgical changes only. The verify stage's job was to surface
  failures; do not volunteer a redesign.
- Honest re-run. If your fix didn't actually make the artifact pass,
  record `red` or `mixed` — do not claim `green` you didn't observe.
- Deferred items are not failures of this stage; they are inputs to
  human triage on the PR.
