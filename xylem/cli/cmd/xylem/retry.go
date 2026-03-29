package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
)

func newRetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry <vessel-id>",
		Short: "Retry a failed vessel with failure context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdRetry(deps.q, args[0])
		},
	}
	return cmd
}

func cmdRetry(q *queue.Queue, id string) error {
	vessel, err := q.FindByID(id)
	if err != nil {
		return fmt.Errorf("retry error: %w", err)
	}

	if vessel.State != queue.StateFailed {
		return fmt.Errorf("error: vessel %s is not in failed state (current: %s)", id, vessel.State)
	}

	newID := retryID(vessel.ID, q)

	meta := make(map[string]string)
	for k, v := range vessel.Meta {
		meta[k] = v
	}
	meta["retry_of"] = vessel.ID
	if vessel.Error != "" {
		meta["retry_error"] = vessel.Error
	}
	if vessel.FailedPhase != "" {
		meta["failed_phase"] = vessel.FailedPhase
	}
	if vessel.GateOutput != "" {
		meta["gate_output"] = vessel.GateOutput
	}

	newVessel := queue.Vessel{
		ID:          newID,
		Source:      vessel.Source,
		Ref:         vessel.Ref,
		Skill:       vessel.Skill,
		Prompt:      vessel.Prompt,
		Meta:        meta,
		State:       queue.StatePending,
		CreatedAt:   time.Now().UTC(),
		RetryOf:     vessel.ID,
		FailedPhase: vessel.FailedPhase,
		GateOutput:  vessel.GateOutput,
	}

	if err := q.Enqueue(newVessel); err != nil {
		return fmt.Errorf("enqueue retry: %w", err)
	}
	fmt.Printf("Created retry vessel %s (retrying %s)\n", newVessel.ID, vessel.ID)
	return nil
}

func retryID(originalID string, q *queue.Queue) string {
	vessels, _ := q.List()
	maxRetry := 0
	prefix := originalID + "-retry-"
	for _, v := range vessels {
		if strings.HasPrefix(v.ID, prefix) {
			numStr := strings.TrimPrefix(v.ID, prefix)
			if n, err := strconv.Atoi(numStr); err == nil && n > maxRetry {
				maxRetry = n
			}
		}
	}
	return fmt.Sprintf("%s-retry-%d", originalID, maxRetry+1)
}
