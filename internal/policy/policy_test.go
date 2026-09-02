package policy

import "testing"

func TestDefaultPolicy(t *testing.T) {
	if got:=Evaluate("run_tests"); !got.Allowed || got.Decision!="CONTINUE" { t.Fatalf("unexpected decision: %+v",got) }
}
func TestDestructiveActionsAreGated(t *testing.T) {
	for _, action:=range []string{"production_deploy","destructive_migration","destructive_data","credential_change","force_push"} {
		if got:=Evaluate(action); got.Allowed || got.Decision!="APPROVE" { t.Fatalf("%s was not gated: %+v",action,got) }
	}
}
