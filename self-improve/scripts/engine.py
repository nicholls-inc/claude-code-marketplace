#!/usr/bin/env python3
"""self-improve engine — the deterministic, evidence-keeping half of the routine.

The routine is a controller, not a suggestion box: it measures, acts only on
evidence, and checks whether its own past changes moved the metric.

Subcommands:
  digest    scan recent transcripts + config + state + open evaluations -> JSON
  record    read JSON suggestions from stdin -> validate -> insert 'pending'
  resolve   set a suggestion accepted | rejected | applied | deferred
  evaluate  record the keep/revert verdict on an applied change
  measure   print current proxy metrics (debug)
  commit    git-commit an applied change for audit + one-step revert

Stdlib only. Paths honour $CLAUDE_CONFIG_DIR (default ~/.claude),
$SELF_IMPROVE_DB (default <root>/state.db). Config + lessons live at <root>.
"""
import argparse
import glob
import json
import os
import re
import sqlite3
import subprocess
import sys
import time
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CLAUDE_DIR = Path(os.environ.get("CLAUDE_CONFIG_DIR", Path.home() / ".claude"))
DB_PATH = Path(os.environ.get("SELF_IMPROVE_DB", ROOT / "state.db"))
LESSONS = Path(os.environ.get("SELF_IMPROVE_LESSONS", ROOT / "lessons.md"))

DEFAULT_CONFIG = {
    "slack_channel": "D06UE1F68G4",
    "lookback_days": 7,
    "max_suggestions": 3,
    "challenge_slots": 1,        # quarantine for lower-evidence/speculative picks
    "min_command_recurrence": 5, # don't suggest automating a command seen < this
    "eval_after_days": 7,        # wait this long before judging an applied change
}

METRIC_KINDS = ("correction_rate", "skill_triggers", "command_freq", "none")

CORRECTION_RE = re.compile(
    r"\b(no,|don'?t|do not|actually|that'?s wrong|that is wrong|incorrect|"
    r"stop|revert|undo|instead|i said|not what i|why did you|you shouldn'?t|"
    r"rewrite|redo|that'?s not)\b",
    re.IGNORECASE,
)


def now_iso():
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def now_ts():
    return time.time()


def parse_iso(s):
    try:
        return datetime.fromisoformat(s).timestamp()
    except Exception:
        return None


def load_config():
    cfg = dict(DEFAULT_CONFIG)
    p = ROOT / "config.json"
    if p.exists():
        try:
            cfg.update(json.loads(p.read_text()))
        except Exception:
            pass
    return cfg


# --------------------------------------------------------------------------- #
# state
# --------------------------------------------------------------------------- #
def db():
    con = sqlite3.connect(DB_PATH)
    con.execute(
        """CREATE TABLE IF NOT EXISTS suggestions(
            id INTEGER PRIMARY KEY,
            created_at TEXT, category TEXT, target TEXT, title TEXT,
            rationale TEXT, proposed_change TEXT, benefit TEXT,
            evidence TEXT, falsifier TEXT,
            metric_kind TEXT DEFAULT 'none', metric_key TEXT DEFAULT '',
            is_challenge INTEGER DEFAULT 0, is_self_edit INTEGER DEFAULT 0,
            status TEXT DEFAULT 'pending',
            slack_channel TEXT, slack_ts TEXT,
            applied_at TEXT, baseline_value REAL, baseline_at TEXT,
            eval_verdict TEXT, eval_at TEXT,
            resolved_at TEXT, outcome_note TEXT)"""
    )
    # additive migration: benefit added after the first release; older DBs lack it
    cols = {r[1] for r in con.execute("PRAGMA table_info(suggestions)")}
    if "benefit" not in cols:
        con.execute("ALTER TABLE suggestions ADD COLUMN benefit TEXT DEFAULT ''")
    return con


def append_lesson(line):
    if not LESSONS.exists():
        LESSONS.write_text(
            "# self-improve lessons\n\n"
            "Hard constraints the routine reads every run. Edit freely; the "
            "routine must not propose anything that violates a line here.\n\n"
        )
    with open(LESSONS, "a") as fh:
        fh.write(line.rstrip() + "\n")


# --------------------------------------------------------------------------- #
# transcript scan  (one window-parameterised scan powers digest + metrics)
# --------------------------------------------------------------------------- #
def iter_blocks(obj):
    msg = obj.get("message") if isinstance(obj.get("message"), dict) else obj
    role = msg.get("role") or obj.get("type")
    content = msg.get("content", [])
    if isinstance(content, str):
        content = [{"type": "text", "text": content}]
    if not isinstance(content, list):
        content = []
    return role, content


def scan(since_ts):
    files = glob.glob(str(CLAUDE_DIR / "projects" / "*" / "*.jsonl"))
    files += glob.glob(str(CLAUDE_DIR / "projects" / "*" / "sessions" / "*.jsonl"))
    sessions = 0
    by_project = Counter()
    correction_count = 0
    snippets = []
    bash = Counter()
    skill_inv = Counter()
    sizes = []
    for f in files:
        try:
            if os.path.getmtime(f) < since_ts:
                continue
        except OSError:
            continue
        sessions += 1
        proj = Path(f).parent.name
        if proj == "sessions":
            proj = Path(f).parent.parent.name
        by_project[proj] += 1
        try:
            sizes.append((proj, Path(f).name, os.path.getsize(f)))
            with open(f, "r", errors="ignore") as fh:
                for line in fh:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        obj = json.loads(line)
                    except Exception:
                        continue
                    role, content = iter_blocks(obj)
                    for b in content:
                        if not isinstance(b, dict):
                            continue
                        bt = b.get("type")
                        if bt == "text" and role == "user":
                            txt = b.get("text", "") or ""
                            if txt.lstrip().startswith("/"):
                                skill_inv[txt.lstrip().split()[0]] += 1
                            elif CORRECTION_RE.search(txt):
                                correction_count += 1
                                if len(snippets) < 12:
                                    snippets.append(txt.strip()[:160])
                        elif bt == "tool_use":
                            name = b.get("name", "")
                            inp = b.get("input", {}) or {}
                            if name == "Bash":
                                c = (inp.get("command", "") or "").strip()
                                if c:
                                    bash[c.split("&&")[0].strip()[:60]] += 1
                            elif name == "Skill":
                                skill_inv[str(inp.get("name") or inp.get("skill") or "?")] += 1
        except OSError:
            continue
    sizes.sort(key=lambda t: t[2], reverse=True)
    return {
        "sessions": sessions,
        "by_project": dict(by_project.most_common(10)),
        "correction_count": correction_count,
        "correction_rate": round(correction_count / max(sessions, 1), 3),
        "correction_snippets": snippets,
        "bash": bash,                       # full Counter (for metric matching)
        "skill_invocations": dict(skill_inv),
        "largest_sessions": [
            {"project": p, "file": fn, "bytes": b} for p, fn, b in sizes[:5]
        ],
    }


def metric_value(scan_result, kind, key):
    if kind == "correction_rate":
        return scan_result["correction_rate"]
    if kind == "skill_triggers":
        si = scan_result["skill_invocations"]
        return float(si.get(key, 0) + si.get("/" + key, 0))
    if kind == "command_freq":
        k = (key or "")[:40]
        return float(sum(v for c, v in scan_result["bash"].items() if c.startswith(k)))
    return None


# --------------------------------------------------------------------------- #
# config inventory
# --------------------------------------------------------------------------- #
def config_inventory():
    def names(pat, parent=False):
        return sorted(
            Path(p).parent.name if parent else Path(p).stem
            for p in glob.glob(str(CLAUDE_DIR / pat))
        )

    cm = CLAUDE_DIR / "CLAUDE.md"
    settings = CLAUDE_DIR / "settings.json"
    hooks = False
    if settings.exists():
        try:
            hooks = "hooks" in settings.read_text()
        except Exception:
            pass
    return {
        "claude_md_present": cm.exists(),
        "claude_md_bytes": cm.stat().st_size if cm.exists() else 0,
        "skills": names("skills/*/SKILL.md", parent=True),
        "agents": names("agents/*.md"),
        "commands_legacy": names("commands/*.md"),
        "settings_present": settings.exists(),
        "hooks_configured": hooks,
    }


# --------------------------------------------------------------------------- #
# digest
# --------------------------------------------------------------------------- #
def state_snapshot(cfg):
    con = db()
    pending = [
        dict(zip(
            ["id", "category", "title", "benefit", "slack_channel", "slack_ts",
             "created_at", "is_self_edit"],
            [r[0], r[1], r[2], r[3], r[4], r[5], r[6], bool(r[7])]))
        for r in con.execute(
            """SELECT id,category,title,benefit,slack_channel,slack_ts,created_at,
                      is_self_edit
               FROM suggestions WHERE status='pending' ORDER BY id""")
    ]
    # closed-loop: applied changes old enough to judge, not yet judged
    evals = []
    cutoff = now_ts() - cfg["eval_after_days"] * 86400
    for r in con.execute(
        """SELECT id,title,metric_kind,metric_key,baseline_value,baseline_at,
                  falsifier,applied_at,outcome_note FROM suggestions
           WHERE status='applied' AND metric_kind!='none' AND eval_verdict IS NULL"""
    ):
        (sid, title, mk, mkey, base, base_at, fals, applied_at, note) = r
        a_ts = parse_iso(applied_at or "")
        if a_ts is None or a_ts > cutoff:
            continue  # not enough post-change data yet
        followup = metric_value(scan(a_ts), mk, mkey)
        evals.append({
            "id": sid, "title": title, "metric_kind": mk, "metric_key": mkey,
            "baseline_value": base, "followup_value": followup,
            "falsifier": fals, "applied_at": applied_at,
        })
    acc = {}
    for cat, status, n in con.execute(
        "SELECT category,status,COUNT(*) FROM suggestions GROUP BY category,status"
    ):
        acc.setdefault(cat, {})[status] = n
    suppressed = [
        c for c, s in acc.items()
        if s.get("rejected", 0) >= 2 and not s.get("accepted") and not s.get("applied")
    ]
    recent = [
        {"id": r[0], "category": r[1], "title": r[2], "status": r[3]}
        for r in con.execute(
            """SELECT id,category,title,status FROM suggestions
               WHERE status!='pending' ORDER BY COALESCE(resolved_at,eval_at) DESC
               LIMIT 10""")
    ]
    return {
        "pending": pending,
        "pending_evaluations": evals,
        "acceptance_by_category": acc,
        "suppressed_categories": suppressed,
        "recent_resolved": recent,
    }


def cmd_digest(args):
    cfg = load_config()
    s = scan(now_ts() - cfg["lookback_days"] * 86400)
    thr = cfg["min_command_recurrence"]
    out = {
        "now": now_iso(),
        "config": cfg,
        "claude_dir": str(CLAUDE_DIR),
        "lessons": LESSONS.read_text() if LESSONS.exists() else "",
        "config_inventory": config_inventory(),
        "transcripts": {
            "sessions": s["sessions"],
            "by_project": s["by_project"],
            "correction_count": s["correction_count"],
            "correction_rate": s["correction_rate"],
            "correction_snippets": s["correction_snippets"],
            "actionable_commands": [
                {"cmd": c, "n": n} for c, n in s["bash"].most_common(20) if n >= thr
            ],
            "bash_top": [{"cmd": c, "n": n} for c, n in s["bash"].most_common(15)],
            "skill_invocations": s["skill_invocations"],
            "skills_never_invoked": sorted(
                set(config_inventory()["skills"])
                - {k.lstrip("/") for k in s["skill_invocations"]}),
            "largest_sessions": s["largest_sessions"],
            "_note": "heuristic counts; command recurrence below "
                     f"min_command_recurrence ({thr}) is filtered from actionable_commands",
        },
        "state": state_snapshot(cfg),
    }
    print(json.dumps(out, indent=2))


# --------------------------------------------------------------------------- #
# record  (evidence gate + quarantine)
# --------------------------------------------------------------------------- #
def cmd_record(args):
    cfg = load_config()
    items = json.load(sys.stdin)
    if isinstance(items, dict):
        items = [items]
    grounded, challenge, dropped = [], [], []
    for it in items:
        if not (it.get("benefit") or "").strip():
            dropped.append({"title": it.get("title"), "why": "no benefit (what does the user gain?)"})
            continue
        ch = int(it.get("is_challenge", 0))
        if not ch:
            if not (it.get("evidence") or "").strip():
                dropped.append({"title": it.get("title"), "why": "no evidence (conjecture)"})
                continue
            if not (it.get("falsifier") or "").strip():
                dropped.append({"title": it.get("title"), "why": "no falsifier"})
                continue
            mk = it.get("metric_kind", "none")
            if mk not in METRIC_KINDS:
                dropped.append({"title": it.get("title"), "why": f"bad metric_kind {mk}"})
                continue
            grounded.append(it)
        else:
            challenge.append(it)
    challenge = challenge[: cfg["challenge_slots"]]
    keep = (grounded + challenge)[: cfg["max_suggestions"]]
    con = db()
    ids = []
    for it in keep:
        cur = con.execute(
            """INSERT INTO suggestions(created_at,category,target,title,rationale,
               proposed_change,benefit,evidence,falsifier,metric_kind,metric_key,
               is_challenge,is_self_edit,status,slack_channel,slack_ts)
               VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?, 'pending', ?, ?)""",
            (now_iso(), it.get("category", ""), it.get("target", ""),
             it.get("title", ""), it.get("rationale", ""),
             it.get("proposed_change", ""), it.get("benefit", ""),
             it.get("evidence", ""),
             it.get("falsifier", ""), it.get("metric_kind", "none"),
             it.get("metric_key", ""), int(it.get("is_challenge", 0)),
             int(it.get("is_self_edit", 0)), args.slack_channel, args.slack_ts))
        ids.append(cur.lastrowid)
    con.commit()
    print(json.dumps({"recorded": ids, "dropped": dropped}, indent=2))


# --------------------------------------------------------------------------- #
# resolve / evaluate / measure / commit
# --------------------------------------------------------------------------- #
def cmd_resolve(args):
    cfg = load_config()
    con = db()
    row = con.execute(
        "SELECT category,title,metric_kind,metric_key,baseline_at FROM suggestions WHERE id=?",
        (args.id,)).fetchone()
    if not row:
        print(json.dumps({"error": "no such id"})); return
    category, title, mk, mkey, base_at = row
    fields = {"status": args.status, "resolved_at": now_iso(),
              "outcome_note": args.note or ""}
    if args.status == "applied":
        fields["applied_at"] = now_iso()
        if mk and mk != "none" and not base_at:
            base = metric_value(scan(now_ts() - cfg["lookback_days"] * 86400), mk, mkey)
            fields["baseline_value"] = base
            fields["baseline_at"] = now_iso()
    con.execute(
        f"UPDATE suggestions SET {','.join(k+'=?' for k in fields)} WHERE id=?",
        (*fields.values(), args.id))
    con.commit()
    if args.status == "rejected":
        append_lesson(f"- [{now_iso()[:10]}] ({category}) {title} — rejected: "
                      f"{args.note or 'no reason given'}")
    print(json.dumps({"resolved": args.id, "status": args.status,
                      "baseline_captured": fields.get("baseline_value")}))


def cmd_evaluate(args):
    con = db()
    row = con.execute("SELECT category,title FROM suggestions WHERE id=?",
                      (args.id,)).fetchone()
    con.execute(
        "UPDATE suggestions SET eval_verdict=?, eval_at=?, outcome_note=? WHERE id=?",
        (args.verdict, now_iso(), args.note or "", args.id))
    con.commit()
    if args.verdict in ("nochange", "regressed") and row:
        append_lesson(f"- [{now_iso()[:10]}] ({row[0]}) {row[1]} — {args.verdict} "
                      f"after apply: {args.note or 'metric did not improve'}")
    print(json.dumps({"evaluated": args.id, "verdict": args.verdict}))


def cmd_measure(args):
    cfg = load_config()
    s = scan(now_ts() - cfg["lookback_days"] * 86400)
    print(json.dumps({"correction_rate": s["correction_rate"],
                      "correction_count": s["correction_count"],
                      "sessions": s["sessions"]}, indent=2))


def cmd_commit(args):
    paths = args.paths or [str(CLAUDE_DIR / "skills"), str(CLAUDE_DIR / "CLAUDE.md")]
    try:
        inside = subprocess.run(
            ["git", "-C", str(CLAUDE_DIR), "rev-parse", "--is-inside-work-tree"],
            capture_output=True, text=True)
        if inside.returncode != 0:
            print(json.dumps({"committed": False,
                              "note": f"{CLAUDE_DIR} is not a git repo — run `git init` "
                                      "there (see README) for audit + one-step revert"}))
            return
        subprocess.run(["git", "-C", str(CLAUDE_DIR), "add", *paths],
                       capture_output=True, text=True)
        msg = f"self-improve #{args.id}: {args.message}"
        r = subprocess.run(["git", "-C", str(CLAUDE_DIR), "commit", "-m", msg],
                           capture_output=True, text=True)
        print(json.dumps({"committed": r.returncode == 0,
                          "note": (r.stdout or r.stderr).strip()[:200]}))
    except FileNotFoundError:
        print(json.dumps({"committed": False, "note": "git not found on PATH"}))


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("digest")
    sub.add_parser("measure")
    r = sub.add_parser("record")
    r.add_argument("--slack-channel", default="")
    r.add_argument("--slack-ts", default="")
    rs = sub.add_parser("resolve")
    rs.add_argument("--id", type=int, required=True)
    rs.add_argument("--status", required=True,
                    choices=["accepted", "rejected", "applied", "deferred"])
    rs.add_argument("--note", default="")
    ev = sub.add_parser("evaluate")
    ev.add_argument("--id", type=int, required=True)
    ev.add_argument("--verdict", required=True,
                    choices=["improved", "nochange", "regressed"])
    ev.add_argument("--note", default="")
    cm = sub.add_parser("commit")
    cm.add_argument("--id", type=int, required=True)
    cm.add_argument("--message", default="apply")
    cm.add_argument("--paths", nargs="*")
    args = ap.parse_args()
    {"digest": cmd_digest, "measure": cmd_measure, "record": cmd_record,
     "resolve": cmd_resolve, "evaluate": cmd_evaluate, "commit": cmd_commit}[args.cmd](args)


if __name__ == "__main__":
    main()
