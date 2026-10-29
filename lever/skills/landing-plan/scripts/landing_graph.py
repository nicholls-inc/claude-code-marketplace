#!/usr/bin/env python3
"""Build a landing plan (dependency waves + per-PR evidence) for a set of open PRs.

Edges come from:
  - stacking: a PR whose base branch is another PR's head branch
  - declared dependencies in the body: "depends on #N", "after #N", "stacked on #N"
  - shared paths: two PRs that touch the same file, or both touch a --shared glob

A dependency outside the supplied set (a declared #N, or a base branch that is neither the
default branch nor another PR's head) is never dropped: it is listed as unresolved, and the
PR is not ready, until #N is verified merged. Overlap edges never contradict an order that
already follows from other edges, so they cannot create a cycle.

Independent PRs share a wave; each wave is chunked into merge-queue batches.
Prints markdown to stdout. Read-only: it never merges, enqueues, or edits.

Usage:
  python3 landing_graph.py 101 102 103 --repo owner/name \
      [--shared '.github/workflows/*' --shared 'migrations/*'] \
      [--batch-size 4] [--cycle-min 30] [--flake-rate 0.2]
"""
import argparse
import fnmatch
import json
import re
import subprocess
import sys

FIELDS = ("number,title,body,baseRefName,headRefName,headRefOid,files,isDraft,"
          "mergeable,reviewDecision,statusCheckRollup")
DEP_RE = re.compile(r"(?:depends on|after|stacked on|requires)\s+#(\d+)", re.I)


def gh_pr(repo, n):
    out = subprocess.run(["gh", "pr", "view", str(n), "--repo", repo, "--json", FIELDS],
                         capture_output=True, text=True)
    if out.returncode != 0:
        sys.exit(f"gh pr view {n} failed: {out.stderr.strip()}")
    return json.loads(out.stdout)


def gh_pr_state(repo, n):
    """State of a PR outside the set ("MERGED", "OPEN", ...), or None if it can't be read."""
    out = subprocess.run(["gh", "pr", "view", str(n), "--repo", repo, "--json", "state"],
                         capture_output=True, text=True)
    return json.loads(out.stdout).get("state") if out.returncode == 0 else None


def gh_default_branch(repo):
    out = subprocess.run(["gh", "repo", "view", repo, "--json", "defaultBranchRef",
                          "--jq", ".defaultBranchRef.name"], capture_output=True, text=True)
    return out.stdout.strip() or None if out.returncode == 0 else None


def checks_state(pr):
    """Summarize checks on the PR's current head only."""
    rollup = pr.get("statusCheckRollup") or []
    fail = pend = ok = 0
    for c in rollup:
        concl = (c.get("conclusion") or c.get("state") or "").upper()
        status = (c.get("status") or "").upper()
        if status and status != "COMPLETED" and not c.get("conclusion"):
            pend += 1
        elif concl in ("SUCCESS", "NEUTRAL", "SKIPPED"):
            ok += 1
        elif concl in ("PENDING", "EXPECTED", "QUEUED", "IN_PROGRESS", ""):
            pend += 1
        else:
            fail += 1
    if not rollup:
        return "none"
    if fail:
        return f"red ({fail} failing)"
    if pend:
        return f"pending ({pend})"
    return f"green ({ok})"


def build_edges(prs, shared_globs, default_branch=None):
    by_head = {p["headRefName"]: p["number"] for p in prs.values()}
    edges = {n: set() for n in prs}  # n -> set of PRs that must land before n
    reasons = {}
    external = {n: {} for n in prs}  # n -> {dependency outside the set: reasons}; unresolved until verified

    def add(after, before, why):
        if after == before:
            return
        if before not in prs:
            external[after].setdefault(before, set()).add(why)
            return
        edges[after].add(before)
        reasons.setdefault((before, after), set()).add(why)

    def depends(x, y):
        """True if x must land after y, directly or through other PRs."""
        seen, todo = set(), [x]
        while todo:
            for d in edges[todo.pop()] - seen:
                if d == y:
                    return True
                seen.add(d)
                todo.append(d)
        return False

    for n, p in prs.items():
        if p["baseRefName"] in by_head:
            add(n, by_head[p["baseRefName"]], "stacked")
        elif default_branch and p["baseRefName"] != default_branch:
            add(n, f"branch {p['baseRefName']}", "stacked on a branch outside this set")
        for m in DEP_RE.findall(p.get("body") or ""):
            add(n, int(m), "declared")

    nums = sorted(prs)
    paths = {n: {f["path"] for f in prs[n].get("files") or []} for n in nums}
    for i, a in enumerate(nums):
        for b in nums[i + 1:]:
            if depends(a, b) or depends(b, a):
                continue  # already ordered, possibly transitively
            overlap = paths[a] & paths[b]
            why = None
            if overlap:
                why = "overlap: " + ", ".join(sorted(overlap)[:3])
            else:
                for g in shared_globs:
                    if any(fnmatch.fnmatch(x, g) for x in paths[a]) and \
                       any(fnmatch.fnmatch(x, g) for x in paths[b]):
                        why = f"shared: {g}"
                        break
            if why:
                # no declared order: chain lower number first, flag for the human
                add(b, a, why)
    return edges, reasons, external


def waves(edges):
    remaining = {n: set(d) for n, d in edges.items()}
    out = []
    while remaining:
        ready = sorted(n for n, d in remaining.items() if not d)
        if not ready:
            sys.exit(f"cycle among PRs: {sorted(remaining)}; resolve by hand")
        out.append(ready)
        for n in ready:
            del remaining[n]
        for d in remaining.values():
            d.difference_update(ready)
    return out


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("prs", nargs="+", type=int)
    ap.add_argument("--repo", required=True)
    ap.add_argument("--shared", action="append", default=[],
                    help="glob treated as coupling when both PRs touch it (repeatable)")
    ap.add_argument("--batch-size", type=int, default=4)
    ap.add_argument("--cycle-min", type=float, default=30.0,
                    help="one CI/queue cycle in minutes (measure it; don't guess)")
    ap.add_argument("--flake-rate", type=float, default=0.2,
                    help="share of runs that fail first try and need a rerun")
    a = ap.parse_args()

    prs = {n: gh_pr(a.repo, n) for n in a.prs}
    edges, reasons, external = build_edges(prs, a.shared, gh_default_branch(a.repo))
    plan = waves(edges)
    state = {d: gh_pr_state(a.repo, d) for deps in external.values() for d in deps if isinstance(d, int)}
    unresolved = {n: sorted((d for d in deps if state.get(d) != "MERGED"), key=str)
                  for n, deps in external.items()}

    print(f"# Landing plan: {a.repo} ({len(prs)} PRs)\n")
    print("## Evidence per PR (exact head)\n")
    print("| PR | Title | Head | Draft | Mergeable | Review | Checks | Must land after |")
    print("|---|---|---|---|---|---|---|---|")
    for n in sorted(prs):
        p = prs[n]
        after = ", ".join([f"#{d}" for d in sorted(edges[n])] +
                          [f"#{d} (unresolved)" if isinstance(d, int) else f"{d} (unresolved)"
                           for d in unresolved[n]]) or "-"
        print(f"| #{n} | {p['title'][:60]} | `{p['headRefOid'][:10]}` | "
              f"{'yes' if p['isDraft'] else 'no'} | {p.get('mergeable') or '?'} | "
              f"{p.get('reviewDecision') or 'none'} | {checks_state(p)} | {after} |")

    if reasons:
        print("\n## Edges\n")
        for (before, after), why in sorted(reasons.items()):
            print(f"- #{before} → #{after}: {'; '.join(sorted(why))}")
        if any(w.startswith(("overlap", "shared")) for ws in reasons.values() for w in ws):
            print("\n_Overlap edges are ordered by PR number; confirm or flip the order by hand._")

    if any(external.values()):
        print("\n## Dependencies outside this set\n")
        for n in sorted(external):
            for d, why in sorted(external[n].items(), key=lambda kv: str(kv[0])):
                name = f"#{d}" if isinstance(d, int) else d
                status = "merged" if state.get(d) == "MERGED" else \
                    f"UNRESOLVED ({(state.get(d) or 'unknown').lower()}; land or verify it first)"
                print(f"- #{n} needs {name}: {'; '.join(sorted(why))} — {status}")

    print("\n## Waves\n")
    batches = 0
    for i, w in enumerate(plan, 1):
        chunks = [w[j:j + a.batch_size] for j in range(0, len(w), a.batch_size)]
        batches += len(chunks)
        for k, c in enumerate(chunks, 1):
            tag = f"wave {i}" + (f" batch {k}" if len(chunks) > 1 else "")
            print(f"- **{tag}:** " + ", ".join(f"#{n}" for n in c))

    blockers = [n for n in prs if prs[n]["isDraft"] or not checks_state(prs[n]).startswith("green")
                or unresolved[n]]
    if blockers:
        print("\n**Not ready (draft, not green on head, or an unresolved dependency):** "
              + ", ".join(f"#{n}" for n in sorted(blockers)))

    serial = len(prs) * a.cycle_min * (1 + a.flake_rate) / 60
    batched = batches * a.cycle_min * (1 + a.flake_rate) / 60
    print("\n## Estimate\n")
    print(f"- serial (one PR per cycle): ~{serial:.1f} h")
    print(f"- batched ({batches} queue batches): ~{batched:.1f} h")
    print(f"- assumptions: cycle {a.cycle_min:.0f} min, flake rate {a.flake_rate:.0%}; "
          "record the actual time afterwards")


if __name__ == "__main__":
    main()
