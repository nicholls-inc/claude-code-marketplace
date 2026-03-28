package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/worktree"
)

func newCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove stale worktrees and old phase outputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return cmdCleanup(deps.cfg, deps.q, deps.wt, dryRun)
		},
	}
	cmd.Flags().Bool("dry-run", false, "Preview what would be removed")
	return cmd
}

func cmdCleanup(cfg *config.Config, q *queue.Queue, wt *worktree.Manager, dryRun bool) error {
	if err := cleanupWorktrees(wt, dryRun); err != nil {
		return err
	}

	cleanupPhaseOutputs(cfg, q, dryRun)

	return nil
}

func cleanupWorktrees(wt *worktree.Manager, dryRun bool) error {
	ctx := context.Background()
	trees, err := wt.ListXylem(ctx)
	if err != nil {
		return fmt.Errorf("error listing worktrees: %w", err)
	}
	if len(trees) == 0 {
		fmt.Println("No xylem worktrees found.")
		return nil
	}

	removed := 0
	for _, t := range trees {
		if dryRun {
			fmt.Printf("Would remove: %s\n", t.Path)
			removed++
			continue
		}
		if err := wt.Remove(ctx, t.Path); err != nil {
			fmt.Fprintf(os.Stderr, "error removing %s: %v\n", t.Path, err)
			continue
		}
		fmt.Printf("Removed %s\n", t.Path)
		removed++
	}

	if dryRun {
		fmt.Printf("\n%d worktree(s) would be removed (dry-run — no changes made)\n", removed)
	} else {
		fmt.Printf("\nRemoved %d worktree(s)\n", removed)
	}
	return nil
}

func cleanupPhaseOutputs(cfg *config.Config, q *queue.Queue, dryRun bool) {
	phasesDir := filepath.Join(cfg.StateDir, "phases")
	if _, err := os.Stat(phasesDir); os.IsNotExist(err) {
		return // no phases directory yet
	}

	vessels, err := q.List()
	if err != nil {
		log.Printf("warn: could not read queue for phase cleanup: %v", err)
		return
	}

	cutoff := time.Now().Add(-168 * time.Hour) // 7 days

	// Build set of terminal vessel IDs older than cutoff
	terminalIDs := make(map[string]bool)
	for _, v := range vessels {
		isTerminal := v.State == queue.StateCompleted || v.State == queue.StateFailed ||
			v.State == queue.StateCancelled || v.State == queue.StateTimedOut
		if isTerminal && v.EndedAt != nil && v.EndedAt.Before(cutoff) {
			terminalIDs[v.ID] = true
		}
	}

	entries, err := os.ReadDir(phasesDir)
	if err != nil {
		log.Printf("warn: could not read phases directory: %v", err)
		return
	}

	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !terminalIDs[entry.Name()] {
			continue
		}
		dirPath := filepath.Join(phasesDir, entry.Name())
		if dryRun {
			fmt.Printf("Would remove phase outputs: %s\n", dirPath)
		} else {
			if err := os.RemoveAll(dirPath); err != nil {
				log.Printf("warn: failed to remove %s: %v", dirPath, err)
				continue
			}
			fmt.Printf("Removed phase outputs: %s\n", dirPath)
		}
		removed++
	}

	if removed > 0 {
		fmt.Printf("Cleaned up %d phase output directory(ies)\n", removed)
	}
}
