package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Preferences stores user choices across interactive sessions.
type Preferences struct {
	LastHarness     string            `json:"last_harness,omitempty"`
	LastModel       string            `json:"last_model,omitempty"`
	LastProject     string            `json:"last_project,omitempty"`
	ProjectSessions map[string]string `json:"project_sessions,omitempty"`
	ProjectHarness  map[string]string `json:"project_harness,omitempty"`
	ProjectModels   map[string]string `json:"project_models,omitempty"`
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
		ProjectModels:   make(map[string]string),
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
	if p.ProjectModels == nil {
		p.ProjectModels = make(map[string]string)
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
	if p.ProjectModels == nil {
		p.ProjectModels = make(map[string]string)
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

// GetLastModel returns the last selected model for a project or globally, with harness defaults.
func GetLastModel(projectRoot, harness string) string {
	prefs := LoadPreferences()
	if m, ok := prefs.ProjectModels[projectRoot]; ok && m != "" {
		return m
	}
	if prefs.LastModel != "" {
		return prefs.LastModel
	}
	switch strings.ToLower(harness) {
	case "antigravity", "agy":
		return "gemini-3.8-flash-high"
	case "codex":
		return "o3"
	default:
		return "auto"
	}
}

// SetLastModel updates the last selected model for a project.
func SetLastModel(projectRoot, model string) {
	prefs := LoadPreferences()
	prefs.LastModel = model
	if projectRoot != "" {
		if prefs.ProjectModels == nil {
			prefs.ProjectModels = make(map[string]string)
		}
		prefs.ProjectModels[projectRoot] = model
	}
	_ = SavePreferences(prefs)
}

// ModelDisplayName returns a user-friendly label for a given model ID.
func ModelDisplayName(modelID, harness string) string {
	switch strings.ToLower(modelID) {
	case "gemini-3.8-flash-high", "gemini-3.8-flash":
		return "Gemini 3.8 Flash (High)"
	case "gemini-3.7-flash-high", "gemini-3.7-flash":
		return "Gemini 3.7 Flash (High)"
	case "gemini-3.1-pro-high", "gemini-3.1-pro":
		return "Gemini 3.1 Pro (High)"
	case "claude-sonnet-4-6":
		return "Claude Sonnet 4.6 (Thinking)"
	case "claude-opus-4-6-thinking":
		return "Claude Opus 4.6 (Thinking)"
	case "o3":
		return "o3 (High)"
	case "o3-mini":
		return "o3-mini"
	case "gpt-4.5":
		return "gpt-4.5"
	case "gpt-4o":
		return "gpt-4o"
	case "auto", "":
		if strings.EqualFold(harness, "codex") {
			return "Codex (o3 / High)"
		}
		if strings.EqualFold(harness, "antigravity") || strings.EqualFold(harness, "agy") {
			return "Gemini 3.8 Flash (High)"
		}
		return "Supervised Autonomy"
	default:
		return modelID
	}
}
