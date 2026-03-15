# Codebase Logic Distribution Analyzer

Static analysis tools that classify every function/method in a codebase by where its logic lives. Answers the question: **"what percentage of a real codebase is reachable by formal verification tools like Dafny or Lean?"**

Two analyzers are included:

- **Python analyzer** (`logic_distribution.py`) — targets Django, Flask, FastAPI, and other Python frameworks
- **TypeScript analyzer** (`ts/logic_distribution.ts`) — targets React, Next.js, Express, NestJS, Vue, Angular, and other JS/TS frameworks

## Quick Start

### Python

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

### TypeScript

```bash
cd ts && npm install

# Analyze a JS/TS project
npx tsx logic_distribution.ts /path/to/react

# Verbose output
npx tsx logic_distribution.ts /path/to/nextjs-app --verbose

# Spot-check 20 random functions
npx tsx logic_distribution.ts /path/to/express-api --spot-check 20

# Summary only / custom output / skip JSON — same flags as the Python analyzer
npx tsx logic_distribution.ts /path/to/project --summary-only
npx tsx logic_distribution.ts /path/to/project --output results.json
npx tsx logic_distribution.ts /path/to/project --no-json
```

Analyzes `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, and `.mts` files using the TypeScript compiler API. Automatically skips `node_modules`, `dist`, `build`, and other output directories.

## Classification Categories

### Python analyzer

| Category | Description |
|---|---|
| **DATABASE_ORM** | ORM queries, raw SQL, transactions — logic that executes inside the database |
| **MODEL_VALIDATION** | Model `clean()`/`save()` overrides, signal handlers, model properties, custom model methods without ORM queries |
| **VIEW_MIDDLEWARE** | Views, serializers, middleware, permissions, forms, template tags, GraphQL mutations/resolvers — framework-orchestrated request/response handling |
| **PURE_FUNCTION** | No Django imports, no ORM, no IO, no side effects — **formally verifiable** |
| **EXTERNAL_IO** | HTTP requests, file IO, subprocess, email, Celery tasks, cache operations |
| **TEST_CODE** | Functions in test files (excluded from main analysis) |
| **CONFIGURATION** | Migrations, settings, app configs (excluded from main analysis) |

### TypeScript analyzer

| Category | Description |
|---|---|
| **DATABASE_ORM** | Prisma, TypeORM, Sequelize, Knex, Drizzle, Mongoose, MongoDB, Redis — ORM/database operations |
| **SCHEMA_VALIDATION** | Zod, Yup, Joi, class-validator, io-ts, Valibot, Arktype — schema definitions and validation |
| **VIEW_FRAMEWORK** | React components, JSX, hooks, Express/Fastify route handlers, NestJS controllers, GraphQL resolvers, Redux, tRPC — framework-orchestrated UI and request handling |
| **PURE_FUNCTION** | No framework imports, no ORM, no IO, no side effects — **formally verifiable** |
| **EXTERNAL_IO** | fs, child_process, net, http/https, axios, fetch, WebSocket, AWS SDK — IO operations |
| **TEST_CODE** | Files matching `.test.*`, `.spec.*`, `__tests__/`, `__mocks__/`, `.stories.tsx` (excluded from main analysis) |
| **CONFIGURATION** | webpack/vite/rollup configs, migrations, seeds, `.env` files (excluded from main analysis) |

### Priority Rules

When a function touches multiple categories, the highest-priority match wins:

**Python:**

1. Test file → TEST_CODE
2. Migration/settings file → CONFIGURATION
3. Contains ORM operations → DATABASE_ORM
4. Contains external IO → EXTERNAL_IO
5. Framework class method or uses Django-imported names → VIEW_MIDDLEWARE
6. Model/manager method without ORM → MODEL_VALIDATION
7. None of the above → PURE_FUNCTION

**TypeScript:**

1. Test file → TEST_CODE
2. Config file → CONFIGURATION
3. Contains ORM/database operations → DATABASE_ORM
4. Contains external IO → EXTERNAL_IO
5. Framework component/handler/hook → VIEW_FRAMEWORK
6. Contains schema validation → SCHEMA_VALIDATION
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

### Comparative Results (6 codebases) — TypeScript

#### By lines of code (%)

| Category | React | VSCode | Strapi | Cal.com | Excalidraw | Payload |
|---|---|---|---|---|---|---|
| DATABASE_ORM | 5.0% | 7.4% | 15.3% | 23.3% | 5.0% | 23.5% |
| SCHEMA_VALIDATION | 0.1% | 0.0% | 1.1% | 1.5% | 0.0% | 0.4% |
| VIEW_FRAMEWORK | 26.7% | 1.3% | 43.8% | 45.1% | 27.1% | 30.1% |
| PURE_FUNCTION | 65.6% | **89.9%** | 36.1% | 27.1% | **65.0%** | 40.0% |
| EXTERNAL_IO | 2.5% | 1.4% | 3.7% | 3.0% | 2.9% | 5.9% |

#### Raw numbers

| Project | Stack | Total funcs | Main funcs | Excl. tests | Excl. config |
|---|---|---|---|---|---|
| **React** | React (framework) | 11,466 | 8,223 | 2,637 | 606 |
| **VSCode** | Electron + TypeScript | 59,128 | 54,962 | 4,144 | 22 |
| **Strapi** | Node + Koa | 5,450 | 4,758 | 590 | 102 |
| **Cal.com** | Next.js + Prisma | 12,699 | 11,881 | 767 | 51 |
| **Excalidraw** | React + Canvas | 2,389 | 2,225 | 164 | 0 |
| **Payload** | Next.js + MongoDB | 5,611 | 4,630 | 903 | 78 |

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

7. **TypeScript codebases skew even higher on PURE_FUNCTION** — VSCode at 90% and React at 66% dwarf the Python numbers. Pure computation dominates in codebases that aren't directly wired to databases or HTTP frameworks.

8. **Full-stack TS apps mirror Django apps** — Cal.com (27% pure, 45% view/framework) and Strapi (36% pure, 44% view/framework) look structurally similar to Saleor and Dispatch. The framework + pure split is consistent regardless of language.

9. **SCHEMA_VALIDATION is negligible everywhere** — despite Zod/Yup's popularity, schema code is consistently <2% of lines. Validation logic is small even when it's pervasive.

### Implications for Formal Verification

Across 14 codebases spanning 8+ frameworks and ~1.5M lines of analyzed code:

- **~20-30% of a typical Python web application is formally verifiable** by tools like Dafny — 2-3x the hypothesized 10%
- **TypeScript codebases range from 27% to 90% pure** — desktop/library code (VSCode, React) is overwhelmingly pure; full-stack apps (Cal.com, Strapi) are comparable to Python web apps
- The exception is IoT/hardware-integration code (Home Assistant: 11%) where almost everything interacts with external state
- Data-processing and library applications (Redash: 46%, Excalidraw: 65%, VSCode: 90%) have the highest verification potential
- The highest-value targets remain in the **non-pure code** — ORM query correctness, state machine transitions, authorization logic — where bugs are most impactful but current tools can't reach

## Known Limitations

### Shared

1. **Indirect calls**: Functions calling other functions that in turn perform ORM/IO are not detected. Only direct usage in the function body is analyzed.
2. **Single-file scope**: Each function is classified independently. Cross-function data flow is not analyzed.
3. **Pure function false positives**: Functions classified as pure by absence may have side effects through indirect calls, dynamic attribute access, or closures over mutable state.

### Python-specific

4. **Dynamic dispatch**: Python's dynamic nature means some calls can't be resolved statically (e.g., `getattr`, `**kwargs` forwarding).
5. **Class hierarchy**: Only direct base classes are checked, not the full MRO. A class inheriting from a custom `BaseView` that itself inherits from `View` won't be detected unless `BaseView` is in the detection set.
6. **GraphQL detection**: Relies on common Graphene/Strawberry patterns. Custom GraphQL frameworks may not be detected.
7. **No type inference**: The analysis doesn't use type information to determine if a variable is a QuerySet, Model instance, etc.

### TypeScript-specific

8. **No type-level analysis**: The analyzer uses AST pattern matching, not the TypeScript type checker. It cannot determine if a variable is a `PrismaClient` instance via type flow.
9. **Re-exports and barrel files**: Imports that pass through barrel files (`index.ts`) may not be recognized as framework imports if the original module name is lost.
10. **Dynamic imports**: `import()` expressions and `require()` with variable paths are not traced.
11. **Decorator detection**: Limited to NestJS and class-validator patterns. Custom decorators from other frameworks may not trigger classification.
