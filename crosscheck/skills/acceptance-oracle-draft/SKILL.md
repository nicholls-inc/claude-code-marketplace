---
name: acceptance-oracle-draft
description: >-
  Scope the top-N user-observable flows for a repo and emit mechanically-verifiable
  acceptance-scenario skeletons (YAML/JSON) plus a runner-script stub. Does NOT run
  the scenarios — that's CI's job. Enforces a strict mechanical-verification-only
  rule: subjective criteria ("UX feels good") must be quantified or rejected.
  Layer 5 proxy for user-perspective / empirical assurance — measures whether the
  spec was the right spec. Triggers: "acceptance oracle", "draft scenarios",
  "user-observable flows", "acceptance scenarios", "scenario skeletons".
argument-hint: "[optional: target surface — cli | http | daemon | github | ui, or path]"
---

# /acceptance-oracle-draft — User-Perspective Scenario Skeletons

## Description

Layer 5 proxy in the assurance hierarchy — **user-perspective / empirical assurance**. Scopes the top-N user-observable flows for a repo and emits mechanically-verifiable scenario skeletons the user can commit, extend, and wire into CI.

This skill is a methodology doc and scaffold generator. It does **not** execute scenarios; that is the CI job's responsibility. The skill's sole deliverables are: (a) scenario YAML/JSON files, (b) a runner-script stub, (c) a placeholder CI workflow step, and (d) an explicit list of flows that were **rejected** because they could not be mechanically verified.

In Lamport's sense (*Who Builds a House without Drawing Blueprints?*, CACM 2015), an acceptance scenario is a blueprint — the durable artefact the implementation must conform to, written in advance so that imprecision in user-observable behaviour is surfaced before code is written rather than after. The mechanical-verification rule below is what makes the blueprint analogy hold: a blueprint that cannot be checked against the building is not a blueprint.

**CRUCIAL RULE — mechanical verification only.** Every scenario's `then` assertion must be programmatically checkable: exit codes, regex matches, JSON schema validation, HTTP status + body schema, measurable latency thresholds, file existence/content hashes. Subjective criteria ("UX feels good", "response is readable", "user is satisfied") must either be quantified (e.g., "CLI startup < 300 ms measured via `hyperfine`", "response body matches schema `responses/ok.schema.json`") or rejected and reported out-of-scope. The oracle cannot proxy user-perspective assurance if its signals are themselves subjective.

See `references/scenario-schema.md` for the full schema, runner-script pseudocode, and worked rewrites of subjective criteria into mechanical ones.

## Instructions

You are drafting an external acceptance oracle for the user's repo. The oracle sits outside the coding-agent's write loop (scenarios are authored by humans or LLM-drafted then human-approved) and measures user-observable behaviour. Your job is to scope the flows, draft skeletons, and be ruthless about mechanical verifiability.

### Step 1: Identify User-Observable Surfaces

Before proposing any scenario, enumerate the surfaces the repo exposes to external observers. Candidate categories:

- **CLI commands** — binaries users run directly. Signals: exit code, stdout, stderr, side-effect files.
- **Public HTTP/gRPC API endpoints** — request → response. Signals: status code, response body schema, headers, latency.
- **Daemon / background behaviours** — long-running processes. Signals: log lines matching a pattern within a time budget, health-check endpoint status, emitted events on a queue.
- **GitHub integration flows** — webhook handlers, CI bots, PR-comment posters. Signals: emitted comment text matching regex, labels applied, status check posted.
- **UI interactions** — if the repo ships a UI, only the *programmatically-observable* parts count (DOM assertions via Playwright/Selenium, accessibility-tree shape, network calls emitted). Pure-visual assertions are rejected.

Detect surfaces automatically by inspecting repo state (do not ask cold). Check for:

- `cmd/`, `cli/`, `main.go` and any binary entrypoints in `package.json` `bin:`, `pyproject.toml` `[project.scripts]`, etc. → CLI surface.
- `openapi.yaml`/`openapi.json`/`swagger.*`, `routes/`, framework-specific route files (`controllers/`, `*Controller.*`, `*.routes.ts`), gRPC `.proto` files → HTTP/gRPC surface.
- Long-running daemon entrypoints (`systemd` units, Dockerfile `CMD`, `*Daemon*`, `*Worker*`, queue consumers) → daemon surface.
- `.github/workflows/`, webhook-handler routes, `*WebhookHandler*` → GitHub integration surface.
- `packages/*/src`, `apps/web/`, framework UI directories → UI surface (only the programmatically-observable parts count; flag as such).

Report what was detected with file:line evidence per surface. **Default to including every detected surface with N=10** unless the user redirects. Present a single confirmation pass:

> Detected surfaces (default top-N = 10 each):
> - CLI: <evidence path>
> - HTTP: <evidence path>
> - <...>
>
> Reply `keep` to accept, or list the surfaces to include / N override (e.g. `cli only N=5`).

The "do not proceed until the user confirms" gate is removed — agent proceeds with the detected default when the user does not redirect within the same turn. Surface selection is genuinely scope (a real decision) but it does not require a blocking elicitation when the agent's detection is unambiguous.

### Step 2: Pre-fill Flow Candidates per Surface

Mirror `/draft-invariants` §1a discipline: pre-fill flow candidates from repo state with file:line citations, then present them as a single confirmation pass. Do **not** elicit cold by asking the three questions below — those are now the *agent's checklist for what to extract*, not a script for interviewing the user.

For each selected surface, scan the repo for these signals and draft candidates:

- **CLI surface** — `<binary> --help` output (parse subcommands), README quickstart blocks, error message catalog (`grep` for `Error:`, `return fmt.Errorf`, `raise <Class>` with messages), exit code returns. The first-run path comes from the help text + README; error modes come from the error-message catalog.
- **HTTP/gRPC** — OpenAPI / proto definitions (every documented endpoint is a candidate), route handler files for status codes and error responses, existing test files for assertion shapes.
- **Daemon** — startup log lines (`grep` for `log.Info` / `print` at the top of `main`), signal-handler implementations, queue-emission patterns.
- **GitHub integration** — workflow `on:` triggers, webhook handler routes, comment-templates / status-check posters in handler source.
- **UI** — route definitions, e2e test specs (these often already define the mechanical signals — adopt them verbatim).

For each candidate flow, draft:

1. **Happy-path signal:** `<observable signal with file:line evidence>` — agent's first-pass; if no candidate, flag the gap.
2. **At least two error modes:** drafted from grepped error returns; cite the source location for each.
3. **Mechanical success signal:** drafted from API docs / test assertions / CLI help text. This is the **only** field where the agent should defer to a single follow-up question if it cannot extract a concrete predicate from repo state — the success signal is the irreducible governance commitment per Step 7's mechanical-verification rule.

Present the candidate list to the user in a single confirmation pass (per surface):

```
## CLI surface — candidate flows (drafted from <evidence paths>)

| # | Flow name | Happy path | Error modes | Success signal | Priority |
|---|-----------|------------|-------------|----------------|----------|
| 1 | <slug> | <citation>:<lines> | <2+ citations> | <predicate> | P0 |
| 2 | <slug> | ... | ... | ... | P1 |
```

The user red-pens: strikes flows that aren't load-bearing, adjusts priorities, fills in success signals the agent couldn't extract. **Only flows where the success signal cannot be drafted from repo state become an irreducible AskUserQuestion** — the rest go through as confirm-or-edit.

Collect approved flows as `[(surface, flow-name, expected-signal, priority)]`. Priority values: `P0` (blocking regression — must pass for release), `P1` (important), `P2` (nice to have). Keep `P0` flows to roughly 1/3 of the top-N so the gate remains actionable when something fails.

### Step 3: Draft Scenario Skeletons

For each flow, draft a scenario following the schema in `references/scenario-schema.md`. The schema requires:

- `name` — stable kebab-case identifier.
- `surface` — one of the Step 1 categories.
- `priority` — `P0` / `P1` / `P2`.
- `given` — preconditions (fixture files, env vars, repo state). Must be reproducible on a clean CI runner.
- `when` — the action (command line, HTTP request, daemon signal). Must be expressible as a shell invocation or a structured request.
- `then` — the **mechanically-verifiable** assertion. Must compile to a deterministic predicate. See schema doc for the allowed assertion types (`exit_code`, `stdout_matches`, `http_status`, `json_schema`, `latency_under_ms`, `file_exists`, `file_sha256`, `log_line_within_seconds`).
- `timeout` — wall-clock limit in seconds. Defaults: CLI 30s, HTTP 10s, daemon 60s.

Each scenario goes in its own file at `acceptance/scenarios/<flow-name>.yaml` in the target repo (propose this path; the user may override). YAML is the primary format; include a JSON equivalent in the reference doc for teams that prefer JSON.

### Step 4: Emit the Runner-Script Stub

Generate `acceptance/run.sh` (or `acceptance/run.<lang>` if the repo has an obvious test runner — pytest, go test, jest). Default is language-neutral bash + jq walking the scenario files and dispatching per-surface. See `references/scenario-schema.md` §Runner for the pseudocode.

The stub must:

1. Enumerate `acceptance/scenarios/*.yaml`.
2. For each scenario: run `when`, capture stdout/stderr/exit code/timing, evaluate `then` mechanically, report pass/fail with the scenario name and the reason.
3. Exit non-zero if any `P0` scenario failed. `P1`/`P2` failures emit warnings but don't fail the run (configurable via `--strict`).
4. Emit a single JUnit-style XML or JSON report at `acceptance/report.xml` or `.json` so CI can surface it.

Do not implement anything the runner can't support — if the schema permits assertions the stub doesn't handle, annotate `# TODO: implement <assertion>` in the stub and document it in the PR description the user will use.

### Step 5: Emit a Placeholder CI Workflow Step

Generate a `.github/workflows/acceptance-oracle.yml` snippet (or equivalent for GitLab / Circle if the repo uses them) that:

- Triggers on PRs touching the user-observable source paths you identified in Step 1 (emit the `paths:` filter explicitly; do not use a wildcard).
- Runs the oracle runner from Step 4.
- Uploads the report as an artifact.
- Fails the check on any `P0` failure.

This is a placeholder — the user may already have a CI system with established conventions. Mark it clearly as `# GENERATED BY /acceptance-oracle-draft — adjust to match your CI conventions`.

### Step 6: Propose a Storage Boundary

Recommend that scenarios live **outside** the coding-agent's write scope to preserve the external-oracle property (the external-oracle principle: scenarios are authored by humans, not by the agent under test — see the assurance-hierarchy notes if your repo has them checked in under `docs/research/`). Two options:

- **Sibling directory** — e.g. `../<repo-name>-acceptance/` outside the main working tree.
- **Separate repo** — e.g. `<org>/<repo-name>-acceptance`, referenced as a git submodule or CI checkout.

For the initial draft, scenarios can live at `acceptance/scenarios/` inside the repo so the user can review the pattern, but the skill **must** flag this and recommend the user move them before wiring the oracle into the merge gate. Document the move in the PR description.

### Step 7: Explicitly Enumerate Rejected Flows

This is non-negotiable. At the end of the output, print a section titled `## Rejected Flows (out of scope for this oracle)` listing every flow proposed during Step 2 that could not be mechanically verified, with the reason. Example:

```
## Rejected Flows (out of scope for this oracle)

| Flow | Why rejected | Suggested alternative |
|---|---|---|
| "CLI help output is readable" | Subjective; no measurable signal. | Replace with "help output contains sections `USAGE`, `COMMANDS`, `FLAGS` (regex check)" — see §mechanical-rewrites in scenario-schema.md. |
| "Dashboard looks nice" | Pure-visual; no programmatic observable. | Out of scope. Consider manual QA checklist or screenshot-diff tool. |
| "Error message is helpful" | Subjective. | Replace with "error message contains the failing path and a `hint:` prefix (regex check)". |
```

This section is the **coverage boundary** of the oracle. The user needs it to know what the oracle is **not** catching.

### Step 8: Evidence Summary and Decisions for Review

Split the post-draft handoff into two blocks: an Evidence Summary the agent pre-fills with what it just did, and Decisions for Review the human acts on at PR time.

```
## Evidence Summary (agent-verified during this run)

- Mechanical-verification rule: every scenario's `then` assertion was checked against the schema's allowed assertion types (`exit_code`, `stdout_matches`, `http_status`, `json_schema`, `latency_under_ms`, `file_exists`, `file_sha256`, `log_line_within_seconds`). Subjective phrasings were rejected to §Rejected Flows. <N> scenarios passed, <M> were rejected.
- Surfaces covered: <list>. Source-path filters for the CI workflow were derived from the detected surface roots: <list>.
- Runner stub written at acceptance/run.sh; supports all assertion types used by the drafted scenarios. TODOs annotated for any unhandled assertion types.
- Rejected flows enumerated in §Rejected Flows with reasons and suggested mechanical rewrites.

## Decisions for Review (human owns these at PR time)

- [ ] Storage boundary: the initial draft lives at `acceptance/scenarios/` inside the repo for review. Decide whether to move it to a sibling directory or separate repo before wiring the oracle into the merge gate. (See Step 6 for the rationale.)
- [ ] Kill-criterion calibration: confirm that at least 3 of the drafted scenarios are realistically deployable within 4 weeks; if fewer, the top-N is too ambitious and should be re-scoped.
- [ ] 12-week target: confirm 10 scenarios is the right active-count target for this repo's release cadence.
- [ ] False-positive ceiling: confirm the 30% scenario-brittleness ceiling is acceptable; tighten if the repo has a stricter regression policy.
- [ ] Regression test: introduce a deliberate regression in a covered flow and confirm the gate triggers locally (`./acceptance/run.sh` returns non-zero).
```

The Evidence Summary block is closed-loop — the agent fills it in with what it just did and the human reads to verify. The Decisions block is the irreducible human-judgment surface. Do not mix them.

## Arguments

Optional: target surface to scope (`cli`, `http`, `daemon`, `github`, `ui`) or a path hint. If omitted, the skill asks the user to choose in Step 1.

Examples:
- `/acceptance-oracle-draft` — full interactive scoping across all detected surfaces.
- `/acceptance-oracle-draft cli` — scope only the CLI surface.
- `/acceptance-oracle-draft cli/` — scope the CLI surface rooted at `cli/`.

## References

- `references/scenario-schema.md` — YAML/JSON scenario schema, runner-script pseudocode, and worked rewrites of subjective criteria into mechanical ones.
