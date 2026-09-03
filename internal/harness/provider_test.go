package harness

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func TestCodexAdapterNormalizesJSONLEventsAndUsage(t *testing.T) {
	root := t.TempDir()
	command := writeProviderFixture(t, `#!/bin/sh
if [ "$1" = "--version" ]; then printf 'codex-fixture 1.0\n'; exit 0; fi
printf '%s\n' '{"type":"thread.started","thread_id":"thread-codex-1"}'
printf '%s\n' '{"type":"turn.started"}'
printf '%s\n' '{"type":"item.started","item":{"id":"cmd-1","type":"command_execution","command":"go test ./..."}}'
printf '%s\n' '{"type":"item.completed","item":{"id":"cmd-1","type":"command_execution","command":"go test ./...","exit_code":0}}'
printf '%s\n' '{"type":"item.completed","item":{"id":"msg-1","type":"agent_message","text":"implemented token=supersecret"}}'
printf '%s\n' '{"type":"turn.completed","usage":{"input_tokens":12,"output_tokens":8}}'
`)
	a := NewCodexAdapter(context.Background(), root, Config{Command: command, TimeoutSeconds: 10})
	if err := a.Discover(); err != nil {
		t.Fatal(err)
	}
	if a.Version() != "codex-fixture 1.0" {
		t.Fatalf("unexpected provider version: %q", a.Version())
	}
	if _, err := a.Start(domain.WorkPacket{WorkOrderID: "WO-1", Objective: "fixture"}); err != nil {
		t.Fatal(err)
	}
	observations := drainProvider(t, a)
	if a.SessionID() != "thread-codex-1" {
		t.Fatalf("session id was not captured: %q", a.SessionID())
	}
	if !hasObservationType(observations, "SESSION_STARTED") || !hasObservationType(observations, "COMMAND_COMPLETED") || !hasObservationType(observations, "COMPLETION_CLAIM") {
		t.Fatalf("missing normalized observations: %+v", observations)
	}
	var claim domain.Observation
	for _, item := range observations {
		if item.Type == "COMPLETION_CLAIM" {
			claim = item
		}
	}
	if strings.Contains(claim.Summary, "supersecret") || strings.Contains(stringify(claim.Payload), "supersecret") {
		t.Fatalf("provider secret was not redacted: %+v", claim)
	}
	if usage, ok := observations[len(observations)-1].Payload["usage"]; !ok || usage == nil {
		t.Fatalf("usage was not preserved in provider evidence: %+v", observations[len(observations)-1])
	}
	if status, err := a.Result(); err != nil || status != domain.StatusCompleted {
		t.Fatalf("unexpected provider result: %v %v", status, err)
	}
}

func TestAntigravityAdapterNormalizesStreamJSONAndResumesConversation(t *testing.T) {
	root := t.TempDir()
	command := writeProviderFixture(t, `#!/bin/sh
if [ "$1" = "--version" ]; then printf 'agy-fixture 1.0\n'; exit 0; fi
printf '%s\n' '{"type":"init","conversation_id":"conversation-agy-1"}'
printf '%s\n' '{"type":"step_update","step_type":"tool_call","status":"started","tool_name":"terminal","tool_info":{"command":"go test ./..."}}'
printf '%s\n' '{"type":"step_update","step_type":"tool_call","status":"completed","tool_name":"terminal","tool_info":{"exit_code":0}}'
printf '%s\n' '{"type":"step_update","step_type":"agent_response","text_delta":"implemented fixture"}'
printf '%s\n' '{"type":"result","status":"success","conversation_id":"conversation-agy-1","response":"implemented fixture","usage":{"input_tokens":10,"output_tokens":6}}'
`)
	a := NewAntigravityAdapter(context.Background(), root, Config{Command: command, TimeoutSeconds: 10})
	if err := a.Discover(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Start(domain.WorkPacket{WorkOrderID: "WO-2", Objective: "fixture"}); err != nil {
		t.Fatal(err)
	}
	observations := drainProvider(t, a)
	if a.SessionID() != "conversation-agy-1" {
		t.Fatalf("conversation id was not captured: %q", a.SessionID())
	}
	if !hasObservationType(observations, "TOOL_STARTED") || !hasObservationType(observations, "TOOL_COMPLETED") || !hasObservationType(observations, "COMPLETION_CLAIM") {
		t.Fatalf("missing normalized Antigravity observations: %+v", observations)
	}
	if status, err := a.Result(); err != nil || status != domain.StatusCompleted {
		t.Fatalf("unexpected provider result: %v %v", status, err)
	}

	checkpoint := domain.Checkpoint{WorkOrderID: "WO-2", HarnessID: "antigravity", HarnessSessionID: a.SessionID()}
	if _, err := a.ResumePacket(checkpoint, domain.WorkPacket{WorkOrderID: "WO-2", Objective: "continue fixture"}); err != nil {
		t.Fatal(err)
	}
	if got := a.args("follow-up", a.SessionID()); !hasArg(got, "--conversation") || !hasValue(got, a.SessionID()) {
		t.Fatalf("resume command omitted conversation id: %v", got)
	}
	_ = drainProvider(t, a)
}

func TestAntigravityAdapterNormalizesNestedRealStreamJSON(t *testing.T) {
	root := t.TempDir()
	command := writeProviderFixture(t, `#!/bin/sh
if [ "$1" = "--version" ]; then printf '1.1.24\n'; exit 0; fi
printf '%s\n' '{"event":"init","conversation_id":"conv-real-123","init":{"cwd":"/tmp","tools":["run_command"]}}'
printf '%s\n' '{"event":"step_update","step_update":{"conversation_id":"conv-real-123","step_index":0,"step_type":"tool_call","status":"started","tool_name":"run_command"}}'
printf '%s\n' '{"event":"step_update","step_update":{"conversation_id":"conv-real-123","step_index":1,"step_type":"agent_response","text_delta":"implemented real change"}}'
printf '%s\n' '{"event":"result","result":{"conversation_id":"conv-real-123","status":"SUCCESS","response":"implemented real change","usage":{"input_tokens":100,"output_tokens":25}}}'
`)
	a := NewAntigravityAdapter(context.Background(), root, Config{Command: command, TimeoutSeconds: 10})
	if err := a.Discover(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Start(domain.WorkPacket{WorkOrderID: "WO-REAL", Objective: "real test"}); err != nil {
		t.Fatal(err)
	}
	observations := drainProvider(t, a)
	if a.SessionID() != "conv-real-123" {
		t.Fatalf("conversation id was not captured: %q", a.SessionID())
	}
	if !hasObservationType(observations, "TOOL_STARTED") || !hasObservationType(observations, "COMPLETION_CLAIM") {
		t.Fatalf("missing normalized Antigravity observations: %+v", observations)
	}
	status, err := a.Result()
	if err != nil || status != domain.StatusCompleted {
		t.Fatalf("unexpected provider result: %v %v", status, err)
	}
}

func TestProviderSelectionAndMetadata(t *testing.T) {
	if normalizeProvider("agy") != "antigravity" || normalizeProvider("antigravity-ide") != "antigravity" || normalizeProvider("codex") != "codex" {
		t.Fatal("provider aliases were not normalized")
	}
	a := NewAdapter(context.Background(), t.TempDir(), Config{Provider: "agy", Command: "agy", TimeoutSeconds: 10})
	provider, ok := a.(*CLIAdapter)
	if !ok || provider.HarnessID() != "antigravity" {
		t.Fatalf("unexpected adapter for agy: %#v", a)
	}
	metadata := provider.Metadata()
	if metadata.Provider != "antigravity" || !hasValue(metadata.Control, "resume") || len(metadata.Limitations) == 0 {
		t.Fatalf("incomplete provider metadata: %+v", metadata)
	}
}

func TestLoadConfigResolvesProviderDefaults(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".keystone"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".keystone", "harness.json"), []byte(`{"provider":"agy","name":"team-agent"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "antigravity" || cfg.Command != "agy" || cfg.Name != "team-agent" || cfg.TimeoutSeconds != 300 {
		t.Fatalf("provider defaults were not resolved: %+v", cfg)
	}
}

func TestProviderDiagnosticsRedactAuthenticationURLs(t *testing.T) {
	diagnostic := redactDiagnostic("Authentication required: https://accounts.example.test/oauth?code_challenge=secret-value")
	if strings.Contains(diagnostic, "accounts.example.test") || strings.Contains(diagnostic, "secret-value") {
		t.Fatalf("diagnostic URL was not redacted: %q", diagnostic)
	}
}

func drainProvider(t *testing.T, adapter Adapter) []domain.Observation {
	t.Helper()
	var observations []domain.Observation
	for {
		items, err := adapter.Observe()
		observations = append(observations, items...)
		if err == io.EOF {
			return observations
		}
		if err != nil {
			t.Fatalf("observe failed: %v", err)
		}
	}
}

func hasObservationType(observations []domain.Observation, typ string) bool {
	for _, item := range observations {
		if item.Type == typ {
			return true
		}
	}
	return false
}

func writeProviderFixture(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "provider-fixture.sh")
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func stringify(value any) string {
	b, _ := json.Marshal(value)
	return strings.TrimSpace(string(b))
}

func hasValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestCLIAdapterDirectiveFormatting(t *testing.T) {
	root := t.TempDir()

	// 1. Antigravity goal directive
	agy := NewAntigravityAdapter(context.Background(), root, Config{Directive: "goal"})
	agyArgs := agy.args("implement feature X", "")
	if !hasValue(agyArgs, "/goal implement feature X") {
		t.Fatalf("expected Antigravity to format prompt with /goal prefix, got: %v", agyArgs)
	}

	// 2. Antigravity boost directive
	agyBoost := NewAntigravityAdapter(context.Background(), root, Config{Directive: "boost"})
	boostArgs := agyBoost.args("analyze architecture", "")
	if !hasValue(boostArgs, "/boost analyze architecture") {
		t.Fatalf("expected Antigravity to format prompt with /boost prefix, got: %v", boostArgs)
	}

	// 3. Codex goal directive defaults to o3 model
	codexGoal := NewCodexAdapter(context.Background(), root, Config{Directive: "goal"})
	codexArgs := codexGoal.args("finish project", "")
	if !hasValue(codexArgs, "o3") {
		t.Fatalf("expected Codex goal directive to use o3 model, got: %v", codexArgs)
	}

	// 4. Codex browser directive includes search
	codexBrowser := NewCodexAdapter(context.Background(), root, Config{Directive: "browser"})
	browserArgs := codexBrowser.args("search docs", "")
	if !hasValue(browserArgs, "--search") {
		t.Fatalf("expected Codex browser directive to include --search, got: %v", browserArgs)
	}

	// 5. Codex btw directive includes read-only sandbox
	codexBtw := NewCodexAdapter(context.Background(), root, Config{Directive: "btw"})
	btwArgs := codexBtw.args("what is this?", "")
	if !hasValue(btwArgs, "read-only") {
		t.Fatalf("expected Codex btw directive to include read-only sandbox, got: %v", btwArgs)
	}
}
