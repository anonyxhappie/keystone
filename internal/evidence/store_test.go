package evidence

import (
	"testing"

	"github.com/anonyxhappie/keystone/v2/internal/state"
)

func TestEvidenceValidityDependsOnInputs(t *testing.T) {
	s := state.New(t.TempDir())
	if _, err := s.Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	e, err := RecordScoped(s, "WO-1", "test", "tests passed", "abc", "inputs-1", []string{"OBS-1"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !ValidFor(e, "abc", "inputs-1") {
		t.Fatal("expected evidence to be valid")
	}
	if ValidFor(e, "changed", "inputs-1") {
		t.Fatal("evidence should be stale for changed commit")
	}
	if err := Invalidate(s, e, "changed commit"); err != nil {
		t.Fatal(err)
	}
	e.Valid = false
	if EnsureValid(e) == nil {
		t.Fatal("invalid evidence was accepted")
	}
}
