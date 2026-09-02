package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
)

func TestCompileWithImpactRanksChangedAndInstructionContext(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "internal"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("rules"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "auth.go"), []byte("package internal"), 0644); err != nil {
		t.Fatal(err)
	}
	p := CompileWithImpact(root, domain.WorkPacket{Objective: "auth change"}, []string{"internal/auth.go"})
	if len(p.Context) < 2 || p.Context[0].Path != "internal/auth.go" || p.Context[0].Reason == "" {
		t.Fatalf("unexpected context: %+v", p.Context)
	}
}
