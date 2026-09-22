package api

import (
	"context"
	"net/netip"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/learn"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// learnTargetNetworks turns the routing and address tables the last sweep read
// off SNMP-answering devices into target networks the operator can switch on
// (seed#2695). New networks are recorded disabled; nothing here starts scanning
// anything.
//
// It runs after a sweep rather than on a timer of its own because its whole
// input is that sweep's result, and it is best-effort: a learner that cannot
// write must not make a scan look failed.
//
// Every sweep reaches it: the discovery service's own observer covers the
// startup sweep, the rescan ticker and an interface change, and the manual
// scan button calls it directly because that path drives DeviceDiscovery
// below the service.
func (s *Server) learnTargetNetworks(ctx context.Context, discovered []*discovery.DiscoveredDevice) {
	if s.discoverySettings == nil || s.deviceDiscovery() == nil {
		return
	}

	devices := routingViews(discovered)
	host := hostRoutesVia(s.deviceDiscovery().GetInterfaceName(), gateway.GetAllRoutes)
	if len(devices) == 0 && len(host) == 0 {
		return
	}

	subnet, _ := s.deviceDiscovery().GetSubnetInfo()
	candidates := learn.Candidates(devices, host, localPrefixes(subnet))
	if len(candidates) == 0 {
		return
	}

	added, err := s.discoverySettings.Learn(candidates)
	logger := logging.FromContext(ctx)
	if err != nil {
		logger.WarnContext(ctx, "Could not record learned target networks", "error", err)
		return
	}
	if added > 0 {
		logger.InfoContext(ctx, "Learned target networks from SNMP routing data",
			"added", added, "candidates", len(candidates))
	}
}

// hostRoutesVia is Seed's own view of what lies behind the active interface:
// the rows of the host's forwarding table that leave through it. A site
// reached over a static route or a routing daemon is named here whether or
// not any router on it answers SNMP, which is the one learning source that
// survives a network whose devices are closed to us (seed#2695).
//
// Only the interface and the family are decided here. Which of those routes
// is worth offering as a sweep target is the learner's range rule, and
// duplicating it would give it two places to disagree with itself — the
// default route, for one, is dropped there as a /0 and not by a guard here.
func hostRoutesVia(iface string, read func() ([]gateway.RouteInfo, error)) []learn.HostRoute {
	if iface == "" {
		return nil
	}
	routes, err := read()
	if err != nil {
		return nil
	}
	var out []learn.HostRoute
	for _, route := range routes {
		if route.Interface != iface || route.Family != "inet" {
			continue
		}
		out = append(out, learn.HostRoute{
			Destination: route.Destination,
			Prefix:      route.Prefix,
			Gateway:     route.Gateway,
		})
	}
	return out
}

// routingViews reduces the discovered devices to the routing and address rows
// the learner reads, skipping the ones no SNMP walk reached.
func routingViews(devices []*discovery.DiscoveredDevice) []learn.Device {
	views := make([]learn.Device, 0, len(devices))
	for _, device := range devices {
		if device == nil || device.SNMPData == nil || device.IP == "" {
			continue
		}
		view := learn.Device{IP: device.IP}
		for _, route := range device.SNMPData.Routing {
			view.Routes = append(view.Routes, learn.Route{
				Destination: route.Destination,
				Prefix:      route.Prefix,
				Type:        route.Type,
				Protocol:    route.Protocol,
			})
		}
		for _, address := range device.SNMPData.IPAddresses {
			view.Addresses = append(view.Addresses, learn.Address{
				Address: address.Address,
				Prefix:  address.Prefix,
			})
		}
		if len(view.Routes) == 0 && len(view.Addresses) == 0 {
			continue
		}
		views = append(views, view)
	}
	return views
}

// localPrefixes is the network the sweep already covers without being told to:
// the active interface's own subnet, which needs no learned toggle.
func localPrefixes(subnet string) []netip.Prefix {
	prefix, err := netip.ParsePrefix(subnet)
	if err != nil {
		return nil
	}
	return []netip.Prefix{prefix.Masked()}
}
