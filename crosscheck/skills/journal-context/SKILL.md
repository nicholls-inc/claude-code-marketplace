---
name: journal-context
add-mode: bootstrap
description: >-
  Deterministic walk of the directory tree from a starting path up to the
  enclosing git repository root, dumping every JOURNAL.md it encounters in
  walk order (deepest first, root last). No LLM in the walk. Use before
  non-trivial design work in any directory to load the narrative record
  above that location. Triggers: "journal context", "walk the journals",
  "load journal context", "what journals are above this file".
argument-hint: "[path]  (defaults to current directory)"
---

# /journal-context — Walk `JOURNAL.md` from a path to the repo root

## Description

`/journal-context` walks the directory tree upward from a given starting path to the enclosing git repository root and prints the contents of every `JOURNAL.md` it encounters along the way. The walk is deterministic — no LLM, no clock, no random source, no network — and read-only: it creates and modifies nothing on disk and runs no state-mutating git command.

The walk-up rule for this repo lives in the root `AGENTS.md`: *"before any non-trivial change, walk up from the file or directory you're touching to the repo root and read every `JOURNAL.md` you pass."* This skill is how an agent or human actually performs that walk without inventing the loader each time.

The seven invariants that pin behaviour — walk shape, ordering, determinism, read-only semantics, symlink handling, the empty-case message, and the `=== <path> ===` delimiter shape — live in [`docs/invariants/journal-context.md`](docs/invariants/journal-context.md). Read those first if anything below looks ambiguous.

## When to invoke

- Before substantial design work in a directory, to load the historical context above it.
- When auditing a PR, to surface what the journals already say about the area the diff touches.
- When orienting on an unfamiliar part of the tree.

Not for: linting journal entries (a separate `/journal-lint` skill is the right home for that, when it exists), authoring new entries (those are hand-written in the PR that introduces them), or cross-repo orchestration (one repo per invocation; concatenate at the caller if needed).

## Instructions

You are running a deterministic walk. Do not paraphrase, summarise, filter, or reorder the journal contents — the caller invoked this skill to get the raw text. If you find yourself reasoning about the entries before emitting them, stop: that reasoning is a separate step the caller can do with the output.

### Step 1 — Run the walk script

Invoke `scripts/walk.sh` with the path argument (or no argument to default to the current working directory). The script is the entire skill — it implements the seven invariants. Do not reimplement the walk inline.

```
bash crosscheck/skills/journal-context/scripts/walk.sh <path>
```

The script prints to stdout. Errors and the empty-case messages go to stdout as well (they are part of the output the caller asked for); only fatal pre-conditions (e.g. non-existent path) write to stderr and exit non-zero.

### Step 2 — Emit the script's output verbatim

Pass the script's stdout through to the caller unchanged. Do not strip the `=== <path> ===` delimiters, do not collapse blank lines between files, do not truncate long files.

If the script's exit code is non-zero, surface the stderr message and the exit code; do not invent a fallback walk.

### Step 3 — Do not act on the content

This skill ends with emitting the walked text. Drawing inferences from journal entries, surfacing tensions, or composing a summary are all separate steps that belong to whoever invoked the skill. Returning structured analysis out of `/journal-context` would conflate the deterministic-walk layer with the LLM-consumes-signals layer, which is the boundary the no-LLM-in-the-walk invariant (I3) exists to keep clean.

## Arguments

A single positional path argument. Defaults to the current working directory.

Examples:

- `/journal-context` — walk from CWD to the repo root.
- `/journal-context crosscheck/skills/journal-context/SKILL.md` — walk from this skill's directory upward.
- `/journal-context crosscheck/mcp-server/` — walk from the MCP server tree upward.
- `/journal-context /tmp/some-other-repo/src/foo.py` — walks within `/tmp/some-other-repo`'s git tree, not back into formal-verify.

## Verification checklist

Run each item against `scripts/walk.sh` before considering a change to the skill complete. The test suite under `tests/` exercises each one — run `bash tests/run_tests.sh` to confirm.

- [ ] Walking from a deep file emits `JOURNAL.md` files in deepest-first order, terminating at the repo root's `JOURNAL.md` (I1, I2).
- [ ] Each file's content is preceded by a `=== <path-relative-to-repo-root> ===` line on its own line (I7).
- [ ] Re-running with the same input produces byte-identical output (I3).
- [ ] `git status` is unchanged after a run; the script's `--dry-run` and live mode are the same (I4).
- [ ] A path outside any git repository emits the I6 "not inside a git repository" message and exits 0.
- [ ] A path inside a git repository with no `JOURNAL.md` above it emits the I6 "no JOURNAL.md found above …" message and exits 0.
- [ ] A walk through a symlinked ancestor terminates at the symlink's canonical toplevel and does not loop (I5).
- [ ] A non-existent path exits with code 2 and a clear error to stderr.

## References

- [`docs/invariants/journal-context.md`](docs/invariants/journal-context.md) — the seven invariants this skill ships against.
- [`scripts/walk.sh`](scripts/walk.sh) — the implementation.
- [`tests/run_tests.sh`](tests/run_tests.sh) — invariant-anchored test driver.
- Root [`AGENTS.md`](../../../AGENTS.md) — the walk-up rule this skill enforces.
- v2 retrospective [§3.3, §3.4, §5.2](../../docs/add/.retrospective/findings-and-methodology-v2.md) — the design context.
