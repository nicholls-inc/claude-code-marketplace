Good! Now I have enough context. Let me proceed with the review according to the instructions.

## Step 1 — Back-translate the plan

Reading `plan.md` end-to-end without looking back at the issue:

The plan creates a **deterministic test-strength verification layer** for crosscheck by building `/assurance-probe`, a skill that evaluates whether the tests protecting existing invariants are adequate. It operates through three probes deployed in phases:

**Phase 1** introduces a **mutation probe** that parses "Failure condition" clauses from invariant documentation, generates 1-3 targeted source mutations per invariant, applies each mutation, runs the covering test, and reports whether the test killed the mutation (test is effective), survived (test is vacuous), or errored (test is broken). The probe outputs at most 3 findings per run in GitHub issue format, tracks signal-to-noise ratios in a CSV file, and includes a deterministic reproducer script. This phase is Python-only.

**Phase 2** (gated on Phase 1 achieving SNR ≥ 1:3) adds a **vacuity probe** that temporarily deletes a covering test, measures branch-coverage delta, and flags tests with zero delta as non-load-bearing.

**Phase 3** (gated on Phase 2 success) adds a **generator probe** that inspects Hypothesis test strategies to identify failure regions the generator cannot reach.

The skill integrates with the existing rotation workflow, uses issue-only output like `/spec-adversary`, enforces a kill criterion (SNR <1:5 over 4 weeks → retire), and is verified through deterministic properties: mutation determinism, reproducer bit-identity, bounded output (≤3 findings), and tracker integrity.

## Step 2 — Diff against intent

Based on the preamble context:
- **Issue title**: "crosscheck: assurance-probe — deterministic test-strength layer (design discussion)"
- **Classification**: algorithmic
- **Track**: formal

Now comparing to my back-translation:

The issue title describes a **deterministic test-strength layer** (design discussion). My back-translation describes a skill that:
1. Tests whether invariant-protecting tests are strong enough (mutation/vacuity/generator probes)
2. Is deterministic (bit-identical reproducers, bounded output, phase-gated)
3. Sits at Layer 4 of the assurance hierarchy (deterministic implementation–spec alignment verification)

**Assessment: ALIGNED**

The plan appears to directly address a test-strength verification layer with deterministic properties. However, I have a concern about whether this is the *right* Layer 4 concern based on the assurance hierarchy doc I read. Let me verify this in Step 3.

## Step 3 — Probe (run every probe; no skipping)

### 1. **Hidden assumptions**

- **Step 2 (mutation framework)**: Assumes the "Failure condition" clause in invariant docs follows a parseable pattern. The risk register acknowledges "complex conditions" as unsupported (step 159), but the plan never states what a "simple condition" is or shows an example. **Concern 1**: What grammar does the mutation parser accept? The plan should either cite an existing invariant doc pattern or include a reference grammar.

- **Step 3 (vacuity probe)**: Assumes `pytest --cov` is installed and that coverage can be meaningfully measured. Also assumes tests can be "deleted temporarily" and "restored" without side effects. Risk register addresses restoration (line 143-144: use Git worktree), but never addresses the assumption that coverage tools exist or are configured. **Concern 2**: What if the adopter repo has no coverage tooling installed? Should Phase 2 fail-fast or auto-install?

- **Step 5 (reproducer template)**: Assumes the reproducer can be "bit-identical on re-run" (line 38). But the plan never states what state the reproducer locks down (git commit hash? Python version? dependency versions? system clock?). **Concern 3**: What does "bit-identical on same commit" (line 111) mean? Same git SHA? Same machine? Same virtualenv?

- **Step 11 (byfuglien routing)**: States "probe is rotation-based, not per-PR" (line 71). But the plan never defines what "rotation-based" means in operational terms. **Concern 4**: How often should `/assurance-probe` run? Who triggers it? Is there a cron job, or is it manual invocation guided by `/assurance-status`?

- **Verification approach (line 119)**: "Tracker integrity" property assumes a single writer (line 119-122). What if two `/assurance-probe` runs execute concurrently? **Concern 5**: Is concurrent execution prevented, or does the property test need to verify append-only semantics under contention?

**Pass** on most architectural assumptions, but **five sub-concerns raised** above.

### 2. **Missing edge cases**

- **Empty `Failure condition` clause**: Step 2 (line 17) says "Parse `Failure condition` clause" but never says what happens if the clause is missing or empty. Risk register line 158-159 covers "complex" clauses but not missing ones. Should the probe skip that invariant, warn, or error?

- **Mutation kills test framework itself**: What if a mutation breaks imports or syntax such that `pytest` cannot even load the test file? Risk register line 18 has "errored" verdict, but does this cover test-file syntax errors vs. test execution errors?

- **Zero invariants in a module**: Step 1 (line 9) mentions "1-3 targeted source mutations per invariant." What if a module doc exists but has zero invariants? Or the invariant doc exists but the module code is missing?

- **SNR denominator is zero**: Line 44 tracks `accepted, rejected, deferred`. What if zero findings are proposed over 4 weeks? Is SNR 0/0 or undefined? Does the kill criterion trigger?

- **Reproducer script is committed but later repo history is rebased/squashed**: Line 38 says reproducer is "committed to adopter repo". What if the commit SHA it references no longer exists after a force-push?

**Concern 6**: Handle edge cases for missing/empty failure conditions, zero-invariant modules, zero-finding runs, and rebased history in reproducer scripts.

### 3. **Test adequacy**

- **Unit test for mutation parser** (line 81): "Property: mutations are deterministic (same input → same mutations)". This is a **tautology test** — determinism is trivially true if you call the function twice in a row. The real test is: does the parser produce the *correct* mutations for a given `Failure condition`? The test "generate expected mutations for common patterns" (line 83) is closer, but the plan does not define "expected" or "common patterns."

- **Integration test for reproducer** (line 90): "Run reproducer script twice, assert bit-identical output." This is good, but it does not test whether the reproducer **accurately reproduces the finding**. If the reproducer always prints "killed" regardless of the actual mutation, this test would pass. **Concern 7**: Add a negative test: "Reproducer on a different commit (or with mutation reverted) must produce different output."

- **E2E test** (line 94): "Scaffold fake adopter repo with invariant doc + test." Does the fake repo include a real Python test that *can* kill a mutation? Or is it a mock? If mocks, this is test theatre. **Concern 8**: E2E test must use a real, executable invariant doc + covering test + killable mutation, not scaffolded stubs.

**Concern 9**: Mutation parser tests must verify correctness, not just determinism. Include an oracle (e.g., a reference table of `Failure condition` → expected mutation).

### 4. **Verification track fit**

The plan claims `formal` track (line 3), but the verification approach (lines 100-134) mostly describes **deterministic property testing** (Layer 4), not formal Dafny proofs (Layer 1). The skill *orchestrates test execution and mutation* (line 103), which is inherently imperative and IO-bound — not Dafny-tractable.

**However**, the formal track decision in crosscheck can mean "the thing being verified is deterministic" rather than "the verifier itself is Dafny-proved". Looking at the assurance hierarchy doc (line 14), Layer 4 skills like `/invariant-coverage-scaffold` are listed under "deterministic" with `byfuglien` or `hellebuyck` ownership, not under "formally verified pure code" (Layer 1).

**Re-assessment**: The plan's use of "formal" likely means "Layer 4 deterministic verification" in the crosscheck taxonomy, not "Layer 1 Dafny proof." The verification section title "Deterministic verification (Layer 4)" (line 105) confirms this.

**Pass** — track fit is correct if "formal" is shorthand for "deterministic Layer 4." But the terminology is ambiguous. **Concern 10**: Clarify in plan.md line 3 that "formal" means "Layer 4 deterministic" to avoid confusion with Layer 1 Dafny proofs.

### 5. **Scope creep**

The issue title says "design discussion." The plan *implements* the design with 12 steps including code generation, templates, tests, docs updates, and a demo. Is this scope creep?

Re-reading the issue title: "assurance-probe — deterministic test-strength layer (design discussion)". This could mean:
- **Interpretation A**: Design a test-strength layer (output: design doc only, no code).
- **Interpretation B**: Implement the test-strength layer based on a design discussion (output: working skill).

The preamble says `plan_step_count: 12` and includes steps like "Create mutation framework" and "Update byfuglien agent," which strongly suggests Interpretation B.

Given that the analyze stage succeeded and classified this as `algorithmic`, and given that all other referenced issues in this workflow (e.g., crosscheck features like `/spec-adversary`) are full implementations, **I conclude Interpretation B is correct**.

**Pass** — no scope creep detected. The issue requests implementation of a design, not just a design doc.

### 6. **Reversibility / blast radius**

- **Step 11** modifies `crosscheck/agents/byfuglien.md` (shared orchestrator). If `/assurance-probe` has high false-positive rate, can it be removed cleanly from byfuglien without breaking routing for other skills? The plan does not show the byfuglien diff, so I cannot assess backward compatibility. **Concern 11**: Document the byfuglien routing addition as additive-only (no modification of existing routes).

- **Steps 8-9** modify `crosscheck/docs/assurance-hierarchy.md` and `crosscheck/README.md`. These are high-visibility docs. If the skill is retired due to low SNR, do these docs need to be reverted? The plan does not address documentation rollback. **Concern 12**: Add a note in the risk register that doc updates should be conditional (e.g., "Layer 4 (experimental)" until Phase 1 demonstrates SNR ≥ 1:3).

- **Blast radius**: The skill writes to `.assurance/probe-tracker.csv` in the **adopter repo**. If the skill has bugs, it could corrupt this file. The plan has a tracker integrity property (line 119-122) that checks append-only semantics, but does not address recovery if the CSV is corrupted. **Concern 13**: Add a backup/restore mechanism or checksum validation to tracker writes.

**Pass with concerns** — reversibility is feasible, but three sub-concerns raised above.

## Step 4 — Missing invariants (spec-adversary)

### Missing invariant 1
**Statement**: Every mutation generated by the mutation probe must correspond to a path through the implementation that violates the stated `Failure condition`, **and** that path must be reachable by inputs the covering test's generator can produce.

**Why it matters**: If a mutation introduces a failure condition that the test's input generator cannot reach (e.g., mutation adds `if x < -1000` but test only uses `x ≥ 0`), the probe will report "test survived" — a false positive. The test might be adequate for the *actual* input domain, but the probe flags it as weak.

**Triage**: **accept** — This is the core soundness property of mutation testing. Without it, Phase 1 findings are noisy. The generator probe (Phase 3) partially addresses this, but it's deferred. Phase 1 should either:
- Require `Failure condition` clauses to include input preconditions, OR
- Filter mutations to only those reachable by the test's observed input coverage.

This should be added to Step 2 or the risk register as a known limitation with mitigation.

### Missing invariant 2
**Statement**: The reproducer script must capture **and verify** the exact Python environment (interpreter version, dependency versions, OS) in which the mutation was tested, and emit a clear error if run in a different environment.

**Why it matters**: "Bit-identical on same commit" (line 111) is insufficient. If the reproducer runs on Python 3.9 but the original probe used Python 3.12, the test behavior may differ (syntax changes, stdlib changes). The user sees non-identical output and loses trust in the reproducer.

**Triage**: **accept** — Reproducibility is a stated goal (lines 38, 111). The plan should specify that the reproducer template includes:
- Python version check (e.g., `assert sys.version_info >= (3, 12)`)
- Dependency lockfile reference (e.g., `requirements.txt` SHA)
- Clear error message if environment mismatch detected.

Add to Step 5 (reproducer template spec).

### Missing invariant 3
**Statement**: The probe must never propose the same finding twice across separate runs (idempotence), unless the underlying invariant doc or covering test has changed.

**Why it matters**: If the probe runs weekly and the user defers a finding (rejects or marks as low-priority), the probe should not re-propose the same mutation on the next run. This pollutes the issue tracker and degrades SNR.

**Triage**: **defer** — This is a quality-of-life improvement, not a soundness property. The plan's bounded output (≤3 findings per run, line 12) and rotation-based workflow (line 71) suggest infrequent runs, so re-proposing is less likely to dominate. However, if SNR becomes a problem in practice, add a "seen findings" log to `.assurance/probe-tracker.csv` or a separate `.assurance/probe-seen.csv`.

Worth a follow-up issue if Phase 1 demonstrates high SNR but user feedback reports annoyance from duplicates.

## Decide

**Verdict: REVISE**

**Intent gap**: aligned (the plan implements a deterministic test-strength layer as requested).

**Reasons for revision**:
1. Five hidden assumptions need clarification (mutation grammar, coverage tooling, reproducer environment bounds, rotation semantics, concurrent execution).
2. Six missing edge cases need handling (empty failure conditions, zero-invariant modules, SNR edge cases, test framework crashes, rebased history).
3. Three test adequacy gaps (mutation parser correctness oracle, reproducer negative test, E2E must use real executable test).
4. One track terminology ambiguity (clarify "formal" means Layer 4, not Layer 1 Dafny).
5. Three reversibility concerns (byfuglien routing backward compatibility, doc rollback plan, tracker corruption recovery).
6. Two missing invariants at **accept** priority (mutation-test soundness, reproducer environment capture).

The plan is **structurally sound** and **aligned with intent**, but has **12 distinct concerns** across soundness, determinism, and operational clarity that must be addressed before implementation.

```json
{
  "preferred_next_label": "revise",
  "context_updates": {
    "review_verdict": "revise",
    "intent_gap": "aligned",
    "review_concerns": "1. Hidden assumption (Step 2, line 17): Define the grammar/pattern for parseable 'Failure condition' clauses, or cite example invariant docs that demonstrate the expected format.\n2. Hidden assumption (Step 3, line 23-24): Specify behavior when coverage tooling (pytest-cov) is absent — fail-fast with actionable error or auto-install?\n3. Hidden assumption (Step 5, line 38, 111): Define 'bit-identical on same commit' — which variables are locked (git SHA, Python version, dependencies, OS)? Add explicit environment capture to reproducer template.\n4. Hidden assumption (Step 11, line 71): Define 'rotation-based' operationally — who triggers the probe, how often, via what mechanism (manual, cron, /assurance-status recommendation)?\n5. Hidden assumption (Verification line 119-122): Tracker integrity property assumes single writer. Address concurrent execution: prevent it, or prove append-only semantics under contention.\n6. Missing edge cases: Handle empty/missing Failure condition clauses (skip, warn, or error?), zero-invariant modules, zero-finding runs (SNR 0/0), test framework syntax errors vs execution errors, reproducer scripts referencing rebased-away commits.\n7. Test adequacy (line 81-84): Mutation parser test 'determinism' is tautological. Add correctness oracle — reference table mapping Failure condition examples to expected mutations.\n8. Test adequacy (line 90-92): Reproducer integration test should include negative case — run reproducer on different commit or with mutation reverted, assert output differs.\n9. Test adequacy (line 94-98): E2E test must use real executable Python test + real killable mutation, not mocked/scaffolded stubs (verify this is not test theatre).\n10. Track terminology (line 3): Clarify 'formal' means 'Layer 4 deterministic property testing', not 'Layer 1 Dafny proof', to avoid confusion.\n11. Reversibility (Step 11): Document byfuglien.md routing addition as additive-only (no modification of existing routes) to preserve backward compatibility if skill is retired.\n12. Reversibility (Steps 8-9): Add risk register note that README/assurance-hierarchy.md updates should be marked experimental (e.g., 'Layer 4 (Phase 1)') until SNR ≥ 1:3 demonstrated.\n13. Reversibility (line 119-122): Add backup/restore or checksum validation to tracker CSV writes to recover from corruption.\n14. Missing invariant (ACCEPT): Mutation soundness — every generated mutation must violate a Failure condition reachable by the covering test's input generator. Address in Step 2 or risk register as known Phase 1 limitation (Phase 3 generator probe partially mitigates).\n15. Missing invariant (ACCEPT): Reproducer environment capture — template must verify Python version, dependency versions, emit clear error on mismatch. Add to Step 5 spec."
  }
}
```