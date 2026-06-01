package main

// Cross-toolchain guard for the ADD invariant-heading grammar (issue #227).
//
// Three shipped artifacts independently parse or gate on the invariant-heading
// form, and they used to disagree: the add-orchestrator quality gate required
// h2 `## I<N>:`, while the invariant-coverage-scaffold templates parsed the
// legacy bold-prefix `**I1. Name.**`. A module that passed one failed the
// other. The conformance oracle did not catch it — it checks reference and
// inventory integrity, not semantic agreement between two parsers.
//
// This guard pins every parser/gate to ONE canonical grammar by extracting the
// real pattern each artifact ships and running a shared corpus of headings
// through all of them. If any artifact's pattern drifts, the corpus stops
// agreeing and this test fails. It is deliberately extraction-based, not a
// string-literal match, so it survives reformatting but still bites on a
// genuine grammar change.
//
// Canonical grammar (skills/draft-invariants/SKILL.md Step 3): an h2 heading
// `## I<N>: <Name>`, where <N> is digits with an optional lowercase
// sub-invariant suffix (I1, I1a). If you change this, change it in:
//   - agents/add-orchestrator.md Step 6 quality-gate grep
//   - skills/invariant-coverage-scaffold/references/{python,go,typescript}-template.md HEADER_RE
//   - skills/assurance-init/SKILL.md emitted invariant-doc template
//   - skills/draft-invariants/SKILL.md Step 3 heading convention
// ...and this test, which will fail until they agree again.

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// canonicalHeading is the single source of truth this guard pins everything to.
var canonicalHeading = regexp.MustCompile(`^## (I\d+[a-z]?):`)

// headingCorpus is the shared accept/reject set. Every shipped pattern must
// classify each line exactly as canonicalHeading does.
var headingCorpus = []struct {
	line string
	want bool
}{
	{"## I1: RedactionSentinelString", true},
	{"## I7: BothSetFileWins", true},
	{"## I42: Foo", true},
	{"## I1a: SubInvariant", true}, // optional lowercase sub-invariant suffix
	{"### I1: Name", false},        // h3, not h2
	{"#### I1: x", false},          // h4
	{"#I1: nospace", false},        // not an h2 heading
	{" ## I1: leadingspace", false},
	{"**I1. RedactionSentinelString.**", false},          // legacy bold-prefix form
	{"## Invariants", false},                             // section header, not an invariant
	{"## Engine selection — single embedding API", false}, // prose-section heading
	{"## A1: WrongPrefix", false},                        // non-I prefix
	{"## i1: lowercasePrefix", false},                    // lowercase prefix
	{"I1: not a heading", false},
	{"**I1:** inline reference, not a heading", false},
}

// crosscheckRoot resolves the plugin root regardless of whether the test binary
// runs from crosscheck/conformance (the usual `go test ./...` case) or the repo
// root.
func crosscheckRoot(t *testing.T) string {
	t.Helper()
	for _, cand := range []string{"..", "crosscheck", "."} {
		if _, err := os.Stat(filepath.Join(cand, "agents", "add-orchestrator.md")); err == nil {
			return cand
		}
	}
	t.Fatalf("could not locate the crosscheck plugin root (agents/add-orchestrator.md not found from %q)", mustGetwd(t))
	return ""
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	return wd
}

// extract pulls the first capture group of locator out of the named file,
// failing loudly if the anchor has moved (a rename should trip this guard, not
// silently skip it).
func extract(t *testing.T, root, rel string, locator *regexp.Regexp) string {
	t.Helper()
	path := filepath.Join(root, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	m := locator.FindSubmatch(data)
	if m == nil {
		t.Fatalf("could not locate the heading pattern in %s via /%s/ — if you renamed or reformatted it, update this guard", rel, locator)
	}
	return string(m[1])
}

// assertAgrees compiles a shipped pattern and checks it classifies the corpus
// exactly as the canonical grammar does.
func assertAgrees(t *testing.T, source, pattern string) {
	t.Helper()
	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("%s: shipped pattern %q does not compile as a Go regexp: %v", source, pattern, err)
	}
	for _, c := range headingCorpus {
		got := re.MatchString(c.line)
		if got != c.want {
			t.Errorf("%s pattern %q disagrees with canonical on %q: got match=%v, want %v",
				source, pattern, c.line, got, c.want)
		}
	}
}

// TestCanonicalGrammarSelfConsistent is a sanity check that the corpus matches
// the canonical regex as annotated — guards against a typo in the corpus itself.
func TestCanonicalGrammarSelfConsistent(t *testing.T) {
	for _, c := range headingCorpus {
		if got := canonicalHeading.MatchString(c.line); got != c.want {
			t.Errorf("corpus annotation wrong for %q: canonical match=%v, annotated want=%v", c.line, got, c.want)
		}
	}
}

// TestHeadingGrammarAgreement is the cross-toolchain guard: the orchestrator
// gate grep and all three coverage-scaffold parsers must accept the same
// heading grammar.
func TestHeadingGrammarAgreement(t *testing.T) {
	root := crosscheckRoot(t)

	cases := []struct {
		source  string
		rel     string
		locator *regexp.Regexp
	}{
		{
			source:  "add-orchestrator Step 6 grep",
			rel:     filepath.Join("agents", "add-orchestrator.md"),
			locator: regexp.MustCompile("grep -cE '([^']*)'"),
		},
		{
			source:  "python-template HEADER_RE",
			rel:     filepath.Join("skills", "invariant-coverage-scaffold", "references", "python-template.md"),
			locator: regexp.MustCompile(`HEADER_RE = re\.compile\(r"([^"]*)"\)`),
		},
		{
			source:  "go-template headerRe",
			rel:     filepath.Join("skills", "invariant-coverage-scaffold", "references", "go-template.md"),
			locator: regexp.MustCompile("headerRe\\s*=\\s*regexp\\.MustCompile\\(`([^`]*)`\\)"),
		},
		{
			source:  "typescript-template HEADER_RE",
			rel:     filepath.Join("skills", "invariant-coverage-scaffold", "references", "typescript-template.md"),
			locator: regexp.MustCompile(`const HEADER_RE = /([^/]*)/;`),
		},
	}

	for _, c := range cases {
		pattern := extract(t, root, c.rel, c.locator)
		assertAgrees(t, c.source, pattern)
	}
}

// TestAssuranceInitEmitsCanonical checks that the skeleton /assurance-init
// writes uses canonical h2 invariant headings and no legacy bold-prefix emit.
func TestAssuranceInitEmitsCanonical(t *testing.T) {
	root := crosscheckRoot(t)
	path := filepath.Join(root, "skills", "assurance-init", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	multiCanonical := regexp.MustCompile(`(?m)^## I\d+[a-z]?:`)
	if n := len(multiCanonical.FindAll(data, -1)); n < 2 {
		t.Errorf("assurance-init SKILL.md should emit >=2 canonical h2 invariant headings, found %d", n)
	}
	legacyEmit := regexp.MustCompile(`(?m)^\*\*I\d+\.\s`)
	if legacyEmit.Match(data) {
		t.Errorf("assurance-init SKILL.md still emits a legacy bold-prefix invariant heading (**I<N>. ...); migrate it to '## I<N>: <Name>'")
	}
}
