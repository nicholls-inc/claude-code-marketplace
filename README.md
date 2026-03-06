# Claude Code Plugin Marketplace

A collection of Claude Code plugins for enhanced development workflows.

## Plugins

### [crosscheck](./crosscheck/)

Crosscheck Claude's code claims with [Dafny](https://dafny.org/) formal verification. The LLM proposes Dafny specs and implementations, the Dafny verifier acts as a hard correctness gate, and only verified code gets extracted to the target language.

**Skills:** `/spec-iterate`, `/generate-verified`, `/extract-code`, `/lightweight-verify`
**Agent:** `verify-orchestrator`
**MCP tools:** `dafny_verify`, `dafny_compile`, `dafny_cleanup`

### [awesome-copilot](./awesome-copilot/)

Meta prompts that help you discover and install curated GitHub Copilot agents, instructions, prompts, and skills from the [awesome-copilot repository](https://github.com/github/awesome-copilot).

**Skills:** `/suggest-agents`, `/suggest-instructions`, `/suggest-prompts`, `/suggest-skills`
**Agent:** `project-scaffold`

## Installation

Each plugin is self-contained. Point Claude Code at the plugin directory:

```bash
claude plugin add /path/to/crosscheck
claude plugin add /path/to/awesome-copilot
```

See each plugin's README for prerequisites and setup details.
