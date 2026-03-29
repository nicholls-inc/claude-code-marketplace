package ctxmgr

import (
	"testing"

	"pgregory.net/rapid"
)

// genSegment generates an arbitrary Segment, including negative token values
// to exercise clamping in Utilization().
func genSegment() *rapid.Generator[Segment] {
	return rapid.Custom(func(t *rapid.T) Segment {
		return Segment{
			Name:     rapid.StringN(1, 20, 20).Draw(t, "name"),
			Content:  rapid.String().Draw(t, "content"),
			Tokens:   rapid.IntRange(-100, 10000).Draw(t, "tokens"),
			Durable:  rapid.Bool().Draw(t, "durable"),
			Source:   rapid.StringN(0, 10, 10).Draw(t, "source"),
			Priority: rapid.IntRange(0, 100).Draw(t, "priority"),
		}
	})
}

// genWindow generates an arbitrary Window.
func genWindow() *rapid.Generator[Window] {
	return rapid.Custom(func(t *rapid.T) Window {
		segs := rapid.SliceOfN(genSegment(), 0, 20).Draw(t, "segments")
		maxTokens := rapid.IntRange(1, 100000).Draw(t, "maxTokens")
		return Window{Segments: segs, MaxTokens: maxTokens}
	})
}

func TestProp_UtilizationBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		w := genWindow().Draw(t, "window")
		u := w.Utilization()
		if u < 0.0 || u > 1.0 {
			t.Fatalf("Utilization() = %f, want [0.0, 1.0]", u)
		}
	})
}

func TestProp_CompactPreservesDurable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		w := genWindow().Draw(t, "window")
		cfg := CompactionConfig{
			Threshold:       rapid.Float64Range(0.0, 1.0).Draw(t, "threshold"),
			PreserveDurable: true,
		}

		durBefore := w.DurableTokens()
		result := Compact(&w, cfg)
		durAfter := result.DurableTokens()

		if durAfter != durBefore {
			t.Fatalf("DurableTokens changed: before=%d, after=%d", durBefore, durAfter)
		}

		// Also verify every durable segment survived.
		durCount := 0
		for _, s := range w.Segments {
			if s.Durable {
				durCount++
			}
		}
		resultDurCount := 0
		for _, s := range result.Segments {
			if s.Durable {
				resultDurCount++
			}
		}
		if resultDurCount != durCount {
			t.Fatalf("durable segment count changed: before=%d, after=%d", durCount, resultDurCount)
		}
	})
}

func TestProp_PipelineSortDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 10).Draw(t, "n")
		procs := make([]Processor, n)
		for i := range procs {
			procs[i] = Processor{
				Name:     rapid.StringN(1, 10, 10).Draw(t, "name"),
				Priority: rapid.IntRange(0, 100).Draw(t, "priority"),
				Fn:       func(_ StepContext) []Segment { return nil },
			}
		}
		p1 := NewPipeline(procs...)
		p2 := NewPipeline(procs...)

		if len(p1.processors) != len(p2.processors) {
			t.Fatalf("different lengths: %d vs %d", len(p1.processors), len(p2.processors))
		}
		for i := range p1.processors {
			if p1.processors[i].Name != p2.processors[i].Name {
				t.Fatalf("order differs at %d: %q vs %q", i, p1.processors[i].Name, p2.processors[i].Name)
			}
			if p1.processors[i].Priority != p2.processors[i].Priority {
				t.Fatalf("priority differs at %d: %d vs %d", i, p1.processors[i].Priority, p2.processors[i].Priority)
			}
		}
	})
}

func TestProp_CompactReducesUtilization(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Build a window that has at least one working segment and is over threshold.
		durSegs := rapid.SliceOfN(rapid.Custom(func(t *rapid.T) Segment {
			return Segment{
				Name:    rapid.StringN(1, 10, 10).Draw(t, "name"),
				Tokens:  rapid.IntRange(0, 1000).Draw(t, "tokens"),
				Durable: true,
			}
		}), 0, 5).Draw(t, "durSegs")

		workSegs := rapid.SliceOfN(rapid.Custom(func(t *rapid.T) Segment {
			return Segment{
				Name:    rapid.StringN(1, 10, 10).Draw(t, "name"),
				Tokens:  rapid.IntRange(1, 1000).Draw(t, "tokens"),
				Durable: false,
			}
		}), 1, 5).Draw(t, "workSegs")

		allSegs := append(durSegs, workSegs...)
		total := 0
		for _, s := range allSegs {
			total += s.Tokens
		}
		if total == 0 {
			return // skip trivial case
		}

		// Set maxTokens so utilization is above the threshold.
		threshold := 0.5
		maxTokens := int(float64(total) / (threshold + 0.1))
		if maxTokens <= 0 {
			maxTokens = 1
		}

		w := &Window{Segments: allSegs, MaxTokens: maxTokens}
		if w.Utilization() <= threshold {
			return // skip if the random draw doesn't exceed threshold
		}

		result := Compact(w, CompactionConfig{Threshold: threshold, PreserveDurable: true})

		// The documented invariant: UsedTokens() <= MaxTokens after Compact(),
		// unless durable segments alone exceed the budget.
		durableTokens := result.DurableTokens()
		if result.UsedTokens() > result.MaxTokens && durableTokens <= result.MaxTokens {
			t.Fatalf("Compact() violated UsedTokens <= MaxTokens: used=%d, max=%d (durable=%d)",
				result.UsedTokens(), result.MaxTokens, durableTokens)
		}

		// Also verify compaction never increases tokens.
		if result.UsedTokens() > w.UsedTokens() {
			t.Fatalf("Compact increased tokens: %d > %d", result.UsedTokens(), w.UsedTokens())
		}
	})
}

func TestProp_EstimateTokensNonNegative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "s")
		n := EstimateTokens(s)
		if n < 0 {
			t.Fatalf("EstimateTokens(%q) = %d, want >= 0", s, n)
		}
	})
}

func TestProp_AssembleDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 5).Draw(t, "n")
		procs := make([]Processor, n)
		for i := range procs {
			tok := rapid.IntRange(0, 1000).Draw(t, "tokens")
			dur := rapid.Bool().Draw(t, "durable")
			pName := rapid.StringN(1, 10, 10).Draw(t, "pname")
			pri := rapid.IntRange(0, 100).Draw(t, "priority")
			procs[i] = Processor{
				Name:     pName,
				Priority: pri,
				Fn: func(_ StepContext) []Segment {
					return []Segment{{Name: pName, Tokens: tok, Durable: dur}}
				},
			}
		}

		step := StepContext{
			StepName:        "test",
			Phase:           "plan",
			AvailableTokens: 10000,
		}

		p1 := NewPipeline(procs...)
		p2 := NewPipeline(procs...)
		w1 := p1.Assemble(step, 10000)
		w2 := p2.Assemble(step, 10000)

		if len(w1.Segments) != len(w2.Segments) {
			t.Fatalf("different segment counts: %d vs %d", len(w1.Segments), len(w2.Segments))
		}
		if w1.UsedTokens() != w2.UsedTokens() {
			t.Fatalf("different used tokens: %d vs %d", w1.UsedTokens(), w2.UsedTokens())
		}
		for i := range w1.Segments {
			if w1.Segments[i].Name != w2.Segments[i].Name {
				t.Fatalf("segment[%d] name differs: %q vs %q", i, w1.Segments[i].Name, w2.Segments[i].Name)
			}
			if w1.Segments[i].Tokens != w2.Segments[i].Tokens {
				t.Fatalf("segment[%d] tokens differ: %d vs %d", i, w1.Segments[i].Tokens, w2.Segments[i].Tokens)
			}
		}
	})
}
