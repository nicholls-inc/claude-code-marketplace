package queue

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

type VesselState string

const (
	StatePending   VesselState = "pending"
	StateRunning   VesselState = "running"
	StateCompleted VesselState = "completed"
	StateFailed    VesselState = "failed"
	StateCancelled VesselState = "cancelled"
)

// validTransitions defines the allowed state transitions. Each key is a current
// state and the value is the set of states it may transition to.
var validTransitions = map[VesselState]map[VesselState]bool{
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

type Vessel struct {
	ID        string            `json:"id"`
	Source    string            `json:"source"`
	Ref       string            `json:"ref,omitempty"`
	Skill     string            `json:"skill,omitempty"`
	Prompt    string            `json:"prompt,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
	State     VesselState       `json:"state"`
	CreatedAt time.Time         `json:"created_at"`
	StartedAt *time.Time        `json:"started_at,omitempty"`
	EndedAt   *time.Time        `json:"ended_at,omitempty"`
	Error     string            `json:"error,omitempty"`
}

type Queue struct {
	path     string
	lockPath string
}

func New(path string) *Queue {
	return &Queue{path: path, lockPath: path + ".lock"}
}

func (q *Queue) Enqueue(vessel Vessel) error {
	return q.withLock(func() error {
		vessels, err := q.readAllVessels()
		if err != nil {
			return err
		}
		vessels = append(vessels, vessel)
		return q.writeAllVessels(vessels)
	})
}

func (q *Queue) Dequeue() (*Vessel, error) {
	var out *Vessel
	err := q.withLock(func() error {
		vessels, err := q.readAllVessels()
		if err != nil {
			return err
		}

		for i := range vessels {
			if vessels[i].State != StatePending {
				continue
			}
			now := time.Now().UTC()
			vessels[i].State = StateRunning
			vessels[i].StartedAt = &now
			vessels[i].Error = ""

			vessel := vessels[i]
			out = &vessel
			return q.writeAllVessels(vessels)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (q *Queue) Update(id string, state VesselState, errMsg string) error {
	return q.withLock(func() error {
		vessels, err := q.readAllVessels()
		if err != nil {
			return err
		}

		for i := range vessels {
			if vessels[i].ID != id {
				continue
			}

			// Validate state transition.
			allowed, knownState := validTransitions[vessels[i].State]
			if !knownState {
				return fmt.Errorf("%w: unknown current state %s for vessel %s", ErrInvalidTransition, vessels[i].State, id)
			}
			if !allowed[state] {
				return fmt.Errorf("%w: cannot move vessel %s from %s to %s", ErrInvalidTransition, id, vessels[i].State, state)
			}

			now := time.Now().UTC()
			vessels[i].State = state
			switch state {
			case StateRunning:
				if vessels[i].StartedAt == nil {
					vessels[i].StartedAt = &now
				}
				vessels[i].EndedAt = nil
				vessels[i].Error = ""
			case StateFailed:
				vessels[i].EndedAt = &now
				vessels[i].Error = errMsg
			case StateCompleted, StateCancelled:
				vessels[i].EndedAt = &now
				vessels[i].Error = ""
			default:
				vessels[i].Error = ""
			}
			return q.writeAllVessels(vessels)
		}

		return fmt.Errorf("vessel %s not found", id)
	})
}

func (q *Queue) List() ([]Vessel, error) {
	var vessels []Vessel
	err := q.withRLock(func() error {
		var readErr error
		vessels, readErr = q.readAllVessels()
		return readErr
	})
	return vessels, err
}

func (q *Queue) ListByState(state VesselState) ([]Vessel, error) {
	vessels, err := q.List()
	if err != nil {
		return nil, err
	}

	filtered := make([]Vessel, 0, len(vessels))
	for _, vessel := range vessels {
		if vessel.State == state {
			filtered = append(filtered, vessel)
		}
	}
	return filtered, nil
}

func (q *Queue) Cancel(id string) error {
	return q.withLock(func() error {
		vessels, err := q.readAllVessels()
		if err != nil {
			return err
		}

		for i := range vessels {
			if vessels[i].ID != id {
				continue
			}
			if vessels[i].State != StatePending {
				return fmt.Errorf("cannot cancel vessel %s in state %s", id, vessels[i].State)
			}
			now := time.Now().UTC()
			vessels[i].State = StateCancelled
			vessels[i].EndedAt = &now
			vessels[i].Error = ""
			return q.writeAllVessels(vessels)
		}

		return fmt.Errorf("vessel %s not found", id)
	})
}

func (q *Queue) HasRef(ref string) bool {
	vessels, err := q.List()
	if err != nil {
		return false
	}

	for _, vessel := range vessels {
		if vessel.Ref == ref && vessel.State != StateCancelled {
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

func (q *Queue) readAllVessels() ([]Vessel, error) {
	f, err := os.Open(q.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Vessel{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var (
		vessels     = make([]Vessel, 0)
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

		var vessel Vessel
		if err := json.Unmarshal([]byte(line), &vessel); err != nil {
			skipped++
			log.Printf("warn: skipping malformed queue entry at line %d: %v (content: %s)", lineNum, err, line)
			continue
		}
		migrateLegacyVessel(&vessel, []byte(line))
		vessels = append(vessels, vessel)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if skipped > 0 {
		return vessels, fmt.Errorf("%d malformed queue entries skipped", skipped)
	}

	return vessels, nil
}

// migrateLegacyVessel populates the new generic fields from legacy
// issue_url/issue_num JSON fields when reading old queue entries.
func migrateLegacyVessel(v *Vessel, raw []byte) {
	if v.Source != "" {
		return // already migrated
	}
	var legacy struct {
		IssueURL string `json:"issue_url"`
		IssueNum int    `json:"issue_num"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return
	}
	if legacy.IssueURL != "" {
		v.Source = "github-issue"
		v.Ref = legacy.IssueURL
		if v.Meta == nil {
			v.Meta = make(map[string]string)
		}
		if legacy.IssueNum != 0 {
			v.Meta["issue_num"] = strconv.Itoa(legacy.IssueNum)
		}
	}
}

func (q *Queue) writeAllVessels(vessels []Vessel) error {
	lines := make([]string, 0, len(vessels))
	for _, vessel := range vessels {
		b, err := json.Marshal(vessel)
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
