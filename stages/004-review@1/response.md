Now I have a thorough understanding of the existing structure. Let me conduct the full adversarial review.

---

## Step 1 — Back-translation (from plan.md only)

The plan creates a new Layer 4 skill called `/assurance-probe` that deterministically measures the **strength** (not just presence/absence) of tests covering declared invariants. It reads invariant doc files to identify covered invariant IDs (excluding aspirational ones), reads their covering test files, and applies a 5-dimension rubric based on observable counts and keyword presence — boundary/edge case coverage, property-based vs example-based, assertion post-condition scope, mutation probe hints, composite scenario breadth — to score each test 1–5. It emits a sorted "weakest first" strength table. The skill is then registered in hellebuyck's routing table and task-classification table, added to `docs/skills.md`'s Layer 4 section, added to `docs/assurance-hierarchy.md`'s Layer 4 row, and mentioned in `README.md`. The plan proposes four semi-formal execution-trace verifications: rubric determinism, aspirational exclusion, zero-assertion edge case, and a patch comparison on hellebuyck's classification table.

---

## Step 2 — Diff against intent

Now let me check the original issue description more carefully. The issue is referenced as "crosscheck: assurance-probe — deterministic test-strength layer (design discussion)".

The analyze stage framing describes this as:
- **Issue**: Proposes a new Layer 4 skill `/assurance-probe` — a deterministic test-strength measurement layer — structured as a design discussion
- **Classification**: refactor (adding new skill/capability)
- **Track**: semi-formal

The plan creates exactly that: a new `/assurance-probe` skill registered across four documentation/configuration files. The issue title says "design discussion," implying this may be a proposal/discussion, not a firm "ship this feature" request. The plan treats it as a concrete implementation.

**Gap assessment**: The word "design discussion" in the issue title raises a subtle question — does the issue ask for a concrete implementation or a design proposal? But given the issue is filed against a plugin that ships skills as SKILL.md files, and the plan track is "semi-formal" (meaning a new SKILL.md is the artifact), this is aligned. The plan correctly creates the skill.

**Verdict: Aligned** — the plan addresses what was asked.

---

## Step 3 — Probes

### Probe 1: Hidden assumptions

**Concern 1** (plan Step 1): The skill is described as parsing `docs/invariants/<module>.md` to extract invariant IDs, and then finding covering tests via `// Invariant <ID>:` comments. This silently assumes the repo has already adopted the invariant-coverage convention established by `/invariant-coverage-scaffold`. The plan does not specify whether `/assurance-probe` requires the onboarding gate (ROADMAP, horizon dirs, invariants dir, protected-surfaces rules) to be present before running — unlike `/intent-check`, `/spec-adversary`, and `/assurance-status` which all enforce the gate. If `/assurance-probe` runs on an unonboarded repo, it will silently find no invariants and emit an empty table, which may mislead users into thinking test strength is "clean."

**Concern 2** (plan Step 1): The plan states the skill reads "the covering test(s)" for each invariant, but does not specify whether it looks for a single `// Invariant <ID>:` comment or all occurrences across the repo. If one invariant is covered by 3 test files, does the skill score all 3 separately, take the max, take the min, or aggregate? The plan is silent on multi-test-file scenarios. The `assurance-status` skill handles this through the coverage script output; `/assurance-probe` has no analogous delegate.

**Concern 3** (plan Step 1): The rubric dimension "mutation probe hint" is described as being observable from test structure, but the plan doesn't define what constitutes a "mutation probe hint." Is it presence of `mutmut`, `pitest`, or comment keywords like `#mutant`? Without this being specified in the SKILL.md, the rubric is not fully deterministic — different LLM runs may interpret "mutation probe hint" differently, violating the plan's own key invariant that "No LLM round-trip on test quality."

### Probe 2: Missing edge cases

**Concern 4**: The plan handles `<!-- aspirational -->` exclusion and zero-assertion tests, but doesn't specify what happens when:
- A module has a `docs/invariants/<module>.md` that exists but contains zero non-aspirational invariant IDs (all marked aspirational) — the plan does not specify that the skill emits "no covered invariants found" vs an empty table vs an error.
- An invariant ID is covered by a test with `// Invariant <ID>:` but the test file cannot be read (permission error, binary file, etc.) — the plan says "skill does not error" only for the zero-assertion case.

These are edge cases a SKILL.md should explicitly address for a skill claiming to be "deterministic."

### Probe 3: Test adequacy

The plan proposes four semi-formal execution trace checks. These are appropriate for a semi-formal track. However:

**Concern 5**: The "patch comparison" test for hellebuyck.md checks only that "existing rows must be byte-identical" — but this only catches row *deletion* or *mutation* of existing rows. It does not verify that the new row's routing (trigger phrases) doesn't *shadow* an existing row via semantic overlap. The plan's risk register mentions this ("trigger phrases are distinct") but the patch comparison test doesn't verify it — the byte-identity check is only a structural diff, not a semantic routing conflict check. Since the routing table is evaluated top-to-bottom (presumably), a semantically overlapping row could shadow others without any byte-level change to existing rows.

### Probe 4: Verification track fit

Pass — the plan correctly claims semi-formal track for a skill that is itself a SKILL.md artifact (LLM-interpreted methodology document). The execution-trace verifications are appropriate for testing a skill description. No Dafny tractability concerns.

### Probe 5: Scope creep

Pass — the plan touches exactly the files listed in the issue's `files_involved` list from the analyze stage: `crosscheck/skills/assurance-probe/SKILL.md` (new), `crosscheck/agents/hellebuyck.md`, `crosscheck/docs/assurance-hierarchy.md`, `crosscheck/docs/skills.md`, `crosscheck/README.md`. Note that `crosscheck/.claude-plugin/plugin.json` is in the `files_involved` list but the plan does not touch it — plugin.json does not enumerate skills (it only registers MCP servers), so this omission is correct and not a gap.

### Probe 6: Reversibility / blast radius

Pass — all changes are additive. The new SKILL.md is a new file. The edits to hellebuyck.md, skills.md, assurance-hierarchy.md, and README.md are table/list appends. No shared infrastructure is destructively modified. The "byte-identical existing rows" patch comparison is the right safeguard for hellebuyck.md.

---

## Step 4 — Missing invariants (spec-adversary)

### Missing invariant 1 — Onboarding gate requirement must be stated explicitly

**Statement**: `/assurance-probe` must either (a) require the same onboarding gate as other hellebuyck skills (`docs/invariants/` present, ROADMAP, etc.) or (b) explicitly document that it runs without the gate and what it emits on an empty/non-existent invariants dir.

- **Why it matters**: Without this, a user running `/assurance-probe` on an unonboarded repo gets an empty strength table, which may be silently interpreted as "all tests are strong" or "there's nothing to check." This is the exact failure mode `/assurance-status` was designed to prevent. If `/assurance-probe` is a Layer 4 skill invoked by hellebuyck, it inherits hellebuyck's onboarding-gate pattern — but the plan doesn't specify this.
- **Triage**: **accept** — must be stated in the SKILL.md before merge. The hellebuyck Phase 2 gating pattern is a non-negotiable design constraint; the plan must either confirm probe is gated or justify why it isn't.

### Missing invariant 2 — Multi-file coverage aggregation rule

**Statement**: When a single invariant ID is covered by more than one test file, the skill must state a deterministic aggregation rule (e.g., score the weakest, score the strongest, list all separately).

- **Why it matters**: Without a specified aggregation rule, two runs of the skill on the same repo (one reading files in alphabetical order, one in modification-time order) could produce different strength tables, violating the plan's central claim of determinism.
- **Triage**: **accept** — the plan explicitly claims "No LLM round-trip on test quality — outputs are directly derived from observable test structure" and "deterministic 5-dimension strength rubric." Multi-file aggregation breaks this unless specified.

### Missing invariant 3 — Rubric dimension operationalization for "mutation probe hint"

**Statement**: Every rubric dimension must be defined in terms of specific, grep-able, language-agnostic keywords or structural markers; "mutation probe hint" must be concretized (e.g., presence of `mutmut`, `pitest`, `stryker`, comment substring `mutation`, or the literal absence of any such keyword = score 0 on this dimension).

- **Why it matters**: A rubric dimension that requires LLM judgment to evaluate is not a deterministic rubric. The plan's risk register acknowledges this risk but doesn't resolve it in the skill's design. If the SKILL.md says "mutation probe hint" without operationalizing it, implementations (LLM runs of the skill) will vary.
- **Triage**: **accept** — this is the same risk the plan's own risk register flags ("Rubric subjectivity leaks through"), but the plan leaves the operationalization to the implementer of the SKILL.md. The adversarial review must flag that the operationalization must be included in the SKILL.md, not left implicit.

---

## Decision

Three `accept`-priority missing invariants: onboarding gate requirement, multi-file aggregation rule, and rubric operationalization. Plus three probes that are not clean passes (Concerns 1–3, 5). This is a **revise**.

```json
{
  "preferred_next_label": "revise",
  "context_updates": {
    "review_verdict": "revise",
    "intent_gap": "aligned",
    "review_concerns": "1. The SKILL.md for /assurance-probe must specify whether it requires the hellebuyck onboarding gate (docs/invariants/ present, ROADMAP, protected-surfaces rules) or explicitly documents what it emits on an unonboarded repo. Without this, a user on an unonboarded repo receives a silently empty strength table, which is indistinguishable from 'all tests are strong.' Every other hellebuyck skill either enforces the gate or documents the exception — /assurance-probe must do the same. (Probe 1, Missing invariant 1)\n2. The SKILL.md must state a deterministic aggregation rule for the case where a single invariant ID is covered by more than one test file. Without a specified rule (weakest / strongest / list-all-separately), two runs reading files in different orders can produce different strength tables, directly violating the plan's claim of determinism. (Probe 1, Missing invariant 2)\n3. Every rubric dimension must be operationalized as specific, grep-able, language-agnostic keywords or structural markers in the SKILL.md. 'Mutation probe hint' is named as a dimension but never concretized (e.g., presence of 'mutmut', 'pitest', 'stryker', '#mutant' in the file, or zero score otherwise). A dimension that requires LLM judgment to evaluate is not a deterministic rubric — the plan's own risk register flags this but does not resolve it in the skill design. (Probe 1, Missing invariant 3)\n4. The 'patch comparison' verification test for hellebuyck.md only verifies byte-identity of existing rows (catching deletion/mutation), not semantic routing conflicts. The new row's trigger phrases ('test strength', 'how strong are the tests', 'probe invariant coverage', 'weak tests') must be checked against existing trigger signals for semantic overlap — particularly against 'invariant coverage', 'coverage gate', 'scaffold invariant check' (which routes to /invariant-coverage-scaffold) and 'assurance status' patterns. The plan must add a trigger-phrase non-overlap check to the verification section, not just rely on the risk register statement that 'trigger phrases are distinct.' (Probe 3, Concern 5)"
  }
}
```