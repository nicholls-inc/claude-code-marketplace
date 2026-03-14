package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/worktree"
)

func newCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove stale worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return cmdCleanup(deps.wt, dryRun)
		},
	}
	cmd.Flags().Bool("dry-run", false, "Preview what would be removed")
	return cmd
}

func cmdCleanup(wt *worktree.Manager, dryRun bool) error {
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
