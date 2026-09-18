# pr-swarm — Claude Code Plugin

A router-first, multi-agent pull request review swarm and automated review triage system for Claude Code.

The plugin provides two cooperative skills:
1. **`/pr-swarm`** (Read/Review): Orchestrates a multi-agent review pipeline across specialized lenses, dedupes findings, and posts structured inline comments alongside a persistent sticky summary.
2. **`/review-triage`** (Write/Triage): Acts on review feedback, reconciles unresolved threads, strictly preserves human review discussions, commits automated fixes with loop-prevention trailers, and verifies auto-merge preconditions.

---

## Architecture Overview

`pr-swarm` divides review responsibilities into an asymmetric read/write pair:

```
                  ┌─────────────────────────────────┐
                  │           Pull Request          │
                  └───────────────┬─────────────────┘
                                  │
          ┌───────────────────────┴───────────────────────┐
          │                                               │
          ▼ (Read only)                                   ▼ (Write / Triage)
   /pr-swarm                                       /review-triage
┌───────────────────────────┐                   ┌───────────────────────────┐
│ 1. Haiku Router Pass      │                   │ 1. Fetch & Paginate Units │
│ 2. Scoped Delegations:    │                   │ 2. Human Participation    │
│    • security             │                   │    Gate (INV-3)           │
│    • test-theatre         │                   │ 3. Classification:        │
│    • xp                   │                   │    • Actionable           │
│    • crosscheck/byfuglien │                   │    • Nit                  │
│    • crosscheck/hellebuyck│                   │    • Ambiguous (Ladder)   │
│ 3. Cross-lens Dedupe      │                   │ 4. Fix, Commit, & Push    │
│ 4. Sticky Summary Upsert  │                   │ 5. Reply & Resolve        │
│ 5. Inline Comments        │                   │ 6. Precondition Checks &  │
│    (event=COMMENT only)   │                   │    Auto-Merge Arming      │
└───────────────────────────┘                   └───────────────────────────┘
```

---

## Installation

Install from the marketplace:

```bash
claude plugin install pr-swarm@nicholls
```

Or load directly from a local marketplace repository checkout.

---

## Skills

### 1. `/pr-swarm` — Router-First PR Review

Reviews a single pull request using a cheap-first router that selectively delegates risky hunks to specialized model lenses.

#### Usage

```bash
/pr-swarm <pr-ref> [--head-sha <sha>] [--round <n>] [--dry-run]
```

#### Arguments

- `<pr-ref>` *(required)*: Pull request number (e.g. `1870`) or full GitHub pull request URL. If omitted in direct human invocations, attempts to detect the PR for the current Git branch.
- `--head-sha <sha>`: Commit SHA to anchor findings against.
- `--round <n>`: Round number (defaults to `1`).
- `--dry-run`: Runs the full review pipeline, computes findings and verdict, but writes **zero** bytes to GitHub. Outputs the verbatim rendered inline comments, sticky summary markdown, and exact commands that would have been executed.

#### Review Workflow

1. **Diff Resolution**: Fetches PR metadata, full diff, and existing PR comments (inline and issue level).
2. **Automated Comment Indexing**: Catalogs existing automated comments (bots, pr-swarm headers, and Crosscheck signatures) to avoid re-raising covered points.
3. **Router Pass (`haiku`)**: Evaluates the full diff, assigns a danger grade (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`), and establishes a delegation plan.
   - *Mandatory Delegation*: Triggered automatically if diff > 400 lines, or touches auth, billing, database migrations, concurrency, or CI/CD paths.
4. **Delegation Pass (`sonnet` / `opus`)**: Dispatches up to 6 parallel, awaited lens reviewers scoped to specific hunks.
5. **Synthesis & Deduplication**: Merges overlapping findings across reviewers, identifies convergence, dedupes against existing automated comments on the PR, and computes a risk verdict.
6. **Posting**:
   - Inline comments posted in a single batch review with `event=COMMENT`.
   - Single sticky summary upserted (`<!-- pr-swarm-summary -->`) with verdict, findings, convergence, and coverage status.
7. **Structured Contract**: Returns a typed JSON contract containing `head_sha`, `round`, `verdict`, `findings`, and `degradations`.

---

### 2. `/review-triage` — Feedback Triage & Convergence Loop

Processes review comments and unanchored findings, resolves feedback within approved autonomy boundaries, commits fixes, and gates merge readiness.

#### Usage

```bash
/review-triage <pr-ref> [--max-rounds <n>] [--head-sha <sha>] [--dry-run] [--merge-strategy <strategy>]
```

#### Arguments

- `<pr-ref>` *(required)*: Pull request number or URL.
- `--max-rounds <n>`: Maximum triage rounds (defaults to `3`).
- `--head-sha <sha>`: Head commit expected at start.
- `--dry-run`: Performs triage analysis without modifying code, pushing commits, resolving threads, or arming auto-merge.
- `--merge-strategy <strategy>`: Merge strategy (`squash`, `merge`, or `rebase`; defaults to repository standard, typically `squash`).

#### Triage Workflow

1. **Unit Ingestion**: Collects unresolved review threads and unanchored pr-swarm findings.
2. **Human-Participation Gate (INV-3)**: Checks whether any human participated in the thread. Human threads are **strictly deferred** without automated resolution or reply.
3. **Classification**:
   - **Actionable**: High/Critical concrete findings with localized fixes.
   - **Nit**: Low-severity items resolved with an explanatory note.
   - **Ambiguous**: Routed through the autonomy ladder (*Just do it*, *Do it but recommend and ask*, *Stop and ask*).
4. **Fix Application**: Applies verified code changes, commits with the trailer:
   ```
   PR-Swarm-Automated: true
   ```
   and pushes to the PR branch.
5. **Thread Resolution**: Replies to bot/swarm threads with commit SHAs and marks threads resolved.
6. **Auto-Merge Preconditions**: Verifies required checks, branch protection, dismiss-stale-approvals settings, and arms auto-merge only when all safety criteria are met.

---

## Review Lenses & Personas

Prompts for all review personas are self-contained under `skills/pr-swarm/references/lenses.md`:

| Lens | Focus Areas |
| --- | --- |
| `security` | Authentication/authorization, IDOR, SQL/command/prompt injection, SSRF, secrets, crypto, data exposure. |
| `test-theatre` | Tautological tests, missing assertions, mock soup, overly loose checks, testing the framework instead of the change. |
| `xp` | Extreme Programming rules of simple design: intent revelation, duplication elimination, cohesion, YAGNI. |
| `crosscheck/byfuglien` | Correctness, fault paths, patch equivalence, verification adequacy, concurrency. (Vendored from the marketplace's `crosscheck` plugin). |
| `crosscheck/hellebuyck` | Specification coverage, intent alignment, governance, protected surfaces, target invariant collision check. (Vendored from the marketplace's `crosscheck` plugin). |

---

## Safety Guarantees & Invariants

1. **Never Approve**: Neither `pr-swarm` nor `review-triage` will ever submit an approving review to GitHub. Reviews are posted with `event=COMMENT`. Human approval remains the sovereign gate.
2. **No Unsolicited Human Interruption**: Automated tools never resolve, reply to, or close threads where a human has participated.
3. **Loop Prevention Guard**: All triage commits include the `PR-Swarm-Automated: true` Git trailer to allow CI workflow self-push guards to prevent infinite trigger loops.
4. **Single Sticky Summary**: Exactly one top-level summary comment is maintained per PR via idempotent upserts (`<!-- pr-swarm-summary -->`).
5. **Strict Auto-Merge Requirements**: Auto-merge is only armed if the repository permits it, required checks pass, stale approvals are dismissed on push, and human reviews are satisfied.

---

## Reference Documentation

- [`skills/pr-swarm/references/github-ops.md`](skills/pr-swarm/references/github-ops.md): Canonical `gh` CLI and GraphQL commands with pagination, scratch file handling, and verification records.
- [`skills/pr-swarm/references/lenses.md`](skills/pr-swarm/references/lenses.md): Verbatim prompt definitions for all reviewer lenses.
- [`skills/review-triage/references/triage-rubric.md`](skills/review-triage/references/triage-rubric.md): Human participation gate rules, classification rubric, and autonomy ladder.
