# Demo 2: Semi-formal Reasoning — SWE-bench scikit-learn__scikit-learn-14087

**"The crash site is not the root cause."**

## The bug

`LogisticRegressionCV(refit=False).fit(X, y)` throws `IndexError:
too many indices for array`. The crash is at array indexing (line 2194).
The root cause is 25 lines earlier: `self.multi_class` (raw user input
`'auto'`) is used instead of `multi_class` (resolved to `'ovr'`).

Source: https://github.com/scikit-learn/scikit-learn/issues/14059

## Setup (before the demo)

- No Docker needed
- Run `reproduce_error.py` from this directory (the parent)
- Open Claude Code in the `workspace/` subdirectory — its
  `.claude/settings.local.json` prevents Claude from reading
  files outside `workspace/`, so it can't see this script or
  the reproducer's root-cause explanation

## Step 1: Show the Crash (1 min)

```bash
python reproduce_error.py
```

Output: `IndexError` with explanation showing the crash site.

> "This is a real SWE-bench task. LogisticRegressionCV crashes when
> refit=False. The traceback points to an array indexing expression.
> Let's see what Claude does with this."

## Step 2: Ask Vanilla Claude to Fix It (1-2 min)

```
Look at logistic_cv.py. It crashes with:

  IndexError: too many indices for array

at the line: class_coefs[:, i, best_indices[i], :]

when called with LogisticRegressionCV(multi_class='auto', refit=False).fit(X, y)
where y has 2 classes. What's the root cause and how do you fix it?
```

**Expected vanilla response:** Claude will likely focus on the crash
site — the 4D array indexing expression. It may suggest:
- Reshaping the array
- Adding a dimension check
- Changing the indexing pattern
- Or it may notice `self.multi_class` but not explain WHY it matters

**If Claude nails it:** Rare but possible. Pivot to: "Claude got it,
but watch the difference in HOW it gets there. No evidence chain, no
alternative hypotheses, no structured trace. In a real codebase with
10,000 lines, you need the structured approach."

## Step 3: Use /locate-fault (2-3 min)

```
/locate-fault "LogisticRegressionCV.fit() crashes with IndexError:
too many indices for array at the line class_coefs[:, i, best_indices[i], :]
when refit=False and multi_class='auto'. The error is in logistic_cv.py."
```

**What the audience sees — the 4-phase analysis:**

**Phase 1 — Test Semantics:**
- The error is IndexError at 4D indexing of a 3D array
- refit=False triggers the code path
- multi_class='auto' is the triggering parameter

**Phase 2 — Code Path Tracing:**
- Traces from `fit()` entry → `_check_multi_class()` resolves 'auto' to 'ovr'
- Local `multi_class` = 'ovr' (line ~80)
- But `self.multi_class` remains 'auto' (never reassigned)
- Traces to the branching point (line ~120 in extracted code):
  `if self.multi_class == 'ovr'` — evaluates to False!
- Takes the else branch (multinomial indexing) on OVR-shaped data

**Phase 3 — Divergence Analysis:**
- Expected: OVR mode → 3D indexing (`coefs_paths[i, best_indices[i], :]`)
- Actual: multinomial branch → 4D indexing (`coefs_paths[:, i, best_indices[i], :]`)
- Mismatch because wrong variable determined the branch

**Phase 4 — Ranked Predictions:**
1. **Root cause** (HIGH confidence): `self.multi_class` should be `multi_class`
   at the branching condition. `self.multi_class` retains 'auto', while
   `multi_class` holds the resolved value.
   - File: logistic_cv.py, line ~120
   - Evidence: `_check_multi_class()` at line ~80 resolves 'auto' → 'ovr',
     but the check 40 lines later uses the un-resolved attribute

**Key moment:** When Phase 2 reveals the two variables:

> "There are TWO variables named multi_class in this code. The local
> variable was resolved from 'auto' to 'ovr'. The instance attribute
> `self.multi_class` still holds 'auto'. The bug is using the wrong one.
>
> This is a 5-character fix: remove `self.`. But finding it required
> tracing from the crash site through the variable resolution logic,
> 25 lines upstream."

## Step 4: Show the Actual Fix (30s)

```python
# The actual SWE-bench fix (from the PR):
# Line 2170:
-  if self.multi_class == 'ovr':
+  if multi_class == 'ovr':
```

> "Five characters. That's the fix. But without structured reasoning
> that traces execution paths and checks alternative hypotheses, you'd
> be staring at the array indexing trying to fix the symptom."

## Step 5: Closing (30s)

> "The /locate-fault certificate forced Claude through 4 phases:
> understand the test, trace the code path, find the divergence, rank
> the predictions. Every claim cites a line number. You can audit the
> reasoning, not just trust the conclusion."
>
> "This is the difference between 'I think the bug is here' and 'here's
> a structured proof that the bug is here, with evidence you can verify.'"

## Timing Budget

| Step | Duration |
|------|----------|
| Show the crash | 1:00 |
| Vanilla Claude | 1:30 |
| /locate-fault | 2:30 |
| Show actual fix | 0:30 |
| Closing | 0:30 |
| **Total** | **~6:30** |

## Contingencies

**Vanilla Claude correctly identifies self.multi_class as the issue:**
"Good — Claude caught it. But notice: no structured trace, no
alternative hypotheses checked, no evidence chain. In a 2000-line
file with dozens of variables, you need the systematic approach.
The /locate-fault certificate is auditable."

**/locate-fault focuses on the wrong code path:**
Follow up: "Can you trace from _check_multi_class through to the
refit=False branch? What value does multi_class hold vs self.multi_class?"

**The audience asks "why not just use a debugger?":**
"A debugger requires you to already suspect the right location. Here,
the traceback points at array indexing — that's where you'd set your
breakpoint. You'd be debugging the SYMPTOM, not the CAUSE. Structured
reasoning starts from the crash and works backward through ALL code
paths, not just the one you guessed."
