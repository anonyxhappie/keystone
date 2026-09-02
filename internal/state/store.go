package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/project"
	"github.com/anonyxhappie/keystone/internal/runtime"
)

const Dir = ".keystone"

type Store struct{ Root string }

func New(root string) Store { return Store{Root: root} }

func (s Store) Init(name string, capabilities []domain.Capability) (domain.Project, error) {
	if err := os.MkdirAll(filepath.Join(s.Root, Dir, "requirements"), 0o700); err != nil {
		return domain.Project{}, err
	}
	for _, d := range []string{"architecture", "decisions", "assumptions", "work", "checkpoints", "evidence", "learning", "policies", "manifests", "artifacts", "harnesses", "harness-sessions", "harness-runs", "validations", "releases", "deployments", "environments", "incidents", "approvals", "control"} {
		if err := os.MkdirAll(filepath.Join(s.Root, Dir, d), 0o700); err != nil {
			return domain.Project{}, err
		}
	}
	now := time.Now().UTC()
	p := domain.Project{SchemaVersion: "2", ID: "project-1", Root: s.Root, Name: name, Capabilities: capabilities, InstructionFiles: project.InstructionFiles(s.Root), Topology: project.Topology(s.Root), CreatedAt: now, UpdatedAt: now}
	if err := s.write("project.json", p); err != nil {
		return domain.Project{}, err
	}
	if err := s.Write("policies/default-v1.json", domain.Policy{ID: "default-v1", Name: "default", Version: "1", Rules: map[string]string{"destructive": "approval", "workspace": "confined", "production": "approval"}}); err != nil {
		return domain.Project{}, err
	}
	if err := s.SaveSnapshot(Snapshot{SchemaVersion: "2", Lifecycle: "REQUEST", Machine: runtime.New(), UpdatedAt: now}); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".keystone-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
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
	dir, err := os.Open(filepath.Dir(p))
	if err != nil {
		return err
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		return err
	}
	if err := dir.Close(); err != nil {
		return err
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
	realBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(s.Root)
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", err
	}
	if realBase != filepath.Join(realRoot, Dir) {
		return "", fmt.Errorf(".keystone itself is a symlink: %q", base)
	}
	parent := filepath.Dir(p)
	for {
		if _, statErr := os.Lstat(parent); statErr == nil {
			break
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", fmt.Errorf("state path has no existing parent: %q", path)
		}
		parent = next
	}
	realParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	if realParent != realBase && !strings.HasPrefix(realParent, realBase+string(os.PathSeparator)) {
		return "", fmt.Errorf("state path follows a symlink outside .keystone: %q", path)
	}
	if _, err := os.Lstat(p); err == nil {
		realPath, evalErr := filepath.EvalSymlinks(p)
		if evalErr != nil {
			return "", evalErr
		}
		if realPath != realBase && !strings.HasPrefix(realPath, realBase+string(os.PathSeparator)) {
			return "", fmt.Errorf("state path follows a symlink outside .keystone: %q", path)
		}
	}
	return p, nil
}
