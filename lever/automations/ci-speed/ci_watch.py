#!/usr/bin/env python3
"""ci-speed watch: deterministic. No model, no API key. Needs `gh` authenticated (GH_TOKEN).

Measures time-to-green p50/p90, first-try pass rate, flake rate and the critical path on real PRs to the
default branch, compares them with the targets, and (unless --dry-run) opens or updates ONE goal issue.
Prints a JSON record. Writes `goal_issue=<number>` to $GITHUB_OUTPUT when a goal issue was opened or
needs the lane; the LLM lane only runs on that signal.

Usage: python3 ci_watch.py --workflows "CI,E2E" --p50 12 --p90 25 --first-try 80 [--window-days 7]
       [--hours 9-17] [--goal-label ci-goal] [--dry-run]
Runs cancelled by a newer push are excluded. Every number carries n. A number that cannot be computed is
"unavailable", never estimated.
"""
import argparse, collections, json, math, os, subprocess, sys
from datetime import datetime, timedelta, timezone


def gh(*a):
    return subprocess.run(["gh", *a], check=True, capture_output=True, text=True).stdout


def ts(s):
    return datetime.fromisoformat(s.replace("Z", "+00:00"))


def q(xs, p):
    return sorted(xs)[min(len(xs) - 1, int(round(p * (len(xs) - 1))))] if xs else None


ap = argparse.ArgumentParser()
ap.add_argument("--workflows", required=True, help="comma-separated REQUIRED_WORKFLOWS names")
ap.add_argument("--p50", type=float, required=True)
ap.add_argument("--p90", type=float, required=True)
ap.add_argument("--first-try", type=float, required=True, help="target percent")
ap.add_argument("--window-days", type=int, default=7)
ap.add_argument("--hours", default="", help="busy UTC hours on weekdays, e.g. 9-17; empty = all")
ap.add_argument("--goal-label", default="ci-goal")
ap.add_argument("--dry-run", action="store_true")
a = ap.parse_args()

wfs = [w.strip() for w in a.workflows.split(",")]
since = (datetime.now(timezone.utc) - timedelta(days=a.window_days)).strftime("%Y-%m-%d")
by_sha = collections.defaultdict(dict)  # sha -> workflow -> run
reruns = []
for w in wfs:
    runs = json.loads(gh("run", "list", "--workflow", w, "--event", "pull_request", "--created", f">={since}",
                         "--limit", "500", "--json",
                         "databaseId,headSha,createdAt,updatedAt,conclusion,attempt,status,startedAt"))
    for r in runs:
        if r["status"] != "completed" or r["conclusion"] == "cancelled":
            continue
        if r.get("attempt", 1) > 1:
            reruns.append(r["conclusion"] == "success")
        cur = by_sha[r["headSha"]].get(w)
        if cur is None or r.get("attempt", 1) >= cur.get("attempt", 1):
            by_sha[r["headSha"]][w] = r

lo, hi = (int(x) for x in a.hours.split("-")) if a.hours else (0, 24)
ttg, first_try, last_wf = [], [], collections.Counter()
for sha, d in by_sha.items():
    if set(d) != set(wfs):
        continue
    start = min(ts(r["createdAt"]) for r in d.values())
    if start.weekday() >= 5 and a.hours:
        continue
    if not (lo <= start.hour < hi):
        continue
    if not all(r["conclusion"] == "success" for r in d.values()):
        continue
    end = max(ts(r["updatedAt"]) for r in d.values())
    ttg.append((end - start).total_seconds() / 60)
    first_try.append(all(r.get("attempt", 1) == 1 for r in d.values()))
    last_wf[max(d, key=lambda w: ts(d[w]["updatedAt"]))] += 1

n = len(ttg)
rec = {
    "job": "ci-speed-watch", "window_days": a.window_days, "n": n,
    "p50_min": round(q(ttg, 0.5), 1) if n else "unavailable (no samples)",
    "p90_min": round(q(ttg, 0.9), 1) if n else "unavailable (no samples)",
    "first_try_pass_pct": round(100 * sum(first_try) / n, 1) if n else "unavailable (no samples)",
    "flake_rate": {"reruns": len(reruns), "passed": sum(reruns)} if reruns else "unavailable (no reruns)",
    "critical_path": last_wf.most_common(1)[0][0] if last_wf else "unavailable",
    "targets": {"p50": a.p50, "p90": a.p90, "first_try": a.first_try},
}
miss = []
if n:
    if rec["p50_min"] > a.p50: miss.append("p50")
    if rec["p90_min"] > a.p90: miss.append("p90")
    if rec["first_try_pass_pct"] < a.first_try: miss.append("first_try")
rec["missed"] = miss
rec["status"] = "NOOP(on target)" if not miss else "ACTED(goal)"

if miss and not a.dry_run:
    found = json.loads(gh("issue", "list", "--label", a.goal_label, "--state", "open", "--json", "number,body"))
    body = (f"Missed: {', '.join(miss)}\n\n```json\n{json.dumps(rec, indent=2)}\n```\n\n"
            "Opened by the deterministic ci-speed watch. The lane writes the hypothesis list and the keep bar.")
    if found:
        num = found[0]["number"]
        old = json.loads(found[0]["body"].split("```json\n")[-1].split("\n```")[0]) if "```json" in found[0]["body"] else {}
        moved = any(isinstance(old.get(k), (int, float)) and isinstance(rec[k], (int, float)) and old[k]
                    and abs(rec[k] - old[k]) / old[k] > 0.10 for k in ("p50_min", "p90_min", "first_try_pass_pct"))
        if moved:
            gh("issue", "comment", str(num), "--body", body)
        goal = None  # never wake the lane twice for an owned goal
    else:
        url = gh("issue", "create", "--title", f"CI speed goal: {', '.join(miss)} over target",
                 "--body", body, "--label", a.goal_label).strip()
        goal = url.rsplit("/", 1)[-1]
        rec["goal_issue"] = goal
        out = os.environ.get("GITHUB_OUTPUT")
        if out:
            open(out, "a").write(f"goal_issue={goal}\n")
print(json.dumps(rec, indent=2))
summ = os.environ.get("GITHUB_STEP_SUMMARY")
if summ:
    open(summ, "a").write("```json\n" + json.dumps(rec, indent=2) + "\n```\n")
