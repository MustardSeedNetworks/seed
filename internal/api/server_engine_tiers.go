package api

// Per-tier engine gating (Stage A5.9, item 4 of the V1.0 wrap-up).
//
// Engine registration runs through registerEngineIfLicensed, which
// looks up the engine's minimum license tier and skips registration
// when the current license sits below it. The Free tier ships with
// the basic Seed functionality (probe + retention); Starter adds
// SNMP visibility (snmp-poller + the four topology reconcilers);
// Pro adds proactive alerting (the two alert pipelines + the opt-
// in syslog/trap listeners).
//
// This matches the locked tier matrix from
// msn-docs-internal/10-Portal-License/LICENSE_STRATEGY.md.

import (
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// minTierForEngine returns the minimum license tier required to
// register an engine. Unknown engine names default to Free so a
// new engine doesn't accidentally get blocked by the gate until
// the operator deliberately puts it on a paid tier.
func minTierForEngine(name string) license.Tier {
	switch name {
	case "probe", "retention":
		return license.TierFree
	case "snmp-poller",
		"topology-sysinfo-reconciler",
		"topology-iftable-reconciler",
		"topology-edge-reconciler",
		"topology-arp-reconciler":
		return license.TierStarter
	case "alert-listener-pipeline",
		"alert-observation-pipeline",
		"syslog-udp",
		"snmp-trap-v2c":
		return license.TierPro
	}
	return license.TierFree
}

// registerEngineIfLicensed registers eng with the lifecycle
// registry when the current license meets the engine's minimum
// tier; otherwise it logs a skip and returns nil. The caller-side
// API is identical to a plain Register call so swapping in the
// gate is a one-line change at each init site.
//
// Semantics when no license.Manager is configured:
//
//   - Pre-license state (services.Auth.License == nil) is treated
//     as Pro tier. This matches the dev / fresh-install / test
//     experience where everything should be available; production
//     deployments always configure a Manager (the install flow
//     creates a Free license file even when no paid key is
//     entered), so the gate kicks in there.
//   - A configured Manager that reports TierFree gates Starter +
//     Pro engines out, as intended for the Free SKU.
func registerEngineIfLicensed(services *ServiceContainer, eng engine.Engine) error {
	if services == nil || services.Engines == nil {
		return nil
	}
	tier := effectiveTier(services)
	minTier := minTierForEngine(eng.Name())
	if tier < minTier {
		logging.GetLogger().Info("engine skipped: license tier below minimum",
			"engine", eng.Name(),
			"current_tier", int(tier),
			"min_tier", int(minTier),
		)
		return nil
	}
	return services.Engines.Register(eng)
}

// effectiveTier picks the tier to gate against. nil Manager (dev,
// tests, pre-install) -> Pro; otherwise delegate to the existing
// licenseTierAdapter which already nil-guards GetState() etc.
func effectiveTier(services *ServiceContainer) license.Tier {
	if services.Auth == nil || services.Auth.License == nil {
		return license.TierPro
	}
	return licenseTierAdapter{lm: services.Auth.License}.GetTier()
}
