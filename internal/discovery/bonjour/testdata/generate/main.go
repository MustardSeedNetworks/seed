// Command generate writes the mDNS fixtures the bonjour decoder is tested
// against. Run it from the package directory:
//
//	go run ./testdata/generate
//
// The fixtures are synthesised rather than committed captures on purpose: a
// real capture off any segment carries its owner's device names, and this
// repository is public. What is NOT invented is the shapes — each scenario
// below mirrors a packet layout measured on the wire on 2026-09-15 against
// macOS's mDNSResponder, and the split-records pair exists because that
// responder answered a service-type PTR query with the instance PTR alone and
// sent SRV, TXT and A only when asked for them by name.
package main

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"

	"golang.org/x/net/dns/dnsmessage"
)

func main() {
	fixtures := map[string][]byte{
		// The service-type enumeration of RFC 6763 §9, as observed.
		"meta-response.bin": build(nil, answers(
			ptr("_services._dns-sd._udp.local.", "_airplay._tcp.local."),
			ptr("_services._dns-sd._udp.local.", "_raop._tcp.local."),
			ptr("_services._dns-sd._udp.local.", "_companion-link._tcp.local."),
			ptr("_services._dns-sd._udp.local.", "_ipp._tcp.local."),
		)),
		// A fully-populated announcement: PTR in Answers, SRV/TXT/A riding in
		// Additionals, address on the listening segment.
		"local-instance.bin": build(
			answers(ptr("_airplay._tcp.local.", "Living Room._airplay._tcp.local.")),
			additionals(
				srv("Living Room._airplay._tcp.local.", "apple-tv.local.", 7000),
				txt("Living Room._airplay._tcp.local.", "model=AppleTV6,2", "flags=0x244", "noval"),
				a("apple-tv.local.", "192.168.20.40"),
			),
		),
		// The reflector signature: the advertised host is on another subnet,
		// while the sender (supplied by the test, not the packet) is here.
		"reflected-instance.bin": build(
			answers(ptr("_ipp._tcp.local.", "Front Desk Printer._ipp._tcp.local.")),
			additionals(
				srv("Front Desk Printer._ipp._tcp.local.", "printer-vlan40.local.", 631),
				txt("Front Desk Printer._ipp._tcp.local.", "rp=ipp/print", "pdl=application/pdf"),
				a("printer-vlan40.local.", "10.44.40.61"),
			),
		),
		// A multi-homed host: one on-segment address and one docker bridge
		// address. It is local, and a rule that demanded every address be
		// on-segment would call every developer laptop a reflector.
		"multihomed-instance.bin": build(
			answers(ptr("_http._tcp.local.", "Build Box._http._tcp.local.")),
			additionals(
				srv("Build Box._http._tcp.local.", "build-box.local.", 8080),
				a("build-box.local.", "192.168.20.51"),
				a("build-box.local.", "172.17.0.1"),
			),
		),
		// A non-ASCII instance name. dnsmessage passes the UTF-8 through
		// untouched, so the technician reads the name its owner typed.
		"utf8-instance.bin": build(
			answers(ptr("_raop._tcp.local.", "Café._raop._tcp.local.")),
			additionals(
				srv("Café._raop._tcp.local.", "speaker.local.", 5000),
				a("speaker.local.", "192.168.20.72"),
			),
		),
		// An instance whose name contains a literal dot, hand-assembled
		// because the library refuses to build one. dnsmessage rejects the
		// WHOLE packet for it ("Name: invalid dns name"), so the service is
		// invisible; the fixture exists to pin that as known behaviour rather
		// than let it surprise someone later.
		"dotted-instance.bin": dottedInstancePacket(),
		// The macOS shape: the instance PTR alone, details in a later packet.
		"split-ptr.bin": build(answers(
			ptr("_companion-link._tcp.local.", "Studio Display._companion-link._tcp.local."),
		), nil),
		"split-details.bin": build(answers(
			srv("Studio Display._companion-link._tcp.local.", "studio-display.local.", 62078),
			txt("Studio Display._companion-link._tcp.local.", "rpBA=A1:B2:C3"),
			a("studio-display.local.", "192.168.20.88"),
		), nil),
		// An AAAA-only advertisement, used to prove that an interface with no
		// global IPv6 prefix does not report every v6 host as off-segment.
		"v6-instance.bin": build(
			answers(ptr("_airplay._tcp.local.", "Bedroom._airplay._tcp.local.")),
			additionals(
				srv("Bedroom._airplay._tcp.local.", "bedroom.local.", 7000),
				aaaa("bedroom.local.", "2001:db8:44::10"),
			),
		),
		// A query, not a response: a browse must ignore other stations' asks.
		"query.bin": mustQuery("_services._dns-sd._udp.local."),
	}
	// Bytes no DNS parser will accept.
	fixtures["malformed.bin"] = []byte{0x00, 0x01, 0xff, 0xff, 0x7f}

	dir := filepath.Join("testdata")
	for name, data := range fixtures {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			panic(err)
		}
		fmt.Printf("%-26s %4d bytes\n", name, len(data))
	}
}

type section struct{ resources []dnsmessage.Resource }

func answers(rs ...dnsmessage.Resource) *section     { return &section{resources: rs} }
func additionals(rs ...dnsmessage.Resource) *section { return &section{resources: rs} }

func build(ans, add *section) []byte {
	b := dnsmessage.NewBuilder(make([]byte, 0, 1024), dnsmessage.Header{Response: true, Authoritative: true})
	b.EnableCompression()
	must(b.StartAnswers())
	if ans != nil {
		for _, r := range ans.resources {
			appendResource(&b, r)
		}
	}
	must(b.StartAuthorities())
	must(b.StartAdditionals())
	if add != nil {
		for _, r := range add.resources {
			appendResource(&b, r)
		}
	}
	out, err := b.Finish()
	must(err)
	return out
}

func appendResource(b *dnsmessage.Builder, r dnsmessage.Resource) {
	switch body := r.Body.(type) {
	case *dnsmessage.PTRResource:
		must(b.PTRResource(r.Header, *body))
	case *dnsmessage.SRVResource:
		must(b.SRVResource(r.Header, *body))
	case *dnsmessage.TXTResource:
		must(b.TXTResource(r.Header, *body))
	case *dnsmessage.AResource:
		must(b.AResource(r.Header, *body))
	case *dnsmessage.AAAAResource:
		must(b.AAAAResource(r.Header, *body))
	}
}

func header(name string, t dnsmessage.Type) dnsmessage.ResourceHeader {
	return dnsmessage.ResourceHeader{
		Name:  dnsmessage.MustNewName(name),
		Type:  t,
		Class: dnsmessage.ClassINET,
		TTL:   4500,
	}
}

func ptr(owner, target string) dnsmessage.Resource {
	return dnsmessage.Resource{
		Header: header(owner, dnsmessage.TypePTR),
		Body:   &dnsmessage.PTRResource{PTR: dnsmessage.MustNewName(target)},
	}
}

func srv(owner, target string, port uint16) dnsmessage.Resource {
	return dnsmessage.Resource{
		Header: header(owner, dnsmessage.TypeSRV),
		Body:   &dnsmessage.SRVResource{Port: port, Target: dnsmessage.MustNewName(target)},
	}
}

func txt(owner string, strs ...string) dnsmessage.Resource {
	return dnsmessage.Resource{
		Header: header(owner, dnsmessage.TypeTXT),
		Body:   &dnsmessage.TXTResource{TXT: strs},
	}
}

func a(owner, addr string) dnsmessage.Resource {
	ip := netip.MustParseAddr(addr).As4()
	return dnsmessage.Resource{
		Header: header(owner, dnsmessage.TypeA),
		Body:   &dnsmessage.AResource{A: ip},
	}
}

func aaaa(owner, addr string) dnsmessage.Resource {
	ip := netip.MustParseAddr(addr).As16()
	return dnsmessage.Resource{
		Header: header(owner, dnsmessage.TypeAAAA),
		Body:   &dnsmessage.AAAAResource{AAAA: ip},
	}
}

// dottedInstancePacket assembles "Office No. 5._ipp._tcp.local" byte by byte.
// dnsmessage.NewName refuses the name, so the packet cannot be built with the
// builder used for every other fixture.
func dottedInstancePacket() []byte {
	raw := []byte{0x00, 0x00, 0x84, 0x00, 0, 0, 0, 1, 0, 0, 0, 0}
	raw = appendLabel(raw, "Office No. 5")
	for _, l := range []string{"_ipp", "_tcp", "local"} {
		raw = appendLabel(raw, l)
	}
	raw = append(raw, 0x00)
	// PTR, IN, TTL 4500, rdata = a compression pointer back to the name.
	raw = append(raw, 0x00, 0x0c, 0x00, 0x01, 0x00, 0x00, 0x11, 0x94, 0x00, 0x02, 0xc0, 0x0c)
	return raw
}

func appendLabel(raw []byte, label string) []byte {
	raw = append(raw, byte(len(label)))
	return append(raw, label...)
}

func mustQuery(name string) []byte {
	b := dnsmessage.NewBuilder(make([]byte, 0, 256), dnsmessage.Header{})
	must(b.StartQuestions())
	must(b.Question(dnsmessage.Question{
		Name:  dnsmessage.MustNewName(name),
		Type:  dnsmessage.TypePTR,
		Class: dnsmessage.ClassINET,
	}))
	out, err := b.Finish()
	must(err)
	return out
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
