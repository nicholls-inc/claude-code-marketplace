package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/pit-crew/cli/internal/queue"
)

func makeControlConfig(dir string) *config.Config {
	return &config.Config{
		Repo:     "owner/repo",
		StateDir: dir,
		Exclude:  []string{},
		Tasks:    map[string]config.Task{},
	}
}

func TestPauseCreatesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	cmdPause(cfg, nil)

	if _, err := os.Stat(filepath.Join(dir, "paused")); err != nil {
		t.Error("expected pause marker to exist")
	}
}

func TestPauseIdempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	cmdPause(cfg, nil)
	cmdPause(cfg, nil)

	if _, err := os.Stat(filepath.Join(dir, "paused")); err != nil {
		t.Error("expected pause marker to still exist after double pause")
	}
}

func TestResumeRemovesMarker(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	cmdPause(cfg, nil)
	cmdResume(cfg, nil)

	if _, err := os.Stat(filepath.Join(dir, "paused")); err == nil {
		t.Error("expected pause marker to be removed")
	}
}

func TestResumeIdempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	cmdResume(cfg, nil)
}

func TestPauseResumeRoundtrip(t *testing.T) {
	dir := t.TempDir()
	cfg := makeControlConfig(dir)

	if isPaused(cfg) {
		t.Error("should not be paused initially")
	}
	cmdPause(cfg, nil)
	if !isPaused(cfg) {
		t.Error("should be paused after pause")
	}
	cmdResume(cfg, nil)
	if isPaused(cfg) {
		t.Error("should not be paused after resume")
	}
}

func TestCancelPendingJob(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))
	now := time.Now().UTC()
	q.Enqueue(queue.Job{ID: "issue-1", IssueNum: 1, Skill: "fix-bug", State: queue.StatePending, CreatedAt: now}) //nolint:errcheck

	cmdCancel(q, []string{"issue-1"})

	jobs, _ := q.List()
	if jobs[0].State != queue.StateCancelled {
		t.Errorf("expected cancelled, got %s", jobs[0].State)
	}
}

func TestCancelNonExistentJob(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(filepath.Join(dir, "queue.jsonl"))

	if err := q.Cancel("issue-999"); err == nil {
		t.Error("expected error cancelling non-existent job")
	}
}
