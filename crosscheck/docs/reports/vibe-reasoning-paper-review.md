# Paper Review: "Vibe Coding Needs Vibe Reasoning" vs. Crosscheck Plugin

**Paper:** Mitchell & Shaaban, "Position: Vibe Coding Needs Vibe Reasoning: Improving Vibe Coding with Formal Verification" (LMPL '25)
**arXiv:** 2511.00202

## Paper Summary

The paper argues that vibe coding (iterative LLM-driven development) is fragile because natural-language constraints accumulate and conflict over time, causing what the authors call **constraint-reconciliation decay**. They classify existing LLM+formal-methods systems into two types:

- **Type I**: Formal methods filter LLM outputs and provide counterexample feedback (e.g., loop-invariant synthesis)
- **Type II**: Formal methods post-process LLM outputs by construction (e.g., constrained decoding)

Both require the developer to supply and maintain specifications, which is burdensome during rapid iteration.

The authors propose **Type III ("Vibe Reasoning")**: an autonomous side-car system with three pillars:

1. **Autoformalization** — LLM-generates specifications from developer intent (explicit prompts + implicit patterns), using templates for common patterns
2. **Continuous Verification** — Lightweight, incremental checks that match development pace; the system selects appropriate verification techniques
3. **Integration** — Verifier feedback flows into the LLM's edit loop; failures are incorporated into the build process; developers can approve/reject/relax specs

Their proof-of-concept targets TypeScript with four autoformalization templates: Exhaustive Switch, Discriminated Union, Union Alias, and `satisfies` Guard. It integrates with Claude Code via hooks.

---

## Comparison: Paper vs. Crosscheck

| Dimension | Crosscheck | Vibe Reasoning (Paper) |
|-----------|-----------|----------------------|
| **Verification backend** | Dafny (full theorem prover via Z3) | Language-native type system + lightweight syntactic checks |
| **Scope** | Single-function algorithmic correctness | Cross-file, cross-module structural invariants |
| **Specification source** | User-initiated: developer describes what to verify | Auto-generated: system proposes specs from code patterns |
| **When it runs** | On-demand (user invokes `/spec-iterate`) | Continuously, as a side-car on every code change |
| **Developer burden** | High: must understand Dafny, approve specs, guide iteration | Low: system proposes, developer approves/rejects |
| **Verification weight** | Heavy: Docker container, Dafny + Z3 solver, 120s timeout | Light: TypeScript compiler, syntactic pattern matching |
| **Feedback target** | The developer (presents verified code for integration) | The LLM agent (feeds verification failures into edit loop) |
| **Constraint management** | None: each verification is stateless, one function at a time | Central concern: tracks accumulated specs, detects conflicts |
| **Integration model** | Type I: verify → feedback → iterate (manual) | Type III: autoformalize → verify → integrate (autonomous) |

### Where Crosscheck is stronger

- **Proof depth**: Dafny provides mathematical certainty about algorithmic properties (preconditions, postconditions, loop invariants, termination). The paper's PoC only catches syntactic pattern violations.
- **Multi-language extraction**: Crosscheck compiles verified Dafny to Python and Go with boilerplate stripping and property-based test generation. The paper is TypeScript-only.
- **Difficulty metrics**: Crosscheck surfaces solver time, resource count, proof hint count, and trivial-proof detection. The paper has no equivalent.
- **Lightweight fallback**: `/lightweight-verify` already provides design-by-contract, property-based testing, and documented-invariants modes for when full verification is overkill.

### Where the paper identifies genuine gaps in Crosscheck

- **No autoformalization**: Crosscheck requires the developer to articulate what to verify. It never proposes "you should verify this."
- **No continuous/side-car mode**: Crosscheck runs only when invoked. It has no presence in the ongoing edit loop.
- **No constraint tracking**: Each verification is stateless. Crosscheck doesn't remember prior specs or detect when a new edit regresses a previously-verified property.
- **No cross-module awareness**: Crosscheck verifies single Dafny files. It cannot detect partial-propagation bugs or state-machine divergence across files.
- **LLM feedback loop is manual**: Verification results are presented to the developer, who must decide what to do. The system never autonomously feeds failures back into the coding agent.

---

## Assessment: What to adopt (and what to push back on)

### Idea 1: Autoformalization — Adopt, but differently

**Paper's proposal**: LLM-generated specifications from templates (Exhaustive Switch, Discriminated Union, etc.).

**Pushback**: The paper's templates are TypeScript-specific syntactic patterns. Bolting TypeScript type-system checks onto a Dafny-based plugin would be architectural incoherence. Crosscheck's value proposition is *deeper-than-types* verification.

**What to do instead**: Add an **auto-spec suggestion skill** (`/suggest-specs`) that analyzes a function and proposes Dafny specifications the user might want to verify. This is autoformalization at the *Dafny level*, not the type-system level. The skill would:

- Read a function's signature, docstring, and call sites
- Propose candidate preconditions/postconditions in natural language
- Let the user approve, edit, or reject before entering the `/spec-iterate` loop
- Flag implicit invariants (e.g., "this function is called in a loop — should the accumulated result satisfy X?")

This captures the paper's core insight (reduce spec-writing burden) while staying within Crosscheck's architectural identity.

### Idea 2: Continuous verification side-car — Push back

**Paper's proposal**: Run verification on every code change via hooks.

**Pushback**: This makes sense for the paper's lightweight checks (does a switch exhaust a union type? ~0ms). It does **not** make sense for Dafny verification (Docker container, Z3 solver, 10-120s per check). Running Dafny on every edit would be:

- Slow enough to block the developer
- Expensive in compute (Docker spin-up per change)
- Noisy (partial edits will fail verification constantly)

**What to do instead**: Two things that capture the spirit without the overhead:

1. **Post-commit verification hook** (optional): After a commit or before a PR, re-verify any Dafny specs associated with changed files. This is "continuous" at a practical granularity. Requires maintaining a mapping of source files → Dafny spec files.

2. **Lightweight continuous checks as a separate concern**: If we want the paper's fast syntactic checks, that should be a *separate plugin* (or an extension to the `semiformal` plugin), not shoehorned into Crosscheck. Crosscheck's identity is deep verification. Mixing in linter-weight checks would dilute it.

### Idea 3: Constraint tracking / regression detection — Adopt

**Paper's proposal**: The system tracks accumulated specifications and detects when new edits regress previously-verified properties.

**This is Crosscheck's biggest gap.** Today, you verify a function, extract code, and forget. If someone edits the extracted code later, the Dafny guarantees silently evaporate.

**What to do**: Add a **spec registry** — a lightweight manifest (e.g., `.crosscheck/specs.json`) that records:

- Which functions have verified Dafny specs
- The Dafny source file path (or inline spec hash)
- The extracted code file path and function signature
- Last-verified timestamp and difficulty metrics

Then add a **`/check-regressions`** skill that:

- Scans the registry for specs whose associated source files have changed since last verification
- Re-runs `dafny_verify` on affected specs
- Reports which properties still hold and which need re-verification
- Can be wired into a pre-commit hook or CI pipeline

This is the single highest-value idea from the paper for Crosscheck.

### Idea 4: LLM feedback integration — Adopt partially

**Paper's proposal**: Verifier failures feed directly back into the LLM's edit loop, autonomously.

**Pushback on full autonomy**: The orchestrator already iterates up to 5 times on Dafny errors. But making verification failures *autonomously* redirect the coding agent (outside of an explicit `/spec-iterate` session) is risky: the agent might "fix" code to satisfy a stale or wrong spec rather than updating the spec. The paper acknowledges this risk only briefly.

**What to do instead**: When Crosscheck is used within the orchestrator, it already does LLM feedback integration. The improvement would be:

- When `/check-regressions` detects a failure, generate a **structured diagnostic** that can be pasted into a Claude Code session: "Function `split_energy` at `billing/utils.py:42` no longer satisfies postcondition `period1 + period2 == total`. The edit at line 45 changed..."
- Let the *developer* decide whether to feed this to the agent or update the spec
- Optionally support an `--auto-fix` flag for CI contexts where the developer has opted in

### Idea 5: Soft constraints / spec relaxation — Nice to have

**Paper's proposal**: Developers can relax specs from hard constraints to soft suggestions.

**What to do**: This maps naturally to Crosscheck's existing tiered approach. A spec could be marked as:

- **Hard**: Must pass `dafny_verify` (current behavior)
- **Soft**: Generate property-based tests from the postconditions, but don't require formal proof

The spec registry (Idea 3) could track constraint strength. `/lightweight-verify` already provides the "soft" path; the gap is just connecting it to the registry.

---

## Recommended priority order

| Priority | Idea | Effort | Impact |
|----------|------|--------|--------|
| **1** | Spec registry + `/check-regressions` | Medium | High — addresses Crosscheck's biggest gap (stateless verification) |
| **2** | `/suggest-specs` autoformalization skill | Medium | High — lowers the barrier to entry, captures implicit invariants |
| **3** | Structured diagnostics for regression failures | Low | Medium — improves developer workflow when regressions are found |
| **4** | Soft constraint support in registry | Low | Medium — connects existing `/lightweight-verify` to the registry |
| **5** | Post-commit verification hook | Low | Low-Medium — useful for CI, but niche |

### What NOT to adopt

- **Continuous side-car on every edit**: Wrong granularity for Dafny. Would make Crosscheck slow and noisy.
- **TypeScript-specific autoformalization templates**: Architecturally misaligned. These belong in a separate linter-weight plugin.
- **Fully autonomous LLM feedback loop**: Too risky without developer-in-the-loop. Crosscheck's current approach (human approves spec → iterate → human reviews) is the right default.

---

## Conclusion

The paper correctly diagnoses that formal verification in vibe coding needs to be **continuous, low-friction, and developer-collaborative** rather than one-shot and expert-driven. Crosscheck already handles depth (Dafny proofs) and has a lightweight fallback (`/lightweight-verify`), but it is **stateless and on-demand only**.

The highest-leverage improvement is a **spec registry with regression detection** — this transforms Crosscheck from a one-shot verification tool into an ongoing correctness guardian, which is the paper's central thesis. Autoformalization via `/suggest-specs` is the second priority, reducing the developer burden that the paper rightly identifies as the primary adoption barrier.

The paper's proof-of-concept (TypeScript syntactic checks via hooks) is orthogonal to Crosscheck and better served by a separate plugin. Trying to merge lightweight linting into a Dafny-based deep verification system would compromise both.
