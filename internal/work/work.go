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

func NewOrder(request string) domain.WorkOrder {
	now := time.Now().UTC()
	readOnly := IsReadOnlyRequest(request)
	constraints := []string{}
	if readOnly {
		constraints = append(constraints, "read-only: do not modify any repository files")
	}
	return domain.WorkOrder{
		ID:            fmt.Sprintf("WO-%d", now.UnixNano()),
		SourceRequest: request,
		Objective:     strings.TrimSpace(request),
		Risk:          domain.Risk{Level: "low", Score: 0},
		Autonomy:      "assist",
		Status:        domain.StatusPlanned,
		CreatedAt:     now,
		UpdatedAt:     now,
		Constraints:   constraints,
		ReadOnly:      readOnly,
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
	}
}
