package observation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Event struct {
	ID             string         `json:"id"`
	RunID          string         `json:"runId"`
	Sequence       uint64         `json:"sequence"`
	Type           string         `json:"type"`
	Source         string         `json:"source"`
	Timestamp      time.Time      `json:"timestamp"`
	Payload        map[string]any `json:"payload,omitempty"`
	OperationID    string         `json:"operationId,omitempty"`
	IdempotencyKey string         `json:"idempotencyKey,omitempty"`
}

type Journal struct {
	path        string
	mu          sync.Mutex
	sequence    uint64
	idempotency map[string]Event
	ids         map[string]Event
}

func Open(path string) (*Journal, error) {
	events, err := read(path)
	if err != nil {
		return nil, err
	}
	n, seen, ids := index(events)
	return &Journal{path: path, sequence: n, idempotency: seen, ids: ids}, nil
}

func (j *Journal) Append(e Event) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(j.path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(j.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock journal: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	// Refresh under the file lock so a long-lived process cannot reuse a
	// sequence number after another process appended an event.
	existing, err := read(j.path)
	if err != nil {
		return err
	}
	j.sequence, j.idempotency, j.ids = index(existing)
	if e.ID != "" {
		if previous, ok := j.ids[e.ID]; ok {
			if previous.Type != e.Type || previous.RunID != e.RunID {
				return fmt.Errorf("event id %q already exists with different content", e.ID)
			}
			return nil
		}
	}
	if e.IdempotencyKey != "" {
		if previous, ok := j.idempotency[e.IdempotencyKey]; ok {
			if previous.Type != e.Type || previous.RunID != e.RunID {
				return fmt.Errorf("idempotency key %q already exists with different content", e.IdempotencyKey)
			}
			return nil
		}
	}
	j.sequence++
	e.Sequence = j.sequence
	if e.ID == "" {
		e.ID = fmt.Sprintf("OBS-%d", e.Sequence)
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if e.IdempotencyKey != "" {
		j.idempotency[e.IdempotencyKey] = e
	}
	j.ids[e.ID] = e
	return nil
}

// Replay returns valid events in order and reports malformed lines instead of
// silently treating a damaged journal as a valid empty history.
func (j *Journal) Replay(runID string) ([]Event, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	events, err := read(j.path)
	if err != nil {
		return nil, err
	}
	if runID == "" {
		return events, nil
	}
	out := make([]Event, 0)
	for _, event := range events {
		if event.RunID == runID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (j *Journal) Events() ([]Event, error) { return j.Replay("") }

func read(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	events := []Event{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	var last uint64
	ids := map[string]bool{}
	for scanner.Scan() {
		line++
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("journal line %d: %w", line, err)
		}
		if event.Sequence == 0 || event.Sequence <= last {
			return nil, fmt.Errorf("journal line %d: non-increasing sequence", line)
		}
		if event.ID == "" {
			return nil, fmt.Errorf("journal line %d: missing event id", line)
		}
		if ids[event.ID] {
			return nil, fmt.Errorf("journal line %d: duplicate event id %q", line, event.ID)
		}
		ids[event.ID] = true
		last = event.Sequence
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func index(events []Event) (uint64, map[string]Event, map[string]Event) {
	var sequence uint64
	idempotency := map[string]Event{}
	ids := map[string]Event{}
	for _, event := range events {
		if event.Sequence > sequence {
			sequence = event.Sequence
		}
		if event.IdempotencyKey != "" {
			idempotency[event.IdempotencyKey] = event
		}
		ids[event.ID] = event
	}
	return sequence, idempotency, ids
}
