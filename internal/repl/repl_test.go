package repl

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/harness"
	"github.com/anonyxhappie/keystone/internal/state"
)

func setupTestProject(t *testing.T) string {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":       "module testrepl\n\ngo 1.23\n",
		"main.go":      "package main\n\nfunc main() {}\n",
		"main_test.go": "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := state.New(root)
	if _, err := s.Init("test-repl", []domain.Capability{{Kind: "language", Name: "go"}, {Kind: "test", Name: "go"}}); err != nil {
		t.Fatal(err)
	}

	harnessCfg := harness.Config{Name: "test-harness", Command: "sh", Args: []string{"-c", "read line; echo done"}, TimeoutSeconds: 10}
	cfgBytes, _ := json.Marshal(harnessCfg)
	_ = os.WriteFile(filepath.Join(root, state.Dir, "harness.json"), cfgBytes, 0600)

	return root
}

func TestREPLSlashCommands(t *testing.T) {
	root := setupTestProject(t)

	input := strings.Join([]string{
		"/help",
		"/harness",
		"/harness codex",
		"/sessions",
		"/new",
		"/projects",
		"/status",
		"/clear",
		"/exit",
	}, "\n") + "\n"

	in := strings.NewReader(input)
	var out bytes.Buffer

	r := New(root, "antigravity", in, &out)
	err := r.Run()
	if err != nil {
		t.Fatalf("unexpected REPL error: %v", err)
	}

	output := out.String()
	for _, expected := range []string{
		"Available Slash Commands",
		"Active harness:",
		"antigravity",
		"Switched active harness to",
		"codex",
		"Started fresh conversation context",
		"Local Code Projects",
		"Keystone Project Status",
		"Exiting Keystone interactive session",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("missing expected text %q in REPL output: %s", expected, output)
		}
	}
}

func TestREPLExecutePrompt(t *testing.T) {
	root := setupTestProject(t)

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	codexDir := filepath.Join(tempHome, ".codex")
	_ = os.MkdirAll(codexDir, 0755)
	indexLine := `{"id":"SES-EXISTING-1","thread_name":"Existing Test Thread","updated_at":"2026-03-13T15:28:39Z"}` + "\n"
	_ = os.WriteFile(filepath.Join(codexDir, "session_index.jsonl"), []byte(indexLine), 0600)

	input := strings.Join([]string{
		"/sessions",
		"/resume 1",
		"/harness auto",
		"inspect the codebase",
		"/exit",
	}, "\n") + "\n"

	in := strings.NewReader(input)
	var out bytes.Buffer

	r := New(root, "auto", in, &out)
	err := r.Run()
	if err != nil {
		t.Fatalf("unexpected REPL error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Resumed session #1: SES-EXISTING-1") {
		t.Errorf("expected session #1 resumed, got output: %s", output)
	}
	if !strings.Contains(output, "RUN COMPLETE") {
		t.Errorf("expected RUN COMPLETE after prompt execution, got output: %s", output)
	}
}
