package orchestrator

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// --- SelectPattern tests ---

func TestSelectPattern(t *testing.T) {
	tests := []struct {
		name string
		attr MissionAttributes
		want Pattern
	}{
		{
			name: "single file single domain returns sequential",
			attr: MissionAttributes{FileCount: 1, DomainCount: 1, ToolCount: 1, EstimatedComplexity: "low"},
			want: PatternSequential,
		},
		{
			name: "zero files zero domains returns sequential",
			attr: MissionAttributes{FileCount: 0, DomainCount: 0, ToolCount: 0, EstimatedComplexity: "low"},
			want: PatternSequential,
		},
		{
			name: "high complexity many tools returns orchestrator-workers",
			attr: MissionAttributes{FileCount: 10, DomainCount: 3, ToolCount: 5, EstimatedComplexity: "high"},
			want: PatternOrchestratorWorkers,
		},
		{
			name: "multi-domain many files returns parallel",
			attr: MissionAttributes{FileCount: 10, DomainCount: 2, ToolCount: 2, EstimatedComplexity: "medium"},
			want: PatternParallel,
		},
		{
			name: "multi-domain few files returns handoff",
			attr: MissionAttributes{FileCount: 3, DomainCount: 2, ToolCount: 1, EstimatedComplexity: "low"},
			want: PatternHandoff,
		},
		{
			name: "medium complexity single domain many files returns sequential",
			attr: MissionAttributes{FileCount: 10, DomainCount: 1, ToolCount: 2, EstimatedComplexity: "medium"},
			want: PatternSequential,
		},
		{
			name: "high complexity few tools multi-domain returns parallel",
			attr: MissionAttributes{FileCount: 8, DomainCount: 3, ToolCount: 2, EstimatedComplexity: "high"},
			want: PatternParallel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectPattern(tt.attr)
			if got != tt.want {
				t.Fatalf("SelectPattern(%+v) = %s, want %s", tt.attr, got, tt.want)
			}
		})
	}
}

// --- Pattern and AgentStatus String tests ---

func TestPatternString(t *testing.T) {
	tests := []struct {
		p    Pattern
		want string
	}{
		{PatternSequential, "sequential"},
		{PatternParallel, "parallel"},
		{PatternOrchestratorWorkers, "orchestrator-workers"},
		{PatternHandoff, "handoff"},
		{Pattern(99), "unknown(99)"},
	}
	for _, tt := range tests {
		if got := tt.p.String(); got != tt.want {
			t.Errorf("Pattern(%d).String() = %q, want %q", int(tt.p), got, tt.want)
		}
	}
}

func TestAgentStatusString(t *testing.T) {
	tests := []struct {
		s    AgentStatus
		want string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusTimedOut, "timed_out"},
		{AgentStatus(99), "unknown(99)"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("AgentStatus(%d).String() = %q, want %q", int(tt.s), got, tt.want)
		}
	}
}

// --- NewOrchestrator tests ---

func TestNewOrchestratorDefaultSummaryChars(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	if o.config.SummaryMaxChars != DefaultSummaryMaxChars {
		t.Fatalf("expected default SummaryMaxChars %d, got %d", DefaultSummaryMaxChars, o.config.SummaryMaxChars)
	}
}

func TestNewOrchestratorCustomSummaryChars(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{SummaryMaxChars: 500})
	if o.config.SummaryMaxChars != 500 {
		t.Fatalf("expected SummaryMaxChars 500, got %d", o.config.SummaryMaxChars)
	}
}

// --- AddAgent tests ---

func TestAddAgentSuccess(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{MaxSubAgents: 5})
	if err := o.AddAgent("a1", "task-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(o.topology.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(o.topology.Nodes))
	}
	if o.topology.Nodes[0].Status != StatusPending {
		t.Errorf("new agent should be Pending, got %s", o.topology.Nodes[0].Status)
	}
}

func TestAddAgentEmptyID(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	err := o.AddAgent("", "task")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestAddAgentDuplicateID(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "task-1")
	err := o.AddAgent("a1", "task-2")
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention duplicate, got: %v", err)
	}
}

func TestAddAgentMaxReached(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{MaxSubAgents: 2})
	_ = o.AddAgent("a1", "task-1")
	_ = o.AddAgent("a2", "task-2")
	err := o.AddAgent("a3", "task-3")
	if err == nil {
		t.Fatal("expected error when max sub-agents reached")
	}
	if !strings.Contains(err.Error(), "max") {
		t.Errorf("error should mention max, got: %v", err)
	}
}

func TestAddAgentNoLimitWhenZero(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{MaxSubAgents: 0})
	for i := 0; i < 100; i++ {
		if err := o.AddAgent(fmt.Sprintf("agent-%d", i), "t"); err != nil {
			t.Fatalf("unexpected error adding agent %d: %v", i, err)
		}
	}
}

// --- AddEdge tests ---

func TestAddEdgeSuccess(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	_ = o.AddAgent("a2", "t2")
	if err := o.AddEdge("a1", "a2", "depends"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(o.topology.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(o.topology.Edges))
	}
}

func TestAddEdgeUnknownSource(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	err := o.AddEdge("unknown", "a1", "depends")
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestAddEdgeUnknownTarget(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	err := o.AddEdge("a1", "unknown", "depends")
	if err == nil {
		t.Fatal("expected error for unknown target")
	}
}

func TestAddEdgeSelfLoop(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	err := o.AddEdge("a1", "a1", "depends")
	if err == nil {
		t.Fatal("expected error for self-loop")
	}
}

func TestAddEdgeDuplicate(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	_ = o.AddAgent("a2", "t2")
	if err := o.AddEdge("a1", "a2", "depends"); err != nil {
		t.Fatalf("first AddEdge failed: %v", err)
	}
	// Adding the exact same edge again is silently accepted (no dedup).
	// The current implementation does not reject duplicate edges; this test
	// documents that behavior. Both copies appear in the topology.
	if err := o.AddEdge("a1", "a2", "depends"); err != nil {
		t.Fatalf("duplicate AddEdge unexpectedly failed: %v", err)
	}
	topo := o.GetTopology()
	if len(topo.Edges) != 2 {
		t.Fatalf("expected 2 edges (duplicate accepted), got %d", len(topo.Edges))
	}
}

func TestAddEdgeCycleDetected(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	_ = o.AddAgent("a2", "t2")
	_ = o.AddAgent("a3", "t3")
	_ = o.AddEdge("a1", "a2", "depends")
	_ = o.AddEdge("a2", "a3", "depends")
	err := o.AddEdge("a3", "a1", "depends")
	if err == nil {
		t.Fatal("expected error for cycle")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention cycle, got: %v", err)
	}
}

// --- UpdateAgent tests ---

func TestUpdateAgentSuccess(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	err := o.UpdateAgent("a1", StatusRunning, 100, time.Second, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slot := o.Metrics()["a1"]
	if slot.Status != StatusRunning {
		t.Errorf("expected Running, got %s", slot.Status)
	}
	if slot.TokensUsed != 100 {
		t.Errorf("expected 100 tokens, got %d", slot.TokensUsed)
	}
	if slot.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
}

func TestUpdateAgentSetsEndedAt(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	_ = o.UpdateAgent("a1", StatusCompleted, 200, 2*time.Second, "")
	slot := o.Metrics()["a1"]
	if slot.EndedAt == nil {
		t.Fatal("expected EndedAt to be set for completed agent")
	}
}

func TestUpdateAgentFailed(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	_ = o.UpdateAgent("a1", StatusFailed, 50, time.Second, "out of memory")
	slot := o.Metrics()["a1"]
	if slot.Status != StatusFailed {
		t.Errorf("expected Failed, got %s", slot.Status)
	}
	if slot.Error != "out of memory" {
		t.Errorf("expected error 'out of memory', got %q", slot.Error)
	}
	if slot.EndedAt == nil {
		t.Error("expected EndedAt to be set for failed agent")
	}
}

func TestUpdateAgentTimedOut(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	_ = o.UpdateAgent("a1", StatusTimedOut, 0, 30*time.Second, "deadline exceeded")
	slot := o.Metrics()["a1"]
	if slot.Status != StatusTimedOut {
		t.Errorf("expected TimedOut, got %s", slot.Status)
	}
	if slot.EndedAt == nil {
		t.Error("expected EndedAt to be set for timed-out agent")
	}
}

func TestUpdateAgentUnknown(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	err := o.UpdateAgent("nope", StatusRunning, 0, 0, "")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

// --- SetResult / GetResult tests ---

func TestSetResultTruncatesSummary(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{SummaryMaxChars: 10})
	_ = o.AddAgent("a1", "t1")
	err := o.SetResult(SubAgentResult{
		AgentID: "a1",
		Summary: "this is a very long summary that exceeds the limit",
		Success: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := o.GetResult("a1")
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if len(r.Summary) > 10 {
		t.Errorf("summary should be truncated to 10, got %d chars", len(r.Summary))
	}
}

func TestSetResultUnknownAgent(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	err := o.SetResult(SubAgentResult{AgentID: "nope"})
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestGetResultNil(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	if r := o.GetResult("a1"); r != nil {
		t.Fatalf("expected nil result, got %+v", r)
	}
}

// --- ActiveAgents / CompletedAgents / FailedAgents tests ---

func TestAgentFilterFunctions(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	_ = o.AddAgent("a2", "t2")
	_ = o.AddAgent("a3", "t3")
	_ = o.AddAgent("a4", "t4")
	_ = o.UpdateAgent("a1", StatusRunning, 0, 0, "")
	_ = o.UpdateAgent("a2", StatusCompleted, 0, 0, "")
	_ = o.UpdateAgent("a3", StatusFailed, 0, 0, "err")
	_ = o.UpdateAgent("a4", StatusTimedOut, 0, 0, "timeout")

	if got := len(o.ActiveAgents()); got != 1 {
		t.Errorf("ActiveAgents: expected 1, got %d", got)
	}
	if got := len(o.CompletedAgents()); got != 1 {
		t.Errorf("CompletedAgents: expected 1, got %d", got)
	}
	// FailedAgents includes both Failed and TimedOut
	if got := len(o.FailedAgents()); got != 2 {
		t.Errorf("FailedAgents: expected 2, got %d", got)
	}
}

// --- TruncateSummary tests ---

func TestTruncateSummary(t *testing.T) {
	tests := []struct {
		name      string
		summary   string
		maxChars int
		wantLen   int
	}{
		{
			name:      "short summary unchanged",
			summary:   "hello",
			maxChars: 100,
			wantLen:   5,
		},
		{
			name:      "exact length unchanged",
			summary:   "abcde",
			maxChars: 5,
			wantLen:   5,
		},
		{
			name:      "long summary truncated",
			summary:   strings.Repeat("x", 3000),
			maxChars: 2000,
			wantLen:   2000,
		},
		{
			name:      "zero maxChars uses default",
			summary:   strings.Repeat("x", 3000),
			maxChars: 0,
			wantLen:   DefaultSummaryMaxChars,
		},
		{
			name:      "negative maxChars uses default",
			summary:   strings.Repeat("x", 3000),
			maxChars: -1,
			wantLen:   DefaultSummaryMaxChars,
		},
		{
			name:      "empty summary",
			summary:   "",
			maxChars: 100,
			wantLen:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateSummary(tt.summary, tt.maxChars)
			if len(got) != tt.wantLen {
				t.Fatalf("TruncateSummary(len=%d, %d) length = %d, want %d", len(tt.summary), tt.maxChars, len(got), tt.wantLen)
			}
		})
	}
}

// --- ValidateTopology tests ---

func TestValidateTopologyNil(t *testing.T) {
	if err := ValidateTopology(nil); err == nil {
		t.Fatal("expected error for nil topology")
	}
}

func TestValidateTopologyEmpty(t *testing.T) {
	topo := &AgentTopology{Nodes: nil, Edges: nil}
	if err := ValidateTopology(topo); err != nil {
		t.Fatalf("expected no error for empty topology, got: %v", err)
	}
}

func TestValidateTopologyDuplicateID(t *testing.T) {
	topo := &AgentTopology{
		Nodes: []AgentSlot{{ID: "a1"}, {ID: "a1"}},
	}
	err := ValidateTopology(topo)
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention duplicate, got: %v", err)
	}
}

func TestValidateTopologyUnknownEdgeSource(t *testing.T) {
	topo := &AgentTopology{
		Nodes: []AgentSlot{{ID: "a1"}},
		Edges: []Edge{{From: "unknown", To: "a1", Type: "dep"}},
	}
	err := ValidateTopology(topo)
	if err == nil {
		t.Fatal("expected error for unknown edge source")
	}
}

func TestValidateTopologyUnknownEdgeTarget(t *testing.T) {
	topo := &AgentTopology{
		Nodes: []AgentSlot{{ID: "a1"}},
		Edges: []Edge{{From: "a1", To: "unknown", Type: "dep"}},
	}
	err := ValidateTopology(topo)
	if err == nil {
		t.Fatal("expected error for unknown edge target")
	}
}

func TestValidateTopologyCycle(t *testing.T) {
	topo := &AgentTopology{
		Nodes: []AgentSlot{{ID: "a1"}, {ID: "a2"}, {ID: "a3"}},
		Edges: []Edge{
			{From: "a1", To: "a2", Type: "dep"},
			{From: "a2", To: "a3", Type: "dep"},
			{From: "a3", To: "a1", Type: "dep"},
		},
	}
	err := ValidateTopology(topo)
	if err == nil {
		t.Fatal("expected error for cycle")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention cycle, got: %v", err)
	}
}

func TestValidateTopologyOrphan(t *testing.T) {
	topo := &AgentTopology{
		Nodes: []AgentSlot{{ID: "a1"}, {ID: "a2"}, {ID: "orphan"}},
		Edges: []Edge{{From: "a1", To: "a2", Type: "dep"}},
	}
	err := ValidateTopology(topo)
	if err == nil {
		t.Fatal("expected error for orphan agent")
	}
	if !strings.Contains(err.Error(), "orphan") {
		t.Errorf("error should mention orphan, got: %v", err)
	}
}

func TestValidateTopologyValid(t *testing.T) {
	topo := &AgentTopology{
		Nodes: []AgentSlot{{ID: "a1"}, {ID: "a2"}, {ID: "a3"}},
		Edges: []Edge{
			{From: "a1", To: "a2", Type: "dep"},
			{From: "a2", To: "a3", Type: "dep"},
		},
	}
	if err := ValidateTopology(topo); err != nil {
		t.Fatalf("expected valid topology, got error: %v", err)
	}
}

// --- NewCommunicationFile tests ---

func TestNewCommunicationFile(t *testing.T) {
	before := time.Now()
	cf := NewCommunicationFile("agent-a", "agent-b", "result", "/tmp/msg.json")
	after := time.Now()

	if cf.From != "agent-a" {
		t.Errorf("From = %q, want %q", cf.From, "agent-a")
	}
	if cf.To != "agent-b" {
		t.Errorf("To = %q, want %q", cf.To, "agent-b")
	}
	if cf.Type != "result" {
		t.Errorf("Type = %q, want %q", cf.Type, "result")
	}
	if cf.FilePath != "/tmp/msg.json" {
		t.Errorf("FilePath = %q, want %q", cf.FilePath, "/tmp/msg.json")
	}
	if cf.CreatedAt.Before(before) || cf.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v not between %v and %v", cf.CreatedAt, before, after)
	}
}

// --- Metrics tests ---

func TestMetricsReturnsAllAgents(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	_ = o.AddAgent("a2", "t2")
	_ = o.UpdateAgent("a1", StatusRunning, 100, time.Second, "")
	_ = o.UpdateAgent("a2", StatusCompleted, 200, 2*time.Second, "")

	m := o.Metrics()
	if len(m) != 2 {
		t.Fatalf("expected 2 entries in metrics, got %d", len(m))
	}
	if m["a1"].TokensUsed != 100 {
		t.Errorf("a1 tokens = %d, want 100", m["a1"].TokensUsed)
	}
	if m["a2"].TokensUsed != 200 {
		t.Errorf("a2 tokens = %d, want 200", m["a2"].TokensUsed)
	}
}

// --- GetTopology tests ---

func TestGetTopologyReturnsCopy(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{})
	_ = o.AddAgent("a1", "t1")
	_ = o.AddAgent("a2", "t2")
	_ = o.AddEdge("a1", "a2", "dep")

	topo := o.GetTopology()

	// Mutate the returned topology.
	topo.Nodes = append(topo.Nodes, AgentSlot{ID: "injected"})
	topo.Edges = append(topo.Edges, Edge{From: "a2", To: "a1", Type: "bad"})
	topo.Pattern = PatternHandoff

	// Verify the orchestrator's internal state is unchanged.
	internal := o.GetTopology()
	if len(internal.Nodes) != 2 {
		t.Fatalf("expected 2 internal nodes, got %d", len(internal.Nodes))
	}
	if len(internal.Edges) != 1 {
		t.Fatalf("expected 1 internal edge, got %d", len(internal.Edges))
	}
	if internal.Pattern != PatternSequential {
		t.Fatalf("expected default pattern (sequential), got %s", internal.Pattern)
	}
}

// --- HandleFailure tests ---

func TestHandleFailureRetryPolicy(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{FailurePolicy: "retry"})
	_ = o.AddAgent("a1", "task-1")
	_ = o.UpdateAgent("a1", StatusFailed, 100, time.Second, "boom")

	action, err := o.HandleFailure("a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionRetry {
		t.Fatalf("expected ActionRetry, got %s", action)
	}
}

func TestHandleFailureContinuePolicy(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{FailurePolicy: "continue"})
	_ = o.AddAgent("a1", "task-1")
	_ = o.UpdateAgent("a1", StatusFailed, 50, time.Second, "error")

	action, err := o.HandleFailure("a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionSkip {
		t.Fatalf("expected ActionSkip, got %s", action)
	}
}

func TestHandleFailureFailFastPolicy(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{FailurePolicy: "fail-fast"})
	_ = o.AddAgent("a1", "task-1")
	_ = o.UpdateAgent("a1", StatusTimedOut, 0, 30*time.Second, "deadline exceeded")

	action, err := o.HandleFailure("a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionEscalate {
		t.Fatalf("expected ActionEscalate, got %s", action)
	}
}

func TestHandleFailureDefaultPolicy(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{FailurePolicy: "unknown-policy"})
	_ = o.AddAgent("a1", "task-1")
	_ = o.UpdateAgent("a1", StatusFailed, 0, 0, "err")

	action, err := o.HandleFailure("a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionEscalate {
		t.Fatalf("expected ActionEscalate for unknown policy, got %s", action)
	}
}

func TestHandleFailureUnknownAgent(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{FailurePolicy: "retry"})
	_, err := o.HandleFailure("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("error should mention unknown agent, got: %v", err)
	}
}

func TestHandleFailureNotFailed(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{FailurePolicy: "retry"})
	_ = o.AddAgent("a1", "task-1")
	_ = o.UpdateAgent("a1", StatusRunning, 0, 0, "")

	_, err := o.HandleFailure("a1")
	if err == nil {
		t.Fatal("expected error for non-failed agent")
	}
	if !strings.Contains(err.Error(), "not in a failed state") {
		t.Errorf("error should mention not in a failed state, got: %v", err)
	}

	// Also test with completed status.
	_ = o.UpdateAgent("a1", StatusCompleted, 100, time.Second, "")
	_, err = o.HandleFailure("a1")
	if err == nil {
		t.Fatal("expected error for completed agent")
	}
}

// --- BuildFailureReport tests ---

func TestBuildFailureReportBasic(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{FailurePolicy: "retry"})
	_ = o.AddAgent("a1", "task-1")
	_ = o.UpdateAgent("a1", StatusFailed, 150, 5*time.Second, "out of memory")

	report, err := o.BuildFailureReport("a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.AgentID != "a1" {
		t.Errorf("AgentID = %q, want %q", report.AgentID, "a1")
	}
	if report.Task != "task-1" {
		t.Errorf("Task = %q, want %q", report.Task, "task-1")
	}
	if report.Error != "out of memory" {
		t.Errorf("Error = %q, want %q", report.Error, "out of memory")
	}
	if report.Status != StatusFailed {
		t.Errorf("Status = %s, want %s", report.Status, StatusFailed)
	}
	if report.TokensUsed != 150 {
		t.Errorf("TokensUsed = %d, want 150", report.TokensUsed)
	}
	if report.WallClock != 5*time.Second {
		t.Errorf("WallClock = %v, want %v", report.WallClock, 5*time.Second)
	}
	if report.Action != ActionRetry {
		t.Errorf("Action = %s, want %s", report.Action, ActionRetry)
	}
	if len(report.CompletedDeps) != 0 {
		t.Errorf("CompletedDeps should be empty, got %v", report.CompletedDeps)
	}
	if len(report.FailedDeps) != 0 {
		t.Errorf("FailedDeps should be empty, got %v", report.FailedDeps)
	}
}

func TestBuildFailureReportWithDeps(t *testing.T) {
	o := NewOrchestrator(OrchestratorConfig{FailurePolicy: "continue"})
	_ = o.AddAgent("dep-ok", "upstream-ok")
	_ = o.AddAgent("dep-fail", "upstream-fail")
	_ = o.AddAgent("target", "main-task")
	_ = o.AddEdge("dep-ok", "target", "depends")
	_ = o.AddEdge("dep-fail", "target", "depends")
	_ = o.UpdateAgent("dep-ok", StatusCompleted, 100, time.Second, "")
	_ = o.UpdateAgent("dep-fail", StatusFailed, 50, time.Second, "upstream error")
	_ = o.UpdateAgent("target", StatusFailed, 0, 2*time.Second, "dependency failure")

	report, err := o.BuildFailureReport("target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Action != ActionSkip {
		t.Errorf("Action = %s, want %s", report.Action, ActionSkip)
	}
	if len(report.CompletedDeps) != 1 || report.CompletedDeps[0] != "dep-ok" {
		t.Errorf("CompletedDeps = %v, want [dep-ok]", report.CompletedDeps)
	}
	if len(report.FailedDeps) != 1 || report.FailedDeps[0] != "dep-fail" {
		t.Errorf("FailedDeps = %v, want [dep-fail]", report.FailedDeps)
	}
}

// --- FailureAction String tests ---

func TestFailureActionString(t *testing.T) {
	tests := []struct {
		a    FailureAction
		want string
	}{
		{ActionRetry, "retry"},
		{ActionSkip, "skip"},
		{ActionEscalate, "escalate"},
		{FailureAction(99), "unknown(99)"},
	}
	for _, tt := range tests {
		if got := tt.a.String(); got != tt.want {
			t.Errorf("FailureAction(%d).String() = %q, want %q", int(tt.a), got, tt.want)
		}
	}
}
