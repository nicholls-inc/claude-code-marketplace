# Invariants: `/journal-context`

**Source:** `crosscheck/skills/journal-context/SKILL.md`, `crosscheck/skills/journal-context/scripts/walk.sh`
**Covering tests:** `crosscheck/skills/journal-context/tests/run_tests.sh` (bash, tagged with `# Invariant Ix: <Name>`)

## Purpose

`/journal-context <path>` is a deterministic walk of the directory tree from a starting path up to the enclosing git repository root, emitting the contents of every `JOURNAL.md` it encounters along the way. The skill is the §3.4 layer-2 enforcement piece for the §3.3 walk-up rule — the cheap deterministic instrument that fires the rule without putting an LLM in the loop. Agents and humans invoke it to load the narrative record before making non-trivial changes; the rule lives in `AGENTS.md`, this skill lets the rule actually fire.

## Invariants

### I1 — Walk shape: from the path's directory up to the repo root, inclusive

The walk starts at the directory containing the input path (the path itself if it is a directory; its parent if it is a file). It proceeds via the parent-of relation, visiting each ancestor directory exactly once, and terminates at — and includes — the enclosing git repository root (`git rev-parse --show-toplevel`). The walk does not cross into a parent repository when the input is inside a nested or submodule repo; the innermost repo's root is the stopping point.

```
walk(p) = sequence of directories such that
  d₀ = (isdir(p) ? p : dirname(p))
  d_{i+1} = parent(d_i)
  terminate at d_k = git-toplevel(p), inclusive
```

**Covering test:** `# Invariant I1: walk covers path-dir up to git toplevel inclusive`

---

### I2 — Ordering: deepest first, root last

Files are emitted in walk order — the `JOURNAL.md` closest to the input path comes first, the repo-root `JOURNAL.md` comes last. Within each file, content is emitted verbatim; the skill does not reorder, filter, paginate, or truncate entries inside a file. The "newest first" property is a convention of how journal entries are written, not something this skill enforces.

**Covering test:** `# Invariant I2: emit order is deepest-shard first`

---

### I3 — Determinism: same filesystem state, same output

Given identical filesystem state and identical input path, the skill produces byte-identical output. The walk consults no LLM, no clock, no random source, and makes no network call. The only inputs are the directory tree and the location of the enclosing git toplevel.

**Covering test:** `# Invariant I3: same input + tree → byte-identical output`

---

### I4 — Read-only: no filesystem or git mutation

The skill creates, modifies, and deletes no files. It does not run any git command that mutates state (no `fetch`, `commit`, `checkout`, `add`, `rm`, `clean`, `reset`, `pull`, `push`). It may invoke read-only git commands such as `rev-parse` and `ls-files`.

**Covering test:** `# Invariant I4: no filesystem or git mutation`

---

### I5 — Symbolic links: walk literal parents, not link targets

When the input path or one of its ancestors is a symbolic link, the walk uses the literal parent-of relation on the path's components (not the resolved link target). This keeps the walk's semantics stable under filesystem layout changes that move symlink targets around, and prevents a misconfigured symlink from quietly redirecting the walk into an unrelated tree.

**Covering test:** `# Invariant I5: symlinks do not redirect the walk`

---

### I6 — Empty result is explicit, not silent

If the walk completes with zero `JOURNAL.md` files encountered — either because none exist along the path, or because the input path is not inside any git repository — the skill emits an explicit human- and agent-readable message naming the condition. It does not return zero bytes. A silent empty output is indistinguishable from a tool failure to a downstream consumer, and that ambiguity is the failure mode this invariant exists to prevent.

**Covering test:** `# Invariant I6: zero-journal walk emits an explicit message`

---

### I7 — File boundaries visible in output, fixed delimiter shape

Each `JOURNAL.md`'s content is preceded by a delimiter line on its own line, immediately followed by the file's content. The delimiter shape is fixed at:

```
=== <path-relative-to-repo-root> ===
```

For example: `=== crosscheck/skills/JOURNAL.md ===`. The path is always relative to the enclosing repo root (the same root `git rev-parse --show-toplevel` returns in I1); absolute paths and `~`-expanded paths are not emitted. No trailing delimiter follows a file's content — the next `=== … ===` line marks the next boundary, and the empty case (I6) replaces the entire delimited block with its single explicit message.

The delimiter is fixed shape because downstream consumers — agents loading context, humans skimming output, and any future `/journal-lint` parser — are expected to grep or split on it. A floating delimiter form would force every consumer to reinvent the parser.

**Covering test:** `# Invariant I7: each file's content is preceded by an === <path> === delimiter`

## Carve-outs / known scope limits

- **Filename match only, not semantic.** The skill matches the literal filename `JOURNAL.md`. A fixture or test directory containing a file named `JOURNAL.md` that is not a real journal will be included. The skill is filename-deterministic; semantic filtering is the journal author's responsibility.
- **No content validation.** The skill does not lint journal entries for shape (date / type / why / links), missing `Supersedes:` targets, or contradictions between entries. That is the job of a separate `/journal-lint` skill (v2 §3.4) if and when it is built.
- **No filtering by date, type, or relevance.** The skill emits every journal file it walks past in full. A consumer that wants only recent entries, or only entries of a specific type, filters the output itself.
- **No cross-repo composition.** A single invocation walks within one git repository's tree. Multi-repo orchestration (monorepos with submodules, sibling repos checked out under a shared parent) is out of scope for v1; the working shape if that need arises is to invoke the skill once per repo and concatenate, not to teach the walk to cross repo boundaries.
- **Not a context loader.** The skill emits text. Putting that text into an agent's context window is the caller's job; the skill does not call agent APIs, write to a context cache, or otherwise manage the lifecycle of how its output is consumed.
