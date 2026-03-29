package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/reporter"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/runner"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/source"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/worktree"
)

func newDrainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drain",
		Short: "Dequeue pending vessels and launch Claude sessions",
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
	r.Sources = buildSourceMap(cfg, q, cmdRunner)
	r.Reporter = buildReporter(cfg, cmdRunner)

	// Check waiting vessels before draining pending ones
	r.CheckWaitingVessels(ctx)

	result, err := r.Drain(ctx)
	if err != nil {
		return &exitError{code: 2, err: fmt.Errorf("drain error: %w", err)}
	}
	fmt.Printf("Completed %d, failed %d, skipped %d, waiting %d\n", result.Completed, result.Failed, result.Skipped, result.Waiting)
	if result.Failed > 0 {
		return &exitError{code: 1}
	}
	return nil
}

func buildSourceMap(cfg *config.Config, q *queue.Queue, cmdRunner source.CommandRunner) map[string]source.Source {
	sources := make(map[string]source.Source)
	for _, srcCfg := range cfg.Sources {
		if srcCfg.Type == "github" {
			tasks := make(map[string]source.GitHubTask, len(srcCfg.Tasks))
			for name, t := range srcCfg.Tasks {
				tasks[name] = source.GitHubTask{
					Labels: t.Labels,
					Skill:  t.Skill,
				}
			}
			gh := &source.GitHub{
				Repo:      srcCfg.Repo,
				Tasks:     tasks,
				Exclude:   srcCfg.Exclude,
				Queue:     q,
				CmdRunner: cmdRunner,
			}
			sources[gh.Name()] = gh
		}
	}
	return sources
}

func buildReporter(cfg *config.Config, cmdRunner reporter.Runner) *reporter.Reporter {
	// Find the first GitHub source repo for reporting
	for _, srcCfg := range cfg.Sources {
		if srcCfg.Type == "github" && srcCfg.Repo != "" {
			return &reporter.Reporter{Runner: cmdRunner, Repo: srcCfg.Repo}
		}
	}
	return nil
}

func dryRunDrain(cfg *config.Config, q *queue.Queue) error {
	vessels, err := q.ListByState(queue.StatePending)
	if err != nil {
		return &exitError{code: 2, err: fmt.Errorf("error reading queue: %w", err)}
	}
	if len(vessels) == 0 {
		fmt.Println("No pending vessels.")
		return nil
	}
	fmt.Printf("%-14s  %-14s  %-20s  %s\n", "ID", "Source", "Skill", "Command")
	fmt.Printf("%-14s  %-14s  %-20s  %s\n", "----", "------", "-----", "-------")
	for _, j := range vessels {
		skill := j.Skill
		if skill == "" {
			skill = "(prompt)"
		}
		var cmd string
		if j.Prompt != "" {
			cmd = fmt.Sprintf("%s -p %q --max-turns %d", cfg.Claude.Command, truncate(j.Prompt, 40), cfg.MaxTurns)
		} else {
			cmd = fmt.Sprintf("%s -p \"/%s %s\" --max-turns %d", cfg.Claude.Command, j.Skill, j.Ref, cfg.MaxTurns)
		}
		fmt.Printf("%-14s  %-14s  %-20s  %s\n", j.ID, j.Source, skill, cmd)
	}
	fmt.Printf("\n%d vessel(s) would be drained (dry-run — no sessions launched)\n", len(vessels))
	return nil
}
