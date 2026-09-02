package checkpoint

import (
	"fmt"
	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/state"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func Save(s state.Store, c domain.Checkpoint) error {
	if c.ID == "" {
		c.ID = fmt.Sprintf("CP-%d", time.Now().UnixNano())
	}
	return s.Write("checkpoints/"+c.ID+".json", c)
}

func Latest(s state.Store, workOrderID string) (domain.Checkpoint, error) {
	entries, err := os.ReadDir(filepath.Join(s.Root, state.Dir, "checkpoints"))
	if err != nil {
		return domain.Checkpoint{}, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for i := len(names) - 1; i >= 0; i-- {
		var c domain.Checkpoint
		if err := s.Read("checkpoints/"+names[i], &c); err != nil {
			continue
		}
		if workOrderID == "" || c.WorkOrderID == workOrderID {
			return c, nil
		}
	}
	return domain.Checkpoint{}, fmt.Errorf("no checkpoint found")
}
