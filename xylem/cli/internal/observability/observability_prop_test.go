package observability

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// genSignalData generates a random SignalData.
func genSignalData() *rapid.Generator[SignalData] {
	return rapid.Custom(func(t *rapid.T) SignalData {
		return SignalData{
			Type:  rapid.SampledFrom([]string{"Repetition", "ToolFailureRate", "ContextThrash", "TaskStall"}).Draw(t, "type"),
			Value: rapid.Float64Range(0.0, 1.0).Draw(t, "value"),
			Level: rapid.SampledFrom([]string{"Normal", "Warning", "Critical"}).Draw(t, "level"),
		}
	})
}

// genSignalSlice generates a slice of SignalData.
func genSignalSlice() *rapid.Generator[[]SignalData] {
	return rapid.Custom(func(t *rapid.T) []SignalData {
		n := rapid.IntRange(0, 20).Draw(t, "count")
		signals := make([]SignalData, n)
		for i := range n {
			signals[i] = genSignalData().Draw(t, "signal")
		}
		return signals
	})
}

// genAgentData generates a random AgentData.
func genAgentData() *rapid.Generator[AgentData] {
	return rapid.Custom(func(t *rapid.T) AgentData {
		return AgentData{
			ID:         rapid.StringMatching(`[a-z0-9-]{1,20}`).Draw(t, "id"),
			Task:       rapid.SampledFrom([]string{"fix-bug", "implement-feature", "refactor"}).Draw(t, "task"),
			Status:     rapid.SampledFrom([]string{"running", "completed", "failed"}).Draw(t, "status"),
			TokensUsed: rapid.IntRange(0, 100000).Draw(t, "tokens"),
		}
	})
}

// genMissionData generates a random MissionData.
func genMissionData() *rapid.Generator[MissionData] {
	return rapid.Custom(func(t *rapid.T) MissionData {
		return MissionData{
			ID:         rapid.StringMatching(`[a-z0-9-]{1,20}`).Draw(t, "id"),
			Complexity: rapid.SampledFrom([]string{"low", "medium", "high"}).Draw(t, "complexity"),
			Source:     rapid.SampledFrom([]string{"github", "manual", "cron"}).Draw(t, "source"),
			TaskCount:  rapid.IntRange(0, 50).Draw(t, "task_count"),
		}
	})
}

func TestPropSignalAttributeCount(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		signals := genSignalSlice().Draw(t, "signals")
		attrs := SignalSpanAttributes(signals)
		if len(attrs) != 2*len(signals) {
			t.Fatalf("expected %d attributes, got %d", 2*len(signals), len(attrs))
		}
	})
}

func TestPropAttributeKeysAlwaysLowercase(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		signals := genSignalSlice().Draw(t, "signals")
		attrs := SignalSpanAttributes(signals)
		for _, a := range attrs {
			if a.Key != strings.ToLower(a.Key) {
				t.Fatalf("key %q is not lowercase", a.Key)
			}
		}

		agent := genAgentData().Draw(t, "agent")
		for _, a := range AgentSpanAttributes(agent) {
			if a.Key != strings.ToLower(a.Key) {
				t.Fatalf("key %q is not lowercase", a.Key)
			}
		}

		mission := genMissionData().Draw(t, "mission")
		for _, a := range MissionSpanAttributes(mission) {
			if a.Key != strings.ToLower(a.Key) {
				t.Fatalf("key %q is not lowercase", a.Key)
			}
		}
	})
}

func TestPropAgentAttributesAlwaysContainID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		agent := genAgentData().Draw(t, "agent")
		attrs := AgentSpanAttributes(agent)
		found := false
		for _, a := range attrs {
			if a.Key == "agent.id" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("agent.id attribute not found")
		}
	})
}

func TestPropMissionAttributesAlwaysContainID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mission := genMissionData().Draw(t, "mission")
		attrs := MissionSpanAttributes(mission)
		found := false
		for _, a := range attrs {
			if a.Key == "mission.id" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("mission.id attribute not found")
		}
	})
}
