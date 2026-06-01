package api

import (
	"context"

	"github.com/krisarmstrong/seed/internal/modules/harvest"
)

// Modules holds the long-lived components that own real background lifecycle.
// Stateless request/response logic lives in the handlers and the api service
// groupings (built directly from the feature packages); only components that own
// background work belong here. Currently that is the report scheduler (harvest);
// the sap/canopy/shell/roots "module" facades were dead parallel wiring and have
// been removed (see docs/architecture/PHASE3_RECONCILE_PROPOSAL.md).
type Modules struct {
	Harvest *harvest.Module
}

// Start initializes and starts all modules.
func (m *Modules) Start(ctx context.Context) error {
	if m.Harvest != nil {
		if err := m.Harvest.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop gracefully shuts down all modules.
func (m *Modules) Stop() error {
	if m.Harvest != nil {
		_ = m.Harvest.Stop()
	}
	return nil
}
