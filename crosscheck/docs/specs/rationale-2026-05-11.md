---
Date: 2026-05-11
Status: Draft
Layer: 4 (semi-formal rationales) of the 6-layer assurance hierarchy
Owner: hellebuyck
Audience: Harry (red-pen). The operational prompt for the agent lives in `../../skills/rationale/SKILL.md`; this doc is the design intent behind that prompt.
Companions: `../../skills/rationale/SKILL.md`, `../../agents/hellebuyck.md`, `../assurance-hierarchy.md`
---

# /rationale — design intent snapshot

## 1. Why it exists

Most code an LLM writes isn't amenable to a single verification technique. A sort function might be provable in Dafny. A web handler might want a property test, a type check, and a human judgement that the error message is actionable. Asking *"is this code correct?"* of a real codebase means juggling several verification methods at once, and there's no shared structure for what each technique is contributing.

`/rationale` is the structuring move. It decomposes the high-level claim *"this code is adequate"* into smaller subclaims, tags each leaf by *how* it should be verified — formal, behavioural, static, semantic — and discharges each via the appropriate technique. The output is a checklist where, if every leaf holds, the root claim holds by construction.

The grounding is *Abductive Vibe Coding* (Murphy/Babikian/Chechik, arXiv:2601.01199) — hierarchical rationales inspired by Goal Structuring Notation, plus the abductive framing that the rationale states *what would need to be true* for the artefact to be adequate (`../research/perplexity-crosscheck-analysis-may-2026.md:262`). `/rationale` is Crosscheck's developer-facing instantiation.

## 2. What it does

The user invokes `/rationale [path] [optional: requirements]`. The skill:

1. Reads the code and gathers requirements (user description, docstrings, existing tests, calling context).
2. Builds a hierarchical claim tree rooted at *"Code is adequate for [requirements]"*, decomposed into trust-boundary / structural / behavioural / non-functional branches, then specialised to the module.
3. Classifies each leaf claim by verification method.
4. Attempts verification per class — Dafny or Lean for FORMAL, generated test code for BEHAVIORAL, citations for STATIC, written prompt to the human for SEMANTIC.
5. Presents a checklist where each item is traceable through the tree to the root.

**Tree shape.** The top-level decomposition is `C0` trust-boundary / `C1` structural / `C2` behavioural / `C3` non-functional. Trust boundaries appear as a first-class branch because they bound what the rest of the tree can verify at all — a STATIC leaf claiming *"`sort.py:42` implements quicksort correctly"* is meaningless if the comparison function is `{:extern}` or pulls from the network. Promoting trust boundaries from a final-checklist bullet (see `../../skills/rationale/SKILL.md:172`) to a branch root makes them part of the argument structure rather than a footnote.

The deliverable is the checklist. It is not persisted to disk by default — it appears in the conversation and the human decides what to do with it (paste into a PR description, file it as an issue, discard it, iterate).

**Targets.** SKILL.md today handles one target — implementation verification, where the artefact is code and the requirements are about its behaviour. The goal-structured argument plus the four-class leaf taxonomy are target-agnostic, but two pieces of the skill are target-specific and need to be extended together to broaden the supported targets:

- *Decomposition templates.* Implementation uses `C0 trust / C1 structural / C2 behavioural / C3 non-functional`. Spec analysis (arguing that an invariant doc adequately captures a module's safety properties) wants something like `C0 trust / C1 completeness / C2 internal consistency / C3 mechanical verifiability`. Other targets — acceptance-scenario adequacy, test-suite adequacy, others — get their own templates.
- *Discharge routes.* For implementation, FORMAL routes to Dafny or the Lean pipeline (§4). For spec analysis, FORMAL might route to `/intent-check`'s round-trip pipeline, or to nothing (some claims may not be discharge-able formally and stay STATIC).

Multi-target support is design intent. SKILL.md catching up to it is downstream work — a separate PR with its own snapshot, not part of this one.

## 3. Leaf classification

Four tags, each with a discharge route:

| Tag | What it means | Discharge |
|---|---|---|
| `[FORMAL]` | Provable mathematically | Layer 1: candidate Dafny spec → `/spec-iterate`. Layer 4: candidate Lean spec → `/lean-spec` → `/lean-impl` → `/drt-oracle`. See §4. |
| `[BEHAVIORAL]` | Tested by execution | Generate test cases or property-based test code |
| `[STATIC]` | Verifiable by reading | Cite `file:line` evidence; deterministic post-process verifies the citation |
| `[SEMANTIC]` | Human domain judgement | Add to human review checklist with relevant context |

Classification guidance (`../../skills/rationale/SKILL.md:82-90`):

- "for all" / "there exists" / "is a permutation of" → FORMAL
- Invariant preservation (sorting, bounds, conservation laws) → FORMAL
- Edge cases (empty input, null, overflow) → BEHAVIORAL
- Type correctness, field presence, structural shape → STATIC
- Business rule alignment, naming, UX quality → SEMANTIC
- Performance under load → BEHAVIORAL (benchmark)
- Error-message quality → SEMANTIC

## 4. Verification per class

**FORMAL.** Two discharge routes by assurance layer:

- *Layer 1 — pure code shipping to production.* The skill drafts a candidate Dafny spec (preconditions + postconditions) and offers to call `dafny_verify`. If verification passes, the leaf is verified. If it fails, the skill suggests `/spec-iterate` for iterative refinement. The Dafny implementation is the production artefact (extracted to Python or Go); this is the classical Crosscheck Layer 1 case.
- *Layer 4 — implementation matches a model.* For code where Dafny doesn't reach (impure, effectful, networked, concurrent), the skill drafts a Lean 4 specification stub and walks `/lean-spec` → `/lean-impl` → `/correspondence-review` → `/drt-oracle`. The Lean model is not shipped — it runs as a differential-testing oracle against the production implementation. Routing here is Layer 4 (impl-vs-model alignment), not Layer 1 (pure verified production code); `/lean-impl` and `/drt-oracle` are Layer 4 skills even though they exercise formal-methods machinery.

The skill picks the route by asking whether the leaf names a property of pure code with a clean functional shape (Layer 1) or a behavioural correspondence between an effectful implementation and an abstract model (Layer 4).

**BEHAVIORAL.** The skill generates test code (concrete cases or property-based) and presents it. It does not run the tests. Marking is *"tests generated — run to verify."*

**STATIC.** The skill reads the code and cites a specific `file:line` range as evidence. Example from `../../skills/rationale/SKILL.md:108`: *"C1.2 verified — all required fields set at `model.py:42-48`."*

A fast deterministic post-process reads each cited range and checks that the keywords from the claim text appear there (or that the cited identifier is defined within the range). Citations failing the check are returned for re-citation or downgraded to SEMANTIC for human review. The check runs without an LLM in the loop — a pure read-and-string-search pass. This catches fabricated citations without relying on the agent to be honest about them.

**SEMANTIC.** The skill states what the human must judge and provides the relevant code context. Marking is *"human review required."*

## 5. Output shape

The output (`../../skills/rationale/SKILL.md:120-159`) has three blocks:

- The claim tree (the decomposition).
- A verification-results table — one row per leaf, checked or unchecked, with a one-line evidence pointer.
- A summary table by verification method (FORMAL/STATIC/BEHAVIORAL/SEMANTIC × total/verified/pending).

The shape is borrowed wholesale from the Goal Structuring Notation tradition: the root claim is the safety case, the leaves are the evidence items, the tree edges are the argument. The checklist is the auditable artefact.

## 6. Seams with neighbouring skills

- **`/spec-iterate`** is the discharge target for Layer 1 FORMAL claims. `/rationale` produces a candidate Dafny spec; `/spec-iterate` verifies it.
- **`/lean-spec` → `/lean-impl` → `/correspondence-review` → `/drt-oracle`** is the discharge chain for Layer 4 FORMAL claims. `/rationale` decides routing; the Lean pipeline runs the differential-testing verification with Lean as the model.
- **`/spec-adversary`** is the complement on the other side of the assurance hierarchy. `/rationale` builds an adequacy argument from claims the agent identified. `/spec-adversary` probes for invariants the spec is failing to capture. The handoff is named in `../agents.md:35`: *"Byfuglien's `/rationale` produces an adequacy argument for a CRUD endpoint — hellebuyck's `/spec-adversary` probes for missing invariants the rationale didn't anticipate."* The byfuglien attribution in that quote precedes the Layer 4 reassignment in this snapshot; the seam stands, the ownership moves to hellebuyck.
- **`/intent-check`** runs the round-trip check on the spec itself. A FORMAL claim's candidate Dafny or Lean spec is the kind of artefact `/intent-check` would consume to ask *"does this spec actually describe what the test exercises?"*
- **`/assurance-probe`** measures test strength via mutation. A BEHAVIORAL leaf's generated tests could plausibly be exercised by `/assurance-probe` to check they're strong enough; the skill does not chain to it (see §7).
- **`/reason`**, **`/compare-patches`**, **`/locate-fault`**, **`/trace-execution`** are the semi-formal reasoning cluster `/rationale` shares structural DNA with. The SKILL.md opening line names the role: *"Bridges formal verification and semi-formal reasoning."*

## 7. Non-goals

- **Not a code reviewer.** `/rationale` does not flag bugs, suggest refactors, or critique style. If the code is wrong, the rationale will (correctly) fail to discharge a leaf — but the deliverable is the argument structure, not the bug.
- **Not a test runner.** Generated tests are presented, not executed.
- **Not a coverage tool.** The checklist is per-invocation. It does not aggregate across rationales into a project-level coverage view.
- **Not a spec-completeness check.** That's `/spec-adversary`'s job. If a *branch* is missing from the tree, `/rationale` will not notice. Structural soundness — *"if all leaves hold, the root holds"* — is conditional on adequate decomposition. See §8 for the working recommendation on closing this gap inside `/rationale` itself.
- **Not persisted to disk by default.** The checklist appears in the conversation; the human decides whether to file it, paste it into a PR, or discard it. Persistence may earn its place later; nothing in current use motivates it.
- **Not an orchestrator.** `/rationale` does not chain to `/assurance-probe` for BEHAVIORAL test strength, to `/intent-check` for candidate spec validation, or to `/lean-impl` for Layer 4 FORMAL discharge. Skills stay modular; agents (hellebuyck, byfuglien, or a user-driven sequence) compose them.

## 8. Open — deferred until field evidence

**Tree completeness.** Structural soundness presumes adequate decomposition. There is no adversary against missing *branches* — the tree could be sound under *"if all leaves hold, root holds"* and still silently fail because a branch was never enumerated. `/spec-adversary` handles the analogous concern for invariant docs, not rationale trees.

**Deferred.** No completeness handling is added in this snapshot. The skill relies on the agent's tree-building judgement; field evidence is the right calibration for whether the gap matters in practice. If invocations start producing sound-but-incomplete trees with non-obvious missing branches, two options are on the shelf:

- *Sibling skill `/rationale-adversary`*, mirroring `/spec-adversary` at the rationale-tree level. Cost: a new skill and the surface area that comes with it.
- *In-skill completeness pass inside `/rationale`* — after the tree is discharged, generate 2–3 candidate branches that might be missing and ask the user to accept, reject, or defer each. Cost: lower; lives inside `/rationale`'s already-abductive framing.

Trigger to revisit: field reports of `/rationale` invocations where the agent built a sound tree but the user noticed (or was bitten by) a missing branch.

## 9. Verification approach for this spec

This spec describes prompt content in `../../skills/rationale/SKILL.md`. There is no executable model to verify against. The acceptance criterion is behavioural: a user invokes `/rationale` on a real function and gets a claim tree with a `C0` trust-boundary branch, classified leaves, attempts at verification per class (Dafny for Layer 1 FORMAL, the Lean pipeline for Layer 4 FORMAL, deterministic citation-checked STATIC, generated BEHAVIORAL test code, human-prompt SEMANTIC), and a traceable checklist whose summary table balances. Validation is by direct use, not by a separate verification pipeline.

The orchestrator's existing soundness check stays in place: *"`/rationale` tree structure is valid: if all leaves hold, the root holds"* (`../../agents/byfuglien.md:147`). That check is structural — it does not catch missing branches; see §7 non-goal and the deferred handling in §8.

This is a dated snapshot, not a living document. If `/rationale` changes after merge, the right move is a new snapshot dated to the change, with a journal entry that supersedes this one.
