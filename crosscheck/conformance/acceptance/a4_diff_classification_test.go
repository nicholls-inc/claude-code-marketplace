//go:build acceptance

package acceptance

import "testing"

// A4 — diff-classification enforced (DETERMINISTIC).
//
// Seed: phase4-agent-handoff.md ("Commit-shape classifier"). Pure function over
// commit grammar.
// Contract: a classifier accepts ONLY the three legal commit shapes —
// implementation, governance-amendment{propagated-discovery|intent-refinement|
// drift|retraction}, new-invariant — and rejects anything else.
//
// Currently RED by design: ClassifyCommitShape is an unimplemented seam owned by
// the Phase 4 build and returns ErrCapabilityAbsent for every input.
func TestA4DiffClassificationEnforced(t *testing.T) {
	o := Registry[3]

	cases := []struct {
		name    string
		subject string
		body    string
		want    CommitShape // ShapeIllegal means "must be rejected"
	}{
		{"impl", "implementation: drive parser to green", "", ShapeImplementation},
		{"gov-drift", "governance-amendment: weaken I3", "amendment-kind: drift", ShapeGovernanceAmendment},
		{"gov-refine", "governance-amendment: tighten I1", "amendment-kind: intent-refinement", ShapeGovernanceAmendment},
		{"new-inv", "new-invariant: add I7 for cache module", "", ShapeNewInvariant},
		{"illegal-bare", "fix stuff", "", ShapeIllegal},
		{"illegal-gov-no-subtag", "governance-amendment: change I2", "", ShapeIllegal},
	}

	var failures int
	for _, c := range cases {
		got, err := ClassifyCommitShape(c.subject, c.body)
		if err != nil {
			failures++
			continue
		}
		if got != c.want {
			failures++
		}
	}
	if failures > 0 {
		t.Fatalf("A4 FAIL — %d/%d cases unclassified (commit-shape classifier unimplemented; Phase 4 owns it)\n  pass condition: %s",
			failures, len(cases), o.Pass)
	}
}
