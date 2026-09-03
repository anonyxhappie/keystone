package work

import "testing"

func TestNewOrderPreservesRequest(t *testing.T) {
	o := NewOrder("  add a feature  ")
	if o.SourceRequest != "  add a feature  " || o.Objective != "add a feature" {
		t.Fatalf("unexpected order: %+v", o)
	}
	p := Packet(o)
	if p.WorkOrderID != o.ID || len(p.CompletionCriteria) != 2 {
		t.Fatalf("unexpected packet: %+v", p)
	}
}

func TestAssessRiskDetectsPolicySensitiveRequests(t *testing.T) {
	risk := AssessRisk("deploy this to production")
	if risk.Level != "high" || len(risk.Factors) == 0 {
		t.Fatalf("expected high-risk request: %+v", risk)
	}
	if release := AssessRisk("prepare release notes"); release.Level != "release" {
		t.Fatalf("expected release risk: %+v", release)
	}
}

func TestReadOnlyRequestDetectionAndPacket(t *testing.T) {
	cases := []string{
		"Audit the repository. Do not modify any files.",
		"Inspect codebase. Do not make changes.",
		"Run read-only audit of configuration",
		"Inspect only the architecture",
	}
	for _, c := range cases {
		if !IsReadOnlyRequest(c) {
			t.Fatalf("expected %q to be recognized as read-only", c)
		}
		o := NewOrder(c)
		if !o.ReadOnly {
			t.Fatalf("expected order for %q to be marked ReadOnly", c)
		}
		p := Packet(o)
		if !p.ReadOnly {
			t.Fatalf("expected packet for %q to be marked ReadOnly", c)
		}
		hasCriteria := false
		for _, crit := range p.CompletionCriteria {
			if crit == "no repository mutations occurred" {
				hasCriteria = true
			}
		}
		if !hasCriteria {
			t.Fatalf("expected packet for %q to include no mutations criterion", c)
		}
	}
}

func TestDirectiveExtractionAndEscalation(t *testing.T) {
	// 1. Explicit slash command extraction
	dir, clean := ExtractDirective("/goal implement the entire user authentication system")
	if dir != "goal" || clean != "implement the entire user authentication system" {
		t.Fatalf("unexpected extract: dir=%q, clean=%q", dir, clean)
	}

	dirBtw, _ := ExtractDirective("/btw what is the database schema?")
	if dirBtw != "btw" {
		t.Fatalf("expected btw directive, got %q", dirBtw)
	}
	orderBtw := NewOrder("/btw what is the database schema?")
	if !orderBtw.ReadOnly {
		t.Fatalf("expected /btw to be automatically marked ReadOnly")
	}
	if orderBtw.Directive != "btw" {
		t.Fatalf("expected order directive to be btw, got %q", orderBtw.Directive)
	}

	// 2. Heuristic auto-escalation
	escGoal := DetectAutonomousEscalation("refactor all repository adapters to use contexts")
	if escGoal != "goal" {
		t.Fatalf("expected auto-escalation to goal, got %q", escGoal)
	}

	escBoost := DetectAutonomousEscalation("perform deep analysis on the system architecture")
	if escBoost != "boost" {
		t.Fatalf("expected auto-escalation to boost, got %q", escBoost)
	}

	// 3. NewOrder with auto-escalation
	orderGoal := NewOrder("refactor all repository adapters")
	if orderGoal.Directive != "goal" {
		t.Fatalf("expected auto-escalated directive 'goal', got %q", orderGoal.Directive)
	}
}
