package signal

import (
	"testing"
	"time"
)

// --- ComputeRepetition tests ---

func TestComputeRepetition(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		events []TraceEvent
		wantLo float64
		wantHi float64
	}{
		{
			name:   "empty trace",
			events: nil,
			wantLo: 0.0,
			wantHi: 0.0,
		},
		{
			name: "single event",
			events: []TraceEvent{
				{Content: "hello world", Timestamp: now},
			},
			wantLo: 0.0,
			wantHi: 0.0,
		},
		{
			name: "no repetition",
			events: []TraceEvent{
				{Content: "the quick brown fox jumps", Timestamp: now},
				{Content: "completely different content here xyz", Timestamp: now.Add(time.Second)},
			},
			wantLo: 0.0,
			wantHi: 0.3,
		},
		{
			name: "full repetition",
			events: []TraceEvent{
				{Content: "identical content here", Timestamp: now},
				{Content: "identical content here", Timestamp: now.Add(time.Second)},
				{Content: "identical content here", Timestamp: now.Add(2 * time.Second)},
			},
			wantLo: 0.9,
			wantHi: 1.0,
		},
		{
			name: "partial repetition",
			events: []TraceEvent{
				{Content: "the quick brown fox", Timestamp: now},
				{Content: "the quick brown dog", Timestamp: now.Add(time.Second)},
				{Content: "something entirely new and novel", Timestamp: now.Add(2 * time.Second)},
			},
			wantLo: 0.2,
			wantHi: 0.8,
		},
		{
			name: "events with no content are skipped",
			events: []TraceEvent{
				{ToolName: "bash", Timestamp: now},
				{ToolName: "read", Timestamp: now.Add(time.Second)},
			},
			wantLo: 0.0,
			wantHi: 0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeRepetition(tc.events)
			if got < tc.wantLo || got > tc.wantHi {
				t.Errorf("ComputeRepetition() = %v, want in [%v, %v]", got, tc.wantLo, tc.wantHi)
			}
		})
	}
}

// --- ComputeToolFailureRate tests ---

func TestComputeToolFailureRate(t *testing.T) {
	tests := []struct {
		name   string
		events []TraceEvent
		want   float64
	}{
		{
			name:   "no events",
			events: nil,
			want:   0.0,
		},
		{
			name: "no tool events",
			events: []TraceEvent{
				{Content: "thinking..."},
				{Content: "more thinking"},
			},
			want: 0.0,
		},
		{
			name: "no failures",
			events: []TraceEvent{
				{ToolName: "bash", Success: true},
				{ToolName: "read", Success: true},
			},
			want: 0.0,
		},
		{
			name: "all failures",
			events: []TraceEvent{
				{ToolName: "bash", Success: false},
				{ToolName: "read", Success: false},
			},
			want: 1.0,
		},
		{
			name: "mixed results",
			events: []TraceEvent{
				{ToolName: "bash", Success: true},
				{ToolName: "read", Success: false},
				{ToolName: "bash", Success: true},
				{ToolName: "write", Success: false},
			},
			want: 0.5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeToolFailureRate(tc.events)
			if got != tc.want {
				t.Errorf("ComputeToolFailureRate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- ComputeEfficiency tests ---

func TestComputeEfficiency(t *testing.T) {
	tests := []struct {
		name     string
		events   []TraceEvent
		baseline int
		want     float64
	}{
		{
			name:     "empty trace",
			events:   nil,
			baseline: 10,
			want:     0.0,
		},
		{
			name:     "zero baseline",
			events:   []TraceEvent{{Content: "x"}},
			baseline: 0,
			want:     0.0,
		},
		{
			name:     "at baseline",
			events:   make([]TraceEvent, 10),
			baseline: 10,
			want:     1.0,
		},
		{
			name:     "2x baseline",
			events:   make([]TraceEvent, 20),
			baseline: 10,
			want:     2.0,
		},
		{
			name:     "below baseline",
			events:   make([]TraceEvent, 5),
			baseline: 10,
			want:     0.5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeEfficiency(tc.events, tc.baseline)
			if got != tc.want {
				t.Errorf("ComputeEfficiency() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- ComputeContextThrash tests ---

func TestComputeContextThrash(t *testing.T) {
	tests := []struct {
		name   string
		events []TraceEvent
		want   float64
	}{
		{
			name:   "empty trace",
			events: nil,
			want:   0.0,
		},
		{
			name: "no compactions",
			events: []TraceEvent{
				{Type: "tool_call"},
				{Type: "tool_call"},
				{Type: "tool_call"},
			},
			want: 0.0,
		},
		{
			name: "all compactions",
			events: []TraceEvent{
				{Type: "compaction"},
				{Type: "context_reset"},
			},
			want: 1.0,
		},
		{
			name: "frequent compactions",
			events: []TraceEvent{
				{Type: "tool_call"},
				{Type: "compaction"},
				{Type: "tool_call"},
				{Type: "compaction"},
				{Type: "tool_call"},
			},
			want: 0.4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeContextThrash(tc.events)
			if got != tc.want {
				t.Errorf("ComputeContextThrash() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- ComputeTaskStall tests ---

func TestComputeTaskStall(t *testing.T) {
	now := time.Now()
	window := 5 * time.Minute

	tests := []struct {
		name   string
		events []TraceEvent
		window time.Duration
		want   float64
	}{
		{
			name:   "empty trace",
			events: nil,
			window: window,
			want:   0.0,
		},
		{
			name: "single event",
			events: []TraceEvent{
				{Timestamp: now, ToolName: "bash", Success: true},
			},
			window: window,
			want:   0.0,
		},
		{
			name: "active progress within window",
			events: []TraceEvent{
				{Timestamp: now, ToolName: "bash", Success: true},
				{Timestamp: now.Add(6 * time.Minute), ToolName: "read", Success: true},
				{Timestamp: now.Add(8 * time.Minute), ToolName: "write", Success: true},
			},
			window: window,
			want:   0.0,
		},
		{
			name: "stalled no successful tool in final window",
			events: []TraceEvent{
				{Timestamp: now, ToolName: "bash", Success: true},
				{Timestamp: now.Add(6 * time.Minute), Content: "thinking..."},
				{Timestamp: now.Add(7 * time.Minute), Content: "still thinking"},
				{Timestamp: now.Add(11 * time.Minute), Content: "more thinking"},
			},
			window: window,
			want:   1.0,
		},
		{
			name: "span shorter than window",
			events: []TraceEvent{
				{Timestamp: now},
				{Timestamp: now.Add(2 * time.Minute)},
			},
			window: window,
			want:   0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeTaskStall(tc.events, tc.window)
			if got != tc.want {
				t.Errorf("ComputeTaskStall() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- Classify tests ---

func TestClassify(t *testing.T) {
	threshold := ThresholdConfig{Warning: 0.3, Critical: 0.7}

	tests := []struct {
		name  string
		value float64
		want  ThresholdLevel
	}{
		{"below warning", 0.1, Normal},
		{"at warning", 0.3, Warning},
		{"between warning and critical", 0.5, Warning},
		{"at critical", 0.7, Critical},
		{"above critical", 0.9, Critical},
		{"zero value", 0.0, Normal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.value, threshold)
			if got != tc.want {
				t.Errorf("Classify(%v) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

// --- Assess tests ---

func TestAssess(t *testing.T) {
	tests := []struct {
		name    string
		signals []Signal
		want    HealthLevel
	}{
		{
			name: "all normal",
			signals: []Signal{
				{Type: Repetition, Level: Normal},
				{Type: ToolFailureRate, Level: Normal},
				{Type: EfficiencyScore, Level: Normal},
				{Type: ContextThrash, Level: Normal},
				{Type: TaskStall, Level: Normal},
			},
			want: Excellent,
		},
		{
			name: "one warning",
			signals: []Signal{
				{Type: Repetition, Level: Warning},
				{Type: ToolFailureRate, Level: Normal},
				{Type: EfficiencyScore, Level: Normal},
			},
			want: Good,
		},
		{
			name: "two warnings",
			signals: []Signal{
				{Type: Repetition, Level: Warning},
				{Type: ToolFailureRate, Level: Warning},
				{Type: EfficiencyScore, Level: Normal},
			},
			want: Neutral,
		},
		{
			name: "one critical",
			signals: []Signal{
				{Type: Repetition, Level: Critical},
				{Type: ToolFailureRate, Level: Normal},
				{Type: EfficiencyScore, Level: Normal},
			},
			want: Poor,
		},
		{
			name: "two criticals",
			signals: []Signal{
				{Type: Repetition, Level: Critical},
				{Type: ToolFailureRate, Level: Critical},
				{Type: EfficiencyScore, Level: Normal},
			},
			want: Severe,
		},
		{
			name:    "empty signals",
			signals: nil,
			want:    Excellent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ss := SignalSet{Signals: tc.signals}
			got := ss.Assess()
			if got != tc.want {
				t.Errorf("Assess() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- Get tests ---

func TestSignalSetGet(t *testing.T) {
	ss := SignalSet{
		Signals: []Signal{
			{Type: Repetition, Value: 0.5, Level: Warning},
			{Type: ToolFailureRate, Value: 0.1, Level: Normal},
		},
	}

	sig, ok := ss.Get(Repetition)
	if !ok {
		t.Fatal("expected to find Repetition signal")
	}
	if sig.Value != 0.5 {
		t.Errorf("expected value 0.5, got %v", sig.Value)
	}

	_, ok = ss.Get(TaskStall)
	if ok {
		t.Error("expected TaskStall to not be found")
	}
}

// --- Worst tests ---

func TestSignalSetWorst(t *testing.T) {
	tests := []struct {
		name    string
		signals []Signal
		want    ThresholdLevel
	}{
		{
			name:    "empty",
			signals: nil,
			want:    "",
		},
		{
			name: "all normal",
			signals: []Signal{
				{Type: Repetition, Level: Normal},
				{Type: ToolFailureRate, Level: Normal},
			},
			want: Normal,
		},
		{
			name: "mixed levels",
			signals: []Signal{
				{Type: Repetition, Level: Normal},
				{Type: ToolFailureRate, Level: Critical},
				{Type: EfficiencyScore, Level: Warning},
			},
			want: Critical,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ss := SignalSet{Signals: tc.signals}
			got := ss.Worst()
			if got.Level != tc.want {
				t.Errorf("Worst().Level = %v, want %v", got.Level, tc.want)
			}
		})
	}
}

// --- DefaultConfig tests ---

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	expectedTypes := []SignalType{Repetition, ToolFailureRate, EfficiencyScore, ContextThrash, TaskStall}
	for _, st := range expectedTypes {
		tc, ok := cfg.Thresholds[st]
		if !ok {
			t.Errorf("DefaultConfig missing threshold for %v", st)
			continue
		}
		if tc.Warning <= 0 {
			t.Errorf("DefaultConfig threshold %v has non-positive warning: %v", st, tc.Warning)
		}
		if tc.Critical <= 0 {
			t.Errorf("DefaultConfig threshold %v has non-positive critical: %v", st, tc.Critical)
		}
		if tc.Warning > tc.Critical {
			t.Errorf("DefaultConfig threshold %v: warning (%v) > critical (%v)", st, tc.Warning, tc.Critical)
		}
	}

	if cfg.StallWindow <= 0 {
		t.Errorf("DefaultConfig StallWindow should be positive, got %v", cfg.StallWindow)
	}
	if cfg.EfficiencyBaseline <= 0 {
		t.Errorf("DefaultConfig EfficiencyBaseline should be positive, got %v", cfg.EfficiencyBaseline)
	}
}

// --- Compute integration test ---

func TestComputeIntegration(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()

	events := []TraceEvent{
		{Timestamp: now, ToolName: "bash", Success: true, Content: "hello world"},
		{Timestamp: now.Add(time.Minute), ToolName: "read", Success: true, Content: "different content"},
		{Timestamp: now.Add(2 * time.Minute), ToolName: "write", Success: false, Content: "another thing"},
	}

	ss := Compute(events, cfg)
	if len(ss.Signals) != 5 {
		t.Fatalf("expected 5 signals, got %d", len(ss.Signals))
	}

	for _, sig := range ss.Signals {
		if sig.Value < 0 {
			t.Errorf("signal %v has negative value: %v", sig.Type, sig.Value)
		}
	}
}
