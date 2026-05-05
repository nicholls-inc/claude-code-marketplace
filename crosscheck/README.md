# Crosscheck

AI agents write plausible code fast. Plausible isn't the same as correct, and "looks fine" doesn't survive contact with production. Crosscheck gives you six layers of progressively stronger correctness checks, and four of them are shipping today.

Crosscheck checks Claude's code claims with two orchestrator agents — `byfuglien` (implementation) and `hellebuyck` (specification) — coordinating three pillars of assurance. The first pillar is formal verification with Dafny — code is verified against its spec, then compiled to Python or Go via the Dafny backends. Layer 1 covers the verified core; embedding it correctly in your application is your responsibility, and Layer 2 of the hierarchy (compilation correctness) treats the Dafny-to-Python/Go backend as part of your trusted computing base — see [`./docs/research/assurance-hierarchy.md`](./docs/research/assurance-hierarchy.md). The second pillar is semi-formal reasoning, which forces evidence-grounded certificates before any conclusion about a piece of code. The third is a 6-layer assurance hierarchy with governance scaffolding, so claims about correctness keep holding as the code evolves.

![03122-ezgif com-optimize](https://github.com/user-attachments/assets/260bd90a-59d1-4d5e-aada-4411d2db397b)

Why this matters for AI-driven development: the failure mode of an LLM coder isn't bad syntax — it's silent semantic drift, deleted guarantees, and under-specified contracts. Every layer here is a different machine-checkable trap for exactly that.

**Recommended order** (the default workflow, evaluations first):

1. `/assurance-layer-audit` — diagnose what's actually reachable in your repo before you commit to any layer.
2. `/assurance-init` — scaffold the governance skeleton: ROADMAP, protected surfaces, seed invariant docs.
3. `/acceptance-oracle-draft` — lock the user-observable flows into mechanically-verifiable scenarios. Treat as a user-perspective oracle that sits alongside the hierarchy (the docs file it under "Supporting Workflow Elements" — see [`./docs/research/assurance-hierarchy.md`](./docs/research/assurance-hierarchy.md)). Highest-leverage starting point per the literature on intent formalization (Lahiri 2026; Phoenix-style "evaluations as the codebase").
4. `/intent-check` and `/invariant-coverage-scaffold` — pin invariant prose to covering tests so the two cannot drift, and run round-trip intent verification on AI-drafted invariants (~96% accuracy on curated benchmarks; real-PR accuracy unmeasured — see [`./docs/research/assurance-hierarchy.md`](./docs/research/assurance-hierarchy.md)).
5. **Optional, when the code shape supports it:** `/spec-iterate` → `/generate-verified` → `/extract-code` for Dafny-verified cores. Empirical reach for Layer 1 is roughly 22–27% of typical full-stack codebases ([`./docs/research/logic-distribution-analysis.md`](./docs/research/logic-distribution-analysis.md)); reach for your repo will vary. `/lightweight-verify` adds contracts and property tests where full proof is overkill.
6. `/spec-adversary` once a module has a ratified invariant doc — iterative best-effort probing for undocumented properties; it has its own signal-to-noise kill criteria (see the SKILL).
7. **Weekly cadence:** `/assurance-status` for drift detection and `/assurance-roadmap-check` to keep ROADMAP item statuses honest.

Layers shipping today (1, 4, 5, 6) plus the orthogonal semi-formal reasoning skills are summarised in the persona table below; full per-skill catalogue at [`./docs/skills.md`](./docs/skills.md).

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

**Byfuglien** (/ˈbʌflɪn/) owns the implementation chain: formal verification with Dafny and semi-formal reasoning over existing code. It owns Layer 1 (Dafny formal verification) and the regression-detection slice of Layer 4 (`/check-regressions`). Layers 2–3 are deliberately not addressed — Layer 2 is a trusted-computing-base concern, Layer 3 (contract graph verification) is a research frontier outside niche stacks; see [`./docs/research/assurance-hierarchy.md`](./docs/research/assurance-hierarchy.md). Byfuglien also owns the four semi-formal reasoning skills, which sit outside the layer hierarchy entirely. Named after Dustin Byfuglien, the crosschecking enforcer: no unsupported claim survives, no unverified code ships.

**Hellebuyck** owns the specification chain: Layers 4–6 of the assurance hierarchy (impl–spec alignment, spec–intent alignment, and spec completeness) plus the governance scaffolding that keeps specs honest as code evolves. Named after Connor Hellebuyck, the goalie — the last line of defence when proof runs out and you have to argue that the spec itself was the right one.

The split is principled in shape but **asymmetric in substance** — Byfuglien anchors a single deterministic layer and four orthogonal reasoning skills; Hellebuyck carries three layers (deterministic → probabilistic → best-effort) plus governance. The table below names the skill ownership directly so the asymmetry is legible.

| Byfuglien (impl chain) | Hellebuyck (spec chain) | Orthogonal: semi-formal reasoning |
|---|---|---|
| Layer 1: `/spec-iterate`, `/generate-verified`, `/extract-code`, `/lightweight-verify`, `/suggest-specs` | Layer 4 (alignment): `/invariant-coverage-scaffold`, `/protected-surface-amend` | `/reason` |
| Layer 4 (regression): `/check-regressions` | Layer 5: `/intent-check`, `/acceptance-oracle-draft` | `/compare-patches` |
| Spec-management bridge: `/rationale` | Layer 6: `/spec-adversary` | `/locate-fault` |
|  | Governance index: `/assurance-init`, `/assurance-layer-audit`, `/assurance-status`, `/assurance-roadmap-check` | `/trace-execution` |

The four semi-formal reasoning skills are not part of the 6-layer hierarchy. They are evidence-grounded code-analysis tools adapted from Ugare & Chandra (2026) and live as a third axis — Byfuglien-routed because they reason about implementation, but layer-agnostic.

→ For the full handoff seam between the two agents, see [`./docs/agents.md`](./docs/agents.md).

## Skills overview

The persona table above maps every skill to its orchestrator and (where applicable) layer. The shape of the catalogue, by purpose:

**Formal verification** — Dafny-backed proofs of business logic, with optional lightweight contracts and property-based tests when full proof is overkill.

**Semi-formal reasoning** — evidence-grounded code analysis adapted from "Agentic Code Reasoning" (Ugare & Chandra, 2026): premises, execution traces, and alternative-hypothesis checks before any conclusion. Sits outside the 6-layer hierarchy.

**Spec management & adequacy** — keep verified specs from drifting, propose new spec targets, and bridge formal and informal verification with structured adequacy arguments.

**Assurance hierarchy & governance** — onboard a repo, audit its reach on the ladder, draft acceptance oracles, run the round-trip intent check, and keep governance notes from rotting.

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
