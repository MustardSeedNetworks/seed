package api

// The SNMP polling boundaries in the licence catalogue (seed#2327, owner
// decision 2026-09-03). estate_polling is a count, not a switch: gating the
// polling-targets routes would make all SNMP polling Pro, and polling plus
// topology is what Starter sells. server_monitoring and bgp_monitoring are the
// host_resources and bgp4_mib collectors, which only run at Pro.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/bgp4"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/hostresources"
)

// Polling targets each tier may hold. Pro (estate_polling) is unlimited. Free
// cannot poll at all — the snmp-poller engine is Starter — so its one target
// is somewhere to stage a device before upgrading.
const (
	freePollingTargets    = 1
	starterPollingTargets = 25
)

// collectorFeature names the feature a Pro-only collector needs. Any other
// collector runs wherever the poller runs.
func collectorFeature(name string) (string, bool) {
	switch name {
	case hostresources.Name:
		return "server_monitoring", true
	case bgp4.Name:
		return "bgp_monitoring", true
	}
	return "", false
}

// pollingTargetLimit is how many polling targets the licence allows; 0 is
// unlimited.
func (s *Server) pollingTargetLimit() int {
	switch {
	case s.hasFeature("estate_polling"):
		return 0
	case s.effectiveTier() >= license.TierStarter:
		return starterPollingTargets
	default:
		return freePollingTargets
	}
}

// collectorLicensed reports whether the licence covers the named collector.
func (s *Server) collectorLicensed(name string) bool {
	feature, gated := collectorFeature(name)
	return !gated || s.hasFeature(feature)
}

// unlicensedCollectorFeature returns the feature the first collector in chain
// that the licence does not cover would need.
func (s *Server) unlicensedCollectorFeature(chain []string) (string, bool) {
	for _, name := range chain {
		if feature, gated := collectorFeature(name); gated && !s.hasFeature(feature) {
			return feature, true
		}
	}
	return "", false
}

// licensedPollerTargets caps what the poller reads at the licence limit, so a
// licence that shrinks (a Pro trial that becomes Starter) stops polling the
// targets past it instead of polling them for good. The order is the
// repository's, by name, so which targets keep polling is stable.
type licensedPollerTargets struct {
	snmp.PollerStorage

	limit func() int
}

func (l licensedPollerTargets) ListEnabled(ctx context.Context) ([]*polling.Target, error) {
	targets, err := l.PollerStorage.ListEnabled(ctx)
	if limit := l.limit(); err == nil && limit > 0 && len(targets) > limit {
		targets = targets[:limit]
	}
	return targets, err
}
