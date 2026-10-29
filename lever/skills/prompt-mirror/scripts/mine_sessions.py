#!/usr/bin/env python3
"""Extract your own typed messages from local Claude Code session history.

Stdlib only. Read-only on the inputs. Writes a local, redacted dataset plus
aggregate stats for prompt-mirror and steer-once. Nothing leaves the machine.

Sources (all optional, merged, de-duplicated by session). <home> is
$CLAUDE_CONFIG_DIR, else ~/.claude:
  <home>/projects/<encoded-cwd>/<session-id>.jsonl      high confidence
  <home>/projects/<encoded-cwd>/<session-id>/subagents/agent-*.jsonl
                                                        subagent traffic, counted
                                                        as machine messages only
  <home>/history.jsonl                                  medium confidence typed-
                                                        prompt log, used only to
                                                        backfill sessions whose
                                                        transcript was cleaned up

The transcript format is internal to Claude Code and drifts between releases.
Readers skip lines they do not understand. Run with --debug-keys to print key
and value histograms (never message text) when the counts look wrong.

Usage:
  python3 mine_sessions.py                     # last 30 days, redacted text
  python3 mine_sessions.py --since 90d --no-text
  python3 mine_sessions.py --since 2026-01-01 --repo my-org/my-repo
  python3 mine_sessions.py --redact-terms ~/.config/prompt-mirror/terms.txt
  python3 mine_sessions.py --debug-keys        # format check, no content
"""

from __future__ import annotations

import argparse
import csv
import json
import os
import re
import statistics
import sys
from collections import Counter, defaultdict
from datetime import datetime, timedelta, timezone
from pathlib import Path

# ---------------------------------------------------------------- filtering

SCRATCH_CWD = re.compile(r"^(/private)?/(tmp|var/folders)(/|$)")
RESUME_TEXT = re.compile(r"^\s*continue from where you left off\.?\s*$", re.I)
# App/agent-generated templates such as "[INLINE CODE REVIEW REQUEST] ...".
BRACKET_TEMPLATE = re.compile(r"^\s*\[\[?[A-Z][A-Z0-9 _:/\-]{3,}\]")
# "[Request interrupted by user]" / "... for tool use]": an Esc abort, not typed text.
INTERRUPT = re.compile(r"^\s*\[Request interrupted by user\b")
# Injected context blocks that wrap or precede typed text.
INJECTED_TAGS = (
    "system-reminder", "system_reminder", "user-prompt-submit-hook",
    "local-command-caveat", "ide_selection", "ide_opened_file",
    "session-start-hook", "pasted_content",
)
INJECTED_BLOCK = re.compile(
    r"<(%s)\b[^>]*>.*?</\1>" % "|".join(re.escape(t) for t in INJECTED_TAGS),
    re.S,
)
# Slash commands, ! bash mode and local command output written into the
# transcript as user turns. Typed, but not prompts: counted, not scored.
COMMAND_MARKERS = re.compile(
    r"<(command-name|command-message|command-args|local-command-stdout|"
    r"local-command-stderr|bash-input|bash-stdout|bash-stderr)>"
)
# Skill bodies, hook feedback and compaction summaries injected as user turns.
META_TEXT = re.compile(
    r"^\s*(Base directory for this skill:|Stop hook feedback:|"
    r"\S+ hook (feedback|error)|Caveat: The messages below were generated|"
    r"This session is being continued from a previous conversation)", re.I)
# If any of these survive stripping, the message was machine-authored:
# peer/cross-session messages, wakes, child-session events, webhooks.
MACHINE_MARKERS = re.compile(
    r"<(agent-message|wake|child-session-event|webhook-payload|task-notification|"
    r"scheduled-task|cross_session_message|teammate-message|channel)\b"
)
SCHEDULED_MARKERS = re.compile(r"<(wake|scheduled-task)\b")
# Machine buckets that count as "work following each typed message".
MACHINE_WORK = ("agent", "agent_kickoff", "automated", "scheduled", "system", "subagent")
AGENT_REPORT = re.compile(r"^\s*#{1,3} ", re.M)
# Third-person briefs ("User request verbatim: ...") written by a coordinator.
AGENT_BRIEF = re.compile(r"\b(user'?s? (request|direction|asks|said|wants)|verbatim|the user)\b", re.I)

TEMPLATE_MIN_CHARS = 60
TEMPLATE_MIN_SESSIONS = 4

# ------------------------------------------------------------ classification
# Approximate English regexes. Treat rates as relative signals, not truth.

FLAGS = {
    "status_ask": r"\b(status|etas?|where are we|any updates?|how'?s it going|"
                  r"what'?s left|hour by hour|time since last|progress report|"
                  r"how (long|much longer))\b",
    "correction": r"(^|\b)(no[,.!]|nope|wrong|why did you|why are you|why is|"
                  r"not what i|i (said|told you|asked)|that'?s not|instead of|"
                  r"stop |don'?t |do not |you (forgot|missed|ignored))",
    "recurrence": r"\b(again|i told you|i already (said|told|asked)|as i said|"
                  r"how many times|didn'?t i (give|tell|ask)|keep (telling|asking)|"
                  r"still (doing|not)|for the \w+ time)\b",
    "stuck": r"\b(still (broken|failing|not working|doesn'?t|the same|slow|seeing|"
             r"red|happening)|doesn'?t work|didn'?t (work|fix|help)|same (error|"
             r"issue|problem|bug)|not fixed|no change)\b",
    "measurement": r"\b(feedback loop|baseline|measur\w*|benchmark|p50|p90|p95|"
                   r"repro\w*|failing test|regression test|write a test|logs?|"
                   r"profil\w*|fps|wall.?clock|metric|a/?b|noise)\b",
    "done_when": r"\b(done when|until|acceptance|success (is|means|criteria)|"
                 r"definition of done|should (pass|be green|work)|ci (is )?green|"
                 r"green|merged?|so that|goal is|expected (result|output)|"
                 r"verify|verified)\b",
    "stop_condition": r"\b(stop (when|if|after)|budget|time.?box|at most|"
                      r"no more than|max(imum)? \d|give up|if (you'?re )?(blocked|"
                      r"stuck)|escalate|within \d+|for the next \d+ ?(h|hours|"
                      r"min)|i'?ll be (asleep|away|out|offline)|by \d{1,2}(am|pm))\b",
    "fanout": r"\b(in parallel|parallel|sub-?sessions?|child sessions?|"
              r"sub-?agents?|fan.?out|1 of \d+|orchestrat\w*|spawn)\b",
    "nudge": r"^\s*(continue|keep going|go|proceed|go on|resume|next)\s*[.!]*\s*$"
             r"|why did you stop|you stopped|are you (still )?working",
    "quote_back": r"(^|\n)\s*> \S|\byou said\b|\bfrom you earlier\b",
    "screenshot_text": r"\bscreenshot\b|\.png\b|\.jpe?g\b|\U0001F4F7",
}

# Correction taxonomy shared with steer-once. One message may hit several.
CLASSES = {
    "gate": r"(don'?t|do not|no) need (authori[sz]ation|approval|permission|a go)|"
            r"preflight|\blocks?\b|bureaucra\w*|ceremony|exact go|"
            r"you (have|already have) (permission|approval)",
    "over_ask": r"don'?t ask|do not ask|stop asking|just do it|without asking|"
                r"no questions|don'?t (wait|check) (on|with) me",
    "premature_stop": r"why did you stop|never stop|don'?t stop|keep going|"
                      r"you stopped|why (are you|is it) idle|not done",
    "polling": r"\bwaiting\b|why (are you|r u) waiting|stop (polling|sleeping)|"
               r"\bsleep(ing)?\b|do them all|while you wait",
    "status_gap": FLAGS["status_ask"],
    "verification_theatre": r"since (the )?last (commit|merge|pr)|what did you "
                            r"(spend|do)|hours? (since|without)|preparation is not|"
                            r"too much (validation|verification)",
    "stale_claim": r"\bstale\b|out of date|not (true|correct|accurate)|are you sure|"
                   r"source of truth|double.?check|that'?s (wrong|incorrect)",
    "shallow_fix": r"band.?aid|root cause|actually fix|instead of (just )?"
                   r"(disabling|skipping|ignoring|suppressing)|workaround|"
                   r"disable (the )?(lint|test)|skip(ping)? (the )?tests?",
    "letter_of_law": r"i (asked|told) you to|you didn'?t follow|not what i "
                     r"(asked|meant|said)|supposed to|didn'?t i (give|tell|ask)",
    "scope_creep": r"why (did|are) you (open|creat|add|chang|touch|edit)\w*|"
                   r"i didn'?t ask|no (extra|additional) (commentary|content|text)|"
                   r"out of scope|don'?t touch",
    "process_default": r"should be (our|the) default|\bby default\b|we always|"
                       r"every time we|our convention|from now on",
    "verbosity": r"too (long|verbose|wordy)|be concise|more concise|shorter|"
                 r"tl;?dr|plain (english|language)|explain (plainly|simply)",
    "runtime_gap": r"re-?kick|stalled|\bquota\b|rate.?limit|disk (space|full)|"
                   r"\bleak(s|ing|ed)?\b|zombie|\bhung\b|crash(ed|es)?",
}

CONCESSION = re.compile(
    r"^\W*(you'?re right|you are right|agreed|fair (point|challenge)|good catch|"
    r"my mistake|i apologi[sz]e|sorry)", re.I)

SENSITIVE = re.compile(
    r"\b(promo(tion)?s?|perf(ormance)? reviews?|calibration|compensation|salary|"
    r"layoffs?|leveling|pip|disciplin\w+|medical|diagnos\w+|therap\w+|"
    r"divorce|lawyer|lawsuit|visa|immigration)\b", re.I)
HEATED = re.compile(
    r"\b(fuck\w*|shit\w*|damn\w*|wtf|stupid|idiot\w*|dumb\w*|crap\w*|"
    r"bullshit|garbage|useless)\b", re.I)

# ---------------------------------------------------------------- redaction

REDACTIONS = [
    (re.compile(r"-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----", re.S), "[SECRET]"),
    (re.compile(r"\b(gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|"
                r"sk-[A-Za-z0-9_\-]{20,}|AKIA[0-9A-Z]{16}|xox[abprs]-[A-Za-z0-9\-]+|"
                r"AIza[0-9A-Za-z_\-]{30,})\b"), "[SECRET]"),
    (re.compile(r"(?i)\b(bearer|token|password|passwd|secret|api[_-]?key)\s*[:=]\s*\S+"), r"\1=[SECRET]"),
    (re.compile(r"\b[0-9a-f]{32,}\b", re.I), "[HEX]"),
    (re.compile(r"\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b", re.I), "[ID]"),
    (re.compile(r"[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}"), "[EMAIL]"),
    (re.compile(r"https?://[^\s)>\]]+"), "[URL]"),
    (re.compile(r"(/Users|/home)/[^/\s]+"), "~"),
    (re.compile(r"[A-Za-z]:\\Users\\[^\\\s]+"), "~"),
    (re.compile(r"(?<![\w/])@[A-Za-z0-9][A-Za-z0-9\-]{1,38}\b"), "@[USER]"),
]


def redact(text: str, extra: list, max_chars: int) -> str:
    for pat, repl in REDACTIONS:
        text = pat.sub(repl, text)
    for pat in extra:
        text = pat.sub("[REDACTED]", text)
    if len(text) > max_chars:
        text = text[:max_chars].rstrip() + " [...truncated]"
    return text


# ------------------------------------------------------------------ helpers

def parse_ts(value):
    if not value:
        return None
    try:
        v = str(value).strip().replace("Z", "+00:00")
        if " " in v and "T" not in v:
            v = v.replace(" ", "T", 1)
        dt = datetime.fromisoformat(v)
        return dt if dt.tzinfo else dt.replace(tzinfo=timezone.utc)
    except ValueError:
        return None


def parse_since(value: str) -> datetime:
    m = re.fullmatch(r"(\d+)d", value)
    if m:
        return datetime.now(timezone.utc) - timedelta(days=int(m.group(1)))
    dt = parse_ts(value)
    if not dt:
        sys.exit(f"bad --since/--until value: {value!r} (use 30d or YYYY-MM-DD)")
    return dt


def clean_text(raw: str) -> str:
    return INJECTED_BLOCK.sub("", raw or "").strip()


def is_machine_text(text: str) -> bool:
    if not text or RESUME_TEXT.match(text) or BRACKET_TEMPLATE.match(text):
        return True
    return bool(MACHINE_MARKERS.search(text))


def looks_like_agent_report(text: str) -> bool:
    # Store rows have no source tag; long multi-heading markdown is almost
    # always an agent-written brief or report rather than something typed.
    words = len(text.split())
    if len(AGENT_REPORT.findall(text)) >= 3 and words > 300:
        return True
    return words > 80 and bool(AGENT_BRIEF.search(text))


def word_count(text: str) -> int:
    return len(re.findall(r"\S+", text))


def compile_map(d):
    return {k: re.compile(v, re.I | re.M) for k, v in d.items()}


FLAG_RE = compile_map(FLAGS)
CLASS_RE = compile_map(CLASSES)


def pct(n, d):
    return round(100.0 * n / d, 1) if d else None


def quantiles(values):
    if not values:
        return {"n": 0}
    s = sorted(values)

    def q(p):
        return s[min(len(s) - 1, int(round(p * (len(s) - 1))))]

    return {"n": len(s), "p50": q(0.5), "p75": q(0.75), "p90": q(0.9),
            "mean": round(statistics.fmean(s), 1)}


def norm_for_template(text: str) -> str:
    # Prefix match catches templates with a variable tail (eval questions, slices).
    return re.sub(r"\s+", " ", text.lower()).strip()[:80]


# ------------------------------------------------------------------ readers

def scan_events(state_dir: Path, since, until, repo_filter, read_assistant, stats):
    """Return (sessions covered by events.jsonl, candidate typed messages)."""
    covered, out = set(), []
    if not state_dir.is_dir():
        return covered, out
    since_epoch = since.timestamp()
    for entry in os.scandir(state_dir):
        path = Path(entry.path) / "events.jsonl"
        try:
            if path.stat().st_mtime < since_epoch:
                continue
            fh = open(path, "r", encoding="utf-8", errors="replace")
        except OSError:
            continue
        sid = entry.name
        covered.add(sid)
        cwd = repo = ""
        first_user_seen = False
        session_rows = []
        with fh:
            for line in fh:
                if '"user.message"' in line:
                    pass
                elif '"session.start"' in line or '"abort"' in line:
                    pass
                elif read_assistant and '"assistant.message"' in line:
                    pass
                else:
                    continue
                try:
                    ev = json.loads(line)
                except json.JSONDecodeError:
                    stats["bad_lines"] += 1
                    continue
                etype = ev.get("type")
                data = ev.get("data") or {}
                if etype == "session.start":
                    ctx = data.get("context") or {}
                    if isinstance(ctx, dict):
                        cwd = ctx.get("cwd") or cwd
                        repo = ctx.get("repository") or repo
                    continue
                ts = parse_ts(ev.get("timestamp"))
                in_window = ts is not None and since <= ts <= until
                if etype == "user.message":
                    is_first = not first_user_seen
                    first_user_seen = True
                if not in_window:
                    continue
                if etype == "abort":
                    if data.get("reason") == "user_initiated":
                        stats["aborts_user"] += 1
                    continue
                if etype == "assistant.message":
                    content = (data.get("content") or "").strip()
                    if content and not ev.get("agentId"):
                        stats["assistant_msgs"] += 1
                        if CONCESSION.match(content):
                            stats["assistant_concessions"] += 1
                    continue
                if etype != "user.message":
                    continue
                source = data.get("source")
                if ev.get("agentId"):
                    stats["machine.subagent"] += 1
                    continue
                if data.get("isAutopilotContinuation") or source == "autopilot":
                    stats["machine.autopilot"] += 1
                    continue
                if isinstance(source, str) and source.startswith("agent-"):
                    stats["machine.agent"] += 1
                    continue
                if source == "system":
                    stats["machine.system"] += 1
                    continue
                if source not in (None, "user"):
                    stats["machine.other"] += 1
                    continue
                text = clean_text(data.get("content") or "")
                if is_machine_text(text):
                    stats["machine.template"] += 1
                    continue
                # A coordinator's kickoff brief to a child session has no source
                # tag; its envelope survives only in transformedContent.
                if MACHINE_MARKERS.search(data.get("transformedContent") or ""):
                    stats["machine.agent_kickoff"] += 1
                    continue
                images = sum(
                    1 for a in (data.get("attachments") or [])
                    if isinstance(a, dict) and str(a.get("mimeType", "")).startswith("image/")
                )
                session_rows.append({
                    "session": sid, "ts": ts, "text": text,
                    "delivery": data.get("delivery"), "mode": data.get("agentMode"),
                    "kickoff": is_first, "images": images, "confidence": "high",
                })
        if not session_rows:
            continue
        if cwd and SCRATCH_CWD.match(cwd):
            stats["dropped.scratch_cwd"] += len(session_rows)
            continue
        if repo_filter and repo_filter not in f"{cwd} {repo}".lower():
            continue
        for r in session_rows:
            r["repo"] = repo or ""
        out.extend(session_rows)
    return covered, out


def open_store(db_path: Path):
    for uri in (f"file:{db_path}?mode=ro", f"file:{db_path}?immutable=1"):
        try:
            con = sqlite3.connect(uri, uri=True, timeout=5)
            con.execute("select 1 from turns limit 1")
            return con
        except sqlite3.Error:
            continue
    return None


def scan_store(db_path: Path, covered, since, until, repo_filter, stats):
    con = open_store(db_path) if db_path.exists() else None
    if con is None:
        return []
    cols = {r[1] for r in con.execute("pragma table_info(sessions)")}
    repo_col = "s.repository" if "repository" in cols else "''"
    cwd_col = "s.cwd" if "cwd" in cols else "''"
    q = (f"select t.session_id, t.turn_index, t.user_message, t.timestamp, "
         f"{cwd_col}, {repo_col} from turns t left join sessions s on s.id = t.session_id "
         f"where t.user_message is not null and t.timestamp >= ?")
    out = []
    for sid, idx, msg, ts_raw, cwd, repo in con.execute(q, (since.strftime("%Y-%m-%d"),)):
        if sid in covered:
            continue
        ts = parse_ts(ts_raw)
        if ts is None or ts < since or ts > until:
            continue
        cwd = cwd or ""
        if SCRATCH_CWD.match(cwd):
            stats["dropped.scratch_cwd"] += 1
            continue
        if repo_filter and repo_filter not in f"{cwd} {repo or ''}".lower():
            continue
        text = clean_text(msg)
        if is_machine_text(text) or looks_like_agent_report(text):
            stats["machine.store_heuristic"] += 1
            continue
        out.append({
            "session": sid, "ts": ts, "text": text, "delivery": None, "mode": None,
            "kickoff": idx == 0, "images": 0, "confidence": "medium", "repo": repo or "",
        })
    con.close()
    return out


# ---------------------------------------------------------------- analysis

def analyse(rows, stats):
    # A prompt that recurs verbatim across many sessions is either a template
    # (eval harness, pasted fan-out prompt) or a person retyping the same ask,
    # which is exactly what automation discovery looks for. Only uncertain-source
    # rows are dropped; high-confidence typed rows are kept and flagged.
    sessions_by_text = defaultdict(set)
    for r in rows:
        if len(r["text"]) >= TEMPLATE_MIN_CHARS:
            sessions_by_text[norm_for_template(r["text"])].add(r["session"])
    templated = {k for k, v in sessions_by_text.items() if len(v) >= TEMPLATE_MIN_SESSIONS}
    kept = []
    for r in rows:
        r["repeat_key"] = None
        if len(r["text"]) >= TEMPLATE_MIN_CHARS and norm_for_template(r["text"]) in templated:
            if r["confidence"] != "high":
                stats["dropped.templated"] += 1
                continue
            r["repeat_key"] = norm_for_template(r["text"])
        kept.append(r)
    kept.sort(key=lambda r: (r["session"], r["ts"]))

    for r in kept:
        t = r["text"]
        r["words"] = word_count(t)
        flags = {k for k, p in FLAG_RE.items() if p.search(t)}
        if r["images"] or "screenshot_text" in flags:
            flags.add("screenshot")
        flags.discard("screenshot_text")
        if r["repeat_key"]:
            flags.add("repeated_prompt")
        r["flags"] = sorted(flags)
        r["classes"] = sorted(k for k, p in CLASS_RE.items() if p.search(t))
        r["heated"] = bool(HEATED.search(t))
        r["sensitive"] = bool(SENSITIVE.search(t))
    return kept


STOPWORDS = set(
    "the a an to of and or is it in on for this that be are was you i me my we our "
    "your with as at by from not do don't dont just please can will would should so "
    "if then there here what why how it's its all any get got have has had".split())


def text_withheld(r, args):
    """Mirror of the message output: text that is withheld there never feeds aggregates."""
    return (args.no_text or (r["sensitive"] and not args.keep_sensitive)
            or (r["heated"] and not args.keep_heated))


def build_stats(rows, stats, args):
    n = len(rows)

    def has(f, rs=rows):
        return sum(1 for r in rs if f in r["flags"])

    delivery = Counter(r["delivery"] for r in rows if r["delivery"])
    dn = sum(delivery.values())
    machine = {k.split(".", 1)[1]: v for k, v in stats.items() if k.startswith("machine.")}
    machine_total = sum(machine.get(k, 0) for k in
                        ("agent", "agent_kickoff", "autopilot", "system", "subagent"))
    kick = [r for r in rows if r["kickoff"]]

    by_session = defaultdict(list)
    for r in rows:
        by_session[r["session"]].append(r)

    loop_smells = []
    for sid, rs in by_session.items():
        stuck = has("stuck", rs)
        shots = has("screenshot", rs)
        measured = has("measurement", rs) > 0
        reasons = []
        if stuck >= 2 and not measured:
            reasons.append(f"stuck x{stuck} with no measurement ask")
        if shots >= 2:
            reasons.append(f"human-as-instrument: {shots} screenshots")
        if len(rs) >= 40 and stuck >= 1:
            reasons.append(f"long session ({len(rs)} typed msgs) with stuck markers")
        if reasons:
            loop_smells.append({
                "session": sid[:8], "typed_msgs": len(rs), "reasons": reasons,
                "first": rs[0]["ts"].isoformat(), "last": rs[-1]["ts"].isoformat(),
            })
    loop_smells.sort(key=lambda x: -x["typed_msgs"])

    class_hits, class_sessions = Counter(), defaultdict(set)
    for r in rows:
        for c in r["classes"]:
            class_hits[c] += 1
            class_sessions[c].add(r["session"])
    classes = {
        c: {"hits": class_hits[c], "sessions": len(class_sessions[c]),
            "per_100_msgs": round(100.0 * class_hits[c] / n, 2)}
        for c in sorted(class_hits, key=lambda c: -class_hits[c])
    }

    bigram_sessions = defaultdict(set)
    for r in rows:
        if not ({"correction", "recurrence"} & set(r["flags"])) or text_withheld(r, args):
            continue
        # Redact first (built-in rules and --redact-terms), then drop the placeholders.
        safe = re.sub(r"@?\[[A-Z]+\]|~", " ", redact(r["text"], args.extra, sys.maxsize))
        plain = re.sub(r"\S*[/\\@]\S*|`[^`]*`", " ", safe.lower())
        toks = [t for t in re.findall(r"[a-z][a-z'\-]+", plain) if t not in STOPWORDS]
        for a, b in set(zip(toks, toks[1:])):
            bigram_sessions[f"{a} {b}"].add(r["session"])
    repeated = sorted(
        ({"phrase": k, "sessions": len(v)} for k, v in bigram_sessions.items()
         if len(v) >= args.min_sessions),
        key=lambda x: -x["sessions"])[:30]

    # Same prompt typed in several sessions: a candidate for an automation.
    groups = defaultdict(list)
    for i, r in enumerate(rows):
        if r["repeat_key"]:
            groups[r["repeat_key"]].append((i, r))
    candidates = []
    for members in groups.values():
        rs = [r for _, r in members]
        cand = {"sessions": len({r["session"] for r in rs}), "messages": len(rs),
                "message_ids": [i for i, _ in members],
                "first": min(r["ts"] for r in rs).isoformat(), "last": max(r["ts"] for r in rs).isoformat()}
        if not any(text_withheld(r, args) for r in rs):
            cand["text"] = redact(rs[0]["text"], args.extra, args.max_chars)
        candidates.append(cand)
    candidates.sort(key=lambda c: (-c["sessions"], -c["messages"]))

    weekly = defaultdict(Counter)
    for r in rows:
        y, w, _ = r["ts"].isocalendar()
        c = weekly[f"{y}-W{w:02d}"]
        c["msgs"] += 1
        for f in ("status_ask", "correction", "recurrence", "nudge", "stuck"):
            c[f] += f in r["flags"]
        if r["delivery"]:
            c["delivery_known"] += 1
            c["steering"] += r["delivery"] == "steering"
    trend = [{
        "week": k, "msgs": c["msgs"],
        "status_ask_pct": pct(c["status_ask"], c["msgs"]),
        "correction_pct": pct(c["correction"], c["msgs"]),
        "recurrence_pct": pct(c["recurrence"], c["msgs"]),
        "nudges": c["nudge"],
        "steering_pct": pct(c["steering"], c["delivery_known"]),
    } for k, c in sorted(weekly.items())]

    am = stats["assistant_msgs"]
    return {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "window": {"since": args.since_dt.isoformat(), "until": args.until_dt.isoformat()},
        "filters": {"repo": args.repo or None, "sources": args.source},
        "typed": {
            "messages": n,
            "sessions": len(by_session),
            "high_confidence": sum(1 for r in rows if r["confidence"] == "high"),
            "medium_confidence": sum(1 for r in rows if r["confidence"] == "medium"),
            "words": quantiles([r["words"] for r in rows]),
        },
        "machine_messages": machine,
        "machine_per_typed": round(machine_total / n, 1) if n else None,
        "dropped": {k.split(".", 1)[1]: v for k, v in stats.items() if k.startswith("dropped.")},
        "delivery": {"known": dn, "steering_pct": pct(delivery["steering"], dn),
                     "queued_pct": pct(delivery["queued"], dn), "idle_pct": pct(delivery["idle"], dn)},
        "rates_pct_of_typed": {
            f: pct(has(f), n) for f in ("status_ask", "correction", "recurrence", "nudge",
                                        "stuck", "measurement", "fanout", "screenshot",
                                        "quote_back")
        },
        "heated_pct": pct(sum(1 for r in rows if r["heated"]), n),
        "aborts_user_initiated": stats["aborts_user"],
        "assistant": {
            "messages": am,
            "concession_openers": stats["assistant_concessions"],
            "concessions_per_100": round(100.0 * stats["assistant_concessions"] / am, 2) if am else None,
        },
        "kickoffs": {
            "n": len(kick),
            "words": quantiles([r["words"] for r in kick]),
            "has_done_when_pct": pct(has("done_when", kick), len(kick)),
            "has_stop_condition_pct": pct(has("stop_condition", kick), len(kick)),
            "has_measurement_pct": pct(has("measurement", kick), len(kick)),
            "fanout_pct": pct(has("fanout", kick), len(kick)),
        },
        "correction_classes": classes,
        "repeated_correction_phrases": repeated,
        "automation_candidates": candidates[:30],
        "loop_smells": loop_smells[:50],
        "weekly": trend,
        "notes": [
            "Regex classifiers are approximate; read samples before trusting a rate.",
            "Delivery (steering/queued/idle) exists only on newer events; store rows lack it.",
            "Medium-confidence rows come from the session store, which has no source tag.",
            "Phrases and candidate text are redacted like messages and skip withheld rows; "
            "--no-text leaves numbers only.",
        ],
    }


# -------------------------------------------------------------------- main

def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--home", default=os.environ.get("COPILOT_HOME", str(Path.home() / ".copilot")),
                    help="Copilot home directory (default: $COPILOT_HOME or ~/.copilot)")
    ap.add_argument("--since", default="30d", help="30d or YYYY-MM-DD (default 30d)")
    ap.add_argument("--until", default=None, help="YYYY-MM-DD (default now)")
    ap.add_argument("--source", choices=("both", "events", "store"), default="both")
    ap.add_argument("--repo", default="", help="only sessions whose cwd/repository contains this")
    ap.add_argument("--out", default=None, help="output dir (default <home>/prompt-mirror/<YYYY-MM-DD>)")
    ap.add_argument("--no-text", action="store_true", help="write stats and flags only, no message text")
    ap.add_argument("--keep-sensitive", action="store_true",
                    help="keep text of messages that look personal/HR (default withheld)")
    ap.add_argument("--keep-heated", action="store_true",
                    help="keep text of profane/heated messages (default withheld)")
    ap.add_argument("--redact-terms", default=None,
                    help="file with one term per line (names, codenames, hosts) to mask")
    ap.add_argument("--max-chars", type=int, default=600, help="truncate each message (default 600)")
    ap.add_argument("--min-sessions", type=int, default=3,
                    help="repeated-phrase threshold in distinct sessions (default 3)")
    ap.add_argument("--skip-assistant", action="store_true",
                    help="skip assistant messages (faster; disables concession counts)")
    ap.add_argument("--format", choices=("json", "csv", "both"), default="both")
    args = ap.parse_args()

    home = Path(os.path.expanduser(args.home))
    args.since_dt = parse_since(args.since)
    args.until_dt = parse_since(args.until) if args.until else datetime.now(timezone.utc)

    extra = args.extra = []
    if args.redact_terms:
        for line in Path(os.path.expanduser(args.redact_terms)).read_text().splitlines():
            term = line.strip()
            if term and not term.startswith("#"):
                extra.append(re.compile(re.escape(term), re.I))

    stats = Counter()
    covered, rows = set(), []
    if args.source in ("both", "events"):
        covered, rows = scan_events(home / "session-state", args.since_dt, args.until_dt,
                                    args.repo.lower(), not args.skip_assistant, stats)
    if args.source in ("both", "store"):
        rows += scan_store(home / "session-store.db", covered, args.since_dt, args.until_dt,
                           args.repo.lower(), stats)
    if not rows:
        print("No typed messages found. Check --home, --since and --source.", file=sys.stderr)

    rows = analyse(rows, stats)
    summary = build_stats(rows, stats, args)

    out = (Path(os.path.expanduser(args.out)) if args.out
           else home / "prompt-mirror" / datetime.now().strftime("%Y-%m-%d"))
    out.mkdir(parents=True, exist_ok=True)
    os.chmod(out, 0o700)

    records = []
    for i, r in enumerate(rows):
        rec = {
            "id": i, "session": r["session"][:8], "ts": r["ts"].isoformat(),
            "confidence": r["confidence"], "delivery": r["delivery"], "mode": r["mode"],
            "kickoff": r["kickoff"], "words": r["words"], "flags": r["flags"],
            "classes": r["classes"], "heated": r["heated"], "sensitive": r["sensitive"],
        }
        if not args.no_text:
            if r["sensitive"] and not args.keep_sensitive:
                rec["text"] = "[withheld: looks personal or HR-related]"
            elif r["heated"] and not args.keep_heated:
                rec["text"] = "[withheld: heated]"
            else:
                rec["text"] = redact(r["text"], extra, args.max_chars)
        records.append(rec)

    written = [out / "stats.json"]
    if args.format in ("json", "both"):
        written.append(out / "messages.jsonl")
        with open(written[-1], "w", encoding="utf-8") as fh:
            for rec in records:
                fh.write(json.dumps(rec, ensure_ascii=False) + "\n")
    if args.format in ("csv", "both"):
        written.append(out / "messages.csv")
        fields = ["id", "session", "ts", "confidence", "delivery", "mode", "kickoff", "words",
                  "flags", "classes", "heated", "sensitive"] + ([] if args.no_text else ["text"])
        with open(written[-1], "w", encoding="utf-8", newline="") as fh:
            w = csv.DictWriter(fh, fieldnames=fields)
            w.writeheader()
            for rec in records:
                w.writerow({**rec, "flags": ";".join(rec["flags"]), "classes": ";".join(rec["classes"])})
    with open(written[0], "w", encoding="utf-8") as fh:
        json.dump(summary, fh, indent=2, default=str)
    for f in written:
        os.chmod(f, 0o600)

    t = summary["typed"]
    print(f"typed messages: {t['messages']} across {t['sessions']} sessions "
          f"(high {t['high_confidence']}, medium {t['medium_confidence']})")
    print(f"median words: {t['words'].get('p50')}  steering: {summary['delivery']['steering_pct']}%  "
          f"status asks: {summary['rates_pct_of_typed']['status_ask']}%")
    print("wrote " + ", ".join(str(p) for p in written))
    return 0


if __name__ == "__main__":
    sys.exit(main())
