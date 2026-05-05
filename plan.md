# Plan: crosscheck: assurance-probe — deterministic test-strength layer (design discussion)

## Verification track
semi-formal

## Steps

1. **Create `/assurance-probe` skill** — `crosscheck/skills/assurance-probe/SKILL.md`  
   New Layer 4 skill that deterministically measures test strength for covered invariants. Given a module and its invariant doc, the skill:
   - Parses `docs/invariants/<module>.md` to extract all invariant IDs (not `<!-- aspirational -->`)
   - For each covered invariant (those with `// Invariant <ID>:` test comments), reads the covering test(s)
   - Applies a deterministic 5-dimension strength rubric (coverage of boundary/edge cases, property-based vs example-based, assertions post-condition scope, mutation probe hint, composite scenario breadth)
   - Emits a structured strength table (invariant ID, test file, strength score 1–5, per-dimension evidence, gap description)
   - Produces a sorted "weakest first" action list of the top-N invariants most likely to miss a real bug
   - No LLM round-trip on test quality — outputs are directly derived from observable test structure (line counts, assertion counts, `Hypothesis`/`fast-check`/`gopter` presence, range checks, negative case presence)

2. **Register skill in `hellebuyck.md`** — `crosscheck/agents/hellebuyck.md`  
   - Add `/assurance-probe` row to the **Verification (Spec Chain)** table (Layer 4, deterministic)
   - Add task-classification row: trigger "test strength", "how strong are the tests", "probe invariant coverage", "weak tests" → `/assurance-probe`
   - Add skill-path line to Phase 3 Execute block

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
- **Patch comparison anchor** — the `hellebuyck.md` task-classification table before vs. after the addition: the new row must not change any existing row's routing

## Verification approach

Semi-formal — three execution trace checks and one patch comparison:

1. **Execution trace (strength-rubric determinism):** read the skill's rubric definition; manually trace its evaluation over a minimal synthetic test file (e.g. a 3-line test with one assertion and no property framework); confirm the score equals what the rubric formula produces. Anchor: the rubric produces integer scores 1–5 from observable counts only, no LLM judgment.

2. **Execution trace (aspirational exclusion):** trace the skill's Step 1 (parse invariant doc) against a doc fragment containing `<!-- aspirational -->`; confirm the ID is absent from the output table.

3. **Execution trace (zero-assertion edge case):** trace Step 2 (read test) for a synthetic test file with `// Invariant I1: Foo` but no assertion keywords; confirm score=1, gap="no assertions found", no crash.

4. **Patch comparison (`hellebuyck.md`):** diff the before/after of the task-classification table; the existing rows must be byte-identical; only one new row is added.

## Risk register

- **Rubric subjectivity leaks through** — if any dimension involves LLM judgment it is no longer deterministic; mitigation: all 5 dimensions are observable counts or keyword presence checks (no inference step)
- **New skill disturbs hellebuyck routing** — adding a new classification row could shadow an existing one; mitigation: the trigger phrases ("test strength", "probe invariant coverage") are distinct from all current rows; verify with patch comparison
- **README/docs drift** — updating 4 files touches documentation that is not gated; mitigation: these are additive-only appends to existing tables and bullet lists; no existing content is modified
- **Layer assignment ambiguity** — the issue title says "deterministic test-strength layer" implying Layer 4; risk of mis-classifying it as Layer 5 (probabilistic); mitigation: the rubric is observable-count-based, so it is deterministic and belongs at Layer 4 by the hierarchy's definition
