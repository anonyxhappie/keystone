package harness

import (
	"encoding/json"
	"github.com/anonyxhappie/keystone/v2/internal/domain"
)

func Render(p domain.WorkPacket) string {
	b, _ := json.MarshalIndent(p, "", "  ")
	return "Execute this Keystone work packet. Preserve its requirements and constraints. Report changes, validation, blockers, and claims. Do not claim verification without evidence.\n\n" + string(b)
}
