package project

import (
	"os"
	"path/filepath"
	"sort"
)

func InstructionFiles(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, e os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() {
			if path != root && (e.Name() == ".git" || e.Name() == ".keystone" || e.Name() == "node_modules" || e.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		name := e.Name()
		if name == "AGENTS.md" || name == "CLAUDE.md" || name == ".cursorrules" {
			rel, _ := filepath.Rel(root, path)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}
func Topology(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, e os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() {
			if path != root && (e.Name() == ".git" || e.Name() == ".keystone" || e.Name() == "node_modules" || e.Name() == "vendor") {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(root, path)
			if rel != "." {
				out = append(out, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	sort.Strings(out)
	if len(out) > 256 {
		out = out[:256]
	}
	return out
}
