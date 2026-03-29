package reporter

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"
)

type mockRunner struct {
	lastArgs []string
	lastBody string // extracted from --body arg
	err      error
}

func (m *mockRunner) RunOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	m.lastArgs = append([]string{name}, args...)
	for i, arg := range args {
		if arg == "--body" && i+1 < len(args) {
			m.lastBody = args[i+1]
		}
	}
	return nil, m.err
}

func TestPhaseComplete(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	err := r.PhaseComplete(context.Background(), 42, "analyze", 2*time.Minute+15*time.Second, "some output here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(mock.lastBody, "phase `analyze` completed") {
		t.Errorf("expected phase name in comment, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "2m15s") {
		t.Errorf("expected duration in comment, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "some output here") {
		t.Errorf("expected output in comment, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "<details>") {
		t.Errorf("expected details block in comment, got: %s", mock.lastBody)
	}
}

func TestPhaseCompleteGhArgs(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	_ = r.PhaseComplete(context.Background(), 42, "analyze", time.Second, "out")

	wantArgs := []string{"gh", "issue", "comment", "42", "--repo", "owner/repo", "--body"}
	if len(mock.lastArgs) < len(wantArgs) {
		t.Fatalf("expected at least %d args, got %d: %v", len(wantArgs), len(mock.lastArgs), mock.lastArgs)
	}
	for i, want := range wantArgs {
		if mock.lastArgs[i] != want {
			t.Errorf("arg[%d]: expected %q, got %q", i, want, mock.lastArgs[i])
		}
	}
}

func TestPhaseCompleteTruncation(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	longOutput := strings.Repeat("x", MaxOutputLen+1000)
	err := r.PhaseComplete(context.Background(), 7, "build", time.Second, longOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(mock.lastBody, "(output truncated") {
		t.Errorf("expected truncation note in comment, got length %d", len(mock.lastBody))
	}
	if strings.Contains(mock.lastBody, longOutput) {
		t.Error("expected output to be truncated, but full output was present")
	}
}

func TestPhaseCompleteExactMaxLen(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	exactOutput := strings.Repeat("y", MaxOutputLen)
	err := r.PhaseComplete(context.Background(), 1, "test", time.Second, exactOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(mock.lastBody, "(output truncated") {
		t.Error("output at exactly MaxOutputLen should not be truncated")
	}
}

func TestVesselFailed(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	err := r.VesselFailed(context.Background(), 10, "implement", "segfault in handler", "gate check output here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(mock.lastBody, "failed at phase `implement`") {
		t.Errorf("expected phase name in comment, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "**Error:** segfault in handler") {
		t.Errorf("expected error message in comment, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "gate check output here") {
		t.Errorf("expected gate output in comment, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "<details>") {
		t.Errorf("expected details block for gate output, got: %s", mock.lastBody)
	}
}

func TestVesselFailedNoGateOutput(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	err := r.VesselFailed(context.Background(), 10, "implement", "segfault", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(mock.lastBody, "<details>") {
		t.Errorf("expected no details block when gate output is empty, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "failed at phase `implement`") {
		t.Errorf("expected phase name in comment, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "**Error:** segfault") {
		t.Errorf("expected error message in comment, got: %s", mock.lastBody)
	}
}

func TestVesselCompleted(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	phases := []PhaseResult{
		{Name: "analyze", Duration: 2*time.Minute + 15*time.Second, Status: "completed"},
		{Name: "implement", Duration: 5*time.Minute + 30*time.Second, Status: "completed"},
		{Name: "pr", Duration: time.Minute, Status: "completed"},
	}

	err := r.VesselCompleted(context.Background(), 5, phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(mock.lastBody, "all phases completed") {
		t.Errorf("expected header in comment, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "| analyze | 2m15s | completed |") {
		t.Errorf("expected analyze row in table, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "| implement | 5m30s | completed |") {
		t.Errorf("expected implement row in table, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "| pr | 1m0s | completed |") {
		t.Errorf("expected pr row in table, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "Total: 8m45s") {
		t.Errorf("expected total duration in comment, got: %s", mock.lastBody)
	}
	if !strings.Contains(mock.lastBody, "| Phase | Duration | Status |") {
		t.Errorf("expected table header in comment, got: %s", mock.lastBody)
	}
}

func TestLabelTimeout(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	err := r.LabelTimeout(context.Background(), 99, "approved", "implement", 30*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "xylem — timed out waiting for label `approved` on phase `implement` after 30m0s"
	if mock.lastBody != want {
		t.Errorf("expected %q, got %q", want, mock.lastBody)
	}
}

func TestGhFailureNonFatal(t *testing.T) {
	ghErr := errors.New("gh: command not found")
	tests := []struct {
		name string
		call func(r *Reporter) error
	}{
		{
			name: "PhaseComplete",
			call: func(r *Reporter) error {
				return r.PhaseComplete(context.Background(), 1, "analyze", time.Second, "out")
			},
		},
		{
			name: "VesselFailed",
			call: func(r *Reporter) error {
				return r.VesselFailed(context.Background(), 1, "analyze", "err", "gate")
			},
		},
		{
			name: "VesselCompleted",
			call: func(r *Reporter) error {
				return r.VesselCompleted(context.Background(), 1, []PhaseResult{{Name: "a", Duration: time.Second, Status: "completed"}})
			},
		},
		{
			name: "LabelTimeout",
			call: func(r *Reporter) error {
				return r.LabelTimeout(context.Background(), 1, "label", "phase", time.Minute)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockRunner{err: ghErr}
			r := &Reporter{Runner: mock, Repo: "owner/repo"}

			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(nil)

			err := tc.call(r)
			if err != nil {
				t.Errorf("expected nil error (non-fatal), got: %v", err)
			}
			if !strings.Contains(buf.String(), "warn:") {
				t.Errorf("expected warning log, got: %q", buf.String())
			}
		})
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLen    int
		wantTrunc bool
	}{
		{
			name:      "short string",
			input:     "hello",
			maxLen:    10,
			wantTrunc: false,
		},
		{
			name:      "exact length",
			input:     "hello",
			maxLen:    5,
			wantTrunc: false,
		},
		{
			name:      "over limit",
			input:     "hello world",
			maxLen:    5,
			wantTrunc: true,
		},
		{
			name:      "empty string",
			input:     "",
			maxLen:    10,
			wantTrunc: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateOutput(tc.input, tc.maxLen)
			if tc.wantTrunc {
				if !strings.Contains(result, "(output truncated") {
					t.Errorf("expected truncation note, got: %q", result)
				}
				if !strings.HasPrefix(result, tc.input[:tc.maxLen]) {
					t.Errorf("expected result to start with first %d chars", tc.maxLen)
				}
			} else {
				if result != tc.input {
					t.Errorf("expected unchanged output %q, got %q", tc.input, result)
				}
			}
		})
	}
}

func TestPostCommentArgs(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		issueNum int
		wantArgs []string
	}{
		{
			name:     "standard issue",
			repo:     "owner/repo",
			issueNum: 42,
			wantArgs: []string{"gh", "issue", "comment", "42", "--repo", "owner/repo", "--body"},
		},
		{
			name:     "different repo and issue",
			repo:     "org/project",
			issueNum: 123,
			wantArgs: []string{"gh", "issue", "comment", "123", "--repo", "org/project", "--body"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockRunner{}
			r := &Reporter{Runner: mock, Repo: tc.repo}

			_ = r.PhaseComplete(context.Background(), tc.issueNum, "test", time.Second, "out")

			if len(mock.lastArgs) < len(tc.wantArgs) {
				t.Fatalf("expected at least %d args, got %d: %v", len(tc.wantArgs), len(mock.lastArgs), mock.lastArgs)
			}
			for i, want := range tc.wantArgs {
				if mock.lastArgs[i] != want {
					t.Errorf("arg[%d]: expected %q, got %q", i, want, mock.lastArgs[i])
				}
			}
			// The last arg should be the --body value
			if mock.lastArgs[len(mock.lastArgs)-1] == "--body" {
				t.Error("expected a body value after --body flag")
			}
		})
	}
}

func TestVesselCompletedTotalDuration(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	phases := []PhaseResult{
		{Name: "a", Duration: 30 * time.Second, Status: "completed"},
		{Name: "b", Duration: 90 * time.Second, Status: "completed"},
	}

	_ = r.VesselCompleted(context.Background(), 1, phases)

	if !strings.Contains(mock.lastBody, "Total: 2m0s") {
		t.Errorf("expected total 2m0s, got: %s", mock.lastBody)
	}
}

func TestVesselFailedGhArgs(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "org/project"}

	_ = r.VesselFailed(context.Background(), 55, "build", "compile error", "")

	wantArgs := []string{"gh", "issue", "comment", "55", "--repo", "org/project", "--body"}
	if len(mock.lastArgs) < len(wantArgs) {
		t.Fatalf("expected at least %d args, got %d: %v", len(wantArgs), len(mock.lastArgs), mock.lastArgs)
	}
	for i, want := range wantArgs {
		if mock.lastArgs[i] != want {
			t.Errorf("arg[%d]: expected %q, got %q", i, want, mock.lastArgs[i])
		}
	}
}

func TestIssueNumFormatting(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	_ = r.PhaseComplete(context.Background(), 1234, "test", time.Second, "out")

	// Verify the issue number is formatted as a string in the args
	found := false
	for _, arg := range mock.lastArgs {
		if arg == "1234" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected issue number '1234' in args, got: %v", mock.lastArgs)
	}
}

func TestPhaseCompleteCommentFormat(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	_ = r.PhaseComplete(context.Background(), 1, "deploy", 5*time.Second, "deployed successfully")

	// Verify the full structure matches the spec format
	expectedParts := []string{
		"**xylem — phase `deploy` completed** (5s)",
		"<details>",
		"<summary>Phase output (click to expand)</summary>",
		"deployed successfully",
		"</details>",
	}
	for _, part := range expectedParts {
		if !strings.Contains(mock.lastBody, part) {
			t.Errorf("expected comment to contain %q, got:\n%s", part, mock.lastBody)
		}
	}
}

func TestVesselFailedCommentFormat(t *testing.T) {
	mock := &mockRunner{}
	r := &Reporter{Runner: mock, Repo: "owner/repo"}

	_ = r.VesselFailed(context.Background(), 1, "test", "assertion failed", "FAIL: TestFoo")

	expectedParts := []string{
		"**xylem — failed at phase `test`**",
		"**Error:** assertion failed",
		"<details>",
		"<summary>Gate output (click to expand)</summary>",
		"FAIL: TestFoo",
		"</details>",
	}
	for _, part := range expectedParts {
		if !strings.Contains(mock.lastBody, part) {
			t.Errorf("expected comment to contain %q, got:\n%s", part, mock.lastBody)
		}
	}
}
