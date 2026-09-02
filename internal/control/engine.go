package control

import (
	stdcontext "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/artifact"
	"github.com/anonyxhappie/keystone/v2/internal/checkpoint"
	"github.com/anonyxhappie/keystone/v2/internal/context"
	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/evidence"
	"github.com/anonyxhappie/keystone/v2/internal/git"
	"github.com/anonyxhappie/keystone/v2/internal/harness"
	"github.com/anonyxhappie/keystone/v2/internal/learning"
	"github.com/anonyxhappie/keystone/v2/internal/observation"
	"github.com/anonyxhappie/keystone/v2/internal/policy"
	"github.com/anonyxhappie/keystone/v2/internal/runtime"
	"github.com/anonyxhappie/keystone/v2/internal/state"
	"github.com/anonyxhappie/keystone/v2/internal/supervisor"
	"github.com/anonyxhappie/keystone/v2/internal/validation"
	"github.com/anonyxhappie/keystone/v2/internal/work"
)

type Limits struct {
	MaxAttempts      int
	MaxObservations  int
	MaxWallTime      time.Duration
	MaxContextTokens int
	MaxToolCalls     int
}

func DefaultLimits() Limits {
	return Limits{MaxAttempts: 2, MaxObservations: 256, MaxWallTime: 10 * time.Minute, MaxContextTokens: 20000, MaxToolCalls: 200}
}

type Engine struct {
	Root           string
	Store          state.Store
	Journal        *observation.Journal
	Limits         Limits
	AdapterFactory func() harness.Adapter
	resume         *resumeInput
}
type resumeInput struct {
	Order    domain.WorkOrder
	Snapshot state.Snapshot
}
type Report struct {
	RunID           string              `json:"runId"`
	WorkOrderID     string              `json:"workOrderId"`
	State           runtime.State       `json:"state"`
	NextAction      domain.NextAction   `json:"nextAction"`
	Findings        []domain.Finding    `json:"findings,omitempty"`
	EvidenceIDs     []string            `json:"evidenceIds,omitempty"`
	ChangedFiles    []string            `json:"changedFiles,omitempty"`
	ContextManifest string              `json:"contextManifest,omitempty"`
	Validations     []validation.Result `json:"validations,omitempty"`
	Attempts        int                 `json:"attempts"`
	Error           string              `json:"error,omitempty"`
}

var ErrPaused = errors.New("run paused by explicit control request")

func Open(root string) (*Engine, error) {
	s := state.New(root)
	if !s.Initialized() {
		var project domain.Project
		if err := s.Read("project.json", &project); err != nil {
			return nil, fmt.Errorf("Keystone is not initialized: %w", err)
		}
	}
	j, err := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	if err != nil {
		return nil, err
	}
	return &Engine{Root: root, Store: s, Journal: j, Limits: DefaultLimits()}, nil
}

func (e *Engine) Run(ctx stdcontext.Context, request string, adapter harness.Adapter) (Report, error) {
	if strings.TrimSpace(request) == "" {
		return Report{}, fmt.Errorf("request cannot be empty")
	}
	if e.Limits.MaxAttempts <= 0 {
		e.Limits = DefaultLimits()
	}
	ctx, cancel := stdcontext.WithTimeout(ctx, e.Limits.MaxWallTime)
	defer cancel()
	resuming := e.resume != nil
	var order domain.WorkOrder
	var req domain.Requirement
	var reqID string
	var runID string
	var m *runtime.Machine
	var report Report
	if resuming {
		order = e.resume.Order
		if len(order.Requirements) > 0 {
			reqID = order.Requirements[0]
			if err := e.Store.Read("requirements/"+reqID+".json", &req); err != nil {
				return Report{}, fmt.Errorf("load requirement %s: %w", reqID, err)
			}
		}
		runID = e.resume.Snapshot.RunID
		if runID == "" {
			runID = order.RunID
		}
		m = e.resume.Snapshot.Machine
		if m == nil {
			m = runtime.New()
		}
		report = Report{RunID: runID, WorkOrderID: order.ID, State: m.State, NextAction: e.resume.Snapshot.NextActionOr(m), Findings: e.resume.Snapshot.Findings, EvidenceIDs: e.resume.Snapshot.EvidenceIDs, ChangedFiles: e.resume.Snapshot.ChangedFiles, ContextManifest: e.resume.Snapshot.ContextManifest}
		if err := e.event(runID, "RESUME_REQUESTED", map[string]any{"workOrderId": order.ID, "state": m.State}); err != nil {
			return report, err
		}
		if m.State == runtime.Blocked {
			if !hasApproval(e.Store, runID) {
				if err := e.transition(m, runtime.Ask, "blocked work requires explicit review or approval", &report); err != nil {
					return report, err
				}
				return report, nil
			}
			if err := e.transition(m, runtime.Ask, "explicit approval permits recovery", &report); err != nil {
				return report, err
			}
			if err := e.transition(m, runtime.Approve, "approval provenance recorded", &report); err != nil {
				return report, err
			}
			if err := e.transition(m, runtime.Dispatch, "resume approved work", &report); err != nil {
				return report, err
			}
		}
		if m.State == runtime.Ask {
			if !hasApproval(e.Store, runID) {
				return report, nil
			}
			if err := e.transition(m, runtime.Approve, "approval provenance recorded", &report); err != nil {
				return report, err
			}
			if err := e.transition(m, runtime.Dispatch, "resume approved work", &report); err != nil {
				return report, err
			}
		}
		if m.State == runtime.Approve {
			if err := e.transition(m, runtime.Dispatch, "resume approved work", &report); err != nil {
				return report, err
			}
		}
		if m.State == runtime.Decide {
			if err := e.decide(m, runtime.ContinueDecision, "resume requested from durable decision", &report); err != nil {
				return report, err
			}
		}
		if m.Terminal() {
			return report, nil
		}
		if m.State == runtime.Replan {
			if err := e.transition(m, runtime.Plan, "resume requested from durable replan", &report); err != nil {
				return report, err
			}
		}
		if m.State == runtime.Correct {
			if err := e.transition(m, runtime.Context, "resume requested from durable correction", &report); err != nil {
				return report, err
			}
		}
		if m.State == runtime.Continue {
			if err := e.transition(m, runtime.Dispatch, "resume requested from durable continuation", &report); err != nil {
				return report, err
			}
		}
	} else {
		order = work.NewOrder(request)
		order.SchemaVersion = "2"
		order.Autonomy = "full-auto"
		order.Risk = work.AssessRisk(request)
		order.RunID = fmt.Sprintf("RUN-%d", time.Now().UnixNano())
		runID = order.RunID
		reqID = fmt.Sprintf("REQ-%d", time.Now().UnixNano())
		req = domain.Requirement{SchemaVersion: "2", ID: reqID, Title: order.Objective, Description: order.SourceRequest, Acceptance: []string{"work objective is satisfied", "appropriate validation passes"}, Status: domain.StatusPlanned, Source: "user", Provenance: []string{runID}}
		order.Requirements = []string{req.ID}
		if err := e.Store.Write("work/"+order.ID+".json", order); err != nil {
			return Report{}, err
		}
		if err := e.Store.Write("requirements/"+req.ID+".json", req); err != nil {
			return Report{}, err
		}
		m = runtime.New()
		report = Report{RunID: runID, WorkOrderID: order.ID, State: m.State}
		if err := e.event(runID, "REQUEST_ACCEPTED", map[string]any{"request": order.SourceRequest, "workOrderId": order.ID, "requirementId": req.ID}); err != nil {
			return report, err
		}
		baseline := git.Inspect(ctx, e.Root)
		if err := e.event(runID, "GIT_BASELINE", map[string]any{"available": baseline.Available, "dirty": baseline.Dirty, "head": baseline.Head, "diffDigest": baseline.DiffDigest, "changedFiles": baseline.ChangedFiles, "error": baseline.Error}); err != nil {
			return report, err
		}
		if err := e.persist(m, report, nil); err != nil {
			return report, err
		}
		for _, next := range []runtime.State{runtime.Understand, runtime.Assess, runtime.Plan} {
			if err := e.transition(m, next, "canonical lifecycle", &report); err != nil {
				return e.block(ctx, m, report, err)
			}
		}
		if requiresApproval(order.Risk) {
			decision := policy.Evaluate("high_risk_request")
			if err := e.event(runID, "POLICY_DECISION", map[string]any{"action": "request-risk", "decision": decision.Decision, "allowed": false, "requiresApproval": true, "reason": decision.Reason, "risk": order.Risk}); err != nil {
				return report, err
			}
			return e.block(ctx, m, report, fmt.Errorf("request risk %s requires explicit approval before execution", order.Risk.Level))
		}
	}
	order.Status = domain.StatusRunning
	order.UpdatedAt = time.Now().UTC()
	if err := e.Store.Write("work/"+order.ID+".json", order); err != nil {
		return report, err
	}
	var current harness.Adapter = adapter
	var lastFindings []domain.Finding
	allEvidence := append([]string(nil), report.EvidenceIDs...)
	changed := append([]string(nil), report.ChangedFiles...)
	var allValidations []validation.Result
	previousActions := []string{}
	for attempt := 1; attempt <= e.Limits.MaxAttempts; attempt++ {
		report.Attempts = attempt
		if attempt > 1 {
			if err := e.transition(m, runtime.Context, "correction requires refreshed context", &report); err != nil {
				return e.block(ctx, m, report, err)
			}
		}
		packet := context.CompileWithImpact(e.Root, work.Packet(order), changed)
		for _, l := range learning.Active(e.Store, "project") {
			packet.Context = append(packet.Context, domain.ContextRef{Type: "learning", Path: "learning/" + l.ID, Reason: "active, evidence-backed project learning", Source: "learning", Relevance: 0.7, TokenEstimate: len(l.ProposedChange) / 4})
		}
		packet.Validation = validation.PlanFor(order.Risk, projectCapabilities(e.Store)).Checks
		report.ContextManifest = fmt.Sprintf("work/%s.packet.%d.json", order.ID, attempt)
		if err := e.Store.Write(report.ContextManifest, packet); err != nil {
			return e.block(ctx, m, report, err)
		}
		if e.Limits.MaxContextTokens > 0 && packetTokens(packet) > e.Limits.MaxContextTokens {
			return e.block(ctx, m, report, fmt.Errorf("context budget exceeded: %d > %d tokens", packetTokens(packet), e.Limits.MaxContextTokens))
		}
		if m.State == runtime.Plan {
			if err := e.transition(m, runtime.Context, "progressive context compiled", &report); err != nil {
				return e.block(ctx, m, report, err)
			}
		}
		if m.State == runtime.Context {
			if err := e.transition(m, runtime.Dispatch, "work packet ready for harness", &report); err != nil {
				return e.block(ctx, m, report, err)
			}
		}
		if current == nil {
			if cfg, err := harness.LoadConfig(e.Root); err == nil {
				current = harness.NewConfiguredProcess(ctx, cfg)
			}
		}
		if current == nil {
			return e.block(ctx, m, report, errors.New("no executable harness is configured"))
		}
		if err := current.Discover(); err != nil {
			return e.block(ctx, m, report, fmt.Errorf("harness discovery: %w", err))
		}
		harnessSession, err := current.Start(packet)
		if err != nil {
			return e.block(ctx, m, report, fmt.Errorf("harness start: %w", err))
		}
		harnessID := "unknown"
		if identified, ok := current.(harness.HarnessIdentity); ok {
			harnessID = identified.HarnessID()
		}
		if err := e.Store.Write("harnesses/"+harnessID+".json", domain.Harness{ID: harnessID, Name: harnessID, Capabilities: current.Capabilities(), Version: "configured"}); err != nil {
			return e.block(ctx, m, report, err)
		}
		startedAt := time.Now().UTC()
		if err := e.Store.Write("harness-sessions/"+harnessSession+".json", domain.HarnessSession{ID: harnessSession, HarnessID: harnessID, RunID: runID, Status: domain.StatusRunning, StartedAt: startedAt}); err != nil {
			return e.block(ctx, m, report, err)
		}
		harnessRunID := fmt.Sprintf("HR-%s-%d", runID, attempt)
		if err := e.Store.Write("harness-runs/"+harnessRunID+".json", domain.HarnessRun{ID: harnessRunID, WorkOrderID: order.ID, HarnessID: harnessID, Status: domain.StatusRunning, StartedAt: startedAt}); err != nil {
			return e.block(ctx, m, report, err)
		}
		if err := e.event(runID, "RUN_STARTED", map[string]any{"workOrderId": order.ID, "sessionId": harnessSession, "harnessCapabilities": current.Capabilities()}); err != nil {
			return e.block(ctx, m, report, err)
		}
		if err := e.event(runID, "PROMPT_SENT", map[string]any{"sessionId": harnessSession, "workOrderId": order.ID, "contextItems": len(packet.Context), "validationChecks": len(packet.Validation)}); err != nil {
			return e.block(ctx, m, report, err)
		}
		for _, required := range []string{"observe", "result"} {
			if !contains(current.Capabilities(), required) {
				if err := e.event(runID, "OBSERVABILITY_LIMITATION", map[string]any{"capability": required, "reason": "adapter does not expose this lifecycle surface"}); err != nil {
					return e.block(ctx, m, report, err)
				}
			}
		}
		if err := e.transition(m, runtime.Execute, "harness started", &report); err != nil {
			return e.block(ctx, m, report, err)
		}
		if err := e.transition(m, runtime.Observe, "collect normalized harness observations", &report); err != nil {
			return e.block(ctx, m, report, err)
		}
		observations, observeErr := collect(ctx, current, e.Limits.MaxObservations, func() bool {
			_, err := os.Stat(filepath.Join(e.Root, state.Dir, "control", "pause.json"))
			return err == nil
		})
		if errors.Is(observeErr, ErrPaused) {
			_ = current.Interrupt()
			pausedStatus, _ := current.Result()
			if err := e.Store.Write("harness-sessions/"+harnessSession+".json", domain.HarnessSession{ID: harnessSession, HarnessID: harnessID, RunID: runID, Status: pausedStatus, StartedAt: startedAt}); err != nil {
				return e.block(ctx, m, report, err)
			}
			pausedFinishedAt := time.Now().UTC()
			if err := e.Store.Write("harness-runs/"+harnessRunID+".json", domain.HarnessRun{ID: harnessRunID, WorkOrderID: order.ID, HarnessID: harnessID, Status: pausedStatus, StartedAt: startedAt, FinishedAt: &pausedFinishedAt}); err != nil {
				return e.block(ctx, m, report, err)
			}
			report.Error = "run paused"
			report.State = m.State
			report.NextAction = m.NextAction(order.Risk.Level, true, "resume from checkpoint after explicit pause")
			if err := e.checkpoint(m, report, changed, nil); err != nil {
				return report, err
			}
			snap, err := e.Store.LoadSnapshot()
			if err != nil {
				return e.block(ctx, m, report, err)
			}
			snap.Paused = true
			snap.LastError = report.Error
			if err := e.Store.SaveSnapshot(snap); err != nil {
				return report, err
			}
			return report, nil
		}
		claims := []string{}
		actions := []string{}
		observationIDs := []string{}
		for _, o := range observations {
			if o.ID != "" {
				observationIDs = append(observationIDs, o.ID)
			}
			if o.Type == "COMPLETION_CLAIM" {
				claims = append(claims, o.Summary)
			}
			if o.Type == "TOOL_STARTED" || o.Type == "COMMAND_STARTED" {
				actions = append(actions, o.Type+":"+o.Summary)
			}
			if err := e.event(runID, "OBSERVATION", map[string]any{"observation": o}); err != nil {
				return e.block(ctx, m, report, err)
			}
		}
		if observeErr != nil && observeErr != io.EOF {
			if err := e.event(runID, "OBSERVATION_GAP", map[string]any{"error": observeErr.Error()}); err != nil {
				return e.block(ctx, m, report, err)
			}
		}
		status, resultErr := current.Result()
		if resultErr != nil {
			observeErr = resultErr
		}
		if observeErr != nil && status == domain.StatusUnknown {
			status = domain.StatusFailed
		}
		finishedAt := time.Now().UTC()
		if err := e.Store.Write("harness-sessions/"+harnessSession+".json", domain.HarnessSession{ID: harnessSession, HarnessID: harnessID, RunID: runID, Status: status, StartedAt: startedAt}); err != nil {
			return e.block(ctx, m, report, err)
		}
		if err := e.Store.Write("harness-runs/"+harnessRunID+".json", domain.HarnessRun{ID: harnessRunID, WorkOrderID: order.ID, HarnessID: harnessID, Status: status, StartedAt: startedAt, FinishedAt: &finishedAt}); err != nil {
			return e.block(ctx, m, report, err)
		}
		if err := e.event(runID, "RUN_STOPPED", map[string]any{"sessionId": harnessSession, "harnessRunId": harnessRunID, "status": status}); err != nil {
			return e.block(ctx, m, report, err)
		}
		if err := e.transition(m, runtime.Verify, "harness result requires deterministic validation", &report); err != nil {
			return e.block(ctx, m, report, err)
		}
		caps := projectCapabilities(e.Store)
		plan := validation.PlanFor(order.Risk, caps)
		gitState := git.Inspect(ctx, e.Root)
		if gitState.Available {
			changed = gitState.ChangedFiles
		}
		report.ChangedFiles = changed
		harnessEvidence, err := evidence.RecordScoped(e.Store, order.ID, "harness-result", "harness process result", gitState.Head, gitState.DiffDigest, observationIDs, status == domain.StatusCompleted)
		if err != nil {
			return e.block(ctx, m, report, err)
		}
		allEvidence = append(allEvidence, harnessEvidence.ID)
		if err := e.event(runID, "EVIDENCE_RECORDED", map[string]any{"evidenceId": harnessEvidence.ID, "type": harnessEvidence.Type, "status": harnessEvidence.Verification}); err != nil {
			return e.block(ctx, m, report, err)
		}
		if gitState.Available {
			ev, err := evidence.RecordScoped(e.Store, order.ID, "git-state", "Git state observed", gitState.Head, gitState.DiffDigest, observationIDs, true)
			if err != nil {
				return e.block(ctx, m, report, err)
			}
			allEvidence = append(allEvidence, ev.ID)
			if err := e.event(runID, "EVIDENCE_RECORDED", map[string]any{"evidenceId": ev.ID, "type": ev.Type, "status": ev.Verification}); err != nil {
				return e.block(ctx, m, report, err)
			}
		}
		validations := []validation.Result{}
		validationIDs := []string{}
		policyBlocked := false
		for _, command := range validation.Commands(plan, e.Root, caps) {
			decision := policy.CommandWithApproval(command.Args, e.Root, hasApproval(e.Store, runID))
			if err := e.event(runID, "POLICY_DECISION", map[string]any{"command": command.Name, "decision": decision.Decision, "allowed": decision.Allowed, "requiresApproval": decision.RequiresApproval, "reason": decision.Reason}); err != nil {
				return e.block(ctx, m, report, err)
			}
			if !decision.Allowed {
				validations = append(validations, validation.Result{Name: command.Name, Tier: command.Tier, Passed: false, ExitCode: -1, Stderr: decision.Reason})
				policyBlocked = true
				continue
			}
			r := validation.Run(ctx, command)
			artifactRefs := []string{}
			if r.Stdout != "" || r.Stderr != "" {
				a, artifactErr := artifact.SaveText(e.Store, "validation-output", r.Stdout+"\n"+r.Stderr)
				if artifactErr != nil {
					return e.block(ctx, m, report, artifactErr)
				}
				artifactRefs = append(artifactRefs, a.ID)
				r.ArtifactRefs = artifactRefs
				r.Stdout = truncate(r.Stdout, 4096)
				r.Stderr = truncate(r.Stderr, 4096)
			}
			validations = append(validations, r)
			if err := e.event(runID, "VALIDATION_RESULT", map[string]any{"name": r.Name, "passed": r.Passed, "exitCode": r.ExitCode, "duration": r.Duration.String()}); err != nil {
				return e.block(ctx, m, report, err)
			}
			summary := r.Name + " validation"
			ev, err := evidence.RecordScopedArtifacts(e.Store, order.ID, "validation", summary, gitState.Head, gitState.DiffDigest, observationIDs, artifactRefs, r.Passed)
			if err != nil {
				return e.block(ctx, m, report, err)
			}
			allEvidence = append(allEvidence, ev.ID)
			if err := e.event(runID, "EVIDENCE_RECORDED", map[string]any{"evidenceId": ev.ID, "type": ev.Type, "status": ev.Verification}); err != nil {
				return e.block(ctx, m, report, err)
			}
			validationID := fmt.Sprintf("VAL-%s-%d-%d", runID, attempt, len(validations))
			validationIDs = append(validationIDs, validationID)
			if err := e.Store.Write("validations/"+validationID+".json", domain.Validation{ID: validationID, Name: r.Name, Tier: r.Tier, Passed: r.Passed, EvidenceID: ev.ID}); err != nil {
				return e.block(ctx, m, report, err)
			}
		}
		allValidations = append(allValidations, validations...)
		report.EvidenceIDs = append([]string(nil), allEvidence...)
		report.Validations = append([]validation.Result(nil), allValidations...)
		validationPassed := status == domain.StatusCompleted && len(validations) > 0 && allPassed(validations)
		if err := e.transition(m, runtime.Evaluate, "validation and evidence evaluated", &report); err != nil {
			return e.block(ctx, m, report, err)
		}
		findings := supervisor.Evaluate(supervisor.Result{Status: string(status), Claims: claims, ChangedFiles: changed, ValidationPassed: validationPassed, PreviousActions: append(previousActions, actions...), RequirementsSatisfied: validationPassed, ScopeAllowed: true, ToolCount: len(actions), MaxToolCount: e.Limits.MaxToolCalls, ContextTokens: packetTokens(packet), MaxContextTokens: e.Limits.MaxContextTokens})
		previousActions = append(previousActions, actions...)
		lastFindings = findings
		report.Findings = findings
		if err := e.event(runID, "FINDINGS_RECORDED", map[string]any{"findings": findings}); err != nil {
			return e.block(ctx, m, report, err)
		}
		if len(findings) > 0 {
			l := domain.Learning{ID: fmt.Sprintf("L-%s-%d", runID, attempt), Scope: "project", Observation: "supervisor finding: " + findings[0].Type, BeforeEvidenceIDs: append([]string(nil), allEvidence...), EvidenceIDs: append([]string(nil), allEvidence...), Confidence: findings[0].Confidence, ProposedChange: findings[0].RecommendedAction, Rollback: "supersede this learning record and restore the prior strategy", Status: "OBSERVED", Version: 1}
			saved, err := learning.Transition(e.Store, l, "CANDIDATE", "")
			if err != nil {
				return e.block(ctx, m, report, err)
			}
			if err := e.event(runID, "LEARNING_CANDIDATE", map[string]any{"learningId": saved.ID, "finding": findings[0].Type}); err != nil {
				return e.block(ctx, m, report, err)
			}
		}
		if err := e.transition(m, runtime.Supervise, "deterministic supervisor evaluated outcome", &report); err != nil {
			return e.block(ctx, m, report, err)
		}
		if err := e.transition(m, runtime.Decide, "select bounded next action", &report); err != nil {
			return e.block(ctx, m, report, err)
		}
		if hasFinding(findings, supervisor.ExcessiveActivity) {
			if err := e.decide(m, runtime.StopDecision, "hard activity limit exceeded", &report); err != nil {
				return e.block(ctx, m, report, err)
			}
			report.State = m.State
			report.NextAction = m.NextAction(order.Risk.Level, true, "hard activity limit exceeded")
			report.Error = "hard activity limit exceeded"
			order.Status = domain.StatusFailed
			order.UpdatedAt = time.Now().UTC()
			if err := e.Store.Write("work/"+order.ID+".json", order); err != nil {
				return report, err
			}
			if err := e.checkpoint(m, report, changed, findings); err != nil {
				return report, err
			}
			return report, nil
		}
		if policyBlocked {
			if err := e.decide(m, runtime.BlockDecision, "validation command requires explicit policy approval", &report); err != nil {
				return e.block(ctx, m, report, err)
			}
			report.State = m.State
			report.NextAction = m.NextAction(order.Risk.Level, false, "validation command requires explicit approval")
			report.EvidenceIDs = allEvidence
			report.Validations = allValidations
			order.Status = domain.StatusBlocked
			order.UpdatedAt = time.Now().UTC()
			if err := e.Store.Write("work/"+order.ID+".json", order); err != nil {
				return report, err
			}
			if err := e.checkpoint(m, report, changed, findings); err != nil {
				return report, err
			}
			return report, nil
		}
		if validationPassed && len(highFindings(findings)) == 0 {
			completionPolicy := policy.Evaluate("complete")
			if err := e.event(runID, "POLICY_DECISION", map[string]any{"action": "complete", "decision": completionPolicy.Decision, "allowed": completionPolicy.Allowed, "requiresApproval": completionPolicy.RequiresApproval, "reason": completionPolicy.Reason}); err != nil {
				return e.block(ctx, m, report, err)
			}
			if !completionPolicy.Allowed {
				return e.block(ctx, m, report, fmt.Errorf("completion blocked by policy: %s", completionPolicy.Reason))
			}
			if err := e.decide(m, runtime.CompleteDecision, "requirements and validation have corroborating evidence", &report); err != nil {
				return e.block(ctx, m, report, err)
			}
			report.State = m.State
			report.NextAction = m.NextAction(order.Risk.Level, true, "verified completion")
			report.EvidenceIDs = allEvidence
			report.Validations = allValidations
			if reqID != "" {
				req.Status = domain.StatusVerified
				req.WorkOrderID = order.ID
				req.ChangedFiles = append([]string(nil), changed...)
				req.ValidationIDs = append([]string(nil), validationIDs...)
				req.EvidenceIDs = append([]string(nil), allEvidence...)
				if err := e.Store.Write("requirements/"+req.ID+".json", req); err != nil {
					return report, err
				}
			}
			order.Status = domain.StatusCompleted
			order.UpdatedAt = time.Now().UTC()
			if err := e.Store.Write("work/"+order.ID+".json", order); err != nil {
				return report, err
			}
			if err := e.checkpoint(m, report, changed, nil); err != nil {
				return report, err
			}
			return report, nil
		}
		if attempt < e.Limits.MaxAttempts {
			if err := e.decide(m, runtime.CorrectDecision, "validation failed or completion was unsupported", &report); err != nil {
				return e.block(ctx, m, report, err)
			}
			previousHarness := "unknown"
			if identified, ok := current.(harness.HarnessIdentity); ok {
				previousHarness = identified.HarnessID()
			}
			if e.AdapterFactory != nil {
				current = e.AdapterFactory()
			} else {
				current = nil
			}
			if current != nil {
				nextHarness := "unknown"
				if identified, ok := current.(harness.HarnessIdentity); ok {
					nextHarness = identified.HarnessID()
				}
				if err := e.event(runID, "HARNESS_SWITCHED", map[string]any{"from": previousHarness, "to": nextHarness, "attempt": attempt + 1, "reason": "bounded correction after failed verification"}); err != nil {
					return e.block(ctx, m, report, err)
				}
			}
			if err := e.checkpoint(m, report, changed, findings); err != nil {
				return report, err
			}
			continue
		}
		if err := e.decide(m, runtime.StopDecision, "bounded attempts exhausted without verified completion", &report); err != nil {
			return e.block(ctx, m, report, err)
		}
		report.State = m.State
		report.NextAction = m.NextAction(order.Risk.Level, true, "safe stop after failed verification")
		report.EvidenceIDs = allEvidence
		report.Validations = allValidations
		report.Error = "completion was not verified"
		order.Status = domain.StatusFailed
		order.UpdatedAt = time.Now().UTC()
		if err := e.Store.Write("work/"+order.ID+".json", order); err != nil {
			return report, err
		}
		if err := e.checkpoint(m, report, changed, lastFindings); err != nil {
			return report, err
		}
		return report, nil
	}
	return report, nil
}

func (e *Engine) transition(m *runtime.Machine, to runtime.State, reason string, report *Report) error {
	from := m.State
	if err := m.TransitionTo(to, reason); err != nil {
		return err
	}
	report.State = m.State
	report.NextAction = m.NextAction("low", true, reason)
	if err := e.event(report.RunID, "STATE_TRANSITION", map[string]any{"from": from, "to": to, "reason": reason}); err != nil {
		return err
	}
	return e.persist(m, *report, nil)
}
func (e *Engine) decide(m *runtime.Machine, d runtime.Decision, reason string, report *Report) error {
	from := m.State
	if err := m.ApplyDecision(d, reason); err != nil {
		return err
	}
	decisionID := fmt.Sprintf("DEC-%s-%d", report.RunID, time.Now().UnixNano())
	if err := e.Store.Write("decisions/"+decisionID+".json", domain.Decision{ID: decisionID, Summary: string(d), Rationale: reason, Source: "keystone-control", CreatedAt: time.Now().UTC()}); err != nil {
		return err
	}
	report.State = m.State
	report.NextAction = m.NextAction("low", d != runtime.BlockDecision, reason)
	if err := e.event(report.RunID, "DECISION", map[string]any{"from": from, "decision": d, "decisionId": decisionID, "reason": reason, "state": m.State}); err != nil {
		return err
	}
	return e.persist(m, *report, nil)
}
func (e *Engine) event(runID, typ string, payload map[string]any) error {
	return e.Journal.Append(observation.Event{RunID: runID, Type: typ, Source: "keystone-control", OperationID: fmt.Sprintf("%s:%s:%d", runID, typ, time.Now().UnixNano()), Payload: payload})
}
func (e *Engine) persist(m *runtime.Machine, report Report, checkpointID *string) error {
	snap := state.Snapshot{SchemaVersion: "2", Lifecycle: string(m.State), RunID: report.RunID, WorkOrderID: report.WorkOrderID, Machine: m, NextAction: &report.NextAction, Findings: report.Findings, EvidenceIDs: report.EvidenceIDs, ChangedFiles: report.ChangedFiles, ContextManifest: report.ContextManifest, UpdatedAt: time.Now().UTC(), LastError: report.Error}
	if checkpointID != nil {
		snap.CheckpointID = *checkpointID
	}
	return e.Store.SaveSnapshot(snap)
}
func (e *Engine) checkpoint(m *runtime.Machine, report Report, changed []string, findings []domain.Finding) error {
	id := fmt.Sprintf("CP-%s-%d", report.RunID, report.Attempts)
	c := domain.Checkpoint{SchemaVersion: "2", ID: id, WorkOrderID: report.WorkOrderID, RunID: report.RunID, State: string(m.State), ChangedFiles: changed, ContextManifest: report.ContextManifest, HarnessID: "local-process", NextAction: &report.NextAction, CreatedAt: time.Now().UTC()}
	for _, t := range m.History {
		c.Completed = append(c.Completed, string(t.To))
	}
	if len(findings) > 0 {
		for _, f := range findings {
			c.Blockers = append(c.Blockers, f.Type)
		}
	}
	if err := checkpoint.Save(e.Store, c); err != nil {
		return err
	}
	if err := e.event(report.RunID, "CHECKPOINT_CREATED", map[string]any{"checkpointId": id, "state": m.State}); err != nil {
		return err
	}
	return e.persist(m, report, &id)
}
func (e *Engine) block(ctx stdcontext.Context, m *runtime.Machine, report Report, err error) (Report, error) {
	report.Error = err.Error()
	if report.WorkOrderID != "" {
		var order domain.WorkOrder
		if readErr := e.Store.Read("work/"+report.WorkOrderID+".json", &order); readErr == nil {
			order.Status = domain.StatusBlocked
			order.UpdatedAt = time.Now().UTC()
			if writeErr := e.Store.Write("work/"+report.WorkOrderID+".json", order); writeErr != nil {
				return report, writeErr
			}
			for _, requirementID := range order.Requirements {
				var requirement domain.Requirement
				if readErr := e.Store.Read("requirements/"+requirementID+".json", &requirement); readErr == nil {
					requirement.Status = domain.StatusBlocked
					if writeErr := e.Store.Write("requirements/"+requirementID+".json", requirement); writeErr != nil {
						return report, writeErr
					}
				}
			}
		}
	}
	if m.State != runtime.Blocked && m.State != runtime.Stopped {
		if transitionErr := e.transition(m, runtime.Blocked, err.Error(), &report); transitionErr != nil {
			return report, transitionErr
		}
	}
	report.State = m.State
	report.NextAction = m.NextAction("high", false, err.Error())
	if checkpointErr := e.checkpoint(m, report, report.ChangedFiles, nil); checkpointErr != nil {
		return report, checkpointErr
	}
	return report, nil
}

func (e *Engine) Continue(ctx stdcontext.Context, adapter harness.Adapter) (Report, error) {
	snapshot, err := e.Store.LoadSnapshot()
	if err != nil {
		snapshot, err = RecoverSnapshot(e, "")
		if err != nil {
			return Report{}, fmt.Errorf("load or recover snapshot: %w", err)
		}
	}
	if snapshot.Machine == nil {
		return Report{}, fmt.Errorf("snapshot has no recoverable machine")
	}
	if snapshot.Paused {
		snapshot.Paused = false
		_ = os.Remove(filepath.Join(e.Root, state.Dir, "control", "pause.json"))
		if err := e.Store.SaveSnapshot(snapshot); err != nil {
			return Report{}, err
		}
	}
	if snapshot.WorkOrderID == "" {
		return Report{}, fmt.Errorf("no resumable work order")
	}
	var order domain.WorkOrder
	if err := e.Store.Read("work/"+snapshot.WorkOrderID+".json", &order); err != nil {
		return Report{}, err
	}
	if snapshot.Machine.Terminal() {
		return Report{RunID: snapshot.RunID, WorkOrderID: order.ID, State: snapshot.Machine.State, NextAction: snapshot.NextActionOr(snapshot.Machine)}, nil
	}
	recoveryReport := Report{RunID: snapshot.RunID, WorkOrderID: order.ID, State: snapshot.Machine.State, NextAction: snapshot.NextActionOr(snapshot.Machine), Findings: snapshot.Findings, EvidenceIDs: snapshot.EvidenceIDs}
	if snapshot.Machine.State == runtime.Observe || snapshot.Machine.State == runtime.Execute || snapshot.Machine.State == runtime.Verify || snapshot.Machine.State == runtime.Evaluate || snapshot.Machine.State == runtime.Supervise {
		if err := recoverToDispatch(e, snapshot.Machine, &recoveryReport); err != nil {
			return Report{}, err
		}
		snapshot.Machine.State = runtime.Dispatch
	}
	e.resume = &resumeInput{Order: order, Snapshot: snapshot}
	defer func() { e.resume = nil }()
	return e.Run(ctx, order.SourceRequest, adapter)
}
func collect(ctx stdcontext.Context, a harness.Adapter, max int, paused func() bool) ([]domain.Observation, error) {
	out := []domain.Observation{}
	for i := 0; i < max; i++ {
		if paused != nil && paused() {
			return out, ErrPaused
		}
		type result struct {
			items []domain.Observation
			err   error
		}
		ch := make(chan result, 1)
		go func() { v, err := a.Observe(); ch <- result{v, err} }()
		ticker := time.NewTicker(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			ticker.Stop()
			_ = a.Interrupt()
			return out, ctx.Err()
		case r := <-ch:
			ticker.Stop()
			if r.err != nil {
				return out, r.err
			}
			if len(r.items) == 0 {
				return out, nil
			}
			out = append(out, r.items...)
		case <-ticker.C:
			if paused != nil && paused() {
				ticker.Stop()
				_ = a.Interrupt()
				return out, ErrPaused
			}
		}
	}
	_ = a.Interrupt()
	return out, fmt.Errorf("observation limit exceeded")
}

func recoverToDispatch(e *Engine, m *runtime.Machine, report *Report) error {
	if m.State == runtime.Execute {
		if err := e.transition(m, runtime.Observe, "recover interrupted execution", report); err != nil {
			return err
		}
	}
	if m.State == runtime.Observe {
		if err := e.transition(m, runtime.Verify, "recover from observation checkpoint", report); err != nil {
			return err
		}
	}
	if m.State == runtime.Verify {
		if err := e.transition(m, runtime.Evaluate, "recover from verification checkpoint", report); err != nil {
			return err
		}
	}
	if m.State == runtime.Evaluate {
		if err := e.transition(m, runtime.Supervise, "recover from evaluation checkpoint", report); err != nil {
			return err
		}
	}
	if m.State == runtime.Supervise {
		if err := e.transition(m, runtime.Decide, "recover from supervision checkpoint", report); err != nil {
			return err
		}
	}
	if m.State == runtime.Decide {
		if err := e.decide(m, runtime.CorrectDecision, "replan interrupted execution", report); err != nil {
			return err
		}
	}
	if m.State == runtime.Correct {
		if err := e.transition(m, runtime.Context, "rebuild context after interruption", report); err != nil {
			return err
		}
	}
	if m.State == runtime.Context {
		if err := e.transition(m, runtime.Dispatch, "redispatch after interruption", report); err != nil {
			return err
		}
	}
	return nil
}
func allPassed(rs []validation.Result) bool {
	for _, r := range rs {
		if !r.Passed {
			return false
		}
	}
	return true
}
func highFindings(fs []domain.Finding) []domain.Finding {
	out := []domain.Finding{}
	for _, f := range fs {
		if f.Severity == "high" || f.Severity == "critical" {
			out = append(out, f)
		}
	}
	return out
}

func hasFinding(fs []domain.Finding, typ string) bool {
	for _, f := range fs {
		if f.Type == typ {
			return true
		}
	}
	return false
}
func packetTokens(p domain.WorkPacket) int {
	n := 0
	for _, c := range p.Context {
		n += c.TokenEstimate
	}
	return n
}
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[artifact-backed]"
}
func projectCapabilities(s state.Store) []domain.Capability {
	var p domain.Project
	if s.Read("project.json", &p) != nil {
		return nil
	}
	return p.Capabilities
}
func hasCapability(cs []domain.Capability, kind, name string) bool {
	for _, c := range cs {
		if c.Kind == kind && c.Name == name {
			return true
		}
	}
	return false
}

func requiresApproval(risk domain.Risk) bool {
	return risk.Level == "high" || risk.Level == "release"
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func hasApproval(s state.Store, runID string) bool {
	entries, err := os.ReadDir(filepath.Join(s.Root, state.Dir, "approvals"))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var a domain.Approval
		if s.Read("approvals/"+entry.Name(), &a) == nil && (a.RunID == "" || a.RunID == runID) && (a.Action == "" || a.Action == "CONTINUE" || a.Action == "APPROVE") {
			return true
		}
	}
	return false
}
func DecodeObservation(payload map[string]any) (domain.Observation, error) {
	raw, ok := payload["observation"]
	if !ok {
		return domain.Observation{}, fmt.Errorf("event has no observation")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return domain.Observation{}, err
	}
	var o domain.Observation
	return o, json.Unmarshal(b, &o)
}
