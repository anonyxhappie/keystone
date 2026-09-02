package learning

import (
	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/state"
)

func SaveCandidate(s state.Store, l domain.Learning) error {
	if l.Status == "" {
		l.Status = "CANDIDATE"
	}
	if l.Version == 0 {
		l.Version = 1
	}
	return s.Write("learning/"+l.ID+".json", l)
}
