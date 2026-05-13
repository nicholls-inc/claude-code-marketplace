---
name: extract-code
description: >-
  Compile verified Dafny to Python or Go with Dafny runtime boilerplate stripped.
  Presents clean output ready for project integration, with type mappings and
  property-based test suggestions. Use after /generate-verified produces a verified
  implementation. Triggers: "extract to python", "extract to go", "compile dafny".
argument-hint: "to [python|go]"
---

# /extract-code — Compile & Extract to Python/Go

## Description

Compile a verified Dafny program to Python or Go, strip Dafny runtime boilerplate, and present clean output ready for integration into a project.

## Instructions

You are a formal verification expert helping the user extract verified code to their target language. The Dafny program should already be verified (via `/generate-verified` or the user's own verification).

### Step 1: Determine Target

Check the user's arguments for the target language:
- `to python` or `to py` → Python
- `to go` or `to golang` → Go

If not specified, **infer from repo state**:
- `go.mod` present → Go
- `pyproject.toml` / `setup.py` / `requirements*.txt` / `Pipfile` present → Python
- Both present → use the language with the most source files; report the choice.

Only fall back to asking the user when neither is present or both have comparable source-file counts. Detection is a one-line report ("Detected language: <X> from <evidence path>; pass `to <other>` to override"), not a chat-blocking gate.

### Step 2: Compile

Call `dafny_compile` with the verified Dafny source and target language.

If compilation fails:
- Present the errors clearly
- Check if the source actually verifies first (suggest `/generate-verified` if not)
- Common issues: extern methods without implementations, unsupported features for target

### Step 3: Post-Extraction Checks

After successful compilation, review the extracted code and warn about:

| Detected Pattern | Alert |
|---|---|
| `_dafny.` references remain in output | "Some Dafny runtime types remain in output. These need manual replacement with native equivalents." Provide a mapping table. |
| Large generated code (>200 lines) | "Complex Dafny programs produce verbose output. Review for idiomatic patterns in your target language." |
| `{:extern}` methods in the original source | "Extern methods were not verified—their implementations are trust boundaries. You must provide implementations for these." |
| `ensures` clauses present in Dafny source | "Your verified postconditions can be translated to property-based tests. See Step 4.5 below." |

### Step 4: Present Clean Output

For each output file, present:
1. The file path and content
2. Any Dafny runtime type mappings needed:

**Python type mappings:**
| Dafny Type | Python Replacement |
|---|---|
| `_dafny.BigRational` | `float` or `fractions.Fraction` |
| `_dafny.Seq` | `list` or `tuple` |
| `_dafny.Map` | `dict` |
| `_dafny.Set` | `set` or `frozenset` |
| `_dafny.BigOrdinal` | `int` |

**Go type mappings:**
| Dafny Type | Go Replacement |
|---|---|
| `dafny.Int` | `int` or `*big.Int` |
| `dafny.Real` | `*big.Rat` or `float64` |
| `dafny.Seq` | `[]T` (typed slice) |
| `dafny.Map` | `map[K]V` |
| `dafny.Set` | Custom or `map[T]bool` |

### Step 4.5: Generate Property-Based Tests (write to disk)

Analyze the Dafny source's `ensures` clauses and translate them into property-based test code for the target language. This bridges the abstraction gap between verified Dafny specifications and the extracted code, which may use different types, precision, or mutability semantics.

Write the generated tests to disk so an orchestrator/CI can execute them without the user copy-pasting:

- Python: `tests/test_<extracted_module>_properties.py` (or `<extracted_module>_properties_test.py` if the repo uses suffix convention)
- Go: `<extracted_module>_properties_test.go` (next to the extracted source file)

Report the written paths. The code blocks below show the shape of what is written; the actual files are the artifact.

**For Python (using Hypothesis):**

Generate `@given` decorated test functions that encode each postcondition. For example, given a verified sort function with `ensures forall i :: 0 <= i < |result| - 1 ==> result[i] <= result[i+1]` and `ensures multiset(result) == multiset(input)`:

```python
from hypothesis import given
import hypothesis.strategies as st

@given(st.lists(st.integers()))
def test_sort_is_ordered(xs):
    result = verified_sort(xs)
    for i in range(len(result) - 1):
        assert result[i] <= result[i + 1]

@given(st.lists(st.integers()))
def test_sort_is_permutation(xs):
    result = verified_sort(xs)
    assert sorted(result) == sorted(xs)
```

**For Go (using rapid):**

Generate `rapid.Check` test functions. For example:

```go
func TestSortOrdered(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        xs := rapid.SliceOf(rapid.Int()).Draw(t, "xs")
        result := VerifiedSort(xs)
        for i := 0; i < len(result)-1; i++ {
            if result[i] > result[i+1] {
                t.Fatalf("not sorted at index %d", i)
            }
        }
    })
}
```

**Divergence Warning Table:**

When translating postconditions, watch for these semantic gaps between Dafny and the target language:

| Detected Divergence | Warning |
|---|---|
| Dafny `int` (unbounded) vs fixed-width target integers | Add overflow tests with boundary values (MAX_INT, MIN_INT) |
| Dafny `real` vs target `float` | Floating-point loses precision — add epsilon-tolerance tests |
| Dafny `seq` vs mutable list/slice | Aliasing bugs possible — test with shared references |
| `{:extern}` methods in source | Extern implementations not verified — write focused unit tests |
| Dafny `BigRational` compiled output | Runtime type not native — may need type assertions |

### Step 5.5: Register Spec

After successful extraction, register the verified spec in the project's spec registry (`.crosscheck/specs.json`) by default. This enables `/check-regressions` to detect when future code changes invalidate the verified properties.

**If the registry file does not exist:** create it with `{"version": 1, "specs": []}` and proceed. The registry is the load-bearing artifact for regression detection; opting out at extraction time means the spec is invisible to future runs of `/check-regressions`, which is almost never what the user wants. If the user wants to opt out, they can remove the entry afterward via `jq` or a hand edit — that is a one-line operation and `git revert` is always available.

**If the registry exists:** auto-register without asking. The user opted into the registry by creating it; re-asking each time is admin ceremony.

Report the write ("Registered spec `<id>` in .crosscheck/specs.json"). The user reverts if needed.

**Generate the entry with:**
- `id`: A slug derived from the Dafny method name (e.g., `MaxOfArray` → `max-of-array`)
- `function`: The Dafny method/function name
- `description`: Brief natural-language description of what the spec verifies
- `dafnySource`: Path to the Dafny source file (if saved) or note that it was inline
- `dafnySourceHash`: SHA-256 hash of the Dafny source content
- `extractedCode.file`: Path to the extracted code file
- `extractedCode.function`: Function name in the extracted code
- `extractedCode.language`: `"python"` or `"go"`
- `constraint`: `"hard"` (default — must pass `dafny_verify` on re-check)
- `lastVerified`: Current ISO 8601 timestamp
- `difficulty`: Metrics from the verification run (solver time, resource count, proof hints, trivial flag)
- `trustBoundaries`: Collected from the Abstraction Gap Checklist — any items that represent ongoing trust assumptions

See `skills/check-regressions/references/registry-schema.md` for the full schema reference.

### Step 5: Integration Guidance

Provide brief guidance on:
- How to integrate the extracted code into an existing project
- Any formatting/linting to apply (`black`/`gofmt`)
- Test suggestions to validate the extracted code matches the verified behavior

**Abstraction Gap Checklist:**

The checklist mixes items the agent verified during this run with items only the human or integration-test layer can confirm. Split per byfuglien's rule — pre-fill what the agent observed, keep human-action items as decisions.

```
## Evidence Summary (agent-verified during this run)

- Compilation succeeded via dafny_compile to <language>.
- `_dafny.` runtime references: <count remaining, with file:line refs, or "none">.
- Type mapping table generated for <language>; replacements flagged where they alter precision (e.g., BigRational → float).
- {:extern} methods detected in source: <list, or "none">.
- Property-based tests written to <test file paths>.

## Decisions for Review / Integration

- [ ] Boundary values tested (empty inputs, maximum sizes, zero, negative) — review the PBT files for coverage; add cases if gaps remain.
- [ ] If extern methods exist: their implementations are tested independently — verify those tests exist.
- [ ] Integration tests cover the function in its actual calling context — add at the call site if missing.
- [ ] Dafny limitation gaps don't affect your use case (IO, concurrency, float precision).
- [ ] Informally-stated requirements not in the Dafny spec are covered by other tests.
```

## Arguments

Target language specification.

Examples:
- `/extract-code to python`
- `/extract-code to go`
