# Round-trip informalization: the two verbatim prompts

This document contains the two prompt templates used by `/intent-check`. They are intentionally verbatim and intentionally strict — the calibration work in `/Users/harry.nicholls/repos/xylem/docs/assurance/next/15-intent-check-calibration.md` identified both as load-bearing fixes for the FP rate (rationale blindness, carve-out blindness, contradictory-output models).

Do not soften either prompt without updating the skill's verification checklist and re-running the kill-criterion math on the rolling window.

## Prompt 1: Back-translator (BLIND to invariant prose)

**Inputs:** `{code}`, `{test}`. **Not** the invariant prose — passing it voids the round-trip.

```
You are a code reviewer describing what a piece of code and its covering test
actually guarantee, in plain English. You have NOT been shown any specification
or intent prose. You must describe only what the code + test enforce, not what
they were meant to enforce.

Output TWO sections, both mandatory, in this exact order.

### Section 1: Behavioural guarantees

Write a single plain-text paragraph describing the behavioural guarantees the
code and test together enforce. Be concrete — name state transitions, quantified
ranges, preserved fields, and what is checked vs. what is merely set up.

### Section 2: Design rationale comments

List every comment block of 3+ lines AND every single-line comment whose text
contains any of these rationale markers (case-insensitive):

  because, since, artefact, artifact, workaround, zeroed, intentional,
  skipping, ignore, on purpose, deliberately, known, caveat

For each, output:
  <file:line-range>
  > <the comment body, verbatim, preserving internal line breaks>

If no such comments exist, output exactly:
  None.

Do NOT summarise rationale comments. Quote them verbatim. The diff-checker
treats Section 2 as authoritative evidence when a gap is apparent; if you
paraphrase, you destroy that evidence.

---

CODE:
{code}

---

TEST:
{test}
```

### Why Section 2 is mandatory

Calibration finding FP #6 (xylem session 2026-04-23): a test at
`queue_invariants_prop_test.go:452-457` carried a 6-line rationale explaining
that clock values were zeroed because of wall-clock drift between runs. The
earlier single-paragraph back-translator described the *behaviour* accurately
("ignores clock values") but did not surface the *rationale*. The diff-checker
then compared the literal behaviour against the spec's literal wording and
flagged a substantive gap — a false positive.

Forcing Section 2 to list rationale comments verbatim, with file+line anchors,
makes the rationale first-class evidence rather than an optional summary.

## Prompt 2: Diff-checker (sees both original intent + back-translation)

**Inputs:** `{invariant_prose}`, `{back_translation}` (both sections from Prompt 1).

```
You are evaluating whether a back-translation of a code+test pair matches the
original invariant prose. Your job is to decide if any apparent gap is
substantive, non-substantive, or the artefact of a scope carve-out.

You MUST complete Step 1 before rendering any verdict. Steps are ordered
because forcing visible intermediate output is more reliable than instructing
you to "consider" something.

## Step 1: Scan for scope carve-outs

Search the invariant prose for these scope markers (case-insensitive):

  "Not covered", "caller-responsibility", "precondition", "aspirational",
  "known violation", "privileged", "exempt", "out of scope", "does not apply"

List every carve-out you find:
  - Quote the clause verbatim.
  - State what it exempts from the guarantee.

If none found, write exactly: "No carve-outs found."

Also read Section 2 of the back-translation (design rationale comments). Any
apparent gap that is explained by a rationale comment is NOT substantive.

## Scope modifier taxonomy (apply to each carve-out you find)

| Marker                                         | Meaning                                                                                          |
|------------------------------------------------|--------------------------------------------------------------------------------------------------|
| "caller-responsibility" / "precondition"       | The guarantee is conditional. A test that does not exercise the violation case is NOT a gap.     |
| "Not covered" / "out of scope" / "exempt"      | Explicitly excluded. Any difference in this area is non-substantive by definition.               |
| "aspirational"                                 | Future-direction note, not a current guarantee. Never flag as a gap.                             |
| "known violation"                              | Acknowledged divergence. Non-substantive.                                                        |
| "privileged"                                   | Restricted code-path. Only the privileged path needs to satisfy the invariant.                   |

## Step 2: Evaluate each apparent gap

For each behavioural difference between the invariant prose and the
back-translation, apply the taxonomy from Step 1. A gap is substantive only
if NONE of the carve-outs or rationale comments apply to it.

## Step 3: Emit the verdict

Output a single JSON object with exactly these keys:

{
  "match": true | false,
  "mismatch_reason": "<string, >= 20 chars when match=false, empty when match=true>",
  "mismatch_category": one of:
      "spec_scope_mismatch",   // gap falls within a carve-out; non-substantive
      "weaker_guarantee",      // test proves strictly less than spec claims
      "missing_property",      // a specific property is untested
      "missing_coverage",      // a subset of inputs/states is uncovered
      "rationale_explains",    // in-source comment explains the divergence
      "carve_out_applies",     // scope modifier excludes this area
      "clean_match"            // no gap found
  ,
  "confidence_pct": 0-100,
  "confidence_basis": one of:
      "carve-out-found",       // scope modifier applied
      "rationale-found",       // in-source justification comment found
      "rationale-absent",      // no justification; gap is literal
      "spec-ambiguous",        // spec prose unclear
      "code-ambiguous"         // code unclear; verdict uncertain
}

`confidence_pct` is your confidence that your VERDICT is correct — not your
confidence that the code is correct. 0 = you are guessing; 100 = you are
certain.

Do not emit prose outside the JSON object.

---

INVARIANT PROSE:
{invariant_prose}

---

BACK-TRANSLATION (both sections from the blind back-translator):
{back_translation}
```

### Why Step 1 is mandatory (not merely recommended)

Calibration: the diff-checker has been observed to capture the spec's main
claim but silently ignore conditional scope modifiers when rendering a verdict.
This is a latent failure mode that hides behind partially-correct findings.
Forcing a *visible* Step 1 output — quoted clauses, classified by taxonomy —
means the model cannot silently skip the scan.

## Semantic validation (applied by Claude after parsing, not part of the prompt)

After unmarshaling the diff-checker's JSON, the skill applies two defensive
rules:

1. `match == true` AND `mismatch_reason` non-empty → contradictory output.
   Flip to `match=false, confidence_pct=40, confidence_basis="spec-ambiguous",
   mismatch_category="missing_property"`. Fail-closed interpretation is safer
   than trusting either half of the contradiction.
2. `match == false` AND `len(strip(mismatch_reason)) < 20` → reject as
   truncation. Ask the user to re-run.

These rules live in the skill, not the prompt, so they catch model
contradictions even if the prompt is tampered with.

## Placeholder expansion order (exact)

| Placeholder         | Source                                                      |
|---------------------|-------------------------------------------------------------|
| `{code}`            | Staged code diff relevant to the invariant (scoped, not whole PR) |
| `{test}`            | Covering property test body, including surrounding rationale comments |
| `{invariant_prose}` | The relevant invariant section from `docs/invariants/<module>.md` |
| `{back_translation}`| Section 1 + Section 2 from Prompt 1, verbatim               |

Claude must substitute these placeholders literally, without paraphrasing,
truncating, or re-formatting the inputs. If a file is too large to fit in the
context window, scope it down rather than summarising it — a summary by the
same LLM that runs the pipeline is contamination.
