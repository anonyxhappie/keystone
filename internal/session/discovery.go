package session

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/state"
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

// DiscoverSessions returns recent sessions from Keystone and all installed harnesses.
func DiscoverSessions(workspaceRoot string) []Session {
	all := []Session{}
	seen := map[string]bool{}

	// 1. Keystone local project sessions
	keystoneSessions := discoverKeystoneSessions(workspaceRoot)
	for _, s := range keystoneSessions {
		if !seen[s.ID] {
			seen[s.ID] = true
			all = append(all, s)
		}
	}

	// 2. Antigravity conversations
	agySessions := discoverAntigravitySessions(workspaceRoot)
	for _, s := range agySessions {
		if !seen[s.ID] {
			seen[s.ID] = true
			all = append(all, s)
		}
	}

	// 3. Codex sessions
	codexSessions := discoverCodexSessions(workspaceRoot)
	for _, s := range codexSessions {
		if !seen[s.ID] {
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

func discoverKeystoneSessions(root string) []Session {
	var list []Session
	sessionDir := filepath.Join(root, state.Dir, "harness-sessions")
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return list
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sessionDir, entry.Name()))
		if err != nil {
			continue
		}
		var hs domain.HarnessSession
		if err := json.Unmarshal(data, &hs); err != nil {
			continue
		}
		modTime := hs.StartedAt
		if info, err := entry.Info(); err == nil && modTime.IsZero() {
			modTime = info.ModTime()
		}
		title := fmt.Sprintf("Keystone Run (%s)", hs.Status)
		list = append(list, Session{
			ID:           hs.ID,
			Harness:      hs.HarnessID,
			Title:        title,
			Workspace:    root,
			LastModified: modTime,
		})
	}
	return list
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
		query := "SELECT conversation_id, title, preview, workspace_uris, datetime(last_modified_time) FROM conversation_summaries ORDER BY last_modified_time DESC LIMIT 30;"
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sqlite3", "-separator", "|", dbPath, query)
		out, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(out))
			for scanner.Scan() {
				parts := strings.Split(scanner.Text(), "|")
				if len(parts) >= 5 {
					cid := parts[0]
					title := parts[1]
					preview := parts[2]
					ws := parts[3]
					timeStr := parts[4]
					if title == "" {
						title = preview
					}
					if title == "" {
						title = "Antigravity Conversation"
					}
					t, _ := time.Parse("2006-01-02 15:04:05", timeStr)
					list = append(list, Session{
						ID:           cid,
						Harness:      "antigravity",
						Title:        title,
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

	// Fallback to conversations directory scan
	convDir := filepath.Join(home, ".gemini", "antigravity-cli", "conversations")
	entries, err := os.ReadDir(convDir)
	if err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".db") {
				cid := strings.TrimSuffix(entry.Name(), ".db")
				if info, err := entry.Info(); err == nil {
					list = append(list, Session{
						ID:           cid,
						Harness:      "antigravity",
						Title:        "Antigravity Conversation " + cid[:8],
						Workspace:    workspaceRoot,
						LastModified: info.ModTime(),
					})
				}
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
				title = "Codex Thread " + item.ID[:8]
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
