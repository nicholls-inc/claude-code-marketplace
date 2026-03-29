package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
)

func newEnqueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enqueue",
		Short: "Manually enqueue a task for Claude to work on",
		RunE: func(cmd *cobra.Command, args []string) error {
			wf, _ := cmd.Flags().GetString("workflow")
			ref, _ := cmd.Flags().GetString("ref")
			prompt, _ := cmd.Flags().GetString("prompt")
			promptFile, _ := cmd.Flags().GetString("prompt-file")
			srcName, _ := cmd.Flags().GetString("source")
			id, _ := cmd.Flags().GetString("id")
			return cmdEnqueue(deps.q, wf, ref, prompt, promptFile, srcName, id)
		},
	}
	cmd.Flags().String("workflow", "", "Workflow to invoke (e.g., fix-bug, implement-feature)")
	cmd.Flags().String("ref", "", "Task reference (URL, ticket ID, description)")
	cmd.Flags().String("prompt", "", "Direct prompt to pass to Claude")
	cmd.Flags().String("prompt-file", "", "Read prompt from file (mutually exclusive with --prompt)")
	cmd.Flags().String("source", "manual", "Source identifier")
	cmd.Flags().String("id", "", "Custom vessel ID (auto-generated if empty)")
	return cmd
}

func cmdEnqueue(q *queue.Queue, wf, ref, prompt, promptFile, srcName, id string) error {
	if prompt != "" && promptFile != "" {
		return fmt.Errorf("--prompt and --prompt-file are mutually exclusive")
	}

	if promptFile != "" {
		data, err := os.ReadFile(promptFile)
		if err != nil {
			return fmt.Errorf("read prompt file: %w", err)
		}
		prompt = string(data)
	}

	if wf == "" && prompt == "" {
		return fmt.Errorf("at least one of --workflow or --prompt/--prompt-file is required")
	}

	if id == "" {
		id = fmt.Sprintf("task-%d", time.Now().UnixMilli())
	}

	vessel := queue.Vessel{
		ID:        id,
		Source:    srcName,
		Ref:       ref,
		Workflow:  wf,
		Prompt:    prompt,
		State:     queue.StatePending,
		CreatedAt: time.Now().UTC(),
	}
	if err := q.Enqueue(vessel); err != nil {
		return fmt.Errorf("enqueue error: %w", err)
	}
	fmt.Printf("Enqueued vessel %s (workflow=%s, source=%s)\n", vessel.ID, vessel.Workflow, vessel.Source)
	return nil
}
