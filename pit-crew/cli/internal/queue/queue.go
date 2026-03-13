package queue

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

type JobState string

const (
	StatePending   JobState = "pending"
	StateRunning   JobState = "running"
	StateCompleted JobState = "completed"
	StateFailed    JobState = "failed"
	StateCancelled JobState = "cancelled"
)

// validTransitions defines the allowed state transitions. Each key is a current
// state and the value is the set of states it may transition to.
var validTransitions = map[JobState]map[JobState]bool{
	StatePending: {
		StateRunning:   true,
		StateCancelled: true,
	},
	StateRunning: {
		StateCompleted: true,
		StateFailed:    true,
		StateCancelled: true,
	},
	StateFailed: {
		StatePending: true, // allow retry
	},
	// Terminal states: no transitions out of completed or cancelled.
	StateCompleted: {},
	StateCancelled: {},
}

// ErrInvalidTransition is returned when a state transition is not allowed.
var ErrInvalidTransition = errors.New("invalid state transition")

type Job struct {
	ID        string     `json:"id"`
	IssueURL  string     `json:"issue_url"`
	IssueNum  int        `json:"issue_num"`
	Skill     string     `json:"skill"`
	State     JobState   `json:"state"`
	CreatedAt time.Time  `json:"created_at"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Error     string     `json:"error,omitempty"`
}

type Queue struct {
	path     string
	lockPath string
}

func New(path string) *Queue {
	return &Queue{path: path, lockPath: path + ".lock"}
}

func (q *Queue) Enqueue(job Job) error {
	return q.withLock(func() error {
		jobs, err := q.readAllJobs()
		if err != nil {
			return err
		}
		jobs = append(jobs, job)
		return q.writeAllJobs(jobs)
	})
}

func (q *Queue) Dequeue() (*Job, error) {
	var out *Job
	err := q.withLock(func() error {
		jobs, err := q.readAllJobs()
		if err != nil {
			return err
		}

		for i := range jobs {
			if jobs[i].State != StatePending {
				continue
			}
			now := time.Now().UTC()
			jobs[i].State = StateRunning
			jobs[i].StartedAt = &now
			jobs[i].Error = ""

			job := jobs[i]
			out = &job
			return q.writeAllJobs(jobs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (q *Queue) Update(id string, state JobState, errMsg string) error {
	return q.withLock(func() error {
		jobs, err := q.readAllJobs()
		if err != nil {
			return err
		}

		for i := range jobs {
			if jobs[i].ID != id {
				continue
			}

			// Validate state transition.
			allowed, knownState := validTransitions[jobs[i].State]
			if !knownState {
				return fmt.Errorf("%w: unknown current state %s for job %s", ErrInvalidTransition, jobs[i].State, id)
			}
			if !allowed[state] {
				return fmt.Errorf("%w: cannot move job %s from %s to %s", ErrInvalidTransition, id, jobs[i].State, state)
			}

			now := time.Now().UTC()
			jobs[i].State = state
			switch state {
			case StateRunning:
				if jobs[i].StartedAt == nil {
					jobs[i].StartedAt = &now
				}
				jobs[i].EndedAt = nil
				jobs[i].Error = ""
			case StateFailed:
				jobs[i].EndedAt = &now
				jobs[i].Error = errMsg
			case StateCompleted, StateCancelled:
				jobs[i].EndedAt = &now
				jobs[i].Error = ""
			default:
				jobs[i].Error = ""
			}
			return q.writeAllJobs(jobs)
		}

		return fmt.Errorf("job %s not found", id)
	})
}

func (q *Queue) List() ([]Job, error) {
	var jobs []Job
	err := q.withRLock(func() error {
		var readErr error
		jobs, readErr = q.readAllJobs()
		return readErr
	})
	return jobs, err
}

func (q *Queue) ListByState(state JobState) ([]Job, error) {
	jobs, err := q.List()
	if err != nil {
		return nil, err
	}

	filtered := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		if job.State == state {
			filtered = append(filtered, job)
		}
	}
	return filtered, nil
}

func (q *Queue) Cancel(id string) error {
	return q.withLock(func() error {
		jobs, err := q.readAllJobs()
		if err != nil {
			return err
		}

		for i := range jobs {
			if jobs[i].ID != id {
				continue
			}
			if jobs[i].State != StatePending {
				return fmt.Errorf("cannot cancel job %s in state %s", id, jobs[i].State)
			}
			now := time.Now().UTC()
			jobs[i].State = StateCancelled
			jobs[i].EndedAt = &now
			jobs[i].Error = ""
			return q.writeAllJobs(jobs)
		}

		return fmt.Errorf("job %s not found", id)
	})
}

func (q *Queue) HasIssue(issueNum int) bool {
	jobs, err := q.List()
	if err != nil {
		return false
	}

	for _, job := range jobs {
		if job.IssueNum == issueNum && job.State != StateCancelled {
			return true
		}
	}
	return false
}

func (q *Queue) withLock(fn func() error) error {
	lock := flock.New(q.lockPath)
	if err := lock.Lock(); err != nil {
		return err
	}
	defer func() {
		if unlockErr := lock.Unlock(); unlockErr != nil {
			log.Printf("warn: failed to unlock queue: %v", unlockErr)
		}
	}()
	return fn()
}

func (q *Queue) withRLock(fn func() error) error {
	lock := flock.New(q.lockPath)
	if err := lock.RLock(); err != nil {
		return err
	}
	defer func() {
		if unlockErr := lock.Unlock(); unlockErr != nil {
			log.Printf("warn: failed to unlock queue: %v", unlockErr)
		}
	}()
	return fn()
}

func (q *Queue) readAllJobs() ([]Job, error) {
	f, err := os.Open(q.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Job{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var (
		jobs     = make([]Job, 0)
		lineNum  int
		skipped  int
	)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var job Job
		if err := json.Unmarshal([]byte(line), &job); err != nil {
			skipped++
			log.Printf("warn: skipping malformed queue entry at line %d: %v (content: %s)", lineNum, err, line)
			continue
		}
		jobs = append(jobs, job)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if skipped > 0 {
		return jobs, fmt.Errorf("%d malformed queue entries skipped", skipped)
	}

	return jobs, nil
}

func (q *Queue) writeAllJobs(jobs []Job) error {
	lines := make([]string, 0, len(jobs))
	for _, job := range jobs {
		b, err := json.Marshal(job)
		if err != nil {
			return err
		}
		lines = append(lines, string(b))
	}

	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}

	return os.WriteFile(q.path, []byte(content), 0o644)
}
