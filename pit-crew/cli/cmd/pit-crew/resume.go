package main

import (
	"fmt"
	"os"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
)

func cmdResume(cfg *config.Config, args []string) {
	if !isPaused(cfg) {
		fmt.Println("Not paused.")
		return
	}
	if err := os.Remove(pauseMarkerPath(cfg)); err != nil {
		fmt.Fprintf(os.Stderr, "error removing pause marker: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Scanning resumed.")
}
