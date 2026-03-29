package evaluator

import (
	"context"
	"encoding/json"
	"testing"

	"pgregory.net/rapid"
)

// --- Property: same-ID rejection ---

func TestPropSameIDRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "id")
		gen := &stubGenerator{id: id, outputs: []string{"x"}}
		eval := &stubEvaluator{id: id, results: []*EvalResult{{Score: QualityScore{Overall: 1.0}}}}
		_, err := NewLoop(gen, eval, EvalConfig{})
		if err == nil {
			t.Fatal("expected error when generator and evaluator share the same ID")
		}
	})
}

// --- Property: different IDs always accepted (with valid config) ---

func TestPropDifferentIDsAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		genID := rapid.StringMatching(`gen-[a-z0-9]{1,10}`).Draw(t, "genID")
		evalID := rapid.StringMatching(`eval-[a-z0-9]{1,10}`).Draw(t, "evalID")
		// genID starts with "gen-" and evalID with "eval-", so they're always different.
		gen := &stubGenerator{id: genID, outputs: []string{"x"}}
		eval := &stubEvaluator{id: evalID, results: []*EvalResult{{Score: QualityScore{Overall: 1.0}}}}
		loop, err := NewLoop(gen, eval, EvalConfig{})
		if err != nil {
			t.Fatalf("unexpected error for different IDs %q/%q: %v", genID, evalID, err)
		}
		if loop == nil {
			t.Fatal("expected non-nil loop")
		}
	})
}

// --- Property: loop always terminates within MaxIterations ---

func TestPropLoopTerminatesWithinMaxIterations(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxIter := rapid.IntRange(1, 10).Draw(t, "maxIter")

		// Build enough outputs and results for maxIter iterations, all failing.
		outputs := make([]string, maxIter)
		results := make([]*EvalResult, maxIter)
		for i := 0; i < maxIter; i++ {
			outputs[i] = "output"
			results[i] = &EvalResult{
				Score:    QualityScore{Overall: 0.1},
				Feedback: []Issue{{Severity: SeverityLow, Description: "not great"}},
			}
		}

		gen := &stubGenerator{id: "gen-prop", outputs: outputs}
		eval := &stubEvaluator{id: "eval-prop", results: results}
		cfg := EvalConfig{
			MaxIterations: maxIter,
			PassThreshold: 0.9,
		}

		loop, err := NewLoop(gen, eval, cfg)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		lr, err := loop.Run(context.Background(), "task")
		if err != nil {
			t.Fatalf("run: %v", err)
		}

		if lr.Iterations > maxIter {
			t.Errorf("ran %d iterations, max was %d", lr.Iterations, maxIter)
		}
		if len(lr.History) > maxIter {
			t.Errorf("history has %d entries, max was %d", len(lr.History), maxIter)
		}
	})
}

// --- Property: scores are always in [0.0, 1.0] ---

func TestPropScoreBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		score := rapid.Float64Range(0, 1).Draw(t, "score")
		w1 := rapid.Float64Range(0.01, 0.99).Draw(t, "w1")
		w2 := 1.0 - w1

		q := QualityScore{
			Criteria: map[string]float64{
				"a": score,
				"b": rapid.Float64Range(0, 1).Draw(t, "score2"),
			},
		}
		criteria := []Criterion{
			{Name: "a", Weight: w1},
			{Name: "b", Weight: w2},
		}
		ws := q.WeightedScore(criteria)
		if ws < 0 || ws > 1 {
			t.Errorf("weighted score %f out of [0, 1]", ws)
		}
	})
}

// --- Property: pass result has score >= threshold ---

func TestPropPassImpliesScoreAboveThreshold(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.Float64Range(0.01, 0.99).Draw(t, "threshold")
		score := rapid.Float64Range(threshold, 1.0).Draw(t, "score")

		gen := &stubGenerator{id: "gen-p", outputs: []string{"ok"}}
		eval := &stubEvaluator{id: "eval-p", results: []*EvalResult{
			{Score: QualityScore{Overall: score}},
		}}
		cfg := EvalConfig{
			MaxIterations: 1,
			PassThreshold: threshold,
		}
		loop, err := NewLoop(gen, eval, cfg)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		lr, err := loop.Run(context.Background(), "task")
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		if !lr.FinalResult.Pass {
			t.Errorf("score %f >= threshold %f but Pass is false", score, threshold)
		}
	})
}

// --- Property: failing score < threshold ---

func TestPropFailImpliesScoreBelowThreshold(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.Float64Range(0.02, 0.99).Draw(t, "threshold")
		score := rapid.Float64Range(0.0, threshold-0.001).Draw(t, "score")

		gen := &stubGenerator{id: "gen-f", outputs: []string{"bad"}}
		eval := &stubEvaluator{id: "eval-f", results: []*EvalResult{
			{Score: QualityScore{Overall: score}},
		}}
		cfg := EvalConfig{
			MaxIterations: 1,
			PassThreshold: threshold,
		}
		loop, err := NewLoop(gen, eval, cfg)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		lr, err := loop.Run(context.Background(), "task")
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		if lr.FinalResult.Pass {
			t.Errorf("score %f < threshold %f but Pass is true", score, threshold)
		}
	})
}

// --- Property: SaveReport/LoadReport round-trip ---

func TestPropReportRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		iterations := rapid.IntRange(1, 5).Draw(t, "iterations")
		converged := rapid.Bool().Draw(t, "converged")
		overall := rapid.Float64Range(0, 1).Draw(t, "overall")

		history := make([]EvalResult, iterations)
		for i := range history {
			history[i] = EvalResult{
				Pass:        i == iterations-1 && converged,
				Score:       QualityScore{Overall: rapid.Float64Range(0, 1).Draw(t, "histScore")},
				Iteration:   i + 1,
				EvaluatorID: "eval-rt",
			}
		}

		original := &LoopResult{
			FinalResult: &EvalResult{
				Pass:        converged,
				Score:       QualityScore{Overall: overall},
				Iteration:   iterations,
				EvaluatorID: "eval-rt",
			},
			Iterations: iterations,
			History:    history,
			Converged:  converged,
		}

		// Round-trip through JSON (same serialisation used by Save/LoadReport).
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var loaded LoopResult
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if loaded.Converged != original.Converged {
			t.Errorf("converged mismatch: got %v, want %v", loaded.Converged, original.Converged)
		}
		if loaded.Iterations != original.Iterations {
			t.Errorf("iterations mismatch: got %d, want %d", loaded.Iterations, original.Iterations)
		}
		if loaded.FinalResult.Score.Overall != original.FinalResult.Score.Overall {
			t.Errorf("overall mismatch: got %f, want %f", loaded.FinalResult.Score.Overall, original.FinalResult.Score.Overall)
		}
		if len(loaded.History) != len(original.History) {
			t.Errorf("history len mismatch: got %d, want %d", len(loaded.History), len(original.History))
		}
	})
}

// --- Property: ValidateConfig rejects weights far from 1.0 ---

func TestPropValidateConfigRejectsBadWeights(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate two weights that don't sum to ~1.0.
		w1 := rapid.Float64Range(0.0, 0.3).Draw(t, "w1")
		w2 := rapid.Float64Range(0.0, 0.3).Draw(t, "w2")
		// w1 + w2 <= 0.6, which is far from 1.0.

		cfg := EvalConfig{
			Criteria: []Criterion{
				{Name: "a", Weight: w1, Threshold: 0.5},
				{Name: "b", Weight: w2, Threshold: 0.5},
			},
			PassThreshold: 0.7,
		}
		if err := ValidateConfig(cfg); err == nil {
			t.Errorf("expected error for weights summing to %f", w1+w2)
		}
	})
}

// --- Property: deterministic evaluation produces same result ---

func TestPropDeterministicEvaluation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		score := rapid.Float64Range(0, 1).Draw(t, "score")
		threshold := rapid.Float64Range(0.01, 0.99).Draw(t, "threshold")

		run := func() *LoopResult {
			gen := &stubGenerator{id: "gen-d", outputs: []string{"out"}}
			eval := &stubEvaluator{id: "eval-d", results: []*EvalResult{
				{Score: QualityScore{Overall: score}},
			}}
			cfg := EvalConfig{MaxIterations: 1, PassThreshold: threshold}
			loop, err := NewLoop(gen, eval, cfg)
			if err != nil {
				t.Fatalf("setup: %v", err)
			}
			lr, err := loop.Run(context.Background(), "task")
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			return lr
		}

		r1 := run()
		r2 := run()

		if r1.FinalResult.Pass != r2.FinalResult.Pass {
			t.Error("determinism violation: pass differs between identical runs")
		}
		if r1.Iterations != r2.Iterations {
			t.Error("determinism violation: iterations differ between identical runs")
		}
		if r1.Converged != r2.Converged {
			t.Error("determinism violation: converged differs between identical runs")
		}
	})
}
