package source

import (
	"context"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
)

// Source discovers tasks and produces Vessels.
type Source interface {
	// Name returns the source identifier (e.g., "github-issue").
	Name() string
	// Scan discovers new tasks and returns vessels to enqueue.
	Scan(ctx context.Context) ([]queue.Vessel, error)
	// OnStart is called when a vessel from this source begins running.
	// Used for side effects like adding an "in-progress" label.
	OnStart(ctx context.Context, vessel queue.Vessel) error
	// BranchName generates the git branch name for this vessel.
	BranchName(vessel queue.Vessel) string
}

// CommandRunner abstracts subprocess execution for testing.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}
