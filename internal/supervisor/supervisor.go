package supervisor

import (
	"strings"

	"github.com/anonyxhappie/keystone/internal/domain"
)

const (
	UnsupportedCompletion = "UNVERIFIED_COMPLETION"
	RequirementDrift      = "REQUIREMENT_DRIFT"
	Loop                  = "LOOP"
)

type Result struct {
	Status string
	Claims []string
	ChangedFiles []string
	ValidationPassed bool
	PreviousActions []string
}

func Evaluate(r Result) []domain.Finding {
	var findings []domain.Finding
	if strings.EqualFold(r.Status, "completed") && !r.ValidationPassed {
		findings = append(findings, domain.Finding{ID:"F-UNVERIFIED", Type:UnsupportedCompletion, Severity:"high", Confidence:0.99, RecommendedAction:"RUN_VALIDATION"})
	}
	if len(r.PreviousActions) >= 2 && r.PreviousActions[len(r.PreviousActions)-1] == r.PreviousActions[len(r.PreviousActions)-2] {
		findings = append(findings, domain.Finding{ID:"F-LOOP", Type:Loop, Severity:"warning", Confidence:0.97, RecommendedAction:"REPLAN"})
	}
	return findings
}
