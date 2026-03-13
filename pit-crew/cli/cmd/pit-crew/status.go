package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

func cmdStatus(q *queue.Queue, args []string) {
	jsonMode := false
	stateFilter := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "--state":
			if i+1 < len(args) {
				stateFilter = args[i+1]
				i++
			}
		}
	}

	var jobs []queue.Job
	var err error
	if stateFilter != "" {
		jobs, err = q.ListByState(queue.JobState(stateFilter))
	} else {
		jobs, err = q.List()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading queue: %v\n", err)
		os.Exit(1)
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if jobs == nil {
			jobs = []queue.Job{}
		}
		enc.Encode(jobs) //nolint:errcheck
		return
	}

	if len(jobs) == 0 {
		fmt.Println("No jobs in queue.")
		return
	}

	fmt.Printf("%-14s  %-6s  %-20s  %-10s  %-12s  %s\n",
		"ID", "Issue", "Skill", "State", "Started", "Duration")
	fmt.Printf("%-14s  %-6s  %-20s  %-10s  %-12s  %s\n",
		"----", "-----", "-----", "-----", "-------", "--------")

	counts := map[queue.JobState]int{}
	for _, j := range jobs {
		counts[j.State]++
		started := "—"
		duration := "—"
		if j.StartedAt != nil {
			started = j.StartedAt.UTC().Format("15:04 UTC")
			end := time.Now()
			if j.EndedAt != nil {
				end = *j.EndedAt
			}
			duration = end.Sub(*j.StartedAt).Round(time.Second).String()
		}
		fmt.Printf("%-14s  #%-5d  %-20s  %-10s  %-12s  %s\n",
			j.ID, j.IssueNum, j.Skill, string(j.State), started, duration)
	}

	fmt.Printf("\nSummary: %d pending, %d running, %d completed, %d failed\n",
		counts[queue.StatePending], counts[queue.StateRunning],
		counts[queue.StateCompleted], counts[queue.StateFailed])
}

func pauseMarkerPath(cfg *config.Config) string {
	return filepath.Join(cfg.StateDir, "paused")
}

func isPaused(cfg *config.Config) bool {
	_, err := os.Stat(pauseMarkerPath(cfg))
	return err == nil
}
