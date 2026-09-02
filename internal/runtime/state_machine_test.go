package runtime

import "testing"

func TestCanonicalLifecycleCompletesOnlyThroughDecision(t *testing.T) {
	m := New()
	path := []State{Understand, Assess, Plan, Context, Dispatch, Execute, Observe, Verify, Evaluate, Supervise, Decide}
	for _, s := range path {
		if err := m.TransitionTo(s, "test"); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.ApplyDecision(CompleteDecision, "verified"); err != nil {
		t.Fatal(err)
	}
	if !m.Terminal() || m.State != Complete {
		t.Fatalf("unexpected terminal state: %s", m.State)
	}
}
func TestInvalidTransitionRejected(t *testing.T) {
	m := New()
	if err := m.TransitionTo(Complete, "unverified"); err == nil {
		t.Fatal("expected invalid completion transition")
	}
}
func TestRecoveryBranches(t *testing.T) {
	m := New()
	for _, s := range []State{Understand, Assess, Plan, Context, Dispatch, Execute, Observe, Verify, Evaluate, Supervise, Decide} {
		if err := m.TransitionTo(s, ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.ApplyDecision(CorrectDecision, "drift"); err != nil {
		t.Fatal(err)
	}
	if err := m.TransitionTo(Context, "correct"); err != nil {
		t.Fatal(err)
	}
}

func TestNextActionPointsToCanonicalSuccessor(t *testing.T) {
	m := New()
	if a := m.NextAction("low", true, "intake"); a.Type != "UNDERSTAND" || a.RequiresApproval {
		t.Fatalf("unexpected request action: %+v", a)
	}
	for _, s := range []State{Understand, Assess, Plan, Context, Dispatch, Execute, Observe, Verify, Evaluate, Supervise, Decide} {
		if err := m.TransitionTo(s, "test"); err != nil {
			t.Fatal(err)
		}
	}
	if a := m.NextAction("low", true, "decision"); a.Type != "CONTINUE" {
		t.Fatalf("unexpected decision action: %+v", a)
	}
}

func TestAllDecisionBranchesAreStateMachineTransitions(t *testing.T) {
	for _, d := range []Decision{ContinueDecision, CorrectDecision, ReplanDecision, AskDecision, ApproveDecision, BlockDecision, StopDecision, CompleteDecision} {
		m := New()
		for _, s := range []State{Understand, Assess, Plan, Context, Dispatch, Execute, Observe, Verify, Evaluate, Supervise, Decide} {
			if err := m.TransitionTo(s, "test"); err != nil {
				t.Fatal(err)
			}
		}
		if err := m.ApplyDecision(d, "test"); err != nil {
			t.Fatalf("%s: %v", d, err)
		}
	}
}

func TestReducerRejectsTamperedTransitionSource(t *testing.T) {
	if _, err := Reduce([]Event{{Type: "transition", From: Assess, To: Understand}}); err == nil {
		t.Fatal("expected reducer to reject a tampered source state")
	}
}
