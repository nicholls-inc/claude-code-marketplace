# Specula adoption: informal specification

Status: Draft

Scope: five candidate mechanisms lifted from Specula (arXiv:2607.25333v2) and assessed against Crosscheck as it exists in this repository. Prose only. No Lean, no Dafny, no TLA+, no test code, no skill edits. The document stops at the sign-off gate.

Discipline followed: `crosscheck/skills/informal-spec/SKILL.md` (externalised contract before implementation, quantified statements only, ambiguities as direct questions with no defaults, hard stop at sign-off) and `crosscheck/skills/draft-invariants/SKILL.md` (spec before code, every claim anchored in a real failure, `file:lines` citation for every claim about current behaviour).

## Sources read

### The paper

Cheng, Pial, Tang, Su, Ma, Hackett, Beschastnikh, Huang, Xu. *Specula: Scaling formal specifications for autonomous model checking of system code.* arXiv:2607.25333v2, 3 Aug 2026. Read in full, including references. Sections drawn on most heavily: §2.2, §3.1, §3.2, §3.3.1, §3.3.2, §3.4.1, §3.5.1, §3.5.2, §4, §5.1, §5.1.1, §5.1.2, §5.2, §5.3, §5.4, §5.5, §6, §7. Figures 1, 4, 5, 6, 7, 9, 10, 11 and Tables 1, 2, 3, 4, 5.

### Crosscheck files read, in the order the task prescribed

| Path | Lines | Note |
|---|---|---|
| `crosscheck/docs/research/assurance-hierarchy.md` | 133 | Includes the four VGD prerequisites at line 15. |
| `crosscheck/skills/assurance-layer-audit/SKILL.md` | 314 | |
| `crosscheck/skills/assurance-probe/SKILL.md` | 308 | |
| `crosscheck/skills/intent-check/SKILL.md` | 247 | |
| `crosscheck/skills/correspondence-review/SKILL.md` | 216 | |
| `crosscheck/skills/drt-oracle/SKILL.md` | 272 | |
| `crosscheck/skills/informal-spec/SKILL.md` | 223 | |
| `crosscheck/skills/draft-invariants/SKILL.md` | 515 | |

### Files the task named that do not exist in this repository

Two of the four prescribed reading-order paths are absent. I state that rather than infer their contents.

**`crosscheck/docs/assurance/ROADMAP.md` and the horizon directories beneath it do not exist.** `find . -type d -name assurance` returns nothing anywhere in the repository. `docs/assurance/` is an artefact that `/assurance-init` *scaffolds in an adopting repository*, not an artefact this repository keeps for itself (`crosscheck/skills/assurance-init/SKILL.md:100`, `:122`). Read instead:

- `crosscheck/skills/assurance-init/SKILL.md:100-131` for the ROADMAP section list, the Dual-Track Enforcement Principle, the four horizon tables and the Kill Criteria section.
- `crosscheck/skills/assurance-init/SKILL.md:122-151` for the four horizon directories and their scoping language.
- `crosscheck/skills/assurance-init/SKILL.md:142-148` for the `Status:` vocabulary (`Not started`, `In progress`, `Blocked`, `Done`, `Deferred` with a required `Reason:`), which the Sequencing section below uses verbatim.
- `crosscheck/skills/assurance-roadmap-check/SKILL.md:52-106` for how the roadmap is audited for drift.
- `crosscheck/docs/research/phase-roadmap-may-2026.md` for the delivery roadmap this repository actually maintains, which uses a different Status vocabulary (`Done`, `Done locally; awaiting push`, `Deferred`) in a phase-index table at lines 9-23.

**`crosscheck/.claude/rules/protected-surfaces.md` does not exist.** `find . -type d -name rules` returns nothing. The only file under `crosscheck/.claude/` is `settings.local.json`. Thirty-three references across the skill catalogue point at the path, so the absence is a scaffolding gap rather than a design choice. Read instead:

- `crosscheck/skills/assurance-init/SKILL.md:152-201` for the verbatim two-class partition template that creates the file, including the Class A list at `:163-176` and the Class B list at `:177-186`.
- `crosscheck/skills/protected-surface-amend/SKILL.md` in full for the consumer, notably the hard refusal when the rule file is missing (`:48-49`) and the hard refusal when no governing roadmap item exists (`:113-116`).

### Terms in the task that I could not locate in the repository

I searched case-insensitively across all Markdown for `pre-regist`, `preregistration`, `eval mandate`, `oracle independence`, `independent oracle`, `validity invariant` and `freeze`. **No document in this repository names a "Crosscheck eval mandate", a "pre-registration freeze", or a "protected validity invariant".** The nearest documented analogues, which I use in their place and name as such throughout:

- `self-improve/SKILL.md:15-17` and `:47-52`: "propose, measure against a pre-registered metric, keep only what improves it", with a required `falsifier` field and the rule "no measurement, no suggestion". This is the closest thing in the repository to a pre-registration discipline, and it governs the self-improvement controller rather than the assurance hierarchy.
- `crosscheck/skills/intent-check/SKILL.md:30-40` and `:46-60`: three threshold environment variables, a documented default of 30% over 14 days with `n >= 3`, and an explicit disclosure that the numbers are "founder intuition, not labelled-pilot data".
- `crosscheck/skills/assurance-probe/SKILL.md:60` and `:231-234`: an SNR kill criterion and two phase gates that must be met from tracker data before a later probe phase runs.
- `crosscheck/docs/research/assurance-hierarchy.md:93-103`: the calibration section that admits no calibration trace exists.

The governance question the task asks about the freeze is answered against these analogues in M2, and the missing document is raised as ambiguity A1.

### Supporting files read to state a gap precisely

`crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md`; `crosscheck/docs/research/crosscheck-tla-vgd-addendum.md`; `crosscheck/docs/assurance-hierarchy.md`; `crosscheck/skills/assurance-init/SKILL.md`; `crosscheck/skills/protected-surface-amend/SKILL.md`; `crosscheck/skills/assurance-roadmap-check/SKILL.md`; `crosscheck/skills/lean-impl/SKILL.md`; `CLAUDE.md`; `AGENTS.md`; `JOURNAL.md`; `crosscheck/JOURNAL.md`; the `formal-verification/` tree (eight files, the shipped `power` smoke case).

### Why this path

The task preferred `crosscheck/docs/research/specula-adoption_informal.md` and I kept it. Two conventions collide at that path and the choice is deliberate. Peer documents in `crosscheck/docs/research/` use plain kebab-case with no suffix (`crosscheck-tla-vgd-addendum.md`, `literature-review.md`, `assurance-hierarchy.md`), and the `_informal.md` suffix belongs to `/informal-spec`'s output directory, `formal-verification/specs/<module>_informal.md` (`crosscheck/skills/informal-spec/SKILL.md:123`, and the one shipped instance at `formal-verification/specs/power_informal.md`). This document is an informal specification in that skill's sense rather than a research report, so the suffix carries information worth keeping; `docs/research/` is the right directory because the subject is adoption of external work rather than a module contract. It is not an ADR: it records no decision, so `crosscheck/docs/research/adr/` and the repo-root `docs/decisions/` are both wrong for it.

## Provenance labelling used throughout

Every number quoted from the paper carries one of three labels.

**(A) Author-reported and self-benchmarked.** The quality benchmark in Table 4 is SysMoBench, reference [12]. Comparing author lists: Specula has nine authors, SysMoBench [12] has ten (Cheng, Tang, Ma E., Hackett, He, Su, Beschastnikh, Huang, Ma X., Xu). **Eight of Specula's nine authors are also authors on the benchmark it scores 100% on.** The task's draft said seven of nine; the correct count is eight of nine, the exception being Pial. Specula scores 100% on all four SysMoBench metrics and 100% overall (Table 4).

**(B) Author-reported with an internal check.** "Specula reports no false positive as all the bugs are reproduced at the code level" (§5.1). §5.1.1 gives the mechanics: for the 14 systems checked with the latest version, Specula reported 136 bugs, reproduced 134 at code level and encoded them in tests, and reproduced the other two without observing severe consequences, then counted those two as masked bugs. The reproduction is performed by the same system that found the violation, so the check is internal rather than independent.

**(C) Independently confirmed.** Of 249 bugs found, the authors reported 89; 68 have been confirmed by developers and 24 fixed (§5.1). The independently confirmed figure is 68, which is 27% of the 249 the paper counts.

**The baseline comparison in §5.2 is confounded by compute.** All three approaches used the same prompts, but Table 5 shows Specula running 110 to 402 minutes per system against 11 to 40 for Agent-Raw and 13 to 69 for Agent-TLA+, at 4.8 to 37 times the cost of Agent-Raw and 1.8 to 65 times the cost of Agent-TLA+. Part of the 62-versus-3 bug gap in Figure 11 is budget rather than method, and the paper does not run an equal-budget arm. Every kill criterion below is set against what the paper licenses after that discount, not against the headline gap.

## Current Crosscheck state

### Correspondence checking between a model and source

`/correspondence-review` is the only skill that establishes model-to-source correspondence, and it does so by reading and judging. It pairs each Lean `def` with a source range using the `-- src:` comment that `/lean-impl` leaves (`crosscheck/skills/correspondence-review/SKILL.md:73-77`), then walks a four-step rubric to classify the pair as `exact`, `abstraction`, `approximation` or `mismatch` (`:83-99`). The classification is explicitly a judgement under an anti-optimism rule: "if you are uncertain whether a definition is `exact` or `abstraction`, classify it `abstraction` and document the uncertainty" (`:51`). Divergences carry `file:line` citations (`:20`, `:141`) and each classification must name the rubric step that drove it (`:100`), but no step executes anything. The output gates the next skill: `mismatch > 0` blocks `/drt-oracle` (`:170-181`, and the mirror gate at `crosscheck/skills/drt-oracle/SKILL.md:95-97`), and `approximation` regions are skipped (`crosscheck/skills/drt-oracle/SKILL.md:112`).

The only mechanical check in the pipeline runs downstream. `/drt-oracle` generates random inputs, drives a Lean runner and a production adapter on each, and compares outputs (`crosscheck/skills/drt-oracle/SKILL.md:146-168`), defaulting to 1000 inputs per definition (`:152`). It classifies each divergence into four classes, one of which, `correspondence error`, feeds back to `/correspondence-review` (`:197-201`, `:221-227`). Random inputs are the only source of test cases; nothing in the repository records or replays production executions. The shipped smoke case confirms the shape: `formal-verification/tests/power/` drives a Python SUT against a reference oracle with a seeded generator.

### Oracle independence

The Lean pipeline constructs a second opinion by authoring a model in a different language from the production code, then testing production against it. The independence rests on two things, neither of them mechanical. First, human sign-off: `/informal-spec` stops hard and requires a literal `Human sign-off: <YYYY-MM-DD>` marker before `/lean-spec` may run (`crosscheck/skills/informal-spec/SKILL.md:173-191`), and the marker regex gates the next skill. Second, cognitive separation between skills: `/lean-impl` writes the model and `/correspondence-review` audits it, and the skill states the reason for the split as "impl mode trusts the source and chases `lake build` clean; review mode distrusts the model and chases divergences" (`crosscheck/skills/correspondence-review/SKILL.md:45`).

`/intent-check` constructs a different kind of independence, structural rather than authorial: the back-translator sees only `{code, test}` and never the invariant prose (`crosscheck/skills/intent-check/SKILL.md:74-76`), and a second model compares the back-translation against the prose (`:85-104`). Two defensive rules catch model self-contradiction independently of the prompt (`:106-120`).

No skill in the repository requires that two artefacts be authored from disjoint evidence, and no skill freezes an artefact against later revision once a check has run against it. The nearest thing to a freeze is the content hash in `.assurance/intent-check-attestation.json`, which detects that a protected file changed after the check but does not forbid the change (`crosscheck/skills/intent-check/SKILL.md:137-162`).

### Probes

`/assurance-probe` runs three probes over property-based tests: a mutation probe that parses the `Failure condition` clause of an invariant doc and generates one to three targeted source mutations, a vacuity probe that deletes the covering test and measures branch-coverage delta, and a generator probe that reports failure regions the Hypothesis strategy cannot reach (`crosscheck/skills/assurance-probe/SKILL.md:24-38`). All three probe *tests against source*. None probes a specification for internal well-formedness. Phase 1 is Python-only (`:298`), handles simple predicates only (`:300`), and gates Phase 2 on a Phase-1 SNR of at least 1:3 over 20 runs (`:232-234`). The kill criterion retires the probe for a module at SNR below 1:5 over four weeks with a 20-run minimum (`:60`).

The word "vacuity" in this skill means "the test is not load-bearing for branch coverage" (`:35-38`). It does not mean "the specification is trivially satisfiable", which is the sense Specula's static analyser addresses.

### False-positive tracking

`/intent-check` appends exactly one row per run to `.assurance/intent-check-fp-tracker.csv` with four fixed columns: `date`, `invariant_touched`, `phase_verdict`, `human_verdict` (`crosscheck/skills/intent-check/SKILL.md:122-135`). The schema is frozen: "do not add columns and do not rename columns. Stability across repos is what makes the rows directly concatenatable for cross-repo calibration analysis" (`:135`). `human_verdict` is left empty at run time and a reviewer later fills it with `genuine`, `genuine-planted`, `partial` or `spurious` (`:133`). The rolling rate counts `spurious` over classified rows in the window and refuses to run above the threshold (`:52-60`).

The row records *that* a verdict was spurious. It does not record *why*, and it does not record which artefact was wrong. The `mismatch_category` field the diff-checker returns (`:100`) is the closest available signal, and it is written into the attestation (`:150-157`) but not into the tracker. `/assurance-probe`'s tracker has a different shape, `date,module,proposed,accepted,rejected,deferred,skipped` (`crosscheck/skills/assurance-probe/SKILL.md:178-180`), which records triage disposition rather than cause. Neither tracker can answer "which stage of the pipeline produced this error".

### Layer routing

`/assurance-layer-audit` routes on three keys, in this order. Language, detected from ecosystem manifests against a nine-row table (`crosscheck/skills/assurance-layer-audit/SKILL.md:32-50`). Ecosystem tooling, applied as per-layer rules that are keyed on language throughout: Layer 1 has one bullet each for Go, Python, TypeScript, Rust, Ruby/Java/Elixir/C# and Haskell (`:95-101`), and Layer 2 has one bullet each for Rust, Go, Python/Ruby/JavaScript, Java/C#/Kotlin and C/C++ (`:103-109`). Then per-module VGD prerequisites, at Step 4.5, which assesses two to four modules against the four prerequisites named at `crosscheck/docs/research/assurance-hierarchy.md:15` and emits a recommended engine combination (`:128-171`). Step 6's prioritisation heuristics end with "per-module routing wins over per-layer routing" (`:223`).

A code-shape criterion already exists in the repository, but not in the skill. ADR-0001 defines a five-clause routing heuristic keyed on shape rather than language: state machine, branching workflow with rollback, rule-interaction surface, invariant-rich data model, concurrency or distribution model (`crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:44-52`), with the explicit note that "the routing criterion is rule-density / state-explosion, not 'non-distributed code generally'" (`:54`). The ADR is Accepted as a decision with implementation deferred (`:3`), and the detection logic is listed as deferred work item 2, to land with a `/behavioral-spec-init` skill that does not exist (`:70-77`). No verifier is shipped for that row: the tooling stance is orchestrate rather than embed (`:56-60`) and the first model checker has not been chosen (`:74`).

The integration-boundary row has no verifier at any layer. Layer 3 targets "approximately 75% of code surface area that sits at integration boundaries" (`crosscheck/docs/research/assurance-hierarchy.md:43`), and both the research doc and the audit skill say end-to-end subgraph verification is aspirational with no general-purpose tool (`crosscheck/docs/research/assurance-hierarchy.md:85`; `crosscheck/skills/assurance-layer-audit/SKILL.md:111-115`).

---

## M1. Trace validation

### Mechanism as implemented in Specula

Specula runs trace validation in three steps (§3.3.1). An agent emits an instrumentation plan that, for each TLA+ action, names the action's origin in code, a triggering point and the state variables to capture. A second pass inserts calls to a trace library at those origins, recording an action name and the captured state per invocation. A third pass generates a TLA+ replay harness in which every model action is wrapped to match the incoming event, execute the code-level action and check that the post-state agrees with the recorded snapshot; TLC then advances one event at a time until the trace is exhausted. Failure means the model does not conform to the code and must be repaired. The tooling around this is substantial: a per-language trace library, a mutex-based recorder for distributed systems and a timebox-based recorder for concurrent ones, a trace inspector, and a debugger with breakpoints and expression evaluation, written because "raw trace validation reports only the depth at which TLC stops, not the failure state" and the true divergence may originate in an auxiliary variable that drifted many actions earlier (§4).

### Gap in Crosscheck

The task's draft gap statement is correct that `/correspondence-review` is an LLM judgement and that nothing mechanical establishes model-to-source correspondence (`crosscheck/skills/correspondence-review/SKILL.md:51`, `:83-99`, `:100`). It is wrong in one respect that changes the shape of the proposal.

Specula's trace validation replays a *behavioural* model: a set of state variables, an initial predicate and a next-state relation expressed as atomic actions (§2.1). Crosscheck has no behavioural model. `/lean-impl` produces a *functional* model, a set of Lean `def`s each carrying a `-- src:` range (`crosscheck/skills/lean-impl/SKILL.md:137`), and ADR-0001 records that Crosscheck has no behavioural stratum at all, with the decision to add one at Layer 4 taken but unimplemented (`crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:3`, `:70-77`). Replaying a state-transition trace against a Lean `def` is not defined, because a `def` has no post-state to compare a snapshot against.

What *is* defined against a Lean `def`, and what Crosscheck currently lacks, is a recorded call. `/drt-oracle` already builds both arms of the comparison: a Lean runner that reads one input on stdin and prints a result (`crosscheck/skills/drt-oracle/SKILL.md:118-131`) and a SUT adapter mirroring that shape (`:133-144`). Both are driven exclusively by a seeded generator (`:152`). Nothing records the inputs production actually saw. The task's claim that trace validation "reaches code that neither the Lean pipeline nor the Dafny pipeline covers" is therefore right about the *destination* and wrong about the *route*: the reach comes from using real executions instead of sampled ones, not from replaying state transitions.

The gap also lands squarely on `/drt-oracle`'s own admitted weakness. Prerequisite 3 says that if the input space cannot be sampled well, "DRT will hit shallow surfaces only" (`crosscheck/skills/drt-oracle/SKILL.md:58`), and the skill's honesty clause says "*not finding* a witness is not a soundness guarantee" (`:77`). Recorded production inputs are the one input source that does not depend on a generator being good.

### Proposed change

Split the candidate and adopt only the half that fits.

**M1a, adopt: recorded-input replay as a mechanical arm of correspondence classification.** Add a recorded-input mode to `/drt-oracle` and a mechanical arm to `/correspondence-review`. On the production side, an instrumentation step records `(input, output)` pairs at the call sites of the functions the informal spec's Module boundary names, using the same source ranges `/lean-impl` already wrote as `-- src:` comments. On the model side, the existing Lean runner consumes those recorded inputs instead of generated ones. A recorded pair on which the Lean model and the recorded output disagree is a mechanical divergence at a named `-- src:` range, and it either contradicts an `exact` classification or falls inside a documented `abstraction`. `/correspondence-review` gains a step that consumes this evidence before it classifies, so classifications made against contradicting evidence are impossible rather than merely discouraged. Files affected: `crosscheck/skills/drt-oracle/SKILL.md` (a sixth D4-adjacent input mode, plus report fields for recorded-input provenance), `crosscheck/skills/correspondence-review/SKILL.md` (a new pre-classification step and a new refusal), and `crosscheck/skills/lean-impl/SKILL.md` (the `-- src:` comment becomes load-bearing for instrumentation as well as for review).

**M1b, do not adopt yet: trace validation of a behavioural model.** Full trace validation requires the artefact ADR-0001 defers. It cannot be specified before the behavioural-spec layer exists, the first model checker is chosen (`crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:74`) and the orchestrate-not-embed stance is exercised once (`:56-60`). Raised as A4.

### Invariants

**I1.** For every Lean `def` that `/correspondence-review` classifies `exact`, there exists at least one recorded `(input, output)` pair on which the Lean runner reproduces the recorded output, or the classification is downgraded to `abstraction` with the absence of evidence stated in the definition's entry. *Anchor:* Specula §3.3.2 documents the failure this prevents in the opposite direction. Agents "tend to overfit the model to match traces by weakening guards, adding permissive transitions, or even hardcoding trace-specific state updates", and the paper's Kudu-Raft case (Figure 6) shows an unconditional log-suffix overwrite that passed replay. Crosscheck's exposure is the mirror image: a classification asserted with no execution behind it at all.

**I2.** For every recorded pair on which the model and the recorded output disagree, the divergence report names the `-- src:` range, the recording site, the Lean identifier and the classification the disagreement contradicts. *Anchor:* Specula §4 states that raw trace validation reports only the depth at which TLC stops, not the failure state, and that the true divergence may originate in an auxiliary variable that drifted many actions earlier; the authors wrote a debugger to recover the divergence point. A replay arm that reports only "a mismatch occurred" reproduces the defect the paper had to build a tool to fix.

**I3.** No recorded input is admitted into the replay set unless the informal spec's Module boundary admits it (`crosscheck/skills/informal-spec/SKILL.md:79`). *Anchor:* `/correspondence-review` classifies `mismatch` on "behavioural disagreement on at least one in-spec input" (`crosscheck/skills/correspondence-review/SKILL.md:93`); an out-of-spec recorded input would trip that rule against a model that is correct on its stated domain.

**I4.** Recorded-input replay never widens the set of regions `/drt-oracle` will test: for every region classified `approximation` or `mismatch`, the recorded-input arm is skipped and the skip is reported. *Anchor:* the existing mismatch gate (`crosscheck/skills/drt-oracle/SKILL.md:95-97`) and skip-list construction (`:112`), which exist because "DRT against either is wasted compute and noisy signal" (`:65`).

### Kill criterion

Retire recorded-input replay when, over a rolling eight-week window containing at least 20 replay runs across at least three modules, fewer than one in five replay-detected divergences survives human triage as either a correspondence misclassification or a production bug. That is a signal-to-noise ratio below 1:4, one notch stricter than `/assurance-probe`'s 1:5 retirement line (`crosscheck/skills/assurance-probe/SKILL.md:60`) because replay divergences are mechanical and should be cheaper to adjudicate than surviving mutations. Review at the weekly `/assurance-status` cadence (`crosscheck/docs/assurance-hierarchy.md:24`), retire at the second consecutive breaching window.

Second arm, on cost rather than signal: retire if the median added wall-clock per module across those 20 runs exceeds the median cost of the generated-input arm by more than a factor of three. Specula's own accounting says "most of the cost goes toward TLA+ model generation and code instrumentation" (§5.3, provenance A), so instrumentation cost is the documented risk, and there is no published figure to set the multiple from. The factor of three is a stated assumption, not a measured threshold, and it is raised as A5.

### What this does NOT catch

- **Behaviour reachable only under interleaving, partial failure or message reordering.** A recorded call carries an input and an output, not a schedule. This is exactly the class Specula's model checking reaches (§5.1: deadlocks, split-brain, stalled failover) and it belongs to the deferred behavioural stratum at Layer 4 per ADR-0001, not here.
- **Inputs production has never produced.** Replay is bounded by what the recorder saw. Generated-input DRT stays the owner of the unobserved surface, at Layer 1 per `crosscheck/docs/research/assurance-hierarchy.md:27`.
- **A model that is wrong in the same direction as the production code.** Replay compares the model against production, so an error both share passes. Layer 5 `/intent-check` owns spec-versus-intent, and Layer 6 `/spec-adversary` owns spec completeness.
- **Whether the recorded output was correct.** Replay establishes agreement, not correctness. Layer 5 and Layer 6 own that.
- **Anything on a region classified `approximation`.** Per I4 the arm is skipped there; the region remains uncovered and the omission is Layer 1's, reported rather than fixed.

### Cost

Instrumentation is the documented expense. Specula corrected code instrumentation on 33 of its 48 systems and repaired instrumentation errors within three rounds (§5.4, provenance A), and instrumentation accounted for 22.2% of the mistakes the conformance loop caught (§5.4, provenance A). The trace library is per-language, exposing "per-language emit functions" (§4), with two distinct recording mechanisms, mutex-based for distributed systems and timebox-based for concurrent ones (§4). Specula's end-to-end cost of 1.43 to 9.86 hours and $19 to $168 per system, median 3.69 hours and $57 (§5.3, provenance A), does not transfer: those figures cover invariant generation, reference and scenario model generation, model checking and bug reproduction, and the M1a proposal contains none of that. The only figure that transfers as a bound is the 22.2% instrumentation share, and it transfers as a warning that a fifth of the loop's corrective work lands on the recording layer.

Crosscheck-specific cost, unmeasured: one recorder per production language, against `/drt-oracle`'s existing adapter surface, which already accepts Python, Go, TypeScript and Rust source paths (`crosscheck/skills/drt-oracle/SKILL.md:103`). Note that Specula's seven languages are C, C#, C++, Erlang, Go, Java and Rust (§5), and Python is not among them, so the paper carries no evidence for the language Crosscheck's Dafny extraction targets first (`crosscheck/skills/assurance-layer-audit/SKILL.md:97`).

---

## M2. Bidirectional validation

### Mechanism as implemented in Specula

Trace validation checks that code-level actions can occur in the model but does not explore whether the model also admits illegal actions, so an agent can pass it by weakening guards (§3.3.2). Specula therefore model-checks each repaired model against protocol-level invariants, and Figure 6 shows the case: in Kudu-Raft the agent repaired the follower accept path by overwriting the log suffix unconditionally, which replayed cleanly and then let a delayed append-entries message truncate a committed entry, violating State Machine Safety. The conformance loop runs both arms until "the generated TLA+ models permit all code-level traces and satisfy protocol-level invariants (or it finds a code bug)" (§3.5.1). §6 names this the durable core: "trace validation confirms the model admits code-level executions, and model checking exposes states the model incorrectly allows".

### Gap in Crosscheck

The mechanism is real and the pairing argument is sound. **The task's claim about what the pairing buys is wrong, and I reject it.**

The claim is that pairing is "a cheaper construction of the oracle independence the Crosscheck eval mandate requires, because the second oracle is a checker rather than an independently authored model". Three problems.

First, there is no locatable eval mandate to satisfy (see Sources read), so there is no stated independence requirement for the pairing to construct more cheaply.

Second, and substantively: TLC is a checker, but what it checks against is not. Specula's protocol-level invariants are written by the same agent from the same artefacts as the model. §3.1 has the agent summarising both protocol-level and code-level invariants from code, comments, documents, tests, issues and revision history, and Table 2 shows the evidence base overlapping heavily across both, with implementation code and comments cited for 87.35% of invariants. §3.3.2 states plainly that Specula "does not assume AI-generated invariants and repaired models are always correct". The second arm is therefore a second *agent-authored artefact* mechanically compared against the first. That is contradiction detection between two outputs of one author, not oracle independence.

Third, the paper closes the loop that would have made the pairing independent. §3.5.2 permits revising the invariants when the agent judges the original wrong, gated on the agent supplying strong evidence and on an LLM judging that evidence strong. Once the same agent can move the second arm, the first arm no longer constrains it. The authors' own safeguard is a hope about dynamics rather than a mechanism: "if agents make mistakes on invariants, the process will not converge but continue to iterate" (§3.5.1), and convergence rests on "the assumption that agents improve over the iterations" (§3.5.2). The one case that needed four revisions, SONiC's link manager, is the paper's own illustration that the invariant arm moves (§5.4).

The Crosscheck gap that *is* real, and narrower than the draft claims: nothing in the Lean pipeline rejects a model for being too permissive. `/correspondence-review` finds behaviour the model has that the source lacks and classifies it `mismatch` (`crosscheck/skills/correspondence-review/SKILL.md:79`), but that is a judgement by reading, and `/drt-oracle` compares outputs on sampled inputs (`crosscheck/skills/drt-oracle/SKILL.md:146-168`) without ever asking whether the model satisfies its own stated properties. The theorems `/lean-spec` states carry `sorry` bodies and `/correspondence-review` only assesses their impact, explicitly declining to discharge them (`crosscheck/skills/correspondence-review/SKILL.md:102-112`). So the properties are named and never checked, in either direction.

### Proposed change

Adopt the pairing; reject the independence claim; refuse the revision behaviour.

**Adopt.** Add a second arm to the Lean pipeline that is adversarial to permissiveness, paired with M1a's replay arm. Concretely, `/drt-oracle` gains a property arm: alongside comparing model output against production output on each input, it evaluates the informal spec's numbered invariants (`crosscheck/skills/informal-spec/SKILL.md:145-148`) against the model's own output, and reports a model that satisfies replay while violating a stated invariant as a distinct finding class. The report gains a fifth divergence class beside the existing four (`crosscheck/skills/drt-oracle/SKILL.md:197-201`): *permissive model*, meaning the model reproduced production behaviour and violated a signed-off invariant. Files affected: `crosscheck/skills/drt-oracle/SKILL.md` and `crosscheck/skills/correspondence-review/SKILL.md`.

**Refuse the revision behaviour.** Specula's §3.5.2 invariant revision is not adoptable into Crosscheck as it stands, for a reason internal to Crosscheck rather than to the paper. The invariants live in an artefact carrying a human sign-off marker whose format is load-bearing and regex-gated (`crosscheck/skills/informal-spec/SKILL.md:179-185`), and the sign-off prompt requires the human to confirm that "every ambiguity in the Ambiguities section has been resolved" (`:177`). An agent that revises a signed-off invariant mid-run invalidates the marker while leaving it in place, which is the one failure the marker exists to prevent. In marker terms, the correct behaviour on a permissive-model finding that the agent believes stems from a wrong invariant is to stop and route back to `/informal-spec` for a fresh sign-off, exactly as `/correspondence-review`'s mismatch gate routes back today (`crosscheck/skills/correspondence-review/SKILL.md:170-181`). That is a loop with a human in it, which is slower than Specula's and is the price of the freeze.

**On the freeze.** I cannot state how an adopted loop interacts with a pre-registration freeze I could not find (A1). Against the nearest documented analogues, the answer is: **M1a can be adopted without importing the revision behaviour, and so can M2's property arm, provided the loop's stopping rule routes through `/informal-spec`'s sign-off rather than through agent judgement.** Neither arm needs to rewrite an invariant to make progress. M1a compares a model against recordings and can only conclude that a classification was wrong. M2's property arm compares a model against already-signed-off invariants and can only conclude that the model is permissive or that the invariant is contradicted. Deciding *which* is Specula's §3.5.2 judgement, and Crosscheck does not have to make it inside the loop: it can stop and hand the choice to the human who signed off. The threshold discipline in `/intent-check` (`crosscheck/skills/intent-check/SKILL.md:30-40`) and the phase gates in `/assurance-probe` (`crosscheck/skills/assurance-probe/SKILL.md:231-234`) are both pre-committed and both read from a tracker rather than from a model's opinion; the property arm should follow that pattern.

### Invariants

**I5.** For every model that passes the replay arm, the property arm evaluates every numbered invariant in the module's informal spec, and the report states one of {satisfied, violated, not evaluable} per invariant with no invariant omitted. *Anchor:* Specula §3.3.2 documents that trace validation alone accepts overfitted repairs, and Figure 6 shows a specific repair that replay accepted and one protocol-level invariant rejected. An arm that evaluates a subset silently reintroduces the gap on the unevaluated invariants.

**I6.** No agent-authored change to a signed-off invariant is applied inside a loop iteration. Every permissive-model finding that the agent attributes to a wrong invariant terminates the loop and returns to `/informal-spec`'s sign-off gate. *Anchor:* Specula §3.5.2 permits mid-run revision gated on an LLM judging the evidence strong, and the authors themselves record that convergence depends on the assumption that agents improve; the SONiC link-manager case took four revisions (§5.4). Crosscheck's countervailing constraint is documented at `crosscheck/skills/informal-spec/SKILL.md:179-185`.

**I7.** No loop terminates by reporting conformance while any invariant is marked `not evaluable`. Either the invariant becomes evaluable or the loop terminates with an explicit incompleteness verdict naming the invariant. *Anchor:* speculative. Specula reports that 99.10% of its invariants are safety properties and only 0.90% are liveness (§5.1), and it checks most liveness through TLC's deadlock detection rather than directly, so the paper does not document what happens to an invariant a checker cannot evaluate. This invariant is proposed to prevent silent narrowing and has no failure behind it.

**I8.** The loop runs at most a fixed, pre-declared number of iterations per module, and on exhaustion it reports the last state rather than the best state. *Anchor:* Specula §5.4 reports that all recorded runs converged, that 91.3% of invariant and model errors were fixed in one iteration and that none took more than four (provenance A), and §3.5.2 states that "in practice, we run Specula under a budget in time or cost". The bound is documented; the convergence guarantee is an assumption the authors name as such.

### Kill criterion

Two thresholds, both computed from the same tracker rows the M4 attribution column would add.

Retire the property arm if, over 20 paired runs across at least three modules, it produces zero findings that the replay arm did not already produce. An arm that never contradicts the other arm is not a second opinion. Specula's own effect size is the reference point and it is small in absolute terms: the conformance loop attributed 17.3% of the mistakes it caught to invariants, against 60.5% to the model (§5.4, provenance A), and the paper reports a single worked case where invariant checking exposed an overfitted repair (Figure 6). Twenty runs is the same sample floor `/assurance-probe` uses for its phase gate (`crosscheck/skills/assurance-probe/SKILL.md:232`).

Retire the arm, and escalate to governance, if more than 30% of its findings are resolved by weakening a signed-off invariant rather than by correcting the model. That is the overfit-migration rate, and 30% mirrors the `/intent-check` kill threshold deliberately (`crosscheck/skills/intent-check/SKILL.md:36`), on the same reasoning stated there: take the layer offline before it corrupts more than one in three of the artefacts it is supposed to protect. Review fortnightly, aligned to the 14-day `/intent-check` window (`:38`).

### What this does NOT catch

- **A permissive model that satisfies every stated invariant.** The arm is bounded by the invariant set. Layer 6 `/spec-adversary` owns the missing-property question (`crosscheck/docs/research/assurance-hierarchy.md:91`).
- **A wrong invariant that both arms happen to satisfy.** Layer 5 `/intent-check` owns spec-versus-intent.
- **Permissiveness over reachable states rather than over inputs.** The property arm evaluates invariants on outputs of sampled or recorded calls, not over a reachable-state space. The exhaustive version is the deferred behavioural stratum at Layer 4 per ADR-0001.
- **Reward hacking in the harness rather than in the model.** Specula documents this failure at §5.5, where a weaker model hacked six of 39 reproductions by injecting illegal state directly. Nothing in this proposal detects a harness that has been made to pass. Unowned; raised as A6.

### Cost

Cheap relative to the arms it pairs with, and the paper offers no isolated figure. Specula's cost accounting attributes most spend to model generation and instrumentation rather than to checking (§5.3, provenance A), and the checking side ran to 1.9 billion generated states on the single worst module, CometBFT's PBTS, while visiting 393 million distinct ones (§5.3, provenance A). Crosscheck's property arm does no state exploration at all; it evaluates invariants on outputs already computed by the replay arm, so its marginal cost is one evaluation per invariant per input. The real cost is human: I6 puts a sign-off round-trip in the loop, and the paper's contrasting figure for the cost of *not* doing that is 91.3% of errors fixed in one autonomous iteration (§5.4, provenance A).

---

## M3. Atomic-transition well-formedness probe

### Mechanism as implemented in Specula

Specula's authors wrote a static analyser "for checking if each action correctly specifies an atomic transition over all declared variables", naming it "an important feature missed by SANY and TLC" and citing the upstream TLA+ issue for the absent `UNCHANGED` check (§4, reference [2], which is tlaplus/tlaplus issue 677). Together with SANY and TLC, the tool lets agents "catch and correct common mistakes such as missing UNCHANGED clauses, duplicate assignments, inconsistent action structures, and malformed configuration files" (§4). Every one of those four defect classes is a TLA+ syntactic obligation: an action must say something about every declared variable, must not assign one twice, and must be shaped like its siblings.

### Gap in Crosscheck

**There is no gap, and the candidate placement is wrong. I reject M3 as drafted.** Three reasons.

The probe's subject does not exist in Crosscheck. The defect class is "an action failed to constrain a declared state variable". Crosscheck has no actions and no declared state variables: ADR-0001 records that Crosscheck has no behavioural-specification stratum and that the decision to add one is deferred with no model checker chosen (`crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:3`, `:70-77`). A well-formedness probe for a language the plugin does not emit has nothing to read.

For the two verifiers Crosscheck does ship, the analogous checks already exist and are not in `/assurance-probe`. For Dafny, frame conditions are the analogue of `UNCHANGED`, and the Dafny verifier enforces `modifies` clauses as part of verification; the plugin routes that through `dafny_verify` on touched specs (`crosscheck/docs/research/assurance-hierarchy.md:87`). For Lean, the analogue of "every declared variable is constrained" is "every function in the spec's Module boundary has a model counterpart, and no model definition lacks a source counterpart", and `/correspondence-review` already checks exactly that in both directions (`crosscheck/skills/correspondence-review/SKILL.md:79`, `:81`), with a `def := sorry` refusal upstream (`:57`) and a `lake build` gate (`:63`). The remaining Lean analogue, whether a stated theorem is vacuous, is partially covered elsewhere: `/generate-verified` already flags empty lemma bodies and trivial proofs as possible over-claimed postconditions (`crosscheck/skills/generate-verified/SKILL.md:65-66`).

The proposed host cannot accept it. `/assurance-probe` measures *test strength against source*, not specification well-formedness. Its mutation probe parses the `Failure condition` clause of an invariant doc and mutates Python source (`crosscheck/skills/assurance-probe/SKILL.md:26-31`, `:298`); its vacuity probe deletes a covering test and measures branch-coverage delta (`:35-38`). The word "vacuity" there means "the test is not load-bearing", not "the specification is trivially satisfiable". Adding a spec-well-formedness probe would make the skill's tracker rows incomparable across probe kinds, and the tracker is the input to the phase gates that decide whether Phase 2 may run at all (`:231-245`). That is a schema break for no signal on the layers Crosscheck currently reaches.

### Proposed change

None. The transferable question the task poses, what the equivalent well-formedness check is for a non-TLA+ verifier, has an answer per verifier and none of the answers needs a new probe: Dafny's is frame conditions, enforced by the verifier; Lean's is bidirectional counterpart completeness, enforced by `/correspondence-review`; the genuinely uncovered case is the behavioural specification that ADR-0001 defers. When `/behavioral-spec-init` is scoped, the atomic-transition check belongs in *its* parser gate, because ADR-0001 already says that skill should "gate on parser success not on model-check success" (`crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:75`) and Specula's finding is precisely that parser success is insufficient. That is a note for a future scoping pass, not a change to specify now. Raised as A7.

### Invariants

Not applicable. No change is proposed, so there is nothing to guarantee.

### Kill criterion

Not applicable.

### What this does NOT catch

Not applicable. The omission the mechanism would have covered, well-formedness of a behavioural specification, is owned by Layer 4 per ADR-0001 and is uncovered today because the layer's implementing skill does not exist.

### Cost

Not applicable. For the record, Specula's own cost note is that the analyser is one of three tools handed to the agent and the paper reports no separate cost for it (§4).

---

## M4. Error-attribution telemetry

### Mechanism as implemented in Specula

Specula reports where its loops caught agent mistakes, attributed by artefact. The conformance loop caught and repaired mistakes on the model (60.5%), code instrumentation (22.2%) and invariants (17.3%) (§5.4). It reports convergence in the same terms: instrumentation errors were repaired within three rounds, 91.3% of invariant and model errors were fixed in one iteration, and none took more than four, with the single four-revision case traced to a specific wrong invariant in SONiC's link manager (§5.4). It reports disposition for violations entering reproduction: 47.5% reproduced as real bugs, 48.8% were discharged with evidence drawn from the system and fed back to repair the model or its invariants, and 1.0% were bugs the system could not reproduce (§5.4). §5.1.1 gives the false-positive accounting in the same shape: 136 reported, 134 reproduced and encoded in tests, two reproduced without severe consequence and counted as masked.

### Gap in Crosscheck

Real, and precisely locatable. `/intent-check`'s tracker has four columns, `date,invariant_touched,phase_verdict,human_verdict` (`crosscheck/skills/intent-check/SKILL.md:127`), and the reviewer's `human_verdict` takes one of four values, `genuine`, `genuine-planted`, `partial`, `spurious` (`:133`). The rolling rate counts `spurious` over classified rows and trips at 30% (`:52-60`). Nothing records which artefact was wrong. A row marked `spurious` is consistent with the invariant prose being imprecise, the covering test being mis-linked, the diff being out of scope, the back-translator lacking the domain noun, or the diff-checker mis-scanning a carve-out. The skill enumerates the first and fourth of those as known blind spots (`:207-213`) but does not measure their relative frequency.

The consequence is that the kill criterion is uninterpretable when it trips. `/intent-check` tells the user, on tripping, that "the prompt or model is the problem, not the user's invariant docs" (`:215`). That conclusion is asserted, not measured, and the tracker cannot support it. `crosscheck/docs/research/assurance-hierarchy.md:95` already concedes the underlying weakness: the thresholds are "founder intuition, not labelled-pilot data" with "no calibration trace".

`/assurance-probe`'s tracker has the same shortcoming in a different shape. Its columns record disposition, `date,module,proposed,accepted,rejected,deferred,skipped` (`crosscheck/skills/assurance-probe/SKILL.md:178-180`), and its `Reject` triage examples show cause being written into free prose in a GitHub issue rather than into a field (`:281-291`). Its own SNR gate reads `accepted` over `rejected` (`:236-245`), so a rejection caused by an unreachable mutation and one caused by a bad `Failure condition` clause count identically against the gate.

### Proposed change

Add a cause column to both trackers and a distribution block to the dashboard.

Extend `.assurance/intent-check-fp-tracker.csv` with one column, `attributed_stage`, filled by the human reviewer at the same time as `human_verdict` and drawn from a closed vocabulary matched to the skill's own pipeline: `invariant-prose`, `covering-test`, `diff-scope`, `back-translator`, `diff-checker`, `unattributed`. Extend `.assurance/probe-tracker.csv` with one column, `reject_cause`, drawn from a closed vocabulary matched to that skill's own documented rejection reasons: `generator-unreachable`, `mutation-invalid`, `failure-condition-imprecise`, `test-weak-confirmed`, `unattributed`. Add a distribution block to `/assurance-status`'s dashboard reporting each tracker's cause distribution over the same rolling window the FP rate uses, and quote the distribution in the tripping message so the user reads a measured attribution rather than an asserted one.

Files affected: `crosscheck/skills/intent-check/SKILL.md` (Step 5, Step 8's report block, Step 9's blind-spot list), `crosscheck/skills/intent-check/references/fp-tracker-schema.md`, `crosscheck/skills/assurance-probe/SKILL.md` (tracker schema and triage sections), `crosscheck/skills/assurance-status/SKILL.md` (dashboard), and the squad workflow scripts that read the same tracker, `crosscheck/docs/examples/workflows/tier-b/assurance_squad_select.py` and `assurance_pr_gate_plan.py`.

The schema break is the substance of this proposal and it is why it is not free. `crosscheck/skills/intent-check/SKILL.md:135` states "do not add columns and do not rename columns", with a stated reason: cross-repo concatenability for calibration analysis. That instruction is the thing this mechanism asks to change, and it must change in one coordinated edit across the skill, the reference schema, the status dashboard and the two workflow scripts, or the parity requirement at `:40` breaks.

### Invariants

**I9.** Every tracker row whose `human_verdict` is non-empty also carries a non-empty `attributed_stage` (or `reject_cause`), and a row failing this is excluded from both the numerator and the denominator of the rolling rate. *Anchor:* the existing exclusion rule for empty `human_verdict` (`crosscheck/skills/intent-check/SKILL.md:52`) exists so unreviewed rows cannot move the rate; a half-classified row would move it while carrying no attribution.

**I10.** The attribution vocabulary is closed and every value maps to exactly one named stage of the skill that wrote the row. No free-text values are accepted. *Anchor:* Specula §5.4 reports attribution against exactly three artefacts, model, instrumentation and invariants, summing to 100%, which is what makes the numbers comparable across 48 systems. `/assurance-probe`'s current practice of writing cause into free prose in a GitHub issue (`crosscheck/skills/assurance-probe/SKILL.md:281-291`) is the counterexample: 20 rejections produce 20 unaggregatable sentences.

**I11.** When a kill criterion trips, the message quotes the cause distribution over the same window as the rate, with counts per vocabulary value. *Anchor:* `crosscheck/skills/intent-check/SKILL.md:215` currently asserts the cause on tripping ("the prompt or model is the problem, not the user's invariant docs") on no evidence, and `crosscheck/docs/research/assurance-hierarchy.md:95` concedes there is no calibration trace behind the threshold.

**I12.** Every consumer of a tracker file reads the same schema version, and a consumer encountering a row count of columns it does not recognise refuses rather than silently ignoring the extra column. *Anchor:* Crosscheck's own documented failure. `crosscheck/docs/research/phase-roadmap-may-2026.md` Phase 0 records that "the first import had schema drift between the squad scripts and the skill: different file path, different verdict marker, different window", requiring a second harmonisation commit across six files. This mechanism reopens exactly that seam.

### Kill criterion

Retire the attribution column if, over a rolling 90-day window containing at least 30 classified rows, either of the following holds. More than 40% of rows carry `unattributed`, meaning reviewers cannot in practice assign cause and the column is collecting noise. Or one vocabulary value accounts for more than 90% of rows in three consecutive windows, meaning the distribution carries no routing information that a single sentence in the skill's documentation would not carry more cheaply.

The 30-row floor is taken from the repository's own stated calibration threshold: "tune them for your tolerance once you have >= 30 classified human verdicts" (`crosscheck/skills/intent-check/SKILL.md:40`). The 90-day window is longer than the 14-day FP window because attribution counts accumulate more slowly than verdicts. Review quarterly.

Both thresholds are stated assumptions. Specula's distribution, 60.5 / 22.2 / 17.3 (§5.4, provenance A), is informative but not transferable: it attributes across three artefacts of a pipeline Crosscheck does not run, over runs whose count the paper does not give, on 48 systems in languages that exclude Python. It licenses the claim that attribution across a small closed vocabulary is achievable and that the distribution is uneven without being degenerate. It does not license any specific threshold, and the 40% and 90% figures are raised as A8.

### What this does NOT catch

- **Whether the attributed stage is the *right* stage.** Attribution is a reviewer's judgement recorded in a field. Nothing checks it. Layer 5 owns reviewer calibration and has no mechanism for it.
- **Errors in stages the vocabulary does not name.** A defect in the attestation hash logic or the tracker append logic has no value to take, and lands in `unattributed`.
- **Cause for rows nobody reviews.** Per I9 unreviewed rows are excluded, so an unreviewed backlog silently shrinks the sample. The existing exclusion rule has the same property (`crosscheck/skills/intent-check/SKILL.md:52`).
- **Anything about bug-finding effectiveness.** This is telemetry about the checkers, not about the code. The layer-attribution question that `crosscheck/docs/research/crosscheck-tla-vgd-addendum.md:100` names as "the single largest credibility-enhancing publication" Crosscheck could ship is a different and larger mechanism.

### Cost

Near zero to run, and the whole cost sits in the schema migration. Specula reports no separate cost for its telemetry, because attribution falls out of a loop that already routes each finding to a repair path (§3.5.1, Figure 7). Crosscheck's cost is one extra field per human review, plus one coordinated edit across five files, plus the loss of concatenability with any tracker row written before the migration. The last item is the real expense and it is the reason the schema was frozen in the first place (`crosscheck/skills/intent-check/SKILL.md:135`).

---

## M5. Code-shape routing

### Mechanism as implemented in Specula

Specula routes on behaviour rather than on syntax, and the paper is explicit about why this works: "Specula applies to system code written in any programming language as it abstracts the code in the TLA+ model" (§1), and TLA+ "is not specific to any implementation languages" (§2.1). The evidence is 48 systems spanning seven languages, C, C#, C++, Erlang, Go, Java and Rust, at 2K to 95K lines (§5, Table 1). The SONiC case is the sharpest instance: five modules in C++, C and Rust, checked "with no module-specific tuning", with at least one bug in each (§5.1.2, Table 3). What Specula routes *on* is not language but scope: it "is instructed to target core logic of concurrent, distributed systems, which requires formal methods and model checking" (§5.1).

Two corrections to the task's framing. Language-agnosticism holds for the model and not for the harness: the trace library "exposes per-language emit functions" and uses two different recording mechanisms depending on whether the system is distributed or concurrent (§4). And Python is not among the seven languages, so the paper carries no evidence for the language Crosscheck's Dafny path targets first.

### Gap in Crosscheck

Partly closed already, and the open part is narrower than the draft says.

The language-keyed routing the draft objects to is real and is the skill's spine. Step 2 detects language from manifests (`crosscheck/skills/assurance-layer-audit/SKILL.md:32-50`), and Step 4's per-layer rules are keyed on language throughout, with the instruction to "apply the following ecosystem rules verbatim" (`:93`), a Layer 1 bullet per language (`:95-101`) and a Layer 2 bullet per language (`:103-109`).

But shape-keyed per-module routing already exists in two places. Step 4.5 assesses each module against the four VGD prerequisites and emits a recommended engine combination, with the routing logic at `:163-168` and the priority rule "per-module routing wins over per-layer routing" at `:223`. Prerequisite 1, deterministic algebraic semantics, is a shape criterion in substance: it fails on "heavy framework callbacks, hidden global state, non-determinism in error paths, or behaviour that depends on wall-clock time" (`crosscheck/skills/informal-spec/SKILL.md:55`). And ADR-0001 already states a five-clause code-shape criterion in the exact register the draft asks for, keyed on state machines, branching workflows with rollback, rule-interaction surfaces, invariant-rich data models and concurrency or distribution (`crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:44-52`), with the explicit correction that the criterion is rule-density rather than distribution (`:54`).

The open gap is that ADR-0001's criterion is not in the skill. The ADR lists the detection logic as deferred work to land with a `/behavioral-spec-init` skill that does not exist (`:70-77`), and `/assurance-layer-audit` Step 4.5's only nod to it is a single conditional line, "#1 `pass` + module is rule-dense (state machines, role hierarchies, workflow branches) -> flag for Layer 4 behavioral-spec enrichment per ADR-0001 (pending implementation)" (`crosscheck/skills/assurance-layer-audit/SKILL.md:168`). A flag pointing at an unimplemented skill is not routing.

On the draft's three rows. The first, concurrent or distributed core logic, has a *decided* verifier and no shipped one: ADR-0001 chose Layer 4 placement and recommended TLA+ first, and both the choice of checker and the implementing skill are open (`:74-75`). The second, pure computation, has two shipped engines with documented reach (`crosscheck/docs/research/assurance-hierarchy.md:27`, `:81`). The third, integration boundaries, has no verifier at any layer and the repository already says so twice (`crosscheck/docs/research/assurance-hierarchy.md:43`, `:85`; `crosscheck/skills/assurance-layer-audit/SKILL.md:111-115`).

### Proposed change

Add a code-shape row to Step 4.5's per-module table, sourced from ADR-0001's criterion rather than invented here, and require the audit to print the honest blank.

`/assurance-layer-audit` Step 4.5 gains a shape classification per module, evaluated before the four prerequisites and reported alongside them. The classification uses three values and a fourth for indeterminacy: `concurrent-or-distributed-core` for the ADR-0001 clauses on state machines, branching workflows with rollback, rule-interaction surfaces, invariant-rich data models and concurrency or distribution; `pure-computation` for input-to-output transformations, which ADR-0001 explicitly routes to Dafny (`crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:52`); `integration-boundary` for modules whose state is external and whose paths are largely sequential; and `mixed` where two apply, which the ADR anticipates with "many real modules need both engines" (`:52`).

The routing table the skill prints must read as follows on the verifier column, and the third row must stay blank.

| Shape | Verifier today | Basis |
|---|---|---|
| Concurrent or distributed core logic | None shipped. Placement decided, checker not chosen, skill not built. | `crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:3`, `:74-75` |
| Pure computation | Dafny verify-and-extract; Lean model plus DRT | `crosscheck/docs/research/assurance-hierarchy.md:27`, `:81` |
| Integration boundaries | None. Pairwise contracts partial in some ecosystems; end-to-end unaddressed. | `crosscheck/docs/research/assurance-hierarchy.md:43`, `:85`; `crosscheck/skills/assurance-layer-audit/SKILL.md:111-115` |

The `.assurance/layer-audit-result.json` schema gains one field per module assessment, `code_shape`, so `/assurance-init` can consume it the way it already consumes `module_assessments[]` for seed-module selection (`crosscheck/skills/assurance-init/SKILL.md:90`). Files affected: `crosscheck/skills/assurance-layer-audit/SKILL.md` (Step 4.5, Step 5's Notes column, Step 6's gap rows, Step 7.5's JSON schema, Step 8's checklist) and `crosscheck/skills/assurance-init/SKILL.md` (Step 6.5's consumption of the cached audit).

Language detection stays. It is not redundant with shape: Layer 2 reach is a property of the toolchain and nothing else (`crosscheck/skills/assurance-layer-audit/SKILL.md:103-109`), and Specula's own harness is per-language even where its model is not (§4).

### Invariants

**I13.** Every module the audit assesses carries exactly one shape classification from the closed set {`concurrent-or-distributed-core`, `pure-computation`, `integration-boundary`, `mixed`}, and the classification cites the specific ADR-0001 clause or the specific absence of one that drove it. *Anchor:* `crosscheck/skills/assurance-layer-audit/SKILL.md:293` already requires that per-layer projections be "derived from the detected tooling, not copied from any reference example", and `:298` requires every "not addressable" claim to name the missing tool. An uncited shape verdict would be the one unevidenced cell in a table whose every other cell is evidenced.

**I14.** For every module classified `integration-boundary`, the audit's verifier column reads that no verifier is available and names Layers 4 to 6 as the fallback. It never names a verifier. *Anchor:* `crosscheck/docs/research/assurance-hierarchy.md:85` and `crosscheck/skills/assurance-layer-audit/SKILL.md:111-115`. The audit's own governing instruction is "be honest about limits" and "when in doubt, err toward the more pessimistic reading" (`:24`, `:91`).

**I15.** For every module classified `concurrent-or-distributed-core`, the audit states that the routing target is decided and unimplemented, and cites the ADR. It does not emit a recommended next skill for that row. *Anchor:* `crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:3` and `:70-77`. Step 6 requires each gap row to name "a concrete next skill" (`crosscheck/skills/assurance-layer-audit/SKILL.md:301`); for this row there is none, so the requirement must be relaxed explicitly rather than satisfied by naming a skill that does not exist.

**I16.** Shape classification never overrides a prerequisite verdict. A module classified `pure-computation` whose prerequisite 1 verdict is `fail` is still routed away from Layer 1. *Anchor:* the existing routing logic at `crosscheck/skills/assurance-layer-audit/SKILL.md:163-168`, where prerequisite 1 failing is what removes Layer 1 from consideration, and `:223`, where per-module routing wins over per-layer routing.

### Kill criterion

Retire the shape row if, over 20 audits covering at least 50 module assessments, the human reviewer re-classifies more than 30% of shape verdicts. That is the same threshold as `/intent-check`'s and for the same reason: a classifier wrong on more than one in three cases is worse than no classifier, because it routes work confidently. Review quarterly, since audits are pre-onboarding diagnostics rather than per-PR checks (`crosscheck/skills/assurance-layer-audit/SKILL.md:20`).

A second retirement condition, on value rather than accuracy: retire if across those 50 assessments the shape verdict never changes the recommended engine combination that the four prerequisites alone would have produced. Specula's evidence for shape-over-language routing is strong on its own terms, five SONiC modules in three languages with no module-specific tuning and at least one bug in each (§5.1.2, provenance A for the bug counts and C for none of them), but it is evidence that a *behavioural model* is language-agnostic, not evidence that a shape classifier improves a routing decision between Dafny and Lean. The second condition tests the part the paper does not license.

### What this does NOT catch

- **Whether the module the audit assessed is the module that matters.** Step 4.5 auto-detects two to four modules by ecosystem heuristics and otherwise asks the user (`crosscheck/skills/assurance-layer-audit/SKILL.md:132-139`). A shape verdict on the wrong module is precise and useless. Unowned.
- **Shape drift.** The audit runs once, before onboarding. `/assurance-roadmap-check` owns staleness for roadmap items (`crosscheck/skills/assurance-roadmap-check/SKILL.md:22`) and nothing owns staleness for the audit artefact.
- **Anything about the concurrent-or-distributed-core row beyond naming it.** Per I15 the row is a diagnosis with no treatment. Layer 4 owns it, pending ADR-0001's deferred work.
- **Cross-module shape.** A system whose every module is a pure computation can still have an emergent state machine across them. Layer 3 owns composition and reaches nothing end-to-end (`crosscheck/docs/research/assurance-hierarchy.md:41`, `:85`).

### Cost

Cheapest of the five. The audit already reads the repository and already emits a per-module table (`crosscheck/skills/assurance-layer-audit/SKILL.md:141-154`); the shape verdict adds one row per module to an existing table and one field to an existing JSON artefact. No new tooling, no new dependency, no runtime.

Specula's transferable cost figure is a warning rather than a price. Its 48 systems required no per-language modelling work for the model, but they did require per-language instrumentation (§4), and its per-system cost ran to a median of 3.69 hours and $57 (§5.3, provenance A). Nothing in that number attaches to routing. The cost the paper does document for getting routing wrong is indirect: with a weaker model, Specula found 10 of the 62 bugs at 61% of the budget, roughly $59 per bug against $16 (§5.5, provenance A), which is the strongest evidence in the paper that this class of mechanism is sensitive to the agent driving it rather than to the routing table.

---

## Cross-mechanism dependencies

**M2 depends on M1a, and M1a does not depend on M2.** This is the pairing's whole point and the direction matters. M1a alone rewards a permissive model, because a model that reproduces every recorded output while admitting behaviour production never produces passes replay. Specula documents this failure directly (§3.3.2) and Figure 6 gives the worked case. Adopting M1a without M2 therefore adds a mechanical arm whose incentive gradient points the wrong way, and Crosscheck's existing counterweight is weak: `/correspondence-review`'s check for model-only behaviour is a reading judgement (`crosscheck/skills/correspondence-review/SKILL.md:79`) that M1a's mechanical evidence would tend to outrank in practice. M2 without M1a is coherent but small: the property arm would evaluate invariants against a model driven only by generated inputs, which is a useful check on the model and not a conformance check at all.

**M4 depends on nothing and is depended on by M1a's and M2's kill criteria.** Both kill criteria above are stated as ratios over triaged findings by cause, and neither is computable from the current tracker schemas. M4 is therefore a prerequisite for measuring M1a and M2 rather than for running them, which makes it the correct first item: it is the only one of the five that improves the repository's ability to retire the other four.

**M5 depends on nothing among M1 to M4.** It touches a different skill at a different phase of the workflow. It has a soft dependency on ADR-0001's deferred work in the sense that its first row stays blank until `/behavioral-spec-init` exists, but the row's value is the honest blank, so the dependency does not block adoption.

**M3 is rejected and nothing depends on it.** The re-siting note, that the atomic-transition check belongs in `/behavioral-spec-init`'s parser gate, creates a dependency in the opposite direction: ADR-0001's deferred work item 2 would depend on Specula's finding, not the reverse.

**M1b depends on ADR-0001's deferred work in full.** Behavioural trace validation needs a behavioural model, a chosen checker and an exercised orchestration stance. All three are open.

## Sequencing

Proposed roadmap items, one per adopted mechanism, using the `Status:` vocabulary at `crosscheck/skills/assurance-init/SKILL.md:142-148` and the four horizon directories at `:122-151`. **None of these can be filed as written today, because `docs/assurance/` does not exist in this repository** (see Sources read and Protected surfaces impact). Item 0 fixes that and is a hard prerequisite for the other four, because `/protected-surface-amend` refuses without both the roadmap directory (`crosscheck/skills/protected-surface-amend/SKILL.md:113-116`) and the protected-surfaces rule file (`:48-49`), and every one of the other items edits a `SKILL.md`.

**Item 0. Scaffold this repository's own assurance governance.** Horizon: `immediate/`. Status: `Not started`. Scope: create `docs/assurance/ROADMAP.md`, the four horizon directories and `.claude/rules/protected-surfaces.md` for the marketplace repository itself, per `/assurance-init` Steps 3 to 5. The plugin has shipped the scaffolding skill and has never run it on itself, which is why 33 references point at a file that does not exist. Blocker to name if this is deliberate: whether the marketplace repository is intentionally out of scope for its own governance (A2).

**Item 1. Error-attribution telemetry (M4).** Horizon: `immediate/`. Status: `Not started`. Scope: the `attributed_stage` and `reject_cause` columns, the `/assurance-status` distribution block, and the coordinated migration across the five files named in M4. Ordered first among the mechanisms because it is the cheapest, because it breaks a frozen schema and so should not queue behind other tracker changes, and because the other items' kill criteria read from it.

**Item 2. Code-shape routing in `/assurance-layer-audit` (M5).** Horizon: `next/`. Status: `Not started`. Scope: the shape classification in Step 4.5, the three-row routing table with the integration-boundary row blank, the `code_shape` field in the audit JSON, and the Step 8 checklist rows. Independent of items 1, 3 and 4.

**Item 3. Recorded-input replay (M1a).** Horizon: `next/`. Status: `Blocked`. `Blocker:` item 4, because M1a must not ship without the permissiveness counterweight; see Cross-mechanism dependencies. Scope: the production-side recorder, the recorded-input mode in `/drt-oracle`, and the pre-classification evidence step plus refusal in `/correspondence-review`.

**Item 4. Bidirectional validation, property arm (M2).** Horizon: `next/`. Status: `Not started`. Scope: the property arm in `/drt-oracle`, the fifth divergence class, and the stopping rule that routes a suspected-wrong-invariant finding back to `/informal-spec`'s sign-off gate rather than revising in place. Ships with or before item 3.

**Item 5. Behavioural trace validation (M1b) and the atomic-transition parser gate (M3 re-sited).** Horizon: `aspirational/`. Status: `Deferred`. `Reason:` both require the behavioural-specification stratum that ADR-0001 accepted as a decision and deferred as implementation, with no model checker chosen (`crosscheck/docs/research/adr/0001-behavioral-specs-at-layer-4.md:3`, `:70-77`). Supersedes nothing; superseded by whatever scoping pass takes ADR-0001 deferred work items 1 and 2.

## Protected surfaces impact

Two prior findings frame this section. First, the authoritative partition file does not exist, so classification below is derived from the template that would create it (`crosscheck/skills/assurance-init/SKILL.md:163-186`), not from the file itself. Second, `/protected-surface-amend` currently refuses twice over on this repository: once because the rule file is absent (`crosscheck/skills/protected-surface-amend/SKILL.md:48-49`) and once because no roadmap item can be located under a `docs/assurance/` that does not exist (`:113-116`). The second refusal is unconditional and "is not relaxed in agent mode" (`:116`). **Every mechanism above is therefore ungovernable until item 0 lands.**

Classification. `SKILL.md` files are Class A by the template's fourth bullet, "any file the harness interprets as 'ground truth' for agent behaviour" (`crosscheck/skills/assurance-init/SKILL.md:175`), reinforced by `CLAUDE.md`'s commit convention that `SKILL.md` and `agents/*.md` are "behavioral artifacts" and "functional code". The tracker reference schemas and the squad workflow scripts are Class A under the workflow-definitions and runtime-prompt-template bullets (`:172-174`). No Class B file is touched by any mechanism: nothing here edits `docs/invariants/*.md` or a covering property test.

Governance-note skeletons follow, per the block at `crosscheck/skills/protected-surface-amend/SKILL.md:134-184`. They are skeletons only. **No edit is applied and none should be, until item 0 lands and each block's Governing Roadmap Item resolves to a real path.**

### Skeleton A. M4, tracker schema and dashboard

```markdown
## Protected-Surface Amendment

**Target file(s):** crosscheck/skills/intent-check/SKILL.md (+4 others, see Diff Plan)
**Class:** A — Harness/workflow definitions
**Matched rule:** REQUIRES HUMAN VERIFICATION: no `.claude/rules/protected-surfaces.md` exists in this repo; classification derived from the Class A template at crosscheck/skills/assurance-init/SKILL.md:175.
**Date:** <YYYY-MM-DD>

### Change Description

Adds one column to each of two tracker schemas (`attributed_stage` to the intent-check FP tracker, `reject_cause` to the probe tracker), adds a cause-distribution block to the /assurance-status dashboard, and quotes the distribution in the kill-criterion tripping message. Replaces the frozen-schema instruction at intent-check/SKILL.md:135.

### Rationale

REQUIRES HUMAN VERIFICATION: Rationale draft below is not anchored to a concrete trigger. No incident, audit-finding ID or kill-criterion breach exists for this change; it originates in an external paper. Draft: the kill criterion at intent-check/SKILL.md:215 asserts a cause the tracker cannot evidence, and assurance-hierarchy.md:95 concedes there is no calibration trace behind the threshold. Reviewer must either accept a documentation-gap anchor or escalate.

### Governing Roadmap Item

- **Path:** REQUIRES HUMAN VERIFICATION: `docs/assurance/` does not exist. Blocked pending Sequencing item 0.
- **Title:** Error-attribution telemetry (Sequencing item 1)
- **Scope coverage:** n/a until item 0 lands.

### Authority

- **Authoriser:** REQUIRES HUMAN VERIFICATION: named human required; this document was drafted by an agent.
- **Role:** <PR author / module owner>

### Diff Plan

| # | File | Lines | Invariant ID / Stage | Action |
|---|------|-------|----------------------|--------|
| 1 | crosscheck/skills/intent-check/SKILL.md | 122-135, 179-196, 200-216 | Step 5 / Step 8 / Step 9 | replaced |
| 2 | crosscheck/skills/intent-check/references/fp-tracker-schema.md | whole file | tracker schema | replaced |
| 3 | crosscheck/skills/assurance-probe/SKILL.md | 109-113, 176-189, 253-292 | tracker + triage | replaced |
| 4 | crosscheck/skills/assurance-status/SKILL.md | Step 2.4 | dashboard | added |
| 5 | crosscheck/docs/examples/workflows/tier-b/*.py | tracker readers | reader parity | replaced |

### Test / Coverage Impact

- No Class B invariant is touched; no covering property test changes.
- BLOCKING: schema parity. intent-check/SKILL.md:40 requires every consumer of the tracker to compute the rate identically. phase-roadmap-may-2026.md Phase 0 records a prior drift between the squad scripts and the skill that took a second commit to harmonise. All five files must land in one PR.
- Pre-migration rows lose concatenability with post-migration rows; state the cut-over date in the reference schema.

### Review Checklist

- [ ] Rationale is anchored to a concrete trigger (not "cleanup" or "robustness").
- [ ] Authoriser is a named human (not a bot, not an agent).
- [ ] Governing roadmap item exists and actually covers this change.
- [ ] Diff plan enumerates every affected file, line range, and invariant ID / stage.
- [ ] Class A edits: downstream attestation / intent-check baseline refresh is queued.
- [ ] All `REQUIRES HUMAN VERIFICATION:` markers above have been resolved.
```

### Skeleton B. M5, audit routing

```markdown
## Protected-Surface Amendment

**Target file(s):** crosscheck/skills/assurance-layer-audit/SKILL.md (+1 other, see Diff Plan)
**Class:** A — Harness/workflow definitions
**Matched rule:** REQUIRES HUMAN VERIFICATION: rule file absent; derived from assurance-init/SKILL.md:175.
**Date:** <YYYY-MM-DD>

### Change Description

Adds a code-shape classification to Step 4.5 per module, sourced verbatim from ADR-0001's routing heuristic at lines 44-52; adds a three-row routing table whose integration-boundary row names no verifier; adds a `code_shape` field to the Step 7.5 JSON artefact; relaxes Step 6's "name a concrete next skill" requirement for the concurrent-or-distributed-core row; adds three Step 8 checklist rows.

### Rationale

Anchored: ADR-0001 is Accepted as a decision with implementation deferred (line 3) and lists the detection logic as deferred work item 2 (lines 70-77). The skill's only current acknowledgement is a single flag pointing at an unimplemented skill (assurance-layer-audit/SKILL.md:168).

### Governing Roadmap Item

- **Path:** REQUIRES HUMAN VERIFICATION: `docs/assurance/` does not exist. Blocked pending Sequencing item 0.
- **Title:** Code-shape routing (Sequencing item 2)
- **Scope coverage:** n/a until item 0 lands.

### Authority

- **Authoriser:** REQUIRES HUMAN VERIFICATION: named human required.
- **Role:** <PR author / module owner>

### Diff Plan

| # | File | Lines | Invariant ID / Stage | Action |
|---|------|-------|----------------------|--------|
| 1 | crosscheck/skills/assurance-layer-audit/SKILL.md | 128-171, 172-198, 199-226, 237-284, 285-306 | Steps 4.5, 5, 6, 7.5, 8 | added / replaced |
| 2 | crosscheck/skills/assurance-init/SKILL.md | Step 6.5 | cached-audit consumption | replaced |

### Test / Coverage Impact

- No Class B invariant touched.
- The JSON artefact schema gains a field; /assurance-init reads it (assurance-init/SKILL.md:90). Both files land in one PR.
- BLOCKING risk if the integration-boundary row is filled with any verifier name: that would contradict assurance-hierarchy.md:85 and the audit's own honesty instruction at assurance-layer-audit/SKILL.md:24.

### Review Checklist

- [ ] Rationale is anchored to a concrete trigger.
- [ ] Authoriser is a named human.
- [ ] Governing roadmap item exists and actually covers this change.
- [ ] Diff plan enumerates every affected file and stage.
- [ ] Integration-boundary row names no verifier.
- [ ] All `REQUIRES HUMAN VERIFICATION:` markers resolved.
```

### Skeleton C. M1a and M2, Lean pipeline

```markdown
## Protected-Surface Amendment

**Target file(s):** crosscheck/skills/drt-oracle/SKILL.md (+2 others, see Diff Plan)
**Class:** A — Harness/workflow definitions
**Matched rule:** REQUIRES HUMAN VERIFICATION: rule file absent; derived from assurance-init/SKILL.md:175.
**Date:** <YYYY-MM-DD>

### Change Description

Adds a recorded-input mode and a property arm to /drt-oracle, extending the divergence taxonomy from four classes to five (adding `permissive model`); adds a pre-classification mechanical-evidence step and a matching refusal to /correspondence-review; makes the `-- src:` comment in /lean-impl load-bearing for instrumentation as well as for review. Adds a loop stopping rule that routes a suspected-wrong-invariant finding back to /informal-spec's sign-off gate and forbids in-loop revision of a signed-off invariant.

### Rationale

Anchored to two documented positions rather than to an incident. /drt-oracle states that its input generation may "hit shallow surfaces only" (line 58) and that not finding a witness is not a soundness guarantee (line 77); /correspondence-review states that its classification is a judgement to be downgraded under uncertainty (line 51). REQUIRES HUMAN VERIFICATION: reviewer must confirm that a documented-limitation anchor is acceptable, or supply an incident.

### Governing Roadmap Item

- **Path:** REQUIRES HUMAN VERIFICATION: `docs/assurance/` does not exist. Blocked pending Sequencing item 0.
- **Title:** Recorded-input replay + property arm (Sequencing items 3 and 4)
- **Scope coverage:** n/a until item 0 lands.

### Authority

- **Authoriser:** REQUIRES HUMAN VERIFICATION: named human required.
- **Role:** <PR author / module owner>

### Diff Plan

| # | File | Lines | Invariant ID / Stage | Action |
|---|------|-------|----------------------|--------|
| 1 | crosscheck/skills/drt-oracle/SKILL.md | 34-45, 85-116, 118-168, 170-219, 240-257 | D4 cases, Step 0, Steps 1-3, Step 4 report, checklist | added / replaced |
| 2 | crosscheck/skills/correspondence-review/SKILL.md | 71-82, 83-100, 170-181, 186-202 | Steps 1, 2, 5, checklist | added / replaced |
| 3 | crosscheck/skills/lean-impl/SKILL.md | 137, 204 | `-- src:` contract | replaced |

### Test / Coverage Impact

- No Class B invariant touched.
- The shipped smoke case at formal-verification/tests/power/ exercises the generated-input path only; a recorded-input smoke case is required in the same PR or the new mode ships unexercised.
- BLOCKING: /informal-spec's sign-off marker regex (informal-spec/SKILL.md:185) becomes load-bearing for a second consumer. Any change to the marker format now breaks two gates.

### Review Checklist

- [ ] Rationale is anchored to a concrete trigger.
- [ ] Authoriser is a named human.
- [ ] Governing roadmap item exists and actually covers this change.
- [ ] No invariant is being weakened purely to make a failing check pass.
- [ ] The property arm cannot revise a signed-off invariant in-loop (I6).
- [ ] All `REQUIRES HUMAN VERIFICATION:` markers resolved.
```

## Ambiguities

A1. Where is the Crosscheck eval mandate, and what does it say? I searched all Markdown in this repository case-insensitively for `pre-regist`, `preregistration`, `eval mandate`, `oracle independence`, `independent oracle`, `validity invariant` and `freeze`, and found no document naming any of them; is the mandate an unwritten policy, a document in another repository, or a document that should exist here?

A2. Is the marketplace repository deliberately exempt from its own assurance governance, or is the absence of `docs/assurance/` and `.claude/rules/protected-surfaces.md` an unclosed scaffolding gap that 33 in-repo references already assume is closed?

A3. When the pre-registration freeze and the sign-off marker disagree, which wins? Concretely: a signed-off informal spec whose invariant the property arm contradicts, where the human who signed off is unavailable and the loop has run out of iterations.

A4. Should the behavioural-specification stratum that ADR-0001 defers be scoped now, given that M1b and the re-sited M3 both depend on it and neither can be specified without it?

A5. What multiple of the generated-input arm's cost is an acceptable ceiling for the recorded-input arm before the mechanism is retired? I used a factor of three and it is a stated assumption with nothing behind it.

A6. Who owns detection of a harness that has been made to pass? Specula documents six of 39 reproductions being hacked by injecting illegal state under a weaker model (§5.5), and nothing in Crosscheck or in this proposal detects it.

A7. Does the atomic-transition well-formedness check belong in `/behavioral-spec-init`'s parser gate when that skill is scoped, and if so should ADR-0001's deferred work item 2 be amended now to record the requirement while the source is fresh?

A8. What `unattributed` ceiling and what single-value concentration ceiling should retire the attribution column? I used 40% and 90% and both are stated assumptions; Specula's 60.5 / 22.2 / 17.3 distribution licenses neither.

A9. Should the recorded-input recorder be allowed to run against production, or only against staging and test environments? Specula records from running system code under its own instrumentation (§3.3.1, §4) in a research setting; Crosscheck adopters would be recording from their own systems, and the input-capture question is a data-handling question this document does not touch.

A10. Specula generates *scenarios* from artefacts and projects a reference model into scenario-specific models by disabling, coarsening and serialising actions (§3.2.2, §3.2.3, Figure 5), reporting that breadth-first search sufficed for 187 of 200 violations because the decomposition kept each state space tractable (§5.1). This is a sixth mechanism and I have not specified it, per the five-entry cap. Is scenario-based decomposition worth a separate assessment, given that Crosscheck's nearest analogue is the Module boundary section of `/informal-spec` (`crosscheck/skills/informal-spec/SKILL.md:79`), which scopes but does not project?
