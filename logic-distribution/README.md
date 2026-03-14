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

### Key Findings

The original hypothesis predicted ~10% pure functions. Saleor shows **31.6% by function count (25.6% by lines)** — significantly higher than expected. However, this deserves nuance:

1. **DATABASE_ORM at 25.6%/35.3%** — Close to the hypothesized 40%. The line-of-code percentage (35.3%) is closer to the prediction, suggesting ORM functions are larger than average.

2. **VIEW_MIDDLEWARE at 40.0%/36.9%** — Saleor uses GraphQL (Graphene) heavily rather than traditional Django views/DRF. This category absorbed what would be "framework model methods" in the hypothesis, since Graphene mutations handle validation and state transitions.

3. **MODEL_VALIDATION at 1.1%** — Much lower than the hypothesized 25%. In Saleor's architecture, business rules live in GraphQL mutations (VIEW_MIDDLEWARE) and service functions rather than on Django model methods.

4. **PURE_FUNCTION at 31.6%/25.6%** — Significantly higher than hypothesized 10%. Many of these are:
   - Data transformation helpers (preparing payloads, formatting)
   - Business logic computation (tax calculations, pricing, discounts)
   - Type conversion and validation utilities
   - Many LOW-confidence classifications that may be framework-adjacent

5. **EXTERNAL_IO at 1.7%** — Lower than hypothesized 5%. Saleor abstracts external integrations behind plugin interfaces, concentrating IO in fewer functions.

### Hypothesis vs Reality

| Category | Hypothesis | Saleor (functions) | Saleor (lines) |
|---|---|---|---|
| DATABASE_ORM | ~40% | 25.6% | 35.3% |
| MODEL_VALIDATION | ~25% | 1.1% | 0.3% |
| VIEW_MIDDLEWARE | ~20% | 40.0% | 36.9% |
| PURE_FUNCTION | ~10% | 31.6% | 25.6% |
| EXTERNAL_IO | ~5% | 1.7% | 1.9% |

The hypothesis was partially supported for DATABASE_ORM (by lines) but significantly off for other categories. The key insight is that **architecture matters enormously** — Saleor's GraphQL-first design pushes logic into different categories than a traditional Django views+DRF app would.

The formal verification opportunity (~25-31% pure functions) is much larger than hypothesized, though many of these functions are small helpers. The HIGH-confidence pure functions represent the most promising targets for formal verification.

## Known Limitations

1. **Indirect ORM access**: Functions that call other functions which in turn call ORM are not detected. Only direct ORM usage in the function body is analyzed.
2. **Dynamic dispatch**: Python's dynamic nature means some calls can't be resolved statically (e.g., `getattr`, `**kwargs` forwarding).
3. **Class hierarchy**: Only direct base classes are checked, not the full MRO. A class inheriting from a custom `BaseView` that itself inherits from `View` won't be detected unless `BaseView` is in the detection set.
4. **GraphQL detection**: Relies on common Graphene/Strawberry patterns. Custom GraphQL frameworks may not be detected.
5. **Pure function false positives**: Functions classified as pure by absence may actually have side effects through indirect calls, dynamic attribute access, or closures over mutable state.
6. **Single-file scope**: Each function is classified independently. Cross-function data flow is not analyzed.
7. **No type inference**: The analysis doesn't use type information to determine if a variable is a QuerySet, Model instance, etc.
