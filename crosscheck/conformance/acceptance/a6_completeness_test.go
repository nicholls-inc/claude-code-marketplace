//go:build acceptance

package acceptance

import "testing"

// A6 — E2E / completeness (JUDGED).
//
// Seed: field reports #61 and #60.
// Contract (two failure modes folded into one oracle):
//   - #61: component-correct verification that misses end-to-end integration
//     must FAIL, not pass — verifying the leaves is not verifying the whole.
//   - #60: incomplete verification must never be silently treated as sufficient
//     — partial coverage has to surface as partial, not green.
//
// Currently RED by design: no judged-oracle harness ships to score a scenario
// run against this rubric.
func TestA6E2ECompleteness(t *testing.T) {
	o := Registry[5]
	if o.ID != "A6" {
		t.Fatalf("registry drift: expected A6, got %s", o.ID)
	}
	scn, err := scenarioPath("a6_completeness.md")
	if err != nil {
		t.Fatalf("A6 seed (%s) fixture: %v", o.Seed, err)
	}
	run, err := RunJudged(scn)
	if err != nil {
		t.Fatalf("A6 [%s, seed %s] PENDING — %v\n  pass condition: %s",
			o.Class, o.Seed, err, o.Pass)
	}
	if run.Verdict != "pass" {
		t.Fatalf("A6 FAIL — judge verdict %q, want \"pass\"\n  pass condition: %s",
			run.Verdict, o.Pass)
	}
}
