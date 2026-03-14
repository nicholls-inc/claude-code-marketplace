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
VIEW_MIDDLEWARE             2,665          42,416        40.5%      37.5%
PURE_FUNCTION               2,040          28,329        31.0%      25.0%
EXTERNAL_IO                   113           2,199         1.7%       1.9%
-------------------------------------------------------------------------
TOTAL                       6,573         113,219       100.0%     100.0%

Excluded: 13800 test functions, 494 configuration functions
```

### Comparative Results (8 codebases)

#### By lines of code (%) — Django apps

| Category | Hypothesis | Saleor | Zulip | NetBox | Oscar |
|---|---|---|---|---|---|
| DATABASE_ORM | ~40% | 35.3% | **37.4%** | **40.8%** | 22.8% |
| MODEL_VALIDATION | ~25% | 0.3% | 0.7% | 3.6% | **15.5%** |
| VIEW_MIDDLEWARE | ~20% | 37.5% | 31.4% | 31.2% | **38.3%** |
| PURE_FUNCTION | ~10% | 25.0% | 26.6% | 22.6% | 22.7% |
| EXTERNAL_IO | ~5% | 1.9% | 4.0% | 1.8% | 0.7% |

#### By lines of code (%) — Other Python frameworks

| Category | Hypothesis | Django (framework) | Redash (Flask) | Dispatch (FastAPI) | Home Assistant |
|---|---|---|---|---|---|
| DATABASE_ORM | ~40% | 16.7% | 26.6% | 21.5% | 5.7% |
| MODEL_VALIDATION | ~25% | 0.6% | 2.5% | 0.0% | 0.0% |
| VIEW_MIDDLEWARE | ~20% | **48.1%** | 16.3% | **48.8%** | **82.0%** |
| PURE_FUNCTION | ~10% | 31.8% | **45.6%** | 24.4% | 10.9% |
| EXTERNAL_IO | ~5% | 2.9% | 9.0% | 5.3% | 1.4% |

#### Raw numbers

| Project | Framework | Total funcs | Main funcs | Excl. tests | Excl. config |
|---|---|---|---|---|---|
| **Saleor** | Django + GraphQL | 20,867 | 6,573 | 13,800 | 494 |
| **Zulip** | Django + DRF | 12,124 | 5,537 | 6,255 | 332 |
| **NetBox** | Django + DRF | 5,539 | 2,848 | 2,607 | 84 |
| **django-oscar** | Django (traditional) | 4,148 | 2,009 | 2,065 | 74 |
| **Django** | Framework itself | 29,468 | 8,224 | 20,733 | 511 |
| **Redash** | Flask + SQLAlchemy | 2,516 | 1,429 | 1,011 | 76 |
| **Netflix Dispatch** | FastAPI + SQLAlchemy | 3,010 | 2,448 | 562 | 0 |
| **Home Assistant** | Custom (HA Core) | 85,942 | 41,100 | 44,842 | 0 |

### Key Findings

1. **DATABASE_ORM tracks the hypothesis in Django apps** — Saleor (35%), Zulip (37%), and NetBox (41%) confirm the ~40% prediction by lines. Non-Django apps show lower ORM percentages because SQLAlchemy patterns are more spread across service layers.

2. **PURE_FUNCTION is consistently 2-3x the hypothesized 10%** — across Django apps it's remarkably stable at **22-27% by lines**. This holds for Dispatch (24%) too. The formally verifiable slice is much larger than expected.

3. **The outliers are informative**:
   - **Home Assistant at 82% VIEW_MIDDLEWARE** — nearly the entire codebase is entity classes and integration lifecycle code. Only 10.9% is pure, because HA entities inherently interact with hardware/state.
   - **Redash at 46% PURE** — a data dashboarding tool has proportionally more pure data transformation logic than any other project.
   - **Django framework itself at 48% VIEW_MIDDLEWARE** — the framework is mostly framework code (unsurprisingly), but 32% pure function is notable — a lot of Django's internals are testable computation.

4. **MODEL_VALIDATION is dead in practice** — only django-oscar (15.5%) puts meaningful logic on models. Every other project, regardless of framework, keeps model methods minimal (0-4%). Business rules live in views, mutations, services, or entity methods.

5. **EXTERNAL_IO is universally low (1-9%)** — well-architected apps abstract IO. Redash is highest at 9% due to its data source connectors (querying external databases/APIs is its core function).

6. **Framework architecture determines VIEW_MIDDLEWARE vs PURE split** — the combined VIEW_MIDDLEWARE + PURE_FUNCTION is ~60-70% across all projects. What varies is the ratio between them: framework-heavy apps (HA, Django itself) skew toward VIEW_MIDDLEWARE; data-processing apps (Redash) skew toward PURE.

### Implications for Formal Verification

Across 8 codebases spanning 4 frameworks and ~1M lines of analyzed code:

- **~20-30% of a typical Python web application is formally verifiable** by tools like Dafny — 2-3x the hypothesized 10%
- The exception is IoT/hardware-integration code (Home Assistant: 11%) where almost everything interacts with external state
- Data-processing applications (Redash: 46%) have the highest verification potential
- The highest-value targets remain in the **non-pure 70-80%** — ORM query correctness, state machine transitions, authorization logic — where bugs are most impactful but current tools can't reach

## Known Limitations

1. **Indirect ORM access**: Functions that call other functions which in turn call ORM are not detected. Only direct ORM usage in the function body is analyzed.
2. **Dynamic dispatch**: Python's dynamic nature means some calls can't be resolved statically (e.g., `getattr`, `**kwargs` forwarding).
3. **Class hierarchy**: Only direct base classes are checked, not the full MRO. A class inheriting from a custom `BaseView` that itself inherits from `View` won't be detected unless `BaseView` is in the detection set.
4. **GraphQL detection**: Relies on common Graphene/Strawberry patterns. Custom GraphQL frameworks may not be detected.
5. **Pure function false positives**: Functions classified as pure by absence may actually have side effects through indirect calls, dynamic attribute access, or closures over mutable state.
6. **Single-file scope**: Each function is classified independently. Cross-function data flow is not analyzed.
7. **No type inference**: The analysis doesn't use type information to determine if a variable is a QuerySet, Model instance, etc.
