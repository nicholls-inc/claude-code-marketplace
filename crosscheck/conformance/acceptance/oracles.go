package acceptance

// Shared, dependency-free scaffolding for the A1–A6 acceptance oracles. This
// file is intentionally UNtagged (no //go:build acceptance) so the package
// always has buildable Go files under the default tags; the actual checks live
// in the build-tagged *_test.go files.

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrPendingRatification is returned by every judged-oracle seam until the
// maintainer ratifies the A1–A6 contracts (RATIFY.md) AND the Phase 4 build +
// judged-oracle harness exist. It is intentionally non-nil so the acceptance
// lane stays RED by design.
var ErrPendingRatification = errors.New(
	"judged oracle pending: ratify RATIFY.md and build the scenario runner + LLM judge before this can pass")

// ErrCapabilityAbsent marks a deterministic oracle whose target mechanism
// (mode tags, commit-shape classifier, drift-stop agent) does not yet ship.
var ErrCapabilityAbsent = errors.New("target capability not yet shipped")

// OracleClass is the ADR-002 split.
type OracleClass int

const (
	// Deterministic: a pure function over repo state / git history / commit grammar.
	Deterministic OracleClass = iota
	// Judged: a scripted scenario run whose transcript is scored by an LLM judge.
	Judged
)

func (c OracleClass) String() string {
	if c == Judged {
		return "judged"
	}
	return "deterministic"
}

// Oracle is the metadata each A-series check carries; Pass mirrors the one-line
// pass condition listed in RATIFY.md so the two cannot silently drift.
type Oracle struct {
	ID    string // "A1".."A6"
	Name  string
	Class OracleClass
	Seed  string // field-report / design provenance, e.g. "#149" or "phase4-agent-handoff.md"
	Pass  string // one-line pass condition
}

// Registry is the canonical list. Keep in sync with RATIFY.md.
var Registry = []Oracle{
	{"A1", "greenfield / spec-consult", Judged, "#149",
		"given a written prose spec, the workflow consumes it and does NOT cold-elicit contract questions (e.g. \"name your load-bearing modules\")."},
	{"A2", "bootstrap / legacy derive", Judged, "ADR-001 (transitional mode)",
		"given an existing repo with no spec, invariant docs are derived from the code, not re-elicited."},
	{"A3", "mode-tagging enforceable", Deterministic, "ADR-001 (operating modes)",
		"every load-bearing module declares a valid add-mode tag in {add, bootstrap, transitional}."},
	{"A4", "diff-classification enforced", Deterministic, "phase4-agent-handoff.md (commit-shape classifier)",
		"a classifier accepts only the three legal commit shapes and rejects anything else."},
	{"A5", "Phase 4 drift-stop", Judged, "phase4-agent-handoff.md (two-tier completion + defer/kill)",
		"when the only path to green weakens invariant I, the agent STOPS and emits a drift packet instead of weakening I."},
	{"A6", "E2E / completeness", Judged, "#61, #60",
		"component-correct verification that misses end-to-end integration FAILS; incomplete verification is never silently treated as sufficient."},
}

// LegalCommitShapes are the three shapes the Phase 4 loop may emit
// (phase4-agent-handoff.md, "Commit-shape classifier").
var LegalCommitShapes = []CommitShape{
	ShapeImplementation, ShapeGovernanceAmendment, ShapeNewInvariant,
}

// CommitShape is one classification result.
type CommitShape string

const (
	ShapeImplementation      CommitShape = "implementation"
	ShapeGovernanceAmendment CommitShape = "governance-amendment"
	ShapeNewInvariant        CommitShape = "new-invariant"
	ShapeIllegal             CommitShape = "" // anything the classifier must reject
)

// GovernanceSubtags are the required sub-tags on a governance-amendment commit.
var GovernanceSubtags = []string{
	"propagated-discovery", "intent-refinement", "drift", "retraction",
}

// ClassifyCommitShape is the A4 seam the Phase 4 build owns. It returns the
// commit shape for a (subject, body) pair, or ShapeIllegal for anything outside
// the three legal shapes. The grammar is the one the Phase 4 agent
// (agents/lowry.md) emits and enforces:
//
//	implementation: <summary>          -> ShapeImplementation
//	new-invariant: <summary>           -> ShapeNewInvariant
//	governance-amendment: <summary>    -> ShapeGovernanceAmendment, but ONLY if the
//	                                      body carries `amendment-kind: <kind>` with
//	                                      kind in GovernanceSubtags; otherwise illegal.
//
// Rejection is signalled by ShapeIllegal with a nil error — the error return is
// reserved for genuine evaluation faults, so callers can distinguish "rejected"
// from "errored".
func ClassifyCommitShape(subject, body string) (CommitShape, error) {
	subject = strings.TrimSpace(subject)
	switch {
	case strings.HasPrefix(subject, string(ShapeImplementation)+":"):
		return ShapeImplementation, nil
	case strings.HasPrefix(subject, string(ShapeNewInvariant)+":"):
		return ShapeNewInvariant, nil
	case strings.HasPrefix(subject, string(ShapeGovernanceAmendment)+":"):
		if governanceAmendmentKind(body) != "" {
			return ShapeGovernanceAmendment, nil
		}
		return ShapeIllegal, nil
	default:
		return ShapeIllegal, nil
	}
}

// governanceAmendmentKind returns the valid amendment-kind declared in a commit
// body, or "" if none of GovernanceSubtags is present. A governance-amendment
// commit MUST carry exactly one.
func governanceAmendmentKind(body string) string {
	const prefix = "amendment-kind:"
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		for _, valid := range GovernanceSubtags {
			if val == valid {
				return val
			}
		}
	}
	return ""
}

// JudgedRun is the transcript+verdict a judged oracle consumes once the
// scenario runner + LLM judge exist.
type JudgedRun struct {
	Scenario   string
	Transcript string
	Verdict    string // "pass" | "fail" | ""
}

// RunJudged is the seam the future judged-oracle harness fills. Until then it
// returns ErrPendingRatification so the check is runnable and RED by design.
func RunJudged(scenarioPath string) (JudgedRun, error) {
	return JudgedRun{Scenario: scenarioPath}, ErrPendingRatification
}

// ── repo helpers (deterministic oracles) ────────────────────────────────────

// crosscheckRoot climbs from the test's CWD to the crosscheck/ root, found as
// the first ancestor holding both skills/ and agents/.
func crosscheckRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 8; i++ {
		if dirExists(filepath.Join(dir, "skills")) && dirExists(filepath.Join(dir, "agents")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("could not locate crosscheck/ root (an ancestor with skills/ and agents/)")
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// loadBearingModules returns the SKILL.md and agent .md files that carry
// behavior — the modules a mode tag would have to live on.
func loadBearingModules(root string) ([]string, error) {
	var mods []string
	skills := filepath.Join(root, "skills")
	entries, err := os.ReadDir(skills)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				p := filepath.Join(skills, e.Name(), "SKILL.md")
				if fileExists(p) {
					mods = append(mods, p)
				}
			}
		}
	}
	agents := filepath.Join(root, "agents")
	aents, err := os.ReadDir(agents)
	if err == nil {
		for _, e := range aents {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				mods = append(mods, filepath.Join(agents, e.Name()))
			}
		}
	}
	sort.Strings(mods)
	if len(mods) == 0 {
		return nil, errors.New("no load-bearing modules found under skills/ or agents/")
	}
	return mods, nil
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// frontmatterValue extracts a `key: value` from a YAML frontmatter block.
// Returns "" if absent. Deliberately tiny — no YAML dependency.
func frontmatterValue(path, key string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := string(b)
	if !strings.HasPrefix(s, "---") {
		return ""
	}
	rest := s[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, key+":"))
		}
	}
	return ""
}

// scenarioPath resolves a seed scenario fixture next to this package.
func scenarioPath(name string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "scenarios", name)
	if !fileExists(p) {
		return "", errors.New("missing seed scenario fixture: " + p)
	}
	return p, nil
}
