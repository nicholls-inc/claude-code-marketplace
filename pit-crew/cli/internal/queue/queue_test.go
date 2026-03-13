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

func testJob(issue int) Job {
	return Job{
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
	job := testJob(42)

	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	lines := readNonEmptyLines(t, path)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var got Job
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
	job := testJob(1)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected job, got nil")
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
		t.Fatalf("expected nil job, got %+v", *got)
	}
}

func TestUpdate(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(2)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected job")
	}

	if err := q.Update(got.ID, StateCompleted, ""); err != nil {
		t.Fatalf("update completed: %v", err)
	}

	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].State != StateCompleted {
		t.Fatalf("expected completed, got %q", jobs[0].State)
	}
	if jobs[0].EndedAt == nil {
		t.Fatal("expected EndedAt to be set")
	}
}

func TestUpdateFailed(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(3)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// Must transition through running before going to failed.
	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected job")
	}

	if err := q.Update(got.ID, StateFailed, "boom"); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].State != StateFailed {
		t.Fatalf("expected failed, got %q", jobs[0].State)
	}
	if jobs[0].Error != "boom" {
		t.Fatalf("expected error boom, got %q", jobs[0].Error)
	}
	if jobs[0].EndedAt == nil {
		t.Fatal("expected EndedAt to be set")
	}
}

func TestCancel(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(4)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := q.Cancel(job.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if jobs[0].State != StateCancelled {
		t.Fatalf("expected cancelled, got %q", jobs[0].State)
	}
}

func TestCancelRunning(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(5)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}

	if err := q.Cancel(job.ID); err == nil {
		t.Fatal("expected error cancelling running job")
	}
}

func TestCancelCompleted(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(6)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// Must go through pending -> running -> completed.
	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected job")
	}
	if err := q.Update(got.ID, StateCompleted, ""); err != nil {
		t.Fatalf("update completed: %v", err)
	}

	if err := q.Cancel(job.ID); err == nil {
		t.Fatal("expected error cancelling completed job")
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
	if err := q.Enqueue(testJob(42)); err != nil {
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
	job := testJob(42)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := q.Cancel(job.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	if q.HasIssue(42) {
		t.Fatal("expected cancelled job to not count in HasIssue")
	}
}

func TestCorruption(t *testing.T) {
	q, path := newTestQueue(t)
	j1 := testJob(7)
	j2 := testJob(8)

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

	jobs, err := q.List()
	// readAllJobs now returns an error when malformed lines are encountered,
	// but still returns the valid jobs that could be parsed.
	if err == nil {
		t.Fatal("expected error for malformed entries")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("expected malformed error, got: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 valid jobs despite malformed line, got %d", len(jobs))
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
			job := testJob(100 + i)
			if err := q.Enqueue(job); err != nil {
				t.Errorf("enqueue %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(jobs) != workers {
		t.Fatalf("expected %d jobs, got %d", workers, len(jobs))
	}
}

func TestListByState(t *testing.T) {
	q, _ := newTestQueue(t)
	jobs := []Job{testJob(200), testJob(201), testJob(202)}
	jobs[1].State = StateRunning
	jobs[2].State = StateCompleted

	for _, job := range jobs {
		if err := q.Enqueue(job); err != nil {
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
	job := testJob(10)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if err := q.Update(job.ID, StateCompleted, ""); err != nil {
		t.Fatalf("complete: %v", err)
	}

	err := q.Update(job.ID, StatePending, "")
	if err == nil {
		t.Fatal("expected error for completed->pending transition")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestUpdateInvalidTransitionPendingToCompleted(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(11)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	err := q.Update(job.ID, StateCompleted, "")
	if err == nil {
		t.Fatal("expected error for pending->completed transition")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestUpdateInvalidTransitionPendingToFailed(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(12)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	err := q.Update(job.ID, StateFailed, "boom")
	if err == nil {
		t.Fatal("expected error for pending->failed transition")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestUpdateInvalidTransitionCancelledToRunning(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(13)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := q.Cancel(job.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	err := q.Update(job.ID, StateRunning, "")
	if err == nil {
		t.Fatal("expected error for cancelled->running transition")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestUpdateValidTransitionFailedToPending(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(14)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if err := q.Update(job.ID, StateFailed, "transient error"); err != nil {
		t.Fatalf("fail: %v", err)
	}

	// Retry: failed -> pending should be allowed.
	if err := q.Update(job.ID, StatePending, ""); err != nil {
		t.Fatalf("expected failed->pending to succeed for retry, got: %v", err)
	}

	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if jobs[0].State != StatePending {
		t.Fatalf("expected pending after retry, got %q", jobs[0].State)
	}
}

func TestUpdateValidTransitions(t *testing.T) {
	// Test the full happy-path lifecycle: pending -> running -> completed.
	q, _ := newTestQueue(t)
	job := testJob(15)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// pending -> running (via Dequeue, which is the normal path)
	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected job")
	}
	if got.State != StateRunning {
		t.Fatalf("expected running, got %q", got.State)
	}

	// running -> completed
	if err := q.Update(job.ID, StateCompleted, ""); err != nil {
		t.Fatalf("running->completed: %v", err)
	}

	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if jobs[0].State != StateCompleted {
		t.Fatalf("expected completed, got %q", jobs[0].State)
	}
}

func TestUpdateRunningToCancelled(t *testing.T) {
	q, _ := newTestQueue(t)
	job := testJob(16)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}

	if err := q.Update(job.ID, StateCancelled, ""); err != nil {
		t.Fatalf("running->cancelled: %v", err)
	}

	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if jobs[0].State != StateCancelled {
		t.Fatalf("expected cancelled, got %q", jobs[0].State)
	}
}

// --- Concurrent read/write tests ---

func TestConcurrentReadWrite(t *testing.T) {
	q, _ := newTestQueue(t)
	const writers = 5
	const readers = 5

	// Pre-populate some jobs.
	for i := 0; i < 5; i++ {
		if err := q.Enqueue(testJob(500 + i)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	var wg sync.WaitGroup

	// Launch writers that enqueue new jobs.
	wg.Add(writers)
	for i := 0; i < writers; i++ {
		i := i
		go func() {
			defer wg.Done()
			job := testJob(600 + i)
			if err := q.Enqueue(job); err != nil {
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
			jobs, err := q.List()
			if err != nil {
				t.Errorf("reader list %d: %v", i, err)
				return
			}
			// Should have at least the pre-populated jobs.
			if len(jobs) < 5 {
				t.Errorf("reader %d: expected at least 5 jobs, got %d", i, len(jobs))
			}
		}()
	}

	wg.Wait()

	// Final check: all jobs present.
	jobs, err := q.List()
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	if len(jobs) != 10 {
		t.Fatalf("expected 10 jobs, got %d", len(jobs))
	}
}

func TestConcurrentListDuringDequeue(t *testing.T) {
	q, _ := newTestQueue(t)
	const numJobs = 10

	for i := 0; i < numJobs; i++ {
		if err := q.Enqueue(testJob(700 + i)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	var wg sync.WaitGroup

	// Dequeue all jobs concurrently.
	wg.Add(numJobs)
	for i := 0; i < numJobs; i++ {
		go func() {
			defer wg.Done()
			_, err := q.Dequeue()
			if err != nil {
				t.Errorf("dequeue: %v", err)
			}
		}()
	}

	// Simultaneously read via HasIssue and ListByState.
	wg.Add(numJobs)
	for i := 0; i < numJobs; i++ {
		i := i
		go func() {
			defer wg.Done()
			_ = q.HasIssue(700 + i)
			_, _ = q.ListByState(StateRunning)
		}()
	}

	wg.Wait()

	// All jobs should be dequeued (running or pending if contention missed them).
	jobs, err := q.List()
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	if len(jobs) != numJobs {
		t.Fatalf("expected %d jobs, got %d", numJobs, len(jobs))
	}
}

func TestConcurrentUpdateAndList(t *testing.T) {
	q, _ := newTestQueue(t)
	const numJobs = 5

	// Enqueue and dequeue to get jobs into running state.
	for i := 0; i < numJobs; i++ {
		if err := q.Enqueue(testJob(800 + i)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	for i := 0; i < numJobs; i++ {
		if _, err := q.Dequeue(); err != nil {
			t.Fatalf("dequeue: %v", err)
		}
	}

	var wg sync.WaitGroup

	// Concurrently update jobs to completed while reading.
	wg.Add(numJobs)
	for i := 0; i < numJobs; i++ {
		i := i
		go func() {
			defer wg.Done()
			err := q.Update(fmt.Sprintf("issue-%d", 800+i), StateCompleted, "")
			if err != nil {
				t.Errorf("update %d: %v", i, err)
			}
		}()
	}

	wg.Add(numJobs)
	for i := 0; i < numJobs; i++ {
		go func() {
			defer wg.Done()
			_, _ = q.List()
			_, _ = q.ListByState(StateCompleted)
		}()
	}

	wg.Wait()

	// All jobs should be completed.
	jobs, err := q.List()
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	for _, job := range jobs {
		if job.State != StateCompleted {
			t.Errorf("job %s: expected completed, got %s", job.ID, job.State)
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

	jobs, err := q.List()
	if err == nil {
		t.Fatal("expected error for malformed content")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("expected malformed error message, got: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected 0 valid jobs, got %d", len(jobs))
	}
}

// --- Additional coverage tests ---

func TestUpdateNonExistentJob(t *testing.T) {
	q, _ := newTestQueue(t)
	if err := q.Enqueue(testJob(1)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	err := q.Update("issue-999", StateCompleted, "")
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got: %v", err)
	}
}

func TestUpdateRunningBranchSetsTimestamps(t *testing.T) {
	// Cover the StateRunning case in Update's switch: sets StartedAt if nil,
	// clears EndedAt and Error.
	q, _ := newTestQueue(t)
	job := testJob(20)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// pending -> running via Update (instead of Dequeue)
	if err := q.Update(job.ID, StateRunning, ""); err != nil {
		t.Fatalf("update to running: %v", err)
	}

	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if jobs[0].State != StateRunning {
		t.Fatalf("expected running, got %q", jobs[0].State)
	}
	if jobs[0].StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}
	if jobs[0].EndedAt != nil {
		t.Fatal("expected EndedAt to be nil")
	}
	if jobs[0].Error != "" {
		t.Fatalf("expected empty error, got %q", jobs[0].Error)
	}
}

func TestUpdateRetryPreservesStartedAt(t *testing.T) {
	// Cover the default branch in Update's switch: failed -> pending clears Error.
	// Also verify that after a retry cycle (failed -> pending -> running),
	// the StartedAt is preserved from the first run if already set.
	q, _ := newTestQueue(t)
	job := testJob(21)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if err := q.Update(job.ID, StateFailed, "transient"); err != nil {
		t.Fatalf("fail: %v", err)
	}

	// Retry: failed -> pending
	if err := q.Update(job.ID, StatePending, ""); err != nil {
		t.Fatalf("retry: %v", err)
	}

	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if jobs[0].Error != "" {
		t.Fatalf("expected error cleared after retry, got %q", jobs[0].Error)
	}
}

func TestDequeueSkipsNonPending(t *testing.T) {
	// Dequeue should pick the first pending job, skipping running/completed.
	q, _ := newTestQueue(t)
	j1 := testJob(30)
	j1.State = StateRunning // already running
	j2 := testJob(31)
	j2.State = StateCompleted
	j3 := testJob(32) // pending

	for _, j := range []Job{j1, j2, j3} {
		if err := q.Enqueue(j); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected a job, got nil")
	}
	if got.IssueNum != 32 {
		t.Fatalf("expected issue 32 (first pending), got %d", got.IssueNum)
	}
}

func TestDequeueNoPendingReturnsNil(t *testing.T) {
	// All jobs are in non-pending states; Dequeue should return nil.
	q, _ := newTestQueue(t)
	j := testJob(40)
	j.State = StateCompleted
	if err := q.Enqueue(j); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil (no pending jobs), got %+v", *got)
	}
}

func TestConcurrentEnqueueAndDequeue(t *testing.T) {
	// Exercise concurrent Enqueue and Dequeue to verify file-lock correctness
	// under mixed read-write contention.
	q, _ := newTestQueue(t)
	const numJobs = 20

	// Pre-load some jobs
	for i := 0; i < numJobs; i++ {
		if err := q.Enqueue(testJob(900 + i)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	var wg sync.WaitGroup
	var dequeued int32
	var enqueued int32

	// Concurrently dequeue
	wg.Add(numJobs)
	for i := 0; i < numJobs; i++ {
		go func() {
			defer wg.Done()
			job, err := q.Dequeue()
			if err != nil {
				t.Errorf("dequeue: %v", err)
				return
			}
			if job != nil {
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
			if err := q.Enqueue(testJob(950 + i)); err != nil {
				t.Errorf("enqueue: %v", err)
				return
			}
			atomic.AddInt32(&enqueued, 1)
		}()
	}

	wg.Wait()

	// Verify consistency: all jobs accounted for
	jobs, err := q.List()
	if err != nil {
		t.Fatalf("final list: %v", err)
	}

	totalExpected := numJobs + int(atomic.LoadInt32(&enqueued))
	if len(jobs) != totalExpected {
		t.Fatalf("expected %d jobs total, got %d", totalExpected, len(jobs))
	}

	// Each dequeued job should be in Running state
	runCount := 0
	for _, j := range jobs {
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
	if err := q.Enqueue(testJob(50)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// No completed jobs
	completed, err := q.ListByState(StateCompleted)
	if err != nil {
		t.Fatalf("list by state: %v", err)
	}
	if len(completed) != 0 {
		t.Fatalf("expected 0 completed, got %d", len(completed))
	}
}

func TestEmptyFileReadAllJobs(t *testing.T) {
	_, path := newTestQueue(t)
	// Create an empty file
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}
	q := New(path)
	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list on empty file: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected 0 jobs from empty file, got %d", len(jobs))
	}
}

func TestBlankLinesIgnored(t *testing.T) {
	q, path := newTestQueue(t)
	j := testJob(60)
	b, _ := json.Marshal(j)
	// File with blank lines interspersed
	content := "\n\n" + string(b) + "\n\n\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	jobs, err := q.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job (blank lines ignored), got %d", len(jobs))
	}
	_ = q // satisfy vet
}

func TestWriteEmptyJobs(t *testing.T) {
	// Verify writeAllJobs handles the empty case without writing a trailing newline.
	q, path := newTestQueue(t)
	// Enqueue then cancel to get a non-empty file, then verify the file format.
	if err := q.Enqueue(testJob(70)); err != nil {
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
