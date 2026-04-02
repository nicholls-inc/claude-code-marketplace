# Demo 5: Patch Comparison — SWE-bench_Pro ansible FQCN validation

**"Both patches fix the keyword bug. Only one is correct."**

## The bug

A name validator for Python-based collections accepts Python keywords
(`def`, `class`, `return`) as valid names. Since collections map to
Python packages (`import ansible_collections.def.return`), keyword names
cause SyntaxError at import time.

Two patches fix the keyword rejection. They look similar but differ
semantically on soft keywords (`match`, `case`, `type`).

Source: SWE-bench_Pro `ansible__ansible-f327e65d11bb905ed9f15996024f857a95592629`

## Setup (before the demo)

- No Docker needed
- Open Claude Code in this directory

## Step 1: Show the Bug (1 min)

```bash
pytest test_validate.py -v -k "keyword" 2>&1 | tail -20
```

> "The validator uses `str.isidentifier()` to check names. Problem is,
> `'def'.isidentifier()` returns True. Python keywords ARE valid
> identifiers — they're just reserved. Two developers submitted patches."

## Step 2: Show Both Patches Side by Side (1 min)

```bash
diff -u validate_name_bug.py patch_a.py
diff -u validate_name_bug.py patch_b.py
```

Point out:
- **Patch A**: Adds `and not keyword.iskeyword(name)` — uses stdlib
- **Patch B**: Adds `and name not in _RESERVED_NAMES` — hardcoded frozenset

> "Both patches reject `def.collection` and `import.utils`. The question
> is: are they semantically equivalent? Do they accept and reject exactly
> the same inputs?"

## Step 3: Ask Vanilla Claude to Compare (1-2 min)

```
Look at patch_a.py and patch_b.py. Both fix the keyword validation bug
in validate_name_bug.py. Are these two patches semantically equivalent?
Do they accept and reject exactly the same set of inputs?
```

**Expected vanilla response:** Claude typically says:
- "Patch B is more restrictive because it also blocks builtins like `list`, `dict`"
- May or may not notice the soft keyword difference
- Often concludes "Patch A is better because it's simpler and uses stdlib"
- Rarely constructs a SPECIFIC counterexample showing different behavior

**The real answer:** The patches differ in THREE ways:
1. **Soft keywords** (match, case, type): Patch A accepts them (correct — they're valid identifiers). Patch B rejects them (wrong — they're context-dependent, usable as names).
2. **Builtins** (list, dict, str): Patch A accepts them (correct — they shadow builtins but are legal). Patch B rejects them (over-restrictive).
3. **Future keywords**: Patch A auto-updates with Python. Patch B is frozen at 3.10.

## Step 4: Use /compare-patches (2-3 min)

```
/compare-patches "Compare patch_a.py and patch_b.py as fixes for the
keyword validation bug in validate_name_bug.py. Are they semantically
equivalent? If not, provide specific inputs that produce different
results and explain which patch is correct."
```

**What the audience sees:**

**Structural diff:**
- Patch A: `keyword.iskeyword(name)` — dynamic, stdlib-backed
- Patch B: `name not in _RESERVED_NAMES` — static frozenset

**Counterexample 1: Soft keywords**
```
Input: 'match.pattern'
Patch A: is_valid_collection_name → True  (keyword.iskeyword('match') → False)
Patch B: is_valid_collection_name → False ('match' in _RESERVED_NAMES → True)
```

Verdict: Patch A is correct. `match` is a soft keyword in Python 3.10+ — it's
context-dependent and CAN be used as an identifier. `import collections.match.pattern`
is valid Python.

**Counterexample 2: Builtins**
```
Input: 'list.tools'
Patch A: is_valid_collection_name → True  (keyword.iskeyword('list') → False)
Patch B: is_valid_collection_name → False ('list' in _RESERVED_NAMES → True)
```

Verdict: Patch A is correct. `list` is a builtin, not a keyword. Shadowing it
is legal (though discouraged). `import collections.list.tools` works.

**Counterexample 3: Maintenance burden**
Patch B hardcodes Python 3.10 keywords. If Python 3.13 adds a new keyword,
Patch B won't reject it until someone manually updates the set. Patch A
uses `keyword.iskeyword()` which tracks the running interpreter automatically.

**Key moment:** When the counterexample runs:

> "The /compare-patches analysis didn't just say 'they're different' —
> it constructed THREE specific inputs where the patches disagree, ran
> both, and showed the output. `match.pattern` is the killer: Patch B
> rejects a valid collection name because it confused soft keywords
> with hard keywords."

## Step 5: Verify with Tests (30s)

```bash
# Run soft keyword tests against both patches
python -c "
from patch_a import is_valid_collection_name as check_a
from patch_b import is_valid_collection_name as check_b

for name in ['match.pattern', 'case.handler', 'type.checker', 'list.tools']:
    print(f'{name:20s}  A={check_a(name)}  B={check_b(name)}')
"
```

> "Patch B over-rejects. In a collection ecosystem, this means
> `community.match` or `community.type` — perfectly valid names —
> would be rejected. That's a breaking change disguised as a bug fix."

## Timing Budget

| Step | Duration |
|------|----------|
| Show the bug | 1:00 |
| Show both patches | 1:00 |
| Vanilla Claude | 1:30 |
| /compare-patches | 2:30 |
| Verify | 0:30 |
| **Total** | **~6:30** |

## Contingencies

**Claude correctly identifies the soft keyword difference:**
"Good analysis. But the /compare-patches certificate goes further —
it also identifies the builtins issue and the maintenance burden.
And it provides runnable counterexamples, not just descriptions."

**/compare-patches doesn't find the builtins difference:**
"It found the critical one: soft keywords. That's the difference
between a correct fix and a breaking change. The builtins issue is
a bonus — the soft keyword issue is the deal-breaker."
