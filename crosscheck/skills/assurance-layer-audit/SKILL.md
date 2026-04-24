---
name: assurance-layer-audit
description: >-
  Entry-point diagnostic for the 6-layer assurance hierarchy. Inspects a repo,
  detects language and tooling, and emits a per-layer projection table (current
  reach + ecosystem limits) plus a prioritised gap list. Run this FIRST, before
  /assurance-init, to scope realistic reach for your codebase. Triggers:
  "layer audit", "assurance audit", "hierarchy reach", "where am I on the
  assurance ladder", "what can I verify".
argument-hint: "[optional: path to repo root, defaults to cwd]"
---

# /assurance-layer-audit — Repo-Specific Hierarchy Projection

## Description

Pre-onboarding diagnostic for the 6-layer assurance hierarchy (see
`assurance-hierarchy.md` in the research literature). Inspects a repo, detects
its primary language and existing tooling, and emits a repo-specific projection
table for Layers 1–6 — **derived for this repo**, not copied from the canonical
xylem projection. Concludes with a prioritised gap list and a recommended next
step (typically `/assurance-init`).

Run before any other `/assurance-*` skill, so the user knows what reach is
realistic for their ecosystem before they invest in scaffolding.

## Instructions

You are an assurance-architecture reviewer producing a repo-specific projection
of the 6-layer assurance hierarchy. Your output is a diagnostic, not a plan —
the plan comes later via `/assurance-init`. Be honest about limits: a Go repo
cannot address Layer 2, a Python repo loses most of Layer 1, and almost nobody
has Layer 3 end-to-end subgraph verification today. Say so explicitly.

### Step 1: Locate the Repo Root

Resolve the target directory from the argument, or default to the current
working directory. Confirm it is a git repo (`.git/` present). If it is not,
ask the user whether to continue against a non-git directory or abort.

Record the absolute path. Everything below is relative to it.

### Step 2: Detect Primary Language(s)

Detect language(s) by scanning for ecosystem manifests at the repo root and
one level below. Use this mapping:

| Manifest | Language |
|---|---|
| `go.mod`, `go.sum` | Go |
| `pyproject.toml`, `setup.py`, `setup.cfg`, `requirements*.txt`, `Pipfile`, `poetry.lock` | Python |
| `package.json`, `tsconfig.json`, `pnpm-lock.yaml`, `yarn.lock` | TypeScript / JavaScript |
| `Cargo.toml`, `Cargo.lock` | Rust |
| `Gemfile`, `*.gemspec` | Ruby |
| `pom.xml`, `build.gradle`, `build.gradle.kts` | Java / Kotlin |
| `*.csproj`, `*.sln`, `*.fsproj` | C# / .NET |
| `mix.exs` | Elixir |
| `*.cabal`, `stack.yaml` | Haskell |

If multiple manifests are present (e.g., a Go CLI with a TypeScript frontend),
record **all** of them and designate the one with the largest source-file
count as **primary**. All subsequent reasoning anchors on the primary; the
secondary is noted as a separate track in the projection.

If no manifest is recognised, ask the user to name the language and note
"projection assumes user-stated language" in the output.

### Step 3: Detect Existing Tooling

Scan the repo for signals of assurance-relevant tooling. Do not run anything —
file presence is sufficient for this diagnostic.

**Test framework:**
- Go: `*_test.go` files, `testing` imports, `testify` in `go.mod`.
- Python: `tests/` dir, `pytest.ini`, `pyproject.toml` `[tool.pytest]`, `tox.ini`, `conftest.py`.
- TypeScript: `vitest.config.*`, `jest.config.*`, `*.test.ts`, `*.spec.ts`.
- Rust: `#[cfg(test)]` blocks, `tests/` dir, `cargo test` invocations in CI.
- Ruby: `spec/`, `.rspec`, `Rakefile` test tasks.
- Java: `src/test/java/`, JUnit dependencies in `pom.xml` / `build.gradle`.

**Property-based testing:**
- Go: `gopter`, `rapid` imports.
- Python: `hypothesis` in dependencies.
- TypeScript: `fast-check` in `package.json`.
- Rust: `proptest`, `quickcheck` in `Cargo.toml`.
- Ruby: `rantly`.
- Java: `jqwik`.

**Formal verification / proof tooling:**
- Dafny: `*.dfy` files, `.crosscheck/specs.json`, `dafny` mentions in CI.
- Gobra: `*.gobra` files or `// +gobra` annotations in Go.
- Lean: `*.lean`, `lakefile.lean`, `leanpkg.toml`.
- F*: `*.fst`, `*.fsti`.
- Coq / Rocq: `*.v` with `Proof.` / `Qed.`.
- TLA+: `*.tla`, `*.cfg`.

**Governance / hygiene signals:**
- `.pre-commit-config.yaml`, `lefthook.yml`, `.husky/`.
- CI config: `.github/workflows/*.yml`, `.gitlab-ci.yml`, `.circleci/config.yml`, `Jenkinsfile`.
- Existing assurance scaffolding: `docs/assurance/`, `docs/invariants/`, `.claude/rules/protected-surfaces.md`.
- Type checking: `mypy.ini`, `pyrightconfig.json`, `tsconfig.json` `strict: true`.
- Linters: `.golangci.yml`, `ruff.toml`, `.eslintrc*`, `clippy.toml`.

Record every positive signal with its file path so the user can verify.

### Step 4: Derive Per-Layer Reach

For each of the 6 layers, reason about: the canonical reach (what the layer
aspires to), the ecosystem-specific tooling limit, and this repo's reach today
given what Step 3 found. When in doubt, err toward the more pessimistic
reading — the point of the audit is to prevent over-commitment.

Apply the following ecosystem rules verbatim.

**Layer 1 — Formally verified pure code.**
- Go: Reachable for pure, sequential, side-effect-free functions via Dafny
  through the `crosscheck` plugin (`/spec-iterate` → `/generate-verified` →
  `/extract-code`). State explicitly that Dafny has no concurrency model.
- Python: Reachable via Dafny → Python extraction, with the caveat that Python
  has no static type system to carry the guarantees. Extracted code is
  `_dafny.BigRational`-style, not native floats.
- TypeScript / JavaScript: Reachable in principle via Dafny's JS backend, or
  via `lemmafit` (Midspiral's Dafny-to-TS watcher) if installed. Note that
  the `crosscheck` plugin's `/extract-code` skill currently targets Python
  and Go only — TS/JS extraction is not wired through `crosscheck` today.
- Rust: Reachable via Rust-native verifiers (Verus, Prusti, Creusot, Kani for
  bounded model checking). Rust is the only ecosystem with **native**
  verification; Dafny extraction is not the primary route.
- Ruby / Java / Elixir / C#: **No practical Layer 1 tooling** in the
  `crosscheck` plugin today. State explicitly that Layer 1 requires a
  verifier-bridge skill that does not yet exist for this ecosystem.
- Haskell: LiquidHaskell gives refinement types; separate toolchain from
  `crosscheck`.

**Layer 2 — Compilation correctness.**
- Rust: MIR-level verification (Verus, Kani) provides **partial** coverage —
  the verifier reasons about MIR, not the final machine code, so the LLVM
  backend is still in the trusted base. Stronger than Go/Python but not CompCert-grade.
- Go: **Not addressable.** No verified Go compiler exists; the Go toolchain
  is part of the trusted computing base.
- Python / Ruby / JavaScript: **Not addressable.** The interpreter / JIT is
  part of the trusted base.
- Java / C# / Kotlin: **Not addressable** for the VM/JIT in the general case.
- C / C++: CompCert gives a verified C compiler path if the repo actually uses it.
- Never claim Layer 2 reach without a verified-compilation toolchain actually
  present in the repo.

**Layer 3 — Contract graph verification.**
- Go: Pairwise contracts via Gobra possible for narrow scope; end-to-end
  subgraph verification is aspirational (no Go-native contract-graph verifier
  in general use).
- Rust: Prusti / Creusot offer pairwise contract reasoning; a Rust+Lean
  contract-graph-verifier exists as a PoC.
- Python / TypeScript: Runtime contract libraries (`icontract`, `dry-python`,
  `zod`) give dynamic checks, **not formal** subgraph verification. State this.
- Almost always: "Pairwise: partial; end-to-end: aspirational" is the honest
  reading.

**Layer 4 — Implementation–spec alignment.**
- Available whenever Layer 1 is available. Delivered by re-running the
  verifier on changed spec/code. For `crosscheck`-based stacks, the hook is
  `dafny_verify` on touched `.dfy` files gated by a pre-commit hook or CI job.

**Layer 5 — Spec–intent alignment.**
- Portable via the forthcoming `/intent-check` skill (two-LLM round-trip
  back-translation). Probabilistic (~96% on curated benchmarks); expect
  unknown real-PR performance until the skill is running in the repo.
- Available in principle for any language — the check operates on prose
  invariants, covering tests, and code diffs, not on the language itself.

**Layer 6 — Spec completeness.**
- Best-effort. Delivered by `/spec-adversary` (candidate-invariant proposals)
  and `/acceptance-oracle-draft` (mechanically verifiable user-flow scenarios).
- No ecosystem limit beyond the absence of an operational `docs/invariants/`
  tree — both skills need at least one module invariant doc to anchor against.

### Step 5: Emit the Projection Table

Produce a markdown section titled `## Assurance Hierarchy — <repo> Projection`
that mirrors the xylem projection format. For each layer, emit a row with:

- **Layer #** and short name.
- **Hierarchy reach** — the canonical one-line description (what the layer
  aspires to).
- **This repo's reach today** — derived from Steps 2–4. State tooling limits
  explicitly.

Example shape (derive the content; do not copy xylem's values):

```
## Assurance Hierarchy — <repo> Projection

**Primary language:** Go (secondary: TypeScript frontend, tracked separately).

**Layer 1 (formally verified pure code).** Reachable for pure, sequential
functions via Dafny through the `crosscheck` plugin. No `.dfy` files or spec
registry present today — Layer 1 is **available but not yet used**. Candidates
likely in `internal/queue/` and `internal/retry/` based on file names; confirm
via `/suggest-specs`.

**Layer 2 (compilation correctness).** **Not addressable.** No verified Go
compiler exists; `gc` is part of the trusted computing base.

**Layer 3 (contract graph verification).** Pairwise contracts via Gobra
possible near-term for a narrow module (not installed). End-to-end subgraph
verification aspirational — no Go-native contract-graph verifier in general
use.

**Layer 4 (implementation-spec alignment).** Available the moment Layer 1
lands. Requires `dafny_verify` wired into pre-commit or CI on touched `.dfy`
files. Not wired today.

**Layer 5 (spec-intent alignment).** Available via `/intent-check` once
`docs/invariants/` exists with at least one module doc. Probabilistic (~96%);
enforce the 30% FP kill criterion from the start.

**Layer 6 (spec completeness).** Best-effort. Blocked on the same prerequisite
(`docs/invariants/`). `/spec-adversary` and `/acceptance-oracle-draft`
will be runnable after `/assurance-init`.
```

Derive every bullet from what Steps 2–4 actually found. If the repo already
has `.dfy` files, say so. If it has `docs/invariants/` already populated, say
so and recommend `/assurance-status` rather than `/assurance-init`.

### Step 6: Prioritised Gap List

Emit a top-5 gap list, ordered by expected payoff per hour of effort, not by
layer number. For each gap, state: the problem, the recommended next skill,
and the expected layer reach delta.

Use this table shape:

```
## Top 5 Gaps (prioritised)

| # | Gap | Recommended skill | Expected reach delta |
|---|-----|-------------------|----------------------|
| 1 | No governance skeleton (`docs/assurance/`, `.claude/rules/protected-surfaces.md`) | `/assurance-init` | Unblocks Layers 4–6 |
| 2 | No module invariant docs (`docs/invariants/`) | `/draft-invariants` on 2–3 load-bearing modules | Unblocks Layers 5–6 |
| 3 | No coverage gate (invariant-ID ↔ test-comment) | `/invariant-coverage-scaffold` | Mechanical Layer 4 enforcement |
| 4 | No Layer 1 kernel candidates pinned | `/suggest-specs` on `internal/queue/`, `internal/retry/` | Makes Layer 1 concrete |
| 5 | No spec-intent alignment pipeline | `/intent-check` (after gap 2) | Turns on Layer 5 |
```

Prioritisation heuristics:
- Governance scaffolding first — nothing else works without `docs/assurance/`
  and the protected-surface file.
- Invariant docs before any LLM-backed verification — `/intent-check` and
  `/spec-adversary` both anchor on them.
- Coverage gate before Layer 1 kernel work — a verified kernel with no
  coverage gate drifts silently.
- Layer 1 kernel work before Layer 5 probabilistic checks — deterministic
  assurance first, probabilistic assurance second.
- Layer 6 work last — best-effort skills benefit from every preceding layer.

Cap the list at five items. If there are fewer, emit fewer; if more, keep the
top five and note how many were dropped.

### Step 7: Recommend the Next Step

End with a single-line recommendation that names the next skill and any
adjustments implied by the gap list:

- **Onboarding-gate-failing repos** (missing `docs/assurance/` etc.):
  "Next: `/assurance-init` — this repo has no governance skeleton yet."
- **Partially onboarded repos** (some scaffolding present): "Next:
  `/assurance-status` — partial scaffolding detected; run the onboarding gate
  before scaffolding more."
- **Fully onboarded repos** (everything present): "Next: `/assurance-status`
  for the Phase 2 dashboard, then `/assurance-roadmap-check` for drift."

Attach any audit-specific adjustments, e.g., "Skip `/invariant-coverage-scaffold`
until after `/draft-invariants` runs on at least one module," or "Language
detected as Ruby — `/invariant-coverage-scaffold` v1 does not yet cover Ruby;
hand-roll a coverage script for now."

### Step 8: Verification Checklist

```
## Verification Checklist

- [ ] Primary language detected with a cited manifest path (e.g., `go.mod:1`)
- [ ] Secondary languages recorded separately, not merged into the primary projection
- [ ] Existing tooling signals cited with file paths, not just named
- [ ] Per-layer projection derived from the detected tooling, not copied from the xylem canonical
- [ ] Every "not addressable" claim names the specific missing tool (e.g., "no verified Go compiler")
- [ ] Rust repos note MIR-level partial Layer 2 reach rather than silent omission
- [ ] Python / Ruby / Java / C# repos explicitly state Layer 1 is unreachable without a verifier bridge
- [ ] Gap list capped at 5, ordered by payoff/hour, each row names a concrete next skill
- [ ] Final recommendation names the next skill and any adjustments from the gap list
- [ ] Output is diagnostic, not prescriptive — no scaffolding written by this skill
```

## Arguments

Optional path to the repo root. Defaults to the current working directory.

Examples:
- `/assurance-layer-audit` — audit the current directory
- `/assurance-layer-audit ~/repos/my-service` — audit a specific repo
- `/assurance-layer-audit .` — explicit cwd (same as no argument)
