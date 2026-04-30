// Layer 4 Dafny spec for parseDafnyOutput
// Source: mcp-server/src/tools/verify.ts
// Invariant doc: crosscheck/docs/invariants/parseDafnyOutput.md
//
// Proof bodies use `assume false` (Mandate 1 — Phase 2 completion target).
// Opaque predicate bodies are placeholders; their normative contracts are the
// postconditions of the invariant lemmas.

module ParseDafnyOutputSpec {

  // ── Output type ──────────────────────────────────────────────────────────────

  datatype ParseResult = ParseResult(errors: seq<string>, warnings: seq<string>)

  // ── Auxiliary predicates ──────────────────────────────────────────────────────
  //
  // Each predicate models a JavaScript regex test on a trimmed line.
  // {:opaque} hides the placeholder body so the verifier treats the predicate
  // as uninterpreted at call sites that have not called reveal.
  // Phase 2 replaces the placeholder body with a character-level Dafny model.

  // ContainsErrorCI — mirrors: /Error/i.test(line)
  ghost predicate {:opaque} ContainsErrorCI(line: string) { true }

  // ContainsWarningCI — mirrors: /Warning/i.test(line)
  ghost predicate {:opaque} ContainsWarningCI(line: string) { true }

  // IsVerifierSummaryLine — mirrors: /^Dafny program verifier/.test(line)
  ghost predicate {:opaque} IsVerifierSummaryLine(line: string) { true }

  // ── Specification function ────────────────────────────────────────────────────
  //
  // ParseDafnyOutputFn is the formal counterpart of the TypeScript function.
  // Body is a placeholder; normative contracts are the invariant lemmas below.
  // Phase 2 replaces this with a verified recursive line-by-line parser.

  function ParseDafnyOutputFn(stdout: string, stderr: string): ParseResult {
    ParseResult([], [])
  }

  // ── Invariant lemmas ──────────────────────────────────────────────────────────
  //
  // Each lemma encodes one invariant from docs/invariants/parseDafnyOutput.md.
  // `assume false` in the body is a Mandate 1 placeholder; Phase 2 replaces it
  // with a structural induction on the line decomposition of (stdout, stderr).

  // I1: All errors contain "error" (case-insensitive). (doc §I1)
  lemma I1_ErrorsContainErrorCI(stdout: string, stderr: string)
    ensures forall e :: e in ParseDafnyOutputFn(stdout, stderr).errors
        ==> ContainsErrorCI(e)
  { assume false; }

  // I2: All warnings contain "warning" (case-insensitive). (doc §I2)
  lemma I2_WarningsContainWarningCI(stdout: string, stderr: string)
    ensures forall w :: w in ParseDafnyOutputFn(stdout, stderr).warnings
        ==> ContainsWarningCI(w)
  { assume false; }

  // I3: Errors and warnings are disjoint. (doc §I3)
  lemma I3_ErrorsWarningsDisjoint(stdout: string, stderr: string)
    ensures forall s :: s in ParseDafnyOutputFn(stdout, stderr).errors
        ==> s !in ParseDafnyOutputFn(stdout, stderr).warnings
  { assume false; }

  // I4: Error classification takes precedence over Warning. (doc §I4)
  // A line in errors is never simultaneously in warnings.
  // Named separately from I3 to encode the priority rule explicitly.
  lemma I4_ErrorPrecedenceOverWarning(stdout: string, stderr: string)
    ensures forall s :: s in ParseDafnyOutputFn(stdout, stderr).errors
        ==> s !in ParseDafnyOutputFn(stdout, stderr).warnings
  { assume false; }

  // I5: Verifier summary lines are excluded from errors. (doc §I5)
  // Lines starting with "Dafny program verifier" never appear in errors.
  lemma I5_VerifierSummaryExcludedFromErrors(stdout: string, stderr: string)
    ensures forall e :: e in ParseDafnyOutputFn(stdout, stderr).errors
        ==> !IsVerifierSummaryLine(e)
  { assume false; }

  // I6: Empty input produces empty output. (doc §I6)
  lemma I6_EmptyInputEmptyOutput()
    ensures var r := ParseDafnyOutputFn("", "");
        r.errors == [] && r.warnings == []
  {
    // Witnesses directly by the placeholder function body.
    // Phase 2 proof holds by case split on empty input string.
  }

  // I7: |errors| + |warnings| ≤ non-empty line count in combined input. (doc §I7)
  // The parser never invents lines that do not appear in the input.
  // Full proof requires a formal model of string line decomposition.
  lemma I7_CategorizationBoundedByInput(stdout: string, stderr: string)
    ensures |ParseDafnyOutputFn(stdout, stderr).errors|
          + |ParseDafnyOutputFn(stdout, stderr).warnings|
         <= CountNonEmptyLines(stdout + "\n" + stderr)
  { assume false; }

  // ── Helper: count non-empty lines ────────────────────────────────────────────

  // CountNonEmptyLines: models TypeScript's
  //   combined.split("\n").filter(l => l.trim()).length
  // Implemented as a recursive scan over the character sequence.
  function CountNonEmptyLines(s: string): nat {
    CountLinesFrom(s, 0, 0, false)
  }

  // Scan `s` from position `i`, accumulating non-empty line count in `acc`.
  // `inNonEmpty` tracks whether the current line has seen a non-whitespace char.
  function CountLinesFrom(s: string, i: int, acc: nat, inNonEmpty: bool): nat
    requires 0 <= i <= |s|
    decreases |s| - i
  {
    if i == |s| then
      if inNonEmpty then acc + 1 else acc
    else if s[i] == '\n' then
      CountLinesFrom(s, i + 1, if inNonEmpty then acc + 1 else acc, false)
    else
      var nonWS := s[i] != ' ' && s[i] != '\t' && s[i] != '\r';
      CountLinesFrom(s, i + 1, acc, inNonEmpty || nonWS)
  }

}
