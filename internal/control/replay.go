package control

import (
	"encoding/json"
	"fmt"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/observation"
	"github.com/anonyxhappie/keystone/v2/internal/runtime"
)

type ReplayReport struct {
	RunID            string                  `json:"runId"`
	WorkOrderID      string                  `json:"workOrderId,omitempty"`
	Request          string                  `json:"request,omitempty"`
	State            string                  `json:"state"`
	HarnessID        string                  `json:"harnessId,omitempty"`
	HarnessSessionID string                  `json:"harnessSessionId,omitempty"`
	Machine          *runtime.Machine        `json:"machine"`
	NextAction       *domain.NextAction      `json:"nextAction,omitempty"`
	Findings         []domain.Finding        `json:"findings,omitempty"`
	PolicyDecisions  []domain.PolicyDecision `json:"policyDecisions,omitempty"`
	EvidenceIDs      []string                `json:"evidenceIds,omitempty"`
	Decisions        []string                `json:"decisions,omitempty"`
	Claims           []string                `json:"claims,omitempty"`
	Observations     []domain.Observation    `json:"observations,omitempty"`
	Gaps             []string                `json:"gaps,omitempty"`
	Events           []observation.Event     `json:"events"`
}

func Replay(runID string, events []observation.Event) (ReplayReport, error) {
	r := ReplayReport{RunID: runID, Events: events}
	inputs := []runtime.Event{}
	for _, event := range events {
		if event.RunID != "" && event.RunID != runID && event.RunID != "project" {
			continue
		}
		switch event.Type {
		case "REQUEST_ACCEPTED":
			if req, ok := event.Payload["request"].(string); ok && req != "" {
				r.Request = req
			}
			if woID, ok := event.Payload["workOrderId"].(string); ok && woID != "" {
				r.WorkOrderID = woID
			}
		case "HARNESS_METADATA":
			if hid, ok := event.Payload["harnessId"].(string); ok && hid != "" {
				r.HarnessID = hid
			}
			if sid, ok := event.Payload["sessionId"].(string); ok && sid != "" {
				r.HarnessSessionID = sid
			}
		case "RUN_STARTED":
			if woID, ok := event.Payload["workOrderId"].(string); ok && woID != "" && r.WorkOrderID == "" {
				r.WorkOrderID = woID
			}
			if sid, ok := event.Payload["sessionId"].(string); ok && sid != "" && r.HarnessSessionID == "" {
				r.HarnessSessionID = sid
			}
		case "HARNESS_SWITCHED":
			if to, ok := event.Payload["to"].(string); ok && to != "" {
				r.HarnessID = to
			}
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
		case "POLICY_DECISION":
			b, err := json.Marshal(event.Payload)
			if err == nil {
				var pd domain.PolicyDecision
				if json.Unmarshal(b, &pd) == nil {
					r.PolicyDecisions = append(r.PolicyDecisions, pd)
				}
			}
		case "FINDINGS_RECORDED":
			if raw, ok := event.Payload["findings"]; ok {
				b, err := json.Marshal(raw)
				if err == nil {
					var findings []domain.Finding
					if json.Unmarshal(b, &findings) == nil {
						r.Findings = findings
					}
				}
			}
		case "EVIDENCE_RECORDED":
			if id, ok := event.Payload["evidenceId"].(string); ok {
				r.EvidenceIDs = append(r.EvidenceIDs, id)
			}
		case "OBSERVATION":
			o, err := DecodeObservation(event.Payload)
			if err != nil {
				return r, fmt.Errorf("replay observation: %w", err)
			}
			r.Observations = append(r.Observations, o)
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
	r.State = string(m.State)
	reason := "reconstructed from event replay"
	if len(m.History) > 0 {
		reason = m.History[len(m.History)-1].Reason
	}
	act := m.NextAction("low", !m.Terminal(), reason)
	r.NextAction = &act
	return r, nil
}
