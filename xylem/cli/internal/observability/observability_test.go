package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/signal"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSignalSpanAttributesBasic(t *testing.T) {
	signals := []SignalData{
		{Type: "Repetition", Value: 0.75, Level: "Warning"},
	}
	attrs := SignalSpanAttributes(signals)
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attrs))
	}
	if attrs[0].Key != "signals.repetition.value" {
		t.Errorf("expected key signals.repetition.value, got %s", attrs[0].Key)
	}
	if attrs[0].Value != "0.7500" {
		t.Errorf("expected value 0.7500, got %s", attrs[0].Value)
	}
	if attrs[1].Key != "signals.repetition.level" {
		t.Errorf("expected key signals.repetition.level, got %s", attrs[1].Key)
	}
	if attrs[1].Value != "Warning" {
		t.Errorf("expected value Warning, got %s", attrs[1].Value)
	}
}

func TestSignalSpanAttributesEmpty(t *testing.T) {
	attrs := SignalSpanAttributes(nil)
	if len(attrs) != 0 {
		t.Fatalf("expected 0 attributes for empty input, got %d", len(attrs))
	}
}

func TestSignalSpanAttributesCount(t *testing.T) {
	signals := []SignalData{
		{Type: "Repetition", Value: 0.5, Level: "Normal"},
		{Type: "ToolFailureRate", Value: 0.1, Level: "Normal"},
		{Type: "ContextThrash", Value: 0.9, Level: "Critical"},
	}
	attrs := SignalSpanAttributes(signals)
	if len(attrs) != 2*len(signals) {
		t.Errorf("expected %d attributes, got %d", 2*len(signals), len(attrs))
	}
}

func TestAgentSpanAttributesContainsID(t *testing.T) {
	agent := AgentData{ID: "agent-1", Task: "fix-bug", Status: "running", TokensUsed: 500}
	attrs := AgentSpanAttributes(agent)
	found := false
	for _, a := range attrs {
		if a.Key == "agent.id" && a.Value == "agent-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected agent.id attribute with value agent-1")
	}
}

func TestAgentSpanAttributesContainsStatus(t *testing.T) {
	agent := AgentData{ID: "agent-1", Task: "fix-bug", Status: "running", TokensUsed: 500}
	attrs := AgentSpanAttributes(agent)
	found := false
	for _, a := range attrs {
		if a.Key == "agent.status" && a.Value == "running" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected agent.status attribute with value running")
	}
}

func TestMissionSpanAttributesContainsID(t *testing.T) {
	mission := MissionData{ID: "mission-42", Complexity: "high", Source: "github", TaskCount: 5}
	attrs := MissionSpanAttributes(mission)
	found := false
	for _, a := range attrs {
		if a.Key == "mission.id" && a.Value == "mission-42" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected mission.id attribute with value mission-42")
	}
}

func TestFormatAttributeKey(t *testing.T) {
	got := FormatAttributeKey("Signals", "Value")
	if got != "signals.value" {
		t.Errorf("expected signals.value, got %s", got)
	}
}

func TestFormatAttributeKeyLowercase(t *testing.T) {
	got := FormatAttributeKey("AGENT", "STATUS")
	if got != "agent.status" {
		t.Errorf("expected agent.status, got %s", got)
	}
}

// --- Signal bridge tests (from bridge.go) ---

func TestSignalToSignalData(t *testing.T) {
	sig := signal.Signal{Type: signal.Repetition, Value: 0.5, Level: signal.Warning}
	data := SignalToSignalData(sig)
	if data.Type != "Repetition" {
		t.Errorf("expected Type Repetition, got %s", data.Type)
	}
	if data.Value != 0.5 {
		t.Errorf("expected Value 0.5, got %f", data.Value)
	}
	if data.Level != "Warning" {
		t.Errorf("expected Level Warning, got %s", data.Level)
	}
}

func TestSignalSetToSignalData(t *testing.T) {
	set := signal.SignalSet{
		Signals: []signal.Signal{
			{Type: signal.Repetition, Value: 0.1, Level: signal.Normal},
			{Type: signal.ToolFailureRate, Value: 0.4, Level: signal.Warning},
			{Type: signal.EfficiencyScore, Value: 1.5, Level: signal.Normal},
			{Type: signal.ContextThrash, Value: 0.8, Level: signal.Critical},
			{Type: signal.TaskStall, Value: 0.0, Level: signal.Normal},
		},
	}
	data := SignalSetToSignalData(set)
	if len(data) != 5 {
		t.Fatalf("expected 5 SignalData, got %d", len(data))
	}
	// Verify order and types are preserved.
	expectedTypes := []string{"Repetition", "ToolFailureRate", "EfficiencyScore", "ContextThrash", "TaskStall"}
	for i, d := range data {
		if d.Type != expectedTypes[i] {
			t.Errorf("data[%d].Type = %s, want %s", i, d.Type, expectedTypes[i])
		}
	}
}

func TestSignalSetSpanAttributesCount(t *testing.T) {
	set := signal.SignalSet{
		Signals: []signal.Signal{
			{Type: signal.Repetition, Value: 0.1, Level: signal.Normal},
			{Type: signal.ToolFailureRate, Value: 0.4, Level: signal.Warning},
			{Type: signal.EfficiencyScore, Value: 1.5, Level: signal.Normal},
			{Type: signal.ContextThrash, Value: 0.8, Level: signal.Critical},
			{Type: signal.TaskStall, Value: 0.0, Level: signal.Normal},
		},
	}
	attrs := SignalSetSpanAttributes(set)
	// 5 signals * 2 attrs each + 4 aggregate = 14
	if len(attrs) != 14 {
		t.Errorf("expected 14 attributes, got %d", len(attrs))
	}
}

func TestSignalSetSpanAttributesHealth(t *testing.T) {
	set := signal.SignalSet{
		Signals: []signal.Signal{
			{Type: signal.Repetition, Value: 0.1, Level: signal.Normal},
			{Type: signal.ToolFailureRate, Value: 0.1, Level: signal.Normal},
		},
	}
	attrs := SignalSetSpanAttributes(set)
	expectedHealth := set.HealthString()
	found := false
	for _, a := range attrs {
		if a.Key == "signals.health" {
			found = true
			if a.Value != expectedHealth {
				t.Errorf("signals.health = %s, want %s", a.Value, expectedHealth)
			}
			break
		}
	}
	if !found {
		t.Error("signals.health attribute not found")
	}
}

func TestSignalSetSpanAttributesEmpty(t *testing.T) {
	set := signal.SignalSet{}
	attrs := SignalSetSpanAttributes(set)
	// Empty set: 0 per-signal attrs + 4 aggregate = 4
	if len(attrs) != 4 {
		t.Errorf("expected 4 attributes for empty set, got %d", len(attrs))
	}
}

// --- Tracer tests (from tracer.go) ---

// newTestTracer creates a Tracer backed by an in-memory span exporter so
// that tests never write to stdout or require an OTel collector.
func newTestTracer(t *testing.T) (*Tracer, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	tr := provider.Tracer("xylem-test")
	return &Tracer{provider: provider, tracer: tr}, exporter
}

func TestNewTracerDefault(t *testing.T) {
	tracer, err := NewTracer(DefaultTracerConfig())
	if err != nil {
		t.Fatalf("NewTracer returned error: %v", err)
	}
	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}
	_ = tracer.Shutdown(context.Background())
}

func TestNewTracerShutdown(t *testing.T) {
	tracer, err := NewTracer(DefaultTracerConfig())
	if err != nil {
		t.Fatalf("NewTracer returned error: %v", err)
	}
	if err := tracer.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestStartSpanAndEnd(t *testing.T) {
	tracer, _ := newTestTracer(t)
	defer tracer.Shutdown(context.Background())

	sc := tracer.StartSpan(context.Background(), "test-span", nil)
	sc.End()
	// No panic means success.
}

func TestStartSpanWithAttributes(t *testing.T) {
	tracer, exporter := newTestTracer(t)
	defer tracer.Shutdown(context.Background())

	attrs := SignalSpanAttributes([]SignalData{
		{Type: "Repetition", Value: 0.5, Level: "Warning"},
	})
	sc := tracer.StartSpan(context.Background(), "signal-span", attrs)
	sc.End()

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}
	span := spans[0]
	if span.Name != "signal-span" {
		t.Errorf("expected span name signal-span, got %s", span.Name)
	}
	// Signal attributes should be present.
	attrMap := make(map[string]string)
	for _, a := range span.Attributes {
		attrMap[string(a.Key)] = a.Value.AsString()
	}
	if v, ok := attrMap["signals.repetition.value"]; !ok || v != "0.5000" {
		t.Errorf("missing or wrong signals.repetition.value, got %q", v)
	}
	if v, ok := attrMap["signals.repetition.level"]; !ok || v != "Warning" {
		t.Errorf("missing or wrong signals.repetition.level, got %q", v)
	}
}

func TestAttachSpanAttributes(t *testing.T) {
	tracer, exporter := newTestTracer(t)
	defer tracer.Shutdown(context.Background())

	attrs := []SpanAttribute{
		{Key: "mission.id", Value: "m-1"},
		{Key: "mission.complexity", Value: "high"},
	}
	sc := tracer.StartSpan(context.Background(), "attach-test", nil)
	sc.AddAttributes(attrs)
	sc.End()

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}
	found := map[string]string{}
	for _, a := range spans[0].Attributes {
		found[string(a.Key)] = a.Value.AsString()
	}
	for _, a := range attrs {
		if got, ok := found[a.Key]; !ok {
			t.Errorf("attribute %q not found on span", a.Key)
		} else if got != a.Value {
			t.Errorf("attribute %q: expected %q, got %q", a.Key, a.Value, got)
		}
	}
}

func TestSpanContextRecordError(t *testing.T) {
	tracer, exporter := newTestTracer(t)
	defer tracer.Shutdown(context.Background())

	sc := tracer.StartSpan(context.Background(), "error-span", nil)
	sc.RecordError(errors.New("test error"))
	sc.End()

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}
	if len(spans[0].Events) == 0 {
		t.Error("expected at least one event (error) on span")
	}
	// Verify the error event has the expected exception message.
	foundErr := false
	for _, ev := range spans[0].Events {
		if ev.Name == "exception" {
			for _, a := range ev.Attributes {
				if a.Key == attribute.Key("exception.message") && a.Value.AsString() == "test error" {
					foundErr = true
				}
			}
		}
	}
	if !foundErr {
		t.Error("expected exception event with message 'test error'")
	}
}

func TestSpanContextReturnsContext(t *testing.T) {
	tracer, _ := newTestTracer(t)
	defer tracer.Shutdown(context.Background())

	parent := context.Background()
	sc := tracer.StartSpan(parent, "ctx-span", nil)
	defer sc.End()

	got := sc.Context()
	if got == nil {
		t.Fatal("expected non-nil context")
	}
	// The returned context should differ from the parent since it carries
	// the span.
	if got == parent {
		t.Error("expected context to differ from parent (should carry span)")
	}
}
