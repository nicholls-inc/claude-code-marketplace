# Reference workflows

Concrete examples of what the 6-layer assurance hierarchy looks like once
it's wired into a real repository. Crosscheck-the-plugin describes the
hierarchy in prose; this directory shows it as YAML, Python, and gh-aw
agentic workflow markdown.

These files are **reference material, not vendored dependencies**. Copy
them into your target repository, adapt the paths and language conventions
to match your stack, then maintain them locally. They are not pulled in
through the plugin install path.

A worked end-to-end walkthrough on a non-proprietary domain (an in-memory
work queue) lives in [`example.md`](./example.md).

## What's here

```
workflows/
├── README.md                                — this file
├── example.md                               — queue-service walkthrough
├── tier-a/                                  — static, no extra prerequisites
│   ├── assurance.yml
│   ├── pre-commit-invariant-coverage.yaml
│   ├── check_invariant_coverage.py
│   ├── scenarios.template.yaml
│   └── run_acceptance.py
└── tier-b/                                  — gh-aw agentic workflows
    ├── assurance-pr-gate.md
    ├── assurance-recheck.md
    ├── assurance-squad.md
    ├── assurance_pr_gate_plan.py
    └── assurance_squad_select.py
```

## Tier A — static enforcement (no LLM, no extra prerequisites)

These are the deterministic Layer 4 and Layer 5 enforcement points. They
run on every push/PR using stock GitHub Actions and pre-commit. No
agentic infrastructure required.

| File | Lives at (in target repo) | What it does | Layer |
|---|---|---|---|
| `assurance.yml` | `.github/workflows/assurance.yml` | CI half of the dual-track invariant-coverage gate | L4 |
| `pre-commit-invariant-coverage.yaml` | snippet for `.pre-commit-config.yaml` | Local half of the dual-track gate (mirrors the CI workflow) | L4 |
| `check_invariant_coverage.py` | `scripts/check_invariant_coverage.py` | Bidirectional check between `docs/invariants/*.md` and `# Invariant <ID>: <name>` test comments | L4 |
| `scenarios.template.yaml` | `docs/assurance/acceptance/scenarios.yaml` | Template for `/crosscheck:acceptance-oracle-draft` output | L5 |
| `run_acceptance.py` | `docs/assurance/acceptance/run_acceptance.py` | Generic runner that loads scenarios.yaml and dispatches to per-scenario runner functions | L5 |

The dual-track principle: every Layer 4 check has both a pre-commit hook
and a CI job. Pre-commit catches drift before it ships; CI catches drift
that bypassed the hook (`--no-verify`, fresh clones, web edits). Both run
the same script so a passing local commit and a passing CI build agree
on what coverage means.

## Tier B — agentic workflows (requires gh-aw)

These are the Layer 5/6 agentic workflows. They run `/crosscheck:` skills
via the [`gh-aw`](https://github.com/githubnext/gh-aw) (GitHub Agentic
Workflows) framework, which compiles agentic markdown sources to GitHub
Actions YAML at install time.

**Prerequisite:** install gh-aw and have it compile the `.md` sources to
`.lock.yml` files in `.github/workflows/`. The lock files are not shipped
here — they are 70-120KB each and are best regenerated locally so they
match your gh-aw version. From the target repo:

```bash
# Install gh-aw (one time)
gh extension install githubnext/gh-aw

# Compile each workflow's .md source to its .lock.yml
gh aw compile .github/workflows/assurance-pr-gate.md
gh aw compile .github/workflows/assurance-recheck.md
gh aw compile .github/workflows/assurance-squad.md
```

| File | Lives at (in target repo) | What it does | Layer |
|---|---|---|---|
| `assurance-pr-gate.md` | `.github/workflows/assurance-pr-gate.md` | Mandatory L5 gate on PRs touching protected surfaces; runs `/crosscheck:intent-check` per changed invariant with content-hashed attestation cache | L5 |
| `assurance-recheck.md` | `.github/workflows/assurance-recheck.md` | Force-recheck handler; bypasses the cache when a contributor comments `/assurance-recheck <ID>` | L5 |
| `assurance-squad.md` | `.github/workflows/assurance-squad.md` | Daily phase-weighted task selector across the 11 assurance lifecycle tasks (audit, scaffold, draft, coverage, acceptance, adversary, kill-criterion alert, Dafny promotion, …) | L4–6 |
| `assurance_pr_gate_plan.py` | `.github/workflows/scripts/assurance_pr_gate_plan.py` | Deterministic Python pre-step for the PR-Gate; computes per-invariant content hashes and cache hits before the agent runs | — |
| `assurance_squad_select.py` | `.github/workflows/scripts/assurance_squad_select.py` | Deterministic Python pre-step for the Squad; reads repo state, computes weights, draws 2 tasks | — |

## Layer mapping at a glance

| Layer | Workflow | Owner |
|---|---|---|
| L4 (deterministic invariant coverage) | `assurance.yml` + pre-commit hook | hellebuyck |
| L4 (governance scaffolding, ROADMAP drift) | `assurance-squad.md` (T2, T6, T11) | hellebuyck |
| L5 (intent-check on changed invariants) | `assurance-pr-gate.md` | hellebuyck |
| L5 (force-recheck) | `assurance-recheck.md` | hellebuyck |
| L5 (acceptance scenarios) | `scenarios.template.yaml` + `run_acceptance.py` | hellebuyck |
| L5 (FP-tracker review) | `assurance-squad.md` (T8, T9) | hellebuyck |
| L6 (spec adversary) | `assurance-squad.md` (T7) | hellebuyck |
| L1–3 (implementation chain) | hand-off announcement only | byfuglien |

Layer 1–3 workflows are deliberately not in this set — Dafny verification
is invoked by the user via `/crosscheck:spec-iterate` →
`/crosscheck:generate-verified` → `/crosscheck:extract-code`, not by a
scheduled workflow. The squad announces hand-offs to byfuglien when a
module's invariant doc sets `dafny_candidate: true`.

## FP-tracker schema — harmonised with `/crosscheck:intent-check`

The squad's kill-criterion check (`assurance_squad_select.py`) and the
PR-Gate / Recheck verdict templates all read the same FP tracker that
`/crosscheck:intent-check` writes:

| Field | Value |
|---|---|
| File path | `.assurance/intent-check-fp-tracker.csv` |
| Spurious marker | `human_verdict == "spurious"` (lowercase, post-`.strip()`) |
| Partial verdict | counted as not-spurious (the pipeline still caught something) |
| Empty `human_verdict` | excluded from both numerator and denominator (awaiting review) |
| Window | rolling 14 days |
| Minimum sample size | `n ≥ 3` before the kill criterion can fire |
| Threshold | FP rate ≥ 30 % |

Schema parity matters: the squad must read what the skill writes, and
both must use the same window so the same rate is shown to the user
across the PR-Gate sticky comment, the Recheck verdict, and the squad's
status dashboard. The initial import of these workflows had a drift
between the squad and the skill (different file path, different verdict
marker, different window) — that drift was harmonised in the same series
of commits that brought the workflows in. See the git log for the
provenance.

The 30 % / 14-day / `n ≥ 3` numbers themselves are founder intuition,
not labelled-pilot data. Tune them for your tolerance once you have ≥30
classified human verdicts.

## How to adapt to your repo

1. **Read [`example.md`](./example.md)** for an end-to-end walkthrough
   on a generic queue-service domain.
2. **Tier A first.** Wire up `assurance.yml` and the pre-commit snippet
   pointing at `check_invariant_coverage.py`. This gives you the Layer 4
   coverage gate with no LLM cost.
3. **Author your invariants.** Use `/crosscheck:assurance-init` and
   `/crosscheck:draft-invariants`. The coverage gate has nothing to do
   until `docs/invariants/*.md` exist.
4. **Tier B when ready.** Once you have ≥3 invariant modules and a
   Layer-4 gate that's been green for a few weeks, install gh-aw and
   compile the Tier B workflows. The squad is daily-cadence by
   default; the PR-Gate fires on protected-surface PRs only.
5. **Calibrate the kill threshold.** The 30 % FP rate / 14-day window /
   `n ≥ 3` minimum are founder intuition, not labelled-pilot data. Tune
   them for your tolerance once you have ≥30 classified human verdicts.

## Adapting the comment convention to other languages

`check_invariant_coverage.py` matches both `# Invariant <ID>: <name>`
(Python, shell) and `// Invariant <ID>: <name>` (Go, TypeScript, Rust).
For other comment styles (`-- ` for SQL, `;; ` for Lisp, etc.), extend
`COMMENT_RE` in the script.

## What this is NOT

- **Not a vendored dependency.** The plugin install path does not pull
  these files in. Copy them into your repo manually.
- **Not auto-generated by skills.** `/crosscheck:invariant-coverage-scaffold`
  generates a similar pre-commit + CI pair from scratch in your repo.
  These references are for cross-checking the shape of that output and
  for hand-rolling extensions.
- **Not the only valid shape.** The dual-track principle, the protected
  surfaces partition, and the 6-layer hierarchy are load-bearing. The
  specific filenames, cron schedules, and gh-aw mechanics are not.
