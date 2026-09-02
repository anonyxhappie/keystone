package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anonyxhappie/keystone/internal/domain"
)

const Dir = ".keystone"

type Store struct{ Root string }

func New(root string) Store { return Store{Root: root} }

func (s Store) Init(name string, capabilities []domain.Capability) (domain.Project, error) {
	if err := os.MkdirAll(filepath.Join(s.Root, Dir, "requirements"), 0o755); err != nil {
		return domain.Project{}, err
	}
	for _, d := range []string{"architecture", "decisions", "assumptions", "work", "checkpoints", "evidence", "learning", "policies", "manifests"} {
		if err := os.MkdirAll(filepath.Join(s.Root, Dir, d), 0o755); err != nil {
			return domain.Project{}, err
		}
	}
	p := domain.Project{SchemaVersion: "1", ID: "project-1", Root: s.Root, Name: name, Capabilities: capabilities}
	if err := s.write("project.json", p); err != nil {
		return domain.Project{}, err
	}
	state := map[string]any{"schemaVersion": "1", "lifecycle": "DEVELOPMENT", "workOrder": nil, "checkpoint": nil}
	if err := s.write("state.json", state); err != nil {
		return domain.Project{}, err
	}
	return p, nil
}

func (s Store) Write(path string, v any) error { return s.write(path, v) }
func (s Store) Read(path string, v any) error {
	p, err := s.safePath(path)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
func (s Store) write(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	p, err := s.safePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".keystone-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, p); err != nil {
		return fmt.Errorf("write %s: %w", p, err)
	}
	return nil
}

func (s Store) safePath(path string) (string, error) {
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
		return "", fmt.Errorf("invalid state path %q", path)
	}
	base, err := filepath.Abs(filepath.Join(s.Root, Dir))
	if err != nil {
		return "", err
	}
	p, err := filepath.Abs(filepath.Join(base, path))
	if err != nil {
		return "", err
	}
	if p != base && !strings.HasPrefix(p, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("state path escapes .keystone: %q", path)
	}
	return p, nil
}
