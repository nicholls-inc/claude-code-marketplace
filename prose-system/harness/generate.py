#!/usr/bin/env python3
"""Generation harness for the prose experiments.

Each sample is one Messages API call with a fresh context. The call holds the
condition's prompt and nothing else: no system prompt unless the condition
defines one, no tools, no conversation history.

Usage:
    python generate.py --briefs ../briefs/dev.json --conditions ../conditions/exp1-3.json \
        --condition baseline --model MODEL_ID --n 10 --out ../samples
    python generate.py ... --dry-run          # print the plan and the prompts, call nothing

Rules the harness enforces:
    - A stored sample is never overwritten or regenerated. Reruns fill gaps only.
    - A brief set that is not frozen is refused unless --allow-draft is given.
    - A held-out brief set is refused unless --confirm-held-out is given.
    - --max-calls caps the number of API calls in one run.
    - Sampling settings stay at the API defaults. The request sends model,
      max_tokens and messages, plus a system prompt only if the condition has one.
    - No refusal fallback is configured. A fallback would let a second model
      write the sample, so a refusal is stored as a refusal and not replaced.

Every sample is stored as JSON with the exact request, the model that served it,
the stop reason, token usage, the request id and a UTC timestamp. Samples that
ended normally also get a .txt file beside the JSON for scripts/profile.py.

Credentials come from the environment (ANTHROPIC_API_KEY or another source the
SDK resolves). The harness never reads or prints a key.
"""
import argparse
import datetime as dt
import hashlib
import json
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

MAX_TOKENS = 16000


def load_json(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))


def build_request(model, condition, brief):
    """The exact keyword arguments sent to messages.create."""
    user = condition["template"].replace("{prompt}", brief["prompt"])
    req = {"model": model, "max_tokens": MAX_TOKENS,
           "messages": [{"role": "user", "content": user}]}
    if condition.get("system"):
        req["system"] = condition["system"]
    return req


def sample_path(out, condition_name, brief_id, index):
    return Path(out) / condition_name / brief_id / ("%02d.json" % index)


def plan(briefs, conditions, wanted, n, out, model):
    """Every (condition, brief, index) that has no stored sample yet."""
    jobs = []
    for cname in wanted:
        cond = conditions["conditions"][cname]
        for brief in briefs["briefs"]:
            for i in range(1, n + 1):
                path = sample_path(out, cname, brief["id"], i)
                if path.exists():
                    continue
                jobs.append({"path": path, "condition": cname, "brief": brief, "index": i,
                             "request": build_request(model, cond, brief)})
    return jobs


def call(client, job, briefs, conditions):
    import anthropic
    req = job["request"]
    record = {
        "condition": job["condition"],
        "conditions_version": conditions.get("version"),
        "brief_id": job["brief"]["id"],
        "brief_set": briefs.get("set"),
        "brief_set_version": briefs.get("version"),
        "index": job["index"],
        "model_requested": req["model"],
        "request": req,
        "prompt_sha256": hashlib.sha256(
            json.dumps(req, sort_keys=True).encode("utf-8")).hexdigest(),
        "sampling": "API defaults; temperature, top_p, top_k, thinking and effort not sent",
        "timestamp_utc": dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds"),
    }
    try:
        msg = client.messages.create(**req)
    except anthropic.RateLimitError as e:
        return job, None, "rate limited after retries: %s" % e.message
    except anthropic.APIStatusError as e:
        return job, None, "API error %s: %s" % (e.status_code, e.message)
    except anthropic.APIConnectionError as e:
        return job, None, "connection error: %s" % e
    text = "".join(b.text for b in msg.content if b.type == "text")
    record.update({
        "model_served": msg.model,
        "stop_reason": msg.stop_reason,
        "usage": msg.usage.to_dict(),
        "request_id": msg._request_id,
        "sdk_version": anthropic.__version__,
        # A sample counts only if the model finished on its own and served the request itself.
        "valid": msg.stop_reason == "end_turn" and bool(text.strip()),
        "text": text,
    })
    return job, record, None


def store(job, record):
    path = job["path"]
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "x", encoding="utf-8") as f:      # "x": fail rather than overwrite
        json.dump(record, f, indent=2, ensure_ascii=False)
    if record["valid"]:
        with open(path.with_suffix(".txt"), "x", encoding="utf-8") as f:
            f.write(record["text"])


def main(argv=None, client=None):
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--briefs", required=True)
    ap.add_argument("--conditions", required=True)
    ap.add_argument("--condition", action="append", required=True,
                    help="condition name; repeat for several")
    ap.add_argument("--model", required=True, help="exact model identifier; there is no default")
    ap.add_argument("--n", type=int, default=10, help="samples per brief per condition")
    ap.add_argument("--out", required=True, help="samples directory")
    ap.add_argument("--max-calls", type=int, default=0, help="stop after this many calls; 0 is no cap")
    ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--allow-draft", action="store_true")
    ap.add_argument("--confirm-held-out", action="store_true")
    args = ap.parse_args(argv)

    briefs, conditions = load_json(args.briefs), load_json(args.conditions)
    if not briefs.get("frozen") and not args.allow_draft:
        sys.exit("Brief set %r is not frozen. The owner approves and freezes it first. "
                 "Use --allow-draft only for a smoke test." % briefs.get("set"))
    if briefs.get("set") == "held-out" and not args.confirm_held_out:
        sys.exit("This is the held-out set. It confirms a result and is never used for tuning. "
                 "Pass --confirm-held-out if that is what this run is.")
    unknown = [c for c in args.condition if c not in conditions["conditions"]]
    if unknown:
        sys.exit("Unknown condition(s): %s" % ", ".join(unknown))
    ids = [b["id"] for b in briefs["briefs"]]
    if len(ids) != len(set(ids)):
        sys.exit("Duplicate brief ids in %s" % args.briefs)

    jobs = plan(briefs, conditions, args.condition, args.n, args.out, args.model)
    total = len(args.condition) * len(ids) * args.n
    print("%d samples wanted, %d already stored, %d to generate" % (total, total - len(jobs), len(jobs)))
    if args.max_calls and len(jobs) > args.max_calls:
        print("Capped at %d calls by --max-calls" % args.max_calls)
        jobs = jobs[:args.max_calls]
    if args.dry_run:
        seen = set()
        for j in jobs:
            key = (j["condition"], j["brief"]["id"])
            if key not in seen:
                seen.add(key)
                print("\n--- %s / %s ---\n%s" % (key[0], key[1], j["request"]["messages"][0]["content"]))
        return 0
    if not jobs:
        return 0

    if client is None:
        import anthropic
        client = anthropic.Anthropic()
    done = failed = invalid = 0
    with ThreadPoolExecutor(max_workers=args.workers) as pool:
        for fut in as_completed([pool.submit(call, client, j, briefs, conditions) for j in jobs]):
            job, record, err = fut.result()
            if err:
                failed += 1
                print("FAILED %s: %s" % (job["path"], err), file=sys.stderr)
                continue
            store(job, record)
            done += 1
            if not record["valid"]:
                invalid += 1
                print("stored but not valid (%s): %s" % (record["stop_reason"], job["path"]))
            elif record["model_served"] != record["model_requested"]:
                print("note: asked for %s, served by %s: %s"
                      % (record["model_requested"], record["model_served"], job["path"]))
    print("stored %d, of which %d not valid; %d failed and can be retried by rerunning"
          % (done, invalid, failed))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
