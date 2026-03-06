# crosscheck — Claude Code Plugin

Crosscheck Claude's code claims with Dafny formal verification. The LLM proposes Dafny specs and implementations, the Dafny verifier acts as a hard correctness gate, and only verified code gets extracted to the target language. No Dafny artifacts are committed—only clean Python/Go output.

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

### Skills

Three independent skills for granular control:

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

### Orchestrator Agent

For end-to-end workflows, use the `verify-orchestrator` agent which chains all three skills:

1. Spec refinement until user approves
2. Verified implementation until Dafny accepts
3. Code extraction to target language

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

## Testing

```bash
# Verify Docker image works
docker run crosscheck-dafny:latest --version

# Run MCP server smoke tests
./scripts/test-mcp.sh
```

## Known Limitations

- **IO/networking**: Cannot be formally verified; requires `{:extern}` stubs
- **Concurrency**: Dafny does not model concurrency; only sequential correctness is verified
- **External libraries**: Calls to external libraries are trust boundaries
- **Go generics**: Compile via type erasure to `interface{}`; type assertions may be needed
- **Dafny `real` type**: Compiles to `BigRational`, not native floats
