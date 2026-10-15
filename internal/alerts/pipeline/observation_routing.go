package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"slices"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// maxRouteEvidence bounds the routes one alert lists. A BGP session dropping
// on an edge router withdraws thousands of prefixes at once; the alert carries
// the count of all of them and the first maxRouteEvidence by prefix.
const maxRouteEvidence = 50

// routingObsPayload mirrors the subset of the routing Observation the rules
// read.
type routingObsPayload struct {
	Routes []struct {
		Destination string `json:"Destination"`
		Mask        string `json:"Mask"`
		NextHop     string `json:"NextHop"`
		Proto       int    `json:"Proto"`
	} `json:"Routes"`
	Truncated bool `json:"Truncated"`
}

// routeEntry is one prefix of a target's routing table: every next hop the
// table lists for it (more than one under ECMP) and the protocol it was
// learned by.
type routeEntry struct {
	nextHops []string
	proto    int
}

// routeSnapshot is one target's previous routing table, keyed by prefix.
type routeSnapshot struct {
	routes    map[string]routeEntry
	truncated bool
}

// routePrefix renders a destination and mask as CIDR, or dest/mask when the
// mask is not contiguous.
func routePrefix(dest, mask string) string {
	if m := net.ParseIP(mask).To4(); m != nil {
		if ones, bits := net.IPMask(m).Size(); bits != 0 {
			return fmt.Sprintf("%s/%d", dest, ones)
		}
	}
	return dest + "/" + mask
}

func decodeRouting(obs *observation.SNMPObservation) (routeSnapshot, bool) {
	var pay routingObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return routeSnapshot{}, false
	}
	snap := routeSnapshot{routes: make(map[string]routeEntry, len(pay.Routes)), truncated: pay.Truncated}
	for _, r := range pay.Routes {
		prefix := routePrefix(r.Destination, r.Mask)
		entry := snap.routes[prefix]
		entry.nextHops = append(entry.nextHops, r.NextHop)
		entry.proto = r.Proto
		snap.routes[prefix] = entry
	}
	for prefix, entry := range snap.routes {
		slices.Sort(entry.nextHops)
		snap.routes[prefix] = entry
	}
	return snap, true
}

func (p *ObservationPipeline) lookupRouting(key string) (routeSnapshot, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	snap, ok := p.routeState[key]
	return snap, ok
}

func (p *ObservationPipeline) recordRouting(key string, snap routeSnapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.routeState[key] = snap
}

func (p *ObservationPipeline) primeRouting(obs *observation.SNMPObservation) {
	if snap, ok := decodeRouting(obs); ok {
		p.recordRouting("route/"+obs.TargetID, snap)
	}
}

// evaluateRouting compares a router's table with its previous poll. A prefix
// that is gone raises RuleRouteWithdrawn; a prefix whose next hops differ
// raises RuleRouteChange. Each is one alert per poll listing every prefix
// affected, because one cause (a peer or a link) moves many prefixes at once.
// A new prefix raises nothing: networks grow by design.
//
// A table read short at routing.MaxRows cannot show that a prefix past the cut
// is gone, so a truncated poll raises no withdrawal; next-hop changes on the
// prefixes it did read are still reported.
func (p *ObservationPipeline) evaluateRouting(
	ctx context.Context, obs *observation.SNMPObservation,
) int {
	cur, ok := decodeRouting(obs)
	if !ok {
		return 0
	}
	key := "route/" + obs.TargetID
	prev, seen := p.lookupRouting(key)
	p.recordRouting(key, cur)
	if !seen {
		return 0
	}

	var withdrawn []alerts.RouteWithdrawal
	var changed []alerts.RouteNextHopChange
	var withdrawnPrefixes, changedPrefixes []string
	for _, prefix := range slices.Sorted(maps.Keys(prev.routes)) {
		was := prev.routes[prefix]
		now, present := cur.routes[prefix]
		switch {
		case !present && !cur.truncated:
			withdrawn = append(withdrawn, alerts.RouteWithdrawal{
				Prefix: prefix, NextHops: was.nextHops, Proto: was.proto,
			})
			withdrawnPrefixes = append(withdrawnPrefixes, prefix)
		case present && !slices.Equal(was.nextHops, now.nextHops):
			changed = append(changed, alerts.RouteNextHopChange{
				Prefix: prefix, From: was.nextHops, To: now.nextHops,
			})
			changedPrefixes = append(changedPrefixes, prefix)
		}
	}

	count := 0
	if len(withdrawn) > 0 {
		alert := &alerts.Alert{
			Type:     alerts.TypeConnectivity,
			Severity: alerts.SeverityWarning,
			Title:    fmt.Sprintf("%d route(s) withdrawn on %s", len(withdrawn), obs.TargetID),
			Message:  fmt.Sprintf("No longer in the routing table: %s", prefixSummary(withdrawnPrefixes)),
			Source:   obs.TargetID,
			Metadata: evidenceJSON(alerts.RouteWithdrawnEvidence{
				Withdrawn: len(withdrawn), Routes: withdrawn[:min(len(withdrawn), maxRouteEvidence)],
			}),
		}
		if p.fire(ctx, alerts.RuleRouteWithdrawn, key, alert) {
			count++
		}
	}
	if len(changed) > 0 {
		alert := &alerts.Alert{
			Type:     alerts.TypeConnectivity,
			Severity: alerts.SeverityInfo,
			Title:    fmt.Sprintf("%d route(s) changed next hop on %s", len(changed), obs.TargetID),
			Message:  fmt.Sprintf("Next hop changed for: %s", prefixSummary(changedPrefixes)),
			Source:   obs.TargetID,
			Metadata: evidenceJSON(alerts.RouteChangeEvidence{
				Changed: len(changed), Routes: changed[:min(len(changed), maxRouteEvidence)],
			}),
		}
		if p.fire(ctx, alerts.RuleRouteChange, key, alert) {
			count++
		}
	}
	return count
}

// prefixSummary names the first few prefixes for an alert message.
func prefixSummary(prefixes []string) string {
	const shown = 5
	if len(prefixes) <= shown {
		return strings.Join(prefixes, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(prefixes[:shown], ", "), len(prefixes)-shown)
}
