# ADD operating modes

ADD applies to three situations, and the governance that is *appropriate* differs in each. A module carries its situation as a frontmatter `add-mode` tag; skills and gates read the tag and branch. This is the v1 wiring of ADR-001 (operating modes) and ADR-004 (greenfield entrypoints), grounded in the `ngst` greenfield field test (see [`reports/add-greenfield-field-report-ngst.md`](reports/add-greenfield-field-report-ngst.md)).

## The three modes

| Mode | Applied to | Governance | `add-mode` tag |
|---|---|---|---|
| **bootstrap** | existing code, no spec | recovered post-hoc — invariants *derived from the code and tests*, protected surfaces retrofitted | `add-mode: bootstrap` |
| **add** | a written spec (or clean slate) | prefigured — invariants *derived top-down from the spec*, which is the intent proxy | `add-mode: add` |
| **transitional** | a repo holding *both* | per-module: each module is consulted for its own tag | repo-level state; **not** a per-module tag |

`transitional` describes a *repo*, never a module. A module is always `bootstrap` or `add`. A repo is `transitional` the moment its modules disagree — the common case when ADD is adopted on an existing codebase while new modules are written spec-first.

**Default.** A module with no `add-mode` tag defaults to `bootstrap` (ADR-001 §Consequences: existing users must not break). The conformance oracle's AUTO mode-tag check (`crosscheck/conformance` AUTO 6) nonetheless requires every crosscheck skill/agent to *declare* its tag explicitly, so the default never hides an un-triaged module.

## Selecting a mode (the entry-point rule)

`add-orchestrator` Step 0 selects the mode from what the repo actually contains:

1. **A spec is present** (user-supplied path, or a discovered spec) → **`add` / spec-consult**. Consume the written spec as the contract. **Do not cold-elicit what the spec already answers** — this is the #149/#150 failure the greenfield run hit: `/draft-invariants` interviewing against a spec that already held the answers produced a referential-integrity failure. Drafted invariant docs are tagged `add-mode: add`.
2. **Existing code, no spec** → **`bootstrap` / legacy-derive**. Derive invariants from the code, tests, and error handling — not from a cold interview. Drafted docs are tagged `add-mode: bootstrap`. (`ngst`'s `secrets` module is the worked example: invariants drafted from the existing loader caught the `_FILE`-wins contradiction.)
3. **Empty or near-empty repo, no spec** → **greenfield / intent-elicit**. Capture intent first, then derive a spec, then re-enter at case 1. The dedicated greenfield skills (ADR-004 S2.1–S2.4: `/intent-elicit`, `/spec-derive`, `/intent-check-prose`, `/spec-adversary-prose`) are not yet built; until they ship, `/assurance-init` seeds the skeleton and the operator supplies intent.

## How skills honour the tag

A mode-aware skill reads the module's `add-mode` and branches **legibly** (ADR-001: "mode-conditional governance must not become a maze of branching logic"):

- **bootstrap** — invariants may be recovered from code; a consolidation/audit pass does **not** flag the module as drifted merely for lacking a top-down intent trail.
- **add** — the spec is the contract; an audit pass *does* flag a spec amended without a corresponding invariant amendment.

v1 wires mode *selection* (Step 0), mode *tagging* (enforced on crosscheck's own modules by AUTO 6), and the three *entrypoints* above. Full per-skill mode-aware branching across every skill, and the greenfield S2.1–S2.4 skill set, are incremental follow-ups tracked under epic #217.
