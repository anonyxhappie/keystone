package checkpoint

import (
	"fmt"
	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/state"
	"time"
)

func Save(s state.Store, c domain.Checkpoint) error {
	if c.ID == "" {
		c.ID = fmt.Sprintf("CP-%d", time.Now().UnixNano())
	}
	return s.Write("checkpoints/"+c.ID+".json", c)
}
