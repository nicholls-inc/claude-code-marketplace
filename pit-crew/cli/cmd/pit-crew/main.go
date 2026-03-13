package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/worktree"
)

func main() {
	configPath := ".pit-crew.yml"
	args := os.Args[1:]

	for len(args) >= 2 && args[0] == "--config" {
		configPath = args[1]
		args = args[2:]
	}

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		usage()
		os.Exit(0)
	}

	subcommand := args[0]
	rest := args[1:]

	for _, tool := range []string{"gh", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s not found on PATH\n", tool)
			os.Exit(1)
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config %s: %v\n", configPath, err)
		os.Exit(1)
	}

	queueFile := filepath.Join(cfg.StateDir, "queue.jsonl")
	q := queue.New(queueFile)
	wt := worktree.New(".", &realCmdRunner{})

	switch subcommand {
	case "scan":
		cmdScan(cfg, q, rest)
	case "drain":
		cmdDrain(cfg, q, wt, rest)
	case "status":
		cmdStatus(q, rest)
	case "pause":
		cmdPause(cfg, rest)
	case "resume":
		cmdResume(cfg, rest)
	case "cancel":
		cmdCancel(q, rest)
	case "cleanup":
		cmdCleanup(wt, rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", subcommand)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`pit-crew — autonomous issue agent scheduler

Usage:
  pit-crew [--config <path>] <subcommand> [flags]

Subcommands:
  scan      Query GitHub for actionable issues and enqueue jobs
  drain     Dequeue pending jobs and launch Claude sessions
  status    Show queue state and job summary
  pause     Pause scan and drain operations
  resume    Resume paused operations
  cancel    Cancel a queued or running job
  cleanup   Remove stale worktrees

Flags:
  --config <path>   Config file path (default: .pit-crew.yml)
  --help            Show this help

`)
}
