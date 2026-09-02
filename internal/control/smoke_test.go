package control

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/harness"
	"github.com/anonyxhappie/keystone/v2/internal/observation"
	"github.com/anonyxhappie/keystone/v2/internal/state"
)

func TestLiveAntigravitySmoke(t *testing.T) {
	if _, err := exec.LookPath("agy"); err != nil {
		t.Skip("agy is not on PATH; skipping live Antigravity smoke")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	root := t.TempDir()
	initRunnableFixture(t, root)

	adapter := harness.NewAntigravityAdapter(ctx, root, harness.Config{
		Command:        "agy",
		TimeoutSeconds: 30,
	})
	if err := adapter.Discover(); err != nil {
		t.Skipf("agy discovery failed: %v", err)
	}

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	report, err := e.Run(ctx, "say hello", adapter)
	if err != nil {
		t.Fatalf("live antigravity run failed: %v", err)
	}

	if report.HarnessID != "antigravity" {
		t.Fatalf("unexpected harness ID: %s", report.HarnessID)
	}
	if report.HarnessSessionID == "" {
		t.Fatal("expected non-empty Antigravity session ID")
	}
	if report.State != "COMPLETE" {
		t.Fatalf("expected COMPLETE report, got: %+v", report)
	}
	if len(report.EvidenceIDs) == 0 {
		t.Fatal("expected evidence recorded for live Antigravity run")
	}

	// Verify replay of the live run
	j, err := observation.Open(root + "/" + state.Dir + "/events.jsonl")
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
	if replayed.State != "COMPLETE" || replayed.HarnessID != "antigravity" {
		t.Fatalf("replay of live run failed: %+v", replayed)
	}
}

func TestLiveCodexFailClosedSmoke(t *testing.T) {
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex is not on PATH; skipping live Codex smoke")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	root := t.TempDir()
	initRunnableFixture(t, root)

	adapter := harness.NewCodexAdapter(ctx, root, harness.Config{
		Command:        "codex",
		TimeoutSeconds: 20,
	})
	if err := adapter.Discover(); err != nil {
		t.Skipf("codex discovery failed: %v", err)
	}

	e, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Limits.MaxAttempts = 1

	report, err := e.Run(ctx, "verify harmless live run", adapter)
	if err != nil {
		t.Fatalf("unexpected error running codex: %v", err)
	}

	// Because the local Codex environment has reached its OpenAI usage limit,
	// the provider exits immediately on startup.
	// Keystone must fail closed safely, capturing the harness ID, logging the error,
	// and transitioning to BLOCKED with requires-approval next action.
	if report.State == "COMPLETE" {
		t.Fatal("Codex must not be granted COMPLETE when external provider returns usage limit error")
	}
	if report.HarnessID != "codex" {
		t.Fatalf("expected harness ID 'codex', got: %s", report.HarnessID)
	}
	if report.Error == "" {
		t.Fatal("expected error diagnostic in report")
	}
	if report.NextAction.Allowed {
		t.Fatal("expected run to fail closed (Allowed == false)")
	}
}
