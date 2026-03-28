package reporter

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// MaxOutputLen is the maximum length of phase output included in comments.
const MaxOutputLen = 64000

// Runner abstracts command execution for testing.
type Runner interface {
	RunOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Reporter posts phase progress comments to GitHub issues.
type Reporter struct {
	Runner Runner
	Repo   string // "owner/name"
}

// PhaseResult holds the outcome of a single phase for the summary comment.
type PhaseResult struct {
	Name     string
	Duration time.Duration
	Status   string // "completed" or "failed"
}

// PhaseComplete posts a comment on the GitHub issue when a phase completes successfully.
func (r *Reporter) PhaseComplete(ctx context.Context, issueNum int, phaseName string, duration time.Duration, output string) error {
	truncated := truncateOutput(output, MaxOutputLen)

	body := fmt.Sprintf(
		"**xylem — phase `%s` completed** (%s)\n\n<details>\n<summary>Phase output (click to expand)</summary>\n\n%s\n\n</details>",
		phaseName, duration, truncated,
	)

	if err := postComment(ctx, r.Runner, r.Repo, issueNum, body); err != nil {
		log.Printf("warn: failed to post phase-complete comment for issue %d: %v", issueNum, err)
	}
	return nil
}

// VesselFailed posts a failure comment on the GitHub issue.
func (r *Reporter) VesselFailed(ctx context.Context, issueNum int, phaseName string, errMsg string, gateOutput string) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**xylem — failed at phase `%s`**\n\n**Error:** %s", phaseName, errMsg)

	if gateOutput != "" {
		fmt.Fprintf(&sb, "\n\n<details>\n<summary>Gate output (click to expand)</summary>\n\n%s\n\n</details>", gateOutput)
	}

	if err := postComment(ctx, r.Runner, r.Repo, issueNum, sb.String()); err != nil {
		log.Printf("warn: failed to post vessel-failed comment for issue %d: %v", issueNum, err)
	}
	return nil
}

// VesselCompleted posts a summary comment when all phases complete.
func (r *Reporter) VesselCompleted(ctx context.Context, issueNum int, phases []PhaseResult) error {
	var sb strings.Builder
	sb.WriteString("**xylem — all phases completed**\n\n")
	sb.WriteString("| Phase | Duration | Status |\n")
	sb.WriteString("|-------|----------|--------|\n")

	var total time.Duration
	for _, p := range phases {
		fmt.Fprintf(&sb, "| %s | %s | %s |\n", p.Name, p.Duration, p.Status)
		total += p.Duration
	}

	fmt.Fprintf(&sb, "\nTotal: %s", total)

	if err := postComment(ctx, r.Runner, r.Repo, issueNum, sb.String()); err != nil {
		log.Printf("warn: failed to post vessel-completed comment for issue %d: %v", issueNum, err)
	}
	return nil
}

// LabelTimeout posts a timeout comment on the GitHub issue.
func (r *Reporter) LabelTimeout(ctx context.Context, issueNum int, label string, phaseName string, waited time.Duration) error {
	body := fmt.Sprintf("xylem — timed out waiting for label `%s` on phase `%s` after %s", label, phaseName, waited)

	if err := postComment(ctx, r.Runner, r.Repo, issueNum, body); err != nil {
		log.Printf("warn: failed to post label-timeout comment for issue %d: %v", issueNum, err)
	}
	return nil
}

// postComment calls gh issue comment with the given body.
func postComment(ctx context.Context, runner Runner, repo string, issueNum int, body string) error {
	_, err := runner.RunOutput(ctx, "gh", "issue", "comment", fmt.Sprintf("%d", issueNum), "--repo", repo, "--body", body)
	return err
}

// truncateOutput truncates s to maxLen characters, appending a note if truncated.
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n\n(output truncated — full output in .xylem/phases/<id>/<phase>.output)"
}
