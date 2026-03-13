# field-report — Claude Code Plugin

Generate structured performance reports on plugins, skills, and agents by analysing Claude Code session conversations. This plugin produces evidence-based narrative reports with actionable recommendations for artifact developers.

## Installation

Point Claude Code at this plugin directory or install from the marketplace:

```bash
claude plugin install field-report@nicholls
```

## Usage

### /field-report

Generate a structured field report for one subject using one Claude Code session as evidence.

Usage: `/field-report <session-id or description> <subject-name> [path/to/definition]`

Examples:
- `/field-report ses_abc123 reason` — Explicit session ID and subject
- `/field-report "last session" crosscheck/reason` — Natural language session selector
- `/field-report "today's session" byfuglien crosscheck/agents/byfuglien.md` — With explicit definition path

## Analysis Dimensions

The report evaluates the subject across six key dimensions:
1. **Task Completion**: Did the subject help complete the user goal?
2. **Instruction Adherence**: How closely did execution match subject instructions?
3. **Tool Usage Patterns**: Evaluation of tool selection and execution behavior.
4. **Conversation Efficiency**: Measurement of progress quality and momentum.
5. **Error Handling**: Inspection of observable failures and recoveries.
6. **Input Clarity**: Evaluation of initial request clarity and clarification needs.

## Report Output

Reports are saved to the `field-reports/` directory in your working directory. Filenames follow the convention `{subject-slug}--{session-id-short}--{YYYY-MM-DD}.md`. Each report includes a header metadata block, detailed analysis sections, lessons learned, and a summary.

## Sanitisation

Reports are privacy-safe. The plugin applies explicit redaction rules:
- **STRIP**: Email addresses, API keys, secrets, environment variable values, and personal names.
- **KEEP**: File paths, tool names, error messages, and code patterns.

## Limitations

- Session discovery is heuristic. Explicit session IDs are more reliable than natural language descriptions.
- Each report covers exactly one session and one subject.
- Tool usage analysis depends on the session data format and availability of tool-call traces.
- No baseline comparisons or historical trending across multiple reports.
- Sessions must have at least 10 messages involving the subject to ensure sufficient evidence density.
