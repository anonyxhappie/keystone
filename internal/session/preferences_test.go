package session

import (
	"testing"
)

func TestPreferencesRoundTrip(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	prefs := LoadPreferences()
	if prefs.LastHarness != "" {
		t.Errorf("expected empty LastHarness initially, got %q", prefs.LastHarness)
	}

	SetLastHarness("/path/to/project", "antigravity")
	SetLastSession("/path/to/project", "sess-12345")
	SetLastModel("/path/to/project", "gemini-3.8-flash-high")

	loadedHarness := GetLastHarness("/path/to/project")
	if loadedHarness != "antigravity" {
		t.Errorf("expected antigravity, got %q", loadedHarness)
	}

	loadedSession := GetLastSession("/path/to/project")
	if loadedSession != "sess-12345" {
		t.Errorf("expected sess-12345, got %q", loadedSession)
	}

	loadedModel := GetLastModel("/path/to/project", "antigravity")
	if loadedModel != "gemini-3.8-flash-high" {
		t.Errorf("expected gemini-3.8-flash-high, got %q", loadedModel)
	}
	disp := ModelDisplayName(loadedModel, "antigravity")
	if disp != "Gemini 3.8 Flash (High)" {
		t.Errorf("expected Gemini 3.8 Flash (High), got %q", disp)
	}
}
