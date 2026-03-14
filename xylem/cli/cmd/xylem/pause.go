package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
)

func newPauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause",
		Short: "Pause scan and drain operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdPause(deps.cfg)
		},
	}
}

func cmdPause(cfg *config.Config) error {
	marker := pauseMarkerPath(cfg)
	if isPaused(cfg) {
		fmt.Println("Already paused.")
		return nil
	}
	if err := os.MkdirAll(cfg.StateDir, 0o755); err != nil {
		return fmt.Errorf("error creating state dir: %w", err)
	}
	if err := os.WriteFile(marker, []byte{}, 0o644); err != nil {
		return fmt.Errorf("error creating pause marker: %w", err)
	}
	fmt.Println("Scanning paused. Run `xylem resume` to resume.")
	return nil
}
