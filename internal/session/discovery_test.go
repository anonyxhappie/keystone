package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/state"
)

func TestDiscoverSessions(t *testing.T) {
	root := t.TempDir()
	sessDir := filepath.Join(root, state.Dir, "harness-sessions")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatal(err)
	}

	session := domain.HarnessSession{
		ID:        "test-session-123",
		HarnessID: "antigravity",
		RunID:     "RUN-1",
		Status:    domain.StatusCompleted,
		StartedAt: time.Now().UTC(),
	}
	data, _ := json.Marshal(session)
	if err := os.WriteFile(filepath.Join(sessDir, "test-session-123.json"), data, 0600); err != nil {
		t.Fatal(err)
	}

	sessions := DiscoverSessions(root)
	found := false
	for _, s := range sessions {
		if s.ID == "test-session-123" {
			found = true
			if s.Harness != "antigravity" {
				t.Fatalf("expected harness antigravity, got %q", s.Harness)
			}
		}
	}
	if !found {
		t.Fatalf("expected to discover test-session-123, got: %+v", sessions)
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
