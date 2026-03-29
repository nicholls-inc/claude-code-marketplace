// Package signal provides lightweight heuristics computed from agent traces
// without LLM involvement. These are cheap, fast first-pass assessments of
// agent health used to decide whether to trigger more expensive evaluation.
package signal

import (
	"time"
)

// SignalType identifies a specific behavioral signal extracted from a trace.
type SignalType string

const (
	// Repetition measures bigram similarity in agent content, where 1.0 means
	// fully repetitive output.
	Repetition SignalType = "Repetition"
	// ToolFailureRate measures the ratio of failed tool calls to total tool calls.
	ToolFailureRate SignalType = "ToolFailureRate"
	// EfficiencyScore measures actual turns relative to an expected baseline,
	// where values above 1.0 indicate inefficiency.
	EfficiencyScore SignalType = "EfficiencyScore"
	// ContextThrash measures the frequency of context compaction or reset events.
	ContextThrash SignalType = "ContextThrash"
	// TaskStall detects whether the agent has made no progress within a time window.
	TaskStall SignalType = "TaskStall"
)

// ThresholdLevel classifies a signal value into one of three tiers.
type ThresholdLevel string

const (
	// Normal indicates the signal is within acceptable range; no action needed.
	Normal ThresholdLevel = "Normal"
	// Warning indicates the signal has crossed a threshold that should trigger
	// further evaluation (e.g. an LLM-based assessment).
	Warning ThresholdLevel = "Warning"
	// Critical indicates the signal has crossed a threshold that should halt
	// execution and alert the operator.
	Critical ThresholdLevel = "Critical"
)

// HealthLevel represents the aggregate health assessment of an agent session.
type HealthLevel string

const (
	// Excellent means all signals are normal with no concerns.
	Excellent HealthLevel = "Excellent"
	// Good means signals are mostly normal with minor concerns.
	Good HealthLevel = "Good"
	// Neutral means there are some warning signals but nothing critical.
	Neutral HealthLevel = "Neutral"
	// Poor means there are multiple warnings or a single critical signal.
	Poor HealthLevel = "Poor"
	// Severe means there are multiple critical signals requiring immediate action.
	Severe HealthLevel = "Severe"
)

// TraceEvent represents a single event from an agent execution trace.
type TraceEvent struct {
	Type       string    `json:"type"`
	Timestamp  time.Time `json:"timestamp"`
	ToolName   string    `json:"tool_name,omitempty"`
	Success    bool      `json:"success"`
	TokensUsed int       `json:"tokens_used"`
	Content    string    `json:"content,omitempty"`
}

// Signal holds a computed signal value with its classification.
type Signal struct {
	Type  SignalType     `json:"type"`
	Value float64        `json:"value"`
	Level ThresholdLevel `json:"level"`
}

// ThresholdConfig defines the warning and critical thresholds for a signal.
// INV: Warning <= Critical for monotonic classification.
type ThresholdConfig struct {
	Warning  float64 `json:"warning"`
	Critical float64 `json:"critical"`
}

// SignalConfig holds configuration for all signal computations.
type SignalConfig struct {
	Thresholds         map[SignalType]ThresholdConfig `json:"thresholds"`
	StallWindow        time.Duration                 `json:"stall_window"`
	EfficiencyBaseline int                           `json:"efficiency_baseline"`
}

// SignalSet is a collection of computed signals with helper methods.
type SignalSet struct {
	Signals []Signal `json:"signals"`
}

// DefaultConfig returns a SignalConfig with sensible defaults for all thresholds.
func DefaultConfig() SignalConfig {
	return SignalConfig{
		Thresholds: map[SignalType]ThresholdConfig{
			Repetition:      {Warning: 0.3, Critical: 0.7},
			ToolFailureRate: {Warning: 0.3, Critical: 0.6},
			EfficiencyScore: {Warning: 2.0, Critical: 4.0},
			ContextThrash:   {Warning: 0.3, Critical: 0.6},
			TaskStall:       {Warning: 0.5, Critical: 1.0},
		},
		StallWindow:        5 * time.Minute,
		EfficiencyBaseline: 10,
	}
}

// Compute calculates all signals from a trace and returns them as a SignalSet.
// INV: Never panics, even for empty traces.
// INV: All signal values are non-negative.
func Compute(events []TraceEvent, config SignalConfig) SignalSet {
	rep := ComputeRepetition(events)
	tfr := ComputeToolFailureRate(events)
	eff := ComputeEfficiency(events, config.EfficiencyBaseline)
	ct := ComputeContextThrash(events)
	ts := ComputeTaskStall(events, config.StallWindow)

	signals := []Signal{
		{Type: Repetition, Value: rep, Level: Classify(rep, config.Thresholds[Repetition])},
		{Type: ToolFailureRate, Value: tfr, Level: Classify(tfr, config.Thresholds[ToolFailureRate])},
		{Type: EfficiencyScore, Value: eff, Level: Classify(eff, config.Thresholds[EfficiencyScore])},
		{Type: ContextThrash, Value: ct, Level: Classify(ct, config.Thresholds[ContextThrash])},
		{Type: TaskStall, Value: ts, Level: Classify(ts, config.Thresholds[TaskStall])},
	}

	return SignalSet{Signals: signals}
}

// ComputeRepetition calculates a bigram similarity score from event content.
// Returns a value in [0,1] where 1.0 means fully repetitive.
// INV: Return value is in [0.0, 1.0].
// INV: Empty trace returns 0.0.
func ComputeRepetition(events []TraceEvent) float64 {
	var contents []string
	for _, e := range events {
		if e.Content != "" {
			contents = append(contents, e.Content)
		}
	}
	if len(contents) < 2 {
		return 0.0
	}

	// Build bigrams from each content string and count how many consecutive
	// pairs share bigrams.
	totalPairs := 0
	matchingPairs := 0
	for i := 1; i < len(contents); i++ {
		prev := bigrams(contents[i-1])
		curr := bigrams(contents[i])
		if len(prev) == 0 || len(curr) == 0 {
			continue
		}
		totalPairs++
		sim := bigramSimilarity(prev, curr)
		if sim > 0.5 {
			matchingPairs++
		}
	}
	if totalPairs == 0 {
		return 0.0
	}
	return float64(matchingPairs) / float64(totalPairs)
}

// bigrams returns the set of character bigrams in s.
func bigrams(s string) map[string]int {
	runes := []rune(s)
	if len(runes) < 2 {
		return nil
	}
	m := make(map[string]int, len(runes)-1)
	for i := 0; i < len(runes)-1; i++ {
		m[string(runes[i:i+2])]++
	}
	return m
}

// bigramSimilarity computes the Dice coefficient between two bigram multisets.
func bigramSimilarity(a, b map[string]int) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0.0
	}
	intersection := 0
	for k, v := range a {
		if bv, ok := b[k]; ok {
			if v < bv {
				intersection += v
			} else {
				intersection += bv
			}
		}
	}
	totalA := 0
	for _, v := range a {
		totalA += v
	}
	totalB := 0
	for _, v := range b {
		totalB += v
	}
	denom := totalA + totalB
	if denom == 0 {
		return 0.0
	}
	return float64(2*intersection) / float64(denom)
}

// ComputeToolFailureRate calculates the ratio of failed tool calls to total
// tool calls. Returns a value in [0,1].
// INV: Return value is in [0.0, 1.0].
// INV: Returns 0.0 when there are no tool events.
func ComputeToolFailureRate(events []TraceEvent) float64 {
	total := 0
	failures := 0
	for _, e := range events {
		if e.ToolName == "" {
			continue
		}
		total++
		if !e.Success {
			failures++
		}
	}
	if total == 0 {
		return 0.0
	}
	return float64(failures) / float64(total)
}

// ComputeEfficiency calculates the ratio of actual turns to the expected
// baseline. Values above 1.0 indicate the agent is using more turns than
// expected; values below 1.0 mean it is ahead of schedule.
// INV: Return value is non-negative.
// INV: Returns 0.0 for empty traces or zero baseline.
func ComputeEfficiency(events []TraceEvent, baseline int) float64 {
	if baseline <= 0 || len(events) == 0 {
		return 0.0
	}
	return float64(len(events)) / float64(baseline)
}

// ComputeContextThrash calculates the frequency of context compaction or reset
// events relative to total events. Returns a value in [0,1].
// INV: Return value is in [0.0, 1.0].
// INV: Returns 0.0 for empty traces.
func ComputeContextThrash(events []TraceEvent) float64 {
	if len(events) == 0 {
		return 0.0
	}
	compactions := 0
	for _, e := range events {
		if e.Type == "compaction" || e.Type == "context_reset" {
			compactions++
		}
	}
	return float64(compactions) / float64(len(events))
}

// ComputeTaskStall detects whether the agent has made no meaningful progress
// within the given time window. Returns 1.0 if stalled, 0.0 otherwise.
// INV: Return value is 0.0 or 1.0.
// INV: Returns 0.0 for empty traces or traces shorter than the window.
func ComputeTaskStall(events []TraceEvent, window time.Duration) float64 {
	if len(events) < 2 {
		return 0.0
	}

	// Find the time span of events.
	first := events[0].Timestamp
	last := events[len(events)-1].Timestamp
	span := last.Sub(first)

	if span < window {
		return 0.0
	}

	// Check whether any successful tool call occurred in the final window.
	cutoff := last.Add(-window)
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Timestamp.Before(cutoff) {
			break
		}
		if events[i].Success && events[i].ToolName != "" {
			return 0.0
		}
	}
	return 1.0
}

// Classify determines the ThresholdLevel for a signal value against the given
// threshold configuration.
// INV: Monotonic — if value < warning threshold, returns Normal; if value >=
//
//	critical threshold, returns Critical; otherwise Warning.
//
// INV: Never returns Critical if value < critical threshold.
func Classify(value float64, threshold ThresholdConfig) ThresholdLevel {
	if value >= threshold.Critical {
		return Critical
	}
	if value >= threshold.Warning {
		return Warning
	}
	return Normal
}

// Get returns the signal of the given type and true if found, or a zero Signal
// and false if not present.
func (s SignalSet) Get(t SignalType) (Signal, bool) {
	for _, sig := range s.Signals {
		if sig.Type == t {
			return sig, true
		}
	}
	return Signal{}, false
}

// Worst returns the signal with the highest severity. If multiple signals share
// the worst level, the first one encountered is returned.
// INV: For an empty SignalSet, returns a zero-value Signal.
func (s SignalSet) Worst() Signal {
	if len(s.Signals) == 0 {
		return Signal{}
	}
	worst := s.Signals[0]
	for _, sig := range s.Signals[1:] {
		if levelRank(sig.Level) > levelRank(worst.Level) {
			worst = sig
		}
	}
	return worst
}

// Assess aggregates the signal set into an overall HealthLevel.
// INV: Deterministic — same SignalSet always produces the same HealthLevel.
func (s SignalSet) Assess() HealthLevel {
	criticals := 0
	warnings := 0
	for _, sig := range s.Signals {
		switch sig.Level {
		case Critical:
			criticals++
		case Warning:
			warnings++
		}
	}

	switch {
	case criticals >= 2:
		return Severe
	case criticals == 1:
		return Poor
	case warnings >= 2:
		return Neutral
	case warnings == 1:
		return Good
	default:
		return Excellent
	}
}

// levelRank returns a numeric rank for a ThresholdLevel for comparison.
func levelRank(l ThresholdLevel) int {
	switch l {
	case Critical:
		return 2
	case Warning:
		return 1
	default:
		return 0
	}
}
