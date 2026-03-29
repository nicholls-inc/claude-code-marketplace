package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/scanner"
)

func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Query GitHub for actionable issues and enqueue vessels",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return cmdScan(deps.cfg, deps.q, &realCmdRunner{}, dryRun)
		},
	}
	cmd.Flags().Bool("dry-run", false, "Preview what would be queued")
	return cmd
}

func cmdScan(cfg *config.Config, q *queue.Queue, runner scanner.CommandRunner, dryRun bool) error {

	if dryRun {
		return dryRunScan(cfg, q, runner)
	}

	s := scanner.New(cfg, q, runner)
	result, err := s.Scan(context.Background())
	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}
	if result.Paused {
		fmt.Println("Scanning is paused. Run `xylem resume` to resume.")
		return nil
	}
	fmt.Printf("Added %d vessels, skipped %d\n", result.Added, result.Skipped)
	return nil
}

func dryRunScan(cfg *config.Config, q *queue.Queue, runner scanner.CommandRunner) error {
	tmpFile, err := os.CreateTemp("", "xylem-dryrun-*.jsonl")
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	dryQ := queue.New(tmpFile.Name())
	s := scanner.New(cfg, dryQ, runner)
	result, err := s.Scan(context.Background())
	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}
	if result.Paused {
		fmt.Println("Scanning is paused.")
		return nil
	}
	vessels, _ := dryQ.List()
	if len(vessels) == 0 {
		fmt.Println("No new issues found.")
		return nil
	}
	fmt.Printf("%-14s  %-14s  %-20s  %s\n", "ID", "Source", "Workflow", "Ref")
	fmt.Printf("%-14s  %-14s  %-20s  %s\n", "----", "------", "--------", "---")
	for _, j := range vessels {
		fmt.Printf("%-14s  %-14s  %-20s  %s\n", j.ID, j.Source, j.Workflow, j.Ref)
	}
	fmt.Printf("\n%d candidate(s) would be queued (dry-run — no changes made)\n", len(vessels))
	return nil
}
