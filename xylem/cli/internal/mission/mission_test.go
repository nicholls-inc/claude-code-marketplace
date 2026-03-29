package mission

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
		{
			name:    "duplicate task IDs",
			mission: m,
			tasks: []Task{
				{ID: "t-1", MissionID: m.ID, Description: "first", Status: Pending},
				{ID: "t-1", MissionID: m.ID, Description: "second", Status: Pending},
			},
			criteria: criteria,
			steps:    steps,
			wantErr:  "duplicate task ID",
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
		t.Fatalf("Tasks len: got %d, want %d", len(loaded.Tasks), len(original.Tasks))
	}
	// Verify task fields including Dependencies.
	for i, want := range original.Tasks {
		got := loaded.Tasks[i]
		if got.ID != want.ID {
			t.Errorf("Tasks[%d].ID: got %q, want %q", i, got.ID, want.ID)
		}
		if got.Description != want.Description {
			t.Errorf("Tasks[%d].Description: got %q, want %q", i, got.Description, want.Description)
		}
		if got.Status != want.Status {
			t.Errorf("Tasks[%d].Status: got %q, want %q", i, got.Status, want.Status)
		}
		if len(got.Dependencies) != len(want.Dependencies) {
			t.Errorf("Tasks[%d].Dependencies len: got %d, want %d", i, len(got.Dependencies), len(want.Dependencies))
		} else {
			for j, dep := range want.Dependencies {
				if got.Dependencies[j] != dep {
					t.Errorf("Tasks[%d].Dependencies[%d]: got %q, want %q", i, j, got.Dependencies[j], dep)
				}
			}
		}
	}
	if len(loaded.Criteria) != len(original.Criteria) {
		t.Errorf("Criteria len: got %d, want %d", len(loaded.Criteria), len(original.Criteria))
	}
	// Verify CreatedAt round-trips.
	if loaded.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero after round-trip")
	}
	if !loaded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", loaded.CreatedAt, original.CreatedAt)
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

// --- Path sanitization tests ---

func TestSaveContractRejectsPathTraversal(t *testing.T) {
	tests := []struct {
		name      string
		missionID string
		wantErr   string
	}{
		{
			name:      "forward slash",
			missionID: "../etc/passwd",
			wantErr:   "path separator",
		},
		{
			name:      "backslash",
			missionID: `..\..\secret`,
			wantErr:   "path separator",
		},
		{
			name:      "dotdot only",
			missionID: "..",
			wantErr:   "path traversal",
		},
		{
			name:      "embedded slash",
			missionID: "foo/bar",
			wantErr:   "path separator",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			c := SprintContract{MissionID: tc.missionID}
			err := SaveContract(c, dir)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestLoadContractRejectsPathTraversal(t *testing.T) {
	tests := []struct {
		name      string
		missionID string
		wantErr   string
	}{
		{
			name:      "forward slash",
			missionID: "../etc/passwd",
			wantErr:   "path separator",
		},
		{
			name:      "dotdot only",
			missionID: "..",
			wantErr:   "path traversal",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			_, err := LoadContract(tc.missionID, dir)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

// --- mockPoster for ContractPoster tests ---

type mockPoster struct {
	called   bool
	contract SprintContract
	err      error
}

func (m *mockPoster) PostContract(_ context.Context, c SprintContract) error {
	m.called = true
	m.contract = c
	return m.err
}

// --- FormatContractMarkdown tests ---

func sampleContract() SprintContract {
	return SprintContract{
		MissionID: "m-fmt-001",
		Tasks: []Task{
			{ID: "t-1", MissionID: "m-fmt-001", Description: "implement login", Status: Pending},
			{ID: "t-2", MissionID: "m-fmt-001", Description: "add tests", Status: Pending},
		},
		Criteria: []Criterion{
			{Name: "coverage", Description: "code coverage above threshold", Threshold: 0.8, Required: true},
		},
		VerificationSteps: []VerificationStep{
			{Type: "test", Command: "go test ./...", Description: "run unit tests"},
			{Type: "manual", Description: "review code"},
		},
		CreatedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	}
}

func TestFormatContractMarkdownContainsMissionID(t *testing.T) {
	c := sampleContract()
	md := FormatContractMarkdown(c)
	if !strings.Contains(md, c.MissionID) {
		t.Errorf("markdown should contain mission ID %q, got:\n%s", c.MissionID, md)
	}
}

func TestFormatContractMarkdownContainsTasks(t *testing.T) {
	c := sampleContract()
	md := FormatContractMarkdown(c)
	for _, task := range c.Tasks {
		if !strings.Contains(md, task.Description) {
			t.Errorf("markdown should contain task description %q, got:\n%s", task.Description, md)
		}
	}
}

func TestFormatContractMarkdownContainsCriteria(t *testing.T) {
	c := sampleContract()
	md := FormatContractMarkdown(c)
	for _, cr := range c.Criteria {
		if !strings.Contains(md, cr.Name) {
			t.Errorf("markdown should contain criterion name %q, got:\n%s", cr.Name, md)
		}
		if !strings.Contains(md, cr.Description) {
			t.Errorf("markdown should contain criterion description %q, got:\n%s", cr.Description, md)
		}
	}
}

func TestFormatContractMarkdownContainsVerificationSteps(t *testing.T) {
	c := sampleContract()
	md := FormatContractMarkdown(c)
	for _, vs := range c.VerificationSteps {
		if !strings.Contains(md, vs.Description) {
			t.Errorf("markdown should contain step description %q, got:\n%s", vs.Description, md)
		}
		if !strings.Contains(md, vs.Type) {
			t.Errorf("markdown should contain step type %q, got:\n%s", vs.Type, md)
		}
		if vs.Command != "" && !strings.Contains(md, vs.Command) {
			t.Errorf("markdown should contain step command %q, got:\n%s", vs.Command, md)
		}
	}
}

// --- SaveAndPost tests ---

func TestSaveAndPostNilPoster(t *testing.T) {
	dir := t.TempDir()
	c := SprintContract{
		MissionID: "m-nilpost",
		Tasks:     []Task{{ID: "t-1", MissionID: "m-nilpost", Description: "do thing", Status: Pending}},
		Criteria:  []Criterion{{Name: "pass", Threshold: 1.0, Required: true}},
		CreatedAt: time.Now(),
	}

	err := SaveAndPost(context.Background(), c, dir, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify file was saved.
	path := filepath.Join(dir, "contracts", "m-nilpost.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected contract file to exist at %s: %v", path, err)
	}
}

func TestSaveAndPostSuccess(t *testing.T) {
	dir := t.TempDir()
	c := SprintContract{
		MissionID: "m-post-ok",
		Tasks:     []Task{{ID: "t-1", MissionID: "m-post-ok", Description: "do thing", Status: Pending}},
		Criteria:  []Criterion{{Name: "pass", Threshold: 1.0, Required: true}},
		CreatedAt: time.Now(),
	}
	poster := &mockPoster{}

	err := SaveAndPost(context.Background(), c, dir, poster)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !poster.called {
		t.Fatal("expected poster to be called")
	}
	if poster.contract.MissionID != c.MissionID {
		t.Errorf("poster received MissionID %q, want %q", poster.contract.MissionID, c.MissionID)
	}

	// Verify file was saved.
	path := filepath.Join(dir, "contracts", "m-post-ok.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected contract file to exist: %v", err)
	}
}

func TestSaveAndPostSaveError(t *testing.T) {
	// Use a non-writable directory to force save failure.
	dir := "/nonexistent/path/that/does/not/exist"
	c := SprintContract{
		MissionID: "m-save-err",
		Tasks:     []Task{{ID: "t-1", MissionID: "m-save-err", Description: "do thing", Status: Pending}},
		Criteria:  []Criterion{{Name: "pass", Threshold: 1.0, Required: true}},
		CreatedAt: time.Now(),
	}
	poster := &mockPoster{}

	err := SaveAndPost(context.Background(), c, dir, poster)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "save and post") {
		t.Errorf("expected error containing 'save and post', got %q", err.Error())
	}
	if poster.called {
		t.Fatal("poster should NOT be called when save fails")
	}
}

func TestSaveAndPostPostError(t *testing.T) {
	dir := t.TempDir()
	c := SprintContract{
		MissionID: "m-post-err",
		Tasks:     []Task{{ID: "t-1", MissionID: "m-post-err", Description: "do thing", Status: Pending}},
		Criteria:  []Criterion{{Name: "pass", Threshold: 1.0, Required: true}},
		CreatedAt: time.Now(),
	}
	poster := &mockPoster{err: errors.New("API unavailable")}

	err := SaveAndPost(context.Background(), c, dir, poster)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "post") {
		t.Errorf("expected error containing 'post', got %q", err.Error())
	}

	// INV: Local file is always written if SaveContract succeeds.
	path := filepath.Join(dir, "contracts", "m-post-err.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("contract file should exist even when post fails: %v", err)
	}
}
