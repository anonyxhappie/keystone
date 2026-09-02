package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/state"
)

func ID(kind, summary string) string { h:=sha256.Sum256([]byte(kind+"\x00"+summary)); return "EV-"+hex.EncodeToString(h[:6]) }
func Record(s state.Store, workOrder, kind, summary, commit string, passed bool) (domain.Evidence,error) {
	status:=domain.StatusFailed; if passed { status=domain.StatusCompleted }
	e:=domain.Evidence{ID:ID(kind,summary),Type:kind,Status:status,WorkOrderID:workOrder,Commit:commit,Summary:summary,Valid:passed,CreatedAt:time.Now().UTC()}
	return e,s.Write("evidence/"+e.ID+".json",e)
}
func EnsureValid(e domain.Evidence) error { if !e.Valid { return fmt.Errorf("evidence %s is invalid",e.ID) }; return nil }
