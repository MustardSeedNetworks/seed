package bonjour

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

// segment is the prefix set of a listening interface with IPv4 only — the
// common handheld case, and the one that makes the IPv6 family rule matter.
func segment() []netip.Prefix {
	return []netip.Prefix{netip.MustParsePrefix("192.168.20.0/24")}
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

// browseOf folds the named fixtures in, all from one source address, and
// assembles the result the handler would serve.
func browseOf(
	t *testing.T, src string, local []netip.Prefix, names ...string,
) (*collector, []string, []ServiceInstance, ReflectorStatus) {
	t.Helper()
	c := newCollector(defaultLimits())
	from := netip.MustParseAddr(src)
	for _, n := range names {
		c.observe(fixture(t, n), from)
	}
	types, services, reflector := c.assemble(local)
	return c, types, services, reflector
}

func TestMetaResponseYieldsServiceTypesAndNoInstances(t *testing.T) {
	_, types, services, reflector := browseOf(t, "192.168.20.77", segment(), "meta-response.bin")

	want := []string{"_airplay._tcp.local", "_companion-link._tcp.local", "_ipp._tcp.local", "_raop._tcp.local"}
	if len(types) != len(want) {
		t.Fatalf("service types = %v, want %v", types, want)
	}
	for i, w := range want {
		if types[i] != w {
			t.Errorf("service type %d = %q, want %q", i, types[i], w)
		}
	}
	if len(services) != 0 {
		t.Errorf("meta response produced %d services, want 0: its PTR targets are types, not instances", len(services))
	}
	// Traffic arrived and nothing was off-segment, so the honest verdict is
	// local-only rather than no-traffic.
	if reflector.State != StateLocalOnly {
		t.Errorf("state = %q, want %q", reflector.State, StateLocalOnly)
	}
}

func TestLocalInstanceCarriesHostPortAddressAndTXT(t *testing.T) {
	_, _, services, reflector := browseOf(t, "192.168.20.40", segment(), "local-instance.bin")

	if len(services) != 1 {
		t.Fatalf("services = %d, want 1", len(services))
	}
	svc := services[0]
	if svc.Instance != "Living Room" {
		t.Errorf("instance = %q, want %q", svc.Instance, "Living Room")
	}
	if svc.Type != "_airplay._tcp" {
		t.Errorf("type = %q, want %q", svc.Type, "_airplay._tcp")
	}
	if svc.Host != "apple-tv.local" {
		t.Errorf("host = %q, want %q", svc.Host, "apple-tv.local")
	}
	if svc.Port != 7000 {
		t.Errorf("port = %d, want 7000", svc.Port)
	}
	if len(svc.Addresses) != 1 || svc.Addresses[0] != "192.168.20.40" {
		t.Errorf("addresses = %v, want [192.168.20.40]", svc.Addresses)
	}
	if svc.TXT["model"] != "AppleTV6,2" {
		t.Errorf("txt[model] = %q, want %q", svc.TXT["model"], "AppleTV6,2")
	}
	// RFC 6763 §6.4: a key with no '=' is present with no value.
	if v, ok := svc.TXT["noval"]; !ok || v != "" {
		t.Errorf(`txt["noval"] = %q present=%v, want "" present=true`, v, ok)
	}
	if svc.Origin != OriginLocal {
		t.Errorf("origin = %q, want %q", svc.Origin, OriginLocal)
	}
	if reflector.State != StateLocalOnly {
		t.Errorf("state = %q, want %q", reflector.State, StateLocalOnly)
	}
}

func TestOffSegmentHostFromOnSegmentSenderReadsAsReflected(t *testing.T) {
	// The reflector's own address is on this segment; the host it advertises
	// is not. That pairing is the whole diagnosis.
	_, _, services, reflector := browseOf(t, "192.168.20.1", segment(), "reflected-instance.bin")

	if len(services) != 1 {
		t.Fatalf("services = %d, want 1", len(services))
	}
	if services[0].Origin != OriginOffSegment {
		t.Errorf("origin = %q, want %q", services[0].Origin, OriginOffSegment)
	}
	if reflector.State != StateReflected {
		t.Fatalf("state = %q, want %q", reflector.State, StateReflected)
	}
	if len(reflector.RemoteSubnets) != 1 || reflector.RemoteSubnets[0] != "10.44.40.0/24" {
		t.Errorf("remote subnets = %v, want [10.44.40.0/24]", reflector.RemoteSubnets)
	}
	if len(reflector.ForwardedBy) != 1 || reflector.ForwardedBy[0] != "192.168.20.1" {
		t.Errorf("forwarded by = %v, want [192.168.20.1]", reflector.ForwardedBy)
	}
	if len(reflector.RoutedFrom) != 0 {
		t.Errorf("routed from = %v, want empty: the sender is on this segment", reflector.RoutedFrom)
	}
}

func TestOffSegmentSenderReadsAsRoutedNotReflected(t *testing.T) {
	// Same packet, different sender: nothing re-originated it here, a router
	// forwarded the multicast. Different fault, so a different state.
	_, _, _, reflector := browseOf(t, "10.44.40.1", segment(), "reflected-instance.bin")

	if reflector.State != StateRouted {
		t.Fatalf("state = %q, want %q", reflector.State, StateRouted)
	}
	if len(reflector.RoutedFrom) != 1 || reflector.RoutedFrom[0] != "10.44.40.1" {
		t.Errorf("routed from = %v, want [10.44.40.1]", reflector.RoutedFrom)
	}
	if len(reflector.ForwardedBy) != 0 {
		t.Errorf("forwarded by = %v, want empty: no on-segment sender re-originated it", reflector.ForwardedBy)
	}
}

func TestMultiHomedHostIsLocalNotReflected(t *testing.T) {
	// One on-segment address is enough. Demanding all of them would report a
	// reflector on every host with a docker bridge or a VPN.
	_, _, services, reflector := browseOf(t, "192.168.20.51", segment(), "multihomed-instance.bin")

	if len(services) != 1 {
		t.Fatalf("services = %d, want 1", len(services))
	}
	if services[0].Origin != OriginLocal {
		t.Errorf("origin = %q, want %q (addresses %v)", services[0].Origin, OriginLocal, services[0].Addresses)
	}
	if reflector.State != StateLocalOnly {
		t.Errorf("state = %q, want %q", reflector.State, StateLocalOnly)
	}
}

func TestIPv6HostIsUnknownWhenTheInterfaceHasNoIPv6Prefix(t *testing.T) {
	// An IPv4-only interface can say nothing about an AAAA-only host. Calling
	// it off-segment would report a reflector on any segment with IPv6 on it.
	_, _, services, reflector := browseOf(t, "192.168.20.90", segment(), "v6-instance.bin")

	if len(services) != 1 {
		t.Fatalf("services = %d, want 1", len(services))
	}
	if services[0].Origin != OriginUnknown {
		t.Errorf("origin = %q, want %q", services[0].Origin, OriginUnknown)
	}
	if reflector.State != StateLocalOnly {
		t.Errorf("state = %q, want %q", reflector.State, StateLocalOnly)
	}
}

func TestIPv6HostIsClassifiedWhenTheInterfaceHasAnIPv6Prefix(t *testing.T) {
	dual := []netip.Prefix{
		netip.MustParsePrefix("192.168.20.0/24"),
		netip.MustParsePrefix("2001:db8:20::/64"),
	}
	_, _, services, reflector := browseOf(t, "192.168.20.90", dual, "v6-instance.bin")

	if services[0].Origin != OriginOffSegment {
		t.Errorf("origin = %q, want %q", services[0].Origin, OriginOffSegment)
	}
	if len(reflector.RemoteSubnets) != 1 || reflector.RemoteSubnets[0] != "2001:db8:44::/64" {
		t.Errorf("remote subnets = %v, want [2001:db8:44::/64]", reflector.RemoteSubnets)
	}
}

func TestRecordsSplitAcrossPacketsAreJoined(t *testing.T) {
	// macOS's mDNSResponder answers a service-type PTR query with the
	// instance PTR and nothing else, then serves SRV/TXT/A when asked by
	// name. A decoder that expected one self-contained packet would list this
	// instance with no host and no port.
	_, _, services, _ := browseOf(t, "192.168.20.88", segment(), "split-ptr.bin", "split-details.bin")

	if len(services) != 1 {
		t.Fatalf("services = %d, want 1", len(services))
	}
	svc := services[0]
	if svc.Host != "studio-display.local" || svc.Port != 62078 {
		t.Errorf("host/port = %q/%d, want studio-display.local/62078", svc.Host, svc.Port)
	}
	if len(svc.Addresses) != 1 || svc.Addresses[0] != "192.168.20.88" {
		t.Errorf("addresses = %v, want [192.168.20.88]", svc.Addresses)
	}
	if svc.TXT["rpBA"] != "A1:B2:C3" {
		t.Errorf("txt[rpBA] = %q, want A1:B2:C3", svc.TXT["rpBA"])
	}
}

func TestNonASCIIInstanceNameSurvivesDecoding(t *testing.T) {
	_, _, services, _ := browseOf(t, "192.168.20.72", segment(), "utf8-instance.bin")

	if len(services) != 1 {
		t.Fatalf("services = %d, want 1", len(services))
	}
	if got, want := services[0].Instance, "Café"; got != want {
		t.Errorf("instance = %q, want %q", got, want)
	}
	if services[0].Type != "_raop._tcp" {
		t.Errorf("type = %q, want _raop._tcp", services[0].Type)
	}
}

func TestInstanceNameContainingADotIsInvisible(t *testing.T) {
	// A known limit, pinned rather than papered over: dnsmessage rejects the
	// whole packet when a label contains a literal dot, so "Office No. 5"
	// never reaches the collector. The browse must survive it — one
	// undecodable neighbour cannot end the scan — and the next packet must
	// still be read.
	c := newCollector(defaultLimits())
	src := netip.MustParseAddr("192.168.20.61")
	c.observe(fixture(t, "dotted-instance.bin"), src)

	if c.responses != 0 {
		t.Errorf("responses = %d, want 0: the packet does not unpack", c.responses)
	}
	if len(c.instances) != 0 {
		t.Errorf("instances = %d, want 0", len(c.instances))
	}

	c.observe(fixture(t, "local-instance.bin"), netip.MustParseAddr("192.168.20.40"))
	if len(c.instances) != 1 {
		t.Errorf("instances after the next packet = %d, want 1: the browse must continue", len(c.instances))
	}
}

func TestQueriesAndMalformedPacketsAreIgnored(t *testing.T) {
	c, types, services, reflector := browseOf(t, "192.168.20.5", segment(), "query.bin", "malformed.bin")

	if c.responses != 0 {
		t.Errorf("responses = %d, want 0: neither packet is an mDNS response", c.responses)
	}
	if len(types) != 0 || len(services) != 0 {
		t.Errorf("types = %v services = %v, want both empty", types, services)
	}
	if reflector.State != StateNoTraffic {
		t.Errorf("state = %q, want %q: nothing was observed, which is not evidence either way",
			reflector.State, StateNoTraffic)
	}
}

func TestPTRsThatAreNotDNSSDAreNotInstances(t *testing.T) {
	// PTR is shared with reverse lookup and with underscore-prefixed names
	// that are not DNS-SD at all. A service type is an underscore label over
	// _tcp or _udp; nothing else advertises an instance.
	for _, owner := range []string{
		"40.20.168.192.in-addr.arpa", // reverse lookup
		"_dns-update._domain.local",  // underscore label, wrong transport
		"apple-tv.local",             // a plain host
	} {
		c := newCollector(defaultLimits())
		c.observePTR(owner, "something._airplay._tcp.local", netip.MustParseAddr("192.168.20.40"))

		if len(c.instances) != 0 {
			t.Errorf("%q: instances = %d, want 0", owner, len(c.instances))
		}
		if len(c.types) != 0 {
			t.Errorf("%q: types = %d, want 0", owner, len(c.types))
		}
	}
}

func TestBoundsStopAnUntrustedSegmentFromGrowingTheBrowse(t *testing.T) {
	c := newCollector(limits{
		maxServiceTypes:    1,
		maxInstances:       1,
		maxAddressesPerSvc: 1,
		maxTXTPairs:        1,
		maxTXTValueBytes:   4,
	})
	src := netip.MustParseAddr("192.168.20.9")

	c.observePTR(trimRoot(dnssdMetaQuery), "_airplay._tcp.local", src)
	c.observePTR(trimRoot(dnssdMetaQuery), "_raop._tcp.local", src)
	if len(c.types) != 1 {
		t.Errorf("types = %d, want 1 (bounded)", len(c.types))
	}

	c.observePTR("_airplay._tcp.local", "A._airplay._tcp.local", src)
	c.observePTR("_airplay._tcp.local", "B._airplay._tcp.local", src)
	if len(c.instances) != 1 {
		t.Errorf("instances = %d, want 1 (bounded)", len(c.instances))
	}

	inst := c.instances["A._airplay._tcp.local"]
	c.mergeTXT(inst, []string{"k=0123456789", "second=x"})
	if got := inst.txt["k"]; got != "0123" {
		t.Errorf("txt value = %q, want %q (truncated to the byte bound)", got, "0123")
	}
	if len(inst.txt) != 1 {
		t.Errorf("txt pairs = %d, want 1 (bounded)", len(inst.txt))
	}

	c.observeAddr("h.local", netip.MustParseAddr("192.168.20.1"))
	c.observeAddr("h.local", netip.MustParseAddr("192.168.20.2"))
	if len(c.addrs["h.local"]) != 1 {
		t.Errorf("addresses = %d, want 1 (bounded)", len(c.addrs["h.local"]))
	}

	if !c.truncated {
		t.Error("truncated = false, want true: every bound above was reached")
	}
}

func TestDuplicateAddressIsRecordedOnce(t *testing.T) {
	c := newCollector(defaultLimits())
	addr := netip.MustParseAddr("192.168.20.40")
	c.observeAddr("apple-tv.local", addr)
	c.observeAddr("apple-tv.local", addr)

	if got := len(c.addrs["apple-tv.local"]); got != 1 {
		t.Errorf("addresses = %d, want 1", got)
	}
	if c.truncated {
		t.Error("truncated = true, want false: a repeat is not a bound")
	}
}

func TestLinkLocalAndLoopbackClassifyNothing(t *testing.T) {
	addrs := []netip.Addr{
		netip.MustParseAddr("169.254.3.4"),
		netip.MustParseAddr("127.0.0.1"),
	}
	if got := classifyOrigin(addrs, segment()); got != OriginUnknown {
		t.Errorf("origin = %q, want %q", got, OriginUnknown)
	}
}

func TestLocalPrefixesDropTheOnesEveryHostHas(t *testing.T) {
	in := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.1/8"),
		netip.MustParsePrefix("169.254.9.9/16"),
		netip.MustParsePrefix("fe80::1/64"),
		netip.MustParsePrefix("192.168.20.77/24"),
	}
	got := localPrefixesOf(in)
	if len(got) != 1 || got[0].String() != "192.168.20.0/24" {
		t.Fatalf("prefixes = %v, want [192.168.20.0/24] (masked)", got)
	}
}

func TestSplitInstanceRejectsWhatIsNotAnInstance(t *testing.T) {
	for _, name := range []string{
		"_airplay._tcp.local",        // the service type itself
		"apple-tv.local",             // a plain host
		"40.20.168.192.in-addr.arpa", // a reverse lookup
		"",                           // nothing
	} {
		if _, _, ok := splitInstance(name); ok {
			t.Errorf("splitInstance(%q) reported an instance", name)
		}
	}
	instance, serviceType, ok := splitInstance("Living Room._airplay._tcp.local")
	if !ok || instance != "Living Room" || serviceType != "_airplay._tcp" {
		t.Errorf("splitInstance = %q/%q/%v, want %q/%q/true", instance, serviceType, ok, "Living Room", "_airplay._tcp")
	}
}

func TestAddressStageAsksForEveryUnresolvedSRVTarget(t *testing.T) {
	// Measured on the wire: a responder answers SRV without the target's
	// address. Without this stage every service classifies as
	// OriginUnknown and the cross-subnet verdict can never be reached.
	c := newCollector(defaultLimits())
	src := netip.MustParseAddr("192.168.20.77")
	c.observe(fixture(t, "split-ptr.bin"), src)
	c.observe(fixture(t, "split-details.bin"), src)

	// split-details carries the address, so nothing is left to ask.
	if got := len(addressQuestions(c)); got != 0 {
		t.Errorf("questions = %d, want 0: the address is already known", got)
	}

	// Drop the address and the host must be asked for again, as A and AAAA.
	delete(c.addrs, "studio-display.local")
	questions := addressQuestions(c)
	if len(questions) != 2 {
		t.Fatalf("questions = %d, want 2 (A and AAAA)", len(questions))
	}
	types := map[dnsmessage.Type]bool{}
	for _, q := range questions {
		types[q.Type] = true
		if q.Name.String() != "studio-display.local." {
			t.Errorf("question name = %q, want studio-display.local.", q.Name.String())
		}
	}
	if !types[dnsmessage.TypeA] || !types[dnsmessage.TypeAAAA] {
		t.Errorf("question types = %v, want both A and AAAA", types)
	}
}

func TestAddressStageSkipsInstancesWithNoSRV(t *testing.T) {
	c := newCollector(defaultLimits())
	c.observe(fixture(t, "split-ptr.bin"), netip.MustParseAddr("192.168.20.88"))

	if got := len(addressQuestions(c)); got != 0 {
		t.Errorf("questions = %d, want 0: there is no host name to ask about yet", got)
	}
}
