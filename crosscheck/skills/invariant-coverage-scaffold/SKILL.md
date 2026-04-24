---
name: invariant-coverage-scaffold
description: >-
  Generate a bidirectional invariant-to-test coverage gate for a repo that uses
  docs/invariants/*.md plus `// Invariant <ID>: <Name>` test comments. Emits both a
  pre-commit hook and a CI job (dual-track enforcement), adapted to the target
  repo's language and test framework. v1 supports Go, Python, TypeScript;
  Rust/Ruby/Java are deferred to later versions. Triggers: "invariant coverage",
  "coverage gate", "scaffold invariant check", "enforce invariant tests".
argument-hint: "[optional: language override — go | python | typescript]"
---

# /invariant-coverage-scaffold — Invariant Coverage Gate Generator

## Description

Scaffold the mechanical gate that enforces the bidirectional link between invariant IDs declared in `docs/invariants/*.md` and `// Invariant <ID>: <Name>` comments in property-test files (e.g. `// Invariant I1: QueueEnqueueDequeue`). The gate catches two failure modes:

1. **Silent test drop.** An invariant exists in the doc but no test references it → coverage is claimed that does not exist.
2. **Orphan test comment.** A test references an invariant ID that is not declared in any module doc → stale or typo'd reference.

This skill generates both enforcement points required by the dual-track enforcement principle:

- A **pre-commit hook** (< 5 s, fast, actionable error) that blocks the commit locally.
- A **CI job** that runs on every PR regardless of how the change was authored.

The skill is a generator, not a runtime. It emits scripts and config files adapted to the target repo's language and tooling, then wires them into the existing hook framework and CI system.

**Language coverage (v1):** Go, Python, TypeScript. **Deferred to later versions:** Rust, Ruby, Java. If the target repo is one of the deferred languages, emit a clear notice and stop — do not fabricate a template.

**Boundary vs `/assurance-init`:** `/assurance-init` scaffolds the governance skeleton (ROADMAP, horizon dirs, `docs/invariants/` with seed module docs, `.claude/rules/protected-surfaces.md`). This skill installs the mechanical coverage gate on top. Expected ordering:

1. `/assurance-init` (or the equivalent manual setup) — creates `docs/invariants/*.md` with `**I1. Name.**` headers.
2. `/draft-invariants` for each seeded module — populates real invariant prose.
3. `/invariant-coverage-scaffold` (this skill) — wires the enforcement gate.

If `docs/invariants/` is missing when this skill runs, stop and recommend `/assurance-init` before proceeding. Do not create `docs/invariants/` as a side effect — that is `/assurance-init`'s responsibility.

## Instructions

You are generating a portable invariant-coverage gate. Your output is committed into the user's repo. Be conservative: detect before you write, show your choices before you emit files, and always emit both enforcement points.

### Step 1: Detect Target Language

Check for marker files in this priority order:

1. `go.mod` at the repo root → **Go**
2. `pyproject.toml` (or `setup.py`, `setup.cfg`) at the repo root → **Python**
3. `package.json` at the repo root → **TypeScript** (if `tsconfig.json` also present) or JavaScript

If **multiple markers** are found (e.g., a Go backend with a TypeScript frontend), list each candidate and ask the user which to prioritise. Do not silently pick one.

If **none** of Go/Python/TypeScript markers are found, stop and emit:

```
Unsupported language for v1. Found: <detected markers or 'none'>.
v1 supports Go, Python, TypeScript. Rust, Ruby, Java are deferred to later versions.
If the target language has a similar doc-plus-test layout, a follow-up skill version
can add it — file a request citing this skill.
```

If the user passes an explicit language argument (`go`, `python`, `typescript`), respect it and skip detection.

### Step 2: Detect Test Framework

Given the chosen language, identify the property-test framework the repo already uses so the generated script looks for the right file glob.

| Language | Primary framework | Secondary | Default test glob |
|---|---|---|---|
| Go | `testing` stdlib with `gopter` or `rapid` | plain `testing` | `**/*_invariants_prop_test.go`, fallback `**/*_test.go` |
| Python | `pytest` + `hypothesis` | plain `unittest` | `tests/**/*.py`, fallback `**/*_test.py` |
| TypeScript | `vitest` or `jest` + `fast-check` | plain `vitest`/`jest` | `**/*.test.ts`, `**/*.spec.ts` |

Detect by grepping for the framework's import in existing tests. Report what you found: "Detected `github.com/leanovate/gopter` imports in 4 files — using gopter-style test glob."

If no tests exist yet, pick the default glob and note: "No existing property tests found. Generated script uses the default glob `<glob>`; adjust in `scripts/check_invariant_coverage.<ext>` if your convention differs."

### Step 3: Detect Pre-commit Framework

Check, in order:

1. `.pre-commit-config.yaml` → **pre-commit.com**
2. `lefthook.yml` / `lefthook.yaml` → **lefthook**
3. `package.json` with a `husky` devDependency or `.husky/` directory → **husky**
4. None of the above → **standalone**

For `standalone`, emit a warning and still generate a shell-executable `scripts/pre-commit-invariant-coverage.sh` that the user can wire up manually (e.g., via a `git config core.hooksPath` or direct `.git/hooks/pre-commit` entry). Document this in the final report.

### Step 4: Detect CI System

Check, in order:

1. `.github/workflows/` → **GitHub Actions**
2. `.gitlab-ci.yml` → **GitLab CI**
3. Neither → **generic**

For `generic`, emit a `ci/invariant-coverage.sh` shell stub the user can call from whatever CI they have, with a warning noting the CI integration is manual.

### Step 5: Confirm Plan With User

Before writing any files, print a summary and ask for confirmation:

```
Plan:
  Language:          <go | python | typescript>
  Test framework:    <detected framework>
  Test glob:         <glob>
  Pre-commit:        <pre-commit.com | lefthook | husky | standalone (warning)>
  CI:                <github-actions | gitlab-ci | generic (warning)>
  Files to create:
    - scripts/check_invariant_coverage.<ext>
    - <hook config snippet appended to <hook-config-file>>
    - <ci workflow file or step>
  Files to read-only inspect:
    - docs/invariants/*.md
    - <test files matching glob>
Proceed? [y/N]
```

Wait for explicit confirmation. If the user declines, stop cleanly — do not write partial output.

### Step 6: Emit Coverage Script

Use the language-specific template under `references/`:

- Go → `references/go-template.md`
- Python → `references/python-template.md`
- TypeScript → `references/typescript-template.md`

Each template contains:

- The script body (invariant-ID parser for `docs/invariants/**/*.md`; test-comment grep for the language's test glob).
- Placeholders (marked `<placeholder>`) for repo-specific values: test glob, output path, allowed invariant-ID prefix regex.
- The pre-commit config snippet for every supported hook framework.
- The CI workflow snippet for GitHub Actions and GitLab CI.
- The actionable error message (including the exact fix command) the script prints on failure.

If a coverage script already exists at the target path, **do not overwrite**. Diff the existing content against the template, show the user the differences, and ask before replacing. On a clean re-run with no drift, skip the write and report "script already up to date".

Fill the placeholders using the choices from Steps 1–4. Write the script to the standard location:

- Go → `scripts/check_invariant_coverage.go` (as a `//go:build ignore` tool) or `cli/cmd/check-invariants/main.go` if a `cli/` tree already exists.
- Python → `scripts/check_invariant_coverage.py` (executable, `#!/usr/bin/env python3`).
- TypeScript → `scripts/check-invariant-coverage.ts` (runnable via `npx tsx` or `node --loader`).

Make the script executable (`chmod +x` equivalent) if applicable for the language.

### Step 7: Wire In The Pre-commit Hook

Append the hook entry from the template to the detected hook config. Do **not** rewrite the file — append or insert, preserving existing hooks. For each framework:

- **pre-commit.com** — add a `repos[].hooks[]` entry with `id: invariant-coverage`, `files` regex covering `docs/invariants/.*\\.md$` and the test glob.
- **lefthook** — add a `pre-commit.commands.invariant-coverage` entry.
- **husky** — add a line to `.husky/pre-commit` (creating the file if needed).
- **standalone** — write `scripts/pre-commit-invariant-coverage.sh` and print the exact `git config core.hooksPath` command the user needs to run.

The hook must run in **under 10 seconds on a repo with ~200 invariants** (see Verification Checklist). If the script is slower, optimise the parser before shipping.

### Step 8: Wire In The CI Job

Add a CI job or step that calls the same script. Use the CI snippet from the template.

- **GitHub Actions** — if `.github/workflows/ci.yml` or similar exists, add a step. If none exist, create `.github/workflows/invariant-coverage.yml` as a minimal standalone workflow.
- **GitLab CI** — add a job to `.gitlab-ci.yml`.
- **Generic** — emit `ci/invariant-coverage.sh` with a clear "Call this from your CI pipeline" comment.

Before adding a step or job, **grep the target file for an existing `invariant-coverage` entry**. If one is already present, skip the write and report "CI step already wired". Never append a duplicate step on a re-run.

The CI step must fail the build (`exit 1`) when the script fails. Do not downgrade to warning-only — that defeats dual-track enforcement.

### Step 9: Declare The Opt-out Marker

Aspirational invariants (declared but not yet test-covered) use the literal HTML comment `<!-- aspirational -->` on the same line as the invariant header. This marker is **skipped by the missing-coverage check**; the orphan-comment check still applies.

The script must parse and respect this marker. The regex for detection (HTML comment, possibly preceded by whitespace) is included in every template.

Document the marker convention by appending a short note to `docs/invariants/README.md` if one exists, or emitting a standalone `docs/invariants/COVERAGE.md` if it does not. Do **not** overwrite existing coverage prose — append only.

### Step 10: Report

Emit a concise report to the user:

```
Created:
  - scripts/check_invariant_coverage.<ext>
  - <hook-config-changes>
  - <ci-workflow-changes>
  - <doc-append-or-create>

Verification:
  Run the script locally:
    $ <exact command>
  Expected output on a healthy repo: "invariant coverage OK"
  Expected output on a gap: "missing coverage for <ID>" or "orphan comment at <file:line>"

Next steps:
  - Commit the new files.
  - Run /draft-invariants on any module that has no invariant doc yet.
  - Add `// Invariant <ID>: <Name>` comments to property tests that lack them.
```

### Verification Checklist

Present this checklist alongside the generated files so the user can confirm the gate is correctly wired:

```
## Verification Checklist

Run each item against the generated script before considering the gate live:

- [ ] Invariants declared in `docs/invariants/**/*.md` without a covering test → script exits non-zero and prints the missing ID.
- [ ] Test comments referencing non-existent invariant IDs → script exits non-zero and prints the orphan file:line.
- [ ] Invariants marked `<!-- aspirational -->` are skipped by the missing-coverage check but orphan-comment checks still apply.
- [ ] Script runs in under 10 seconds on a repo with ~200 invariants (measure with `time <command>`).
- [ ] Pre-commit hook is wired and fires on changes to `docs/invariants/**/*.md` or property-test files.
- [ ] CI job is wired and fails the build on coverage gaps (verify with a throwaway PR that removes a test comment).
- [ ] Error messages include the exact fix command (e.g., `add \`// Invariant <ID>: <Name>\` to a test`).
- [ ] Regex format is strict: `// Invariant <ID>: <Name>` is the only accepted comment form; `// Invariant: <ID>` and similar variations are rejected.
- [ ] Invariant IDs contain a digit by construction (typos that drop the digit surface as missing-coverage on the original ID, not as orphans — documented behavior).
```

## Arguments

Optional language override. If omitted, the skill auto-detects.

Examples:
- `/invariant-coverage-scaffold` — auto-detect language and framework.
- `/invariant-coverage-scaffold go` — force Go template.
- `/invariant-coverage-scaffold python` — force Python template.
- `/invariant-coverage-scaffold typescript` — force TypeScript template.

## References

- `references/go-template.md` — Go script, hook configs, CI snippets.
- `references/python-template.md` — Python script, hook configs, CI snippets.
- `references/typescript-template.md` — TypeScript script, hook configs, CI snippets.

Each reference file contains the script template, the pre-commit hook config for every supported framework, the CI job snippet for GitHub Actions and GitLab CI, and the actionable error message printed on failure.
