package orchestrator

import (
	"fmt"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// --- Property: simple missions always get sequential ---

func TestPropSimpleMissionSequential(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		attrs := MissionAttributes{
			FileCount:           rapid.IntRange(0, 1).Draw(t, "files"),
			DomainCount:         rapid.IntRange(0, 1).Draw(t, "domains"),
			ToolCount:           rapid.IntRange(0, 20).Draw(t, "tools"),
			EstimatedComplexity: rapid.SampledFrom([]string{"low", "medium", "high"}).Draw(t, "complexity"),
		}
		p := SelectPattern(attrs)
		if p != PatternSequential {
			t.Fatalf("FileCount<=1 && DomainCount<=1 should yield Sequential, got %s for %+v", p, attrs)
		}
	})
}

// --- Property: high-complexity many-tools missions get orchestrator-workers ---

func TestPropHighComplexityManyToolsOrchestratorWorkers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		attrs := MissionAttributes{
			FileCount:           rapid.IntRange(2, 100).Draw(t, "files"),
			DomainCount:         rapid.IntRange(2, 20).Draw(t, "domains"),
			ToolCount:           rapid.IntRange(4, 20).Draw(t, "tools"),
			EstimatedComplexity: "high",
		}
		p := SelectPattern(attrs)
		if p != PatternOrchestratorWorkers {
			t.Fatalf("high complexity + ToolCount>3 should yield OrchestratorWorkers, got %s for %+v", p, attrs)
		}
	})
}

// --- Property: pattern is always a valid value ---

func TestPropPatternAlwaysValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		attrs := MissionAttributes{
			FileCount:           rapid.IntRange(0, 100).Draw(t, "files"),
			DomainCount:         rapid.IntRange(0, 20).Draw(t, "domains"),
			ToolCount:           rapid.IntRange(0, 20).Draw(t, "tools"),
			EstimatedComplexity: rapid.SampledFrom([]string{"low", "medium", "high"}).Draw(t, "complexity"),
		}
		p := SelectPattern(attrs)
		valid := p == PatternSequential || p == PatternParallel ||
			p == PatternOrchestratorWorkers || p == PatternHandoff
		if !valid {
			t.Fatalf("SelectPattern returned invalid pattern: %s", p)
		}
	})
}

// --- Property: TruncateSummary never exceeds maxTokens ---

func TestPropTruncateSummaryBound(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		summary := rapid.String().Draw(t, "summary")
		maxTokens := rapid.IntRange(1, 10000).Draw(t, "maxTokens")
		result := TruncateSummary(summary, maxTokens)
		if len(result) > maxTokens {
			t.Fatalf("TruncateSummary produced %d chars, max was %d", len(result), maxTokens)
		}
	})
}

// --- Property: TruncateSummary preserves short strings ---

func TestPropTruncatePreservesShort(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		summary := rapid.StringN(0, 50, 50).Draw(t, "summary")
		maxTokens := rapid.IntRange(50, 10000).Draw(t, "maxTokens")
		result := TruncateSummary(summary, maxTokens)
		if result != summary {
			t.Fatalf("short summary was modified: got %q, want %q", result, summary)
		}
	})
}

// --- Property: every added agent appears in topology and metrics ---

func TestPropAgentTracking(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "numAgents")
		o := NewOrchestrator(OrchestratorConfig{})
		ids := make([]string, n)
		for i := 0; i < n; i++ {
			ids[i] = fmt.Sprintf("agent-%d", i)
			if err := o.AddAgent(ids[i], "task"); err != nil {
				t.Fatalf("AddAgent(%q): %v", ids[i], err)
			}
		}
		topo := o.GetTopology()
		if len(topo.Nodes) != n {
			t.Fatalf("topology has %d nodes, expected %d", len(topo.Nodes), n)
		}
		metrics := o.Metrics()
		for _, id := range ids {
			if _, ok := metrics[id]; !ok {
				t.Fatalf("agent %q not found in metrics", id)
			}
		}
	})
}

// --- Property: agent IDs are unique (random IDs, some may collide) ---

func TestPropAgentIDsUnique(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "numAgents")
		o := NewOrchestrator(OrchestratorConfig{})
		attempted := make(map[string]struct{})
		for i := 0; i < n; i++ {
			id := rapid.StringMatching(`[a-z]{1,5}`).Draw(t, fmt.Sprintf("id-%d", i))
			_ = o.AddAgent(id, "task")
			attempted[id] = struct{}{}
		}
		topo := o.GetTopology()
		if len(topo.Nodes) != len(attempted) {
			t.Fatalf("topology has %d nodes, expected %d unique IDs attempted", len(topo.Nodes), len(attempted))
		}
		seen := make(map[string]struct{})
		for _, node := range topo.Nodes {
			if _, dup := seen[node.ID]; dup {
				t.Fatalf("duplicate agent ID %q in topology", node.ID)
			}
			seen[node.ID] = struct{}{}
		}
	})
}

// --- Property: duplicate agent ID always rejected ---

func TestPropDuplicateAgentRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "id")
		o := NewOrchestrator(OrchestratorConfig{})
		_ = o.AddAgent(id, "first")
		err := o.AddAgent(id, "second")
		if err == nil {
			t.Fatalf("expected error adding duplicate ID %q", id)
		}
	})
}

// --- Property: failed agents are captured with error ---

func TestPropFailedAgentsCaptured(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 10).Draw(t, "numAgents")
		o := NewOrchestrator(OrchestratorConfig{})
		failedIDs := make(map[string]string) // id -> error msg
		for i := 0; i < n; i++ {
			id := fmt.Sprintf("agent-%d", i)
			_ = o.AddAgent(id, "task")
			if rapid.Bool().Draw(t, fmt.Sprintf("fail-%d", i)) {
				errMsg := fmt.Sprintf("error-%d", i)
				_ = o.UpdateAgent(id, StatusFailed, 0, time.Second, errMsg)
				failedIDs[id] = errMsg
			} else {
				_ = o.UpdateAgent(id, StatusCompleted, 0, time.Second, "")
			}
		}
		failed := o.FailedAgents()
		if len(failed) != len(failedIDs) {
			t.Fatalf("expected %d failed agents, got %d", len(failedIDs), len(failed))
		}
		for _, f := range failed {
			expectedErr, ok := failedIDs[f.ID]
			if !ok {
				t.Fatalf("agent %q reported as failed but wasn't marked", f.ID)
			}
			if f.Error != expectedErr {
				t.Fatalf("agent %q error = %q, want %q", f.ID, f.Error, expectedErr)
			}
		}
	})
}

// --- Property: DAG with no back-edges passes validation ---

func TestPropDAGTopologyValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 10).Draw(t, "numAgents")
		o := NewOrchestrator(OrchestratorConfig{})
		for i := 0; i < n; i++ {
			_ = o.AddAgent(fmt.Sprintf("a%d", i), "task")
		}
		// Only add forward edges (i -> j where i < j) to guarantee a DAG.
		// Connect all nodes in a chain to avoid orphans.
		for i := 0; i < n-1; i++ {
			_ = o.AddEdge(fmt.Sprintf("a%d", i), fmt.Sprintf("a%d", i+1), "dep")
		}
		if err := ValidateTopology(o.GetTopology()); err != nil {
			t.Fatalf("expected valid DAG topology, got: %v", err)
		}
	})
}

// --- Property: adding a back-edge to a chain creates a cycle ---

func TestPropBackEdgeCreatesCycle(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(3, 10).Draw(t, "numAgents")
		o := NewOrchestrator(OrchestratorConfig{})
		for i := 0; i < n; i++ {
			_ = o.AddAgent(fmt.Sprintf("a%d", i), "task")
		}
		for i := 0; i < n-1; i++ {
			_ = o.AddEdge(fmt.Sprintf("a%d", i), fmt.Sprintf("a%d", i+1), "dep")
		}
		// Adding a back-edge from last to first should fail.
		err := o.AddEdge(fmt.Sprintf("a%d", n-1), "a0", "dep")
		if err == nil {
			t.Fatal("expected cycle error when adding back-edge")
		}
	})
}
