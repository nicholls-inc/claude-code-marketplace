# Re-Assessment of the Crosscheck Critical Analysis

The previous critic produced a thoughtful piece of work, but it was crippled by a single retrieval gap — they could not fetch GitHub `blob/` or `raw.githubusercontent.com` URLs, which means they never read the four most load-bearing documents: `docs/research/literature-review.md`, `docs/research/assurance-hierarchy.md`, the SKILL.md files, or the four internal review documents in `docs/reports/`. With the actual files in hand, the headline critique inverts. Below I keep the previous critic's structure (numbered findings; my verdict in bold) so the deltas are easy to audit.

## What the critic got wrong

### F1 — "The 96% has no traceable source" → **WRONG. The figure is properly attributed.**

The 96% figure is **96.3%**, traced explicitly in two places the critic could not read:

- `docs/research/literature-review.md:29` (Midspiral entry): *"Reported accuracy is 96.3%, acknowledged as a development benchmark rather than a formal evaluation."*
- `docs/research/assurance-hierarchy.md:43`: *"Claimcheck reports approximately 96.3% accuracy using structural separation (two models, with the informalizer blind to the original requirement), though this is acknowledged as a development benchmark rather than a formal evaluation."*

The number comes from **Midspiral's `claimcheck` tool**, not TiCoder, not Endres et al., not anything the critic speculated about. The hierarchy doc explicitly hedges that this is a development benchmark, not a formal evaluation. The critic's entire CX2 ("the 96% should be sourced... the most charitable explanation is an internal benchmark... the least charitable is someone rounded TiCoder's 90.40%") is invalidated.

### F2 — "`/intent-check` is autoformalization disguised as intent formalization" → **WRONG. It's the structural-separation pattern from Midspiral claimcheck.**

`/intent-check`'s SKILL.md states its lineage in the second paragraph: *"The skill is a portable analog of Midspiral's `claimcheck` and an internal Go-binary precursor."* The critical structural distinction the critic missed is in `references/round-trip-prompt.md` and in the SKILL itself:

- The back-translator receives **only `{code}` and `{test}`** — never the invariant prose. The prompt says, verbatim: *"You have NOT been shown any specification or intent prose. You must describe only what the code + test enforce, not what they were meant to enforce."*
- A **second LLM** then compares the prose intent against this independently-produced behavioural description.

This is **not** autoformalization back-translation (where you translate F → NL and check it against the original NL prompt; Lahiri rejects this because the original NL is incomplete). It is **structurally adversarial**: the back-translator cannot tautologically restate the prose because it never sees it. This is exactly Lahiri's "structural separation" pattern, applied at the (code+test) layer rather than the spec layer. The critic conflated two different round-trip techniques.

The skill also ships hardening the critic didn't know about: a 30% rolling-FP kill criterion, contradictory-output detection (auto-flips `match=true` with non-empty `mismatch_reason`), a truncation guard (`mismatch_reason < 20 chars` → reject), and a mandatory rationale-comment extraction (calibration finding FP #6) that the critic's "subtranslation alignment" suggestion is actually a coarser version of.

### F3 — "Three of the four cited arXiv papers I could not corroborate" → **WRONG. All three exist; subagent verified.**

A delegated agent verified all three on arXiv:

- **Ugare & Chandra, "Agentic Code Reasoning"** (arXiv 2603.01896, March 2026) — abstract describes "semi-formal reasoning" with patch-equivalence (78%→88%, 93% real-world), Defects4J fault localization (+5pts Top-5), and code QA (87% RubberDuckBench). Maps cleanly to `/compare-patches`, `/locate-fault`, `/reason`, `/trace-execution`. `/reason`'s SKILL.md cites it by name.
- **Murphy, Babikian & Chechik, "Abductive Vibe Coding (Extended Abstract)"** (arXiv 2601.01199, January 2026, U. Toronto) — proposes hierarchical claim trees with `(C1 ∧ … ∧ Cn) ⟹ C` decomposition. The internal review at `docs/reports/abductive-vibe-coding-review.md` documents the design choice: adopt the claim-tree shape, **reject** the paper's Lean backend in favor of prompt-driven `/rationale`.
- **Mitchell & Shaaban, "Vibe Coding Needs Vibe Reasoning"** (arXiv 2511.00202, LMPL '25) — autoformalization side-car. The internal review at `docs/reports/vibe-reasoning-paper-review.md` is explicit about what was kept (constraint-tracking → `/check-regressions`) and what was rejected ("continuous side-car on every edit: wrong granularity for Dafny"; "bolting TypeScript type-system checks onto a Dafny-based plugin would be architectural incoherence").

The critic's retrieval failure was a false negative — likely because two of the three papers post-date the January 2026 training cutoff.

### F4 — "`/protected-surface-amend` may protect code rather than spec, contradicting Fowler" → **WRONG. It strictly protects governance artifacts.**

The protected-surfaces partition, scaffolded by `/assurance-init` at Step 5, is **binary and exhaustive**:

- **Class A — Harness/workflow definitions**: `.claude/agents/**`, `.claude/rules/**`, workflow YAMLs (`.github/workflows/**`, `.gitlab-ci.yml`), prompt templates, "any file the harness interprets as 'ground truth' for agent behaviour."
- **Class B — Module invariant specifications and tests**: `docs/invariants/*.md` and the property-test files that cover them (e.g., `**/invariants_prop_test.{go,py,ts}`).

Application source code (`src/cache.py`, business logic, anything outside these two classes) is **explicitly not** a protected surface. The amendment workflow at Step 7 even states: *"No invariant is being weakened to make a failing test pass"* — the direction is fix-the-code, not fix-the-spec, which is **exactly** Fowler-aligned. The critic's most-uncertain Fowler-vs-Crosscheck conflict (CX7) dissolves: protected surfaces are spec/intent files, not code.

### F5 — "`/assurance-status`'s 'FP rate' is unclear" → **RESOLVED.**

Step 2.4 of the SKILL pins it down to spec-level precision: it's the rolling 14-day false-positive rate of `/intent-check` runs, computed as `count(human_verdict=="spurious") / count(rows in window with non-empty human_verdict)`. Window must have ≥3 classified rows; verdict thresholds are `OK <20%`, `AT RISK 20-30%`, `TRIPPED ≥30%`. The 30% kill criterion takes Layer 5 offline — it is not "operational vs epistemic" handwaving as the critic suggested.

### F6 — "`/spec-adversary` is mutation testing under a less rigorous name" → **PARTIALLY RIGHT, BUT THE SKILL IS HONEST ABOUT IT.**

The SKILL is explicit: *"Layer 6 of the assurance hierarchy — spec completeness — has no deterministic tool. This skill operationalises the 'adversarial invariant proposer' pattern... Success is 'at least one non-obvious candidate per meaningful change,' not 'zero missed properties.'"* It uses HIGH/MEDIUM/LOW confidence + an accept/reject/defer triage block + signal-to-noise kill criteria (S/N <1:5 after 4 weeks → retire). The critic's improvement suggestion (add SPOTs/MutDafny-style mutation testing as a discrimination signal) is a fair upgrade — but it's not a misrepresentation; it's a Layer-6 best-effort tool that says so.

### F7 — "Lahiri's 'Intent Formalization' is uncited" → **MISLEADING.**

The critic is right that Lahiri's essay isn't cited *by name* in the README. But the literature review **does** cite Midspiral's claimcheck, which is the operational tool the essay points at. The Midspiral entry is over half a page and engages with the technique directly. This is "different citation strategy," not "ignored prior art."

### F8 — "Provably correct Python/Go" overclaim in README vs. "trust your toolchain" hedge in docs → **CORRECT.**

The README at line 10 says *"Dafny-verified Python or Go. The compiler refuses to emit code that doesn't satisfy its spec."* — but the docs at `docs/assurance-hierarchy.md` are honest that Layer 2 is *"Not addressed — trust your toolchain."* The critic's recommendation to rephrase the README is fair.

### F9 — "Byfuglien/Hellebuyck symmetry is misleading" → **PARTIALLY CORRECT.**

The README sells two-orchestrator symmetry, but the actual layer ownership is asymmetric: Byfuglien owns Layer 1 + four out-of-hierarchy semi-formal reasoning skills; Hellebuyck owns Layers 4–6 + governance. The agents.md files are honest about this; the README compresses it. Fair recommendation to surface the asymmetry.

## What the critic got right

- **README "six layers" framing oversells what ships** (Layers 2–3 are punted, docs say so honestly, README doesn't).
- **`/spec-adversary` could be strengthened** with mutation-based discrimination signals (FMCAD-2024-style or SPOT-style).
- **`/intent-check` could be strengthened** with sub-translation-style fragment alignment (nl2spec-style) — though this is additive, not corrective; the structural separation it already has is valid.
- **Crosscheck has no TiCoder-style interactive Yes/No/Don't-know disambiguation** — true gap.
- **The semi-formal reasoning skills sit outside the layer hierarchy** — accurate observation; they are an orthogonal third concern.

## What I'd add the critic missed

1. **`docs/reports/crosscheck-field-report.md` is openly self-critical**: on a Django billing case, it concludes *"The verification was correct but not useful for this task"*, *"Real bugs were elsewhere"*, *"Extracted code wasn't usable"*. This is the source of the complexity-gating and `/lightweight-verify`-as-default recommendations baked into Byfuglien's classification table. The plugin already internalises the "Dafny is too heavyweight by default" critique the critic raises.

2. **`docs/research/logic-distribution-analysis.md`** is a static analysis across 14 codebases (~2.5M LOC) showing pure functions are 22–27% of full-stack web apps, with bounds 15–25% after correction. This is real empirical grounding for the reach claims that the critic treated as ungrounded.

3. **`/reason` and the semi-formal reasoning skills are genuinely structured**, not marketing language. Each output requires numbered PREMISES with `[STATIC|SEMANTIC|BEHAVIORAL|FORMAL]` tags + mandatory `file:line` citations + alternative-hypothesis check + confidence assessment. The format is enforced by Byfuglien's Phase-4 validation gates, which reject outputs that skip steps.

4. **The internal reviews show deliberate divergence from cited papers** — not unattributed copying. The vibe-reasoning-paper-review explicitly rejects the paper's TypeScript-template approach as "architectural incoherence" with a Dafny backend. This is the opposite of the "uncritically operationalises Ugare & Chandra" framing the critic gestured at.

## Bottom line

The previous critic's three load-bearing claims — (1) the 96% has no source, (2) `/intent-check` is autoformalization mismarketed as intent formalization, (3) three of four cited papers can't be corroborated — are all wrong, all because of the same retrieval gap. Once the docs/research, docs/reports, and skill SKILL.md files are read, the picture inverts: the citations are real, the technique lineage is documented, the FP-rate semantics are pinned to spec-level precision, and the protected-surface partition is governance-only (Fowler-aligned).

The real remaining critique surface is much narrower: (a) the README oversells "provably correct" relative to the docs' honest "trust your toolchain"; (b) the Byfuglien/Hellebuyck symmetry hides Layer 1 ownership asymmetry; (c) `/spec-adversary` could be strengthened with mutation testing; (d) `/intent-check` could add sub-translation-style fragment alignment as a complement to (not replacement for) structural separation; (e) the semi-formal reasoning skills don't sit cleanly inside the 6-layer hierarchy. These are improvements, not credibility issues.

The previous analysis should be treated as a stress-test for what the README looks like to a reader who can't open the linked docs — and the corrective is to lift more of `docs/research/literature-review.md` (specifically the Midspiral claimcheck attribution and the 96.3% figure) into the README itself, so the citation chain doesn't depend on a clickthrough that the critic's environment couldn't follow.
