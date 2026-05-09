# Crosscheck: Final Synthesis

*Definitive comparative assessment. Replaces all prior rounds. Reconciles four sources: an independent cold-read, an independent grounded re-read, a privileged in-repo review, and an adversarial pressure-test.*

---

## How this document was built

This synthesis combines: (a) two independent comparative assessments produced from the original research brief, with different levels of access to the Crosscheck repo; (b) a privileged in-repo review that corrected three load-bearing factual errors in the cold-read; (c) an adversarial round-2 pressure-test that conceded those corrections and probed deeper. Where the four sources agree, the agreement is treated as established. Where they disagree, the synthesis adopts the more privileged source for facts and the more adversarial source for critique. Both round-1 cold-read claims that the privileged review demonstrated were wrong (96% unsourced; `/intent-check` mismarketed as autoformalization; cited arXiv papers can't be corroborated; protected surfaces conflict with Fowler) have been removed from this document and do not appear in the ledger.

---

## 1. What Crosscheck actually is

Crosscheck is a Claude Code plugin that wraps a six-layer assurance model around AI-assisted coding. Layers 1–3 harden the implementation; layers 4–6 harden specifications, intent, and governance. Two orchestrator agents — Byfuglien (implementation chain) and Hellebuyck (specification chain) — route work across the layers. The plugin's claim to value is not algorithmic novelty; it is the productisation of techniques the literature has separately validated, plus a governance scaffolding (kill criteria, protected surfaces, FP trackers, invariant-coverage gates) that is closer to a research contribution in operational discipline than in any single technical primitive.

The nine theoretical commitments are: (1) the *plausible vs correct* framing of LLM failure modes; (2) the *two-orchestrator split* along an impl/spec seam; (3) the *six-layer assurance hierarchy*; (4) *Dafny as Layer 1* with self-disclosed scope limits; (5) *round-trip informalization* in `/intent-check`, claimed at ~96.3% accuracy; (6) *adversarial spec probing* in `/spec-adversary`; (7) *governance scaffolding against drift* (invariant-coverage gates, protected-surface amendments, regression re-verification, FP rate / kill criteria); (8) *semi-formal evidence certificates* in `/rationale`, `/reason`, `/compare-patches`, `/locate-fault`, `/trace-execution`; (9) *cited research grounding* in Ugare & Chandra "Agentic Code Reasoning" (arXiv 2603.01896), Murphy/Babikian/Chechik "Abductive Vibe Coding" (arXiv 2601.01199), and Mitchell & Shaaban "Vibe Coding Needs Vibe Reasoning" (arXiv 2511.00202).

The factual record on each commitment, after privileged review:

- **The 96.3% number is sourced.** It traces to Midspiral's `claimcheck` development benchmark (Graciolli & Amin, Feb 2026), measured on 108 comparisons across 36 requirement-lemma pairs in 5 domains, with 8 deliberately planted bogus lemmas. Both the Midspiral blog post and Crosscheck's `docs/research/literature-review.md` and `docs/research/assurance-hierarchy.md` explicitly hedge it as "a development benchmark rather than a formal evaluation." Crosscheck describes `/intent-check` as "a portable analog of Midspiral's `claimcheck` and an internal Go-binary precursor."

- **`/intent-check` is structural separation, not autoformalization back-translation.** The back-translator LLM receives only `{code, test}` and is explicitly prompted that it has *not* been shown the specification or intent prose. A second LLM compares the prose intent against this independently-produced behavioural description. The pattern matches Midspiral `claimcheck` exactly. The skill ships hardenings: a 30% rolling-FP kill criterion, contradictory-output detection, a truncation guard on `mismatch_reason`, and a mandatory rationale-comment extraction.

- **All three cited 2025–2026 arXiv papers exist** and Crosscheck applies them with deliberate, documented divergence. Internal review documents in `docs/reports/` show, for example, that the Lean backend from "Abductive Vibe Coding" was deferred in favour of prompt-driven `/rationale`, and the TypeScript side-car from "Vibe Coding Needs Vibe Reasoning" was rejected as architecturally incoherent with a Dafny backend (correct on reading the paper directly — the side-car is bound to TypeScript's exhaustiveness and never-typed assertions in a way that doesn't generalise to Dafny).

- **Protected surfaces are governance artefacts, not application code.** The `/assurance-init` Step 5 partition is binary: Class A is harness/workflow definitions (`.claude/agents/**`, workflow YAMLs, prompt templates); Class B is invariant specifications and their property tests (`docs/invariants/*.md`, `**/invariants_prop_test.{go,py,ts}`). Application source code is explicitly excluded. The amend workflow rule "no invariant is being weakened to make a failing test pass" enforces fix-the-code-not-the-spec, which is Fowler-aligned.

- **The semi-formal reasoning skills are genuinely structured.** Outputs require numbered premises tagged `[STATIC|SEMANTIC|BEHAVIORAL|FORMAL]` with mandatory `file:line` citations, an alternative-hypothesis check, and a confidence assessment. Format is enforced by Byfuglien's Phase-4 validation gates.

- **`/assurance-status`'s "FP rate" is precisely defined.** Rolling 14-day false-positive rate of `/intent-check` runs, computed as `count(human_verdict=="spurious") / count(rows in window with non-empty human_verdict)`, requiring ≥3 classified rows. Thresholds: OK <20%, AT RISK 20–30%, TRIPPED ≥30%. The TRIPPED state takes Layer 5 offline.

- **Crosscheck has documented, honest self-criticism.** `docs/reports/crosscheck-field-report.md` (Django billing case) concludes "the verification was correct but not useful for this task," "real bugs were elsewhere," "extracted code wasn't usable" — and the field report is presented as the source of subsequent design decisions on complexity gating and `/lightweight-verify`.

- **Crosscheck has empirical reach calibration.** `docs/research/logic-distribution-analysis.md` reports static analysis across 14 codebases (~2.5M LOC) showing pure functions are 22–27% of full-stack web apps (15–25% after correction). This is the actual reach ceiling for Layer 1.

---

## 2. Final ledger across the nine commitments

| # | Commitment | RiSE / academic | nl2spec / TiCoder | Phoenix / semi-formal | Synthesis verdict |
|---|---|---|---|---|---|
| 1 | Plausible vs correct framing | **Confirmed** | **Confirmed** | **Confirmed** | Settled. No remaining critique. |
| 2 | Two-orchestrator split | **Refined** — RiSE diagnoses spec as the heavier side; Crosscheck's split is principled but presents symmetric-looking ownership while actually weighting Byfuglien at 1 layer + 4 orthogonal skills vs Hellebuyck at 3 layers + governance | Orthogonal | **Refined** — Vibe Reasoning's continuous-sidecar argument suggests Hellebuyck should own more continuous monitoring | Principled in shape, asymmetric in substance, under-disclosed in README |
| 3 | Six-layer assurance hierarchy | Conceptually aligned but unvalidated empirically | Orthogonal | **Refined** — pace-layer thinking would partition by regeneration speed rather than verification technique | Useful as practitioner onboarding; not yet evidence-based |
| 4 | Dafny as Layer 1 | **Confirmed in principle, constrained in scope** — Lahiri's Dafny work supports it for pure cores; FMCAD spec evaluation requires Dafny-grade specs | Orthogonal | **Constrained** — Phoenix would treat Dafny code as regenerable from spec, not as the durable artefact | Defensible for the 22–27% reach band; README overpromises beyond it |
| 5 | `/intent-check` round-trip, ~96.3% | **Partially confirmed in technique** (structural separation is sound), **constrained in metric** (Lahiri FMCAD argues correctness/completeness should be reported, not single accuracy) | **Refined** — nl2spec's sub-translation pattern is the natural complement | **Confirmed in spirit, autonomous in execution** — TiCoder shows interactive yes/no/undefined unlocks the bottleneck, which Crosscheck currently doesn't have | Sourced and structurally sound; metric is one-dimensional and unvalidated outside the Midspiral corpus |
| 6 | `/spec-adversary` | **Refined** — known technique under a less rigorous name; SPOTs (Swamy & Lahiri 2026), IronSpec (OSDI '24), MutDafny (2511.15403) provide deterministic discrimination signals it lacks | Orthogonal | Confirmed in spirit | Honest about its scope; could be augmented with mutation kicker on HIGH-confidence proposals |
| 7 | Governance scaffolding | **Confirmed** | Orthogonal | **Strongly confirmed** — closest match to Phoenix's "non-deletable primitives" thesis | Crosscheck's strongest contribution; rare in the literature at this operational depth |
| 8 | Semi-formal evidence certificates | **Directly confirmed** by Ugare & Chandra | Orthogonal | **Confirmed** — Murphy et al.'s claim trees are an abductive variant; Crosscheck adopts the shape and defers the Lean backend | Faithful adaptation; "certificate" terminology is slightly stronger than the cited papers warrant |
| 9 | Cited research grounding | Faithful at the level of ideas; deliberate divergences are documented in internal review docs | n/a | n/a | Genuinely engaged with the literature; could foreground Lahiri's "Intent Formalization" essay (2603.17150) and FMCAD 2024 more prominently |

**Where the lenses disagree about a commitment**, the disagreement is concentrated on commitment 4 (Dafny as Layer 1). Academic and Phoenix lenses point in the same direction — *lighten the integration boundary, centre evaluations and lightweight specs* — but academic accepts logical contracts as the high end of a useful spectrum, while Phoenix would treat all code (including Dafny code) as regenerable from the durable layer. Crosscheck currently sits in the academic position; the README marketing surface drifts further than the docs themselves do.

---

## 3. The seven cross-examinations: final answers

**CX1. Right slice of the intent-formalization spectrum?** No, and academic and Phoenix lenses are pushing from the same direction. Lahiri's spectrum (tests → code contracts → logical contracts → DSLs) places the value-per-friction peak in the middle. TiCoder gets +22 to +37 percentage points pass@1 on MBPP with one user query. Endres et al. (FSE 2024) catch real Defects4J bugs with LLM-generated postconditions. Phoenix says evaluations are the codebase. Crosscheck's *anchor* at Dafny is defensible for a small, identified band of code; its *default narrative weight* on Dafny is misaligned with both lenses. The fix is the README and routing defaults, not the engine.

**CX2. Does `/intent-check`'s round-trip correspond to a validated technique?** Yes. Structural separation (back-translator blind to prose) is structurally adversarial and is the same pattern Midspiral's `claimcheck` blog post articulates. The 96.3% number is sourced and self-disclaimed by its authors. The remaining critique is not provenance but base rate: 8 of 36 lemmas (22%) were *planted* bogus, which inflates the discrimination task relative to a wild distribution where real spec drift is presumably rarer than 22% of all lemmas. The number should be read as an upper bound on a synthetic benchmark, not as wild detection rate. The literature offers two complementary techniques the skill currently doesn't use: nl2spec-style fragment-level sub-translation alignment (post-hoc, after a mismatch fires) and Lahiri-FMCAD-style soundness/completeness reporting (alongside the FP rate, not replacing it).

**CX3. How does `/spec-adversary` compare to mutation/completeness methods?** It's a known technique under a less rigorous name, and the SKILL is honest about that ("Layer 6 has no deterministic tool"). IronSpec (Goldweber et al., OSDI 2024) and MutDafny (arXiv 2511.15403) provide deterministic mutation-based signals against Dafny specs. The full IronSpec/MutDafny campaigns are not plugin-shaped (heavyweight, multi-node), but the *signal* — "here is a one-line mutant the spec fails to kill" — is plugin-shaped: it's one `dafny verify` call. A scoped optional `--mutate` mode that runs a fixed 5-operator MutDafny-derived set on the changed function would convert MEDIUM-confidence proposals into HIGH on objective evidence.

**CX4. Does the two-agent split align with how RiSE frames the problem?** Principled in shape, asymmetric in substance. RiSE diagnoses spec validation as the bottleneck and verification as downstream. Crosscheck's actual ownership graph (Byfuglien at 1 layer + 4 orthogonal semi-formal reasoning skills; Hellebuyck at 3 layers + governance) reflects this asymmetry, but the README presents the two orchestrators as if their workloads were equal. A user who reads the README does not see this. The semi-formal reasoning skills sit *outside* the layer hierarchy entirely — they are an orthogonal third axis (Ugare-Chandra-style premise-tagged certificates), and routing them under "Byfuglien's chain" flattens the trust calculus a serious user must perform.

**CX5. What does Crosscheck not do that prior art has shown matters?** The five real gaps:

1. **Spec soundness/completeness metrics** in the Lahiri FMCAD sense — currently absent; FP rate is a different metric.
2. **TiCoder-style interactive disambiguation** (yes/no/undefined on candidate tests) — absent; this is the empirically validated unlock for intent formalization at scale.
3. **Mutation-based discrimination signals on `/spec-adversary`** — absent; would tighten Layer 6 cheaply.
4. **Continuous verification hooks** in the Vibe Reasoning Type-III-sidecar sense — Crosscheck's flows are discrete (run-on-invocation) rather than continuous (run-on-edit).
5. **Cross-family back-translation in `/intent-check`** — currently the back-translator is a different prompt with potentially the same model family as the spec author; correlated drift can survive structural separation.

The compositionality-of-change-intent gap (Lahiri open-problem #2) is also real but is correctly out of scope for a plugin.

**CX6. Where does Crosscheck genuinely advance the state of the art?** Honest answer: it doesn't advance the algorithmic SOTA, it productionises and operationalises it. Its contributions are architectural and operational. The strongest specific contributions are:

- The **governance scaffolding** (kill criteria, protected surfaces, FP trackers, invariant-coverage gates as both pre-commit and CI) is operationalised at a depth few research prototypes reach.
- The **6-layer hierarchy as practitioner onboarding** is a reusable framing tool, even though the empirical calibration of the layer model is still missing.
- The **deployment as a Claude Code plugin** with real Dafny verification in a Docker sandbox is a real engineering artefact, not a prompt pack.
- **The internal review documents** in `docs/reports/` (field report's Django billing post-mortem, abductive-vibe-coding-review, vibe-reasoning-paper-review) are an unusual methodological signal — Crosscheck's author publishes their own design failures and rejection reasoning, not just their successes.

This is a valid finding. "Productionises existing ideas with rare operational discipline" is more durable than novelty in this space.

**CX7. Is Crosscheck protecting the right artefact?** Mostly, yes. The privileged review settled the central question: protected surfaces are strictly governance artefacts (Class A: harness/workflow definitions; Class B: invariant specs and their property tests), and application source is explicitly excluded. The amend rule "no invariant is being weakened to make a failing test pass" enforces fix-the-code-not-the-spec direction. By Fowler's Deletion Test, Crosscheck passes: delete the Python/Go output and Dafny code, the invariants and acceptance oracles persist, the system is regenerable. The remaining tension is not the protected-surface partition but the README's narrative weight on Dafny Layer 1 — which can read as code-as-asset to a user who doesn't open the docs.

`/acceptance-oracle-draft` is the Phoenix-aligned skill in the catalogue; both independent assessments converge on this. It deserves more prominence in the default workflow than the current docs give it.

---

## 4. The remaining critique surface

After the privileged corrections and the adversarial pressure-test, the actual critique surface is narrower than any single round suggested. There are three categories.

**Category A — Real and operational.** These argue for code or configuration changes:

- **A1. Cross-family back-translation in `/intent-check`.** Structural separation defeats tautological restatement; it does not defeat correlated drift between code and tests written under the same misinterpretation of prose. The Midspiral `claimcheck` blog explicitly names this risk ("if the LLM misreads the Dafny, the comparison is comparing two wrong things"). Running the back-translator under a different model family than the code/spec drafter partially decorrelates the drift.
- **A2. Optional MutDafny-derived mutation kicker on `/spec-adversary`.** A 5-operator scoped mode (relational-operator swap, off-by-one, conditional negation, return-value zeroing, default-value substitution) with a 60-second budget would give HIGH-confidence proposals an objective tie-breaker. Keeps the SKILL's "best-effort" framing intact.
- **A3. Post-mismatch sub-translation alignment in `/intent-check`.** When `match=false` fires, run a single follow-up call that produces nl2spec-style `(prose-span, formal-fragment)` pairs to localise the divergence. Cost: one extra API call per mismatch. Benefit: the field-report-style "verification was correct but not useful" failure mode becomes actionable.
- **A4. Calibration of the 30% kill threshold.** Either cite a labelled trace ("threshold set by N-day pilot, M human verdicts, trip line at distribution elbow") or label the threshold as "founder intuition pending operational data" and make it configurable. The Noisy-but-Valid framework (Feng et al., arXiv:2601.20913) is the published protocol for deriving such thresholds.

**Category B — Real and architectural.** These argue for catalogue and documentation reshape:

- **B1. README chain-of-trust on Layer 1.** "Provably correct Python/Go" should be replaced with "Verified in Dafny (Layer 1); compiled to Python/Go via the Dafny backends; embedding correctness is your responsibility." The docs already say this; the README just needs to inherit the docs' register.
- **B2. README reach-ceiling disclosure.** State once that Layer 1 targets pure functional cores at empirically 22–27% of typical full-stack codebases per `docs/research/logic-distribution-analysis.md`, with the remaining ~75% governed by Layers 3–6.
- **B3. Persona / orthogonal-axis disclosure.** Add a one-paragraph table to the README that names skills under each persona and flags the four semi-formal reasoning skills (`/reason`, `/compare-patches`, `/locate-fault`, `/trace-execution`) as orthogonal to both — they belong in a third column, not under either persona. Resolves both the symmetry-illusion and the layer-vs-non-layer flattening.
- **B4. Skill-catalogue consolidation toward Phoenix-style primitives.** The current catalogue has substantial breadth (verify-core-logic, maintain-invariants, maintain-evaluations, govern-protected-surfaces, probe-specs is a candidate five-primitive grouping). Single entrypoint per primitive with subcommands rather than a flat skill list reduces cognitive load and makes the architectural shape legible.
- **B5. Default-workflow re-centring on evaluations.** The default Crosscheck path should be: `/assurance-init` → `/acceptance-oracle-draft` → interactive disambiguation (new skill, see C2 below) → optional Dafny Layer 1. This aligns the user mental model with both Phoenix and the intent-formalization literature without changing what the engine does.

**Category C — Real and methodological.** These argue for adding capabilities the literature has shown matter:

- **C1. Lahiri-FMCAD-style soundness/completeness metrics.** Implement a `/spec-eval` skill that, given a Dafny spec and its tests, computes correctness and completeness scores via Hoare-triple encoding with mutated outputs. For modules with invariant docs but no Dafny code, approximate via property tests and mutation testing. Surface alongside FP rate, not as a replacement.
- **C2. TiCoder-style interactive test clarification.** Add a mode where Crosscheck proposes tests or acceptance scenarios and asks the user yes/no/undefined. Use responses to prune scenarios and refine invariants. Track pass@k@m and accept@m. This is the empirically validated unlock for intent formalization that Crosscheck currently has no analogue for.
- **C3. Continuous verification hooks (Vibe Reasoning Type-III sidecar).** A lightweight CI mode or daemon that runs invariant coverage, `/intent-check` on protected surfaces, and acceptance oracles on a schedule or per-branch basis, rather than only on manual invocation. Prioritises by change impact and past failures.
- **C4. SPOTs-inspired spec gap probe.** Generate small proof-oriented tests for Dafny modules or property tests; measure whether invariants/specs fail when they should. Complements `/spec-adversary` with deterministic gap detection.

Categories that **dissolve** post-corrections: 96% is unsourced (it's sourced); `/intent-check` is autoformalization mismarketed (it's structural separation); cited papers can't be corroborated (they exist); protected surfaces conflict with Fowler (they are governance-only).

---

## 5. Threats to validity that hold up

Six assumptions in Crosscheck that the prior art puts pressure on, in order of severity:

1. **The 6-layer hierarchy is normative, not empirically calibrated.** No data exists yet on which layers catch real bugs, how often kill criteria fire, or what false-negative rates look like across the layer stack. This is the largest gap between Crosscheck-as-documented and Crosscheck-as-evidence-based.
2. **`/intent-check` is treated as a hard merge gate, but its quantitative basis is one-dimensional.** Single-accuracy on a synthetic discrimination task with a 22% planted-error base rate is not a soundness/completeness statement. Until C1 ships, the gate carries more weight than the metric supports.
3. **Invariant docs are treated as ground truth.** `/intent-check` and `/spec-adversary` build on top of them. RiSE and FMCAD 2024 both show specs themselves can be wrong, weak, or vacuous. Until spec-quality metrics are integrated, the machinery rests on unaudited foundations.
4. **High-discipline maintenance is assumed.** Protected surfaces, invariant coverage gates, FP trackers, and governance notes only work if a team consistently maintains them. Without strong automation and clear payoff signals, governance scaffolding decays — the documented Django billing field-report case is a partial canary for this.
5. **Skill-catalogue complexity itself.** A large flat catalogue increases cognitive load. The Phoenix lens specifically warns that breadth-as-architecture is "better prompts dressed as architecture." The 5-primitive consolidation in B4 is the targeted fix.
6. **Dafny-verified cores dominate risk.** The right axis here is *functional vs behavioral specification*, not "I/O, concurrency, integration." Dafny verifies functional correctness on pure cores; in many real systems the most serious bugs are *behavioral* — emergent from rule interactions, state transitions, or workflow branching across reachable states a human can't enumerate by inspection. This bug class is exactly what TLA+/P/Alloy address, and it is not limited to distributed code: Cedar (stateless, single-node, rule-dense) shows the relevant criterion is rule-density / state-explosion, not concurrency or distribution. Treating Layer 1 as "the hard part" risks under-investing in (a) Layers 2–3 where general implementation bugs occur and (b) the Layer 4 enrichment path (executable behavioral specs) where rule-dense modules need verification Dafny cannot provide.

---

## 6. Improvements ranked by impact-to-effort

Combining the operational, architectural, and methodological items above:

**Tier 1 — high impact, low effort.** README chain-of-trust on Layer 1 (B1). Reach-ceiling disclosure (B2). Persona / orthogonal-axis disclosure (B3). Default-workflow re-centring (B5). Threshold calibration disclosure (A4). Cite Lahiri's "Intent Formalization" essay and FMCAD 2024 prominently. All of these are documentation changes the docs already support; the README just needs to inherit the docs' register.

**Tier 2 — high impact, medium effort.** Cross-family back-translation in `/intent-check` (A1). Post-mismatch sub-translation alignment (A3). Optional mutation kicker on `/spec-adversary` (A2). TiCoder-style interactive disambiguation as a new skill (C2). Skill-catalogue consolidation into ~5 Phoenix-style primitives (B4).

**Tier 3 — medium-to-high impact, medium-to-high effort.** Lahiri-FMCAD soundness/completeness metrics (`/spec-eval`, C1). Continuous verification hooks / Type-III sidecar mode (C3). SPOTs-inspired spec gap probe (C4).

**Tier 4 — high impact, high effort, longer horizon.** Empirical calibration of the 6-layer hierarchy itself: instrument the plugin to publish *which layer caught what*, *which kill criteria fired and when*, and *what fraction of the codebase reached each layer*. Without this, the hierarchy remains an elegant theory rather than evidence-based engineering practice. This is the single largest gap between Crosscheck-as-documented and Crosscheck-as-justified.

The single highest-leverage move is the consolidation of the documentation surface — Tier 1 above, plus a one-page "Honest Map" at the top of the README that lists each layer, what it actually guarantees, what fraction of code it reaches, who owns the skill, which cited paper it derives from, and the kept-vs-deferred decision in one column. Every Tier-1 item is a specific piece of that map. Every documented round-1 misreading would have been prevented by it. The docs already contain every word of the map; surfacing it as the *first* thing a prospective user reads converts a class of predictable misreadings into non-issues without changing a single piece of code.

---

## 7. Open questions

These remained unresolvable from outside the repo and depend on author intent or unpublished implementation details:

1. **Has the 96.3% been re-measured on a non-Midspiral corpus?** The number is sourced and self-disclaimed by its authors. It has not, to public knowledge, been validated on Crosscheck's own all-LLM port against user repositories.
2. **Is `/lightweight-verify`-as-default actually wired into the workflow YAMLs**, or is the recommendation documentary while the inherited Dafny-by-default routing is what actually runs? The field report grounds the recommendation; the wiring is what tests whether the lesson was internalised.
3. **What is the calibration trace behind the 30% FP kill threshold?** Founder intuition or labelled-pilot data?
4. **Have any cross-layer interactions been documented over time on a multi-module system?** End-to-end workflows for combining Dafny Layer 1 with Hellebuyck's Layers 4–6 are sketched but not, as far as I could verify, demonstrated on a realistic case study.
5. **Does the Intent Stack reference framework (intentstack.org) inform Crosscheck's layer model directly, or is the convergence on layered governance independent?** The parallel agent's report names the Intent Stack as a peer; Crosscheck's docs do not appear to engage with it explicitly.
6. **What is Crosscheck's empirical layer-attribution data?** Tier 4 above. If even a small labelled trace exists internally — "in N runs, layer X caught M bugs" — publishing it would be the single most credibility-enhancing artefact the plugin could ship.

---

## 8. Net assessment

Crosscheck is a serious, methodologically honest plugin that productionises a slice of the AI-coding-assurance literature with rare operational discipline. Its contributions are architectural and operational rather than algorithmic: the governance scaffolding (kill criteria, FP trackers, protected surfaces, invariant-coverage gates as both pre-commit and CI), the 6-layer onboarding framing, the publication of internal failure post-mortems, the deliberate documented divergence from cited papers (Lean deferred, TS side-car rejected). These are the durable contributions, and they are unusual in the field at this depth. The Dafny verification engine and the structural-separation `/intent-check` skill are faithful adaptations of upstream work (Midspiral `claimcheck`, the Dafny ecosystem, Ugare-Chandra reasoning), not novel primitives.

The actionable critique surface is narrower than the round-1 cold-read suggested but real: a documentation surface that consistently overpromises relative to honest underlying docs; a 96.3% number whose base rate is more synthetic than the framing implies; a `/spec-adversary` skill that could be sharpened cheaply with mutation kickers; an `/intent-check` skill that could add cross-family back-translation and post-mismatch sub-translation alignment without violating its existing design; a missing TiCoder-style interactive layer where the literature shows the largest empirical unlock; a six-layer hierarchy that remains uncalibrated against real bug-catch data. The single highest-leverage change is documentary, not algorithmic: a one-page "Honest Map" at the top of the README that surfaces what the docs already say. The pattern across multiple independent assessments is consistent: when the README is read cold, certain misreadings are predictable; when the docs are read in full, they are usually right. That asymmetry is the one thing worth fixing first, because it would prevent the next critic from spending three rounds rediscovering the same thing.
