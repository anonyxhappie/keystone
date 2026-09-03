package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func TestCompileWithImpactRanksChangedAndInstructionContext(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "internal"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("rules"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "auth.go"), []byte("package internal"), 0644); err != nil {
		t.Fatal(err)
	}
	p := CompileWithImpact(root, domain.WorkPacket{Objective: "auth change"}, []string{"internal/auth.go"})
	if len(p.Context) < 2 || p.Context[0].Path != "internal/auth.go" || p.Context[0].Reason == "" {
		t.Fatalf("unexpected context: %+v", p.Context)
	}
}

func TestContextUnderBudget(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("Project readme content"), 0644)
	_ = os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("Contract instructions"), 0644)
	_ = os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package main\nfunc TestMain(t *testing.T){}"), 0644)

	packet := domain.WorkPacket{Objective: "audit test"}
	budget := 10000

	p, err := PlanContext(root, packet, nil, budget)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ContextTokens > budget {
		t.Fatalf("expected context tokens %d <= budget %d", p.ContextTokens, budget)
	}
	if len(p.Context) != 3 {
		t.Fatalf("expected 3 context items, got %d", len(p.Context))
	}
	for _, d := range p.ContextDecisions {
		if d.Action != "retained" {
			t.Fatalf("expected all decisions to be retained, got: %+v", d)
		}
	}
}

func TestContextSlightlyOverBudget(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte(strings.Repeat("a", 1000)), 0644) // 250 tokens
	_ = os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(strings.Repeat("b", 1000)), 0644) // 250 tokens
	_ = os.Mkdir(filepath.Join(root, "tests"), 0755)
	_ = os.WriteFile(filepath.Join(root, "tests", "unit_test.go"), []byte("package tests\nfunc TestUnit(t *testing.T){}\n"+strings.Repeat("// comment\n", 500)), 0644) // ~1300 tokens

	packet := domain.WorkPacket{Objective: "audit project"}
	budget := 800 // less than 250 + 250 + 1300 = 1800

	p, err := PlanContext(root, packet, nil, budget)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ContextTokens > budget {
		t.Fatalf("expected context tokens %d <= budget %d", p.ContextTokens, budget)
	}
	// The test file should have been compressed
	foundCompressed := false
	for _, c := range p.Context {
		if c.Path == "tests/unit_test.go" && c.Compressed {
			foundCompressed = true
			if c.Summary == "" {
				t.Fatalf("expected non-empty summary for compressed file")
			}
		}
	}
	if !foundCompressed {
		t.Fatalf("expected tests/unit_test.go to be compressed, got context: %+v", p.Context)
	}
}

func TestContextHeavilyOverBudget(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte(strings.Repeat("a", 1000)), 0644) // 250 tokens
	_ = os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(strings.Repeat("b", 1000)), 0644) // 250 tokens
	_ = os.Mkdir(filepath.Join(root, "tests"), 0755)

	// Create 20 large test files, each ~2000 tokens = ~40,000 tokens total
	for i := 1; i <= 20; i++ {
		content := fmt.Sprintf("package tests\nfunc TestFeature%d(t *testing.T){}\n%s", i, strings.Repeat("x := 1\n", 1000))
		_ = os.WriteFile(filepath.Join(root, "tests", fmt.Sprintf("test_%d_test.go", i)), []byte(content), 0644)
	}

	packet := domain.WorkPacket{Objective: "perform comprehensive audit"}
	budget := 3000

	p, err := PlanContext(root, packet, nil, budget)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ContextTokens > budget {
		t.Fatalf("expected context tokens %d <= budget %d", p.ContextTokens, budget)
	}

	// Mandatory files must be present
	hasAgents := false
	hasReadme := false
	for _, c := range p.Context {
		if c.Path == "AGENTS.md" {
			hasAgents = true
		}
		if c.Path == "README.md" {
			hasReadme = true
		}
	}
	if !hasAgents || !hasReadme {
		t.Fatalf("expected mandatory files retained, got: %+v", p.Context)
	}

	// Decisions audit trail must record compressed/omitted actions
	var compressedCount, omittedCount, retainedCount int
	for _, d := range p.ContextDecisions {
		switch d.Action {
		case "compressed":
			compressedCount++
		case "omitted":
			omittedCount++
		case "retained":
			retainedCount++
		}
	}
	if compressedCount == 0 && omittedCount == 0 {
		t.Fatalf("expected compressed or omitted items, got decisions: %+v", p.ContextDecisions)
	}
}

func TestPreservationOfMandatoryInstructions(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("Root README content"), 0644)
	_ = os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("Root AGENTS contract rules"), 0644)
	_ = os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("Claude instructions"), 0644)
	_ = os.WriteFile(filepath.Join(root, ".cursorrules"), []byte("Cursor rules"), 0644)

	_ = os.Mkdir(filepath.Join(root, "pkg"), 0755)
	_ = os.WriteFile(filepath.Join(root, "pkg", "code.go"), []byte("package pkg"), 0644)
	_ = os.WriteFile(filepath.Join(root, "pkg", "code_test.go"), []byte("package pkg\n"+strings.Repeat("// filler\n", 500)), 0644)

	changed := []string{"pkg/code.go"}
	packet := domain.WorkPacket{Objective: "fix bug"}
	budget := 1000

	p, err := PlanContext(root, packet, changed, budget)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ContextTokens > budget {
		t.Fatalf("expected context tokens %d <= budget %d", p.ContextTokens, budget)
	}

	paths := map[string]bool{}
	for _, c := range p.Context {
		paths[c.Path] = true
	}

	for _, mandatory := range []string{"README.md", "AGENTS.md", "CLAUDE.md", ".cursorrules", "pkg/code.go"} {
		if !paths[mandatory] {
			t.Fatalf("expected mandatory item %q to be preserved in context", mandatory)
		}
	}
}

func TestRelevanceBasedPruning(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("Project readme"), 0644)
	_ = os.Mkdir(filepath.Join(root, "tests"), 0755)
	_ = os.WriteFile(filepath.Join(root, "tests", "billing_payment_test.go"), []byte("package tests\nfunc TestBillingPayment(t *testing.T){}\n"+strings.Repeat("// fill\n", 300)), 0644)
	_ = os.WriteFile(filepath.Join(root, "tests", "irrelevant_misc_test.go"), []byte("package tests\nfunc TestMisc(t *testing.T){}\n"+strings.Repeat("// fill\n", 300)), 0644)

	packet := domain.WorkPacket{Objective: "fix billing payment processing"}
	budget := 900

	p, err := PlanContext(root, packet, nil, budget)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	billingRetained := false
	for _, c := range p.Context {
		if c.Path == "tests/billing_payment_test.go" {
			billingRetained = true
		}
	}
	if !billingRetained {
		t.Fatalf("expected objective-matching billing test to be prioritized and retained")
	}
}

func TestStructuralSummarization(t *testing.T) {
	root := t.TempDir()
	tsContent := `
import { describe, it } from 'vitest';
describe('Authentication Service', () => {
  it('authenticates valid token', () => {});
  it('rejects expired token', () => {});
});
`
	_ = os.WriteFile(filepath.Join(root, "auth.test.ts"), []byte(tsContent), 0644)
	summary, tokens := CompressFile(root, "auth.test.ts")
	if !strings.Contains(summary, "Authentication Service") || !strings.Contains(summary, "authenticates valid token") {
		t.Fatalf("unexpected test summary: %s", summary)
	}
	if tokens < 10 || tokens > 100 {
		t.Fatalf("unexpected token count: %d", tokens)
	}

	mdContent := "# Main Title\n## Overview\nSome text\n## Architecture\nDetails\n"
	_ = os.WriteFile(filepath.Join(root, "docs.md"), []byte(mdContent), 0644)
	mdSummary, mdTokens := CompressFile(root, "docs.md")
	if !strings.Contains(mdSummary, "Overview") || !strings.Contains(mdSummary, "Architecture") {
		t.Fatalf("unexpected markdown summary: %s", mdSummary)
	}
	if mdTokens < 10 || mdTokens > 80 {
		t.Fatalf("unexpected markdown token count: %d", mdTokens)
	}
}

func TestImpossibleToFitContext(t *testing.T) {
	root := t.TempDir()
	// Create a mandatory file that alone exceeds the budget
	_ = os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(strings.Repeat("rules and instructions\n", 1000)), 0644) // ~5700 tokens

	packet := domain.WorkPacket{Objective: "run task"}
	budget := 500 // budget is 500, but mandatory AGENTS.md is 5700 tokens

	_, err := PlanContext(root, packet, nil, budget)
	if err == nil {
		t.Fatalf("expected error when mandatory context cannot fit in budget")
	}
	if !strings.Contains(err.Error(), "mandatory context items") || !strings.Contains(err.Error(), "exceed configured budget") {
		t.Fatalf("expected clear explanation of mandatory budget overflow, got: %v", err)
	}
}

func TestContextAuditTrail(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("readme"), 0644)
	_ = os.WriteFile(filepath.Join(root, "test1_test.go"), []byte("package main\nfunc TestOne(t *testing.T){}"), 0644)
	_ = os.WriteFile(filepath.Join(root, "test2_test.go"), []byte("package main\nfunc TestTwo(t *testing.T){}"), 0644)

	packet := domain.WorkPacket{Objective: "audit"}
	p, err := PlanContext(root, packet, nil, 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(p.ContextDecisions) == 0 {
		t.Fatalf("expected context decisions recorded")
	}
	for _, d := range p.ContextDecisions {
		if d.Path == "" || d.Action == "" || d.Reason == "" {
			t.Fatalf("incomplete decision record: %+v", d)
		}
	}
}

func TestDeterministicContextPlanning(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("readme content"), 0644)
	_ = os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("contract rules"), 0644)
	for i := 1; i <= 5; i++ {
		_ = os.WriteFile(filepath.Join(root, fmt.Sprintf("sample_%d_test.go", i)), []byte(fmt.Sprintf("package s\nfunc Test%d(t *testing.T){}", i)), 0644)
	}

	packet := domain.WorkPacket{Objective: "deterministic test audit"}
	p1, err1 := PlanContext(root, packet, nil, 1000)
	p2, err2 := PlanContext(root, packet, nil, 1000)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}

	b1, _ := json.Marshal(p1)
	b2, _ := json.Marshal(p2)
	if string(b1) != string(b2) {
		t.Fatalf("expected identical deterministic context planning output across runs")
	}
}
