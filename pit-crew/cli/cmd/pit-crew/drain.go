package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/runner"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/worktree"
)

func newDrainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drain",
		Short: "Dequeue pending jobs and launch Claude sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return cmdDrain(deps.cfg, deps.q, deps.wt, dryRun)
		},
	}
	cmd.Flags().Bool("dry-run", false, "Preview what would be drained")
	return cmd
}

func cmdDrain(cfg *config.Config, q *queue.Queue, wt *worktree.Manager, dryRun bool) error {
	if dryRun {
		return dryRunDrain(cfg, q)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cmdRunner := &realCmdRunner{}
	r := runner.New(cfg, q, wt, cmdRunner)
	result, err := r.Drain(ctx)
	if err != nil {
		return &exitError{code: 2, err: fmt.Errorf("drain error: %w", err)}
	}
	fmt.Printf("Completed %d, failed %d, skipped %d\n", result.Completed, result.Failed, result.Skipped)
	if result.Failed > 0 {
		return &exitError{code: 1}
	}
	return nil
}

func dryRunDrain(cfg *config.Config, q *queue.Queue) error {
	jobs, err := q.ListByState(queue.StatePending)
	if err != nil {
		return &exitError{code: 2, err: fmt.Errorf("error reading queue: %w", err)}
	}
	if len(jobs) == 0 {
		fmt.Println("No pending jobs.")
		return nil
	}
	fmt.Printf("%-12s  %-6s  %-20s  %s\n", "ID", "Issue", "Skill", "Command")
	fmt.Printf("%-12s  %-6s  %-20s  %s\n", "----", "-----", "-----", "-------")
	for _, j := range jobs {
		cmd := fmt.Sprintf("%s -p \"/%s %s\" --max-turns %d", cfg.Claude.Command, j.Skill, j.IssueURL, cfg.MaxTurns)
		fmt.Printf("%-12s  #%-5d  %-20s  %s\n", j.ID, j.IssueNum, j.Skill, cmd)
	}
	fmt.Printf("\n%d job(s) would be drained (dry-run — no sessions launched)\n", len(jobs))
	return nil
}
