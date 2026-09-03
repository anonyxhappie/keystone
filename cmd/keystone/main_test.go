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

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/harness"
	"github.com/anonyxhappie/keystone/internal/runtime"
	"github.com/anonyxhappie/keystone/internal/state"
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

func writeFixtureHarness(t *testing.T, root string) {
	t.Helper()
	harnessCfg := harness.Config{
		Name:           "test-harness",
		Command:        "sh",
		Args:           []string{"-c", "read req; echo '[command_completed] go test ./...'; echo done; exit 0"},
		TimeoutSeconds: 30,
	}
	b, err := json.Marshal(harnessCfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, state.Dir, "harness.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
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
	if strings.TrimSpace(out) != version {
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
	if !strings.Contains(out, "harnesses") || !strings.Contains(out, version) {
		t.Fatalf("doctor output unexpected: %q", out)
	}

	// 4. keystone ask
	out = captureOutput(func() {
		runAsk(root, []string{"implement", "a", "feature"})
	})
	if !strings.Contains(out, "implement a feature") {
		t.Fatalf("ask output missing objective: %q", out)
	}

	// Configure a deterministic test harness in .keystone/harness.json.
	writeFixtureHarness(t, root)

	// 5. keystone validate
	out = captureOutput(func() {
		runValidate(root)
	})
	if !strings.Contains(out, "\"passed\": true") || !strings.Contains(out, "go-build") {
		t.Fatalf("validate output unexpected: %q", out)
	}

	// 6. keystone run (with --json for programmatic output)
	out = captureOutput(func() {
		runRun(root, []string{"--json", "add", "feature", "safely"})
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

func TestParseRunArgs(t *testing.T) {
	cases := []struct {
		args        []string
		wantHarness string
		wantJSON    bool
		wantReq     string
		wantErr     bool
	}{
		{
			args:        []string{"--harness", "codex", "inspect", "the", "codebase"},
			wantHarness: "codex",
			wantJSON:    false,
			wantReq:     "inspect the codebase",
			wantErr:     false,
		},
		{
			args:        []string{"--harness", "antigravity", "--json", "inspect", "the", "codebase"},
			wantHarness: "antigravity",
			wantJSON:    true,
			wantReq:     "inspect the codebase",
			wantErr:     false,
		},
		{
			args:        []string{"--harness=auto", "audit", "only"},
			wantHarness: "auto",
			wantJSON:    false,
			wantReq:     "audit only",
			wantErr:     false,
		},
		{
			args:    []string{"--harness", "invalid-harness", "audit"},
			wantErr: true,
		},
		{
			args:        []string{"plain", "request", "without", "flag"},
			wantHarness: "",
			wantJSON:    false,
			wantReq:     "plain request without flag",
			wantErr:     false,
		},
	}

	for _, c := range cases {
		h, jm, r, err := parseRunArgs(c.args)
		if c.wantErr {
			if err == nil {
				t.Fatalf("expected error for args: %v, got h=%q, r=%q", c.args, h, r)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for args %v: %v", c.args, err)
		}
		if h != c.wantHarness {
			t.Fatalf("args %v: want harness %q, got %q", c.args, c.wantHarness, h)
		}
		if jm != c.wantJSON {
			t.Fatalf("args %v: want json %v, got %v", c.args, c.wantJSON, jm)
		}
		if r != c.wantReq {
			t.Fatalf("args %v: want req %q, got %q", c.args, c.wantReq, r)
		}
	}
}

func TestCLIRichTerminalOutput(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":       "module testrun\n\ngo 1.23\n",
		"main.go":      "package main\n\nfunc main() {}\n",
		"main_test.go": "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := state.New(root)
	if _, err := s.Init("test-project", []domain.Capability{{Kind: "language", Name: "go"}, {Kind: "test", Name: "go"}}); err != nil {
		t.Fatal(err)
	}
	writeFixtureHarness(t, root)

	out := captureOutput(func() {
		runRun(root, []string{"inspect", "the", "repository"})
	})

	if !strings.Contains(out, "RUN COMPLETE") || !strings.Contains(out, "Keystone") {
		t.Fatalf("expected rich terminal UI, got: %q", out)
	}
}
