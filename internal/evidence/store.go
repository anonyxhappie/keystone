package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/state"
	"time"
)

func ID(kind, summary string) string {
	h := sha256.Sum256([]byte(kind + "\x00" + summary))
	return "EV-" + hex.EncodeToString(h[:6])
}
func Record(s state.Store, workOrder, kind, summary, commit string, passed bool) (domain.Evidence, error) {
	status := domain.StatusFailed
	if passed {
		status = domain.StatusCompleted
	}
	verification := domain.Rejected
	if passed {
		verification = domain.Verified
	}
	e := domain.Evidence{ID: ID(kind, summary), Type: kind, Status: status, Verification: verification, WorkOrderID: workOrder, Commit: commit, Summary: summary, Valid: passed, CreatedAt: time.Now().UTC()}
	return e, s.Write("evidence/"+e.ID+".json", e)
}
func EnsureValid(e domain.Evidence) error {
	if !e.Valid {
		return fmt.Errorf("evidence %s is invalid", e.ID)
	}
	return nil
}

func Invalidate(s state.Store, e domain.Evidence, reason string) error {
	e.Valid = false
	e.Status = domain.StatusStale
	e.Verification = domain.Stale
	e.Summary = e.Summary + " (stale: " + reason + ")"
	return s.Write("evidence/"+e.ID+".json", e)
}

func ValidFor(e domain.Evidence, commit, inputsHash string) bool {
	if !e.Valid || e.Verification != domain.Verified {
		return false
	}
	return (e.Commit == "" || commit == "" || e.Commit == commit) && (e.InputsHash == "" || inputsHash == "" || e.InputsHash == inputsHash)
}
