# ADD retrospective — findings and methodology v2

**Status:** Draft for Harry to test against.
**Date:** 2026-05-11 (revised same day to integrate `audit-against-v2.md`; revised same day again to replace the trailer-based audit trail with the sharded-journal architecture).
**Supersedes:** the prior retrospective findings (those served as a hypothesis to verify against transcripts; the verification produced this).
**Incorporates:** `audit-against-v2.md` — a separate agent's stress-test of v2's original draft against a wider session corpus (piano-play, xylem, ev.shapes, formal-verify, ngst). Five revisions actioned: §3.2 ID calibration, new §3.5a greenfield/spec-unread variant, new §3.7 adversarial review on demand, new §3.8 artifact location and lifecycle, new §4.9 greenfield as the ADD origin case. Section §5.5 gained a failure-mode bullet.
**Third revision (same day, evening):** Harry pushed back on git trailers as the audit-trail surface — trailers are queryable but not human-readable, and a single living spec doc goes stale as projects accumulate many specs. §3.3 was rewritten around three durability layers (invariants / ephemeral specs / narrative journal). §3.4 was replaced with a layered enforcement plan. §3.6, §3.8, §4.3, §5.1–§5.5 updated to follow. The diff-classification taxonomy survives but as journal-entry types rather than commit-trailer values. The decision to use sharded `JOURNAL.md` co-located in the tree, walked up from the edit point per a root `AGENTS.md` rule, was made against research into Karpathy's LLM-wiki concept and the AGENTS.md cross-runtime standard.

This is a working set, not a settled answer. Treat each claim below as something to push back on, refine, or throw out. The prior version was revised three times across one retrospective; this one should expect the same.

---

## 1. State of work

The Phase 1 ADD seed shipped via #152 and the Phase 2 self-validation outputs via #154 are on `main` (commits `622efdb`, `82cd362`). The artifact stack contains:

- One intent doc, one methodology doc, one glossary, one acceptance doc.
- Six numbered ADRs, one INDEX.
- One architectural spec, one behavioral spec.
- Six per-module specs (M1–M6).
- One 250+ line phase-2 seam-validation report bucketed into A/B/C findings.

Every artifact carries a Status block (`Attested v1.0 → v1.1`), prior-attestation timestamps, consumes/produces metadata, and cascade-trigger annotations. Phase 2 surfaced 14 severe/medium defects in Bucket A (e.g., FP-tracker env-var drift from `/intent-check`, missing carve-out scan, missing two-section back-translator structure) and 4 architectural gaps in Bucket B (missing skill adaptations for `/assurance-status`, `/assurance-roadmap-check`, `/protected-surface-amend`, plus Lean-pipeline seam). The user attested v1.1, then surfaced the criticism that triggered this retrospective.

ADR-006 was added inside the re-attestation cycle to resolve a contradiction the rest of the stack created: M2/I1 demanded human-authored attestation commits, but both real attestations were agent-authored under explicit in-session human authorisation. ADR-006 reframes the human signal as PR approval rather than commit authorship. The contradiction it resolves is a self-inflicted one — born of the stack's own ceremony.

The substantive design discoveries on this branch (M2 reusing `/intent-check`'s pipeline; deterministic-only instrumentation; five-class diff classification; PR-merge as the human signal; broader skill-adaptation list) are sound. The *form* the discoveries are recorded in is what broke.

**Why ADD exists in the first place.** The originating case is `ngst` session `2ff1cde8-3ff5-4890-a59a-8107f4978c0e` (2026-05-08/09), documented in `~/repos/ngst/field-reports/crosscheck--2ff1cde8-3ff--2026-05-09.md`. Harry had authored a 1,362-line RFC-2119 spec (`docs/design/fabricator-spec.md`) and decomposed it into 70+ GitHub issues. When he ran `/draft-invariants dispatcher`, the skill issued 9 sequential AskUserQuestion prompts before reading the spec — and §14.3 of that spec already contained one-step-derivable answers to 5 of the 6 substantive contract questions. The friction was not greenfield (the spec existed) and not Crosscheck per se (`/draft-invariants` is a separate user skill), but the assurance workflow was perceived as unitary and the complaint reached the Crosscheck team. ADD was commissioned to solve this class of problem: assurance machinery that runs without flattening existing user context. See §4.9 for the scope-of-applicability gap this exposes in v2's working pattern.

---

## 2. What Harry's normal process actually looks like

Five transcripts were available — webhook spec (40 turns, ~20h), CKG-initial-spec (18 turns, ~8h), CGV-system-design (17 turns, ~2h), CKG-design-refinement (10 turns), CKG-MVP-spec (2 turns, ~2 min). The prior retrospective had only the MVP one. The new evidence both confirms and stretches the earlier picture.

### 2.1 Process vocabulary: absent. Confirmed across all five.

Searches for `diff classification`, `ratification`, `seam validation`, `propagated-discovery`, `attestation`, `Last-attested`, `IC#`, `S#`, `F#`, `M#`, `Bucket A/B` across every assistant turn in every transcript → **zero hits**. The agent never reaches for the methodology vocabulary that the broken-form ADD relies on. What it uses instead:

- Domain vocabulary: "dispatcher," "canceller," "watcher," "scope," "concurrency key," "failure budget," "loop guard," "provenance layer," "compaction," "structural edges," "soundness theorem," "trusted-not-proved."
- RFC 2119 keywords (MUST / MUST NOT / SHOULD / MAY) — but only in the webhook spec, and only after Harry asked for "verified information only, no conjecture." The agent reflects RFC 2119 register back when Harry imposes it; it does not originate the register.
- Formal-methods vocabulary (TLA+ INV-1, LIVE-1, `OneActivePerKey`) when discussing formal models — scoped strictly to the model, not used as governance metaphor.
- Ordinal markers ("the biggest assumption to pressure-test is #1," "section §7A.6") rather than abbreviated IDs.

The closest thing to a process-shaped token across all five transcripts is `Status: Draft` in the front-matter of artifacts. That's it.

### 2.2 Artifact form

Variable, not standardised. Three distinct shapes appear depending on what Harry asked for:

| Session | Sections | RFC 2119 | Formal model | Glossary | Non-goals | Assumptions |
|---|---|---|---|---|---|---|
| Webhook spec | 14 numbered | Yes (Harry imposed) | TLA+ appendix | Yes | "Out of scope (v1)" | "Verified facts and open assumptions" |
| CGV system design | 8 numbered | No | Trust-model tags inherited from upstream | No | Implicit | Inline as Q&A |
| CKG initial design | 6 numbered + companion doc | No | None | No | "Non-Goals" section | "Assumptions" section |
| CKG design refinement | "Section-by-section changes" diff style | No | None | No | Folded into "future work" | Implicit |
| CKG MVP | 11 numbered | No | None | No | "Out" subsection | Implicit |

The common skeleton across all of them: front-matter (Date / Status / Author / sometimes "Based on" or "Companion"), then numbered sections in descriptive titles, then a risks/tradeoffs table or section at the back, then either "Out of scope" or "Open questions" or "Future work." Markdown throughout. ASCII diagrams where useful. Code blocks for type signatures or worked examples.

What's *consistently absent* in every artifact:
- Per-section status fields.
- Multiple simultaneous ID schemes organising internal partition of the doc (IC#/S#/F#/M#/B# style). The wider corpus does use IDs — `OQ-1`, `I1`, `C-1`, `SPEC-01` — but always single-scheme and anchored to an external referent (open question with resolution column, property test, GitHub issue). See §3.3 for the calibration.
- Consumes/produces metadata blocks.
- Diff-classification labels in the artifact body.
- Last-attested timestamps.
- Anything that reads like governance ceremony.

The retrospective's prior assumption that Harry wants "a single living spec doc" was *too narrow*. He wants a doc that fits the task. The webhook spec needed RFC 2119 and a TLA+ appendix because he was handing it to another agent for issue decomposition. The CKG MVP was a 116-character "help me write an MVP" prompt and got a tight self-contained product spec. What's invariant is the absence of process metadata in the reader-facing surface, not the section count.

### 2.3 Tension surfacing — by register and by timing

Tensions surface in *every* assistant turn across long sessions. The pattern in the MVP transcript (one closing observation at end-of-turn) holds; long sessions add two more loci.

**Locus A — mid-response, inline.** In the webhook spec, the agent interrupts its own narrative to flag tensions: *"the concurrency key is doing two jobs that should be separate. Job 1: serialization key... Job 2: cancellation scope... These are two separate things."* In CKG-refinement, the agent inserts four numbered tensions inside a six-section critique (msg 1, §§1/2/4/5). Tensions are surfaced where they apply, bolded, often paired with a proposed resolution.

**Locus B — closing observation at end of turn.** Present in every assistant turn across every transcript. CKG-MVP closes with *"One thing I'd flag…"* — exactly one product-shaped caveat. CKG-refinement msg 1 closes with *"The main new insight is that plan files are a free lunch for extraction quality, and the main design risk is over-investing in sibling session machinery that produces marginal returns."* Same register, same hedging stems ("I'd flag," "the main design risk is," "we should be cautious about generalising").

**Locus C — closing as session-boundary question with embedded nudge.** Unique to multi-turn sessions. CKG-refinement msg 7: *"One thing worth flagging: the MVP spec probably needs a corresponding update… Want me to draft the MVP spec update too?"* — the integrity finding (the prior MVP is now stale) is embedded in a next-step question. The webhook session has a clean version of this almost every turn ("Want me to sketch the actual worker main loop pseudocode next, or look at something else…").

Register across all three loci is identical: first-person, hedged ("may," "I'd flag," "the practical question is"), product-shaped, never process-shaped. The agent does not wait for an opening — it surfaces tensions on its own turn boundary, with or without prompting. **This partially answers the prior worry about silent decay**: the agent's natural inclination is to surface frequently, not stay silent.

### 2.4 Pushback

The webhook session shows substantial pushback by the agent — far more than the prior retrospective inferred. The agent named alternatives, recommended one, gave reasons:

- "Why dropping is the right default" + three reasons (webhook msg 17). Harry: "Agree, let's not queue-and-release."
- "I'd recommend revisiting the deployment shape: **OPA as a colocated sidecar with HTTP-over-loopback**, not WASM-in-process" (msg 21). Harry counter: *"I'd actually prefer to write custom code in Go rather than TypeScript anyway. Does that change the OPA deployment issue or not?"* — Harry shifted the constraint rather than directly accepting or rejecting.
- "AI assistance reduces FV costs by maybe 30–50% for the proof part… Helpful, not transformative for this scale of system" (msg 25 / 27). Harry escalated with the same prompt + harder constraints ("You must perform research before answering. You must verify all your claims before answering").

The CKG sessions show milder pushback (the agent argues against its own enthusiasm: *"we should be cautious about generalising these findings to session transcript value overall"*). Pushback exists but is consistently hedged with reasons; the agent does not simply comply.

Harry's pattern in response: either accepts directly, counters by changing the constraint, or escalates by re-issuing the prompt with explicit "verify your claims, no conjecture" constraints. Pushback works because Harry treats it as legitimate; the agent doesn't get punished for naming a problem.

### 2.5 Harry's prompt shape — what to expect

Numbered, multi-item, section-referenced, declarative. Examples:

> "Re: 2.1 Components, can we formally verify these components? At the very least they must be written in a strongly typed language like Rust or Go. re: 2.4 Defaults table, I assume this will be generated by the system. re: 2.5 Function contract extraction, we don't have many (possibly 0) docstring `requires`/`ensures` clauses. Can we perform AST analysis on the function body to extract constraints?" (CGV msg 11.)

> "Couple notes: 1. Plans are in `~/.claude/plans/` not `~/.claude/projects/<path>/` 2. The session in which a plan was created will contain a lot of vital information about the intent, developer reasoning, back and forth iteration, and context from the broader project. We should find these sessions too." (CKG-refinement msg 2.)

Recurring meta-instructions across all five transcripts:
1. **"Think about my intention / what I really need before you start."** (Webhook msgs 0, 18, 30, 38.)
2. **"Verify your claims; no conjecture."** (Webhook msgs 18, 26; CGV msgs 0, 2.)
3. **"Don't bias / don't include history; just the final design."** (Webhook msgs 18, 38.)
4. **"All assumptions must be documented and verified before finishing."** (Webhook msg 18; CGV msg 2.)

These are Harry's stable prompting idioms. Treat them as durable preferences.

Harry's prompts trend monotonically shorter as a session progresses (CKG-refinement: 201 → 302 → 40 → 48 → 118 chars). Late corrections are "Yes," "Continue," "All of these are valid. Particularly #1." Once Harry has set the constraints, his role becomes gating rather than instructing.

### 2.6 Silent context maintenance — what's actually maintained

Two distinct mechanisms:

**Files as the working memory.** The agent re-reads project files before re-drafting (CKG-refinement msg 7 re-opens `code-knowledge-graph-design-update.md` at lines 1–40 and 400–756, msg 9 re-opens `code-knowledge-graph-mvp.md`). The agent's "model" is the file, not a hidden scratchpad. There is no externalised process artifact (no ledger, no IC#-tracking file, no roadmap doc that surfaces unannounced).

**Cross-session retrieval where the platform supports it.** CGV-system-design msg 3 invokes `conversation_search` and `recent_chats` to recover the prior user-stories conversation. The agent surfaces this transparently: *"I've now reviewed the prior conversations. Most of my original questions are answered by context from those discussions. Let me separate what I now know from what's still genuinely open."*

What is *not* maintained: no notes file, no internal ledger that surfaces later, no "I have noted" / "I'm tracking" phrasing in any transcript. The agent's silent state is the file state plus the conversation context plus retrievable prior conversations. **This matters for ADD**: the audit trail does not need to be a parallel process artifact maintained by the agent — git plus the spec doc plus PR descriptions already are the trail.

### 2.7 Iteration shape

Not "one big draft at session end." Mixed:

- **Webhook spec:** ~18 turns of design discussion, then a single 1,100-line `create_file` in msg 37. No iteration on the markdown afterwards.
- **CGV system design:** three turns of Q&A → comprehensive single draft → targeted `str_replace` patches.
- **CKG initial:** draft → many `str_replace` operations across the file (Python→Go vocabulary shift, MCP→CLI swap). Drift accumulates; agent has to grep for missed references.
- **CKG refinement:** new artifacts (Design Update 2; MVP Changes addendum) delivered one-shot, but explicitly structured as section-by-section diffs against prior docs.

The MVP transcript (2 turns) is the simplest case of a much broader pattern: *Harry sets constraints up front, the agent reads context silently, the agent produces an artifact in Harry's style, the agent closes with one or two forward-looking product observations*. The longer sessions add iteration cycles — but each iteration cycle is a *patch* applied to the living artifact, not a re-draft. In-place edits over hours and across session boundaries.

The prior retrospective's intuition that Harry wants in-line edits, one source of truth, git for history is **correct**. The pattern is observable in every transcript.

### 2.8 Closing the "calibration of silence" worry — partially

The prior retrospective worried that discipline lives in the agent's head and there's no external signal that calibrates it. The transcripts show: the agent surfaces tensions in **every turn** of long sessions, at three distinct loci, in consistent product-shaped register. The risk of "agent never nudges" is low — surfacing is the agent's default.

What the transcripts do *not* answer: whether the agent's *judgement of what counts as a tension worth surfacing* is well-calibrated. The agent might be surfacing low-importance design choices and missing high-importance integrity gaps. The mid-response tensions in CKG-refinement are real and load-bearing (e.g., "the design didn't anticipate that the venv path was the primary differentiator"). The webhook tensions are real and load-bearing (e.g., the concurrency key conflating two scopes). On the available evidence, the agent's tension-detection looks credible. But calibration of judgment is not the same as calibration of cadence, and the transcripts cannot certify the former — only that the cadence is healthy.

---

## 3. Methodology adjustments

The substantive ADD machinery worth keeping is small. The visibility flip is the design.

### 3.1 What survives

- **The diff-classification taxonomy** (propagated-discovery / intent-refinement / drift / retraction / status-transition). Useful as the **type values for journal entries** under the new architecture (§3.3) — five classes, each entry carries one. Same taxonomy, different surface (was: commit trailers; now: human-readable JOURNAL.md entries).
- **PR-merge as the human ratification signal** (the ADR-006 substance, minus the ceremony of ADR-006 itself).
- **Deterministic-only rule for instrumentation.** Any signal-detection tool runs with zero LLM in the loop. LLM judgments consume signals; they don't compute them.
- **M2's prose-vs-prose pipeline reusing `/intent-check` verbatim** — substitutions table, two-section back-translator, mandatory carve-out scan, fail-closed validation, content-hashed attestation. The seam-validation work caught this gap and it's a real one.
- **The skill-adaptation list** — `/assurance-status`, `/assurance-roadmap-check`, `/protected-surface-amend` need ADD-mode awareness, not just the original five.
- **The auditor-as-separate-agent principle** — the agent that authored an artifact should not audit it.

### 3.2 What's thrown out

- Ten separate artifacts (intent + methodology + glossary + ADRs + architectural + behavioral + 6 module specs).
- **Five simultaneous internal-partition ID schemes** (IC# *and* S# *and* F# *and* M# *and* B#) layered on a single project, each one organising the doc's own structure rather than referring out. Single-scheme IDs that anchor to an external referent are preserved — see §3.3 for the calibration.
- Per-artifact Status / Last-attested / Phase / Consumes / Produces frontmatter blocks.
- Bucketed seam-validation reports as a user-facing deliverable.
- "Attested v1.0 → re-attested v1.1" cascade ceremony.
- The "Phase 2 as a separately gated stage" model.
- The methodology doc as a thing humans read.

These artifacts may be preserved as historical record under `.retrospective/`, but they leave the human's reading surface entirely. Their substantive content is either (a) preserved as project-level ADRs at `docs/decisions/<NNNN>-<slug>.md` if cross-shard, (b) absorbed into the relevant module's invariants, or (c) recorded as the introducing journal entries when v3 is rolled out (see §3.3).

### 3.3 The shape of the new methodology

**Three durability layers.** Each layer answers a different question about lifespan and audience.

| Layer | Lifespan | Maintenance | Where it lives |
|---|---|---|---|
| **Invariants** | Durable, load-bearing | Stays in sync with property tests via the existing coverage gate | `docs/invariants/<module>.md` |
| **Specs and plans** | Ephemeral snapshots, dated | Never maintained — they age into history | `docs/specs/`, `docs/plans/`, or wherever they belong |
| **Narrative journal** | Durable, human- and agent-readable | Append-only after merge to main; freely editable in-PR | Sharded `JOURNAL.md` co-located in the tree |

This replaces v2's prior "single living spec doc per project" model, which Harry rejected: specs go stale, projects accumulate many specs and plans over time, and the user shouldn't be doing maintenance to keep one doc canonical (that's a process smell). The *source of truth that feeds into design* is the invariants layer plus the journal — not any individual spec.

**Invariants layer (durable, load-bearing).** Unchanged from current Crosscheck practice. `docs/invariants/<module>.md` anchored to property tests via the `// Invariant <ID>:` convention and the coverage gate. Invariants drift slower than specs because they are load-bearing across many features and the gate enforces sync with tests. This is what survives across the lifespan of the project; specs come and go around it.

**Specs and plans layer (ephemeral snapshots).** Each spec is "design intent as of the date it was committed." Committed once, dated in the front-matter, then left alone. Older specs accumulate as history; nobody reads them expecting current truth. They are reference artifacts for the moment they captured, not living documents. The webhook spec, the CKG MVP, the CGV system design from §2 are good shapes — they were written once and never re-attested.

**Narrative journal layer (durable, append-only).** Sharded `JOURNAL.md` files co-located at semantic boundaries in the directory tree (e.g. `crosscheck/JOURNAL.md`, `crosscheck/skills/JOURNAL.md`, `crosscheck/mcp-server/JOURNAL.md`). Not every directory — only meaningful units of design. Each entry has the shape:

```
## 2026-05-11 — ADD methodology v2 → v3 [ADR-007]

**Type:** intent-refinement
**Touches:** specs/findings-and-methodology-v2.md (revised), invariants/ (none)
**Why:** v2's git-trailers-as-audit-trail isn't human-readable. v3 introduces narrative journal at three durability layers.
**Links:** [ADR-007](../docs/decisions/0007-sharded-journal.md), [v2 retrospective](.retrospective/findings-and-methodology-v2.md)

Brief paragraph in product register explaining what was decided and why anyone reading this in six months should care.
```

Type values are the diff-classification taxonomy from §3.1 (propagated-discovery / intent-refinement / drift / retraction / status-transition). Newest entries first within each file. Append-only after merge: post-merge edits are themselves new entries that supersede earlier ones via a `Supersedes:` link. Pre-merge, freely editable.

**The walk-up rule lives in a root `AGENTS.md`.** Single instruction: *before making non-trivial changes in any directory, walk up to the repo root reading every `JOURNAL.md` you encounter*. AGENTS.md is the cross-runtime standard stewarded by the Linux Foundation's Agentic AI Foundation since mid-2025; it is honored by Cursor, Codex, Claude Code, Copilot, Devin, Windsurf, and Gemini CLI with closest-file precedence. This gives portable, narrative routing without per-tool rule files (Cursor `.cursor/rules/*.mdc`, Copilot `.github/instructions/*.instructions.md`) and without inventing our own loader. Symlink or stub `CLAUDE.md → AGENTS.md` if any tooling still requires the legacy filename.

**ADRs return for cross-shard decisions.** A sharded journal layout means decisions can span multiple shards. ADRs at `docs/decisions/<NNNN>-<slug>.md` are the canonical document for any cross-cutting decision; every JOURNAL.md whose shard the ADR touches gets a one-line entry pointing at it. The journal entry is a pointer; the ADR has the longer rationale. Treat ADRs as cheap documentation, not as governance ceremony — anything cross-cutting earns one. (This is *not* a return of the v1 ADRs, which were local-to-the-stack and ceremony-laden; v3 ADRs are project-level decisions linked from journal pointers.)

**IDs in the spec doc are fine when they anchor to an external referent.** A GitHub issue, an open-question row with a resolution column, a property test (e.g. `// Invariant I1: ...`), or another spec doc all qualify. The v1 ADD failure was *five simultaneous schemes* (IC#/S#/F#/M#/B#) each organising the doc's own internal partition — that is the form to avoid. One scheme that points outward is the working form. Concrete corpus evidence: `piano-play/docs/PRD.md` uses `OQ-1`…`OQ-9` anchored to an Open Questions table with resolution columns; `xylem/docs/plans/sota-gap-implementation-2026-04-11.md` uses `C-1`/`G-1`/`H-1` umbrellas anchored to GitHub issue templates; `ev.shapes/docs/invariants/*.md` uses `I1`/`I2`/… anchored to property tests via the `// Invariant <ID>:` convention and the coverage gate. **Invariant IDs are preserved as-is** — they are load-bearing for the existing invariant-coverage gate and Harry intends to keep using them.

**Git is the history.** No addendum files unless the document is explicitly an addendum to a prior shipped doc (CKG-refinement's Design Update 2 pattern is fine — it's a deliberate diff against a published baseline, not a process artifact). `git log` / `git diff` / `git blame` carry the version trail; the journal carries the *narrative* trail. No version-bump comments inside the doc.

**Conversation looks like a normal spec session.**
- Agent reads context silently (project files, prior conversations where available, every JOURNAL.md walking up from the file in scope).
- Agent surfaces tensions in three loci: mid-response inside numbered structure, at end of turn as a single product caveat, and as session-boundary questions with embedded nudges. Register is always product-shaped, first-person, hedged.
- Agent pushes back substantively with reasons when Harry's framing seems off. Names alternatives, recommends one.
- Agent delivers artifacts via `create_file` for new docs and targeted in-place edits for revisions. Re-reads the file before re-drafting. Adds the journal entry as part of the same PR — entry-writing is part of the deliverable, not a follow-up step.
- Agent never reaches for IC#, Bucket A, attestation, ratification, Last-attested, seam validation, propagated-discovery, or any other ADD-shaped vocabulary in the conversation surface. (Type values like `propagated-discovery` are fine *inside* a JOURNAL.md frontmatter line — that's where they belong; they should not appear in chat prose.) If it catches itself reaching, it's a smell — switch register.

**Ratification = PR approval and merge.** Not a separate ceremony. The PR description summarises the journal entries the PR introduces; reviewers read the entries to understand what's being locked in. Approval-and-merge is the human signal; merge is also the moment those journal entries become append-only. Git provides the anchor — the merged diff defines what was committed.

**Status field on artifacts is minimal.** Two values:

- `Status: Draft` — being iterated.
- `Status: Snapshot` — committed. A spec is a snapshot of intent at the date it was committed; it does not get re-ratified later. If the design changes, a *new* snapshot is committed and a new journal entry supersedes the old one.

That's it. No per-section status, no per-claim attestation, no version numbers in the doc (git tags handle that if needed).

**Integrity nudges in product language.** When the agent finds a gap, contradiction, or unresolved tension, it surfaces as a normal review remark inline or at turn close: *"§4.5 says commit-triggered but §7.2 implies PR-triggered; which is right?"* Not as a structured findings list with severity ratings.

**Self-checks against silence.** The agent periodically audits its own state. Three minimum checks:

1. Has the PR touched directories whose `JOURNAL.md` was not also edited in this PR? (Run `git diff --name-only main...HEAD` and check, per touched directory, whether the corresponding `JOURNAL.md` was modified.) Soft signal — not every change needs a journal entry, but a non-trivial change without one warrants a question.
2. Do journal entries follow the shape (date / type / why / links)? Lint pass — see §3.4.
3. Are there spec snapshots whose claims contradict more recent journal entries in the same shard? Hardest of the three; partially automatable by date-ordering plus keyword overlap; otherwise relies on the agent's integrity loop.

When checks surface anomalies, surface them as a closing-observation product caveat: *"One thing worth flagging: this PR touches `crosscheck/skills/spec-iterate/` but doesn't add a JOURNAL.md entry there. The diff is non-trivial — want me to draft an entry, or is this pure refactor?"* Same register as every other closing observation.

### 3.4 Enforcement layering

The walk-up rule (§3.3) needs something to actually fire it. Build the enforcement stack in this order — each layer is cheaper than the next, and later layers are added only when the earlier ones empirically fail to do the job.

1. **Root `AGENTS.md` instruction.** Free, immediately portable across every major agent harness via the cross-runtime AGENTS.md standard. The walk-up rule lives here. Symlink or stub `CLAUDE.md → AGENTS.md` if any tooling still requires it. **This is the v1 mechanism.**
2. **Crosscheck `/journal-context` skill.** Deterministic walk of the tree from a given path; dumps every `JOURNAL.md` it finds in walk-up order. Agents can be told (in `AGENTS.md` or in skill prompts) to invoke it before substantial design moves; humans can run it for orientation. No LLM in the walk; LLMs consume the output. **Build alongside v1.**
3. **Pre-commit hook.** Detects when a commit touches a directory whose `JOURNAL.md` was not also edited in the same PR. Soft warning, not a block: *"this commit touches `crosscheck/skills/`; is it worth a journal entry, or is it pure refactor?"* Mirrors the existing dual-track pattern in `/assurance-init`. **Add when usage shows where drift creeps in.**
4. **Harness-level hook.** Claude Code post-tool hook, equivalent for other harnesses. Only build if (1)–(3) empirically fail to fire reliably. Not v1.

**Lint passes** (Karpathy LLM-wiki style — see the research note in the third-revision header). Periodic walk of all `JOURNAL.md` files checking for: malformed frontmatter, journal entries that reference missing specs/ADRs, orphan `Supersedes:` links, contradictions between recent entries in the same shard. Likely a Crosscheck skill (e.g. `/journal-lint`). Build when the journal grows large enough that contradictions become plausible — not before.

**Why not Cursor `.cursor/rules/*.mdc` with `globs:` or Copilot `.github/instructions/*.instructions.md` with `applyTo:`.** Both support glob-routed rules and could in principle scope per-directory journal-reading instructions, but they are tool-specific. The cross-runtime AGENTS.md hierarchy (closest-file precedence) gives us the same routing semantics with portability across all major harnesses. We can add per-tool rule files later if a specific tool needs sharper control, but the v1 architecture does not depend on them.

### 3.5 What the agent does at session start, by default

The pattern from the transcripts (mature-repo case — `docs/`, prior conversations, invariant docs, ROADMAP all exist):

1. **Read available context silently.** Project files relevant to the request. Prior conversations if the platform supports retrieval and they exist. **If the request names a module and a spec doc exists, read the spec before issuing any clarification questions.** This is the ngst lesson: §14.3 of `fabricator-spec.md` contained one-step-derivable answers to 5 of 6 contract questions; the skill issued them cold anyway. v2's default reads first.
2. **If the user has not asked for an artifact yet, ask 1–3 structured questions.** Not "what do you want?" — specific decision points the agent needs answered to produce a useful draft. CGV system-design's opening turn is the model: *"Before drafting the system design, I have questions and assumptions that need your input."* If context already answers a question, do not ask it; surface a pre-filled candidate for red-pen confirmation instead.
3. **If the user has asked for an artifact, produce one.** Single comprehensive draft. Match the user's voice. Close with one or two forward-looking observations.
4. **Maintain durable state in files, not in scratchpad.** Re-read the file before any re-draft.

### 3.5a Greenfield and spec-unread variants

The pattern in §3.5 assumes either project files exist (the §3.5 mature-repo case) or context has been retrieved from prior conversations. There are two cases where that assumption breaks. Both need explicit handling because both reproduce the failure mode that motivated ADD.

**Case A — true greenfield.** No `docs/`, no `README.md`, no prior spec, no prior conversations. The agent does not default into the fixed-cadence elicitation interview that broke `/draft-invariants`:

- Ask **one** open-ended question about the project's purpose and intended audience. Not a structured multi-question interview.
- From the answer, produce a single short draft (`Status: Draft`). The draft is the elicitation device — Harry red-pens it rather than answering questions cold. This matches his late-session pattern in the existing corpus where prompts collapse to *"Yes,"* *"Continue,"* *"All of these are valid. Particularly #1"* — once a concrete artifact is on the page, his role becomes gating, not instructing.
- Resist importing conventions from `ev.shapes`/`xylem`/`formal-verify` by inertia. Those are mature-repo conventions; greenfield does not need invariant IDs, a ROADMAP, or a protected-surfaces partition on day one. They can arrive when they earn their place.

**Case B — externalised-but-unread spec (the ngst case).** Context exists but is not in the standard locations the skill happens to glob. Or context exists and the skill's "elicit before reading implementation" rule mis-classifies a written prose spec as implementation:

- Before any clarification question, glob for `docs/design/*spec*.md`, `docs/specs/*.md`, `docs/*.md`, files with RFC 2119 keyword density above threshold, and (if found) read the body.
- If candidate answers exist in the spec, pre-fill them and ask the user to red-pen / confirm. Quote the source — e.g. *"§3.2 line 106 says X; is that still the responsibility, or has it moved?"* — rather than offering invented rephrasings as multi-select options.
- Audit-finding tables (the ngst spec's §14.3 catastrophe map) are a first-class catastrophe corpus, not a fallback. Mine them directly.

**Default discipline.** Both cases share the same rule: the agent never issues a fixed-cadence interview when there is a cheaper discovery move available. Read first; elicit only what reading cannot tell you.

### 3.6 What the agent does *not* do

- Translate the conversation into IC#/S#/F#/M# IDs anywhere visible. (Single outward-anchored ID schemes are fine — see §3.3.)
- Produce a methodology doc for the user to read.
- Schedule "Phase 2 validation" as a separate ceremony. Validation happens inline as the conversation continues — the agent's own integrity-nudge loop.
- Wait for the user to ask "what could go wrong?" before surfacing tensions.
- Re-draft from memory. Always re-read the file first.
- Negotiate the journal-entry classification with the user. Write the entry. If the classification is ambiguous, surface the ambiguity as a product question (*"this looks like both intent-refinement and propagated-discovery — I'll classify it as intent-refinement and note the discovery context in the body, unless you'd prefer the other framing"*).
- Issue a fixed-cadence interview when project files or a prose spec already answer most of the questions (see §3.5a).
- Spawn adversarial reviewer subagents without being asked. Adversarial review is a Harry-initiated pattern — see §3.7.

### 3.7 Adversarial review on demand

There is a productive pattern in the wider corpus that v2's three-loci integrity surfacing (§2.3) does not cover. Session `4d7542e6-d0de-4431-aa5d-c857014811f8` (formal-verify, 2026-05-05): Harry prompts *"Use a couple of subagents to adversarially review your proposal. Then refine it based on their output."* The agent dispatches two reviewers in parallel — one pressure-tests sequencing and dependency logic, one pressure-tests agent-implementability — absorbs the outputs, and refines the proposal as targeted in-place edits to the existing artifact.

This is a *solicited* surfacing mechanism, distinct from the three loci in §2.3 (unsolicited surfacing on the agent's own turn boundary). Stronger signal because the reviewers come fresh to the artifact; demonstrably productive; explicitly invoked by Harry. It should not be folded into §2.3's closing-observation cadence.

**Working shape.**

- Triggered only when Harry explicitly asks for adversarial review. The agent does not spawn reviewers spontaneously — that turns subagent dispatch into ceremony and burns tokens for low signal.
- Dispatch is parallel, not sequential. Each reviewer gets a focused mandate (e.g. "sequencing and dependency logic," "agent-implementability," "hidden assumptions," "what does this not protect against?"). Mandates are scoped to one concern each so outputs are comparable and synthesisable.
- Outputs are absorbed into the existing spec doc as targeted in-place edits, not produced as a separate findings document. (A separate findings doc is the failure mode — see §1 on the Phase-2 seam-validation report.)
- The agent reports what changed in product language at turn close: *"§4.5 tightened after the sequencing reviewer flagged that the canceller can race the watcher; the §7 dependency on `concurrency_key` was removed because the implementability reviewer found three callers couldn't compute it without a database round-trip."*

The audit author recommends dumping session `4d7542e6` before refining §3.7 further. Open question on whether any specific subagent-instruction shape is worth preserving as a reusable template, or whether the per-session focus-area improvisation is sufficient.

### 3.8 Where ADD applies — artifact location and lifecycle

v3's audit trail (sharded `JOURNAL.md` files, PR-merge ratification) presumes the spec doc is under version control in a repo with a working `git log` and an in-tree journal shard. Harry's actual spec artifacts live in four distinct locations with different lifecycles:

| Location | Examples | Lifecycle | ADD applies? |
|---|---|---|---|
| `docs/` in a committed repo | `piano-play/docs/PRD.md`, ev.shapes design docs, `ngst/docs/design/fabricator-spec.md` | Versioned, PR-reviewed | Yes — full journal-entry + ratification discipline |
| `.idea/` (JetBrains scratchpad) | `ev.shapes/.idea/Production-Readiness-Plan.md` | Sometimes committed, sometimes not | Only once committed to a tracked path |
| `~/.claude/plans/` | `agile-nibbling-peacock.md`, `immutable-giggling-gem.md` | Per-user, ephemeral, never in git | No |
| `/tmp/claude-*` | Worker-generated issue files | Truly ephemeral | No |

**Scope statement.** ADD's audit trail requires the spec doc to be under version control in a repo with a working `git log`. Specs that begin in `.idea/`, `~/.claude/plans/`, or `/tmp/` are *pre-ADD drafting surfaces* — private exploration with no audit-trail load. ADD begins when the spec is committed to a tracked path in a real repo. The migration from drafting surface to tracked path is the moment the spec becomes load-bearing for the project, and that is what the journal should mark.

**Implication for the agent.** When working on a spec in `.idea/`, `~/.claude/plans/`, or `/tmp/`, the agent uses §3.5's pattern *without* writing journal entries — pre-commit drafts have no readers depending on them, so a journal entry is premature ceremony. When the user moves the spec into a tracked path (e.g. `git mv .idea/foo.md docs/specs/foo.md`, or any equivalent introduction of a new tracked spec), the agent flags the transition and writes the **introducing journal entry** in the relevant shard's `JOURNAL.md`: type `propagated-discovery` (the discovery being that the draft is now load-bearing enough to track), one paragraph of why this graduated, links to the spec. If the move is cross-shard, the agent also adds an ADR at `docs/decisions/<NNNN>-<slug>.md` and points the journal entry at it. Don't conflate drafting and tracked surfaces; the graduation is the entry's earning moment.

---

## 4. Worries and open questions

### 4.1 Calibration of cadence — answered (mostly)

The transcripts show three loci of integrity surfacing in long sessions, all in consistent product register. The earlier worry that the agent might stay silent under decay is weakly supported by the evidence — the agent's default is to surface frequently. **Open sub-worry:** the agent might be miscalibrated on *which* tensions are worth surfacing. The evidence available (the tensions surfaced were load-bearing) is suggestive but not conclusive. Mitigation: in production use of ADD-shaped sessions, log every surfaced tension and review periodically against outcomes.

### 4.2 The agent's self-audit discipline — unresolved

If the agent maintains journal entries and is responsible for noticing drift between the spec doc and the code, what stops it from rubber-stamping its own work? The prior retrospective named this as the "calibration of confidence" worry. The transcripts show the agent *does* push back on its own conclusions ("we should be cautious about generalising") and *does* surface its own gaps ("the MVP spec probably needs a corresponding update"). So self-pushback is observable. But it's not a discipline — it's a habit. Open question: should ADD codify a "agent must run self-audit checks before declaring a session complete" gate? Or is the existing closing-observation cadence sufficient? The §3.4 enforcement stack (soft pre-commit warning, `/journal-lint` later) covers the structural-integrity side; what it does not cover is the agent's own honesty about whether the journal entry it just wrote is the *right* entry. That remains a habit, not a check.

### 4.3 What happens when the spec and the code diverge

The audit trail (journal entries, PR-merge ratification) captures spec and decision edits. It does not by itself capture *code* edits that should have triggered a journal entry — the five-class taxonomy includes `drift` for exactly this case, but the §3.4 pre-commit hook only fires when a tracked spec/journal file is touched, not when code-only commits should have prompted a `JOURNAL.md` entry. Detecting unilateral code drift is the hard problem and the transcripts don't solve it. Working hypothesis: rely on PR review (humans read the diff and ask "should `JOURNAL.md` have moved?"), plus the soft pre-commit warning when a touched directory's journal was not also edited, plus periodic `/journal-lint` passes. Don't try to mechanise it harder than that in v1.

### 4.4 RFC 2119 — when

The webhook spec uses RFC 2119 because Harry handed it to a downstream agent for issue decomposition. The CKG specs don't because the audience was Harry himself. Open question: does ADD assume RFC 2119, or treat it as a per-project choice? Recommend: per-project. Surface the question once at session start ("Will this spec be handed to a downstream agent or implemented directly? RFC 2119 helps the former, adds friction to the latter") and record the answer in the doc's audience line.

### 4.5 The Lean-pipeline seam

S2.5 in the broken-form architectural spec declares an `implementation:` enum with values `spec-iterate | lean-pipeline | manual | external`. The substance is good — different specs route to different verification chains. The form (an enum declared in a separate architectural spec file) is the problem. Working shape: the spec doc itself names its intended verification path in prose, in a top-level "Verification approach" section. Example: *"This spec is implemented manually. Acceptance criteria are exercised by integration tests in `test/webhook/`. No formal model beyond the TLA+ appendix."* The agent reads this and routes appropriately.

### 4.6 Multi-person teams — deferred

Every transcript is Harry + agent. No second human reviewer. ADR-006's "approver ≠ PR author" constraint depends on having a second human. Deferred until the team grows. For solo work, "approver = author" is the trivial case; the human signal is the merge button.

### 4.7 The auditor agent — keep separate or fold in?

ADR-003 made the auditor a third agent role peer to byfuglien/hellebuyck. The new methodology has less ceremony for it to enforce — most of the "consolidation pass" work disappears when there's no IC/S/F/M ledger to consolidate. Open question: is the auditor still needed, or does the existing dual byfuglien/hellebuyck structure plus the §3.4 enforcement stack cover it? Lean toward fold-in: byfuglien handles verification-implementation, hellebuyck handles spec-and-governance (including journal-entry shape checks, the walk-up-was-followed audit, and the ratification gate at PR merge). No third agent unless one emerges naturally.

### 4.8 Phase model — gone

The broken-form ADD had Phases 0–5 (intent capture → spec → ratification → implementation → maintenance → retirement). The new methodology doesn't need them. `Status: Draft` / `Status: Snapshot` covers the spec lifecycle (specs are dated artifacts, not living documents — see §3.3). The implementation lifecycle is whatever the codebase already does. Open question: do any of the other Crosscheck skills (`/assurance-init`, `/assurance-status`) depend on the phase model? If so, those dependencies need to come out; the assurance hierarchy can survive without ADD-style phases since it's already governed by Layers 4–6.

### 4.9 Greenfield as the ADD origin case

v2's original five-transcript evidence base was narrow — three CKG-family sessions plus the webhook and CGV designs. All five are *mature-repo freeform design conversations*. The wider corpus audit found a class of sessions v2 cannot have observed: greenfield repos and spec-rich repos where the skill mis-handled existing context. The ngst case (see §1, end note, and `~/repos/ngst/field-reports/crosscheck--2ff1cde8-3ff--2026-05-09.md`) is the canonical example. ADD was commissioned to solve this class of problem. v2's working pattern as originally drafted reproduced it.

The §3.5a additions address the methodology shape. The open questions below are what's left to resolve before implementation can be evaluated against the actual origin case.

**Two failure modes share the same symptom.**

- *True greenfield.* No files to read. The agent's "read context silently" default has nothing to anchor on; the failure mode is either fixed-cadence interview (the `/draft-invariants` pattern) or cargo-culting mature-repo conventions by inertia.
- *Externalised-but-unread.* Files exist; the skill doesn't read them. The agent's "elicit before reading implementation" rule mis-classifies prose specs as implementation and interviews cold against context that already has answers.

§3.5a's discipline ("read first, elicit only what reading cannot tell you") is the working response to both. But the discipline is not yet tested:

**Open questions for implementation to answer.**

1. **What's the threshold at which the agent stops reading and starts asking?** In a 1,362-line spec the agent should read at least the TOC plus §3 (responsibility) and §14 (audit findings) before any AskUserQuestion. In a 50-line draft the agent should read all of it. Is there a clean rule, or is this calibrated per-call?
2. **What's the glob list?** `docs/design/*spec*.md`, `docs/specs/*.md`, `docs/*.md` is a starting set. Does it need to cover `.idea/` (per §3.8)? `~/.claude/plans/` if the user references one? How does the agent know which spec is load-bearing when multiple exist?
3. **Does §3.2's throw-out list survive greenfield?** Per-artifact Status, the phase model, and the methodology doc were all discarded as overhead. In a fresh repo, *some* lightweight Status field may be necessary precisely because there's no other context. The audit raises this as worth re-checking; v2 has not done that work yet.
4. **The journal discipline in greenfield.** A single-commit greenfield spec doc still wants an introducing journal entry in the project root's `JOURNAL.md` (the only shard that exists yet). Type is `propagated-discovery` (the discovery being the spec's existence). Confirm this is the right type and not a sixth value. Open sub-question: at what point does a greenfield repo earn a *second* `JOURNAL.md` shard? Probably the first time a subdirectory acquires meaningful design weight independent of root — but this is calibration to do during the §5.3 greenfield-target run, not in advance.
5. **What does the §5.3 first task look like for a greenfield project?** v2's §5.3 picks a real Crosscheck-internal spec — but that's a mature repo. The harder test of the methodology is a greenfield Crosscheck plugin or a fresh user-skill repo. Should the first task be a *both* test (one mature-repo spec session and one greenfield spec session) before declaring the methodology validated?

This is the highest-leverage gap remaining in v2. The other §4 open questions are calibrations; this one is scope-of-applicability. If the methodology can't be exercised against ngst-style sessions without reproducing the 9-prompt friction band, ADD has not solved the problem it was commissioned to solve.

---

## 5. Handoff for whoever implements

### 5.1 What to throw out

- `crosscheck/docs/add/methodology.md`
- `crosscheck/docs/add/glossary.md`
- `crosscheck/docs/add/intent.md`
- `crosscheck/docs/add/acceptance.md`
- `crosscheck/docs/add/decisions/INDEX.md` and all six ADR files (note: ADRs return in v3 in a *different form* — project-level cross-shard decisions at `docs/decisions/<NNNN>-<slug>.md`, linked from journal pointers; see §3.3. The v1 ADRs being thrown out are the local-to-the-stack ceremony-laden kind, not the new project-level kind.)
- `crosscheck/docs/add/specs/architectural.md`
- `crosscheck/docs/add/specs/behavioral.md`
- `crosscheck/docs/add/specs/modules/M1–M6.md`
- `crosscheck/.assurance/phase-2-seam-validation-20260509T182403Z.md`

Or: keep them in a `.retrospective/` directory as historical record, but remove all references from the live docs surface (no links in any agent prompt, no skill reference, no README mention).

### 5.2 What to keep

- The substantive design discoveries:
  - M2 reuses `/intent-check` verbatim (substitutions table for input shape).
  - Deterministic-only signal computation; LLMs consume signals, never compute them.
  - PR-merge approval as the human ratification signal.
  - Five-class diff-classification taxonomy as **journal-entry type values** (was: trailer values).
  - Skill-adaptation list extended to `/assurance-status`, `/assurance-roadmap-check`, `/protected-surface-amend`.

- The infrastructure pieces worth building first (in this order):
  1. **Root `AGENTS.md`** with the walk-up rule (§3.3, §3.4). Free, immediately portable.
  2. **`/journal-context` Crosscheck skill** — deterministic walk of the tree from a given path, dumps every `JOURNAL.md` in walk-up order. No LLM in the walk. Mirrors the dual-track pattern in `/assurance-init`.
  3. (Later, when usage shows where drift creeps in) the soft pre-commit hook that warns when a touched directory's `JOURNAL.md` wasn't also edited.
  4. (Later, when journals grow) the `/journal-lint` skill for periodic health checks.

### 5.3 First task

Pick **two** real specs to drive against — one mature-repo, one greenfield-or-spec-rich. The asymmetry matters: v2 was originally drafted from mature-repo evidence alone, and §4.9 names the greenfield/ngst case as the unvalidated half of the applicability claim.

- **Mature-repo target.** A Crosscheck-internal spec (the MCP-server spec is a candidate; an upcoming skill spec is a candidate). Tests v2's working pattern in the conditions it was originally derived from.
- **Greenfield-or-spec-rich target.** A fresh Crosscheck plugin, a fresh user-skill repo, or a re-run of the ngst dispatcher session against the actual `fabricator-spec.md`. Tests §3.5a's discipline (read first, elicit only what reading cannot tell you) against the failure mode ADD was commissioned to solve.

**Do not** start by writing the methodology doc — that's what broke the first time. For each target:

1. Drafting a single spec doc, in Harry's voice, with no process metadata visible. `Status: Draft` while in flight; `Status: Snapshot` once committed.
2. Running it through PR review with Harry as approver. Treat merge as ratification. PR description summarises the journal entries the PR introduces.
3. Adding a journal entry to the relevant shard's `JOURNAL.md` in the same PR. Verifying the entry follows the shape (date / type / why / links). If the move is cross-shard, also adding an ADR at `docs/decisions/` and pointing the entry at it.
4. Iterating on the doc once or twice, each iteration as its own PR, each PR adding a follow-up journal entry that supersedes (via `Supersedes:` link) the prior one where appropriate.

The greenfield run is the harder test. If it reproduces a fixed-cadence interview, an inertia-imported invariant ID scheme, or a methodology doc the user has to read, v2 is wrong about §3.5a and needs another revision before the methodology can be declared validated.

Only after both runs succeed, write the methodology doc that describes what you did — and write it for the *agent* (a SKILL.md or an extension to `byfuglien.md` / `hellebuyck.md`), not for the human. The human shouldn't need to read a methodology doc to use ADD; the agent makes ADD invisible.

### 5.4 What success looks like

Harry does a spec session and it feels like the webhook spec session or the CKG-MVP session. He doesn't notice ADD running. Six months later, somebody asks "what was decided about §4 of the webhook spec?" and they walk the tree from the relevant file upward, reading every `JOURNAL.md` they pass — the answer is in narrative form, newest first, in plain product register. Or they run `/journal-context webhook/spec.md` and it dumps the same content for them. Agents touching nearby files do the same walk before making changes, anchored by the root `AGENTS.md` rule. Nobody reads `docs/add/methodology.md` to do their job, because there isn't one — or if there is one, only agents read it.

### 5.5 What failure looks like (so we can catch it early)

- Harry says "this feels heavy and slow."
- The agent reaches for IC#, Bucket A, attestation, ratification, propagated-discovery in a normal spec conversation.
- A spec doc accumulates Status / Last-attested / Phase / Consumes / Produces frontmatter.
- A "Phase 2 validation pass" appears as a separate session that produces a 250+ line bucketed findings report.
- An ADR has to be added mid-session to reconcile two ceremonies the methodology required.
- The methodology doc has more readers than the spec docs it governs.
- **The agent runs a fixed-cadence elicitation interview** (the `/draft-invariants` 9-prompt pattern) when project files or a prose spec already contain the answers — or when a single open-ended question would have produced a draft Harry could red-pen. This is the ngst failure mode and the one ADD was commissioned to solve; it must not be reproduced under the new methodology.
- A doc accumulates more than one ID scheme that organises its own internal partition (e.g. simultaneous F# and B# numbering with no external referent). Single outward-anchored schemes are fine — see §3.3.
- An adversarial-review session produces a separate findings document instead of in-place edits to the spec it reviewed (see §3.7).
- A `JOURNAL.md` accumulates per-section status fields, frontmatter blocks, or governance metadata beyond the entry shape (date / type / why / links). The methodology doc is returning in disguise.
- Journal entries are written in process register (*"propagated-discovery cascade triggered re-evaluation of the spec attestation per ADR-006"*) instead of product register (*"after we found that test X covered an edge case the spec hadn't named, we updated §4 to mention it"*). Type values like `propagated-discovery` belong in the frontmatter; the body should sound like a colleague explaining a decision over coffee.
- A spec doc gets re-edited and re-committed to "stay current" instead of being left as a snapshot and superseded by a later snapshot + journal entry. Specs are dated artifacts, not living documents.
- The walk-up rule in `AGENTS.md` is empirically being skipped — agents touch a deep file without reading the journals above it. (Symptom: agents propose approaches the journal already rejected.) Means the v1 enforcement is insufficient and the §3.4 stack needs the next layer.

If any of these appear, stop. The form is wrong again.

---

## 6. Final note on register

Writing this document, I had to resist reaching for "Bucket A," "the seam-validation pass," "the discovery cascade." The vocabulary is sticky once introduced. The transcripts make clear: Harry's vocabulary is product-shaped, hedged in the first person, and short. When the agent slips into process register, that's the moment to switch back. The discipline isn't a process — it's a register.
