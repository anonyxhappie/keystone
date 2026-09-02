package delivery

import (
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/evidence"
	"github.com/anonyxhappie/keystone/internal/state"
)

func TestProductionDeploymentRequiresApproval(t *testing.T) {
	s := state.New(t.TempDir())
	if _, err := s.Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	ev, err := evidence.RecordScoped(s, "WO-1", "test", "tests passed", "abc", "inputs", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	r, err := CreateRelease(s, "2.0.1", []string{ev.ID}, "main")
	if err != nil {
		t.Fatal(err)
	}
	d, decision, err := CreateDeployment(s, r, domain.Environment{Name: "production", Kind: "production", Protected: true})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || !decision.RequiresApproval || d.Status != domain.StatusBlocked {
		t.Fatalf("unexpected deployment: %+v %+v", d, decision)
	}
}

func TestIncidentBecomesNormalWorkOrder(t *testing.T) {
	o := IncidentOrder(domain.Incident{Summary: "latency regression", Severity: "high"})
	if o.Objective == "" || o.Risk.Level != "high" || len(o.Constraints) == 0 {
		t.Fatalf("unexpected incident order: %+v", o)
	}
}

func TestDeliveryPersistsEnvironmentAndIncidentTraceability(t *testing.T) {
	s := state.New(t.TempDir())
	if _, err := s.Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	ev, err := evidence.RecordScoped(s, "WO-1", "operational-ci", "CI passed", "abc", "inputs", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	release, err := CreateRelease(s, "2.0.0", []string{ev.ID}, "abc")
	if err != nil {
		t.Fatal(err)
	}
	if release.GitReference != "abc" || len(release.OperationalEvidenceIDs) != 1 {
		t.Fatalf("release traceability missing: %+v", release)
	}
	if _, _, err := CreateDeployment(s, release, domain.Environment{Name: "staging", Kind: "staging"}); err != nil {
		t.Fatal(err)
	}
	incident, order, err := RecordIncident(s, domain.Incident{Summary: "latency regression", Severity: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if incident.WorkOrderID != order.ID {
		t.Fatalf("incident work order not linked: %+v %+v", incident, order)
	}
}

func TestReleaseRequiresVerifiedRequirementsWhenLinked(t *testing.T) {
	s := state.New(t.TempDir())
	if _, err := s.Init("fixture", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Write("requirements/REQ-1.json", domain.Requirement{ID: "REQ-1", Status: domain.StatusPlanned}); err != nil {
		t.Fatal(err)
	}
	ev, err := evidence.RecordScoped(s, "WO-1", "verification", "verified", "abc", "", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateReleaseWithRequirements(s, "2.0.0", []string{"REQ-1"}, []string{ev.ID}, "abc"); err == nil {
		t.Fatal("expected unverified requirement to block release")
	}
	if err := s.Write("requirements/REQ-1.json", domain.Requirement{ID: "REQ-1", Status: domain.StatusVerified}); err != nil {
		t.Fatal(err)
	}
	release, err := CreateReleaseWithRequirements(s, "2.0.0", []string{"REQ-1"}, []string{ev.ID}, "abc")
	if err != nil || len(release.RequirementIDs) != 1 {
		t.Fatalf("verified requirement was not linked: %+v %v", release, err)
	}
}
