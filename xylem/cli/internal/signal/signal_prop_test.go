package signal

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// genTraceEvent generates a random TraceEvent.
func genTraceEvent() *rapid.Generator[TraceEvent] {
	return rapid.Custom(func(t *rapid.T) TraceEvent {
		eventType := rapid.SampledFrom([]string{
			"tool_call", "content", "compaction", "context_reset", "thinking",
		}).Draw(t, "type")
		toolName := rapid.SampledFrom([]string{
			"", "bash", "read", "write", "grep", "glob",
		}).Draw(t, "tool_name")
		content := rapid.SampledFrom([]string{
			"", "hello world", "the quick brown fox", "identical content",
			"some analysis output", "error: file not found",
		}).Draw(t, "content")
		return TraceEvent{
			Type:       eventType,
			Timestamp:  time.Now().Add(time.Duration(rapid.IntRange(0, 600).Draw(t, "offset")) * time.Second),
			ToolName:   toolName,
			Success:    rapid.Bool().Draw(t, "success"),
			TokensUsed: rapid.IntRange(0, 10000).Draw(t, "tokens"),
			Content:    content,
		}
	})
}

// genTraceEvents generates a slice of TraceEvents with ordered timestamps.
func genTraceEvents() *rapid.Generator[[]TraceEvent] {
	return rapid.Custom(func(t *rapid.T) []TraceEvent {
		n := rapid.IntRange(0, 50).Draw(t, "count")
		events := make([]TraceEvent, n)
		base := time.Now()
		for i := range n {
			events[i] = genTraceEvent().Draw(t, "event")
			events[i].Timestamp = base.Add(time.Duration(i) * time.Minute)
		}
		return events
	})
}

// genThresholdConfig generates a ThresholdConfig where Warning <= Critical.
func genThresholdConfig() *rapid.Generator[ThresholdConfig] {
	return rapid.Custom(func(t *rapid.T) ThresholdConfig {
		w := rapid.Float64Range(0.0, 1.0).Draw(t, "warning")
		// Critical must be >= warning.
		c := rapid.Float64Range(w, 1.0).Draw(t, "critical")
		return ThresholdConfig{Warning: w, Critical: c}
	})
}

// Property 1: For any generated trace events, all rate-based signals are in [0.0, 1.0].
func TestPropertyRateBasedSignalsBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genTraceEvents().Draw(t, "events")

		rep := ComputeRepetition(events)
		if rep < 0.0 || rep > 1.0 {
			t.Fatalf("ComputeRepetition out of [0,1]: %v", rep)
		}

		tfr := ComputeToolFailureRate(events)
		if tfr < 0.0 || tfr > 1.0 {
			t.Fatalf("ComputeToolFailureRate out of [0,1]: %v", tfr)
		}

		ct := ComputeContextThrash(events)
		if ct < 0.0 || ct > 1.0 {
			t.Fatalf("ComputeContextThrash out of [0,1]: %v", ct)
		}
	})
}

// Property 2: For any signal value and threshold config where warning <= critical,
// Classify is monotonic.
func TestPropertyClassifyMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tc := genThresholdConfig().Draw(t, "config")
		v1 := rapid.Float64Range(0.0, 2.0).Draw(t, "v1")
		v2 := rapid.Float64Range(v1, 2.0).Draw(t, "v2")

		l1 := Classify(v1, tc)
		l2 := Classify(v2, tc)

		if levelRank(l2) < levelRank(l1) {
			t.Fatalf("Classify not monotonic: Classify(%v)=%v but Classify(%v)=%v",
				v1, l1, v2, l2)
		}
	})
}

// Property 3: For any trace, Compute never panics.
func TestPropertyComputeNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genTraceEvents().Draw(t, "events")
		cfg := DefaultConfig()
		// This should never panic.
		ss := Compute(events, cfg)
		if len(ss.Signals) != 5 {
			t.Fatalf("expected 5 signals, got %d", len(ss.Signals))
		}
	})
}

// Property 4: For any signal set, Assess() is deterministic.
func TestPropertyAssessDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genTraceEvents().Draw(t, "events")
		cfg := DefaultConfig()
		ss := Compute(events, cfg)

		h1 := ss.Assess()
		h2 := ss.Assess()
		if h1 != h2 {
			t.Fatalf("Assess() not deterministic: %v != %v", h1, h2)
		}
	})
}

// Property 5: ComputeRepetition of all-identical content approaches 1.0.
func TestPropertyRepetitionIdenticalContent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		content := rapid.StringMatching(`[a-z]{10,50}`).Draw(t, "content")
		n := rapid.IntRange(5, 20).Draw(t, "count")

		events := make([]TraceEvent, n)
		base := time.Now()
		for i := range n {
			events[i] = TraceEvent{
				Content:   content,
				Timestamp: base.Add(time.Duration(i) * time.Second),
			}
		}

		rep := ComputeRepetition(events)
		// With all-identical content of sufficient length, similarity should
		// be high (the Dice coefficient of identical bigram sets is 1.0).
		if rep < 0.8 {
			t.Fatalf("ComputeRepetition with identical content = %v, expected >= 0.8", rep)
		}
	})
}

// Property 6: ComputeToolFailureRate with no tool events returns 0.0.
func TestPropertyToolFailureRateNoTools(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 20).Draw(t, "count")
		events := make([]TraceEvent, n)
		for i := range n {
			events[i] = TraceEvent{
				Content: rapid.String().Draw(t, "content"),
				// ToolName deliberately left empty.
			}
		}

		rate := ComputeToolFailureRate(events)
		if rate != 0.0 {
			t.Fatalf("ComputeToolFailureRate with no tools = %v, expected 0.0", rate)
		}
	})
}
