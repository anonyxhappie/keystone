package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Preferences stores user choices across interactive sessions.
type Preferences struct {
	LastHarness     string            `json:"last_harness,omitempty"`
	LastProject     string            `json:"last_project,omitempty"`
	ProjectSessions map[string]string `json:"project_sessions,omitempty"`
	ProjectHarness  map[string]string `json:"project_harness,omitempty"`
}

var (
	prefMu sync.Mutex
)

func prefPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".keystone", "preferences.json")
	}
	dir := filepath.Join(home, ".keystone")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "preferences.json")
}

// LoadPreferences reads preferences from disk.
func LoadPreferences() Preferences {
	prefMu.Lock()
	defer prefMu.Unlock()

	p := Preferences{
		ProjectSessions: make(map[string]string),
		ProjectHarness:  make(map[string]string),
	}

	data, err := os.ReadFile(prefPath())
	if err != nil {
		return p
	}

	_ = json.Unmarshal(data, &p)
	if p.ProjectSessions == nil {
		p.ProjectSessions = make(map[string]string)
	}
	if p.ProjectHarness == nil {
		p.ProjectHarness = make(map[string]string)
	}
	return p
}

// SavePreferences persists updated preferences to disk.
func SavePreferences(p Preferences) error {
	prefMu.Lock()
	defer prefMu.Unlock()

	if p.ProjectSessions == nil {
		p.ProjectSessions = make(map[string]string)
	}
	if p.ProjectHarness == nil {
		p.ProjectHarness = make(map[string]string)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(prefPath(), data, 0600)
}

// GetLastSession returns the last selected session for a project.
func GetLastSession(projectRoot string) string {
	prefs := LoadPreferences()
	if sess, ok := prefs.ProjectSessions[projectRoot]; ok && sess != "" {
		return sess
	}
	return ""
}

// SetLastSession updates the last selected session for a project.
func SetLastSession(projectRoot, sessionID string) {
	prefs := LoadPreferences()
	prefs.ProjectSessions[projectRoot] = sessionID
	prefs.LastProject = projectRoot
	_ = SavePreferences(prefs)
}

// GetLastHarness returns the last selected harness for a project or globally.
func GetLastHarness(projectRoot string) string {
	prefs := LoadPreferences()
	if h, ok := prefs.ProjectHarness[projectRoot]; ok && h != "" {
		return h
	}
	if prefs.LastHarness != "" {
		return prefs.LastHarness
	}
	return "auto"
}

// SetLastHarness updates the last selected harness.
func SetLastHarness(projectRoot, harness string) {
	prefs := LoadPreferences()
	prefs.LastHarness = harness
	if projectRoot != "" {
		prefs.ProjectHarness[projectRoot] = harness
	}
	_ = SavePreferences(prefs)
}
