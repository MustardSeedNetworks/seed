package api

// anomaly_platform.go builds the one server-owned anomaly engine (ADR-0029): a
// single Coordinator over a merged catalog (every producer's defs) and the
// unified store, shared by every producer instead of each owning its own engine.
// It is the realization of ADR-0021's "ONE engine, ONE store, used everywhere".

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	probeanomaly "github.com/MustardSeedNetworks/seed/internal/probe/anomaly"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
)

// initAnomalyPlatform constructs the single shared anomaly Coordinator and wires
// it into the producers that feed it. It runs before initProbeEngine so the
// probe producer can be built over the shared Coordinator, and it injects the
// Coordinator into the already-constructed Wi-Fi visibility component (built in
// the cmd layer before the server owns the engine).
//
// The catalog is the union of every producer's defs (Wi-Fi ∪ probe ∪ future);
// NewCatalog's duplicate-id rejection is the fail-fast guard that two domains
// never ship a colliding def id. A malformed merged catalog is a programming
// error: it is logged and leaves s.anomalyCoord nil, so the producers degrade to
// off rather than aborting server start.
//
// Load-on-start is performed here, once, before any producer observes (ADR-0029
// §5): the merged engine holds every def, so Restore no longer silently drops one
// producer's persisted rows as orphans. The producers therefore do not load.
func (s *Server) initAnomalyPlatform(db *database.DB) {
	defs := append(append([]anomaly.Def{}, wifianomaly.Defs()...), probeanomaly.Defs()...)
	cat, err := anomaly.NewCatalog(defs...)
	if err != nil {
		logging.GetLogger().Error("anomaly platform disabled: merged catalog invalid", "error", err)
		return
	}

	coord := anomaly.NewCoordinator(anomaly.NewEngine(cat), db.Anomalies())
	s.anomalyCoord = coord

	// Inject the shared Coordinator into the Wi-Fi visibility producer, which the
	// cmd layer built before the server owned the engine.
	if s.background != nil && s.background.WiFiVisibility != nil {
		s.background.WiFiVisibility.SetCoordinator(coord)
	}

	// Single server-owned load-on-start, before any producer's loop runs.
	if n, loadErr := coord.Load(context.Background()); loadErr != nil {
		logging.GetLogger().Warn("anomaly load-on-start failed", "error", loadErr)
	} else if n > 0 {
		logging.GetLogger().Info("restored persisted anomalies", "count", n)
	}
}
