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
