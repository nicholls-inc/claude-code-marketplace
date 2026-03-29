package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Criterion defines an evaluation criterion for a sprint contract.
type Criterion struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Threshold   float64 `json:"threshold"`
	Required    bool    `json:"required"`
}

// VerificationStep describes how to verify a deliverable.
type VerificationStep struct {
	Type        string `json:"type"` // test, lint, formal, manual
	Command     string `json:"command"`
	Description string `json:"description"`
}

// SprintContract is the agreement that defines "done" before work begins.
//
// INV: Sprint contract always has at least one criterion.
type SprintContract struct {
	MissionID         string             `json:"mission_id"`
	Tasks             []Task             `json:"tasks"`
	Criteria          []Criterion        `json:"criteria"`
	VerificationSteps []VerificationStep `json:"verification_steps"`
	CreatedAt         time.Time          `json:"created_at"`
	AcceptedAt        *time.Time         `json:"accepted_at,omitempty"`
}

// contractsDir is the subdirectory under the base dir where contracts are stored.
const contractsDir = "contracts"

// validateMissionID checks that a mission ID is safe for use as a filename
// component — it must not contain path separators or ".." traversal.
func validateMissionID(id string) error {
	if strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("mission ID %q contains path separator", id)
	}
	// After rejecting separators, the only remaining traversal risk is bare "..".
	if id == ".." {
		return fmt.Errorf("mission ID %q contains path traversal component", id)
	}
	return nil
}

// checkDuplicateTaskIDs returns the first duplicate task ID found, or "".
func checkDuplicateTaskIDs(tasks []Task) string {
	seen := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		if seen[t.ID] {
			return t.ID
		}
		seen[t.ID] = true
	}
	return ""
}

// NewContract creates a SprintContract after validating that the inputs meet
// the minimum requirements, including duplicate task ID detection.
func NewContract(mission Mission, tasks []Task, criteria []Criterion, steps []VerificationStep) (*SprintContract, error) {
	if err := ValidateMission(mission); err != nil {
		return nil, fmt.Errorf("new contract: %w", err)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("new contract: at least one task is required")
	}
	if len(criteria) == 0 {
		return nil, fmt.Errorf("new contract: at least one criterion is required")
	}
	if dup := checkDuplicateTaskIDs(tasks); dup != "" {
		return nil, fmt.Errorf("new contract: duplicate task ID %q", dup)
	}

	return &SprintContract{
		MissionID:         mission.ID,
		Tasks:             tasks,
		Criteria:          criteria,
		VerificationSteps: steps,
		CreatedAt:         time.Now(),
	}, nil
}

// ValidateContract checks that a sprint contract has all required fields.
func ValidateContract(c SprintContract) error {
	if strings.TrimSpace(c.MissionID) == "" {
		return fmt.Errorf("validate contract: mission_id is required")
	}
	if len(c.Tasks) == 0 {
		return fmt.Errorf("validate contract: at least one task is required")
	}
	if len(c.Criteria) == 0 {
		return fmt.Errorf("validate contract: at least one criterion is required")
	}
	// INV: Task IDs within a decomposition are unique.
	if dup := checkDuplicateTaskIDs(c.Tasks); dup != "" {
		return fmt.Errorf("validate contract: duplicate task ID %q", dup)
	}
	return nil
}

// Accept marks the contract as accepted by setting AcceptedAt to the current
// time. Returns an error if the contract has already been accepted.
func (c *SprintContract) Accept() error {
	if c.AcceptedAt != nil {
		return fmt.Errorf("accept contract: already accepted at %s", c.AcceptedAt.Format(time.RFC3339))
	}
	now := time.Now()
	c.AcceptedAt = &now
	return nil
}

// SaveContract writes the contract as JSON to <dir>/contracts/<mission-id>.json.
//
// INV: Contract JSON round-trips perfectly.
func SaveContract(c SprintContract, dir string) error {
	if err := validateMissionID(c.MissionID); err != nil {
		return fmt.Errorf("save contract: %w", err)
	}
	target := filepath.Join(dir, contractsDir)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("save contract: create dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("save contract: marshal: %w", err)
	}
	path := filepath.Join(target, c.MissionID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save contract: write: %w", err)
	}
	return nil
}

// LoadContract reads a contract from <dir>/contracts/<mission-id>.json.
func LoadContract(missionID string, dir string) (*SprintContract, error) {
	if err := validateMissionID(missionID); err != nil {
		return nil, fmt.Errorf("load contract: %w", err)
	}
	path := filepath.Join(dir, contractsDir, missionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load contract: read: %w", err)
	}
	var c SprintContract
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("load contract: unmarshal: %w", err)
	}
	return &c, nil
}
