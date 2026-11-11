#!/usr/bin/env python3
"""Tables for experiments on stored samples. Standard library only.

Usage:
    python3 analyse.py SAMPLES_DIR [--json out.json] [--boot 2000]

SAMPLES_DIR holds <condition>/<brief_id>/<nn>.json records. The script reports each
measure pooled by condition, and by condition within each kind of prose, with a 95%
bootstrap interval. The bootstrap resamples whole samples within each brief, so a
brief keeps its weight and the interval reflects variation between samples.

Rhythm measures use sentence pairs inside a document only. No pair spans two samples.
"""
import argparse
import importlib.util
import json
import random
import re
import sys
from collections import defaultdict
from pathlib import Path

spec = importlib.util.spec_from_file_location("prose_profile", Path(__file__).with_name("profile.py"))
P = importlib.util.module_from_spec(spec)
spec.loader.exec_module(P)

CONJ = {"and", "but", "so", "or", "yet", "because", "which", "nor"}
MEASURES = ["words_per_sample", "mean_len", "cv_len", "lag1", "short_long", "le4_share",
            "conj_start", "em_dash", "semicolon", "colon", "paren", "joiners", "comma",
            "numbers_per_100", "names_per_100", "heading_share", "list_share"]


def features(text):
    """Raw counts for one sample."""
    m = P.measure(text)
    cleaned = P.clean(text)
    sents = [s for s in P.sentences(cleaned) if P.WORD.findall(s)]
    lens = m["lens"]
    firsts = [P.WORD.findall(s)[0].lower() for s in sents]
    return {
        "lens": lens, "words": m["words"], "numbers": m["numbers"], "names": m["names"],
        "marks": m["marks"], "comma": cleaned.count(","),
        "le4": sum(1 for x in lens if x <= 4),
        "conj": sum(1 for w in firsts if w in CONJ),
        "heading": bool(re.search(r"(?m)^\s*(#{1,6}\s|(\*\*|__)[^*_\n]+\2:?\s*$)", text)),
        "list": bool(re.search(r"(?m)^\s*(?:[-*+]|\d+[.)])\s", text)),
        "raw_dash": text.count("—") + text.count("–"),
    }


def pooled(samples):
    n = len(samples)
    lens = [x for s in samples for x in s["lens"]]
    words = sum(s["words"] for s in samples)
    if not lens or not words:
        return {}
    mean = sum(lens) / len(lens)
    var = sum((x - mean) ** 2 for x in lens) / len(lens)
    pairs = num = sl = 0
    for s in samples:
        L = s["lens"]
        for i in range(len(L) - 1):
            pairs += 1
            num += (L[i] - mean) * (L[i + 1] - mean)
            sl += L[i] <= 8 and L[i + 1] >= 2.5 * L[i]
    k = 1000 / words
    marks = {key: sum(s["marks"][key] for s in samples) for key in ("em_dash", "semicolon", "colon", "paren")}
    out = {
        "words_per_sample": words / n, "mean_len": mean, "cv_len": var ** 0.5 / mean,
        "lag1": (num / pairs) / var if pairs and var else 0.0,
        "short_long": sl / pairs if pairs else 0.0,
        "le4_share": sum(s["le4"] for s in samples) / len(lens),
        "conj_start": sum(s["conj"] for s in samples) / len(lens),
        "comma": sum(s["comma"] for s in samples) * k,
        "numbers_per_100": sum(s["numbers"] for s in samples) * 100 / words,
        "names_per_100": sum(s["names"] for s in samples) * 100 / words,
        "heading_share": sum(s["heading"] for s in samples) / n,
        "list_share": sum(s["list"] for s in samples) / n,
    }
    for key, c in marks.items():
        out[key] = c * k
    out["joiners"] = sum(marks.values()) * k
    return out


def boot(by_brief, reps, rng):
    """by_brief: {brief: [features]}. Returns {measure: (point, lo, hi)}."""
    point = pooled([s for v in by_brief.values() for s in v])
    draws = defaultdict(list)
    for _ in range(reps):
        res = [rng.choice(v) for v in by_brief.values() for _ in v]
        for key, val in pooled(res).items():
            draws[key].append(val)
    out = {}
    for key in MEASURES:
        d = sorted(draws[key])
        out[key] = (point[key], d[int(0.025 * len(d))], d[int(0.975 * len(d)) - 1]) if d else (point[key],) * 3
    return out


def fmt(key, t):
    dec = 0 if key == "words_per_sample" else (1 if key in ("mean_len", "em_dash", "semicolon", "colon", "paren", "joiners", "comma", "numbers_per_100", "names_per_100") else 2 if key in ("cv_len", "lag1", "heading_share", "list_share") else 3)
    return "%.*f [%.*f, %.*f]" % (dec, t[0], dec, t[1], dec, t[2])


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("dir")
    ap.add_argument("--json")
    ap.add_argument("--boot", type=int, default=2000)
    ap.add_argument("--briefs", default=str(Path(__file__).parent.parent / "briefs" / "dev.json"))
    args = ap.parse_args()
    kinds = {b["id"]: b["kind"] for b in json.loads(Path(args.briefs).read_text())["briefs"]}
    rng = random.Random(20261111)

    data = defaultdict(lambda: defaultdict(list))      # condition -> brief -> [features]
    raw_dash = defaultdict(int)
    for p in sorted(Path(args.dir).glob("*/*/*.json")):
        r = json.loads(p.read_text(encoding="utf-8"))
        if not r.get("valid"):
            continue
        f = features(r["text"])
        data[r["condition"]][r["brief_id"]].append(f)
        raw_dash[r["condition"]] += f["raw_dash"]

    result = {}
    groups = ["all"] + sorted(set(kinds.values()))
    for g in groups:
        result[g] = {}
        for cond, by_brief in data.items():
            sub = {b: v for b, v in by_brief.items() if g == "all" or kinds.get(b) == g}
            if sub:
                result[g][cond] = boot(sub, args.boot, rng)
    conds = list(data)
    for g in groups:
        print("\n## %s" % g)
        print("| measure | " + " | ".join(conds) + " |")
        print("| --- |" + " --- |" * len(conds))
        for key in MEASURES:
            print("| %s | " % key + " | ".join(fmt(key, result[g][c][key]) if c in result[g] else "" for c in conds) + " |")
    print("\nraw em and en dash characters per condition:", dict(raw_dash))
    print("samples per condition:", {c: sum(len(v) for v in d.values()) for c, d in data.items()})
    if args.json:
        Path(args.json).write_text(json.dumps(result, indent=1))


if __name__ == "__main__":
    sys.exit(main())
