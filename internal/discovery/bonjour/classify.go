package bonjour

import (
	"net/netip"
	"sort"
)

// assemble joins the collected records into services and returns the
// cross-subnet verdict, both classified against local.
func (c *collector) assemble(local []netip.Prefix) ([]string, []ServiceInstance, ReflectorStatus) {
	types := make([]string, 0, len(c.types))
	for t := range c.types {
		types = append(types, t)
	}
	sort.Strings(types)

	ordered := make([]*instanceRecords, 0, len(c.instances))
	for _, inst := range c.instances {
		ordered = append(ordered, inst)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].seq < ordered[j].seq })

	services := make([]ServiceInstance, 0, len(ordered))
	verdict := newVerdict()
	for _, inst := range ordered {
		svc := c.service(inst, local)
		verdict.add(svc, inst, local)
		services = append(services, svc)
	}
	return types, services, verdict.result(c.responses)
}

func (c *collector) service(inst *instanceRecords, local []netip.Prefix) ServiceInstance {
	instance, serviceType, _ := splitInstance(inst.name)
	addrs := c.addrs[inst.srvHost]

	svc := ServiceInstance{
		Instance:        instance,
		Type:            serviceType,
		Host:            inst.srvHost,
		Port:            inst.srvPort,
		Addresses:       addrText(addrs),
		Origin:          classifyOrigin(addrs, local),
		SourceAddresses: addrText(sortedAddrs(inst.sources)),
	}
	if len(inst.txt) > 0 {
		svc.TXT = inst.txt
	}
	return svc
}

// classifyOrigin decides whether an advertisement describes a host on this
// segment. One on-segment address is enough: a multi-homed host that also
// advertises a docker bridge, a VPN or a second NIC is local, and treating it
// as reflected would report a reflector on every developer laptop.
//
// Addresses seed cannot place — link-local, loopback — are skipped rather
// than counted as off-segment, and an address family the interface has no
// prefix for is skipped too: an interface with no global IPv6 would otherwise
// make every AAAA look like it came from another subnet.
func classifyOrigin(addrs []netip.Addr, local []netip.Prefix) Origin {
	classifiable := 0
	for _, a := range addrs {
		if !placeable(a) || !haveFamily(local, a) {
			continue
		}
		classifiable++
		if withinAny(a, local) {
			return OriginLocal
		}
	}
	if classifiable == 0 {
		return OriginUnknown
	}
	return OriginOffSegment
}

func placeable(a netip.Addr) bool {
	return a.IsValid() && !a.IsLoopback() && !a.IsLinkLocalUnicast() && !a.IsUnspecified()
}

func haveFamily(local []netip.Prefix, a netip.Addr) bool {
	for _, p := range local {
		if p.Addr().Is4() == a.Is4() {
			return true
		}
	}
	return false
}

func withinAny(a netip.Addr, local []netip.Prefix) bool {
	for _, p := range local {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// verdict accumulates the evidence for the cross-subnet state.
type verdict struct {
	sawOffSegment bool
	remote        map[string]struct{}
	forwardedBy   map[string]struct{}
	routedFrom    map[string]struct{}
}

func newVerdict() *verdict {
	return &verdict{
		remote:      make(map[string]struct{}),
		forwardedBy: make(map[string]struct{}),
		routedFrom:  make(map[string]struct{}),
	}
}

func (v *verdict) add(svc ServiceInstance, inst *instanceRecords, local []netip.Prefix) {
	for src := range inst.sources {
		if placeable(src) && haveFamily(local, src) && !withinAny(src, local) {
			v.routedFrom[src.String()] = struct{}{}
		}
	}
	if svc.Origin != OriginOffSegment {
		return
	}
	v.sawOffSegment = true
	for _, a := range svc.Addresses {
		if addr, err := netip.ParseAddr(a); err == nil && placeable(addr) && haveFamily(local, addr) {
			v.remote[groupingPrefix(addr).String()] = struct{}{}
		}
	}
	// The sender that re-originated it is the reflector — but only when the
	// sender is itself on this segment. A sender that is off-segment is the
	// routed case, already recorded above, and naming it as the forwarder
	// would claim a reflector where there is a route.
	for src := range inst.sources {
		if withinAny(src, local) {
			v.forwardedBy[src.String()] = struct{}{}
		}
	}
}

// groupingPrefix buckets a remote address so the UI can name the subnet it
// came from. seed cannot know the remote mask, so it groups on the
// conventional boundary — /24 and /64 — and the UI says "subnet", not
// "network".
func groupingPrefix(a netip.Addr) netip.Prefix {
	bits := 64
	if a.Is4() {
		bits = 24
	}
	p, err := a.Prefix(bits)
	if err != nil {
		return netip.PrefixFrom(a, a.BitLen())
	}
	return p
}

func (v *verdict) result(responses int) ReflectorStatus {
	r := ReflectorStatus{
		RemoteSubnets: sortedKeys(v.remote),
		ForwardedBy:   sortedKeys(v.forwardedBy),
		RoutedFrom:    sortedKeys(v.routedFrom),
	}
	switch {
	case len(v.routedFrom) > 0:
		r.State = StateRouted
	case v.sawOffSegment:
		r.State = StateReflected
	case responses == 0:
		r.State = StateNoTraffic
	default:
		r.State = StateLocalOnly
	}
	return r
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedAddrs(m map[netip.Addr]struct{}) []netip.Addr {
	out := make([]netip.Addr, 0, len(m))
	for a := range m {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	return out
}

func addrText(addrs []netip.Addr) []string {
	if len(addrs) == 0 {
		return nil
	}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.String())
	}
	return out
}
