package harness

import (
	"encoding/json"
	"github.com/anonyxhappie/keystone/internal/domain"
)

func Render(p domain.WorkPacket) string {
	b, _ := json.MarshalIndent(p, "", "  ")
	header := "Execute this Keystone work packet. Preserve its requirements and constraints. Report changes, validation, blockers, and claims. Do not claim verification without evidence."
	if p.ReadOnly {
		header = "CRITICAL POLICY CONSTRAINT: This is an explicit READ-ONLY execution. Do NOT modify, create, or delete any files under any circumstances. Inspect and report only. Keystone enforces this constraint via independent repository baseline inspection.\n\n" + header
	}
	return header + "\n\n" + string(b)
}
