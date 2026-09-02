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
			at := e.At
			if at.IsZero() {
				at = time.Unix(0, 0).UTC()
			}
			err = m.transitionAt(e.To, e.Reason, at)
		case "decision":
			err = m.ApplyDecision(e.Decision, e.Reason)
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
	Request: {Understand: true}, Understand: {Assess: true}, Assess: {Plan: true, Ask: true, Blocked: true},
	Plan: {Context: true}, Context: {Dispatch: true}, Dispatch: {Execute: true}, Execute: {Observe: true},
	Observe: {Verify: true}, Verify: {Evaluate: true}, Evaluate: {Supervise: true}, Supervise: {Decide: true},
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
	return m.TransitionTo(target, reason)
}

func (m *Machine) NextAction(risk string, allowed bool, reason string) domain.NextAction {
	typeName := string(m.State)
	if m.State == Decide {
		typeName = string(ContinueDecision)
	}
	if m.Terminal() {
		typeName = string(CompleteDecision)
	}
	h := sha256.Sum256([]byte(string(m.State) + "\x00" + reason))
	return domain.NextAction{ID: "ACT-" + hex.EncodeToString(h[:6]), Type: typeName, Reason: reason, Risk: risk, Allowed: allowed, RequiresApproval: !allowed}
}

func (m *Machine) Terminal() bool { return m.State == Complete || m.State == Stopped }
