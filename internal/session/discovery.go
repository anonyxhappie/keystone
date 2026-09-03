package session

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session describes a conversation session from Keystone or a harness.
type Session struct {
	ID           string    `json:"id"`
	Harness      string    `json:"harness"`
	Title        string    `json:"title"`
	Preview      string    `json:"preview"`
	Workspace    string    `json:"workspace"`
	LastModified time.Time `json:"lastModified"`
	Active       bool      `json:"active,omitempty"`
}

// Project describes a local project workspace.
type Project struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	LastModified time.Time `json:"lastModified"`
	Active       bool      `json:"active"`
}

// matchesWorkspace returns true if the session's recorded workspace matches workspaceRoot.
func matchesWorkspace(sessionWorkspace, workspaceRoot string) bool {
	if workspaceRoot == "" {
		return true
	}
	cleanRoot := filepath.Clean(workspaceRoot)
	ws := strings.TrimSpace(sessionWorkspace)
	if ws == "" {
		return false
	}

	// Handle JSON array string e.g. ["file:///Users/akshay/Desktop/code/losal"]
	if strings.HasPrefix(ws, "[") {
		var uris []string
		if err := json.Unmarshal([]byte(ws), &uris); err == nil {
			for _, u := range uris {
				cleanU := filepath.Clean(strings.TrimPrefix(u, "file://"))
				if cleanU == cleanRoot || strings.HasPrefix(cleanU, cleanRoot) {
					return true
				}
			}
			return false
		}
	}

	cleanWs := filepath.Clean(strings.TrimPrefix(ws, "file://"))
	return cleanWs == cleanRoot || strings.HasPrefix(cleanWs, cleanRoot)
}

// DiscoverSessions returns recent sessions from Keystone and all installed harnesses,
// strictly scoped to the specified workspaceRoot when provided.
func DiscoverSessions(workspaceRoot string) []Session {
	all := []Session{}
	seen := map[string]bool{}

	// 1. Antigravity conversations
	agySessions := discoverAntigravitySessions(workspaceRoot)
	for _, s := range agySessions {
		if !seen[s.ID] && matchesWorkspace(s.Workspace, workspaceRoot) {
			seen[s.ID] = true
			all = append(all, s)
		}
	}

	// 2. Codex sessions
	codexSessions := discoverCodexSessions(workspaceRoot)
	for _, s := range codexSessions {
		if !seen[s.ID] && matchesWorkspace(s.Workspace, workspaceRoot) {
			seen[s.ID] = true
			all = append(all, s)
		}
	}

	// Sort newest first
	sort.Slice(all, func(i, j int) bool {
		return all[i].LastModified.After(all[j].LastModified)
	})

	return all
}

func discoverAntigravitySessions(workspaceRoot string) []Session {
	var list []Session
	home, err := os.UserHomeDir()
	if err != nil {
		return list
	}

	dbPath := filepath.Join(home, ".gemini", "antigravity-cli", "conversation_summaries.db")
	if _, err := os.Stat(dbPath); err == nil {
		// Use sqlite3 CLI if available
		query := "SELECT conversation_id, title, preview, workspace_uris, datetime(last_modified_time) FROM conversation_summaries ORDER BY last_modified_time DESC LIMIT 100;"
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sqlite3", "-separator", "|", dbPath, query)
		out, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(out))
			for scanner.Scan() {
				parts := strings.Split(scanner.Text(), "|")
				if len(parts) >= 5 {
					cid := strings.TrimSpace(parts[0])
					title := strings.TrimSpace(parts[1])
					preview := strings.TrimSpace(parts[2])
					ws := strings.TrimSpace(parts[3])
					timeStr := strings.TrimSpace(parts[4])

					if strings.Contains(preview, "This version of Antigravity CLI is no longer supported") {
						continue
					}

					name := title
					if name == "" {
						name = preview
					}
					if name == "" {
						name = "Antigravity Session " + cid[:8]
					}
					var t time.Time
					for _, layout := range []string{
						"2006-01-02 15:04:05.999999-07:00",
						"2006-01-02 15:04:05.999999+00:00",
						"2006-01-02 15:04:05.999999",
						"2006-01-02 15:04:05",
						time.RFC3339,
						time.RFC3339Nano,
					} {
						if parsed, err := time.Parse(layout, timeStr); err == nil {
							t = parsed
							break
						}
					}
					list = append(list, Session{
						ID:           cid,
						Harness:      "antigravity",
						Title:        name,
						Preview:      preview,
						Workspace:    ws,
						LastModified: t,
					})
				}
			}
			if len(list) > 0 {
				return list
			}
		}
	}

	return list
}

func discoverCodexSessions(workspaceRoot string) []Session {
	var list []Session
	home, err := os.UserHomeDir()
	if err != nil {
		return list
	}

	// 1. Query state_5.sqlite if present to get real cwd per thread
	dbPath := filepath.Join(home, ".codex", "state_5.sqlite")
	if _, err := os.Stat(dbPath); err == nil {
		query := "SELECT id, title, cwd, datetime(updated_at, 'unixepoch') FROM threads WHERE archived = 0 ORDER BY updated_at DESC LIMIT 100;"
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sqlite3", "-separator", "|", dbPath, query)
		out, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(out))
			for scanner.Scan() {
				parts := strings.Split(scanner.Text(), "|")
				if len(parts) >= 4 {
					id := strings.TrimSpace(parts[0])
					rawTitle := parts[1]
					cwd := strings.TrimSpace(parts[2])
					timeStr := strings.TrimSpace(parts[3])

					// Sanitize title: pick first non-empty meaningful line
					title := ""
					for _, line := range strings.Split(rawTitle, "\n") {
						trimmed := strings.TrimSpace(line)
						if trimmed != "" && !strings.HasPrefix(trimmed, ">>>") && !strings.HasPrefix(trimmed, "[") {
							title = trimmed
							break
						}
					}
					if title == "" {
						title = "Codex Session " + id[:8]
					}
					var t time.Time
					for _, layout := range []string{
						"2006-01-02 15:04:05",
						time.RFC3339,
					} {
						if parsed, err := time.Parse(layout, timeStr); err == nil {
							t = parsed
							break
						}
					}
					list = append(list, Session{
						ID:           id,
						Harness:      "codex",
						Title:        title,
						Workspace:    cwd,
						LastModified: t,
					})
				}
			}
			if len(list) > 0 {
				return list
			}
		}
	}

	// 2. Fallback to session_index.jsonl
	indexFile := filepath.Join(home, ".codex", "session_index.jsonl")
	f, err := os.Open(indexFile)
	if err != nil {
		return list
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item struct {
			ID         string `json:"id"`
			ThreadName string `json:"thread_name"`
			UpdatedAt  string `json:"updated_at"`
		}
		if err := json.Unmarshal([]byte(line), &item); err == nil && item.ID != "" {
			t, _ := time.Parse(time.RFC3339, item.UpdatedAt)
			title := item.ThreadName
			if title == "" {
				title = "Codex Session " + item.ID[:8]
			}
			list = append(list, Session{
				ID:           item.ID,
				Harness:      "codex",
				Title:        title,
				Workspace:    workspaceRoot,
				LastModified: t,
			})
		}
	}
	return list
}

// DiscoverProjects returns a list of local projects/workspaces.
func DiscoverProjects(currentRoot string) []Project {
	projects := map[string]Project{}

	// Add current
	if currentRoot != "" {
		projects[currentRoot] = Project{
			Path:         currentRoot,
			Name:         filepath.Base(currentRoot),
			LastModified: time.Now(),
			Active:       true,
		}
	}

	// Read ~/.gemini/projects.json
	if home, err := os.UserHomeDir(); err == nil {
		pPath := filepath.Join(home, ".gemini", "projects.json")
		if data, err := os.ReadFile(pPath); err == nil {
			var cfg struct {
				Projects map[string]string `json:"projects"`
			}
			if err := json.Unmarshal(data, &cfg); err == nil {
				for path, name := range cfg.Projects {
					if _, err := os.Stat(path); err == nil {
						projects[path] = Project{
							Path:         path,
							Name:         name,
							LastModified: time.Now().Add(-24 * time.Hour),
							Active:       path == currentRoot,
						}
					}
				}
			}
		}
	}

	// Scan parent directory of currentRoot (e.g. /Users/akshay/Desktop/code/)
	if currentRoot != "" {
		parent := filepath.Dir(currentRoot)
		if entries, err := os.ReadDir(parent); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
					dirPath := filepath.Join(parent, entry.Name())
					// check for git or keystone
					isProject := false
					if _, err := os.Stat(filepath.Join(dirPath, ".git")); err == nil {
						isProject = true
					}
					if _, err := os.Stat(filepath.Join(dirPath, ".keystone")); err == nil {
						isProject = true
					}
					if isProject {
						info, _ := entry.Info()
						modTime := time.Now()
						if info != nil {
							modTime = info.ModTime()
						}
						projects[dirPath] = Project{
							Path:         dirPath,
							Name:         entry.Name(),
							LastModified: modTime,
							Active:       dirPath == currentRoot,
						}
					}
				}
			}
		}
	}

	var result []Project
	for _, p := range projects {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Active != result[j].Active {
			return result[i].Active
		}
		return result[i].Name < result[j].Name
	})
	return result
}
