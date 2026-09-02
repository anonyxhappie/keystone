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
	StatusClaimed   Status = "CLAIMED"
	StatusVerified  Status = "VERIFIED"
	StatusRejected  Status = "REJECTED"
	StatusStale     Status = "STALE"
)

type VerificationStatus string

const (
	Claimed           VerificationStatus = "CLAIMED"
	PartiallyVerified VerificationStatus = "PARTIALLY_VERIFIED"
	Verified          VerificationStatus = "VERIFIED"
	Rejected          VerificationStatus = "REJECTED"
	Stale             VerificationStatus = "STALE"
)

type Project struct {
	SchemaVersion    string       `json:"schemaVersion" yaml:"schemaVersion"`
	ID               string       `json:"id" yaml:"id"`
	Root             string       `json:"root" yaml:"root"`
	Name             string       `json:"name" yaml:"name"`
	Capabilities     []Capability `json:"capabilities" yaml:"capabilities"`
	CreatedAt        time.Time    `json:"createdAt" yaml:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt" yaml:"updatedAt"`
	InstructionFiles []string     `json:"instructionFiles,omitempty" yaml:"instructionFiles,omitempty"`
	Topology         []string     `json:"topology,omitempty" yaml:"topology,omitempty"`
}

type Capability struct {
	ID     string   `json:"id,omitempty" yaml:"id,omitempty"`
	Kind   string   `json:"kind" yaml:"kind"`
	Name   string   `json:"name" yaml:"name"`
	Source string   `json:"source" yaml:"source"`
	Values []string `json:"values,omitempty" yaml:"values,omitempty"`
}

type Requirement struct {
	SchemaVersion string   `json:"schemaVersion,omitempty" yaml:"schemaVersion,omitempty"`
	ID            string   `json:"id" yaml:"id"`
	Title         string   `json:"title" yaml:"title"`
	Description   string   `json:"description" yaml:"description"`
	Acceptance    []string `json:"acceptance,omitempty" yaml:"acceptance,omitempty"`
	Status        Status   `json:"status" yaml:"status"`
	Source        string   `json:"source,omitempty" yaml:"source,omitempty"`
	Provenance    []string `json:"provenance,omitempty" yaml:"provenance,omitempty"`
	WorkOrderID   string   `json:"workOrderId,omitempty" yaml:"workOrderId,omitempty"`
	ChangedFiles  []string `json:"changedFiles,omitempty" yaml:"changedFiles,omitempty"`
	ValidationIDs []string `json:"validationIds,omitempty" yaml:"validationIds,omitempty"`
	EvidenceIDs   []string `json:"evidenceIds,omitempty" yaml:"evidenceIds,omitempty"`
}

type Decision struct {
	ID        string    `json:"id"`
	Summary   string    `json:"summary"`
	Rationale string    `json:"rationale,omitempty"`
	Source    string    `json:"source,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
type Assumption struct {
	ID         string  `json:"id"`
	Statement  string  `json:"statement"`
	Confidence float64 `json:"confidence"`
	Status     string  `json:"status"`
	Source     string  `json:"source,omitempty"`
}
type Constraint struct {
	ID        string `json:"id"`
	Statement string `json:"statement"`
	Type      string `json:"type"`
	Source    string `json:"source,omitempty"`
}

type WorkOrder struct {
	SchemaVersion string    `json:"schemaVersion,omitempty" yaml:"schemaVersion,omitempty"`
	ID            string    `json:"id" yaml:"id"`
	SourceRequest string    `json:"sourceRequest" yaml:"sourceRequest"`
	Objective     string    `json:"objective" yaml:"objective"`
	Requirements  []string  `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	Constraints   []string  `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Risk          Risk      `json:"risk" yaml:"risk"`
	Autonomy      string    `json:"autonomy" yaml:"autonomy"`
	Status        Status    `json:"status" yaml:"status"`
	CreatedAt     time.Time `json:"createdAt" yaml:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	RunID         string    `json:"runId,omitempty" yaml:"runId,omitempty"`
}

type Risk struct {
	Level     string   `json:"level" yaml:"level"`
	Factors   []string `json:"factors,omitempty" yaml:"factors,omitempty"`
	Score     int      `json:"score" yaml:"score"`
	Rationale string   `json:"rationale,omitempty" yaml:"rationale,omitempty"`
}

type WorkPacket struct {
	SchemaVersion      string       `json:"schemaVersion,omitempty" yaml:"schemaVersion,omitempty"`
	WorkOrderID        string       `json:"workOrderId" yaml:"workOrderId"`
	Objective          string       `json:"objective" yaml:"objective"`
	Requirements       []string     `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	Constraints        []string     `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Context            []ContextRef `json:"context,omitempty" yaml:"context,omitempty"`
	Validation         []string     `json:"validation,omitempty" yaml:"validation,omitempty"`
	CompletionCriteria []string     `json:"completionCriteria,omitempty" yaml:"completionCriteria,omitempty"`
}

type ContextRef struct {
	Type          string  `json:"type,omitempty" yaml:"type,omitempty"`
	Path          string  `json:"path" yaml:"path"`
	Reason        string  `json:"reason" yaml:"reason"`
	Source        string  `json:"source" yaml:"source"`
	Relevance     float64 `json:"relevance,omitempty" yaml:"relevance,omitempty"`
	TokenEstimate int     `json:"tokenEstimate,omitempty" yaml:"tokenEstimate,omitempty"`
}

type NextAction struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Reason           string   `json:"reason"`
	Target           string   `json:"target,omitempty"`
	Inputs           []string `json:"inputs,omitempty"`
	Risk             string   `json:"risk"`
	Allowed          bool     `json:"allowed"`
	RequiresApproval bool     `json:"requiresApproval"`
}

type Observation struct {
	SchemaVersion string         `json:"schemaVersion,omitempty" yaml:"schemaVersion,omitempty"`
	ID            string         `json:"id" yaml:"id"`
	RunID         string         `json:"runId" yaml:"runId"`
	Type          string         `json:"type" yaml:"type"`
	Summary       string         `json:"summary" yaml:"summary"`
	Payload       map[string]any `json:"payload,omitempty" yaml:"payload,omitempty"`
	Timestamp     time.Time      `json:"timestamp" yaml:"timestamp"`
	Provenance    []string       `json:"provenance,omitempty" yaml:"provenance,omitempty"`
}

type Evidence struct {
	ID             string             `json:"id" yaml:"id"`
	Type           string             `json:"type" yaml:"type"`
	Status         Status             `json:"status" yaml:"status"`
	WorkOrderID    string             `json:"workOrderId" yaml:"workOrderId"`
	Commit         string             `json:"commit,omitempty" yaml:"commit,omitempty"`
	InputsHash     string             `json:"inputsHash,omitempty" yaml:"inputsHash,omitempty"`
	Summary        string             `json:"summary" yaml:"summary"`
	Valid          bool               `json:"valid" yaml:"valid"`
	Verification   VerificationStatus `json:"verification" yaml:"verification"`
	ObservationIDs []string           `json:"observationIds,omitempty" yaml:"observationIds,omitempty"`
	ArtifactRefs   []string           `json:"artifactRefs,omitempty" yaml:"artifactRefs,omitempty"`
	CreatedAt      time.Time          `json:"createdAt" yaml:"createdAt"`
}

type Finding struct {
	ID                string   `json:"id" yaml:"id"`
	Type              string   `json:"type" yaml:"type"`
	Severity          string   `json:"severity" yaml:"severity"`
	Confidence        float64  `json:"confidence" yaml:"confidence"`
	EvidenceIDs       []string `json:"evidenceIds,omitempty" yaml:"evidenceIds,omitempty"`
	RecommendedAction string   `json:"recommendedAction,omitempty" yaml:"recommendedAction,omitempty"`
	Explanation       string   `json:"explanation,omitempty" yaml:"explanation,omitempty"`
	Provenance        []string `json:"provenance,omitempty" yaml:"provenance,omitempty"`
}

type PolicyDecision struct {
	Decision         string `json:"decision" yaml:"decision"`
	Reason           string `json:"reason" yaml:"reason"`
	Policy           string `json:"policy" yaml:"policy"`
	Allowed          bool   `json:"allowed" yaml:"allowed"`
	RequiresApproval bool   `json:"requiresApproval" yaml:"requiresApproval"`
}

type Checkpoint struct {
	ID              string      `json:"id" yaml:"id"`
	WorkOrderID     string      `json:"workOrderId" yaml:"workOrderId"`
	State           string      `json:"state" yaml:"state"`
	Completed       []string    `json:"completed,omitempty" yaml:"completed,omitempty"`
	Pending         []string    `json:"pending,omitempty" yaml:"pending,omitempty"`
	ChangedFiles    []string    `json:"changedFiles,omitempty" yaml:"changedFiles,omitempty"`
	LastCommit      string      `json:"lastCommit,omitempty" yaml:"lastCommit,omitempty"`
	ContextManifest string      `json:"contextManifest,omitempty" yaml:"contextManifest,omitempty"`
	Unresolved      []string    `json:"unresolvedQuestions,omitempty" yaml:"unresolvedQuestions,omitempty"`
	Blockers        []string    `json:"blockers,omitempty" yaml:"blockers,omitempty"`
	SchemaVersion   string      `json:"schemaVersion,omitempty" yaml:"schemaVersion,omitempty"`
	HarnessID       string      `json:"harnessId,omitempty" yaml:"harnessId,omitempty"`
	RunID           string      `json:"runId,omitempty" yaml:"runId,omitempty"`
	NextAction      *NextAction `json:"nextAction,omitempty" yaml:"nextAction,omitempty"`
	CreatedAt       time.Time   `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
}

type Learning struct {
	ID                string   `json:"id" yaml:"id"`
	Scope             string   `json:"scope" yaml:"scope"`
	Observation       string   `json:"observation" yaml:"observation"`
	BeforeEvidenceIDs []string `json:"beforeEvidenceIds,omitempty" yaml:"beforeEvidenceIds,omitempty"`
	EvidenceIDs       []string `json:"evidenceIds,omitempty" yaml:"evidenceIds,omitempty"`
	Confidence        float64  `json:"confidence" yaml:"confidence"`
	ProposedChange    string   `json:"proposedChange" yaml:"proposedChange"`
	Outcome           string   `json:"outcome,omitempty" yaml:"outcome,omitempty"`
	Rollback          string   `json:"rollback,omitempty" yaml:"rollback,omitempty"`
	Status            string   `json:"status" yaml:"status"`
	Version           int      `json:"version" yaml:"version"`
}

type Artifact struct {
	ID        string    `json:"id"`
	Path      string    `json:"path,omitempty"`
	Digest    string    `json:"digest,omitempty"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"createdAt"`
}
type Harness struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities,omitempty"`
	Version      string   `json:"version,omitempty"`
}

type HarnessSession struct {
	ID        string    `json:"id"`
	HarnessID string    `json:"harnessId"`
	RunID     string    `json:"runId"`
	Status    Status    `json:"status"`
	StartedAt time.Time `json:"startedAt"`
}
type HarnessRun struct {
	ID          string     `json:"id"`
	WorkOrderID string     `json:"workOrderId"`
	HarnessID   string     `json:"harnessId"`
	Status      Status     `json:"status"`
	Claim       string     `json:"claim,omitempty"`
	StartedAt   time.Time  `json:"startedAt"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
}
type Validation struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Tier       int    `json:"tier"`
	Passed     bool   `json:"passed"`
	EvidenceID string `json:"evidenceId,omitempty"`
}
type Policy struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Rules   map[string]string `json:"rules,omitempty"`
}
type Approval struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`
	RunID      string    `json:"runId,omitempty"`
	ApprovedBy string    `json:"approvedBy"`
	Reason     string    `json:"reason,omitempty"`
	At         time.Time `json:"at"`
}
type Release struct {
	ID                     string    `json:"id"`
	Version                string    `json:"version"`
	Status                 Status    `json:"status"`
	GitReference           string    `json:"gitReference,omitempty"`
	RequirementIDs         []string  `json:"requirementIds,omitempty"`
	EvidenceIDs            []string  `json:"evidenceIds,omitempty"`
	SecurityEvidenceIDs    []string  `json:"securityEvidenceIds,omitempty"`
	OperationalEvidenceIDs []string  `json:"operationalEvidenceIds,omitempty"`
	CreatedAt              time.Time `json:"createdAt"`
}
type Deployment struct {
	ID          string `json:"id"`
	Environment string `json:"environment"`
	Status      Status `json:"status"`
	ReleaseID   string `json:"releaseId,omitempty"`
}
type Incident struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Severity    string `json:"severity"`
	WorkOrderID string `json:"workOrderId,omitempty"`
}

type Environment struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Variables []string `json:"variables,omitempty"`
	Protected bool     `json:"protected"`
}
