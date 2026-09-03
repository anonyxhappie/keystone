package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func TestGenerateTurn1Prompt(t *testing.T) {
	order := domain.WorkOrder{
		ID:            "WO-123",
		RunID:         "RUN-123",
		SourceRequest: "Implement Losal milestone 1",
		ReadOnly:      false,
	}
	packet := domain.WorkPacket{
		WorkOrderID:        "WO-123",
		Requirements:       []string{"REQ-1"},
		CompletionCriteria: []string{"all tests pass"},
		Validation:         []string{"go"},
	}

	p := Generate(order, packet, 1, "antigravity", "", "manifest-1.json", nil, nil)
	if p.ID != "PRM-RUN-123-1" {
		t.Fatalf("unexpected prompt ID: %s", p.ID)
	}
	if p.Turn != 1 {
		t.Fatalf("unexpected turn: %d", p.Turn)
	}
	if p.CreatedAt.IsZero() {
		t.Fatalf("expected non-zero CreatedAt")
	}
	if !strings.Contains(p.Content, "Turn #1 Direction") {
		t.Fatalf("content missing turn 1 direction: %s", p.Content)
	}
	if !strings.Contains(p.Content, "Implement Losal milestone 1") {
		t.Fatalf("content missing request: %s", p.Content)
	}
}

func TestGenerateTurn2RecoveryPrompt(t *testing.T) {
	order := domain.WorkOrder{
		ID:            "WO-123",
		RunID:         "RUN-123",
		SourceRequest: "Run migration and tests",
	}
	packet := domain.WorkPacket{
		WorkOrderID: "WO-123",
		Validation:  []string{"go"},
	}
	diag := &domain.FailureDiagnosis{
		Class:                domain.FailureEnvironmentBlocker,
		Summary:              "PostgreSQL is unreachable at localhost:5432",
		RecoveryInstruction:  "Diagnose and recover PostgreSQL container.",
		RecoverableByHarness: true,
	}

	p := Generate(order, packet, 2, "antigravity", "sess-456", "manifest-2.json", diag, nil)
	if p.Turn != 2 {
		t.Fatalf("unexpected turn: %d", p.Turn)
	}
	if p.HarnessSessionID != "sess-456" {
		t.Fatalf("unexpected session ID: %s", p.HarnessSessionID)
	}
	if !strings.Contains(p.Content, "Turn #2 Direction: PostgreSQL is unreachable at localhost:5432") {
		t.Fatalf("content missing turn 2 direction: %s", p.Content)
	}
	if !strings.Contains(p.Content, "Diagnose and recover PostgreSQL container") {
		t.Fatalf("content missing recovery instruction: %s", p.Content)
	}
}

func TestGenerateReadOnlyPrompt(t *testing.T) {
	order := domain.WorkOrder{
		ID:            "WO-123",
		RunID:         "RUN-123",
		SourceRequest: "Inspect repo. Do not modify files.",
		ReadOnly:      true,
	}
	packet := domain.WorkPacket{
		WorkOrderID: "WO-123",
		ReadOnly:    true,
	}

	p := Generate(order, packet, 1, "codex", "", "manifest-1.json", nil, nil)
	if !strings.Contains(p.Content, "CRITICAL POLICY CONSTRAINT: This is an explicit READ-ONLY execution") {
		t.Fatalf("content missing read-only warning: %s", p.Content)
	}
}

func TestTimestampsNeverZero(t *testing.T) {
	order := domain.WorkOrder{
		ID:            "WO-1",
		RunID:         "RUN-1",
		SourceRequest: "test",
	}
	p := Generate(order, domain.WorkPacket{}, 1, "auto", "", "", nil, nil)
	if p.CreatedAt.Format(time.RFC3339) == "0001-01-01T00:00:00Z" {
		t.Fatalf("CreatedAt is zero timestamp!")
	}
}
