// Package bonjour enumerates DNS-SD (Bonjour) services on the attached
// segment and reports whether mDNS from other subnets is reaching it.
//
// mDNS is link-local multicast: it does not cross a router. AirPlay and
// AirPrint therefore fail between VLANs unless someone configured a
// reflector (Avahi's reflector, a Cisco or Aruba Bonjour gateway) to
// re-originate the traffic on each segment. Seed answers the two questions a
// technician has at the wall port: what is advertised here, and is anything
// from elsewhere arriving (#364).
//
// The decode and classify halves are pure and are what the tests exercise;
// browse.go owns the socket.
package bonjour

import (
	"net/netip"
	"time"
)

// Origin says where a service's advertised addresses live relative to the
// segment seed is listening on.
type Origin string

const (
	// OriginLocal means at least one advertised address is inside a prefix of
	// the listening interface. A multi-homed host that also advertises a
	// docker bridge or VPN address is local, not reflected, which is why one
	// on-segment address is enough.
	OriginLocal Origin = "local"
	// OriginOffSegment means every advertised address is outside every local
	// prefix: the record describes a host seed cannot be sharing a link with,
	// so something forwarded it here.
	OriginOffSegment Origin = "off-segment"
	// OriginUnknown means the advertisement carried no address seed could
	// classify — no A/AAAA at all, or only link-local and loopback ones.
	OriginUnknown Origin = "unknown"
)

// ReflectorState is the verdict over the whole capture window.
type ReflectorState string

const (
	// StateNoTraffic means no mDNS response arrived at all. This is not
	// evidence about a reflector in either direction, and the UI says so:
	// an idle segment and a blocked one look identical from here.
	StateNoTraffic ReflectorState = "no-traffic"
	// StateLocalOnly means mDNS arrived and every advertisement described a
	// host on this segment. No cross-subnet mDNS was seen.
	StateLocalOnly ReflectorState = "local-only"
	// StateReflected means an on-segment sender advertised hosts that are not
	// on this segment: the signature of a reflector re-originating another
	// VLAN's announcements here.
	StateReflected ReflectorState = "reflected"
	// StateRouted means a response arrived from a source address that is not
	// on this segment at all — a router forwarding 224.0.0.251 rather than a
	// reflector re-originating it. Rarer, and a different misconfiguration,
	// so it is reported separately rather than folded into reflected.
	StateRouted ReflectorState = "routed"
)

// ServiceInstance is one DNS-SD instance as advertised on the wire.
type ServiceInstance struct {
	// Instance is the human-readable instance name, with DNS escaping undone.
	Instance string `json:"instance"`
	// Type is the service type, e.g. "_airplay._tcp".
	Type string `json:"type"`
	// Host is the SRV target, e.g. "living-room.local".
	Host string `json:"host,omitempty"`
	// Port is the SRV port. Zero when no SRV record was seen.
	Port uint16 `json:"port"`
	// Addresses are the A/AAAA records for Host, as text.
	Addresses []string `json:"addresses,omitempty"`
	// TXT is the parsed key=value metadata. Keys with no '=' map to "".
	TXT map[string]string `json:"txt,omitempty"`
	// Origin classifies Addresses against the listening interface's prefixes.
	Origin Origin `json:"origin"`
	// SourceAddresses are the packet source addresses that advertised this
	// instance. A reflector puts its own address here while Addresses stay
	// remote, which is exactly what makes the two fields worth separating.
	SourceAddresses []string `json:"sourceAddresses,omitempty"`
}

// ReflectorStatus is the cross-subnet verdict and the evidence behind it. The
// sentence a technician reads is composed by the UI from State and these
// fields, so it exists in every locale rather than only in the server's.
type ReflectorStatus struct {
	State ReflectorState `json:"state"`
	// RemoteSubnets are the /24-or-/64 groupings of off-segment addresses
	// seen, so the UI can name which VLANs are arriving.
	RemoteSubnets []string `json:"remoteSubnets,omitempty"`
	// ForwardedBy lists the on-segment senders that advertised off-segment
	// hosts — the reflector itself, when there is one.
	ForwardedBy []string `json:"forwardedBy,omitempty"`
	// RoutedFrom lists source addresses that were themselves off-segment.
	RoutedFrom []string `json:"routedFrom,omitempty"`
}

// BrowseResult is one browse.
type BrowseResult struct {
	Interface string `json:"interface,omitempty"`
	// LocalPrefixes are the prefixes every classification was made against,
	// reported so a surprising verdict can be checked rather than trusted.
	LocalPrefixes   []string          `json:"localPrefixes,omitempty"`
	ServiceTypes    []string          `json:"serviceTypes"`
	Services        []ServiceInstance `json:"services"`
	ReflectorStatus ReflectorStatus   `json:"reflector"`
	// ResponsesObserved counts mDNS responses read, including ones that
	// carried nothing seed could use.
	ResponsesObserved int `json:"responsesObserved"`
	// Truncated is true when a bound in limits was reached and the browse
	// stopped collecting rather than growing without limit.
	Truncated  bool          `json:"truncated"`
	Duration   time.Duration `json:"-"`
	DurationMs int64         `json:"durationMs"`
}

// limits bound what an untrusted segment can make seed allocate. A browse is
// a diagnostic over a hostile broadcast domain, not a database.
type limits struct {
	maxServiceTypes    int
	maxInstances       int
	maxAddressesPerSvc int
	maxTXTPairs        int
	maxTXTValueBytes   int
}

// The default bounds. A browse runs over a broadcast domain seed does not
// control, so every collection it grows has a ceiling; these are set well
// above what a real segment advertises (a busy office floor runs to tens of
// service types and low hundreds of instances) and exist to stop a hostile
// or broken responder, not to filter a real one.
const (
	defaultMaxServiceTypes    = 64
	defaultMaxInstances       = 256
	defaultMaxAddressesPerSvc = 8
	defaultMaxTXTPairs        = 32
	defaultMaxTXTValueBytes   = 256
)

func defaultLimits() limits {
	return limits{
		maxServiceTypes:    defaultMaxServiceTypes,
		maxInstances:       defaultMaxInstances,
		maxAddressesPerSvc: defaultMaxAddressesPerSvc,
		maxTXTPairs:        defaultMaxTXTPairs,
		maxTXTValueBytes:   defaultMaxTXTValueBytes,
	}
}

// localPrefixesOf reduces an interface's addresses to the prefixes a peer
// would share with it. Link-local and loopback are dropped: every host has
// them, so they classify nothing.
func localPrefixesOf(addrs []netip.Prefix) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(addrs))
	for _, p := range addrs {
		a := p.Addr()
		if a.IsLoopback() || a.IsLinkLocalUnicast() || a.IsLinkLocalMulticast() || a.IsUnspecified() {
			continue
		}
		out = append(out, p.Masked())
	}
	return out
}
