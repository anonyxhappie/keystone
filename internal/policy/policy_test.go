package policy

import (
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func TestDefaultPolicy(t *testing.T) {
	if got := Evaluate("run_tests"); !got.Allowed || got.Decision != "CONTINUE" {
		t.Fatalf("unexpected decision: %+v", got)
	}
}
func TestDestructiveActionsAreGated(t *testing.T) {
	for _, action := range []string{"production_deploy", "destructive_migration", "destructive_data", "credential_change", "force_push"} {
		if got := Evaluate(action); got.Allowed || got.Decision != "APPROVE" {
			t.Fatalf("%s was not gated: %+v", action, got)
		}
	}
}

func TestCommandPolicyRejectsWorkspaceEscapeAndDestructiveCommands(t *testing.T) {
	if d := Command([]string{"git", "reset", "--hard"}, "/workspace"); d.Allowed || !d.RequiresApproval {
		t.Fatalf("destructive command was allowed: %+v", d)
	}
	if d := Command([]string{"tool", "/tmp/outside"}, "/workspace"); d.Allowed || d.Decision != "BLOCK" {
		t.Fatalf("workspace escape was allowed: %+v", d)
	}
	if d := CommandWithApproval([]string{"git", "reset", "--hard"}, "/workspace", true); !d.Allowed || !d.RequiresApproval {
		t.Fatalf("explicit approval did not authorize command: %+v", d)
	}
}

func TestEvaluateHarnessSelection(t *testing.T) {
	d := EvaluateHarnessSelection("codex", false, "command not found")
	if d.Allowed || !d.RequiresApproval || d.Decision != "REQUIRE_APPROVAL" {
		t.Fatalf("expected REQUIRE_APPROVAL for unavailable harness: %+v", d)
	}

	d2 := EvaluateHarnessSelection("codex", true, "")
	if !d2.Allowed || d2.RequiresApproval || d2.Decision != "ALLOW" {
		t.Fatalf("expected ALLOW for available harness: %+v", d2)
	}
}

func TestEvaluateReadOnly(t *testing.T) {
	d := EvaluateReadOnly(nil)
	if !d.Allowed || d.Decision != "ALLOW" {
		t.Fatalf("expected ALLOW for zero mutations: %+v", d)
	}

	mutations := []domain.FileMutation{{Path: "file.txt", Action: "created"}}
	d2 := EvaluateReadOnly(mutations)
	if d2.Allowed || d2.Decision != "STOP" {
		t.Fatalf("expected STOP for mutations: %+v", d2)
	}
}
