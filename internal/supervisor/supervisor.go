package supervisor

import (
	"strings"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
)

const (
	UnsupportedCompletion = "UNVERIFIED_COMPLETION"
	RequirementDrift      = "REQUIREMENT_DRIFT"
	ArchitectureDrift     = "ARCHITECTURE_DRIFT"
	ScopeExpansion        = "UNEXPECTED_SCOPE"
	Loop                  = "LOOP"
	RepeatedFailure       = "REPEATED_FAILURE"
	ExcessiveActivity     = "EXCESSIVE_ACTIVITY"
	StaleAssumption       = "STALE_ASSUMPTION"
	PrematureCompletion   = "PREMATURE_COMPLETION"
)

type Result struct {
	Status                string
	Claims                []string
	ChangedFiles          []string
	ValidationPassed      bool
	PreviousActions       []string
	RequirementsSatisfied bool
	ScopeAllowed          bool
	ArchitectureDrift     bool
	StaleAssumptions      bool
	ToolCount             int
	MaxToolCount          int
	ContextTokens         int
	MaxContextTokens      int
}

func Evaluate(r Result) []domain.Finding {
	var findings []domain.Finding
	add := func(id, typ, severity string, confidence float64, action, explanation string) {
		findings = append(findings, domain.Finding{ID: id, Type: typ, Severity: severity, Confidence: confidence, RecommendedAction: action, Explanation: explanation, Provenance: []string{"deterministic-supervisor"}})
	}
	if strings.EqualFold(r.Status, "completed") && !r.ValidationPassed {
		add("F-UNVERIFIED", UnsupportedCompletion, "high", 0.99, "RUN_VALIDATION", "completion was reported without passing validation")
	}
	if r.RequirementsSatisfied == false && len(r.Claims) > 0 && r.ValidationPassed {
		add("F-PREMATURE", PrematureCompletion, "high", 0.95, "CORRECT", "the harness claim is not corroborated by requirement evidence")
	}
	if r.ScopeAllowed == false && len(r.ChangedFiles) > 0 {
		add("F-SCOPE", ScopeExpansion, "high", 0.97, "ASK", "changed files exceed the approved work scope")
	}
	if r.ArchitectureDrift {
		add("F-ARCH", ArchitectureDrift, "high", 0.85, "REPLAN", "observed changes conflict with recorded architecture")
	}
	if r.StaleAssumptions {
		add("F-ASSUMPTION", StaleAssumption, "warning", 0.82, "ASK", "an assumption used by the work is stale")
	}
	if len(r.PreviousActions) >= 2 && r.PreviousActions[len(r.PreviousActions)-1] == r.PreviousActions[len(r.PreviousActions)-2] {
		add("F-LOOP", Loop, "warning", 0.97, "REPLAN", "the same action was repeated without a state change")
	}
	if len(r.PreviousActions) >= 3 && strings.EqualFold(r.PreviousActions[len(r.PreviousActions)-1], r.PreviousActions[len(r.PreviousActions)-2]) && strings.EqualFold(r.PreviousActions[len(r.PreviousActions)-2], r.PreviousActions[len(r.PreviousActions)-3]) {
		add("F-REPEAT", RepeatedFailure, "high", 0.99, "STOP", "the same failed approach repeated three times")
	}
	if r.MaxToolCount > 0 && r.ToolCount > r.MaxToolCount {
		add("F-TOOLS", ExcessiveActivity, "high", 0.99, "STOP", "tool activity exceeded the configured hard limit")
	}
	if r.MaxContextTokens > 0 && r.ContextTokens > r.MaxContextTokens {
		add("F-CONTEXT", ExcessiveActivity, "high", 0.99, "STOP", "context usage exceeded the configured hard limit")
	}
	return findings
}

func Review(r Result) []domain.Finding {
	findings := Evaluate(r)
	if len(r.ChangedFiles) > 0 && r.ValidationPassed {
		findings = append(findings, domain.Finding{ID: "F-REVIEW", Type: "ALTERNATIVE_REVIEW", Severity: "info", Confidence: 0.65, RecommendedAction: "CONTINUE", Explanation: "independent review requested; inspect correctness, architecture fit, reuse, security, tests, and complexity", Provenance: []string{"deterministic-review-input"}})
	}
	return findings
}
