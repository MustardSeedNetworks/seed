package api

import (
	"context"

	"github.com/krisarmstrong/seed/internal/modules/harvest"
	"github.com/krisarmstrong/seed/internal/services"
)

// Modules contains the long-lived components with real Start/Stop lifecycle.
// Stateless request/response logic lives in the handlers + the api services
// groupings; only components that own background work belong here.
type Modules struct {
	Sap     *services.Module
	Harvest *harvest.Module
}

// Start initializes and starts all modules.
func (m *Modules) Start(ctx context.Context) error {
	// Start modules in dependency order
	if m.Sap != nil {
		if err := m.Sap.Start(ctx); err != nil {
			return err
		}
	}
	if m.Harvest != nil {
		if err := m.Harvest.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop gracefully shuts down all modules.
func (m *Modules) Stop() error {
	// Stop modules in reverse order
	if m.Harvest != nil {
		_ = m.Harvest.Stop()
	}
	if m.Sap != nil {
		_ = m.Sap.Stop()
	}
	return nil
}
