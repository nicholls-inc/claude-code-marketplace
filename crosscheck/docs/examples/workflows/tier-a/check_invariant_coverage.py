#!/usr/bin/env python3
"""Bidirectional invariant-to-test coverage gate.

Cross-checks invariant IDs declared in ``docs/invariants/**/*.md`` against
``# Invariant <ID>: <Name>`` comments in the repo's pytest test files.

Catches two failure modes:

1. Missing coverage — an invariant exists in the doc but no test references
   it. Coverage is claimed that does not exist.
2. Orphan comment — a test references an invariant ID that is not declared
   in any module doc. The reference is stale or typo'd.

Aspirational invariants (declared but not yet test-covered) are flagged in
their doc header with the literal HTML comment ``<!-- aspirational -->``.
The missing-coverage check skips them; the orphan-comment check still
applies.

Exits 0 on success, 1 on a coverage gap, 2 on a setup error.

Run locally:

    python3 scripts/check_invariant_coverage.py

This script is invoked by the ``invariant-coverage`` pre-commit hook
(``.pre-commit-config.yaml``) and the ``assurance`` GitHub Actions workflow
(``.github/workflows/assurance.yml``). The dual-track principle in
``docs/assurance/ROADMAP.md`` requires both enforcement points.
"""

from __future__ import annotations

import pathlib
import re
import sys
from dataclasses import dataclass

# Matches an invariant heading like ``**Q1. FIFO_ORDER.**``
# Capture group is the ID (e.g. ``Q1``).
HEADER_RE = re.compile(r"^\*\*([A-Z]+\d+[a-z]?)\.\s")

# Matches a test-comment like ``# Invariant I1: <NAME>.``
# Accepts both ``//`` and ``#`` to keep the gate portable across languages
# even though Python is the only consumer today.
COMMENT_RE = re.compile(r"^\s*(?://|#)\s*Invariant\s+([A-Z]+\d+[a-z]?):\s")

# Aspirational marker: ``<!-- aspirational -->`` on the same line as a
# ``**IN. Name.**`` header means the invariant is declared but not yet
# expected to have a covering test. Adding a covering test removes the
# marker; removing the marker without adding a test triggers the gate.
ASPIRATIONAL_RE = re.compile(r"<!--\s*aspirational\s*-->")

# Test files scanned for ``# Invariant <ID>:`` comments. The default below
# follows the pytest convention ``test_*.py`` under ``tests/``. Extend this
# list when a new test root needs to participate (e.g. nested package test
# directories or non-pytest layouts).
TEST_GLOBS: list[str] = [
    "tests/**/test_*.py",
]

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
            # Module attribution for an orphan comment is "the directory
            # containing the test", which is a coarse approximation but
            # sufficient for the diagnostic — the file:line is the actual
            # locator.
            out.extend(scan(path, COMMENT_RE, module=path.parent.name))
    return out


def main() -> int:
    if not INVARIANT_DIR.is_dir():
        print(
            f"error: {INVARIANT_DIR} not found — run /assurance-init first",
            file=sys.stderr,
        )
        return 2

    invariants = parse_invariants()
    comments = parse_comments()

    # We compare on (module, id) so the same ID in two different module
    # docs (e.g. both `module-a.md` and `module-b.md` declaring `I1`) is
    # correctly attributed — the test comment must live in a file under
    # the matching module's test directory. The current attribution is
    # coarse (uses the test file's parent directory name); see
    # parse_comments above. Tightening this is tracked under the Next
    # horizon item for the coverage gate in docs/assurance/ROADMAP.md.
    #
    # For v1, we relax to ID-only matching so the gate is usable without
    # imposing a strict directory-naming convention. Module-aware matching
    # can land in a follow-up.
    covered = {c.id for c in comments}
    declared = {i.id for i in invariants}
    missing = [i for i in invariants if not i.aspirational and i.id not in covered]
    orphans = [c for c in comments if c.id not in declared]

    if not missing and not orphans:
        print("invariant coverage OK")
        return 0

    if missing:
        print(
            "Missing coverage — add `# Invariant <ID>: <Name>` above the property test:",
            file=sys.stderr,
        )
        for i in missing:
            print(
                f"  - {i.module}/{i.id} declared at {i.path}:{i.line} (no covering test)",
                file=sys.stderr,
            )
    if orphans:
        print(
            "Orphan test comments — ID not declared in any invariant doc:",
            file=sys.stderr,
        )
        for c in orphans:
            print(f"  - {c.id} at {c.path}:{c.line}", file=sys.stderr)
    print(
        "\nFix: add the missing comment, remove the orphan, or mark the "
        "invariant `<!-- aspirational -->` in its doc header.\n"
        "See docs/invariants/README.md for the test-comment convention.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    sys.exit(main())
