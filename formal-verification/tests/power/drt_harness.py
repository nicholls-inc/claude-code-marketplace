"""DRT harness for the Power module — K3 smoke test.

Drives random `(base, exp)` inputs through the oracle (Lean-faithful reference
or, once the Lean image is built, the Lean runner) and the SUT (production
Python). Reports divergences with witnesses, applies a simple shrink, and
emits a report at `formal-verification/tests/power/drt_report.md`.

Per /drt-oracle's Step 4, divergences are classified into one of four
categories. For this smoke, the planted bug is a "general implementation
bug" (loop off-by-one) — D7a's most common case.
"""

from __future__ import annotations

import argparse
import json
import os
import random
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

HERE = Path(__file__).resolve().parent
SUT = HERE / "sut" / "power.py"
ORACLE_REFERENCE = HERE / "oracle_reference.py"


@dataclass
class Divergence:
    iteration: int
    base: int
    exp: int
    oracle_output: int | None
    oracle_exit: int
    sut_output: int | None
    sut_exit: int


def _run(script: Path, payload: dict) -> tuple[int | None, int]:
    """Invoke a CLI implementation; return (parsed result, exit code)."""
    proc = subprocess.run(
        [sys.executable, str(script)],
        input=json.dumps(payload),
        capture_output=True,
        text=True,
        timeout=10,
    )
    if proc.returncode != 0:
        return None, proc.returncode
    try:
        return int(json.loads(proc.stdout.strip())["result"]), 0
    except (json.JSONDecodeError, KeyError, ValueError):
        return None, -1


def _gen_inputs(rng: random.Random, count: int) -> list[tuple[int, int]]:
    """Generate (base, exp) inputs over a small range."""
    return [(rng.randint(0, 5), rng.randint(0, 16)) for _ in range(count)]


def _shrink(div: Divergence, oracle: Path, sut: Path) -> Divergence:
    """Halve `base` and `exp` toward zero while the divergence still reproduces."""
    best = div
    for _ in range(20):
        candidates = []
        if best.base > 0:
            candidates.append((best.base // 2, best.exp))
        if best.exp > 0:
            candidates.append((best.base, best.exp // 2))
        progressed = False
        for b, e in candidates:
            payload = {"base": b, "exp": e}
            o, oc = _run(oracle, payload)
            s, sc = _run(sut, payload)
            if o != s or oc != sc:
                best = Divergence(div.iteration, b, e, o, oc, s, sc)
                progressed = True
                break
        if not progressed:
            break
    return best


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--count", type=int, default=int(os.environ.get("DRT_COUNT", "200")))
    parser.add_argument("--seed", type=int, default=int(os.environ.get("DRT_SEED", "20260508")))
    parser.add_argument(
        "--oracle",
        choices=("python-reference", "lean-runner"),
        default="python-reference",
        help=(
            "Which oracle to use. python-reference uses oracle_reference.py "
            "(Lean-faithful equivalent); lean-runner invokes the Lean runner "
            "via lake exe (requires the Lean Docker image to be built and "
            "the harness wired)."
        ),
    )
    parser.add_argument("--report", type=Path, default=HERE / "drt_report.md")
    args = parser.parse_args()

    if args.oracle == "lean-runner":
        sys.exit(
            "error: lean-runner oracle path is not yet wired in this smoke. "
            "Build the Lean image (scripts/build-lean-docker.sh) and add the "
            "PowerRunner.lean exec target before invoking with --oracle lean-runner."
        )

    oracle = ORACLE_REFERENCE
    rng = random.Random(args.seed)
    inputs = _gen_inputs(rng, args.count)

    divergences: list[Divergence] = []
    passes = 0
    for i, (base, exp) in enumerate(inputs):
        payload = {"base": base, "exp": exp}
        o, oc = _run(oracle, payload)
        s, sc = _run(SUT, payload)
        if o == s and oc == sc:
            passes += 1
            continue
        div = Divergence(i, base, exp, o, oc, s, sc)
        divergences.append(_shrink(div, oracle, SUT))

    report = _build_report(args, oracle, divergences, passes)
    args.report.write_text(report, encoding="utf-8")
    print(report)
    return 0 if not divergences else 1


def _build_report(args, oracle: Path, divergences: list[Divergence], passes: int) -> str:
    lines: list[str] = []
    lines.append("# DRT report: Power")
    lines.append("")
    lines.append(f"Run: smoke (`{args.oracle}` oracle)")
    lines.append(f"RNG seed: {args.seed}")
    lines.append(f"Inputs per def: {args.count}")
    lines.append("D4 case: (a) hand-written production code with no formal verification")
    lines.append("Correspondence doc: formal-verification/correspondence/Power.md")
    lines.append("")
    lines.append("## Verdict summary")
    lines.append("")
    lines.append("| Status | Count |")
    lines.append("|---|---|")
    lines.append(f"| Passed (no divergence) | {passes} |")
    lines.append(f"| Failed (divergence found) | {len(divergences)} |")
    lines.append("| Skipped (approximation) | 0 |")
    lines.append("| Excluded (fully Dafny-verified) | 0 |")
    lines.append("")
    if divergences:
        lines.append("## Failures")
        lines.append("")
        first = divergences[0]
        lines.append(f"### `power` — {len(divergences)} divergences")
        lines.append("")
        lines.append("- **D4 case applied.** (a)")
        lines.append(
            f"- **Witness (minimised).** Input `(base={first.base}, exp={first.exp})` produced "
            f"oracle output `{first.oracle_output}` and SUT output `{first.sut_output}`."
        )
        lines.append(f"- **RNG iteration of first witness.** {first.iteration}")
        lines.append(
            "- **Divergence classification.** general implementation bug "
            "(loop off-by-one in the iterative SUT — D7a's most common Cedar-bug class)."
        )
        lines.append(
            f"- **Reproducer.** `python {Path(__file__).name} --seed {args.seed} --count {args.count}`"
        )
        lines.append("")
        lines.append(f"Total divergences (post-shrink): {len(divergences)} of {args.count} inputs.")
        lines.append("")
    lines.append("## Passes")
    lines.append("")
    lines.append(f"- `power`: {passes} inputs no divergence (seed {args.seed}).")
    lines.append("")
    lines.append("## Skips (approximation)")
    lines.append("")
    lines.append("(none)")
    lines.append("")
    lines.append("## Exclusions (fully Dafny-verified slices)")
    lines.append("")
    lines.append("(none — no Dafny spec for `power` in this smoke)")
    lines.append("")
    lines.append("## Smoke-test scope note")
    lines.append("")
    lines.append(
        "The oracle in this run is the Python-faithful reference at "
        "`formal-verification/tests/power/oracle_reference.py`, used because the "
        "Lean image is not yet built in this session (sandbox restriction on "
        "`~/.docker/buildx/activity/`). Once the image builds, swap to the Lean "
        "runner via `--oracle lean-runner`. The harness logic, divergence "
        "minimisation, and witness reporting are exercised end-to-end by this run."
    )
    return "\n".join(lines) + "\n"


if __name__ == "__main__":
    sys.exit(main())
