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
