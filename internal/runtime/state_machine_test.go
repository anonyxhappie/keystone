package runtime

import "testing"

func TestCanonicalLifecycleCompletesOnlyThroughDecision(t *testing.T) {
 m:=New()
 path:=[]State{Understand,Assess,Plan,Context,Dispatch,Execute,Observe,Verify,Evaluate,Supervise,Decide}
 for _,s:=range path { if err:=m.TransitionTo(s,"test");err!=nil{t.Fatal(err)} }
 if err:=m.ApplyDecision(CompleteDecision,"verified");err!=nil{t.Fatal(err)}
 if !m.Terminal() || m.State!=Complete {t.Fatalf("unexpected terminal state: %s",m.State)}
}
func TestInvalidTransitionRejected(t *testing.T){m:=New();if err:=m.TransitionTo(Complete,"unverified");err==nil{t.Fatal("expected invalid completion transition")}}
func TestRecoveryBranches(t *testing.T){m:=New();for _,s:=range []State{Understand,Assess,Plan,Context,Dispatch,Execute,Observe,Verify,Evaluate,Supervise,Decide}{if err:=m.TransitionTo(s,"");err!=nil{t.Fatal(err)}};if err:=m.ApplyDecision(CorrectDecision,"drift");err!=nil{t.Fatal(err)};if err:=m.TransitionTo(Context,"correct");err!=nil{t.Fatal(err)}}
