package observability

import (
	"fmt"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/signal"
)

// SignalToSignalData converts a single signal.Signal into the observability
// package's SignalData type by mapping typed enums to their string
// representations.
// INV: Output Type == string(sig.Type), Value == sig.Value, Level == string(sig.Level).
func SignalToSignalData(sig signal.Signal) SignalData {
	return SignalData{
		Type:  string(sig.Type),
		Value: sig.Value,
		Level: string(sig.Level),
	}
}

// SignalSetToSignalData converts all signals in a SignalSet to a slice of
// SignalData suitable for attribute extraction.
// INV: len(result) == len(set.Signals).
func SignalSetToSignalData(set signal.SignalSet) []SignalData {
	data := make([]SignalData, len(set.Signals))
	for i, sig := range set.Signals {
		data[i] = SignalToSignalData(sig)
	}
	return data
}

// SignalSetSpanAttributes converts a SignalSet into a complete set of span
// attributes including per-signal value/level pairs and aggregate health
// attributes. This composes SignalSetToSignalData and SignalSpanAttributes,
// then appends 4 aggregate attributes.
// INV: Output contains exactly 2*len(set.Signals) + 4 attributes.
func SignalSetSpanAttributes(set signal.SignalSet) []SpanAttribute {
	data := SignalSetToSignalData(set)
	attrs := SignalSpanAttributes(data)

	worst := set.Worst()
	attrs = append(attrs,
		SpanAttribute{Key: "signals.health", Value: set.HealthString()},
		SpanAttribute{Key: "signals.should_evaluate", Value: fmt.Sprintf("%t", set.ShouldEvaluate())},
		SpanAttribute{Key: "signals.worst.type", Value: string(worst.Type)},
		SpanAttribute{Key: "signals.worst.level", Value: string(worst.Level)},
	)

	return attrs
}
