# Prompt: Validate and update the ADD methodology retrospective against transcripts

You're picking up a multi-session retrospective on a project called ADD ("Assurance Driven Development"). The prior agent (Claude, opus-4-7) worked with the user, Harry, on building ADD into the Crosscheck plugin. After substantial spec work and a re-drafting cycle, Harry surfaced sharp criticism of how the process was unfolding. The retrospective that followed produced a working set of findings about what went wrong and how ADD should be reshaped. The prior agent could not read the transcripts of Harry's normal spec conversations (the harness wouldn't load .txt/.jsonl files), so the picture rests partly on inference. You're being run in an environment with direct access to those transcripts. Your job is to read them, test the findings, update them where the transcripts contradict or extend the picture, and produce a consolidated findings document.

## What's in the codebase

The work-product branch is `claude/validate-add-phase-2-bFneS` in the `nicholls-inc/claude-code-marketplace` repo. Read it first.

Key files to absorb:
- `crosscheck/docs/add/methodology.md`, `glossary.md`, `intent.md` — the seed Harry attested before the prior agent's work began.
- `crosscheck/docs/add/decisions/ADR-001..ADR-006.md` — architecture decisions. ADR-006 was added during the re-drafting cycle; it captures the decision that agent-authored attestation commits are permitted, with PR approval as the human signal.
- `crosscheck/docs/add/specs/architectural.md`, `behavioral.md`, `modules/M1..M6.md` — the agent-authored Phase 1 spec stack. Note the lattice of cross-references (consumes/produces metadata, IC# / S# / F# / M# numbering) and the per-artifact Status/Last-attested fields. This is the *form* of ADD that broke.
- `crosscheck/.assurance/phase-2-seam-validation-20260509T182403Z.md` — a 370-line bucketed-findings report the prior agent produced. It identified real substantive issues (e.g., M2 should reuse the existing `/intent-check` pipeline rather than re-derive it) but the form is exactly what Harry called out as process-shaped overhead.
- `crosscheck/skills/intent-check/SKILL.md`, `crosscheck/skills/spec-adversary/SKILL.md`, `crosscheck/skills/assurance-init/SKILL.md`, `crosscheck/agents/byfuglien.md`, `crosscheck/agents/hellebuyck.md` — the existing Crosscheck plugin surfaces ADD was supposed to integrate with. Provide the operational reference for the discipline the prior agent reinvented.

## What Harry said about it

Direct quotes from the retrospective (treat as authoritative):

> As a user of Assurance Driven Development, this process was painful for me. It felt heavy and slow. I considered abandoning the process at least once, and we're not even at the point we can start implementation.

> The assurance and governance language is too pervasive in the process, specifically wording like "That resolves the I1 tension cleanly. Encoding it as a new ADR (ADR-006), then cascading to M2 and M5", "Every IC is consumed by at least one S section", "no hard contradictions", and "seam validation pass". It felt so different from my usual spec-creation conversation. I felt like I was serving the process rather than the process serving me.

> The ADD process should be hidden, almost invisible, while also being adhered to. I want the spec process to be similar to my normal process with you, which is more like a discussion that culminates in a single file documenting the system spec. Ideally I can follow my existing process while ADD artefacts are drafted in the background, and the integrity of ADD artefacts is assessed as our conversation continues so that the agent can nudge me for more detail or highlight an inconsistency or work through a tension in the spec when the ADD artefacts necessitate it.

> I want in-line edits so there's one source of truth and I don't have to cross-reference or mentally reconcile different files. Git provides the history and ADD will provide the audit trail (just because the human-focused spec doc has in-line edits does not mean we weaken the ADD artefact ratification process, that's stays strict and structured).

> "ratification" is PR approval and merge.

## Current findings (your starting point — verify and revise)

### Failure modes of the design that broke

1. **Communication style.** The agent leaned on process vocabulary (IC#, S#, F#, M#, "seam validation," "Bucket A/B," "propagated-discovery cascade") because it was precise and ready to hand. For Harry it was noise. Translation should have been constant.

2. **Process pacing.** Several ceremonies (Phase 2 as a separately gated stage, the seam-validation report with bucketed findings, the attestation / re-attestation cycle, the I1-vs-operational-reality debate that grew into ADR-006) pull weight at scale (multi-team, audit-required) but at single-user / single-agent scale they were friction without proportional benefit. The cleanest example: the prior agent noticed mid-session it was spending product time on process plumbing and rationalised continuing rather than tabling it.

3. **Artifact decomposition.** ADD as built produces ten separate documents (intent + methodology + glossary + 6 ADRs + architectural + behavioral + 6 module specs), each carrying status fields and consumes/produces metadata, all cross-linked. This makes sense for a compliance auditor reading later. It does not make sense for a human reasoning with an agent in real time. Harry wants a single living spec doc.

### What Harry's normal process looks like (from artifacts + the one transcript)

- **Single artifact (or two — requirements + design).** RFC 2119 keywords, glossary, explicit non-goals, named assumptions, formal models (e.g., TLA+) in appendices when warranted.
- **The spec describes the system, not the spec process.** Domain vocabulary throughout — "the dispatcher," "the canceller," "the watcher." No process metadata visible to the reader.
- **Discipline is composed, not imposed.** A single section called "Verified facts and open assumptions" does the work ADD's diff classification, provenance tracking, open-questions list, and drift detection do across many separate artifacts.
- **Git-native iteration.** In-line edits to a living doc. Git provides version history (`git log` / `git diff` / `git blame`). The CKG addendum-based pattern was a workaround for Claude Web's lack of git, not Harry's preferred shape.
- **Conversation is brief and substantive.** The MVP spec session the prior agent did read was 2 minutes wall-clock — one user prompt (three sentences), one agent response. Agent read existing context silently, produced one artifact in Harry's style, closed with one product-language observation: "If the blind comparison lands in the 'helps sometimes' bucket, that's your signal to fast-track backfill before anything else." That closing observation is the model for what an integrity nudge should look like — anticipatory, evidence-grounded, in product language, one sentence.

### Methodology adjustments

The substantive ADD machinery (status, integrity, evidence, drift detection, audit trail) is worth keeping. The visibility flips:

- **Single living spec doc** in Harry's voice and conventions. Audience labelled at top. RFC 2119 throughout. Status field minimal. No process metadata in the reader-facing surface.
- **Git provides history.** No addendum files. No version-bump comments inside the doc.
- **ADD provides the audit trail as a sidecar.** Working hypothesis: per-commit trailers (auto-written by the agent in the shape of `Spec-Diff-Classification: ...`, never negotiated in conversation) plus possibly a sidecar JSON/SQLite that indexes PRs by what they ratified, evidence cited, drift class. Open question whether the sidecar is necessary or trailers alone (queryable via `git log --grep`) carry enough.
- **Ratification = PR approval and merge.** Not a separate ceremony. PR description carries the evidence. Approval-and-merge is the human signal. Git provides the anchor (no need to invent stable section IDs or per-claim ratification records — what was ratified is whatever the merged PR's diff covers).
- **Conversation looks like normal spec work.** Agent reads context silently. Produces artifact in Harry's style. Closes with one or two product-language observations anticipating downstream tensions. No process vocabulary leaks. If the agent reaches for "Bucket A" or "IC11," that's the smell — switch register.
- **Integrity nudges in product language.** When the agent finds a real gap, contradiction, or unresolved tension, surface as a normal review remark ("section 4.5 says commit-triggered but section 7.2 implies PR-triggered; which is right?"). Not as a structured findings list.
- **Self-checks against silence.** The agent periodically runs integrity audits of its own sidecar — does every ratified PR have evidence cited, are evidence sources still resolvable, has any ratified section been edited without a follow-up ratification PR. Surface anomalies. Without this, discipline relies on the agent never making a mistake.

### Two foundational worries

1. **Calibration of silence.** Discipline lives in the agent's head and the sidecar it maintains. There's no live external signal that tells Harry whether the agent is over-cautious (never nudges, silent decay sets in) or under-cautious (audit records confidence that isn't earned). Self-checks help; they don't solve it.

2. **Closing-observation cadence in long sessions.** The MVP transcript was one prompt + one response. Real spec sessions can be many turns over hours. *When* does the agent surface integrity findings — at the end, mid-conversation, on prompt? The pattern from the short transcript doesn't tell us. The harder transcripts should.

### Substantive design decisions worth keeping from the broken-form work

The artifact form was wrong; some substance is right.

- M2's prose-vs-prose validation reusing `/intent-check`'s existing pipeline (substitutions table; inherited two-section back-translator, mandatory carve-out scan, fail-closed semantic validation, content-hashed attestation).
- The PR-merge approval gate concept from ADR-006.
- Deterministic-only-no-LLM rule for any instrumentation tool.
- Five-class diff classification (propagated-discovery / intent-refinement / drift / retraction / status-transition) — useful as agent-maintained metadata, not as user-facing ceremony.
- Broader skill-adaptation list: `/assurance-status`, `/assurance-roadmap-check`, `/protected-surface-amend` need ADD-mode awareness, not just the original five.

## What you have access to that the prior agent didn't

Conversation transcripts. Likely:
- The Code Knowledge Graph initial spec session (where the design and requirements docs were drafted from scratch).
- The CKG design refinement session (which produced Design Update 2, informed by Harry's investigation of 342 sessions).
- The Fabricator / webhook spec conversation (which produced the RFC 2119 + TLA+ spec).

These are the harder cases. The MVP-session transcript the prior agent saw was a clean case (distill an existing design to MVP). The harder transcripts should show the moments where the discipline matters most — surfacing tension, working through gaps, the agent pushing back.

Paths:
- ~/Downloads/webhook-spec-conversation.jsonl
- ~/Downloads/cgv-system-design.jsonl
- ~/Downloads/git-cg-initial-spec.jsonl
- ~/Downloads/git-cg-design-refinement.jsonl

## Your task

1. Read the branch `claude/validate-add-phase-2-bFneS` end to end. You need the broken-form design in your head before the transcripts make their full point.
2. Read the transcripts.
3. Test the findings above against what you actually observe. Specifically look for:
   - **Tension surfacing.** When the agent noticed an inconsistency, gap, or contradiction, what did it do? What register? Did it interrupt the flow or wait for an opening?
   - **Pushback.** When Harry's framing seemed off to the agent, did the agent push back? How?
   - **Silent context maintenance.** Did the agent maintain durable state outside the conversation — notes, draft artifacts, internal models that surfaced later?
   - **Conversation pacing.** Long or short? Many turns or few? Where were the pauses and what happened in them?
   - **What the artifact looked like at session end.** Visible iteration toward the polished form, or did one draft emerge close to final?
   - **Process vocabulary.** Did the agent reach for any process-shaped terms? Or stay in product language throughout?
   - **Closing-observation pattern at scale.** In a multi-turn session, when did the agent produce forward-looking observations? Consistent with the MVP-session pattern, or different?
   - **Harry's corrections.** Where did he push back on the agent? Those corrections are gold — they reveal what he expects.
4. Update the findings. Confirm what holds. Replace what's wrong. Add what's missing. Treat the version above as a hypothesis to verify, not as settled.
5. Pay specific attention to the two open worries (calibration of silence, closing-observation cadence in long sessions). The transcripts may answer one or both.

## Output

Produce a markdown document at `crosscheck/docs/add/.retrospective/findings-and-methodology-v2.md` (Harry can move it). Structure:

1. **State of work** — brief summary of where ADD-the-project stands on this branch.
2. **What we observed about Harry's normal process** — patterns from artifacts and transcripts. Cite specifics from the transcripts where they confirm or change the prior picture.
3. **Methodology adjustments** — the working version of how ADD should be shaped, post-retrospective. Supersedes the explicit-process design currently on the branch.
4. **Worries and open questions** — what's still uncertain, what needs to be resolved before implementation, what design problems remain.
5. **Handoff for whoever implements** — concrete, actionable. What to keep, what to throw out, what the first task is.

Resist the existing process vocabulary when writing. Use Harry's register (terse, precise, evidence-grounded). If you reach for "IC1" or "Bucket A" or "propagated-discovery," switch register.

Expect Harry to revise this picture. He revised the prior agent's three times across the retrospective alone. Your output is a hypothesis for him to test, not a settled answer.
