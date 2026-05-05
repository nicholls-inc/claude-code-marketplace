#!/usr/bin/env python3
"""
Assurance PR-Gate — per-invariant plan computation.

For each invariant in changed `docs/invariants/*.md` files, computes a content
hash over (invariant_id, prose, covering_test_sources, module_source) and
checks the attestation cache. Writes <work-dir>/pr_gate_plan.json.

Cache short-circuit:
  - cache_hit && !kill_criterion_active → use_cached
  - otherwise                            → run_intent_check

Granularity is per-invariant (not per-module): a module typically has several
invariants and a code change usually touches one. Per-module hashing would
over-invalidate.

Required env:
  GITHUB_BASE_REF       — base branch (e.g. "main"); falls back to "main"
  GH_AW_PR_BODY         — full PR body text (for amendment-block detection)
  GITHUB_WORKSPACE      — repo root (set by GitHub Actions automatically)
  ASSURANCE_WORK_DIR    — output dir; falls back to a per-invocation tempdir

Outputs:
  <work-dir>/pr_gate_plan.json — { invariants, kill_active, ... }

Security notes:
  - All filesystem reads/writes go through `_safe_open()` which asserts the
    resolved path lies under `REPO_ROOT` (or `WORK_DIR` for the output).
    This closes path-traversal (CWE-22) when paths are constructed from
    docs/<file> patterns or env-controlled work dirs.
  - The git invocation uses `shutil.which("git")` (full path) and validates
    the base-ref against a strict regex before interpolation, closing
    command-injection (CWE-78) on `GITHUB_BASE_REF`.
  - `WORK_DIR` defaults to `tempfile.gettempdir()/gh-aw` rather than a
    hard-coded `/tmp/gh-aw`, satisfying ruff S108. Workflows that pin the
    docs path (`/tmp/gh-aw/...`) export `ASSURANCE_WORK_DIR` to match.
"""

from __future__ import annotations

import hashlib
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile

REPO_ROOT = os.path.realpath(os.environ.get("GITHUB_WORKSPACE", os.getcwd()))
WORK_DIR = os.path.realpath(
    os.environ.get("ASSURANCE_WORK_DIR", os.path.join(tempfile.gettempdir(), "gh-aw"))
)
os.makedirs(WORK_DIR, exist_ok=True)

ATTEST_DIR = os.path.join(REPO_ROOT, "docs/assurance/attestations")
INVARIANT_DIR = os.path.join(REPO_ROOT, "docs/invariants")

# Strict regex for git ref names: alnum, dot, dash, underscore, slash. Rejects
# shell metacharacters, spaces, and option-like leading dashes. Used to
# validate `GITHUB_BASE_REF` before interpolation into a subprocess argv.
_REF_RE = re.compile(r"^[A-Za-z0-9._/-]+$")


def _safe_open(path: str, mode: str = "r"):
    """Open `path` after asserting it resolves inside REPO_ROOT or WORK_DIR.

    Raises ValueError on traversal attempts. This is the single read/write
    gate for the script — every `open()` call routes through here.
    """
    real = os.path.realpath(path)
    allowed_roots = (REPO_ROOT, WORK_DIR)
    if real not in allowed_roots and not any(
        real.startswith(root + os.sep) for root in allowed_roots
    ):
        raise ValueError(f"refusing to open path outside repo/work-dir: {path!r}")
    if "b" in mode:
        return open(real, mode)  # noqa: SIM115 — caller manages lifetime
    return open(real, mode, encoding="utf-8")  # noqa: SIM115


# ---------- kill-criterion ----------
kill_path = os.path.join(REPO_ROOT, ".assurance/kill-criterion.json")
kill_active = False
if os.path.isfile(kill_path):
    try:
        with _safe_open(kill_path) as f:
            kill_active = bool(json.load(f).get("active", False))
    except (json.JSONDecodeError, OSError, ValueError):
        kill_active = False


# ---------- changed files ----------
def git_diff_files() -> list[str]:
    """Return changed file paths via `git diff --name-only origin/<base>...HEAD`.

    Validates the base ref against `_REF_RE` before interpolation. Falls back
    to `main` if the env var is missing or fails validation. Uses the full
    `git` executable path resolved via `shutil.which()` to avoid PATH lookups
    at runtime (CWE-426 mitigation).
    """
    raw_base = os.environ.get("GITHUB_BASE_REF", "main")
    base = raw_base if _REF_RE.match(raw_base) else "main"
    git_bin = shutil.which("git")
    if git_bin is None:
        return []
    proc = subprocess.run(
        [git_bin, "diff", "--name-only", f"origin/{base}...HEAD"],  # nosemgrep: python.lang.security.audit.dangerous-subprocess-use-tainted-env-args
        capture_output=True,
        text=True,
        cwd=REPO_ROOT,
        check=False,
    )
    if proc.returncode != 0:
        # Fall back to comparing against main directly if origin/<base> is missing.
        proc = subprocess.run(  # nosemgrep: python.lang.security.audit.dangerous-subprocess-use-tainted-env-args
            [git_bin, "diff", "--name-only", "main...HEAD"],
            capture_output=True,
            text=True,
            cwd=REPO_ROOT,
            check=False,
        )
    return [line for line in proc.stdout.splitlines() if line.strip()]


changed = git_diff_files()
changed_invariant_docs = [
    c for c in changed if c.startswith("docs/invariants/") and c.endswith(".md")
]


# ---------- parse invariant doc → invariant IDs ----------
INVARIANT_ID_RE = re.compile(
    r"^##+\s*Invariant\s+([A-Z0-9_-]+)\s*[:\-]?\s*(.*)$",
    re.MULTILINE | re.IGNORECASE,
)


def parse_invariants(path: str) -> list[tuple[str, str, str]]:
    """Return [(id, name, body)] for each invariant section in the doc."""
    with _safe_open(path) as f:
        full = f.read()
    matches = list(INVARIANT_ID_RE.finditer(full))
    out = []
    for i, m in enumerate(matches):
        start = m.end()
        end = matches[i + 1].start() if i + 1 < len(matches) else len(full)
        out.append((m.group(1), m.group(2).strip(), full[start:end].strip()))
    return out


# ---------- find covering test for an invariant ID ----------
TEST_GLOBS = [
    "tests/**/*.py",
    "tests/**/*.go",
    "tests/**/*.ts",
    "tests/**/*.tsx",
    "**/*_property_test.py",
    "**/*_property_test.go",
    "**/*.property.test.ts",
    "**/*.property.test.tsx",
]


def find_covering_test_source(invariant_id: str) -> list[tuple[str, str]]:
    """Return [(rel_path, source)] for files containing `Invariant <ID>`."""
    needle = f"Invariant {invariant_id}"
    matches: list[tuple[str, str]] = []
    seen: set[str] = set()
    for pattern in TEST_GLOBS:
        for path in pathlib.Path(REPO_ROOT).glob(pattern):
            rel = str(path.relative_to(REPO_ROOT))
            if rel in seen:
                continue
            try:
                # path is from glob within REPO_ROOT; pathlib.read_text() is
                # acceptable here because the path is already constrained.
                src = path.read_text(encoding="utf-8")
            except (OSError, UnicodeDecodeError):
                continue
            if needle in src:
                matches.append((rel, src))
                seen.add(rel)
    return matches


# ---------- per-invariant content hash ----------
def content_hash(
    invariant_id: str,
    prose: str,
    test_sources: list[tuple[str, str]],
    module_source: str,
) -> str:
    h = hashlib.sha256()
    h.update(b"v1\n")  # bump on cache-key schema change
    h.update(invariant_id.encode())
    h.update(b"\n")
    h.update(prose.encode())
    h.update(b"\n")
    for path, src in sorted(test_sources):
        h.update(path.encode())
        h.update(b"\n")
        h.update(src.encode())
        h.update(b"\n")
    h.update(module_source.encode())
    return h.hexdigest()


# ---------- module source for a given invariant doc ----------
def module_source_for(invariant_doc_path: str) -> str:
    """Heuristic: invariant doc `docs/invariants/<module>.md` → module source.

    Looks for a top-level dir or package matching <module>; falls back to
    concatenating all source files under it. Returns empty string if no
    candidate exists, which signals "cannot hash precisely; conservatively
    re-run instead of risking a false cache hit."
    """
    name = pathlib.Path(invariant_doc_path).stem
    candidates = [
        os.path.join(REPO_ROOT, name),
        os.path.join(REPO_ROOT, "src", name),
        os.path.join(REPO_ROOT, name + ".py"),
        os.path.join(REPO_ROOT, name + ".go"),
        os.path.join(REPO_ROOT, name + ".ts"),
    ]
    code_suffixes = {".py", ".go", ".ts", ".tsx", ".js"}
    for candidate in candidates:
        if os.path.isdir(candidate):
            buf: list[str] = []
            for p in sorted(pathlib.Path(candidate).rglob("*")):
                if p.is_file() and p.suffix in code_suffixes:
                    try:
                        # Inside REPO_ROOT-rooted rglob; safe.
                        buf.append(p.read_text(encoding="utf-8"))
                    except (OSError, UnicodeDecodeError):
                        continue
            return "".join(buf)
        if os.path.isfile(candidate):
            try:
                with _safe_open(candidate) as f:
                    return f.read()
            except (OSError, UnicodeDecodeError, ValueError):
                return ""
    return ""


# ---------- plan per invariant ----------
plan: dict = {
    "invariants": [],
    "kill_active": kill_active,
    "amendment_reminder_needed": False,
    "dafny_handoff_needed": False,
    "protected_files_touched": [],
}

for doc in changed_invariant_docs:
    abs_doc = os.path.join(REPO_ROOT, doc)
    for inv_id, inv_name, inv_body in parse_invariants(abs_doc):
        tests = find_covering_test_source(inv_id)
        msrc = module_source_for(doc)
        chash = content_hash(inv_id, inv_body, tests, msrc)
        attestation_path = os.path.join(ATTEST_DIR, f"{chash}.json")
        cache_hit = os.path.isfile(attestation_path) and not kill_active
        cached = None
        if cache_hit:
            try:
                with _safe_open(attestation_path) as f:
                    cached = json.load(f)
            except (json.JSONDecodeError, OSError, ValueError):
                cache_hit = False
        plan["invariants"].append(
            {
                "doc": doc,
                "invariant_id": inv_id,
                "invariant_name": inv_name,
                "covering_tests": [p for p, _ in tests],
                "missing_covering_test": len(tests) == 0,
                "content_hash": chash,
                "cache_hit": cache_hit,
                "cached_attestation": cached,
                "action": "use_cached" if cache_hit else "run_intent_check",
                "module_source_resolvable": bool(msrc),
            }
        )

# ---------- amendment reminder needed? ----------
PROTECTED_PREFIXES_A = (
    ".github/workflows/",
    ".claude/rules/",
    "agents/",
)
# Class B docs are markdown specs only. Mirrors
# `.claude/rules/protected-surfaces.md`, which scopes Class B to
# `docs/invariants/*.md` and `docs/assurance/**/*.md`. Structural
# placeholders (`.gitkeep`, etc.) under these dirs aren't load-bearing
# invariant content and shouldn't trip the amendment reminder.
PROTECTED_PREFIXES_B_MD = ("docs/invariants/", "docs/assurance/")


def is_protected(path: str) -> bool:
    if path.startswith(PROTECTED_PREFIXES_A):
        return True
    if path.startswith(PROTECTED_PREFIXES_B_MD) and path.endswith(".md"):
        return True
    return "_property_test" in path or ".property.test" in path


protected_changes = [c for c in changed if is_protected(c)]
plan["protected_files_touched"] = protected_changes

pr_body = os.environ.get("GH_AW_PR_BODY", "")
plan["amendment_reminder_needed"] = (
    bool(protected_changes) and "## Governance Amendment" not in pr_body
)

# ---------- dafny hand-off detection ----------
for doc in changed_invariant_docs:
    try:
        with _safe_open(os.path.join(REPO_ROOT, doc)) as f:
            head = f.read(2048)
    except (OSError, UnicodeDecodeError, ValueError):
        continue
    if "dafny_candidate: true" in head:
        plan["dafny_handoff_needed"] = True
        break

out_path = os.path.join(WORK_DIR, "pr_gate_plan.json")
with _safe_open(out_path, "w") as f:
    json.dump(plan, f, indent=2)

print(json.dumps(plan, indent=2))
print(f"\n[assurance-pr-gate] wrote {out_path}", file=sys.stderr)
