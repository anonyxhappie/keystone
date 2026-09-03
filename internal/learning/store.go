package learning

import (
	"fmt"
	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/state"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func SaveCandidate(s state.Store, l domain.Learning) error {
	if l.ID == "" {
		l.ID = fmt.Sprintf("L-%d", time.Now().UnixNano())
	}
	if l.Status == "" {
		l.Status = "CANDIDATE"
	}
	if l.Version == 0 {
		l.Version = 1
	}
	return s.Write("learning/"+l.ID+".json", l)
}

func Transition(s state.Store, l domain.Learning, status string, outcome string) (domain.Learning, error) {
	if status != "OBSERVED" && status != "CANDIDATE" && status != "EVALUATED" && status != "ACTIVE" && status != "REJECTED" && status != "SUPERSEDED" {
		return l, fmt.Errorf("invalid learning status %q", status)
	}
	if l.Status == "ACTIVE" && status == "CANDIDATE" {
		return l, fmt.Errorf("active learning cannot move backwards")
	}
	if outcome != "" {
		l.Observation = l.Observation + "; outcome: " + outcome
		l.Outcome = outcome
	}
	l.Status = status
	if l.Version == 0 {
		l.Version = 1
	}
	if status == "ACTIVE" {
		l.Version++
	}
	if err := s.Write("learning/"+l.ID+".json", l); err != nil {
		return l, err
	}
	return l, nil
}

func Activate(s state.Store, l domain.Learning, outcome string) (domain.Learning, error) {
	return Transition(s, l, "ACTIVE", outcome)
}
func Reject(s state.Store, l domain.Learning, reason string) (domain.Learning, error) {
	return Transition(s, l, "REJECTED", reason)
}
func Supersede(s state.Store, l domain.Learning, reason string) (domain.Learning, error) {
	return Transition(s, l, "SUPERSEDED", reason)
}

func Active(s state.Store, scope string) []domain.Learning {
	entries, err := os.ReadDir(filepath.Join(s.Root, state.Dir, "learning"))
	if err != nil {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	active := []domain.Learning{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var l domain.Learning
		if s.Read("learning/"+entry.Name(), &l) == nil && l.Status == "ACTIVE" && (scope == "" || l.Scope == scope) {
			active = append(active, l)
		}
	}
	return active
}

// CaptureLesson records a learned rule or insight durably in .keystone/learning/
func CaptureLesson(s state.Store, summary, details string) (domain.Learning, error) {
	now := time.Now().UTC()
	l := domain.Learning{
		ID:             fmt.Sprintf("L-%d", now.UnixNano()),
		Scope:          "project",
		Status:         "ACTIVE",
		Version:        1,
		Observation:    summary,
		ProposedChange: details,
		Outcome:        "captured via harness /learn workflow",
		Confidence:     0.95,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.Write("learning/"+l.ID+".json", l); err != nil {
		return l, err
	}
	return l, nil
}
