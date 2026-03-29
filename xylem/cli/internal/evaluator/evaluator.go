package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// Severity classifies the severity of an evaluation issue.
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// String returns a human-readable label for the severity.
func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// Issue describes a single problem found during evaluation.
type Issue struct {
	Severity    Severity `json:"severity"`
	Description string   `json:"description"`
	Location    string   `json:"location"`
	Suggestion  string   `json:"suggestion"`
}

// QualityScore holds the overall and per-criterion scores for an evaluation.
type QualityScore struct {
	Overall  float64            `json:"overall"`
	Criteria map[string]float64 `json:"criteria"`
	Issues   []Issue            `json:"issues"`
}

// WeightedScore computes a weighted sum of per-criterion scores. Criteria
// present in the score's map but absent from the supplied slice are ignored.
// The result is clamped to [0.0, 1.0].
func (q QualityScore) WeightedScore(criteria []Criterion) float64 {
	var total float64
	var weightSum float64
	for _, c := range criteria {
		score, ok := q.Criteria[c.Name]
		if !ok {
			continue
		}
		total += score * c.Weight
		weightSum += c.Weight
	}
	if weightSum == 0 {
		return 0
	}
	result := total / weightSum
	return clamp(result, 0, 1)
}

// EvalResult is the output of a single evaluation pass.
type EvalResult struct {
	Pass        bool         `json:"pass"`
	Score       QualityScore `json:"score"`
	Feedback    []Issue      `json:"feedback"`
	Iteration   int          `json:"iteration"`
	EvaluatorID string       `json:"evaluator_id"`
}

// Criterion defines one dimension of quality evaluation.
type Criterion struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Threshold   float64 `json:"threshold"`
}

// EvalConfig controls the gen-eval loop behavior.
type EvalConfig struct {
	Criteria      []Criterion `json:"criteria"`
	MaxIterations int         `json:"max_iterations"`
	PassThreshold float64     `json:"pass_threshold"`
}

// DefaultMaxIterations is used when MaxIterations is zero.
const DefaultMaxIterations = 3

// DefaultPassThreshold is used when PassThreshold is zero.
const DefaultPassThreshold = 0.7

// EvalIntensity controls how thorough the evaluation should be.
type EvalIntensity int

const (
	Lightweight EvalIntensity = iota
	Standard
	Thorough
)

// String returns a human-readable label.
func (e EvalIntensity) String() string {
	switch e {
	case Lightweight:
		return "lightweight"
	case Standard:
		return "standard"
	case Thorough:
		return "thorough"
	default:
		return fmt.Sprintf("intensity(%d)", int(e))
	}
}

// Generator produces output for a given task, optionally incorporating
// feedback from a prior evaluation iteration.
type Generator interface {
	Generate(ctx context.Context, task string, feedback []Issue) (string, error)
	ID() string
}

// Evaluator assesses generated output against a set of criteria.
type Evaluator interface {
	Evaluate(ctx context.Context, output string, criteria []Criterion) (*EvalResult, error)
	ID() string
}

// Loop orchestrates iterative generation and evaluation.
type Loop struct {
	generator Generator
	evaluator Evaluator
	config    EvalConfig
}

// LoopResult summarises the outcome of a complete gen-eval loop.
type LoopResult struct {
	FinalResult *EvalResult  `json:"final_result"`
	Iterations  int          `json:"iterations"`
	History     []EvalResult `json:"history"`
	Converged   bool         `json:"converged"`
}

// weightTolerance is the allowed deviation from 1.0 when summing criteria
// weights.
const weightTolerance = 0.01

// ValidateConfig checks that an EvalConfig is well-formed:
//   - weights sum to approximately 1.0
//   - thresholds are in [0, 1]
//   - max iterations is non-negative
//   - pass threshold is in [0, 1]
func ValidateConfig(config EvalConfig) error {
	if config.MaxIterations < 0 {
		return fmt.Errorf("validate config: max_iterations must be non-negative, got %d", config.MaxIterations)
	}

	passThreshold := config.PassThreshold
	if passThreshold == 0 {
		passThreshold = DefaultPassThreshold
	}
	if passThreshold < 0 || passThreshold > 1 {
		return fmt.Errorf("validate config: pass_threshold must be in [0, 1], got %f", config.PassThreshold)
	}

	if len(config.Criteria) > 0 {
		seen := make(map[string]bool, len(config.Criteria))
		var weightSum float64
		for _, c := range config.Criteria {
			if seen[c.Name] {
				return fmt.Errorf("validate config: duplicate criterion name: %q", c.Name)
			}
			seen[c.Name] = true
			if c.Weight < 0 {
				return fmt.Errorf("validate config: criterion %q weight must be non-negative, got %f", c.Name, c.Weight)
			}
			if c.Threshold < 0 || c.Threshold > 1 {
				return fmt.Errorf("validate config: criterion %q threshold must be in [0, 1], got %f", c.Name, c.Threshold)
			}
			weightSum += c.Weight
		}
		if math.Abs(weightSum-1.0) > weightTolerance {
			return fmt.Errorf("validate config: criteria weights must sum to ~1.0, got %f", weightSum)
		}
	}

	return nil
}

// NewLoop creates a Loop after validating that the generator and evaluator
// have different IDs (structural separation guarantee) and that the config is
// valid.
func NewLoop(gen Generator, eval Evaluator, config EvalConfig) (*Loop, error) {
	if gen == nil {
		return nil, fmt.Errorf("new loop: generator must not be nil")
	}
	if eval == nil {
		return nil, fmt.Errorf("new loop: evaluator must not be nil")
	}
	if gen.ID() == eval.ID() {
		return nil, fmt.Errorf("new loop: generator and evaluator must have different IDs, both are %q", gen.ID())
	}
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}
	return &Loop{
		generator: gen,
		evaluator: eval,
		config:    config,
	}, nil
}

// Run executes the gen-eval loop: generate, evaluate, and if the evaluation
// does not pass, feed back issues and iterate up to MaxIterations.
func (l *Loop) Run(ctx context.Context, task string) (*LoopResult, error) {
	maxIter := l.config.MaxIterations
	if maxIter == 0 {
		maxIter = DefaultMaxIterations
	}

	passThreshold := l.config.PassThreshold
	if passThreshold == 0 {
		passThreshold = DefaultPassThreshold
	}

	result := &LoopResult{}
	var feedback []Issue

	for i := 1; i <= maxIter; i++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("loop run: %w", ctx.Err())
		default:
		}

		output, err := l.generator.Generate(ctx, task, feedback)
		if err != nil {
			return nil, fmt.Errorf("loop run: generate iteration %d: %w", i, err)
		}

		evalResult, err := l.evaluator.Evaluate(ctx, output, l.config.Criteria)
		if err != nil {
			return nil, fmt.Errorf("loop run: evaluate iteration %d: %w", i, err)
		}
		if evalResult == nil {
			return nil, fmt.Errorf("loop run: evaluate iteration %d: evaluator returned nil result", i)
		}

		evalResult.Iteration = i
		evalResult.EvaluatorID = l.evaluator.ID()
		evalResult.Pass = evalResult.Score.Overall >= passThreshold

		result.History = append(result.History, *evalResult)
		result.FinalResult = evalResult
		result.Iterations = i

		if evalResult.Pass {
			result.Converged = true
			return result, nil
		}

		feedback = evalResult.Feedback
	}

	return result, nil
}

// SelectIntensity chooses an evaluation intensity based on task complexity
// and signal health indicators. Both arguments are free-form strings that
// are categorised into buckets.
func SelectIntensity(complexity, signalHealth string) EvalIntensity {
	switch complexity {
	case "high", "critical":
		return Thorough
	case "low", "trivial":
		if signalHealth == "healthy" || signalHealth == "good" {
			return Lightweight
		}
		return Standard
	default:
		// "medium" or unrecognised
		switch signalHealth {
		case "degraded", "unhealthy", "bad":
			return Thorough
		case "healthy", "good":
			return Standard
		default:
			return Standard
		}
	}
}

// SaveReport writes a LoopResult as JSON to the given directory under the
// filename "quality-report.json".
func SaveReport(dir string, lr *LoopResult) error {
	data, err := json.MarshalIndent(lr, "", "  ")
	if err != nil {
		return fmt.Errorf("save report: marshal: %w", err)
	}
	path := filepath.Join(dir, "quality-report.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save report: write: %w", err)
	}
	return nil
}

// LoadReport reads a LoopResult from "quality-report.json" in the given
// directory.
func LoadReport(dir string) (*LoopResult, error) {
	path := filepath.Join(dir, "quality-report.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load report: read: %w", err)
	}
	var lr LoopResult
	if err := json.Unmarshal(data, &lr); err != nil {
		return nil, fmt.Errorf("load report: unmarshal: %w", err)
	}
	return &lr, nil
}

// clamp restricts v to the range [lo, hi].
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
