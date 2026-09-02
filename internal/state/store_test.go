package state

import (
	"github.com/anonyxhappie/keystone/internal/domain"
	"os"
	"testing"
)

func TestInitAndRead(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	p, err := s.Init("fixture", []domain.Capability{{Kind: "vcs", Name: "git"}})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "fixture" {
		t.Fatal(p.Name)
	}
	var got domain.Project
	if err := s.Read("project.json", &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "project-1" || len(got.Capabilities) != 1 {
		t.Fatalf("unexpected project: %+v", got)
	}
	if _, err := os.Stat(root + "/.keystone/evidence"); err != nil {
		t.Fatal(err)
	}
}
