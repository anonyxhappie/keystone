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
	"github.com/anonyxhappie/keystone/internal/repl"
	"github.com/anonyxhappie/keystone/internal/runtime"
	"github.com/anonyxhappie/keystone/internal/session"
	"github.com/anonyxhappie/keystone/internal/state"
	"github.com/anonyxhappie/keystone/internal/ui"
	"github.com/anonyxhappie/keystone/internal/validation"
	"github.com/anonyxhappie/keystone/internal/work"
	"golang.org/x/term"
)

const version = "2.1.3"

func parseFlags(args []string) (root, harness, model string, remaining []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if (arg == "-C" || arg == "--root") && i+1 < len(args) {
			root = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--root=") {
			root = strings.TrimPrefix(arg, "--root=")
		} else if (arg == "--harness" || arg == "-harness") && i+1 < len(args) {
			harness = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--harness=") || strings.HasPrefix(arg, "-harness=") {
			harness = strings.TrimPrefix(strings.TrimPrefix(arg, "--harness="), "-harness=")
		} else if (arg == "--model" || arg == "-model" || arg == "-m") && i+1 < len(args) {
			model = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--model=") || strings.HasPrefix(arg, "-model=") {
			model = strings.TrimPrefix(strings.TrimPrefix(arg, "--model="), "-model=")
		} else {
			remaining = append(remaining, arg)
		}
	}
	return
}

func ensureDefaultInit(root string) bool {
	s := state.New(root)
	if s.Initialized() {
		return false
	}
	caps := project.Detect(root)
	p, err := s.Init(filepath.Base(root), caps)
	if err != nil {
		return false
	}
	if err := s.Write("project.json", p); err != nil {
		return false
	}
	j, err := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	if err == nil {
		_ = j.Append(observation.Event{
			Type:        "PROJECT_INITIALIZED",
			Source:      "system",
			OperationID: fmt.Sprintf("auto-init:%d", time.Now().UnixNano()),
			Payload:     map[string]any{"name": p.Name, "capabilities": caps, "auto": true},
		})
	}
	return true
}

func main() {
	parsedRoot, flagHarness, flagModel, args := parseFlags(os.Args[1:])
	root := parsedRoot
	if root == "" {
		root = os.Getenv("KEYSTONE_ROOT")
	}
	if root == "" {
		root, _ = os.Getwd()
	}

	if len(args) < 1 {
		runREPL(root, flagHarness, flagModel)
		return
	}

	switch args[0] {
	case "repl", "interactive", "shell":
		harnessArg := flagHarness
		if len(args) > 1 {
			harnessArg = args[1]
		}
		runREPL(root, harnessArg, flagModel)
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
	case "version", "--version", "-v":
		fmt.Println(version)
	case "help", "--help", "-h":
		usage()
	default:
		runRun(root, args)
	}
}

func runREPL(root, flagHarness, flagModel string) {
	if ensureDefaultInit(root) {
		fmt.Fprintf(os.Stdout, "%s Initialized default Keystone project state (.keystone/)\n\n", ui.Green+"✔"+ui.Reset)
	}

	harness := flagHarness
	model := flagModel

	if term.IsTerminal(int(os.Stdin.Fd())) {
		savedHarness := session.GetLastHarness(root)
		if harness == "" {
			harness = session.PromptHarness(os.Stdin, os.Stdout, savedHarness)
		}
		session.SetLastHarness(root, harness)

		savedModel := session.GetLastModel(root, harness)
		if model == "" {
			model = session.PromptModel(os.Stdin, os.Stdout, harness, savedModel)
		}
		session.SetLastModel(root, model)
	} else {
		if harness == "" {
			harness = session.GetLastHarness(root)
		}
		if model == "" {
			model = session.GetLastModel(root, harness)
		}
	}

	r := repl.New(root, harness, model, os.Stdin, os.Stdout)
	if err := r.Run(); err != nil {
		fatal(err)
	}
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  keystone                              Start interactive terminal shell (Claude Code / Antigravity style)")
	fmt.Println("  keystone --harness <name>             Start interactive terminal shell with specified harness")
	fmt.Println("  keystone run [--harness <name>] <req> Execute a single supervised request")
	fmt.Println("  keystone status                       Inspect current project status")
	fmt.Println("  keystone validate                     Run deterministic validation checks")
	fmt.Println("  keystone review                       Inspect supervisor review findings")
	fmt.Println("  keystone replay <run-id>              Replay events of a run")
	fmt.Println("  keystone doctor                       Check environment and harnesses")
	fmt.Println("  keystone version                      Print version")
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
	ensureDefaultInit(root)
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
	ensureDefaultInit(root)
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

func parseRunArgs(args []string) (harnessName string, jsonMode bool, request string, err error) {
	words := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--json" || arg == "-json" {
			jsonMode = true
			continue
		}
		if arg == "--harness" || arg == "-harness" {
			if i+1 >= len(args) {
				return "", false, "", fmt.Errorf("--harness requires an argument (e.g. codex, antigravity, auto)")
			}
			harnessName = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--harness=") {
			harnessName = strings.TrimPrefix(arg, "--harness=")
			continue
		}
		if strings.HasPrefix(arg, "-harness=") {
			harnessName = strings.TrimPrefix(arg, "-harness=")
			continue
		}
		words = append(words, arg)
	}

	if harnessName != "" {
		norm := strings.ToLower(strings.TrimSpace(harnessName))
		if norm != "codex" && norm != "antigravity" && norm != "agy" && norm != "auto" {
			return "", false, "", fmt.Errorf("invalid harness %q; valid options are: codex, antigravity, auto", harnessName)
		}
		harnessName = norm
	}

	request = strings.TrimSpace(strings.Join(words, " "))
	return harnessName, jsonMode, request, nil
}

func runRun(root string, args []string) {
	ensureDefaultInit(root)
	harnessFlag, jsonMode, request, err := parseRunArgs(args)
	if err != nil {
		fatal(err)
	}
	if request == "" {
		fatal(fmt.Errorf("run requires a request"))
	}
	e, err := control.Open(root)
	if err != nil {
		fatal(err)
	}
	e.RequestedHarness = harnessFlag
	e.AdapterFactory = localFactory(root, harnessFlag)

	var term *ui.Terminal
	if !jsonMode {
		term = ui.New(os.Stdout)
		term.Header(root, harnessFlag, work.IsReadOnlyRequest(request), request)
		e.OnEvent = term.OnEvent
		e.OnObservation = term.OnObservation
	}

	report, runErr := e.Run(context.Background(), request, nil)

	if jsonMode {
		if runErr != nil {
			fatal(runErr)
		}
		printJSON(report)
		return
	}

	summaries := []ui.ValidationSummary{}
	for _, v := range report.Validations {
		summary := ""
		if !v.Passed {
			if strings.Contains(v.Stderr, "Can't reach database server") || strings.Contains(v.Stdout, "Can't reach database server") {
				summary = "Environment blocker: PostgreSQL not reachable at localhost:5432"
			} else if strings.Contains(v.Stdout, "new blank line at EOF") {
				summary = "Git style: trailing blank line at EOF"
			} else if v.Stderr != "" {
				summary = strings.Split(strings.TrimSpace(v.Stderr), "\n")[0]
			} else {
				summary = fmt.Sprintf("exit code %d", v.ExitCode)
			}
		}
		summaries = append(summaries, ui.ValidationSummary{
			Name:    v.Name,
			Passed:  v.Passed,
			Summary: summary,
		})
	}

	term.Report(
		report.RunID,
		report.WorkOrderID,
		report.HarnessID,
		report.HarnessSessionID,
		string(report.State),
		string(report.NextAction.Type),
		report.NextAction.Reason,
		report.ReadOnly,
		len(report.Mutations),
		report.ContextTokens,
		report.Attempts,
		e.Limits.MaxAttempts,
		report.Error,
		summaries,
	)

	if report.NextAction.Type == "ASK" || report.NextAction.RequiresApproval {
		if term.PromptApproval(report.NextAction.Reason) {
			runApprove(root, []string{"CONTINUE"})
			runContinue(root)
		}
	}
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
	ensureDefaultInit(root)
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
	reviewOutput := map[string]any{"state": snap.Lifecycle, "findings": snap.Findings, "invalidEvidence": invalidEvidence, "recommendation": recommendation}
	if snap.HarnessSelection != nil {
		reviewOutput["harnessSelection"] = snap.HarnessSelection
	}
	if snap.ReadOnly {
		reviewOutput["readOnly"] = snap.ReadOnly
	}
	if len(snap.Mutations) > 0 {
		reviewOutput["mutations"] = snap.Mutations
	}
	printJSON(reviewOutput)
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

func localFactory(root string, requested ...string) func() harness.Adapter {
	req := ""
	if len(requested) > 0 {
		req = requested[0]
	}
	return func() harness.Adapter {
		if req != "" && req != "auto" {
			adapter, _, err := harness.SelectHarness(context.Background(), root, req)
			if err == nil && adapter != nil {
				return adapter
			}
			return nil
		}
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
