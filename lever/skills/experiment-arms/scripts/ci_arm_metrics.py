#!/usr/bin/env python3
"""Summarize ci_arm_driver.py output for each arm: TTG, critical lane, billed minutes, first-try green, reruns.

Usage: python3 ci_arm_metrics.py <config.json> [--lane-prefix "E2E"] [--lane-label ubuntu]

Billed minutes = the sum over jobs of ceil(job minutes) x runner multiplier. Set the multipliers in
the config as `runner_multipliers` ({label: multiplier}). Unknown labels are reported, never guessed.
'a1' counts attempt-1 jobs only (the cost of a clean run); 'all' includes reruns. Leave out
path-filtered workflows that only the treatment's own diff triggers: set `save_jobs_for_workflows`
to the always-on CI workflows.

Cost is never assumed to be zero. A run with a job on an unknown runner label, a billed job with
no label or no end time, or no saved job snapshot at all has an unknown cost: it prints as
"unknown" and its billed_* fields are null with "cost_complete": false, so arm_stats.py gives no
verdict on cost until the data is complete.

Also writes samples_<tag>.jsonl with one row per (round, arm) for arm_stats.py:
  {"round", "arm", "value" (TTG min), "valid" (green), "lane", "billed_a1", "billed_all",
   "cost_complete", "cost_issues", "reruns"}
"""
import argparse
import collections
import glob
import json
import math
import os
import statistics as st
from datetime import datetime

DEFAULT_MULT = {"ubuntu-latest": 1, "windows-latest": 2, "macos-latest": 10}


def ts(s):
    return datetime.fromisoformat(s.replace("Z", "+00:00"))


def q(xs, p):
    return sorted(xs)[min(len(xs) - 1, int(round(p * (len(xs) - 1))))] if xs else float("nan")


def med(xs):
    return st.median(xs) if xs else float("nan")


ap = argparse.ArgumentParser()
ap.add_argument("config")
ap.add_argument("--lane-prefix", default="", help="job-name prefix of the critical lane, e.g. 'E2E'")
ap.add_argument("--lane-label", default="", help="runner-label prefix of the critical lane, e.g. 'ubuntu'")
args = ap.parse_args()
C = json.load(open(args.config))
D = os.path.dirname(os.path.abspath(args.config))
TAG = C.get("tag", "round")
MULT = C.get("runner_multipliers", DEFAULT_MULT)


def os_of(label):
    l = label.lower()
    return "mac" if "mac" in l else "win" if "win" in l else "linux"


rows, unknown = [], collections.Counter()
for line in open(os.path.join(D, f"arms_{TAG}.jsonl")):
    r = json.loads(line)
    n = r.pop("round")
    for arm, a in r.items():
        row = dict(round=n, arm=arm, result=a.get("result"), ttg=a.get("ttg_min"), rr=a["reruns"],
                   causes=[x["failed"] for x in a.get("rerun_log", [])], lane=None,
                   billed={"all": collections.Counter(), "a1": collections.Counter()}, a1_fails=[], issues=set())
        lane_end = []
        snaps = glob.glob(os.path.join(D, f"arms_jobs_{TAG}_r{n}_{arm}_*.json"))
        if not snaps:
            row["issues"].add("no job snapshot")
        for f in snaps:
            for j in json.load(open(f))["jobs"]:
                if not j.get("started_at") or j.get("conclusion") == "skipped":
                    continue  # never ran on a runner, so nothing was billed
                if not j.get("completed_at"):
                    row["issues"].add("job still running at snapshot")
                    continue
                if not j.get("labels"):
                    row["issues"].add("job without runner label")
                    continue
                lab = j["labels"][0]
                if lab not in MULT:
                    unknown[lab] += 1
                    row["issues"].add(f"unknown runner {lab}")
                    continue
                mins = math.ceil((ts(j["completed_at"]) - ts(j["started_at"])).total_seconds() / 60) * MULT[lab]
                row["billed"]["all"][os_of(lab)] += mins
                if j["run_attempt"] == 1:
                    row["billed"]["a1"][os_of(lab)] += mins
                    if args.lane_prefix and j["name"].startswith(args.lane_prefix) and lab.startswith(args.lane_label):
                        lane_end.append(ts(j["completed_at"]))
                    if j["conclusion"] in ("failure", "cancelled", "timed_out"):
                        row["a1_fails"].append(j["name"])
        if lane_end and a.get("first_created"):
            row["lane"] = round((max(lane_end) - ts(a["first_created"])).total_seconds() / 60, 2)
        rows.append(row)

if unknown:
    print("UNKNOWN runner labels (add them to runner_multipliers):", dict(unknown))
if any(r["issues"] for r in rows):
    print("WARNING: billed cost is incomplete for some runs; it is reported as unknown, and no cost "
          "decision should be made until every job and runner rate is accounted for")


def tot(r, k):
    """Billed minutes, or None when any part of the run's cost is unknown."""
    return None if r["issues"] else sum(r["billed"][k].values())


for arm in sorted({r["arm"] for r in rows}):
    R = [r for r in rows if r["arm"] == arm]
    G = [r for r in R if r["result"] == "green"]
    T = [r["ttg"] for r in G]
    L = [r["lane"] for r in G if r["lane"]]
    print(f"== {arm}: n={len(R)} green={len(G)} | TTG p50 {med(T):.1f} p90 {q(T, .9):.1f}"
          + (f" | lane p50 {med(L):.1f} p90 {q(L, .9):.1f}" if L else ""))
    incomplete = [r for r in G if r["issues"]]
    for k in ("a1", "all"):
        if incomplete:
            issues = sorted({i for r in incomplete for i in r["issues"]})
            print(f"   billed[{k}] unknown: cost incomplete in {len(incomplete)}/{len(G)} green runs ({'; '.join(issues)})")
        elif G:
            print(f"   billed[{k}] p50 {med([tot(r, k) for r in G]):.0f} min/run  "
                  + " ".join(f"{o} {med([r['billed'][k][o] for r in G]):.0f}" for o in ("linux", "mac", "win")))
    print(f"   first-try green {sum(r['rr'] == 0 and r['result'] == 'green' for r in R)}/{len(R)}"
          f"  reruns/PR {st.mean([r['rr'] for r in R]):.2f}")
    for r in R:
        if r["rr"] or r["result"] != "green":
            print(f"     r{r['round']} {r['result']} reruns={r['rr']} causes={r['causes']} "
                  f"a1_fails={sorted(set(r['a1_fails']))}")

with open(os.path.join(D, f"samples_{TAG}.jsonl"), "w") as f:
    for r in rows:
        f.write(json.dumps({"round": r["round"], "arm": r["arm"], "value": r["ttg"], "valid": r["result"] == "green",
                            "lane": r["lane"], "billed_a1": tot(r, "a1"), "billed_all": tot(r, "all"),
                            "cost_complete": not r["issues"], "cost_issues": sorted(r["issues"]),
                            "reruns": r["rr"]}) + "\n")
print(f"wrote samples_{TAG}.jsonl")
