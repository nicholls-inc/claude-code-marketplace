// Layer 4 Dafny spec for extractDifficultyMetrics
// Source: mcp-server/src/tools/verify.ts
// Invariant doc: crosscheck/docs/invariants/extractDifficultyMetrics.md
//
// Proof bodies use `assume false` (Mandate 1 — Phase 2 completion target).
// Trivially-true lemmas (I1–I3) are discharged by Dafny's nat type system.
// Opaque predicate bodies are placeholders; their normative contracts are the
// postconditions of the invariant lemmas.

module ExtractDifficultyMetricsSpec {

  // ── Output type ──────────────────────────────────────────────────────────────
  //
  // DifficultyMetrics mirrors the TypeScript return type of extractDifficultyMetrics.
  // Optional fields are modelled as Option<nat> to capture the null | non-negative
  // JavaScript runtime types. Using nat (not int) makes I1–I3 trivially true
  // by Dafny's type system.

  datatype Option<T> = None | Some(value: T)

  datatype DifficultyMetrics = DifficultyMetrics(
    solverTimeMs: Option<nat>,     // null | Math.round(parseFloat(match) * 1000)
    resourceCount: Option<nat>,    // null | parsed resource count from output
    proofHintCount: nat,           // count of calc/assert/forall hint uses in source
    emptyLemmaBodyCount: nat,      // count of lemmas with `{}` or whitespace-only body
    trivialProof: bool             // true iff proofHintCount == 0 and other conditions
  )

  // ── Auxiliary predicates ──────────────────────────────────────────────────────

  // ContainsLemmaKeyword: true iff `source` contains the substring "lemma".
  // Models JavaScript: source.includes("lemma").
  // {:opaque} hides the placeholder body so the verifier treats it as
  // uninterpreted at call sites. Phase 2 replaces the body with a
  // character-level Dafny substring model.
  ghost predicate {:opaque} ContainsLemmaKeyword(source: string) { true }

  // ── Specification function ────────────────────────────────────────────────────
  //
  // ExtractDifficultyMetricsFn is the formal counterpart of the TypeScript function.
  // Body is a placeholder; normative contracts are the invariant lemmas below.
  // Phase 2 replaces this with a verified parser over rawOutput.

  function ExtractDifficultyMetricsFn(source: string, rawOutput: string): DifficultyMetrics {
    DifficultyMetrics(None, None, 0, 0, false)
  }

  // ── Invariant lemmas ──────────────────────────────────────────────────────────
  //
  // Each lemma encodes one invariant from docs/invariants/extractDifficultyMetrics.md.

  // I1: proofHintCount and emptyLemmaBodyCount are always non-negative. (doc §I1)
  //
  // Trivially discharged by the nat type: Dafny's nat is a subtype of int with
  // the invariant 0 ≤ n. No assume false needed.
  lemma I1_NonNegativeCounts(source: string, rawOutput: string)
    ensures var m := ExtractDifficultyMetricsFn(source, rawOutput);
        m.proofHintCount >= 0 && m.emptyLemmaBodyCount >= 0
  {
    // nat ≥ 0 is a Dafny type invariant. Holds by the placeholder function body.
  }

  // I2: solverTimeMs is None or a non-negative integer. (doc §I2)
  //
  // Trivially discharged: Option<nat> can only be None or Some(n) where n: nat ≥ 0.
  lemma I2_SolverTimeMsNoneOrNonNegative(source: string, rawOutput: string)
    ensures var m := ExtractDifficultyMetricsFn(source, rawOutput);
        m.solverTimeMs.None? || m.solverTimeMs.value >= 0
  {
    // Option<nat>.Some.value is a nat, so value ≥ 0 by type.
    // Placeholder body returns None, so None? is true.
  }

  // I3: resourceCount is None or a non-negative integer. (doc §I3)
  lemma I3_ResourceCountNoneOrNonNegative(source: string, rawOutput: string)
    ensures var m := ExtractDifficultyMetricsFn(source, rawOutput);
        m.resourceCount.None? || m.resourceCount.value >= 0
  {
    // Same reasoning as I2.
  }

  // I4: If source contains no "lemma" keyword, emptyLemmaBodyCount is 0. (doc §I4)
  //
  // The empty-lemma regex `/lemma\s+\w+[^{]*\{\s*\}/g` can only match when
  // "lemma" appears in the source string. If "lemma" is absent, the regex
  // produces no matches, so emptyLemmaBodyCount = 0.
  //
  // Phase 2 proof: reveal ContainsLemmaKeyword, perform structural induction
  // over the regex-match semantics modelled as a character scan.
  lemma I4_LemmaCountRequiresLemmaKeyword(source: string, rawOutput: string)
    requires !ContainsLemmaKeyword(source)
    ensures ExtractDifficultyMetricsFn(source, rawOutput).emptyLemmaBodyCount == 0
  { assume false; }

  // I5: trivialProof is true only when proofHintCount is 0. (doc §I5)
  //
  // The TypeScript definition:
  //   trivialProof = (proofHintCount === 0 && emptyLemmaBodyCount > 0)
  //               || (proofHintCount === 0 && solverTimeMs < 2000)
  // Both disjuncts require proofHintCount === 0.
  //
  // Phase 2 proof: unfold the placeholder function body or the real implementation,
  // case-split on the two disjuncts of the trivialProof definition.
  lemma I5_TrivialProofRequiresZeroHints(source: string, rawOutput: string)
    ensures var m := ExtractDifficultyMetricsFn(source, rawOutput);
        m.trivialProof ==> m.proofHintCount == 0
  { assume false; }

  // I5_Contrapositive: non-zero proof hint count implies trivialProof is false. (doc §I5)
  //
  // Stated separately from I5 to make the contrapositive directly accessible to
  // callers without requiring them to apply modus tollens manually.
  lemma I5_Contrapositive(source: string, rawOutput: string)
    ensures var m := ExtractDifficultyMetricsFn(source, rawOutput);
        m.proofHintCount > 0 ==> !m.trivialProof
  { assume false; }

}
