package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show queue state and vessel summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			stateFilter, _ := cmd.Flags().GetString("state")
			return cmdStatus(deps.q, jsonMode, stateFilter)
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().String("state", "", "Filter by vessel state")
	return cmd
}

func cmdStatus(q *queue.Queue, jsonMode bool, stateFilter string) error {
	var vessels []queue.Vessel
	var err error
	if stateFilter != "" {
		vessels, err = q.ListByState(queue.VesselState(stateFilter))
	} else {
		vessels, err = q.List()
	}
	if err != nil {
		return fmt.Errorf("error reading queue: %w", err)
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if vessels == nil {
			vessels = []queue.Vessel{}
		}
		enc.Encode(vessels) //nolint:errcheck
		return nil
	}

	if len(vessels) == 0 {
		fmt.Println("No vessels in queue.")
		return nil
	}

	fmt.Printf("%-14s  %-14s  %-20s  %-10s  %-30s  %-12s  %s\n",
		"ID", "Source", "Workflow", "State", "Info", "Started", "Duration")
	fmt.Printf("%-14s  %-14s  %-20s  %-10s  %-30s  %-12s  %s\n",
		"----", "------", "--------", "-----", "----", "-------", "--------")

	counts := map[queue.VesselState]int{}
	for _, j := range vessels {
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
		wf := j.Workflow
		if wf == "" {
			wf = "(prompt)"
		}
		info := vesselInfo(j)
		fmt.Printf("%-14s  %-14s  %-20s  %-10s  %-30s  %-12s  %s\n",
			j.ID, j.Source, wf, string(j.State), info, started, duration)
	}

	fmt.Printf("\nSummary: %d pending, %d running, %d completed, %d failed, %d cancelled, %d waiting, %d timed_out\n",
		counts[queue.StatePending], counts[queue.StateRunning],
		counts[queue.StateCompleted], counts[queue.StateFailed],
		counts[queue.StateCancelled], counts[queue.StateWaiting],
		counts[queue.StateTimedOut])
	return nil
}

// vesselInfo returns additional context for the Info column based on vessel state.
func vesselInfo(v queue.Vessel) string {
	if v.State == queue.StateWaiting && v.WaitingFor != "" {
		elapsed := "unknown"
		if v.WaitingSince != nil {
			elapsed = time.Since(*v.WaitingSince).Round(time.Second).String()
		}
		return fmt.Sprintf("waiting for %q (%s)", v.WaitingFor, elapsed)
	}
	return ""
}

func pauseMarkerPath(cfg *config.Config) string {
	return filepath.Join(cfg.StateDir, "paused")
}

func isPaused(cfg *config.Config) bool {
	_, err := os.Stat(pauseMarkerPath(cfg))
	return err == nil
}
