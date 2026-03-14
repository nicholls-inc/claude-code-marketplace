package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

func newCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <job-id>",
		Short: "Cancel a queued or running job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdCancel(deps.q, args[0])
		},
	}
}

func cmdCancel(q *queue.Queue, id string) error {
	if err := q.Cancel(id); err != nil {
		return fmt.Errorf("cancel error: %w", err)
	}
	fmt.Printf("Cancelled job %s\n", id)
	return nil
}
