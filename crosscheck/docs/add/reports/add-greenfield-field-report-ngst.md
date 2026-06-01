# ADD greenfield field report — `ngst`

**Status:** Field-test findings (analysis)
**Analysis date:** 2026-06-01
**Subject run:** the greenfield ADD build of `~/repos/ngst` (2026-05-08 → 2026-05-14)
**Evidence basis:** the `ngst` repo's committed ADD artifacts (`JOURNAL.md`, `docs/design/ngst-spec.md`, 16 `docs/invariants/*.md`, `docs/assurance/ROADMAP.md`, the `add-session-59a24679cf243a01/` session artifacts, the `spec-adversary` records) plus the build conversation transcripts (sessions `2ff1cde8`, `2764b31a`, `15f94ff7`, `e11dca2c`, `a108b3a9`, `d6c275da`).

---

## 1. Why this report exists

`crosscheck/docs/add/README.md` names two preconditions before ADD v3 earns a committed methodology doc: *"at least one mature-repo and one greenfield spec session have been driven against it."* The mature-repo run was recorded on 2026-05-15 (`JOURNAL.md`). **This report records the greenfield run.** With both now driven, the v3 *shape* is no longer an untested hypothesis — it has produced a real, partially-shipped greenfield application.

`ngst` is a Go webhook orchestrator (GitHub ingress → Postgres-mediated dispatcher → OPA-evaluated rules → Fabro REST workflow launch) built **spec-first, invariant-anchored, from a near-empty repo**. It is the strongest greenfield evidence available, and every open child of epic #217 (#218 Phase 4, #219 modes, #220 auditor, #227 heading convention) is implicated by what the run surfaced.

## 2. What the greenfield run exercised

| ADD step | How `ngst` exercised it | Artifact |
|---|---|---|
| Bootstrap / governance install | `AGENTS.md` walk-up rule + invariant-anchored workflow; dual-track coverage gate (`lefthook` + CI) | `AGENTS.md`, `scripts/check_invariant_coverage.go` |
| Spec-first invariant drafting | `internal/secrets` retrofit: invariants drafted in English, signed off, then tests + impl | `docs/invariants/secrets.md` |
| `add-orchestrator` spec-driven fast path | 14 modules drafted in parallel from the 83 KB spec; batched 3-axis audit; operator triage; marker-file hash discipline | `.assurance/add-session-59a24679cf243a01/` |
| Layer-6 adversarial completeness | `/spec-adversary` on `github-client`; per-run tracker with kill criteria | `.assurance/spec-adversary/` |
| Phase-4-style implementation | `cli` module: invariants → tests → implementation, with independent pre- **and** post-implementation review (byfuglien + hellebuyck) | `JOURNAL.md` 2026-05-14, `docs/invariants/cli.md` |
| Diff classification in practice | Every `JOURNAL.md` entry carries a `Type:` (`propagated-discovery` / `intent-refinement` / `drift` / `retraction` / `status-transition`) | `ngst/JOURNAL.md` |

This is the whole pipeline, end to end, on a real greenfield codebase — not a toy.

## 3. What worked (validation evidence)

**V1 — Invariant-first review catches real bugs that "looks fine" review misses.** The single sharpest result: drafting `secrets.md` surfaced a *mutually-exclusive contradiction* between two committed artifacts — issue #18's acceptance criteria said "both env and `_FILE` set is an error," while `CONVENTIONS.md` said "`_FILE` wins." A "looks fine" review of the passing test would have shipped the wrong one and broken any deployment that left a default env var set while mounting a file secret. Invariant-first forced the contradiction into the open before merge. ngst's own JOURNAL calls this *"the first concrete example of what invariant-first review catches that 'looks fine' review does not."*

**V2 — The parallel fast path surfaces spec *gaps*, not just doc nits.** The `add-orchestrator` batched audit over 14 modules produced 26 findings; the coverage axis caught behaviours the spec itself left open — §7.4 hot-reload rebuild-failure handling, §3.3 launch-side 5xx retry, §12.4 metrics completeness, §14.2 open verification items — each promoted to a new invariant. None would have appeared in code review until a production incident.

**V3 — Cross-module consistency catches ownership conflicts invisible to per-module work.** The audit found two modules (`fabro-client`, `watcher`) both claiming a MUST to inspect the Fabro cancellation-reason — contradictory. And the core safety property (at-most-one-active-per-key) and the atomic budget transition were each duplicated across two modules. All resolved by designating a canonical owner. This class of conflict is *structurally invisible* to sequential single-module drafting.

**V4 — Independent multi-agent review tightens contracts.** On the `cli` module, byfuglien (contract lens) applied a strict reading of I2 ("stdout MUST be JSON under `--json`") that would have parked the error envelope on stderr; the team kept it on stdout because the test sketch required `json.Valid(stdout)` on the fault path. hellebuyck (governance lens) caught a *coverage-gate blocker* — the gate derives module name from `filepath.Base(filepath.Dir(p))`, so tests in `cmd/ngst/` would tag `module=ngst` while `cli.md` declares `module=cli`, breaking the bidirectional link. Both were caught **pre-merge**; the module-derivation bug was only visible once tests existed, never to code review.

**V5 — Operator triage was disciplined and evidence-driven.** 17 accept-fix / 5 defer / 4 reject / 0 spec-amendments, each with a concrete brief or revisit condition. Zero "looks fine" verdicts. The triage log (`triage-log.md`) is a durable, reviewable record.

**V6 — Honest deferral, no premature graduation.** All six `cli` invariants stayed `<!-- aspirational -->` at merge because their subcommands return `not_implemented` until dependent modules land. The invariants are ratified by tests; graduation is gated on readiness. Discipline, not theatre.

## 4. Friction and gaps (the open work, observed)

**G1 — Cold-elicitation against an answered spec (→ referential-integrity failure #150).** `/draft-invariants` opened with the *correct* methodology preamble ("anchor against the spec and skeleton") and then immediately ran a cold contract interview the spec already answered. The operator had to redirect (*"here's the spec excerpt"*). When the inferred answers drifted from the spec, the result was referential-integrity failure **#150**. The skill's prose matched the methodology; its behaviour did not. This is exactly the failure the **A1 acceptance oracle** encodes and the failure **greenfield mode (#219)** must prevent with a *"did the spec already answer this?"* gate before any elicitation.

**G2 — Heading-convention divergence (observed in the wild).** The 14-module parallel draft produced **five different heading styles** across subagents (`## I1: Name`, `## IN.1:`, `## I1` no-colon, prose-section-with-buried-ID, and bold-prefix `**I1.**`). Audit finding F1 (High) flagged it; it was *deferred* to the coverage-scaffold rollout. This is the precise inconsistency **#227** unifies — and #227's fix (canonical `## I<N>:` across every parser/gate + a cross-toolchain guard test, in flight as PR #228, unmerged) is therefore *empirically motivated by this run*, not a hypothetical.

**G3 — Manual triage with no stratification or mechanical auto-close.** The operator hand-read all 26 findings, checked boxes, and maintained the triage log by hand. There was no severity stratification, no batch-accept, and no auto-close of mechanical findings — the bottleneck the mature-repo run also reported (orchestrator-improvements #2 / issues #205–#208).

**G4 — Manual chaining and governance bookkeeping.** The operator manually chained `draft → protected-surface-amend → commit → invariant-coverage-scaffold`; manually wrote the `cli.md` Class-B Governance amendment; manually ticked acceptance-criteria checkboxes; manually updated each doc's `Audit:` / `Status:` line. None of this was automated. This is the entire surface a **Phase 4 run-to-green agent (#218)** is meant to own.

**G5 — No quorum gate on multi-agent review.** byfuglien's and hellebuyck's reviews ran independently and returned ACCEPT-WITH-AMENDMENTS, but nothing *gated* merge on their verdicts — the accept/reject negotiation was manual. A Phase 4 agent must track reviewer verdicts and block merge on unresolved dissent.

**G6 — Deferred-coverage lifecycle untracked.** `cli` I4/I6 shipped as partial coverage with honest "Not covered" notes, but nothing created follow-up tickets or a re-check rule for when the blocking module (runs-state) lands. The ROADMAP item was hand-maintained.

**G7 — No post-draft bidirectional auditor.** #150 escaped because nothing mechanically compared the final invariants against the spec clauses after drafting. The `findings-coverage.md` matrices ([spec § → invariant], [audit-finding → invariant]) and `session.json`'s content hash are exactly the inputs a **Phase 5 auditor (#220)** would ingest to render settled / active / drifted verdicts — but no agent consumes them yet.

## 5. Findings → open ADD work

| Finding | Implication | Issue |
|---|---|---|
| G2 heading divergence | Canonical `## I<N>:` + cross-toolchain guard | **#227 — fix in flight (PR #228, unmerged)** |
| G1 cold-elicitation #150 | Greenfield mode: spec-diff gate before elicitation; spec preferred over operator answers; placeholder-skeleton = "fill, don't overwrite" | **#219** |
| (modes in practice) | `secrets` was `bootstrap` (retrofit); the spec-driven modules are `add`; the repo is `transitional`. ngst is a live transitional repo — the mode taxonomy maps cleanly | **#219** |
| G3 manual triage | Severity stratification + mechanical-finding auto-close + pre-populated triage skeleton | **#218** (acceptance behaviours #205–#208) |
| G4 manual chaining/amendment | Consume the triage log → generate per-decision PRs; run the coverage gate **early** (catch module-derivation); auto-emit the protected-surface-amendment block; auto-tick verifiable acceptance criteria | **#218** |
| G5 no review quorum | Track byfuglien/hellebuyck verdicts; block merge on dissent | **#218** |
| G6 deferred-coverage lifecycle | Parse "Not covered" → follow-up tickets + re-check rule keyed to the blocking module | **#218** |
| G7 no auditor | Ingest the coverage matrices + `session.json` hash + JOURNAL `Type:` lines → settled/active/drifted verdicts; #150 is the symptom an auditor cures | **#220** |

The `cli`-module session is, in effect, **a hand-run of the Phase 4 loop**: it is the worked template `agents/lowry.md` should mechanise. The `add-orchestrator` session validates the *front half* (spec → triaged invariants); the gap is the *back half* (triaged invariants → green implementation under governance).

## 6. Assessment: is v3 validated enough to commit a canonical methodology?

**The shape is validated; the automation is not yet complete.** The greenfield run demonstrates that the core ADD loop — spec → parallel invariant draft → batched audit → disciplined triage → invariant-anchored implementation with independent review — *works on a real greenfield codebase and catches real bugs* (V1–V6). Combined with the mature-repo run, both README preconditions are met.

But the same run shows the methodology is still **closing its own gaps**: the back half (Phase 4 run-to-green), greenfield mode, and the Phase 5 auditor are exactly the open children #218/#219/#220, and the heading convention (#227) was only just unified. Committing a single canonical `methodology.md` *now* would document a methodology whose automation is mid-flight — it would lead the implementation rather than follow it, reintroducing the `plausible ≠ correct` gap the conformance oracle exists to catch.

**Recommendation:** keep the methodology deliberately distributed (README + JOURNAL + `orchestrator-improvements.md` + the archived ADRs + this report **are** the methodology record) until the open children land; then write the canonical `docs/add/methodology.md` *describing what shipped*, ratified against both field reports. On this branch the committed ledger (`crosscheck/conformance/claims.json`) still holds `CLAIM-METHODOLOGY-COMMITTED` at `known-gap` ("Never committed"); this report argues it *should move to `reviewed-disclosed`* — but that ledger transition is pending **#229** (which resolves the claim explicitly) — for the sharper reason recorded here: **not "v3 unvalidated," but "v3 core-validated and evolving; the canonical doc should follow the automation, not lead it."**

## 7. Evidence index

- `~/repos/ngst/JOURNAL.md` — the narrative record (bootstrap → secrets → bulk-invariants → cli).
- `~/repos/ngst/.assurance/add-session-59a24679cf243a01/` — `session.json` (marker + hash), `module-map.md`, `glossary.md`, `findings-{coverage,consistency,quality}.md`, `triage-log.md`.
- `~/repos/ngst/docs/invariants/*.md` — 16 module invariant docs (83 invariants).
- `~/repos/ngst/.assurance/spec-adversary/2026-05-12-github-client.md` + tracker.
- `~/repos/ngst/docs/assurance/ROADMAP.md` — Layer-4 active; the 6-layer projection.
- Build sessions: `2ff1cde8`, `2764b31a` (origin / #149-#150), `15f94ff7`, `e11dca2c` (orchestrator fast path), `a108b3a9`, `d6c275da` (cli Phase-4 loop) under `~/.claude/projects/-Users-harry-nicholls-repos-ngst/`.
