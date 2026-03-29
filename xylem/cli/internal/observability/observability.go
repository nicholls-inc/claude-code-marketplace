// Package observability provides span attribute types for OpenTelemetry
// instrumentation of xylem components. It defines attribute-extraction
// functions that convert domain types into key-value pairs suitable for
// attaching to OTel spans.
//
// This is a foundation package: it defines the attribute schema and
// conversion logic. Actual OTel SDK integration (creating spans, exporting
// traces) is a follow-up.
package observability

import (
	"fmt"
	"strings"
)

// SpanAttribute represents a single key-value pair for span annotation.
// Keys follow OTel semantic conventions using dot-separated namespaces.
type SpanAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SignalData holds signal information for attribute extraction.
type SignalData struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
	Level string  `json:"level"`
}

// AgentData holds agent information for attribute extraction.
type AgentData struct {
	ID         string `json:"id"`
	Task       string `json:"task"`
	Status     string `json:"status"`
	TokensUsed int    `json:"tokens_used"`
}

// MissionData holds mission information for attribute extraction.
type MissionData struct {
	ID         string `json:"id"`
	Complexity string `json:"complexity"`
	Source     string `json:"source"`
	TaskCount  int    `json:"task_count"`
}

// FormatAttributeKey constructs a namespaced attribute key.
// INV: Output is always lowercase and dot-separated.
func FormatAttributeKey(namespace, name string) string {
	return strings.ToLower(namespace) + "." + strings.ToLower(name)
}

// SignalSpanAttributes converts signal data into span attributes. Each signal
// produces two attributes: one for the value and one for the level.
// INV: Output contains exactly 2*len(signals) attributes.
func SignalSpanAttributes(signals []SignalData) []SpanAttribute {
	attrs := make([]SpanAttribute, 0, 2*len(signals))
	for _, s := range signals {
		prefix := FormatAttributeKey("signals", s.Type)
		attrs = append(attrs, SpanAttribute{
			Key:   prefix + ".value",
			Value: fmt.Sprintf("%.4f", s.Value),
		})
		attrs = append(attrs, SpanAttribute{
			Key:   prefix + ".level",
			Value: s.Level,
		})
	}
	return attrs
}

// AgentSpanAttributes converts agent data into span attributes.
// INV: Output always includes "agent.id" and "agent.status".
func AgentSpanAttributes(agent AgentData) []SpanAttribute {
	return []SpanAttribute{
		{Key: "agent.id", Value: agent.ID},
		{Key: "agent.task", Value: agent.Task},
		{Key: "agent.status", Value: agent.Status},
		{Key: "agent.tokens_used", Value: fmt.Sprintf("%d", agent.TokensUsed)},
	}
}

// MissionSpanAttributes converts mission data into span attributes.
// INV: Output always includes "mission.id".
func MissionSpanAttributes(mission MissionData) []SpanAttribute {
	return []SpanAttribute{
		{Key: "mission.id", Value: mission.ID},
		{Key: "mission.complexity", Value: mission.Complexity},
		{Key: "mission.source", Value: mission.Source},
		{Key: "mission.task_count", Value: fmt.Sprintf("%d", mission.TaskCount)},
	}
}
