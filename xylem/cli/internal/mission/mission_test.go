package mission

import (
	"strings"
	"testing"
	"time"
)

// validMission returns a Mission with all required fields populated.
func validMission(t *testing.T) Mission {
	t.Helper()
	return Mission{
		ID:          "m-001",
		Description: "Implement login flow",
		Source:      "github",
		SourceRef:   "owner/repo#42",
		Constraints: Constraint{
			MaxRetries:  3,
			TokenBudget: 50000,
			TimeBudget:  10 * time.Minute,
			BlastRadius: []string{"src/*.go"},
		},
		CreatedAt: time.Now(),
	}
}

// --- ValidateMission tests ---

func TestValidateMission(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Mission)
		wantErr string
	}{
		{
			name:   "valid mission",
			modify: func(_ *Mission) {},
		},
		{
			name:    "missing ID",
			modify:  func(m *Mission) { m.ID = "" },
			wantErr: "ID is required",
		},
		{
			name:    "whitespace-only ID",
			modify:  func(m *Mission) { m.ID = "   " },
			wantErr: "ID is required",
		},
		{
			name:    "missing description",
			modify:  func(m *Mission) { m.Description = "" },
			wantErr: "description is required",
		},
		{
			name: "negative retries",
			modify: func(m *Mission) {
				m.Constraints.MaxRetries = -1
			},
			wantErr: "max_retries must be non-negative",
		},
		{
			name: "negative token budget",
			modify: func(m *Mission) {
				m.Constraints.TokenBudget = -100
			},
			wantErr: "token_budget must be non-negative",
		},
		{
			name: "negative time budget",
			modify: func(m *Mission) {
				m.Constraints.TimeBudget = -1 * time.Second
			},
			wantErr: "time_budget must be non-negative",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := validMission(t)
			tc.modify(&m)
			err := ValidateMission(m)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

// --- ValidateConstraint tests ---

func TestValidateConstraint(t *testing.T) {
	tests := []struct {
		name       string
		constraint Constraint
		wantErr    string
	}{
		{
			name: "valid constraint",
			constraint: Constraint{
				MaxRetries:  5,
				TokenBudget: 10000,
				TimeBudget:  5 * time.Minute,
				BlastRadius: []string{"*.go", "docs/**"},
			},
		},
		{
			name: "zero values are valid",
			constraint: Constraint{
				MaxRetries:  0,
				TokenBudget: 0,
				TimeBudget:  0,
			},
		},
		{
			name: "negative retries",
			constraint: Constraint{
				MaxRetries: -1,
			},
			wantErr: "max_retries must be non-negative",
		},
		{
			name: "negative token budget",
			constraint: Constraint{
				TokenBudget: -50,
			},
			wantErr: "token_budget must be non-negative",
		},
		{
			name: "invalid glob pattern",
			constraint: Constraint{
				BlastRadius: []string{"[invalid"},
			},
			wantErr: "invalid blast_radius glob",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConstraint(tc.constraint)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

// --- AnalyzeComplexity tests ---

func TestAnalyzeComplexity(t *testing.T) {
	tests := []struct {
		name        string
		description string
		fileCount   int
		domainCount int
		want        ComplexityLevel
	}{
		{
			name:        "simple - short description, few files, single domain",
			description: "fix typo",
			fileCount:   1,
			domainCount: 0,
			want:        Simple,
		},
		{
			name:        "moderate - several files",
			description: "update handler",
			fileCount:   3,
			domainCount: 0,
			want:        Moderate,
		},
		{
			name:        "moderate - multi domain",
			description: "add endpoint",
			fileCount:   1,
			domainCount: 1,
			want:        Moderate,
		},
		{
			name:        "moderate - longer description",
			description: strings.Repeat("a", 100),
			fileCount:   0,
			domainCount: 0,
			want:        Moderate,
		},
		{
			name:        "complex - many files",
			description: "refactor module",
			fileCount:   10,
			domainCount: 0,
			want:        Complex,
		},
		{
			name:        "complex - many domains",
			description: "cross-cutting concern",
			fileCount:   0,
			domainCount: 3,
			want:        Complex,
		},
		{
			name:        "complex - very long description",
			description: strings.Repeat("x", 500),
			fileCount:   0,
			domainCount: 0,
			want:        Complex,
		},
		{
			name:        "boundary - just below moderate file threshold",
			description: "small fix",
			fileCount:   2,
			domainCount: 0,
			want:        Simple,
		},
		{
			name:        "boundary - just below complex file threshold",
			description: "medium change",
			fileCount:   9,
			domainCount: 0,
			want:        Moderate,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AnalyzeComplexity(tc.description, tc.fileCount, tc.domainCount)
			if got != tc.want {
				t.Errorf("AnalyzeComplexity(%q, %d, %d) = %q, want %q",
					tc.description, tc.fileCount, tc.domainCount, got, tc.want)
			}
		})
	}
}

// --- CheckBlastRadius tests ---

func TestCheckBlastRadius(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		allowed []string
		wantErr string
	}{
		{
			name:    "within radius",
			paths:   []string{"main.go", "util.go"},
			allowed: []string{"*.go"},
		},
		{
			name:    "outside radius",
			paths:   []string{"main.go", "config.yaml"},
			allowed: []string{"*.go"},
			wantErr: `path "config.yaml" does not match`,
		},
		{
			name:    "wildcard allows all",
			paths:   []string{"anything.txt", "deeply.nested"},
			allowed: []string{"*"},
		},
		{
			name:    "empty radius denies all",
			paths:   []string{"any.go"},
			allowed: []string{},
			wantErr: `path "any.go" does not match`,
		},
		{
			name:    "empty paths always passes",
			paths:   []string{},
			allowed: []string{},
		},
		{
			name:    "multiple patterns match different paths",
			paths:   []string{"main.go", "README.md"},
			allowed: []string{"*.go", "*.md"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckBlastRadius(tc.paths, tc.allowed)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

// --- Contract creation tests ---

func TestNewContract(t *testing.T) {
	m := validMission(t)
	tasks := []Task{{ID: "t-1", MissionID: m.ID, Description: "do thing", Status: Pending}}
	criteria := []Criterion{{Name: "tests pass", Threshold: 1.0, Required: true}}
	steps := []VerificationStep{{Type: "test", Command: "go test ./...", Description: "run tests"}}

	tests := []struct {
		name     string
		mission  Mission
		tasks    []Task
		criteria []Criterion
		steps    []VerificationStep
		wantErr  string
	}{
		{
			name:     "valid contract",
			mission:  m,
			tasks:    tasks,
			criteria: criteria,
			steps:    steps,
		},
		{
			name:     "invalid mission",
			mission:  Mission{},
			tasks:    tasks,
			criteria: criteria,
			steps:    steps,
			wantErr:  "ID is required",
		},
		{
			name:     "empty tasks",
			mission:  m,
			tasks:    []Task{},
			criteria: criteria,
			steps:    steps,
			wantErr:  "at least one task is required",
		},
		{
			name:     "empty criteria",
			mission:  m,
			tasks:    tasks,
			criteria: []Criterion{},
			steps:    steps,
			wantErr:  "at least one criterion is required",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewContract(tc.mission, tc.tasks, tc.criteria, tc.steps)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if c == nil {
					t.Fatal("expected contract, got nil")
				}
				if c.MissionID != tc.mission.ID {
					t.Errorf("MissionID = %q, want %q", c.MissionID, tc.mission.ID)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

// --- ValidateContract tests ---

func TestValidateContract(t *testing.T) {
	tests := []struct {
		name    string
		c       SprintContract
		wantErr string
	}{
		{
			name: "valid",
			c: SprintContract{
				MissionID: "m-001",
				Tasks:     []Task{{ID: "t-1"}},
				Criteria:  []Criterion{{Name: "c-1"}},
			},
		},
		{
			name: "missing mission ID",
			c: SprintContract{
				Tasks:    []Task{{ID: "t-1"}},
				Criteria: []Criterion{{Name: "c-1"}},
			},
			wantErr: "mission_id is required",
		},
		{
			name: "no tasks",
			c: SprintContract{
				MissionID: "m-001",
				Criteria:  []Criterion{{Name: "c-1"}},
			},
			wantErr: "at least one task is required",
		},
		{
			name: "no criteria",
			c: SprintContract{
				MissionID: "m-001",
				Tasks:     []Task{{ID: "t-1"}},
			},
			wantErr: "at least one criterion is required",
		},
		{
			name: "duplicate task IDs",
			c: SprintContract{
				MissionID: "m-001",
				Tasks:     []Task{{ID: "t-1"}, {ID: "t-1"}},
				Criteria:  []Criterion{{Name: "c-1"}},
			},
			wantErr: "duplicate task ID",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateContract(tc.c)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

// --- Contract Accept tests ---

func TestContractAccept(t *testing.T) {
	t.Run("sets timestamp", func(t *testing.T) {
		c := &SprintContract{MissionID: "m-001"}
		if err := c.Accept(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.AcceptedAt == nil {
			t.Fatal("expected AcceptedAt to be set")
		}
	})

	t.Run("double accept returns error", func(t *testing.T) {
		c := &SprintContract{MissionID: "m-001"}
		if err := c.Accept(); err != nil {
			t.Fatalf("unexpected error on first accept: %v", err)
		}
		err := c.Accept()
		if err == nil {
			t.Fatal("expected error on double accept, got nil")
		}
		if !strings.Contains(err.Error(), "already accepted") {
			t.Errorf("expected 'already accepted' in error, got %q", err.Error())
		}
	})
}

// --- Contract save/load round-trip tests ---

func TestContractSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	accepted := now.Add(1 * time.Hour)

	original := SprintContract{
		MissionID: "m-roundtrip",
		Tasks: []Task{
			{ID: "t-1", MissionID: "m-roundtrip", Description: "task one", Status: Pending},
			{ID: "t-2", MissionID: "m-roundtrip", Description: "task two", Dependencies: []string{"t-1"}, Status: InProgress},
		},
		Criteria: []Criterion{
			{Name: "tests", Description: "all tests pass", Threshold: 1.0, Required: true},
		},
		VerificationSteps: []VerificationStep{
			{Type: "test", Command: "go test ./...", Description: "run unit tests"},
		},
		CreatedAt:  now,
		AcceptedAt: &accepted,
	}

	if err := SaveContract(original, dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadContract("m-roundtrip", dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Verify key fields round-trip.
	if loaded.MissionID != original.MissionID {
		t.Errorf("MissionID: got %q, want %q", loaded.MissionID, original.MissionID)
	}
	if len(loaded.Tasks) != len(original.Tasks) {
		t.Errorf("Tasks len: got %d, want %d", len(loaded.Tasks), len(original.Tasks))
	}
	if len(loaded.Criteria) != len(original.Criteria) {
		t.Errorf("Criteria len: got %d, want %d", len(loaded.Criteria), len(original.Criteria))
	}
	if loaded.AcceptedAt == nil {
		t.Fatal("AcceptedAt should not be nil after round-trip")
	}
	if !loaded.AcceptedAt.Equal(*original.AcceptedAt) {
		t.Errorf("AcceptedAt: got %v, want %v", loaded.AcceptedAt, original.AcceptedAt)
	}
}

func TestLoadContractNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadContract("nonexistent", dir)
	if err == nil {
		t.Fatal("expected error loading nonexistent contract")
	}
}
