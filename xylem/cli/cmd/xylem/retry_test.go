package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
)

func newRetryTestQueue(t *testing.T) *queue.Queue {
	t.Helper()
	return queue.New(filepath.Join(t.TempDir(), "queue.jsonl"))
}

func TestRetryCreatesNewVessel(t *testing.T) {
	q := newRetryTestQueue(t)
	now := time.Now().UTC()
	v := queue.Vessel{
		ID: "issue-42", Source: "github-issue", Skill: "fix-bug",
		Ref: "https://github.com/owner/repo/issues/42",
		Meta:  map[string]string{"issue_num": "42"},
		State: queue.StatePending, CreatedAt: now,
	}
	q.Enqueue(v) //nolint:errcheck
	q.Update("issue-42", queue.StateRunning, "")            //nolint:errcheck
	q.Update("issue-42", queue.StateFailed, "test error")   //nolint:errcheck

	err := cmdRetry(q, "issue-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vessels, _ := q.List()
	var found bool
	for _, v := range vessels {
		if v.ID == "issue-42-retry-1" {
			found = true
			if v.State != queue.StatePending {
				t.Errorf("retry should be pending, got %s", v.State)
			}
			if v.Skill != "fix-bug" {
				t.Errorf("retry should have same skill, got %s", v.Skill)
			}
			if v.Source != "github-issue" {
				t.Errorf("retry should have same source, got %s", v.Source)
			}
			if v.Ref != "https://github.com/owner/repo/issues/42" {
				t.Errorf("retry should have same ref, got %s", v.Ref)
			}
			if v.RetryOf != "issue-42" {
				t.Errorf("retry_of should be issue-42, got %s", v.RetryOf)
			}
			if v.Meta["retry_of"] != "issue-42" {
				t.Errorf("meta retry_of should be issue-42, got %s", v.Meta["retry_of"])
			}
			if v.Meta["retry_error"] != "test error" {
				t.Errorf("meta retry_error should be 'test error', got %s", v.Meta["retry_error"])
			}
			if v.Meta["issue_num"] != "42" {
				t.Errorf("meta issue_num should be preserved, got %s", v.Meta["issue_num"])
			}
		}
	}
	if !found {
		t.Error("retry vessel issue-42-retry-1 not found")
	}
}

func TestRetryNonFailedVessel(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, q *queue.Queue)
	}{
		{
			name: "pending",
			setup: func(t *testing.T, q *queue.Queue) {
				t.Helper()
				q.Enqueue(queue.Vessel{ID: "issue-1", Source: "manual", State: queue.StatePending, CreatedAt: time.Now().UTC()}) //nolint:errcheck
			},
		},
		{
			name: "completed",
			setup: func(t *testing.T, q *queue.Queue) {
				t.Helper()
				q.Enqueue(queue.Vessel{ID: "issue-1", Source: "manual", State: queue.StatePending, CreatedAt: time.Now().UTC()}) //nolint:errcheck
				q.Update("issue-1", queue.StateRunning, "")                                                                     //nolint:errcheck
				q.Update("issue-1", queue.StateCompleted, "")                                                                   //nolint:errcheck
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := newRetryTestQueue(t)
			tt.setup(t, q)

			err := cmdRetry(q, "issue-1")
			if err == nil {
				t.Fatal("expected error retrying non-failed vessel")
			}
			if !strings.Contains(err.Error(), "not in failed state") {
				t.Errorf("expected 'not in failed state' error, got: %v", err)
			}
		})
	}
}

func TestRetryNonexistentVessel(t *testing.T) {
	q := newRetryTestQueue(t)

	err := cmdRetry(q, "issue-999")
	if err == nil {
		t.Fatal("expected error retrying non-existent vessel")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestRetryMultipleRetries(t *testing.T) {
	q := newRetryTestQueue(t)
	now := time.Now().UTC()

	q.Enqueue(queue.Vessel{ID: "issue-42", Source: "manual", Skill: "fix-bug", State: queue.StatePending, CreatedAt: now}) //nolint:errcheck
	q.Update("issue-42", queue.StateRunning, "")                                                                           //nolint:errcheck
	q.Update("issue-42", queue.StateFailed, "first error")                                                                 //nolint:errcheck

	// First retry
	if err := cmdRetry(q, "issue-42"); err != nil {
		t.Fatalf("first retry: %v", err)
	}

	// Second retry of the original
	if err := cmdRetry(q, "issue-42"); err != nil {
		t.Fatalf("second retry: %v", err)
	}

	vessels, _ := q.List()
	retryIDs := make(map[string]bool)
	for _, v := range vessels {
		if strings.HasPrefix(v.ID, "issue-42-retry-") {
			retryIDs[v.ID] = true
		}
	}

	if !retryIDs["issue-42-retry-1"] {
		t.Error("expected issue-42-retry-1")
	}
	if !retryIDs["issue-42-retry-2"] {
		t.Error("expected issue-42-retry-2")
	}
}

func TestRetryIDGeneration(t *testing.T) {
	tests := []struct {
		name       string
		originalID string
		existing   []string
		expected   string
	}{
		{
			name:       "first retry",
			originalID: "issue-42",
			existing:   nil,
			expected:   "issue-42-retry-1",
		},
		{
			name:       "second retry",
			originalID: "issue-42",
			existing:   []string{"issue-42-retry-1"},
			expected:   "issue-42-retry-2",
		},
		{
			name:       "gap in numbering",
			originalID: "issue-42",
			existing:   []string{"issue-42-retry-1", "issue-42-retry-3"},
			expected:   "issue-42-retry-4",
		},
		{
			name:       "unrelated vessels ignored",
			originalID: "issue-42",
			existing:   []string{"issue-99-retry-1"},
			expected:   "issue-42-retry-1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := newRetryTestQueue(t)
			now := time.Now().UTC()
			for _, id := range tt.existing {
				q.Enqueue(queue.Vessel{ID: id, Source: "manual", State: queue.StatePending, CreatedAt: now}) //nolint:errcheck
			}

			got := retryID(tt.originalID, q)
			if got != tt.expected {
				t.Errorf("retryID(%q) = %q, want %q", tt.originalID, got, tt.expected)
			}
		})
	}
}

func TestRetryPreservesPrompt(t *testing.T) {
	q := newRetryTestQueue(t)
	now := time.Now().UTC()

	q.Enqueue(queue.Vessel{
		ID: "task-1", Source: "manual", Prompt: "fix the thing",
		State: queue.StatePending, CreatedAt: now,
	}) //nolint:errcheck
	q.Update("task-1", queue.StateRunning, "")        //nolint:errcheck
	q.Update("task-1", queue.StateFailed, "timed out") //nolint:errcheck

	if err := cmdRetry(q, "task-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vessels, _ := q.List()
	for _, v := range vessels {
		if v.ID == "task-1-retry-1" {
			if v.Prompt != "fix the thing" {
				t.Errorf("expected prompt preserved, got %q", v.Prompt)
			}
			return
		}
	}
	t.Error("retry vessel not found")
}
