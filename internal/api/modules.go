package api

import (
	"context"

	"github.com/krisarmstrong/seed/internal/reporting"
)

// Modules holds the long-lived components that own real background lifecycle.
// Stateless request/response logic lives in the handlers and the api service
// groupings (built directly from the feature packages); only components that own
// background work belong here. Currently that is the report scheduler
// (internal/reporting); the dead "module" facades were parallel wiring and have
// been removed (see docs/architecture/PHASE3_RECONCILE_PROPOSAL.md).
type Modules struct {
	Reporting *reporting.Module
}

// Start initializes and starts all modules.
func (m *Modules) Start(ctx context.Context) error {
	if m.Reporting != nil {
		if err := m.Reporting.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop gracefully shuts down all modules.
func (m *Modules) Stop() error {
	if m.Reporting != nil {
		_ = m.Reporting.Stop()
	}
	return nil
}
