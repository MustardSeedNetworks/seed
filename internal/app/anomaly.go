package app

// anomaly.go holds the composition-root helper that re-hydrates store-backed
// anomalies (ADR-0029 follow-up). The unified store does not persist the
// catalog-static Impact / FollowUps — they are re-derived on read from the
// shared, server-owned engine's catalog, so both anomaly endpoints (Wi-Fi and
// probe) present the same complete view the in-memory Snapshot would.

import (
	"github.com/MustardSeedNetworks/seed/internal/anomaly"
)

// enrichAnomalies fills each anomaly's catalog-static Impact / FollowUps from the
// shared engine's catalog (capability-degraded follow-ups, identical to the live
// projection). It mutates in place and returns the slice for chaining. A nil
// engine (anomaly platform not wired / test harness) leaves the list exactly as
// the store returned it.
func enrichAnomalies(engine *anomaly.Engine, list []anomaly.Anomaly) []anomaly.Anomaly {
	if engine == nil {
		return list
	}
	for i := range list {
		list[i] = engine.EnrichStatic(list[i])
	}
	return list
}
