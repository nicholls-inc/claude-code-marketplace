package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfig controls how the OTel tracer provider is initialised.
type TracerConfig struct {
	// ServiceName identifies the instrumented service (e.g. "xylem").
	ServiceName string

	// ServiceVersion is the SemVer string of the service.
	ServiceVersion string

	// Endpoint is the OTLP collector address. When empty the tracer
	// falls back to a stdout exporter suitable for local development.
	Endpoint string

	// SampleRate controls the fraction of traces that are recorded.
	// Must be in the range [0.0, 1.0].
	SampleRate float64
}

// Tracer wraps an OTel TracerProvider and a named Tracer for xylem spans.
type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// SpanContext pairs an active span with its propagation context.
type SpanContext struct {
	span trace.Span
	ctx  context.Context
}

// DefaultTracerConfig returns a TracerConfig suitable for local development:
// service "xylem", 100% sampling, stdout export.
func DefaultTracerConfig() TracerConfig {
	return TracerConfig{
		ServiceName: "xylem",
		SampleRate:  1.0,
	}
}

// NewTracer creates and registers a global TracerProvider. When
// config.Endpoint is empty a stdout exporter is used; otherwise a future
// OTLP exporter path can be added here.
// INV: Returned Tracer is non-nil when err is nil.
func NewTracer(config TracerConfig) (*Tracer, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.TraceIDRatioBased(config.SampleRate)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(provider)

	t := provider.Tracer(config.ServiceName)

	return &Tracer{
		provider: provider,
		tracer:   t,
	}, nil
}

// Shutdown flushes pending spans and shuts down the TracerProvider.
func (t *Tracer) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}

// StartSpan begins a new span with the given name and attributes.
// The returned SpanContext must be ended by calling End.
func (t *Tracer) StartSpan(ctx context.Context, name string, attrs []SpanAttribute) SpanContext {
	ctx, span := t.tracer.Start(ctx, name)
	AttachSpanAttributes(span, attrs)
	return SpanContext{span: span, ctx: ctx}
}

// AddAttributes sets additional key-value pairs on the span.
func (sc SpanContext) AddAttributes(attrs []SpanAttribute) {
	AttachSpanAttributes(sc.span, attrs)
}

// End completes the span.
func (sc SpanContext) End() {
	sc.span.End()
}

// RecordError records an error on the span without ending it.
func (sc SpanContext) RecordError(err error) {
	sc.span.RecordError(err)
}

// Context returns the context carrying the span, suitable for propagation
// to child spans.
func (sc SpanContext) Context() context.Context {
	return sc.ctx
}

// AttachSpanAttributes converts a slice of SpanAttribute into OTel
// attribute.KeyValue pairs and sets them on the given span. This bridges
// the domain attribute schema defined in this package to the OTel SDK.
// INV: Each SpanAttribute produces exactly one attribute.KeyValue on the span.
func AttachSpanAttributes(span trace.Span, attrs []SpanAttribute) {
	if len(attrs) == 0 {
		return
	}
	kvs := make([]attribute.KeyValue, len(attrs))
	for i, a := range attrs {
		kvs[i] = attribute.String(a.Key, a.Value)
	}
	span.SetAttributes(kvs...)
}
