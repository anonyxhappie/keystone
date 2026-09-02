package work

import (
	"fmt"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
)

func NewOrder(request string) domain.WorkOrder {
	return domain.WorkOrder{ID: fmt.Sprintf("WO-%d", time.Now().UnixNano()), SourceRequest: request, Objective: strings.TrimSpace(request), Risk: domain.Risk{Level: "low", Score: 0}, Autonomy: "assist", Status: domain.StatusPlanned, CreatedAt: time.Now().UTC()}
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
	return domain.WorkPacket{WorkOrderID: o.ID, Objective: o.Objective, Requirements: o.Requirements, Constraints: o.Constraints, CompletionCriteria: []string{"requested objective satisfied", "relevant validation passes"}}
}
