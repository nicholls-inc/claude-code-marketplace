package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

	if err := q.Update(job.ID, StateFailed, "boom"); err != nil {
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
	if err := q.Update(job.ID, StateCompleted, ""); err != nil {
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
	if err != nil {
		t.Fatalf("list corrupted file: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 valid jobs, got %d", len(jobs))
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
