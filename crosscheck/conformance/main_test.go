package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree materialises a map of repo-relative paths to file contents under a
// fresh temp dir and returns the root. Parent dirs are created as needed.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	return root
}

func hasMatch(items []string, substr string) bool {
	for _, s := range items {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantName string // expected value of "name" key, "" if absent
		wantDesc bool   // whether "description" key is present
		wantKeys int
	}{
		{
			name: "present_full",
			content: "---\nname: reason\ndescription: does things\n---\n# body\n" +
				"prose here that is long enough to not be empty",
			wantName: "reason",
			wantDesc: true,
			wantKeys: 2,
		},
		{
			name:     "absent_no_leading_marker",
			content:  "# /assurance-probe\n\n**Layer**: 4\nlots of prose follows here",
			wantName: "",
			wantDesc: false,
			wantKeys: 0,
		},
		{
			name: "folded_scalar",
			content: "---\nname: drt-oracle\ndescription: >-\n  A folded scalar value\n" +
				"  spanning lines.\nargument-hint: \"[x]\"\n---\nbody text long enough",
			wantName: "drt-oracle",
			wantDesc: true,
			wantKeys: 3, // name, description, argument-hint
		},
		{
			name:     "no_closing_marker",
			content:  "---\nname: broken\ndescription: x\nstill no closing fence",
			wantName: "",
			wantDesc: false,
			wantKeys: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fm, body := parseFrontmatter(tc.content)
			if body != tc.content {
				t.Errorf("body mutated; want original content returned verbatim")
			}
			if got := fm["name"]; got != tc.wantName {
				t.Errorf("name = %q, want %q", got, tc.wantName)
			}
			if _, ok := fm["description"]; ok != tc.wantDesc {
				t.Errorf("description present = %v, want %v", ok, tc.wantDesc)
			}
			if len(fm) != tc.wantKeys {
				t.Errorf("key count = %d, want %d (got %v)", len(fm), tc.wantKeys, fm)
			}
		})
	}
}

func TestReferencedTokens(t *testing.T) {
	doc := "see `/reason` and `/drt-oracle`, also use /crosscheck:lean-spec here. " +
		"`not-a-token` lacks the slash; `/reason` repeats."
	got := referencedTokens(doc)
	want := []string{"drt-oracle", "lean-spec", "reason"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("referencedTokens = %v, want %v", got, want)
	}
}

func TestDocumented(t *testing.T) {
	doc := "intro `/reason` mid, and `byfuglien` agent, run /trace-execution now, " +
		"plus crosscheck:lean-impl invocation."
	tests := []struct {
		name string
		want bool
	}{
		{"reason", true},          // `/reason`
		{"byfuglien", true},       // `byfuglien`
		{"trace-execution", true}, // /trace-execution<space>
		{"lean-impl", true},       // crosscheck:lean-impl
		{"journal-context", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := documented(tc.name, doc); got != tc.want {
				t.Errorf("documented(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// baseTree returns a minimal-but-valid plugin tree: one well-formed skill, one
// well-formed agent, both documented, plus an empty ledger. Callers mutate it.
func baseTree() map[string]string {
	return map[string]string{
		"skills/reason/SKILL.md": "---\nname: reason\ndescription: reasons about code\n---\n" +
			"# /reason\n\nA skill body long enough to clear the empty threshold easily.",
		"agents/byfuglien.md": "---\nname: byfuglien\ndescription: orchestrates verification\n---\n" +
			"# Byfuglien\n\nbody text.",
		"README.md":               "Crosscheck ships `/reason` and the `byfuglien` orchestrator.",
		"conformance/claims.json": `{"version":1,"narrative_claims":[]}`,
	}
}

func TestPhantomDetection(t *testing.T) {
	files := baseTree()
	// Reference a skill that does not exist on disk.
	files["README.md"] += " It also offers `/ghost-skill` which is not implemented."
	r := analyze(writeTree(t, files))

	if !hasMatch(r.errors, "[phantom]") || !hasMatch(r.errors, "ghost-skill") {
		t.Errorf("expected phantom error for ghost-skill, got errors: %v", r.errors)
	}
	// The real artifacts must not produce phantom errors.
	if hasMatch(r.errors, "reason") || hasMatch(r.errors, "byfuglien") {
		t.Errorf("unexpected phantom error for real artifacts: %v", r.errors)
	}
}

func TestRoutingIntegrity(t *testing.T) {
	files := baseTree()
	// An agent that routes to a skill which does not exist on disk.
	files["agents/router.md"] = "---\nname: router\ndescription: routes work\n---\n" +
		"# Router\n\nFor proofs, hand to `/reason`. For specs, run `/crosscheck:ghost-route` next."
	files["README.md"] += " The `router` agent coordinates the chain."
	r := analyze(writeTree(t, files))

	if !hasMatch(r.errors, "[routing]") || !hasMatch(r.errors, "ghost-route") {
		t.Errorf("expected routing error for ghost-route, got errors: %v", r.errors)
	}
	// A real skill the agent routes to must not be flagged.
	if hasMatch(r.errors, "routes to '/reason'") {
		t.Errorf("valid routing target '/reason' must not be flagged: %v", r.errors)
	}
}

// TestRoutingIgnoresFrontmatter pins the AUTO 5 fix: a `/crosscheck:x` token in
// an agent's frontmatter `description:` is documentation, not a routing edge, so
// it must NOT raise a routing error. Only the agent body is scanned.
func TestRoutingIgnoresFrontmatter(t *testing.T) {
	files := baseTree()
	// The phantom token lives ONLY in the frontmatter description; the body
	// routes to nothing unresolved.
	files["agents/describer.md"] = "---\nname: describer\n" +
		"description: orchestrates the chain; can invoke /crosscheck:phantom-fm style skills\n---\n" +
		"# Describer\n\nFor proofs, hand to `/reason`."
	files["README.md"] += " The `describer` agent coordinates the chain."
	r := analyze(writeTree(t, files))

	if hasMatch(r.errors, "phantom-fm") {
		t.Errorf("a /crosscheck:x token in frontmatter must not raise a routing error: %v", r.errors)
	}
	// Sanity: the same token in the *body* would still be caught.
	files["agents/describer.md"] = "---\nname: describer\ndescription: orchestrates the chain\n---\n" +
		"# Describer\n\nFor specs, run `/crosscheck:phantom-body` next."
	r = analyze(writeTree(t, files))
	if !hasMatch(r.errors, "[routing]") || !hasMatch(r.errors, "phantom-body") {
		t.Errorf("a phantom routing token in the body must still be caught: %v", r.errors)
	}
}

// TestStripFrontmatter pins the body-extraction helper that AUTO 5 relies on.
func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "strips_block",
			content: "---\nname: x\ndescription: d\n---\nbody line one\nbody line two",
			want:    "body line one\nbody line two",
		},
		{
			name:    "no_frontmatter_returned_verbatim",
			content: "# Heading\n\nplain body, no frontmatter",
			want:    "# Heading\n\nplain body, no frontmatter",
		},
		{
			name:    "unterminated_returned_verbatim",
			content: "---\nname: broken\nno closing fence",
			want:    "---\nname: broken\nno closing fence",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripFrontmatter(tc.content); got != tc.want {
				t.Errorf("stripFrontmatter() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOrphanDetection(t *testing.T) {
	files := baseTree()
	// Add a skill that no doc mentions.
	files["skills/lonely/SKILL.md"] = "---\nname: lonely\ndescription: undocumented\n---\n" +
		"# /lonely\n\nbody text long enough to not be empty at all."
	r := analyze(writeTree(t, files))

	if !hasMatch(r.warnings, "[orphan]") || !hasMatch(r.warnings, "lonely") {
		t.Errorf("expected orphan warning for 'lonely', got warnings: %v", r.warnings)
	}
	// Orphan is a WARNING, never an ERROR.
	if hasMatch(r.errors, "lonely") {
		t.Errorf("orphan must not be an error: %v", r.errors)
	}
}

func TestStructuralMissingFrontmatter(t *testing.T) {
	files := baseTree()
	// This mirrors the real assurance-probe defect: a SKILL.md that opens with a
	// prose header instead of YAML frontmatter cannot register as a skill.
	files["skills/probe/SKILL.md"] = "# /probe\n\n**Layer**: 4\n\nprose body, no frontmatter at all here."
	r := analyze(writeTree(t, files))

	if !hasMatch(r.errors, "[structural]") || !hasMatch(r.errors, "probe") ||
		!hasMatch(r.errors, "missing frontmatter keys ['description', 'name']") {
		t.Errorf("expected structural missing-frontmatter error for 'probe', got: %v", r.errors)
	}
}

func TestStructuralNameMismatch(t *testing.T) {
	files := baseTree()
	files["skills/widget/SKILL.md"] = "---\nname: gadget\ndescription: mislabelled\n---\n" +
		"# /widget\n\nbody text long enough to not be empty here."
	r := analyze(writeTree(t, files))
	if !hasMatch(r.errors, "skill dir 'widget' != frontmatter name 'gadget'") {
		t.Errorf("expected dir/name mismatch error, got: %v", r.errors)
	}
}

func TestStructuralEmptySkill(t *testing.T) {
	files := baseTree()
	files["skills/tiny/SKILL.md"] = "---\nname: tiny\ndescription: d\n---\n"
	r := analyze(writeTree(t, files))
	if !hasMatch(r.errors, "skill 'tiny': SKILL.md is effectively empty") {
		t.Errorf("expected empty-skill error, got: %v", r.errors)
	}
}

func boolp(b bool) *bool { return &b }

func TestPresentArtifactLedger(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expect    *bool
		create    bool // whether to create the path under root
		wantError bool
	}{
		{"absent_expected_absent", "agents/lowry.md", boolp(false), false, false},
		{"present_expected_absent", "agents/here.md", boolp(false), true, true},
		{"present_expected_present", "agents/here.md", boolp(true), true, false},
		{"absent_expected_present", "agents/gone.md", boolp(true), false, true},
		{"absent_default_present", "agents/gone.md", nil, false, true}, // default expect_present=true
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files := baseTree()
			if tc.create {
				files[tc.path] = "placeholder"
			}
			expectField := "true"
			if tc.expect != nil && !*tc.expect {
				expectField = "false"
			}
			ledgerJSON := `{"version":1,"narrative_claims":[{"id":"C","claim":"c","reality":"r",` +
				`"status":"known-gap","check":{"type":"present_artifact","path":"` + tc.path + `"`
			if tc.expect != nil {
				ledgerJSON += `,"expect_present":` + expectField
			}
			ledgerJSON += `}}]}`
			files["conformance/claims.json"] = ledgerJSON

			r := analyze(writeTree(t, files))
			gotError := hasMatch(r.errors, "[ledger] claim C auto-check failed")
			if gotError != tc.wantError {
				t.Errorf("ledger error = %v, want %v (errors: %v)", gotError, tc.wantError, r.errors)
			}
		})
	}
}

func TestUnreviewedLedgerFails(t *testing.T) {
	files := baseTree()
	files["conformance/claims.json"] = `{"version":1,"narrative_claims":[` +
		`{"id":"C-NEW","claim":"c","reality":"r","status":"unreviewed","check":{"type":"manual"}}]}`
	r := analyze(writeTree(t, files))
	if !hasMatch(r.errors, "[ledger] claim C-NEW is UNREVIEWED") {
		t.Errorf("expected UNREVIEWED ledger error, got: %v", r.errors)
	}
}

func TestKnownGapNeedsTracking(t *testing.T) {
	// A known-gap with no tracked_in link fails CI.
	files := baseTree()
	files["conformance/claims.json"] = `{"version":1,"narrative_claims":[` +
		`{"id":"C-GAP","claim":"c","reality":"r","status":"known-gap","check":{"type":"manual"}}]}`
	r := analyze(writeTree(t, files))
	if !hasMatch(r.errors, "claim C-GAP is a known-gap with no tracked_in link") {
		t.Errorf("expected known-gap-without-tracking error, got: %v", r.errors)
	}

	// The same gap with a tracked_in link is clean.
	files["conformance/claims.json"] = `{"version":1,"narrative_claims":[` +
		`{"id":"C-GAP","claim":"c","reality":"r","status":"known-gap",` +
		`"tracked_in":"https://github.com/nicholls-inc/claude-code-marketplace/issues/217","check":{"type":"manual"}}]}`
	r = analyze(writeTree(t, files))
	if hasMatch(r.errors, "no tracked_in link") {
		t.Errorf("known-gap with tracking should not error, got: %v", r.errors)
	}
}

func TestReportPassFail(t *testing.T) {
	pass := report(result{})
	if !strings.Contains(pass, "RESULT: PASS") {
		t.Errorf("empty result should report PASS, got:\n%s", pass)
	}
	fail := report(result{errors: []string{"boom"}})
	if !strings.Contains(fail, "RESULT: FAIL") {
		t.Errorf("result with errors should report FAIL, got:\n%s", fail)
	}
}

// TestGoldenRealTree pins the oracle's verdict against the actual crosscheck/
// plugin tree (the parent of this package dir). It asserts the stable inventory
// and the post-fix gate state: assurance-probe now has frontmatter, the
// journal-context orphan WARNING remains a human decision, and all seven ledger
// claims hold (three of them known-gap present_artifact/manual checks;
// CLAIM-METHODOLOGY-COMMITTED and CLAIM-SELF-COVERAGE both triaged to
// reviewed-disclosed per epic #217 / issue #221).
func TestGoldenRealTree(t *testing.T) {
	root := ".." // package dir is crosscheck/conformance; plugin root is crosscheck/
	if _, err := os.Stat(filepath.Join(root, "skills")); err != nil {
		t.Skipf("real plugin tree not found at %s: %v", root, err)
	}
	r := analyze(root)

	if len(r.skills) != 30 {
		t.Errorf("skills discovered = %d, want 30", len(r.skills))
	}
	if len(r.agents) != 3 {
		t.Errorf("agents discovered = %d, want 3", len(r.agents))
	}
	if len(r.refTokens) != 30 {
		t.Errorf("referenced tokens = %d, want 30", len(r.refTokens))
	}
	if len(r.ledger) != 7 {
		t.Fatalf("ledger claims = %d, want 7", len(r.ledger))
	}

	// journal-context is now documented in the README skills overview, so it must
	// no longer be flagged as an orphan.
	if hasMatch(r.warnings, "journal-context") {
		t.Errorf("journal-context should be documented, but is still flagged as an orphan: %v", r.warnings)
	}

	// The remaining known-gap claims must all be present in the ledger.
	wantGaps := []string{"CLAIM-PHASE4", "CLAIM-MODES", "CLAIM-AUDITOR"}
	for _, id := range wantGaps {
		found := false
		for _, c := range r.ledger {
			if c.ID == id {
				found = true
				if c.Status != "known-gap" {
					t.Errorf("claim %s status = %q, want known-gap", id, c.Status)
				}
			}
		}
		if !found {
			t.Errorf("missing expected ledger claim %s", id)
		}
	}

	// CLAIM-METHODOLOGY-COMMITTED was triaged to reviewed-disclosed (epic #217
	// decision: close as archived — the v1 methodology was retracted on
	// 2026-05-11, not promoted to canonical).
	{
		found := false
		for _, c := range r.ledger {
			if c.ID == "CLAIM-METHODOLOGY-COMMITTED" {
				found = true
				if c.Status != "reviewed-disclosed" {
					t.Errorf("claim CLAIM-METHODOLOGY-COMMITTED status = %q, want reviewed-disclosed", c.Status)
				}
			}
		}
		if !found {
			t.Errorf("missing expected ledger claim CLAIM-METHODOLOGY-COMMITTED")
		}
	}

	// CLAIM-SELF-COVERAGE was triaged to reviewed-disclosed once a second
	// trunk-level self-check (the AUTO 5 orchestration-graph integrity check)
	// shipped beyond this oracle itself (issue #221).
	{
		found := false
		for _, c := range r.ledger {
			if c.ID == "CLAIM-SELF-COVERAGE" {
				found = true
				if c.Status != "reviewed-disclosed" {
					t.Errorf("claim CLAIM-SELF-COVERAGE status = %q, want reviewed-disclosed", c.Status)
				}
			}
		}
		if !found {
			t.Errorf("missing expected ledger claim CLAIM-SELF-COVERAGE")
		}
	}

	// Post-fix expectation: the gate is GREEN. assurance-probe now parses, and the
	// present_artifact ledger checks (lowry/methodology/auditor absent) all hold,
	// so there are no ERRORs.
	if hasMatch(r.errors, "assurance-probe") {
		t.Errorf("assurance-probe should have valid frontmatter post-fix; errors: %v", r.errors)
	}
	if len(r.errors) != 0 {
		t.Errorf("expected RESULT PASS (0 errors) post-fix, got %d: %v", len(r.errors), r.errors)
	}
}
