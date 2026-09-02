package work

import (
	"fmt"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func NewOrder(request string) domain.WorkOrder {
	return domain.WorkOrder{ID: fmt.Sprintf("WO-%d", time.Now().UnixNano()), SourceRequest: request, Objective: strings.TrimSpace(request), Risk: domain.Risk{Level: "low", Score: 0}, Autonomy: "assist", Status: domain.StatusPlanned, CreatedAt: time.Now().UTC()}
}

func Packet(o domain.WorkOrder) domain.WorkPacket {
	return domain.WorkPacket{WorkOrderID: o.ID, Objective: o.Objective, Requirements: o.Requirements, Constraints: o.Constraints, CompletionCriteria: []string{"requested objective satisfied", "relevant validation passes"}}
}
