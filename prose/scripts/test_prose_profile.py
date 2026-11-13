#!/usr/bin/env python3
"""Checks for prose_profile.py on constructed text. Run: python3 test_prose_profile.py"""
import importlib.util
import io
import sys
import tempfile
from contextlib import redirect_stdout
from pathlib import Path

spec = importlib.util.spec_from_file_location("prose_profile", Path(__file__).with_name("prose_profile.py"))
P = importlib.util.module_from_spec(spec)
spec.loader.exec_module(P)

HOOKY = " ".join(["It broke. " + "The pump on the corner of the street had been drawing water from a well that sat "
                  "a few feet from a leaking cesspit for most of that summer." for _ in range(12)])
EVEN = " ".join(["The pump on the corner drew its water from a shallow well near the old cesspit. "
                 "Most households on the street fetched their water there because it tasted cleaner than the rest. "
                 "Deaths in the district rose within three days of the first case, and they fell after the handle was removed."
                 for _ in range(8)])


def run(argv):
    buf = io.StringIO()
    with redirect_stdout(buf):
        P.main(argv)
    return buf.getvalue()


def main():
    fails = []

    def check(cond, label):
        if not cond:
            fails.append(label)

    ph, pe = P.profile(HOOKY), P.profile(EVEN)
    check(ph["short_long"] > 0.4 and len(ph["hooks"]) >= 10, "short-then-long pairs are found")
    check(pe["short_long"] == 0 and not pe["hooks"], "even text has no short-then-long pairs")
    check(ph["lag1"] < -0.5, "alternating text has a negative lag-1")
    check(P.profile("Too short.") is None, "very short text is not measured")
    ref, _ = P.reference("narrative")
    check(P.distance(pe, ref) < P.distance(ph, ref), "even text ranks closer to the reference than alternating text")
    check(P.flag(0.0, {"p10": 0.0004, "p90": 6.0}, 1) == "", "a value equal to the range edge at shown precision is not flagged")
    check(P.flag(30.0, {"p10": 15.0, "p90": 26.8}, 1) == "HIGH", "a value above the range is flagged HIGH")
    for kind in P.KINDS:
        check(set(P.RANK_KEYS) <= set(P.reference(kind)[0]["percentiles"]), "reference has rank keys for " + kind)

    with tempfile.TemporaryDirectory() as d:
        d = Path(d)
        (d / "a.txt").write_text(HOOKY)
        (d / "b.txt").write_text(EVEN)
        out = run(["rank", str(d / "a.txt"), str(d / "b.txt"), "--kind", "narrative"])
        lines = [l for l in out.splitlines() if l[:1].isdigit()]
        check(lines and lines[0].rstrip().endswith("b.txt"), "rank puts the even text first")
        check("Short sentence followed" in run(["check", str(d / "a.txt")]), "check lists the pairs")
        out = run(["rank", str(d / "a.txt"), str(d / "missing.txt"), "--kind", "narrative"])
        check("a.txt" in out, "rank skips a missing candidate file")
        try:
            run(["rank", str(d / "missing.txt")])
            check(False, "rank stops when no candidate can be measured")
        except SystemExit as e:
            check(isinstance(e.code, str), "rank stops when no candidate can be measured")
        (d / "m.txt").write_text("John Snow mapped 616 deaths around the Broad Street pump in 1854.")
        (d / "draft.txt").write_text("In 1854 John Snow mapped 616 deaths. He told Henry Whitehead that 73 houses were affected.")
        out = run(["specifics", str(d / "draft.txt"), "--material", str(d / "m.txt")])
        check("73" in out and "Whitehead" in out, "specifics lists a number and a name missing from the material")
        check("616 " not in out.split("Capitalised")[0].split("material: ")[1][:3] and "Snow " not in out.split("Capitalised")[1][:40],
              "specifics does not list items that are in the material")

    for f in fails:
        print("FAIL", f)
    print("%d failure(s)" % len(fails))
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
