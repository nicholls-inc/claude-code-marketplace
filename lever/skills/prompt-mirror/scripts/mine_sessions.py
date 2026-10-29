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
import hashlib
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
    if value is None or value == "":
        return None
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        # history.jsonl uses epoch milliseconds.
        try:
            return datetime.fromtimestamp(value / 1000 if value > 1e11 else value, timezone.utc)
        except (OverflowError, OSError, ValueError):
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
    # History rows have no source tag; long multi-heading markdown is almost
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
#
# Claude Code transcript lines (verified 2026-10; internal, drifts):
#   {type: "user"|"assistant"|"attachment"|"queue-operation"|"system"|...,
#    uuid, parentUuid, sessionId, timestamp, cwd, gitBranch, isSidechain,
#    userType, entrypoint, version, message: {role, content}, ...}
# A user line is typed by a person when turnOrigin == "human" or
# origin.kind == "human". Older builds carry neither; then a user line with
# text content, no toolUseResult, not isMeta / isCompactSummary and no
# injected markers is a candidate. Mid-turn messages arrive as an
# attachment {type: "queued_command", origin, prompt, commandMode}; a
# queue-operation "enqueue" long before the user turn means "queued".

QUEUED_GAP_S = 2.0
ORIGIN_BUCKETS = (
    ("human", None), ("peer", "agent"), ("agent", "agent"), ("coordinator", "agent_kickoff"),
    ("parent", "agent_kickoff"), ("task", "system"), ("notification", "system"),
    ("hook", "meta"), ("sched", "scheduled"), ("cron", "scheduled"), ("loop", "scheduled"),
    ("wake", "scheduled"), ("trigger", "scheduled"), ("webhook", "agent"),
)


def origin_bucket(value):
    """Map a non-human turnOrigin / origin.kind to a machine bucket."""
    v = str(value).lower()
    for needle, bucket in ORIGIN_BUCKETS[1:]:
        if needle in v:
            return bucket
    return "automated"


def entry_origin(ev):
    """Return (origin_value or None). 'human' means typed by a person."""
    to = ev.get("turnOrigin")
    if isinstance(to, str) and to:
        return to
    o = ev.get("origin")
    if isinstance(o, dict) and o.get("kind"):
        return str(o["kind"])
    return None


def content_parts(message):
    """Return (text, n_images, has_tool_result) from a message.content."""
    if not isinstance(message, dict):
        return "", 0, False
    c = message.get("content")
    if isinstance(c, str):
        return c, 0, False
    text, images, tool = [], 0, False
    if isinstance(c, list):
        for b in c:
            if not isinstance(b, dict):
                continue
            bt = b.get("type")
            if bt == "text" and isinstance(b.get("text"), str):
                text.append(b["text"])
            elif bt == "image":
                images += 1
            elif bt == "tool_result":
                tool = True
    return "\n".join(text), images, tool


_REPO_CACHE = {}
_REMOTE = re.compile(r"url\s*=\s*\S*?[:/]([\w.\-]+/[\w.\-]+?)(\.git)?\s*$", re.M)


def repo_for_cwd(cwd):
    """owner/name from <cwd>/.git/config if it is still on disk (read-only)."""
    if not cwd:
        return ""
    if cwd not in _REPO_CACHE:
        repo = ""
        try:
            cfg = Path(cwd, ".git", "config").read_text(errors="replace")
            m = _REMOTE.search(cfg)
            repo = m.group(1) if m else ""
        except OSError:
            pass
        _REPO_CACHE[cwd] = repo
    return _REPO_CACHE[cwd]


def iter_jsonl(path: Path, stats):
    try:
        fh = open(path, "r", encoding="utf-8", errors="replace")
    except OSError:
        return
    with fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                ev = json.loads(line)
            except json.JSONDecodeError:
                stats["bad_lines"] += 1
                continue
            if isinstance(ev, dict):
                yield ev


def classify_text(raw):
    """Return (bucket, cleaned_text). bucket None means candidate typed text."""
    if INTERRUPT.match(raw or ""):
        return "interrupt", ""
    if COMMAND_MARKERS.search(raw or ""):
        return "command", ""
    text = clean_text(raw)
    if not text:
        return "meta", ""
    if META_TEXT.match(text):
        return "meta", ""
    if SCHEDULED_MARKERS.search(text):
        return "scheduled", ""
    if MACHINE_MARKERS.search(text):
        return "agent", ""
    if is_machine_text(text):
        return "template", ""
    return None, text


def count_subagents(session_dir: Path, since, until, stats):
    """Prompts inside subagent transcripts: machine traffic, never typed."""
    sub = session_dir / "subagents"
    if not sub.is_dir():
        return
    for f in sub.glob("agent-*.jsonl"):
        for ev in iter_jsonl(f, stats):
            if ev.get("type") != "user" or ev.get("toolUseResult") is not None:
                continue
            ts = parse_ts(ev.get("timestamp"))
            if ts is None or not (since <= ts <= until):
                continue
            _, _, tool = content_parts(ev.get("message"))
            if not tool and not ev.get("isMeta"):
                stats["machine.subagent"] += 1


def scan_transcripts(projects: Path, since, until, repo_filter, read_assistant, stats):
    """Return (covered session ids, spans by cwd, typed texts by cwd, rows)."""
    covered, spans, seen_text, out = set(), defaultdict(list), defaultdict(set), []
    if not projects.is_dir():
        return covered, spans, seen_text, out
    since_epoch = since.timestamp()
    for proj in os.scandir(projects):
        if not proj.is_dir():
            continue
        for entry in os.scandir(proj.path):
            if not entry.name.endswith(".jsonl") or not entry.is_file():
                continue
            path = Path(entry.path)
            try:
                if path.stat().st_mtime < since_epoch:
                    continue
            except OSError:
                continue
            sid = entry.name[:-6]
            covered.add(sid)
            scan_one(path, sid, since, until, repo_filter, read_assistant, stats,
                     spans, seen_text, out, covered)
    return covered, spans, seen_text, out


def scan_one(path, sid, since, until, repo_filter, read_assistant, stats,
             spans, seen_text, out, covered):
    cwd = ""
    first_prompt_seen = False
    session_rows, session_machine = [], Counter()
    enqueued = []          # (ts, text prefix) of queue-operation enqueues
    qc_seen = set()        # queued_command prompts already counted
    absorbed = []          # (ts, text) from queue-operation remove/absorbed_mid_turn
    concession_ids = set()
    first_ts = last_ts = None
    for ev in iter_jsonl(path, stats):
        etype = ev.get("type")
        if ev.get("sessionId"):
            covered.add(str(ev["sessionId"]))
        if isinstance(ev.get("cwd"), str) and ev["cwd"]:
            cwd = cwd or ev["cwd"]
        ts = parse_ts(ev.get("timestamp"))
        if ts is not None:
            first_ts = ts if first_ts is None or ts < first_ts else first_ts
            last_ts = ts if last_ts is None or ts > last_ts else last_ts
        in_window = ts is not None and since <= ts <= until
        if ev.get("isSidechain") or ev.get("agentId"):
            # Older builds kept subagent traffic in the main file.
            if (etype == "user" and in_window and not ev.get("isMeta")
                    and ev.get("toolUseResult") is None
                    and not content_parts(ev.get("message"))[2]):
                session_machine["subagent"] += 1
            continue

        if etype == "queue-operation":
            op, content = ev.get("operation"), ev.get("content")
            if op == "enqueue" and isinstance(content, str) and ts is not None:
                enqueued.append((ts, content.strip()[:200]))
            elif (op == "remove" and isinstance(content, str) and in_window
                  and "mid_turn" in str(ev.get("reason") or "")):
                absorbed.append((ts, content))
            continue

        if etype == "attachment":
            att = ev.get("attachment")
            if not isinstance(att, dict) or att.get("type") != "queued_command" or not in_window:
                continue
            prompt = att.get("prompt")
            if not isinstance(prompt, str):
                continue
            qc_seen.add(prompt.strip()[:200])
            origin = (att.get("origin") or {}).get("kind") if isinstance(att.get("origin"), dict) else None
            mode = att.get("commandMode")
            if origin and origin != "human":
                session_machine[origin_bucket(origin)] += 1
                continue
            if mode not in (None, "prompt") or (att.get("isMeta") and origin != "human"):
                session_machine["system" if mode == "task-notification" else "meta"] += 1
                continue
            bucket, text = classify_text(prompt)
            if bucket:
                session_machine[bucket] += 1
                continue
            session_rows.append({"session": sid, "ts": ts, "text": text, "delivery": "steering",
                                 "mode": None, "kickoff": False, "images": 0, "confidence": "high"})
            continue

        if etype == "assistant":
            if not (read_assistant and in_window):
                continue
            msg = ev.get("message") or {}
            if not isinstance(msg, dict) or msg.get("model") == "<synthetic>":
                continue
            text, _, _ = content_parts(msg)
            mid = msg.get("id") or ev.get("uuid")
            if not text.strip() or mid in concession_ids:
                continue
            concession_ids.add(mid)
            stats["assistant_msgs"] += 1
            if CONCESSION.match(text.strip()):
                stats["assistant_concessions"] += 1
            continue

        if etype != "user":
            continue
        raw, images, tool = content_parts(ev.get("message"))
        if ev.get("toolUseResult") is not None or tool:
            continue                      # tool_result turn
        is_first = not first_prompt_seen
        if not ev.get("isMeta"):
            first_prompt_seen = True
        if not in_window:
            continue
        if INTERRUPT.match(raw or ""):
            stats["aborts_user"] += 1
            continue
        if ev.get("isCompactSummary"):
            session_machine["compact"] += 1
            continue
        if ev.get("isMeta"):
            session_machine["meta"] += 1
            continue
        origin = entry_origin(ev)
        if origin and origin != "human":
            session_machine[origin_bucket(origin)] += 1
            continue
        bucket, text = classify_text(raw)
        if bucket == "agent" and is_first:
            bucket = "agent_kickoff"      # a coordinator's brief to this child session
        if bucket:
            session_machine[bucket] += 1
            continue
        delivery = "idle"
        key = (raw or "").strip()[:200]
        for ets, etext in reversed(enqueued):
            if etext == key and ets <= ts:
                if (ts - ets).total_seconds() > QUEUED_GAP_S:
                    delivery = "queued"
                break
        session_rows.append({
            "session": sid, "ts": ts, "text": text, "delivery": delivery,
            "mode": ev.get("permissionMode"), "kickoff": is_first, "images": images,
            "confidence": "high",
        })

    # Builds that log mid-turn absorption without a queued_command attachment.
    for ats, content in absorbed:
        if content.strip()[:200] in qc_seen:
            continue
        bucket, text = classify_text(content)
        if bucket:
            session_machine[bucket] += 1
            continue
        session_rows.append({"session": sid, "ts": ats, "text": text, "delivery": "steering",
                             "mode": None, "kickoff": False, "images": 0, "confidence": "high"})

    count_subagents(path.with_suffix(""), since, until, stats)
    if cwd and first_ts:
        spans[cwd].append((first_ts, last_ts))
    if cwd and SCRATCH_CWD.match(cwd):
        stats["dropped.scratch_cwd"] += len(session_rows)
        return
    repo = repo_for_cwd(cwd)
    if repo_filter and repo_filter not in f"{cwd} {repo}".lower():
        return
    for k, v in session_machine.items():
        stats[f"machine.{k}"] += v
    for r in session_rows:
        r["repo"] = repo
        seen_text[cwd].add(r["text"].strip())
    out.extend(session_rows)


def scan_history(path: Path, covered, spans, seen_text, since, until, repo_filter, stats):
    """Backfill from the typed-prompt log for sessions with no transcript left."""
    if not path.is_file():
        return []
    out, last_by_project = [], {}
    for ev in iter_jsonl(path, stats):
        text_raw = ev.get("display")
        if not isinstance(text_raw, str):
            continue
        ts = parse_ts(ev.get("timestamp"))
        if ts is None or ts < since or ts > until:
            continue
        cwd = str(ev.get("project") or "")
        sid = str(ev.get("sessionId") or "")
        if sid and sid in covered:
            continue
        if any(a <= ts <= b for a, b in spans.get(cwd, ())):
            continue                      # transcript exists and already decided
        if text_raw.strip() in seen_text.get(cwd, ()):
            continue
        if SCRATCH_CWD.match(cwd):
            stats["dropped.scratch_cwd"] += 1
            continue
        repo = repo_for_cwd(cwd)
        if repo_filter and repo_filter not in f"{cwd} {repo}".lower():
            continue
        stripped = text_raw.strip()
        if stripped.startswith("/") or stripped.startswith("!"):
            stats["machine.command"] += 1
            continue
        bucket, text = classify_text(text_raw)
        if bucket or looks_like_agent_report(text):
            stats["machine.history_heuristic"] += 1
            continue
        if not sid:
            # No session id: a new pseudo-session after a 30-minute gap per project.
            prev = last_by_project.get(cwd)
            if prev is None or (ts - prev[0]).total_seconds() > 1800:
                prev = (ts, "h" + hashlib.sha1(f"{cwd}|{ts.isoformat()}".encode()).hexdigest()[:11])
            last_by_project[cwd] = (ts, prev[1])
            sid = prev[1]
        out.append({"session": sid, "ts": ts, "text": text, "delivery": None, "mode": None,
                    "kickoff": False, "images": 0, "confidence": "medium", "repo": repo})
    first = {}
    for r in sorted(out, key=lambda r: r["ts"]):
        first.setdefault(r["session"], r)
    for r in first.values():
        r["kickoff"] = True
    return out


def debug_keys(home: Path, since):
    """Print key and value histograms of the sources. Never prints message text."""
    stats = Counter()
    keys, values, blocks, atts, hist = Counter(), Counter(), Counter(), Counter(), Counter()
    files = Counter()
    short = ("type", "turnOrigin", "promptSource", "isMeta", "isSidechain", "userType",
             "entrypoint", "operation", "reason", "isCompactSummary", "permissionMode", "subtype")
    projects = home / "projects"
    since_epoch = since.timestamp()
    for path in (projects.glob("*/*.jsonl") if projects.is_dir() else ()):
        kinds = [("main", path)] + [("subagent", p) for p in path.with_suffix("").glob("subagents/agent-*.jsonl")]
        for kind, p in kinds:
            try:
                if p.stat().st_mtime < since_epoch:
                    continue
            except OSError:
                continue
            files[kind] += 1
            for ev in iter_jsonl(p, stats):
                t = ev.get("type")
                for k in ev:
                    keys[(kind, t, k)] += 1
                for k in short:
                    if k in ev and not isinstance(ev[k], (dict, list)):
                        values[(kind, t, k, str(ev[k])[:40])] += 1
                if isinstance(ev.get("origin"), dict):
                    values[(kind, t, "origin.kind", str(ev["origin"].get("kind"))[:40])] += 1
                msg = ev.get("message")
                if isinstance(msg, dict):
                    c = msg.get("content")
                    if isinstance(c, str):
                        blocks[(kind, t, "str")] += 1
                    elif isinstance(c, list):
                        for b in c:
                            if isinstance(b, dict):
                                blocks[(kind, t, str(b.get("type")))] += 1
                att = ev.get("attachment")
                if isinstance(att, dict):
                    o = att.get("origin")
                    atts[(kind, str(att.get("type")), str(att.get("commandMode")),
                          str(o.get("kind")) if isinstance(o, dict) else "-")] += 1
    for ev in iter_jsonl(home / "history.jsonl", stats):
        for k in ev:
            hist[k] += 1

    def dump(title, counter):
        print(f"## {title}")
        for k, v in sorted(counter.items(), key=lambda kv: (str(kv[0]))):
            print(f"  {v:6d}  {' | '.join(map(str, k)) if isinstance(k, tuple) else k}")

    print(f"home: {home}  since: {since.date()}  bad_lines: {stats['bad_lines']}")
    dump("files", files)
    dump("keys (file, type, key)", keys)
    dump("short values (file, type, key, value)", values)
    dump("message.content blocks (file, type, block)", blocks)
    dump("attachments (file, type, commandMode, origin.kind)", atts)
    dump("history.jsonl keys", hist)


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
    machine_total = sum(machine.get(k, 0) for k in MACHINE_WORK)
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
            "Delivery: steering = absorbed mid-turn (queued_command), queued = enqueued "
            "while a turn ran and delivered after it, idle = otherwise; history rows lack it.",
            "Medium-confidence rows come from ~/.claude/history.jsonl, which has no source tag.",
            "machine_per_typed counts subagent prompts, peer/cross-session messages, "
            "scheduled wakes, task notifications and non-human turn origins; slash "
            "commands, skill/hook injections and compaction summaries are counted "
            "separately and excluded.",
            "The transcript format is internal to Claude Code; rerun with --debug-keys "
            "if counts look wrong.",
            "Phrases and candidate text are redacted like messages and skip withheld rows; "
            "--no-text leaves numbers only.",
        ],
    }


# -------------------------------------------------------------------- main

def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--home", default=os.environ.get("CLAUDE_CONFIG_DIR") or str(Path.home() / ".claude"),
                    help="Claude Code config directory (default: $CLAUDE_CONFIG_DIR or ~/.claude)")
    ap.add_argument("--since", default="30d", help="30d or YYYY-MM-DD (default 30d)")
    ap.add_argument("--until", default=None, help="YYYY-MM-DD (default now)")
    ap.add_argument("--source", choices=("both", "transcripts", "history", "events", "store"),
                    default="both",
                    help="transcripts (projects/*.jsonl), history (history.jsonl) or both; "
                         "events/store are accepted aliases")
    ap.add_argument("--repo", default="", help="only sessions whose cwd/repository contains this")
    ap.add_argument("--out", default=None, help="output dir (default <home>/lever/prompt-mirror/<YYYY-MM-DD>)")
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
    ap.add_argument("--debug-keys", action="store_true",
                    help="print key/value histograms of the sources (no message text) and exit")
    args = ap.parse_args()
    args.source = {"events": "transcripts", "store": "history"}.get(args.source, args.source)

    home = Path(os.path.expanduser(args.home))
    args.since_dt = parse_since(args.since)
    args.until_dt = parse_since(args.until) if args.until else datetime.now(timezone.utc)

    extra = args.extra = []
    if args.redact_terms:
        for line in Path(os.path.expanduser(args.redact_terms)).read_text().splitlines():
            term = line.strip()
            if term and not term.startswith("#"):
                extra.append(re.compile(re.escape(term), re.I))

    if args.debug_keys:
        debug_keys(home, args.since_dt)
        return 0

    stats = Counter()
    covered, spans, seen_text, rows = set(), {}, {}, []
    if args.source in ("both", "transcripts"):
        covered, spans, seen_text, rows = scan_transcripts(
            home / "projects", args.since_dt, args.until_dt, args.repo.lower(),
            not args.skip_assistant, stats)
    if args.source in ("both", "history"):
        rows += scan_history(home / "history.jsonl", covered, spans, seen_text,
                             args.since_dt, args.until_dt, args.repo.lower(), stats)
    if not rows:
        print("No typed messages found. Check --home, --since and --source.", file=sys.stderr)

    rows = analyse(rows, stats)
    summary = build_stats(rows, stats, args)

    out = (Path(os.path.expanduser(args.out)) if args.out
           else home / "lever" / "prompt-mirror" / datetime.now().strftime("%Y-%m-%d"))
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
