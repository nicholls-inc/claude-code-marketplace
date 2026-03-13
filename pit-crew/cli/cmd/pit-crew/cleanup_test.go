package main

import (
	"context"
	"strings"
	"testing"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/worktree"
)

type emptyWorktreeRunner struct{}

func (e *emptyWorktreeRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return []byte{}, nil
}

func TestCleanupNoWorktrees(t *testing.T) {
	dir := t.TempDir()
	wt := worktree.New(dir, &emptyWorktreeRunner{})

	out := captureStdout(func() { cmdCleanup(wt, nil) })
	if !strings.Contains(out, "No pit-crew worktrees") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestCleanupDryRunNoWorktrees(t *testing.T) {
	dir := t.TempDir()
	wt := worktree.New(dir, &emptyWorktreeRunner{})

	out := captureStdout(func() { cmdCleanup(wt, []string{"--dry-run"}) })
	if !strings.Contains(out, "No pit-crew worktrees") {
		t.Errorf("expected empty message, got: %s", out)
	}
}
