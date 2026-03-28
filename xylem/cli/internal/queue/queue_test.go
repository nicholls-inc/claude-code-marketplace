package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// helperTransitionToWaiting transitions a vessel from pending -> running -> waiting via the queue.
func helperTransitionToWaiting(t *testing.T, q *Queue, id string) {
	t.Helper()
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if err := q.Update(id, StateWaiting, ""); err != nil {
		t.Fatalf("update to waiting: %v", err)
	}
}

func newTestQueue(t *testing.T) (*Queue, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "queue.jsonl")
	return New(path), path
}

func testVessel(issue int) Vessel {
	return Vessel{
		ID:     fmt.Sprintf("issue-%d", issue),
		Source: "github-issue",
		Ref:    fmt.Sprintf("https://github.com/example/repo/issues/%d", issue),
		Skill:  "fix-bug",
		Meta:   map[string]string{"issue_num": fmt.Sprintf("%d", issue)},
		State:  StatePending,
		CreatedAt: time.Now().UTC(),
	}
}

func readNonEmptyLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read queue file: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func TestEnqueue(t *testing.T) {
	q, path := newTestQueue(t)
	vessel := testVessel(42)

	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	lines := readNonEmptyLines(t, path)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var got Vessel
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("unmarshal line: %v", err)
	}
	if got.ID != "issue-42" {
		t.Fatalf("expected id issue-42, got %q", got.ID)
	}
	if got.Source != "github-issue" {
		t.Fatalf("expected source github-issue, got %q", got.Source)
	}
	if got.Ref != "https://github.com/example/repo/issues/42" {
		t.Fatalf("expected ref issue URL, got %q", got.Ref)
	}
	if got.State != StatePending {
		t.Fatalf("expected state pending, got %q", got.State)
	}
}

func TestDequeue(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(1)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected vessel, got nil")
	}
	if got.State != StateRunning {
		t.Fatalf("expected running, got %q", got.State)
	}
	if got.StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}
}

func TestDequeueEmpty(t *testing.T) {
	q, _ := newTestQueue(t)
	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue empty: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil vessel, got %+v", *got)
	}
}

func TestUpdate(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(2)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected vessel")
	}

	if err := q.Update(got.ID, StateCompleted, ""); err != nil {
		t.Fatalf("update completed: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vessels) != 1 {
		t.Fatalf("expected 1 vessel, got %d", len(vessels))
	}
	if vessels[0].State != StateCompleted {
		t.Fatalf("expected completed, got %q", vessels[0].State)
	}
	if vessels[0].EndedAt == nil {
		t.Fatal("expected EndedAt to be set")
	}
}

func TestUpdateFailed(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(3)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// Must transition through running before going to failed.
	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected vessel")
	}

	if err := q.Update(got.ID, StateFailed, "boom"); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vessels) != 1 {
		t.Fatalf("expected 1 vessel, got %d", len(vessels))
	}
	if vessels[0].State != StateFailed {
		t.Fatalf("expected failed, got %q", vessels[0].State)
	}
	if vessels[0].Error != "boom" {
		t.Fatalf("expected error boom, got %q", vessels[0].Error)
	}
	if vessels[0].EndedAt == nil {
		t.Fatal("expected EndedAt to be set")
	}
}

func TestCancel(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(4)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := q.Cancel(vessel.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if vessels[0].State != StateCancelled {
		t.Fatalf("expected cancelled, got %q", vessels[0].State)
	}
}

func TestCancelRunning(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(5)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}

	if err := q.Cancel(vessel.ID); err == nil {
		t.Fatal("expected error cancelling running vessel")
	}
}

func TestCancelCompleted(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(6)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// Must go through pending -> running -> completed.
	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected vessel")
	}
	if err := q.Update(got.ID, StateCompleted, ""); err != nil {
		t.Fatalf("update completed: %v", err)
	}

	if err := q.Cancel(vessel.ID); err == nil {
		t.Fatal("expected error cancelling completed vessel")
	}
}

func TestCancelNotFound(t *testing.T) {
	q, _ := newTestQueue(t)
	if err := q.Cancel("issue-999"); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestHasRef(t *testing.T) {
	q, _ := newTestQueue(t)
	if err := q.Enqueue(testVessel(42)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !q.HasRef("https://github.com/example/repo/issues/42") {
		t.Fatal("expected HasRef to be true for enqueued ref")
	}
	if q.HasRef("https://github.com/example/repo/issues/99") {
		t.Fatal("expected HasRef to be false for unknown ref")
	}
}

func TestHasRefCancelled(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(42)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := q.Cancel(vessel.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	if q.HasRef("https://github.com/example/repo/issues/42") {
		t.Fatal("expected cancelled vessel to not count in HasRef")
	}
}

func TestCorruption(t *testing.T) {
	q, path := newTestQueue(t)
	j1 := testVessel(7)
	j2 := testVessel(8)

	b1, err := json.Marshal(j1)
	if err != nil {
		t.Fatalf("marshal j1: %v", err)
	}
	b2, err := json.Marshal(j2)
	if err != nil {
		t.Fatalf("marshal j2: %v", err)
	}

	content := strings.Join([]string{string(b1), "{not-json", string(b2)}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write corruption file: %v", err)
	}

	vessels, err := q.List()
	// readAllVessels now returns an error when malformed lines are encountered,
	// but still returns the valid vessels that could be parsed.
	if err == nil {
		t.Fatal("expected error for malformed entries")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("expected malformed error, got: %v", err)
	}
	if len(vessels) != 2 {
		t.Fatalf("expected 2 valid vessels despite malformed line, got %d", len(vessels))
	}
}

func TestConcurrentEnqueue(t *testing.T) {
	q, _ := newTestQueue(t)
	const workers = 10

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			vessel := testVessel(100 + i)
			if err := q.Enqueue(vessel); err != nil {
				t.Errorf("enqueue %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vessels) != workers {
		t.Fatalf("expected %d vessels, got %d", workers, len(vessels))
	}
}

func TestListByState(t *testing.T) {
	q, _ := newTestQueue(t)
	vessels := []Vessel{testVessel(200), testVessel(201), testVessel(202)}
	vessels[1].State = StateRunning
	vessels[2].State = StateCompleted

	for _, vessel := range vessels {
		if err := q.Enqueue(vessel); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	pending, err := q.ListByState(StatePending)
	if err != nil {
		t.Fatalf("list by state: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
}

// --- State transition validation tests ---

func TestUpdateInvalidTransitionCompletedToPending(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(10)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if err := q.Update(vessel.ID, StateCompleted, ""); err != nil {
		t.Fatalf("complete: %v", err)
	}

	err := q.Update(vessel.ID, StatePending, "")
	if err == nil {
		t.Fatal("expected error for completed->pending transition")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestUpdateInvalidTransitionPendingToCompleted(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(11)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	err := q.Update(vessel.ID, StateCompleted, "")
	if err == nil {
		t.Fatal("expected error for pending->completed transition")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestUpdateInvalidTransitionPendingToFailed(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(12)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	err := q.Update(vessel.ID, StateFailed, "boom")
	if err == nil {
		t.Fatal("expected error for pending->failed transition")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestUpdateInvalidTransitionCancelledToRunning(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(13)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := q.Cancel(vessel.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	err := q.Update(vessel.ID, StateRunning, "")
	if err == nil {
		t.Fatal("expected error for cancelled->running transition")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestUpdateValidTransitionFailedToPending(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(14)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if err := q.Update(vessel.ID, StateFailed, "transient error"); err != nil {
		t.Fatalf("fail: %v", err)
	}

	// Retry: failed -> pending should be allowed.
	if err := q.Update(vessel.ID, StatePending, ""); err != nil {
		t.Fatalf("expected failed->pending to succeed for retry, got: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if vessels[0].State != StatePending {
		t.Fatalf("expected pending after retry, got %q", vessels[0].State)
	}
}

func TestUpdateRunningToCancelled(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(16)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}

	if err := q.Update(vessel.ID, StateCancelled, ""); err != nil {
		t.Fatalf("running->cancelled: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if vessels[0].State != StateCancelled {
		t.Fatalf("expected cancelled, got %q", vessels[0].State)
	}
}

func TestConcurrentUpdateAndList(t *testing.T) {
	q, _ := newTestQueue(t)
	const numVessels = 5

	// Enqueue and dequeue to get vessels into running state.
	for i := 0; i < numVessels; i++ {
		if err := q.Enqueue(testVessel(800 + i)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	for i := 0; i < numVessels; i++ {
		if _, err := q.Dequeue(); err != nil {
			t.Fatalf("dequeue: %v", err)
		}
	}

	var wg sync.WaitGroup

	// Concurrently update vessels to completed while reading.
	wg.Add(numVessels)
	for i := 0; i < numVessels; i++ {
		i := i
		go func() {
			defer wg.Done()
			err := q.Update(fmt.Sprintf("issue-%d", 800+i), StateCompleted, "")
			if err != nil {
				t.Errorf("update %d: %v", i, err)
			}
		}()
	}

	wg.Add(numVessels)
	for i := 0; i < numVessels; i++ {
		go func() {
			defer wg.Done()
			_, _ = q.List()
			_, _ = q.ListByState(StateCompleted)
		}()
	}

	wg.Wait()

	// All vessels should be completed.
	vessels, err := q.List()
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	for _, vessel := range vessels {
		if vessel.State != StateCompleted {
			t.Errorf("vessel %s: expected completed, got %s", vessel.ID, vessel.State)
		}
	}
}

// --- Additional coverage tests ---

func TestUpdateNonExistentVessel(t *testing.T) {
	q, _ := newTestQueue(t)
	if err := q.Enqueue(testVessel(1)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	err := q.Update("issue-999", StateCompleted, "")
	if err == nil {
		t.Fatal("expected error for non-existent vessel")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got: %v", err)
	}
}

func TestUpdateRunningBranchSetsTimestamps(t *testing.T) {
	// Cover the StateRunning case in Update's switch: sets StartedAt if nil,
	// clears EndedAt and Error.
	q, _ := newTestQueue(t)
	vessel := testVessel(20)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// pending -> running via Update (instead of Dequeue)
	if err := q.Update(vessel.ID, StateRunning, ""); err != nil {
		t.Fatalf("update to running: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if vessels[0].State != StateRunning {
		t.Fatalf("expected running, got %q", vessels[0].State)
	}
	if vessels[0].StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}
	if vessels[0].EndedAt != nil {
		t.Fatal("expected EndedAt to be nil")
	}
	if vessels[0].Error != "" {
		t.Fatalf("expected empty error, got %q", vessels[0].Error)
	}
}

func TestDequeueSkipsNonPending(t *testing.T) {
	// Dequeue should pick the first pending vessel, skipping running/completed.
	q, _ := newTestQueue(t)
	j1 := testVessel(30)
	j1.State = StateRunning // already running
	j2 := testVessel(31)
	j2.State = StateCompleted
	j3 := testVessel(32) // pending

	for _, j := range []Vessel{j1, j2, j3} {
		if err := q.Enqueue(j); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected a vessel, got nil")
	}
	if got.ID != "issue-32" {
		t.Fatalf("expected issue-32 (first pending), got %s", got.ID)
	}
}

func TestBlankLinesIgnored(t *testing.T) {
	q, path := newTestQueue(t)
	j := testVessel(60)
	b, _ := json.Marshal(j)
	// File with blank lines interspersed
	content := "\n\n" + string(b) + "\n\n\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vessels) != 1 {
		t.Fatalf("expected 1 vessel (blank lines ignored), got %d", len(vessels))
	}
	_ = q // satisfy vet
}

func TestLegacyJSONLMigration(t *testing.T) {
	_, path := newTestQueue(t)
	q := New(path)

	// Write a legacy-format entry with issue_url and issue_num
	legacy := `{"id":"issue-42","issue_url":"https://github.com/example/repo/issues/42","issue_num":42,"skill":"fix-bug","state":"pending","created_at":"2026-01-01T00:00:00Z"}`
	if err := os.WriteFile(path, []byte(legacy+"\n"), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vessels) != 1 {
		t.Fatalf("expected 1 vessel, got %d", len(vessels))
	}
	v := vessels[0]
	if v.Source != "github-issue" {
		t.Fatalf("expected source github-issue, got %q", v.Source)
	}
	if v.Ref != "https://github.com/example/repo/issues/42" {
		t.Fatalf("expected ref from issue_url, got %q", v.Ref)
	}
	if v.Meta["issue_num"] != "42" {
		t.Fatalf("expected meta issue_num=42, got %q", v.Meta["issue_num"])
	}
}

// --- v2 state and field tests ---

func TestWaitingState(t *testing.T) {
	tests := []struct {
		name      string
		toState   VesselState
		errMsg    string
		wantErr   bool
		wantState VesselState
	}{
		{
			name:      "running to waiting",
			toState:   StateWaiting,
			wantState: StateWaiting,
		},
		{
			name:      "waiting to running (resume)",
			toState:   StateRunning,
			wantState: StateRunning,
		},
		{
			name:      "waiting to timed_out",
			toState:   StateTimedOut,
			errMsg:    "gate timeout",
			wantState: StateTimedOut,
		},
		{
			name:      "waiting to cancelled",
			toState:   StateCancelled,
			wantState: StateCancelled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q, _ := newTestQueue(t)
			vessel := testVessel(500)
			if err := q.Enqueue(vessel); err != nil {
				t.Fatalf("enqueue: %v", err)
			}

			// Get to running state.
			if _, err := q.Dequeue(); err != nil {
				t.Fatalf("dequeue: %v", err)
			}

			// For tests starting from waiting, transition to waiting first.
			if tc.toState != StateWaiting {
				if err := q.Update(vessel.ID, StateWaiting, ""); err != nil {
					t.Fatalf("transition to waiting: %v", err)
				}
			}

			err := q.Update(vessel.ID, tc.toState, tc.errMsg)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Update() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			vessels, err := q.List()
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if vessels[0].State != tc.wantState {
				t.Fatalf("expected state %s, got %s", tc.wantState, vessels[0].State)
			}
		})
	}

	// Verify waiting does NOT allow -> completed or -> failed.
	t.Run("waiting to completed is invalid", func(t *testing.T) {
		q, _ := newTestQueue(t)
		vessel := testVessel(501)
		if err := q.Enqueue(vessel); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		helperTransitionToWaiting(t, q, vessel.ID)

		err := q.Update(vessel.ID, StateCompleted, "")
		if err == nil {
			t.Fatal("expected error for waiting->completed transition")
		}
		if !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("expected ErrInvalidTransition, got: %v", err)
		}
	})

	t.Run("waiting to failed is invalid", func(t *testing.T) {
		q, _ := newTestQueue(t)
		vessel := testVessel(502)
		if err := q.Enqueue(vessel); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		helperTransitionToWaiting(t, q, vessel.ID)

		err := q.Update(vessel.ID, StateFailed, "boom")
		if err == nil {
			t.Fatal("expected error for waiting->failed transition")
		}
		if !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("expected ErrInvalidTransition, got: %v", err)
		}
	})
}

func TestWaitingStateDoesNotSetEndedAt(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(503)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	helperTransitionToWaiting(t, q, vessel.ID)

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if vessels[0].EndedAt != nil {
		t.Fatal("expected EndedAt to be nil for waiting state")
	}
}

func TestTimedOutState(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(510)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	helperTransitionToWaiting(t, q, vessel.ID)

	if err := q.Update(vessel.ID, StateTimedOut, "gate timeout"); err != nil {
		t.Fatalf("update to timed_out: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if vessels[0].State != StateTimedOut {
		t.Fatalf("expected timed_out, got %s", vessels[0].State)
	}
	if vessels[0].EndedAt == nil {
		t.Fatal("expected EndedAt to be set for timed_out")
	}
	if vessels[0].Error != "gate timeout" {
		t.Fatalf("expected error 'gate timeout', got %q", vessels[0].Error)
	}

	// timed_out is terminal — no transitions out.
	for _, target := range []VesselState{StatePending, StateRunning, StateWaiting, StateCompleted, StateFailed, StateCancelled} {
		err := q.Update(vessel.ID, target, "")
		if err == nil {
			t.Fatalf("expected error for timed_out->%s transition", target)
		}
		if !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("expected ErrInvalidTransition for timed_out->%s, got: %v", target, err)
		}
	}
}

func TestV2VesselFields(t *testing.T) {
	q, _ := newTestQueue(t)
	now := time.Now().UTC()
	vessel := Vessel{
		ID:           "v2-test-1",
		Source:       "github-issue",
		Ref:          "https://github.com/example/repo/issues/99",
		Skill:        "fix-bug",
		State:        StatePending,
		CreatedAt:    now,
		CurrentPhase: 2,
		PhaseOutputs: map[string]string{"plan": "done", "implement": "in-progress"},
		GateRetries:  3,
		WaitingSince: &now,
		WaitingFor:   "review-label",
		WorktreePath: "/tmp/worktree-abc",
		FailedPhase:  "implement",
		GateOutput:   "label not found",
		RetryOf:      "v2-test-0",
	}

	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vessels) != 1 {
		t.Fatalf("expected 1 vessel, got %d", len(vessels))
	}

	got := vessels[0]
	if got.CurrentPhase != 2 {
		t.Fatalf("expected CurrentPhase 2, got %d", got.CurrentPhase)
	}
	if got.PhaseOutputs["plan"] != "done" {
		t.Fatalf("expected PhaseOutputs[plan]=done, got %q", got.PhaseOutputs["plan"])
	}
	if got.PhaseOutputs["implement"] != "in-progress" {
		t.Fatalf("expected PhaseOutputs[implement]=in-progress, got %q", got.PhaseOutputs["implement"])
	}
	if got.GateRetries != 3 {
		t.Fatalf("expected GateRetries 3, got %d", got.GateRetries)
	}
	if got.WaitingSince == nil {
		t.Fatal("expected WaitingSince to be set")
	}
	if got.WaitingFor != "review-label" {
		t.Fatalf("expected WaitingFor 'review-label', got %q", got.WaitingFor)
	}
	if got.WorktreePath != "/tmp/worktree-abc" {
		t.Fatalf("expected WorktreePath '/tmp/worktree-abc', got %q", got.WorktreePath)
	}
	if got.FailedPhase != "implement" {
		t.Fatalf("expected FailedPhase 'implement', got %q", got.FailedPhase)
	}
	if got.GateOutput != "label not found" {
		t.Fatalf("expected GateOutput 'label not found', got %q", got.GateOutput)
	}
	if got.RetryOf != "v2-test-0" {
		t.Fatalf("expected RetryOf 'v2-test-0', got %q", got.RetryOf)
	}
}

func TestBackwardCompat(t *testing.T) {
	_, path := newTestQueue(t)
	q := New(path)

	// Write a JSONL line with only v1 fields (no v2 fields in JSON).
	v1JSON := `{"id":"compat-1","source":"github-issue","ref":"https://github.com/example/repo/issues/1","skill":"fix-bug","state":"pending","created_at":"2026-01-01T00:00:00Z"}`
	if err := os.WriteFile(path, []byte(v1JSON+"\n"), 0o644); err != nil {
		t.Fatalf("write v1 json: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vessels) != 1 {
		t.Fatalf("expected 1 vessel, got %d", len(vessels))
	}

	got := vessels[0]
	// v1 fields present.
	if got.ID != "compat-1" {
		t.Fatalf("expected id compat-1, got %q", got.ID)
	}
	if got.State != StatePending {
		t.Fatalf("expected pending, got %q", got.State)
	}
	// v2 fields should be zero values.
	if got.CurrentPhase != 0 {
		t.Fatalf("expected CurrentPhase 0, got %d", got.CurrentPhase)
	}
	if got.PhaseOutputs != nil {
		t.Fatalf("expected nil PhaseOutputs, got %v", got.PhaseOutputs)
	}
	if got.GateRetries != 0 {
		t.Fatalf("expected GateRetries 0, got %d", got.GateRetries)
	}
	if got.WaitingSince != nil {
		t.Fatal("expected nil WaitingSince")
	}
	if got.WaitingFor != "" {
		t.Fatalf("expected empty WaitingFor, got %q", got.WaitingFor)
	}
	if got.WorktreePath != "" {
		t.Fatalf("expected empty WorktreePath, got %q", got.WorktreePath)
	}
	if got.FailedPhase != "" {
		t.Fatalf("expected empty FailedPhase, got %q", got.FailedPhase)
	}
	if got.GateOutput != "" {
		t.Fatalf("expected empty GateOutput, got %q", got.GateOutput)
	}
	if got.RetryOf != "" {
		t.Fatalf("expected empty RetryOf, got %q", got.RetryOf)
	}
}

func TestUpdateVessel(t *testing.T) {
	t.Run("update phase tracking fields", func(t *testing.T) {
		q, _ := newTestQueue(t)
		vessel := testVessel(600)
		if err := q.Enqueue(vessel); err != nil {
			t.Fatalf("enqueue: %v", err)
		}

		// Read it back, modify v2 fields, and persist.
		vessels, err := q.List()
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		updated := vessels[0]
		updated.CurrentPhase = 3
		updated.PhaseOutputs = map[string]string{"plan": "ok", "implement": "ok", "test": "running"}
		updated.WorktreePath = "/tmp/wt-600"

		if err := q.UpdateVessel(updated); err != nil {
			t.Fatalf("UpdateVessel: %v", err)
		}

		got, err := q.FindByID(vessel.ID)
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if got.CurrentPhase != 3 {
			t.Fatalf("expected CurrentPhase 3, got %d", got.CurrentPhase)
		}
		if got.PhaseOutputs["test"] != "running" {
			t.Fatalf("expected PhaseOutputs[test]=running, got %q", got.PhaseOutputs["test"])
		}
		if got.WorktreePath != "/tmp/wt-600" {
			t.Fatalf("expected WorktreePath '/tmp/wt-600', got %q", got.WorktreePath)
		}
	})

	t.Run("update non-existent vessel returns error", func(t *testing.T) {
		q, _ := newTestQueue(t)
		ghost := Vessel{ID: "ghost-999"}
		err := q.UpdateVessel(ghost)
		if err == nil {
			t.Fatal("expected error for non-existent vessel")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not-found error, got: %v", err)
		}
	})
}

func TestFindByID(t *testing.T) {
	t.Run("find existing vessel", func(t *testing.T) {
		q, _ := newTestQueue(t)
		vessel := testVessel(700)
		if err := q.Enqueue(vessel); err != nil {
			t.Fatalf("enqueue: %v", err)
		}

		got, err := q.FindByID(vessel.ID)
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if got.ID != vessel.ID {
			t.Fatalf("expected id %s, got %s", vessel.ID, got.ID)
		}
		if got.Source != vessel.Source {
			t.Fatalf("expected source %s, got %s", vessel.Source, got.Source)
		}
	})

	t.Run("find non-existent vessel returns error", func(t *testing.T) {
		q, _ := newTestQueue(t)
		_, err := q.FindByID("does-not-exist")
		if err == nil {
			t.Fatal("expected error for non-existent vessel")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not-found error, got: %v", err)
		}
	})
}

func TestCancelWaitingVessel(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(900)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	helperTransitionToWaiting(t, q, vessel.ID)

	// Cancel the waiting vessel.
	if err := q.Cancel(vessel.ID); err != nil {
		t.Fatalf("cancel waiting vessel: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if vessels[0].State != StateCancelled {
		t.Fatalf("expected cancelled, got %s", vessels[0].State)
	}
	if vessels[0].EndedAt == nil {
		t.Fatal("expected EndedAt to be set after cancel")
	}
}

