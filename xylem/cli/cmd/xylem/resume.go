package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
)

func newResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume paused operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdResume(deps.cfg)
		},
	}
}

func cmdResume(cfg *config.Config) error {
	if !isPaused(cfg) {
		fmt.Println("Not paused.")
		return nil
	}
	if err := os.Remove(pauseMarkerPath(cfg)); err != nil {
		return fmt.Errorf("error removing pause marker: %w", err)
	}
	fmt.Println("Scanning resumed.")
	return nil
}
