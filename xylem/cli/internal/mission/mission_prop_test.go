package mission

import (
	"encoding/json"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// genConstraint generates a valid Constraint.
func genConstraint(t *rapid.T) Constraint {
	return Constraint{
		MaxRetries:  rapid.IntRange(0, 100).Draw(t, "max_retries"),
		TokenBudget: rapid.IntRange(0, 1_000_000).Draw(t, "token_budget"),
		TimeBudget:  time.Duration(rapid.IntRange(0, 3600).Draw(t, "time_budget")) * time.Second,
		BlastRadius: rapid.SliceOfN(rapid.StringMatching(`[a-z*]+\.[a-z]+`), 0, 5).Draw(t, "blast_radius"),
	}
}

// genMission generates a valid Mission.
func genMission(t *rapid.T) Mission {
	return Mission{
		ID:          rapid.StringMatching(`m-[a-z0-9]{3,10}`).Draw(t, "id"),
		Description: rapid.StringMatching(`[A-Za-z ]{5,50}`).Draw(t, "description"),
		Source:      rapid.SampledFrom([]string{"github", "linear", "manual"}).Draw(t, "source"),
		SourceRef:   rapid.StringMatching(`[a-z/]+#[0-9]+`).Draw(t, "source_ref"),
		Constraints: genConstraint(t),
		CreatedAt:   time.Now(),
	}
}

// --- Property: valid missions pass validation ---

func TestPropValidMissionPassesValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		m := genMission(t)
		if err := ValidateMission(m); err != nil {
			t.Fatalf("valid mission failed validation: %v", err)
		}
	})
}

// --- Property: negative constraint values are rejected ---

func TestPropNegativeConstraintRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := Constraint{
			MaxRetries:  rapid.IntRange(-1000, -1).Draw(t, "max_retries"),
			TokenBudget: rapid.IntRange(0, 100).Draw(t, "token_budget"),
			TimeBudget:  time.Duration(rapid.IntRange(0, 100).Draw(t, "time_budget")) * time.Second,
		}
		if err := ValidateConstraint(c); err == nil {
			t.Fatal("expected error for negative max_retries")
		}
	})
}

// --- Property: contract JSON round-trip ---

func TestPropContractJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

		missionID := rapid.StringMatching(`m-[a-z0-9]{3,8}`).Draw(t, "mission_id")
		taskCount := rapid.IntRange(1, 5).Draw(t, "task_count")

		tasks := make([]Task, taskCount)
		for i := range tasks {
			tasks[i] = Task{
				ID:          rapid.StringMatching(`t-[a-z0-9]{3,8}`).Draw(t, "task_id"),
				MissionID:   missionID,
				Description: rapid.StringMatching(`[a-z ]{5,20}`).Draw(t, "desc"),
				Status:      Pending,
			}
		}

		criteria := []Criterion{{
			Name:      rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "crit_name"),
			Threshold: float64(rapid.IntRange(0, 100).Draw(t, "threshold")) / 100.0,
			Required:  rapid.Bool().Draw(t, "required"),
		}}

		original := SprintContract{
			MissionID: missionID,
			Tasks:     tasks,
			Criteria:  criteria,
			CreatedAt: now,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded SprintContract
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.MissionID != original.MissionID {
			t.Errorf("MissionID mismatch: %q != %q", decoded.MissionID, original.MissionID)
		}
		if len(decoded.Tasks) != len(original.Tasks) {
			t.Errorf("Tasks len mismatch: %d != %d", len(decoded.Tasks), len(original.Tasks))
		}
		if len(decoded.Criteria) != len(original.Criteria) {
			t.Errorf("Criteria len mismatch: %d != %d", len(decoded.Criteria), len(original.Criteria))
		}
		if !decoded.CreatedAt.Equal(original.CreatedAt) {
			t.Errorf("CreatedAt mismatch: %v != %v", decoded.CreatedAt, original.CreatedAt)
		}
	})
}

// --- Property: AnalyzeComplexity is deterministic ---

func TestPropComplexityDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		desc := rapid.StringMatching(`[a-z ]{0,600}`).Draw(t, "desc")
		files := rapid.IntRange(0, 20).Draw(t, "files")
		domains := rapid.IntRange(0, 5).Draw(t, "domains")

		a := AnalyzeComplexity(desc, files, domains)
		b := AnalyzeComplexity(desc, files, domains)
		if a != b {
			t.Errorf("non-deterministic: %q != %q for (%q, %d, %d)", a, b, desc, files, domains)
		}
	})
}

// --- Property: CheckBlastRadius with ["*"] allows all single-segment paths ---

func TestPropBlastRadiusWildcardAllows(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// filepath.Match("*", p) matches any single-segment path (no slashes).
		path := rapid.StringMatching(`[a-z]{1,20}\.[a-z]{1,5}`).Draw(t, "path")
		err := CheckBlastRadius([]string{path}, []string{"*"})
		if err != nil {
			t.Fatalf("wildcard should allow %q: %v", path, err)
		}
	})
}

// --- Property: CheckBlastRadius with [] denies all paths ---

func TestPropBlastRadiusEmptyDenies(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		path := rapid.StringMatching(`[a-z]{1,20}\.[a-z]{1,5}`).Draw(t, "path")
		err := CheckBlastRadius([]string{path}, []string{})
		if err == nil {
			t.Fatalf("empty allowed should deny %q", path)
		}
	})
}
