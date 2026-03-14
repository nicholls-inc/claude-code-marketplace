package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/worktree"
)

// setupTestDeps injects test dependencies into the global deps variable,
// bypassing PersistentPreRunE which requires gh/git on PATH and a real config.
func setupTestDeps(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	deps = &appDeps{
		cfg: &config.Config{
			Repo:     "owner/repo",
			StateDir: dir,
			Exclude:  []string{},
			Tasks:    map[string]config.Task{},
			Claude:   config.ClaudeConfig{Command: "claude"},
		},
		q:  queue.New(filepath.Join(dir, "queue.jsonl")),
		wt: worktree.New(dir, &emptyWorktreeRunner{}),
	}
}

func TestCobraSubcommandRegistration(t *testing.T) {
	cmd := newRootCmd()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}

	expected := []string{"scan", "drain", "status", "pause", "resume", "cancel", "cleanup"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q to be registered", name)
		}
	}
	if len(cmd.Commands()) != len(expected) {
		t.Errorf("expected %d subcommands, got %d", len(expected), len(cmd.Commands()))
	}
}

func TestCobraUnknownSubcommand(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"bogus"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}

func TestCobraCancelRequiresArgs(t *testing.T) {
	setupTestDeps(t)
	cmd := newRootCmd()
	// Skip PersistentPreRunE
	cmd.PersistentPreRunE = nil
	cmd.SetArgs([]string{"cancel"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when cancel has no args")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected ExactArgs error, got: %v", err)
	}
}

func TestCobraStatusJsonFlag(t *testing.T) {
	setupTestDeps(t)
	cmd := newRootCmd()
	cmd.PersistentPreRunE = nil
	cmd.SetArgs([]string{"status", "--json"})

	out := captureStdout(func() {
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// --json with empty queue should output []
	trimmed := strings.TrimSpace(out)
	if trimmed != "[]" {
		t.Errorf("expected '[]' for --json empty status, got: %q", trimmed)
	}
}

func TestCobraScanDryRunFlag(t *testing.T) {
	setupTestDeps(t)
	cmd := newRootCmd()
	cmd.PersistentPreRunE = nil

	// With no tasks configured and dry-run, scan should execute without error
	// (empty scan returns "No new issues found")
	out := captureStdout(func() {
		cmd.SetArgs([]string{"scan", "--dry-run"})
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Since no tasks are configured, scan returns immediately with no output
	// or "No new issues found" — the key test is that the flag was parsed.
	_ = out // flag parsing verified by successful execution
}

func TestCobraHelpFlag(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--help"})

	out := captureStdout(func() {
		// --help causes Execute to return nil
		_ = cmd.Execute()
	})

	if !strings.Contains(out, "pit-crew") {
		t.Errorf("expected 'pit-crew' in help text, got: %s", out)
	}
	if !strings.Contains(out, "Available Commands") {
		t.Errorf("expected 'Available Commands' in help text, got: %s", out)
	}
}
