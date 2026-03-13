package main

import (
	"fmt"
	"os"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

func cmdCancel(q *queue.Queue, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pit-crew cancel <job-id>")
		os.Exit(1)
	}
	id := args[0]
	if err := q.Cancel(id); err != nil {
		fmt.Fprintf(os.Stderr, "cancel error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Cancelled job %s\n", id)
}
