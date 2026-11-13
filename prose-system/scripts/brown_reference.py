#!/usr/bin/env python3
"""Human reference profile on the Brown corpus (non-fiction genres), as run on 11 November 2026.

Needs: pip install nltk numpy. Downloads the Brown corpus on first run.
Writes reference/brown_reference.json when run from the package root.

Caveats
- Brown is American English published in 1961. Use it as a reference for edited human
  prose, not for current web writing.
- This script counts on Brown's own tokenisation and sentence boundaries. scripts/profile.py
  works on raw text with its own splitter, so the two are not exactly comparable. Build a
  current reference with profile.py on raw human text before using the profile as an alarm.
- Brown doubles each semicolon token (";" ";"). The count below treats a doubled pair as one.
- Dashes in Brown are the token "--".
"""
import json
import statistics as st
from pathlib import Path

import nltk
import numpy as np

nltk.download("brown", quiet=True)
from nltk.corpus import brown  # noqa: E402

GENRES = ["news", "editorial", "reviews", "government", "learned", "hobbies", "lore", "belles_lettres"]


def metrics(sents):
    lens, words, dash, semi, colon, paren = [], 0, 0, 0, 0, 0
    for s in sents:
        w = [t for t in s if any(c.isalnum() for c in t)]
        if not w:
            continue
        lens.append(len(w))
        words += len(w)
        dash += sum(1 for t in s if t in ("--", "\u2014"))
        i = 0
        while i < len(s):
            if s[i] == ";":
                semi += 1
                if i + 1 < len(s) and s[i + 1] == ";":
                    i += 1
            i += 1
        colon += sum(1 for t in s if t == ":")
        paren += sum(1 for t in s if t == "(")
    n = len(lens)
    m = st.mean(lens)
    sd = st.pstdev(lens)
    num = sum((lens[i] - m) * (lens[i + 1] - m) for i in range(n - 1))
    den = sum((x - m) ** 2 for x in lens)
    k = 1000 / words
    return {
        "sentences": n, "words": words,
        "mean_len": round(m, 1), "cv_len": round(sd / m, 2), "lag1": round(num / den, 2),
        "dash": round(dash * k, 1), "semicolon": round(semi * k, 1),
        "colon": round(colon * k, 1), "paren": round(paren * k, 1),
        "joiners": round((dash + semi + colon + paren) * k, 1),
    }


def main():
    out = {"by_genre": {g: metrics(brown.sents(categories=g)) for g in GENRES}}
    out["pooled"] = metrics(brown.sents(categories=GENRES))
    docs = [metrics(brown.sents(fileids=f)) for f in brown.fileids(categories=GENRES)]
    pct = lambda key: [round(float(x), 2) for x in np.percentile([d[key] for d in docs], [10, 50, 90])]
    out["per_document"] = {
        "documents": len(docs),
        "dash_p10_p50_p90": pct("dash"),
        "joiners_p10_p50_p90": pct("joiners"),
        "lag1_p10_p50_p90": pct("lag1"),
        "documents_with_no_dash": sum(1 for d in docs if d["dash"] == 0),
        "documents_with_negative_lag1": sum(1 for d in docs if d["lag1"] < 0),
    }
    Path("reference").mkdir(exist_ok=True)
    Path("reference/brown_reference.json").write_text(json.dumps(out, indent=2))
    print(json.dumps(out["pooled"], indent=2))
    print(json.dumps(out["per_document"], indent=2))


if __name__ == "__main__":
    main()
