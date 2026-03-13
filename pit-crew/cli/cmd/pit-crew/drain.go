package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/runner"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/worktree"
)

func cmdDrain(cfg *config.Config, q *queue.Queue, wt *worktree.Manager, args []string) {
	dryRun := false
	for _, a := range args {
		if a == "--dry-run" {
			dryRun = true
		}
	}

	if dryRun {
		dryRunDrain(cfg, q)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cmdRunner := &realCmdRunner{}
	r := runner.New(cfg, q, wt, cmdRunner)
	result, err := r.Drain(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drain error: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("Completed %d, failed %d, skipped %d\n", result.Completed, result.Failed, result.Skipped)
	if result.Failed > 0 {
		os.Exit(1)
	}
}

func dryRunDrain(cfg *config.Config, q *queue.Queue) {
	jobs, err := q.ListByState(queue.StatePending)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading queue: %v\n", err)
		os.Exit(2)
	}
	if len(jobs) == 0 {
		fmt.Println("No pending jobs.")
		return
	}
	fmt.Printf("%-12s  %-6s  %-20s  %s\n", "ID", "Issue", "Skill", "Command")
	fmt.Printf("%-12s  %-6s  %-20s  %s\n", "----", "-----", "-----", "-------")
	for _, j := range jobs {
		cmd := fmt.Sprintf("%s -p \"/%s %s\" --max-turns %d", cfg.Claude.Command, j.Skill, j.IssueURL, cfg.MaxTurns)
		fmt.Printf("%-12s  #%-5d  %-20s  %s\n", j.ID, j.IssueNum, j.Skill, cmd)
	}
	fmt.Printf("\n%d job(s) would be drained (dry-run — no sessions launched)\n", len(jobs))
}
