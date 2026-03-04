# Formal Verify

A Claude Code plugin that enables formal verification of code using [Dafny](https://dafny.org/). Write formally verified specifications, generate implementations that satisfy them, and extract clean Python or Go code — all from within Claude Code.

The key idea: the LLM proposes Dafny specs and implementations, the Dafny verifier acts as a hard correctness gate, and only verified code gets extracted to the target language. No Dafny artifacts are committed — only clean Python/Go output is the deliverable.

## Prerequisites

- **Docker** — Dafny runs in an isolated container
- **Node.js** >= 18
- **Claude Code** with plugin support

## Installation

1. Build the Docker image:

```bash
./scripts/build-docker.sh
```

2. Build the MCP server:

```bash
cd mcp-server
npm install
npm run build
```

3. Install the plugin:

```bash
claude plugin add /path/to/formal-verify
```

## Quick Start

The plugin provides three skills that form a verification pipeline:

```
/spec-iterate  →  /generate-verified  →  /extract-code
   (spec)           (implement)            (compile)
```

**1. Draft and verify a specification:**

```
/spec-iterate "function that returns the maximum element of a non-empty integer array"
```

The AI drafts a Dafny spec with preconditions and postconditions, then verifies it (up to 5 attempts).

**2. Generate a verified implementation:**

```
/generate-verified
```

The AI fills in the implementation body with loop invariants and assertions, verifying until it passes.

**3. Extract clean code:**

```
/extract-code to python
```

Compiles the verified Dafny to Python (or Go), strips Dafny runtime boilerplate, and returns clean code ready for integration.

### Orchestrator Agent

For an end-to-end workflow, use the `verify-orchestrator` agent which chains all three phases automatically.

## Skills

### `/spec-iterate` — Specification Refinement

Iteratively drafts and verifies Dafny specifications from a natural language description.

- Analyses the description for Dafny limitations (IO, concurrency, external libs)
- Drafts a spec with preconditions, postconditions, and invariants
- Verifies the spec against Dafny (max 5 attempts)
- Presents the verified spec for approval before proceeding

### `/generate-verified` — Verified Implementation

Generates an implementation body that satisfies an approved specification.

- Fills in the method body with loop invariants, assertions, and lemmas
- Verifies the full program (max 5 attempts)
- Warns about post-generation pitfalls (`real` types, Go generics, underscore identifiers)

### `/extract-code` — Compile and Extract

Compiles verified Dafny to Python or Go and strips boilerplate.

- Removes Dafny runtime imports (`_dafny`, `_System`)
- Excludes generated runtime files
- Provides type mapping guidance for the target language

## MCP Tools

The plugin exposes three tools to Claude Code via MCP:

| Tool | Description |
|------|-------------|
| `dafny_verify` | Verifies Dafny source code and returns structured errors/warnings |
| `dafny_compile` | Compiles Dafny to Python or Go, stripping boilerplate |
| `dafny_cleanup` | Removes stale temp directories older than 30 minutes |

All tools run Dafny inside Docker with: 512MB memory, 1 CPU, no network access, and a 120-second timeout.

## Architecture

```
formal-verify/
├── mcp-server/                # Node.js/TypeScript MCP server
│   ├── src/
│   │   ├── index.ts           # Server entry, tool registration
│   │   ├── docker.ts          # Docker container execution
│   │   ├── tempdir.ts         # Temp directory lifecycle
│   │   └── tools/
│   │       ├── verify.ts      # dafny_verify tool
│   │       ├── compile.ts     # dafny_compile tool
│   │       └── cleanup.ts     # dafny_cleanup tool
│   ├── src/__tests__/         # Unit, property, integration, MCP, e2e tests
│   ├── Dockerfile             # Multi-stage Dafny image
│   └── package.json
├── skills/                    # Claude Code skills
│   ├── spec-iterate/          # /spec-iterate
│   ├── generate-verified/     # /generate-verified
│   └── extract-code/          # /extract-code
├── agents/
│   └── verify-orchestrator.md # End-to-end orchestrator agent
├── scripts/
│   ├── build-docker.sh        # Build the Dafny Docker image
│   └── test-mcp.sh            # MCP server smoke tests
└── .claude-plugin/
    └── plugin.json            # Plugin configuration
```

## Testing

```bash
cd mcp-server

# Unit, property, integration, and MCP tests
npm test

# Watch mode
npm run test:watch

# End-to-end tests (requires Docker and built image)
npm run test:e2e
```

The test suite includes:

- **Unit tests** — Individual functions (parsing, boilerplate stripping, exclusion logic)
- **Property-based tests** — Fast-check generators for edge case discovery
- **Integration tests** — Multi-function workflows (file collection, temp directories)
- **MCP protocol tests** — Server contract verification
- **End-to-end tests** — Full Docker integration with real Dafny verification

## Known Limitations

- **IO / Networking** — Cannot be formally verified; requires `{:extern}` stubs
- **Concurrency** — Dafny verifies sequential correctness only
- **External libraries** — Calls to external libraries are trust boundaries
- **Go generics** — Type erasure to `interface{}`; may need manual type assertions
- **Dafny `real` type** — Compiles to `_dafny.BigRational`, not native floats
- **Mutable state** — Supported but harder to verify; functional design preferred

## Configuration

The Docker image name defaults to `formal-verify-dafny:latest` and can be overridden via the `DAFNY_DOCKER_IMAGE` environment variable in `plugin.json`.
