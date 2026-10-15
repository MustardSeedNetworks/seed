package api

import (
	"net/http"
	"net/netip"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/ipam"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// arpInUseWindow is how recently a router must have held an ARP entry for its
// address to count as in use. A day spans the common DHCP lease and keeps a
// host that sleeps overnight counted, while an address released last week
// stops holding a slot.
const arpInUseWindow = 24 * time.Hour

// IPAMSubnet is one network's address utilization.
type IPAMSubnet struct {
	CIDR   string      `json:"cidr"`
	Name   string      `json:"name,omitempty"`
	Source ipam.Source `json:"source"         jsonschema:"enum=local,enum=configured,enum=learned"`
	// Size is the number of assignable host addresses.
	Size           uint64  `json:"size"`
	Used           int     `json:"used"`
	Free           uint64  `json:"free"`
	UtilizationPct float64 `json:"utilizationPct"`
	// InUse lists the addresses seen in use, in address order.
	InUse []string `json:"inUse"`
}

// IPAMSubnetsResponse is GET /api/v1/ipam/subnets.
type IPAMSubnetsResponse struct {
	Subnets []IPAMSubnet `json:"subnets"`
}

func (s *Server) ipamRoutes() []route.Route {
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/ipam/subnets",
			Handler: s.handleIPAMSubnets,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
	}
}

// handleIPAMSubnets reports the utilization of every network Seed knows
// about: the active interface's subnet and the target networks, less the
// learned ones an operator dismissed (P-D5).
func (s *Server) handleIPAMSubnets(w http.ResponseWriter, r *http.Request) {
	bindings, err := s.topologyQueries.ARP(r.Context(), topology.ARPListOptions{
		Since: time.Now().Add(-arpInUseWindow),
	})
	if err != nil {
		logging.FromContext(r.Context()).ErrorContext(r.Context(),
			"list topology_arp_bindings for IPAM failed", "error", err)
		writeTopologyError(w, r, err, "Failed to read ARP bindings")
		return
	}

	var localSubnet string
	var devices []*discovery.DiscoveredDevice
	if s.deviceDiscovery() != nil {
		localSubnet, _ = s.deviceDiscovery().GetSubnetInfo()
		devices = s.deviceDiscovery().GetDevices()
	}
	networks := ipamNetworks(localSubnet, s.discoverySettings.Subnets())

	usages := ipam.Utilization(networks, addressesInUse(devices, bindings))
	subnets := make([]IPAMSubnet, 0, len(usages))
	for _, usage := range usages {
		addrs := make([]string, 0, len(usage.InUse))
		for _, addr := range usage.InUse {
			addrs = append(addrs, addr.String())
		}
		subnets = append(subnets, IPAMSubnet{
			CIDR:           usage.Prefix.String(),
			Name:           usage.Name,
			Source:         usage.Source,
			Size:           usage.Size,
			Used:           len(usage.InUse),
			Free:           usage.Free(),
			UtilizationPct: usage.Percent(),
			InUse:          addrs,
		})
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
		IPAMSubnetsResponse{Subnets: subnets})
}

// ipamNetworks lists the active interface's subnet first, so a target network
// that repeats it is reported as local. A learned network the operator
// dismissed is left out.
func ipamNetworks(localSubnet string, targets []config.SubnetConfig) []ipam.Network {
	var networks []ipam.Network
	if prefix, err := netip.ParsePrefix(localSubnet); err == nil {
		networks = append(networks, ipam.Network{Prefix: prefix, Source: ipam.SourceLocal})
	}
	for _, subnet := range targets {
		prefix, err := netip.ParsePrefix(subnet.CIDR)
		if err != nil || subnet.Decision == config.LearnedDismissed {
			continue
		}
		source := ipam.SourceConfigured
		if subnet.Learned {
			source = ipam.SourceLearned
		}
		networks = append(networks, ipam.Network{Prefix: prefix, Name: subnet.Name, Source: source})
	}
	return networks
}

// addressesInUse is every address seen in use: each swept device's own, the
// interface addresses of the devices an SNMP walk reached, and the routers'
// recent ARP entries.
func addressesInUse(devices []*discovery.DiscoveredDevice, bindings []*topology.ARPBinding) []netip.Addr {
	var out []netip.Addr
	add := func(address string) {
		if addr, err := netip.ParseAddr(address); err == nil {
			out = append(out, addr)
		}
	}
	for _, device := range devices {
		add(device.IP)
		if device.SNMPData != nil {
			for _, address := range device.SNMPData.IPAddresses {
				add(address.Address)
			}
		}
	}
	for _, binding := range bindings {
		add(binding.IPAddress)
	}
	return out
}
