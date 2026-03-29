package cost

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// AgentRole identifies the role an agent plays in a mission.
type AgentRole string

const (
	RolePlanner   AgentRole = "planner"
	RoleGenerator AgentRole = "generator"
	RoleEvaluator AgentRole = "evaluator"
	RoleSubAgent  AgentRole = "sub_agent"
)

// Purpose identifies why tokens were consumed.
type Purpose string

const (
	PurposeContext    Purpose = "context"
	PurposeReasoning Purpose = "reasoning"
	PurposeToolCall  Purpose = "tool_call"
	PurposeCompaction Purpose = "compaction"
	PurposeEvaluation Purpose = "evaluation"
)

// UsageRecord captures a single token-usage event.
type UsageRecord struct {
	MissionID    string    `json:"mission_id"`
	AgentRole    AgentRole `json:"agent_role"`
	Purpose      Purpose   `json:"purpose"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	Timestamp    time.Time `json:"timestamp"`
}

// Budget defines token and cost limits for a mission.
type Budget struct {
	TokenLimit   int           `json:"token_limit"`
	CostLimitUSD float64      `json:"cost_limit_usd"`
	Window       time.Duration `json:"window"`
}

// BudgetAlert records a budget threshold event.
type BudgetAlert struct {
	Type      string    `json:"type"` // "warning" or "exceeded"
	Current   float64   `json:"current"`
	Limit     float64   `json:"limit"`
	Timestamp time.Time `json:"timestamp"`
}

// CostReport summarises cost data for a mission.
type CostReport struct {
	MissionID     string             `json:"mission_id"`
	TotalTokens   int                `json:"total_tokens"`
	TotalCostUSD  float64            `json:"total_cost_usd"`
	ByRole        map[AgentRole]float64 `json:"by_role"`
	ByPurpose     map[Purpose]float64   `json:"by_purpose"`
	ByModel       map[string]float64    `json:"by_model"`
	RecordCount   int                `json:"record_count"`
	GeneratedAt   time.Time          `json:"generated_at"`
}

// ModelLadder maps each agent role to a preferred model name.
type ModelLadder struct {
	Roles map[AgentRole]string `json:"roles"`
}

// Anomaly describes a metric that deviates from the historical average.
type Anomaly struct {
	MissionID string  `json:"mission_id"`
	Metric    string  `json:"metric"`
	Expected  float64 `json:"expected"`
	Actual    float64 `json:"actual"`
	Ratio     float64 `json:"ratio"`
}

const (
	warningThreshold  = 0.8
	exceededThreshold = 1.0
	anomalyThreshold  = 2.0
)

// Tracker accumulates usage records and enforces budget constraints.
type Tracker struct {
	mu          sync.Mutex
	budget      *Budget
	records     []UsageRecord
	alerts      []BudgetAlert
	exceeded    bool
	totalCost   float64
	totalTokens int
}

// NewTracker creates a Tracker with an optional budget. Pass nil for no budget.
func NewTracker(budget *Budget) *Tracker {
	return &Tracker{
		budget:  budget,
		records: make([]UsageRecord, 0),
		alerts:  make([]BudgetAlert, 0),
	}
}

// Record adds a usage record and checks budget constraints. It returns an error
// only for invalid input, not for exceeding the budget.
func (t *Tracker) Record(record UsageRecord) error {
	if record.CostUSD < 0 {
		return fmt.Errorf("record usage: cost must be non-negative, got %f", record.CostUSD)
	}
	if record.InputTokens < 0 || record.OutputTokens < 0 {
		return fmt.Errorf("record usage: token counts must be non-negative")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.records = append(t.records, record)
	t.totalCost += record.CostUSD
	t.totalTokens += record.InputTokens + record.OutputTokens

	if t.budget == nil {
		return nil
	}

	totalCost := t.totalCost
	totalTokens := t.totalTokens

	// Check cost budget.
	if t.budget.CostLimitUSD > 0 {
		utilization := totalCost / t.budget.CostLimitUSD
		if utilization >= exceededThreshold && !t.exceeded {
			t.exceeded = true
			t.alerts = append(t.alerts, BudgetAlert{
				Type:      "exceeded",
				Current:   totalCost,
				Limit:     t.budget.CostLimitUSD,
				Timestamp: record.Timestamp,
			})
		} else if utilization >= warningThreshold && !t.exceeded && !t.hasWarningForLimitLocked(t.budget.CostLimitUSD) {
			t.alerts = append(t.alerts, BudgetAlert{
				Type:      "warning",
				Current:   totalCost,
				Limit:     t.budget.CostLimitUSD,
				Timestamp: record.Timestamp,
			})
		}
	}

	// Check token budget.
	if t.budget.TokenLimit > 0 {
		utilization := float64(totalTokens) / float64(t.budget.TokenLimit)
		if utilization >= exceededThreshold && !t.exceeded {
			t.exceeded = true
			t.alerts = append(t.alerts, BudgetAlert{
				Type:      "exceeded",
				Current:   float64(totalTokens),
				Limit:     float64(t.budget.TokenLimit),
				Timestamp: record.Timestamp,
			})
		} else if utilization >= warningThreshold && !t.exceeded && !t.hasWarningForLimitLocked(float64(t.budget.TokenLimit)) {
			t.alerts = append(t.alerts, BudgetAlert{
				Type:      "warning",
				Current:   float64(totalTokens),
				Limit:     float64(t.budget.TokenLimit),
				Timestamp: record.Timestamp,
			})
		}
	}

	return nil
}

// hasWarningForLimitLocked returns true if a warning alert with the given limit
// has already been issued. Must be called with t.mu held.
func (t *Tracker) hasWarningForLimitLocked(limit float64) bool {
	for _, a := range t.alerts {
		if a.Type == "warning" && a.Limit == limit {
			return true
		}
	}
	return false
}

// TotalCost returns the sum of CostUSD across all records.
func (t *Tracker) TotalCost() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalCost
}

// TotalTokens returns the sum of input and output tokens across all records.
func (t *Tracker) TotalTokens() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalTokens
}

// CostByRole returns cost totals grouped by agent role.
func (t *Tracker) CostByRole() map[AgentRole]float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	m := make(map[AgentRole]float64)
	for _, r := range t.records {
		m[r.AgentRole] += r.CostUSD
	}
	return m
}

// CostByPurpose returns cost totals grouped by purpose.
func (t *Tracker) CostByPurpose() map[Purpose]float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	m := make(map[Purpose]float64)
	for _, r := range t.records {
		m[r.Purpose] += r.CostUSD
	}
	return m
}

// CostByModel returns cost totals grouped by model name.
func (t *Tracker) CostByModel() map[string]float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	m := make(map[string]float64)
	for _, r := range t.records {
		m[r.Model] += r.CostUSD
	}
	return m
}

// BudgetExceeded returns true once the budget has been exceeded. Once true it
// never reverts to false (monotonic).
func (t *Tracker) BudgetExceeded() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.exceeded
}

// BudgetUtilization returns the fraction of the cost budget used (0.0 to 1.0+).
// Returns 0 if no budget is set.
func (t *Tracker) BudgetUtilization() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.budget == nil || t.budget.CostLimitUSD <= 0 {
		return 0
	}
	return t.totalCost / t.budget.CostLimitUSD
}

// Alerts returns a copy of all budget alerts in chronological order.
func (t *Tracker) Alerts() []BudgetAlert {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]BudgetAlert, len(t.alerts))
	copy(out, t.alerts)
	return out
}

// Report produces a CostReport for the given mission ID from all tracked records.
func (t *Tracker) Report(missionID string) *CostReport {
	t.mu.Lock()
	defer t.mu.Unlock()

	report := &CostReport{
		MissionID:   missionID,
		ByRole:      make(map[AgentRole]float64),
		ByPurpose:   make(map[Purpose]float64),
		ByModel:     make(map[string]float64),
		GeneratedAt: time.Now(),
	}

	for _, r := range t.records {
		report.TotalTokens += r.InputTokens + r.OutputTokens
		report.TotalCostUSD += r.CostUSD
		report.ByRole[r.AgentRole] += r.CostUSD
		report.ByPurpose[r.Purpose] += r.CostUSD
		report.ByModel[r.Model] += r.CostUSD
		report.RecordCount++
	}

	return report
}

// DefaultModelLadder returns a sensible default model assignment per role.
func DefaultModelLadder() ModelLadder {
	return ModelLadder{
		Roles: map[AgentRole]string{
			RolePlanner:   "claude-sonnet-4-20250514",
			RoleGenerator: "claude-sonnet-4-20250514",
			RoleEvaluator: "claude-haiku-35-20241022",
			RoleSubAgent:  "claude-haiku-35-20241022",
		},
	}
}

// DetectAnomalies compares the current report against a history of reports and
// returns anomalies where the current value exceeds the historical average by
// more than the anomaly threshold (2x).
func DetectAnomalies(current *CostReport, history []*CostReport) []Anomaly {
	if len(history) == 0 || current == nil {
		return nil
	}

	var anomalies []Anomaly

	// Average total cost from history.
	var avgCost float64
	var avgTokens float64
	for _, h := range history {
		avgCost += h.TotalCostUSD
		avgTokens += float64(h.TotalTokens)
	}
	avgCost /= float64(len(history))
	avgTokens /= float64(len(history))

	if avgCost > 0 {
		ratio := current.TotalCostUSD / avgCost
		if ratio > anomalyThreshold {
			anomalies = append(anomalies, Anomaly{
				MissionID: current.MissionID,
				Metric:    "total_cost_usd",
				Expected:  avgCost,
				Actual:    current.TotalCostUSD,
				Ratio:     ratio,
			})
		}
	}

	if avgTokens > 0 {
		ratio := float64(current.TotalTokens) / avgTokens
		if ratio > anomalyThreshold {
			anomalies = append(anomalies, Anomaly{
				MissionID: current.MissionID,
				Metric:    "total_tokens",
				Expected:  avgTokens,
				Actual:    float64(current.TotalTokens),
				Ratio:     ratio,
			})
		}
	}

	// Per-role anomalies.
	roleAvgs := make(map[AgentRole]float64)
	for _, h := range history {
		for role, cost := range h.ByRole {
			roleAvgs[role] += cost
		}
	}
	for role, sum := range roleAvgs {
		avg := sum / float64(len(history))
		if avg > 0 {
			actual, ok := current.ByRole[role]
			if !ok {
				continue
			}
			ratio := actual / avg
			if ratio > anomalyThreshold {
				anomalies = append(anomalies, Anomaly{
					MissionID: current.MissionID,
					Metric:    fmt.Sprintf("role_%s_cost", role),
					Expected:  avg,
					Actual:    actual,
					Ratio:     ratio,
				})
			}
		}
	}

	return anomalies
}

// SaveReport writes a CostReport as JSON to the given file path.
func SaveReport(path string, report *CostReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

// LoadReport reads a CostReport from a JSON file.
func LoadReport(path string) (*CostReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read report: %w", err)
	}
	var report CostReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("unmarshal report: %w", err)
	}
	return &report, nil
}
