# Claude Code Plugin Marketplace

A collection of Claude Code plugins. Each plugin is a self-contained directory with its own skills, agents, and optional MCP server.

## Plugins

### [crosscheck](./crosscheck/)

Crosschecks Claude's code claims using [Dafny](https://dafny.org/) formal verification for provably correct Python/Go code, plus semi-formal reasoning for structured code analysis.

| | |
|---|---|
| **MCP server** | `dafny_verify`, `dafny_compile`, `dafny_cleanup` |
| **Docker isolation** | Dafny 4.11.0 in a sandboxed container (no network, 512 MB memory, 120 s timeout) |
| **Formal verification skills** | `/spec-iterate`, `/generate-verified`, `/extract-code`, `/lightweight-verify` |
| **Spec management & adequacy skills** | `/check-regressions`, `/suggest-specs`, `/rationale` |
| **Semi-formal reasoning skills** | `/reason`, `/compare-patches`, `/locate-fault`, `/trace-execution` |
| **Agent** | `byfuglien` — unified task classification, skill routing, and output validation |

**Prerequisites:** Docker, Node.js >= 18

### [awesome-copilot](./awesome-copilot/)

Meta prompts that help you discover and install curated GitHub Copilot agents, instructions, prompts, and skills from the [awesome-copilot repository](https://github.com/github/awesome-copilot).

| | |
|---|---|
| **Skills** | `/suggest-agents`, `/suggest-instructions`, `/suggest-prompts`, `/suggest-skills` |
| **Agent** | `project-scaffold` — end-to-end project scaffolding |

## Installation

Add the marketplace, then install plugins:

```bash
# Add the marketplace
claude plugin marketplace add nicholls-inc/claude-code-marketplace

# Install plugins
claude plugin install crosscheck@nicholls
claude plugin install awesome-copilot@nicholls
```

See each plugin's README for prerequisites and setup details.

## Development — crosscheck

```bash
cd crosscheck/mcp-server
npm install
npm run build              # Type-check + esbuild bundle → dist/index.js
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

### Dafny limitations to keep in mind

- No IO/networking verification — requires `{:extern}` trust boundaries
- No concurrency modeling — sequential correctness only
- Go output uses type erasure to `interface{}` — may need type assertions
- `real` type compiles to `_dafny.BigRational`, not native floats
