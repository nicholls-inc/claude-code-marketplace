#!/usr/bin/env python3
"""Surface profile of prose samples. Standard library only.

Usage:
    python profile.py sample.txt                 # one file
    python profile.py samples/cond_a/*.txt       # several files: per-file rows plus a pooled row
    python profile.py samples/cond_a/*.txt --json out.json
    python profile.py samples/cond_a/*.txt --ngrams 4 --min-files 3

What it reports
    words, sentences
    mean, sd and coefficient of variation of sentence length in words
    lag1: lag-1 autocorrelation of sentence length (pooled row concatenates files in the order given)
    short_long: share of sentence transitions where a sentence of <= 8 words is followed
                by one at least 2.5 times longer
    per 1,000 words: em_dash, semicolon, colon, paren, and joiners (their sum)
    per 100 words: numbers (tokens containing a digit) and names (capitalised words not at sentence start)
    repeated n-grams that occur in at least --min-files files

This is an alarm, not a quality score. Rhythm measures are unreliable on a single short
document. Pool at least several thousand words before reading lag1 or short_long.
Markdown headings, tables, list markers and code blocks are stripped before counting.
"""
import argparse
import json
import math
import re
import sys
from collections import Counter, defaultdict
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


def summarise(lens, words, numbers, names, marks):
    n = len(lens)
    if n == 0 or words == 0:
        return {"words": 0, "sentences": 0}
    mean = sum(lens) / n
    sd = math.sqrt(sum((x - mean) ** 2 for x in lens) / n)
    trans = max(n - 1, 1)
    sl = sum(1 for i in range(n - 1) if lens[i] <= 8 and lens[i + 1] >= 2.5 * lens[i]) / trans
    k = 1000 / words
    ac = lag1(lens)
    row = {
        "words": words, "sentences": n,
        "mean_len": round(mean, 1), "sd_len": round(sd, 1), "cv_len": round(sd / mean, 2),
        "lag1": None if ac is None else round(ac, 2),
        "short_long": round(sl, 3),
    }
    for name, c in marks.items():
        row[name] = round(c * k, 1)
    row["joiners"] = round(sum(marks.values()) * k, 1)
    row["numbers_per_100"] = round(numbers * 100 / words, 1)
    row["names_per_100"] = round(names * 100 / words, 1)
    return row


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("files", nargs="+")
    ap.add_argument("--json", help="write full results to this path")
    ap.add_argument("--ngrams", type=int, default=4, help="n-gram size for repetition across files")
    ap.add_argument("--min-files", type=int, default=3, help="report n-grams found in at least this many files")
    args = ap.parse_args()

    rows, pooled_lens = {}, []
    tot = {"words": 0, "numbers": 0, "names": 0}
    tot_marks = Counter()
    gram_files = defaultdict(set)
    for f in args.files:
        m = measure(Path(f).read_text(encoding="utf-8", errors="replace"))
        rows[f] = summarise(m["lens"], m["words"], m["numbers"], m["names"], m["marks"])
        pooled_lens.extend(m["lens"])
        for key in tot:
            tot[key] += m[key]
        tot_marks.update(m["marks"])
        t = m["tokens"]
        for i in range(len(t) - args.ngrams + 1):
            gram_files[" ".join(t[i:i + args.ngrams])].add(f)
    pooled = summarise(pooled_lens, tot["words"], tot["numbers"], tot["names"],
                       {k: tot_marks[k] for k in ("em_dash", "semicolon", "colon", "paren")})
    repeated = sorted(((len(v), g) for g, v in gram_files.items() if len(v) >= args.min_files), reverse=True)

    cols = ["words", "sentences", "mean_len", "cv_len", "lag1", "short_long",
            "em_dash", "semicolon", "colon", "paren", "joiners", "numbers_per_100", "names_per_100"]
    print("%-28s" % "file" + "".join("%11s" % c[:10] for c in cols))
    for f, r in rows.items():
        print("%-28s" % Path(f).name[:27] + "".join("%11s" % r.get(c, "") for c in cols))
    if len(rows) > 1:
        print("%-28s" % "POOLED" + "".join("%11s" % pooled.get(c, "") for c in cols))
        if repeated:
            print("\n%d-grams in >= %d files (top 20):" % (args.ngrams, args.min_files))
            for count, g in repeated[:20]:
                print("  %3d  %s" % (count, g))
    if args.json:
        Path(args.json).write_text(json.dumps(
            {"files": rows, "pooled": pooled,
             "repeated_ngrams": [{"files": c, "ngram": g} for c, g in repeated[:200]]}, indent=2))


if __name__ == "__main__":
    sys.exit(main())
