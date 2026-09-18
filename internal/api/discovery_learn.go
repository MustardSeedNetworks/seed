package api

import (
	"context"
	"net/netip"

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
func (s *Server) learnTargetNetworks(ctx context.Context) {
	if s.discoverySettings == nil || s.deviceDiscovery() == nil {
		return
	}

	devices := routingViews(s.deviceDiscovery().GetDevices())
	if len(devices) == 0 {
		return
	}

	subnet, _ := s.deviceDiscovery().GetSubnetInfo()
	candidates := learn.Candidates(devices, localPrefixes(subnet))
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
