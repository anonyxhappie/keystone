package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSessions(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	codexDir := filepath.Join(tempHome, ".codex")
	_ = os.MkdirAll(codexDir, 0755)
	indexLine := `{"id":"019c3d2f-b087-7503-bc5e-670103f5f67e","thread_name":"Draft Agentic RAG specification","updated_at":"2026-03-13T15:28:39Z"}` + "\n"
	_ = os.WriteFile(filepath.Join(codexDir, "session_index.jsonl"), []byte(indexLine), 0600)

	sessions := DiscoverSessions(tempHome)
	found := false
	for _, s := range sessions {
		if s.ID == "019c3d2f-b087-7503-bc5e-670103f5f67e" {
			found = true
			if s.Title != "Draft Agentic RAG specification" {
				t.Fatalf("expected title 'Draft Agentic RAG specification', got %q", s.Title)
			}
			if s.Harness != "codex" {
				t.Fatalf("expected harness codex, got %q", s.Harness)
			}
		}
	}
	if !found {
		t.Fatalf("expected to discover codex session, got: %+v", sessions)
	}
}

func TestDiscoverProjects(t *testing.T) {
	parent := t.TempDir()
	projA := filepath.Join(parent, "proj-a")
	projB := filepath.Join(parent, "proj-b")
	_ = os.MkdirAll(filepath.Join(projA, ".git"), 0755)
	_ = os.MkdirAll(filepath.Join(projB, ".keystone"), 0755)

	projects := DiscoverProjects(projA)
	if len(projects) < 2 {
		t.Fatalf("expected at least 2 projects, got %d", len(projects))
	}
	activeFound := false
	for _, p := range projects {
		if p.Path == projA && p.Active {
			activeFound = true
		}
	}
	if !activeFound {
		t.Fatalf("expected projA to be active, got: %+v", projects)
	}
}
