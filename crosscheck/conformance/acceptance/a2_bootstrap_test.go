//go:build acceptance

package acceptance

import "testing"

// A2 — bootstrap / legacy derive (JUDGED).
//
// Seed: ADR-001 (transitional mode). Provenance is design, not a field report.
// Contract: given an existing repo with NO spec, invariant docs are DERIVED
// from the code, not re-elicited from the user — the code already encodes the
// intent, so the workflow must read it, not ask for it again.
//
// Currently RED by design: no judged-oracle harness ships; the transitional
// entrypoint is unwired (CLAIM-MODES).
func TestA2BootstrapLegacyDerive(t *testing.T) {
	o := Registry[1]
	if o.ID != "A2" {
		t.Fatalf("registry drift: expected A2, got %s", o.ID)
	}
	scn, err := scenarioPath("a2_bootstrap.md")
	if err != nil {
		t.Fatalf("A2 seed (%s) fixture: %v", o.Seed, err)
	}
	run, err := RunJudged(scn)
	if err != nil {
		t.Fatalf("A2 [%s, seed %s] PENDING — %v\n  pass condition: %s",
			o.Class, o.Seed, err, o.Pass)
	}
	if run.Verdict != "pass" {
		t.Fatalf("A2 FAIL — judge verdict %q, want \"pass\"\n  pass condition: %s",
			run.Verdict, o.Pass)
	}
}
