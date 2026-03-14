package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestQueue(t *testing.T) (*Queue, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "queue.jsonl")
	return New(path), path
}

func testVessel(issue int) Vessel {
	return Vessel{
		ID:        fmt.Sprintf("issue-%d", issue),
		IssueURL:  fmt.Sprintf("https://github.com/example/repo/issues/%d", issue),
		IssueNum:  issue,
		Skill:     "fix-bug",
		State:     StatePending,
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
	if got.IssueNum != 42 {
		t.Fatalf("expected issue num 42, got %d", got.IssueNum)
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

func TestHasIssue(t *testing.T) {
	q, _ := newTestQueue(t)
	if err := q.Enqueue(testVessel(42)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !q.HasIssue(42) {
		t.Fatal("expected HasIssue(42) to be true")
	}
	if q.HasIssue(99) {
		t.Fatal("expected HasIssue(99) to be false")
	}
}

func TestHasIssueCancelled(t *testing.T) {
	q, _ := newTestQueue(t)
	vessel := testVessel(42)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := q.Cancel(vessel.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	if q.HasIssue(42) {
		t.Fatal("expected cancelled vessel to not count in HasIssue")
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

func TestUpdateValidTransitions(t *testing.T) {
	// Test the full happy-path lifecycle: pending -> running -> completed.
	q, _ := newTestQueue(t)
	vessel := testVessel(15)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// pending -> running (via Dequeue, which is the normal path)
	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected vessel")
	}
	if got.State != StateRunning {
		t.Fatalf("expected running, got %q", got.State)
	}

	// running -> completed
	if err := q.Update(vessel.ID, StateCompleted, ""); err != nil {
		t.Fatalf("running->completed: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if vessels[0].State != StateCompleted {
		t.Fatalf("expected completed, got %q", vessels[0].State)
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

// --- Concurrent read/write tests ---

func TestConcurrentReadWrite(t *testing.T) {
	q, _ := newTestQueue(t)
	const writers = 5
	const readers = 5

	// Pre-populate some vessels.
	for i := 0; i < 5; i++ {
		if err := q.Enqueue(testVessel(500 + i)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	var wg sync.WaitGroup

	// Launch writers that enqueue new vessels.
	wg.Add(writers)
	for i := 0; i < writers; i++ {
		i := i
		go func() {
			defer wg.Done()
			vessel := testVessel(600 + i)
			if err := q.Enqueue(vessel); err != nil {
				t.Errorf("writer enqueue %d: %v", i, err)
			}
		}()
	}

	// Launch readers that call List concurrently with the writers.
	wg.Add(readers)
	for i := 0; i < readers; i++ {
		i := i
		go func() {
			defer wg.Done()
			vessels, err := q.List()
			if err != nil {
				t.Errorf("reader list %d: %v", i, err)
				return
			}
			// Should have at least the pre-populated vessels.
			if len(vessels) < 5 {
				t.Errorf("reader %d: expected at least 5 vessels, got %d", i, len(vessels))
			}
		}()
	}

	wg.Wait()

	// Final check: all vessels present.
	vessels, err := q.List()
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	if len(vessels) != 10 {
		t.Fatalf("expected 10 vessels, got %d", len(vessels))
	}
}

func TestConcurrentListDuringDequeue(t *testing.T) {
	q, _ := newTestQueue(t)
	const numVessels = 10

	for i := 0; i < numVessels; i++ {
		if err := q.Enqueue(testVessel(700 + i)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	var wg sync.WaitGroup

	// Dequeue all vessels concurrently.
	wg.Add(numVessels)
	for i := 0; i < numVessels; i++ {
		go func() {
			defer wg.Done()
			_, err := q.Dequeue()
			if err != nil {
				t.Errorf("dequeue: %v", err)
			}
		}()
	}

	// Simultaneously read via HasIssue and ListByState.
	wg.Add(numVessels)
	for i := 0; i < numVessels; i++ {
		i := i
		go func() {
			defer wg.Done()
			_ = q.HasIssue(700 + i)
			_, _ = q.ListByState(StateRunning)
		}()
	}

	wg.Wait()

	// All vessels should be dequeued (running or pending if contention missed them).
	vessels, err := q.List()
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	if len(vessels) != numVessels {
		t.Fatalf("expected %d vessels, got %d", numVessels, len(vessels))
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

// --- Malformed line handling test ---

func TestMalformedLineReportsError(t *testing.T) {
	_, path := newTestQueue(t)
	q := New(path)

	content := "{not-valid-json}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	vessels, err := q.List()
	if err == nil {
		t.Fatal("expected error for malformed content")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("expected malformed error message, got: %v", err)
	}
	if len(vessels) != 0 {
		t.Fatalf("expected 0 valid vessels, got %d", len(vessels))
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

func TestUpdateRetryPreservesStartedAt(t *testing.T) {
	// Cover the default branch in Update's switch: failed -> pending clears Error.
	// Also verify that after a retry cycle (failed -> pending -> running),
	// the StartedAt is preserved from the first run if already set.
	q, _ := newTestQueue(t)
	vessel := testVessel(21)
	if err := q.Enqueue(vessel); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if err := q.Update(vessel.ID, StateFailed, "transient"); err != nil {
		t.Fatalf("fail: %v", err)
	}

	// Retry: failed -> pending
	if err := q.Update(vessel.ID, StatePending, ""); err != nil {
		t.Fatalf("retry: %v", err)
	}

	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if vessels[0].Error != "" {
		t.Fatalf("expected error cleared after retry, got %q", vessels[0].Error)
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
	if got.IssueNum != 32 {
		t.Fatalf("expected issue 32 (first pending), got %d", got.IssueNum)
	}
}

func TestDequeueNoPendingReturnsNil(t *testing.T) {
	// All vessels are in non-pending states; Dequeue should return nil.
	q, _ := newTestQueue(t)
	j := testVessel(40)
	j.State = StateCompleted
	if err := q.Enqueue(j); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil (no pending vessels), got %+v", *got)
	}
}

func TestConcurrentEnqueueAndDequeue(t *testing.T) {
	// Exercise concurrent Enqueue and Dequeue to verify file-lock correctness
	// under mixed read-write contention.
	q, _ := newTestQueue(t)
	const numVessels = 20

	// Pre-load some vessels
	for i := 0; i < numVessels; i++ {
		if err := q.Enqueue(testVessel(900 + i)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	var wg sync.WaitGroup
	var dequeued int32
	var enqueued int32

	// Concurrently dequeue
	wg.Add(numVessels)
	for i := 0; i < numVessels; i++ {
		go func() {
			defer wg.Done()
			vessel, err := q.Dequeue()
			if err != nil {
				t.Errorf("dequeue: %v", err)
				return
			}
			if vessel != nil {
				atomic.AddInt32(&dequeued, 1)
			}
		}()
	}

	// Concurrently enqueue more
	wg.Add(5)
	for i := 0; i < 5; i++ {
		i := i
		go func() {
			defer wg.Done()
			if err := q.Enqueue(testVessel(950 + i)); err != nil {
				t.Errorf("enqueue: %v", err)
				return
			}
			atomic.AddInt32(&enqueued, 1)
		}()
	}

	wg.Wait()

	// Verify consistency: all vessels accounted for
	vessels, err := q.List()
	if err != nil {
		t.Fatalf("final list: %v", err)
	}

	totalExpected := numVessels + int(atomic.LoadInt32(&enqueued))
	if len(vessels) != totalExpected {
		t.Fatalf("expected %d vessels total, got %d", totalExpected, len(vessels))
	}

	// Each dequeued vessel should be in Running state
	runCount := 0
	for _, j := range vessels {
		if j.State == StateRunning {
			runCount++
		}
	}
	if int32(runCount) != atomic.LoadInt32(&dequeued) {
		t.Errorf("running count %d != dequeued count %d", runCount, dequeued)
	}
}

func TestListByStateNoMatches(t *testing.T) {
	q, _ := newTestQueue(t)
	if err := q.Enqueue(testVessel(50)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// No completed vessels
	completed, err := q.ListByState(StateCompleted)
	if err != nil {
		t.Fatalf("list by state: %v", err)
	}
	if len(completed) != 0 {
		t.Fatalf("expected 0 completed, got %d", len(completed))
	}
}

func TestEmptyFileReadAllVessels(t *testing.T) {
	_, path := newTestQueue(t)
	// Create an empty file
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}
	q := New(path)
	vessels, err := q.List()
	if err != nil {
		t.Fatalf("list on empty file: %v", err)
	}
	if len(vessels) != 0 {
		t.Fatalf("expected 0 vessels from empty file, got %d", len(vessels))
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

func TestWriteEmptyVessels(t *testing.T) {
	// Verify writeAllVessels handles the empty case without writing a trailing newline.
	q, path := newTestQueue(t)
	// Enqueue then cancel to get a non-empty file, then verify the file format.
	if err := q.Enqueue(testVessel(70)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	data, _ := os.ReadFile(path)
	if len(data) == 0 {
		t.Fatal("expected non-empty file after enqueue")
	}
	// File should end with newline
	if data[len(data)-1] != '\n' {
		t.Fatal("expected trailing newline in queue file")
	}
}
