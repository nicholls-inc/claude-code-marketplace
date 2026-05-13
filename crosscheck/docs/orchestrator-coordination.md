# Orchestrator coordination and artifact conventions

This doc is the normative reference for three load-bearing patterns that every crosscheck skill is expected to follow when invoked by an orchestrator (`add-orchestrator`, `byfuglien`, `hellebuyck`) or by another skill in a chain:

1. **Marker-file coordination** — how a skill detects it is being driven by an orchestrator and defers its in-skill governance gates.
2. **Findings-file artifacts** — how a skill emits human-reviewable output as a structured file rather than inline chat.
3. **Working-directory persistence** — where chained skills write intermediate artifacts so the human is not a state-carrier between invocations.

These patterns exist because the architectural distinction crosscheck is built around is:

- The human's job is **governance**: vision, scope, and rendering verdicts on agent-produced artifacts (specs, invariants, protected-surface amendments). Typically this happens via PR review.
- The agent's job is **everything mechanical** that produces the artifact the human governs.

If a skill funnels the human into running follow-up skills, asks questions whose answers are inferable from repo state, or hands the human a checklist of analysis the agent already performed, it has the architecture inverted. The three patterns below are the mechanisms that let skills stay on the right side of that line.

---

## 1. Marker-file coordination

### When a skill is marker-aware

A skill MUST adopt marker-file coordination if it satisfies any of:

- It is dispatched by `add-orchestrator`, `byfuglien`, or `hellebuyck` as part of a multi-step workflow.
- It exposes an in-skill governance gate (red-pen loop, sign-off prompt, accept/reject triage) that would duplicate a downstream batched review the orchestrator owns.
- It is intended to be invocable by an implementer agent (e.g. an agent making a protected-surface edit calling `/protected-surface-amend`) without re-asking the human questions answerable from the staged diff.

Skills that are already marker-aware: `/draft-invariants` (§1c — canonical reference), `/audit-spec-coverage`, `/audit-invariant-consistency`, `/intent-check`.

Skills that MUST become marker-aware as part of the refactor:
`/protected-surface-amend`, `/assurance-init`, `/assurance-probe`, `/suggest-specs`, `/spec-adversary`.

### Canonical marker schema

The marker lives at `.assurance/add-session-<id>/session.json` (or any ancestor directory of the skill's cwd). The schema is fixed by `add-orchestrator` and consumed identically by every marker-aware skill:

```json
{
  "session_id": "<16-hex-char nonce>",
  "created_at": "YYYY-MM-DDTHH:MM:SSZ",
  "spec_path": "<path>",
  "modules": ["<m1>", "<m2>", ...],
  "hash_inputs": ["<spec_path>", "<glossary_path>", "<module_map_path>"],
  "hash_algorithm": "sha256",
  "hash_value": "<lowercase-hex-sha256>",
  "hash_discipline_ref": "crosscheck/skills/intent-check/references/attestation-schema.md"
}
```

Hash computation follows `crosscheck/skills/intent-check/references/attestation-schema.md` lines 76–92: sorted absolute paths, raw bytes concatenated with no delimiter, single SHA-256, lowercase hex.

### Detection, validation, effect

A marker-aware skill follows this protocol at the top of its execution:

1. **Detect.** Walk from cwd to git repo root looking for `.assurance/add-session-*/session.json`. If multiple match, use the most recently modified.

2. **Validate.** Refuse with a clear error if any of:
   - The marker JSON does not parse or is missing required fields → *"Marker file malformed — delete `.assurance/add-session-<id>/session.json` and re-run `add-orchestrator`."*
   - The target subject (module, file, finding-ID) is not in `modules` → treat as marker-absent (the marker is scoped to a different session).
   - The recomputed `hash_value` does not match → *"Marker hash mismatch — the spec, glossary, or module-map has changed since the orchestrator pre-flight. Re-run `add-orchestrator` to regenerate the marker."*

3. **Effect (valid marker).** Suppress the in-skill governance gate. Where the skill would normally write `Status: Final` or block on a sign-off prompt, instead:
   - Write `Status: Draft` (preserves the standard taxonomy).
   - Add an `Audit: pending session <session_id>` line.
   - Hand control back to the orchestrator without prompting the user.

4. **Effect (marker absent).** Standard in-skill governance gate runs as before. There is no caller-supplied flag to bypass it; direct invocations from a user always get the standard behavior.

### Coordination, not tamper-resistance

The marker is a **coordination mechanism, not a security boundary**. It ensures the in-skill and orchestrator-side governance gates do not both fire on the same subject, and it catches typo'd manual marker files via the hash check. Sub-agents share filesystem and tool-permission scope with the parent agent, so a determined actor (malicious user prompt, compromised sub-agent) can write a forged marker. Maintainers must not treat the marker as a tamper-resistant attestation. The actual safety net is the in-skill governance gate, which fires whenever the marker is absent.

This caveat is load-bearing and MUST be reproduced verbatim in any new skill's marker-handling section.

### Sign-off semantics

The standard skill-level sign-off obligation (e.g. "user has explicitly signed off on English before any test code is written") is satisfied by **EITHER** the in-skill gate **OR** the orchestrator's downstream batched review of the per-skill findings files. The orchestrator's apply step records the sign-off via the `Audit:` line update; that line is the durable evidence the obligation transferred.

---

## 2. Findings-file artifacts

### Why files, not inline chat

When a skill's output is something a human or an orchestrator will review and act on, it MUST be emitted as a structured file at a predictable path, not as an inline chat block. Inline chat:

- Forces the human to be the state-carrier between skills.
- Cannot be consumed by an orchestrator (the orchestrator does not read its sub-agents' chat).
- Decays on context truncation.
- Cannot be PR-reviewed.

The findings-file pattern was established by `/audit-spec-coverage` and `/audit-invariant-consistency`. It is now the standard for every output the human or an orchestrator reviews.

### Path scheme

```
Orchestrator mode (marker present): .assurance/add-session-<id>/findings-<category>.md
Standalone mode (no marker):        <cwd>/findings-<category>.md
                                  or .assurance/<skill-name>/<scope>-<YYYY-MM-DD>.md
```

`<category>` distinguishes outputs from different skills running in the same session (e.g. `findings-coverage.md`, `findings-consistency.md`). `<scope>` for standalone-mode dated files is typically a module name or short slug.

### Required frontmatter

```yaml
---
session: <id-or-"standalone">
category: <coverage|consistency|adversary|probe|drift|status|...>
generated_at: <YYYY-MM-DDTHH:MM:SSZ>
spec_path: <path-or-null>
scope_glob: <glob-or-null>
total_findings: <n>
---
```

### 4-path triage block

Every finding intended for human triage MUST include this block verbatim. The four paths are exhaustive and mutually exclusive:

```markdown
**Triage (mark exactly one):**
- [ ] Accept (fix invariant/code/test) — <one-line fix description>
- [ ] Accept (amend spec via /protected-surface-amend) — <one-line note on spec edit>
- [ ] Reject — <reason>
- [ ] Defer — <revisit condition>
```

The `Accept (amend spec)` path is first-class. A finding that surfaces a spec gap is a legitimate signal even when the code is correct; the triage system must accommodate it without forcing the human to invent a fifth path.

### Capping discipline

Reviewer fatigue is the dominant failure mode for Layer-5/Layer-6 outputs. Skills MUST cap their findings:

- `/spec-adversary`: ≤ 3 proposals (calibrated against solo maintainers)
- `/audit-spec-coverage`, `/audit-invariant-consistency`: ≤ 15 findings
- `/assurance-probe`: ≤ 3 issue findings

Skills that hit the cap MUST note dropped candidates in a structured block so a downstream re-run can pick them up. A skill that produces 50 findings is producing zero usable findings.

### "What this does NOT catch" section

Every probabilistic or best-effort findings file MUST end with a verbatim honesty section enumerating the failure modes the skill cannot detect. The discipline:

- List 3–5 concrete gap classes (not generic disclaimers).
- Where another skill catches the gap, name it.
- Add run-specific caveats discovered during this run.

Reference: `/audit-spec-coverage` Step 8 and `/spec-adversary` Step 7.

### Kill criteria

Findings-file skills MUST surface signal-to-noise ratios via a tracker file (`.assurance/<skill>-tracker.md` or `<skill>-tracker.csv`). Each run appends a row; when the SNR falls below the skill's defined threshold (typically < 1:5 after N runs), the skill emits a warning that calibration is needed. This is how we detect when a skill has degraded into noise.

---

## 3. Working-directory persistence

### The state-carrier problem

Several skills currently produce their primary output (a verified spec, a generated implementation, a contract-annotated function) into chat scrollback only. When a downstream skill consumes that output, the user has to be the state-carrier — re-pasting or re-establishing the artifact across invocations. This breaks composability and forces user-as-operator behavior.

The fix: every skill that produces an artifact a downstream skill or human will consume MUST write it to a stable location under `.crosscheck/work/<convention>/`. Three conventions cover the existing skill set; new skills should fit one of them or extend with a new convention.

### `.crosscheck/work/dafny/<spec-id>/` — chained Dafny pipeline

Used by `/spec-iterate` → `/generate-verified` → `/extract-code` → `/check-regressions`. These four skills are explicitly designed to chain, and the human is currently the state-carrier between them.

```
.crosscheck/work/dafny/<spec-id>/
├── spec.dfy              # written by /spec-iterate after sign-off
├── impl.dfy              # written by /generate-verified after verification
├── extracted.py | .go    # written by /extract-code
└── registry-entry.json   # consumed by /check-regressions
```

`<spec-id>` defaults to the slugified primary function name; the user may override at `/spec-iterate` invocation time. Multiple specs per repo are expected; the directory is per-spec.

### `.crosscheck/work/rationale/<timestamp>-<short-slug>/` — one-shot rationales

Used by `/rationale`. The skill is one-shot (not chained) but its output (claim tree, generated tests) is currently chat-only.

```
.crosscheck/work/rationale/<YYYY-MM-DD-HHMMSS>-<slug>/
├── claim-tree.md
├── tests/                # generated test files written here, not chat
└── verification-summary.md
```

The timestamp + slug naming prevents collisions when the user runs `/rationale` multiple times on related but distinct questions.

### `.crosscheck/work/lightweight/<target-path-as-slug>/` — single-target verification

Used by `/lightweight-verify`. The skill is also one-shot but is target-keyed (a specific function or module), so the directory should reflect the target rather than a timestamp.

```
.crosscheck/work/lightweight/<slugified-target-path>/
├── annotated.py | .go    # the function with contracts/assertions applied
└── tests/                # property-based or runtime-check tests
```

If the user runs `/lightweight-verify` on the same target twice, the second run overwrites the first (with a one-line summary of what changed). This is intentional — the skill's output is the current verification state for that target, not a history of attempts.

### Cross-cutting rules

- **Never write artifacts to chat scrollback as a substitute for a file write.** Echo the path and a summary; do not paste the file contents inline as primary output.
- **The `.crosscheck/` directory is gitignored by convention.** Repos that want to commit verified specs should copy them into their canonical location (e.g. `formal-verification/specs/`) explicitly. This keeps `.crosscheck/work/` as a scratch area without forcing committment of intermediate state.
- **Cleanup is the user's prerogative.** Skills do not delete entries in `.crosscheck/work/` unless explicitly invoked to. The directory grows over time; that is fine.

---

## How to cite this doc from a SKILL.md

When a skill adopts one of these patterns, it should not re-document the pattern in its own SKILL.md. Instead, a one-line reference suffices:

```markdown
This skill is marker-aware. See
[`crosscheck/docs/orchestrator-coordination.md#1-marker-file-coordination`](../../docs/orchestrator-coordination.md)
for the detection/validation protocol.
```

Skill-specific deviations (e.g. a different findings cap, a custom triage path) are documented in the SKILL.md itself with a note on why the deviation is justified.

---

## Open issues for future work

- **Tamper-resistance for cross-trust orchestrator handoffs.** The marker scheme is coordination-only by design. If a future use case requires actual tamper-resistance (e.g. a remote agent invoking a skill), the marker would need to be signed by a trust anchor outside the shared filesystem. Not currently in scope.
- **Cross-repo orchestration.** All paths above are repo-relative. If an orchestrator drives skills across multiple repos in a single session, the `.assurance/add-session-<id>/` directory needs a canonical host repo. Punt until the use case appears.
- **Garbage collection.** `.crosscheck/work/` grows monotonically. A future `/crosscheck-gc` skill could prune entries older than N days or whose source files no longer exist. Not required for the initial refactor.
