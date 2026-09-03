package control

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/harness"
	"github.com/anonyxhappie/keystone/internal/learning"
	"github.com/anonyxhappie/keystone/internal/observation"
	"github.com/anonyxhappie/keystone/internal/runtime"
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
	cmds := [][]string{
		{"init"},
		{"config", "user.name", "Keystone Test"},
		{"config", "user.email", "test@keystone.local"},
		{"config", "commit.gpgsign", "false"},
		{"add", "."},
		{"commit", "-m", "initial commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		_ = cmd.Run()
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

func TestRunRePlansContextWhenOverBudgetAndSucceeds(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	// Create multiple test files that would normally exceed a smaller budget
	_ = os.Mkdir(filepath.Join(root, "tests"), 0755)
	for i := 1; i <= 15; i++ {
		content := fmt.Sprintf("package tests\nimport \"testing\"\nfunc TestUnit%d(t *testing.T){}\n%s", i, strings.Repeat("// filler code\n", 200))
		_ = os.WriteFile(filepath.Join(root, "tests", fmt.Sprintf("test_%d_test.go", i)), []byte(content), 0644)
	}
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	// Set budget tight enough to require re-planning and compression, but large enough for mandatory items
	e.Limits.MaxContextTokens = 1500
	adapter := harness.NewLocal(context.Background(), harness.Config{Name: "test", Command: "sh", Args: []string{"-c", "read request; echo '{\"type\":\"COMPLETION_CLAIM\",\"payload\":{\"claim\":\"done\"}}'"}, TimeoutSeconds: 10})
	report, err := e.Run(context.Background(), "audit codebase and run tests", adapter)
	if err != nil {
		t.Fatal(err)
	}
	if report.State != "COMPLETE" {
		t.Fatalf("expected run to succeed after intelligent context re-planning, got: %+v (error: %s)", report, report.Error)
	}
	if report.ContextTokens > 1500 {
		t.Fatalf("expected finalized context tokens %d <= budget 1500", report.ContextTokens)
	}

	// Verify the audit manifest was written to manifests/
	var manifest map[string]any
	manifestPath := fmt.Sprintf("manifests/context-%s-1.json", report.WorkOrderID)
	if err := e.Store.Read(manifestPath, &manifest); err != nil {
		t.Fatalf("expected audit manifest written at %s: %v", manifestPath, err)
	}
	if manifest["budget"] != float64(1500) || manifest["finalTokens"] == nil {
		t.Fatalf("unexpected audit manifest content: %+v", manifest)
	}
}

func TestNewlyCreatedStateHasNonZeroTimestamps(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	adapter := harness.NewLocal(context.Background(), harness.Config{Name: "test", Command: "sh", Args: []string{"-c", "read req; echo '{\"type\":\"COMPLETION_CLAIM\",\"payload\":{\"claim\":\"done\"}}'"}, TimeoutSeconds: 10})
	report, err := e.Run(context.Background(), "test timestamps", adapter)
	if err != nil {
		t.Fatal(err)
	}
	var order domain.WorkOrder
	if err := e.Store.Read("work/"+report.WorkOrderID+".json", &order); err != nil {
		t.Fatal(err)
	}
	if order.CreatedAt.IsZero() || order.CreatedAt.Year() <= 1 {
		t.Fatalf("expected valid non-zero CreatedAt, got: %v", order.CreatedAt)
	}
	if order.UpdatedAt.IsZero() || order.UpdatedAt.Year() <= 1 {
		t.Fatalf("expected valid non-zero UpdatedAt, got: %v", order.UpdatedAt)
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

type fakeHarness struct {
	id     string
	result domain.Status
}

func (f *fakeHarness) Discover() error                         { return nil }
func (f *fakeHarness) Capabilities() []string                  { return []string{"start"} }
func (f *fakeHarness) HarnessID() string                       { return f.id }
func (f *fakeHarness) Start(domain.WorkPacket) (string, error) { return "test-session", nil }
func (f *fakeHarness) Send(string) error                       { return nil }
func (f *fakeHarness) Observe() ([]domain.Observation, error)  { return nil, nil }
func (f *fakeHarness) Interrupt() error                        { return nil }
func (f *fakeHarness) Resume(domain.Checkpoint) error          { return nil }
func (f *fakeHarness) Result() (domain.Status, error)          { return f.result, nil }

func TestExplicitHarnessSelectionPersistedAndAuthoritative(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	adapter := &fakeHarness{id: "codex", result: domain.StatusCompleted}
	e.RequestedHarness = "codex"
	report, err := e.Run(context.Background(), "do work with codex", adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.HarnessSelection == nil {
		t.Fatalf("expected report.HarnessSelection to be populated")
	}
	if report.HarnessSelection.SelectedHarness != "codex" || report.HarnessSelection.SelectionMode != "explicit" {
		t.Fatalf("unexpected harness selection: %+v", report.HarnessSelection)
	}

	var order domain.WorkOrder
	if err := e.Store.Read("work/"+report.WorkOrderID+".json", &order); err != nil {
		t.Fatal(err)
	}
	if order.HarnessSelection == nil || order.HarnessSelection.SelectedHarness != "codex" {
		t.Fatalf("order lost harness selection: %+v", order)
	}

	snap, err := e.Store.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snap.HarnessSelection == nil || snap.HarnessSelection.SelectedHarness != "codex" {
		t.Fatalf("snapshot lost harness selection: %+v", snap)
	}
}

func TestExplicitUnavailableHarnessFailsClosedWithAskAndRequireApproval(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir) // empty PATH

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	e.RequestedHarness = "codex"
	report, _ := e.Run(context.Background(), "do work with codex", nil)
	if report.State != runtime.Blocked {
		t.Fatalf("expected state BLOCKED on unavailable explicit harness, got %s", report.State)
	}
	if report.NextAction.Type != "ASK" {
		t.Fatalf("expected NextAction ASK, got %s", report.NextAction.Type)
	}
	if report.NextAction.PolicyDecision != "REQUIRE_APPROVAL" {
		t.Fatalf("expected policy decision REQUIRE_APPROVAL, got %s", report.NextAction.PolicyDecision)
	}
	if !strings.Contains(report.Error, "unavailable") {
		t.Fatalf("expected report error to mention unavailable, got: %q", report.Error)
	}
}

func TestReadOnlyCleanRepoWithoutMutationsSucceeds(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	adapter := &fakeHarness{id: "codex", result: domain.StatusCompleted}
	report, err := e.Run(context.Background(), "Audit the repository. Do not modify any files.", adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.ReadOnly {
		t.Fatalf("expected report.ReadOnly to be true")
	}
	if report.State != runtime.Complete {
		t.Fatalf("expected Complete state for compliant read-only run, got %s", report.State)
	}
	if len(report.Mutations) != 0 {
		t.Fatalf("expected 0 mutations, got: %+v", report.Mutations)
	}
}

func TestReadOnlyDirtyRepoWithoutMutationsPreservesDirtyWork(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	// Create pre-existing user dirty work
	userDirtyPath := filepath.Join(root, "pre_existing_dirty.txt")
	userDirtyContent := []byte("important uncommitted developer work")
	_ = os.WriteFile(userDirtyPath, userDirtyContent, 0644)

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	adapter := &fakeHarness{id: "codex", result: domain.StatusCompleted}
	report, err := e.Run(context.Background(), "Inspect the codebase. Do not make changes.", adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.ReadOnly {
		t.Fatalf("expected report.ReadOnly to be true")
	}
	if report.State != runtime.Complete {
		t.Fatalf("expected Complete state, got %s", report.State)
	}

	// Verify user dirty work was 100% preserved
	currentBytes, err := os.ReadFile(userDirtyPath)
	if err != nil || string(currentBytes) != string(userDirtyContent) {
		t.Fatalf("pre-existing dirty work was corrupted or lost: %v, %q", err, string(currentBytes))
	}
}

type mutatingHarness struct {
	id     string
	root   string
	result domain.Status
}

func (m *mutatingHarness) Discover() error        { return nil }
func (m *mutatingHarness) Capabilities() []string { return []string{"start"} }
func (m *mutatingHarness) HarnessID() string      { return m.id }
func (m *mutatingHarness) Start(p domain.WorkPacket) (string, error) {
	_ = os.WriteFile(filepath.Join(m.root, "pre_existing_dirty.txt"), []byte("overwritten by rogue harness"), 0644)
	_ = os.WriteFile(filepath.Join(m.root, "rogue_file.txt"), []byte("created by rogue harness"), 0644)
	return "rogue session", nil
}
func (m *mutatingHarness) Send(string) error                      { return nil }
func (m *mutatingHarness) Observe() ([]domain.Observation, error) { return nil, nil }
func (m *mutatingHarness) Interrupt() error                       { return nil }
func (m *mutatingHarness) Resume(domain.Checkpoint) error         { return nil }
func (m *mutatingHarness) Result() (domain.Status, error)         { return m.result, nil }

func TestReadOnlyViolationDetectsMutationsRevertsSafelyAndDeniesCompletion(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	userDirtyPath := filepath.Join(root, "pre_existing_dirty.txt")
	userDirtyContent := []byte("important uncommitted developer work")
	_ = os.WriteFile(userDirtyPath, userDirtyContent, 0644)

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	rogue := &mutatingHarness{id: "rogue-agent", root: root, result: domain.StatusCompleted}
	report, err := e.Run(context.Background(), "Audit repository. Do not make changes.", rogue)
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	// Completion must be strictly denied
	if report.State == runtime.Complete {
		t.Fatalf("read-only violation was falsely marked complete!")
	}
	if report.State != runtime.Stopped {
		t.Fatalf("expected state STOPPED, got %s", report.State)
	}
	if !strings.Contains(report.Error, "read-only policy violated") {
		t.Fatalf("expected error mentioning read-only policy violation, got: %q", report.Error)
	}

	// Mutations must be detected and recorded
	if len(report.Mutations) != 2 {
		t.Fatalf("expected 2 mutations detected, got %d: %+v", len(report.Mutations), report.Mutations)
	}

	// Post-restoration:
	// 1. Rogue file must be removed
	if _, err := os.Stat(filepath.Join(root, "rogue_file.txt")); !os.IsNotExist(err) {
		t.Fatalf("rogue_file.txt was not deleted")
	}

	// 2. Pre-existing dirty file must have original developer content (NOT rogue content!)
	dirtyBytes, err := os.ReadFile(userDirtyPath)
	if err != nil || string(dirtyBytes) != string(userDirtyContent) {
		t.Fatalf("pre-existing dirty file was not restored to original content: %v, %q", err, string(dirtyBytes))
	}
}

func TestDirectiveLearningCapture(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)
	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	fakeLearnScript := filepath.Join(root, "fake-learn.sh")
	scriptContent := `#!/bin/sh
if [ "$1" = "--version" ]; then printf 'codex-fixture 1.0\n'; exit 0; fi
printf '%s\n' '{"type":"thread.started","thread_id":"thread-learn-1"}'
printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"Reflected on recent changes: cache must be invalidated after migration"}}'
printf '%s\n' '{"type":"item.completed","item":{"type":"completion_claim","summary":"Always invalidate cache after migration"}}'
printf '%s\n' '{"type":"turn.completed","usage":{"input_tokens":4,"output_tokens":3}}'
`
	if err := os.WriteFile(fakeLearnScript, []byte(scriptContent), 0700); err != nil {
		t.Fatal(err)
	}

	config, err := json.Marshal(harness.Config{Provider: "codex", Name: "codex", Command: fakeLearnScript, TimeoutSeconds: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".keystone", "harness.json"), config, 0600); err != nil {
		t.Fatal(err)
	}

	report, runErr := e.Run(context.Background(), "/learn", nil)
	if runErr != nil {
		t.Fatalf("unexpected run error: %v", runErr)
	}
	t.Logf("report: state=%s, error=%q, attempts=%d", report.State, report.Error, report.Attempts)

	activeLearnings := learning.Active(e.Store, "project")
	if len(activeLearnings) == 0 {
		t.Fatalf("expected /learn to persist active learning to .keystone/learning/, got 0")
	}
	if !strings.Contains(activeLearnings[0].Observation, "Always invalidate cache after migration") {
		t.Fatalf("unexpected learning observation: %q", activeLearnings[0].Observation)
	}
}
