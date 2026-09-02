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
