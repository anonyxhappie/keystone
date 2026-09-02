package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anonyxhappie/keystone/internal/domain"
)

const Dir = ".keystone"

type Store struct{ Root string }

func New(root string) Store { return Store{Root: root} }

func (s Store) Init(name string, capabilities []domain.Capability) (domain.Project, error) {
	if err := os.MkdirAll(filepath.Join(s.Root, Dir, "requirements"), 0o755); err != nil { return domain.Project{}, err }
	for _, d := range []string{"architecture","decisions","assumptions","work","checkpoints","evidence","learning","policies","manifests"} {
		if err := os.MkdirAll(filepath.Join(s.Root, Dir, d), 0o755); err != nil { return domain.Project{}, err }
	}
	p := domain.Project{SchemaVersion:"1", ID:"project-1", Root:s.Root, Name:name, Capabilities:capabilities}
	if err := s.write("project.json", p); err != nil { return domain.Project{}, err }
	state := map[string]any{"schemaVersion":"1", "lifecycle":"DEVELOPMENT", "workOrder":nil, "checkpoint":nil}
	if err := s.write("state.json", state); err != nil { return domain.Project{}, err }
	return p, nil
}

func (s Store) Write(path string, v any) error { return s.write(path, v) }
func (s Store) Read(path string, v any) error {
	b, err := os.ReadFile(filepath.Join(s.Root, Dir, path)); if err != nil { return err }
	return json.Unmarshal(b, v)
}
func (s Store) write(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  "); if err != nil { return err }
	p := filepath.Join(s.Root, Dir, path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { return err }
	if err := os.WriteFile(p, append(b, '\n'), 0o644); err != nil { return fmt.Errorf("write %s: %w", p, err) }
	return nil
}
