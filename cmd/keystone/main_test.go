package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/harness"
	"github.com/anonyxhappie/keystone/v2/internal/runtime"
	"github.com/anonyxhappie/keystone/v2/internal/state"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestCLIAuditAll13Commands(t *testing.T) {
	root := t.TempDir()

	// Create minimal Go files for validation in fixture
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module clifixture\n\ngo 1.23\n"), 0600)
	_ = os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\nfunc Add(a, b int) int { return a + b }\n"), 0600)
	_ = os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) { if Add(1, 2) != 3 { t.Fatal(\"fail\") } }\n"), 0600)

	// 1. keystone version
	out := captureOutput(func() {
		fmt.Println(version)
	})
	if strings.TrimSpace(out) != "2.1.0" {
		t.Fatalf("unexpected version output: %q", out)
	}

	// 2. keystone init
	out = captureOutput(func() {
		runInit(root)
	})
	if !strings.Contains(out, "schemaVersion") || !strings.Contains(out, "capabilities") {
		t.Fatalf("init output missing schemaVersion/capabilities: %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, state.Dir, "project.json")); err != nil {
		t.Fatalf("project.json not created: %v", err)
	}

	// 3. keystone doctor
	out = captureOutput(func() {
		runDoctor(root)
	})
	if !strings.Contains(out, "harnesses") || !strings.Contains(out, "2.1.0") {
		t.Fatalf("doctor output unexpected: %q", out)
	}

	// 4. keystone ask
	out = captureOutput(func() {
		runAsk(root, []string{"implement", "a", "feature"})
	})
	if !strings.Contains(out, "implement a feature") {
		t.Fatalf("ask output missing objective: %q", out)
	}

	// Configure a test harness in .keystone/harness.json
	harnessConfig := harness.Config{
		Name:           "test-harness",
		Command:        "sh",
		Args:           []string{"-c", "read req; echo '[command_completed] go test ./...'; echo done; exit 0"},
		TimeoutSeconds: 30,
	}
	hConfigBytes, _ := json.Marshal(harnessConfig)
	_ = os.WriteFile(filepath.Join(root, state.Dir, "harness.json"), hConfigBytes, 0600)

	// 5. keystone validate
	out = captureOutput(func() {
		runValidate(root)
	})
	if !strings.Contains(out, "\"passed\": true") || !strings.Contains(out, "go-build") {
		t.Fatalf("validate output unexpected: %q", out)
	}

	// 6. keystone run
	out = captureOutput(func() {
		runRun(root, []string{"add", "feature", "safely"})
	})
	if !strings.Contains(out, "\"state\": \"COMPLETE\"") {
		t.Fatalf("run did not complete: %q", out)
	}

	var runReport struct {
		RunID       string `json:"runId"`
		WorkOrderID string `json:"workOrderId"`
		State       string `json:"state"`
	}
	_ = json.Unmarshal([]byte(out), &runReport)
	if runReport.RunID == "" {
		t.Fatalf("run report missing RunID: %s", out)
	}

	// 7. keystone status
	out = captureOutput(func() {
		runStatus(root)
	})
	if !strings.Contains(out, runReport.RunID) || !strings.Contains(out, "COMPLETE") {
		t.Fatalf("status output unexpected: %q", out)
	}

	// 8. keystone review
	out = captureOutput(func() {
		runReview(root)
	})
	if !strings.Contains(out, "COMPLETE") {
		t.Fatalf("review output unexpected: %q", out)
	}

	// 9. keystone replay <run-id>
	out = captureOutput(func() {
		runReplay(root, []string{runReport.RunID})
	})
	if !strings.Contains(out, runReport.RunID) || !strings.Contains(out, "COMPLETE") {
		t.Fatalf("replay output unexpected: %q", out)
	}

	// 10. keystone stop
	m := runtime.New()
	_ = m.TransitionTo(runtime.Understand, "test")
	_ = m.TransitionTo(runtime.Assess, "test")
	snap := state.Snapshot{
		SchemaVersion: "2",
		Lifecycle:     string(m.State),
		RunID:         runReport.RunID,
		WorkOrderID:   runReport.WorkOrderID,
		Machine:       m,
		UpdatedAt:     time.Now().UTC(),
	}
	_ = state.New(root).SaveSnapshot(snap)

	out = captureOutput(func() {
		runStop(root)
	})
	if !strings.Contains(out, "STOPPED") {
		t.Fatalf("stop output unexpected: %q", out)
	}

	// 11. keystone pause
	m2 := runtime.New()
	_ = m2.TransitionTo(runtime.Understand, "test")
	_ = m2.TransitionTo(runtime.Assess, "test")
	snap.Machine = m2
	snap.Lifecycle = string(m2.State)
	_ = state.New(root).SaveSnapshot(snap)
	out = captureOutput(func() {
		runPause(root)
	})
	if strings.TrimSpace(out) != "paused" {
		t.Fatalf("pause output unexpected: %q", out)
	}

	// 12. keystone approve
	m3 := runtime.New()
	m3.State = runtime.Ask
	snap.Machine = m3
	snap.Lifecycle = string(m3.State)
	_ = state.New(root).SaveSnapshot(snap)
	out = captureOutput(func() {
		runApprove(root, []string{"CONTINUE"})
	})
	if !strings.Contains(out, "approval recorded") {
		t.Fatalf("approve output unexpected: %q", out)
	}

	// 13. keystone continue
	out = captureOutput(func() {
		runContinue(root)
	})
	if !strings.Contains(out, "COMPLETE") {
		t.Fatalf("continue output unexpected: %q", out)
	}
}
