package policy

import "testing"

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
