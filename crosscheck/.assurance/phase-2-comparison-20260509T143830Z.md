# Phase 2 Comparison Report — ADD into Crosscheck

**Author:** Claude Code agent (Phase 2 self-validation)
**Date (UTC):** 2026-05-09T14:38:30Z
**Inputs:**
- Companion file: `.assurance/phase-2-back-translation-20260509T143830Z.md` (the agent's blind back-translation)
- `crosscheck/docs/add/intent.md` (Drafted v1.0)
- `crosscheck/docs/add/specs/architectural.md` (Drafted v1.0)
- `crosscheck/docs/add/methodology.md`, `glossary.md`, `decisions/INDEX.md`, `decisions/ADR-001` … `ADR-005`, `acceptance.md` (all read during cold read)

**Status:** Drafted — input to the human's attestation decision; not itself an attestation.

**Out-of-band note on path.** The acceptance protocol writes outputs to `.assurance/`. The agent placed them under `crosscheck/.assurance/` because the ADD seed lives at `crosscheck/docs/add/`, and the existing `.assurance/` directory in the repo is `crosscheck/mcp-server/.assurance/`. If the human prefers `<repo-root>/.assurance/` or `crosscheck/docs/add/.assurance/`, the files can be moved as part of attestation; the path is not load-bearing.

---

## 1. Matches per Intent Claim

For each `IC`, I report whether the back-translation captured the claim, the architectural spec consumes it, and the consuming sections plausibly satisfy the claim. *Full* means the back-translation captured the substance and the spec coverage looks adequate; *Partial* means substance captured but coverage has a gap; *Divergent* means the back-translation interpretation drifted from the seed; *Silent* means the back-translation said nothing about it.

| IC | Title (paraphrased) | Back-translation match | Spec consumption | Verdict |
|---|---|---|---|---|
| IC1 | Empty-repo entrypoint | Full | S2.1, S3.1 | Full |
| IC2 | Phase 0 explicit, repo-resident artifact | Full | S2.1 | Full |
| IC3 | Phase 1 spec stack derives from intent | Full | S2.2 | Partial — see § 3.1 |
| IC4 | Phase 2 prose-vs-prose validation | Full | S2.3, S2.4 | Partial — see § 5.3 (FP definition) |
| IC5 | Three operating modes with per-module tagging | Full but flagged | S1.1, S1.2 | Partial — see § 4.1 (transitional mode wording) |
| IC6 | Auditor agent runs consolidation passes | Full | S5.1, S5.2 | Full |
| IC7 | Diff classification enforced | Full | S6.1 | Partial — see § 5.7 (rebase/squash, framework detection) |
| IC8 | Deterministic instrumentation | Full | S4.1, S4.2 | Partial — see § 3.5 (one extra signal in spec) |
| IC9 | Existing flows unchanged | Full | S3.* (and S1.1 default) | Full but coverage table incomplete (§ 3.4) |
| IC10 | Documentation surfaces ADD honestly | Full | S7.1, S7.2, S7.3, S7.4 | Full |

Of ten ICs: seven Full, three Partial. No Silent or Divergent.

---

## 2. Gaps in intent

Things the back-translation expected the intent doc to address but it does not.

**G-Intent-1 — Mode-detection responsibility.**
IC1 says the entrypoint "either recognises the empty-repo state and routes" or "is a new ADD-mode skill the user invokes directly." The intent doc does not specify *who or what is responsible for detecting empty-repo state vs. mode tags vs. ADD-mode state*. The architectural spec spreads this across `/assurance-layer-audit` (S3.1), `/assurance-init` (S3.2), and `/acceptance-oracle-draft` (S3.5), but the intent claim is silent on whether mode detection is per-skill, plugin-level, or a separate concern. Severity: low (the spec resolves it; the intent claim could be more precise but does not need to be).

**G-Intent-2 — Threat-model traceability.**
The intent doc enumerates `TM1`–`TM6` but only `TM2` and `TM4` appear in ADR-005 and ADR-003 `Consumes:` lines respectively. `TM1` (doc proliferation), `TM3` (bootstrap-user surprise), `TM5` (premature attestation optimisation), `TM6` (hypothesis-status drift) are not consumed by any ADR or `S` section. The architectural mitigations exist (small-seed discipline; additive S3 adaptations; explicit per-artifact attestation; honest README), but no `consumes: TMx` line traces them. Severity: low (the mitigations exist; the linkage graph is incomplete on the threat side).

**G-Intent-3 — Auditor verdict-report mutability.**
IC6 says the Auditor produces verdicts without modifying artifacts. The intent doc does not address whether the Auditor's *own report* is itself a protected ADD artifact (and therefore subject to the diff-classification gate). Acceptance A5 names the path (`docs/add/audit/<date>.md`), implying the report is a repo-resident artifact. But ADR-005's protected-paths list does not include `docs/add/audit/`. Severity: medium — without explicit treatment, the auditor's report could be silently amended in a way the methodology does not catch.

---

## 3. Gaps in spec

Things the intent doc states that the architectural spec does not adequately cover.

**G-Spec-1 — The seam between architectural spec and `/spec-iterate` is not declared.**
IC3 says: "the existing `/spec-iterate` flow is reachable from the spec stack — but it is invoked *because* the architectural spec said so, not as a standalone entrypoint." ADR-004 echoes this: "the architectural spec just declares the seam." However, the architectural spec's "What this spec deliberately does not specify" section says the spec-iterate flow internals are out of scope, *and the seam itself is not declared anywhere in S1–S8*. There is no S section instructing an architectural spec author to emit a `produces: <invokes /spec-iterate on F-section X>` declaration, no schema for that declaration, and no integrity rule checking it. Severity: **high** — an agent drafting `behavioral.md` or per-module functional specs has no guidance on how to record the seam, and may either inline `/spec-iterate` content or skip the invocation entirely. Recommend a new section S2.5 ("Seam to `/spec-iterate`") or an extension to S1.2's integrity rules.

**G-Spec-2 — Frontmatter format omits `phase:` field.**
S1.1 declares the frontmatter as `mode:` and `attested-at:`. S1.2 says "the integrity check is mode-aware about which phase the module is in, recorded in the same frontmatter." The frontmatter as declared in S1.1 has no `phase:` field. Severity: medium — easy fix in S1.1 to add `phase: 0 | 1 | 2 | 3 | 4 | 5`, but the omission is concrete and will block agent drafting.

**G-Spec-3 — Coverage table for IC9 is incomplete.**
The IC ↦ S coverage table maps `IC9 → S3.* (additive only)`. But IC9 is *also* satisfied by S1.1's "default to bootstrap" rule (which is what makes un-tagged repos behave as before). The coverage table should read `IC9 → S1.1, S3.*`. Severity: low — purely a table edit, but the table is itself a load-bearing artifact (acceptance A8 implies it will be checked mechanically).

**G-Spec-4 — CI-system / pre-commit-framework detection ownership.**
S6.1 says the pre-commit hook is "integrated with the existing pre-commit framework detected by `/assurance-init` (pre-commit.com, lefthook, or husky)" and the CI job is "added to the CI system detected by `/assurance-init` (GitHub Actions, GitLab CI, or CircleCI)." But S3.2's delta spec for `/assurance-init` says only that the "name 1-3 load-bearing modules" question is replaced when ADD-mode state is detected. *Detecting and classifying the surrounding pre-commit framework and CI system is not added as a responsibility of `/assurance-init` anywhere in the architectural spec.* This responsibility is implicitly assumed but unowned. Severity: **medium-high** — the agent drafting S6.1's enforcement gates will need this; the assumption should be explicit, either as an extension to S3.2 or as a new S section.

**G-Spec-5 — Diff-shape signal not in IC8 enumeration.**
IC8 enumerates four signals (edit-frequency, change-coupling, orphan detection, cascade-pending). S4.1 enumerates five (those four plus *diff-shape analysis*). The fifth signal is fine — it strengthens the contract — but IC8's "at least minimal" qualifier should be explicitly cross-referenced from S4.1, or the fifth signal should be added to IC8's enumeration. Severity: low.

**G-Spec-6 — Auditor verdict on the seed itself (acceptance A7) lacks a write-path.**
Acceptance A7 says the seed artifacts transition to Ratified through one consolidation pass. ADR-003 forbids the Auditor from writing to `docs/add/`. The transition-to-Ratified therefore requires a follow-up commit by Byfuglien, Hellebuyck, or a human, classified per ADR-005. The architectural spec does not specify which agent owns this commit or how the workflow is triggered. Severity: medium.

**G-Spec-7 — `/spec-adversary` mode flag is recommended but not committed.**
S2.4 says: "Recommendation: extend the existing skill with a mode flag, because adversarial probing is structurally similar across modes... The agent may overrule this if the implementation is genuinely cleaner as a mode; document the choice in a follow-up ADR." This means the architectural spec leaves the *integration shape* of S2.4 open; downstream artifacts (S3.4, the coverage table) will be consistent only after the agent decides. Severity: low — the deferred-ADR mechanism is healthy, but acknowledge that S2.4 is not fully Phase-1-decided.

---

## 4. Contradictions

Statements that cannot both be true.

**C-1 — Transitional mode: per-module tag or repo-level state?**

The methodology says: "Module boundaries carry the mode tag; governance applied to each module is mode-appropriate." The methodology lists three modes (bootstrap / ADD / transitional). A reader could plausibly conclude that any of the three values can appear as a per-module tag.

The glossary says: "transitional mode — A repo where some modules originated in bootstrap mode and others in ADD mode, with appropriate per-module governance."

ADR-001 says explicitly: "Transitional mode (`mode: transitional`). Refers to repo-level state, not module-level. A repo containing a mix of bootstrap-mode and ADD-mode modules is in transitional mode. There is no `mode: transitional` tag on any individual module."

The architectural spec S1.1 enumerates only `mode: add | bootstrap` — consistent with ADR-001.

The intent doc IC5 says "Three operating modes with per-module tagging." On a strict reading, this is wrong: only two modes are per-module-tagged.

**Resolution:** The methodology phrasing is technically correct (each module *has* a mode tag, drawn from `{add, bootstrap}`; the *repo* is in transitional mode if modules disagree) but the wording invites the wrong mental model. ADR-001 is unambiguous. Severity: medium — this is the highest-leverage clarification. Recommend amending IC5 to read "three operating modes with per-module tagging for the originative two (bootstrap, ADD); transitional is the repo-level descriptor when modules disagree." Recommend amending the methodology's "Operating modes" section's transitional paragraph to disambiguate.

**No other hard contradictions found.**

---

## 5. Under-specifications

Architectural sections vague enough to admit incompatible implementations.

**U-1 — Auditor's owned skills.** S5.1 says "Skills owned (none in v1 — the auditor calls `/add-instrumentation` and renders verdicts directly)." But S4.1 admits `/add-instrumentation` may be a script *or* a skill ("if it has substantive prompt content, it is a skill; if it is purely deterministic, it is a script"). If it's a skill, who owns it — the Auditor (contradicting S5.1) or Hellebuyck or Byfuglien? My adopted interpretation: if `/add-instrumentation` is implemented as a skill, it is **plugin-level** and *invoked* (not *owned*) by the Auditor. Hellebuyck owns it under the spec-stack chain. But the architectural spec should commit on this.

**U-2 — `/intent-check-prose` FP definition.** S2.3 says "The 30% kill criterion applies, configurable per the existing `/intent-check` env-var pattern." But the existing `/intent-check` defines a false positive in terms of the test signal (the `(invariant prose, covering test, code diff)` triple). Without a covering test, *what counts as a FP in prose-vs-prose?* My adopted interpretation: a FP is a flagged divergence the human reviewer attests is spurious (e.g., wording difference but semantic match). But the architectural spec should commit on this and document the human-feedback channel that updates the tracker.

**U-3 — Transitional-mode wording.** Tracked as C-1 above; included here because the wording-level fix is an under-specification correction.

**U-4 — Frontmatter `phase:` field.** Tracked as G-Spec-2 above.

**U-5 — Diff-classification log format.** S6.1 lists CSV columns and adds "JSON-lines is acceptable as an alternative format if the agent's implementation prefers it." ADR-005 also defers between CSV/JSONL/SQLite. The downstream consumers (Auditor's prompt template, future tooling) need a stable schema; "either is fine" delays a decision the Phase-1 spec ought to make. My adopted interpretation: commit on JSON-lines (`.assurance/diff-classification-log.jsonl`) for parser ergonomics. Architectural spec should commit, not defer.

**U-6 — `/spec-iterate` invocation declaration.** Tracked as G-Spec-1 above.

**U-7 — Mode-detection mechanism for S3 adaptations.** S3.1 ("detect empty-repo state"), S3.2 ("detect ADD-mode state — `docs/add/` already exists with an Attested intent doc"), and S3.5 ("detect empty-source-tree state") all assume reliable detection without specifying *how*. False-positive scenarios: a repo with only `.gitignore`, `LICENSE`, `README.md`. False-negative scenarios: a repo with `docs/add/` half-populated mid-Phase-1. My adopted interpretation: detection is conservative — empty-repo means literally no source files matching the layer-audit's standard manifest list; ADD-mode-state means `docs/add/intent.md` exists AND its `Status: Attested`. Architectural spec should commit on the predicate.

**U-8 — Auditor naming.** ADR-003 and S5.1 leave the Auditor's name to the agent and human. Coverage tables, README updates, and all downstream references will need the name. My adopted interpretation (placeholder only): I will refer to the agent as "the Auditor" with slug `auditor` until a name is chosen. The choice is ratified separately by attestation.

**U-9 — Squash/rebase handling for diff classification.** S6.1 does not address what happens when a feature branch is squash-merged (potentially losing per-commit trailers) or rebased (rewriting commit SHAs). My adopted interpretation: the CI gate runs on the final commit set on the merge target; squash-merging into the protected branch must be configured to preserve trailers (or to require the squashed commit to carry a trailer summarising classification). Architectural spec should commit.

**U-10 — Auditor input scaling.** S4.2 says the deterministic output is fed verbatim to the Auditor's prompt. For a large repo, the JSONL output may exceed an LLM context window. My adopted interpretation: v1 ships with a per-pass token budget and warns when exceeded; multi-shot processing is a future extension. Not blocking but worth noting.

---

## 6. Out-of-scope drift candidates

Items the back-translation thought ought to be in scope but the negative-space (`N`) list excludes. These are candidate intent-refinements the human may want to reconsider — or, equally validly, may want to leave excluded.

**D-1 — Behavioral-spec model checker integration (N4).** The methodology's open-problem #5 reads: "ADD without a behavioral-spec layer is mostly Dafny-flavoured TDD." If the behavioral-spec layer is the load-bearing differentiator and N4 excludes its model-checker integration, v1 ships a methodology that, by its own admission, is mostly Dafny-flavoured TDD. The exclusion is principled (model-checker integrations are large) but it does materially weaken the v1 claim. Suggested reconsideration: a low-fidelity behavioral-spec authoring skill (no checker, just structured prose) that *prepares* for a Phase-2-iteration model-checker integration. This is plausibly already in scope per S2 plus IC3 — the methodology speaks of behavioral specs as prose with cross-references — so the methodology may already address this. Worth a confirmation pass.

**D-2 — Empirical layer attribution (N1).** Methodology calls this "the single most credibility-enhancing artifact ADD could ship." Excluded because it requires field data. Risk: chicken-and-egg — adoption needs evidence; evidence needs adoption. Suggested reconsideration: ship a *minimal* layer-attribution scaffold in v1 (a CSV that ADD-mode users can fill in manually as they ship bug fixes; the scaffold is pre-instrumented) so that the *first* ADD-mode user generates data even without automated instrumentation. Low cost; high option value. The human may still defer, but the option is worth surfacing.

**D-3 — A `/diff-classify` interactive skill.** Excluded by ADR-004 in favor of commit-time enforcement. The exclusion is principled. *No reconsideration recommended* — surfaced here for completeness only.

**D-4 — A `/consolidation-pass` skill.** Excluded by ADR-004 in favor of the Auditor agent workflow. Same reasoning. *No reconsideration recommended.*

**D-5 — A separate ADD plugin.** Excluded by ADR-001 alternative A4. *No reconsideration recommended.*

**D-6 — Replacing Byfuglien/Hellebuyck (N7).** Excluded. *No reconsideration recommended.*

**D-7 — TiCoder-style disambiguation skill (N6).** Excluded but `/intent-elicit` does some of the work. Risk: the line between "elicitation" and "disambiguation" is thin. If `/intent-elicit` becomes the de facto disambiguation skill, it grows surface beyond what S2.1 specifies. Suggested reconsideration: clarify where elicitation ends and disambiguation begins, in S2.1's behavior contract.

---

## 7. Adversarial probing of the spec stack (Step 4 of the protocol)

Per Step 4, for each `S` section: what edge case does it not address; what failure mode does it silently assume away; what downstream artifact does it need to produce that is not declared in `produces:`. I list the **top 8 highest-leverage gaps**, ranked by how likely they are to bite the agent during downstream drafting.

**P-1 — S1.2: ADD-mode integrity for modules in *transition between phases*.** The integrity rule branches on phase; when a module is being moved Phase-3 → Phase-4 (skeleton → first implementation commit), the rule must handle the in-progress state. Not addressed. *Confidence: high — this will hit the agent on the first ADD-mode build.*

**P-2 — S2.3: FP semantics in prose-vs-prose.** As U-2 above. The 30% threshold inherits from `/intent-check` but the FP definition does not transfer cleanly. *Confidence: high — the agent will ship `/intent-check-prose` blind on this without an explicit decision.*

**P-3 — S6.1: trailer survival under rebase / squash-merge.** As U-9 above. *Confidence: medium-high — every team that uses squash-merge into protected branches will hit this.*

**P-4 — S5.1: tool-restriction enforcement mechanism.** The Auditor must demonstrably lack write access to `docs/add/`, `docs/invariants/`, `agents/`, `skills/`, `.claude/rules/`. Acceptance A5 stipulates the test ("a command that attempts a write fails with a clear error") but the *mechanism* — Claude Code allowlist? Skill-level tool restriction? File-system permissions? — is unspecified. *Confidence: medium-high.*

**P-5 — S4.1: deterministic instrumentation language and runtime.** S4.1 says "The agent picks something appropriate (likely Python or a small Go binary) and documents the choice." This is fine for a prototype but the choice has consequences (CI portability, speed, dependency footprint). Without commitment, downstream tooling that consumes the instrumentation output may end up coupling to language-specific quirks. *Confidence: medium.*

**P-6 — S2.1: human-attestation commit shape.** S2.1 says `/intent-elicit` "refuses to mark Status=Attested without explicit human confirmation in the same exchange." But the actual commit that flips Status from Drafted to Attested must be *authored by the human* per the methodology. The skill emits a commit; what is the commit's authorship? If `/intent-elicit` writes the Drafted artifact and the human's only "explicit confirmation" is in chat, no human commit exists for the attestation. Suggested fix: `/intent-elicit` emits Drafted; the human then either (a) edits the file to set `Status: Attested` and commits manually, or (b) invokes a separate `/attest` skill that is human-driven. *Confidence: medium-high — the workflow shape is genuinely ambiguous.*

**P-7 — S4.2: empty-signals case.** As U-10 plus an additional concern: if the deterministic layer finds *no* signals on a pass, the Auditor produces no verdicts. Is this state "all artifacts Settled" or "the instrumentation didn't run correctly"? The Auditor cannot distinguish, and a silent no-op is the worst kind of false-negative. *Confidence: medium — but it bites once and is hard to debug.*

**P-8 — S8: the agent's own classification of *this* Phase 2 work.** Acceptance § "The diff classification on commits creating this directory" says the initial directory-creation commit is `propagated-discovery`. The agent's Phase 2 outputs (`.assurance/phase-2-back-translation-*.md` and this comparison report) are *not* under `docs/add/`, so they do not require classification per ADR-005's protected paths. But the *Drafted → Attested* commit on intent.md / architectural.md / ADRs / acceptance.md *will* require classification. The natural class is `intent-refinement` (because attestation refines status) but ADR-005's four classes are *content-class* not *status-class*. Suggested clarification: either add a fifth `status-transition` class, or document that status-only flips use `propagated-discovery` with an explicit "no content change" justification. *Confidence: medium — bites at the first attestation commit.*

---

## 8. Verdict

### **PASS-WITH-AMENDMENTS**

The seed is substantively correct, internally coherent on the matters that matter most, and ready for human attestation *after* the small set of concrete amendments listed below has been adjudicated. Every IC is consumed by at least one S section. There are no hard contradictions. The methodology, glossary, and ADR-001 reach the same end-state on transitional mode despite phrasing variance. The architectural spec covers the operating modes, the four greenfield skills, the existing-skill adaptations, the deterministic instrumentation, the Auditor agent, the diff-classification gate, and the documentation surface — i.e., every load-bearing piece of the methodology.

The amendments below are *not* re-architecture. They are tightenings of the existing artifacts that an agent drafting downstream specs will hit immediately. Addressing them now costs less than addressing them after the cascade.

### Recommended amendments (ordered by leverage)

The agent does **not** propose to amend the artifacts itself; the recommendation is for the human, per the boundary rule that humans own intent and architecture. The agent will execute amendments only with explicit confirmation.

**A-1 (intent doc + methodology) — Disambiguate transitional mode.**
Amend IC5's headline from "three operating modes with per-module tagging" to "three operating modes; per-module tags carry the originative two (`mode: bootstrap | add`); transitional is the repo-level descriptor when modules disagree." Amend the methodology's "Operating modes" § transitional paragraph to make the same disambiguation explicit. Estimated diff: ~6 lines.

**A-2 (architectural spec, S1.1) — Add `phase:` to frontmatter.**
The integrity rule in S1.2 references a `phase:` field that S1.1 does not declare. Add `phase: 0 | 1 | 2 | 3 | 4 | 5` to the frontmatter format. Estimated diff: ~2 lines.

**A-3 (architectural spec, new section S2.5 or extension to S1.2) — Declare the `/spec-iterate` seam.**
A spec section invoking `/spec-iterate` should record the invocation in `produces:` (or a dedicated declaration) so the integrity check can verify the chain. Without this, IC3's "invoked because the spec said so" is unenforceable. Estimated diff: a new ~15-line subsection.

**A-4 (architectural spec, S3.2) — Add CI-system / pre-commit-framework detection to `/assurance-init`.**
S6.1 references this responsibility; S3.2 should explicitly add it. Estimated diff: ~5 lines added to S3.2.

**A-5 (architectural spec, S2.3 / ADR-004) — Define FP semantics in prose-vs-prose.**
Either commit on "an FP is a flagged divergence the human reviewer attests is spurious" or scope down the kill criterion to a different shape. Estimated diff: ~5 lines.

**A-6 (architectural spec, S6.1) — Address rebase/squash-merge trailer survival.**
Document the policy: squash-merging onto a protected branch requires the squashed commit to carry a trailer; CI gate runs on the final-commit set. Estimated diff: ~4 lines.

**A-7 (architectural spec, S5.1) — Specify the tool-restriction mechanism.**
Commit on whether the Auditor's read-only constraint is enforced via Claude Code allowlist, plugin-level tool restriction, or settings — and how it is verified at runtime. Estimated diff: ~5 lines.

**A-8 (intent doc, IC8 + spec S4.1) — Reconcile signal enumeration.**
Either add diff-shape analysis to IC8, or note in S4.1 that signals beyond the IC8 enumeration are extension points. Estimated diff: ~2 lines.

**A-9 (intent doc, threat-model traceability) — Backfill `consumes:` for TM1, TM3, TM5, TM6.**
The mitigations exist in the architectural spec but are not traced. The integrity check cannot verify that every TM has at least one mitigation otherwise. Estimated diff: a small set of `consumes:` line additions across S sections and ADRs (~8 lines total).

**A-10 (architectural spec, coverage table) — Add S1.1 to the IC9 row.**
Trivial. Estimated diff: 1 line.

**A-11 (acceptance.md / ADR-005) — Status-only commits and protected paths for `docs/add/audit/`.**
Two small clarifications: (a) document how a Drafted → Attested status-flip is classified under ADR-005's four classes; (b) add `docs/add/audit/` to ADR-005's protected-path list (or explicitly exempt it). Estimated diff: ~6 lines.

**A-12 (architectural spec, S5.1) — `/add-instrumentation` ownership if implemented as a skill.**
Commit on which orchestrator owns it (recommend Hellebuyck) and clarify that the Auditor *invokes* but does not *own* the skill. Estimated diff: ~3 lines.

**A-13 (architectural spec, S6.1) — Diff-classification log format.**
Pick CSV or JSON-lines (recommend JSON-lines for parser ergonomics) and stop deferring. Estimated diff: 1 line.

### Drift candidates surfaced for human consideration (not part of A-list)

- **D-1 — Behavioral-spec authoring** (low-fidelity prose-only) is plausibly already in scope; confirm.
- **D-2 — Layer-attribution scaffold** (manual CSV, no instrumentation) as a hedge against the chicken-and-egg risk for v2.

### Not recommended for amendment

- The transitional-mode resolution itself (ADR-001's repo-level decision) — sound; only the wording needs disambiguation.
- The four-class diff classification taxonomy — sound; do not expand to five classes for status flips, just clarify how status flips fit.
- The exclusion of `/diff-classify` and `/consolidation-pass` as standalone skills — sound; principled per ADR-004.
- The Auditor as a third agent role — sound; the audit-author separation argument in ADR-003 is decisive.

---

## 9. Attestation checklist (for the human to tick)

Tick each box as you adjudicate. The agent halts past Phase 2 until the **Authorisation to proceed** box at the end is ticked. Until then, the only work the agent will do is editing this file in response to your direct requests.

The amendments are split into two groups: those the agent can execute mechanically without further judgment (batch-approvable) and those that need your decision on approach before any drafting happens.

### 9.1 Agent-executable amendments (batch approval safe)

Each of these is a small, mechanical edit. The agent will execute on the next exchange after the box is ticked, classifying each as `intent-refinement` per ADR-005 (or `propagated-discovery` if that fits better — agent's call at commit time).

- [x] **A-2** — Add `phase: 0 | 1 | 2 | 3 | 4 | 5` to the S1.1 frontmatter format. ~2 lines in `specs/architectural.md`.
- [x] **A-8** — Add diff-shape analysis to IC8's signal enumeration so it matches S4.1. ~2 lines in `intent.md`.
- [x] **A-10** — Add `S1.1` to the IC9 row of the IC ↦ S coverage table. 1 line in `specs/architectural.md`.
- [x] **A-12** — Clarify in S5.1 that `/add-instrumentation`, if implemented as a skill, is plugin-level and owned by Hellebuyck; the Auditor invokes but does not own it. ~3 lines in `specs/architectural.md`.
- [x] **A-13** — Commit on JSON-lines (`.assurance/diff-classification-log.jsonl`) for the diff-classification log; remove the format alternation in S6.1 and ADR-005. ~2 lines total.

**Batch shortcut:** [x] Approve all five A-* above as a single batch (saves you ticking each one).

### 9.2 Judgment-required amendments (please pick an approach)

Each amendment touches substantive policy or human-authored prose. Tick the option that matches your decision; the agent drafts the change accordingly. If none of the listed options fits, tick "Other" and write a brief note.

- [x] **A-1 — Disambiguate transitional mode.**
  - [x] (a) Amend IC5 wording *and* the methodology's "Operating modes" transitional paragraph as recommended in § 4 / amendment list (joint edit).
  - [ ] (b) Amend the methodology paragraph only; leave IC5 as-is.
  - [ ] (c) Amend IC5 only; leave methodology as-is.
  - [ ] (d) Other: _________________________________

- [x] **A-3 — Declare the `/spec-iterate` seam.**
  - [x] (a) Add a new section S2.5 "Seam to `/spec-iterate`" to the architectural spec.
  - [ ] (b) Extend S1.2's integrity rules instead.
  - [ ] (c) Defer to behavioral spec; the agent declares the seam there.
  - [ ] (d) Other: _________________________________

- [x] **A-4 — Where does pre-commit framework / CI system detection live?**
  - [x] (a) Extend S3.2's `/assurance-init` delta with detection responsibility.
  - [ ] (b) Inline detection in the S6.1 pre-commit hook + CI job stubs themselves (skill-free).
  - [ ] (c) New skill `/governance-detect`, separate from `/assurance-init`.
  - [ ] (d) Other: _________________________________

- [x] **A-5 — FP semantics in `/intent-check-prose`.**
  - [x] (a) Commit on: "FP = a flagged divergence the human reviewer attests is spurious." Reuse the 30% rolling threshold.
  - [ ] (b) Replace the 30% kill criterion with a different shape: _________________________________
  - [ ] (c) Drop the kill criterion for v1 of the prose variant; revisit after field data.
  - [ ] (d) Other: _________________________________

- [x] **A-6 — Diff-classification trailer survival under rebase / squash-merge.**
  - [x] (a) Squash commits to protected branches must carry a summary trailer; CI runs on the final commit only.
  - [ ] (b) Disallow squash-merge to protected branches; require fast-forward or merge commit.
  - [ ] (c) CI walks the pre-squash commit set and validates each.
  - [ ] (d) Other: _________________________________

- [x] **A-7 — Auditor tool-restriction enforcement mechanism.**
  - [ ] (a) Claude Code permission allowlist that denies write tools on protected paths.
  - [x] (b) Plugin-level tool restriction declared in the Auditor's `agents/<auditor>.md` frontmatter.
  - [ ] (c) Filesystem read-only mount on the protected paths during Auditor runs.
  - [ ] (d) Other: _________________________________

- [x] **A-9 — Backfill `consumes:` for TM1, TM3, TM5, TM6.**
  - [ ] (a) Map each unconsumed TM to the existing S section or ADR that mitigates it; add `consumes: TMx` to that section.
  - [x] (b) Defer until the linkage-graph integrity check is implemented and can flag missing TM coverage automatically.
  - [ ] (c) Other: _________________________________

- [x] **A-11a — Classification of status-only commits (Drafted → Attested flips).**
  - [ ] (a) Status flips use `propagated-discovery` with a "status transition; no content change" justification.
  - [x] (b) Add a fifth class `status-transition` to ADR-005's taxonomy.
  - [ ] (c) Other: _________________________________

- [x] **A-11b — Protected-path status of `docs/add/audit/` (Auditor reports).**
  - [x] (a) Add `docs/add/audit/` to ADR-005's protected paths list. (only auditor agent must be able to write reports)
  - [ ] (b) Explicitly exempt; the path is agent-write-append-only with its own write-rule.
  - [ ] (c) Other: _________________________________

### 9.3 Drift candidates (optional reconsideration)

These are negative-space items the agent thought worth surfacing. The default is "leave excluded" — you only need to tick if you want to reconsider.

- [x] **D-1 — Behavioral-spec prose authoring quality** (already nominally in scope but not load-bearing).
  - [ ] (a) Confirm in scope as currently structured; no change.
  - [x] (b) Add an explicit IC requiring behavioral-spec prose to satisfy a quality bar (e.g., every `B` traces to at least one `IC` and at least one `F`).
  - [ ] (c) Other: _________________________________
- [x] **D-2 — Layer-attribution scaffold** (manual CSV, no automation; v1 hedge against the chicken-and-egg).
  - [x] (a) Hold N1 as-is; defer until field data exists.
  - [ ] (b) Reconsider — ship a manual scaffold so the first ADD-mode user begins generating field data.
  - [ ] (c) Other: _________________________________

### 9.4 Seed artifact attestation (Drafted → Attested)

Tick once amendments above are adjudicated and the artifact reflects your intent. Attestation is monotonic — once ticked, the agent treats the artifact as Attested and will modify it only via a supersession ADR (per `glossary.md` § Status field). `methodology.md` and `glossary.md` are already Ratified; no tick required here.

- [x] `crosscheck/docs/add/intent.md` — Drafted → Attested
- [x] `crosscheck/docs/add/decisions/INDEX.md` — Drafted → Attested
- [x] `crosscheck/docs/add/decisions/ADR-001-operating-modes.md` — Drafted → Attested
- [x] `crosscheck/docs/add/decisions/ADR-002-deterministic-llm-split.md` — Drafted → Attested
- [x] `crosscheck/docs/add/decisions/ADR-003-auditor-agent.md` — Drafted → Attested
- [x] `crosscheck/docs/add/decisions/ADR-004-greenfield-skill-set.md` — Drafted → Attested
- [x] `crosscheck/docs/add/decisions/ADR-005-diff-classification.md` — Drafted → Attested
- [x] `crosscheck/docs/add/specs/architectural.md` — Drafted → Attested
- [x] `crosscheck/docs/add/acceptance.md` — Drafted → Attested
- [x] `crosscheck/docs/add/README.md` — Drafted → Attested

### 9.5 Authorisation to proceed

- [x] **Proceed.** Amendments above are adjudicated; seed artifacts above are attested (or scoped down with reasons). The agent is authorised to (a) execute the approved A-* amendments and (b) begin Phase 1 lower-tier drafting (`specs/behavioral.md` and per-module functional specs) per S8 of the architectural spec.

---

## 10. Stop point

Per the protocol § Step 6, the agent halts here. It will not draft `specs/behavioral.md`, any per-module functional spec, any new SKILL.md, any agent definition, or any tool until the human:

1. Reads this comparison report.
2. Either attests Drafted → Attested without amendment, or amends the seed and re-attests.
3. Explicitly authorises the agent to proceed.

Phase 2 attestation is the first hard gate in the methodology being applied to itself. Awaiting human attestation.
