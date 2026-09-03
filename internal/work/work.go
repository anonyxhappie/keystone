package work

import (
	"fmt"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func IsReadOnlyRequest(request string) bool {
	lower := strings.ToLower(request)
	phrases := []string{
		"do not modify any files",
		"do not modify files",
		"do not modify",
		"do not make changes",
		"no changes",
		"without modifying",
		"without making changes",
		"read-only",
		"read only",
		"inspect only",
		"audit only",
	}
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// KnownDirectives lists specialized harness workflows supported across Keystone.
var KnownDirectives = []string{
	"goal",
	"boost",
	"teamwork-preview",
	"browser",
	"learn",
	"schedule",
	"grill-me",
	"btw",
}

// ExtractDirective inspects a prompt for a leading slash directive (e.g. /goal, /boost).
func ExtractDirective(request string) (directive string, cleanRequest string) {
	trimmed := strings.TrimSpace(request)
	if !strings.HasPrefix(trimmed, "/") {
		return "", trimmed
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", trimmed
	}
	cmd := strings.TrimPrefix(strings.ToLower(parts[0]), "/")
	for _, kd := range KnownDirectives {
		if cmd == kd {
			remainder := strings.TrimSpace(trimmed[len(parts[0]):])
			return cmd, remainder
		}
	}
	return "", trimmed
}

// DetectAutonomousEscalation applies heuristics to engage /goal or /boost for complex tasks.
func DetectAutonomousEscalation(request string) string {
	lower := strings.ToLower(request)

	// Boost triggers: deep thinking, multi-agent orchestration, complex architectural design
	boostPhrases := []string{
		"deep analysis",
		"multi-agent",
		"orchestrate",
		"architectural plan",
		"complex redesign",
	}
	for _, phrase := range boostPhrases {
		if strings.Contains(lower, phrase) {
			return "boost"
		}
	}

	// Goal triggers: multi-step objectives, large refactors, complete implementations, migrations
	goalPhrases := []string{
		"refactor the entire",
		"refactor all",
		"migrate",
		"implement from scratch",
		"build the entire",
		"finish all tasks",
		"complete the project",
		"fix all errors",
		"fix all issues",
		"end-to-end implementation",
	}
	for _, phrase := range goalPhrases {
		if strings.Contains(lower, phrase) {
			return "goal"
		}
	}

	return ""
}

func NewOrder(request string) domain.WorkOrder {
	now := time.Now().UTC()
	directive, cleanRequest := ExtractDirective(request)
	if directive == "" {
		directive = DetectAutonomousEscalation(request)
	}

	readOnly := IsReadOnlyRequest(request) || directive == "btw"
	constraints := []string{}
	if readOnly {
		constraints = append(constraints, "read-only: do not modify any repository files")
	}

	obj := strings.TrimSpace(cleanRequest)
	if obj == "" {
		obj = strings.TrimSpace(request)
	}

	return domain.WorkOrder{
		ID:            fmt.Sprintf("WO-%d", now.UnixNano()),
		SourceRequest: request,
		Objective:     obj,
		Risk:          domain.Risk{Level: "low", Score: 0},
		Autonomy:      "assist",
		Status:        domain.StatusPlanned,
		CreatedAt:     now,
		UpdatedAt:     now,
		Constraints:   constraints,
		ReadOnly:      readOnly,
		Directive:     directive,
	}
}

func AssessRisk(request string) domain.Risk {
	lower := strings.ToLower(request)
	risk := domain.Risk{Level: "low", Score: 0, Rationale: "no high-risk operation detected"}
	for _, term := range []string{"deploy", "production", "force push", "credential", "secret", "destructive", "drop table", "truncate"} {
		if strings.Contains(lower, term) {
			risk.Level = "high"
			risk.Score = 3
			risk.Factors = append(risk.Factors, term)
			risk.Rationale = "request contains a policy-sensitive operation"
		}
	}
	if strings.Contains(lower, "release") {
		risk.Level = "release"
		risk.Score = 4
		risk.Factors = append(risk.Factors, "release")
		risk.Rationale = "release work requires release evidence and policy review"
	}
	return risk
}

func Packet(o domain.WorkOrder) domain.WorkPacket {
	criteria := []string{"requested objective satisfied", "relevant validation passes"}
	if o.ReadOnly {
		criteria = append(criteria, "no repository mutations occurred")
	}
	return domain.WorkPacket{
		WorkOrderID:        o.ID,
		Objective:          o.Objective,
		Requirements:       o.Requirements,
		Constraints:        o.Constraints,
		CompletionCriteria: criteria,
		ReadOnly:           o.ReadOnly,
		Directive:          o.Directive,
	}
}
