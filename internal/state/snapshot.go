package state

import (
	"fmt"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/runtime"
)

// Snapshot is a materialized cache. The event journal remains the durable source
// of truth and can reconstruct the machine after this file is lost or corrupted.
type Snapshot struct {
	SchemaVersion    string             `json:"schemaVersion"`
	Lifecycle        string             `json:"lifecycle"`
	RunID            string             `json:"runId,omitempty"`
	WorkOrderID      string             `json:"workOrderId,omitempty"`
	HarnessID        string             `json:"harnessId,omitempty"`
	HarnessSessionID string             `json:"harnessSessionId,omitempty"`
	Machine          *runtime.Machine   `json:"machine,omitempty"`
	NextAction       *domain.NextAction `json:"nextAction,omitempty"`
	Findings         []domain.Finding   `json:"findings,omitempty"`
	EvidenceIDs      []string           `json:"evidenceIds,omitempty"`
	ChangedFiles     []string           `json:"changedFiles,omitempty"`
	ContextManifest  string             `json:"contextManifest,omitempty"`
	CheckpointID     string             `json:"checkpointId,omitempty"`
	Paused           bool               `json:"paused,omitempty"`
	UpdatedAt        time.Time          `json:"updatedAt"`
	LastError        string             `json:"lastError,omitempty"`
}

func (s Store) SaveSnapshot(snapshot Snapshot) error {
	if snapshot.SchemaVersion == "" {
		snapshot.SchemaVersion = "2"
	}
	if snapshot.UpdatedAt.IsZero() {
		snapshot.UpdatedAt = time.Now().UTC()
	}
	return s.Write("state.json", snapshot)
}

func (s Store) LoadSnapshot() (Snapshot, error) {
	var snapshot Snapshot
	if err := s.Read("state.json", &snapshot); err != nil {
		return snapshot, err
	}
	if snapshot.SchemaVersion == "" {
		return snapshot, fmt.Errorf("state snapshot has no schema version")
	}
	if snapshot.Machine != nil && snapshot.Lifecycle != "" && snapshot.Lifecycle != string(snapshot.Machine.State) {
		return snapshot, fmt.Errorf("state snapshot lifecycle %q does not match machine state %q", snapshot.Lifecycle, snapshot.Machine.State)
	}
	return snapshot, nil
}

func (s Store) Initialized() bool {
	_, err := s.LoadSnapshot()
	return err == nil
}

func (s Snapshot) NextActionOr(m *runtime.Machine) domain.NextAction {
	if s.NextAction != nil {
		return *s.NextAction
	}
	if m == nil {
		return domain.NextAction{Type: "ASK", Reason: "state snapshot has no machine", Risk: "high", RequiresApproval: true}
	}
	return m.NextAction("low", true, "reconstruct next action from durable state")
}
