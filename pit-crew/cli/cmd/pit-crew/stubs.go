package main

import (
	"fmt"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/worktree"
)

func cmdScan(cfg *config.Config, q *queue.Queue, args []string) {
	fmt.Println("scan: not implemented")
}

func cmdDrain(cfg *config.Config, q *queue.Queue, wt *worktree.Manager, args []string) {
	fmt.Println("drain: not implemented")
}

func cmdStatus(q *queue.Queue, args []string) {
	fmt.Println("status: not implemented")
}

func cmdPause(cfg *config.Config, args []string) {
	fmt.Println("pause: not implemented")
}

func cmdResume(cfg *config.Config, args []string) {
	fmt.Println("resume: not implemented")
}

func cmdCancel(q *queue.Queue, args []string) {
	fmt.Println("cancel: not implemented")
}

func cmdCleanup(wt *worktree.Manager, args []string) {
	fmt.Println("cleanup: not implemented")
}
