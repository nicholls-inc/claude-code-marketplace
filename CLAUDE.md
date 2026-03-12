# Claude Code Plugin Marketplace

A collection of Claude Code plugins. Each plugin is a self-contained directory with its own skills, agents, and optional MCP server.

## Plugins

### crosscheck (`crosscheck/`)

Crosscheck plugin. Crosschecks Claude's code claims using Dafny formal verification for provably correct Python/Go code, plus semi-formal reasoning for structured code analysis.

- **MCP server** (`crosscheck/mcp-server/`): TypeScript server exposing three tools — `dafny_verify`, `dafny_compile`, `dafny_cleanup`
- **Docker isolation**: Dafny 4.11.0 runs in a sandboxed container (no network, 512MB memory, 120s timeout)
- **Formal verification skills** (`crosscheck/skills/`): `/spec-iterate`, `/generate-verified`, `/extract-code`, `/lightweight-verify`
- **Semi-formal reasoning skills** (`crosscheck/skills/`): `/reason`, `/compare-patches`, `/locate-fault`, `/trace-execution`
- **Orchestrator agent** (`crosscheck/agents/byfuglien.md`): Unified task classification, skill routing, and output validation

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
