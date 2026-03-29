package ctxmgr

import (
	"strings"
	"testing"
)

// seg is a test helper that builds a Segment with sensible defaults.
func seg(t *testing.T, name string, tokens int, durable bool) Segment {
	t.Helper()
	return Segment{
		Name:    name,
		Content: strings.Repeat("x", tokens*4),
		Tokens:  tokens,
		Durable: durable,
		Source:  "test",
	}
}

// staticProcessor returns a Processor whose Fn always yields the given segments.
func staticProcessor(t *testing.T, name string, priority int, segs ...Segment) Processor {
	t.Helper()
	return Processor{
		Name:     name,
		Priority: priority,
		Fn:       func(_ StepContext) []Segment { return segs },
	}
}

func TestNewPipeline(t *testing.T) {
	tests := []struct {
		name       string
		processors []Processor
		wantOrder  []string
	}{
		{
			name:       "empty pipeline",
			processors: nil,
			wantOrder:  nil,
		},
		{
			name: "single processor",
			processors: []Processor{
				{Name: "a", Priority: 1},
			},
			wantOrder: []string{"a"},
		},
		{
			name: "multiple processors sorted by priority",
			processors: []Processor{
				{Name: "c", Priority: 3},
				{Name: "a", Priority: 1},
				{Name: "b", Priority: 2},
			},
			wantOrder: []string{"a", "b", "c"},
		},
		{
			name: "equal priority preserves insertion order",
			processors: []Processor{
				{Name: "first", Priority: 1},
				{Name: "second", Priority: 1},
			},
			wantOrder: []string{"first", "second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPipeline(tt.processors...)
			if len(p.processors) != len(tt.wantOrder) {
				t.Fatalf("got %d processors, want %d", len(p.processors), len(tt.wantOrder))
			}
			for i, want := range tt.wantOrder {
				if p.processors[i].Name != want {
					t.Errorf("processor[%d] = %q, want %q", i, p.processors[i].Name, want)
				}
			}
		})
	}
}

func TestAddProcessor(t *testing.T) {
	p := NewPipeline(
		Processor{Name: "a", Priority: 1},
		Processor{Name: "c", Priority: 3},
	)
	p.AddProcessor(Processor{Name: "b", Priority: 2})

	want := []string{"a", "b", "c"}
	if len(p.processors) != len(want) {
		t.Fatalf("got %d processors, want %d", len(p.processors), len(want))
	}
	for i, w := range want {
		if p.processors[i].Name != w {
			t.Errorf("processor[%d] = %q, want %q", i, p.processors[i].Name, w)
		}
	}
}

func TestAssemble(t *testing.T) {
	tests := []struct {
		name           string
		processors     []Processor
		maxTokens      int
		wantSegments   int
		wantUsedTokens int
	}{
		{
			name:           "empty pipeline produces empty window",
			processors:     nil,
			maxTokens:      1000,
			wantSegments:   0,
			wantUsedTokens: 0,
		},
		{
			name: "single processor contributes segments",
			processors: []Processor{
				staticProcessor(t, "p1", 1, seg(t, "s1", 100, false)),
			},
			maxTokens:      1000,
			wantSegments:   1,
			wantUsedTokens: 100,
		},
		{
			name: "multiple processors combine segments",
			processors: []Processor{
				staticProcessor(t, "p1", 1, seg(t, "s1", 100, false)),
				staticProcessor(t, "p2", 2, seg(t, "s2", 200, true)),
			},
			maxTokens:      1000,
			wantSegments:   2,
			wantUsedTokens: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPipeline(tt.processors...)
			step := StepContext{StepName: "test-step", Phase: "plan", AvailableTokens: tt.maxTokens}
			w := p.Assemble(step, tt.maxTokens)

			if len(w.Segments) != tt.wantSegments {
				t.Errorf("got %d segments, want %d", len(w.Segments), tt.wantSegments)
			}
			if w.UsedTokens() != tt.wantUsedTokens {
				t.Errorf("UsedTokens() = %d, want %d", w.UsedTokens(), tt.wantUsedTokens)
			}
			if w.MaxTokens != tt.maxTokens {
				t.Errorf("MaxTokens = %d, want %d", w.MaxTokens, tt.maxTokens)
			}
		})
	}
}

func TestWindowUtilization(t *testing.T) {
	tests := []struct {
		name      string
		window    Window
		wantUtil  float64
		wantUsed  int
		wantDur   int
		tolerance float64
	}{
		{
			name:      "empty window",
			window:    Window{MaxTokens: 1000},
			wantUtil:  0.0,
			wantUsed:  0,
			wantDur:   0,
			tolerance: 0.001,
		},
		{
			name: "half full",
			window: Window{
				Segments:  []Segment{{Tokens: 500, Durable: false}},
				MaxTokens: 1000,
			},
			wantUtil:  0.5,
			wantUsed:  500,
			wantDur:   0,
			tolerance: 0.001,
		},
		{
			name: "fully used",
			window: Window{
				Segments:  []Segment{{Tokens: 1000, Durable: true}},
				MaxTokens: 1000,
			},
			wantUtil:  1.0,
			wantUsed:  1000,
			wantDur:   1000,
			tolerance: 0.001,
		},
		{
			name: "over budget is clamped to 1.0",
			window: Window{
				Segments:  []Segment{{Tokens: 1500}},
				MaxTokens: 1000,
			},
			wantUtil:  1.0,
			wantUsed:  1500,
			wantDur:   0,
			tolerance: 0.001,
		},
		{
			name:      "zero max tokens returns 0.0",
			window:    Window{MaxTokens: 0},
			wantUtil:  0.0,
			wantUsed:  0,
			wantDur:   0,
			tolerance: 0.001,
		},
		{
			name: "mixed durable and working",
			window: Window{
				Segments: []Segment{
					{Tokens: 300, Durable: true},
					{Tokens: 200, Durable: false},
				},
				MaxTokens: 1000,
			},
			wantUtil:  0.5,
			wantUsed:  500,
			wantDur:   300,
			tolerance: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &tt.window
			if diff := w.Utilization() - tt.wantUtil; diff > tt.tolerance || diff < -tt.tolerance {
				t.Errorf("Utilization() = %f, want %f", w.Utilization(), tt.wantUtil)
			}
			if w.UsedTokens() != tt.wantUsed {
				t.Errorf("UsedTokens() = %d, want %d", w.UsedTokens(), tt.wantUsed)
			}
			if w.DurableTokens() != tt.wantDur {
				t.Errorf("DurableTokens() = %d, want %d", w.DurableTokens(), tt.wantDur)
			}
		})
	}
}

func TestCompact(t *testing.T) {
	tests := []struct {
		name           string
		window         Window
		config         CompactionConfig
		wantSegments   int
		wantDurTokens  int
		wantUsedTokens int
	}{
		{
			name: "below threshold unchanged",
			window: Window{
				Segments:  []Segment{{Name: "a", Tokens: 100, Durable: false}},
				MaxTokens: 1000,
			},
			config:         CompactionConfig{Threshold: 0.95, PreserveDurable: true},
			wantSegments:   1,
			wantDurTokens:  0,
			wantUsedTokens: 100,
		},
		{
			name: "above threshold removes working segments",
			window: Window{
				Segments: []Segment{
					{Name: "durable", Tokens: 100, Durable: true},
					{Name: "working", Tokens: 900, Durable: false},
				},
				MaxTokens: 1000,
			},
			config:         CompactionConfig{Threshold: 0.95, PreserveDurable: true},
			wantSegments:   1,
			wantDurTokens:  100,
			wantUsedTokens: 100,
		},
		{
			name: "preserves all durable segments",
			window: Window{
				Segments: []Segment{
					{Name: "d1", Tokens: 300, Durable: true},
					{Name: "d2", Tokens: 300, Durable: true},
					{Name: "w1", Tokens: 500, Durable: false},
				},
				MaxTokens: 1000,
			},
			config:         CompactionConfig{Threshold: 0.5, PreserveDurable: true},
			wantSegments:   2,
			wantDurTokens:  600,
			wantUsedTokens: 600,
		},
		{
			name: "all-durable window no change possible",
			window: Window{
				Segments: []Segment{
					{Name: "d1", Tokens: 500, Durable: true},
					{Name: "d2", Tokens: 500, Durable: true},
				},
				MaxTokens: 1000,
			},
			config:         CompactionConfig{Threshold: 0.5, PreserveDurable: true},
			wantSegments:   2,
			wantDurTokens:  1000,
			wantUsedTokens: 1000,
		},
		{
			name: "at exact threshold unchanged",
			window: Window{
				Segments:  []Segment{{Name: "a", Tokens: 950, Durable: false}},
				MaxTokens: 1000,
			},
			config:         CompactionConfig{Threshold: 0.95, PreserveDurable: true},
			wantSegments:   1,
			wantDurTokens:  0,
			wantUsedTokens: 950,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Compact(&tt.window, tt.config)
			if len(result.Segments) != tt.wantSegments {
				t.Errorf("got %d segments, want %d", len(result.Segments), tt.wantSegments)
			}
			if result.DurableTokens() != tt.wantDurTokens {
				t.Errorf("DurableTokens() = %d, want %d", result.DurableTokens(), tt.wantDurTokens)
			}
			if result.UsedTokens() != tt.wantUsedTokens {
				t.Errorf("UsedTokens() = %d, want %d", result.UsedTokens(), tt.wantUsedTokens)
			}
		})
	}
}

func TestSelectStrategy(t *testing.T) {
	tests := []struct {
		name           string
		utilization    float64
		hasDurable     bool
		taskComplexity string
		want           Strategy
	}{
		{
			name:           "low utilization returns Write",
			utilization:    0.2,
			hasDurable:     false,
			taskComplexity: "low",
			want:           StrategyWrite,
		},
		{
			name:           "high utilization returns Compress",
			utilization:    0.9,
			hasDurable:     false,
			taskComplexity: "low",
			want:           StrategyCompress,
		},
		{
			name:           "complex task returns Isolate",
			utilization:    0.3,
			hasDurable:     false,
			taskComplexity: "high",
			want:           StrategyIsolate,
		},
		{
			name:           "durable state and medium utilization returns Select",
			utilization:    0.6,
			hasDurable:     true,
			taskComplexity: "low",
			want:           StrategySelect,
		},
		{
			name:           "durable state low utilization returns Write",
			utilization:    0.3,
			hasDurable:     true,
			taskComplexity: "low",
			want:           StrategyWrite,
		},
		{
			name:           "high complexity overrides high utilization",
			utilization:    0.95,
			hasDurable:     true,
			taskComplexity: "high",
			want:           StrategyIsolate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectStrategy(tt.utilization, tt.hasDurable, tt.taskComplexity)
			if got != tt.want {
				t.Errorf("SelectStrategy(%f, %v, %q) = %q, want %q",
					tt.utilization, tt.hasDurable, tt.taskComplexity, got, tt.want)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{name: "empty string", content: "", want: 0},
		{name: "short string", content: "hi", want: 0},
		{name: "four chars", content: "abcd", want: 1},
		{name: "long string", content: strings.Repeat("a", 400), want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.content)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.content, got, tt.want)
			}
		})
	}
}

func TestMetrics(t *testing.T) {
	durSeg := seg(t, "durable", 300, true)
	workSeg := seg(t, "working", 200, false)

	p := NewPipeline(
		staticProcessor(t, "p1", 1, durSeg),
		staticProcessor(t, "p2", 2, workSeg),
	)
	step := StepContext{StepName: "s1", Phase: "plan", AvailableTokens: 1000}
	p.Assemble(step, 1000)

	m := p.Metrics()
	if m.TotalTokensAssembled != 500 {
		t.Errorf("TotalTokensAssembled = %d, want 500", m.TotalTokensAssembled)
	}
	if m.DurableTokens != 300 {
		t.Errorf("DurableTokens = %d, want 300", m.DurableTokens)
	}
	if m.WorkingTokens != 200 {
		t.Errorf("WorkingTokens = %d, want 200", m.WorkingTokens)
	}

	tolerance := 0.001
	if diff := m.WindowFillRate - 0.5; diff > tolerance || diff < -tolerance {
		t.Errorf("WindowFillRate = %f, want 0.5", m.WindowFillRate)
	}

	// Second assembly accumulates TotalTokensAssembled.
	p.Assemble(step, 1000)
	m2 := p.Metrics()
	if m2.TotalTokensAssembled != 1000 {
		t.Errorf("TotalTokensAssembled after second call = %d, want 1000", m2.TotalTokensAssembled)
	}
}

func TestCompactPreservesDurableTokenCount(t *testing.T) {
	w := &Window{
		Segments: []Segment{
			{Name: "d1", Tokens: 400, Durable: true},
			{Name: "w1", Tokens: 600, Durable: false},
		},
		MaxTokens: 1000,
	}
	durBefore := w.DurableTokens()
	result := Compact(w, CompactionConfig{Threshold: 0.5, PreserveDurable: true})
	durAfter := result.DurableTokens()
	if durBefore != durAfter {
		t.Errorf("DurableTokens changed: before=%d, after=%d", durBefore, durAfter)
	}
}
