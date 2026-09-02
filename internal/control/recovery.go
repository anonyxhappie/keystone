package control

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/runtime"
	"github.com/anonyxhappie/keystone/internal/state"
)

// RecoverSnapshot reconstructs the materialized cache from the event journal.
// It only persists the reconstructed cache after the complete journal has been
// validated and reduced.
func RecoverSnapshot(e *Engine, runID string) (state.Snapshot, error) {
	events, err := e.Journal.Replay("")
	if err != nil {
		return state.Snapshot{}, err
	}
	if runID == "" {
		for _, event := range events {
			if event.RunID != "" && event.RunID != "project" {
				runID = event.RunID
			}
		}
	}
	if runID == "" {
		return state.Snapshot{}, fmt.Errorf("journal contains no resumable run")
	}
	inputs := make([]runtime.Event, 0)
	var workOrderID string
	var evidenceIDs []string
	var findings []domain.Finding
	found := false
	for _, event := range events {
		if event.RunID != runID {
			continue
		}
		found = true
		if id, ok := event.Payload["workOrderId"].(string); ok && id != "" {
			workOrderID = id
		}
		switch event.Type {
		case "STATE_TRANSITION":
			from, _ := event.Payload["from"].(string)
			to, _ := event.Payload["to"].(string)
			reason, _ := event.Payload["reason"].(string)
			inputs = append(inputs, runtime.Event{Type: "transition", From: runtime.State(from), To: runtime.State(to), Reason: reason, At: event.Timestamp})
		case "DECISION":
			from, _ := event.Payload["from"].(string)
			decision, _ := event.Payload["decision"].(string)
			reason, _ := event.Payload["reason"].(string)
			inputs = append(inputs, runtime.Event{Type: "decision", From: runtime.State(from), Decision: runtime.Decision(decision), Reason: reason, At: event.Timestamp})
		case "EVIDENCE_RECORDED":
			if id, ok := event.Payload["evidenceId"].(string); ok {
				evidenceIDs = append(evidenceIDs, id)
			}
		case "FINDINGS_RECORDED":
			if raw, ok := event.Payload["findings"]; ok {
				encoded, marshalErr := json.Marshal(raw)
				if marshalErr != nil {
					return state.Snapshot{}, fmt.Errorf("marshal findings during recovery: %w", marshalErr)
				}
				if unmarshalErr := json.Unmarshal(encoded, &findings); unmarshalErr != nil {
					return state.Snapshot{}, fmt.Errorf("decode findings during recovery: %w", unmarshalErr)
				}
			}
		}
	}
	if !found {
		return state.Snapshot{}, fmt.Errorf("journal contains no events for run %s", runID)
	}
	machine, err := runtime.Reduce(inputs)
	if err != nil {
		return state.Snapshot{}, err
	}
	action := machine.NextAction("low", !machine.Terminal(), "reconstructed from event journal")
	snapshot := state.Snapshot{
		SchemaVersion: "2",
		Lifecycle:     string(machine.State),
		RunID:         runID,
		WorkOrderID:   workOrderID,
		Machine:       machine,
		NextAction:    &action,
		Findings:      findings,
		EvidenceIDs:   uniqueStrings(evidenceIDs),
	}
	if _, err := os.Stat(filepath.Join(e.Root, state.Dir, "control", "pause.json")); err == nil {
		snapshot.Paused = true
	}
	if err := e.Store.SaveSnapshot(snapshot); err != nil {
		return state.Snapshot{}, err
	}
	return snapshot, nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
