package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Runner abstracts command execution for testing.
type Runner interface {
	RunOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

// exitCoder is satisfied by exec.ExitError and test doubles.
type exitCoder interface {
	ExitCode() int
}

// RunCommandGate executes a shell command in the given directory and reports
// whether the command passed (exit 0). A non-zero exit is not an error — it
// means the gate did not pass. Only system-level failures (e.g. sh not found)
// are returned as errors.
func RunCommandGate(ctx context.Context, r Runner, dir string, command string) (string, bool, error) {
	output, err := r.RunOutput(ctx, "sh", "-c", fmt.Sprintf("cd %s && %s", shellQuote(dir), command))
	if err != nil {
		if ec, ok := err.(exitCoder); ok && ec.ExitCode() != 0 {
			return string(output), false, nil
		}
		return "", false, fmt.Errorf("run gate command: %w", err)
	}
	return string(output), true, nil
}

// ghLabelsResponse is the JSON shape returned by `gh issue view --json labels`.
type ghLabelsResponse struct {
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// CheckLabel queries GitHub for the given issue and returns whether the
// specified label is present.
func CheckLabel(ctx context.Context, r Runner, repo string, issueNum int, label string) (bool, error) {
	out, err := r.RunOutput(ctx, "gh", "issue", "view", fmt.Sprintf("%d", issueNum), "--repo", repo, "--json", "labels")
	if err != nil {
		return false, fmt.Errorf("check label: %w", err)
	}

	var resp ghLabelsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return false, fmt.Errorf("parse label response: %w", err)
	}

	for _, l := range resp.Labels {
		if l.Name == label {
			return true, nil
		}
	}
	return false, nil
}

// shellQuote wraps s in single quotes, escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
