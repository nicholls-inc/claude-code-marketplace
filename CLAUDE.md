# Claude Code Plugin Marketplace

A collection of Claude Code plugins. Each plugin is a self-contained directory with its own skills, agents, and optional MCP server.

## Plugins

### crosscheck (`crosscheck/`)

Crosscheck plugin. Crosschecks Claude's code claims using Dafny as a verification backend, generating provably correct Python/Go code.

- **MCP server** (`crosscheck/mcp-server/`): TypeScript server exposing three tools — `dafny_verify`, `dafny_compile`, `dafny_cleanup`
- **Docker isolation**: Dafny 4.11.0 runs in a sandboxed container (no network, 512MB memory, 120s timeout)
- **Skills** (`crosscheck/skills/`): `/spec-iterate`, `/generate-verified`, `/extract-code`, `/lightweight-verify`
- **Orchestrator agent** (`crosscheck/agents/verify-orchestrator.md`): End-to-end workflow automation

### semiformal (`semiformal/`)

Semi-formal reasoning plugin. Structures Claude's code analysis with explicit premises, execution traces, and formal conclusions.

- **Skills** (`semiformal/skills/`): `/reason`, `/analyze-code`, `/compare-patches`, `/locate-fault`, `/trace-execution`
- **Orchestrator agent** (`semiformal/agents/reasoning-orchestrator.md`): Automatic task classification and skill routing

### awesome-copilot (`awesome-copilot/`)

Meta prompts for discovering and installing curated GitHub Copilot customizations.

- **Skills** (`awesome-copilot/skills/`): `/suggest-agents`, `/suggest-instructions`, `/suggest-prompts`, `/suggest-skills`
- **Agent** (`awesome-copilot/agents/project-scaffold.md`): End-to-end project scaffolding

## Development — crosscheck

```bash
cd crosscheck/mcp-server
npm install
npm run build            # Type-check + esbuild bundle → dist/index.js
npm test                 # Unit, integration, property, MCP tests (vitest)
npm run test:e2e         # End-to-end tests (requires Docker)
../scripts/build-docker.sh  # Build Dafny Docker image
../scripts/test-mcp.sh      # Smoke tests
```

## Key conventions

- ES modules (type: "module" in package.json)
- Strict TypeScript (ES2022 target, Node16 module resolution)
- Zod for runtime validation of tool inputs
- Tests use vitest with fast-check for property-based testing
- Docker image name configured via `DAFNY_DOCKER_IMAGE` env var (default: `crosscheck-dafny:latest`)

## Dafny limitations to keep in mind

- No IO/networking verification — requires `{:extern}` trust boundaries
- No concurrency modeling — sequential correctness only
- Go output uses type erasure to `interface{}` — may need type assertions
- `real` type compiles to `_dafny.BigRational`, not native floats
