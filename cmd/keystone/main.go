package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	controlctx "github.com/anonyxhappie/keystone/internal/context"
	"github.com/anonyxhappie/keystone/internal/control"
	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/evidence"
	"github.com/anonyxhappie/keystone/internal/git"
	"github.com/anonyxhappie/keystone/internal/harness"
	"github.com/anonyxhappie/keystone/internal/observation"
	"github.com/anonyxhappie/keystone/internal/policy"
	"github.com/anonyxhappie/keystone/internal/project"
	"github.com/anonyxhappie/keystone/internal/runtime"
	"github.com/anonyxhappie/keystone/internal/state"
	"github.com/anonyxhappie/keystone/internal/validation"
	"github.com/anonyxhappie/keystone/internal/work"
)

const version = "2.1.0"

func main() {
	args := os.Args[1:]
	root := os.Getenv("KEYSTONE_ROOT")
	if len(args) >= 2 && (args[0] == "-C" || args[0] == "--root") {
		root = args[1]
		args = args[2:]
	}
	if root == "" {
		root, _ = os.Getwd()
	}
	if len(args) < 1 {
		usage()
		return
	}
	switch args[0] {
	case "init":
		runInit(root)
	case "status":
		runStatus(root)
	case "ask":
		runAsk(root, args[1:])
	case "run":
		runRun(root, args[1:])
	case "continue":
		runContinue(root)
	case "pause":
		runPause(root)
	case "approve":
		runApprove(root, args[1:])
	case "stop":
		runStop(root)
	case "validate":
		runValidate(root)
	case "review":
		runReview(root)
	case "replay":
		runReplay(root, args[1:])
	case "doctor":
		runDoctor(root)
	case "version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("keystone init | status | ask <request> | run <request> | continue | pause | approve | stop | validate | review | replay <run-id> | doctor | version")
}

func runInit(root string) {
	if _, err := os.Stat(filepath.Join(root, state.Dir, "project.json")); err == nil {
		fatal(fmt.Errorf("Keystone is already initialized; refusing to overwrite durable state"))
	}
	caps := project.Detect(root)
	p, err := state.New(root).Init(filepath.Base(root), caps)
	if err != nil {
		fatal(err)
	}
	if err := state.New(root).Write("project.json", p); err != nil {
		fatal(err)
	}
	j, err := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	if err != nil {
		fatal(err)
	}
	if err := j.Append(observation.Event{RunID: "project", Type: "PROJECT_INITIALIZED", Source: "keystone", IdempotencyKey: "project-init", Payload: map[string]any{"projectId": p.ID, "capabilities": caps}}); err != nil {
		fatal(err)
	}
	printJSON(p)
}

func runStatus(root string) {
	var p domain.Project
	if err := state.New(root).Read("project.json", &p); err != nil {
		fatal(fmt.Errorf("Keystone is not initialized: %w", err))
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now().UTC()
	}
	snapshot, err := state.New(root).LoadSnapshot()
	if err != nil {
		fatal(fmt.Errorf("cannot read durable state: %w", err))
	}
	printJSON(map[string]any{"project": p, "state": snapshot})
}

func runAsk(root string, args []string) {
	if len(args) == 0 {
		fatal(fmt.Errorf("ask requires a request"))
	}
	request := strings.Join(args, " ")
	order := work.NewOrder(request)
	order.SchemaVersion = "2"
	packet := controlctx.Compile(root, work.Packet(order))
	if err := state.New(root).Write("work/"+order.ID+".json", order); err != nil {
		fatal(err)
	}
	printJSON(packet)
}

func runRun(root string, args []string) {
	if len(args) == 0 {
		fatal(fmt.Errorf("run requires a request"))
	}
	e, err := control.Open(root)
	if err != nil {
		fatal(err)
	}
	e.AdapterFactory = localFactory(root)
	report, err := e.Run(context.Background(), strings.Join(args, " "), nil)
	if err != nil {
		fatal(err)
	}
	printJSON(report)
}

func runContinue(root string) {
	e, err := control.Open(root)
	if err != nil {
		fatal(err)
	}
	e.AdapterFactory = localFactory(root)
	report, err := e.Continue(context.Background(), nil)
	if err != nil {
		fatal(err)
	}
	printJSON(report)
}

func runPause(root string) {
	s := state.New(root)
	snap, err := s.LoadSnapshot()
	if err != nil {
		fatal(err)
	}
	if snap.Machine == nil || snap.Machine.Terminal() {
		fatal(fmt.Errorf("cannot pause a terminal run"))
	}
	snap.Paused = true
	snap.UpdatedAt = time.Now().UTC()
	if err := s.SaveSnapshot(snap); err != nil {
		fatal(err)
	}
	if err := s.Write("control/pause.json", map[string]any{"runId": snap.RunID, "requestedAt": time.Now().UTC()}); err != nil {
		fatal(err)
	}
	if err := appendEvent(root, snap.RunID, "RUN_PAUSED", map[string]any{"runId": snap.RunID}); err != nil {
		fatal(err)
	}
	fmt.Println("paused")
}

func runApprove(root string, args []string) {
	action := "CONTINUE"
	if len(args) > 0 {
		action = args[0]
	}
	if action != "CONTINUE" && action != "APPROVE" {
		fatal(fmt.Errorf("unsupported approval action %q", action))
	}
	s := state.New(root)
	snap, err := s.LoadSnapshot()
	if err != nil {
		fatal(err)
	}
	if snap.Machine == nil || (snap.Machine.State != runtime.Blocked && snap.Machine.State != runtime.Ask && snap.Machine.State != runtime.Approve) {
		fatal(fmt.Errorf("approval is only valid for blocked or review-required work"))
	}
	approval := domain.Approval{ID: fmt.Sprintf("APR-%d", time.Now().UnixNano()), Action: action, RunID: snap.RunID, ApprovedBy: "cli-user", Reason: "explicit keystone approve command", At: time.Now().UTC()}
	if err := s.Write("approvals/"+approval.ID+".json", approval); err != nil {
		fatal(err)
	}
	snap.Paused = false
	snap.UpdatedAt = time.Now().UTC()
	if err := s.SaveSnapshot(snap); err != nil {
		fatal(err)
	}
	_ = os.Remove(filepath.Join(root, state.Dir, "control", "pause.json"))
	if err := appendEvent(root, snap.RunID, "APPROVAL_RECORDED", map[string]any{"approvalId": approval.ID, "action": action, "approvedBy": approval.ApprovedBy}); err != nil {
		fatal(err)
	}
	fmt.Println("approval recorded; use keystone continue to resume")
}

func runStop(root string) {
	s := state.New(root)
	snap, err := s.LoadSnapshot()
	if err != nil {
		fatal(err)
	}
	if snap.Machine == nil {
		fatal(fmt.Errorf("no active machine"))
	}
	if snap.Machine.Terminal() {
		printJSON(snap)
		return
	}
	from := snap.Machine.State
	if err := snap.Machine.TransitionTo(runtime.Stopped, "explicit user stop"); err != nil {
		fatal(err)
	}
	snap.Lifecycle = string(runtime.Stopped)
	next := snap.Machine.NextAction("low", true, "explicit user stop")
	snap.NextAction = &next
	snap.UpdatedAt = time.Now().UTC()
	if err := appendEvent(root, snap.RunID, "STATE_TRANSITION", map[string]any{"from": from, "to": runtime.Stopped, "reason": "explicit user stop"}); err != nil {
		fatal(err)
	}
	if err := s.SaveSnapshot(snap); err != nil {
		fatal(err)
	}
	if snap.WorkOrderID != "" {
		var order domain.WorkOrder
		if err := s.Read("work/"+snap.WorkOrderID+".json", &order); err == nil {
			order.Status = domain.StatusFailed
			order.UpdatedAt = time.Now().UTC()
			if err := s.Write("work/"+snap.WorkOrderID+".json", order); err != nil {
				fatal(err)
			}
		}
	}
	printJSON(snap)
}

func runValidate(root string) {
	var p domain.Project
	if err := state.New(root).Read("project.json", &p); err != nil {
		fatal(err)
	}
	plan := validation.PlanFor(domain.Risk{Level: "low"}, p.Capabilities)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	results := []validation.Result{}
	decisions := []domain.PolicyDecision{}
	for _, c := range validation.Commands(plan, root, p.Capabilities) {
		decision := policy.Command(c.Args, root)
		decisions = append(decisions, decision)
		if !decision.Allowed {
			results = append(results, validation.Result{Name: c.Name, Tier: c.Tier, Passed: false, ExitCode: -1, Stderr: decision.Reason})
			continue
		}
		results = append(results, validation.Run(ctx, c))
	}
	printJSON(map[string]any{"tier": plan.Tier, "checks": plan.Checks, "decisions": decisions, "results": results})
}

func runReview(root string) {
	s := state.New(root)
	snap, err := s.LoadSnapshot()
	if err != nil {
		fatal(err)
	}
	invalidEvidence := []string{}
	for _, id := range snap.EvidenceIDs {
		var ev domain.Evidence
		if err := s.Read("evidence/"+id+".json", &ev); err != nil || !evidence.ValidFor(ev, ev.Commit, ev.InputsHash) {
			invalidEvidence = append(invalidEvidence, id)
		}
	}
	recommendation := "review findings and valid evidence before approving consequential actions"
	if len(invalidEvidence) == 0 && len(snap.Findings) == 0 && snap.Lifecycle == string(runtime.Complete) {
		recommendation = "evidence and deterministic findings support the recorded completion"
	}
	printJSON(map[string]any{"state": snap.Lifecycle, "findings": snap.Findings, "invalidEvidence": invalidEvidence, "recommendation": recommendation})
}

func runReplay(root string, args []string) {
	if len(args) == 0 {
		fatal(fmt.Errorf("replay requires a run id"))
	}
	j, err := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	if err != nil {
		fatal(err)
	}
	events, err := j.Replay(args[0])
	if err != nil {
		fatal(err)
	}
	report, err := control.Replay(args[0], events)
	if err != nil {
		fatal(err)
	}
	printJSON(report)
}

func runDoctor(root string) {
	printJSON(map[string]any{"version": version, "harnesses": harness.Discover(root), "git": git.Inspect(context.Background(), root)})
}

func localFactory(root string) func() harness.Adapter {
	return func() harness.Adapter {
		cfg, err := harness.LoadConfig(root)
		if err != nil {
			return nil
		}
		return harness.NewAdapter(context.Background(), root, cfg)
	}
}
func appendEvent(root, runID, typ string, payload map[string]any) error {
	j, err := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	if err != nil {
		return err
	}
	return j.Append(observation.Event{RunID: runID, Type: typ, Source: "cli", OperationID: fmt.Sprintf("cli:%d", time.Now().UnixNano()), Payload: payload})
}
func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal(err)
	}
	fmt.Println(string(b))
}
func fatal(err error) { fmt.Fprintln(os.Stderr, "keystone:", err); os.Exit(1) }
