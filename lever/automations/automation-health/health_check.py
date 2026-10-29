#!/usr/bin/env python3
"""automation-health, deterministic part. No model, no API key. Needs `gh` (GH_TOKEN) and PyYAML.

  --mode weekly   inventory every .github/workflows/lever-*.yml, measure runs over --window-days,
                  run the mechanical least-privilege lint, read the state block. Writes --out JSON.
  --mode failure  append one worker failure to the state block, count identical failures in 24 h and
                  say whether the repeat threshold is reached (escalate=true hands off to the LLM job).

The LLM part (rules 5-8 of prompt.md) judges what only a model can: whether a `permissions:` scope or
tool exceeds what the worker's ALLOWED ACTIONS need, which proposal to make, the report. Numbers that
cannot be observed are "unavailable (<why>)", never estimated. Outcomes (ACTED/NOOP/BLOCKED) come from
the run log of model-reaching runs, capped per workflow; job summaries are not exposed by the API.
"""
import argparse, glob, json, os, re, subprocess, sys
from datetime import datetime, timedelta, timezone

import yaml

ap = argparse.ArgumentParser()
ap.add_argument("--mode", choices=["weekly", "failure"], required=True)
ap.add_argument("--window-days", type=int, default=7)
ap.add_argument("--state-issue", default="")  # "" or "0" = none
ap.add_argument("--failed-workflow", default="")
ap.add_argument("--failed-conclusion", default="")
ap.add_argument("--failed-run", default="")
ap.add_argument("--repeat", type=int, default=3)
ap.add_argument("--log-cap", type=int, default=20, help="max run logs read per workflow")
ap.add_argument("--dry-run", action="store_true")
ap.add_argument("--out", default="health.json")
a = ap.parse_args()
if a.state_issue == "0":
    a.state_issue = ""
NOW = datetime.now(timezone.utc)


def gh(*args, check=True):
    r = subprocess.run(["gh", *args], capture_output=True, text=True)
    if check and r.returncode:
        raise RuntimeError(r.stderr.strip()[:200])
    return r.stdout


def ts(s):
    return datetime.fromisoformat(s.replace("Z", "+00:00"))


# ---- state (rule 0) -------------------------------------------------------------------------
STATE_RE = re.compile(r"(## lever state\s*```json\n)(.*?)(\n```)", re.S)


def read_state():
    if not a.state_issue:
        return {}, None, "no --state-issue"
    body = gh("issue", "view", a.state_issue, "--json", "body", "-q", ".body")
    m = STATE_RE.search(body)
    if not m:
        return {}, body, "state block missing: started empty"
    try:
        return json.loads(m.group(2) or "{}"), body, None
    except ValueError:
        return {}, body, "state block unparseable: started empty"


def write_state(state, body):
    new = json.dumps(state, indent=2)
    if body is not None and STATE_RE.search(body):
        body = STATE_RE.sub(lambda m: m.group(1) + new + m.group(3), body, count=1)
    else:
        body = (body or "") + "\n\n## lever state\n```json\n" + new + "\n```\n"
    if a.dry_run:
        print("DRY RUN, new state block:\n" + new)
        return
    subprocess.run(["gh", "issue", "edit", a.state_issue, "--body-file", "-"], input=body, text=True, check=True)


def out_set(k, v):
    f = os.environ.get("GITHUB_OUTPUT")
    if f:
        open(f, "a").write(f"{k}={v}\n")


if a.mode == "failure":
    state, body, note = read_state()
    reason = "timed_out" if a.failed_conclusion == "timed_out" else "crashed"
    fl = state.setdefault("failures", [])
    fl.append({"automation": a.failed_workflow, "reason": reason, "at": NOW.isoformat()})
    state["failures"] = [f for f in fl if ts(f["at"]) > NOW - timedelta(days=7)]
    n = sum(1 for f in state["failures"] if f["automation"] == a.failed_workflow and f["reason"] == reason
            and ts(f["at"]) > NOW - timedelta(hours=24))
    key = f"{a.failed_workflow}:{reason}"
    escalate = n >= a.repeat
    if a.state_issue:
        write_state(state, body)
    rec = {"job": "automation-health", "trigger": "workflow_run", "failed_run": a.failed_run, "key": key,
           "count_24h": n, "threshold": a.repeat, "escalate": escalate, "filed": state.get("filed", {}).get(key),
           "state_note": note,
           "status": "ESCALATE" if escalate else f"NOOP(below threshold, {n}/{a.repeat})"}
    open(a.out, "w").write(json.dumps(rec, indent=2))
    out_set("escalate", str(escalate).lower())
    print(json.dumps(rec, indent=2))
    sys.exit(0)

# ---- weekly ---------------------------------------------------------------------------------
state, body, note = read_state()
since = (NOW - timedelta(days=a.window_days)).strftime("%Y-%m-%d")
files = sorted(glob.glob(".github/workflows/lever-*.yml"))
names = {os.path.basename(f) for f in files}
SECRET_RE = re.compile(r"sk-ant-[A-Za-z0-9_-]{10,}|ghp_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}")
report = []
for f in files:
    text = open(f).read()
    try:
        wf = yaml.safe_load(text)
    except yaml.YAMLError as e:
        report.append({"file": f, "flags": [f"unparseable YAML: {e}"]})
        continue
    on = wf.get(True, wf.get("on", {})) or {}
    on = on if isinstance(on, dict) else {k: {} for k in (on if isinstance(on, list) else [on])}
    jobs = wf.get("jobs", {}) or {}
    flags, perms, tools, models, agent_jobs = [], {}, {}, {}, []
    top = [l[1:].strip() for l in text.splitlines()[:12] if l.startswith("#")]
    top_comment = " ".join(top)
    for jn, j in jobs.items():
        p = j.get("permissions", wf.get("permissions"))
        perms[jn] = p
        if p is None:
            flags.append(f"job {jn}: no permissions: block (inherits the repository default)")
        elif p == "write-all" or (isinstance(p, str) and p.startswith("write")):
            flags.append(f"job {jn}: permissions: {p}")
        if "timeout-minutes" not in j:
            flags.append(f"job {jn}: no timeout-minutes")
        if "concurrency" not in j and "concurrency" not in wf:
            flags.append(f"job {jn}: no concurrency:")
        for st in j.get("steps", []) or []:
            if str(st.get("uses", "")).startswith("anthropics/claude-code-action"):
                agent_jobs.append(jn)
                w = st.get("with", {}) or {}
                args = str(w.get("claude_args", ""))
                m = re.search(r'--allowedTools\s+"([^"]*)"', args) or re.search(r"--allowedTools\s+(\S+)", args)
                tl = [t.strip() for t in re.split(r",(?![^(]*\))", m.group(1))] if m else []
                tools[jn] = tl
                mm = re.search(r"--model\s+(\S+)", args)
                models[jn] = mm.group(1) if mm else "default"
                if not m:
                    flags.append(f"job {jn}: no --allowedTools (the default grants more than the contract)")
                if "--max-turns" not in args:
                    flags.append(f"job {jn}: no --max-turns")
                for t in tl:
                    if re.search(r"gh workflow (enable|disable)", t):
                        flags.append(f"job {jn}: FORBIDDEN tool {t}")
                    if t in ("Bash", "Bash(*)") or re.fullmatch(r"Bash\((gh|git)(:\*)?\)", t):
                        flags.append(f"job {jn}: bare shell tool {t}")
                    if re.search(r"gh pr merge", t):
                        flags.append(f"job {jn}: merge tool {t}: needs L3/L4 or a standing grant")
                    if t == "Bash(git push:*)":
                        flags.append(f"job {jn}: unqualified {t}: restrict the remote and branch pattern")
                    if re.search(r"\.github/workflows", t):
                        flags.append(f"job {jn}: FORBIDDEN tool touching .github/workflows: {t}")
                if "opus" in models[jn] and "because" not in top_comment.lower():
                    flags.append(f"job {jn}: opus model with no reason in the top-of-file comment")
                if "allowed_bots" in w:
                    flags.append(f"job {jn}: allowed_bots set: needs a narrowing if:")
    if "pull_request_target" in on:
        flags.append("pull_request_target trigger")
    ev = [k for k in on if k in ("pull_request", "issues", "issue_comment", "pull_request_review")]
    for jn, j in jobs.items():
        if ev and "if" not in j:
            flags.append(f"job {jn}: {','.join(ev)} trigger with no job if: filter")
    if ev and "claude[bot]" not in text:
        flags.append("event-triggered with no claude[bot] actor guard (ok only if documented as loop-safe)")
    if SECRET_RE.search(text):
        flags.append("literal secret in file")
    if not f.endswith("-improver.yml") and f.replace(".yml", "-improver.yml") not in files:
        flags.append("no improver sibling")
    # runs
    rec = {"file": f, "name": wf.get("name"), "kind": "llm" if agent_jobs else "deterministic",
           "triggers": sorted(map(str, on)), "permissions": perms, "allowedTools": tools, "models": models,
           "flags": flags}
    try:
        runs = json.loads(gh("run", "list", "--workflow", os.path.basename(f), "--created", f">={since}",
                             "--limit", "500", "--json", "databaseId,conclusion,status,event,startedAt,updatedAt"))
        c = {}
        for r in runs:
            c[r["conclusion"] or r["status"]] = c.get(r["conclusion"] or r["status"], 0) + 1
        reached = [r for r in runs if r["conclusion"] not in ("skipped", None)]
        rec["runs"] = {"total": len(runs), "by_conclusion": c, "reached_model_or_steps": len(reached)}
        oc = {"ACTED": 0, "NOOP": 0, "BLOCKED": 0, "unrecorded": 0}
        if agent_jobs:
            for r in [r for r in reached if r["conclusion"] == "success"][: a.log_cap]:
                try:
                    log = gh("run", "view", str(r["databaseId"]), "--log")
                    hits = re.findall(r"\b(ACTED|NOOP|BLOCKED)\(", log)
                    oc[hits[-1] if hits else "unrecorded"] += 1
                except Exception:
                    oc["unrecorded"] += 1
            rec["outcomes_sampled"] = oc
            rec["outcomes_note"] = f"read from run logs, at most {a.log_cap} successful runs; unrecorded = no ACTED/NOOP/BLOCKED line found"
        if rec["runs"]["total"] >= a.log_cap and rec["runs"]["total"] >= 500:
            rec["runs"]["note"] = "list capped at 500"
    except Exception as e:
        rec["runs"] = f"unavailable ({e})"
    report.append(rec)

noop_alerts = []
res = {"job": "automation-health", "trigger": "schedule", "window_days": a.window_days, "date": NOW.isoformat(),
       "state": state, "state_note": note, "workflows": report,
       "model_multipliers": "not computed here: cost per useful output needs useful outputs (LLM job) and your price ratios"}
open(a.out, "w").write(json.dumps(res, indent=2))
print(json.dumps({"workflows": [{"file": r["file"], "runs": r.get("runs"), "flags": r["flags"]} for r in report]}, indent=2))
sm = os.environ.get("GITHUB_STEP_SUMMARY")
if sm:
    with open(sm, "a") as fh:
        fh.write("| workflow | runs | flags |\n|---|---|---|\n")
        for r in report:
            fh.write(f"| {r['file']} | {json.dumps(r.get('runs'))[:80]} | {len(r['flags'])} |\n")
