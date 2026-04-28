# Crosscheck

Crosscheck checks Claude's code claims with two orchestrator agents — `byfuglien` (implementation) and `hellebuyck` (specification) — coordinating three pillars of assurance. The first pillar is formal verification with Dafny, producing provably correct Python/Go from natural-language specs. The second is semi-formal reasoning, which forces evidence-grounded certificates before any conclusion about a piece of code. The third is a 6-layer assurance hierarchy with governance scaffolding, so claims about correctness keep holding as the code evolves.

![03122-ezgif com-optimize](https://github.com/user-attachments/assets/260bd90a-59d1-4d5e-aada-4411d2db397b)

## Quickstart

Install from the marketplace:

```
claude plugin marketplace add nicholls-inc/claude-code-marketplace
claude plugin install crosscheck@nicholls
```

Then ask Claude to use one of the orchestrator agents:

```
Use the byfuglien agent to verify your bug fix
Use the hellebuyck agent to scope this repo's assurance reach
```

## The two orchestrator agents

**Byfuglien** (/ˈbʌflɪn/) owns the implementation chain: formal verification with Dafny and semi-formal reasoning over existing code. It covers Layers 1–3 of the assurance hierarchy — code-level correctness, behavioural evidence, and impl-level invariants. Named after Dustin Byfuglien, the crosschecking enforcer: no unsupported claim survives, no unverified code ships.

**Hellebuyck** owns the specification chain: Layers 4–6 of the assurance hierarchy (impl–spec alignment, spec–intent alignment, and spec completeness) plus the governance scaffolding that keeps specs honest as code evolves. Named after Connor Hellebuyck, the goalie — the last line of defence when proof runs out and you have to argue that the spec itself was the right one.

→ For the full handoff seam between the two agents, see [`./docs/agents.md`](./docs/agents.md).

## Skills overview

**Formal verification** — `/spec-iterate`, `/generate-verified`, `/extract-code`, `/lightweight-verify`. Dafny-backed proofs of business logic, with optional lightweight contracts and property-based tests when full proof is overkill.

**Semi-formal reasoning** — `/reason`, `/compare-patches`, `/locate-fault`, `/trace-execution`. Evidence-grounded code analysis adapted from "Agentic Code Reasoning" (Ugare & Chandra, 2026): premises, execution traces, and alternative-hypothesis checks before any conclusion.

**Spec management & adequacy** — `/check-regressions`, `/suggest-specs`, `/rationale`. Keep verified specs from drifting, propose new spec targets, and bridge formal and informal verification with structured adequacy arguments.

**Assurance hierarchy & governance** — nine skills covering Layers 4–6: `/intent-check`, `/spec-adversary`, `/acceptance-oracle-draft`, `/invariant-coverage-scaffold`, `/protected-surface-amend`, `/assurance-layer-audit`, `/assurance-init`, `/assurance-status`, `/assurance-roadmap-check`. Onboard a repo, audit its reach on the ladder, and keep governance notes from rotting.

→ Full skill catalogue with trigger phrases at [`./docs/skills.md`](./docs/skills.md). For the assurance flow specifically, see [`./docs/assurance-hierarchy.md`](./docs/assurance-hierarchy.md).

### Worked examples

One teaser per category — see the linked docs for full usage.

```
/spec-iterate "function that returns the maximum element of a non-empty integer array"
/reason "Is this function thread-safe?" src/cache.py
/rationale src/sort.py "must return a sorted permutation of the input"
/assurance-layer-audit
```

## Research grounding

- [Agentic Code Reasoning](https://arxiv.org/abs/2603.01896) (Ugare & Chandra, 2026) — structured semi-formal reasoning improves LLM accuracy on patch equivalence, fault localization, and code question answering.
- [Abductive Vibe Coding](https://arxiv.org/abs/2601.01199) (Murphy, Babikian & Chechik, 2026) — hierarchical claim trees and Goal Structuring Notation inform `/rationale`.
- [Vibe Coding Needs Vibe Reasoning](https://arxiv.org/abs/2511.00202) (Mitchell & Shaaban, 2025) — autoformalization and continuous verification shape `/suggest-specs` and `/check-regressions`.
- The 6-layer assurance hierarchy framework — see [`./docs/research/assurance-hierarchy.md`](./docs/research/assurance-hierarchy.md).
- Brief literature review of formal verification + AI-assisted code assurance — see [`./docs/research/literature-review.md`](./docs/research/literature-review.md).

## Prerequisites

- **Docker** — required to run the Dafny verifier in an isolated container
- **Node.js** >= 18 — for the MCP server
- **Claude Code** — with plugin support

## Local install

**1. Clone this repo**

```bash
git clone https://github.com/nicholls-inc/claude-code-marketplace.git
```

**2. Build the Docker image**

```bash
./scripts/build-docker.sh
```

This builds a multi-stage Docker image (~300-400MB) with Dafny 4.11.0 and Z3.

**3. Build the MCP server**

```bash
cd mcp-server
npm install
npm run build
```

**4. Install the plugin**

Point Claude Code at this plugin directory:

```bash
claude --plugin-dir ./crosscheck
```

## MCP tools

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
- **On-demand skill loading**: Orchestrator agents read skill SKILL.md files on-demand via the Read tool, keeping baseline context lean

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

## Known limitations

- **IO/networking**: Cannot be formally verified; requires `{:extern}` stubs
- **Concurrency**: Dafny does not model concurrency; only sequential correctness is verified
- **External libraries**: Calls to external libraries are trust boundaries
- **Go generics**: Compile via type erasure to `interface{}`; type assertions may be needed
- **Dafny `real` type**: Compiles to `BigRational`, not native floats
