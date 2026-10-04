// Package retention drives tiered-retention rollups and purges for
// any time-series source registered with the engine. One loop, one
// tier-aware purge policy, source-pluggable via RollupSource.
//
// Sources registered at startup: probe_results (replacing dead
// internal/health/rollup.go), metrics and flow_records.
//
// Tier horizons:
//
//	Free:    raw 7d only
//	Starter: raw 7d + hourly 30d
//	Pro:     raw 7d + hourly 90d + daily 2y
//
// Tier resolution is dynamic — re-read on each pass — so an in-place
// upgrade takes effect on the next tick without restart.
//
// V1.0 NMS expansion — Stage A0 scaffold (2026-05-30). Replaces the
// V1.0-WIP internal/services/retention/ during Stage A2.
package retention
