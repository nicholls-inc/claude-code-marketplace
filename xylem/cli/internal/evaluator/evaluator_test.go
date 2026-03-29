package evaluator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// --- test doubles ---

type stubGenerator struct {
	id      string
	outputs []string // successive calls return successive elements
	call    int
}

func (s *stubGenerator) Generate(_ context.Context, _ string, _ []Issue) (string, error) {
	if s.call >= len(s.outputs) {
		return "", errors.New("stub: no more outputs")
	}
	out := s.outputs[s.call]
	s.call++
	return out, nil
}

func (s *stubGenerator) ID() string { return s.id }

type stubEvaluator struct {
	id      string
	results []*EvalResult // successive calls return successive elements
	call    int
}

func (s *stubEvaluator) Evaluate(_ context.Context, _ string, _ []Criterion) (*EvalResult, error) {
	if s.call >= len(s.results) {
		return nil, errors.New("stub: no more results")
	}
	r := s.results[s.call]
	s.call++
	return r, nil
}

func (s *stubEvaluator) ID() string { return s.id }

type errGenerator struct{ id string }

func (e *errGenerator) Generate(_ context.Context, _ string, _ []Issue) (string, error) {
	return "", errors.New("generate failed")
}

func (e *errGenerator) ID() string { return e.id }

type nilEvaluator struct{ id string }

func (n *nilEvaluator) Evaluate(_ context.Context, _ string, _ []Criterion) (*EvalResult, error) {
	return nil, nil
}

func (n *nilEvaluator) ID() string { return n.id }

type errEvaluator struct{ id string }

func (e *errEvaluator) Evaluate(_ context.Context, _ string, _ []Criterion) (*EvalResult, error) {
	return nil, errors.New("evaluate failed")
}

func (e *errEvaluator) ID() string { return e.id }

// --- helper criteria ---

func testCriteria() []Criterion {
	return []Criterion{
		{Name: "correctness", Description: "Is the output correct?", Weight: 0.6, Threshold: 0.5},
		{Name: "style", Description: "Code style", Weight: 0.4, Threshold: 0.5},
	}
}

func testConfig() EvalConfig {
	return EvalConfig{
		Criteria:      testCriteria(),
		MaxIterations: 3,
		PassThreshold: 0.7,
	}
}

// --- NewLoop tests ---

func TestNewLoopValid(t *testing.T) {
	gen := &stubGenerator{id: "gen-1"}
	eval := &stubEvaluator{id: "eval-1"}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loop == nil {
		t.Fatal("expected non-nil loop")
	}
}

func TestNewLoopSameIDRejected(t *testing.T) {
	gen := &stubGenerator{id: "same"}
	eval := &stubEvaluator{id: "same"}
	_, err := NewLoop(gen, eval, testConfig())
	if err == nil {
		t.Fatal("expected error when generator and evaluator have same ID")
	}
}

func TestNewLoopNilGenerator(t *testing.T) {
	eval := &stubEvaluator{id: "eval-1"}
	_, err := NewLoop(nil, eval, testConfig())
	if err == nil {
		t.Fatal("expected error for nil generator")
	}
}

func TestNewLoopNilEvaluator(t *testing.T) {
	gen := &stubGenerator{id: "gen-1"}
	_, err := NewLoop(gen, nil, testConfig())
	if err == nil {
		t.Fatal("expected error for nil evaluator")
	}
}

func TestNewLoopInvalidConfig(t *testing.T) {
	gen := &stubGenerator{id: "gen-1"}
	eval := &stubEvaluator{id: "eval-1"}
	cfg := EvalConfig{
		Criteria: []Criterion{
			{Name: "a", Weight: 0.5, Threshold: 0.5},
			// weights don't sum to 1.0
		},
		MaxIterations: 3,
		PassThreshold: 0.7,
	}
	_, err := NewLoop(gen, eval, cfg)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

// --- Run tests ---

func TestRunPassFirstIteration(t *testing.T) {
	gen := &stubGenerator{id: "gen-1", outputs: []string{"output-1"}}
	eval := &stubEvaluator{id: "eval-1", results: []*EvalResult{
		{Score: QualityScore{Overall: 0.9, Criteria: map[string]float64{"correctness": 0.9, "style": 0.9}}},
	}}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	lr, err := loop.Run(context.Background(), "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !lr.Converged {
		t.Error("expected loop to converge")
	}
	if lr.Iterations != 1 {
		t.Errorf("expected 1 iteration, got %d", lr.Iterations)
	}
	if !lr.FinalResult.Pass {
		t.Error("expected final result to pass")
	}
}

func TestRunFailAllIterations(t *testing.T) {
	gen := &stubGenerator{id: "gen-1", outputs: []string{"a", "b", "c"}}
	eval := &stubEvaluator{id: "eval-1", results: []*EvalResult{
		{Score: QualityScore{Overall: 0.3}, Feedback: []Issue{{Severity: SeverityMedium, Description: "bad"}}},
		{Score: QualityScore{Overall: 0.4}, Feedback: []Issue{{Severity: SeverityLow, Description: "meh"}}},
		{Score: QualityScore{Overall: 0.5}, Feedback: []Issue{{Severity: SeverityLow, Description: "ok-ish"}}},
	}}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	lr, err := loop.Run(context.Background(), "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lr.Converged {
		t.Error("expected loop not to converge")
	}
	if lr.Iterations != 3 {
		t.Errorf("expected 3 iterations, got %d", lr.Iterations)
	}
	if lr.FinalResult.Pass {
		t.Error("expected final result not to pass")
	}
}

func TestRunConvergesSecondIteration(t *testing.T) {
	gen := &stubGenerator{id: "gen-1", outputs: []string{"v1", "v2"}}
	eval := &stubEvaluator{id: "eval-1", results: []*EvalResult{
		{Score: QualityScore{Overall: 0.5}, Feedback: []Issue{{Severity: SeverityHigh, Description: "fix it"}}},
		{Score: QualityScore{Overall: 0.8}},
	}}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	lr, err := loop.Run(context.Background(), "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !lr.Converged {
		t.Error("expected loop to converge on second iteration")
	}
	if lr.Iterations != 2 {
		t.Errorf("expected 2 iterations, got %d", lr.Iterations)
	}
	if len(lr.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(lr.History))
	}
}

func TestRunGenerateError(t *testing.T) {
	gen := &errGenerator{id: "gen-1"}
	eval := &stubEvaluator{id: "eval-1"}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = loop.Run(context.Background(), "task")
	if err == nil {
		t.Fatal("expected error from generate failure")
	}
}

func TestRunEvaluateNilResult(t *testing.T) {
	gen := &stubGenerator{id: "gen-1", outputs: []string{"output"}}
	eval := &nilEvaluator{id: "eval-1"}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = loop.Run(context.Background(), "task")
	if err == nil {
		t.Fatal("expected error when evaluator returns nil result")
	}
}

func TestRunEvaluateError(t *testing.T) {
	gen := &stubGenerator{id: "gen-1", outputs: []string{"output"}}
	eval := &errEvaluator{id: "eval-1"}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = loop.Run(context.Background(), "task")
	if err == nil {
		t.Fatal("expected error from evaluate failure")
	}
}

func TestRunContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	gen := &stubGenerator{id: "gen-1", outputs: []string{"x"}}
	eval := &stubEvaluator{id: "eval-1", results: []*EvalResult{
		{Score: QualityScore{Overall: 0.9}},
	}}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = loop.Run(ctx, "task")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRunSetsEvaluatorIDAndIteration(t *testing.T) {
	gen := &stubGenerator{id: "gen-1", outputs: []string{"out"}}
	eval := &stubEvaluator{id: "eval-42", results: []*EvalResult{
		{Score: QualityScore{Overall: 0.9}},
	}}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	lr, err := loop.Run(context.Background(), "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lr.FinalResult.EvaluatorID != "eval-42" {
		t.Errorf("expected evaluator ID %q, got %q", "eval-42", lr.FinalResult.EvaluatorID)
	}
	if lr.FinalResult.Iteration != 1 {
		t.Errorf("expected iteration 1, got %d", lr.FinalResult.Iteration)
	}
}

// --- ValidateConfig tests ---

func TestValidateConfigValid(t *testing.T) {
	err := ValidateConfig(testConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateConfigNegativeMaxIterations(t *testing.T) {
	cfg := testConfig()
	cfg.MaxIterations = -1
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for negative max_iterations")
	}
}

func TestValidateConfigBadPassThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
	}{
		{"below zero", -0.1},
		{"above one", 1.1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.PassThreshold = tc.threshold
			if err := ValidateConfig(cfg); err == nil {
				t.Error("expected error for bad pass threshold")
			}
		})
	}
}

func TestValidateConfigWeightsDontSum(t *testing.T) {
	cfg := EvalConfig{
		Criteria: []Criterion{
			{Name: "a", Weight: 0.3, Threshold: 0.5},
			{Name: "b", Weight: 0.3, Threshold: 0.5},
		},
		MaxIterations: 3,
		PassThreshold: 0.7,
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error when weights don't sum to ~1.0")
	}
}

func TestValidateConfigNegativeWeight(t *testing.T) {
	cfg := EvalConfig{
		Criteria: []Criterion{
			{Name: "a", Weight: -0.5, Threshold: 0.5},
			{Name: "b", Weight: 1.5, Threshold: 0.5},
		},
		PassThreshold: 0.7,
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for negative weight")
	}
}

func TestValidateConfigBadCriterionThreshold(t *testing.T) {
	cfg := EvalConfig{
		Criteria: []Criterion{
			{Name: "a", Weight: 1.0, Threshold: 1.5},
		},
		PassThreshold: 0.7,
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for criterion threshold > 1")
	}
}

func TestValidateConfigDuplicateCriterionName(t *testing.T) {
	cfg := EvalConfig{
		Criteria: []Criterion{
			{Name: "correctness", Weight: 0.5, Threshold: 0.5},
			{Name: "correctness", Weight: 0.5, Threshold: 0.5},
		},
		PassThreshold: 0.7,
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for duplicate criterion name")
	}
}

func TestValidateConfigEmptyCriteria(t *testing.T) {
	cfg := EvalConfig{
		Criteria:      nil,
		MaxIterations: 3,
		PassThreshold: 0.7,
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error for empty criteria, got %v", err)
	}
}

// --- WeightedScore tests ---

func TestWeightedScoreBasic(t *testing.T) {
	q := QualityScore{
		Criteria: map[string]float64{
			"correctness": 0.8,
			"style":       0.6,
		},
	}
	criteria := testCriteria()
	// expected: (0.8*0.6 + 0.6*0.4) / (0.6+0.4) = (0.48 + 0.24) / 1.0 = 0.72
	got := q.WeightedScore(criteria)
	if got < 0.719 || got > 0.721 {
		t.Errorf("expected ~0.72, got %f", got)
	}
}

func TestWeightedScoreNoCriteria(t *testing.T) {
	q := QualityScore{Criteria: map[string]float64{"x": 1.0}}
	got := q.WeightedScore(nil)
	if got != 0 {
		t.Errorf("expected 0 for no criteria, got %f", got)
	}
}

func TestWeightedScoreNoMatchingCriteria(t *testing.T) {
	q := QualityScore{Criteria: map[string]float64{"x": 1.0}}
	criteria := []Criterion{{Name: "y", Weight: 1.0}}
	got := q.WeightedScore(criteria)
	if got != 0 {
		t.Errorf("expected 0 for no matching criteria, got %f", got)
	}
}

func TestWeightedScoreClampedHigh(t *testing.T) {
	// If somehow a criterion score exceeded 1.0 the result should still clamp.
	q := QualityScore{Criteria: map[string]float64{"a": 1.5}}
	criteria := []Criterion{{Name: "a", Weight: 1.0}}
	got := q.WeightedScore(criteria)
	if got != 1.0 {
		t.Errorf("expected exactly 1.0 after clamping, got %f", got)
	}
}

// --- SelectIntensity tests ---

func TestSelectIntensity(t *testing.T) {
	tests := []struct {
		name         string
		complexity   string
		signalHealth string
		want         EvalIntensity
	}{
		{"high complexity", "high", "healthy", Thorough},
		{"critical complexity", "critical", "degraded", Thorough},
		{"low healthy", "low", "healthy", Lightweight},
		{"low good", "low", "good", Lightweight},
		{"low degraded", "low", "degraded", Standard},
		{"trivial healthy", "trivial", "healthy", Lightweight},
		{"medium healthy", "medium", "healthy", Standard},
		{"medium degraded", "medium", "degraded", Thorough},
		{"unknown complexity", "unknown", "good", Standard},
		{"medium unknown signal", "medium", "unknown", Standard},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SelectIntensity(tc.complexity, tc.signalHealth)
			if got != tc.want {
				t.Errorf("SelectIntensity(%q, %q) = %v, want %v", tc.complexity, tc.signalHealth, got, tc.want)
			}
		})
	}
}

// --- Severity.String tests ---

func TestSeverityString(t *testing.T) {
	tests := []struct {
		sev  Severity
		want string
	}{
		{SeverityLow, "low"},
		{SeverityMedium, "medium"},
		{SeverityHigh, "high"},
		{SeverityCritical, "critical"},
		{Severity(99), "severity(99)"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.sev.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// --- EvalIntensity.String tests ---

func TestEvalIntensityString(t *testing.T) {
	tests := []struct {
		val  EvalIntensity
		want string
	}{
		{Lightweight, "lightweight"},
		{Standard, "standard"},
		{Thorough, "thorough"},
		{EvalIntensity(99), "intensity(99)"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.val.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// --- SaveReport / LoadReport round-trip ---

func TestSaveLoadReportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := &LoopResult{
		FinalResult: &EvalResult{
			Pass:        true,
			Score:       QualityScore{Overall: 0.85, Criteria: map[string]float64{"a": 0.9}},
			Iteration:   2,
			EvaluatorID: "eval-1",
		},
		Iterations: 2,
		History: []EvalResult{
			{Pass: false, Score: QualityScore{Overall: 0.5}, Iteration: 1, EvaluatorID: "eval-1"},
			{Pass: true, Score: QualityScore{Overall: 0.85}, Iteration: 2, EvaluatorID: "eval-1"},
		},
		Converged: true,
	}

	if err := SaveReport(dir, original); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(dir, "quality-report.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}

	loaded, err := LoadReport(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Converged != original.Converged {
		t.Errorf("converged: got %v, want %v", loaded.Converged, original.Converged)
	}
	if loaded.Iterations != original.Iterations {
		t.Errorf("iterations: got %d, want %d", loaded.Iterations, original.Iterations)
	}
	if loaded.FinalResult.Score.Overall != original.FinalResult.Score.Overall {
		t.Errorf("overall score: got %f, want %f", loaded.FinalResult.Score.Overall, original.FinalResult.Score.Overall)
	}
	if loaded.FinalResult.Pass != original.FinalResult.Pass {
		t.Errorf("final pass: got %v, want %v", loaded.FinalResult.Pass, original.FinalResult.Pass)
	}
	if loaded.FinalResult.EvaluatorID != original.FinalResult.EvaluatorID {
		t.Errorf("final evaluator ID: got %q, want %q", loaded.FinalResult.EvaluatorID, original.FinalResult.EvaluatorID)
	}
	if len(loaded.History) != len(original.History) {
		t.Errorf("history length: got %d, want %d", len(loaded.History), len(original.History))
	}
	if len(loaded.History) > 0 {
		if loaded.History[0].Score.Overall != original.History[0].Score.Overall {
			t.Errorf("history[0] overall score: got %f, want %f", loaded.History[0].Score.Overall, original.History[0].Score.Overall)
		}
	}
}

func TestLoadReportMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadReport(dir)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSaveReportBadDir(t *testing.T) {
	err := SaveReport("/nonexistent/path/that/does/not/exist", &LoopResult{})
	if err == nil {
		t.Fatal("expected error for bad directory")
	}
}

// --- Run with default config values ---

func TestRunUsesDefaultMaxIterations(t *testing.T) {
	gen := &stubGenerator{id: "gen-1", outputs: []string{"a", "b", "c"}}
	eval := &stubEvaluator{id: "eval-1", results: []*EvalResult{
		{Score: QualityScore{Overall: 0.1}},
		{Score: QualityScore{Overall: 0.2}},
		{Score: QualityScore{Overall: 0.3}},
	}}
	cfg := EvalConfig{
		// MaxIterations 0 -> defaults to 3
		// PassThreshold 0 -> defaults to 0.7
	}
	loop, err := NewLoop(gen, eval, cfg)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	lr, err := loop.Run(context.Background(), "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lr.Iterations != DefaultMaxIterations {
		t.Errorf("expected %d iterations (default), got %d", DefaultMaxIterations, lr.Iterations)
	}
}

func TestRunUsesDefaultPassThreshold(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		wantPass bool
	}{
		{"below default threshold", 0.5, false},
		{"above default threshold", 0.8, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gen := &stubGenerator{id: "gen-1", outputs: []string{"out"}}
			eval := &stubEvaluator{id: "eval-1", results: []*EvalResult{
				{Score: QualityScore{Overall: tc.score}},
			}}
			cfg := EvalConfig{
				MaxIterations: 1,
				// PassThreshold 0 -> defaults to 0.7
			}
			loop, err := NewLoop(gen, eval, cfg)
			if err != nil {
				t.Fatalf("setup: %v", err)
			}
			lr, err := loop.Run(context.Background(), "task")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if lr.FinalResult.Pass != tc.wantPass {
				t.Errorf("score %.1f: got pass=%v, want pass=%v (default threshold 0.7)", tc.score, lr.FinalResult.Pass, tc.wantPass)
			}
		})
	}
}

// --- AdjustForIntensity tests ---

func TestAdjustForIntensityLightweight(t *testing.T) {
	cfg := testConfig()
	adjusted := AdjustForIntensity(cfg, Lightweight)
	if adjusted.MaxIterations != 1 {
		t.Errorf("expected MaxIterations=1 for Lightweight, got %d", adjusted.MaxIterations)
	}
}

func TestAdjustForIntensityStandard(t *testing.T) {
	cfg := EvalConfig{
		Criteria:      testCriteria(),
		MaxIterations: 0, // zero triggers default
		PassThreshold: 0.7,
	}
	adjusted := AdjustForIntensity(cfg, Standard)
	if adjusted.MaxIterations != DefaultMaxIterations {
		t.Errorf("expected MaxIterations=%d for Standard, got %d", DefaultMaxIterations, adjusted.MaxIterations)
	}
}

func TestAdjustForIntensityThorough(t *testing.T) {
	cfg := testConfig()
	adjusted := AdjustForIntensity(cfg, Thorough)
	if adjusted.MaxIterations != 5 {
		t.Errorf("expected MaxIterations=5 for Thorough, got %d", adjusted.MaxIterations)
	}
}

func TestAdjustForIntensityPreservesOtherFields(t *testing.T) {
	cfg := testConfig()
	adjusted := AdjustForIntensity(cfg, Thorough)
	if adjusted.PassThreshold != cfg.PassThreshold {
		t.Errorf("PassThreshold changed: got %f, want %f", adjusted.PassThreshold, cfg.PassThreshold)
	}
	if len(adjusted.Criteria) != len(cfg.Criteria) {
		t.Errorf("Criteria length changed: got %d, want %d", len(adjusted.Criteria), len(cfg.Criteria))
	}
	for i, c := range adjusted.Criteria {
		if c.Name != cfg.Criteria[i].Name {
			t.Errorf("Criteria[%d].Name changed: got %q, want %q", i, c.Name, cfg.Criteria[i].Name)
		}
		if c.Weight != cfg.Criteria[i].Weight {
			t.Errorf("Criteria[%d].Weight changed: got %f, want %f", i, c.Weight, cfg.Criteria[i].Weight)
		}
	}
}

func TestAdjustForIntensityCopiesCriteria(t *testing.T) {
	cfg := testConfig()
	adjusted := AdjustForIntensity(cfg, Standard)
	// Mutate the returned criteria and verify the original is untouched.
	adjusted.Criteria[0].Name = "mutated"
	if cfg.Criteria[0].Name == "mutated" {
		t.Error("modifying returned criteria affected the original — deep copy missing")
	}
}

// --- RunWithIntensity tests ---

func TestRunWithIntensityLightweight(t *testing.T) {
	// Stub always fails so the loop exhausts its iterations.
	gen := &stubGenerator{id: "gen-1", outputs: []string{"a"}}
	eval := &stubEvaluator{id: "eval-1", results: []*EvalResult{
		{Score: QualityScore{Overall: 0.1}, Feedback: []Issue{{Severity: SeverityLow, Description: "bad"}}},
	}}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	lr, err := loop.RunWithIntensity(context.Background(), "task", Lightweight)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lr.Iterations != 1 {
		t.Errorf("expected 1 iteration for Lightweight, got %d", lr.Iterations)
	}
}

func TestRunWithIntensityThorough(t *testing.T) {
	// Stub always fails so the loop runs all 5 iterations.
	outputs := make([]string, 5)
	results := make([]*EvalResult, 5)
	for i := 0; i < 5; i++ {
		outputs[i] = "output"
		results[i] = &EvalResult{
			Score:    QualityScore{Overall: 0.1},
			Feedback: []Issue{{Severity: SeverityLow, Description: "bad"}},
		}
	}
	gen := &stubGenerator{id: "gen-1", outputs: outputs}
	eval := &stubEvaluator{id: "eval-1", results: results}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	lr, err := loop.RunWithIntensity(context.Background(), "task", Thorough)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lr.Iterations != 5 {
		t.Errorf("expected 5 iterations for Thorough, got %d", lr.Iterations)
	}
}

func TestRunWithIntensityRestoresConfig(t *testing.T) {
	gen := &stubGenerator{id: "gen-1", outputs: []string{"a"}}
	eval := &stubEvaluator{id: "eval-1", results: []*EvalResult{
		{Score: QualityScore{Overall: 0.9}},
	}}
	loop, err := NewLoop(gen, eval, testConfig())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	originalMax := loop.config.MaxIterations
	_, err = loop.RunWithIntensity(context.Background(), "task", Lightweight)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loop.config.MaxIterations != originalMax {
		t.Errorf("config not restored: MaxIterations is %d, want %d", loop.config.MaxIterations, originalMax)
	}
}
