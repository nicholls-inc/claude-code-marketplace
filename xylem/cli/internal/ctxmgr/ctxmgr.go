// Package ctxmgr provides named, ordered context processors that assemble
// context windows for agent steps. Context is built through a pipeline of
// processors — not ad-hoc string concatenation — so context assembly is
// observable, testable, and inspectable.
//
// Supports compaction (when utilization exceeds a threshold), strategy
// selection, and separation of durable state from working context. Durable
// segments (architectural decisions, unresolved bugs) survive compaction;
// working segments (verbose tool output, intermediate reasoning) do not.
package ctxmgr

import (
	"sort"
)

// Strategy identifies a context management strategy.
type Strategy string

const (
	// StrategyWrite adds new context without removing existing content.
	StrategyWrite Strategy = "write"
	// StrategySelect picks only relevant context for the step.
	StrategySelect Strategy = "select"
	// StrategyCompress reduces context by removing non-durable segments.
	StrategyCompress Strategy = "compress"
	// StrategyIsolate runs the step with minimal context.
	StrategyIsolate Strategy = "isolate"
)

// Default compaction settings.
const (
	DefaultCompactionThreshold = 0.95
	// tokenEstimateDivisor is the divisor used by EstimateTokens.
	tokenEstimateDivisor = 4
)

// Segment is a chunk of context within a window.
type Segment struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	Tokens   int    `json:"tokens"`
	Durable  bool   `json:"durable"`
	Source   string `json:"source"`
	Priority int    `json:"priority"`
}

// Window is an assembled context window containing ordered segments.
type Window struct {
	Segments  []Segment `json:"segments"`
	MaxTokens int       `json:"max_tokens"`
}

// UsedTokens returns the sum of all segment token counts.
func (w *Window) UsedTokens() int {
	total := 0
	for _, s := range w.Segments {
		total += s.Tokens
	}
	return total
}

// DurableTokens returns the sum of token counts for durable segments.
// INV: DurableTokens() is unchanged by Compact().
func (w *Window) DurableTokens() int {
	total := 0
	for _, s := range w.Segments {
		if s.Durable {
			total += s.Tokens
		}
	}
	return total
}

// Utilization returns the fraction of the token budget currently used.
// INV: Utilization is always in [0.0, 1.0] (clamped at both bounds).
func (w *Window) Utilization() float64 {
	if w.MaxTokens <= 0 {
		return 0.0
	}
	u := float64(w.UsedTokens()) / float64(w.MaxTokens)
	if u < 0.0 {
		return 0.0
	}
	if u > 1.0 {
		return 1.0
	}
	return u
}

// StepContext is the input provided to each processor in the pipeline.
type StepContext struct {
	StepName        string            `json:"step_name"`
	Phase           string            `json:"phase"`
	AvailableTokens int               `json:"available_tokens"`
	Metadata        map[string]string `json:"metadata"`
}

// ProcessorFunc generates segments for a given step.
type ProcessorFunc func(step StepContext) []Segment

// Processor is a named, prioritised context processor.
type Processor struct {
	Name     string        `json:"name"`
	Priority int           `json:"priority"`
	Fn       ProcessorFunc `json:"-"`
}

// Pipeline is an ordered chain of processors that assembles context windows.
// INV: Processor execution order matches priority (lower number = higher
// priority = runs first).
type Pipeline struct {
	processors []Processor
	metrics    Metrics
}

// Metrics tracks utilization statistics for a pipeline.
type Metrics struct {
	WindowFillRate       float64 `json:"window_fill_rate"`
	TotalTokensAssembled int     `json:"total_tokens_assembled"`
	DurableTokens        int     `json:"durable_tokens"`
	WorkingTokens        int     `json:"working_tokens"`
}

// CompactionConfig controls how compaction behaves.
type CompactionConfig struct {
	Threshold       float64 `json:"threshold"`
	PreserveDurable bool    `json:"preserve_durable"`
}

// NewPipeline creates a pipeline from the given processors, sorted by
// priority (ascending).
// INV: Pipeline output is deterministic for same input and same processors.
func NewPipeline(processors ...Processor) *Pipeline {
	p := &Pipeline{
		processors: make([]Processor, len(processors)),
	}
	copy(p.processors, processors)
	sort.SliceStable(p.processors, func(i, j int) bool {
		return p.processors[i].Priority < p.processors[j].Priority
	})
	return p
}

// AddProcessor inserts a processor into the pipeline, maintaining priority
// sort order.
func (p *Pipeline) AddProcessor(proc Processor) {
	p.processors = append(p.processors, proc)
	sort.SliceStable(p.processors, func(i, j int) bool {
		return p.processors[i].Priority < p.processors[j].Priority
	})
}

// Assemble runs all processors in priority order and builds a window.
// INV: Pipeline output is deterministic for same input and same processors.
func (p *Pipeline) Assemble(step StepContext, maxTokens int) *Window {
	w := &Window{MaxTokens: maxTokens}
	for _, proc := range p.processors {
		segments := proc.Fn(step)
		w.Segments = append(w.Segments, segments...)
	}

	p.metrics.TotalTokensAssembled += w.UsedTokens()
	p.metrics.DurableTokens = w.DurableTokens()
	p.metrics.WorkingTokens = w.UsedTokens() - w.DurableTokens()
	p.metrics.WindowFillRate = w.Utilization()
	return w
}

// Metrics returns the current utilization metrics.
func (p *Pipeline) Metrics() Metrics {
	return p.metrics
}

// Compact removes non-durable segments from a window when utilization exceeds
// the configured threshold.
//
// INV: Compaction preserves durable segments when PreserveDurable is true.
// INV: UsedTokens() <= MaxTokens after Compact(), unless durable segments alone exceed the budget.
// INV: DurableTokens() is unchanged by Compact() when PreserveDurable is true.
func Compact(window *Window, config CompactionConfig) *Window {
	if window.Utilization() <= config.Threshold {
		// Return a shallow copy to avoid aliasing: mutations to the returned
		// window must not affect the input.
		copied := *window
		copied.Segments = make([]Segment, len(window.Segments))
		copy(copied.Segments, window.Segments)
		return &copied
	}

	kept := make([]Segment, 0, len(window.Segments))
	for _, s := range window.Segments {
		if config.PreserveDurable && s.Durable {
			kept = append(kept, s)
		}
	}

	return &Window{
		Segments:  kept,
		MaxTokens: window.MaxTokens,
	}
}

// SelectStrategy picks a context management strategy based on current
// utilization, presence of durable state, and task complexity.
func SelectStrategy(utilization float64, hasDurableState bool, taskComplexity string) Strategy {
	if taskComplexity == "high" {
		return StrategyIsolate
	}
	if utilization > 0.8 {
		return StrategyCompress
	}
	if hasDurableState && utilization > 0.5 {
		return StrategySelect
	}
	return StrategyWrite
}

// EstimateTokens provides a rough token count for content using the
// len/4 heuristic.
func EstimateTokens(content string) int {
	return len(content) / tokenEstimateDivisor
}
