package orchestrator

import (
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/cost"
)

// recordTokens records inputTokens to the cost tracker if it is non-nil and
// inputTokens is positive. This centralises UsageRecord construction so that
// UpdateAgent and SetResult share a single code path.
func (o *Orchestrator) recordTokens(inputTokens int) {
	if o.tracker == nil || inputTokens <= 0 {
		return
	}
	_ = o.tracker.Record(cost.UsageRecord{
		MissionID:    o.config.MissionID,
		AgentRole:    cost.RoleSubAgent,
		Purpose:      cost.PurposeReasoning,
		Model:        o.config.DefaultModel,
		InputTokens:  inputTokens,
		OutputTokens: 0,
		Timestamp:    time.Now(),
	})
}

// BudgetExceeded returns true if the cost budget has been exceeded.
// Returns false when no budget is configured (tracker is nil).
func (o *Orchestrator) BudgetExceeded() bool {
	return o.tracker != nil && o.tracker.BudgetExceeded()
}

// CostReport returns a cost report for the current mission, or nil when no
// budget is configured.
func (o *Orchestrator) CostReport() *cost.CostReport {
	if o.tracker == nil {
		return nil
	}
	return o.tracker.Report(o.config.MissionID)
}

// CostAlerts returns all budget alerts, or an empty slice when no budget is
// configured.
func (o *Orchestrator) CostAlerts() []cost.BudgetAlert {
	if o.tracker == nil {
		return []cost.BudgetAlert{}
	}
	return o.tracker.Alerts()
}

// TotalTokenCost returns the cumulative token count recorded by the cost
// tracker, or 0 when no budget is configured.
func (o *Orchestrator) TotalTokenCost() int {
	if o.tracker == nil {
		return 0
	}
	return o.tracker.TotalTokens()
}
