package runtime

import (
	"fmt"
	"time"
)

// State is the canonical lifecycle state shared by every Keystone version.
type State string

const (
	Request State = "REQUEST"
	Understand State = "UNDERSTAND"
	Assess State = "ASSESS"
	Plan State = "PLAN"
	Context State = "CONTEXT"
	Dispatch State = "DISPATCH"
	Execute State = "EXECUTE"
	Observe State = "OBSERVE"
	Verify State = "VERIFY"
	Evaluate State = "EVALUATE"
	Supervise State = "SUPERVISE"
	Decide State = "DECIDE"
	Continue State = "CONTINUE"
	Correct State = "CORRECT"
	Replan State = "REPLAN"
	Ask State = "ASK"
	Approve State = "APPROVE"
	Blocked State = "BLOCKED"
	Stopped State = "STOPPED"
	Complete State = "COMPLETE"
)

type Decision string
const (
	ContinueDecision Decision = "CONTINUE"
	CorrectDecision Decision = "CORRECT"
	ReplanDecision Decision = "REPLAN"
	ValidateDecision Decision = "VALIDATE"
	AskDecision Decision = "ASK"
	ApproveDecision Decision = "APPROVE"
	BlockDecision Decision = "BLOCK"
	StopDecision Decision = "STOP"
	CompleteDecision Decision = "COMPLETE"
)

type Transition struct {
	From State `json:"from"`
	To State `json:"to"`
	Reason string `json:"reason,omitempty"`
	At time.Time `json:"at"`
}

type Machine struct { State State `json:"state"`; History []Transition `json:"history"` }

func New() *Machine { return &Machine{State: Request} }

var allowed = map[State]map[State]bool{
	Request:{Understand:true}, Understand:{Assess:true}, Assess:{Plan:true,Ask:true,Blocked:true},
	Plan:{Context:true}, Context:{Dispatch:true}, Dispatch:{Execute:true}, Execute:{Observe:true},
	Observe:{Verify:true}, Verify:{Evaluate:true}, Evaluate:{Supervise:true}, Supervise:{Decide:true},
	Decide:{Continue:true,Correct:true,Replan:true,Ask:true,Approve:true,Blocked:true,Stopped:true,Complete:true},
	Continue:{Dispatch:true}, Correct:{Context:true}, Replan:{Plan:true}, Ask:{Approve:true,Stopped:true},
	Approve:{Dispatch:true,Stopped:true}, Blocked:{Ask:true,Stopped:true}, Stopped:{}, Complete:{},
}

func (m *Machine) TransitionTo(to State, reason string) error {
	if !allowed[m.State][to] { return fmt.Errorf("invalid Keystone transition %s -> %s", m.State, to) }
	m.History = append(m.History, Transition{From:m.State, To:to, Reason:reason, At:time.Now().UTC()})
	m.State = to
	return nil
}

func (m *Machine) ApplyDecision(d Decision, reason string) error {
	var target State
	switch d { case ContinueDecision: target=Continue; case CorrectDecision: target=Correct; case ReplanDecision: target=Replan; case AskDecision: target=Ask; case ApproveDecision: target=Approve; case BlockDecision: target=Blocked; case StopDecision: target=Stopped; case CompleteDecision: target=Complete; default: return fmt.Errorf("unsupported decision %q", d) }
	return m.TransitionTo(target, reason)
}

func (m *Machine) Terminal() bool { return m.State==Complete || m.State==Stopped }
