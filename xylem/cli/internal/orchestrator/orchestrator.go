package orchestrator

import (
	"fmt"
	"time"
	"unicode/utf8"
)

// Pattern describes the orchestration strategy for a multi-agent mission.
type Pattern int

const (
	PatternSequential        Pattern = iota // agents run one after another
	PatternParallel                         // agents run concurrently
	PatternOrchestratorWorkers              // central orchestrator dispatches to workers
	PatternHandoff                          // one agent hands off to the next
)

// String returns the human-readable name for a Pattern.
func (p Pattern) String() string {
	switch p {
	case PatternSequential:
		return "sequential"
	case PatternParallel:
		return "parallel"
	case PatternOrchestratorWorkers:
		return "orchestrator-workers"
	case PatternHandoff:
		return "handoff"
	default:
		return fmt.Sprintf("unknown(%d)", int(p))
	}
}

// AgentStatus tracks the lifecycle of a sub-agent.
type AgentStatus int

const (
	StatusPending   AgentStatus = iota // not yet started
	StatusRunning                      // currently executing
	StatusCompleted                    // finished successfully
	StatusFailed                       // finished with error
	StatusTimedOut                     // exceeded deadline
)

// String returns the human-readable name for an AgentStatus.
func (s AgentStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusTimedOut:
		return "timed_out"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// AgentSlot holds the state and metrics for a single sub-agent.
type AgentSlot struct {
	ID         string
	Task       string
	Status     AgentStatus
	TokensUsed int
	WallClock  time.Duration
	StartedAt  time.Time
	EndedAt    *time.Time
	Error      string
}

// SubAgentResult is the condensed output from a completed sub-agent.
type SubAgentResult struct {
	AgentID    string
	Summary    string
	Artifacts  []string
	Success    bool
	TokensUsed int
}

// Edge represents a dependency between two agents.
type Edge struct {
	From string
	To   string
	Type string
}

// AgentTopology captures the full graph of agents and their relationships.
type AgentTopology struct {
	Nodes   []AgentSlot
	Edges   []Edge
	Pattern Pattern
}

// MissionAttributes describes a mission for pattern selection heuristics.
type MissionAttributes struct {
	Description         string
	FileCount           int
	DomainCount         int
	ToolCount           int
	EstimatedComplexity string // "low", "medium", "high"
}

// OrchestratorConfig controls orchestrator behaviour.
type OrchestratorConfig struct {
	MaxSubAgents    int
	SummaryMaxChars int
	SubAgentTimeout time.Duration
	FailurePolicy   string // "fail-fast", "continue", "retry"
}

// DefaultSummaryMaxChars is used when SummaryMaxChars is zero.
const DefaultSummaryMaxChars = 2000

// CommunicationFile represents a file-based message between agents.
type CommunicationFile struct {
	From      string
	To        string
	Type      string
	FilePath  string
	CreatedAt time.Time
}

// Orchestrator manages sub-agents, tracks topology, and collects results.
// Orchestrator is not safe for concurrent use. Callers must synchronize
// access externally.
type Orchestrator struct {
	config   OrchestratorConfig
	topology *AgentTopology
	results  map[string]*SubAgentResult
	agentIDs map[string]struct{} // fast uniqueness check
}

// SelectPattern chooses an orchestration pattern based on mission attributes.
// The heuristic is deterministic for the same input.
func SelectPattern(attrs MissionAttributes) Pattern {
	switch {
	case attrs.FileCount <= 1 && attrs.DomainCount <= 1:
		return PatternSequential
	case attrs.EstimatedComplexity == "high" && attrs.ToolCount > 3:
		return PatternOrchestratorWorkers
	case attrs.DomainCount > 1 && attrs.FileCount > 5:
		return PatternParallel
	case attrs.DomainCount > 1:
		return PatternHandoff
	default:
		return PatternSequential
	}
}

// NewOrchestrator creates a new Orchestrator with the given configuration.
// If SummaryMaxChars is zero the default (2000) is used.
func NewOrchestrator(config OrchestratorConfig) *Orchestrator {
	if config.SummaryMaxChars <= 0 {
		config.SummaryMaxChars = DefaultSummaryMaxChars
	}
	return &Orchestrator{
		config: config,
		topology: &AgentTopology{
			Nodes: make([]AgentSlot, 0),
			Edges: make([]Edge, 0),
		},
		results:  make(map[string]*SubAgentResult),
		agentIDs: make(map[string]struct{}),
	}
}

// AddAgent registers a new sub-agent. Returns an error if the ID is empty,
// duplicated, or the maximum number of sub-agents has been reached.
func (o *Orchestrator) AddAgent(id, task string) error {
	if id == "" {
		return fmt.Errorf("add agent: id must not be empty")
	}
	if _, exists := o.agentIDs[id]; exists {
		return fmt.Errorf("add agent: duplicate id %q", id)
	}
	if o.config.MaxSubAgents > 0 && len(o.topology.Nodes) >= o.config.MaxSubAgents {
		return fmt.Errorf("add agent: max sub-agents (%d) reached", o.config.MaxSubAgents)
	}
	slot := AgentSlot{
		ID:     id,
		Task:   task,
		Status: StatusPending,
	}
	o.topology.Nodes = append(o.topology.Nodes, slot)
	o.agentIDs[id] = struct{}{}
	return nil
}

// AddEdge adds a dependency edge between two agents. Both agents must already
// be registered and the edge must not create a cycle.
func (o *Orchestrator) AddEdge(from, to, edgeType string) error {
	if _, ok := o.agentIDs[from]; !ok {
		return fmt.Errorf("add edge: unknown source agent %q", from)
	}
	if _, ok := o.agentIDs[to]; !ok {
		return fmt.Errorf("add edge: unknown target agent %q", to)
	}
	if from == to {
		return fmt.Errorf("add edge: self-loop not allowed (%q)", from)
	}

	// Build a candidate slice without aliasing the existing backing array.
	candidate := Edge{From: from, To: to, Type: edgeType}
	edges := make([]Edge, len(o.topology.Edges)+1)
	copy(edges, o.topology.Edges)
	edges[len(edges)-1] = candidate
	if hasCycle(o.agentIDs, edges) {
		return fmt.Errorf("add edge: would create cycle (%s -> %s)", from, to)
	}

	o.topology.Edges = edges
	return nil
}

// UpdateAgent updates the status and metrics for an existing agent.
func (o *Orchestrator) UpdateAgent(id string, status AgentStatus, tokensUsed int, wallClock time.Duration, errMsg string) error {
	for i := range o.topology.Nodes {
		if o.topology.Nodes[i].ID == id {
			o.topology.Nodes[i].Status = status
			o.topology.Nodes[i].TokensUsed = tokensUsed
			o.topology.Nodes[i].WallClock = wallClock
			o.topology.Nodes[i].Error = errMsg

			if status == StatusRunning && o.topology.Nodes[i].StartedAt.IsZero() {
				now := time.Now()
				o.topology.Nodes[i].StartedAt = now
			}
			if status == StatusCompleted || status == StatusFailed || status == StatusTimedOut {
				now := time.Now()
				o.topology.Nodes[i].EndedAt = &now
			}
			return nil
		}
	}
	return fmt.Errorf("update agent: unknown agent %q", id)
}

// SetResult stores a condensed result for a sub-agent. The summary is
// truncated to SummaryMaxChars.
func (o *Orchestrator) SetResult(result SubAgentResult) error {
	if _, ok := o.agentIDs[result.AgentID]; !ok {
		return fmt.Errorf("set result: unknown agent %q", result.AgentID)
	}
	result.Summary = TruncateSummary(result.Summary, o.config.SummaryMaxChars)
	o.results[result.AgentID] = &result
	return nil
}

// GetTopology returns a deep copy of the current agent topology. Callers may
// freely mutate the returned value without affecting the orchestrator's
// internal state.
func (o *Orchestrator) GetTopology() *AgentTopology {
	nodes := make([]AgentSlot, len(o.topology.Nodes))
	copy(nodes, o.topology.Nodes)
	edges := make([]Edge, len(o.topology.Edges))
	copy(edges, o.topology.Edges)
	return &AgentTopology{
		Nodes:   nodes,
		Edges:   edges,
		Pattern: o.topology.Pattern,
	}
}

// GetResult returns the result for a sub-agent, or nil if not yet available.
func (o *Orchestrator) GetResult(agentID string) *SubAgentResult {
	return o.results[agentID]
}

// ActiveAgents returns agents whose status is Running.
func (o *Orchestrator) ActiveAgents() []AgentSlot {
	return o.agentsByStatus(StatusRunning)
}

// CompletedAgents returns agents whose status is Completed.
func (o *Orchestrator) CompletedAgents() []AgentSlot {
	return o.agentsByStatus(StatusCompleted)
}

// FailedAgents returns agents whose status is Failed or TimedOut.
func (o *Orchestrator) FailedAgents() []AgentSlot {
	var out []AgentSlot
	for _, n := range o.topology.Nodes {
		if n.Status == StatusFailed || n.Status == StatusTimedOut {
			out = append(out, n)
		}
	}
	return out
}

// Metrics returns a map of agent ID to AgentSlot for all tracked agents.
func (o *Orchestrator) Metrics() map[string]AgentSlot {
	m := make(map[string]AgentSlot, len(o.topology.Nodes))
	for _, n := range o.topology.Nodes {
		m[n.ID] = n
	}
	return m
}

// agentsByStatus returns all agents matching the given status.
func (o *Orchestrator) agentsByStatus(status AgentStatus) []AgentSlot {
	var out []AgentSlot
	for _, n := range o.topology.Nodes {
		if n.Status == status {
			out = append(out, n)
		}
	}
	return out
}

// TruncateSummary truncates a summary string so its byte length does not
// exceed maxChars. The truncation is rune-aware: it never splits a
// multi-byte UTF-8 character. This is a character-based truncation, not a
// token-based one; callers should size maxChars accordingly (a typical LLM
// token is approximately 4 characters).
func TruncateSummary(summary string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = DefaultSummaryMaxChars
	}
	if len(summary) <= maxChars {
		return summary
	}
	// Walk runes forward, accumulating byte length, and stop before
	// exceeding the budget. Ranging over a string yields rune boundaries,
	// so we can slice by byte offset without splitting a character.
	end := 0
	for i, r := range summary {
		n := utf8.RuneLen(r)
		if i+n > maxChars {
			break
		}
		end = i + n
	}
	return summary[:end]
}

// ValidateTopology checks an AgentTopology for structural problems:
//   - duplicate agent IDs
//   - edges referencing unknown agents
//   - cycles in dependency edges
//   - orphan agents (no edges at all, when there are >1 nodes and edges exist)
func ValidateTopology(topology *AgentTopology) error {
	if topology == nil {
		return fmt.Errorf("validate topology: nil topology")
	}

	ids := make(map[string]struct{}, len(topology.Nodes))
	for _, n := range topology.Nodes {
		if _, dup := ids[n.ID]; dup {
			return fmt.Errorf("validate topology: duplicate agent id %q", n.ID)
		}
		ids[n.ID] = struct{}{}
	}

	for _, e := range topology.Edges {
		if _, ok := ids[e.From]; !ok {
			return fmt.Errorf("validate topology: edge references unknown source %q", e.From)
		}
		if _, ok := ids[e.To]; !ok {
			return fmt.Errorf("validate topology: edge references unknown target %q", e.To)
		}
	}

	if hasCycle(ids, topology.Edges) {
		return fmt.Errorf("validate topology: cycle detected")
	}

	// Check for orphan agents when edges exist.
	if len(topology.Edges) > 0 && len(topology.Nodes) > 1 {
		connected := make(map[string]struct{})
		for _, e := range topology.Edges {
			connected[e.From] = struct{}{}
			connected[e.To] = struct{}{}
		}
		for id := range ids {
			if _, ok := connected[id]; !ok {
				return fmt.Errorf("validate topology: orphan agent %q", id)
			}
		}
	}

	return nil
}

// hasCycle performs a DFS-based cycle check on the directed graph formed by
// edges among the given node IDs.
func hasCycle(ids map[string]struct{}, edges []Edge) bool {
	adj := make(map[string][]string, len(ids))
	for id := range ids {
		adj[id] = nil
	}
	for _, e := range edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	const (
		white = 0 // unvisited
		gray  = 1 // in current path
		black = 2 // fully explored
	)
	color := make(map[string]int, len(ids))

	var dfs func(string) bool
	dfs = func(u string) bool {
		color[u] = gray
		for _, v := range adj[u] {
			if color[v] == gray {
				return true
			}
			if color[v] == white {
				if dfs(v) {
					return true
				}
			}
		}
		color[u] = black
		return false
	}

	for id := range ids {
		if color[id] == white {
			if dfs(id) {
				return true
			}
		}
	}
	return false
}

// NewCommunicationFile creates a CommunicationFile record with the current
// timestamp.
func NewCommunicationFile(from, to, fileType, filePath string) CommunicationFile {
	return CommunicationFile{
		From:      from,
		To:        to,
		Type:      fileType,
		FilePath:  filePath,
		CreatedAt: time.Now(),
	}
}
