package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
)

func Detect(root string) []domain.Capability {
	var out []domain.Capability
	seen := map[string]bool{}
	add := func(kind, name, source string, values ...string) {
		key := kind + "\x00" + name
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, domain.Capability{Kind: kind, Name: name, Source: source, Values: values})
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		add("vcs", "git", ".git")
	}
	files := map[string]struct{ kind, name string }{
		"package.json": {"language", "javascript/typescript"}, "pyproject.toml": {"language", "python"}, "requirements.txt": {"language", "python"}, "go.mod": {"language", "go"}, "Cargo.toml": {"language", "rust"}, "pom.xml": {"language", "java"}, "build.gradle": {"language", "java"},
		"playwright.config.ts": {"browser", "playwright"}, "playwright.config.js": {"browser", "playwright"}, "vitest.config.ts": {"test", "vitest"}, "pytest.ini": {"test", "pytest"}, "jest.config.js": {"test", "jest"}, "Makefile": {"build", "make"}, "Dockerfile": {"infrastructure", "docker"}, "tsconfig.json": {"typecheck", "typescript"}, ".golangci.yml": {"lint", "golangci-lint"}, "eslint.config.js": {"lint", "eslint"}, "eslint.config.mjs": {"lint", "eslint"},
	}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == ".keystone" || entry.Name() == "node_modules" || entry.Name() == "vendor" || entry.Name() == ".venv") {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		rel, _ := filepath.Rel(root, path)
		if spec, ok := files[name]; ok {
			add(spec.kind, spec.name, filepath.ToSlash(rel))
			if name == "go.mod" {
				add("test", "go", filepath.ToSlash(rel))
				add("build", "go-build", filepath.ToSlash(rel))
			}
			if name == "package.json" {
				var packageFile struct {
					Dependencies    map[string]json.RawMessage `json:"dependencies"`
					DevDependencies map[string]json.RawMessage `json:"devDependencies"`
				}
				if content, readErr := os.ReadFile(path); readErr == nil && json.Unmarshal(content, &packageFile) == nil {
					dependencies := map[string]json.RawMessage{}
					for dependency, value := range packageFile.Dependencies {
						dependencies[dependency] = value
					}
					for dependency, value := range packageFile.DevDependencies {
						dependencies[dependency] = value
					}
					for _, framework := range []string{"react", "next", "express", "fastify", "vue", "svelte"} {
						if _, found := dependencies[framework]; found {
							add("framework", framework, filepath.ToSlash(rel))
						}
					}
				}
			}
		}
		if strings.HasSuffix(name, ".csproj") {
			add("language", "dotnet", filepath.ToSlash(rel))
		}
		low := strings.ToLower(rel)
		if strings.Contains(low, "migration") {
			add("database", "migrations", filepath.ToSlash(rel))
		}
		if strings.HasPrefix(name, "AGENTS") || name == "CLAUDE.md" || name == ".cursorrules" {
			add("instructions", filepath.ToSlash(rel), filepath.ToSlash(rel))
		}
		return nil
	})
	if _, err := os.Stat(filepath.Join(root, ".github", "workflows")); err == nil {
		add("ci", "github-actions", ".github/workflows")
	}
	if _, err := os.Stat(filepath.Join(root, ".eslintrc")); err == nil {
		add("lint", "eslint", ".eslintrc")
	}
	return out
}
