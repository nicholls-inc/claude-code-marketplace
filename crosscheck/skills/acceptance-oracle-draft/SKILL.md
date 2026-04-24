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

Detect surfaces automatically where possible (look for `cmd/`, `cli/`, `main.go`, `openapi.yaml`, `routes/`, `.github/workflows/`, `packages/*/src`), then ask the user:

> Which surfaces should I include? I detected: `<list>`. Confirm or add others. For each selected surface, the default top-N is 10 (range 3–15).

Do not proceed to Step 2 until the user has confirmed the surface list and N.

### Step 2: Elicit Top-N Flows per Surface

For each selected surface, drive flow elicitation with these three questions (ask them explicitly; don't paraphrase):

1. **"What happens if a new user runs this for the first time?"** — captures happy-path and first-run invariants (config not present, auth not set, etc.).
2. **"What happens on error X?"** — enumerate at least two error modes per surface (missing input, malformed input, missing dependency, network failure, auth failure, permission denied).
3. **"What's the observable success signal?"** — forces the user to name the mechanical signal *before* the scenario is drafted. If they can't name one, the flow is a candidate for rejection in Step 7.

Collect flows as a flat list `[(surface, flow-name, expected-signal, priority)]`. Priority values: `P0` (blocking regression — must pass for release), `P1` (important), `P2` (nice to have). Keep `P0` flows to roughly 1/3 of the top-N so the gate remains actionable when something fails.

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

Recommend that scenarios live **outside** the coding-agent's write scope to preserve the external-oracle property (see `docs/research/assurance-hierarchy.md` §External Acceptance Oracle — scenarios authored by humans, not by the agent under test). Two options:

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

### Step 8: Verification Checklist

Present this checklist alongside the emitted files:

```
## Verification Checklist

After this draft, verify:
- [ ] Every scenario's `then` assertion is programmatically checkable — no "user is satisfied", no "looks right", no "feels fast".
- [ ] At least 3 scenarios are ready to go live within 4 weeks (kill criterion: fewer than 3 = top-N was wrong, re-scope).
- [ ] 10 scenarios are active within 12 weeks.
- [ ] A deliberately introduced regression in a covered flow triggers the gate locally (`./acceptance/run.sh` returns non-zero).
- [ ] Scenarios are stored outside the coding-agent's write scope before the oracle becomes a merge gate (sibling dir or separate repo).
- [ ] Rejected flows are listed explicitly so the coverage boundary is visible.
- [ ] The CI workflow snippet targets the correct `paths:` filter (user-observable source paths only).
- [ ] False-positive rate is monitored — if >30% of failures are scenario brittleness rather than real regressions, simplify the scenarios.
```

Fill in the bracketed items with specifics from the current draft.

## Arguments

Optional: target surface to scope (`cli`, `http`, `daemon`, `github`, `ui`) or a path hint. If omitted, the skill asks the user to choose in Step 1.

Examples:
- `/acceptance-oracle-draft` — full interactive scoping across all detected surfaces.
- `/acceptance-oracle-draft cli` — scope only the CLI surface.
- `/acceptance-oracle-draft cli/` — scope the CLI surface rooted at `cli/`.

## References

- `references/scenario-schema.md` — YAML/JSON scenario schema, runner-script pseudocode, and worked rewrites of subjective criteria into mechanical ones.
