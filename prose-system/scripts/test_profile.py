#!/usr/bin/env python3
"""Checks for profile.py on constructed cases. Run: python3 test_profile.py

These cases are written to exercise the splitter and the counters. They are not
prose samples and are not evidence about any condition.
"""
import importlib.util
import sys
from pathlib import Path

spec = importlib.util.spec_from_file_location("prose_profile", Path(__file__).with_name("profile.py"))
p = importlib.util.module_from_spec(spec)
spec.loader.exec_module(p)

CASES = [
    # (name, text, expected sentence lengths in words)
    ("plain", "The pump was shut. Deaths fell within a week.", [4, 5]),
    ("decimal", "The rate rose to 3.5 per cent. It fell later.", [7, 3]),
    ("abbrev title", "Dr. Snow drew a map. Mr. Whitehead checked it.", [5, 4]),
    ("initials", "J. K. Rowling wrote it. T. S. Eliot did not.", [5, 5]),
    ("acronym", "The U.S. Army built it. The U.K. Treasury paid.", [5, 4]),
    ("eg", "Use a joiner, e.g. A colon. Then stop.", [6, 2]),
    ("time", "It left at 9 a.m. on Tuesday. It arrived at 4 p.m. on Friday.", [7, 7]),
    ("quote end", "She said \"stop.\" He did not.", [3, 3]),
    ("question", "Why does it slow? The cell is nearly full.", [4, 5]),
    ("hard wrap", "The tank burst at half past\ntwelve. The wave reached\nthe harbour.", [7, 5]),
    ("list items", "- First item here\n- Second item here", [3, 3]),
    ("heading dropped", "# Title\n\nOne sentence only.", [3]),
    ("ellipsis", "He waited... Nothing came.", [2, 2]),
    ("bold heading dropped", "**The cycle**\n\nOne sentence only.", [3]),
    ("bold lead-in kept", "**Insulation.** A leaky house needs more heat.", [1, 6]),
]

COUNTS = [
    # (name, text, key, expected raw count)
    ("em dash", "It was late—too late.", "em_dash", 1),
    ("spaced en dash", "It was late – too late.", "em_dash", 1),
    ("hyphenated word", "A well-known fact.", "em_dash", 0),
    ("number range", "Pages 10-12 and 1990–1995.", "em_dash", 0),
    ("colon in time", "At 10:30 it stopped.", "colon", 0),
    ("colon", "One thing: it stopped.", "colon", 1),
]

NAMES = [
    # (name, text, expected names, expected numbers)
    ("pronoun I", "Then I'm sure I've seen what I'd feared.", 0, 0),
    ("real names", "Then John Snow went to Broad Street.", 4, 0),
    ("numbers", "In 1854 about 616 people died.", 0, 2),
]


def main():
    bad = 0
    for name, text, want in CASES:
        got = [len(p.WORD.findall(s)) for s in p.sentences(p.clean(text))]
        if got != want:
            bad += 1
            print("FAIL split  %-16s want %s got %s" % (name, want, got))
    for name, text, key, want in COUNTS:
        got = p.measure(text)["marks"][key]
        if got != want:
            bad += 1
            print("FAIL mark   %-16s want %s got %s" % (name, want, got))
    for name, text, wn, wd in NAMES:
        m = p.measure(text)
        if (m["names"], m["numbers"]) != (wn, wd):
            bad += 1
            print("FAIL count  %-16s want %s got %s" % (name, (wn, wd), (m["names"], m["numbers"])))
    total = len(CASES) + len(COUNTS) + len(NAMES)
    print("%d of %d checks passed" % (total - bad, total))
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
