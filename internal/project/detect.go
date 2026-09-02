package project

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func Detect(root string) []domain.Capability {
	var out []domain.Capability
	add := func(kind, name, source string, values ...string) { out = append(out, domain.Capability{Kind:kind, Name:name, Source:source, Values:values}) }
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil { add("vcs", "git", ".git") }
	files := map[string]struct{kind,name string}{
		"package.json":{"language","javascript/typescript"}, "pyproject.toml":{"language","python"}, "requirements.txt":{"language","python"}, "go.mod":{"language","go"}, "Cargo.toml":{"language","rust"}, "pom.xml":{"language","java"}, "build.gradle":{"language","java"}, "*.csproj":{"language","dotnet"},
		"playwright.config.ts":{"browser","playwright"}, "playwright.config.js":{"browser","playwright"}, "vitest.config.ts":{"test","vitest"}, "pytest.ini":{"test","pytest"},
	}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() { continue }
		if spec, ok := files[e.Name()]; ok { add(spec.kind, spec.name, e.Name()) }
		if strings.HasSuffix(e.Name(), ".csproj") { add("language", "dotnet", e.Name()) }
	}
	if _, err := os.Stat(filepath.Join(root, ".github", "workflows")); err == nil { add("ci", "github-actions", ".github/workflows") }
	return out
}
