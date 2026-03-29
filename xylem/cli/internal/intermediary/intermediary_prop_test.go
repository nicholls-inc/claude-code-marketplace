package intermediary

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// intentGen returns a rapid generator for valid intents.
func intentGen() *rapid.Generator[Intent] {
	return rapid.Custom(func(t *rapid.T) Intent {
		return Intent{
			Action:        rapid.StringMatching(`[a-z]+\.[a-z]+`).Draw(t, "action"),
			Resource:      rapid.StringMatching(`/[a-z]+(/[a-z]+)*`).Draw(t, "resource"),
			AgentID:       rapid.StringMatching(`agent-[0-9]+`).Draw(t, "agentID"),
			Justification: rapid.String().Draw(t, "justification"),
		}
	})
}

// ruleGen returns a rapid generator for policy rules.
func ruleGen() *rapid.Generator[Rule] {
	return rapid.Custom(func(t *rapid.T) Rule {
		effect := rapid.SampledFrom([]Effect{Allow, Deny, RequireApproval}).Draw(t, "effect")
		action := rapid.SampledFrom([]string{"*", "file.*", "net.*", "deploy.*"}).Draw(t, "ruleAction")
		resource := rapid.SampledFrom([]string{"*", "/tmp/*", "/etc/*"}).Draw(t, "ruleResource")
		return Rule{Action: action, Resource: resource, Effect: effect}
	})
}

// policyGen returns a rapid generator for policies.
func policyGen() *rapid.Generator[Policy] {
	return rapid.Custom(func(t *rapid.T) Policy {
		rules := rapid.SliceOfN(ruleGen(), 1, 5).Draw(t, "rules")
		name := rapid.StringMatching(`policy-[a-z]+`).Draw(t, "policyName")
		return Policy{Name: name, Rules: rules}
	})
}

// mustTempDir creates a temp directory. The caller must defer os.RemoveAll.
func mustTempDir(t *rapid.T) string {
	dir, err := os.MkdirTemp("", "intermediary-prop-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	return dir
}

func TestProp_EvaluateIsDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policies := rapid.SliceOfN(policyGen(), 0, 3).Draw(t, "policies")
		intent := intentGen().Draw(t, "intent")

		dir := mustTempDir(t)
		defer os.RemoveAll(dir)
		al := NewAuditLog(filepath.Join(dir, "audit.jsonl"))
		inter := NewIntermediary(policies, al, &mockExecutor{})

		r1 := inter.Evaluate(intent)
		r2 := inter.Evaluate(intent)

		if r1.Effect != r2.Effect {
			t.Fatalf("non-deterministic: first=%q second=%q", r1.Effect, r2.Effect)
		}
		if r1.Reason != r2.Reason {
			t.Fatalf("non-deterministic reason: first=%q second=%q", r1.Reason, r2.Reason)
		}
	})
}

func TestProp_DeniedIntentsNeverExecute(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		intent := intentGen().Draw(t, "intent")
		exec := &mockExecutor{}

		// Policy that denies everything.
		policies := []Policy{{
			Name:  "deny-all",
			Rules: []Rule{{Action: "*", Resource: "*", Effect: Deny}},
		}}

		dir := mustTempDir(t)
		defer os.RemoveAll(dir)
		al := NewAuditLog(filepath.Join(dir, "audit.jsonl"))
		inter := NewIntermediary(policies, al, exec)

		effect, _ := inter.Submit(context.Background(), intent)
		if effect != Deny {
			t.Fatalf("expected Deny, got %q", effect)
		}
		if exec.calls() != 0 {
			t.Fatalf("executor called %d times for denied intent", exec.calls())
		}
	})
}

func TestProp_AuditLogLengthEqualsSubmitCount(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "submitCount")
		exec := &mockExecutor{}

		policies := []Policy{{
			Name:  "allow-all",
			Rules: []Rule{{Action: "*", Resource: "*", Effect: Allow}},
		}}

		dir := mustTempDir(t)
		defer os.RemoveAll(dir)
		al := NewAuditLog(filepath.Join(dir, "audit.jsonl"))
		inter := NewIntermediary(policies, al, exec)

		for range n {
			intent := intentGen().Draw(t, "intent")
			_, _ = inter.Submit(context.Background(), intent)
		}

		entries, err := al.Entries()
		if err != nil {
			t.Fatalf("read audit: %v", err)
		}
		if len(entries) != n {
			t.Fatalf("audit entries: got %d, want %d", len(entries), n)
		}
	})
}

func TestProp_ValidateRejectsEmptyActionOrResource(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate intent with empty action.
		intent := Intent{
			Action:   "",
			Resource: rapid.String().Draw(t, "resource"),
			AgentID:  rapid.String().Draw(t, "agentID"),
		}
		if err := ValidateIntent(intent); err == nil {
			t.Fatal("expected error for empty action")
		}

		// Generate intent with empty resource.
		intent = Intent{
			Action:   rapid.StringMatching(`[a-z]+`).Draw(t, "action"),
			Resource: "",
			AgentID:  rapid.String().Draw(t, "agentID2"),
		}
		if err := ValidateIntent(intent); err == nil {
			t.Fatal("expected error for empty resource")
		}
	})
}

func TestProp_GlobStarMatchesEverything(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.StringMatching(`[a-z0-9.]{0,30}`).Draw(t, "value")
		if !MatchGlob("*", value) {
			t.Fatalf("* should match %q", value)
		}
	})
}

func TestProp_GlobExactMatchesSelf(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate strings that are valid glob literals (no special chars).
		value := rapid.StringMatching(`[a-z0-9.]{1,20}`).Draw(t, "value")
		if !MatchGlob(value, value) {
			t.Fatalf("%q should match itself", value)
		}
	})
}

func TestProp_DenyOnlyPolicyDeniesEverything(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Build a policy with only deny rules.
		nRules := rapid.IntRange(1, 5).Draw(t, "nRules")
		rules := make([]Rule, nRules)
		for i := range nRules {
			rules[i] = Rule{Action: "*", Resource: "*", Effect: Deny}
		}
		policies := []Policy{{Name: "deny-only", Rules: rules}}

		dir := mustTempDir(t)
		defer os.RemoveAll(dir)
		al := NewAuditLog(filepath.Join(dir, "audit.jsonl"))
		inter := NewIntermediary(policies, al, &mockExecutor{})

		intent := intentGen().Draw(t, "intent")
		result := inter.Evaluate(intent)
		if result.Effect != Deny {
			t.Fatalf("deny-only policy returned %q for %+v", result.Effect, intent)
		}
	})
}
