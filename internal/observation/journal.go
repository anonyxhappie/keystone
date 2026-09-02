package observation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
}

func Open(path string) (*Journal, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Journal{path: path}, nil
		}
		return nil, err
	}
	defer f.Close()
	var n uint64
	seen := map[string]Event{}
	s := bufio.NewScanner(f)
	line := 0
	for s.Scan() {
		line++
		var e Event
		if err := json.Unmarshal(s.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("journal line %d: %w", line, err)
		}
		if e.Sequence > n {
			n = e.Sequence
		}
		if e.IdempotencyKey != "" {
			seen[e.IdempotencyKey] = e
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return &Journal{path: path, sequence: n, idempotency: seen}, nil
}

func (j *Journal) Append(e Event) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.idempotency == nil {
		j.idempotency = map[string]Event{}
	}
	if e.IdempotencyKey != "" {
		if previous, ok := j.idempotency[e.IdempotencyKey]; ok {
			e.ID = previous.ID
			e.Sequence = previous.Sequence
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
	if err := os.MkdirAll(filepath.Dir(j.path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(j.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(b, '\n')); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if e.IdempotencyKey != "" {
		j.idempotency[e.IdempotencyKey] = e
	}
	return nil
}

// Replay returns valid events in order and reports malformed lines instead of silently
// treating a damaged journal as a valid empty history.
func (j *Journal) Replay(runID string) ([]Event, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	f, err := os.Open(j.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []Event
	s := bufio.NewScanner(f)
	line := 0
	last := uint64(0)
	for s.Scan() {
		line++
		var e Event
		if err := json.Unmarshal(s.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("journal line %d: %w", line, err)
		}
		if e.Sequence == 0 || e.Sequence <= last {
			return nil, fmt.Errorf("journal line %d: non-increasing sequence", line)
		}
		last = e.Sequence
		if runID == "" || e.RunID == runID {
			out = append(out, e)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (j *Journal) Events() ([]Event, error) { return j.Replay("") }
