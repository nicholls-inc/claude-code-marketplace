# Formal Verify

Formal verification plugin for Claude Code. Generates provably correct Python/Go code using Dafny as a verification backend.

## Architecture

- **MCP server** (`formal-verify/mcp-server/`): TypeScript server exposing three tools — `dafny_verify`, `dafny_compile`, `dafny_cleanup`
- **Docker isolation**: Dafny 4.11.0 runs in a sandboxed container (no network, 512MB memory, 120s timeout)
- **Skills** (`formal-verify/skills/`): `/spec-iterate`, `/generate-verified`, `/extract-code`
- **Orchestrator agent** (`formal-verify/agents/verify-orchestrator.md`): End-to-end workflow automation

## Workflow

1. User describes a function → `/spec-iterate` generates a verified Dafny specification
2. `/generate-verified` implements the spec with loop invariants, lemmas, etc.
3. `/extract-code` compiles to Python or Go, strips Dafny boilerplate, delivers clean output

## Development

```bash
cd formal-verify/mcp-server
npm install
npm run build            # TypeScript → dist/
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
- Docker image name configured via `DAFNY_DOCKER_IMAGE` env var (default: `formal-verify-dafny:latest`)

## Dafny limitations to keep in mind

- No IO/networking verification — requires `{:extern}` trust boundaries
- No concurrency modeling — sequential correctness only
- Go output uses type erasure to `interface{}` — may need type assertions
- `real` type compiles to `_dafny.BigRational`, not native floats
