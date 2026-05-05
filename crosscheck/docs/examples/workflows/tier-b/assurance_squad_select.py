#!/usr/bin/env python3
"""
Assurance Squad — phase-weighted task selection (deterministic pre-step).

Reads repo state, computes weights for the 11 task lifecycle, and writes
<work-dir>/task_selection.json. The agent step reads this and executes only
the selected tasks plus the always-run status dashboard.

This script must be deterministic: same inputs → same weights. The stochastic
draw is seeded by the current UTC timestamp so distinct runs differ, but the
selection mechanism is mechanical (no LLM call).

Outputs:
  <work-dir>/task_selection.json — { phase_signals, weights, selected, ... }

Halt conditions:
  - kill_criterion_active (FP rate >= 30 % over rolling 30d, n >= 5):
    zeros out T3 (draft invariants), T5 (acceptance), T7 (spec adversary).
    T9 (kill alert) dominates at weight 100.

Security notes:
  - All filesystem reads route through `_safe_open()` which asserts the
    resolved path lies under `REPO_ROOT`. Closes path-traversal (CWE-22)
    on inputs derived from REPO_ROOT-rooted joins or `glob` results.
  - The weighted draw uses `secrets.SystemRandom` rather than the global
    `random` module. The selection is **not** security-sensitive — it
    chooses which assurance tasks to fire — but a CSPRNG removes ruff
    S311 ambiguity and is essentially free at this scale.
  - `WORK_DIR` defaults to `tempfile.gettempdir()/gh-aw` rather than a
    hard-coded `/tmp/gh-aw`, satisfying ruff S108. Workflows that pin the
    docs path (`/tmp/gh-aw/...`) export `ASSURANCE_WORK_DIR` to match.
"""

from __future__ import annotations

import csv
import datetime
import glob
import json
import os
import secrets
import sys
import tempfile

REPO_ROOT = os.path.realpath(os.environ.get("GITHUB_WORKSPACE", os.getcwd()))
WORK_DIR = os.path.realpath(
    os.environ.get("ASSURANCE_WORK_DIR", os.path.join(tempfile.gettempdir(), "gh-aw"))
)
os.makedirs(WORK_DIR, exist_ok=True)


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


def fexists(p: str) -> bool:
    return os.path.isfile(os.path.join(REPO_ROOT, p))


def dexists(p: str) -> bool:
    return os.path.isdir(os.path.join(REPO_ROOT, p))


# ---------- probe artifacts ----------
has_audit = fexists("docs/assurance/AUDIT.md")
has_roadmap = fexists("docs/assurance/ROADMAP.md")
has_protected = fexists(".claude/rules/protected-surfaces.md")
has_acceptance = dexists("docs/assurance/acceptance")
invariant_files = glob.glob(os.path.join(REPO_ROOT, "docs/invariants/*.md"))
invariant_modules = len(invariant_files)

# Modules with a coverage gate wired (heuristic — refine per repo). Look for
# an "invariant-coverage" hook in pre-commit OR a CI workflow that mentions it.
covered_modules = 0
hooks_path = os.path.join(REPO_ROOT, ".pre-commit-config.yaml")
if os.path.isfile(hooks_path):
    with _safe_open(hooks_path) as f:
        if "invariant-coverage" in f.read():
            # at least one module is gated; per-module count is best-effort
            covered_modules = invariant_modules

# ---------- FP-tracker rolling 30d ----------
fp_count, fp_total, fp_rate = 0, 0, 0.0
fp_csv = os.path.join(REPO_ROOT, ".assurance/fp-tracker.csv")
if os.path.isfile(fp_csv):
    cutoff = datetime.date.today() - datetime.timedelta(days=30)
    with _safe_open(fp_csv) as f:
        for row in csv.DictReader(f):
            try:
                d = datetime.date.fromisoformat(row.get("date", ""))
            except ValueError:
                continue
            if d < cutoff:
                continue
            fp_total += 1
            if row.get("human_verdict", "").upper() == "FP":
                fp_count += 1
    fp_rate = (fp_count / fp_total) if fp_total else 0.0

# Require >=5 samples to avoid tripping on a single early FP.
kill_criterion_active = fp_rate >= 0.30 and fp_total >= 5

# ---------- Dafny candidates ----------
dafny_candidates: list[str] = []
for path in invariant_files:
    try:
        with _safe_open(path) as f:
            head = f.read(2048)
    except (OSError, ValueError):
        continue
    if "dafny_candidate: true" in head:
        dafny_candidates.append(os.path.basename(path).removesuffix(".md"))

# ---------- per-module spec-adversary rotation ----------
adversary_log = os.path.join(REPO_ROOT, ".assurance/spec-adversary-log.csv")
last_adversary: dict[str, str] = {}
if os.path.isfile(adversary_log):
    with _safe_open(adversary_log) as f:
        for row in csv.DictReader(f):
            last_adversary[row["module"]] = row["date"]
modules = [os.path.basename(p).removesuffix(".md") for p in invariant_files]
adversary_target = (
    sorted(modules, key=lambda m: last_adversary.get(m, "0000-00-00"))[0] if modules else None
)

# ---------- weights ----------
W: dict[str, float] = {
    "T1_layer_audit": 50.0 if not has_audit else 1.0,
    "T2_governance_scaffold": 50.0 if (has_audit and not has_roadmap) else 1.0,
    "T3_draft_invariants": 30.0 if (has_roadmap and invariant_modules < 5) else 5.0,
    "T4_coverage_gate": 30.0 if invariant_modules > covered_modules else 2.0,
    "T5_acceptance": 15.0 if (covered_modules >= 1 and not has_acceptance) else 1.0,
    "T6_roadmap_drift": 5.0 if has_roadmap else 0.0,
    "T7_spec_adversary": 20.0 if (invariant_modules >= 3 and adversary_target) else 0.0,
    "T8_fp_review": 25.0 if (0.20 <= fp_rate < 0.30 and fp_total >= 5) else 1.0,
    "T9_kill_criterion": 100.0 if kill_criterion_active else 0.0,
    "T10_dafny_promotion": 10.0 if dafny_candidates else 0.0,
    "T11_coverage_extension": 8.0 if covered_modules >= 1 else 0.0,
}

# ---------- kill-criterion halt ----------
# When tripped, T9 must dominate AND we halt the LLM-driven generative tasks
# (drafting, scenario writing, adversarial probing). Deterministic scaffolding
# tasks (T2, T4, T6) and detection tasks (T10, T11) remain available.
if kill_criterion_active:
    for k in ("T3_draft_invariants", "T5_acceptance", "T7_spec_adversary"):
        W[k] = 0.0

# ---------- weighted draw of 2 distinct tasks ----------
# Use a CSPRNG: the selection is non-cryptographic (it picks which assurance
# tasks fire), but `secrets.SystemRandom` is the same cost and removes the
# ambiguity that ruff S311 flags on the global `random` module.
rng = secrets.SystemRandom()
selected: list[str] = []
remaining = [(k, w) for k, w in W.items() if w > 0]
for _ in range(2):
    if not remaining:
        break
    total = sum(w for _, w in remaining)
    r = rng.uniform(0, total)
    acc = 0.0
    for i, (k, w) in enumerate(remaining):
        acc += w
        if r <= acc:
            selected.append(k)
            remaining.pop(i)
            break

selection = {
    "phase_signals": {
        "has_audit": has_audit,
        "has_roadmap": has_roadmap,
        "has_protected_surfaces": has_protected,
        "has_acceptance": has_acceptance,
        "invariant_modules": invariant_modules,
        "covered_modules": covered_modules,
        "fp_count_30d": fp_count,
        "fp_total_30d": fp_total,
        "fp_rate_rolling_30d": round(fp_rate, 4),
        "kill_criterion_active": kill_criterion_active,
        "dafny_candidates": dafny_candidates,
        "adversary_target": adversary_target,
    },
    "weights": W,
    "selected": selected,
    "always_run": ["TFinal_status_dashboard"],
    "sampled_at_utc": datetime.datetime.now(datetime.UTC)
    .isoformat(timespec="seconds")
    .replace("+00:00", "Z"),
}

out_path = os.path.join(WORK_DIR, "task_selection.json")
with _safe_open(out_path, "w") as f:
    json.dump(selection, f, indent=2)

print(json.dumps(selection, indent=2))
print(f"\n[assurance-squad] wrote {out_path}", file=sys.stderr)
