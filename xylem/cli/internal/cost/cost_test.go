package cost

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	t.Run("nil budget", func(t *testing.T) {
		tr := NewTracker(nil)
		if tr == nil {
			t.Fatal("expected non-nil tracker")
		}
		if tr.BudgetExceeded() {
			t.Fatal("new tracker should not have exceeded budget")
		}
		if tr.TotalCost() != 0 {
			t.Fatal("new tracker should have zero cost")
		}
	})

	t.Run("with budget", func(t *testing.T) {
		b := &Budget{TokenLimit: 1000, CostLimitUSD: 5.0, Window: time.Hour}
		tr := NewTracker(b)
		if tr.BudgetExceeded() {
			t.Fatal("new tracker with budget should not be exceeded")
		}
		if u := tr.BudgetUtilization(); u != 0 {
			t.Fatalf("expected 0 utilization, got %f", u)
		}
	})
}

func TestRecordValidation(t *testing.T) {
	tests := []struct {
		name    string
		record  UsageRecord
		wantErr bool
	}{
		{
			name:    "valid record",
			record:  UsageRecord{CostUSD: 0.5, InputTokens: 100, OutputTokens: 50},
			wantErr: false,
		},
		{
			name:    "zero cost is valid",
			record:  UsageRecord{CostUSD: 0, InputTokens: 0, OutputTokens: 0},
			wantErr: false,
		},
		{
			name:    "negative cost rejected",
			record:  UsageRecord{CostUSD: -1.0, InputTokens: 100, OutputTokens: 50},
			wantErr: true,
		},
		{
			name:    "negative input tokens rejected",
			record:  UsageRecord{CostUSD: 0.5, InputTokens: -1, OutputTokens: 50},
			wantErr: true,
		},
		{
			name:    "negative output tokens rejected",
			record:  UsageRecord{CostUSD: 0.5, InputTokens: 100, OutputTokens: -1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTracker(nil)
			err := tr.Record(tt.record)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTotalCostAndTokens(t *testing.T) {
	tr := NewTracker(nil)
	now := time.Now()

	records := []UsageRecord{
		{CostUSD: 0.10, InputTokens: 100, OutputTokens: 50, Timestamp: now},
		{CostUSD: 0.25, InputTokens: 200, OutputTokens: 100, Timestamp: now},
		{CostUSD: 0.15, InputTokens: 150, OutputTokens: 75, Timestamp: now},
	}

	for _, r := range records {
		if err := tr.Record(r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	wantCost := 0.50
	if got := tr.TotalCost(); !floatEqual(got, wantCost) {
		t.Fatalf("TotalCost() = %f, want %f", got, wantCost)
	}

	wantTokens := 675 // 100+50 + 200+100 + 150+75
	if got := tr.TotalTokens(); got != wantTokens {
		t.Fatalf("TotalTokens() = %d, want %d", got, wantTokens)
	}
}

func TestZeroCostRecordDoesNotChangeTotalCost(t *testing.T) {
	tr := NewTracker(nil)
	now := time.Now()

	if err := tr.Record(UsageRecord{CostUSD: 1.0, InputTokens: 100, OutputTokens: 50, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	before := tr.TotalCost()

	if err := tr.Record(UsageRecord{CostUSD: 0, InputTokens: 0, OutputTokens: 0, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := tr.TotalCost()

	if !floatEqual(before, after) {
		t.Fatalf("zero-cost record changed TotalCost: before=%f, after=%f", before, after)
	}
}

func TestCostByRole(t *testing.T) {
	tr := NewTracker(nil)
	now := time.Now()

	records := []UsageRecord{
		{AgentRole: RolePlanner, CostUSD: 0.30, Timestamp: now},
		{AgentRole: RoleGenerator, CostUSD: 0.50, Timestamp: now},
		{AgentRole: RolePlanner, CostUSD: 0.20, Timestamp: now},
		{AgentRole: RoleEvaluator, CostUSD: 0.10, Timestamp: now},
	}

	for _, r := range records {
		if err := tr.Record(r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	byRole := tr.CostByRole()

	tests := []struct {
		role AgentRole
		want float64
	}{
		{RolePlanner, 0.50},
		{RoleGenerator, 0.50},
		{RoleEvaluator, 0.10},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := byRole[tt.role]; !floatEqual(got, tt.want) {
				t.Fatalf("CostByRole[%s] = %f, want %f", tt.role, got, tt.want)
			}
		})
	}
}

func TestCostByPurpose(t *testing.T) {
	tr := NewTracker(nil)
	now := time.Now()

	records := []UsageRecord{
		{Purpose: PurposeContext, CostUSD: 0.10, Timestamp: now},
		{Purpose: PurposeReasoning, CostUSD: 0.40, Timestamp: now},
		{Purpose: PurposeToolCall, CostUSD: 0.20, Timestamp: now},
		{Purpose: PurposeContext, CostUSD: 0.05, Timestamp: now},
	}

	for _, r := range records {
		if err := tr.Record(r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	byPurpose := tr.CostByPurpose()

	if got := byPurpose[PurposeContext]; !floatEqual(got, 0.15) {
		t.Fatalf("CostByPurpose[context] = %f, want 0.15", got)
	}
	if got := byPurpose[PurposeReasoning]; !floatEqual(got, 0.40) {
		t.Fatalf("CostByPurpose[reasoning] = %f, want 0.40", got)
	}
}

func TestCostByModel(t *testing.T) {
	tr := NewTracker(nil)
	now := time.Now()

	records := []UsageRecord{
		{Model: "claude-sonnet", CostUSD: 0.30, Timestamp: now},
		{Model: "claude-haiku", CostUSD: 0.10, Timestamp: now},
		{Model: "claude-sonnet", CostUSD: 0.20, Timestamp: now},
	}

	for _, r := range records {
		if err := tr.Record(r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	byModel := tr.CostByModel()

	if got := byModel["claude-sonnet"]; !floatEqual(got, 0.50) {
		t.Fatalf("CostByModel[claude-sonnet] = %f, want 0.50", got)
	}
	if got := byModel["claude-haiku"]; !floatEqual(got, 0.10) {
		t.Fatalf("CostByModel[claude-haiku] = %f, want 0.10", got)
	}
}

func TestCostByRoleSumsToTotalCost(t *testing.T) {
	tr := NewTracker(nil)
	now := time.Now()

	records := []UsageRecord{
		{AgentRole: RolePlanner, CostUSD: 0.30, Timestamp: now},
		{AgentRole: RoleGenerator, CostUSD: 0.50, Timestamp: now},
		{AgentRole: RoleEvaluator, CostUSD: 0.10, Timestamp: now},
		{AgentRole: RoleSubAgent, CostUSD: 0.10, Timestamp: now},
	}

	for _, r := range records {
		if err := tr.Record(r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	byRole := tr.CostByRole()
	var sum float64
	for _, v := range byRole {
		sum += v
	}

	if !floatEqual(sum, tr.TotalCost()) {
		t.Fatalf("sum of CostByRole (%f) != TotalCost (%f)", sum, tr.TotalCost())
	}
}

func TestBudgetWarningAlert(t *testing.T) {
	budget := &Budget{CostLimitUSD: 1.0}
	tr := NewTracker(budget)
	now := time.Now()

	// 80% of budget should trigger warning.
	if err := tr.Record(UsageRecord{CostUSD: 0.85, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	alerts := tr.Alerts()
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Type != "warning" {
		t.Fatalf("expected warning alert, got %s", alerts[0].Type)
	}
}

func TestBudgetExceededAlert(t *testing.T) {
	budget := &Budget{CostLimitUSD: 1.0}
	tr := NewTracker(budget)
	now := time.Now()

	// Exceed the budget in one record.
	if err := tr.Record(UsageRecord{CostUSD: 1.5, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tr.BudgetExceeded() {
		t.Fatal("expected BudgetExceeded to be true")
	}

	alerts := tr.Alerts()
	found := false
	for _, a := range alerts {
		if a.Type == "exceeded" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected exceeded alert")
	}
}

func TestBudgetExceededIsMonotonic(t *testing.T) {
	budget := &Budget{CostLimitUSD: 1.0}
	tr := NewTracker(budget)
	now := time.Now()

	if err := tr.Record(UsageRecord{CostUSD: 1.5, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tr.BudgetExceeded() {
		t.Fatal("expected exceeded")
	}

	// Recording zero should not revert exceeded.
	if err := tr.Record(UsageRecord{CostUSD: 0, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tr.BudgetExceeded() {
		t.Fatal("BudgetExceeded must be monotonic — should remain true")
	}
}

func TestTokenBudgetWarning(t *testing.T) {
	budget := &Budget{TokenLimit: 1000}
	tr := NewTracker(budget)
	now := time.Now()

	// 85% of token budget.
	if err := tr.Record(UsageRecord{InputTokens: 500, OutputTokens: 350, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	alerts := tr.Alerts()
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Type != "warning" {
		t.Fatalf("expected warning, got %s", alerts[0].Type)
	}
}

func TestTokenBudgetExceeded(t *testing.T) {
	budget := &Budget{TokenLimit: 1000}
	tr := NewTracker(budget)
	now := time.Now()

	if err := tr.Record(UsageRecord{InputTokens: 600, OutputTokens: 500, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tr.BudgetExceeded() {
		t.Fatal("expected BudgetExceeded for token limit")
	}
}

func TestBudgetUtilization(t *testing.T) {
	tests := []struct {
		name   string
		budget *Budget
		cost   float64
		want   float64
	}{
		{
			name:   "nil budget returns zero",
			budget: nil,
			cost:   1.0,
			want:   0,
		},
		{
			name:   "zero limit returns zero",
			budget: &Budget{CostLimitUSD: 0},
			cost:   1.0,
			want:   0,
		},
		{
			name:   "half budget used",
			budget: &Budget{CostLimitUSD: 2.0},
			cost:   1.0,
			want:   0.5,
		},
		{
			name:   "full budget used",
			budget: &Budget{CostLimitUSD: 1.0},
			cost:   1.0,
			want:   1.0,
		},
		{
			name:   "over budget",
			budget: &Budget{CostLimitUSD: 1.0},
			cost:   1.5,
			want:   1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTracker(tt.budget)
			if tt.cost > 0 {
				if err := tr.Record(UsageRecord{CostUSD: tt.cost, Timestamp: time.Now()}); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
			if got := tr.BudgetUtilization(); !floatEqual(got, tt.want) {
				t.Fatalf("BudgetUtilization() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestBelowWarningThresholdNoAlert(t *testing.T) {
	budget := &Budget{CostLimitUSD: 10.0}
	tr := NewTracker(budget)

	// 50% — below warning threshold.
	if err := tr.Record(UsageRecord{CostUSD: 5.0, Timestamp: time.Now()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tr.Alerts()) != 0 {
		t.Fatalf("expected no alerts below warning threshold, got %d", len(tr.Alerts()))
	}
}

func TestReport(t *testing.T) {
	tr := NewTracker(nil)
	now := time.Now()

	records := []UsageRecord{
		{MissionID: "m1", AgentRole: RolePlanner, Purpose: PurposeContext, Model: "sonnet", InputTokens: 100, OutputTokens: 50, CostUSD: 0.30, Timestamp: now},
		{MissionID: "m1", AgentRole: RoleGenerator, Purpose: PurposeReasoning, Model: "sonnet", InputTokens: 200, OutputTokens: 100, CostUSD: 0.50, Timestamp: now},
		{MissionID: "m1", AgentRole: RoleEvaluator, Purpose: PurposeEvaluation, Model: "haiku", InputTokens: 50, OutputTokens: 25, CostUSD: 0.05, Timestamp: now},
	}

	for _, r := range records {
		if err := tr.Record(r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	report := tr.Report("m1")

	if report.MissionID != "m1" {
		t.Fatalf("MissionID = %s, want m1", report.MissionID)
	}
	if report.TotalTokens != 525 {
		t.Fatalf("TotalTokens = %d, want 525", report.TotalTokens)
	}
	if !floatEqual(report.TotalCostUSD, 0.85) {
		t.Fatalf("TotalCostUSD = %f, want 0.85", report.TotalCostUSD)
	}
	if report.RecordCount != 3 {
		t.Fatalf("RecordCount = %d, want 3", report.RecordCount)
	}

	// Verify breakdowns.
	if !floatEqual(report.ByRole[RolePlanner], 0.30) {
		t.Fatalf("ByRole[planner] = %f, want 0.30", report.ByRole[RolePlanner])
	}
	if !floatEqual(report.ByPurpose[PurposeReasoning], 0.50) {
		t.Fatalf("ByPurpose[reasoning] = %f, want 0.50", report.ByPurpose[PurposeReasoning])
	}
	if !floatEqual(report.ByModel["sonnet"], 0.80) {
		t.Fatalf("ByModel[sonnet] = %f, want 0.80", report.ByModel["sonnet"])
	}
}

func TestReportBreakdownSumsToTotal(t *testing.T) {
	tr := NewTracker(nil)
	now := time.Now()

	records := []UsageRecord{
		{AgentRole: RolePlanner, Purpose: PurposeContext, Model: "a", CostUSD: 0.10, Timestamp: now},
		{AgentRole: RoleGenerator, Purpose: PurposeReasoning, Model: "b", CostUSD: 0.20, Timestamp: now},
		{AgentRole: RoleEvaluator, Purpose: PurposeToolCall, Model: "a", CostUSD: 0.30, Timestamp: now},
	}

	for _, r := range records {
		if err := tr.Record(r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	report := tr.Report("test")

	// ByRole sums to total.
	var roleSum float64
	for _, v := range report.ByRole {
		roleSum += v
	}
	if !floatEqual(roleSum, report.TotalCostUSD) {
		t.Fatalf("ByRole sum (%f) != TotalCostUSD (%f)", roleSum, report.TotalCostUSD)
	}

	// ByPurpose sums to total.
	var purposeSum float64
	for _, v := range report.ByPurpose {
		purposeSum += v
	}
	if !floatEqual(purposeSum, report.TotalCostUSD) {
		t.Fatalf("ByPurpose sum (%f) != TotalCostUSD (%f)", purposeSum, report.TotalCostUSD)
	}

	// ByModel sums to total.
	var modelSum float64
	for _, v := range report.ByModel {
		modelSum += v
	}
	if !floatEqual(modelSum, report.TotalCostUSD) {
		t.Fatalf("ByModel sum (%f) != TotalCostUSD (%f)", modelSum, report.TotalCostUSD)
	}
}

func TestDefaultModelLadder(t *testing.T) {
	ladder := DefaultModelLadder()

	expected := map[AgentRole]string{
		RolePlanner:   "claude-sonnet-4-20250514",
		RoleGenerator: "claude-sonnet-4-20250514",
		RoleEvaluator: "claude-haiku-35-20241022",
		RoleSubAgent:  "claude-haiku-35-20241022",
	}

	if len(ladder.Roles) != len(expected) {
		t.Fatalf("expected %d roles in ladder, got %d", len(expected), len(ladder.Roles))
	}

	for role, wantModel := range expected {
		gotModel, ok := ladder.Roles[role]
		if !ok {
			t.Fatalf("DefaultModelLadder missing role %s", role)
		}
		if gotModel != wantModel {
			t.Fatalf("DefaultModelLadder[%s] = %q, want %q", role, gotModel, wantModel)
		}
	}
}

func TestDetectAnomalies(t *testing.T) {
	tests := []struct {
		name       string
		current    *CostReport
		history    []*CostReport
		wantCount  int
		wantMetric string
	}{
		{
			name:      "nil current returns nil",
			current:   nil,
			history:   []*CostReport{{TotalCostUSD: 1.0}},
			wantCount: 0,
		},
		{
			name:      "empty history returns nil",
			current:   &CostReport{TotalCostUSD: 5.0},
			history:   nil,
			wantCount: 0,
		},
		{
			name:    "cost anomaly detected",
			current: &CostReport{MissionID: "m1", TotalCostUSD: 10.0},
			history: []*CostReport{
				{TotalCostUSD: 2.0},
				{TotalCostUSD: 3.0},
				{TotalCostUSD: 2.5},
			},
			wantCount:  1,
			wantMetric: "total_cost_usd",
		},
		{
			name:    "no anomaly within threshold",
			current: &CostReport{MissionID: "m1", TotalCostUSD: 3.0},
			history: []*CostReport{
				{TotalCostUSD: 2.0},
				{TotalCostUSD: 2.5},
			},
			wantCount: 0,
		},
		{
			name:    "token anomaly detected",
			current: &CostReport{MissionID: "m1", TotalTokens: 10000},
			history: []*CostReport{
				{TotalTokens: 2000},
				{TotalTokens: 2500},
			},
			wantCount:  1,
			wantMetric: "total_tokens",
		},
		{
			name: "role anomaly detected",
			current: &CostReport{
				MissionID: "m1",
				ByRole: map[AgentRole]float64{
					RolePlanner: 5.0,
				},
			},
			history: []*CostReport{
				{ByRole: map[AgentRole]float64{RolePlanner: 1.0}},
				{ByRole: map[AgentRole]float64{RolePlanner: 1.5}},
			},
			wantCount:  1,
			wantMetric: "role_planner_cost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anomalies := DetectAnomalies(tt.current, tt.history)
			if len(anomalies) != tt.wantCount {
				t.Fatalf("DetectAnomalies() returned %d anomalies, want %d", len(anomalies), tt.wantCount)
			}
			if tt.wantCount > 0 && tt.wantMetric != "" {
				if anomalies[0].Metric != tt.wantMetric {
					t.Fatalf("anomaly metric = %s, want %s", anomalies[0].Metric, tt.wantMetric)
				}
				if anomalies[0].Ratio <= anomalyThreshold {
					t.Fatalf("anomaly ratio %f should be > %f", anomalies[0].Ratio, anomalyThreshold)
				}
			}
		})
	}
}

func TestSaveAndLoadReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	original := &CostReport{
		MissionID:    "m-42",
		TotalTokens:  1500,
		TotalCostUSD: 2.75,
		ByRole: map[AgentRole]float64{
			RolePlanner:   1.00,
			RoleGenerator: 1.75,
		},
		ByPurpose: map[Purpose]float64{
			PurposeContext:   0.50,
			PurposeReasoning: 2.25,
		},
		ByModel: map[string]float64{
			"sonnet": 2.75,
		},
		RecordCount: 5,
		GeneratedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	if err := SaveReport(path, original); err != nil {
		t.Fatalf("SaveReport() error: %v", err)
	}

	loaded, err := LoadReport(path)
	if err != nil {
		t.Fatalf("LoadReport() error: %v", err)
	}

	if loaded.MissionID != original.MissionID {
		t.Fatalf("MissionID = %s, want %s", loaded.MissionID, original.MissionID)
	}
	if loaded.TotalTokens != original.TotalTokens {
		t.Fatalf("TotalTokens = %d, want %d", loaded.TotalTokens, original.TotalTokens)
	}
	if !floatEqual(loaded.TotalCostUSD, original.TotalCostUSD) {
		t.Fatalf("TotalCostUSD = %f, want %f", loaded.TotalCostUSD, original.TotalCostUSD)
	}
	if loaded.RecordCount != original.RecordCount {
		t.Fatalf("RecordCount = %d, want %d", loaded.RecordCount, original.RecordCount)
	}
	if !floatEqual(loaded.ByRole[RolePlanner], original.ByRole[RolePlanner]) {
		t.Fatalf("ByRole[planner] mismatch")
	}
}

func TestLoadReportNotFound(t *testing.T) {
	_, err := LoadReport("/nonexistent/path/report.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadReportInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadReport(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAlertsReturnsCopy(t *testing.T) {
	budget := &Budget{CostLimitUSD: 1.0}
	tr := NewTracker(budget)

	if err := tr.Record(UsageRecord{CostUSD: 0.9, Timestamp: time.Now()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	alerts1 := tr.Alerts()
	alerts2 := tr.Alerts()

	if len(alerts1) == 0 {
		t.Fatal("expected at least one alert")
	}

	// Mutating the returned slice should not affect the tracker.
	alerts1[0].Type = "mutated"
	if tr.Alerts()[0].Type == "mutated" {
		t.Fatal("Alerts() should return a copy, not a reference")
	}

	// Both calls should return the same data.
	if alerts1[0].Limit != alerts2[0].Limit {
		t.Fatal("consecutive Alerts() calls should return equivalent data")
	}
}

func TestMultipleWarningsNotDuplicated(t *testing.T) {
	budget := &Budget{CostLimitUSD: 10.0}
	tr := NewTracker(budget)
	now := time.Now()

	// First record at 82% — triggers warning.
	if err := tr.Record(UsageRecord{CostUSD: 8.2, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second record still under exceeded — should not re-trigger warning.
	if err := tr.Record(UsageRecord{CostUSD: 0.5, Timestamp: now}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	warnings := 0
	for _, a := range tr.Alerts() {
		if a.Type == "warning" {
			warnings++
		}
	}
	if warnings != 1 {
		t.Fatalf("expected 1 warning, got %d", warnings)
	}
}

func TestAnomalyRatioAlwaysAboveThreshold(t *testing.T) {
	current := &CostReport{
		MissionID:    "m1",
		TotalCostUSD: 100.0,
		TotalTokens:  50000,
		ByRole: map[AgentRole]float64{
			RolePlanner: 80.0,
		},
	}
	history := []*CostReport{
		{TotalCostUSD: 10.0, TotalTokens: 5000, ByRole: map[AgentRole]float64{RolePlanner: 8.0}},
	}

	anomalies := DetectAnomalies(current, history)
	for _, a := range anomalies {
		if a.Ratio <= anomalyThreshold {
			t.Fatalf("anomaly ratio %f should be > %f for metric %s", a.Ratio, anomalyThreshold, a.Metric)
		}
	}
}

func TestEmptyTrackerReport(t *testing.T) {
	tr := NewTracker(nil)
	report := tr.Report("empty")

	if report.MissionID != "empty" {
		t.Fatalf("MissionID = %s, want empty", report.MissionID)
	}
	if report.TotalTokens != 0 {
		t.Fatalf("TotalTokens = %d, want 0", report.TotalTokens)
	}
	if report.TotalCostUSD != 0 {
		t.Fatalf("TotalCostUSD = %f, want 0", report.TotalCostUSD)
	}
	if report.RecordCount != 0 {
		t.Fatalf("RecordCount = %d, want 0", report.RecordCount)
	}
}

func TestNewWindowedTracker(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		b := Budget{CostLimitUSD: 5.0, Window: time.Hour}
		wt, err := NewWindowedTracker(b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wt == nil {
			t.Fatal("expected non-nil tracker")
		}
		if wt.WindowExceeded() {
			t.Fatal("new windowed tracker should not have exceeded budget")
		}
		if wt.TotalCost() != 0 {
			t.Fatal("new windowed tracker should have zero cost")
		}
		if wt.WindowCost() != 0 {
			t.Fatal("new windowed tracker should have zero window cost")
		}
	})

	t.Run("rejects zero window", func(t *testing.T) {
		b := Budget{CostLimitUSD: 5.0, Window: 0}
		_, err := NewWindowedTracker(b)
		if err == nil {
			t.Fatal("expected error for zero window")
		}
	})

	t.Run("rejects negative window", func(t *testing.T) {
		b := Budget{CostLimitUSD: 5.0, Window: -time.Minute}
		_, err := NewWindowedTracker(b)
		if err == nil {
			t.Fatal("expected error for negative window")
		}
	})
}

func TestWindowedTrackerRecordAndRotate(t *testing.T) {
	b := Budget{CostLimitUSD: 10.0, Window: time.Hour}
	wt, err := NewWindowedTracker(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Record in window 1.
	if err := wt.Record(UsageRecord{CostUSD: 1.0, InputTokens: 100, OutputTokens: 50, Timestamp: base}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := wt.Record(UsageRecord{CostUSD: 2.0, InputTokens: 200, OutputTokens: 100, Timestamp: base.Add(30 * time.Minute)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := wt.WindowCost(); !floatEqual(got, 3.0) {
		t.Fatalf("WindowCost() = %f, want 3.0", got)
	}
	if got := wt.WindowTokens(); got != 450 {
		t.Fatalf("WindowTokens() = %d, want 450", got)
	}

	// Record in window 2 (past the 1-hour boundary).
	if err := wt.Record(UsageRecord{CostUSD: 0.5, InputTokens: 50, OutputTokens: 25, Timestamp: base.Add(61 * time.Minute)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Window cost should only reflect the new window.
	if got := wt.WindowCost(); !floatEqual(got, 0.5) {
		t.Fatalf("after rotation WindowCost() = %f, want 0.5", got)
	}
	if got := wt.WindowTokens(); got != 75 {
		t.Fatalf("after rotation WindowTokens() = %d, want 75", got)
	}

	// Total should still include everything.
	if got := wt.TotalCost(); !floatEqual(got, 3.5) {
		t.Fatalf("TotalCost() = %f, want 3.5", got)
	}
	if got := wt.TotalTokens(); got != 525 {
		t.Fatalf("TotalTokens() = %d, want 525", got)
	}
}

func TestWindowedTrackerPerWindowBudget(t *testing.T) {
	b := Budget{CostLimitUSD: 1.0, Window: time.Hour}
	wt, err := NewWindowedTracker(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Exceed in window 1.
	if err := wt.Record(UsageRecord{CostUSD: 1.5, Timestamp: base}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wt.WindowExceeded() {
		t.Fatal("expected WindowExceeded to be true")
	}

	// Move to window 2 — exceeded should reset.
	if err := wt.Record(UsageRecord{CostUSD: 0.1, Timestamp: base.Add(61 * time.Minute)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt.WindowExceeded() {
		t.Fatal("WindowExceeded should reset after window rotation")
	}
}

func TestWindowedTrackerAlerts(t *testing.T) {
	b := Budget{CostLimitUSD: 1.0, Window: time.Hour}
	wt, err := NewWindowedTracker(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// 85% — triggers warning.
	if err := wt.Record(UsageRecord{CostUSD: 0.85, Timestamp: base}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	alerts := wt.Alerts()
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Type != "warning" {
		t.Fatalf("expected warning, got %s", alerts[0].Type)
	}

	// Push over 100% — triggers exceeded.
	if err := wt.Record(UsageRecord{CostUSD: 0.20, Timestamp: base.Add(10 * time.Minute)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	alerts = wt.Alerts()
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(alerts))
	}
	if alerts[1].Type != "exceeded" {
		t.Fatalf("expected exceeded, got %s", alerts[1].Type)
	}
}

func TestWindowedTrackerTotalCost(t *testing.T) {
	b := Budget{CostLimitUSD: 10.0, Window: time.Hour}
	wt, err := NewWindowedTracker(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Spread records across multiple windows.
	records := []UsageRecord{
		{CostUSD: 1.0, InputTokens: 100, OutputTokens: 50, Timestamp: base},
		{CostUSD: 2.0, InputTokens: 200, OutputTokens: 100, Timestamp: base.Add(30 * time.Minute)},
		{CostUSD: 3.0, InputTokens: 300, OutputTokens: 150, Timestamp: base.Add(90 * time.Minute)},
		{CostUSD: 4.0, InputTokens: 400, OutputTokens: 200, Timestamp: base.Add(150 * time.Minute)},
	}

	for _, r := range records {
		if err := wt.Record(r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Total must always accumulate across all windows.
	if got := wt.TotalCost(); !floatEqual(got, 10.0) {
		t.Fatalf("TotalCost() = %f, want 10.0", got)
	}
	if got := wt.TotalTokens(); got != 1500 {
		t.Fatalf("TotalTokens() = %d, want 1500", got)
	}

	// Window cost should only be the last window.
	if got := wt.WindowCost(); !floatEqual(got, 4.0) {
		t.Fatalf("WindowCost() = %f, want 4.0 (last window only)", got)
	}
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// Verify anomaly detection with exact-boundary ratio (ratio == 2.0 is NOT an anomaly).
func TestAnomalyExactBoundary(t *testing.T) {
	current := &CostReport{MissionID: "m1", TotalCostUSD: 4.0}
	history := []*CostReport{{TotalCostUSD: 2.0}}

	anomalies := DetectAnomalies(current, history)
	if len(anomalies) != 0 {
		t.Fatal("ratio == 2.0 should not be flagged as anomaly (must be > 2.0)")
	}
}

// Verify anomaly detection just above boundary.
func TestAnomalyJustAboveBoundary(t *testing.T) {
	current := &CostReport{MissionID: "m1", TotalCostUSD: 4.01}
	history := []*CostReport{{TotalCostUSD: 2.0}}

	anomalies := DetectAnomalies(current, history)
	if len(anomalies) != 1 {
		t.Fatalf("ratio just above 2.0 should be anomaly, got %d anomalies", len(anomalies))
	}
	if anomalies[0].Ratio <= anomalyThreshold {
		t.Fatalf("ratio %f should exceed threshold %f", anomalies[0].Ratio, anomalyThreshold)
	}
}

func TestBudgetExceededDoesNotProduceWarningFirst(t *testing.T) {
	// When a single record jumps from 0% to >100%, we should get an exceeded
	// alert but NOT a spurious warning first.
	budget := &Budget{CostLimitUSD: 1.0}
	tr := NewTracker(budget)

	if err := tr.Record(UsageRecord{CostUSD: 2.0, Timestamp: time.Now()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	alerts := tr.Alerts()
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d: %+v", len(alerts), alerts)
	}
	if alerts[0].Type != "exceeded" {
		t.Fatalf("expected exceeded alert, got %s", alerts[0].Type)
	}
}

// Stress test: many records produce consistent state.
func TestManyRecords(t *testing.T) {
	tr := NewTracker(nil)
	now := time.Now()
	n := 1000

	for i := 0; i < n; i++ {
		if err := tr.Record(UsageRecord{
			CostUSD:      0.001,
			InputTokens:  10,
			OutputTokens: 5,
			Timestamp:    now,
		}); err != nil {
			t.Fatalf("record %d: unexpected error: %v", i, err)
		}
	}

	wantCost := float64(n) * 0.001
	if got := tr.TotalCost(); math.Abs(got-wantCost) > 1e-6 {
		t.Fatalf("TotalCost() = %f, want ~%f", got, wantCost)
	}

	wantTokens := n * 15
	if got := tr.TotalTokens(); got != wantTokens {
		t.Fatalf("TotalTokens() = %d, want %d", got, wantTokens)
	}
}
