package artifact

import (
	"strings"
	"testing"

	"github.com/anonyxhappie/keystone/internal/state"
)

func TestSaveTextRedactsSensitiveOutputAndCanReadArtifact(t *testing.T) {
	s := state.New(t.TempDir())
	if _, err := s.Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	a, err := SaveText(s, "command-output", "token=abc password=secret token=def")
	if err != nil {
		t.Fatal(err)
	}
	text, err := Read(s, a)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "abc") || strings.Contains(text, "secret") || strings.Contains(text, "def") || !strings.Contains(text, "[REDACTED]") {
		t.Fatalf("artifact was not redacted: %q", text)
	}
}
