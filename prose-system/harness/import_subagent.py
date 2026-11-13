#!/usr/bin/env python3
"""Import samples written by Claude Code subagents.

This is the route used when there is no API key. A subagent is given the
condition's prompt and nothing else from the experimenter, but it is not a bare
API call: it also carries the Claude Code agent system prompt, tool definitions
and the repository's CLAUDE.md. Records from this route are marked as such and
are not comparable with records from generate.py.

The text is copied from the subagent's transcript on disk, so it never passes
through the orchestrating agent. The condition and brief are found by matching
the prompt exactly against the brief set and the conditions file.

Usage:
    python import_subagent.py --transcripts DIR --briefs ../briefs/dev.json \
        --conditions ../conditions/exp1-3.json --out ../samples/subagent/pilot \
        --agent ID [--agent ID ...]

A stored sample is never overwritten. An agent already imported is skipped.
"""
import argparse
import hashlib
import json
import sys
from pathlib import Path


def dedent2(block):
    return "\n".join(l[2:] if l.startswith("  ") else l for l in block.split("\n"))


def read_transcript(path):
    relayed, wrapper = None, "agent-tool"
    prompt, text, last_text, model, first_ts, tools, tool_inputs = None, None, None, None, None, [], []
    usage_by_msg, sys_prompt = {}, None
    for line in open(path, encoding="utf-8"):
        d = json.loads(line)
        first_ts = first_ts or d.get("timestamp")
        att = d.get("attachment") or {}
        if att.get("type") == "prompt_snapshot" and sys_prompt is None:
            sys_prompt = json.dumps(att.get("systemPrompt"), sort_keys=True)
        m = d.get("message") or {}
        c = m.get("content")
        if d.get("type") == "user" and isinstance(c, str):
            # A workflow wraps the task: first the user message that launched the workflow,
            # relayed verbatim, then the script's prompt, each indented by two spaces.
            if c.startswith("[Workflow harness \u2014 user request]"):
                relayed = dedent2(c.split("this request wins:\n", 1)[1])
                wrapper = "workflow"
            elif c.startswith("[Workflow harness \u2014 computed task]"):
                prompt = dedent2(c.split("The computed task text follows:\n", 1)[1])
            elif prompt is None and not c.startswith("<system-reminder>"):
                prompt = c
        if m.get("role") != "assistant" or not isinstance(c, list):
            continue
        model = m.get("model") or model
        if m.get("usage") and m.get("id"):
            usage_by_msg[m["id"]] = m["usage"]          # one usage per API message
        for b in c:
            if b.get("type") == "text" and b.get("text", "").strip():
                last_text = b["text"]
            if b.get("type") == "tool_use":
                if b["name"] == "SubagentHandback":
                    text = b["input"].get("message")
                else:
                    tools.append(b["name"])
                    tool_inputs.append(json.dumps(b.get("input"), ensure_ascii=False))
    how = "handback" if text is not None else "final_text"
    if text is None:
        text = last_text                                # agents that return their final text
    usage = {k: sum(u.get(k) or 0 for u in usage_by_msg.values())
             for k in ("input_tokens", "cache_creation_input_tokens",
                       "cache_read_input_tokens", "output_tokens")}
    usage["api_messages"] = len(usage_by_msg)
    return {"prompt": prompt, "text": text, "model": model, "timestamp": first_ts,
            "tools": tools, "usage": usage, "returned_by": how,
            "relayed_request": relayed, "wrapper": wrapper,
            # A run that touched the experiment's own files may have read the brief or other samples.
            "touched_experiment_files": any("prose-system" in t for t in tool_inputs),
            "system_prompt_sha256": hashlib.sha256((sys_prompt or "").encode()).hexdigest()
            if sys_prompt else None}


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--transcripts", required=True)
    ap.add_argument("--briefs", required=True)
    ap.add_argument("--conditions", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--agent", action="append", default=[])
    ap.add_argument("--all", action="store_true", help="import every agent transcript in the directory")
    ap.add_argument("--manifest", help="JSON with jobs: [{prompt, condition, brief_id, source_sample}]; "
                    "used when each prompt is unique, as in a revision pass")
    ap.add_argument("--skip-unmatched", action="store_true",
                    help="skip transcripts whose prompt matches no condition and brief")
    args = ap.parse_args()

    briefs = json.loads(Path(args.briefs).read_text(encoding="utf-8"))
    conds = json.loads(Path(args.conditions).read_text(encoding="utf-8"))
    by_prompt = {c["template"].replace("{prompt}", b["prompt"]): (cn, b["id"])
                 for cn, c in conds["conditions"].items() for b in briefs["briefs"]}
    source = {}
    if args.manifest:
        for j in json.loads(Path(args.manifest).read_text(encoding="utf-8"))["jobs"]:
            by_prompt[j["prompt"]] = (j["condition"], j["brief_id"])
            source[j["prompt"]] = j.get("source_sample")
    out = Path(args.out)
    imported = {json.loads(p.read_text(encoding="utf-8"))["agent_id"] for p in out.rglob("*.json")}

    agents = list(args.agent)
    if args.all:
        agents += sorted(p.name[len("agent-"):-len(".jsonl")]
                         for p in Path(args.transcripts).glob("agent-*.jsonl"))
    runs = []
    for a in agents:
        if a in imported:
            print("already imported:", a)
            continue
        t = read_transcript(Path(args.transcripts) / ("agent-%s.jsonl" % a))
        if t["prompt"] not in by_prompt:
            if args.skip_unmatched:
                print("skipped, no match:", a)
                continue
            sys.exit("agent %s: prompt matches no condition and brief" % a)
        runs.append((t["timestamp"], a, t))
    for _, a, t in sorted(runs):
        cond, brief = by_prompt[t["prompt"]]
        d = out / cond / brief
        d.mkdir(parents=True, exist_ok=True)
        idx = len(list(d.glob("*.json"))) + 1
        text = t["text"] or ""
        rec = {
            "route": "claude-code-subagent",
            "agent_id": a, "condition": cond, "conditions_version": conds.get("version"),
            "brief_id": brief, "brief_set": briefs.get("set"),
            "brief_set_version": briefs.get("version"), "index": idx,
            "model_served": t["model"], "prompt": t["prompt"],
            "source_sample": source.get(t["prompt"]),
            "context_note": "Prompt sent as the subagent's whole task. The subagent also carried "
                            "the Claude Code agent system prompt, tool definitions and CLAUDE.md.",
            "system_prompt_sha256": t["system_prompt_sha256"],
            "sampling": "harness defaults; not set by the experimenter",
            "tools_used": t["tools"], "returned_by": t["returned_by"],
            "wrapper": t["wrapper"], "relayed_user_request": t["relayed_request"],
            "touched_experiment_files": t["touched_experiment_files"],
            "usage": t["usage"], "timestamp_utc": t["timestamp"],
            "valid": bool(text.strip()),
            "text": text,
        }
        with open(d / ("%02d.json" % idx), "x", encoding="utf-8") as f:
            json.dump(rec, f, indent=2, ensure_ascii=False)
        if rec["valid"]:
            with open(d / ("%02d.txt" % idx), "x", encoding="utf-8") as f:
                f.write(text)
        print("%s -> %s/%s/%02d  %s  tools=%s  out=%d" % (
            a, cond, brief, idx, t["model"], t["tools"], t["usage"]["output_tokens"]))


if __name__ == "__main__":
    sys.exit(main())
