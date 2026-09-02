package harness

import "context"

// NewConfiguredProcess is the generic external-process harness constructor
// used by the control loop. Provider-specific protocols remain outside this
// package; the process speaks Keystone's packet/stdout boundary.
func NewConfiguredProcess(ctx context.Context, cfg Config) *ProcessAdapter {
	return NewProcessAdapter(ctx, cfg)
}
