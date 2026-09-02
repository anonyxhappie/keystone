package delivery

import (
	"fmt"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/evidence"
	"github.com/anonyxhappie/keystone/v2/internal/policy"
	"github.com/anonyxhappie/keystone/v2/internal/state"
	"github.com/anonyxhappie/keystone/v2/internal/work"
)

func CreateRelease(s state.Store, version string, evidenceIDs []string, gitReference string) (domain.Release, error) {
	return CreateReleaseWithRequirements(s, version, nil, evidenceIDs, gitReference)
}

func CreateReleaseWithRequirements(s state.Store, version string, requirementIDs, evidenceIDs []string, gitReference string) (domain.Release, error) {
	if version == "" {
		return domain.Release{}, fmt.Errorf("release version is required")
	}
	if len(evidenceIDs) == 0 {
		return domain.Release{}, fmt.Errorf("release requires evidence")
	}
	for _, id := range requirementIDs {
		var requirement domain.Requirement
		if err := s.Read("requirements/"+id+".json", &requirement); err != nil {
			return domain.Release{}, fmt.Errorf("release requirement %s: %w", id, err)
		}
		if requirement.Status != domain.StatusVerified {
			return domain.Release{}, fmt.Errorf("release requirement %s is not verified", id)
		}
	}
	for _, id := range evidenceIDs {
		var ev domain.Evidence
		if err := s.Read("evidence/"+id+".json", &ev); err != nil {
			return domain.Release{}, fmt.Errorf("release evidence %s: %w", id, err)
		}
		if !evidence.ValidFor(ev, ev.Commit, ev.InputsHash) {
			return domain.Release{}, fmt.Errorf("release evidence %s is not verified", id)
		}
	}
	r := domain.Release{ID: "REL-" + version, Version: version, Status: domain.StatusPlanned, GitReference: gitReference, RequirementIDs: requirementIDs, EvidenceIDs: evidenceIDs, CreatedAt: time.Now().UTC()}
	for _, id := range evidenceIDs {
		var ev domain.Evidence
		if err := s.Read("evidence/"+id+".json", &ev); err == nil {
			if strings.Contains(strings.ToLower(ev.Type), "security") {
				r.SecurityEvidenceIDs = append(r.SecurityEvidenceIDs, id)
			}
			if strings.Contains(strings.ToLower(ev.Type), "operational") || strings.Contains(strings.ToLower(ev.Type), "ci") {
				r.OperationalEvidenceIDs = append(r.OperationalEvidenceIDs, id)
			}
		}
	}
	if err := s.Write("releases/"+r.ID+".json", r); err != nil {
		return r, err
	}
	return r, nil
}

func CreateDeployment(s state.Store, release domain.Release, environment domain.Environment) (domain.Deployment, domain.PolicyDecision, error) {
	action := "staging_deploy"
	if environment.Protected || environment.Kind == "production" {
		action = "production_deploy"
	}
	decision := policy.Evaluate(action)
	d := domain.Deployment{ID: fmt.Sprintf("DEP-%s-%s", release.Version, environment.Name), Environment: environment.Name, ReleaseID: release.ID, Status: domain.StatusBlocked}
	if environment.ID == "" {
		environment.ID = "ENV-" + environment.Name
	}
	if err := s.Write("environments/"+environment.ID+".json", environment); err != nil {
		return d, decision, err
	}
	if decision.Allowed {
		d.Status = domain.StatusPlanned
	}
	if err := s.Write("deployments/"+d.ID+".json", d); err != nil {
		return d, decision, err
	}
	return d, decision, nil
}

func IncidentOrder(i domain.Incident) domain.WorkOrder {
	o := work.NewOrder("Investigate incident: " + i.Summary)
	o.Risk = domain.Risk{Level: "high", Factors: []string{"operational incident"}, Rationale: i.Severity}
	o.Constraints = []string{"preserve evidence and avoid destructive remediation without approval"}
	return o
}

func RecordIncident(s state.Store, i domain.Incident) (domain.Incident, domain.WorkOrder, error) {
	if i.ID == "" {
		i.ID = fmt.Sprintf("INC-%d", time.Now().UnixNano())
	}
	o := IncidentOrder(i)
	i.WorkOrderID = o.ID
	if err := s.Write("incidents/"+i.ID+".json", i); err != nil {
		return i, o, err
	}
	if err := s.Write("work/"+o.ID+".json", o); err != nil {
		return i, o, err
	}
	return i, o, nil
}
