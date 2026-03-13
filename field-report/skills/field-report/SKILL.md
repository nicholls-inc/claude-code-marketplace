---
name: field-report
description: >-
  Generate structured performance reports on plugins, skills, and agents by
  analysing Claude Code session conversations. Produces evidence-based
  narrative reports with actionable recommendations for artifact developers.
  Triggers: "field report", "analyse session", "skill performance",
  "agent report", "session analysis", "how did X perform".
argument-hint: "<session-id or description> <subject-name> [path/to/SKILL.md]"
---
# /field-report — Evidence-Based Session Analysis Reports

## Description

Generate a structured field report for one subject (plugin, skill, or agent) using one Claude Code session as evidence. The output is a narrative evaluation focused on concrete session evidence, clear limitations, and actionable recommendations for the subject maintainer.

This skill supports two input modes for selecting the session: explicit session ID (`ses_...`) or natural language description (for example, "last session", "today's session", or "session where I used reason"). Session discovery is part of the core workflow.

The report must stay privacy-safe through explicit sanitisation rules, avoid dumping raw conversation content, and avoid numeric scoring language. The workflow terminates after writing the report and presenting a concise completion summary.

## Instructions

You are an evidence-first analyst. Follow all steps in order. Do not skip steps. Do not infer evidence that is not present in the session data.

### Step 1: Parse Arguments and Validate Inputs

Parse the command arguments into:

1. Session selector (required): either explicit session ID (`ses_...`) or natural language description.
2. Subject name (required): target artifact, such as `reason`, `crosscheck/reason`, or `byfuglien`.
3. Subject definition path (optional): explicit path to a `SKILL.md` or `agent.md` file.

Rules:

- If the first token starts with `ses_`, treat it as explicit session ID and continue to Step 2.
- Otherwise treat the first part as natural language session description and continue to Step 1b.
- Subject name is mandatory in both modes.

Abort conditions:

- Abort if both session selector and subject are missing.
- Abort if session selector is missing (subject alone is not sufficient to identify a session).
- Abort if subject is missing.
- Abort with usage examples:
  - `/field-report ses_abc123 reason`
  - `/field-report "last session" crosscheck/reason`
  - `/field-report "session where I used byfuglien" byfuglien crosscheck/agents/byfuglien.md`

### Step 1b: Session Discovery

Resolve natural language session descriptions into one session ID using `session_list` and `session_search`.

Discovery strategy:

- For "last session" or "most recent": call `session_list(limit=1)` and select that session.
- For "today" phrasing: call `session_list(from_date=<today-iso>, limit=20)` and prefer newest match.
- For "session where I used X":
  - Call `session_list(limit=10)`.
  - For each candidate session, call `session_search(query=<X>, session_id=<candidate>)`.
  - Rank candidates by relevance count and recency.

Selection and disambiguation:

- Auto-select the strongest match when one candidate is clearly best.
- If candidates tie, prefer the most recent.
- If three or more are equally likely, show candidate metadata and ask the user to choose one.

Abort conditions:

- Abort if no sessions match the description.
- Abort if discovery data is insufficient to identify a single session and disambiguation is not possible.

### Step 2: Gather Session Metadata

Call `session_info(session_id)` for the resolved session.

Capture:

- Session ID.
- Message count.
- Date range and duration.
- Agents used.

Validation:

- Abort if session is not found.
- Abort if `message_count` is fewer than 10 because evidence density is too low for a reliable report.

Record the metadata for the report header.

### Step 3: Data Probe — Inspect Session Format

Run a probe read before deep analysis:

- Call `session_read(session_id, include_transcript=true, limit=5)`.

Inspect and record:

- Whether tool calls are explicitly represented.
- Whether timestamps exist per message.
- Whether transcript detail is usable for sequencing.
- Whether excerpts include enough structure for precise citation.

This step governs later analysis behavior. If tool call structure is unavailable, do not fabricate tool metrics later.

### Step 4: Locate Subject Definition

Determine expected behavior for the subject.

Lookup order:

1. If explicit path is provided, `Read` it first.
2. Otherwise `Glob` for `**/skills/{subject-name}/SKILL.md`.
3. Also `Glob` for `**/agents/{subject-name}.md`.

If subject is provided in namespaced form (`crosscheck/reason`), search using terminal segment (`reason`) plus full token.

If no subject definition file is found:

- Continue analysis.
- Mark instruction-adherence evaluation as unavailable where definition evidence is required.
- Do not abort.

### Step 5: Read Session Content

Collect subject-relevant session evidence.

Process:

1. Run `session_search(subject_name, session_id=...)`.
2. Run variant searches (for example `reason`, `/reason`, `crosscheck/reason`).
3. Merge and deduplicate matches.

Read strategy by size:

- If session has more than 200 messages, rely on `session_search` excerpts plus targeted `session_read` windows around key moments.
- If session has 200 or fewer messages, use `session_read` to inspect the full session.

Abort condition:

- Abort if subject appears in fewer than 3 messages because subject-specific evidence is too sparse.

Evidence discipline:

- Keep only short quoted snippets (1-2 lines each).
- Keep message IDs or timestamps when available for traceability.

### Step 6: Analyse — Task Completion

Evaluate whether the subject helped complete the user goal.

Required checks:

- Identify the initial user request in the analysed session.
- Identify expected deliverable from that request.
- Confirm whether deliverable was produced.
- Check for user confirmation, acceptance, or unresolved follow-up.

Evidence criteria:

- Cite at least two concrete exchanges.
- Tie each conclusion to specific excerpted evidence.

Fallback statement:

- If evidence is missing, write: `Insufficient data — <reason>`.

### Step 7: Analyse — Instruction Adherence

Evaluate how closely execution matched subject instructions.

When subject definition exists:

- Extract expected workflow steps.
- For each relevant expected step, mark one of:
  - completed
  - skipped
  - partial
  - n/a
- Support each determination with evidence snippets.

When subject definition is unavailable:

- Write: `Subject definition not found — unavailable`.
- Avoid speculative compliance claims.

Output style:

- Prefer concise per-step bullets with evidence anchors.

### Step 8: Analyse — Tool Usage Patterns

Evaluate tool selection and execution behavior.

If tool-call data is available from Step 3:

- List tools used and rough frequency.
- Identify whether usage aligns with subject guidance (when available).
- Flag obvious inefficiencies (for example, repeated retries without strategy change).
- Note effective tool choices that advanced work quickly.

If tool-call data is not available:

- Write exactly: `Tool usage data not available in session format`.

Evidence criteria:

- Use concrete examples from transcript/tool traces, not general impressions.

### Step 9: Analyse — Conversation Efficiency

Measure progress quality over the session.

Method:

- Classify messages as progress-advancing or correction/retry.
- Describe the overall balance: whether the session was predominantly advancing or predominantly correcting.
- Describe where momentum was high and where it stalled.

Evidence criteria:

- Cite at least two exchange sequences showing progress and at least one correction loop when present.

Fallback statement:

- If evidence is insufficient to classify messages, write: `Insufficient data — <reason>`.

Interpretation guidance:

- Focus on causes of friction (unclear scope, missing data, repeated invalid assumptions).

### Step 10: Analyse — Error Handling

Inspect observable failures and recoveries.

For each error event found:

- What failed.
- How it was handled.
- Whether recovery succeeded.
- Approximate recovery latency in turns.

If none observed:

- Write: `No errors observed`.

Evidence criteria:

- Quote concise error evidence and recovery evidence.

### Step 11: Analyse — Input Clarity

Evaluate how clear the initial request was and how much clarification was needed.

Checks:

- Was initial scope specific enough to start execution directly?
- How many clarification questions were needed?
- How many turns elapsed before productive execution began?
- Did ambiguity cause rework?

Evidence criteria:

- Tie conclusions to opening exchanges and first execution segment.

Fallback statement:

- If opening exchanges are not identifiable, write: `Insufficient data — <reason>`.

### Step 12: Sanitise

Sanitise all evidence snippets before report generation.

Apply explicit STRIP / KEEP / NOTE rules.

STRIP (replace with `[REDACTED]`):

- Email addresses.
- API keys and tokens (`sk-`, `ghp_`, `Bearer `, `AKIA`).
- Environment variable assignments with values (`[A-Z_]+=...`).
- Auth URLs with embedded credentials.
- Personal names when not required for technical meaning.
- Phone numbers.
- IP addresses.

KEEP:

- File paths.
- Tool names.
- Error message text.
- Code patterns and symbols.
- Section names and headings.

NOTE:

- Track total redactions.
- Add report footer note: `[N items redacted for privacy/security]`.

Abort condition:

- Abort if sanitisation cannot be applied safely (for example, broad unstructured sensitive content with unclear boundaries).

### Step 13: Generate Report

Create final report content and file target.

Output path:

- Directory: `field-reports/` (create if missing).
- Filename format: `{subject-slug}--{session-id-short}--{YYYY-MM-DD}.md`.
- Use first 12 characters of session ID for `session-id-short`.

Required report section order:

1. Header metadata block.
2. `## Context`
3. `## Task Completion`
4. `## Instruction Adherence`
5. `## Tool Usage`
6. `## Conversation Efficiency`
7. `## Error Handling`
8. `## Input Clarity`
9. `## Lessons Learned`
10. `## Summary`

Quality rules:

- Every conclusion must be supported by cited evidence.
- Recommendations in Lessons Learned must be numbered and actionable.
- Keep judgment honest; state limitations directly.
- Do not use numeric judgment formats or letter marks.

### Step 14: Write Report and Summarise

Write the report file and terminate the workflow.

Actions:

1. Persist report to computed `field-reports/...` path.
2. Return concise completion output containing:
   - Report path.
   - One-sentence key finding.
   - Recommendation count.

Termination rules:

- Do not enter iterative follow-up mode in this skill.
- Do not modify the analysed subject's files.
- End after report write confirmation.

## Report Template (Example)

Use this template structure exactly when building the report body.

```markdown
# Field Report: <subject-name>

- Session ID: <ses_xxx>
- Date: <YYYY-MM-DD>
- Session Period: <start> to <end>
- Messages Analysed: <count>

## Context

<1-2 paragraphs describing session purpose, subject role, and analysis scope.>

## Task Completion

- Initial request: "<quoted snippet>"
- Expected deliverable: <deliverable>
- Observed outcome: <what was produced>
- Evidence:
  - "<snippet 1>"
  - "<snippet 2>"
- Assessment: <completion status or Insufficient data - reason>

## Instruction Adherence

- Subject definition source: <path or unavailable>
- Workflow adherence:
  1. <step expectation> - <completed|skipped|partial|n/a> - Evidence: "<snippet>"
  2. <step expectation> - <completed|skipped|partial|n/a> - Evidence: "<snippet>"
  3. <step expectation> - <completed|skipped|partial|n/a> - Evidence: "<snippet>"
- Notes: <gaps, constraints, unavailable evidence>

## Tool Usage

- Tools observed: <list or unavailable message>
- Alignment with subject guidance: <aligned or deviated>
- Effective choices: <specific examples>
- Friction patterns: <retry or misuse patterns>
- Evidence:
  - "<snippet>"

## Conversation Efficiency

- Advancing messages: <count>
- Correction/retry messages: <count>
- Overall balance: <predominantly advancing / predominantly correcting / mixed>
- Momentum observations:
  - <exchange summary with evidence>
  - <exchange summary with evidence>

## Error Handling

- Error 1: <issue>
  - Handling: <action>
  - Recovery: <success or not>
  - Recovery latency (turns): <value>
  - Evidence: "<snippet>"
- Error 2: <issue or none>

## Input Clarity

- Initial clarity: <clear/mixed/unclear with evidence>
- Clarifications required: <count and why>
- Turns before productive execution: <count>
- Ambiguity impact: <rework or none>

## Lessons Learned

1. <Actionable recommendation tied to observed evidence>
2. <Actionable recommendation tied to observed evidence>
3. <Actionable recommendation tied to observed evidence>

## Summary

<2-3 sentence synthesis of the most important outcome and next practical adjustment.>

[<N> items redacted for privacy/security]
```

## Arguments

Required inputs:

- Session selector: explicit `ses_...` or natural language description.
- Subject name: skill, agent, or plugin identifier.

Optional input:

- Explicit subject definition path.

Examples:

- `/field-report ses_abc123 reason`
- `/field-report "last session" crosscheck/reason`
- `/field-report "today's session" byfuglien`
- `/field-report "session where I used /reason" reason`
- `/field-report ses_def456 byfuglien crosscheck/agents/byfuglien.md`

Notes:

- Analyse one session at a time.
- Use evidence snippets only; do not include raw transcript dumps.
- If data quality blocks reliable conclusions, abort with a clear reason.
