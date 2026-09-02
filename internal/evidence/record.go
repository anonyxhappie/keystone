package evidence

import (
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/state"
)

func RecordScoped(s state.Store, workOrder, kind, summary, commit, inputsHash string, observationIDs []string, passed bool) (domain.Evidence, error) {
	return RecordScopedArtifacts(s, workOrder, kind, summary, commit, inputsHash, observationIDs, nil, passed)
}

func RecordScopedArtifacts(s state.Store, workOrder, kind, summary, commit, inputsHash string, observationIDs, artifactRefs []string, passed bool) (domain.Evidence, error) {
	status := domain.StatusRejected
	verification := domain.Rejected
	if passed {
		status = domain.StatusCompleted
		verification = domain.Verified
	}
	e := domain.Evidence{ID: ID(workOrder+"\x00"+kind+"\x00"+inputsHash+"\x00"+strings.Join(observationIDs, "\x00"), summary), Type: kind, Status: status, Verification: verification, WorkOrderID: workOrder, Commit: commit, InputsHash: inputsHash, Summary: summary, Valid: passed, ObservationIDs: observationIDs, ArtifactRefs: artifactRefs, CreatedAt: time.Now().UTC()}
	return e, s.Write("evidence/"+e.ID+".json", e)
}
