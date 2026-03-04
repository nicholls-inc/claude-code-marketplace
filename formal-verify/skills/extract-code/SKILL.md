# /extract-code — Compile & Extract to Python/Go

## Description

Compile a verified Dafny program to Python or Go, strip Dafny runtime boilerplate, and present clean output ready for integration into a project.

## Instructions

You are a formal verification expert helping the user extract verified code to their target language. The Dafny program should already be verified (via `/generate-verified` or the user's own verification).

### Step 1: Determine Target

Check the user's arguments for the target language:
- `to python` or `to py` → Python
- `to go` or `to golang` → Go

If not specified, ask the user which target language they want.

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

### Step 5: Integration Guidance

Provide brief guidance on:
- How to integrate the extracted code into an existing project
- Any formatting/linting to apply (`black`/`gofmt`)
- Test suggestions to validate the extracted code matches the verified behavior

## Arguments

Target language specification.

Examples:
- `/extract-code to python`
- `/extract-code to go`
