// Package mission provides types and logic for mission complexity analysis,
// task decomposition, and constraint validation.
package mission

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ComplexityLevel classifies the complexity of a mission.
type ComplexityLevel string

const (
	// Simple indicates a single-agent mission.
	Simple ComplexityLevel = "Simple"
	// Moderate indicates a mission that may need coordination.
	Moderate ComplexityLevel = "Moderate"
	// Complex indicates a multi-agent mission requiring decomposition.
	Complex ComplexityLevel = "Complex"
)

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	// Pending means the task has not started.
	Pending TaskStatus = "Pending"
	// InProgress means the task is currently being worked on.
	InProgress TaskStatus = "InProgress"
	// Completed means the task finished successfully.
	Completed TaskStatus = "Completed"
	// Failed means the task did not complete successfully.
	Failed TaskStatus = "Failed"
)

// Constraint defines resource and scope limits for a mission.
type Constraint struct {
	MaxRetries  int           `json:"max_retries"`
	TokenBudget int           `json:"token_budget"`
	TimeBudget  time.Duration `json:"time_budget"`
	BlastRadius []string      `json:"blast_radius"` // allowed file glob patterns
}

// Persona defines scope enforcement for an agent working a mission.
type Persona struct {
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Scope        string   `json:"scope"`
	Restrictions []string `json:"restrictions"`
}

// Mission is the top-level unit of work (a GitHub issue, Linear ticket, CLI submission).
type Mission struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Source      string     `json:"source"`
	SourceRef   string     `json:"source_ref"`
	Constraints Constraint `json:"constraints"`
	Persona     *Persona   `json:"persona,omitempty"`
	Context     []string   `json:"context"`  // linked doc paths
	CreatedAt   time.Time  `json:"created_at"`
}

// Task is a decomposed unit of work within a mission.
type Task struct {
	ID           string     `json:"id"`
	MissionID    string     `json:"mission_id"`
	Description  string     `json:"description"`
	Deliverables []string   `json:"deliverables"`
	Dependencies []string   `json:"dependencies"` // task IDs
	Status       TaskStatus `json:"status"`
}

// Decomposition is the result of breaking a mission into tasks.
type Decomposition struct {
	Tasks     []Task `json:"tasks"`
	Rationale string `json:"rationale"`
}

// Complexity thresholds.
const (
	complexFileThreshold   = 10
	complexDomainThreshold = 3
	complexDescLen         = 500

	moderateFileThreshold   = 3
	moderateDomainThreshold = 1
	moderateDescLen         = 100
)

// AnalyzeComplexity determines the complexity level of a mission based on its
// description length, the number of files it touches, and the number of
// distinct domains involved.
//
// INV: Complexity classification is deterministic for same inputs.
func AnalyzeComplexity(description string, fileCount int, domainCount int) ComplexityLevel {
	if fileCount >= complexFileThreshold || domainCount >= complexDomainThreshold || len(description) >= complexDescLen {
		return Complex
	}
	if fileCount >= moderateFileThreshold || domainCount >= moderateDomainThreshold || len(description) >= moderateDescLen {
		return Moderate
	}
	return Simple
}

// ValidateMission checks that a mission has all required fields and that its
// constraints are non-negative.
func ValidateMission(m Mission) error {
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("validate mission: ID is required")
	}
	if strings.TrimSpace(m.Description) == "" {
		return fmt.Errorf("validate mission: description is required")
	}
	if err := ValidateConstraint(m.Constraints); err != nil {
		return fmt.Errorf("validate mission: %w", err)
	}
	return nil
}

// ValidateConstraint checks that all constraint values are non-negative and
// that blast radius patterns are valid globs.
//
// INV: All constraint values are non-negative after validation.
func ValidateConstraint(c Constraint) error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("validate constraint: max_retries must be non-negative, got %d", c.MaxRetries)
	}
	if c.TokenBudget < 0 {
		return fmt.Errorf("validate constraint: token_budget must be non-negative, got %d", c.TokenBudget)
	}
	if c.TimeBudget < 0 {
		return fmt.Errorf("validate constraint: time_budget must be non-negative, got %s", c.TimeBudget)
	}
	for _, pattern := range c.BlastRadius {
		if _, err := filepath.Match(pattern, ""); err != nil {
			return fmt.Errorf("validate constraint: invalid blast_radius glob %q: %w", pattern, err)
		}
	}
	return nil
}

// CheckBlastRadius verifies that every path in paths matches at least one of
// the allowed glob patterns. Returns an error listing the first path that does
// not match any pattern.
//
// INV: Blast radius check rejects paths not matching any allowed pattern.
func CheckBlastRadius(paths []string, allowed []string) error {
	for _, p := range paths {
		if !matchesAny(p, allowed) {
			return fmt.Errorf("check blast radius: path %q does not match any allowed pattern", p)
		}
	}
	return nil
}

// matchesAny reports whether p matches at least one of the given glob patterns.
func matchesAny(p string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, err := filepath.Match(pattern, p); err == nil && matched {
			return true
		}
	}
	return false
}
