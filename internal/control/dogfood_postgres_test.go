package control

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/harness"
	"github.com/anonyxhappie/keystone/internal/observation"
	"github.com/anonyxhappie/keystone/internal/state"
)

// fakePostgresHarness simulates a harness that:
// Turn 1: runs without fixing postgres -> fixture test fails with postgres 5432 connection refused
// Turn 2: receives recovery prompt with postgres instructions -> fixes test/service -> validation passes!
type fakePostgresHarness struct {
	root            string
	turn            int
	receivedPrompts []domain.Prompt
	sessionID       string
}

func (f *fakePostgresHarness) Discover() error { return nil }
func (f *fakePostgresHarness) Capabilities() []string {
	return []string{"start", "send", "observe", "interrupt", "resume", "result", "stdout-observation"}
}
func (f *fakePostgresHarness) HarnessID() string { return "antigravity" }
func (f *fakePostgresHarness) SessionID() string { return f.sessionID }
func (f *fakePostgresHarness) Start(packet domain.WorkPacket) (string, error) {
	f.turn++
	f.sessionID = "sess-antigravity-123"
	return f.sessionID, nil
}
func (f *fakePostgresHarness) DispatchPrompt(prompt domain.Prompt) (string, error) {
	f.turn++
	f.receivedPrompts = append(f.receivedPrompts, prompt)
	if f.sessionID == "" {
		f.sessionID = "sess-antigravity-123"
	}

	if f.turn == 2 {
		// Turn 2: Harness received prompt with Postgres recovery guidance.
		// Harness resolves the issue by writing passing test.
		passingTest := "package fixture\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) {}\n"
		_ = os.WriteFile(filepath.Join(f.root, "fixture_test.go"), []byte(passingTest), 0600)
	}

	return f.sessionID, nil
}
func (f *fakePostgresHarness) Send(s string) error { return nil }
func (f *fakePostgresHarness) Resume(checkpoint domain.Checkpoint) error { return nil }
func (f *fakePostgresHarness) Interrupt() error { return nil }
func (f *fakePostgresHarness) Result() (domain.Status, error) {
	return domain.StatusCompleted, nil
}
func (f *fakePostgresHarness) Observe() ([]domain.Observation, error) {
	if f.turn == 1 {
		return []domain.Observation{
			{
				ID:      "OBS-1",
				Type:    "TOOL_FINISHED",
				Summary: "Ran initial inspection but postgres is not running",
			},
		}, nil
	}
	// Turn 2: recovered
	return []domain.Observation{
		{
			ID:      "OBS-2",
			Type:    "COMMAND_FINISHED",
			Summary: "docker compose up -d postgres && pg_isready -h localhost -p 5432",
		},
		{
			ID:      "OBS-3",
			Type:    "COMPLETION_CLAIM",
			Summary: "PostgreSQL recovered and all validations pass",
		},
	}, nil
}

func TestPostgreSQLEnvironmentBlockerRecoveryLoop(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)

	// Turn 1 setup: fixture_test.go fails with postgres connection refused
	postgresFailTest := "package fixture\n\nimport (\n\t\"testing\"\n)\n\nfunc TestValue(t *testing.T) {\n\tt.Fatal(\"dial tcp 127.0.0.1:5432: connect: connection refused\")\n}\n"
	if err := os.WriteFile(filepath.Join(root, "fixture_test.go"), []byte(postgresFailTest), 0600); err != nil {
		t.Fatal(err)
	}

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Limits.MaxAttempts = 4

	fakeHarness := &fakePostgresHarness{root: root}

	report, err := e.Run(context.Background(), "Run Losal migration and tests", fakeHarness)
	if err != nil {
		t.Fatal(err)
	}

	if report.State != "COMPLETE" {
		t.Fatalf("expected run to reach COMPLETE after recovery, got state: %s (attempts: %d)", report.State, report.Attempts)
	}
	if report.Attempts != 2 {
		t.Fatalf("expected exactly 2 attempts (failure -> recovery), got: %d", report.Attempts)
	}

	// Verify Turn 2 prompt content had postgres diagnosis and recovery instructions
	if len(fakeHarness.receivedPrompts) < 2 {
		t.Fatalf("expected at least 2 dispatched prompts, got %d", len(fakeHarness.receivedPrompts))
	}
	turn2Prompt := fakeHarness.receivedPrompts[1]
	if !strings.Contains(turn2Prompt.Content, "PostgreSQL is unreachable at localhost:5432") {
		t.Fatalf("turn 2 prompt missing PostgreSQL diagnosis: %s", turn2Prompt.Content)
	}
	if !strings.Contains(turn2Prompt.Content, "Diagnose and recover the local database") {
		t.Fatalf("turn 2 prompt missing recovery instruction: %s", turn2Prompt.Content)
	}

	// Read journal events
	j, err := observation.Open(filepath.Join(root, state.Dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	events, err := j.Replay(report.RunID)
	if err != nil {
		t.Fatal(err)
	}

	var hasPromptGenerated bool
	var hasPromptDispatched bool
	var hasHarnessTurnStarted bool
	var hasContextCompiled bool
	var hasSessionResumed bool
	var hasFailureDiagnosis bool
	var hasSwitchedAntigravity bool

	for _, ev := range events {
		switch ev.Type {
		case "PromptGenerated":
			hasPromptGenerated = true
		case "PromptDispatched":
			hasPromptDispatched = true
		case "HarnessTurnStarted":
			hasHarnessTurnStarted = true
		case "ContextCompiled":
			hasContextCompiled = true
		case "FAILURE_DIAGNOSIS":
			hasFailureDiagnosis = true
		case "HARNESS_SESSION_RESUMED":
			hasSessionResumed = true
		case "HARNESS_SWITCHED":
			from := ev.Payload["from"]
			to := ev.Payload["to"]
			if from == "antigravity" && to == "antigravity" {
				hasSwitchedAntigravity = true
			}
		}
	}

	if !hasPromptGenerated {
		t.Fatalf("expected PromptGenerated event")
	}
	if !hasPromptDispatched {
		t.Fatalf("expected PromptDispatched event")
	}
	if !hasHarnessTurnStarted {
		t.Fatalf("expected HarnessTurnStarted event")
	}
	if !hasContextCompiled {
		t.Fatalf("expected ContextCompiled event")
	}
	if !hasFailureDiagnosis {
		t.Fatalf("expected FAILURE_DIAGNOSIS event")
	}
	if !hasSessionResumed {
		t.Fatalf("expected HARNESS_SESSION_RESUMED event on turn 2")
	}
	if hasSwitchedAntigravity {
		t.Fatalf("HARNESS_SWITCHED was emitted for antigravity -> antigravity! This violates harness continuity.")
	}

	// Verify that prompt files were durably written to .keystone/prompts/
	if report.LastPromptID == "" {
		t.Fatalf("expected non-empty LastPromptID in report")
	}
	var storedPrompt domain.Prompt
	if err := state.New(root).Read("prompts/"+report.LastPromptID+".json", &storedPrompt); err != nil {
		t.Fatalf("failed to read durable prompt: %v", err)
	}
	if storedPrompt.ID != report.LastPromptID {
		t.Fatalf("stored prompt ID mismatch: %s != %s", storedPrompt.ID, report.LastPromptID)
	}
	if storedPrompt.CreatedAt.IsZero() {
		t.Fatalf("stored prompt has zero timestamp!")
	}
}

func TestNoProgressLoopDetectionStopsRun(t *testing.T) {
	root := t.TempDir()
	initRunnableFixture(t, root)

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Limits.MaxAttempts = 4
	e.RequestedHarness = "antigravity" // explicit mode

	// A harness that repeatedly fails without making code changes
	failingHarness := harness.NewLocal(context.Background(), harness.Config{
		Name:           "antigravity",
		Command:        "sh",
		Args:           []string{"-c", "read line; echo failed; exit 1"},
		TimeoutSeconds: 5,
	})

	report, err := e.Run(context.Background(), "do something that repeatedly fails", failingHarness)
	if err != nil {
		t.Fatal(err)
	}

	// The loop detection should catch the repeated failure on attempt 2,
	// and transition to ASK rather than looping all the way to MaxAttempts (4).
	if report.State != "BLOCKED" && report.State != "ASK" {
		t.Fatalf("expected BLOCKED/ASK state from loop detection, got %s", report.State)
	}
	if report.Attempts > 2 {
		t.Fatalf("expected loop detection to stop after attempt 2, but ran %d attempts", report.Attempts)
	}

	// Verify NextAction requires human
	if report.NextAction.Type != "ASK" {
		t.Fatalf("expected NextAction Type ASK, got %s", report.NextAction.Type)
	}

	// Verify finding recorded
	foundLoopFinding := false
	for _, f := range report.Findings {
		if strings.Contains(f.Explanation, "loop") {
			foundLoopFinding = true
		}
	}
	if !foundLoopFinding {
		t.Fatalf("expected loop finding in report: %+v", report.Findings)
	}
}
