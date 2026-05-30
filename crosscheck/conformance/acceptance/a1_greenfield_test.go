//go:build acceptance

package acceptance

import "testing"

// A1 — greenfield / spec-consult (JUDGED).
//
// Seed: field report #149.
// Contract: given an empty-or-spec'd repo plus a written prose spec, the
// workflow CONSUMES the existing written spec and does NOT cold-elicit contract
// questions ("name your load-bearing modules"). This is the ADR-001 `add` mode
// behaviour: there is a signed-off spec, so consume — don't re-elicit.
//
// Currently RED by design: no judged-oracle harness (scenario runner + LLM
// judge) ships, so RunJudged returns ErrPendingRatification.
func TestA1GreenfieldSpecConsult(t *testing.T) {
	o := Registry[0]
	if o.ID != "A1" {
		t.Fatalf("registry drift: expected A1, got %s", o.ID)
	}
	scn, err := scenarioPath("a1_greenfield.md")
	if err != nil {
		t.Fatalf("A1 seed (%s) fixture: %v", o.Seed, err)
	}
	run, err := RunJudged(scn)
	if err != nil {
		t.Fatalf("A1 [%s, seed %s] PENDING — %v\n  pass condition: %s",
			o.Class, o.Seed, err, o.Pass)
	}
	if run.Verdict != "pass" {
		t.Fatalf("A1 FAIL — judge verdict %q, want \"pass\"\n  pass condition: %s",
			run.Verdict, o.Pass)
	}
}
