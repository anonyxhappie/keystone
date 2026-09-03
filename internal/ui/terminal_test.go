package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/observation"
)

func TestTerminalHeader(t *testing.T) {
	var buf bytes.Buffer
	term := New(&buf)
	term.Header("/path/to/project", "antigravity", true, "Audit the project. Do not modify files.")

	out := buf.String()
	if !strings.Contains(out, "Keystone") {
		t.Fatalf("expected Keystone banner, got: %q", out)
	}
	if !strings.Contains(out, "/path/to/project") {
		t.Fatalf("expected workspace path, got: %q", out)
	}
	if !strings.Contains(out, "antigravity") {
		t.Fatalf("expected harness name, got: %q", out)
	}
	if !strings.Contains(out, "Read-Only") {
		t.Fatalf("expected Read-Only mode, got: %q", out)
	}
}

func TestTerminalOnEvent(t *testing.T) {
	var buf bytes.Buffer
	term := New(&buf)

	events := []observation.Event{
		{Type: "REQUEST_ACCEPTED", Payload: map[string]any{"workOrderId": "WO-123"}},
		{Type: "GIT_BASELINE_DETAILED", Payload: map[string]any{"preRunFiles": 5}},
		{Type: "CONTEXT_PLAN", Payload: map[string]any{"tokens": 4200}},
		{Type: "HARNESS_SELECTED", Payload: map[string]any{"selectedHarness": "antigravity", "selectionMode": "explicit"}},
		{Type: "RUN_DISPATCHED", Payload: map[string]any{"sessionId": "SES-999"}},
		{Type: "RUN_STOPPED", Payload: map[string]any{"status": "completed"}},
		{Type: "MUTATIONS_DETECTED", Payload: map[string]any{"count": 2}},
		{Type: "MUTATIONS_RESTORED", Payload: map[string]any{"restored": 2}},
		{Type: "POLICY_DECISION", Payload: map[string]any{"action": "edit", "decision": "REQUIRE_APPROVAL", "reason": "destructive action"}},
		{Type: "DECISION", Payload: map[string]any{"decision": "COMPLETE", "reason": "evidence verified"}},
		{Type: "HARNESS_SWITCHED", Payload: map[string]any{"from": "codex", "to": "antigravity", "attempt": 2}},
	}

	for _, ev := range events {
		term.OnEvent(ev)
	}

	out := buf.String()
	for _, expected := range []string{
		"Request accepted",
		"Git baseline captured",
		"Context compiled",
		"Authoritative harness selected",
		"Dispatched to harness",
		"Harness execution turn finished",
		"Repository mutations detected",
		"Non-destructive rollback",
		"Policy gate",
		"Lifecycle decision",
		"Switched harness",
	} {
		if !strings.Contains(out, expected) {
			t.Errorf("missing expected event text %q in output: %q", expected, out)
		}
	}
}

func TestTerminalOnObservation(t *testing.T) {
	var buf bytes.Buffer
	term := New(&buf)

	observations := []domain.Observation{
		{Type: "TOOL_STARTED", Summary: "list_dir /src"},
		{Type: "COMMAND_STARTED", Summary: "npm test"},
		{Type: "FILE_TOUCHED", Summary: "package.json"},
		{Type: "COMPLETION_CLAIM", Summary: "I have audited all endpoints."},
		{Type: "SESSION_STARTED", Summary: "session-abc-123"},
		{Type: "STDOUT", Summary: "First line of stdout\nSecond line"},
	}

	for _, obs := range observations {
		term.OnObservation(obs)
	}

	out := buf.String()
	for _, expected := range []string{
		"list_dir /src",
		"npm test",
		"package.json",
		"I have audited all endpoints.",
		"session-abc-123",
		"First line of stdout",
	} {
		if !strings.Contains(out, expected) {
			t.Errorf("missing expected observation text %q in output: %q", expected, out)
		}
	}
}

func TestTerminalReport(t *testing.T) {
	var buf bytes.Buffer
	term := New(&buf)

	validations := []ValidationSummary{
		{Name: "vitest", Passed: false, Summary: "Environment blocker: PostgreSQL not reachable"},
		{Name: "git-diff", Passed: true},
	}

	term.Report(
		"RUN-100",
		"WO-200",
		"antigravity",
		"ses-300",
		"STOPPED",
		"STOP",
		"safe stop after failed verification",
		true,
		0,
		5108,
		2,
		6,
		"completion was not verified",
		validations,
	)

	out := buf.String()
	for _, expected := range []string{
		"RUN STOPPED",
		"RUN-100",
		"antigravity",
		"ses-300",
		"2 of 6",
		"0 mutations",
		"5108 / 20,000",
		"vitest",
		"PostgreSQL not reachable",
		"git-diff",
		"safe stop after failed verification",
		".keystone/state.json",
	} {
		if !strings.Contains(out, expected) {
			t.Errorf("missing expected report text %q in output: %q", expected, out)
		}
	}
}
