# Report: Merging Semiformal into Crosscheck + Paper Review Improvements

## Objective 1: Merge Strategy

### Current State

**Crosscheck** (4 skills, 1 agent, 1 MCP server):
- `/spec-iterate` — Draft & verify Dafny specifications
- `/generate-verified` — Implement verified Dafny code
- `/extract-code` — Compile to Python/Go
- `/lightweight-verify` — Design-by-contract + property-based tests
- `verify-orchestrator` agent — End-to-end formal verification workflow
- MCP server: `dafny_verify`, `dafny_compile`, `dafny_cleanup` (Docker-isolated)

**Semiformal** (5 skills, 1 agent, no MCP server):
- `/reason` — General-purpose semi-formal code reasoning
- `/analyze-code` — Code question answering with function traces
- `/compare-patches` — Patch equivalence verification
- `/locate-fault` — Fault localization (4-phase)
- `/trace-execution` — Execution path tracing
- `reasoning-orchestrator` agent — Task classification and skill routing

### The Context Efficiency Problem

A combined plugin with **9 skills + 2 agents** risks overwhelming Claude Code's context window. When a plugin loads, every skill description is injected into the system prompt. The current semiformal skills are verbose (each is 100-300 lines of structured templates). Loading all 9 simultaneously would:

1. Consume significant context budget before the user even asks a question
2. Create decision paralysis — Claude must choose among 9 skills
3. Dilute instruction-following quality (more instructions = less adherence to each)

### Recommended Merge Architecture

#### A. Unified Orchestrator as Primary Entry Point

Replace both orchestrators with a single **`crosscheck-orchestrator`** agent that serves as the main entry point. It classifies tasks and routes to the appropriate skill:

```
User request
    │
    ▼
Byfuglien (unified orchestrator)
    │
    ├─ Formal verification tasks ──► /spec-iterate → /generate-verified → /extract-code
    ├─ Lightweight verification ───► /lightweight-verify
    ├─ Regression detection ───────► /check-regressions
    ├─ Spec discovery ─────────────► /suggest-specs
    ├─ Adequacy argument ──────────► /rationale
    ├─ Code reasoning tasks ───────► /reason
    ├─ Patch comparison ───────────► /compare-patches
    ├─ Bug hunting ────────────────► /locate-fault
    └─ Execution tracing ──────────► /trace-execution
```

The unified orchestrator combines both existing orchestrators' task-fitness tables into one:

| Category | Trigger Signals | Skill |
|----------|----------------|-------|
| Algorithms with subtle invariants | Sorting, search, DP, safety-critical | `/spec-iterate` → full pipeline |
| Safety-critical logic | Access control, financial, crypto | `/spec-iterate` → full pipeline |
| Simple transformations, CRUD, IO | Map/filter, DB, HTTP handlers | `/lightweight-verify` |
| Regression check | "Did my changes break anything?", pre-commit review | `/check-regressions` |
| Spec discovery | "What should I verify?", reviewing new code | `/suggest-specs` |
| Adequacy argument | "Is this code adequate?", code + requirements | `/rationale` |
| Code questions | "What does X do?", "Is there a difference?" | `/reason` |
| Patch/diff comparison | Two diffs, "compare these changes" | `/compare-patches` |
| Bug/fault finding | "Why does this fail?", stack traces | `/locate-fault` |
| Execution tracing | "What happens when?", "Trace the flow" | `/trace-execution` |
| General code reasoning | Any other code reasoning question | `/reason` |

#### B. Skill Consolidation to Reduce Context Load

The 9 skills can be reduced to **6** without losing capability:

1. **Merge `/reason` into `/analyze-code`** — `/reason` is a general-purpose version of `/analyze-code`. The `/analyze-code` skill already covers the same 6-step process with more structure. Make `/analyze-code` accept a mode flag: `deep` (current analyze-code behavior with function trace tables and data flow) vs. `general` (current /reason behavior). Rename to **`/reason`** since it's the simpler, more intuitive name.

2. **Keep `/trace-execution` as-is** — sufficiently distinct (builds call graphs, not reasoning certificates).

3. **Keep `/locate-fault` as-is** — the 4-phase structure is unique and well-designed.

4. **Keep `/compare-patches` as-is** — specialized enough to warrant its own skill.

5. **Keep all 4 crosscheck skills as-is** — they form a coherent pipeline.

Result: **11 skills** (4 formal verification + 3 spec management/adequacy + 4 semiformal reasoning).

#### C. Skill Description Compression

Each semiformal skill prompt is 100-300 lines of detailed templates. To reduce context consumption:

1. **Move template details into reference files** (like crosscheck already does with `dafny-spec-patterns.md`). Each skill's SKILL.md becomes a concise 30-50 line overview with a `references/` subdirectory containing the full structured templates.

2. **The orchestrator loads only the routing table**, not every skill's full prompt. Individual skill prompts load only when invoked.

#### D. Directory Structure After Merge

```
crosscheck/
├── .claude-plugin/
│   └── plugin.json                     # v2.1.0, description covers all capabilities
├── agents/
│   └── byfuglien.md                   # Unified orchestrator (replaces both originals)
├── mcp-server/                         # Unchanged
│   └── ...
├── skills/
│   ├── spec-iterate/                   # Formal verification
│   ├── generate-verified/              # Formal verification
│   ├── extract-code/                   # Formal verification (+ Step 5.5 registry)
│   ├── lightweight-verify/             # Formal verification (lightweight)
│   ├── check-regressions/             # Spec management — regression detection
│   │   ├── SKILL.md
│   │   └── references/
│   │       └── registry-schema.md
│   ├── suggest-specs/                  # Spec management — autoformalization
│   │   └── SKILL.md
│   ├── rationale/                      # Bridging — structured adequacy arguments
│   │   └── SKILL.md
│   ├── reason/                         # Semi-formal (merged reason + analyze-code)
│   │   ├── SKILL.md
│   │   └── references/
│   │       └── reasoning-templates.md
│   ├── compare-patches/                # Semi-formal — patch equivalence
│   │   ├── SKILL.md
│   │   └── references/
│   │       └── comparison-templates.md
│   ├── locate-fault/                   # Semi-formal — fault localization
│   │   ├── SKILL.md
│   │   └── references/
│   │       └── fault-localization-templates.md
│   └── trace-execution/               # Semi-formal — execution tracing
│       ├── SKILL.md
│       └── references/
│           └── tracing-templates.md
├── docs/
│   └── reports/                        # Unchanged
├── scripts/                            # Unchanged
├── README.md                           # Updated
└── package.json                        # v2.1.0
```

#### E. Plugin Identity

Update `plugin.json`:

```json
{
  "name": "crosscheck",
  "version": "2.1.0",
  "description": "Crosscheck Claude's code claims — formal verification via Dafny for provably correct Python/Go, plus semi-formal reasoning for structured code analysis, fault localization, and patch comparison."
}
```

The "crosscheck" name still works: the plugin crosschecks Claude's code claims using both formal methods (Dafny) and structured semi-formal reasoning (evidence-grounded certificates).

---

## Objective 2: Improvements from Paper Reviews

### Papers Reviewed

1. **semiformal/PAPER-REVIEW-abductive-vibe-coding.md** — Semiformal plugin's perspective on the "Abductive Vibe Coding" paper (Murphy et al., U of T, 2026)
2. **crosscheck/docs/reports/abductive-vibe-coding-review.md** — Crosscheck's perspective on the same paper
3. **crosscheck/docs/reports/vibe-reasoning-paper-review.md** — Review of "Vibe Coding Needs Vibe Reasoning" (Mitchell & Shaaban, LMPL '25)

### Convergent Findings Across All Three Reviews

All three reviews independently identify the same core themes. The merge creates the opportunity to act on them:

#### Finding 1: Checklist-as-Contract Output (All 3 reviews — HIGH value, LOW effort)

**Problem:** Crosscheck ends with "verified" or "not verified." Semiformal ends with a confidence level (HIGH/MEDIUM/LOW). Neither gives the user a concrete, actionable checklist of "what you must verify for this conclusion to hold."

**Recommendation:** Add a `## Verification Checklist` section to every skill's output template:

- **Formal verification skills** (`/spec-iterate`, `/generate-verified`, `/extract-code`): checklist of trust boundaries — `{:extern}` methods, type mapping assumptions, Dafny limitation gaps (IO, concurrency, `real` vs float), informally-stated properties that were *not* formalized.
- **Semi-formal reasoning skills** (`/reason`, `/compare-patches`, `/locate-fault`, `/trace-execution`): checklist of premises to spot-check, framework behavior assumptions, alternative hypotheses ruled out but revisitable.

This bridges the abductive paper's key insight (decompose adequacy into verifiable items) with the practical output of both skill families.

#### Finding 2: Claim Classification by Verification Type (Reviews 1 & 2 — HIGH value, LOW effort)

**Problem:** Semi-formal skills treat all premises equally. Users can't distinguish "I verified this by reading code" from "this requires domain knowledge" from "this requires running code."

**Recommendation:** Tag every premise/claim with a verification class:
- `[STATIC]` — verifiable by reading code (file:line evidence)
- `[SEMANTIC]` — requires domain knowledge or subjective judgment
- `[BEHAVIORAL]` — requires running code to verify
- `[FORMAL]` — machine-verified via Dafny (new, only in merged plugin)

In the merged plugin, `[FORMAL]` claims can be dispatched to `dafny_verify` when the user wants higher assurance. This is the natural bridge between the two plugin families.

#### Finding 3: Spec Registry + Regression Detection (Review 3 — HIGH value, MEDIUM effort)

**Problem:** Crosscheck is stateless. You verify a function, extract code, and forget. Later edits silently invalidate Dafny guarantees.

**Recommendation:** Add a `.crosscheck/specs.json` manifest tracking:
- Functions with verified Dafny specs
- Dafny source file path or spec hash
- Extracted code file path and function signature
- Last-verified timestamp and difficulty metrics
- Constraint strength (hard = must pass `dafny_verify`, soft = property-based tests suffice)

Add a **`/check-regressions`** skill that scans the registry for specs whose associated source files have changed, re-verifies affected specs, and reports results. This transforms crosscheck from a one-shot tool into an ongoing correctness guardian.

#### Finding 4: Autoformalization / `/suggest-specs` (Reviews 2 & 3 — HIGH value, MEDIUM effort)

**Problem:** Crosscheck requires the developer to articulate what to verify. It never proposes "you should verify this."

**Recommendation:** Add a **`/suggest-specs`** skill that:
- Reads a function's signature, docstring, and call sites
- Proposes candidate preconditions/postconditions in natural language
- Lets the user approve, edit, or reject before entering `/spec-iterate`
- Flags implicit invariants ("this function is called in a loop — should the accumulated result satisfy X?")

This captures the "Vibe Reasoning" paper's core insight (reduce spec-writing burden) while staying within crosscheck's architectural identity.

#### Finding 5: Trust Boundary Tracking (Review 2 — HIGH value, LOW effort)

**Problem:** When Dafny verification succeeds, there's an implicit assumption that the spec captures user intent. This isn't tracked or surfaced.

**Recommendation:** After `/spec-iterate` produces an approved spec, generate a "trust boundary checklist":
- Formalization completeness assumptions
- `{:extern}` trust boundaries
- Dafny limitation gaps (no IO, no concurrency, `real` vs float)
- Informally-stated properties that were *not* formalized

This is a prompt-only change to the orchestrator — no new infrastructure.

#### Finding 6: Structured Rationale Generation (Review 2 — MEDIUM value, MEDIUM effort)

**Problem:** `/lightweight-verify` generates contracts and property-based tests but provides no structured argument connecting "these tests pass" to "the code is adequate."

**Recommendation:** Add a **`/rationale`** skill that:
- Takes code + informal requirements as input
- Generates a hierarchical claim tree
- Classifies each leaf as: `[FORMAL]` → route to `dafny_verify`, `[BEHAVIORAL]` → generate property-based tests, `[SEMANTIC]` → add to human review checklist
- Returns a traceable checklist (each item links back through the argument structure)

This is the deepest integration point between formal and semi-formal reasoning — defer until the merge is stable and findings 1-5 are implemented.

### What NOT to Adopt

All three reviews agree on what to avoid:

- **Lean as verification backend** — doubles infrastructure complexity, unfinished research, no benefit over Dafny+Z3
- **Continuous side-car on every edit** — wrong granularity for Dafny (10-120s per check); would make the plugin slow and noisy
- **Fully autonomous LLM feedback loop** — too risky without developer-in-the-loop; the current "human approves spec → iterate → human reviews" model is the right default
- **TypeScript-specific autoformalization templates** — architecturally misaligned with a Dafny-based plugin

---

## Implementation Priority

| Priority | Change | Effort | Impact | Status |
|----------|--------|--------|--------|--------|
| **1** | Merge semiformal skills into crosscheck directory | Medium | High | DONE |
| **2** | Unified Byfuglien orchestrator with combined routing table | Medium | High | DONE |
| **3** | Skill consolidation (merge /reason + /analyze-code) | Low | Medium | DONE |
| **4** | Verification checklist output for all skills (Findings 1+5 merged) | Low | High | DONE |
| **5** | Claim classification tags ([STATIC]/[SEMANTIC]/[BEHAVIORAL]/[FORMAL]) | Low | High | DONE |
| **6** | Spec registry + `/check-regressions` | Medium | High | DONE |
| **7** | `/suggest-specs` autoformalization | Medium | High | DONE |
| **8** | `/rationale` structured claim trees | Medium | Medium | DONE |

**Notes:**
- Item 4 merged original Findings 1 (verification checklists) and 5 (trust boundary tracking) — they are the same concept applied to different skill families. `/extract-code` already had a partial checklist (Abstraction Gap Checklist); the other 7 skills now have checklists too.
- Original item 4 (compress skill descriptions into references/) was dropped — skills only load into context when invoked by the user, so there's no context overhead from verbose SKILL.md files.
- Items 6-8: Added spec registry with `.crosscheck/specs.json` manifest, `/check-regressions` for regression detection, `/suggest-specs` for autoformalization, and `/rationale` for structured adequacy arguments. `/extract-code` updated with Step 5.5 to register specs in the registry.
