# Plan: crosscheck: assurance-probe — deterministic test-strength layer (design discussion)

## Verification track
semi-formal

## Steps

1. **Create `/assurance-probe` skill** — `crosscheck/skills/assurance-probe/SKILL.md`  
   New Layer 4 skill that deterministically measures test strength for covered invariants. Given a module and its invariant doc, the skill:
   - **Onboarding gate**: Checks for `docs/invariants/<module>.md`, `docs/assurance/ROADMAP.md`, and `.claude/rules/protected-surfaces.md`. If any are missing, emits verbatim: `Repo not onboarded. Missing: <items>. Next: /assurance-init.` and stops — does not emit a strength table, which would be indistinguishable from "all tests are strong." This matches the gate pattern used by `/assurance-status`.
   - Parses `docs/invariants/<module>.md` to extract all invariant IDs (not `<!-- aspirational -->`)
   - For each covered invariant (those with `// Invariant <ID>:` test comments), reads the covering test(s)
   - **Deterministic aggregation rule for multi-file coverage**: When an invariant ID is covered by more than one test file, compute the score for each covering file independently, then take the **minimum** (weakest-wins rule). This ensures the table is deterministic regardless of file-read order and is conservative: the invariant is only as strong as its weakest test. All covering files are listed in the "test file" column as a comma-separated list; per-dimension evidence is sourced from the weakest-scoring file.
   - Applies a deterministic 5-dimension strength rubric where every dimension is evaluated by observable file content only — **no LLM inference step**. Each dimension maps to specific, grep-able markers:
     1. **Boundary/edge-case coverage** (0–1 pt): presence of at least one of `min`, `max`, `empty`, `zero`, `nil`, `null`, `boundary`, `edge`, `-1`, `[]`, `{}` as a whole-word or token in the test body.
     2. **Property-based vs example-based** (0–1 pt): presence of at least one of `@given`, `@example` (Hypothesis), `fc.` or `fast-check` (fast-check), `gopter`, `rapid.Make`, `quickcheck`, `prop.ForAll` in the test file.
     3. **Assertion post-condition scope** (0–1 pt): count of assertion keywords (`assert`, `assertEqual`, `assertRaises`, `expect(`, `should.`, `Must`, `t.Error`, `t.Fatal`) — score = 1 if count ≥ 3, else 0.
     4. **Mutation probe hint** (0–1 pt): presence of at least one of `mutmut`, `pitest`, `stryker`, `#mutant`, `# mutant`, `@mutant`, `mutpy` as a whole-word match anywhere in the test file. Score = 0 if none found. A zero score on this dimension does not penalise the final score beyond the rubric — it is a hint, not a disqualifier; the gap description notes "no mutation probe markers found."
     5. **Composite scenario breadth** (0–1 pt): count of distinct test functions or test methods in the file (patterns: `def test_`, `func Test`, `it(`, `test(`, `describe(`) — score = 1 if count ≥ 2, else 0.
   - Total score 0–5. Strength label: 1 = "minimal", 2 = "weak", 3 = "moderate", 4 = "strong", 5 = "comprehensive".
   - Emits a structured strength table (invariant ID, test file(s), strength score 1–5, per-dimension scores, gap description)
   - Produces a sorted "weakest first" action list of the top-N invariants most likely to miss a real bug

2. **Register skill in `hellebuyck.md`** — `crosscheck/agents/hellebuyck.md`  
   - Add `/assurance-probe` row to the **Verification (Spec Chain)** table (Layer 4, deterministic)
   - Add task-classification row: trigger "test strength", "how strong are the tests", "probe invariant coverage", "weak tests" → `/assurance-probe`
   - Add skill-path line to Phase 3 Execute block
   - Note: trigger phrases are distinct from existing rows — "invariant coverage" routes to `/invariant-coverage-scaffold` (different from "probe invariant coverage"), "assurance status" routes to `/assurance-status` (different from "how strong are the tests"); verified explicitly by the trigger-phrase non-overlap check in the Verification approach below.

3. **Update `crosscheck/docs/skills.md`**  
   - Add `/assurance-probe` row to the "Assurance hierarchy — Layer 4 (impl–spec alignment)" table, with trigger phrases and one-line summary, owner hellebuyck

4. **Update `crosscheck/docs/assurance-hierarchy.md`**  
   - Add `/assurance-probe` to Layer 4 row of the skill→layer mapping table (alongside `/invariant-coverage-scaffold`, `/protected-surface-amend`, `/check-regressions`)

5. **Update `crosscheck/README.md`**  
   - Mention `/assurance-probe` in the Layer 4 bullet under "What you can run right now"
   - Add it to the "Assurance hierarchy & governance" skills overview paragraph

## Tests / properties to add

- **Trace: strength-rubric determinism** — given the same test file, the skill must produce the same strength score on two runs (semi-formal execution trace anchor: deterministic observable inputs → deterministic output)
- **Trace: aspirational exclusion** — invariants tagged `<!-- aspirational -->` are excluded from the probe scope; trace confirms no score is emitted for them
- **Trace: zero-test invariant handling** — if a covered invariant has a test comment but the test body has zero assertions, score = 1 and gap description = "no assertions found"; skill does not error
- **Trace: multi-file weakest-wins** — given invariant I1 covered by file-A (score=4) and file-B (score=2), the emitted row shows score=2 and both files listed; changing the read order does not change the score
- **Trace: unonboarded repo gate** — if `docs/invariants/<module>.md` is absent, the skill emits the verbatim refusal and emits no strength table rows
- **Patch comparison anchor (byte-identity)** — the `hellebuyck.md` task-classification table before vs. after the addition: the new row must not change any existing row's content (bytes of every existing row are identical)
- **Trigger-phrase non-overlap check** — enumerate the trigger signals of the new `/assurance-probe` row ("test strength", "how strong are the tests", "probe invariant coverage", "weak tests") against every existing trigger signal in the Task Classification table; confirm no existing row's trigger signal is a substring-match or synonym match of the new row's triggers; specifically verify "probe invariant coverage" does not overlap with "invariant coverage" / "coverage gate" / "scaffold invariant check" (which routes to `/invariant-coverage-scaffold`) nor with "assurance status" / "status dashboard" patterns

## Verification approach

Semi-formal — five execution trace checks, one patch comparison, and one trigger-phrase non-overlap check:

1. **Execution trace (strength-rubric determinism):** read the skill's rubric definition; manually trace its evaluation over a minimal synthetic test file (e.g. a 3-line test with one assertion and no property framework); confirm the score equals what the rubric formula produces. Anchor: the rubric produces integer scores 0–5 from observable counts and keyword presence only, no LLM judgment.

2. **Execution trace (aspirational exclusion):** trace the skill's Step 1 (parse invariant doc) against a doc fragment containing `<!-- aspirational -->`; confirm the ID is absent from the output table.

3. **Execution trace (zero-assertion edge case):** trace Step 2 (read test) for a synthetic test file with `// Invariant I1: Foo` but no assertion keywords; confirm score=1, gap="no assertions found", no crash.

4. **Execution trace (multi-file weakest-wins):** trace the aggregation rule for invariant I1 with two covering files: file-A scoring 4 and file-B scoring 2. Confirm emitted score=2, test files column = "file-A, file-B" (or "file-B, file-A" — both are correct). Reverse the read order and confirm same score=2. This anchors determinism under different filesystem orderings.

5. **Execution trace (unonboarded repo gate):** trace the skill entry with `docs/invariants/<module>.md` absent; confirm the verbatim refusal message is emitted and no strength table rows appear — the empty-table case is explicitly labelled "repo not onboarded", not "all tests are strong."

6. **Patch comparison (`hellebuyck.md`) — byte-identity:** diff the before/after of the task-classification table; the existing rows must be byte-identical; only one new row is added.

7. **Trigger-phrase non-overlap check:** extract trigger signal tokens from the new `/assurance-probe` row. For each existing task-classification row in `hellebuyck.md`, check whether any of the new trigger phrases ("test strength", "how strong are the tests", "probe invariant coverage", "weak tests") are substring-equal to or would match a keyword in the existing row's trigger signal column. Flag any overlap as a routing conflict. Specifically verify the following high-risk pairs are non-overlapping:
   - "probe invariant coverage" vs "Add invariant coverage", "wire up the gate", "scaffold invariant check" (Bootstrap coverage gate → `/invariant-coverage-scaffold`)
   - "how strong are the tests" vs "assurance status", "weekly check-in" (Status dashboard → `/assurance-status`)
   - "weak tests" vs any existing row

## Risk register

- **Rubric subjectivity leaks through** — if any dimension involves LLM judgment it is no longer deterministic; mitigation: all 5 dimensions are operationalized as observable keyword presence checks or integer counts with explicit threshold; each dimension lists exact grep-able tokens so any LLM evaluating the rubric follows the same deterministic path
- **Multi-file coverage produces non-deterministic scores** — file read order varies across filesystems; mitigation: weakest-wins aggregation rule (take minimum score) is order-independent; the rule is stated explicitly in the SKILL.md
- **New skill disturbs hellebuyck routing** — adding a new classification row could shadow an existing one; mitigation: trigger-phrase non-overlap check (Verification step 7) explicitly checks the high-risk pairs; the risk-register statement "trigger phrases are distinct" is now backed by a concrete verification step, not just assertion
- **Unonboarded repo emits falsely-green table** — without an onboarding gate, missing invariant docs produce an empty table indistinguishable from "all tests strong"; mitigation: explicit gate matches `/assurance-status` Phase 1 pattern; verbatim refusal message required; Verification step 5 confirms the empty-table case is labelled correctly
- **README/docs drift** — updating 4 files touches documentation that is not gated; mitigation: these are additive-only appends to existing tables and bullet lists; no existing content is modified
- **Layer assignment ambiguity** — the issue title says "deterministic test-strength layer" implying Layer 4; risk of mis-classifying it as Layer 5 (probabilistic); mitigation: the rubric is observable-count-based, so it is deterministic and belongs at Layer 4 by the hierarchy's definition
