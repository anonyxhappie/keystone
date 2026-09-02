package supervisor

import "testing"

func TestUnsupportedCompletion(t *testing.T) {
	f := Evaluate(Result{Status: "completed", ValidationPassed: false})
	if len(f) != 1 || f[0].Type != UnsupportedCompletion {
		t.Fatalf("unexpected findings: %+v", f)
	}
}
func TestLoopDetection(t *testing.T) {
	f := Evaluate(Result{Status: "working", ValidationPassed: true, PreviousActions: []string{"read:file", "read:file"}})
	if len(f) != 1 || f[0].Type != Loop {
		t.Fatalf("unexpected findings: %+v", f)
	}
}

func TestSupervisorDetectsScopeAndArchitectureDrift(t *testing.T) {
	f := Evaluate(Result{ChangedFiles: []string{"secrets.txt"}, ScopeAllowed: false, ArchitectureDrift: true})
	if len(f) != 2 || f[0].Type != ScopeExpansion || f[1].Type != ArchitectureDrift {
		t.Fatalf("unexpected findings: %+v", f)
	}
}

func TestSupervisorDetectsRepeatedFailureAfterLoop(t *testing.T) {
	f := Evaluate(Result{PreviousActions: []string{"cmd:x", "cmd:x", "cmd:x"}})
	if len(f) != 2 || f[0].Type != Loop || f[1].Type != RepeatedFailure {
		t.Fatalf("unexpected findings: %+v", f)
	}
}

func TestSupervisorDetectsRequirementDriftAndInefficientReads(t *testing.T) {
	f := Evaluate(Result{RequirementDrift: true, RepeatedReads: []string{"README.md", "README.md"}, EvidenceIDs: []string{"EV-1"}})
	if len(f) != 2 || f[0].Type != RequirementDrift || f[1].Type != InefficientActivity {
		t.Fatalf("unexpected findings: %+v", f)
	}
	if len(f[0].EvidenceIDs) == 0 || f[0].EvidenceIDs[0] != "EV-1" {
		t.Fatalf("finding lost evidence references: %+v", f[0])
	}
}
