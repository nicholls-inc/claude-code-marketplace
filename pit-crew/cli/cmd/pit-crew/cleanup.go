package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/worktree"
)

func cmdCleanup(wt *worktree.Manager, args []string) {
	dryRun := false
	for _, a := range args {
		if a == "--dry-run" {
			dryRun = true
		}
	}

	ctx := context.Background()
	trees, err := wt.ListPitCrew(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing worktrees: %v\n", err)
		os.Exit(1)
	}
	if len(trees) == 0 {
		fmt.Println("No pit-crew worktrees found.")
		return
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
}
