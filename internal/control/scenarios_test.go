package control

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/harness"
	"github.com/anonyxhappie/keystone/v2/internal/observation"
	"github.com/anonyxhappie/keystone/v2/internal/state"
	"github.com/anonyxhappie/keystone/v2/internal/supervisor"
)

func TestResumeRejectsProviderSessionMismatch(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := e.Run(context.Background(), "provider mismatch test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.State != "BLOCKED" {
		t.Fatalf("expected blocked report: %+v", report)
	}

	cp := domain.Checkpoint{
		SchemaVersion:    "2",
		ID:               fmt.Sprintf("CP-%s-1", report.RunID),
		WorkOrderID:      report.WorkOrderID,
		RunID:            report.RunID,
		State:            "ASK",
		HarnessID:        "provider-a",
		HarnessSessionID: "session-from-provider-a",
		CreatedAt:        time.Now().UTC(),
	}
	if err := e.Store.Write("checkpoints/"+cp.ID+".json", cp); err != nil {
		t.Fatal(err)
	}
	snap, _ := e.Store.LoadSnapshot()
	snap.CheckpointID = cp.ID
	snap.HarnessID = "provider-a"
	snap.HarnessSessionID = "session-from-provider-a"
	snap.Lifecycle = "ASK"
	_ = e.Store.SaveSnapshot(snap)

	approval := domain.Approval{ID: "APR-MISMATCH", RunID: report.RunID, Action: "CONTINUE", ApprovedBy: "test", At: time.Now().UTC()}
	_ = e.Store.Write("approvals/"+approval.ID+".json", approval)

	adapterB := harness.NewLocal(context.Background(), harness.Config{
		Name:    "provider-b",
		Command: "sh",
		Args:    []string{"-c", "read req; echo done; exit 0"},
	})
	resumed, err := e.Continue(context.Background(), adapterB)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.State != "COMPLETE" {
		t.Fatalf("resumed run did not complete: %+v", resumed)
	}

	j, err := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	events, err := j.Replay(report.RunID)
	if err != nil {
		t.Fatal(err)
	}
	foundMismatch := false
	for _, event := range events {
		if event.Type == "RESUME_HARNESS_MISMATCH" {
			foundMismatch = true
			if event.Payload["checkpointHarnessId"] != "provider-a" || event.Payload["configuredHarnessId"] != "provider-b" {
				t.Fatalf("unexpected mismatch event payload: %+v", event.Payload)
			}
		}
	}
	if !foundMismatch {
		t.Fatal("expected RESUME_HARNESS_MISMATCH event when resuming across different harnesses")
	}
}

func TestFalseCompletionClaimFailsVerification(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":          "module fixture\n\ngo 1.23\n",
		"fixture.go":      "package fixture\n\nfunc Value() int { return 0 }\n",
		"fixture_test.go": "package fixture\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) { if Value() != 42 { t.Fatal(\"expected 42\") } }\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	caps := []domain.Capability{{Kind: "language", Name: "go"}, {Kind: "test", Name: "go"}}
	if _, err := state.New(root).Init("fixture", caps); err != nil {
		t.Fatal(err)
	}

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Limits.MaxAttempts = 1

	adapter := harness.NewLocal(context.Background(), harness.Config{
		Name:    "deceptive-harness",
		Command: "sh",
		Args:    []string{"-c", "read req; echo 'everything completed successfully and done'; exit 0"},
	})

	report, err := e.Run(context.Background(), "falsely claimed work", adapter)
	if err != nil {
		t.Fatal(err)
	}

	if report.State == "COMPLETE" {
		t.Fatalf("completion was granted despite failed validation: %+v", report)
	}
	if report.State != "STOPPED" {
		t.Fatalf("expected STOPPED state, got: %s", report.State)
	}

	foundUnverified := false
	for _, f := range report.Findings {
		if f.Type == supervisor.UnsupportedCompletion {
			foundUnverified = true
		}
	}
	if !foundUnverified {
		t.Fatalf("expected UNVERIFIED_COMPLETION finding, got: %+v", report.Findings)
	}
}

func TestReplayFailsOnCorruptedEventHistory(t *testing.T) {
	root := t.TempDir()
	if _, err := state.New(root).Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	eventPath := filepath.Join(root, state.Dir, "events.jsonl")
	if err := os.WriteFile(eventPath, []byte("NOT_VALID_JSON\n"), 0600); err != nil {
		t.Fatal(err)
	}
	j, err := observation.Open(eventPath)
	if err == nil {
		_, err = j.Replay("RUN-1")
	}
	if err == nil {
		t.Fatal("expected error opening or replaying corrupted event journal")
	}
	if !strings.Contains(err.Error(), "journal line 1") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestMultiHarnessSwitchingWithStatePreservation(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Limits.MaxAttempts = 1

	adapterA := harness.NewLocal(context.Background(), harness.Config{
		Name:           "harness-alpha",
		Command:        "sh",
		Args:           []string{"-c", "read req; echo '[command_completed] step1'; sleep 10"},
		TimeoutSeconds: 15,
	})

	result := make(chan Report, 1)
	go func() {
		rep, _ := e.Run(context.Background(), "multi-harness workflow", adapterA)
		result <- rep
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snap, err := state.New(root).LoadSnapshot()
		if err == nil && snap.Lifecycle == "OBSERVE" {
			_ = state.New(root).Write("control/pause.json", map[string]string{"runId": snap.RunID})
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	reportA := <-result
	if reportA.Error != "run paused" {
		t.Fatalf("expected paused run: %+v", reportA)
	}

	adapterB := harness.NewLocal(context.Background(), harness.Config{
		Name:           "harness-beta",
		Command:        "sh",
		Args:           []string{"-c", "read req; echo '[command_completed] step2'; echo done; exit 0"},
		TimeoutSeconds: 10,
	})

	_ = os.Remove(filepath.Join(root, state.Dir, "control", "pause.json"))
	snap, _ := state.New(root).LoadSnapshot()
	snap.Paused = false
	_ = state.New(root).SaveSnapshot(snap)

	reportB, err := e.Continue(context.Background(), adapterB)
	if err != nil {
		t.Fatal(err)
	}
	if reportB.State != "COMPLETE" {
		t.Fatalf("resumed run did not complete: %+v", reportB)
	}

	if reportB.WorkOrderID != reportA.WorkOrderID {
		t.Fatalf("work order ID lost: %s != %s", reportB.WorkOrderID, reportA.WorkOrderID)
	}
	if reportB.ContextManifest == "" {
		t.Fatal("context manifest was lost")
	}
	if len(reportB.EvidenceIDs) == 0 {
		t.Fatal("evidence IDs lost")
	}
	j, _ := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	events, err := j.Replay(reportB.RunID)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := Replay(reportB.RunID, events)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.State != "COMPLETE" || replayed.WorkOrderID != reportA.WorkOrderID {
		t.Fatalf("replay failed: %+v", replayed)
	}
}

type DeterministicFakeHarness struct {
	ID           string
	ScriptedType string
	Root         string
	ObsCount     int
}

func (f *DeterministicFakeHarness) Discover() error   { return nil }
func (f *DeterministicFakeHarness) HarnessID() string { return f.ID }
func (f *DeterministicFakeHarness) Capabilities() []string {
	return []string{"start", "send", "observe", "interrupt", "resume", "result", "stdout-observation"}
}
func (f *DeterministicFakeHarness) Start(p domain.WorkPacket) (string, error) {
	return "fake-session-1", nil
}
func (f *DeterministicFakeHarness) Send(prompt string) error          { return nil }
func (f *DeterministicFakeHarness) Resume(cp domain.Checkpoint) error { return nil }
func (f *DeterministicFakeHarness) Interrupt() error                  { return nil }
func (f *DeterministicFakeHarness) Stop() error                       { return nil }
func (f *DeterministicFakeHarness) Result() (domain.Status, error) {
	switch f.ScriptedType {
	case "HARNESS_FAILURE", "REPEATED_FAILURE":
		return domain.StatusFailed, fmt.Errorf("simulated harness crash")
	default:
		return domain.StatusCompleted, nil
	}
}

func (f *DeterministicFakeHarness) Observe() ([]domain.Observation, error) {
	if f.ObsCount > 0 {
		return nil, nil
	}
	f.ObsCount++
	now := time.Now().UTC()
	switch f.ScriptedType {
	case "SUCCESS":
		return []domain.Observation{
			{ID: "OBS-1", Type: "COMMAND_COMPLETED", Summary: "go test ./...", Timestamp: now},
			{ID: "OBS-2", Type: "COMPLETION_CLAIM", Summary: "implemented change", Timestamp: now},
		}, nil
	case "FALSE_COMPLETION":
		return []domain.Observation{
			{ID: "OBS-FC", Type: "COMPLETION_CLAIM", Summary: "done everything perfectly", Timestamp: now},
		}, nil
	case "REQUIREMENT_DRIFT":
		return []domain.Observation{
			{ID: "OBS-RD", Type: "FILE_CHANGED", Summary: "unrelated/file.txt", Timestamp: now},
			{ID: "OBS-RD2", Type: "COMPLETION_CLAIM", Summary: "done unrelated task", Timestamp: now},
		}, nil
	case "EXCESSIVE_ACTIVITY":
		obs := make([]domain.Observation, 25)
		for i := range obs {
			obs[i] = domain.Observation{ID: fmt.Sprintf("OBS-EX-%d", i), Type: "TOOL_STARTED", Summary: "tool call", Timestamp: now}
		}
		return obs, nil
	case "REPEATED_READS":
		return []domain.Observation{
			{ID: "OBS-R1", Type: "FILE_READ", Summary: "data.txt", Timestamp: now},
			{ID: "OBS-R2", Type: "FILE_READ", Summary: "data.txt", Timestamp: now},
			{ID: "OBS-R3", Type: "COMPLETION_CLAIM", Summary: "done", Timestamp: now},
		}, nil
	default:
		return []domain.Observation{
			{ID: "OBS-DEF", Type: "MESSAGE_RECEIVED", Summary: "running", Timestamp: now},
		}, nil
	}
}

func TestDeterministicFakeHarnessAllScenarios(t *testing.T) {
	t.Run("ScenarioA_Success", func(t *testing.T) {
		root := t.TempDir()
		initRunnableFixture(t, root)
		e, _ := Open(root)
		h := &DeterministicFakeHarness{ID: "fake", ScriptedType: "SUCCESS", Root: root}
		report, err := e.Run(context.Background(), "success task", h)
		if err != nil || report.State != "COMPLETE" {
			t.Fatalf("scenario A failed: report=%+v err=%v", report, err)
		}
	})

	t.Run("ScenarioB_FalseCompletionClaim", func(t *testing.T) {
		root := t.TempDir()
		initRunnableFixture(t, root)
		_ = os.WriteFile(filepath.Join(root, "fixture.go"), []byte("package fixture\nfunc Value() int { return 0 }\n"), 0600)
		e, _ := Open(root)
		e.Limits.MaxAttempts = 1
		h := &DeterministicFakeHarness{ID: "fake", ScriptedType: "FALSE_COMPLETION", Root: root}
		report, _ := e.Run(context.Background(), "false claim task", h)
		if report.State == "COMPLETE" {
			t.Fatalf("scenario B failed: false claim was accepted as COMPLETE")
		}
		foundUnverified := false
		for _, f := range report.Findings {
			if f.Type == supervisor.UnsupportedCompletion {
				foundUnverified = true
			}
		}
		if !foundUnverified {
			t.Fatalf("scenario B failed: expected UNVERIFIED_COMPLETION finding: %+v", report.Findings)
		}
	})

	t.Run("ScenarioC_RepeatedFailure", func(t *testing.T) {
		root := t.TempDir()
		initRunnableFixture(t, root)
		e, _ := Open(root)
		e.Limits.MaxAttempts = 2
		e.AdapterFactory = func() harness.Adapter {
			return &DeterministicFakeHarness{ID: "fake-retry", ScriptedType: "REPEATED_FAILURE", Root: root}
		}
		h := &DeterministicFakeHarness{ID: "fake-first", ScriptedType: "REPEATED_FAILURE", Root: root}
		report, _ := e.Run(context.Background(), "repeated fail task", h)
		if report.State != "STOPPED" || report.Attempts != 2 {
			t.Fatalf("scenario C failed: expected STOPPED after 2 attempts: %+v", report)
		}
	})

	t.Run("ScenarioD_ExcessiveActivity", func(t *testing.T) {
		root := t.TempDir()
		initRunnableFixture(t, root)
		e, _ := Open(root)
		e.Limits.MaxToolCalls = 10
		h := &DeterministicFakeHarness{ID: "fake", ScriptedType: "EXCESSIVE_ACTIVITY", Root: root}
		report, _ := e.Run(context.Background(), "excessive task", h)
		if report.State != "STOPPED" || !strings.Contains(report.Error, "activity limit") {
			t.Fatalf("scenario D failed: expected STOPPED on activity limit: %+v", report)
		}
	})

	t.Run("ScenarioE_ApprovalRequirement", func(t *testing.T) {
		root := t.TempDir()
		initRunnableFixture(t, root)
		e, _ := Open(root)
		h := &DeterministicFakeHarness{ID: "fake", ScriptedType: "SUCCESS", Root: root}
		report, _ := e.Run(context.Background(), "destroy production database", h)
		if report.State != "BLOCKED" || report.NextAction.Allowed {
			t.Fatalf("scenario E failed: high risk request was not blocked: %+v", report)
		}
	})

	t.Run("ScenarioF_HarnessFailure", func(t *testing.T) {
		root := t.TempDir()
		initRunnableFixture(t, root)
		e, _ := Open(root)
		e.Limits.MaxAttempts = 1
		h := &DeterministicFakeHarness{ID: "fake", ScriptedType: "HARNESS_FAILURE", Root: root}
		report, _ := e.Run(context.Background(), "failing harness task", h)
		if report.State != "STOPPED" {
			t.Fatalf("scenario F failed: expected STOPPED: %+v", report)
		}
	})

	t.Run("ScenarioG_RepeatedReads", func(t *testing.T) {
		root := t.TempDir()
		initRunnableFixture(t, root)
		e, _ := Open(root)
		h := &DeterministicFakeHarness{ID: "fake", ScriptedType: "REPEATED_READS", Root: root}
		report, _ := e.Run(context.Background(), "repeated reads task", h)
		foundEfficiency := false
		for _, f := range report.Findings {
			if f.Type == supervisor.InefficientActivity {
				foundEfficiency = true
			}
		}
		if !foundEfficiency {
			t.Fatalf("scenario G failed: expected INEFFICIENT_ACTIVITY finding: %+v", report.Findings)
		}
	})
}
