package context

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
)

// Compile preserves the V1 API while selecting a small, provenance-bearing set
// of repository files. It never reads file contents into the work packet.
func Compile(root string, packet domain.WorkPacket) domain.WorkPacket {
	return CompileWithImpact(root, packet, nil)
}

func CompileWithImpact(root string, packet domain.WorkPacket, changed []string) domain.WorkPacket {
	packet.SchemaVersion = "2"
	selected := map[string]domain.ContextRef{}
	add := func(path, typ, reason, source string, relevance float64) {
		path = filepath.ToSlash(filepath.Clean(path))
		if path == "." || strings.HasPrefix(path, "../") {
			return
		}
		if old, ok := selected[path]; ok {
			if relevance > old.Relevance {
				old.Relevance = relevance
				selected[path] = old
			}
			return
		}
		selected[path] = domain.ContextRef{Type: typ, Path: path, Reason: reason, Source: source, Relevance: relevance}
	}
	for _, file := range changed {
		add(file, "changed-file", "directly changed by current work", "git", 1.0)
	}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == ".keystone" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		name := entry.Name()
		low := strings.ToLower(name)
		if name == "README.md" || name == "AGENTS.md" || name == "CLAUDE.md" || name == ".cursorrules" {
			add(rel, "instruction", "project instructions and constraints", "repository", 0.95)
		}
		if strings.Contains(low, "architecture") || strings.Contains(low, "decision") {
			add(rel, "architecture", "architecture or recorded decision", "repository", 0.8)
		}
		if strings.HasSuffix(low, "_test.go") || strings.HasSuffix(low, ".test.ts") || strings.HasPrefix(low, "test_") {
			add(rel, "test", "potentially relevant validation", "repository", 0.55)
		}
		return nil
	})
	words := strings.Fields(strings.ToLower(packet.Objective))
	files := make([]string, 0, len(selected))
	for path := range selected {
		files = append(files, path)
	}
	sort.Strings(files)
	for _, path := range files {
		base := strings.ToLower(filepath.Base(path))
		for _, word := range words {
			if len(word) > 3 && strings.Contains(base, word) {
				ref := selected[path]
				if ref.Relevance < 0.75 {
					ref.Relevance = 0.75
				}
				if ref.Reason == "" {
					ref.Reason = "objective term matches file name"
				}
				selected[path] = ref
				break
			}
		}
	}
	refs := make([]domain.ContextRef, 0, len(selected))
	for _, path := range files {
		refs = append(refs, selected[path])
	}
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Relevance == refs[j].Relevance {
			return refs[i].Path < refs[j].Path
		}
		return refs[i].Relevance > refs[j].Relevance
	})
	if len(refs) > 64 {
		refs = refs[:64]
	}
	for i := range refs {
		refs[i].TokenEstimate = estimate(root, refs[i].Path)
	}
	packet.Context = refs
	return packet
}

func estimate(root, path string) int {
	info, err := os.Stat(filepath.Join(root, path))
	if err != nil {
		return 0
	}
	return int(info.Size() / 4)
}
