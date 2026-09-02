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

func TestSnapshotRejectsLifecycleCorruption(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	if _, err := s.Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	snap, err := s.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	snap.Lifecycle = "COMPLETE"
	if err := s.SaveSnapshot(snap); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadSnapshot(); err == nil {
		t.Fatal("expected lifecycle corruption to be rejected")
	}
}

func TestStatePathCannotEscapeBoundary(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Write("../outside.json", map[string]string{"x": "y"}); err == nil {
		t.Fatal("expected path escape to be rejected")
	}
}

func TestStatePathCannotFollowSymlinkOutsideBoundary(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	if _, err := s.Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, root+"/.keystone/escape"); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := s.Write("escape/file.json", map[string]string{"secret": "no"}); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}
