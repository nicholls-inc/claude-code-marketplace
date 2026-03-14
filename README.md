# Claude Code Plugin Marketplace

A collection of Claude Code plugins. Each plugin is a self-contained directory with its own skills, agents, and optional MCP server.

## Plugins

### [crosscheck](./crosscheck/README.md)

Crosschecks Claude's code claims using [Dafny](https://dafny.org/) formal verification for provably correct Python/Go code, plus semi-formal reasoning for structured code analysis.

### [awesome-copilot](./awesome-copilot/)

Meta prompts that help you discover and install curated GitHub Copilot agents, instructions, prompts, and skills from the [awesome-copilot repository](https://github.com/github/awesome-copilot).

### [field-report](./field-report/)

Generate structured performance reports on plugins, skills, and agents by analysing Claude Code session conversations.

### [xylem](./xylem/README.md)

Autonomous agent scheduling for GitHub issues — scans, queues, and launches Claude Code sessions to fix bugs, implement features, and refine issue descriptions.

## Installation

Add the marketplace, then install plugins:

```bash
# Add the marketplace
claude plugin marketplace add nicholls-inc/claude-code-marketplace

# Install plugins
claude plugin install crosscheck@nicholls
claude plugin install awesome-copilot@nicholls
claude plugin install xylem@nicholls
```

See each plugin's README for prerequisites and setup details.
