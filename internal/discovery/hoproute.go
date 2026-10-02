package discovery

import "net/netip"

// HopRoute is the route a traced hop's own forwarding table holds toward the
// trace target: what the router at that hop says it does with the packet,
// beside where traceroute saw the packet go next (seed#2587).
type HopRoute struct {
	Device      string // the discovered device's own address
	Destination string // matched route's network, e.g. "10.20.0.0"
	Prefix      int
	NextHop     string
	IfIndex     int
	Interface   string // egress interface name; empty when the walk named none
	Type        string
	Protocol    string

	// TableTruncated means the device held more routes than one walk reads,
	// so a more specific route than this one may exist unread.
	TableTruncated bool
}

// RouteForHop finds the discovered device that answers on hopIP, by its own
// address or any address its SNMP walk listed, and returns the longest-prefix
// route its table holds for target. It returns nil when no device owns the
// address, its table was not read, or no route covers target.
func RouteForHop(devices []*DiscoveredDevice, hopIP, target string) *HopRoute {
	hop, err := netip.ParseAddr(hopIP)
	if err != nil {
		return nil
	}
	dst, err := netip.ParseAddr(target)
	if err != nil {
		return nil
	}
	device := deviceAnswering(devices, hop)
	if device == nil {
		return nil
	}
	data := device.SNMPData
	route, ok := longestMatch(data.Routing, dst)
	if !ok {
		return nil
	}
	return &HopRoute{
		Device:         device.IP,
		Destination:    route.Destination,
		Prefix:         route.Prefix,
		NextHop:        route.NextHop,
		IfIndex:        route.IfIndex,
		Interface:      interfaceName(data.Interfaces, route.IfIndex),
		Type:           route.Type,
		Protocol:       route.Protocol,
		TableTruncated: data.RoutingTruncated,
	}
}

// deviceAnswering returns the device with a read route table that owns addr.
// A router answers traceroute from the interface the probe arrived on, which
// is rarely the address discovery found it at, so the walked IP-MIB addresses
// count as well as the device's own.
func deviceAnswering(devices []*DiscoveredDevice, addr netip.Addr) *DiscoveredDevice {
	for _, device := range devices {
		if device == nil || device.SNMPData == nil || len(device.SNMPData.Routing) == 0 {
			continue
		}
		if sameAddr(device.IP, addr) {
			return device
		}
		for _, owned := range device.SNMPData.IPAddresses {
			if sameAddr(owned.Address, addr) {
				return device
			}
		}
	}
	return nil
}

func sameAddr(s string, addr netip.Addr) bool {
	parsed, err := netip.ParseAddr(s)
	return err == nil && parsed.Unmap() == addr.Unmap()
}

// longestMatch is the route a router would forward dst by: the most specific
// one covering it. Routes of the other address family never match.
func longestMatch(routes []SNMPRoute, dst netip.Addr) (SNMPRoute, bool) {
	dst = dst.Unmap()
	var best SNMPRoute
	found := false
	for _, route := range routes {
		network, err := netip.ParseAddr(route.Destination)
		if err != nil {
			continue
		}
		prefix, err := network.Unmap().Prefix(route.Prefix)
		if err != nil || !prefix.Contains(dst) {
			continue
		}
		if !found || route.Prefix > best.Prefix {
			best, found = route, true
		}
	}
	return best, found
}

func interfaceName(interfaces []SNMPInterface, ifIndex int) string {
	for _, iface := range interfaces {
		if ifIndex != 0 && iface.Index == ifIndex {
			if iface.Name != "" {
				return iface.Name
			}
			return iface.Description
		}
	}
	return ""
}
