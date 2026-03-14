# Codebase Logic Distribution Analyzer

Static analysis tool that classifies every function/method in a Django codebase by where its logic lives. Answers the question: **"what percentage of a real codebase is reachable by formal verification tools like Dafny or Lean?"**

## Quick Start

```bash
# Analyze a Django project
python logic_distribution.py /path/to/django/project

# Limit to a specific app
python logic_distribution.py /path/to/project --app myapp

# Verbose output showing each classification decision
python logic_distribution.py /path/to/project --verbose

# Spot-check 20 random functions for manual review
python logic_distribution.py /path/to/project --spot-check 20

# Summary only (skip pure function detail, borderline cases, per-file breakdown)
python logic_distribution.py /path/to/project --summary-only

# Custom JSON output path
python logic_distribution.py /path/to/project --output results.json

# Skip JSON output
python logic_distribution.py /path/to/project --no-json
```

## Classification Categories

| Category | Description |
|---|---|
| **DATABASE_ORM** | ORM queries, raw SQL, transactions — logic that executes inside the database |
| **MODEL_VALIDATION** | Model `clean()`/`save()` overrides, signal handlers, model properties, custom model methods without ORM queries |
| **VIEW_MIDDLEWARE** | Views, serializers, middleware, permissions, forms, template tags, GraphQL mutations/resolvers — framework-orchestrated request/response handling |
| **PURE_FUNCTION** | No Django imports, no ORM, no IO, no side effects — **formally verifiable** |
| **EXTERNAL_IO** | HTTP requests, file IO, subprocess, email, Celery tasks, cache operations |
| **TEST_CODE** | Functions in test files (excluded from main analysis) |
| **CONFIGURATION** | Migrations, settings, app configs (excluded from main analysis) |

### Priority Rules

When a function touches multiple categories, the highest-priority match wins:

1. Test file → TEST_CODE
2. Migration/settings file → CONFIGURATION
3. Contains ORM operations → DATABASE_ORM
4. Contains external IO → EXTERNAL_IO
5. Framework class method or uses Django-imported names → VIEW_MIDDLEWARE
6. Model/manager method without ORM → MODEL_VALIDATION
7. None of the above → PURE_FUNCTION

## Output

### Console Report

1. **Summary table** — function count and lines of code per category with percentages
2. **Pure function detail** — every formally verifiable function with file path, name, and confidence
3. **Borderline cases** — functions that matched multiple categories or have side effects like logging
4. **Per-file breakdown** — category distribution for each Python file

### JSON Output

`analysis_results.json` contains every function's classification with:
- File path, function name, line number, line count
- Category and confidence level (HIGH/MEDIUM/LOW)
- Classification rationale (which heuristic triggered)
- Whether it's a borderline case and why
- All categories that matched before priority resolution

## Confidence Levels

| Level | Meaning |
|---|---|
| **HIGH** | Clearly one category; strong heuristic match |
| **MEDIUM** | Multiple categories matched (priority resolved), or minor side effects (logging/print) |
| **LOW** | Classification by absence/default, or function is in a Django-importing file without direct framework usage |

## Example: Saleor Analysis

Running against [Saleor](https://github.com/saleor/saleor) (~80K lines, Django e-commerce):

```
Category                Functions   Lines of Code  % Functions    % Lines
-------------------------------------------------------------------------
DATABASE_ORM                1,684          39,911        25.6%      35.3%
MODEL_VALIDATION               71             364         1.1%       0.3%
VIEW_MIDDLEWARE             2,629          41,769        40.0%      36.9%
PURE_FUNCTION               2,076          28,976        31.6%      25.6%
EXTERNAL_IO                   113           2,199         1.7%       1.9%
-------------------------------------------------------------------------
TOTAL                       6,573         113,219       100.0%     100.0%

Excluded: 13800 test functions, 494 configuration functions
```

### Comparative Results (4 codebases)

#### By function count (%)

| Category | Hypothesis | Saleor | Zulip | NetBox | Oscar |
|---|---|---|---|---|---|
| DATABASE_ORM | ~40% | 25.6% | 17.6% | 26.7% | 12.9% |
| MODEL_VALIDATION | ~25% | 1.1% | 2.3% | 4.9% | **21.4%** |
| VIEW_MIDDLEWARE | ~20% | 40.0% | 34.0% | 29.5% | 37.3% |
| PURE_FUNCTION | ~10% | 31.6% | 43.3% | 38.3% | 27.7% |
| EXTERNAL_IO | ~5% | 1.7% | 2.9% | 0.6% | 0.6% |

#### By lines of code (%)

| Category | Hypothesis | Saleor | Zulip | NetBox | Oscar |
|---|---|---|---|---|---|
| DATABASE_ORM | ~40% | 35.3% | **37.3%** | **40.8%** | 22.8% |
| MODEL_VALIDATION | ~25% | 0.3% | 0.7% | 3.6% | **15.5%** |
| VIEW_MIDDLEWARE | ~20% | 36.9% | 30.6% | 31.1% | **38.3%** |
| PURE_FUNCTION | ~10% | 25.6% | 27.3% | 22.7% | 22.7% |
| EXTERNAL_IO | ~5% | 1.9% | 4.0% | 1.8% | 0.7% |

#### Raw numbers

| Project | Architecture | Total funcs | Excl. tests | Excl. config |
|---|---|---|---|---|
| **Saleor** | GraphQL (Graphene) | 20,867 | 13,800 | 494 |
| **Zulip** | Traditional Django + DRF | 12,124 | 6,255 | 332 |
| **NetBox** | Django + DRF (REST) | 5,539 | 2,607 | 84 |
| **django-oscar** | Traditional Django (e-commerce framework) | 4,148 | 2,065 | 74 |

### Key Findings

1. **DATABASE_ORM by lines is consistent at 35-41%** across Saleor, Zulip, and NetBox — confirming the hypothesis that ~40% of logic lives in the database. Oscar is lower (23%) because it's a framework (less app-specific queries, more generic patterns). By lines of code, the hypothesis holds well.

2. **PURE_FUNCTION is 3x higher than hypothesized** across all four codebases (23-27% by lines, 28-43% by function count). This is the biggest surprise — formal verification tools have a much larger attack surface than expected. Pure functions include data transformation helpers, business computation (tax, pricing, discounts), formatting, and validation utilities.

3. **MODEL_VALIDATION only matches the hypothesis for django-oscar** (21.4%), which is the most traditional Django architecture. Modern projects (Saleor, Zulip, NetBox) put very little logic directly on model methods (1-5%), preferring service layers and GraphQL mutations.

4. **VIEW_MIDDLEWARE absorbs what was hypothesized as MODEL_VALIDATION** — in practice, business rules live in views/mutations/serializers, not on models. Combined VIEW_MIDDLEWARE + MODEL_VALIDATION ranges from 34-58%, close to the hypothesized 45% combined.

5. **EXTERNAL_IO is consistently low (0.6-4%)** — well-architected Django apps abstract IO behind clean interfaces. Zulip is highest at 4% due to email/webhook integrations.

6. **Architecture barely affects the formally-verifiable slice** — despite very different architectures (GraphQL vs REST vs traditional), the PURE_FUNCTION percentage by lines is remarkably stable at **22-27%** across all four projects. This suggests ~25% is a structural property of Django applications, not an accident of architecture.

### Implications for Formal Verification

The data suggests that **~25% of a Django codebase by lines of code** is reachable by formal verification tools — 2.5x the hypothesized 10%. However:

- Many pure functions are small helpers (median ~10 lines), so the *impact* of verifying them may be lower than the percentage suggests
- The highest-value verification targets are in the remaining 75% — ORM query correctness, state machine transitions, and authorization logic — which current tools can't reach
- The pure function set includes many LOW-confidence classifications that may be framework-adjacent on manual review

## Known Limitations

1. **Indirect ORM access**: Functions that call other functions which in turn call ORM are not detected. Only direct ORM usage in the function body is analyzed.
2. **Dynamic dispatch**: Python's dynamic nature means some calls can't be resolved statically (e.g., `getattr`, `**kwargs` forwarding).
3. **Class hierarchy**: Only direct base classes are checked, not the full MRO. A class inheriting from a custom `BaseView` that itself inherits from `View` won't be detected unless `BaseView` is in the detection set.
4. **GraphQL detection**: Relies on common Graphene/Strawberry patterns. Custom GraphQL frameworks may not be detected.
5. **Pure function false positives**: Functions classified as pure by absence may actually have side effects through indirect calls, dynamic attribute access, or closures over mutable state.
6. **Single-file scope**: Each function is classified independently. Cross-function data flow is not analyzed.
7. **No type inference**: The analysis doesn't use type information to determine if a variable is a QuerySet, Model instance, etc.
