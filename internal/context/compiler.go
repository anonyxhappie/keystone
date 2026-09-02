package context

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func Compile(root string, packet domain.WorkPacket) domain.WorkPacket {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() { continue }
		name := e.Name()
		if strings.EqualFold(name, "README.md") || strings.EqualFold(name, "AGENTS.md") || strings.EqualFold(name, "CLAUDE.md") {
			packet.Context = append(packet.Context, domain.ContextRef{Path:filepath.ToSlash(name), Reason:"project instructions/context", Source:"repository"})
		}
	}
	return packet
}
