package main

import (
	"fmt"
	"os"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
)

func cmdPause(cfg *config.Config, args []string) {
	marker := pauseMarkerPath(cfg)
	if isPaused(cfg) {
		fmt.Println("Already paused.")
		return
	}
	if err := os.MkdirAll(cfg.StateDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating state dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(marker, []byte{}, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error creating pause marker: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Scanning paused. Run `pit-crew resume` to resume.")
}
