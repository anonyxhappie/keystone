package control

import (
	"fmt"

	"github.com/anonyxhappie/keystone/internal/observation"
	"github.com/anonyxhappie/keystone/internal/runtime"
)

type ReplayReport struct {
	RunID       string              `json:"runId"`
	Events      []observation.Event `json:"events"`
	Machine     *runtime.Machine    `json:"machine"`
	EvidenceIDs []string            `json:"evidenceIds,omitempty"`
	Decisions   []string            `json:"decisions,omitempty"`
	Claims      []string            `json:"claims,omitempty"`
	Gaps        []string            `json:"gaps,omitempty"`
}

func Replay(runID string, events []observation.Event) (ReplayReport, error) {
	r := ReplayReport{RunID: runID, Events: events}
	inputs := []runtime.Event{}
	for _, event := range events {
		switch event.Type {
		case "STATE_TRANSITION":
			from, _ := event.Payload["from"].(string)
			to, _ := event.Payload["to"].(string)
			reason, _ := event.Payload["reason"].(string)
			inputs = append(inputs, runtime.Event{Type: "transition", From: runtime.State(from), To: runtime.State(to), Reason: reason, At: event.Timestamp})
		case "DECISION":
			from, _ := event.Payload["from"].(string)
			d, _ := event.Payload["decision"].(string)
			reason, _ := event.Payload["reason"].(string)
			r.Decisions = append(r.Decisions, d)
			inputs = append(inputs, runtime.Event{Type: "decision", From: runtime.State(from), Decision: runtime.Decision(d), Reason: reason, At: event.Timestamp})
		case "EVIDENCE_RECORDED":
			if id, ok := event.Payload["evidenceId"].(string); ok {
				r.EvidenceIDs = append(r.EvidenceIDs, id)
			}
		case "OBSERVATION":
			o, err := DecodeObservation(event.Payload)
			if err != nil {
				return r, fmt.Errorf("replay observation: %w", err)
			}
			if o.Type == "COMPLETION_CLAIM" {
				r.Claims = append(r.Claims, o.Summary)
			}
		case "OBSERVATION_GAP":
			if msg, ok := event.Payload["error"].(string); ok {
				r.Gaps = append(r.Gaps, msg)
			}
		}
	}
	m, err := runtime.Reduce(inputs)
	if err != nil {
		return r, fmt.Errorf("replay run %s: %w", runID, err)
	}
	r.Machine = m
	return r, nil
}
