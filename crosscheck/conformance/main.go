// Command conformance is the Crosscheck conformance / inventory oracle.
//
// It verifies that what the documentation CLAIMS Crosscheck ships matches what
// the filesystem actually contains. This is the bidirectional-coverage-gate
// pattern (docs/invariants <-> tests) lifted to the meta level: docs <->
// artifacts.
//
// Two layers of check:
//
//	AUTO   - deterministic, no false positives, fail CI on ERROR
//	         (artifact<->doc reference integrity + structural integrity)
//	LEDGER - narrative claims from claims.json that can't be auto-verified
//	         (layer/phase/mode counts, terminal states). Surfaced for review,
//	         dated, and tracked. Drift here is a known-gap, not a silent one.
//
// Exit 1 if any AUTO check is ERROR. LEDGER never fails CI by itself, but any
// ledger entry with status 'unreviewed' is promoted to an ERROR (forces
// triage), and a 'present_artifact' auto-check that disagrees with reality is
// likewise promoted to an ERROR.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// docFiles is the user-facing documentation set scanned for references. Files
// that do not exist are skipped.
var docFiles = []string{
	"README.md",
	"docs/skills.md",
	"docs/agents.md",
	"docs/assurance-hierarchy.md",
	"docs/research/assurance-hierarchy.md",
}

// reqKeys are the frontmatter keys every skill and agent must declare.
var reqKeys = []string{"name", "description"}

var (
	fmLineRe    = regexp.MustCompile(`^([a-zA-Z0-9_-]+):\s*(.*)$`)
	slashTokRe  = regexp.MustCompile("\x60/([a-z][a-z0-9-]+)\x60")
	xcheckRe    = regexp.MustCompile(`/crosscheck:([a-z][a-z0-9-]+)`)
	dafnyToolRe = regexp.MustCompile("\x60(dafny_[a-z]+)\x60")
)

// skill is a discovered skills/<dir>/SKILL.md artifact.
type skill struct {
	name  string
	fm    map[string]string
	empty bool
}

// agent is a discovered agents/<stem>.md artifact.
type agent struct {
	name string
	fm   map[string]string
}

// claim is one narrative-ledger entry from claims.json.
type claim struct {
	ID        string     `json:"id"`
	Source    string     `json:"source"`
	Claim     string     `json:"claim"`
	Reality   string     `json:"reality"`
	Status    string     `json:"status"`
	Check     claimCheck `json:"check"`
	TrackedIn string     `json:"tracked_in"`
}

type claimCheck struct {
	Type          string `json:"type"`
	Path          string `json:"path"`
	ExpectPresent *bool  `json:"expect_present"`
}

type ledgerFile struct {
	NarrativeClaims []claim `json:"narrative_claims"`
}

// result holds everything the oracle discovered and decided, so that report
// rendering and the process exit code are pure functions of it.
type result struct {
	skills      []skill
	agents      []agent
	presentDocs []string
	refTokens   []string
	errors      []string
	warnings    []string
	ledger      []claim
}

// parseFrontmatter parses a leading YAML frontmatter block (the text between
// the opening "---" and the next "\n---") into a key->value map. It tolerates
// folded scalars such as `description: >-` (the value is captured verbatim as
// ">-", which is non-empty, so the key counts as present). It returns the
// parsed map and the original content unchanged.
func parseFrontmatter(content string) (map[string]string, string) {
	if !strings.HasPrefix(content, "---") {
		return map[string]string{}, content
	}
	rel := strings.Index(content[3:], "\n---")
	if rel == -1 {
		return map[string]string{}, content
	}
	end := 3 + rel
	fm := map[string]string{}
	for _, line := range strings.Split(content[3:end], "\n") {
		if m := fmLineRe.FindStringSubmatch(line); m != nil {
			fm[m[1]] = strings.TrimSpace(m[2])
		}
	}
	return fm, content
}

// readFile reads a file as a string, mirroring Python's errors="replace": any
// read error yields the empty string rather than aborting.
func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// discoverSkills returns every subdir of skills/ that holds a SKILL.md, sorted
// by directory name.
func discoverSkills(root string) []skill {
	var out []skill
	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		return out
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		sk := filepath.Join(root, "skills", name, "SKILL.md")
		if !fileExists(sk) {
			continue
		}
		fm, body := parseFrontmatter(readFile(sk))
		out = append(out, skill{
			name:  name,
			fm:    fm,
			empty: len(strings.TrimSpace(body)) < 50,
		})
	}
	return out
}

// discoverAgents returns every agents/*.md artifact, sorted by file stem.
func discoverAgents(root string) []agent {
	var out []agent
	matches, err := filepath.Glob(filepath.Join(root, "agents", "*.md"))
	if err != nil {
		return out
	}
	sort.Strings(matches)
	for _, f := range matches {
		fm, _ := parseFrontmatter(readFile(f))
		stem := strings.TrimSuffix(filepath.Base(f), ".md")
		out = append(out, agent{name: stem, fm: fm})
	}
	return out
}

// scanDocs concatenates the existing doc-set files and records which were
// present, mirroring the reference (a leading "\n" before each file's text).
func scanDocs(root string) (docText string, present []string) {
	for _, rel := range docFiles {
		p := filepath.Join(root, rel)
		if fileExists(p) {
			docText += "\n" + readFile(p)
			present = append(present, rel)
		}
	}
	return docText, present
}

// referencedTokens extracts the set of `/token` and /crosscheck:token tokens
// referenced anywhere in the doc text, sorted.
func referencedTokens(docText string) []string {
	set := map[string]struct{}{}
	for _, m := range slashTokRe.FindAllStringSubmatch(docText, -1) {
		set[m[1]] = struct{}{}
	}
	for _, m := range xcheckRe.FindAllStringSubmatch(docText, -1) {
		set[m[1]] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// documented reports whether an artifact name appears in the doc text in any
// of the forms the reference recognises.
func documented(name, docText string) bool {
	return strings.Contains(docText, "`/"+name+"`") ||
		strings.Contains(docText, "crosscheck:"+name) ||
		strings.Contains(docText, "`"+name+"`") ||
		strings.Contains(docText, "/"+name+" ")
}

// pyList renders a string slice the way Python repr does: ['a', 'b'].
func pyList(items []string) string {
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = "'" + s + "'"
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// missingKeys returns the required keys absent from fm, sorted.
func missingKeys(fm map[string]string) []string {
	var missing []string
	for _, k := range reqKeys {
		if _, ok := fm[k]; !ok {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	return missing
}

// analyze runs every AUTO and LEDGER check against the plugin tree rooted at
// root and returns the assembled result.
func analyze(root string) result {
	var r result
	r.skills = discoverSkills(root)
	r.agents = discoverAgents(root)
	docText, present := scanDocs(root)
	r.presentDocs = present
	r.refTokens = referencedTokens(docText)

	known := map[string]struct{}{}
	for _, s := range r.skills {
		known[s.name] = struct{}{}
	}
	for _, a := range r.agents {
		known[a.name] = struct{}{}
	}

	// ---- AUTO 1: structural integrity ----
	for _, s := range r.skills {
		if missing := missingKeys(s.fm); len(missing) > 0 {
			r.errors = append(r.errors, fmt.Sprintf(
				"[structural] skill '%s': SKILL.md missing frontmatter keys %s", s.name, pyList(missing)))
		}
		if n := s.fm["name"]; n != "" && n != s.name {
			r.errors = append(r.errors, fmt.Sprintf(
				"[structural] skill dir '%s' != frontmatter name '%s'", s.name, n))
		}
		if s.empty {
			r.errors = append(r.errors, fmt.Sprintf(
				"[structural] skill '%s': SKILL.md is effectively empty", s.name))
		}
	}
	for _, a := range r.agents {
		if missing := missingKeys(a.fm); len(missing) > 0 {
			r.errors = append(r.errors, fmt.Sprintf(
				"[structural] agent '%s': missing frontmatter keys %s", a.name, pyList(missing)))
		}
		if n := a.fm["name"]; n != "" && n != a.name {
			r.errors = append(r.errors, fmt.Sprintf(
				"[structural] agent file '%s.md' != frontmatter name '%s'", a.name, n))
		}
	}

	// ---- AUTO 2: phantom (doc references an artifact that does not exist) ----
	for _, tok := range r.refTokens {
		if _, ok := known[tok]; !ok {
			r.errors = append(r.errors, fmt.Sprintf(
				"[phantom] docs reference '/%s' but no skills/%s/ or agents/%s.md exists", tok, tok, tok))
		}
	}

	// ---- AUTO 3: orphan (artifact exists but is referenced nowhere) ----
	for _, s := range r.skills {
		if !documented(s.name, docText) {
			r.warnings = append(r.warnings, fmt.Sprintf(
				"[orphan] skill '%s' ships but is not referenced in any doc %s", s.name, pyList(present)))
		}
	}
	for _, a := range r.agents {
		if !documented(a.name, docText) {
			r.warnings = append(r.warnings, fmt.Sprintf(
				"[orphan] agent '%s' ships but is not referenced in any doc", a.name))
		}
	}

	// ---- AUTO 4: MCP tools claimed vs mcp-server source ----
	mcpSrc := readMCPSource(root)
	for _, m := range dafnyToolRe.FindAllStringSubmatch(docText, -1) {
		tool := m[1]
		if mcpSrc != "" && !strings.Contains(mcpSrc, tool) {
			r.warnings = append(r.warnings, fmt.Sprintf(
				"[mcp] README claims MCP tool '%s' but it's not found in mcp-server source", tool))
		}
	}

	// ---- AUTO 5: orchestration-graph integrity (trunk self-coverage) ----
	// The phantom check (AUTO 2) only scans the user-facing doc set, so an
	// orchestrator that routes to a skill/agent which does not exist would slip
	// through. This extends reference integrity to the *trunk*: every skill or
	// agent that an agent's body routes to (via `/crosscheck:x` or `/x`) must
	// resolve to a real artifact. It is the second trunk-level self-check after
	// this oracle itself (see CLAIM-SELF-COVERAGE, issue #221).
	for _, a := range r.agents {
		body := readFile(filepath.Join(root, "agents", a.name+".md"))
		for _, tok := range referencedTokens(body) {
			if _, ok := known[tok]; !ok {
				r.errors = append(r.errors, fmt.Sprintf(
					"[routing] agent '%s' routes to '/%s' but no skills/%s/ or agents/%s.md exists", a.name, tok, tok, tok))
			}
		}
	}

	// ---- LEDGER: narrative claims ----
	r.ledger = loadLedger(root)
	for _, c := range r.ledger {
		if c.Status == "unreviewed" {
			r.errors = append(r.errors, fmt.Sprintf(
				"[ledger] claim %s is UNREVIEWED — triage required", c.ID))
		}
		if c.Status == "known-gap" && strings.TrimSpace(c.TrackedIn) == "" {
			r.errors = append(r.errors, fmt.Sprintf(
				"[ledger] claim %s is a known-gap with no tracked_in link", c.ID))
		}
		if c.Check.Type == "present_artifact" {
			expect := true
			if c.Check.ExpectPresent != nil {
				expect = *c.Check.ExpectPresent
			}
			exists := fileExists(filepath.Join(root, c.Check.Path))
			if exists != expect {
				r.errors = append(r.errors, fmt.Sprintf(
					"[ledger] claim %s auto-check failed: %s present=%v, expected_present=%v",
					c.ID, c.Check.Path, exists, expect))
			}
		}
	}

	return r
}

// readMCPSource concatenates every .ts and .js file under mcp-server/, or ""
// if the directory does not exist.
func readMCPSource(root string) string {
	dir := filepath.Join(root, "mcp-server")
	if !fileExists(dir) {
		return ""
	}
	var sb strings.Builder
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".js") {
			sb.WriteString(readFile(path))
		}
		return nil
	})
	return sb.String()
}

// loadLedger reads conformance/claims.json, or returns nil if absent/unreadable.
func loadLedger(root string) []claim {
	data := readFile(filepath.Join(root, "conformance", "claims.json"))
	if data == "" {
		return nil
	}
	var lf ledgerFile
	if err := json.Unmarshal([]byte(data), &lf); err != nil {
		return nil
	}
	return lf.NarrativeClaims
}

// report renders the human-readable oracle report from a result.
func report(r result) string {
	var b strings.Builder
	line := strings.Repeat("=", 72)
	dash := strings.Repeat("-", 72)
	fmt.Fprintln(&b, line)
	fmt.Fprintln(&b, "CROSSCHECK CONFORMANCE / INVENTORY ORACLE")
	fmt.Fprintln(&b, line)
	fmt.Fprintf(&b, "skills discovered : %d\n", len(r.skills))
	fmt.Fprintf(&b, "agents discovered : %d\n", len(r.agents))
	fmt.Fprintf(&b, "docs scanned      : %s\n", pyList(r.presentDocs))
	fmt.Fprintf(&b, "referenced tokens : %d\n", len(r.refTokens))
	fmt.Fprintln(&b, dash)
	fmt.Fprintf(&b, "ERRORS   : %d\n", len(r.errors))
	for _, e := range r.errors {
		fmt.Fprintf(&b, "  ✗ %s\n", e)
	}
	fmt.Fprintf(&b, "WARNINGS : %d\n", len(r.warnings))
	for _, w := range r.warnings {
		fmt.Fprintf(&b, "  ⚠ %s\n", w)
	}
	fmt.Fprintln(&b, dash)
	fmt.Fprintf(&b, "NARRATIVE LEDGER (%d claims):\n", len(r.ledger))
	for _, c := range r.ledger {
		status := c.Status
		if status == "" {
			status = "?"
		}
		fmt.Fprintf(&b, "  • %s [%s]  %s\n", c.ID, status, c.Claim)
		fmt.Fprintf(&b, "      reality: %s\n", c.Reality)
		if c.TrackedIn != "" {
			fmt.Fprintf(&b, "      tracked: %s\n", c.TrackedIn)
		}
	}
	fmt.Fprintln(&b, line)
	res := "PASS"
	if len(r.errors) > 0 {
		res = "FAIL"
	}
	fmt.Fprintf(&b, "RESULT: %s\n", res)
	return b.String()
}

// defaultRoot resolves the scanned root: an explicit override arg, else the
// "crosscheck" plugin dir relative to the current working directory. The oracle
// is designed to be invoked from the repo root (`go run ./crosscheck/conformance`),
// so the bare default points at the plugin dir there. If that directory is not
// present (e.g. invoked from inside the plugin dir itself), it falls back to ".".
func defaultRoot(args []string) string {
	if len(args) > 1 {
		return args[1]
	}
	if fileExists("crosscheck") {
		return "crosscheck"
	}
	return "."
}

func main() {
	root := defaultRoot(os.Args)
	r := analyze(root)
	fmt.Print(report(r))
	if len(r.errors) > 0 {
		os.Exit(1)
	}
}
