package api

import (
	"context"

	"github.com/krisarmstrong/seed/internal/canopy"
	"github.com/krisarmstrong/seed/internal/modules/harvest"
	"github.com/krisarmstrong/seed/internal/services"
	"github.com/krisarmstrong/seed/internal/services/shell"
)

// Modules contains all application modules for dependency injection.
type Modules struct {
	Sap     *services.Module
	Shell   *shell.Module
	Canopy  *canopy.Module
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
	if m.Shell != nil {
		if err := m.Shell.Start(ctx); err != nil {
			return err
		}
	}
	if m.Canopy != nil {
		if err := m.Canopy.Start(ctx); err != nil {
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
	if m.Canopy != nil {
		_ = m.Canopy.Stop()
	}
	if m.Shell != nil {
		_ = m.Shell.Stop()
	}
	if m.Sap != nil {
		_ = m.Sap.Stop()
	}
	return nil
}
