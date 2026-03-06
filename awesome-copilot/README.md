# awesome-copilot — Claude Code Plugin

Meta prompts that help you discover and install curated GitHub Copilot agents, instructions, prompts, and skills from the [awesome-copilot repository](https://github.com/github/awesome-copilot).

## Installation

Point Claude Code at this plugin directory:

```bash
claude --plugin-dir ./awesome-copilot
```

## Usage

### Skills

Four discovery skills, each targeting a different customization type:

#### `/suggest-agents` — Discover Custom Agents

Analyze your repository and suggest relevant custom agents from awesome-copilot. Compares against locally installed agents in `.github/agents/`, identifies outdated versions, and presents recommendations for approval before installation.

```
/suggest-agents
```

#### `/suggest-instructions` — Discover Instructions

Suggest copilot-instruction files with `applyTo` glob patterns targeting specific file types. Compares against locally installed instructions in `.github/instructions/`.

```
/suggest-instructions
```

#### `/suggest-prompts` — Discover Prompts

Suggest prompt files for common workflows. Compares against locally installed prompts in `.github/prompts/`.

```
/suggest-prompts
```

#### `/suggest-skills` — Discover Skills

Suggest agent skills (self-contained folders with `SKILL.md` and optional bundled assets). Compares against locally installed skills in `.github/skills/`.

```
/suggest-skills
```

### Project Scaffold Agent

For end-to-end setup, use the `project-scaffold` agent which:

1. Analyzes your repository's tech stack and project type
2. Fetches all available customizations from awesome-copilot
3. Cross-references and recommends agents, prompts, instructions, and skills
4. Awaits your approval, then downloads and installs selected items to `.github/`

### Common workflow

Each skill follows the same pattern:

1. **Fetch** the curated list from awesome-copilot
2. **Scan** locally installed customizations
3. **Compare** remote vs local versions to detect updates
4. **Analyze** your repository context for relevance
5. **Present** recommendations with install status
6. **Install** only after explicit user approval
