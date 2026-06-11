package app

// engine.go wires the composition root to the engine-status application
// (use-case) service (ADR-0020): listing registered long-running subsystems
// and their health state. The adapter below implements the narrow
// status.Registry port declared in internal/engine/status over the concrete
// *engine.Registry, resolved through a lazy accessor so a later-set/nil
// registry (the api test harness) is honored, and a nil registry degrades
// the use-case to its empty-list behavior rather than panicking.

import (
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/engine/status"
)

// NewEngineStatus builds the engine-status use-case (ADR-0020) over a lazy
// accessor for the engine registry. A nil registry makes List return nil
// (the pre-strangle empty-response path).
func NewEngineStatus(reg func() *engine.Registry) *status.Service {
	return status.NewService(engineRegistryAdapter{reg: reg})
}

// ── registry adapter ──────────────────────────────────────────────────────────

// engineRegistryAdapter implements status.Registry over *engine.Registry,
// resolving it lazily. Available is checked per call so a later-set value
// (the api test harness) is honored; Engines is only invoked by the use-case
// once Available reports true, so it assumes a non-nil registry.
type engineRegistryAdapter struct {
	reg func() *engine.Registry
}

func (a engineRegistryAdapter) Available() bool          { return a.reg() != nil }
func (a engineRegistryAdapter) Engines() []engine.Engine { return a.reg().Engines() }
