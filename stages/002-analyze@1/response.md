Perfect! Now I have enough context. Let me produce my analysis.

---

# Analysis — Issue #140: assurance-probe (deterministic test-strength layer)

## Issue Summary

**Title:** crosscheck: assurance-probe — deterministic test-strength layer (design discussion)

**Problem Statement:** The crosscheck plugin has a gap in its 6-layer assurance hierarchy. While Layer 4 mechanically checks that every invariant has a covering test (bidirectional ID match), it does **not verify that the covering test would actually fail if the invariant were violated**. A vacuous test—one that imports the module but never exercises the failure condition—currently passes the Layer 4 gate. The issue proposes a new `/crosscheck:assurance-probe` skill to fill this gap via deterministic mutation testing.

## Files Involved

- `crosscheck/skills/spec-adversary/SKILL.md:1-263` — Layer 6 (spec completeness) skill that adversarially probes for missing invariants; provides the issue-only rotation pattern to mirror
- `crosscheck/skills/assurance-init/SKILL.md:1-344` — Bootstrap skill that scaffolds governance; establishes dual-track enforcement and the 6-layer hierarchy structure
- `crosscheck/skills/invariant-coverage-scaffold/SKILL.md:1-150` — Layer 4 skill that installs the **existence** gate (invariant ID ↔ test comment match); the new probe extends this by checking test **strength**
- `crosscheck/docs/assurance-hierarchy.md:1-41` — Documents the 6-layer hierarchy; Layer 4 currently described as "Implementation–spec alignment" (deterministic)
- `crosscheck/README.md:1-100` — Plugin overview; mentions Layer 4 as "your spec and your code can't drift"
- `crosscheck/demo/06_test_adequacy/SCRIPT.md:1-155` — Demo showing how `/rationale` identifies gaps in test coverage via claim-tree adequacy arguments; relevant pattern for mutation failure detection

## Evidence Trace

Since this is a **feature proposal** (not a bug/regression), there's no failing test to trace. Instead, here's the design gap analysis:

### Phase 1: Current Layer 4 Gate Behavior

**What it checks (per `invariant-coverage-scaffold/SKILL.md:17-20`):**
1. Every invariant in `docs/invariants/*.md` has a `# Invariant <ID>: <Name>` comment in a test file
2. Every test comment references an invariant ID that exists in the docs

**What it does NOT check:**
- Whether the test actually asserts the invariant's property
- Whether the test would fail if the invariant were violated

**Evidence:** The skill at `invariant-coverage-scaffold/SKILL.md` emits scripts that grep for comment presence, not test behavior. A test with `# Invariant I1: PRIMARY_WORKSPACE_DETERMINISM` above an empty function body passes.

### Phase 2: Proposed Solution (from issue body)

**Mutation probe (deterministic):**
1. Parse `Failure condition` clause from `docs/invariants/<module>.md`
2. Apply targeted source mutations (e.g., drop `ORDER BY is_primary DESC` for a deterministic-membership invariant)
3. Run pytest on the covering test
4. Record `killed / survived / errored`

**Output:** A committed reproducer script at `scripts/probe/<module>_<YYYYMMDD>.py` + GitHub issue with ≤3 findings + row appended to `.assurance/probe-tracker.csv`

**Pattern mirrors:** `/spec-adversary` (rotation-based, issue-only, SNR kill criterion, bounded output)

### Phase 3: What Makes This Deterministic (Layer 4)

Unlike `/spec-adversary` (Layer 6, best-effort LLM probing), the probe is deterministic because:
1. **Mutation specification is mechanical** — derived from `Failure condition` clause (not LLM-generated)
2. **Test execution is deterministic** — same commit + same mutation → same `killed/survived/errored` verdict
3. **Reproducer is committed** — future runs re-derive the same findings

This fits Layer 4's "deterministic" confidence tier (per `assurance-hierarchy.md:14`).

### Phase 4: Integration Points

**Pre-commit stub:** Already scaffolded by `/assurance-init` (per `assurance-init/SKILL.md:261-280`). The probe does NOT run in pre-commit (too slow); it runs on rotation like `/spec-adversary`.

**Dual-track enforcement:** The probe writes a deterministic script to `scripts/probe/`, but the findings themselves are advisory (issue-only, not a PR gate).

**No overlaps with existing skills:**
- **Completeness** → `/spec-adversary` (Layer 6)
- **Accuracy of invariant prose** → `/intent-check` (Layer 5, probabilistic)
- **Existence of covering test** → `/invariant-coverage-scaffold` (Layer 4, deterministic)
- **Strength of covering test** → **GAP** ← the probe fills this

## Root-Cause Hypothesis

**Primary hypothesis:** The gap exists because the current Layer 4 gate was designed for **presence verification** only (ID bidirectional match), not **behavioral verification** (test actually enforces the invariant). This is architecturally correct—each layer should have a single responsibility—but it leaves test strength unvalidated.

**Why this wasn't implemented earlier:**
- `/invariant-coverage-scaffold` focuses on fast local gates (< 5s pre-commit); mutation testing is too slow
- The rotation-based pattern (`/spec-adversary`) didn't exist at Layer 4 yet, so there was no precedent for "deterministic, issue-only, rotation-gated" workflows

**Alternative hypothesis (kept alive):** The gap could be handled by tightening `/intent-check` (Layer 5) to validate test→invariant alignment rather than just spec→intent alignment. **Refutation:** `/intent-check` is probabilistic (~96% accuracy, round-trip informalization). The mutation probe is deterministic and produces a binary `killed/survived` verdict. Different confidence tiers → different layers.

## Classification

**Classification:** `algorithmic`  
**Verification track:** `formal` (though the implementation is a script, not Dafny—the **determinism** of the output is what matters)

**Rationale:**
- **Not `crud-io`** — this is about test correctness, not database/network glue
- **Not `concurrency`** — no locks/channels/async involved
- **Not `floating-point`** — no IEEE-754 math
- **Not `fault-localize`** — there's no broken code; this is a feature gap
- **Not `refactor`** — adding new functionality, not restructuring existing code
- **`algorithmic`** because the core task is:
  1. Parse structured data (`Failure condition` clauses)
  2. Apply deterministic transformations (source mutations)
  3. Execute tests and classify results (`killed/survived/errored`)
  
  All of these are sequential logic, no I/O trust boundaries, deterministic output.

The verification track is `formal` in the sense that the **output** must be deterministic and reproducible (per the issue's design requirement: "Re-running the script on the same commit MUST yield bit-identical findings"). The implementation mechanism (a Python mutation-testing script) is less important than the guarantee it provides.

---

```json
{
  "context_updates": {
    "issue_ref": "nicholls-inc/claude-code-marketplace#140",
    "issue_title": "crosscheck: assurance-probe — deterministic test-strength layer (design discussion)",
    "analysis_classification": "algorithmic",
    "verification_track": "formal",
    "files_involved": [
      "crosscheck/skills/spec-adversary/SKILL.md",
      "crosscheck/skills/assurance-init/SKILL.md",
      "crosscheck/skills/invariant-coverage-scaffold/SKILL.md",
      "crosscheck/docs/assurance-hierarchy.md",
      "crosscheck/README.md",
      "crosscheck/demo/06_test_adequacy/SCRIPT.md"
    ]
  }
}
```