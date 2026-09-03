package harness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createMockBinary(t *testing.T, dir string, name string, version string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  echo \"%s\"\n  exit 0\nfi\nexit 0\n", version)
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock binary: %v", err)
	}
	return path
}

func TestSelectHarnessExplicitAvailableCodex(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	createMockBinary(t, binDir, "codex", "codex-cli 0.152.1")
	t.Setenv("PATH", binDir)
	ctx := context.Background()

	adapter, selection, err := SelectHarness(ctx, root, "codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter == nil {
		t.Fatalf("expected non-nil adapter")
	}
	if selection.SelectionMode != "explicit" {
		t.Fatalf("expected selectionMode 'explicit', got %q", selection.SelectionMode)
	}
	if selection.SelectedHarness != "codex" {
		t.Fatalf("expected selectedHarness 'codex', got %q", selection.SelectedHarness)
	}
	if selection.PolicyDecision != "ALLOW" {
		t.Fatalf("expected policyDecision 'ALLOW', got %q", selection.PolicyDecision)
	}
}

func TestSelectHarnessExplicitAvailableAntigravity(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	createMockBinary(t, binDir, "agy", "agy 1.1.24")
	t.Setenv("PATH", binDir)
	ctx := context.Background()

	adapter, selection, err := SelectHarness(ctx, root, "antigravity")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter == nil {
		t.Fatalf("expected non-nil adapter")
	}
	if selection.SelectionMode != "explicit" {
		t.Fatalf("expected selectionMode 'explicit', got %q", selection.SelectionMode)
	}
	if selection.SelectedHarness != "antigravity" {
		t.Fatalf("expected selectedHarness 'antigravity', got %q", selection.SelectedHarness)
	}
	if selection.PolicyDecision != "ALLOW" {
		t.Fatalf("expected policyDecision 'ALLOW', got %q", selection.PolicyDecision)
	}
}

func TestSelectHarnessExplicitUnavailableFailsClosedWithoutFallback(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	// only agy exists, but user asks for codex
	createMockBinary(t, binDir, "agy", "agy 1.1.24")
	t.Setenv("PATH", binDir)
	ctx := context.Background()

	adapter, selection, err := SelectHarness(ctx, root, "codex")
	if err == nil {
		t.Fatalf("expected error when explicit harness is unavailable, got nil")
	}
	if adapter != nil {
		t.Fatalf("expected nil adapter on unavailable explicit harness, got %+v", adapter)
	}
	if selection.SelectionMode != "explicit" {
		t.Fatalf("expected selectionMode 'explicit', got %q", selection.SelectionMode)
	}
	if selection.PolicyDecision != "REQUIRE_APPROVAL" {
		t.Fatalf("expected policyDecision 'REQUIRE_APPROVAL', got %q", selection.PolicyDecision)
	}
	if !strings.Contains(selection.SelectionReason, "unavailable") {
		t.Fatalf("expected reason to state unavailable: %q", selection.SelectionReason)
	}
	if !strings.Contains(err.Error(), "antigravity") {
		t.Fatalf("expected error to list available alternatives: %v", err)
	}
}

func TestSelectHarnessInvalidNameReturnsError(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()

	_, selection, err := SelectHarness(ctx, root, "magic-nonexistent-agent")
	if err == nil {
		t.Fatalf("expected error for invalid harness")
	}
	if selection.PolicyDecision != "REQUIRE_APPROVAL" {
		t.Fatalf("expected REQUIRE_APPROVAL for invalid harness, got %q", selection.PolicyDecision)
	}
	if !strings.Contains(err.Error(), "invalid harness") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSelectHarnessAutoMode(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	createMockBinary(t, binDir, "agy", "agy 1.1.24")
	createMockBinary(t, binDir, "codex", "codex-cli 0.152.1")
	t.Setenv("PATH", binDir)
	ctx := context.Background()

	adapter, selection, err := SelectHarness(ctx, root, "auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter == nil {
		t.Fatalf("expected non-nil adapter in auto mode")
	}
	if selection.SelectionMode != "auto" {
		t.Fatalf("expected selectionMode 'auto', got %q", selection.SelectionMode)
	}
	if selection.SelectedHarness == "" {
		t.Fatalf("expected selectedHarness to be populated")
	}
	if selection.SelectionReason == "" {
		t.Fatalf("expected selectionReason to be populated")
	}
	if selection.PolicyDecision != "ALLOW" {
		t.Fatalf("expected policyDecision 'ALLOW', got %q", selection.PolicyDecision)
	}
}
