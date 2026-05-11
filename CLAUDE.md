# Claude Code Plugin Marketplace

A collection of Claude Code plugins. Each plugin is a self-contained directory with its own skills, agents, and optional MCP server.

## Plugins

### crosscheck (`crosscheck/`)

Crosscheck plugin. Crosschecks Claude's code claims using Dafny formal verification for provably correct Python/Go code, plus semi-formal reasoning for structured code analysis.

- **MCP server** (`crosscheck/mcp-server/`): TypeScript server exposing six tools across two engines — Dafny (`dafny_verify`, `dafny_compile`, `dafny_cleanup`) and Lean (`lean_check` for the `/lean-spec`, `/lean-impl`, `/correspondence-review`, and `/drt-oracle` build gates; `lean_run` for `/lean-impl` smoke checks and `/drt-oracle`'s per-def Lean runner; `lean_test` as a compile-time `#guard` path for fixture sanity checks)
- **Docker isolation**: Dafny 4.11.0 in a sandboxed container (no network, 512MB memory, 120s timeout); Lean 4 + Mathlib in a sister container with Mathlib oleans pre-warmed (no network, 2GB memory, 240s timeout)
- **Formal verification skills** (`crosscheck/skills/`): `/spec-iterate`, `/generate-verified`, `/extract-code`, `/lightweight-verify`
- **Lean executable-model + DRT-oracle pipeline** (`crosscheck/skills/`): `/informal-spec`, `/lean-spec`, `/lean-impl`, `/correspondence-review`, `/drt-oracle`
- **Spec management & adequacy skills** (`crosscheck/skills/`): `/check-regressions`, `/suggest-specs`, `/rationale`
- **Semi-formal reasoning skills** (`crosscheck/skills/`): `/reason`, `/compare-patches`, `/locate-fault`, `/trace-execution`
- **Orchestrator agent** (`crosscheck/agents/byfuglien.md`): Unified task classification, skill routing, and output validation

### awesome-copilot (`awesome-copilot/`)

Meta prompts for discovering and installing curated GitHub Copilot customizations.

- **Skills** (`awesome-copilot/skills/`): `/suggest-agents`, `/suggest-instructions`, `/suggest-prompts`, `/suggest-skills`
- **Agent** (`awesome-copilot/agents/project-scaffold.md`): End-to-end project scaffolding

### field-report (`field-report/`)

Field report plugin. Generates structured performance reports on plugins, skills, and agents by analysing Claude Code session conversations.

- **Skills** (`field-report/skills/`): `/field-report`

## Development — crosscheck

```bash
cd crosscheck/mcp-server
npm install
npm run build            # Type-check + esbuild bundle → dist/index.js
npm test                 # Unit, integration, property, MCP tests (vitest)
npm run test:e2e         # End-to-end tests (requires Docker)
../scripts/build-docker.sh       # Build Dafny Docker image
../scripts/build-lean-docker.sh  # Build Lean+Mathlib Docker image (slow first time)
../scripts/test-mcp.sh           # Smoke tests
```

## Key conventions

- ES modules (type: "module" in package.json)
- Strict TypeScript (ES2022 target, Node16 module resolution)
- Zod for runtime validation of tool inputs
- Tests use vitest with fast-check for property-based testing
- Docker images configured via `DAFNY_DOCKER_IMAGE` (default `crosscheck-dafny:latest`) and `LEAN_DOCKER_IMAGE` (default `crosscheck-lean:latest`); Lean memory/cpu via `LEAN_DOCKER_MEMORY` / `LEAN_DOCKER_CPUS`

## Commit conventions

Conventional commits enforced via commitlint + husky. The `docs:` prefix is **blocked** for commits touching behavioral artifact files (`SKILL.md`, `agents/*.md`). These files define agent/skill behavior and are functional code — use `feat:`, `fix:`, or `refactor:` based on the nature of the change.

- `feat(field-report): add new analysis dimension` — new skill behavior
- `fix(crosscheck): correct abort threshold in /reason` — bug fix in skill logic
- `refactor(crosscheck): simplify byfuglien routing` — structural change to agent
- `docs(crosscheck): update README installation steps` — actual documentation (allowed)

## Dafny limitations to keep in mind

- No IO/networking verification — requires `{:extern}` trust boundaries
- No concurrency modeling — sequential correctness only
- Go output uses type erasure to `interface{}` — may need type assertions
- `real` type compiles to `_dafny.BigRational`, not native floats
