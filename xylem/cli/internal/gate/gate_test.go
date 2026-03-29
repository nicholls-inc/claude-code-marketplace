package gate

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// mockRunner returns canned output and error values.
type mockRunner struct {
	output []byte
	err    error
}

func (m *mockRunner) RunOutput(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return m.output, m.err
}

// exitError simulates a non-zero exit code (satisfies exitCoder).
type exitError struct{ code int }

func (e *exitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }
func (e *exitError) ExitCode() int { return e.code }

// --- RunCommandGate tests ---

func TestRunCommandGatePass(t *testing.T) {
	r := &mockRunner{output: []byte("all good\n")}
	out, passed, err := RunCommandGate(context.Background(), r, "/tmp/work", "make test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected gate to pass")
	}
	if out != "all good\n" {
		t.Errorf("expected output %q, got %q", "all good\n", out)
	}
}

func TestRunCommandGateNonZeroExit(t *testing.T) {
	r := &mockRunner{
		output: []byte("FAIL: test_foo\n"),
		err:    &exitError{code: 1},
	}
	out, passed, err := RunCommandGate(context.Background(), r, "/tmp/work", "make test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected gate to not pass on non-zero exit")
	}
	if out != "FAIL: test_foo\n" {
		t.Errorf("expected output %q, got %q", "FAIL: test_foo\n", out)
	}
}

func TestRunCommandGateSystemError(t *testing.T) {
	r := &mockRunner{err: errors.New("exec: sh not found")}
	out, passed, err := RunCommandGate(context.Background(), r, "/tmp/work", "make test")
	if err == nil {
		t.Fatal("expected system error, got nil")
	}
	if passed {
		t.Error("expected gate to not pass on system error")
	}
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestRunCommandGateShellQuotesDir(t *testing.T) {
	// Verify that a directory with spaces is handled. We can't check the
	// exact command string from outside, but the function should not error.
	r := &mockRunner{output: []byte("ok")}
	_, passed, err := RunCommandGate(context.Background(), r, "/tmp/my dir", "ls")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected gate to pass")
	}
}

// --- CheckLabel tests ---

func TestCheckLabelPresent(t *testing.T) {
	r := &mockRunner{
		output: []byte(`{"labels":[{"name":"plan-approved"},{"name":"bug"}]}`),
	}
	found, err := CheckLabel(context.Background(), r, "owner/repo", 42, "plan-approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected label to be found")
	}
}

func TestCheckLabelAbsent(t *testing.T) {
	r := &mockRunner{
		output: []byte(`{"labels":[{"name":"bug"},{"name":"enhancement"}]}`),
	}
	found, err := CheckLabel(context.Background(), r, "owner/repo", 42, "plan-approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected label to not be found")
	}
}

func TestCheckLabelEmptyLabels(t *testing.T) {
	r := &mockRunner{
		output: []byte(`{"labels":[]}`),
	}
	found, err := CheckLabel(context.Background(), r, "owner/repo", 42, "plan-approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected label to not be found with empty labels")
	}
}

func TestCheckLabelMultipleWithMatch(t *testing.T) {
	r := &mockRunner{
		output: []byte(`{"labels":[{"name":"bug"},{"name":"plan-approved"},{"name":"priority-high"}]}`),
	}
	found, err := CheckLabel(context.Background(), r, "owner/repo", 99, "plan-approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected label to be found among multiple labels")
	}
}

func TestCheckLabelGhFails(t *testing.T) {
	r := &mockRunner{err: errors.New("gh: not authenticated")}
	found, err := CheckLabel(context.Background(), r, "owner/repo", 42, "plan-approved")
	if err == nil {
		t.Fatal("expected error when gh fails, got nil")
	}
	if found {
		t.Error("expected found to be false on error")
	}
}

func TestCheckLabelInvalidJSON(t *testing.T) {
	r := &mockRunner{output: []byte(`{not json`)}
	found, err := CheckLabel(context.Background(), r, "owner/repo", 42, "plan-approved")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if found {
		t.Error("expected found to be false on parse error")
	}
}

// --- shellQuote tests ---

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple path", "/tmp/work", "'/tmp/work'"},
		{"path with spaces", "/tmp/my dir", "'/tmp/my dir'"},
		{"path with single quote", "/tmp/it's", "'/tmp/it'\\''s'"},
		{"empty string", "", "''"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shellQuote(tc.in)
			if got != tc.want {
				t.Errorf("shellQuote(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
