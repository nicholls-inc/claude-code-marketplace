//go:build acceptance

package acceptance

import (
	"strings"
	"testing"
)

// A3 — mode-tagging enforceable (DETERMINISTIC).
//
// Seed: ADR-001 (operating modes). Pure function over repo state.
// Contract: every load-bearing module (skills/<x>/SKILL.md and agents/*.md)
// declares a valid `add-mode` tag in {add, bootstrap, transitional}. ADR-001
// makes the mode a module-level frontmatter tag that skills read and branch on.
//
// Currently RED by design: the mode system is unwired (CLAIM-MODES), so no
// module carries the tag and this lists every one that is missing it.
func TestA3ModeTaggingEnforceable(t *testing.T) {
	o := Registry[2]
	valid := map[string]bool{"add": true, "bootstrap": true, "transitional": true}

	root, err := crosscheckRoot()
	if err != nil {
		t.Fatalf("A3 [%s, seed %s]: %v", o.Class, o.Seed, err)
	}
	mods, err := loadBearingModules(root)
	if err != nil {
		t.Fatalf("A3 [%s, seed %s]: %v", o.Class, o.Seed, err)
	}

	var missing []string
	for _, m := range mods {
		if !valid[frontmatterValue(m, "add-mode")] {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("A3 FAIL — %d/%d module(s) lack a valid add-mode tag (mode system unwired; CLAIM-MODES)\n  pass condition: %s\n  missing:\n  %s",
			len(missing), len(mods), o.Pass, strings.Join(missing, "\n  "))
	}
}
