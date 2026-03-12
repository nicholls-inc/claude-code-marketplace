# crosscheck — Claude Code Plugin

Crosscheck Claude's code claims using two complementary approaches: **formal verification** via Dafny for provably correct Python/Go code, and **semi-formal reasoning** for structured code analysis with evidence-grounded certificates.

## Prerequisites

- **Docker** — required to run the Dafny verifier in an isolated container
- **Node.js** >= 18 — for the MCP server
- **Claude Code** — with plugin support

## Installation

### 1. Build the Docker image

```bash
./scripts/build-docker.sh
```

This builds a multi-stage Docker image (~300-400MB) with Dafny 4.11.0 and Z3.

### 2. Build the MCP server

```bash
cd mcp-server
npm install
npm run build
```

### 3. Install the plugin

Point Claude Code at this plugin directory:

```bash
claude --plugin-dir ./crosscheck
```

## Usage

### Skills — Formal Verification

Four skills for Dafny-backed formal verification:

#### `/spec-iterate` — Specification Refinement

Draft and verify a Dafny formal specification from a natural language description.

```
/spec-iterate "function that returns the maximum element of a non-empty integer array"
```

#### `/generate-verified` — Verified Implementation

Generate a Dafny implementation body that satisfies a verified spec.

```
/generate-verified
```

#### `/extract-code` — Compile & Extract

Compile verified Dafny to Python or Go, with boilerplate stripped.

```
/extract-code to python
/extract-code to go
```

#### `/lightweight-verify` — Lightweight Verification Alternatives

For functions where full formal verification is overkill, generate lightweight verification artifacts: design-by-contract assertions, property-based tests, or documented runtime invariant checks.

```
/lightweight-verify "function that returns the maximum element of a non-empty integer list" python
/lightweight-verify "binary search on a sorted array returning the index or -1" go
```

### Skills — Semi-formal Reasoning

Four skills for structured code analysis, adapted from the "Agentic Code Reasoning" paper (Ugare & Chandra, Meta, 2026). Semi-formal reasoning forces evidence gathering, execution tracing, and alternative hypothesis checking before any conclusion — preventing unsupported claims and confirmation bias.

**Key results from the paper:**
- Patch equivalence: 78%-88% accuracy (curated), 93% on real-world patches
- Code question answering: 87% accuracy on RubberDuckBench (+9pp over standard)
- Fault localization: +5-12pp over standard reasoning

#### `/reason` — Semi-formal Code Reasoning

General-purpose structured reasoning for any code question. Produces an evidence-backed certificate with premises, execution traces, alternative hypothesis checks, and formal conclusions. Includes optional deep-analysis mode with function trace tables and data flow analysis for code behavior questions.

```
/reason "Is this function thread-safe?" src/cache.py
/reason "What does this function actually do?" src/utils.ts:42
/reason "Will this refactor change the public API behavior?"
```

#### `/compare-patches` — Patch Equivalence Verification

Determine whether two code patches are semantically equivalent by tracing execution through the test suite.

```
/compare-patches
```

#### `/locate-fault` — Fault Localization

Locate the root cause of a failing test using 4-phase structured analysis: test semantics, code path tracing, divergence analysis, and ranked predictions.

```
/locate-fault "test_year_before_1000 fails with AttributeError" tests/test_dateformat.py
```

#### `/trace-execution` — Execution Path Tracing

Hypothesis-driven execution path tracing that builds complete call graphs from entry point to leaf functions.

```
/trace-execution "format(value, format_string)" django/utils/dateformat.py
/trace-execution "What happens when UserService.authenticate() is called with an expired token?"
```

### Orchestrator Agent — Byfuglien

The `byfuglien` agent is the unified orchestrator. It classifies tasks, routes to the appropriate skill, and validates output quality. Named after Dustin Byfuglien — the crosschecking enforcer.

For formal verification tasks, it runs the full pipeline: spec refinement → verified implementation → code extraction. For reasoning tasks, it selects the right analysis skill and enforces evidence standards.

## Core Principles — Semi-formal Reasoning

1. Every claim must cite file:line evidence
2. Always read actual code — never guess from function names
3. Check alternative hypotheses before concluding
4. The structured format IS the reasoning process, not just output formatting
5. Name resolution matters — check for shadowing at every scope

## MCP Tools

The plugin exposes three MCP tools:

| Tool | Description |
|------|-------------|
| `dafny_verify` | Verify Dafny source code |
| `dafny_compile` | Compile Dafny to Python or Go |
| `dafny_cleanup` | Remove stale temp directories |

## Architecture

- **Docker isolation**: Dafny runs in a container with `--network=none`, 512MB memory limit, 1 CPU, and 120s timeout
- **Source as string**: LLM passes Dafny code directly; the MCP server handles all file I/O internally
- **Boilerplate stripping**: Compiled output has Dafny runtime imports and files removed automatically
- **No Dafny artifacts committed**: Only clean Python/Go output is the deliverable
- **On-demand skill loading**: The Byfuglien agent reads skill SKILL.md files on-demand via the Read tool, keeping baseline context lean

## Development

```bash
cd mcp-server
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

## Known Limitations

- **IO/networking**: Cannot be formally verified; requires `{:extern}` stubs
- **Concurrency**: Dafny does not model concurrency; only sequential correctness is verified
- **External libraries**: Calls to external libraries are trust boundaries
- **Go generics**: Compile via type erasure to `interface{}`; type assertions may be needed
- **Dafny `real` type**: Compiles to `BigRational`, not native floats
