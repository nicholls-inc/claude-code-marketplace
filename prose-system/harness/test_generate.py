#!/usr/bin/env python3
"""Checks for generate.py with a stub client. No API call is made and the stub's
text is a marker string, not a prose sample. Run: python3 test_generate.py
"""
import importlib.util
import json
import sys
import tempfile
from pathlib import Path
from types import SimpleNamespace

spec = importlib.util.spec_from_file_location("generate", Path(__file__).with_name("generate.py"))
g = importlib.util.module_from_spec(spec)
spec.loader.exec_module(g)


class StubMessages:
    def __init__(self):
        self.requests = []

    def create(self, **req):
        self.requests.append(req)
        n = len(self.requests)
        stop = "refusal" if n == 2 else "end_turn"
        return SimpleNamespace(
            content=[SimpleNamespace(type="thinking", thinking=""),
                     SimpleNamespace(type="text", text="STUB OUTPUT %d" % n)],
            model=req["model"], stop_reason=stop, _request_id="req_stub_%d" % n,
            usage=SimpleNamespace(to_dict=lambda: {"input_tokens": 1, "output_tokens": 1}))


def check(cond, label, fails):
    if not cond:
        fails.append(label)


def main():
    fails = []
    with tempfile.TemporaryDirectory() as d:
        d = Path(d)
        briefs = {"set": "dev", "version": "t", "frozen": True,
                  "briefs": [{"id": "b1", "prompt": "PROMPT ONE"}, {"id": "b2", "prompt": "PROMPT TWO"}]}
        conds = {"version": "t", "conditions": {"baseline": {"template": "{prompt}"},
                                                "extra": {"template": "{prompt}\n\nEXTRA"}}}
        (d / "b.json").write_text(json.dumps(briefs))
        (d / "c.json").write_text(json.dumps(conds))
        base = ["--briefs", str(d / "b.json"), "--conditions", str(d / "c.json"),
                "--model", "stub-model", "--out", str(d / "out"), "--workers", "1"]

        stub = SimpleNamespace(messages=StubMessages())
        g.main(base + ["--condition", "baseline", "--condition", "extra", "--n", "2"], client=stub)
        reqs = stub.messages.requests
        check(len(reqs) == 8, "8 calls for 2 conditions x 2 briefs x 2 samples", fails)
        check(all(set(r) == {"model", "max_tokens", "messages"} for r in reqs),
              "request holds model, max_tokens and messages only", fails)
        check(all(len(r["messages"]) == 1 and r["messages"][0]["role"] == "user" for r in reqs),
              "one user message per call", fails)
        check(reqs[0]["messages"][0]["content"] == "PROMPT ONE", "baseline sends the bare prompt", fails)
        check(any(r["messages"][0]["content"] == "PROMPT TWO\n\nEXTRA" for r in reqs),
              "condition template applied", fails)

        stored = sorted((d / "out").rglob("*.json"))
        check(len(stored) == 8, "8 JSON records stored", fails)
        recs = [json.loads(p.read_text()) for p in stored]
        need = {"condition", "brief_id", "index", "model_requested", "model_served", "request",
                "stop_reason", "usage", "timestamp_utc", "valid", "text", "prompt_sha256"}
        check(all(need <= set(r) for r in recs), "record carries the required fields", fails)
        check(sum(1 for r in recs if not r["valid"]) == 1, "the refusal is stored and marked not valid", fails)
        check(len(list((d / "out").rglob("*.txt"))) == 7, "text files only for valid samples", fails)

        stub2 = SimpleNamespace(messages=StubMessages())
        g.main(base + ["--condition", "baseline", "--condition", "extra", "--n", "3"], client=stub2)
        check(len(stub2.messages.requests) == 4, "a rerun generates only the missing samples", fails)
        check(json.loads(stored[0].read_text()) == recs[0], "stored samples are not rewritten", fails)

        stub3 = SimpleNamespace(messages=StubMessages())
        g.main(base + ["--condition", "baseline", "--n", "5", "--max-calls", "1"], client=stub3)
        check(len(stub3.messages.requests) == 1, "--max-calls caps the run", fails)

        for change, flag, label in (({"frozen": False}, None, "an unfrozen set is refused"),
                                    ({"set": "held-out"}, None, "the held-out set is refused without the flag")):
            (d / "b2.json").write_text(json.dumps({**briefs, **change}))
            stub4 = SimpleNamespace(messages=StubMessages())
            try:
                g.main(["--briefs", str(d / "b2.json")] + base[2:] + ["--condition", "baseline"], client=stub4)
                check(False, label, fails)
            except SystemExit as e:
                check(isinstance(e.code, str) and not stub4.messages.requests, label, fails)

    for f in fails:
        print("FAIL", f)
    print("%d failure(s)" % len(fails))
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
