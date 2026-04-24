# Python Template — Invariant Coverage Gate

Emit a Python script that parses invariant IDs from `docs/invariants/**/*.md` and cross-checks them against `// Invariant <ID>: <Name>` (or `# Invariant <ID>: <Name>`) comments in the repo's test files. Placeholders are marked `<placeholder>`.

## Script Template

Write to `scripts/check_invariant_coverage.py` and `chmod +x`.

```python
#!/usr/bin/env python3
"""Bidirectional invariant-to-test coverage gate. Exits 0 ok, 1 on gap, 2 on error."""
from __future__ import annotations

import pathlib
import re
import sys
from dataclasses import dataclass

HEADER_RE = re.compile(r"^\*\*([A-Z]+\d+[a-z]?)\.\s")             # **I1. Name.**
COMMENT_RE = re.compile(r"^\s*(?://|#)\s*Invariant\s+([A-Z]+\d+[a-z]?):\s")
ASPIRATIONAL_RE = re.compile(r"<!--\s*aspirational\s*-->")

TEST_GLOBS: list[str] = ["<test-glob>"]  # e.g. ["tests/**/*_invariants_test.py"]
INVARIANT_DIR = pathlib.Path("docs/invariants")


@dataclass(frozen=True)
class Entry:
    module: str
    id: str
    path: pathlib.Path
    line: int
    aspirational: bool = False


def scan(path: pathlib.Path, pattern: re.Pattern[str], module: str) -> list[Entry]:
    out: list[Entry] = []
    for lineno, line in enumerate(path.read_text(errors="replace").splitlines(), start=1):
        m = pattern.match(line)
        if m:
            out.append(Entry(module, m.group(1), path, lineno, bool(ASPIRATIONAL_RE.search(line))))
    return out


def parse_invariants() -> list[Entry]:
    out: list[Entry] = []
    for doc in sorted(INVARIANT_DIR.glob("*.md")):
        if doc.name in {"README.md", "COVERAGE.md"}:
            continue
        out.extend(scan(doc, HEADER_RE, module=doc.stem))
    return out


def parse_comments() -> list[Entry]:
    out: list[Entry] = []
    seen: set[pathlib.Path] = set()
    for glob in TEST_GLOBS:
        for path in sorted(pathlib.Path(".").glob(glob)):
            if path in seen or not path.is_file():
                continue
            seen.add(path)
            out.extend(scan(path, COMMENT_RE, module=path.parent.name))
    return out


def main() -> int:
    if not INVARIANT_DIR.is_dir():
        print(f"error: {INVARIANT_DIR} not found", file=sys.stderr)
        return 2
    invariants = parse_invariants()
    comments = parse_comments()
    covered = {(c.module, c.id) for c in comments}
    declared = {(i.module, i.id) for i in invariants}
    missing = [i for i in invariants if not i.aspirational and (i.module, i.id) not in covered]
    orphans = [c for c in comments if (c.module, c.id) not in declared]

    if not missing and not orphans:
        print("invariant coverage OK")
        return 0
    if missing:
        print("Missing coverage — add `// Invariant <ID>: <Name>` above the property test:", file=sys.stderr)
        for i in missing:
            print(f"  - {i.module}/{i.id} declared at {i.path}:{i.line} (no covering test)", file=sys.stderr)
    if orphans:
        print("Orphan test comments — ID not declared in any invariant doc:", file=sys.stderr)
        for c in orphans:
            print(f"  - {c.module}/{c.id} at {c.path}:{c.line}", file=sys.stderr)
    print(
        "\nFix: add the missing comment, remove the orphan, "
        "or mark the invariant `<!-- aspirational -->` in its doc header.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    sys.exit(main())
```

Replace `<test-glob>` with the detected glob (e.g. `tests/**/*_invariants_test.py`). Standard library only — no runtime deps.

## Pre-commit

**pre-commit.com (`.pre-commit-config.yaml`):**

```yaml
- repo: local
  hooks:
    - id: invariant-coverage
      name: Invariant coverage (Python)
      entry: python3 scripts/check_invariant_coverage.py
      language: system
      pass_filenames: false
      files: '^(docs/invariants/.*\.md|tests/.*_invariants_test\.py)$'
```

**lefthook (`lefthook.yml`):** add under `pre-commit.commands.invariant-coverage` with `glob: 'docs/invariants/*.md,tests/**/*_invariants_test.py'` and `run: python3 scripts/check_invariant_coverage.py`.

**husky (`.husky/pre-commit`):** append `python3 scripts/check_invariant_coverage.py || exit 1`.

**Standalone (`scripts/pre-commit-invariant-coverage.sh`):** `#!/usr/bin/env bash`, `set -euo pipefail`, then the `python3` line. Wire via `git config core.hooksPath .githooks` plus a symlink.

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
      - uses: actions/setup-python@v5
        with: { python-version: '3.11' }
      - run: python3 scripts/check_invariant_coverage.py
```

**GitLab CI (`.gitlab-ci.yml` job):**

```yaml
invariant-coverage:
  image: python:3.11-slim
  script: [python3 scripts/check_invariant_coverage.py]
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
  - runner/IX at tests/runner/test_runner_invariants.py:88

Fix: add the missing comment, remove the orphan, or mark the invariant `<!-- aspirational -->` in its doc header.
```

The fix command is embedded so an AI coding agent reading the failure can act without human translation.
