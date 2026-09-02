package domain

import "time"

type Status string

const (
	StatusPlanned   Status = "PLANNED"
	StatusRunning   Status = "RUNNING"
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
	StatusBlocked   Status = "BLOCKED"
	StatusUnknown   Status = "UNKNOWN"
)

type Project struct {
	SchemaVersion string       `json:"schemaVersion" yaml:"schemaVersion"`
	ID            string       `json:"id" yaml:"id"`
	Root          string       `json:"root" yaml:"root"`
	Name          string       `json:"name" yaml:"name"`
	Capabilities  []Capability `json:"capabilities" yaml:"capabilities"`
	CreatedAt     time.Time    `json:"createdAt" yaml:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt" yaml:"updatedAt"`
}

type Capability struct {
	Kind   string   `json:"kind" yaml:"kind"`
	Name   string   `json:"name" yaml:"name"`
	Source string   `json:"source" yaml:"source"`
	Values []string `json:"values,omitempty" yaml:"values,omitempty"`
}

type Requirement struct {
	ID          string   `json:"id" yaml:"id"`
	Title       string   `json:"title" yaml:"title"`
	Description string   `json:"description" yaml:"description"`
	Acceptance  []string `json:"acceptance,omitempty" yaml:"acceptance,omitempty"`
	Status      Status   `json:"status" yaml:"status"`
}

type WorkOrder struct {
	ID             string       `json:"id" yaml:"id"`
	SourceRequest  string       `json:"sourceRequest" yaml:"sourceRequest"`
	Objective      string       `json:"objective" yaml:"objective"`
	Requirements   []string     `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	Constraints    []string     `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Risk           Risk         `json:"risk" yaml:"risk"`
	Autonomy       string       `json:"autonomy" yaml:"autonomy"`
	Status         Status       `json:"status" yaml:"status"`
	CreatedAt      time.Time    `json:"createdAt" yaml:"createdAt"`
}

type Risk struct {
	Level     string   `json:"level" yaml:"level"`
	Factors   []string `json:"factors,omitempty" yaml:"factors,omitempty"`
	Score     int      `json:"score" yaml:"score"`
	Rationale string   `json:"rationale,omitempty" yaml:"rationale,omitempty"`
}

type WorkPacket struct {
	WorkOrderID       string       `json:"workOrderId" yaml:"workOrderId"`
	Objective         string       `json:"objective" yaml:"objective"`
	Requirements      []string     `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	Constraints       []string     `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Context           []ContextRef `json:"context,omitempty" yaml:"context,omitempty"`
	Validation        []string     `json:"validation,omitempty" yaml:"validation,omitempty"`
	CompletionCriteria []string    `json:"completionCriteria,omitempty" yaml:"completionCriteria,omitempty"`
}

type ContextRef struct {
	Path   string `json:"path" yaml:"path"`
	Reason string `json:"reason" yaml:"reason"`
	Source string `json:"source" yaml:"source"`
}

type Observation struct {
	ID        string                 `json:"id" yaml:"id"`
	RunID     string                 `json:"runId" yaml:"runId"`
	Type      string                 `json:"type" yaml:"type"`
	Summary   string                 `json:"summary" yaml:"summary"`
	Payload   map[string]any         `json:"payload,omitempty" yaml:"payload,omitempty"`
	Timestamp time.Time              `json:"timestamp" yaml:"timestamp"`
}

type Evidence struct {
	ID         string    `json:"id" yaml:"id"`
	Type       string    `json:"type" yaml:"type"`
	Status     Status    `json:"status" yaml:"status"`
	WorkOrderID string   `json:"workOrderId" yaml:"workOrderId"`
	Commit     string    `json:"commit,omitempty" yaml:"commit,omitempty"`
	InputsHash string    `json:"inputsHash,omitempty" yaml:"inputsHash,omitempty"`
	Summary    string    `json:"summary" yaml:"summary"`
	Valid      bool      `json:"valid" yaml:"valid"`
	CreatedAt  time.Time `json:"createdAt" yaml:"createdAt"`
}

type Finding struct {
	ID              string   `json:"id" yaml:"id"`
	Type            string   `json:"type" yaml:"type"`
	Severity        string   `json:"severity" yaml:"severity"`
	Confidence      float64  `json:"confidence" yaml:"confidence"`
	EvidenceIDs     []string `json:"evidenceIds,omitempty" yaml:"evidenceIds,omitempty"`
	RecommendedAction string `json:"recommendedAction,omitempty" yaml:"recommendedAction,omitempty"`
}

type PolicyDecision struct {
	Decision string   `json:"decision" yaml:"decision"`
	Reason   string   `json:"reason" yaml:"reason"`
	Policy   string   `json:"policy" yaml:"policy"`
	Allowed  bool     `json:"allowed" yaml:"allowed"`
}

type Checkpoint struct {
	ID               string   `json:"id" yaml:"id"`
	WorkOrderID      string   `json:"workOrderId" yaml:"workOrderId"`
	State            string   `json:"state" yaml:"state"`
	Completed        []string `json:"completed,omitempty" yaml:"completed,omitempty"`
	Pending          []string `json:"pending,omitempty" yaml:"pending,omitempty"`
	ChangedFiles     []string `json:"changedFiles,omitempty" yaml:"changedFiles,omitempty"`
	LastCommit       string   `json:"lastCommit,omitempty" yaml:"lastCommit,omitempty"`
	ContextManifest  string   `json:"contextManifest,omitempty" yaml:"contextManifest,omitempty"`
	Unresolved       []string `json:"unresolvedQuestions,omitempty" yaml:"unresolvedQuestions,omitempty"`
	Blockers         []string `json:"blockers,omitempty" yaml:"blockers,omitempty"`
}

type Learning struct {
	ID             string   `json:"id" yaml:"id"`
	Scope          string   `json:"scope" yaml:"scope"`
	Observation    string   `json:"observation" yaml:"observation"`
	EvidenceIDs    []string `json:"evidenceIds,omitempty" yaml:"evidenceIds,omitempty"`
	Confidence     float64  `json:"confidence" yaml:"confidence"`
	ProposedChange string   `json:"proposedChange" yaml:"proposedChange"`
	Status         string   `json:"status" yaml:"status"`
	Version        int      `json:"version" yaml:"version"`
}
