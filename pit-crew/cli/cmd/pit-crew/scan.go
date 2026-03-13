package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/scanner"
)

func cmdScan(cfg *config.Config, q *queue.Queue, args []string) {
	dryRun := false
	for _, a := range args {
		if a == "--dry-run" {
			dryRun = true
		}
	}

	runner := &realCmdRunner{}

	if dryRun {
		dryRunScan(cfg, q, runner)
		return
	}

	s := scanner.New(cfg, q, runner)
	result, err := s.Scan(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		os.Exit(1)
	}
	if result.Paused {
		fmt.Println("Scanning is paused. Run `pit-crew resume` to resume.")
		return
	}
	fmt.Printf("Added %d jobs, skipped %d\n", result.Added, result.Skipped)
}

func dryRunScan(cfg *config.Config, q *queue.Queue, runner scanner.CommandRunner) {
	tmpFile, err := os.CreateTemp("", "pit-crew-dryrun-*.jsonl")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating temp file: %v\n", err)
		os.Exit(1)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	dryQ := queue.New(tmpFile.Name())
	s := scanner.New(cfg, dryQ, runner)
	result, err := s.Scan(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		os.Exit(1)
	}
	if result.Paused {
		fmt.Println("Scanning is paused.")
		return
	}
	jobs, _ := dryQ.List()
	if len(jobs) == 0 {
		fmt.Println("No new issues found.")
		return
	}
	fmt.Printf("%-12s  %-6s  %-20s  %s\n", "ID", "Issue", "Skill", "URL")
	fmt.Printf("%-12s  %-6s  %-20s  %s\n", "----", "-----", "-----", "---")
	for _, j := range jobs {
		fmt.Printf("%-12s  #%-5d  %-20s  %s\n", j.ID, j.IssueNum, j.Skill, j.IssueURL)
	}
	fmt.Printf("\n%d candidate(s) would be queued (dry-run — no changes made)\n", len(jobs))
}
