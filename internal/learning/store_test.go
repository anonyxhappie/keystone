package learning

import (
	"testing"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/state"
)

func TestLearningLifecycleIsExplicitAndVersioned(t *testing.T) {
	s := state.New(t.TempDir())
	if _, err := s.Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	l, err := Transition(s, domain.Learning{ID: "L-1", Scope: "project", Observation: "repeated test failure", ProposedChange: "include affected test"}, "CANDIDATE", "")
	if err != nil {
		t.Fatal(err)
	}
	if l.Status != "CANDIDATE" || l.Version != 1 {
		t.Fatalf("unexpected candidate: %+v", l)
	}
	l, err = Activate(s, l, "validation time improved")
	if err != nil {
		t.Fatal(err)
	}
	if l.Status != "ACTIVE" || l.Version != 2 {
		t.Fatalf("unexpected active record: %+v", l)
	}
	if l.Outcome != "validation time improved" || len(Active(s, "project")) != 1 {
		t.Fatalf("active learning was not persisted for future context: %+v", l)
	}
	l, err = Supersede(s, l, "newer strategy evaluated")
	if err != nil || l.Status != "SUPERSEDED" {
		t.Fatalf("unexpected superseded record: %+v %v", l, err)
	}
}
