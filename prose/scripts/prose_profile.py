#!/usr/bin/env python3
"""Surface profile of prose, compared with a reference of human-written text.

Standard library only.

    prose_profile.py check FILE [FILE...] [--kind KIND] [--json]
        Measures each file and shows where it sits against the human reference.
    prose_profile.py rank FILE [FILE...] [--kind KIND] [--json]
        Orders candidate drafts by closeness to the human reference on the rhythm measures.
    prose_profile.py specifics DRAFT --material FILE [FILE...]
        Lists numbers and names in the draft that do not appear in the material.

KIND is one of: explanation, argument, narrative, technical. Without it the pooled
reference is used.

This is an alarm, not a quality score. A draft can sit inside every range and read
badly. The ranges are the 10th to 90th percentile across 133 human documents, so one
human document in five falls outside each range. Texts under about 300 words are too
short for the rhythm measures.
"""
import argparse
import json
import math
import re
import sys
from pathlib import Path

# A word is a dotted abbreviation (U.S., a.m., e.g.), a number with internal separators
# (3.5, 1,200) or a run of letters and digits with internal apostrophes and hyphens.
WORD = re.compile(r"(?:[^\W\d_]\.){2,}|\d+(?:[.,]\d+)+|[^\W_][^\W_'\u2019\-]*(?:['\u2019\-][^\W_]+)*")
# No sentence break after these: titles and common abbreviations, a single capital
# initial (J. K. Rowling), or an upper-case dotted abbreviation (U.S. Army).
ABBREV = re.compile(r"(?:\b(?:Mr|Mrs|Ms|Dr|Prof|St|vs|etc|e\.g|i\.e|al|No|Fig|Inc|Ltd|Co)\.|"
                    r"(?<![A-Za-z])[A-Z]\.|(?:[A-Z]\.){2,})$")


def clean(text: str) -> str:
    text = re.sub(r"```.*?```", " ", text, flags=re.S)          # code blocks
    out = []
    for line in text.splitlines():
        s = line.strip()
        if not s or s.startswith("#") or s.startswith("|") or s.startswith(">"):
            out.append("")                                       # headings, tables, quotes
            continue
        if re.fullmatch(r"(\*\*|__)[^*_]+\1:?", s):
            out.append("")                                       # a line that is all bold is a heading
            continue
        s, n_marker = re.subn(r"^(?:[-*+]|\d+[.)])\s+(?:\[[ xX]\]\s+)?", "", s)  # list markers
        s = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", s)           # links -> text
        s = re.sub(r"https?://\S+", " ", s)                      # bare urls
        s = re.sub(r"[*_`]", "", s)
        if n_marker:
            out.extend(["", s, ""])                              # a list item is its own paragraph
        else:
            out.append(s)
    return "\n".join(out)


def sentences(text: str):
    """Split paragraphs into sentences. Conservative: splits on . ! ? followed by space and
    an upper-case letter, digit or opening quote, skipping common abbreviations and initials.
    Single line breaks inside a paragraph are treated as spaces, so hard-wrapped text is safe."""
    sents = []
    for para in re.split(r"\n\s*\n", text):
        para = " ".join(para.split())                            # rejoin hard-wrapped lines
        if not para:
            continue
        start = 0
        for m in re.finditer(r"[.!?][\"'\u201d\u2019)]*\s+(?=[A-Z0-9\"'\u201c\u2018(])", para):
            chunk = para[start:m.end()].strip()
            if ABBREV.search(chunk.rstrip("\"'\u201d\u2019) ")):
                continue
            sents.append(chunk)
            start = m.end()
        tail = para[start:].strip()
        if tail:
            sents.append(tail)
    return sents


def lag1(xs):
    n = len(xs)
    if n < 3:
        return None
    m = sum(xs) / n
    den = sum((x - m) ** 2 for x in xs)
    if den == 0:
        return None
    return sum((xs[i] - m) * (xs[i + 1] - m) for i in range(n - 1)) / den


def measure(text: str):
    text = clean(text)
    sents = sentences(text)
    lens, numbers, names, words_all = [], 0, 0, []
    for s in sents:
        toks = WORD.findall(s)
        if not toks:
            continue
        lens.append(len(toks))
        words_all.extend(t.lower() for t in toks)
        numbers += sum(1 for t in toks if any(c.isdigit() for c in t))
        names += sum(1 for t in toks[1:] if t[0].isupper() and len(t) > 1
                     and not t.startswith(("I'", "I\u2019")))     # I'm, I've, I'd are not names
    words = sum(lens)
    marks = {
        # em dash, spaced en dash, spaced hyphen used as a dash, double hyphen
        "em_dash": len(re.findall(r"\u2014|\s\u2013\s|\s-\s|--", text)),
        "semicolon": text.count(";"),
        # colon not inside a time or ratio such as 10:30
        "colon": len(re.findall(r":(?!\d)", text)),
        "paren": text.count("("),
    }
    return {"lens": lens, "words": words, "numbers": numbers, "names": names,
            "marks": marks, "tokens": words_all}


REF_PATH = Path(__file__).resolve().parent.parent / "reference" / "human_reference.json"
KINDS = {"explanation": "explanation", "argument": "argument",
         "narrative": "narrative non-fiction", "technical": "technical description"}
# Measures used to rank candidates. Numbers and names depend on the subject, so they are
# reported and not ranked on.
RANK_KEYS = ["mean_len", "cv_len", "short_long", "le4_share", "lag1", "comma"]
LABELS = [
    ("mean_len", "mean sentence length, words", 1),
    ("cv_len", "spread of sentence length (sd / mean)", 2),
    ("le4_share", "sentences of 4 words or fewer, share", 3),
    ("short_long", "short sentence then a much longer one, share", 3),
    ("lag1", "neighbouring sentences alike in length (lag-1)", 2),
    ("comma", "commas per 1,000 words", 1),
    ("em_dash", "dashes per 1,000 words", 1),
    ("semicolon", "semicolons per 1,000 words", 1),
    ("colon", "colons per 1,000 words", 1),
    ("paren", "parentheses per 1,000 words", 1),
    ("numbers_per_100", "numbers per 100 words", 1),
    ("names_per_100", "names per 100 words", 1),
]


def profile(text):
    m = measure(text)
    lens, words = m["lens"], m["words"]
    n = len(lens)
    if n < 3 or not words:
        return None
    mean = sum(lens) / n
    var = sum((x - mean) ** 2 for x in lens) / n
    pairs = n - 1
    cleaned = clean(text)
    sents = [s for s in sentences(cleaned) if WORD.findall(s)]
    hooks = [(sents[i], sents[i + 1]) for i in range(pairs)
             if lens[i] <= 8 and lens[i + 1] >= 2.5 * lens[i]]
    k = 1000 / words
    out = {
        "words": words, "sentences": n, "mean_len": mean,
        "cv_len": var ** 0.5 / mean if mean else 0.0,
        "lag1": lag1(lens) or 0.0,
        "short_long": len(hooks) / pairs,
        "le4_share": sum(1 for x in lens if x <= 4) / n,
        "comma": cleaned.count(",") * k,
        "numbers_per_100": m["numbers"] * 100 / words,
        "names_per_100": m["names"] * 100 / words,
        "headings": len(re.findall(r"(?m)^\s*(#{1,6}\s|(\*\*|__)[^*_\n]+\2:?\s*$)", text)),
        "list_items": len(re.findall(r"(?m)^\s*(?:[-*+]|\d+[.)])\s", text)),
        "hooks": hooks,
    }
    for key, c in m["marks"].items():
        out[key] = c * k
    return out


def reference(kind):
    ref = json.loads(REF_PATH.read_text(encoding="utf-8"))
    if kind:
        return ref["by_kind"][KINDS[kind]], "human %s texts" % kind
    return ref["all"], "human texts, all kinds"


def flag(value, pct, dec=3):
    """LOW or HIGH when the value is outside the human range at the precision shown."""
    if round(value, dec) < round(pct["p10"], dec):
        return "LOW"
    if round(value, dec) > round(pct["p90"], dec):
        return "HIGH"
    return ""


def distance(p, ref):
    """Sum of squared distances from the human median, each scaled by the human spread."""
    d = 0.0
    for key in RANK_KEYS:
        pct = ref["percentiles"][key]
        scale = (pct["p90"] - pct["p10"]) / 2.56 or 1.0
        d += ((p[key] - pct["p50"]) / scale) ** 2
    return d


def report(path, p, ref, ref_name):
    print("\n%s: %d words, %d sentences. Reference: %s." % (path, p["words"], p["sentences"], ref_name))
    if p["words"] < 300:
        print("  Under 300 words. The rhythm measures are unreliable at this length.")
    print("  %-48s %8s   %-17s %s" % ("measure", "value", "human 10th-90th", ""))
    flags = []
    for key, label, dec in LABELS:
        pct = ref["percentiles"][key]
        f = flag(p[key], pct, dec)
        if f:
            flags.append((key, f))
        print("  %-48s %8.*f   %-17s %s" % (label, dec, p[key],
              "%.*f to %.*f" % (dec, pct["p10"], dec, pct["p90"]), f))
    if p["headings"] or p["list_items"]:
        print("  Formatting: %d heading lines and %d list items. The reference is running prose,"
              " and short heading lines read as short sentences." % (p["headings"], p["list_items"]))
    if p["hooks"]:
        print("  Short sentence followed by one at least 2.5 times longer, %d found. First few:" % len(p["hooks"]))
        for a, b in p["hooks"][:4]:
            print("    - %s || %s" % (a, b if len(b) <= 110 else b[:107] + "..."))
    return flags


def cmd_check(args):
    ref, ref_name = reference(args.kind)
    out = {}
    for f in args.files:
        p = profile(Path(f).read_text(encoding="utf-8", errors="replace"))
        if p is None:
            print("\n%s: too short to measure." % f)
            continue
        flags = report(f, p, ref, ref_name) if not args.json else \
            [(k, flag(p[k], ref["percentiles"][k], d)) for k, _, d in LABELS if flag(p[k], ref["percentiles"][k], d)]
        out[f] = {"profile": {k: v for k, v in p.items() if k != "hooks"},
                  "flags": dict(flags), "distance": round(distance(p, ref), 2),
                  "hooks": [list(h) for h in p["hooks"]]}
    if args.json:
        print(json.dumps(out, indent=1, ensure_ascii=False))
    return 0


def cmd_rank(args):
    ref, ref_name = reference(args.kind)
    rows = []
    for f in args.files:
        p = profile(Path(f).read_text(encoding="utf-8", errors="replace"))
        if p is None:
            continue
        flags = [k for k in RANK_KEYS if flag(p[k], ref["percentiles"][k])]
        rows.append((distance(p, ref), f, p, flags))
    rows.sort(key=lambda r: r[0])
    if args.json:
        print(json.dumps([{"file": f, "distance": round(d, 2), "outside_range": fl,
                           **{k: round(p[k], 3) for k in RANK_KEYS}} for d, f, p, fl in rows], indent=1))
        return 0
    print("Ranked by closeness to %s on: %s." % (ref_name, ", ".join(RANK_KEYS)))
    print("%-4s %-9s %-7s %-6s %-9s %-7s %-7s %-7s %-22s %s" % (
        "rank", "distance", "length", "spread", "<=4 wds", "hooks", "lag-1", "commas", "outside human range", "file"))
    for i, (d, f, p, fl) in enumerate(rows, 1):
        print("%-4d %-9.2f %-7.1f %-6.2f %-9.3f %-7.3f %-7.2f %-7.1f %-22s %s" % (
            i, d, p["mean_len"], p["cv_len"], p["le4_share"], p["short_long"], p["lag1"], p["comma"],
            ", ".join(fl) or "none", f))
    return 0


def norm_number(tok):
    return tok.replace(",", "").rstrip(".").lower()


def cmd_specifics(args):
    material = " ".join(Path(f).read_text(encoding="utf-8", errors="replace") for f in args.material)
    mat_words = {w.lower() for w in WORD.findall(material)}
    mat_numbers = {norm_number(w) for w in WORD.findall(material) if any(c.isdigit() for c in w)}
    numbers, names = [], []
    for s in sentences(clean(Path(args.draft).read_text(encoding="utf-8", errors="replace"))):
        toks = WORD.findall(s)
        for i, t in enumerate(toks):
            if any(c.isdigit() for c in t):
                if norm_number(t) not in mat_numbers:
                    numbers.append((t, s))
            elif i > 0 and t[0].isupper() and len(t) > 1 and not t.startswith(("I'", "I\u2019")):
                base = re.sub(r"['’]s$", "", t.lower())
                if base not in mat_words and base.rstrip("s") not in mat_words:
                    names.append((t, s))
    if args.json:
        print(json.dumps({"numbers": numbers, "names": names}, indent=1, ensure_ascii=False))
        return 0
    print("Numbers in the draft that are not in the material: %d" % len(numbers))
    for t, s in numbers:
        print("  %-12s %s" % (t, s if len(s) <= 140 else s[:137] + "..."))
    print("Capitalised names in the draft that are not in the material: %d" % len(names))
    for t, s in names:
        print("  %-12s %s" % (t, s if len(s) <= 140 else s[:137] + "..."))
    print("A listed item is not proof of an error. Spelled-out numbers and reworded facts are not checked.")
    return 0


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    for name, fn in (("check", cmd_check), ("rank", cmd_rank)):
        sp = sub.add_parser(name)
        sp.add_argument("files", nargs="+")
        sp.add_argument("--kind", choices=sorted(KINDS))
        sp.add_argument("--json", action="store_true")
        sp.set_defaults(fn=fn)
    sp = sub.add_parser("specifics")
    sp.add_argument("draft")
    sp.add_argument("--material", nargs="+", required=True)
    sp.add_argument("--json", action="store_true")
    sp.set_defaults(fn=cmd_specifics)
    args = ap.parse_args(argv)
    return args.fn(args)


if __name__ == "__main__":
    sys.exit(main())
