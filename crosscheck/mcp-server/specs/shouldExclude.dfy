// Layer 4 Dafny spec for shouldExclude
// Source: mcp-server/src/tools/compile.ts
// Invariant doc: crosscheck/docs/invariants/shouldExclude.md
//
// shouldExclude is a pure boolean predicate — its Dafny model can be more
// concrete than parseDafnyOutput since the logic is direct string containment
// and basename extraction with no regex complexity.
//
// Invariant lemmas I1–I3 have verified (non-assume) proofs.
// Lemmas I4–I6 use `assume false` per Mandate 1 where BaseName reasoning
// requires additional auxiliary lemmas not yet developed.

module ShouldExcludeSpec {

  // ── Target type ───────────────────────────────────────────────────────────────

  datatype Target = Py | Go

  // ── String helpers ────────────────────────────────────────────────────────────

  // Contains: true iff `sub` occurs as a contiguous subsequence in `s`.
  predicate Contains(s: string, sub: string) {
    |sub| == 0 ||
    (|sub| <= |s| &&
     exists i :: 0 <= i <= |s| - |sub| && s[i..i + |sub|] == sub)
  }

  // BaseName: characters after the last '/' in `path`, or `path` itself
  // if no '/' is present.  Models Node.js `path.basename(filePath)`.
  function BaseName(path: string): string {
    var i := LastSlashIndex(path);
    if i < 0 then path else path[i + 1..]
  }

  // LastSlashIndex: the index of the rightmost '/' in `s`, or -1 if none.
  function LastSlashIndex(s: string): int
    ensures -1 <= LastSlashIndex(s)
    ensures LastSlashIndex(s) < |s|
    ensures LastSlashIndex(s) >= 0 ==> s[LastSlashIndex(s)] == '/'
    ensures LastSlashIndex(s) < 0 ==> forall i :: 0 <= i < |s| ==> s[i] != '/'
  {
    LastSlashFrom(s, |s| - 1)
  }

  function LastSlashFrom(s: string, i: int): int
    requires -1 <= i < |s| || (|s| == 0 && i == -1)
    ensures -1 <= LastSlashFrom(s, i)
    ensures LastSlashFrom(s, i) < |s|
    ensures LastSlashFrom(s, i) >= 0 ==> s[LastSlashFrom(s, i)] == '/'
    ensures LastSlashFrom(s, i) < 0 ==>
        forall j :: 0 <= j <= i ==> s[j] != '/'
    decreases i + 1
  {
    if i < 0 then -1
    else if s[i] == '/' then i
    else LastSlashFrom(s, i - 1)
  }

  // ── Main predicate ────────────────────────────────────────────────────────────
  //
  // ShouldExclude mirrors the TypeScript function exactly.
  // Exclusion lists from compile.ts:
  //   PYTHON_EXCLUDE_FILES = ["_dafny.py", "__pycache__"]
  //   GO_EXCLUDE_FILES     = ["dafny.go", "System_.go"]
  //   GO_EXCLUDE_DIRS      = ["dafny", "System_"]  (checked as /dir/ segment)

  predicate ShouldExclude(filePath: string, target: Target) {
    var name := BaseName(filePath);
    match target {
      case Py =>
        // name === ex || filePath.includes(ex) for each ex in PYTHON_EXCLUDE_FILES
        name == "_dafny.py" || Contains(filePath, "_dafny.py") ||
        name == "__pycache__" || Contains(filePath, "__pycache__")
      case Go =>
        // GO_EXCLUDE_FILES check (basename equality)
        name == "dafny.go" || name == "System_.go" ||
        // GO_EXCLUDE_DIRS check (path segment containment)
        Contains(filePath, "/dafny/") || Contains(filePath, "/System_/")
    }
  }

  // ── Invariant lemmas ──────────────────────────────────────────────────────────

  // I1: Paths containing /dafny/ are excluded for Go target. (doc §I1)
  // This lemma has a verified proof — no assume false needed.
  lemma I1_DafnyDirExcludedForGo(filePath: string)
    requires Contains(filePath, "/dafny/")
    ensures ShouldExclude(filePath, Go)
  {
    // Unfolds directly: Go branch includes `Contains(filePath, "/dafny/")`.
  }

  // I2: Paths containing /System_/ are excluded for Go target. (doc §I2)
  lemma I2_SystemDirExcludedForGo(filePath: string)
    requires Contains(filePath, "/System_/")
    ensures ShouldExclude(filePath, Go)
  {
    // Unfolds directly: Go branch includes `Contains(filePath, "/System_/")`.
  }

  // I3: Paths containing _dafny.py are excluded for Python target. (doc §I3)
  lemma I3_DafnyPyExcludedForPy(filePath: string)
    requires Contains(filePath, "_dafny.py")
    ensures ShouldExclude(filePath, Py)
  {
    // Unfolds directly: Py branch includes `Contains(filePath, "_dafny.py")`.
  }

  // I4: Paths containing __pycache__ are excluded for Python target. (doc §I4)
  lemma I4_PycacheExcludedForPy(filePath: string)
    requires Contains(filePath, "__pycache__")
    ensures ShouldExclude(filePath, Py)
  {
    // Unfolds directly: Py branch includes `Contains(filePath, "__pycache__")`.
  }

  // I5: Safe filenames (lowercase letters + .py) are not excluded for Python.
  // A filename in { s + ".py" | s ∈ [a-z]{1,8} } is never excluded. (doc §I5)
  // Proof requires characterising BaseName and Contains for safe inputs.
  lemma I5_SafePyFilenameNotExcluded(name: string)
    requires IsSafePyFilename(name)
    ensures !ShouldExclude(name, Py)
  { assume false; }  // Mandate 1: BaseName and Contains reasoning deferred

  // I6: Target isolation — Python patterns don't affect Go target. (doc §I6)
  // Specifically: "_dafny.py" as a bare path is not excluded for Go.
  lemma I6a_DafnyPyNotExcludedForGo()
    ensures !ShouldExclude("_dafny.py", Go)
  {
    // BaseName("_dafny.py") = "_dafny.py" (no slash, so whole string is basename)
    // Go branch: name == "dafny.go"? No. name == "System_.go"? No.
    // Contains("_dafny.py", "/dafny/")? No — "/dafny/" has length 7, "_dafny.py" has length 9
    //   but "_dafny.py"[0..7] == "_dafny." != "/dafny/", so no match.
    // Contains("_dafny.py", "/System_/")? No.
    assume false;  // Mandate 1: formal substring proof deferred
  }

  // I6b: "__pycache__" as a bare path is not excluded for Go.
  lemma I6b_PycacheNotExcludedForGo()
    ensures !ShouldExclude("__pycache__", Go)
  { assume false; }  // Mandate 1

  // ── Helper predicate: safe Python filename ─────────────────────────────────────

  // IsSafePyFilename: the path consists of 1–8 lowercase letters followed by ".py",
  // with no path separators and no patterns matching any Python exclusion rule.
  predicate IsSafePyFilename(path: string) {
    1 <= |path| <= 11 &&
    path[|path| - 3..] == ".py" &&
    (forall i :: 0 <= i < |path| - 3 ==> 'a' <= path[i] <= 'z') &&
    !Contains(path, "_dafny.py") &&
    !Contains(path, "__pycache__") &&
    !Contains(path, "/")
  }

}
