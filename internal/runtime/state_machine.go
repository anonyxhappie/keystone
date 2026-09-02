package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
)

// State is the canonical lifecycle state shared by every Keystone version.
type State string

const (
	Request    State = "REQUEST"
	Understand State = "UNDERSTAND"
	Assess     State = "ASSESS"
	Plan       State = "PLAN"
	Context    State = "CONTEXT"
	Dispatch   State = "DISPATCH"
	Execute    State = "EXECUTE"
	Observe    State = "OBSERVE"
	Verify     State = "VERIFY"
	Evaluate   State = "EVALUATE"
	Supervise  State = "SUPERVISE"
	Decide     State = "DECIDE"
	Continue   State = "CONTINUE"
	Correct    State = "CORRECT"
	Replan     State = "REPLAN"
	Ask        State = "ASK"
	Approve    State = "APPROVE"
	Blocked    State = "BLOCKED"
	Stopped    State = "STOPPED"
	Complete   State = "COMPLETE"
)

type Decision string

const (
	ContinueDecision Decision = "CONTINUE"
	CorrectDecision  Decision = "CORRECT"
	ReplanDecision   Decision = "REPLAN"
	ValidateDecision Decision = "VALIDATE"
	AskDecision      Decision = "ASK"
	ApproveDecision  Decision = "APPROVE"
	BlockDecision    Decision = "BLOCK"
	StopDecision     Decision = "STOP"
	CompleteDecision Decision = "COMPLETE"
)

// NextAction is the machine-readable output shared by assist and full-auto modes.
type NextAction = domain.NextAction

type Transition struct {
	From   State     `json:"from"`
	To     State     `json:"to"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

type Machine struct {
	State   State        `json:"state"`
	History []Transition `json:"history"`
}

// Event is the minimal durable input needed to reconstruct the canonical machine.
// Unknown event types are ignored by Reduce for forward-compatible replay.
type Event struct {
	Type     string    `json:"type"`
	From     State     `json:"from,omitempty"`
	To       State     `json:"to,omitempty"`
	Decision Decision  `json:"decision,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	At       time.Time `json:"at"`
}

func New() *Machine { return &Machine{State: Request} }

func Reduce(events []Event) (*Machine, error) {
	m := New()
	for _, e := range events {
		var err error
		switch e.Type {
		case "transition":
			if e.From != "" && e.From != m.State {
				return nil, fmt.Errorf("replay transition source %s does not match machine state %s", e.From, m.State)
			}
			at := e.At
			if at.IsZero() {
				at = time.Unix(0, 0).UTC()
			}
			err = m.transitionAt(e.To, e.Reason, at)
		case "decision":
			if e.From != "" && e.From != m.State {
				return nil, fmt.Errorf("replay decision source %s does not match machine state %s", e.From, m.State)
			}
			at := e.At
			if at.IsZero() {
				at = time.Unix(0, 0).UTC()
			}
			err = m.applyDecisionAt(e.Decision, e.Reason, at)
		case "":
		default:
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("replay %s: %w", e.Type, err)
		}
	}
	return m, nil
}

var allowed = map[State]map[State]bool{
	Request: {Understand: true, Blocked: true, Stopped: true}, Understand: {Assess: true, Blocked: true, Stopped: true}, Assess: {Plan: true, Ask: true, Blocked: true, Stopped: true},
	Plan: {Context: true, Blocked: true, Stopped: true}, Context: {Dispatch: true, Blocked: true, Stopped: true}, Dispatch: {Execute: true, Blocked: true, Stopped: true}, Execute: {Observe: true, Blocked: true, Stopped: true},
	Observe: {Verify: true, Blocked: true, Stopped: true}, Verify: {Evaluate: true, Blocked: true, Stopped: true}, Evaluate: {Supervise: true, Blocked: true, Stopped: true}, Supervise: {Decide: true, Blocked: true, Stopped: true},
	Decide:   {Continue: true, Correct: true, Replan: true, Verify: true, Ask: true, Approve: true, Blocked: true, Stopped: true, Complete: true},
	Continue: {Dispatch: true}, Correct: {Context: true}, Replan: {Plan: true}, Ask: {Approve: true, Stopped: true},
	Approve: {Dispatch: true, Stopped: true}, Blocked: {Ask: true, Stopped: true}, Stopped: {}, Complete: {},
}

func (m *Machine) TransitionTo(to State, reason string) error {
	return m.transitionAt(to, reason, time.Now().UTC())
}

func (m *Machine) transitionAt(to State, reason string, at time.Time) error {
	if !allowed[m.State][to] {
		return fmt.Errorf("invalid Keystone transition %s -> %s", m.State, to)
	}
	m.History = append(m.History, Transition{From: m.State, To: to, Reason: reason, At: at})
	m.State = to
	return nil
}

func (m *Machine) ApplyDecision(d Decision, reason string) error {
	return m.applyDecisionAt(d, reason, time.Now().UTC())
}

func (m *Machine) applyDecisionAt(d Decision, reason string, at time.Time) error {
	var target State
	switch d {
	case ContinueDecision:
		target = Continue
	case CorrectDecision:
		target = Correct
	case ReplanDecision:
		target = Replan
	case ValidateDecision:
		target = Verify
	case AskDecision:
		target = Ask
	case ApproveDecision:
		target = Approve
	case BlockDecision:
		target = Blocked
	case StopDecision:
		target = Stopped
	case CompleteDecision:
		target = Complete
	default:
		return fmt.Errorf("unsupported decision %q", d)
	}
	return m.transitionAt(target, reason, at)
}

func (m *Machine) NextAction(risk string, allowed bool, reason string) domain.NextAction {
	next := map[State]string{Request: string(Understand), Understand: string(Assess), Assess: string(Plan), Plan: string(Context), Context: string(Dispatch), Dispatch: string(Execute), Execute: string(Observe), Observe: string(Verify), Verify: string(Evaluate), Evaluate: string(Supervise), Supervise: string(Decide), Decide: string(ContinueDecision), Continue: string(Dispatch), Correct: string(Context), Replan: string(Plan), Ask: string(Approve), Approve: string(Dispatch), Blocked: string(Ask), Stopped: string(StopDecision), Complete: string(CompleteDecision)}
	typeName := next[m.State]
	if typeName == "" {
		typeName = string(m.State)
	}
	policyDecision := "ALLOW"
	if !allowed {
		policyDecision = "REQUIRE_APPROVAL"
	}
	h := sha256.Sum256([]byte(string(m.State) + "\x00" + reason))
	return domain.NextAction{ID: "ACT-" + hex.EncodeToString(h[:6]), Type: typeName, Reason: reason, Target: typeName, Risk: risk, PolicyDecision: policyDecision, Allowed: allowed, RequiresApproval: !allowed}
}

func (m *Machine) Terminal() bool { return m.State == Complete || m.State == Stopped }
