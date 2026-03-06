# Claude Code Plugin Marketplace

A collection of Claude Code plugins. Each plugin is a self-contained directory with its own skills, agents, and optional MCP server.

## Plugins

### [crosscheck](./crosscheck/)

Crosscheck Claude's code claims with [Dafny](https://dafny.org/) formal verification. The LLM proposes Dafny specs and implementations, the Dafny verifier acts as a hard correctness gate, and only verified code gets extracted to the target language. No Dafny artifacts are committed — only clean Python/Go output.

| | |
|---|---|
| **Skills** | `/spec-iterate`, `/generate-verified`, `/extract-code`, `/lightweight-verify` |
| **Agent** | `verify-orchestrator` — end-to-end workflow automation |
| **MCP tools** | `dafny_verify`, `dafny_compile`, `dafny_cleanup` |

**Prerequisites:** Docker, Node.js >= 18

### [awesome-copilot](./awesome-copilot/)

Meta prompts that help you discover and install curated GitHub Copilot agents, instructions, prompts, and skills from the [awesome-copilot repository](https://github.com/github/awesome-copilot).

| | |
|---|---|
| **Skills** | `/suggest-agents`, `/suggest-instructions`, `/suggest-prompts`, `/suggest-skills` |
| **Agent** | `project-scaffold` — end-to-end project scaffolding |

## Installation

Each plugin is self-contained. Point Claude Code at the plugin directory:

```bash
claude --plugin-dir ./crosscheck
claude --plugin-dir ./awesome-copilot
```

See each plugin's README for prerequisites and setup details.

## Development — crosscheck

```bash
cd crosscheck/mcp-server
npm install
npm run build              # TypeScript → dist/
npm test                   # Unit, integration, property, MCP tests (vitest)
npm run test:e2e           # End-to-end tests (requires Docker)
../scripts/build-docker.sh # Build Dafny Docker image
../scripts/test-mcp.sh     # Smoke tests
```

### Key conventions

- ES modules (`"type": "module"` in package.json)
- Strict TypeScript (ES2022 target, Node16 module resolution)
- Zod for runtime validation of tool inputs
- vitest with fast-check for property-based testing
- Docker image name configured via `DAFNY_DOCKER_IMAGE` env var (default: `crosscheck-dafny:latest`)
