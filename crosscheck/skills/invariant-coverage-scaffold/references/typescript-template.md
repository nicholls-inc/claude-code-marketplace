# TypeScript Template — Invariant Coverage Gate

Emit a TypeScript script that parses invariant IDs from `docs/invariants/**/*.md` and cross-checks them against `// Invariant <ID>: <Name>` comments in the repo's test files. Placeholders are marked `<placeholder>`.

## Script Template

Write to `scripts/check-invariant-coverage.ts` (runnable via `npx tsx`). Dependency-free — only Node built-ins.

```ts
#!/usr/bin/env npx tsx
// Bidirectional invariant-to-test coverage gate. Exits 0 ok, 1 on gap.
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, basename, dirname, extname } from "node:path";

const HEADER_RE = /^\*\*([A-Z]+\d+[a-z]?)\.\s/;             // **I1. Name.**
const COMMENT_RE = /^\s*\/\/\s*Invariant\s+([A-Z]+\d+[a-z]?):\s/;
const ASPIRATIONAL_RE = /<!--\s*aspirational\s*-->/;

// Replace with repo convention, e.g. [".invariants.test.ts", ".invariants.spec.ts"].
const TEST_SUFFIXES: string[] = ["<test-suffix>"];
const INVARIANT_DIR = "docs/invariants";
const SKIP_DIRS = new Set(["node_modules", ".git", "dist", "build"]);

type Entry = {
  module: string;
  id: string;
  path: string;
  line: number;
  aspirational?: boolean;
};

function* walk(dir: string): Generator<string> {
  for (const entry of readdirSync(dir)) {
    if (SKIP_DIRS.has(entry)) continue;
    const full = join(dir, entry);
    const st = statSync(full);
    if (st.isDirectory()) yield* walk(full);
    else yield full;
  }
}

function scan(path: string, re: RegExp, module: string): Entry[] {
  const out: Entry[] = [];
  readFileSync(path, "utf8").split("\n").forEach((line, i) => {
    const m = re.exec(line);
    if (m) out.push({ module, id: m[1], path, line: i + 1, aspirational: ASPIRATIONAL_RE.test(line) });
  });
  return out;
}

function parseInvariants(): Entry[] {
  const out: Entry[] = [];
  for (const entry of readdirSync(INVARIANT_DIR)) {
    if (extname(entry) !== ".md" || entry === "README.md" || entry === "COVERAGE.md") continue;
    out.push(...scan(join(INVARIANT_DIR, entry), HEADER_RE, basename(entry, ".md")));
  }
  return out;
}

function parseComments(): Entry[] {
  const out: Entry[] = [];
  for (const path of walk(".")) {
    if (!TEST_SUFFIXES.some((s) => path.endsWith(s))) continue;
    out.push(...scan(path, COMMENT_RE, basename(dirname(path))));
  }
  return out;
}

function main(): number {
  const invariants = parseInvariants();
  const comments = parseComments();
  const covered = new Set(comments.map((c) => `${c.module}/${c.id}`));
  const declared = new Set(invariants.map((i) => `${i.module}/${i.id}`));
  const missing = invariants.filter((i) => !i.aspirational && !covered.has(`${i.module}/${i.id}`));
  const orphans = comments.filter((c) => !declared.has(`${c.module}/${c.id}`));

  if (missing.length === 0 && orphans.length === 0) {
    console.log("invariant coverage OK");
    return 0;
  }
  if (missing.length > 0) {
    console.error("Missing coverage — add `// Invariant <ID>: <Name>` above the property test:");
    for (const i of missing) console.error(`  - ${i.module}/${i.id} declared at ${i.path}:${i.line} (no covering test)`);
  }
  if (orphans.length > 0) {
    console.error("Orphan test comments — ID not declared in any invariant doc:");
    for (const c of orphans) console.error(`  - ${c.module}/${c.id} at ${c.path}:${c.line}`);
  }
  console.error("\nFix: add the missing comment, remove the orphan, or mark the invariant `<!-- aspirational -->` in its doc header.");
  return 1;
}

process.exit(main());
```

Replace `<test-suffix>` with the repo convention (e.g. `.invariants.test.ts`). Works under `tsx`, `ts-node`, or Deno with minor tweaks.

## Pre-commit

**pre-commit.com (`.pre-commit-config.yaml`):**

```yaml
- repo: local
  hooks:
    - id: invariant-coverage
      name: Invariant coverage (TypeScript)
      entry: npx tsx scripts/check-invariant-coverage.ts
      language: system
      pass_filenames: false
      files: '^(docs/invariants/.*\.md|.*\.invariants\.(test|spec)\.ts)$'
```

**lefthook (`lefthook.yml`):** add under `pre-commit.commands.invariant-coverage` with `glob: 'docs/invariants/*.md,**/*.invariants.test.ts,**/*.invariants.spec.ts'` and `run: npx tsx scripts/check-invariant-coverage.ts`.

**husky (`.husky/pre-commit`):** append `npx tsx scripts/check-invariant-coverage.ts || exit 1`.

**Standalone (`scripts/pre-commit-invariant-coverage.sh`):** `#!/usr/bin/env bash`, `set -euo pipefail`, then the `npx tsx` line. Wire via `git config core.hooksPath .githooks` plus a symlink.

## CI

**GitHub Actions (`.github/workflows/invariant-coverage.yml`):**

```yaml
name: invariant-coverage
on: [pull_request, push]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: npm ci --ignore-scripts
      - run: npx tsx scripts/check-invariant-coverage.ts
```

**GitLab CI (`.gitlab-ci.yml` job):**

```yaml
invariant-coverage:
  image: node:20
  script:
    - npm ci --ignore-scripts
    - npx tsx scripts/check-invariant-coverage.ts
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH
```

## Error Message

On failure the script prints to stderr:

```
Missing coverage — add `// Invariant <ID>: <Name>` above the property test:
  - queue/I1 declared at docs/invariants/queue.md:42 (no covering test)

Orphan test comments — ID not declared in any invariant doc:
  - runner/IX at src/runner/runner.invariants.test.ts:88

Fix: add the missing comment, remove the orphan, or mark the invariant `<!-- aspirational -->` in its doc header.
```

The fix command is embedded so an AI coding agent reading the failure can act without human translation.
