#!/usr/bin/env python3
"""Paired-round time-to-green (TTG) driver for experiment arms on GitHub Actions.

Each round pushes every arm (treatments + control, and optionally a second control
for A/A) at the same moment with an IDENTICAL nonce commit on top. It opens one draft
do-not-merge PR per arm into the default branch, so CI runs on the merge ref, then
polls each head SHA until every required check is green, red or timed out. Failed
jobs are re-run (up to `reruns`) as soon as the workflow run holding the failure
completes, the same way a person would. Results are appended to arms_<tag>.jsonl.
State is resumable from arms_state_<tag>.json. Create <tag>.stop to stop after the
current poll; rerun the same command to resume.

TTG = earliest workflow run created for the head SHA -> last required check completed.

Config (JSON, path is the only argument); see ../ci-adapter.md for a full example:
  repo, git_dir, tag, arms {name: branch}, nonce_files {path: "line with {n}"},
  required [check names], heavy [check names that must be 'success', not skipped],
  optional: default_branch, base_must_contain, fix_prs, save_jobs_for_workflows,
  rounds, reruns, poll_s, timeout_min, rate_floor, max_arms, branch_prefix,
  pr_title, pr_body, labels, nonce_marker.

Arm results: green (every required check ran and passed), red (a required check still
fails after `reruns`, or a rerun could not be started), invalid-skipped (a heavy check
did not succeed), invalid-missing (a required check never appeared before timeout_min),
timeout (checks still running at timeout_min). Only green arms count as measurements.

Branches: each run gets a random run id, kept in the state file, and pushes to
`lever/arm-<tag>-<run id>-<arm>` unless branch_prefix is set. The driver only rewrites
branches it created itself (recorded in the state file), and every push carries an
expected-old-SHA lease, so it never overwrites somebody else's commits.

Guardrails (dials, with defaults): pause when the core rate limit drops below rate_floor
(3000); poll every poll_s seconds (240) and query by head SHA only; at most max_arms
arms (10); run ONE driver per arm set, because duplicate pushes cancel arms through
concurrency groups. Needs git >= 2.38 (merge-tree --write-tree).
"""
import argparse
import datetime as dt
import json
import os
import re
import secrets
import subprocess
import sys
import time

PASS = {"success", "skipped", "neutral"}
MIN_GIT = (2, 38)
RERUN_ATTEMPTS = 3  # consecutive failures to start a rerun before the arm is declared red


def configure(path):
    """Load the config. Nothing touches the filesystem or network before this runs."""
    global C, REPO, GIT_DIR, TAG, D, STATE, OUT, STOP, REQ, HEAVY, RERUNS, POLL, TIMEOUT_MIN
    global RATE_FLOOR, MAX_ARMS, PREFIX, MARKER, BASE, TITLE, BODY, LABELS
    with open(path) as f:
        C = json.load(f)
    REPO, GIT_DIR, TAG = C["repo"], C["git_dir"], C.get("tag", "round")
    D = os.path.dirname(os.path.abspath(path))
    STATE = os.path.join(D, f"arms_state_{TAG}.json")
    OUT = os.path.join(D, f"arms_{TAG}.jsonl")
    STOP = os.path.join(D, f"{TAG}.stop")
    REQ = C["required"]
    HEAVY = C.get("heavy", [])
    RERUNS = C.get("reruns", 2)
    POLL = C.get("poll_s", 240)
    TIMEOUT_MIN = C.get("timeout_min", 150)
    RATE_FLOOR = C.get("rate_floor", 3000)
    MAX_ARMS = C.get("max_arms", 10)
    PREFIX = C.get("branch_prefix")  # None: run-specific default, set in main() from the state's run id
    MARKER = C.get("nonce_marker", "arm-nonce")
    BASE = C.get("default_branch", "main")
    TITLE = C.get("pr_title", "[measurement — do not merge] {tag} arm {arm}")
    BODY = C.get("pr_body", "Measurement-only PR: `{src}` plus a shared nonce, pushed together with the other "
                            "arms for a paired time-to-green comparison. It will be closed; do not merge.")
    LABELS = C.get("labels", ["do-not-merge"])


def check_git():
    try:
        out = subprocess.run(["git", "--version"], capture_output=True, text=True, check=True).stdout
    except (OSError, subprocess.CalledProcessError):
        sys.exit("git not found; this driver needs git >= 2.38")
    m = re.search(r"(\d+)\.(\d+)", out)
    if not m or (int(m.group(1)), int(m.group(2))) < MIN_GIT:
        sys.exit(f"{out.strip()} is too old: this driver needs git >= 2.38 (git merge-tree --write-tree)")


def branch(arm):
    return f"{PREFIX}{arm}"


def log(*a):
    print(dt.datetime.now(dt.timezone.utc).strftime("%H:%M:%SZ"), *a, flush=True)


def gh(path):
    for _ in range(3):
        try:
            return json.loads(subprocess.check_output(["gh", "api", path], stderr=subprocess.PIPE))
        except subprocess.CalledProcessError as e:
            log("gh api error", path, e.stderr[-200:])
            time.sleep(60)
    raise RuntimeError(path)


def ts(s):
    return dt.datetime.fromisoformat(s.replace("Z", "+00:00"))


def ratecheck():
    core = gh("rate_limit")["resources"]["core"]
    if core["remaining"] < RATE_FLOOR:
        wait = max(60, core["reset"] - time.time() + 30)
        log(f"rate limit low ({core['remaining']} < {RATE_FLOOR}), sleeping {wait:.0f}s")
        time.sleep(wait)


def git(*a):
    return subprocess.check_output(["git", "-C", GIT_DIR, *a], text=True).strip()


def nonce_tree(parent, n):
    """Same nonce lines in every arm: busts caches equally and stops path filters skipping lanes."""
    env = dict(os.environ, GIT_INDEX_FILE=os.path.join(D, f".arms_index_{os.getpid()}"))

    def run(*a, inp=None):
        return subprocess.run(["git", "-C", GIT_DIR, *a], env=env, input=inp, text=True,
                              capture_output=True, check=True).stdout.strip()

    run("read-tree", parent)
    for path, line in C["nonce_files"].items():
        lines = [l for l in git("show", f"{parent}:{path}").split("\n") if MARKER not in l]
        while lines and lines[-1] == "":
            lines.pop()
        blob = run("hash-object", "-w", "--stdin", inp="\n".join(lines) + "\n" + line.format(n=f"{TAG}-{n}") + "\n")
        run("update-index", "--cacheinfo", f"100644,{blob},{path}")
    tree = run("write-tree")
    os.remove(env["GIT_INDEX_FILE"])
    return tree


def open_pr(arm, v):
    br = branch(arm)
    have = subprocess.run(["gh", "pr", "list", "-R", REPO, "--head", br, "--state", "open", "--json", "number",
                           "--jq", ".[0].number"], capture_output=True, text=True).stdout.strip()
    if have:
        return
    out = subprocess.run(["gh", "pr", "create", "-R", REPO, "--draft", "--base", BASE, "--head", br,
                          "--title", TITLE.format(tag=TAG, arm=arm), "--body", BODY.format(src=v["src"])],
                         capture_output=True, text=True)
    url = out.stdout.strip().splitlines()[-1] if out.returncode == 0 and out.stdout.strip() else None
    if url and LABELS:
        # The label may not exist in this repo; the PR title still marks it do-not-merge.
        subprocess.call(["gh", "pr", "edit", url, "--add-label", ",".join(LABELS)],
                        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def remote_heads(branches):
    heads = {}
    for line in git("ls-remote", "origin", *[f"refs/heads/{b}" for b in branches]).splitlines():
        sha, ref = line.split("\t")
        if ref.startswith("refs/heads/"):
            heads[ref[len("refs/heads/"):]] = sha
    return heads


def lease(br, remote, owned):
    """Expected old SHA for a branch this run may rewrite ("" = must not exist); exit if it isn't ours."""
    cur, rec = remote.get(br), owned.get(br)
    if cur is None:
        return ""
    if rec is None:
        sys.exit(f"refusing to push: origin/{br} exists but this run did not create it. "
                 "Delete it, or set a different branch_prefix.")
    # "pending" is a push that may have landed before the driver was interrupted.
    if cur not in (rec.get("sha"), rec.get("pending")):
        sys.exit(f"refusing to push: origin/{br} is at {cur[:10]}, not the commit this run pushed "
                 f"({(rec.get('sha') or rec.get('pending') or '')[:10]}); someone else wrote to it.")
    return cur


def push_round(n, owned, save):
    arms = {a: b for a, b in C["arms"].items() if b}
    git("fetch", "-q", "origin", *set(arms.values()), *([C["base_must_contain"]] if C.get("base_must_contain") else []))
    base = git("rev-parse", f"origin/{C['base_must_contain']}") if C.get("base_must_contain") else None
    fixes = []
    for pr in C.get("fix_prs", []):
        git("fetch", "-q", "origin", f"pull/{pr}/head:refs/remotes/origin/pr-{pr}")
        fixes.append((pr, git("rev-parse", f"origin/pr-{pr}")))
    shas, skipped = {}, {}
    for arm, br in arms.items():
        try:
            parent = git("rev-parse", f"origin/{br}")
        except subprocess.CalledProcessError:
            skipped[arm] = "missing"
            continue
        if base and subprocess.call(["git", "-C", GIT_DIR, "merge-base", "--is-ancestor", base, parent]) != 0:
            skipped[arm] = "not on base"
            continue
        tip = parent
        for pr, fix in fixes:  # base is red: merge the fix into every arm, for measurement only
            mt = subprocess.run(["git", "-C", GIT_DIR, "merge-tree", "--write-tree", tip, fix],
                                capture_output=True, text=True)
            if mt.returncode != 0:
                raise RuntimeError(f"fix #{pr} conflicts with arm {arm}")
            tip = git("commit-tree", mt.stdout.split("\n")[0], "-p", tip, "-p", fix,
                      "-m", f"Merge base fix #{pr} (measurement only)")
        new = git("commit-tree", nonce_tree(tip, n), "-p", tip, "-m", f"measurement {TAG} round {n}")
        shas[arm] = {"sha": new, "src": br, "src_sha": parent}
    if not shas:
        sys.exit(f"round {n}: no arm could be pushed {skipped}")
    remote = remote_heads([branch(a) for a in shas])
    leases, refspecs = [], []
    for arm, v in shas.items():
        br = branch(arm)
        expect = lease(br, remote, owned)
        owned[br] = {"sha": expect or None, "pending": v["sha"]}
        leases.append(f"--force-with-lease=refs/heads/{br}:{expect}")
        refspecs.append(f"{v['sha']}:refs/heads/{br}")
    save()  # record the intent first, so an interrupted push can be reconciled on restart
    push = subprocess.run(["git", "-C", GIT_DIR, "push", "-q", "--atomic", *leases, "origin", *refspecs],
                          capture_output=True, text=True)
    if push.returncode != 0:
        sys.exit(f"push rejected; nothing was overwritten:\n{push.stderr.strip()}")
    for arm, v in shas.items():
        owned[branch(arm)] = {"sha": v["sha"]}
    save()
    for arm, v in shas.items():
        open_pr(arm, v)
    if skipped:
        log("skipped arms", skipped)
    return shas


def required_checks(sha):
    runs = []
    for page in (1, 2, 3):
        r = gh(f"repos/{REPO}/commits/{sha}/check-runs?per_page=100&filter=latest&page={page}")
        runs += r["check_runs"]
        if len(runs) >= r["total_count"]:
            break
    req = {}
    for cr in runs:
        if cr["name"] in REQ:
            cur = req.get(cr["name"])
            if cur is None or (cr["started_at"] or "") > (cur["started_at"] or ""):
                req[cr["name"]] = cr
    return req


def wf_runs(sha):
    return gh(f"repos/{REPO}/actions/runs?head_sha={sha}&per_page=50")["workflow_runs"]


def rerun(a, failed, runs, early):
    """Re-run failed jobs. Returns "started", "wait" (a run holding a failure is still going),
    "none" (no rerunnable workflow run behind the failure) or "error" (gh refused)."""
    rids = sorted({m.group(1) for cr in failed.values()
                   if (m := re.search(r"/runs/(\d+)/job/", cr.get("details_url") or ""))})
    if not rids:
        return "none"
    st = {str(r["id"]): r["status"] for r in runs}
    if any(st.get(r) != "completed" for r in rids):
        return "wait"
    bad = [rid for rid in rids if subprocess.call(["gh", "run", "rerun", rid, "--failed", "-R", REPO]) != 0]
    if bad:
        log(a["arm"], "rerun could not be started for runs", bad)
        return "error"
    a["reruns"] += 1
    a["rerun_errors"] = 0
    a.setdefault("rerun_log", []).append({"at": dt.datetime.now(dt.timezone.utc).isoformat(),
                                         "failed": sorted(failed), "runs": rids, "early": early})
    log(a["arm"], "rerun", a["reruns"], sorted(failed))
    return "started"


def step_arm(a):
    """Advance one arm; return True when it is finished. Every arm reaches a terminal result."""
    runs = wf_runs(a["sha"])
    if runs:
        a["first_created"] = min(r["created_at"] for r in runs)
    req = required_checks(a["sha"])
    all_runs_done = bool(runs) and all(r["status"] == "completed" for r in runs)
    present_done = all(cr["status"] == "completed" for cr in req.values())
    missing = [n for n in REQ if n not in req]
    elapsed = (dt.datetime.now(dt.timezone.utc) - ts(a["pushed_at"])).total_seconds() / 60
    failed = {n: cr for n, cr in req.items() if cr["status"] == "completed" and cr["conclusion"] not in PASS}
    still_running = not present_done or (missing and not all_runs_done)
    if failed and not still_running and a["reruns"] >= RERUNS:
        a.update(result="red", failed=sorted(failed), missing=missing)
        return True
    if req and not failed and not missing and present_done:
        skipped_heavy = [n for n in HEAVY if req.get(n, {}).get("conclusion") != "success"]
        last = max(req.values(), key=lambda c: c["completed_at"])
        a.update(result="green" if not skipped_heavy else "invalid-skipped", missing=missing,
                 skipped_heavy=skipped_heavy, last_required=last["completed_at"], last_required_name=last["name"])
        a["ttg_min"] = round((ts(a["last_required"]) - ts(a["first_created"])).total_seconds() / 60, 2)
        return True
    # Everything below waits or retries, so the deadline covers all of it.
    if elapsed > TIMEOUT_MIN:
        pending = sorted(n for n, cr in req.items() if cr["status"] != "completed")
        result = "red" if failed else "invalid-missing" if missing and not pending else "timeout"
        a.update(result=result, missing=missing, failed=sorted(failed), pending=pending)
        return True
    if failed and a["reruns"] < RERUNS:
        how = rerun(a, failed, runs, early=still_running)
        if how == "none" and not still_running:
            a.update(result="red", failed=sorted(failed), missing=missing,
                     rerun_error="no rerunnable workflow run behind the failed check")
            return True
        if how == "error":
            a["rerun_errors"] = a.get("rerun_errors", 0) + 1
            if a["rerun_errors"] >= RERUN_ATTEMPTS:
                a.update(result="red", failed=sorted(failed), missing=missing,
                         rerun_error=f"gh run rerun failed {a['rerun_errors']} times")
                return True
    # A missing required check stays pending until the deadline; it never counts as green.
    return False


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("config", help="arms JSON config; state and results are written next to it")
    args = ap.parse_args()
    configure(args.config)
    check_git()
    global PREFIX
    if not REQ:
        sys.exit("config: `required` must list at least one check")
    if len([b for b in C["arms"].values() if b]) > MAX_ARMS:
        sys.exit(f"{len(C['arms'])} arms exceeds max_arms={MAX_ARMS}; split the set or raise the dial deliberately")
    if POLL < 240:
        log(f"warning: poll_s={POLL} is below 240s; watch the shared API budget")
    s = json.load(open(STATE)) if os.path.exists(STATE) else {"round": 1, "arms": None}
    s.setdefault("run_id", secrets.token_hex(3))
    owned = s.setdefault("owned", {})  # branch -> commit this run pushed: the ownership record

    def save():
        json.dump(s, open(STATE + ".tmp", "w"), indent=1)
        os.replace(STATE + ".tmp", STATE)

    save()
    if PREFIX is None:
        PREFIX = f"lever/arm-{TAG}-{s['run_id']}-"
    # An unattended caller (the ci-speed lane) pins the push namespace in the environment, which the agent that
    # writes the config and state files can't change: no config or forged ownership record can aim a push elsewhere.
    pinned = os.environ.get("LEVER_ARM_BRANCH_PREFIX")
    if pinned:
        bad = [branch(a) for a in C["arms"] if not branch(a).startswith(pinned) or ".." in branch(a)]
        if bad or BASE.startswith(pinned):
            sys.exit(f"refusing: arm branches must start with LEVER_ARM_BRANCH_PREFIX={pinned!r}: {bad or BASE}")

    while s["round"] <= C.get("rounds", 3) and not os.path.exists(STOP):
        n = s["round"]
        if s["arms"] is None:
            ratecheck()
            now = dt.datetime.now(dt.timezone.utc).isoformat()
            shas = push_round(n, owned, save)
            s["arms"] = {arm: {"arm": arm, **v, "pushed_at": now, "reruns": 0, "done": False}
                         for arm, v in shas.items()}
            save()
            log(f"round {n} pushed", {a: v["sha"][:10] for a, v in shas.items()})
            time.sleep(POLL)
        for a in s["arms"].values():
            if not a["done"]:
                ratecheck()
                a["done"] = step_arm(a)
                if a["done"]:
                    log(f"round {n} {a['arm']} {a['result']} ttg={a.get('ttg_min')} reruns={a['reruns']}")
        save()
        if all(a["done"] for a in s["arms"].values()):
            for a in s["arms"].values():  # snapshot job lists once, for ci_arm_metrics.py
                for r in wf_runs(a["sha"]):
                    if r["path"].endswith(tuple(C.get("save_jobs_for_workflows", []))):
                        jobs = gh(f"repos/{REPO}/actions/runs/{r['id']}/jobs?filter=all&per_page=100")
                        json.dump(jobs, open(os.path.join(D, f"arms_jobs_{TAG}_r{n}_{a['arm']}_{r['id']}.json"), "w"))
            with open(OUT, "a") as f:
                f.write(json.dumps({"round": n, **s["arms"]}) + "\n")
            s.update(round=n + 1, arms=None)
            save()
            continue
        time.sleep(POLL)
    log("done")


if __name__ == "__main__":
    main()
