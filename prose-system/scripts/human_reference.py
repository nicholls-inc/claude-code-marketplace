#!/usr/bin/env python3
"""Reference profile of human-written prose, measured by the same code as the samples.
Standard library only.

Usage:
    python3 scripts/human_reference.py                      # fetch (or reuse cache), measure, write
    python3 scripts/human_reference.py --offline            # cache only, no network
    python3 scripts/human_reference.py --dump URL           # print the extracted text of one source

Reads reference/human_sources.json, fetches each URL, extracts the body paragraphs as
plain text, measures each document with features() and pooled() from analyse.py, and
writes reference/human_reference.json.

The texts are copyrighted. Fetched pages and extracted text go to the cache directory,
which must be outside the repository. Only figures, URLs and titles are written here.

Extraction
    Each host has a rule in RULES: the element that holds the article body, and what to
    skip inside it. Only paragraph text is kept. Headings, lists, tables, block quotes,
    code blocks, figures, captions, footnote markers, navigation and comments are dropped.
    A host without a rule is refused rather than guessed at.
    A source may name a "start_heading"; the text before that heading is then left out
    (used for long articles whose opening sections are background).
    Documents longer than --max-words are truncated at the last paragraph boundary that
    keeps them within the limit. Documents that come out shorter than --min-words are
    dropped and listed under "dropped" in the output.

Measures
    heading_share and list_share are near zero by construction, because extraction removes
    headings and lists (a paragraph that opens with "1." is still read as a list item).
    Do not compare them with the samples.
    Percentiles are across documents, each document measured alone with pooled([doc]).
    pooled() takes lag1 about the mean of the whole pool, so documents with different
    average sentence lengths raise the pooled lag1. The per-document median is the
    figure to set beside a single sample.
"""
import argparse
import hashlib
import importlib.util
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from html.parser import HTMLParser
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent
DEFAULT_CACHE = ("/tmp/claude-0/-home-user-claude-code-marketplace/"
                 "62eaf44b-bbc2-5494-b187-a6b289ca02d8/scratchpad/human_cache/")

spec = importlib.util.spec_from_file_location("prose_analyse", HERE / "analyse.py")
A = importlib.util.module_from_spec(spec)
spec.loader.exec_module(A)
features, pooled = A.features, A.pooled

USER_AGENT = "prose-profile-research/0.1"
# Wikimedia asks for a descriptive agent and rate-limits generic ones.
WIKI_AGENT = "prose-profile-research/0.1 (prose surface statistics; python-urllib; cached, 1 request per second)"

VOID = {"br", "img", "hr", "meta", "link", "input", "source", "wbr", "col", "area", "base", "embed", "track"}
# Never body prose, whatever the host.
SKIP_TAGS = {"script", "style", "noscript", "pre", "table", "figure", "figcaption", "blockquote",
             "nav", "header", "footer", "aside", "form", "sup", "ul", "ol", "dl", "svg", "math",
             "button", "select", "textarea", "iframe", "h1", "h2", "h3", "h4", "h5", "h6",
             "audio", "video", "picture", "label", "details", "template"}
# Skipped without ending the paragraph they sit in.
INLINE = {"sup", "sub", "svg", "math", "button", "label", "select", "span", "a", "img", "code", "em", "i", "b", "strong", "small"}
# A paragraph ends when one of these opens or closes.
BLOCK = {"p", "div", "section", "article", "main", "td", "tr", "li", "center", "body", "html"} | (SKIP_TAGS - INLINE)
# Used to notice a skipped element that the page never closes (an unclosed <ul> in a table cell).
STRUCT = {"div", "td", "tr", "table", "section", "article", "main"}
CHARS = {"\u00a0": " ", "\u200b": "", "\u00ad": "", "\u2060": "",
         "\ufb00": "ff", "\ufb01": "fi", "\ufb02": "fl", "\ufb03": "ffi", "\ufb04": "ffl"}
WIKI_END = r"^(See also|Notes|References|Footnotes|Citations|Sources|Bibliography|Further reading|External links|Notes and references|Explanatory notes|Works cited)\b"

# container: (tag, attribute, value); the value matches a whole id or one class token.
# skip_class: skip any element with a class token or id containing one of these strings.
# mode "p": keep text inside <p>. mode "br": the body has no <p>; paragraphs are split on <br><br>.
# stop_heading: stop at the first heading whose text matches.
# drop_para: drop paragraphs matching this pattern.
# allow_tags: tags in SKIP_TAGS that this host uses for layout and must not be skipped.
# strip_re: removed from the HTML before parsing (inline footnote markers and the like).
# start_after, end_before: cut the HTML to the part between these patterns.
# drop_img_paras: drop a <p> that contains an <img> (hosts that put captions in the same <p>).
# drop_labels: drop paragraphs of six words or fewer with no closing punctuation
#   (headings and labels on hosts that do not mark them up as headings).
MONTHS = "January|February|March|April|May|June|July|August|September|October|November|December"
RULES = {
    "git-scm.com": {"container": ("div", "id", "main"),
                    "skip_class": ["admonitionblock", "imageblock", "listingblock", "sidebarblock", "title"]},
    "hpbn.co": {"container": ("article", None, None), "skip_class": ["note", "example", "sidebar", "warning"]},
    "aosabook.org": {"container": ("body", None, None),
                     "skip_class": ["titlebox", "banner", "figure", "footnote", "author"],
                     "drop_para": r"^(Table|Figure) \d+\.\d+:", "drop_labels": True,
                     "stop_heading": r"^(Footnotes|References|Acknowledge?ments)"},
    "craftinginterpreters.com": {"container": ("article", "class", "chapter"),
                                 "skip_class": ["codehilite", "source-file", "prev-next", "challenges"],
                                 "stop_heading": r"^Challenges"},
    "danluu.com": {"container": ("main", None, None), "skip_class": ["footnote"],
                   "stop_heading": r"^(Appendix|Update|Acknowledge?ments|Thanks)", "drop_para": r"^Thanks to "},
    "paulgraham.com": {"container": ("font", "face", "verdana"), "mode": "br",
                       "strip_re": r"\[<a href=\"#f\d+n\">.*?</a>\]|\[\d{1,2}\]",
                       "end_before": r"<b>Notes?</b>|<b>Thanks</b>",
                       "drop_para": r"^(%s) \d{4}\b.{0,40}$" % MONTHS, "drop_labels": True},
    "joelonsoftware.com": {"container": ("div", "class", "entry-content"), "skip_class": ["sharedaddy", "jp-relatedposts"],
                           "drop_para": r"^(Further reading|Read more about)"},
    "idlewords.com": {"container": ("div", "id", "main"), "allow_tags": ["table"], "drop_labels": True,
                      "drop_para": r"^(Idle Words >|This is (the|a) (text|written) version of)"},
    "martinfowler.com": {"container": ("div", "class", "paperBody"), "skip_class": ["footnote", "foot-ref", "figure", "photo", "soundbite"]},
    "damninteresting.com": {"container": ("div", "class", "main-content"),
                            "skip_class": ["print-only", "exclude-from-print", "credits", "comment", "thumb",
                                           "narrate", "content-end", "caption", "footnote"],
                            "drop_para": r"^(An earlier version of this article|Update[: ]|Correction[: ]|Please (warmly )?(welcome|provide)|Editor.s Note|This story is assembled from)"},
    "theconversation.com": {"container": ("div", "itemprop", "articleBody"),
                            "skip_class": ["caption", "related", "promo", "newsletter", "listen"],
                            "drop_para": r"^(Read more:|Read:|\[.{0,200}\]$|This article (is part of|was|has been)|Listen to|Sign up|This is an? (edited|updated)|A version of this article|Find out more about|What element of science or technology|Correction:|Editor.s note:|Further reading:?$)",
                            "drop_labels": True},
    "news.mit.edu": {"container": ("div", "class", "news-article--content--body--inner"), "mode": "br",
                     "skip_class": ["caption", "related"], "allow_tags": ["sup"],
                     "drop_para": r"^Science and technology journalists pride themselves"},
    "nasa.gov": {"container": ("div", "class", "entry-content"), "skip_class": ["caption", "hds-", "credit"], "drop_img_paras": True,
                 "drop_para": r"^((Left|Right|Middle|Center|Top|Bottom)\b[^.]{0,30}:|Image credit|Credits?:|Footnote:|Originally Published|By [A-Z][a-z]+ [A-Z][a-z]+\b.{0,90}$|[A-Z][a-z]+ [A-Z][a-z]+ NASA.s .{0,80}$|To be continued|News from around the world|Explore more about|For more (information|insight))"},
    "publicdomainreview.org": {"container": ("div", "class", "essay-body"), "skip_class": ["essay__image", "caption", "footnote"]},
    "jvns.ca": {"container": ("article", "class", "hentry"), "skip_class": ["post-tags", "footnote"],
                "drop_para": r"^Thanks to "},
    "en.wikipedia.org": {"container": ("div", "class", "wiki-body"),
                         "skip_class": ["reference", "hatnote", "thumb", "navbox", "infobox", "metadata", "noprint",
                                        "mw-editsection", "reflist", "refbegin", "gallery", "quotebox", "shortdescription",
                                        "IPA", "mw-empty-elt", "sidebar", "templatequote", "ambox", "citation"],
                         "stop_heading": WIKI_END},
}


class Body(HTMLParser):
    def __init__(self, rule, start_heading=None):
        super().__init__(convert_charrefs=True)
        self.rule = rule
        self.waiting = start_heading       # keep nothing until a heading with exactly this text
        self.mode = rule.get("mode", "p")
        self.skip_class = rule.get("skip_class", [])
        self.skip_tags = SKIP_TAGS - set(rule.get("allow_tags", []))
        self.stop_re = re.compile(rule["stop_heading"], re.I) if rule.get("stop_heading") else None
        self.in_container = 0          # nesting depth of the container's tag once inside it
        self.ctag = None
        self.skip_tag, self.skip_depth = None, 0
        self.in_p = False
        self.p_img = False
        self.wrap = 0                  # depth inside <b>, <strong>, <code>, <tt>
        self.plain = False             # the paragraph has text outside those
        self.brs = 0
        self.buf, self.paras = [], []
        self.title, self.in_title = "", False
        self.heading, self.in_heading = [], False
        self.stopped = False
        self.seen_container = False

    def _is_container(self, tag, attrs):
        ctag, cattr, cval = self.rule["container"]
        if tag != ctag:
            return False
        if cattr is None:
            return True
        v = dict(attrs).get(cattr) or ""
        return v == cval if cattr == "id" else cval in v.split()

    def flush(self):
        text = " ".join("".join(self.buf).split())
        self.buf = []
        if self.p_img and self.rule.get("drop_img_paras"):
            text = ""                               # a paragraph that holds an image is its caption
        if not self.plain and len(text.split()) <= 15:
            text = ""                               # all bold is a heading; all code is a listing
        self.p_img = self.plain = False
        if text:
            self.paras.append(text)

    def handle_starttag(self, tag, attrs):
        if tag == "title" and not self.title:
            self.in_title = True
        if self.stopped:
            return
        if not self.in_container:
            if self._is_container(tag, attrs) and not (self.seen_container and self.rule.get("first_only", True)):
                self.in_container, self.ctag, self.seen_container = 1, tag, True
            return
        if tag in VOID:
            if tag == "img" and self.in_p and not self.skip_tag:
                self.p_img = True
            if tag == "br" and not self.skip_tag:
                self.brs += 1
                if self.mode == "br" and self.brs >= 2:
                    self.flush()
                elif self.buf:
                    self.buf.append(" ")
            return
        if tag == self.ctag:
            self.in_container += 1
        if self.skip_tag:
            if tag == self.skip_tag:
                self.skip_depth += 1
            elif tag in STRUCT:
                self.skip_inner[tag] = self.skip_inner.get(tag, 0) + 1
            return
        d = dict(attrs)
        marks = " ".join(d.get(k) or "" for k in ("class", "id", "data-type", "role"))
        if tag in self.skip_tags or any(s in marks for s in self.skip_class):
            if tag not in INLINE:
                self.flush()
                self.in_p = False
            self.skip_tag, self.skip_depth, self.skip_inner = tag, 1, {}
            if tag in ("h1", "h2", "h3", "h4", "h5", "h6"):
                self.in_heading, self.heading = True, []
            return
        if tag in ("b", "strong", "code", "tt"):
            self.wrap += 1
        if tag in BLOCK:
            self.flush()
            self.wrap = 0
            self.in_p = tag == "p"

    def handle_startendtag(self, tag, attrs):
        if tag in VOID:
            self.handle_starttag(tag, attrs)

    def handle_endtag(self, tag):
        if tag == "title":
            self.in_title = False
        if self.stopped or not self.in_container or tag in VOID:
            return
        if self.skip_tag:
            tag_closed = tag
            if tag in STRUCT and tag != self.skip_tag:
                if self.skip_inner.get(tag, 0) > 0:
                    self.skip_inner[tag] -= 1
                else:                               # an ancestor closed: the skipped element was never closed
                    self.skip_depth, tag_closed = 1, self.skip_tag
            if tag_closed == self.skip_tag:
                self.skip_depth -= 1
                if self.skip_depth == 0:
                    self.skip_tag = None
                    if self.in_heading:
                        self.in_heading = False
                        if self.waiting and re.sub(r"\s*\[edit\]$", "", " ".join("".join(self.heading).split())) == self.waiting:
                            self.waiting = None
                        elif self.stop_re and self.stop_re.search(" ".join("".join(self.heading).split())):
                            self.stopped = True
        elif tag in BLOCK:
            self.flush()
            self.in_p = False
        elif tag in ("b", "strong", "code", "tt") and self.wrap:
            self.wrap -= 1
        if tag == self.ctag:
            self.in_container -= 1
            if self.in_container == 0:
                self.flush()

    def handle_data(self, data):
        if self.in_title:
            self.title += data
        if self.stopped or not self.in_container:
            return
        if self.in_heading:
            self.heading.append(data)
        if self.skip_tag:
            return
        if self.waiting:
            return
        if self.mode == "br" or self.in_p:
            if data.strip():
                self.brs = 0
                if not self.wrap and re.search(r"\w", data):
                    self.plain = True
            self.buf.append(data)


def rule_for(url):
    host = urllib.parse.urlsplit(url).netloc.lower()
    host = host[4:] if host.startswith("www.") else host
    if host not in RULES:
        raise KeyError("no extraction rule for host %s" % host)
    return host, RULES[host]


def extract(html, url, start_heading=None):
    """Return (title, [paragraphs]). With start_heading, the text before that heading is left out."""
    _, rule = rule_for(url)
    if rule.get("strip_re"):
        html = re.sub(rule["strip_re"], "", html, flags=re.S)
    if rule.get("start_after"):
        m = re.search(rule["start_after"], html, re.S | re.I)
        if m:
            head = re.search(r"<title[^>]*>.*?</title>", html, re.S | re.I)
            html = (head.group(0) if head else "") + html[m.end():]
    if rule.get("end_before"):
        m = re.search(rule["end_before"], html, re.S | re.I)
        if m:
            html = html[:m.start()]
    p = Body(rule, start_heading)
    p.feed(html)
    p.close()
    p.flush()
    drop = re.compile(rule["drop_para"], re.I) if rule.get("drop_para") else None
    paras = []
    for t in p.paras:
        for junk, plain in CHARS.items():           # no-break and zero-width spaces, soft hyphens, fi and fl ligatures
            t = t.replace(junk, plain)
        t = " ".join(t.split())
        if not re.search(r"[A-Za-z]", t) or (drop and drop.search(t)):
            continue
        closed = re.search(r"[.!?:;,][\"'\u201d\u2019)]*$", t)
        if not closed and len(t.split()) <= (6 if rule.get("drop_labels") else 3):
            continue                                # a label, or debris between two removed code blocks
        paras.append(t)
    title = " ".join(p.title.split())
    return title, paras


def truncate(paras, max_words):
    out, n = [], 0
    for t in paras:
        w = len(t.split())
        if out and n + w > max_words:
            return out, True
        out.append(t)
        n += w
    return out, False


class Fetcher:
    def __init__(self, cache, delay=1.0, offline=False, max_fetches=250):
        self.cache = Path(cache)
        (self.cache / "raw").mkdir(parents=True, exist_ok=True)
        (self.cache / "text").mkdir(parents=True, exist_ok=True)
        self.delay, self.offline, self.max_fetches = delay, offline, max_fetches
        self.last = {}
        self.fetched = 0

    def key(self, url):
        slug = re.sub(r"[^A-Za-z0-9]+", "-", urllib.parse.urlsplit(url).netloc + urllib.parse.urlsplit(url).path)[-60:]
        return hashlib.sha1(url.encode()).hexdigest()[:12] + "-" + slug.strip("-")

    def get(self, url):
        """Return the page as text, from the cache when present."""
        path = self.cache / "raw" / (self.key(url) + ".html")
        if path.exists():
            return path.read_text(encoding="utf-8")
        if self.offline:
            raise IOError("not cached and --offline given: %s" % url)
        if self.fetched >= self.max_fetches:
            raise IOError("fetch limit of %d reached" % self.max_fetches)
        host = urllib.parse.urlsplit(url).netloc
        agent = WIKI_AGENT if host.endswith(("wikipedia.org", "wikimedia.org")) else USER_AGENT
        for attempt in range(5):
            wait = self.last.get(host, 0) + self.delay - time.time()
            if wait > 0:
                time.sleep(wait)
            req = urllib.request.Request(url, headers={"User-Agent": agent, "Accept": "text/html,application/json;q=0.9,*/*;q=0.5"})
            try:
                with urllib.request.urlopen(req, timeout=40) as r:
                    raw = r.read()
                    ctype = r.headers.get("Content-Type", "")
                self.last[host] = time.time()
                self.fetched += 1
                with open(self.cache / "fetch_log.txt", "a") as log:
                    log.write("%s\t200\t%s\n" % (time.strftime("%Y-%m-%dT%H:%M:%S"), url))
                break
            except urllib.error.HTTPError as e:
                self.last[host] = time.time()
                self.fetched += 1
                with open(self.cache / "fetch_log.txt", "a") as log:
                    log.write("%s\t%d\t%s\n" % (time.strftime("%Y-%m-%dT%H:%M:%S"), e.code, url))
                if e.code in (429, 503) and attempt < 4:
                    retry = e.headers.get("Retry-After", "")
                    time.sleep(min(int(retry) if retry.isdigit() else 5 * 2 ** attempt, 120))
                    continue
                raise
        else:
            raise IOError("gave up on %s" % url)
        m = re.search(r"charset=([\w-]+)", ctype) or re.search(r"charset=[\"']?([\w-]+)", raw[:2048].decode("ascii", "replace"))
        text = None
        for enc in ([m.group(1)] if m else []) + ["utf-8", "cp1252"]:
            try:
                text = raw.decode(enc)
                break
            except (UnicodeDecodeError, LookupError):
                continue
        if text is None:
            text = raw.decode("utf-8", "replace")
        path.write_text(text, encoding="utf-8")
        return text


def fetch_url(url):
    """The address actually requested. A Wikipedia permalink (index.php?title=..&oldid=N)
    is read through the action API, which returns the article body without the site chrome."""
    parts = urllib.parse.urlsplit(url)
    if parts.netloc.endswith("wikipedia.org"):
        q = urllib.parse.parse_qs(parts.query)
        if "oldid" not in q:
            raise ValueError("Wikipedia sources need an oldid: %s" % url)
        return "https://%s/w/api.php?action=parse&oldid=%s&prop=text&format=json&formatversion=2" % (parts.netloc, q["oldid"][0])
    return url


def document(src, fetcher, max_words, min_words):
    url = src["url"]
    page = fetcher.get(fetch_url(url))
    if urllib.parse.urlsplit(url).netloc.endswith("wikipedia.org"):
        data = json.loads(page)["parse"]
        page = "<title>%s</title><div class=\"wiki-body\">%s</div>" % (data["title"], data["text"])
    title, paras = extract(page, url, src.get("start_heading"))
    full_words = sum(len(t.split()) for t in paras)
    paras, cut = truncate(paras, max_words)
    text = "\n\n".join(paras) + "\n"
    (fetcher.cache / "text" / (fetcher.key(url) + ".txt")).write_text(text, encoding="utf-8")
    f = features(text)
    return {"title": src.get("title") or title, "text": text, "features": f,
            "truncated": cut, "words_before_truncation": full_words,
            "ok": f["words"] >= min_words}


def percentile(xs, q):
    xs = sorted(xs)
    if not xs:
        return None
    pos = (len(xs) - 1) * q
    lo = int(pos)
    hi = min(lo + 1, len(xs) - 1)
    return xs[lo] + (xs[hi] - xs[lo]) * (pos - lo)


def rounded(d):
    return {k: (round(v, 4) if isinstance(v, float) else v) for k, v in d.items()}


def summary(docs):
    """docs: [(source record, features)]."""
    feats = [f for _, f in docs]
    per_doc = [pooled([f]) for f in feats]
    out = {
        "documents": len(docs),
        "words": sum(f["words"] for f in feats),
        "sentences": sum(len(f["lens"]) for f in feats),
        "sources": sorted({s["source"] for s, _ in docs}),
        "pooled": rounded(pooled(feats)),
        "percentiles": {},
    }
    for key in A.MEASURES:
        vals = [d[key] for d in per_doc if key in d]
        out["percentiles"][key] = {"p10": round(percentile(vals, 0.1), 4),
                                   "p50": round(percentile(vals, 0.5), 4),
                                   "p90": round(percentile(vals, 0.9), 4)}
    return out


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--sources", default=str(ROOT / "reference" / "human_sources.json"))
    ap.add_argument("--out", default=str(ROOT / "reference" / "human_reference.json"))
    ap.add_argument("--cache", default=DEFAULT_CACHE, help="directory for fetched pages and extracted text; outside the repository")
    ap.add_argument("--delay", type=float, default=1.0, help="seconds between requests to one host")
    ap.add_argument("--max-fetches", type=int, default=250)
    ap.add_argument("--max-words", type=int, default=3000)
    ap.add_argument("--min-words", type=int, default=300)
    ap.add_argument("--offline", action="store_true", help="use the cache only")
    ap.add_argument("--dump", metavar="URL", help="print the extracted text of one URL and stop")
    args = ap.parse_args()

    cache = Path(args.cache).resolve()
    if cache == ROOT or ROOT in cache.parents:
        sys.exit("The cache must be outside the repository: the fetched texts are copyrighted.")
    fetcher = Fetcher(cache, args.delay, args.offline, args.max_fetches)

    if args.dump:
        d = document({"url": args.dump}, fetcher, args.max_words, 0)
        print("# %s (%d words by profile.py, truncated: %s)\n" % (d["title"], d["features"]["words"], d["truncated"]))
        print(d["text"])
        return 0

    spec_ = json.loads(Path(args.sources).read_text(encoding="utf-8"))
    kinds = spec_["kinds"]
    docs, dropped = [], []
    for src in spec_["sources"]:
        if src["kind"] not in kinds:
            sys.exit("unknown kind %r for %s" % (src["kind"], src["url"]))
        try:
            d = document(src, fetcher, args.max_words, args.min_words)
        except Exception as e:                                    # one bad page must not stop the run
            dropped.append({"url": src["url"], "reason": "%s: %s" % (type(e).__name__, e)})
            print("DROPPED %s (%s)" % (src["url"], e), file=sys.stderr)
            continue
        if not d["ok"]:
            dropped.append({"url": src["url"], "reason": "only %d words extracted" % d["features"]["words"]})
            print("DROPPED %s (short)" % src["url"], file=sys.stderr)
            continue
        docs.append((src, d))

    result = {
        "note": ("Human-written reference prose, 2000 to 2022, measured with features() and pooled() "
                 "from scripts/analyse.py. Built by scripts/human_reference.py from reference/human_sources.json. "
                 "heading_share and list_share are near zero by construction because extraction keeps body "
                 "paragraphs only. Percentiles are across documents, each measured alone. The pooled lag1 is "
                 "taken about the pool's mean sentence length, so it runs higher than the per-document median. "
                 "Documents over %d words were truncated at a paragraph boundary." % args.max_words),
        "generated": time.strftime("%Y-%m-%d"),
        "max_words": args.max_words, "min_words": args.min_words,
        "all": summary([(s, d["features"]) for s, d in docs]),
        "by_kind": {k: summary([(s, d["features"]) for s, d in docs if s["kind"] == k])
                    for k in kinds if any(s["kind"] == k for s, _ in docs)},
        "documents": [],
        "dropped": dropped,
    }
    for s, d in docs:
        row = {"url": s["url"], "title": d["title"], "kind": s["kind"], "source": s["source"],
               "author": s.get("author"), "year": s["year"]}
        if s.get("brief_id"):
            row["brief_id"] = s["brief_id"]
        row.update({"words": d["features"]["words"], "sentences": len(d["features"]["lens"]),
                    "truncated": d["truncated"], "words_before_truncation": d["words_before_truncation"],
                    "measures": rounded(pooled([d["features"]]))})
        result["documents"].append(row)
    Path(args.out).write_text(json.dumps(result, indent=1, ensure_ascii=False) + "\n", encoding="utf-8")

    cols = ["mean_len", "cv_len", "lag1", "short_long", "le4_share", "conj_start", "em_dash",
            "semicolon", "colon", "paren", "joiners", "comma", "numbers_per_100", "names_per_100"]
    print("%-24s%6s%9s" % ("kind", "docs", "words") + "".join("%9s" % c[:8] for c in cols))
    for name, s in list(result["by_kind"].items()) + [("all", result["all"])]:
        print("%-24s%6d%9d" % (name, s["documents"], s["words"]) + "".join("%9.3f" % s["pooled"][c] for c in cols))
    print("fetched %d pages this run, %d documents, %d dropped" % (fetcher.fetched, len(docs), len(dropped)))
    return 0


if __name__ == "__main__":
    sys.exit(main())
