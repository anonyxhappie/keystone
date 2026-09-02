package control

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/harness"
	"github.com/anonyxhappie/keystone/internal/observation"
	"github.com/anonyxhappie/keystone/internal/state"
)

func initRunnableFixture(t *testing.T, root string) state.Store {
	t.Helper()
	files := map[string]string{
		"go.mod":          "module fixture\n\ngo 1.23\n",
		"fixture.go":      "package fixture\n\nfunc Value() int { return 42 }\n",
		"fixture_test.go": "package fixture\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) { if Value() != 42 { t.Fatal(Value()) } }\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := state.New(root)
	if _, err := s.Init("fixture", []domain.Capability{{Kind: "language", Name: "go"}, {Kind: "test", Name: "go"}}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRunUsesLocalHarnessAndCanonicalLifecycle(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Limits.MaxWallTime = 10 * 1000000000
	adapter := harness.NewLocal(context.Background(), harness.Config{Name: "fixture", Command: "sh", Args: []string{"-c", "read request; echo done"}, TimeoutSeconds: 10})
	report, err := e.Run(context.Background(), "exercise fixture", adapter)
	if err != nil {
		t.Fatal(err)
	}
	if report.State != "COMPLETE" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.NextAction.Type != "COMPLETE" {
		t.Fatalf("unexpected next action: %+v", report.NextAction)
	}
	var harnessEvidence domain.Evidence
	if err := state.New(root).Read("evidence/"+report.EvidenceIDs[0]+".json", &harnessEvidence); err != nil {
		t.Fatal(err)
	}
	if len(harnessEvidence.ObservationIDs) == 0 {
		t.Fatalf("evidence lost observation provenance: %+v", harnessEvidence)
	}
	j, err := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	events, err := j.Replay(report.RunID)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := Replay(report.RunID, events)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Machine.State != "COMPLETE" {
		t.Fatalf("replay did not complete: %+v", replayed.Machine)
	}
}

func TestRunBlocksWithoutExecutableHarness(t *testing.T) {
	root := t.TempDir()
	if _, err := state.New(root).Init("fixture", []domain.Capability{}); err != nil {
		t.Fatal(err)
	}
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := e.Run(context.Background(), "missing harness", nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.State != "BLOCKED" || report.NextAction.Allowed {
		t.Fatalf("unexpected block report: %+v", report)
	}
}

func TestContinueReconstructsBlockedRunAndRequiresApproval(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	first, err := e.Run(context.Background(), "resume me", nil)
	if err != nil {
		t.Fatal(err)
	}
	asked, err := e.Continue(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if asked.State != "ASK" {
		t.Fatalf("continue bypassed approval: %+v", asked)
	}
	approval := domain.Approval{ID: "APR-1", RunID: first.RunID, Action: "CONTINUE", ApprovedBy: "test", At: time.Now().UTC()}
	if err := e.Store.Write("approvals/"+approval.ID+".json", approval); err != nil {
		t.Fatal(err)
	}
	adapter := harness.NewLocal(context.Background(), harness.Config{Command: "sh", Args: []string{"-c", "read request; echo done"}, TimeoutSeconds: 10})
	resumed, err := e.Continue(context.Background(), adapter)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.State != "COMPLETE" || resumed.WorkOrderID != first.WorkOrderID {
		t.Fatalf("run was not reconstructed: %+v", resumed)
	}
}

func TestContinueRecoversCorruptedMaterializedSnapshot(t *testing.T) {
	root := t.TempDir()
	if _, err := state.New(root).Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	first, err := e.Run(context.Background(), "recover me", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, state.Dir, "state.json"), []byte("{corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	recovered, err := e.Continue(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.RunID != first.RunID || recovered.State != "ASK" {
		t.Fatalf("unexpected recovered report: %+v", recovered)
	}
	if _, err := state.New(root).LoadSnapshot(); err != nil {
		t.Fatalf("recovery did not restore materialized state: %v", err)
	}
}

func TestPauseInterruptsBlockedObservation(t *testing.T) {
	root := t.TempDir()
	if _, err := state.New(root).Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Limits.MaxWallTime = 20 * time.Second
	adapter := harness.NewLocal(context.Background(), harness.Config{Command: "sh", Args: []string{"-c", "read request; sleep 10"}, TimeoutSeconds: 20})
	result := make(chan Report, 1)
	go func() {
		report, _ := e.Run(context.Background(), "pause me", adapter)
		result <- report
	}()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, snapshotErr := state.New(root).LoadSnapshot()
		if snapshotErr == nil && snapshot.Lifecycle == "OBSERVE" {
			if err := state.New(root).Write("control/pause.json", map[string]string{"runId": snapshot.RunID}); err != nil {
				t.Fatal(err)
			}
			select {
			case report := <-result:
				if report.Error != "run paused" {
					t.Fatalf("unexpected pause report: %+v", report)
				}
				return
			case <-time.After(5 * time.Second):
				t.Fatal("paused run did not stop")
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("run did not reach observation state")
}

func TestRunSwitchesHarnessAfterFailedAttempt(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	first := harness.NewLocal(context.Background(), harness.Config{Name: "first", Command: "sh", Args: []string{"-c", "read request; echo failed; exit 1"}, TimeoutSeconds: 10})
	second := harness.NewLocal(context.Background(), harness.Config{Name: "second", Command: "sh", Args: []string{"-c", "read request; echo done; exit 0"}, TimeoutSeconds: 10})
	usedFactory := false
	e.AdapterFactory = func() harness.Adapter {
		usedFactory = true
		return second
	}
	report, err := e.Run(context.Background(), "switch harness", first)
	if err != nil {
		t.Fatal(err)
	}
	if !usedFactory || report.State != "COMPLETE" || report.Attempts != 2 {
		t.Fatalf("harness switch did not complete: %+v", report)
	}
	j, err := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	events, err := j.Replay(report.RunID)
	if err != nil {
		t.Fatal(err)
	}
	foundSwitch := false
	for _, event := range events {
		if event.Type == "HARNESS_SWITCHED" {
			foundSwitch = true
		}
	}
	if !foundSwitch {
		t.Fatal("expected durable harness switch event")
	}
}

func TestRunStopsAfterBoundedRepeatedFailures(t *testing.T) {
	root := t.TempDir()
	if _, err := state.New(root).Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Limits.MaxAttempts = 2
	failing := func(name string) harness.Adapter {
		return harness.NewLocal(context.Background(), harness.Config{Name: name, Command: "sh", Args: []string{"-c", "read request; echo failed; exit 1"}, TimeoutSeconds: 10})
	}
	e.AdapterFactory = func() harness.Adapter { return failing("retry") }
	report, err := e.Run(context.Background(), "bounded failure", failing("first"))
	if err != nil {
		t.Fatal(err)
	}
	if report.State != "STOPPED" || report.Attempts != 2 {
		t.Fatalf("run did not stop at retry limit: %+v", report)
	}
	if report.Error != "completion was not verified" {
		t.Fatalf("unexpected bounded failure error: %+v", report)
	}
}

func TestRunBlocksOnContextBudgetExhaustion(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(strings.Repeat("context", 20)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := state.New(root).Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Limits.MaxContextTokens = 1
	report, err := e.Run(context.Background(), "budget", nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.State != "BLOCKED" || !strings.Contains(report.Error, "context budget exceeded") {
		t.Fatalf("unexpected context budget report: %+v", report)
	}
}

func TestHighRiskRunRequiresApprovalBeforeHarnessDispatch(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	adapter := harness.NewLocal(context.Background(), harness.Config{Name: "protected", Command: "sh", Args: []string{"-c", "read request; echo done"}, TimeoutSeconds: 10})
	blocked, err := e.Run(context.Background(), "deploy to production", adapter)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.State != "BLOCKED" || blocked.NextAction.Allowed {
		t.Fatalf("high-risk work was dispatched without approval: %+v", blocked)
	}
	approval := domain.Approval{ID: "APR-HIGH", RunID: blocked.RunID, Action: "CONTINUE", ApprovedBy: "test", At: time.Now().UTC()}
	if err := e.Store.Write("approvals/"+approval.ID+".json", approval); err != nil {
		t.Fatal(err)
	}
	resumed, err := e.Continue(context.Background(), adapter)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.State != "COMPLETE" {
		t.Fatalf("approved high-risk work did not resume: %+v", resumed)
	}
}

func TestEndToEndFixtureRunsDeterministicValidation(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":          "module fixture\n\ngo 1.23\n",
		"fixture.go":      "package fixture\n\nfunc Value() int { return 42 }\n",
		"fixture_test.go": "package fixture\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) { if Value() != 42 { t.Fatal(Value()) } }\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	caps := []domain.Capability{{Kind: "language", Name: "go"}, {Kind: "test", Name: "go"}, {Kind: "build", Name: "go-build"}}
	if _, err := state.New(root).Init("fixture", caps); err != nil {
		t.Fatal(err)
	}
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	adapter := harness.NewLocal(context.Background(), harness.Config{Name: "fixture-process", Command: "sh", Args: []string{"-c", "read request; echo '[command_completed] fixture'; echo done"}, TimeoutSeconds: 10})
	report, err := e.Run(context.Background(), "verify fixture", adapter)
	if err != nil {
		t.Fatal(err)
	}
	if report.State != "COMPLETE" || len(report.Validations) != 2 {
		t.Fatalf("fixture did not complete with validation: %+v", report)
	}
	for _, result := range report.Validations {
		if !result.Passed {
			t.Fatalf("validation failed: %+v", result)
		}
	}
}

func TestRunUsesConfiguredProviderAdapterAndPersistsSessionIdentity(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	providerCommand := filepath.Join(root, "codex-fixture.sh")
	providerScript := `#!/bin/sh
if [ "$1" = "--version" ]; then printf 'codex-fixture 1.0\n'; exit 0; fi
printf '%s\n' '{"type":"thread.started","thread_id":"thread-engine-1"}'
printf '%s\n' '{"type":"item.completed","item":{"type":"command_execution","command":"go test ./...","exit_code":0}}'
printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"implemented fixture"}}'
printf '%s\n' '{"type":"turn.completed","usage":{"input_tokens":4,"output_tokens":3}}'
`
	if err := os.WriteFile(providerCommand, []byte(providerScript), 0700); err != nil {
		t.Fatal(err)
	}
	config, err := json.Marshal(harness.Config{Provider: "codex", Name: "codex", Command: providerCommand, TimeoutSeconds: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".keystone", "harness.json"), config, 0600); err != nil {
		t.Fatal(err)
	}
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := e.Run(context.Background(), "verify configured provider", nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.State != "COMPLETE" || report.HarnessID != "codex" || report.HarnessSessionID != "thread-engine-1" {
		t.Fatalf("provider run did not complete with durable identity: %+v", report)
	}
	var snapshot state.Snapshot
	if err := state.New(root).Read("state.json", &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.HarnessID != report.HarnessID || snapshot.HarnessSessionID != report.HarnessSessionID {
		t.Fatalf("snapshot lost provider identity: %+v", snapshot)
	}
	var checkpoint domain.Checkpoint
	if err := state.New(root).Read("checkpoints/CP-"+report.RunID+"-1.json", &checkpoint); err != nil {
		t.Fatal(err)
	}
	if checkpoint.HarnessSessionID != report.HarnessSessionID {
		t.Fatalf("checkpoint lost provider session: %+v", checkpoint)
	}
}
