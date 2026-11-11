#!/usr/bin/env python3
"""Build one round of blind paired trials for the reader page.

Usage:
    python3 build_round.py --round r0 --a ../samples/subagent/exp1-3/baseline \
        --b ../samples/subagent/exp2b/plain_prose --brief dev-exp-2 --brief dev-tec-3 \
        --per-brief 1 --seed 1

Writes two files under rounds/<round>/:
    trials.json   published with the page. Opaque trial ids, the brief, and two passages
                  labelled A and B. It holds no condition name, file name or model name.
    key.json      stays in the repository. Says which source is A and which is B.

Which source appears as A is decided per trial by the seeded generator. Trial order
is shuffled per reader by the page.
"""
import argparse
import hashlib
import json
import random
import sys
from pathlib import Path

HERE = Path(__file__).parent


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--round", required=True)
    ap.add_argument("--title", default="")
    ap.add_argument("--a", required=True, help="directory of one source: <brief_id>/<nn>.json")
    ap.add_argument("--b", required=True)
    ap.add_argument("--brief", action="append", required=True)
    ap.add_argument("--per-brief", type=int, default=1)
    ap.add_argument("--seed", type=int, default=1)
    ap.add_argument("--briefs-file", default=str(HERE.parent / "briefs" / "dev.json"))
    args = ap.parse_args()

    prompts = {b["id"]: b["prompt"] for b in json.loads(Path(args.briefs_file).read_text())["briefs"]}
    rng = random.Random(args.seed)
    trials, key = [], {}
    for brief in args.brief:
        picks = []
        for src in (args.a, args.b):
            files = sorted(p for p in (Path(src) / brief).glob("*.json")
                           if json.loads(p.read_text())["valid"])
            if len(files) < args.per_brief:
                sys.exit("not enough valid samples for %s in %s" % (brief, src))
            picks.append(rng.sample(files, args.per_brief))
        for i in range(args.per_brief):
            pair = [picks[0][i], picks[1][i]]
            flipped = rng.random() < 0.5
            if flipped:
                pair.reverse()
            tid = "t" + hashlib.sha1(("%s|%s|%d|%d" % (args.round, brief, i, args.seed)).encode()).hexdigest()[:8]
            texts = [json.loads(p.read_text())["text"].strip() for p in pair]
            trials.append({"id": tid, "brief": prompts[brief], "a": texts[0], "b": texts[1]})
            key[tid] = {"brief_id": brief,
                        "a": str(pair[0].relative_to(HERE.parent.resolve()) if pair[0].is_absolute() else pair[0]),
                        "b": str(pair[1].relative_to(HERE.parent.resolve()) if pair[1].is_absolute() else pair[1])}
    out = HERE / "rounds" / args.round
    out.mkdir(parents=True, exist_ok=True)
    (out / "trials.json").write_text(json.dumps(
        {"round": args.round, "title": args.title, "trials": trials}, ensure_ascii=False, indent=1))
    (out / "key.json").write_text(json.dumps(
        {"round": args.round, "seed": args.seed, "source_a": args.a, "source_b": args.b, "trials": key}, indent=1))
    words = sum(len(t["a"].split()) + len(t["b"].split()) for t in trials)
    print("%d trials, %d words to read, written to %s" % (len(trials), words, out))


if __name__ == "__main__":
    sys.exit(main())
