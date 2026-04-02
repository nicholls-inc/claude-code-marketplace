# Crosscheck Demo

Six demonstrations using real SWE-bench tasks to show how crosscheck
produces rigorous, verifiable results that vanilla Claude cannot match.

Demos 1-2 use SWE-bench Verified. Demos 3-6 use SWE-bench_Pro (harder
problems less likely to be in training data).

## Prerequisites

- **Claude Code** with the crosscheck plugin installed
- **Docker** running (Demo 1 only)
- **Python 3.9+** with pytest

## Pre-Demo Checklist

```bash
# Build the Dafny Docker image (if not already built)
cd crosscheck && ./scripts/build-docker.sh

# Pre-warm the container (avoids cold-start delay)
docker run --rm crosscheck-dafny:latest --version

# Verify Demo 1 tests
cd demo/01_formal_verification
pytest test_mod.py -v

# Verify Demo 2 reproducer
cd ../02_semiformal_reasoning
python reproduce_error.py

# Verify Demo 3-6 tests (all should show expected failures)
cd ../03_structured_reasoning && pytest test_coerce.py -v
cd ../04_execution_tracing && pytest test_hostblock.py -v
cd ../05_patch_comparison && pytest test_validate.py -v
cd ../06_test_adequacy && pytest test_parse_duration.py -v
```

## Demo 1: Formal Verification (6-7 min)

**SWE-bench Verified: `sympy__sympy-13177`**

SymPy's `Mod` function simplifies `x**n % x` to `0` unconditionally,
but `1.5**2 % 1.5 = 0.75`. The simplification is only valid when both
values are integers and the exponent is positive — three conditions the
code was missing.

Crosscheck formally proves WHEN `x^n mod x = 0` using Dafny. The
`requires` clauses in the proof are exactly the missing conditions
from SymPy's code.

Skills: `/spec-iterate` -> `/generate-verified` -> `/extract-code`

See [01_formal_verification/SCRIPT.md](01_formal_verification/SCRIPT.md)

## Demo 2: Fault Localization (6-7 min)

**SWE-bench Verified: `scikit-learn__scikit-learn-14087`**

`LogisticRegressionCV(refit=False)` crashes with `IndexError`. The
traceback points at array indexing — but the root cause is 25 lines
earlier, where `self.multi_class` (user input `'auto'`) is used instead
of `multi_class` (resolved to `'ovr'`). A 5-character fix.

Crosscheck's `/locate-fault` traces from crash site to root cause
through a structured 4-phase analysis with file:line evidence.

Skill: `/locate-fault`

See [02_semiformal_reasoning/SCRIPT.md](02_semiformal_reasoning/SCRIPT.md)

## Demo 3: Structured Reasoning (~5 min)

**SWE-bench_Pro: `ansible__ansible-d33bedc4`**

A type coercion function has 5 interacting bugs: bool→int loses tags,
unhashable types crash boolean conversion, bytes treated as Sequence,
floats silently truncate. Claude finds 1-2; `/reason` finds all 5 by
systematically checking every property against every code path.

Skill: `/reason`

See [03_structured_reasoning/SCRIPT.md](03_structured_reasoning/SCRIPT.md)

## Demo 4: Execution Tracing (~5:30 min)

**SWE-bench_Pro: `qutebrowser__qutebrowser-c580ebf0`**

A host blocker does exact-match lookups, so blocking `example.com`
misses `sub.example.com`. The `widened_hostnames()` helper EXISTS in
the same file but isn't called. There's also a whitelist ordering bug.
Claude finds the first issue; `/trace-execution` builds the complete
call graph and reveals both.

Skill: `/trace-execution`

See [04_execution_tracing/SCRIPT.md](04_execution_tracing/SCRIPT.md)

## Demo 5: Patch Comparison (~6:30 min)

**SWE-bench_Pro: `ansible__ansible-f327e65d`**

A name validator accepts Python keywords (`def.collection`). Two
patches fix it: one uses `keyword.iskeyword()`, the other uses a
hardcoded frozenset that over-rejects soft keywords (`match`, `case`,
`type`) and builtins (`list`, `dict`). Claude says "Patch A is cleaner."
`/compare-patches` constructs specific counterexamples.

Skill: `/compare-patches`

See [05_patch_comparison/SCRIPT.md](05_patch_comparison/SCRIPT.md)

## Demo 6: Test Adequacy (~5 min)

**SWE-bench_Pro: `qutebrowser__qutebrowser-96b99780`**

A duration parser has 20 passing tests with 100% line coverage. But
`'5s10s'` silently returns 5000ms (takes first match, ignores second).
Claude says "tests look comprehensive." `/rationale` builds a claim tree
exposing the duplicate-unit gap.

Skill: `/rationale`

See [06_test_adequacy/SCRIPT.md](06_test_adequacy/SCRIPT.md)

## Recommended Presentation Order

**Quick demo (2 picks, ~12 min):**
1. Demo 2 (fault localization) — dramatic "crash site != root cause"
2. Demo 5 (patch comparison) — concrete counterexamples

**Full showcase (all 6, ~35 min):**
1. Demo 2 (fault localization) — no Docker, fast, dramatic
2. Demo 3 (structured reasoning) — "5 bugs, found them all"
3. Demo 5 (patch comparison) — "both patches fix it, one is wrong"
4. Demo 6 (test adequacy) — "100% coverage, zero safety"
5. Demo 4 (execution tracing) — "the function exists, just not called"
6. Demo 1 (formal verification) — the mathematical proof finish

## Q&A Talking Points

**"How long does verification take?"**
5-30 seconds per Dafny verification attempt. The MCP server has a
120-second timeout. Semi-formal reasoning runs at text generation
speed (no Docker needed).

**"These are real SWE-bench tasks?"**
Yes. Demos 1-2 from SWE-bench Verified, Demos 3-6 from SWE-bench_Pro.
The bugs, code patterns, and test expectations are derived from real
open-source projects (SymPy, scikit-learn, Ansible, qutebrowser).

**"Would Claude solve these without crosscheck?"**
It might get close, but without structure. For Demo 2, Claude fixates
on the crash site. For Demo 3, Claude finds 1-2 of 5 bugs. For Demo 5,
Claude says "Patch A is simpler" without constructing counterexamples.
For Demo 6, Claude says "tests look good" on a suite with a silent
data loss hole. The skills enforce systematic methodology.

**"What about languages other than Python?"**
Dafny compiles to Python and Go. Semi-formal reasoning works on
any language Claude can read.

**"How is this different from just asking Claude to think harder?"**
Crosscheck's skills enforce a specific methodology. /locate-fault
MUST complete 4 phases. /reason MUST gather premises with file:line
evidence and check alternative hypotheses. /spec-iterate MUST produce
machine-verified postconditions. /rationale MUST build a claim tree
with verification method classifications. The structure prevents
shortcuts.
